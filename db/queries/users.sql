-- name: CreateUser :one
INSERT INTO users (username, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id, username, email, role, created_at;

-- name: GetUserByIdentifier :one
SELECT id, username, email, password_hash, role
FROM users
WHERE lower(username) = lower(sqlc.arg(identifier)::text)
   OR lower(email) = lower(sqlc.arg(identifier)::text)
LIMIT 1;
