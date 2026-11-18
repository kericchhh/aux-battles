# Song queue unhealthy

Two alerts cover the worker's job loop. **Worker Unavailable** covers a worker
that is not running.

- **Worker Loop Unhealthy:** the worker is running, but polling for jobs or
  sampling the queue has failed for two minutes, usually because of the
  database.
- **Song Queue Waiting Too Long:** the oldest ready job has waited over an
  hour, for five minutes. The worker is stuck or too busy.

## Check

In Grafana **Explore**, query `aux_battles_worker_iteration_operational`,
`aux_battles_worker_queue_sample_success`, `aux_battles_worker_ready_jobs` and
`aux_battles_worker_oldest_ready_job_wait_seconds`. Then check the logs and
the queue:

```sh
docker compose --profile observability logs --tail=100 worker db
docker compose exec -T db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT song_id, state, attempt_count, available_at, lease_expires_at FROM song_jobs WHERE state IN ('"'QUEUED', 'RUNNING'"') ORDER BY available_at LIMIT 10"'
```

## Recover

Fix the error in the worker logs. Both health metrics return to 1 within 30
seconds. A stuck job is reclaimed when its lease expires.

## Drill

Run `docker compose stop db`. **Worker Loop Unhealthy** fires after about two
minutes. Run `docker compose start db` and confirm it resolves.
