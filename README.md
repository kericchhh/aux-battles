# aux-battles

Backend rewrite in Go.

## Run locally

Install the Go version listed in `go.mod`, Docker with Compose, and
[goose](https://github.com/pressly/goose) for database migrations. From the
repository root:

```sh
go install github.com/pressly/goose/v3/cmd/goose@v3.28.0
docker compose up -d --wait db
export DATABASE_URL='postgres://aux:aux_dev@127.0.0.1:5433/aux_battles?sslmode=disable'
goose -dir db/migrations postgres "$DATABASE_URL" up
go run ./cmd/api
```

If `goose` is not found after installation, add `$(go env GOPATH)/bin` to your
`PATH`. The API listens on `http://localhost:8080`; check it with
`curl http://localhost:8080/healthz`.

When serving over HTTPS, set `APP_ENV=production` to issue Secure session
cookies with the `__Host-` prefix.

## Tests

Run `go test ./...`. To include the PostgreSQL auth integration test locally,
start and migrate the database as above, then run:

```sh
TEST_DATABASE_URL="$DATABASE_URL" go test ./...
```
