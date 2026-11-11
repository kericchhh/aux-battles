-- name: ListSongs :many
SELECT id, title, artist, created_at
FROM songs
WHERE status = 'READY'
  AND (sqlc.narg(pattern)::text IS NULL
    OR title ILIKE sqlc.narg(pattern)::text OR artist ILIKE sqlc.narg(pattern)::text)
  AND (created_at, id) < (coalesce(sqlc.narg(before_created_at)::timestamptz, 'infinity'),
    coalesce(sqlc.narg(before_id)::uuid, '00000000-0000-0000-0000-000000000000'))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(max_songs)::integer;

-- name: GetSong :one
SELECT id::text AS id, title, artist
FROM songs
WHERE id = $1::uuid AND status = 'READY';

-- name: GetSongStatus :one
SELECT id::text AS id, title, artist, status, processing_error
FROM songs
WHERE id = $1::uuid;

-- name: GetReadySongAudio :one
SELECT full_song_path, drums_path, bass_path, melody_path, vocals_path
FROM songs
WHERE id = $1::uuid AND status = 'READY';

-- name: CreateSong :one
INSERT INTO songs (title, artist, genre, duration_seconds)
VALUES ($1, $2, $3, $4)
RETURNING id::text AS id, status;
