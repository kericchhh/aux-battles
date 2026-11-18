# Database unavailable

**Database Unavailable** fires when `postgres-exporter` cannot reach
PostgreSQL, or stops reporting, for one minute. Sign-in, battles and uploads
fail, and **Worker Loop Unhealthy** usually fires too. The API's `/healthz`
does not check the database and stays green.

## Check

```sh
docker compose --profile observability ps -a db postgres-exporter db-roles
docker compose logs --tail=100 db
docker compose --profile observability logs --tail=50 postgres-exporter db-roles
```

- **`db` stopped or restarting:** its logs show why, for example a full disk.
- **`password authentication failed for user "aux_monitor"`:** the exporter's
  password changed. Run `docker compose --profile observability up -d` so
  `db-roles` sets it again.

## Recover

```sh
docker compose up -d db
```

The API and worker reconnect on their own; if either was stopped, run
`docker compose up -d api worker`. Confirm `pg_up` is 1 and queued songs are
processed.

## Drill

Run `docker compose stop db`. The alert fires after about a minute. Run
`docker compose start db` and wait for the resolved message.
