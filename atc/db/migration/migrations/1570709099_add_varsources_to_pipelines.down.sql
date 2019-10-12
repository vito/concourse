BEGIN;
  ALTER TABLE pipelines DROP COLUMN var_sources;
COMMIT;
