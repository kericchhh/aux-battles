# Observability

The Compose `observability` profile starts Prometheus, Grafana, Loki, Alloy and
`postgres-exporter` alongside the API and worker. Prometheus scrapes metrics
from the Go processes and, through the exporter, from PostgreSQL. Alloy reads their JSON logs from Docker and sends them to Loki. Grafana queries
Prometheus for the dashboard and alerts, and Loki for log searches.

## Start and verify

From the repository root, after creating `.env` as described in the
[README](../README.md#run-with-docker):

```sh
docker compose --profile observability up --build -d
```

Open [Prometheus targets](http://localhost:9090/targets) and check that `api`,
`worker`, and `prometheus` are up. Open [Grafana](http://localhost:3000). On a
fresh Grafana volume, sign in with `GRAFANA_ADMIN_USER` and
`GRAFANA_ADMIN_PASSWORD` from `.env`; an existing volume keeps its configured
credentials. Grafana provisions the Prometheus and Loki data
sources, the **API and Worker Overview** dashboard, and the **Worker
Unavailable**, **Worker Loop Unhealthy**, and **Song Queue Waiting Too Long**
alerts.

## Telegram notifications

Grafana sends firing and resolved alerts to a Telegram chat through the
`telegram` contact point in
`observability/grafana/provisioning/alerting/contact-points.yaml`. Each message
lists the alert's `summary` and `runbook_url` annotations, so it links to the
matching runbook. Alerts are grouped by folder and name, wait 30 seconds before
the first message, and repeat every four hours while they keep firing.

The file also provisions the notification policy, which replaces Grafana's
whole policy tree on every start and makes it and the contact point read-only
in the UI. Change routing in the file, not in Grafana. Removing the file does
not restore Grafana's default policy; that needs `resetPolicies` in a
provisioning file.

To set it up:

1. In Telegram, send `/newbot` to **@BotFather** and copy the token it returns.
   The token controls the bot, so keep it secret.
2. Send any message to the new bot, then run the command below with the token
   in place of `<token>`. Using `curl` keeps the token out of browser history.
   The number in `"chat":{"id":...}` is the chat ID; group chat IDs are
   negative.

   ```sh
   curl -s 'https://api.telegram.org/bot<token>/getUpdates'
   ```

3. Set `TELEGRAM_BOT_TOKEN` and `TELEGRAM_CHAT_ID` in `.env` and recreate
   Grafana with `docker compose --profile observability up -d grafana`.

Both variables are optional. Without them, Compose uses placeholder values, the
stack starts normally, and notifications fail. The token is passed to Grafana as
an environment variable, so anyone who can run `docker inspect` on this machine
can read it; the Grafana API and UI show it as redacted. To test, open **Alerting > Contact
points** in Grafana and use **Test** on `telegram`, or run the drill in the
[worker unavailable runbook](runbooks/worker-unavailable.md).

Once the services have started, check Loki readiness and generate an API log
entry:

```sh
curl -fsS http://localhost:3100/ready
curl -fsS http://localhost:8080/healthz
```

In Grafana **Explore**, select **Loki**, set the time range to the last hour,
and run:

```logql
{job="aux-battles",service="api"} | json
```

For worker logs, use `{job="aux-battles",service="worker"} | json`. Alloy
collects only the `api` and `worker` services in the current Compose project.
Compose passes its project name to Alloy, so log discovery follows renamed
checkouts and `-p` overrides.

## Metrics and alerts

Prometheus scrapes every 15 seconds. The API exposes
`http://localhost:8080/metrics`; Prometheus reaches the worker at
`worker:9091/metrics` on the Compose network. The dashboard shows scrape
status, Go process metrics, API request rate, 5xx percentage and latency, and
worker attempt counts and duration.

The API exports `aux_battles_http_requests_total` by route and status and
`aux_battles_http_request_duration_seconds`. The worker exports
`aux_battles_worker_attempts_total` by outcome and
`aux_battles_worker_attempt_duration_seconds`. A failed attempt may be retried,
so the attempt counter does not count distinct failed songs. Attempts are
counted when processing returns; a worker crash is not recorded as a failed
attempt. The [worker unavailable runbook](runbooks/worker-unavailable.md)
explains the alert and a local failure and recovery drill.

Every 15 seconds, the worker also samples jobs ready to be claimed, including
while audio processing is active. It exports
`aux_battles_worker_iteration_operational`,
`aux_battles_worker_queue_sample_success`, `aux_battles_worker_ready_jobs`, and
`aux_battles_worker_oldest_ready_job_wait_seconds`. The first two are 0 or 1;
the queue values reset to 0 when sampling fails, so check the sample-success
metric before interpreting an empty queue. Grafana alerts when either worker
health signal stays at 0 for two minutes or the oldest ready job waits over an
hour for five minutes. See the [song queue runbook](runbooks/song-queue-unhealthy.md)
for diagnosis and recovery.

## Database metrics

`postgres-exporter` reads PostgreSQL statistics and Prometheus scrapes it at
`postgres-exporter:9187/metrics` as the `postgres` job. The dashboard shows
whether the database is reachable (`pg_up`), open connections
(`pg_stat_database_numbackends`), database size (`pg_database_size_bytes`), and
committed and rolled back transactions per second. **Database Unavailable**
fires when `pg_up` stays at 0 for one minute or the exporter stops reporting;
see the [database unavailable runbook](runbooks/database-unavailable.md).

The exporter connects as `aux_monitor`, a role with PostgreSQL's built-in
`pg_monitor` permissions and at most three connections. It can read server
statistics and settings, including the text of running queries in
`pg_stat_activity`, but not app tables such as `users` or `sessions`, and it
cannot change data. The app sends values as query parameters, so query text
shows placeholders rather than user data. The one-shot
`db-roles` service creates the role and sets its password from
`POSTGRES_EXPORTER_PASSWORD` in `.env` each time the observability profile
starts, so it works on existing databases and picks up password changes. The
role is not created in a migration because migrations are committed and would
hold the password, and not in a PostgreSQL init script because those run only
when the data volume is first created.

`POSTGRES_EXPORTER_PASSWORD` is only needed for the observability profile. If
it is unset, `db-roles` exits with an error, the exporter does not start, and
**Database Unavailable** fires; the rest of the stack runs normally.

## Logs

Both programs write JSON to standard output with a `service` field and default
to `INFO` logging. Set `LOG_LEVEL=ERROR` when starting Go on the host or with
Compose to show only errors. `docker compose logs -f api worker` still shows
recent logs without Grafana.

API responses include an `X-Request-ID` header. The same ID appears in request
completion and handler error logs; completion logs record method, route pattern,
status, and duration in milliseconds. Processing attempt logs include the song
ID and attempt number, which link stage and outcome events. Startup and
iteration errors may lack those fields. Stage logs record durations for clip
extraction, Demucs, and output validation and publishing.

Search for one request in Grafana Explore by replacing the example ID:

```logql
{job="aux-battles",service="api"} | json | request_id="<request-id>"
```

Search for a song's processing events in the same way:

```logql
{job="aux-battles",service="worker"} | json | song_id="<song-id>"
```

The `service` label selects a stream; `| json` parses fields such as
`request_id` and `song_id` when querying. The IDs are not Loki index labels.
Logs from before a service restart remain searchable until retention removes
them.

## Storage and local access

Loki keeps logs for seven days using its compactor. The `loki_data` volume
stores them across container restarts, and `alloy_data` stores Alloy's reading
position. Prometheus samples and Grafana settings persist in `prometheus_data`
and `grafana_data`. `docker compose --profile observability down` stops the
stack while keeping these named volumes.

The published API, Prometheus, Grafana, and Loki ports bind to localhost. The
worker metrics port is available only on the Compose network. The metrics
endpoints and Loki have no authentication; restrict access if you run this
stack outside a trusted local environment. Alloy mounts the Docker socket to
discover and read container logs. This gives Alloy access to the Docker daemon
even though the socket mount uses `:ro`.

## Configuration and checks

Prometheus reads [`prometheus.yml`](../observability/prometheus.yml), Loki reads
[`loki.yml`](../observability/loki.yml), and Alloy reads
[`alloy.alloy`](../observability/alloy.alloy). Grafana provisioning files in
[`observability/grafana/`](../observability/grafana/) define the data sources,
dashboard, and alert. The dashboard JSON is the provisioned source: after
editing it in Grafana, export it as a V2 Resource JSON file to
`observability/grafana/dashboards/`. Grafana can replace UI changes with that
file on restart.

Validate the local configuration from the repository root:

```sh
docker compose --profile observability config --quiet
docker compose --profile observability run --rm --no-deps alloy validate /etc/alloy/config.alloy
docker compose --profile observability run --rm --no-deps loki -config.file=/etc/loki/config.yml -verify-config
```

CI also validates Compose, Prometheus, Loki, and Alloy configuration, plus
dashboard JSON syntax. It does not start the monitoring stack or test queries
against live metrics.
