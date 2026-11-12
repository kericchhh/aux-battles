# aux-battles

[![CI](https://github.com/kericchhh/aux-battles/actions/workflows/ci.yml/badge.svg)](https://github.com/kericchhh/aux-battles/actions/workflows/ci.yml)

A song guessing game, under development: a Go backend and a React web client.
The backend accepts MP3 uploads, queues processing jobs, and serves a short clip
and four separated stems.

The API and worker use PostgreSQL for song state and job coordination. The worker
uses FFmpeg and Demucs to prepare audio. Docker Compose runs the services locally;
an optional observability stack collects metrics and logs with Prometheus,
Grafana, Loki, and Alloy. GitHub Actions defines build, test, and configuration
checks.

## Run with Docker

From the repository root:

```sh
docker compose up --build -d
curl http://localhost:8080/healthz
```

Compose starts PostgreSQL, applies migrations, and starts the API, a separate
song worker and the web client. Open the game at `http://localhost:8000`; the
web container serves the built client with nginx and forwards `/api` to the API
once its health check passes. The API is also available directly on
`http://localhost:8080`. Use
`docker compose logs -f api worker` to view their JSON logs and
`docker compose down` to stop them. Uploaded MP3s, processed clips, and the
Demucs model cache share the `media` volume. The worker runs on CPU; the first
song may take longer while Demucs downloads its model.

## Observe locally

Start the API, worker, Prometheus, Grafana, Loki, and Alloy:

```sh
docker compose --profile observability up --build -d
```

Open [Grafana](http://localhost:3000) for dashboards and logs, or check
[Prometheus targets](http://localhost:9090/targets). See the
[observability guide](docs/observability.md) for queries, configuration,
retention, and local access details.
The [worker queue runbook](docs/runbooks/song-queue-unhealthy.md) includes a
local database outage drill.

## Development and tests

Run `go test ./...` for the Go tests that do not require PostgreSQL. See the
[development guide](docs/development.md) for running the API and worker on your
host and setting up the PostgreSQL integration tests. The
[API guide](docs/api.md) covers routes and gameplay, the
[worker guide](docs/worker.md) covers song processing, and the
[migration notes](docs/migrations.md) cover applying and rolling back schema
changes.

## CI

Pushes and pull requests to `main`, and manual runs from the Actions tab, run
the `go`, `web`, `images` and `observability` jobs; pushes to `main` also run
`publish`. A newer push to a pull request cancels its older run. Runs on `main`
queue instead, and only the latest queued push is kept.

| Job | What it checks |
| --- | --- |
| `go` | `gofmt`, `sqlc diff`, build, `go vet`, `govulncheck`, and the full test suite with the race detector against two migrated PostgreSQL test databases |
| `web` | Typecheck, lint, formatting, tests and production build of the client in `web/` |
| `images` | Builds the API, worker and web images and scans each with Trivy, failing on fixable high or critical vulnerabilities |
| `observability` | Compose configuration, Grafana dashboard JSON syntax, and Prometheus, Loki and Alloy configuration |
| `publish` | Pushes to `main` only, after `go`, `web` and `images` pass: rebuilds the three images from the build cache and pushes them to GitHub Container Registry |

Published images are `ghcr.io/kericchhh/aux-battles-api`, `aux-battles-worker`
and `aux-battles-web`, tagged with the short commit SHA (`sha-<commit>`) and
`latest`. Only `publish` can write packages; every other job has read-only
access. The observability checks validate configuration files; they do not
start the monitoring stack or test dashboard queries against live metrics.
