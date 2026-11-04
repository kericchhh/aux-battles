# aux-battles

Backend rewrite in Go.

## Run with Docker

From the repository root:

```sh
docker compose up --build -d
curl http://localhost:8080/healthz
```

Compose starts PostgreSQL, applies migrations, and starts the API and a separate
song worker. The API is available on `http://localhost:8080`. Use
`docker compose logs -f api worker` to view their JSON logs and
`docker compose down` to stop them. Uploaded MP3s, processed clips, and the
Demucs model cache share the `media` volume. The worker runs on CPU; the first
song may take longer while Demucs downloads its model.

Log records include the `service` (`api` or `worker`). API responses include an
`X-Request-ID` header; the same ID appears in request completion and handler
error logs. Completion logs record the method, route pattern, status, and
duration in milliseconds. Worker logs use the song ID and attempt to connect
processing events and record durations for clip extraction, Demucs execution,
and output validation and publishing. Both programs default to `INFO` logging.
Set `LOG_LEVEL=ERROR` when starting Go on your host or running
`docker compose up` to show only errors.

## Observe locally

Start the API, worker, Prometheus, and Grafana with the observability profile:

```sh
docker compose --profile observability up --build -d
```

Open [Prometheus targets](http://localhost:9090/targets) to check the `api`,
`worker`, and `prometheus` scrapes. Prometheus collects metrics every 15 seconds.
The API exposes `http://localhost:8080/metrics`; Prometheus reaches the worker's
metrics endpoint at `worker:9091/metrics` on the Compose network.

The metrics endpoints have no authentication. Compose binds the published ports
to localhost and does not publish the worker metrics port. Restrict access to
these endpoints if you run the programs outside this local setup.

Open [Grafana](http://localhost:3000) and sign in with `admin` / `admin`. Grafana loads the Prometheus data source
and the API and Worker Overview dashboard from `observability/grafana/`.
Prometheus samples and Grafana settings persist in the `prometheus_data` and
`grafana_data` volumes.

The API exports `aux_battles_http_requests_total` by route and status and
`aux_battles_http_request_duration_seconds`. The worker exports
`aux_battles_worker_attempts_total` by outcome and
`aux_battles_worker_attempt_duration_seconds`. A failed attempt may be retried,
so the attempt counter does not count distinct failed songs. Attempts are
counted when processing returns; a worker crash is not recorded as a failed
attempt. The dashboard currently shows scrape status and Go process metrics;
the application metrics are available for additional panels.

The dashboard JSON in `observability/grafana/dashboards/` is the provisioned
source. After changing the dashboard in Grafana, export it as a V2 Resource JSON
file to that directory so a fresh Compose run uses the same version. Grafana's
file provider can replace UI changes with the JSON version on restart.

## Run Go on your host

Install the Go version listed in `go.mod`, Docker with Compose, FFmpeg, Python
3.11, and
[goose](https://github.com/pressly/goose) for database migrations. From the
repository root:

```sh
go install github.com/pressly/goose/v3/cmd/goose@v3.28.0
docker compose up -d --wait db
export DATABASE_URL='postgres://aux:aux_dev@127.0.0.1:5433/aux_battles?sslmode=disable'
goose -dir db/migrations postgres "$DATABASE_URL" up
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
before testing:

```sh
PGPASSWORD=aux_dev createdb -h 127.0.0.1 -p 5433 -U aux aux_battles_server_test
PGPASSWORD=aux_dev createdb -h 127.0.0.1 -p 5433 -U aux aux_battles_worker_test
export TEST_DATABASE_URL='postgres://aux:aux_dev@127.0.0.1:5433/aux_battles_server_test?sslmode=disable'
export TEST_WORKER_DATABASE_URL='postgres://aux:aux_dev@127.0.0.1:5433/aux_battles_worker_test?sslmode=disable'
goose -dir db/migrations postgres "$TEST_DATABASE_URL" up
goose -dir db/migrations postgres "$TEST_WORKER_DATABASE_URL" up
go test ./...
```
