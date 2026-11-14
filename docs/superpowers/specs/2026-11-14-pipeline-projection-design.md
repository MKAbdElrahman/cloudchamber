# Pipeline job projection

Date: 2026-11-14

## Intent

A pipeline job has one history: the Message DB stream `job-<job id>`. The `pipeline_jobs` row is a projection of that stream, updated in the same database transaction as each append. The row is rebuilt from the log alone. It exists so lists, idempotency, and the sweep stay ordinary queries.

There are no clients on the old fold. Existing jobs are migrated once onto this log, and the code keeps a single path.

## Why this shape

Each fact has one job. Run facts say what the job is. Step facts say what a step is doing and which file it produced. The row is the fold of those facts and nothing else. One function folds. One function writes. Callers append a fact and stop there.

## Goals

- Every projected column comes from the log: status, failure message, run id, file ids, owner, project, input file, idempotency hash, and the two dates.
- The fact and the row commit together, or neither does.
- A run ends once.
- The screen reads the same status the row stores.
- After the migration, no reader still applies the old status rules.

## Non-goals

- A deadline for a worker that never starts (`ScheduleToStart`).
- Changing activity `StartToClose` timeouts.
- Replacing the transcription, key-frame, or compose stubs.
- Renaming the `sim` package.
- Dropping the row, or folding the stream on every list read.
- Keeping a second status rule for jobs created before this change.

## Anti-goals

- A step fact, or a file column, that changes job status.
- A second writer of status, the failure message, the run id, or a file id.
- A finished job that a later step fact opens again.
- A fold that consults the previous row.

## Approaches

1. **Run facts own the job. Step facts own the steps.** Same-transaction projection, and a one-time migration of existing rows onto that log. Chosen.
2. **Seven-rule fold, kept for old rows.** Step facts also move job status, a succeeded compose step means completed, and a document file id finishes a job that has no facts. Rejected: there is nobody on that behavior, and it leaves two meanings of status in the code.
3. **Later follower, or no status columns.** Rejected earlier: the list would lag the log, or every read would fold the stream.

## What stays

- Stream name `job-<uuid>`. Category `job`. The category advisory lock stays.
- `write_message` stays two statements inside the projection transaction: `SELECT write_message`, then `SELECT` the row by stream name and position.
- `expected_version` stays. An empty stream is `-1`. A wrong version retries. Stable writes do not mint a new id.
- Fact data names stay: `step`, `state`, `progress`, `message`, `fileId`.
- Every fact's metadata keeps `projectId` and `ownerId`.
- Follow stays long-poll. `LISTEN` is a wake-up, not the fact. Follow routes stay off `AuthorizePipeline`.
- Step facts already in a stream stay. The migration adds the run facts those streams are missing. It does not rewrite progress history.

## Three parts

### 1. Log

`Append` is the only write. It stores one fact, folds that job's stream, replaces the projected columns, and commits.

Create, the workflow, and the sweep append run facts. Workers append step facts. None of them update `pipeline_jobs` themselves.

### 2. Facts

A run fact has an empty `step`. There are three, and each job writes each one at most once:

| Fact | State | What it carries | Stable id |
| --- | --- | --- | --- |
| Accepted | `submitted` | metadata `inputFileId`; metadata `idempotencyKeyHash` when the caller sent a key | new UUID v5 namespace, fixed in code |
| Started | `running` | metadata `runId` | new UUID v5 namespace, fixed in code |
| Ended | `completed`, `failed`, or `canceled` | `message` on failure, clipped as today | existing `RunFactID` |

`submitted` and `completed` are legal only on a run fact. A second write of the same stable id returns the fact already stored and does not change it. The first ending wins: a later attempt to fail a completed run, or complete a failed run, leaves the first fact in place. An ending already stored under another id counts as that fact. The migration does not write a second one, and a later `RunFactID` write is skipped when any ending is already in the stream.

The workflow id is `cloudchamber-job-<job id>`. The fold derives it from the job id. It is not a fact.

A step fact names a step and uses `running`, `succeeded`, `skipped`, `failed`, or `canceled`. `progress` while running, `message` when there is one, `fileId` when that fact produced a file. Step facts keep a new id on each append.

### 3. Fold

`Fold` is a pure function of the facts. It does not read the current row. Append and rebuild both write its result over the row.

