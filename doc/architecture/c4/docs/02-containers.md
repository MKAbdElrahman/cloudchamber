## Software architecture

Major deployable parts inside the CloudChamber system and how they cooperate.

![](embed:Containers)

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

Temporal Service is the frontend, history, matching, and internal worker service. Dev runs those roles in one auto-setup process. Workflow history and visibility sit in a separate PostgreSQL instance, so a restart of that process keeps the run. In dev, Temporal server, UI, and its PostgreSQL run in the `temporal-system` namespace; the API and pipeline workers in `cloudchamber-system` reach the frontend over cluster DNS.

The API is the Temporal client. It checks the caller, records the job, and starts ProcessSourceWorkflow. That start returns after History records the execution and schedules the first workflow task. The API stores the run id, marks the job running, and returns the job. If the start is not accepted, the API stores failed with a fixed reason and logs the client error. The workflow worker polls that task after it is queued, and that poll may overlap the API response. The API asks the frontend to cancel an execution and does not change the job row. A list reads the job rows and does not ask the frontend. Opening a job whose row is still open reads the Activity Type the open run has reached and does not store it. A row that is already closed is not read from the frontend again. The workers write the file ids and the closed outcome. Those reads are later requests. They are not part of the workflow trace. The job row records the source file's owner and project.

The workflow worker only polls Temporal. The first workflow task schedules ResolveSource. It then waits. The next activity is queued only after that result is reported, which takes another workflow task. The order is ExtractAudio, Transcribe, ExtractKeyFrames, then ComposeExport. Each writing step stores its object, registers it in the source file's project, and records that file id on the job. ComposeExport then stores completed and clears the failure message and the stopped activity type. One worker polls each task queue. A crash waits until that worker polls again. After a rollout, the new build replays an open run from its history. Media, frames, ASR, and render poll their own task queues. ExtractAudio downloads the source recording and writes 16 kHz mono 16-bit PCM WAV. In dev, ResolveSource, MarkCanceled, and the frames, ASR, and render activities are simulated stand-ins: they check or write objects, and they do not download the source, the audio, or earlier outputs.

A transient activity error is retried on the same activity, and the job stays running. A permanent activity error, or the last of five attempts, is stored by that activity as failed, with its failure message and the stopped activity type, before the activity returns. The workflow returns that error and closes the execution as failed. Outputs already registered stay in the project. A cancel waits for the in-flight activity to close. That activity stores canceled and its own name as the stopped activity type, with no failure message, before it returns. If it had already registered its file, that file id stays. The workflow does not schedule the next activity. A cancel that arrives after a step returns and before the next is scheduled is stored by MarkCanceled on the media queue, using the step already finished. The workflow waits for that write, then returns the cancel. The execution closes as canceled. A task waits until the one worker on that queue polls it.

![](embed:PipelineProcessDynamic)

![](embed:PipelineFailureDynamic)

![](embed:PipelineCancelDynamic)

### LiteLLM gateway

Routes LLM provider calls for pipeline transcription and document steps. It uses a dedicated PostgreSQL instance, separate from application data and Temporal. Operators configure it through an admin UI; pipeline workers call it inside the cluster when real ASR and compose replace the dev stubs.

The proxy does not embed the Langfuse SDK. With the `langfuse_otel` callback and LiteLLM OTel v2, it exports LLM spans to Langfuse over OTLP. Prompt and completion bodies require an explicit capture mode (`span_only` in dev); the OTel v2 default omits them. LLM observability does not go to Tempo.

### LLM observability (Langfuse)

Stores generations, sessions, token usage, and cost for calls that pass through LiteLLM. Langfuse web serves the operator UI and accepts OTLP from LiteLLM in the same cluster. A worker processes events asynchronously. PostgreSQL, ClickHouse, Redis, and S3-compatible blob storage are dedicated to Langfuse and are not shared with application data or pipeline artifacts.

In dev, Langfuse runs in `langfuse-system`, Temporal in `temporal-system`, platform telemetry in `observability-system`, while LiteLLM and pipeline workers stay in `cloudchamber-system`. Go workers do not send traces to Langfuse; they export distributed traces to Tempo over cluster DNS only.

### Platform observability (Tempo, Grafana, Loki, Prometheus, Alloy)

Tempo holds distributed traces from the API service and pipeline workers (OTLP). Prometheus scrapes application metrics from `cloudchamber-system`. Alloy collects pod logs cluster-wide into Loki. Grafana is the dev UI for traces, metrics, and logs. In dev the stack runs in `observability-system`, separate from `cloudchamber-system`, `temporal-system`, and `langfuse-system`. This path is separate from Langfuse, which is scoped to LLM traffic through LiteLLM.

### Artifact storage

Holds the bytes of recordings and pipeline outputs. Pipelines pass file ids, not the file bytes. Dev uses SeaweedFS. Which project a file belongs to is the artifact row in application PostgreSQL. The object key is the owner and the file id.

A project has one owner. Every file belongs to one project, and that project's owner is the file's owner. The person names the project when they start the upload. The API checks they own it, then records the file there before the upload is confirmed. The upload service does not play the recording.

![](embed:UploadTusHooksDynamic)

A job does not take a project from the client. It copies the owner and the project from the source file. Each output the workers register copies the same pair.

A person opens a recording. The API checks they own it and returns a short-lived link. The browser plays the recording from storage. Deleting a file removes the stored recording, then the record of it. A project cannot be deleted while it still has files or jobs.

![](embed:WatchArtifactDynamic)

### Development deployment

Local work runs the same services together in one cluster.

![](embed:DeploymentDevKind)
