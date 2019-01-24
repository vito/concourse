BEGIN;
  -- in 4.2.2 or earlier, volumes with no container ID will get GCed
  -- so when downgrading, this is adequate. The state `attached` for
  -- volume_state will be left behind but unused.
  UPDATE volumes SET state = 'created' WHERE state = 'attached';
COMMIT;
