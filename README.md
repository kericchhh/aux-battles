# aux-battles

Backend rewrite in Go.

## Run with Docker

From the repository root:

```sh
docker compose up --build -d
curl http://localhost:8080/healthz
```

Compose starts PostgreSQL, applies migrations, and starts the API. The API is
available on `http://localhost:8080`. Use `docker compose logs -f api` to view
its logs and `docker compose down` to stop it. Uploaded MP3 files are kept in
the `media` volume.

## Run Go on your host

Install the Go version listed in `go.mod`, Docker with Compose, FFmpeg (for
`ffprobe`), and
[goose](https://github.com/pressly/goose) for database migrations. From the
repository root:

```sh
go install github.com/pressly/goose/v3/cmd/goose@v3.28.0
docker compose up -d --wait db
export DATABASE_URL='postgres://aux:aux_dev@127.0.0.1:5433/aux_battles?sslmode=disable'
goose -dir db/migrations postgres "$DATABASE_URL" up
go run ./cmd/api
```

When running on the host, uploaded files go to `./storage` by default. Set
`MEDIA_ROOT` to choose another directory. Uploads currently create queued jobs;
the processing worker has not been implemented yet, so their status remains
`PROCESSING`.

If `goose` is not found after installation, add `$(go env GOPATH)/bin` to your
`PATH`.

When serving over HTTPS, set `APP_ENV=production` to issue Secure session
cookies with the `__Host-` prefix.

## Tests

Run `go test ./...`. To include the PostgreSQL auth integration test locally,
start and migrate the database as above, then run:

```sh
TEST_DATABASE_URL="$DATABASE_URL" go test ./...
```
