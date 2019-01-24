-- NO_TRANSACTION
ALTER TYPE volume_state ADD VALUE 'attached';
UPDATE volumes SET state = 'attached' WHERE state = 'created';
