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

Temporal Service provides durable workflow orchestration for long-running pipelines (workflow state, task queues, retries/timeouts, visibility). The primary API service starts and queries pipeline workflows. A worker polls each task queue. In dev, the media, transcription, and export activities are simulated stand-ins that still write real files into the source file's project. The dev stack uses auto-setup with a physically isolated PostgreSQL instance for Temporal persistence/visibility and Temporal UI for workflow visibility.

### Artifact storage

Holds each project's recordings and documents. Pipelines pass file ids, not the file bytes. Dev uses SeaweedFS.

A project has one owner. Every file belongs to one project, and that project's owner is the file's owner. The person names the project when they start the upload. The API checks they own it, then records the file there before the upload is confirmed. The upload service does not play the recording.

![](embed:UploadTusHooksDynamic)

A job does not take a project from the client. It copies the owner and the project from the source file. Each output the workers register copies the same pair.

A person opens a recording. The API checks they own it and returns a short-lived link. The browser plays the recording from storage. Deleting a file removes the stored recording, then the record of it. A project cannot be deleted while it still has files or jobs.

![](embed:WatchArtifactDynamic)

### Development deployment

Local work targets a small Kubernetes style layout that mirrors how services run together in real environments, not one off scripts per developer.

![](embed:DeploymentDevKind)
