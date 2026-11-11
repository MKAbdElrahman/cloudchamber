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

Temporal Service is the frontend, history, matching, and internal worker service. Dev runs those roles in one auto-setup process. Workflow history and visibility sit in a separate PostgreSQL instance, so a restart of that process keeps the run.

The API is the Temporal client. It checks the caller, records the job, and starts ProcessSourceWorkflow. That start returns after History records the execution and schedules the first workflow task. The API stores the run id and returns the job. The workflow worker polls that task after it is queued, and that poll may overlap the API response. The API asks the frontend to cancel an execution and leaves the job row open. A list reads the job rows and does not ask the frontend. Opening a job whose row is still open reads that execution and its progress query. A completed execution copies the output file ids onto the row. A failed execution stores failed, the activity error, and the file ids already held. A canceled execution stores canceled, the activity name, and those file ids. A row that is already closed is not read from the frontend again. Those reads are later requests. They are not part of the workflow trace. The job row records the source file's owner and project.

The workflow worker only polls Temporal. The first workflow task records cancel-waits version 2, then schedules ResolveSource. It then waits. The next activity is queued only after that result is reported, which takes another workflow task. The order is ExtractAudio, Transcribe, ExtractKeyFrames, then ComposeExport. The task that schedules ExtractKeyFrames records the frames-queue version first. Production uses worker versioning. A pinned run stays on the build that started it, and two workers of that build poll each task queue, so if one stops the other continues the run. An auto-upgrade run moves to the target build, and that workflow change still has to replay. Dev runs one worker per queue, and a crash waits until that worker polls again. Dev patches workflow changes in code. Media, frames, ASR, and render poll their own task queues. In dev those activities are simulated stand-ins: they check or write objects, and they do not download the source, the audio, or earlier outputs.

A transient activity error is retried on the same activity. A permanent activity error is not retried. The workflow returns that error and closes the execution as failed. Outputs already registered stay in the project. A new run records cancel-waits version 2 and waits for an in-flight activity to close after a cancel request. The activity learns about the cancel on its next heartbeat and stops before writing. If that activity completes anyway, the workflow returns the cancel error before scheduling the next activity, and the progress query stays on the activity that closed. The execution closes as canceled. Version 1 waits and can still schedule the next activity. A run started before version 1 returns when the request arrives. Workers expose poll and schedule-to-start metrics for the metrics service to scrape. Schedule-to-start stays open, so a task waits for the next worker allowed to poll that queue. For a pinned run, that worker is the starting build.

![](embed:PipelineProcessDynamic)

![](embed:PipelineFailureDynamic)

![](embed:PipelineCancelDynamic)

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
