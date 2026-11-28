## Software architecture

Major deployable parts inside the CloudChamber system and how they cooperate. This page embeds one **container** diagram (product slice) plus **component** and **dynamic** views where the story is behavioural. LLM and platform telemetry have separate Structurizr views (**ContainersLLM**, **ContainersObservability**) so the same worker boxes are not drawn three times in the documentation.

![](embed:ContainersProduct)

### API service

Owns business HTTP routes and domain capabilities, talks to the database, relies on auth for caller validation, and uses Temporal for CloudChamber pipeline orchestration.

![](embed:APIServiceC3)

### Auth

Issues and validates tokens so the API service and clients share one trust model without duplicating crypto logic.

![](embed:AuthServiceC3)

### Metrics

Collects and exposes operational signals from the product API process without mixing metrics concerns into domain code.

### Data store

Holds durable relational state shared by domains. Schema changes flow through the same migration discipline as application code.

### Temporal orchestration

Temporal Service is the frontend, history, matching, and internal worker service. Dev runs those roles in one server process, with workflow pause enabled. Workflow history and visibility sit in a separate PostgreSQL instance, so a restart of that process keeps the run. In dev, Temporal server, UI, and its PostgreSQL run in the `temporal-system` namespace; the API and pipeline workers in `cloudchamber-system` reach the frontend over cluster DNS.

### Runs

A workflow is a definition of long-running work, and a run is one execution of it that a person asked for. The platform does not know what a workflow does: a run names its workflow, and a workflow is a workflow function, its activities, and the task queues its workers poll. Converting a recording is the `video-to-document` workflow. Adding a workflow adds a package and registers it. It changes no table and no platform code.

There is one set of endpoints for every workflow: create, list, read, the five commands, and the events of a run. A workflow differs in data, not in endpoints. Each workflow describes itself once: its name, its workflow type and task queue, its steps and their labels, a schema for its input, and a check of that input. The service registers its workflows in one place, and three readers use that description. The controller starts the workflow. The API runs the workflow's check before it stores a run, so a request the workflow would fail on is refused and no run is created. A client reads `GET /v1/workflows` to learn what it can start and what to call each step.

A run is declarative, in the way a Kubernetes resource is. Its spec is what the person wants: the input, whether it is suspended, whether it is canceled, and which attempt it should be on. Its status is what the platform last observed: the phase, the attempt, the steps, and the result or the failure. Each part has one writer.

| Part of the run | Writer | How |
| --- | --- | --- |
| Spec | API | Creating a run and each command: suspend, resume, cancel, retry, restart |
| Status | Run controller | After comparing the spec with what Temporal reports |
| Steps | Workers | An interceptor around every activity |
| Execution | Temporal | History, retries, timers, pause, cancellation |

The API never calls Temporal for a run. Creating a run stores it in the pending phase. A command edits the spec, guarded by the run's generation, so two commands that conflict cannot both win. A command that does not apply to the run's state is refused.

The run controller runs inside the API service and is the only caller of Temporal for runs. For each run that is not settled, it reads the spec, asks Temporal what is true, takes one action, and records the status. It repeats until the two agree. The decision is one pure function, tested as a table.

The controller is built the way a Kubernetes controller is.

- **It hears of changes, and does not poll for them.** The `runs` table announces every change of a spec and every change of a step's state on a PostgreSQL channel, whichever process made the change. The controller listens. A change a person asked for is looked at at once. A change a worker reported is batched for one second, so reports that arrive together cost one look. How far a step has got is not announced.
- **A change is a hint, not a fact.** It says which run to look at. The controller then reads the run and asks Temporal, so a change that is lost, repeated, or late does no harm. A resync every thirty seconds queues every unsettled run, and so does a reconnect of the listening connection.
- **A keyed work queue.** A run queued many times is looked at once, and no run is ever held by two workers. A reconcile that fails is tried again after a wait that doubles with each failure in a row.
- **One replica leads.** Each API replica runs a controller, and the one holding a PostgreSQL advisory lock does the work. The others take over when it stops. Leading avoids duplicated work. It is not what makes the work safe: every write to a run is guarded by the run's generation, and every action on Temporal can be repeated.

| The spec asks | Temporal reports | The controller |
| --- | --- | --- |
| A run exists | No execution | Starts the workflow |
| Suspend | Running | Pauses the execution, then the running activity |
| Not suspended | Paused | Unpauses the execution, then the activity |
| Cancel | Open | Requests cancellation, unpauses a paused execution, and terminates after a grace period |
| A later attempt, from the failure | Closed | Resets the execution to the step that did not finish |
| A later attempt, from the start | Closed | Starts the workflow again |
| Anything | Closed | Records the final phase and the result or the failure |

Suspend stops the running step now. A paused execution schedules no new step, and a paused activity is interrupted on its next heartbeat. The worker's interceptor heartbeats once a second for every step, so a step's context is canceled soon after a suspend or a cancel even when its author wrote no heartbeat. Temporal sends heartbeats at most every 0.8 of the step's heartbeat timeout, so that timeout still bounds how quickly a step stops. A step that records its place in its heartbeat continues from that place when the run resumes. A step whose code ignores its context runs until it returns. Because the interceptor's heartbeat comes from a timer, it keeps arriving when the work has hung, and a worker can turn it off to have a hung step declared dead.

