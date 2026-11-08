# Song queue unhealthy

Grafana evaluates two worker health rules every 30 seconds:

- **Worker Loop Unhealthy** fires when either the job iteration or the queue
  sample reports failure for two minutes. A successful sample means its database
  query completed; a successful iteration found no job or recorded a job state.
- **Song Queue Waiting Too Long** fires when the oldest claimable job has
  waited more than one hour for five minutes. It includes expired running jobs
  with attempts left; retry jobs whose `available_at` time is still in the future
  are excluded.

**Worker Loop Unhealthy** treats a missing health metric as failure while the
worker is scrapeable. Both rules stay Normal when the worker is absent because
**Worker Unavailable** covers a stopped worker or failed scrape. A long queue
can also mean the worker is busy; the queue alert signals delay, not a specific
failure cause.

## Check

1. In Grafana Explore, query `aux_battles_worker_iteration_operational`,
   `aux_battles_worker_queue_sample_success`,
   `aux_battles_worker_ready_jobs`, and
   `aux_battles_worker_oldest_ready_job_wait_seconds` from Prometheus.
2. Check the worker and database containers and recent errors:

   ```sh
   docker compose --profile observability ps worker db
   docker compose --profile observability logs --tail=100 worker db
   ```

3. If the database is available, inspect queued and running jobs:

   ```sh
   docker compose exec db psql -U aux -d aux_battles -c "SELECT song_id, state, attempt_count, available_at, lease_expires_at FROM song_jobs WHERE state IN ('QUEUED', 'RUNNING') ORDER BY available_at LIMIT 10"
   ```

## Recover

Resolve the database or query error shown in worker logs. The worker backs off
failed job polls for at most 30 seconds and samples the queue every 15 seconds.
Confirm both health metrics return to 1. If the queue is old, check whether a
job is processing, whether leases expire and get reclaimed, and whether new
attempts complete. The queue alert returns to Normal when the oldest ready job
falls below one hour or no ready jobs remain.

## Local drill

On a disposable local stack with no upload in progress, stop PostgreSQL:

```sh
docker compose stop db
```

The worker stays scrapeable, but both health metrics should become 0 and
**Worker Loop Unhealthy** should fire after about two minutes. Restore the
database with `docker compose start db`, then confirm both metrics return to 1
and the alert returns to Normal.
