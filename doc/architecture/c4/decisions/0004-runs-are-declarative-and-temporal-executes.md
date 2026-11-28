# 4. Runs are declarative and Temporal executes them

Date: 2026-11-28

## Status

Accepted

## Context

The product needs long-running work that a person can start, suspend, resume, cancel, retry, and restart, and that survives a crash or a deploy. Converting a recording is the first such work. More kinds of work will follow, and the platform should not need to know what each one does.

The first design recorded each pipeline job as an event stream in Message DB, folded that stream into a job row on every append, and proved the fold in Lean. Temporal already kept the history of the same execution, so the same fact was held in Temporal, in the stream, in the row, and in the artifact table. The stage names of the video pipeline were built into the business package, the row, the fold, and the frontend. Every write for every job took one advisory lock, and each second of progress was stored forever.

## Decision

Temporal is the engine and the authority on execution. Workflows are plain Temporal code, and there is no layer that hides Temporal from them. The platform uses what Temporal provides and does not rebuild it: durable history, retries and timers, cancellation and termination, workflow pause, activity pause, and reset.

A run is a row in `runs` and is declarative. Its spec is what the person wants. Its status is what was observed. The API writes the spec. One controller compares the spec with what Temporal reports, acts on the difference, and writes the status. Workers write the steps. No caller writes a part it does not own, and the business package gives each caller an interface with only its own part.

The control policy is one pure function from the spec, the status, and Temporal's report to one action. It is tested as a table. The invariants a run must keep are constraints in the schema.

A workflow is a definition of work, and a run is one execution of it. A workflow is a package: a workflow function, its activities, the task queues they use, and a definition of itself with its step labels, an input schema, and a check of its input. The platform holds no workflow's vocabulary. Every workflow is reached through the same endpoints, and one more endpoint lists the workflows.

Progress is a fraction and a sentence on the step, overwritten in place. Commands and phase changes are appended to `run_events`, which nothing reads to compute state.

## Consequences

Adding a workflow adds a package and one line where workflows are registered. It needs no migration and no change to the platform.

The platform depends on Temporal features that are not yet generally available: workflow pause is pre-release and needs server 1.30 or later with `frontend.WorkflowPauseEnabled`, and activity pause is in public preview. The run API hides both, so either can be replaced by a signal the workflow waits on.

There is no event-sourced history of a run. What happened inside an execution is in Temporal's history for as long as Temporal retains it, and in traces and logs. `run_events` holds the commands and the phase changes.

The controller follows the shape of a Kubernetes controller: it hears of changes on a PostgreSQL channel and does not poll for them, it keeps a keyed work queue, it retries a failed reconcile with backoff, and one replica leads. A change is only a hint to look at a run, so a lost or repeated one does no harm, and a resync every thirty seconds is the safety net.

A run's status trails Temporal by the time the controller takes to look. A change a person asked for is looked at at once. A change a worker reported is looked at within a second. A run whose workflow does more than a second of work after its last step is seen on the next look at a run in transition, or on the resync.

Leading keeps API replicas from repeating each other's work. It does not guarantee that only one acts, because a replica can lose its database connection a moment before it notices. Safety rests on writes guarded by the run's generation and on actions that can be repeated.

A suspend or a cancel reaches a running step on its next heartbeat. The worker heartbeats for every step, so a step needs no code of its own to be stoppable, but the step's heartbeat timeout still bounds how quickly it stops, and a step whose code ignores its context runs until it returns. Temporal cannot kill activity code. Killing a step needs a process boundary, which this design does not have. The worker's heartbeat also hides a step that has hung, and a worker that needs a hung step declared dead turns it off.

The Message DB schema, the job fold, and the Lean proof are removed.
