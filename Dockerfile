# syntax=docker/dockerfile:1

FROM golang:1.27.1-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
RUN CGO_ENABLED=0 go install github.com/pressly/goose/v3/cmd/goose@v3.28.0

COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates ffmpeg \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /data/media/incoming \
    && chown -R 65532:65532 /data/media

COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /go/bin/goose /usr/local/bin/goose
COPY db/migrations /migrations

ENV MEDIA_ROOT=/data/media
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
