DROP INDEX IF EXISTS "tasks_owner_id_INDEX";

ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_owner_id_fkey;
ALTER TABLE tasks DROP COLUMN IF EXISTS owner_id;

DROP TABLE IF EXISTS credential_tokens;

DROP TABLE IF EXISTS accounts;
