-- name: CreateBattle :one
INSERT INTO battles (host_id, invite_code, round_count)
VALUES ($1, $2, $3)
ON CONFLICT (invite_code) DO NOTHING
RETURNING id;

-- name: GetBattleByInvite :one
SELECT id FROM battles WHERE invite_code = $1;

-- name: JoinBattle :one
UPDATE battles
SET guest_id = sqlc.arg(user_id)::uuid, status = 'SELECTING', last_activity_at = now()
WHERE id = sqlc.arg(id)::uuid AND status = 'PENDING'
  AND guest_id IS NULL AND host_id <> sqlc.arg(user_id)::uuid
RETURNING id, round_count;

-- name: CreateBattleRounds :execrows
INSERT INTO rounds (battle_id, round_number)
SELECT sqlc.arg(battle_id)::uuid, n
FROM generate_series(1, sqlc.arg(round_count)::integer) AS n;

-- name: GetBattleForParticipant :one
SELECT * FROM battles
WHERE id = sqlc.arg(id)::uuid
  AND (host_id = sqlc.arg(user_id)::uuid OR guest_id = sqlc.arg(user_id)::uuid);

-- name: ListBattleRounds :many
SELECT r.*, (SELECT count(*) FROM guesses g
    WHERE g.round_id = r.id AND g.user_id = sqlc.arg(user_id)::uuid)::integer AS my_attempts
FROM rounds r WHERE r.battle_id = sqlc.arg(battle_id)::uuid
ORDER BY r.round_number;

-- name: LockBattleForParticipant :one
SELECT * FROM battles
WHERE id = sqlc.arg(id)::uuid
  AND (host_id = sqlc.arg(user_id)::uuid OR guest_id = sqlc.arg(user_id)::uuid)
FOR UPDATE;

-- name: GetLineupSongs :many
SELECT id, title, status, full_song_path, drums_path, bass_path, melody_path, vocals_path
FROM songs WHERE id = ANY(sqlc.arg(song_ids)::uuid[])
ORDER BY id
FOR SHARE;

-- name: SetHostRoundSong :execrows
UPDATE rounds SET host_song_id = sqlc.arg(song_id)::uuid
WHERE id = sqlc.arg(id)::uuid AND status = 'WAITING' AND host_song_id IS NULL;

-- name: SetGuestRoundSong :execrows
UPDATE rounds SET guest_song_id = sqlc.arg(song_id)::uuid
WHERE id = sqlc.arg(id)::uuid AND status = 'WAITING' AND guest_song_id IS NULL;

-- name: ActivateFirstRound :execrows
UPDATE rounds SET status = 'GUESSING'
WHERE battle_id = $1 AND round_number = 1 AND status = 'WAITING'
  AND host_song_id IS NOT NULL AND guest_song_id IS NOT NULL;

-- name: StartBattle :execrows
UPDATE battles SET status = 'ONGOING'
WHERE id = $1 AND status = 'SELECTING' AND current_round = 1;

-- name: AdvanceBattle :execrows
UPDATE battles SET current_round = current_round + 1
WHERE id = $1 AND status = 'ONGOING' AND current_round = $2 AND current_round < round_count;

-- name: FinishBattle :execrows
UPDATE battles SET status = 'FINISHED', end_reason = 'COMPLETED', winner_id = sqlc.narg(winner_id)::uuid
WHERE id = sqlc.arg(id)::uuid AND status = 'ONGOING' AND current_round = round_count;

-- name: TouchBattle :execrows
UPDATE battles SET last_activity_at = now() WHERE id = $1;

-- name: LockNextInactiveBattle :one
SELECT * FROM battles
WHERE (status = 'PENDING'
        AND last_activity_at < now() - make_interval(secs => sqlc.arg(lobby_timeout_seconds)::float8))
   OR (status IN ('SELECTING', 'ONGOING')
        AND last_activity_at < now() - make_interval(secs => sqlc.arg(game_timeout_seconds)::float8))
ORDER BY last_activity_at
LIMIT 1
FOR UPDATE SKIP LOCKED;

-- name: EndBattle :execrows
UPDATE battles
SET status = sqlc.arg(status)::text, end_reason = sqlc.arg(end_reason)::text, winner_id = sqlc.narg(winner_id)::uuid
WHERE id = sqlc.arg(id)::uuid AND status IN ('PENDING', 'SELECTING', 'ONGOING');
