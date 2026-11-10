# Development

Run these commands from the repository root. The examples use a POSIX shell.

## Run Go on your host

Install the Go version listed in `go.mod`, Docker with Compose, FFmpeg, Python
3.11, and
[goose](https://github.com/pressly/goose) for database migrations. From the
repository root:

```sh
go install github.com/pressly/goose/v3/cmd/goose@v3.28.0
docker compose up -d --wait db
export DATABASE_URL='postgres://aux:aux_dev@127.0.0.1:5433/aux_battles?sslmode=disable'
"$(go env GOPATH)/bin/goose" -dir db/migrations postgres "$DATABASE_URL" up
go run ./cmd/api
```

In a second terminal, using the same `DATABASE_URL`:

```sh
python3.11 -m venv .venv-worker
.venv-worker/bin/python -m pip install torch==2.11.0 --index-url https://download.pytorch.org/whl/cpu
.venv-worker/bin/python -m pip install numpy==2.3.5 demucs==4.1.0
export DEMUCS_BIN="$PWD/.venv-worker/bin/demucs"
go run ./cmd/worker
```

The worker also listens on `:9091` for metrics. Set `WORKER_METRICS_ADDR` to a
different address if that port is occupied; for example,
`WORKER_METRICS_ADDR=127.0.0.1:9191 go run ./cmd/worker` keeps it on the host.

The worker keeps only the last 2,000 bytes of FFmpeg and Demucs output for
failure diagnostics. On Linux, macOS, and FreeBSD, cancellation terminates the
command's process group, including descendants that remain in that group.
Other platforms use Go's direct-process cancellation. Waiting on inherited
output pipes is bounded by two seconds after cancellation or process exit.

While processing, the worker renews the job lease every minute. Each renewal
query has a five-second deadline. A renewal failure cancels processing, removes
that attempt's output, and retains the input so the job can be recovered after
lease expiry. If the outcome of the final database update is unknown, the
worker retains both the input and published output to avoid deleting files
that the database may already reference.

Internally, worker iterations distinguish idle polling, completed processing,
scheduled retries, permanent failures, polling errors, and unrecorded attempts.
A successfully recorded retry or permanent failure is an outcome rather than
an operational error; the processing cause remains in the logs. Attempt counters
retain their `succeeded` and `failed` labels.

Both processes use `./storage` by default. Set the same `MEDIA_ROOT` for both
when using another directory. Uploads create queued jobs. The worker extracts
the selected 12-second clip, creates four stems, and changes the song status to
`READY` after all files are present. A failed job makes up to three attempts;
expired leases let another worker resume after a crash. Only `READY` songs
appear in the catalog. Each processing attempt is limited to 30 minutes by
default; set `WORKER_ATTEMPT_TIMEOUT` to a positive Go duration such as `45m`
to change it. The worker also checks the media volume at startup and hourly,
removing old abandoned attempts and inputs while preserving active attempts
and files referenced by songs.

Signed-in clients can fetch a ready song at `GET /songs/{id}/audio/{part}`.
`part` is `full`, `drums`, `bass`, `melody`, or `vocals`. The response is MP3
audio and supports HTTP byte ranges.

If `goose` is not found after installation, add `$(go env GOPATH)/bin` to your
`PATH`.

When serving over HTTPS, set `APP_ENV=production` to issue Secure session
cookies with the `__Host-` prefix. Sessions last seven days; the API deletes
expired sessions at startup and hourly afterwards.

On `SIGTERM` or `Ctrl-C`, the API stops accepting connections and gives
in-flight requests up to eight seconds to finish, then closes any remaining
connections, interrupting slow uploads or downloads. A second signal exits
immediately. Requests must arrive within 30 seconds and responses within
60 seconds, except song uploads and audio downloads, which may take up to five
minutes.

## Tests

Run `go test ./...`. To include PostgreSQL integration tests, start the Compose
database and create two separate test databases so the upload and worker suites
cannot claim each other's jobs. Create the databases once, then migrate them
before testing. Install the PostgreSQL client tools (`createdb`) and goose
before running these commands:

```sh
docker compose up -d --wait db
PGPASSWORD=aux_dev createdb -h 127.0.0.1 -p 5433 -U aux aux_battles_server_test
PGPASSWORD=aux_dev createdb -h 127.0.0.1 -p 5433 -U aux aux_battles_worker_test
export TEST_DATABASE_URL='postgres://aux:aux_dev@127.0.0.1:5433/aux_battles_server_test?sslmode=disable'
export TEST_WORKER_DATABASE_URL='postgres://aux:aux_dev@127.0.0.1:5433/aux_battles_worker_test?sslmode=disable'
"$(go env GOPATH)/bin/goose" -dir db/migrations postgres "$TEST_DATABASE_URL" up
"$(go env GOPATH)/bin/goose" -dir db/migrations postgres "$TEST_WORKER_DATABASE_URL" up
go test ./...
```

## Battle lobbies

Battle routes require a valid session cookie. Create an account at `POST /users`
and log in at `POST /users/login` to obtain one. Use separate cookie jars for
host and guest accounts. With those jars saved as `host.cookies` and
`guest.cookies`, create and join a lobby:

```sh
curl -b host.cookies -H 'Content-Type: application/json' \
  -d '{"rounds":7}' http://localhost:8080/battles
curl -b host.cookies http://localhost:8080/battles/BATTLE_ID
curl -b guest.cookies -H 'Content-Type: application/json' \
  -d '{"inviteCode":"ABC123"}' http://localhost:8080/battles/join
```

Replace `ABC123` with the invite code from the host's battle view, and
`BATTLE_ID` with the ID returned when creating the battle. Omitting `rounds`
from the JSON object defaults to five; the allowed range is one to seven.
Joining creates all rounds and changes the lobby from `PENDING` to `SELECTING`
in one transaction. Only one guest can join; self-joins and subsequent joins
return `409`. An unknown invite code returns `404`. Battle reads return `404`
for both missing battles and nonparticipants, and use `Cache-Control: no-store`.
The view follows the reference client's field names and excludes song answers.
Before reusing the reference frontend, reduce its round selector maximum from
ten to seven to match this backend.

Each player locks an ordered lineup using
`PUT /battles/{battleId}/lineup`. For a two-round battle, the body is:

```json
{"songIds":["FIRST_SONG_UUID","SECOND_SONG_UUID"]}
```

Replace the placeholders with IDs from the curated catalog. Only upload and
catalog tracks whose permissions cover storage, processing, and game playback.
`READY` indicates successful processing, not verification of those permissions.

Choose exactly one distinct song per round. Songs must be `READY` and all five
audio files must be readable. The order of IDs determines the round order.
Both players may choose from the same catalog, but neither can repeat a song
within their own lineup. Once submitted, a lineup cannot be changed.

The first successful submission returns `WAITING_ON_OPPONENT`. Once both lineups
are locked, the response is `BATTLE_STARTED`, the battle becomes `ONGOING`, and
round one becomes `GUESSING`. These state changes are committed together.
Subsequent submissions return `409`; nonparticipants receive `404`. Battle views
show each player's lineup readiness without exposing selected song IDs.
Live updates remain a subsequent step; poll the battle view to read progress.

## Playing rounds

Participants listen through `GET /rounds/{roundId}/audio` and submit guesses to
`POST /rounds/{roundId}/guess`. Take the active round ID from the battle view:

```sh
curl -b host.cookies http://localhost:8080/rounds/ROUND_ID/audio -o round.mp3
curl -b host.cookies -H 'Content-Type: application/json' \
  -d '{"guess":"Song title","expectedAttempt":1}' \
  http://localhost:8080/rounds/ROUND_ID/guess
```

The audio route chooses the opponent's selection and the caller's current stage;
clients cannot select a later stage through this route. It supports byte ranges
and sends `Cache-Control: private, no-store`. Nonparticipants receive `404`.
Future, previous, and finished-battle rounds return `409` to participants.
A player who finishes first can replay their last unlocked stage while waiting.

Each wrong guess unlocks the next stage. A correct answer, or four incorrect
answers, finishes the player's round. Both players must finish before the next
round starts. Completing the final round records the winner, or a draw when
scores are equal. The maximum remains seven rounds.

| Stage | Base points for a correct answer |
| --- | ---: |
| Drums | 100 |
| Bass | 75 |
| Melody | 50 |
| Full clip | 20 |

A correct answer adds 10 points per preceding consecutive round answered
correctly, capped at 20 bonus points. Three consecutive drums answers score
100, 110, and 120. Four wrong guesses score zero and reset that player's streak.
The bonus applies only to a correct answer; wrong guesses within a round do not
reset the streak unless the player exhausts all four attempts. Guess responses
separate `basePoints`, `streakBonus`, and their sum, `pointsAwarded`.

Send `expectedAttempt` from 1 to 4, using `myAttempts + 1` from the battle view.
Concurrent duplicates and stale attempt numbers return `409`; refresh the view
before submitting again. Guesses, scores, round completion, and battle progress
commit together. If a response is lost, refresh rather than assuming the request
failed: the database may already have committed it.

Guesses must contain a title of at most 255 characters. Matching ignores case,
accents, punctuation, bracketed annotations, and trailing `feat.`, `ft.` or
`featuring` credits; a bare `feat` or `ft` word is part of the title. A credit
is kept when removing it would leave fewer than five normalized characters.
Non-Latin letters are retained. Titles shorter than five normalized characters
require an exact match; longer titles allow an edit distance of at most 15% of
the normalized title length, rounded down, with a minimum of one. Empty
normalized guesses are rejected. Responses exclude the answer, opponent song
IDs, and storage paths.

The authenticated catalog audio route remains available for browsing curated
tracks. Round-level stage gating does not prevent someone from identifying a
song using the catalog or an external recognition tool; it is not an anti-cheat
system.

### Scoring migration

Apply `00004_round_scoring.sql` before running this gameplay implementation. It
preserves existing scores, raises the per-round ceiling to 120, and permits the
base-plus-bonus guess scores. Existing battle views can still read these scores.
Rollback restores the old checks only when every stored score satisfies them.
It refuses incompatible scores atomically without changing or deleting data;
inspect and decide how to handle affected matches before retrying rollback:

```sql
SELECT id, host_points, guest_points FROM rounds
WHERE host_points > 100 OR guest_points > 100;
SELECT id, points_awarded FROM guesses
WHERE points_awarded NOT IN (0, 20, 50, 75, 100);
```

Do not roll back the schema while the new guess handlers are still accepting
requests.

Migration `00003_battle_round_limit.sql` enforces the seven-round limit for
existing databases. It fails without converting or deleting data if an existing
battle exceeds seven rounds. Inspect such battles before migrating:

```sql
SELECT id, status, round_count FROM battles WHERE round_count > 7;
```

Resolve those records deliberately before rerunning migrations. The migration's
rollback removes the new constraint and restores the previous maximum of ten.
The PostgreSQL battle tests use `TEST_DATABASE_URL`; the full integration suite
still requires the separate worker database described above.
