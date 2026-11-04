-- name: ListSongs :many
SELECT id::text AS id, title, artist
FROM songs
WHERE status = 'READY'
ORDER BY created_at DESC;

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
