CREATE TABLE IF NOT EXISTS message_store.messages (
  global_position bigserial NOT NULL,
  position bigint NOT NULL,
  time TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() AT TIME ZONE 'utc') NOT NULL,
  stream_name text NOT NULL,
  type text NOT NULL,
  data jsonb,
  metadata jsonb,
  id UUID NOT NULL DEFAULT gen_random_uuid()
);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conrelid = 'message_store.messages'::regclass
      AND contype = 'p'
  ) THEN
    ALTER TABLE message_store.messages
      ADD PRIMARY KEY (global_position) NOT DEFERRABLE INITIALLY IMMEDIATE;
  END IF;
END $$;
