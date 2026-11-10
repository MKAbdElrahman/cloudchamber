## Software architecture

Major deployable parts inside the CloudChamber system and how they cooperate.

![](embed:Containers)

### API service

Owns business HTTP routes and domain capabilities, talks to the database, relies on auth for caller validation, and (planned) uses Temporal for CloudChamber pipeline orchestration.

![](embed:APIServiceC3)

### Auth

Issues and validates tokens so the API service and clients share one trust model without duplicating crypto logic.

![](embed:AuthServiceC3)

### Metrics

Collects and exposes operational signals from the product API process without mixing metrics concerns into domain code.

### Data store

Holds durable relational state shared by domains. Schema changes flow through the same migration discipline as application code.

### Temporal orchestration

Temporal Service provides durable workflow orchestration for long-running pipelines (workflow state, task queues, retries/timeouts, visibility). The primary API service acts as a Temporal client to start/query pipeline workflows (planned), while split-ready pipeline workers poll capability-specific task queues and execute workflow/activity tasks (planned). The dev stack uses auto-setup with a physically isolated PostgreSQL instance for Temporal persistence/visibility and Temporal UI for workflow visibility.

### Artifact storage (S3-compatible)

Production-like pipelines pass artifact refs/IDs (not large blobs) through workflow history. S3-compatible object storage holds source media, extracted audio, transcripts, key frames, and exported documents. The dev infrastructure uses SeaweedFS (`s3://` API, refs only); production uses an operated S3-compatible store with appropriate durability, security, and lifecycle controls.

### Development deployment

Local work targets a small Kubernetes style layout that mirrors how services run together in real environments, not one off scripts per developer.

![](embed:DeploymentDevKind)
