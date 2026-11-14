-- CloudChamber additions on top of Message DB 1.3.0.
-- Functions look up `messages` through the session search path. Pin that path
-- on every message_store function so callers do not have to.

DO $$
DECLARE
    sig regprocedure;
BEGIN
    FOR sig IN
        SELECT p.oid::regprocedure
        FROM pg_proc p
        JOIN pg_namespace n ON n.oid = p.pronamespace
        WHERE n.nspname = 'message_store'
    LOOP
        EXECUTE format('ALTER FUNCTION %s SET search_path = message_store, public', sig);
    END LOOP;
END $$;

CREATE INDEX IF NOT EXISTS messages_project_global
    ON message_store.messages ((metadata->>'projectId'), global_position);

CREATE OR REPLACE FUNCTION message_store.notify_new_message()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM pg_notify('message_store', NEW.global_position::text);
    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS messages_notify ON message_store.messages;
CREATE TRIGGER messages_notify
    AFTER INSERT ON message_store.messages
    FOR EACH ROW
    EXECUTE FUNCTION message_store.notify_new_message();