- Status is the latest ending fact's state when the stream has one. Otherwise it is the latest of the submitted and started facts. A stream with no run fact projects `submitted`. A started fact that lands after the ending records the run id and leaves the ending in charge.
- `failed`: store that fact's message, or `failed` when the message is empty.
- `canceled`, `completed`, `running`, and `submitted` clear the failure message.
- Run id is the `runId` on the started fact, wherever that fact sits, or empty when it is absent.
- Each file column is the `fileId` on the latest succeeded fact for that step that has one. A fact that omits `fileId` leaves an earlier one in place. A step with no such fact projects an empty column.
- Owner, project, input file, and idempotency hash come from the submitted fact. Absent means empty.
- Date created is the submitted fact's message time. Date updated is the latest fact's message time. `write_message` stamps the database clock. The migration, in the same transaction, sets the submitted message's time to the row's date created. Live appends leave the clock alone.
- Workflow id is always the derived id.

## What happens

Create:

1. Append `submitted` and insert the row from that fold. Commit.
2. Start the workflow, outside that transaction.
3. Append `running` with the run id when the start returns one. Append `failed` when the start itself fails. If the workflow has already appended the ending, the started fact still lands, and the ending keeps the status.

Workers append step facts only. The first running fact of a step still fails the step when the append fails. Later progress appends stay best-effort. Publishing a file puts `fileId` on the succeeded step fact and does not touch the row.

The workflow appends the ending fact on the way out: `completed` when it returns success, `failed` or `canceled` when it returns an error. Compose appending its succeeded step fact does not finish the job. The ending fact does.

The sweep lists rows still `submitted` or `running`. When Temporal has closed that execution, the sweep appends the matching ending fact: completed, failed, or canceled. `NotFound` and a still-running execution are skipped.

A crash after the submitted commit and before the start call leaves the job `submitted`. The sweep skips `NotFound`. Closing that gap is a deadline, and deadlines are out of scope.

## Migration

One pass, before the new process serves traffic. For each existing job, through `Append`, at the tail of the stream, and only for facts that are not already there:

1. The submitted fact, from the row's owner, project, input file, and idempotency hash. Then set that message's time to the row's date created.
2. The started fact, when the row's run id is non-empty.
3. A succeeded step fact for each file column that is set and that no succeeded step fact already carries: `extract_audio`, `transcribe`, `extract_keyframes`, `compose_export`.
4. The ending fact, when the row status is `completed`, `failed`, or `canceled` and the stream has no empty-step `completed`, `failed`, or `canceled` fact. A failed row copies its failure message. An ending left by the earlier step-table backfill, including one with a random id, is that fact and is kept.

The fold then replaces the row. A job whose row is `submitted` or `running` gets no ending fact. Appending submitted or started after an existing ending does not reopen the job.

This pass is the last code that reads a row as authority. The request path never branches on "old job" versus "new job".

## Rebuild

The same `Fold`, over the stream from position 0, written over the row. For an operator or a test, not for a request. Dropping the row and rebuilding produces the same projection.

## UI

The badge is the row status. That is the ending fact when the stream has one, and otherwise the latest submitted or started fact. Step display stays on the step facts: the current step, its progress, and the lines under it.

Remove the old client fold: step facts deciding job status, compose success meaning completed, and a document file id completing a job that has no facts. The tests that lock those rules go with them.

## Tests

- The fold of a stream equals the row, with no leftover from the previous row.
- Status tracks the ending fact when one exists, including a started fact or a step fact that arrives after it.
- With no ending, status tracks the latest of submitted and started.
- A step fact does not change status.
- A file id stays when a later fact for that step has none, and a step with no file fact projects an empty column.
- The second ending append returns the first fact unchanged.
- Submitted and started metadata round-trip.
- Stream name stays `job-<uuid>`. `RunFactID` stays stable.
- The fact is absent when the row update fails. The row is unchanged when the fact write fails.
- A migrated job with a terminal row status, a run id, and file columns folds back to those values from the stream alone.

## Left to the builder

The transaction helper, the SQL, where the one-time pass is invoked, and the names of the helpers that build the three run facts. Strong types stay in the business package. Primitive columns stay in the store. Converters stay at the boundary. The two new UUID v5 namespaces are chosen once and locked by the stable-id test.
