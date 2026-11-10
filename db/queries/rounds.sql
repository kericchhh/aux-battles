-- name: LockRoundBattleForParticipant :one
SELECT b.* FROM battles b JOIN rounds r ON r.battle_id = b.id
WHERE r.id = sqlc.arg(round_id)::uuid
  AND (b.host_id = sqlc.arg(user_id)::uuid OR b.guest_id = sqlc.arg(user_id)::uuid)
FOR UPDATE OF b;

-- name: GetBattleTotals :one
SELECT count(*)::integer AS rounds,
    count(*) FILTER (WHERE status = 'FINISHED')::integer AS finished_rounds,
    COALESCE(sum(host_points), 0)::integer AS host_total,
    COALESCE(sum(guest_points), 0)::integer AS guest_total
FROM rounds WHERE battle_id = $1;

-- name: LockRoundForBattle :one
SELECT * FROM rounds WHERE id = $1 AND battle_id = $2 FOR UPDATE;

-- name: CountRoundGuesses :one
SELECT count(*)::integer FROM guesses WHERE round_id = $1 AND user_id = $2;

-- name: GetRoundAnswer :one
SELECT title FROM songs WHERE id = $1 AND status = 'READY' FOR SHARE;

-- name: InsertRoundGuess :execrows
INSERT INTO guesses(round_id, user_id, attempt, guess, correct, points_awarded)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: UpdateRoundPlayerProgress :one
UPDATE rounds SET
    host_stage = CASE WHEN sqlc.arg(is_host)::boolean THEN sqlc.arg(stage)::text ELSE host_stage END,
    guest_stage = CASE WHEN NOT sqlc.arg(is_host)::boolean THEN sqlc.arg(stage)::text ELSE guest_stage END,
    host_points = CASE WHEN sqlc.arg(is_host)::boolean THEN sqlc.arg(points)::integer ELSE host_points END,
    guest_points = CASE WHEN NOT sqlc.arg(is_host)::boolean THEN sqlc.arg(points)::integer ELSE guest_points END,
    host_finished = CASE WHEN sqlc.arg(is_host)::boolean THEN sqlc.arg(finished)::boolean ELSE host_finished END,
    guest_finished = CASE WHEN NOT sqlc.arg(is_host)::boolean THEN sqlc.arg(finished)::boolean ELSE guest_finished END
WHERE id = sqlc.arg(id)::uuid AND status = 'GUESSING'
  AND NOT CASE WHEN sqlc.arg(is_host)::boolean THEN host_finished ELSE guest_finished END
RETURNING *;

-- name: FinishRound :execrows
UPDATE rounds SET status = 'FINISHED'
WHERE id = $1 AND status = 'GUESSING' AND host_finished AND guest_finished;

-- name: ActivateRound :execrows
UPDATE rounds SET status = 'GUESSING'
WHERE battle_id = $1 AND round_number = $2 AND status = 'WAITING'
  AND host_song_id IS NOT NULL AND guest_song_id IS NOT NULL;

-- name: GetRoundAudioForParticipant :one
SELECT b.status AS battle_status, b.current_round, r.round_number, r.status AS round_status,
    s.status AS song_status,
    COALESCE(CASE (CASE WHEN b.host_id = sqlc.arg(user_id)::uuid THEN r.host_stage ELSE r.guest_stage END)
        WHEN 'DRUM' THEN s.drums_path
        WHEN 'BASS' THEN s.bass_path
        WHEN 'MELODY' THEN s.melody_path
        WHEN 'FULL' THEN s.full_song_path
    END, '')::text AS audio_path
FROM rounds r JOIN battles b ON b.id = r.battle_id
LEFT JOIN songs s ON s.id = CASE WHEN b.host_id = sqlc.arg(user_id)::uuid THEN r.guest_song_id ELSE r.host_song_id END
WHERE r.id = sqlc.arg(round_id)::uuid
  AND (b.host_id = sqlc.arg(user_id)::uuid OR b.guest_id = sqlc.arg(user_id)::uuid);
