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
