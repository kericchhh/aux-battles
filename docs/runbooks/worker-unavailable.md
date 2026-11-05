# Worker unavailable

The Grafana **Worker Unavailable** rule evaluates every 30 seconds and fires
when `up{job="worker"}` stays below 1 for two minutes. It also treats missing
query data as alerting. `up` measures whether Prometheus can scrape the worker.
It does not prove that queued songs are being processed.

## Check

1. Open [Prometheus targets](http://localhost:9090/targets) and check whether
   the `worker` target is down. If there is no worker target, check the scrape
   configuration and Prometheus logs; missing data can also trigger this rule.
2. Check the container and recent logs:

   ```sh
   docker compose --profile observability ps worker db
   docker compose --profile observability logs --tail=100 worker
   ```

3. Resolve the startup or connectivity error shown in the logs. If the worker
   is running and the scrape target is still down, check its metrics listener
   at `worker:9091/metrics` from the Compose network and the Prometheus target
   error.

## Recover

Start the worker after resolving the cause:

```sh
docker compose --profile observability up -d worker
```

Confirm the `worker` target returns to `up=1` and the Grafana rule returns to
**Normal**. Check worker logs for `song worker started`. If a song was in
progress, verify its status after the worker has had time to reclaim an expired
lease; a healthy scrape alone does not confirm song processing.

## Local drill

```sh
docker compose stop worker
```

After the next scrape, `up` should be 0. The rule moves from **Pending** to
**Alerting** after about two minutes. Restore the worker with
`docker compose start worker`, then confirm `up=1` and **Normal**.
