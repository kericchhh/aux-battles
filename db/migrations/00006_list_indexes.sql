-- +goose Up
CREATE INDEX battles_host_idx ON battles (host_id);
CREATE INDEX battles_guest_idx ON battles (guest_id);
CREATE INDEX songs_ready_recent_idx ON songs (created_at DESC, id DESC) WHERE status = 'READY';

-- +goose Down
DROP INDEX songs_ready_recent_idx;
DROP INDEX battles_guest_idx;
DROP INDEX battles_host_idx;
