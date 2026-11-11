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

### Artifact storage (S3-compatible)

Production-like pipelines pass file ids (not large blobs) through workflow history. S3-compatible object storage holds source media, extracted audio, transcripts, key frames, and exported documents. The dev infrastructure uses SeaweedFS; production uses an operated S3-compatible store with appropriate durability, security, and lifecycle controls.

A project has one owner. Every file belongs to one project, and that project's owner is the file's owner. The browser names the project when it creates the upload. pre-create checks that the caller owns it. pre-finish records the source file in that project before tusd tells the browser the upload finished. tusd does not serve the bytes.

![](embed:UploadTusHooksDynamic)

A job does not take a project from the client. It copies the owner and the project from the source file. Each output the workers register copies the same pair.

Watching uses the file id. The API checks that the caller owns the file and returns a short-lived URL. The browser then reads the bytes, including range requests, from artifact storage. The bucket and object key stay on the file row. Deleting a file removes the object first, then the row. A project delete is refused while a file or a job still points at it.

![](embed:WatchArtifactDynamic)

### Development deployment

Local work targets a small Kubernetes style layout that mirrors how services run together in real environments, not one off scripts per developer.

![](embed:DeploymentDevKind)
