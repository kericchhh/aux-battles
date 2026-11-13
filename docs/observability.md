# Observability

The Compose `observability` profile starts Prometheus, Grafana, Loki, and Alloy
alongside the API and worker. Prometheus scrapes metrics from the Go processes.
Alloy reads their JSON logs from Docker and sends them to Loki. Grafana queries
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
