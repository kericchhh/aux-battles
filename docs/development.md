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
cookies with the `__Host-` prefix.

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
Guess submission, round audio, and live updates remain subsequent steps.

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
