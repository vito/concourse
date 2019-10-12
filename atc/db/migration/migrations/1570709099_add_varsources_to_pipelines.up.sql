BEGIN;
  ALTER TABLE pipelines ADD COLUMN var_sources text;
COMMIT;