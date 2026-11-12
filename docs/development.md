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

If `goose` is not found after installation, add `$(go env GOPATH)/bin` to your
`PATH`.

Both processes store media in `./storage` by default; set the same
`MEDIA_ROOT` for both to use another directory. See the
[worker guide](worker.md) for how songs are processed and how long each
attempt may take, and the [API guide](api.md) for routes and examples.

## Run the web client

The client in `web/` uses Vite, React and pnpm. With the API on port 8080:

```sh
cd web
pnpm install
pnpm dev
```

Vite serves the client on `http://localhost:5173` and proxies `/api` to
`http://localhost:8080`; set `API_URL` to use another address. Run
`pnpm typecheck`, `pnpm lint`, `pnpm format:check`, `pnpm test` and
`pnpm build` before pushing; CI runs the same checks.

In Compose, the `web` service builds the client with `Dockerfile.web` and serves
it with nginx on `127.0.0.1:8000`. nginx forwards `/api/` to the API without
buffering, so live battle updates stream through, and does not expose
`/api/metrics`. Because the API sees nginx as the client, login and registration
rate limits are shared by everyone using the Compose client.

## API settings

When serving over HTTPS, set `APP_ENV=production` to issue Secure session
cookies with the `__Host-` prefix. Sessions last seven days; the API deletes
expired sessions at startup and hourly afterwards.

On `SIGTERM` or `Ctrl-C`, the API stops accepting connections and gives
in-flight requests up to eight seconds to finish, then closes any remaining
connections, interrupting slow uploads or downloads. A second signal exits
immediately. Requests must arrive within 30 seconds and responses within
60 seconds, except song uploads and audio downloads, which may take up to five
minutes, and live update streams, which stay open for up to 30 minutes and
close at once on shutdown.

The API keeps one database connection open for live updates, outside the
regular pool, and reconnects it automatically.

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

The PostgreSQL battle tests use `TEST_DATABASE_URL`; the full integration suite
still requires the separate worker database described above.

Migrations live in `db/migrations`. Read the [migration notes](migrations.md)
before applying or rolling back one on a database that has data.
