# Worker unavailable

**Worker Unavailable** fires when Prometheus has not been able to scrape the
worker for two minutes. Songs are not processed while it fires.

## Check

```sh
docker compose ps worker db
docker compose logs --tail=100 worker
```

The logs usually show the cause, for example a database it cannot reach. If
the worker is running but the target is down, check the error on
[Prometheus targets](http://localhost:9090/targets).

## Recover

```sh
docker compose --profile observability up -d worker
```

Confirm the target is up, the logs show `song worker started`, and a song in
progress finishes.

## Drill

Run `docker compose stop worker`. The alert fires after about two minutes and
reaches Telegram 30 seconds later. Run `docker compose start worker` and wait
for the resolved message.
