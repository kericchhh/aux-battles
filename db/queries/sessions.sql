-- name: CreateSession :exec
INSERT INTO sessions (token_hash, user_id, expires_at)
VALUES ($1, $2, $3);

-- name: GetValidSession :one
SELECT u.id, u.username, u.email, u.role, s.expires_at
FROM sessions AS s
JOIN users AS u ON u.id = s.user_id
WHERE s.token_hash = $1
  AND s.expires_at > now();

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE token_hash = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at <= now();
