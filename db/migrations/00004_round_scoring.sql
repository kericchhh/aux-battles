-- +goose Up
ALTER TABLE rounds DROP CONSTRAINT rounds_host_points_check;
ALTER TABLE rounds DROP CONSTRAINT rounds_guest_points_check;
ALTER TABLE rounds ADD CONSTRAINT rounds_host_points_check CHECK (host_points BETWEEN 0 AND 120);
ALTER TABLE rounds ADD CONSTRAINT rounds_guest_points_check CHECK (guest_points BETWEEN 0 AND 120);
ALTER TABLE guesses DROP CONSTRAINT guesses_points_awarded_check;
ALTER TABLE guesses ADD CONSTRAINT guesses_points_awarded_check
    CHECK (points_awarded IN (0, 20, 30, 40, 50, 60, 70, 75, 85, 95, 100, 110, 120));

-- +goose Down
-- Adding the old checks first makes rollback fail atomically if newer scores exist.
ALTER TABLE rounds ADD CONSTRAINT rounds_host_points_legacy CHECK (host_points BETWEEN 0 AND 100);
ALTER TABLE rounds ADD CONSTRAINT rounds_guest_points_legacy CHECK (guest_points BETWEEN 0 AND 100);
ALTER TABLE guesses ADD CONSTRAINT guesses_points_awarded_legacy CHECK (points_awarded IN (0, 20, 50, 75, 100));
ALTER TABLE rounds DROP CONSTRAINT rounds_host_points_check;
ALTER TABLE rounds DROP CONSTRAINT rounds_guest_points_check;
ALTER TABLE guesses DROP CONSTRAINT guesses_points_awarded_check;
ALTER TABLE rounds RENAME CONSTRAINT rounds_host_points_legacy TO rounds_host_points_check;
ALTER TABLE rounds RENAME CONSTRAINT rounds_guest_points_legacy TO rounds_guest_points_check;
ALTER TABLE guesses RENAME CONSTRAINT guesses_points_awarded_legacy TO guesses_points_awarded_check;