Workers record steps without the workflow author writing anything. An interceptor around every activity records the step as working when an attempt starts and records how it ended: done, failed, canceled, or paused. An activity may say how far it has got, and that overwrites the step in place at most once a second, so a long step adds no rows. A failure that Temporal will retry leaves the step working.

Every command and every change of phase is appended to an event record, with who asked. Nothing reads that record to compute state.

### Converting a recording

The `video-to-document` workflow runs ResolveSource, ExtractAudio, Transcribe, ExtractKeyFrames, then ComposeDocument. Media, ASR, frames, and render each poll their own task queue, and the workflow worker polls the workflow's queue. ResolveSource checks that the recording is a ready file in the run's project, because the run API does not know what a file is. ExtractAudio downloads the source recording and writes 16 kHz mono 16-bit PCM WAV. A recording with no picture has no key-frame step. Each writing step stores its object and registers it as a file in the run's project. The file id is derived from the run and the kind of file, so a step that runs again replaces its own file. The workflow returns the file ids, and they become the run's result. In dev, Transcribe, ExtractKeyFrames, and ComposeDocument are stand-ins that write a small file of the right type.

The `sample` workflow is the second one. It has three steps and no product meaning, and it exists to exercise the run platform end to end. It runs on the workflow worker.

A transient activity error is retried on the same activity, and the step stays working. A permanent activity error, or the last of five attempts, fails the step, and the run fails with that step's message. Outputs already registered stay in the project. A retry continues from the step that failed. A restart runs every step again.

![](embed:PipelineProcessDynamic)

![](embed:PipelineFailureDynamic)

![](embed:PipelineCancelDynamic)

### LiteLLM gateway

Routes LLM provider calls for pipeline transcription and document steps. It uses a dedicated PostgreSQL instance, separate from application data and Temporal. In dev, LiteLLM and its database run in `litellm-system`. Operators configure it through an admin UI; pipeline workers in `cloudchamber-system` call it via cluster DNS when real ASR and compose replace the dev stubs.

The proxy does not embed the Langfuse SDK. With the `langfuse_otel` callback and LiteLLM OTel v2, it exports LLM spans to Langfuse over OTLP. Prompt and completion bodies require an explicit capture mode (`span_only` in dev); the OTel v2 default omits them. LLM observability does not go to Tempo.

### LLM observability (Langfuse)

Stores generations, sessions, token usage, and cost for calls that pass through LiteLLM. Langfuse web serves the operator UI and accepts OTLP from LiteLLM in the same cluster. A worker processes events asynchronously. PostgreSQL, ClickHouse, Redis, and S3-compatible blob storage are dedicated to Langfuse and are not shared with application data or pipeline artifacts.

In dev, LiteLLM runs in `litellm-system`, Langfuse in `langfuse-system`, Temporal in `temporal-system`, platform telemetry in `observability-system`, while the API and pipeline workers stay in `cloudchamber-system`. Go workers do not send traces to Langfuse; they export distributed traces to Tempo over cluster DNS only.

### Platform observability (Tempo, Grafana, Loki, Prometheus, Alloy)

Tempo holds distributed traces from the API service and pipeline workers (OTLP). Prometheus scrapes application metrics from `cloudchamber-system`. Alloy collects pod logs cluster-wide into Loki. Grafana is the dev UI for traces, metrics, and logs. In dev the stack runs in `observability-system`, separate from `cloudchamber-system`, `temporal-system`, and `langfuse-system`. This path is separate from Langfuse, which is scoped to LLM traffic through LiteLLM.

### Artifact storage

Holds the bytes of recordings and pipeline outputs. Pipelines pass file ids, not the file bytes. Dev uses SeaweedFS. Which project a file belongs to is the artifact row in application PostgreSQL. The object key is the owner and the file id.

A project has one owner. Every file belongs to one project, and that project's owner is the file's owner. The person names the project when they start the upload. The API checks they own it, then records the file there before the upload is confirmed. The upload service does not play the recording.

![](embed:UploadTusHooksDynamic)

A run names its project, and the API refuses a project the caller does not own. The workflow checks that the recording is in that project. Each output the workers register goes into the same project.

A person opens a recording. The API checks they own it and returns a short-lived link. The browser plays the recording from storage. Deleting a file removes the stored recording, then the record of it. A project cannot be deleted while it still has files or runs.

![](embed:WatchArtifactDynamic)

### Development deployment

Local work runs the same services together in one cluster.

![](embed:DeploymentDevKind)

### Production deployment

Production uses the same namespace boundaries as KIND dev: product workloads in `cloudchamber-system`, with `observability-system`, `temporal-system`, `litellm-system`, and `langfuse-system` for platform stacks. Ingress terminates HTTPS in front of the application tier.

![](embed:DeploymentProduction)
