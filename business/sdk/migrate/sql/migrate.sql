-- Version: 1.01
-- Description: Create table users
CREATE TABLE users (
	user_id       UUID        NOT NULL,
	name          TEXT        NOT NULL,
	email         TEXT UNIQUE NOT NULL,
	roles         TEXT[]      NOT NULL,
	password_hash TEXT        NOT NULL,
    department    TEXT        NULL,
    enabled       BOOLEAN     NOT NULL,
	date_created  TIMESTAMP   NOT NULL,
	date_updated  TIMESTAMP   NOT NULL,

	PRIMARY KEY (user_id)
);

-- Version: 1.02
-- Description: Create table products
CREATE TABLE products (
	product_id   UUID           NOT NULL,
    user_id      UUID           NOT NULL,
	name         TEXT           NOT NULL,
    cost         NUMERIC(10, 2) NOT NULL,
	quantity     INT            NOT NULL,
	date_created TIMESTAMP      NOT NULL,
	date_updated TIMESTAMP      NOT NULL,

	PRIMARY KEY (product_id),
	FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

-- Version: 1.03
-- Description: Add products view.
CREATE OR REPLACE VIEW view_products AS
SELECT
    p.product_id,
    p.user_id,
	p.name,
    p.cost,
	p.quantity,
    p.date_created,
    p.date_updated,
    u.name AS user_name
FROM
    products AS p
JOIN
    users AS u ON u.user_id = p.user_id

-- Version: 1.04
-- Description: Create table homes
CREATE TABLE homes (
    home_id       UUID       NOT NULL,
    type          TEXT       NOT NULL,
    user_id       UUID       NOT NULL,
    address_1     TEXT       NOT NULL,
    address_2     TEXT       NULL,
    zip_code      TEXT       NOT NULL,
    city          TEXT       NOT NULL,
    state         TEXT       NOT NULL,
    country       TEXT       NOT NULL,
    date_created  TIMESTAMP  NOT NULL,
    date_updated  TIMESTAMP  NOT NULL,

    PRIMARY KEY (home_id),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

-- Version: 1.05
-- Description: Create table audit
CREATE TABLE audit (
    id          UUID      NOT NULL,
    obj_id      UUID      NOT NULL,
    obj_domain  TEXT      NOT NULL,
    obj_name    TEXT      NOT NULL,
    actor_id    UUID      NOT NULL,
    action      TEXT      NOT NULL,
    data        JSONB     NULL,
    message     TEXT      NULL,
    timestamp   TIMESTAMP NOT NULL,

    PRIMARY KEY (id)
);

-- Version: 1.06
-- Description: Create table pipeline_jobs
CREATE TABLE pipeline_jobs (
    job_id                UUID      NOT NULL,
    owner_id              UUID      NOT NULL,
    workflow_id           TEXT      NOT NULL,
    run_id                TEXT      NOT NULL DEFAULT '',
    source_ref            TEXT      NOT NULL,
    idempotency_key_hash  TEXT      NULL,
    status                TEXT      NOT NULL,
    error_message         TEXT      NULL,
    audio_ref             TEXT      NULL,
    transcript_ref        TEXT      NULL,
    key_frames_ref        TEXT      NULL,
    document_ref          TEXT      NULL,
    date_created          TIMESTAMP NOT NULL,
    date_updated          TIMESTAMP NOT NULL,

    PRIMARY KEY (job_id),
    FOREIGN KEY (owner_id) REFERENCES users(user_id) ON DELETE CASCADE,
    UNIQUE (owner_id, idempotency_key_hash)
);

CREATE INDEX pipeline_jobs_owner_date_idx ON pipeline_jobs (owner_id, date_created DESC);
CREATE INDEX pipeline_jobs_workflow_id_idx ON pipeline_jobs (workflow_id);
-- Version: 1.07
-- Description: Create table uploads
CREATE TABLE uploads (
    object_key     TEXT        NOT NULL,
    owner_id       UUID        NOT NULL,
    bucket         TEXT        NOT NULL,
    object_key_ref TEXT        NOT NULL,
    source_ref     TEXT        NOT NULL,
    size_bytes     BIGINT      NOT NULL DEFAULT 0,
    etag           TEXT        NULL,
    status         TEXT        NOT NULL,
    tusd_upload_id TEXT        NULL,
    date_created   TIMESTAMP   NOT NULL,
    date_updated   TIMESTAMP   NOT NULL,

    PRIMARY KEY (object_key),
    FOREIGN KEY (owner_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX uploads_owner_date_idx ON uploads (owner_id, date_created DESC);

-- Version: 1.08
-- Description: Create table artifacts
CREATE TABLE artifacts (
    file_id      UUID        NOT NULL,
    owner_id     UUID        NOT NULL,
    bucket       TEXT        NOT NULL,
    object_key   TEXT        NOT NULL,
    source_ref   TEXT        NOT NULL,
    kind         TEXT        NOT NULL,
    content_type TEXT        NOT NULL DEFAULT '',
    status       TEXT        NOT NULL,
    date_created TIMESTAMP   NOT NULL,
    date_updated TIMESTAMP   NOT NULL,

    PRIMARY KEY (file_id),
    UNIQUE (source_ref),
    UNIQUE (object_key),
    FOREIGN KEY (owner_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX artifacts_owner_date_idx ON artifacts (owner_id, date_created DESC);

INSERT INTO artifacts (
    file_id, owner_id, bucket, object_key, source_ref, kind, content_type, status, date_created, date_updated
)
SELECT
    split_part(object_key, '/', 3)::uuid,
    owner_id,
    bucket,
    object_key,
    source_ref,
    'source',
    '',
    'ready',
    date_created,
    date_updated
FROM uploads
WHERE status = 'ready'
  AND object_key ~* '^sources/[0-9a-f-]{36}/[0-9a-f-]{36}$'
ON CONFLICT (file_id) DO NOTHING;

-- Version: 1.09
-- Description: Name each artifact and point a job at its input file
ALTER TABLE artifacts ADD COLUMN name TEXT NOT NULL DEFAULT '';

ALTER TABLE pipeline_jobs ADD COLUMN input_file_id UUID NULL REFERENCES artifacts(file_id);

UPDATE pipeline_jobs AS job
SET input_file_id = artifact.file_id
FROM artifacts AS artifact
WHERE artifact.source_ref = job.source_ref
  AND job.input_file_id IS NULL;

-- Version: 1.10
-- Description: Keep bucket and object key on the artifact, and point a job at output files
ALTER TABLE pipeline_jobs
    ADD COLUMN audio_file_id UUID NULL REFERENCES artifacts(file_id),
    ADD COLUMN transcript_file_id UUID NULL REFERENCES artifacts(file_id),
    ADD COLUMN key_frames_file_id UUID NULL REFERENCES artifacts(file_id),
    ADD COLUMN document_file_id UUID NULL REFERENCES artifacts(file_id);

ALTER TABLE pipeline_jobs DROP COLUMN audio_ref;
ALTER TABLE pipeline_jobs DROP COLUMN transcript_ref;
ALTER TABLE pipeline_jobs DROP COLUMN key_frames_ref;
ALTER TABLE pipeline_jobs DROP COLUMN document_ref;
ALTER TABLE pipeline_jobs DROP COLUMN source_ref;

ALTER TABLE artifacts DROP COLUMN source_ref;
ALTER TABLE uploads DROP COLUMN source_ref;

-- Version: 1.11
-- Description: Create table projects
CREATE TABLE projects (
	project_id   UUID      NOT NULL,
	owner_id     UUID      NOT NULL,
	name         TEXT      NOT NULL,
	date_created TIMESTAMP NOT NULL,
	date_updated TIMESTAMP NOT NULL,

	PRIMARY KEY (project_id),
	UNIQUE (project_id, owner_id),
	FOREIGN KEY (owner_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX projects_owner_date_idx ON projects (owner_id, date_created DESC);

-- Version: 1.12
-- Description: A file and a job belong to one project owned by the same user
ALTER TABLE artifacts ADD COLUMN project_id UUID NULL;
ALTER TABLE pipeline_jobs ADD COLUMN project_id UUID NULL;

ALTER TABLE artifacts
	ADD CONSTRAINT artifacts_project_owner_fkey
	FOREIGN KEY (project_id, owner_id) REFERENCES projects (project_id, owner_id);

ALTER TABLE pipeline_jobs
	ADD CONSTRAINT pipeline_jobs_project_owner_fkey
	FOREIGN KEY (project_id, owner_id) REFERENCES projects (project_id, owner_id);

CREATE INDEX artifacts_project_date_idx ON artifacts (project_id, date_created DESC);
CREATE INDEX pipeline_jobs_project_date_idx ON pipeline_jobs (project_id, date_created DESC);

-- Version: 1.13
-- Description: A file and a job must name their project
ALTER TABLE artifacts ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE pipeline_jobs ALTER COLUMN project_id SET NOT NULL;

-- Version: 1.14
-- Description: Drop the unused homes table
DROP TABLE homes;

-- Version: 1.15
-- Description: Drop the unused products table and view
DROP VIEW view_products;
DROP TABLE products;

-- Version: 1.16
-- Description: Record the step a job stopped on. Execution-only statuses are not stored.
ALTER TABLE pipeline_jobs ADD COLUMN stopped_step TEXT NOT NULL DEFAULT '';
UPDATE pipeline_jobs SET status = 'canceled' WHERE status IN ('timed_out', 'terminated');
UPDATE pipeline_jobs SET status = 'failed' WHERE status = 'unknown';

-- Version: 1.17
-- Description: Name the closed activity type and the failure message.
ALTER TABLE pipeline_jobs RENAME COLUMN stopped_step TO stopped_activity_type;
ALTER TABLE pipeline_jobs RENAME COLUMN error_message TO failure_message;

-- Version: 1.18
-- Description: Record each pipeline step on the job. The closed activity type is no longer the progress record.
CREATE TABLE pipeline_job_steps (
    job_id       UUID             NOT NULL,
    step         TEXT             NOT NULL,
    status       TEXT             NOT NULL,
    message      TEXT             NULL,
    progress     DOUBLE PRECISION NULL,
    date_updated TIMESTAMP        NOT NULL,

    PRIMARY KEY (job_id, step),
    FOREIGN KEY (job_id) REFERENCES pipeline_jobs(job_id) ON DELETE CASCADE
);

INSERT INTO pipeline_job_steps (job_id, step, status, date_updated)
SELECT job_id, 'extract_audio', 'succeeded', date_updated
FROM pipeline_jobs
WHERE audio_file_id IS NOT NULL
ON CONFLICT DO NOTHING;

INSERT INTO pipeline_job_steps (job_id, step, status, date_updated)
SELECT job_id, 'transcribe', 'succeeded', date_updated
FROM pipeline_jobs
WHERE transcript_file_id IS NOT NULL
ON CONFLICT DO NOTHING;

INSERT INTO pipeline_job_steps (job_id, step, status, date_updated)
SELECT job_id, 'extract_keyframes', 'succeeded', date_updated
FROM pipeline_jobs
WHERE key_frames_file_id IS NOT NULL
ON CONFLICT DO NOTHING;

INSERT INTO pipeline_job_steps (job_id, step, status, date_updated)
SELECT job_id, 'compose_export', 'succeeded', date_updated
FROM pipeline_jobs
WHERE document_file_id IS NOT NULL
ON CONFLICT DO NOTHING;

INSERT INTO pipeline_job_steps (job_id, step, status, date_updated)
SELECT job_id, 'resolve_source', 'succeeded', date_updated
FROM pipeline_jobs
WHERE audio_file_id IS NOT NULL OR status = 'completed'
ON CONFLICT DO NOTHING;

INSERT INTO pipeline_job_steps (job_id, step, status, date_updated)
SELECT job_id, 'extract_keyframes', 'skipped', date_updated
FROM pipeline_jobs
WHERE status = 'completed' AND key_frames_file_id IS NULL
ON CONFLICT DO NOTHING;

ALTER TABLE pipeline_jobs DROP COLUMN stopped_activity_type;

-- Version: 1.19
-- Description: Append pipeline facts to the message store and drop the step rows.
DO $$
DECLARE
    rec record;
    produced uuid;
BEGIN
    FOR rec IN
        SELECT s.job_id, s.step, s.status, s.message, s.progress,
               j.project_id, j.owner_id,
               j.audio_file_id, j.transcript_file_id, j.key_frames_file_id, j.document_file_id
        FROM pipeline_job_steps s
        JOIN pipeline_jobs j ON j.job_id = s.job_id
        ORDER BY s.job_id,
            CASE s.step
                WHEN 'resolve_source' THEN 1
                WHEN 'extract_audio' THEN 2
                WHEN 'transcribe' THEN 3
                WHEN 'extract_keyframes' THEN 4
                WHEN 'compose_export' THEN 5
                ELSE 9
            END,
            s.date_updated
    LOOP
        produced := CASE rec.step
            WHEN 'extract_audio' THEN rec.audio_file_id
            WHEN 'transcribe' THEN rec.transcript_file_id
            WHEN 'extract_keyframes' THEN rec.key_frames_file_id
            WHEN 'compose_export' THEN rec.document_file_id
            ELSE NULL
        END;

        PERFORM message_store.write_message(
            gen_random_uuid()::text,
            'job-' || rec.job_id::text,
            'Fact',
            jsonb_strip_nulls(jsonb_build_object(
                'step', rec.step,
                'state', rec.status,
                'progress', rec.progress,
                'message', NULLIF(rec.message, ''),
                'fileId', produced
            )),
            jsonb_build_object(
                'projectId', rec.project_id::text,
                'ownerId', rec.owner_id::text
            )
        );
    END LOOP;

    FOR rec IN
        SELECT j.job_id, j.project_id, j.owner_id, j.status, j.failure_message
        FROM pipeline_jobs j
        WHERE j.status IN ('failed', 'canceled')
          AND NOT EXISTS (
              SELECT 1
              FROM pipeline_job_steps s
              WHERE s.job_id = j.job_id AND s.status = j.status
          )
    LOOP
        PERFORM message_store.write_message(
            gen_random_uuid()::text,
            'job-' || rec.job_id::text,
            'Fact',
            jsonb_strip_nulls(jsonb_build_object(
                'state', rec.status,
                'message', NULLIF(rec.failure_message, '')
            )),
            jsonb_build_object(
                'projectId', rec.project_id::text,
                'ownerId', rec.owner_id::text
            )
        );
    END LOOP;
END $$;

DROP TABLE pipeline_job_steps;

-- Version: 1.20
-- Description: Create table runs. A run is a workflow of any kind. The spec columns are what the user wants and the status columns are what the controller observed.
CREATE TABLE runs (
	run_id               UUID      NOT NULL,
	kind                 TEXT      NOT NULL,
	owner_id             UUID      NOT NULL,
	project_id           UUID      NOT NULL,
	idempotency_key_hash TEXT      NULL,
	generation           BIGINT    NOT NULL,

	input                JSONB     NOT NULL DEFAULT '{}'::jsonb,
	suspend              BOOLEAN   NOT NULL DEFAULT FALSE,
	cancel               BOOLEAN   NOT NULL DEFAULT FALSE,
	attempt              INT       NOT NULL DEFAULT 1,
	resume_from          TEXT      NOT NULL DEFAULT 'start',

	phase                TEXT      NOT NULL DEFAULT 'pending',
	observed_generation  BIGINT    NOT NULL DEFAULT 0,
	observed_attempt     INT       NOT NULL DEFAULT 0,
	execution_id         TEXT      NOT NULL DEFAULT '',
	reason               TEXT      NOT NULL DEFAULT '',
	message              TEXT      NOT NULL DEFAULT '',
	steps                JSONB     NOT NULL DEFAULT '{}'::jsonb,
	result               JSONB     NOT NULL DEFAULT 'null'::jsonb,
	cancel_requested_at  TIMESTAMP NULL,
	date_closed          TIMESTAMP NULL,

	date_created         TIMESTAMP NOT NULL,
	date_updated         TIMESTAMP NOT NULL,

	PRIMARY KEY (run_id),
	FOREIGN KEY (owner_id) REFERENCES users(user_id) ON DELETE CASCADE,
	FOREIGN KEY (project_id, owner_id) REFERENCES projects (project_id, owner_id),
	UNIQUE (owner_id, idempotency_key_hash),
	CHECK (generation >= 1),
	CHECK (attempt >= 1),
	CHECK (observed_attempt BETWEEN 0 AND attempt),
	CHECK (observed_generation BETWEEN 0 AND generation),
	CHECK (resume_from IN ('start', 'failure')),
	CHECK (phase IN ('pending', 'running', 'paused', 'succeeded', 'failed', 'canceled', 'terminated'))
);

CREATE INDEX runs_project_date_idx ON runs (project_id, date_created DESC);
CREATE INDEX runs_owner_date_idx ON runs (owner_id, date_created DESC);
CREATE INDEX runs_unsettled_idx ON runs (date_updated)
	WHERE phase IN ('pending', 'running', 'paused') OR observed_attempt < attempt;

-- Version: 1.21
-- Description: Create table run_events. An append-only record of commands and observed changes. Nothing computes state from it.
CREATE TABLE run_events (
	event_id     BIGINT    GENERATED ALWAYS AS IDENTITY,
	run_id       UUID      NOT NULL,
	actor_id     UUID      NULL,
	type         TEXT      NOT NULL,
	message      TEXT      NOT NULL DEFAULT '',
	date_created TIMESTAMP NOT NULL,

	PRIMARY KEY (event_id),
	FOREIGN KEY (run_id) REFERENCES runs(run_id) ON DELETE CASCADE
);

CREATE INDEX run_events_run_idx ON run_events (run_id, event_id);

-- Version: 1.22
-- Description: Drop the pipeline job table and the message store. A conversion is now a run of the video-to-document kind.
DROP TABLE pipeline_jobs;
DROP SCHEMA IF EXISTS message_store CASCADE;

-- Version: 1.23
-- Description: Announce run changes on the run_changed channel so the controller hears of them without polling. A change of spec and a change of any step state are announced. A change of status alone, or of how far a step has got, is not.
CREATE FUNCTION runs_announce() RETURNS trigger AS $$
BEGIN
	IF TG_OP = 'INSERT' OR NEW.generation <> OLD.generation THEN
		PERFORM pg_notify('run_changed', NEW.run_id::text || ' spec');
	ELSIF (SELECT jsonb_object_agg(key, value -> 'state') FROM jsonb_each(NEW.steps))
		IS DISTINCT FROM
		(SELECT jsonb_object_agg(key, value -> 'state') FROM jsonb_each(OLD.steps)) THEN
		PERFORM pg_notify('run_changed', NEW.run_id::text || ' step');
	END IF;
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER runs_announce
	AFTER INSERT OR UPDATE ON runs
	FOR EACH ROW EXECUTE FUNCTION runs_announce();

-- Version: 1.24
-- Description: Name the column for what it holds. A run is one execution of a workflow.
ALTER TABLE runs RENAME COLUMN kind TO workflow;
