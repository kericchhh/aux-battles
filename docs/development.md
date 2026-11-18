# Development

Run these commands from the repository root. They use the default credentials
from `.env.example`.

## Run Go on your host

You need the Go version in `go.mod`, Docker, FFmpeg, Python 3.11,
[goose](https://github.com/pressly/goose) and, for the tests, the PostgreSQL
client tools. Start the database, migrate it and
run the API:

```sh
go install github.com/pressly/goose/v3/cmd/goose@v3.30.1
[ -f .env ] || cp .env.example .env
docker compose up -d --wait db
export DATABASE_URL='postgres://aux:aux_dev@127.0.0.1:5433/aux_battles?sslmode=disable'
"$(go env GOPATH)/bin/goose" -dir db/migrations postgres "$DATABASE_URL" up
go run ./cmd/api
```

In a second terminal, with the same `DATABASE_URL`, install Demucs and run the
worker:

```sh
python3.11 -m venv .venv-worker
.venv-worker/bin/python -m pip install torch==2.11.0 --index-url https://download.pytorch.org/whl/cpu
.venv-worker/bin/python -m pip install numpy==2.3.5 demucs==4.1.0
export DEMUCS_BIN="$PWD/.venv-worker/bin/demucs"
go run ./cmd/worker
```

Both store media in `./storage`; set `MEDIA_ROOT` to change it. The worker
serves metrics on `:9091`; set `WORKER_METRICS_ADDR` to change it.

## Run the web client

With the API on port 8080:

```sh
cd web
pnpm install
pnpm dev
```

Vite serves the client on <http://localhost:5173> and proxies `/api` to the
API; set `API_URL` to use another address. Before pushing, run `pnpm typecheck`,
`pnpm lint`, `pnpm format:check`, `pnpm test` and `pnpm build`.

## Settings

| Variable | Default | Meaning |
| --- | --- | --- |
| `DATABASE_URL` | none | PostgreSQL connection string |
| `MEDIA_ROOT` | `./storage` | Shared media directory for the API and worker |
| `LOG_LEVEL` | `INFO` | Log verbosity |
| `APP_ENV` | none | Set to `production` behind HTTPS for Secure session cookies |
| `WORKER_ATTEMPT_TIMEOUT` | `30m` | Time limit for one processing attempt |

## Tests

`go test ./...` runs the tests that do not need PostgreSQL. To include the
integration tests, create and migrate two test databases, one for the API and
one for the worker, so their suites cannot claim each other's jobs:

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

Read the [migration notes](migrations.md) before applying or rolling back a
migration on a database with data.
