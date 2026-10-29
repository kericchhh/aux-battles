-- +goose Up
CREATE TABLE song_jobs (
    song_id uuid PRIMARY KEY REFERENCES songs(id) ON DELETE CASCADE,
    input_path text NOT NULL CHECK (input_path <> ''),
    clip_start_seconds double precision NOT NULL
        CHECK (clip_start_seconds >= 0 AND clip_start_seconds < 'Infinity'::double precision),
    state text NOT NULL DEFAULT 'QUEUED'
        CHECK (state IN ('QUEUED', 'RUNNING', 'SUCCEEDED', 'FAILED')),
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    available_at timestamptz NOT NULL DEFAULT now(),
    lease_expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    started_at timestamptz,
    finished_at timestamptz
);

CREATE INDEX song_jobs_queued_idx ON song_jobs (available_at, created_at)
    WHERE state = 'QUEUED';

-- +goose Down
DROP TABLE song_jobs;
