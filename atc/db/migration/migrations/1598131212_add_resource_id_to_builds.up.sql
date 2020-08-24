BEGIN;
  ALTER TABLE builds
  ADD COLUMN resource_id integer REFERENCES resources (id) ON DELETE CASCADE;
COMMIT;
