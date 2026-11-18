# Observability

The Compose `observability` profile adds Prometheus, Grafana, Loki, Alloy and
`postgres-exporter`. On Kubernetes, kube-prometheus-stack does this instead;
see the [Kubernetes guide](kubernetes.md).

```sh
docker compose --profile observability up --build -d
```

| Tool | Address | Notes |
| --- | --- | --- |
| Grafana | <http://localhost:3000> | `GRAFANA_ADMIN_USER` and `GRAFANA_ADMIN_PASSWORD` from `.env`, applied on a fresh volume |
| Prometheus | <http://localhost:9090/targets> | `api`, `worker`, `postgres` and `prometheus` should be up |
| Loki | <http://localhost:3100/ready> | Returns `503` for about a minute after starting |

All ports bind to localhost. Prometheus, Loki and the metrics endpoints have no
authentication.

## Telegram notifications

Grafana sends firing and resolved alerts to Telegram, with a link to the
matching runbook.

1. In Telegram, send `/newbot` to **@BotFather** and copy the token.
2. Send any message to the new bot, then find the chat ID in
   `"chat":{"id":...}`:

   ```sh
   curl -s 'https://api.telegram.org/bot<token>/getUpdates'
   ```

3. Set `TELEGRAM_BOT_TOKEN` and `TELEGRAM_CHAT_ID` in `.env` and run
   `docker compose --profile observability up -d grafana`.

Test it with **Alerting > Contact points > Test** in Grafana. Routing lives in
[`contact-points.yaml`](../observability/grafana/provisioning/alerting/contact-points.yaml)
and is read-only in the UI.

## Metrics and alerts

| Metric | Meaning |
| --- | --- |
| `aux_battles_http_requests_total`, `aux_battles_http_request_duration_seconds` | API requests by route and status, and latency |
| `aux_battles_worker_attempts_total`, `aux_battles_worker_attempt_duration_seconds` | Processing attempts by outcome, and duration |
| `aux_battles_worker_iteration_operational`, `aux_battles_worker_queue_sample_success` | Worker health, 0 or 1 |
| `aux_battles_worker_ready_jobs`, `aux_battles_worker_oldest_ready_job_wait_seconds` | Queue length and age |
| `pg_up` and other `pg_*` | PostgreSQL, through `postgres-exporter` with a `pg_monitor` role |

| Alert | Runbook |
| --- | --- |
| Worker Unavailable | [worker-unavailable](runbooks/worker-unavailable.md) |
| Worker Loop Unhealthy, Song Queue Waiting Too Long | [song-queue-unhealthy](runbooks/song-queue-unhealthy.md) |
| Database Unavailable | [database-unavailable](runbooks/database-unavailable.md) |

## Logs

Both services write JSON logs, which Alloy ships to Loki. Search them in
Grafana **Explore** with the Loki data source:

```logql
{job="aux-battles",service="api"} | json | request_id="<request-id>"
{job="aux-battles",service="worker"} | json | song_id="<song-id>"
```

API responses carry the ID in an `X-Request-ID` header. Loki keeps logs for
seven days. Set `LOG_LEVEL` to change verbosity; the default is `INFO`.

## Configuration

| File | Used by |
| --- | --- |
| [`prometheus.yml`](../observability/prometheus.yml) | Prometheus |
| [`loki.yml`](../observability/loki.yml) | Loki |
| [`alloy.alloy`](../observability/alloy.alloy) | Alloy |
| [`grafana/`](../observability/grafana) | Data sources, dashboard and alerts |

The dashboard JSON is the source of truth, and Kubernetes reuses it. After
editing the dashboard in Grafana, export it as a V2 Resource JSON file over
`observability/grafana/dashboards/API & Worker Overview.json`.
