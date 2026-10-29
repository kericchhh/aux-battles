-- name: EnqueueSongJob :exec
INSERT INTO song_jobs (song_id, input_path, clip_start_seconds)
VALUES ($1, $2, $3);
