-- +goose Up
ALTER TABLE battles ADD COLUMN guess_seconds integer CHECK (guess_seconds IN (15, 30));
ALTER TABLE rounds
    ADD COLUMN host_stage_started_at timestamptz,
    ADD COLUMN guest_stage_started_at timestamptz;
ALTER TABLE guesses ADD COLUMN timed_out boolean NOT NULL DEFAULT false;
ALTER TABLE guesses ADD CONSTRAINT guesses_timed_out_check
    CHECK (NOT timed_out OR (NOT correct AND points_awarded = 0));

-- +goose StatementBegin
CREATE FUNCTION notify_battle_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM pg_notify('battle_events', NEW.id::text);
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION notify_round_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM pg_notify('battle_events', NEW.battle_id::text);
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER battles_notify_change AFTER INSERT OR UPDATE ON battles
    FOR EACH ROW EXECUTE FUNCTION notify_battle_change();
CREATE TRIGGER rounds_notify_change AFTER INSERT OR UPDATE ON rounds
    FOR EACH ROW EXECUTE FUNCTION notify_round_change();

-- +goose Down
DROP TRIGGER rounds_notify_change ON rounds;
DROP TRIGGER battles_notify_change ON battles;
DROP FUNCTION notify_round_change();
DROP FUNCTION notify_battle_change();
ALTER TABLE guesses DROP CONSTRAINT guesses_timed_out_check;
ALTER TABLE guesses DROP COLUMN timed_out;
ALTER TABLE rounds DROP COLUMN guest_stage_started_at, DROP COLUMN host_stage_started_at;
ALTER TABLE battles DROP COLUMN guess_seconds;
