-- name: ListSongs :many
SELECT id::text AS id, title, artist
FROM songs
ORDER BY created_at DESC;

-- name: GetSong :one
SELECT id::text AS id, title, artist
FROM songs
WHERE id = $1::uuid;
