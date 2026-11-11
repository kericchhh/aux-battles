-- name: CreateBattle :one
INSERT INTO battles (host_id, invite_code, round_count, guess_seconds)
VALUES (sqlc.arg(host_id)::uuid, sqlc.arg(invite_code)::text, sqlc.arg(round_count)::integer, sqlc.narg(guess_seconds)::integer)
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
UPDATE rounds SET status = 'GUESSING', host_stage_started_at = now(), guest_stage_started_at = now()
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

-- name: GetBattlePlayers :one
SELECT h.username AS host_username, g.username AS guest_username
FROM battles b
JOIN users h ON h.id = b.host_id
LEFT JOIN users g ON g.id = b.guest_id
WHERE b.id = $1;

-- name: ListFinishedRoundSongs :many
SELECT r.round_number,
    hs.title AS host_song_title, hs.artist AS host_song_artist,
    gs.title AS guest_song_title, gs.artist AS guest_song_artist
FROM rounds r
JOIN songs hs ON hs.id = r.host_song_id
JOIN songs gs ON gs.id = r.guest_song_id
WHERE r.battle_id = $1 AND r.status = 'FINISHED'
ORDER BY r.round_number;

-- name: ListBattlesForUser :many
SELECT b.id, b.status, b.round_count, b.current_round, b.invite_code, b.host_id,
    b.winner_id, b.end_reason, b.last_activity_at,
    o.username AS opponent_username,
    coalesce(sum(CASE WHEN b.host_id = sqlc.arg(user_id)::uuid THEN r.host_points ELSE r.guest_points END)
        FILTER (WHERE r.status = 'FINISHED' OR b.status <> 'ONGOING'), 0)::integer AS my_score,
    coalesce(sum(CASE WHEN b.host_id = sqlc.arg(user_id)::uuid THEN r.guest_points ELSE r.host_points END)
        FILTER (WHERE r.status = 'FINISHED' OR b.status <> 'ONGOING'), 0)::integer AS opponent_score
FROM battles b
LEFT JOIN users o ON o.id = CASE WHEN b.host_id = sqlc.arg(user_id)::uuid THEN b.guest_id ELSE b.host_id END
LEFT JOIN rounds r ON r.battle_id = b.id
WHERE b.host_id = sqlc.arg(user_id)::uuid OR b.guest_id = sqlc.arg(user_id)::uuid
GROUP BY b.id, o.username
ORDER BY b.status IN ('PENDING', 'SELECTING', 'ONGOING') DESC, b.last_activity_at DESC, b.id
LIMIT sqlc.arg(max_battles)::integer;

-- name: LockNextExpiredStage :one
SELECT sqlc.embed(b), r.id AS round_id
FROM battles b
JOIN rounds r ON r.battle_id = b.id AND r.round_number = b.current_round
WHERE b.status = 'ONGOING' AND b.guess_seconds IS NOT NULL AND r.status = 'GUESSING'
  AND NOT b.id = ANY(coalesce(sqlc.arg(skip_ids)::uuid[], '{}'))
  AND ((NOT r.host_finished AND r.host_stage_started_at
            + make_interval(secs => b.guess_seconds + sqlc.arg(grace_seconds)::float8) <= now())
    OR (NOT r.guest_finished AND r.guest_stage_started_at
            + make_interval(secs => b.guess_seconds + sqlc.arg(grace_seconds)::float8) <= now()))
ORDER BY least(CASE WHEN NOT r.host_finished THEN r.host_stage_started_at END,
    CASE WHEN NOT r.guest_finished THEN r.guest_stage_started_at END)
    + make_interval(secs => b.guess_seconds)
LIMIT 1
FOR UPDATE OF b SKIP LOCKED;

-- name: GetPlayerStats :one
WITH finished AS (
    SELECT b.winner_id,
        coalesce(sum(CASE WHEN b.host_id = sqlc.arg(user_id)::uuid THEN r.host_points ELSE r.guest_points END), 0) AS score
    FROM battles b
    LEFT JOIN rounds r ON r.battle_id = b.id
    WHERE b.status = 'FINISHED'
      AND (b.host_id = sqlc.arg(user_id)::uuid OR b.guest_id = sqlc.arg(user_id)::uuid)
    GROUP BY b.id
)
SELECT count(*)::integer AS played,
    count(*) FILTER (WHERE winner_id = sqlc.arg(user_id)::uuid)::integer AS wins,
    count(*) FILTER (WHERE winner_id <> sqlc.arg(user_id)::uuid)::integer AS losses,
    count(*) FILTER (WHERE winner_id IS NULL)::integer AS draws,
    coalesce(max(score), 0)::integer AS best_score,
    coalesce(round(avg(score)), 0)::integer AS average_score
FROM finished;
