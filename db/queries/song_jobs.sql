-- name: EnqueueSongJob :exec
INSERT INTO song_jobs (song_id, input_path, clip_start_seconds)
VALUES ($1, $2, $3);

-- name: ClaimSongJob :one
WITH candidate AS (
    SELECT song_id
    FROM song_jobs
    WHERE attempt_count < sqlc.arg(max_attempts)::integer
      AND ((state = 'QUEUED' AND available_at <= now())
        OR (state = 'RUNNING' AND lease_expires_at <= now()))
    ORDER BY available_at, created_at, song_id
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
UPDATE song_jobs AS job
SET state = 'RUNNING',
    attempt_count = job.attempt_count + 1,
    lease_expires_at = now() + interval '10 minutes',
    started_at = coalesce(job.started_at, now()),
    updated_at = now()
FROM candidate
WHERE job.song_id = candidate.song_id
RETURNING job.song_id::text AS song_id, job.input_path,
    job.clip_start_seconds, job.attempt_count;

-- name: CompleteSongJob :one
WITH completed AS (
    UPDATE song_jobs
    SET state = 'SUCCEEDED', lease_expires_at = NULL,
        finished_at = now(), updated_at = now()
    WHERE song_id = sqlc.arg(song_id)::uuid
      AND attempt_count = sqlc.arg(attempt_count)
      AND state = 'RUNNING'
    RETURNING song_id
)
UPDATE songs AS song
SET status = 'READY', processing_error = NULL,
    full_song_path = sqlc.arg(full_song_path)::text,
    drums_path = sqlc.arg(drums_path)::text,
    bass_path = sqlc.arg(bass_path)::text,
    melody_path = sqlc.arg(melody_path)::text,
    vocals_path = sqlc.arg(vocals_path)::text
FROM completed
WHERE song.id = completed.song_id
RETURNING song.id::text AS id;

-- name: RenewSongJob :one
UPDATE song_jobs
SET lease_expires_at = now() + interval '10 minutes', updated_at = now()
WHERE song_id = sqlc.arg(song_id)::uuid
  AND attempt_count = sqlc.arg(attempt_count)
  AND state = 'RUNNING'
  AND lease_expires_at > now()
RETURNING song_id::text AS song_id;

-- name: RetrySongJob :one
UPDATE song_jobs
SET state = 'QUEUED', lease_expires_at = NULL,
    available_at = now() + (sqlc.arg(delay_seconds)::integer * interval '1 second'),
    updated_at = now()
WHERE song_id = sqlc.arg(song_id)::uuid
  AND attempt_count = sqlc.arg(attempt_count)
  AND state = 'RUNNING'
RETURNING song_id::text AS song_id;

-- name: FailSongJob :one
WITH failed AS (
    UPDATE song_jobs
    SET state = 'FAILED', lease_expires_at = NULL,
        finished_at = now(), updated_at = now()
    WHERE song_id = sqlc.arg(song_id)::uuid
      AND attempt_count = sqlc.arg(attempt_count)
      AND state = 'RUNNING'
    RETURNING song_id
)
UPDATE songs AS song
SET status = 'FAILED', processing_error = sqlc.arg(processing_error)::text
FROM failed
WHERE song.id = failed.song_id
RETURNING song.id::text AS id;

-- name: FailExpiredSongJobs :many
WITH expired AS (
    UPDATE song_jobs
    SET state = 'FAILED', lease_expires_at = NULL,
        finished_at = now(), updated_at = now()
    WHERE state = 'RUNNING'
      AND attempt_count >= sqlc.arg(max_attempts)::integer
      AND lease_expires_at <= now()
    RETURNING song_id, input_path
)
UPDATE songs AS song
SET status = 'FAILED',
    processing_error = sqlc.arg(processing_error)::text
FROM expired
WHERE song.id = expired.song_id
RETURNING expired.song_id::text AS song_id, expired.input_path;

-- name: ListSongJobMedia :many
SELECT song_id::text AS song_id, input_path, state, attempt_count, lease_expires_at
FROM song_jobs;

-- name: ListSongOutputPaths :many
SELECT full_song_path, drums_path, bass_path, melody_path, vocals_path
FROM songs;
