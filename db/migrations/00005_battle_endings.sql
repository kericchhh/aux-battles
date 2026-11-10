-- +goose Up
ALTER TABLE battles
    ADD COLUMN last_activity_at timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN end_reason text CHECK (end_reason IN ('COMPLETED', 'FORFEIT', 'TIMEOUT'));
UPDATE battles SET end_reason = 'COMPLETED' WHERE status = 'FINISHED';
ALTER TABLE battles DROP CONSTRAINT battles_status_check;
ALTER TABLE battles ADD CONSTRAINT battles_status_check
    CHECK (status IN ('PENDING', 'SELECTING', 'ONGOING', 'FINISHED', 'CANCELLED'));
ALTER TABLE battles ADD CONSTRAINT battles_end_state_check CHECK (
    (status IN ('FINISHED', 'CANCELLED')) = (end_reason IS NOT NULL)
    AND (status <> 'CANCELLED' OR winner_id IS NULL)
    AND (status <> 'FINISHED' OR end_reason = 'COMPLETED' OR winner_id IS NOT NULL)
);
CREATE INDEX battles_active_activity_idx ON battles (last_activity_at)
    WHERE status IN ('PENDING', 'SELECTING', 'ONGOING');

-- +goose Down
-- Adding the old status check first makes rollback fail atomically if cancelled battles exist.
ALTER TABLE battles ADD CONSTRAINT battles_status_legacy
    CHECK (status IN ('PENDING', 'SELECTING', 'ONGOING', 'FINISHED'));
DROP INDEX battles_active_activity_idx;
ALTER TABLE battles DROP CONSTRAINT battles_end_state_check;
ALTER TABLE battles DROP CONSTRAINT battles_status_check;
ALTER TABLE battles RENAME CONSTRAINT battles_status_legacy TO battles_status_check;
ALTER TABLE battles DROP COLUMN end_reason, DROP COLUMN last_activity_at;
