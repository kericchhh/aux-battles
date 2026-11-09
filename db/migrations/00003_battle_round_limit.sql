-- +goose Up
-- Existing battles above seven rounds must be resolved before this migration.
ALTER TABLE battles ADD CONSTRAINT battles_round_count_max_seven
    CHECK (round_count BETWEEN 1 AND 7);

-- +goose Down
ALTER TABLE battles DROP CONSTRAINT battles_round_count_max_seven;
