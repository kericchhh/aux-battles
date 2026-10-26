-- +goose Up
CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    username text NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL,
    role text NOT NULL DEFAULT 'USER'
        CHECK (role IN ('USER', 'ADMIN')),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_username_unique ON users (lower(username));
CREATE UNIQUE INDEX users_email_unique ON users (lower(email));

CREATE TABLE sessions (
    token_hash char(64) PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE songs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title text NOT NULL,
    artist text NOT NULL,
    genre text NOT NULL,
    duration_seconds integer NOT NULL CHECK (duration_seconds > 0),
    status text NOT NULL DEFAULT 'PROCESSING'
        CHECK (status IN ('PROCESSING', 'READY', 'FAILED')),
    processing_error text,
    full_song_path text,
    drums_path text,
    bass_path text,
    melody_path text,
    vocals_path text,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT ready_song_has_audio CHECK (
        status <> 'READY' OR (
            full_song_path IS NOT NULL
            AND drums_path IS NOT NULL
            AND bass_path IS NOT NULL
            AND melody_path IS NOT NULL
            AND vocals_path IS NOT NULL
        )
    )
);

CREATE INDEX songs_catalog_idx ON songs (status, title, id);

CREATE TABLE battles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id uuid NOT NULL REFERENCES users(id),
    guest_id uuid REFERENCES users(id),
    invite_code text NOT NULL UNIQUE
        CHECK (invite_code ~ '^[A-Z0-9]{6}$'),
    status text NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'SELECTING', 'ONGOING', 'FINISHED')),
    round_count integer NOT NULL DEFAULT 5
        CHECK (round_count BETWEEN 1 AND 10),
    current_round integer NOT NULL DEFAULT 1,
    winner_id uuid REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),

    CHECK (guest_id IS NULL OR guest_id <> host_id),
    CHECK (current_round BETWEEN 1 AND round_count),
    CHECK (
        winner_id IS NULL
        OR winner_id = host_id
        OR (guest_id IS NOT NULL AND winner_id = guest_id)
    )
);

CREATE TABLE rounds (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    battle_id uuid NOT NULL REFERENCES battles(id) ON DELETE CASCADE,
    round_number integer NOT NULL CHECK (round_number > 0),

    -- Each player's selection is what their opponent guesses.
    host_song_id uuid REFERENCES songs(id),
    guest_song_id uuid REFERENCES songs(id),

    host_stage text NOT NULL DEFAULT 'DRUM'
        CHECK (host_stage IN ('DRUM', 'BASS', 'MELODY', 'FULL')),
    guest_stage text NOT NULL DEFAULT 'DRUM'
        CHECK (guest_stage IN ('DRUM', 'BASS', 'MELODY', 'FULL')),
    host_points integer NOT NULL DEFAULT 0
        CHECK (host_points BETWEEN 0 AND 100),
    guest_points integer NOT NULL DEFAULT 0
        CHECK (guest_points BETWEEN 0 AND 100),
    host_finished boolean NOT NULL DEFAULT false,
    guest_finished boolean NOT NULL DEFAULT false,
    status text NOT NULL DEFAULT 'WAITING'
        CHECK (status IN ('WAITING', 'GUESSING', 'FINISHED')),

    UNIQUE (battle_id, round_number),
    CHECK (
        status = 'WAITING'
        OR (host_song_id IS NOT NULL AND guest_song_id IS NOT NULL)
    ),
    CHECK (
        status <> 'FINISHED'
        OR (host_finished AND guest_finished)
    )
);

CREATE UNIQUE INDEX rounds_host_song_unique
    ON rounds (battle_id, host_song_id)
    WHERE host_song_id IS NOT NULL;

CREATE UNIQUE INDEX rounds_guest_song_unique
    ON rounds (battle_id, guest_song_id)
    WHERE guest_song_id IS NOT NULL;

CREATE TABLE guesses (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    round_id uuid NOT NULL REFERENCES rounds(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id),
    attempt integer NOT NULL CHECK (attempt BETWEEN 1 AND 4),
    guess text NOT NULL,
    correct boolean NOT NULL,
    points_awarded integer NOT NULL DEFAULT 0
        CHECK (points_awarded IN (0, 20, 50, 75, 100)),
    created_at timestamptz NOT NULL DEFAULT now(),

    UNIQUE (round_id, user_id, attempt)
);

-- +goose Down
DROP TABLE guesses;
DROP TABLE rounds;
DROP TABLE battles;
DROP TABLE songs;
DROP TABLE sessions;
DROP TABLE users;
