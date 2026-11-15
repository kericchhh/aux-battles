# Database unavailable

The Grafana **Database Unavailable** rule evaluates every 30 seconds and fires
when `pg_up{job="postgres"}` stays below 1 for one minute. `postgres_exporter`
sets `pg_up` to 0 when it cannot connect to PostgreSQL. Missing data also
fires the rule, so it covers a stopped exporter or a broken scrape as well as a
stopped database.

While PostgreSQL is down, sign-in, battles and uploads fail, and the worker
cannot claim jobs, so **Worker Loop Unhealthy** usually fires too. The API's
`/healthz` does not check the database and keeps returning 200.

## Check

1. Open [Prometheus targets](http://localhost:9090/targets). If the `postgres`
   target is down, the exporter is the problem; if it is up, query `pg_up` in
   Prometheus to confirm the exporter cannot reach the database.
2. Check the containers and recent logs:

   ```sh
   docker compose --profile observability ps -a db postgres-exporter db-roles
   docker compose logs --tail=100 db
   docker compose --profile observability logs --tail=50 postgres-exporter db-roles
   ```

3. Common causes:
   - `db` is stopped or restarting: the database logs show why, for example a
     full disk or a bad configuration change.
   - The exporter logs `password authentication failed for user "aux_monitor"`:
     `POSTGRES_EXPORTER_PASSWORD` in `.env` no longer matches the role. Run
     `docker compose --profile observability up -d` so `db-roles` sets the
     password again.
   - `db-roles` exited with an error, so the exporter was never started: its
     logs show why, for example an unset `POSTGRES_EXPORTER_PASSWORD` or a
     database that did not become healthy. The rest of the stack keeps running.

## Recover

Start the database after resolving the cause:

```sh
docker compose up -d db
```

An API or worker that was already running reconnects on its own. One that
starts during the outage, for example after a reboot, checks the database
once at startup and exits if it is unreachable; its `restart: unless-stopped`
policy makes Docker start it again until the database is back, so
`docker compose ps -a api worker` may show it restarting. If either is still not running a minute after
the database is healthy, for example because it was stopped by hand, start it:

```sh
docker compose up -d api worker
```

Confirm `pg_up` returns to 1, the rule returns to **Normal** and a resolved
message arrives in Telegram. Check worker logs for errors and verify that songs
queued during the outage are processed.

## Local drill

On a local stack with no upload in progress, stop PostgreSQL:

```sh
docker compose stop db
```

`pg_up` should become 0 within 15 seconds. The rule moves to **Alerting** after
about a minute and a firing message arrives in Telegram about 30 seconds
later. Restore the database with `docker compose start db`, then confirm
`pg_up=1`, **Normal** and a resolved message within about five minutes.
