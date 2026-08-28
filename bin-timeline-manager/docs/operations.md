# bin-timeline-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| Empty query results | ClickHouse not receiving events | Check subscribe handler queue consumption; verify ClickHouse connection |
| `converting String to *ServiceName is unsupported` | Custom type in domain model scanned from ClickHouse | Ensure `models/event/` uses `string` for `Publisher`; convert at handler boundary |
| High `subscribe_batch_insert_time` | ClickHouse write pressure | Check ClickHouse resource utilization; consider tuning batch size |
| Suspected event loss (timeline gaps) | In-memory event channel overflow — `processEventRun` drops events silently (nil return, no retry) when the 1000-slot channel is full | Check `timeline_manager_subscribe_event_dropped_total`; any nonzero value confirms permanent loss. Correlate with `timeline_manager_subscribe_event_channel_usage` p99 and ClickHouse insert latency |
| SIP analysis returning 503 | Homer API unreachable | Check `homer_api_address` config and Homer service health |
| 27 queues failing to subscribe | RabbitMQ connection dropped | Check RabbitMQ connectivity; service will reconnect automatically |
| Migration failure at startup | Missing `migrations_path` or ClickHouse unavailable | Verify `CLICKHOUSE_ADDRESS` and `MIGRATIONS_PATH` env vars |
| `Dirty database version 4. Fix and force version.` at startup | The instance previously crash-looped on migration 000004's comment-semicolon bug (fixed 2026-07-31) and golang-migrate left `schema_migrations` at `dirty=1, version=4`; `m.Up()` refuses to proceed past a dirty version even after the file is fixed | One-time manual recovery, then restart the service. golang-migrate's ClickHouse `schema_migrations` is an append-only TinyLog table (no mutations; `Version()` reads the newest row by `sequence`), so recovery is an INSERT of a clean row rolling back to version 3: `INSERT INTO schema_migrations (version, dirty, sequence) VALUES (3, 0, toUnixTimestamp64Nano(now64(9)))` — the fixed 000004 then re-runs from a clean state. Its statements are safe to re-run (`ADD INDEX IF NOT EXISTS`; `MATERIALIZE` re-runs are harmless rewrites). Affects any ClickHouse that attempted 000004 between 2026-06-08 and the fix |
| Correlation returns incomplete results for old events after deploy | Migration 000004 `MATERIALIZE COLUMN`/`MATERIALIZE INDEX` run as async background mutations; migrate marks success before they finish | Monitor `SELECT * FROM system.mutations WHERE is_done = 0 OR latest_fail_reason != ''`; wait for completion before relying on historical correlation |

## Debugging Guide

**Check ClickHouse event count:**
```sql
SELECT publisher, count() FROM events GROUP BY publisher ORDER BY count() DESC;
```

**Check recent events for a resource:**
```sql
SELECT timestamp, publisher, type FROM events
WHERE resource_id = '<uuid>'
ORDER BY timestamp DESC
LIMIT 50;
```

**Check subscribe queue depths:**
```bash
kubectl exec -n voipbin deploy/rabbitmq -- rabbitmqctl list_queues name messages | grep -E "(timeline|event)"
```

**Check service logs for ingestion errors:**
```bash
kubectl logs -n voipbin -l app=timeline-manager --tail=200 | grep -E "ERROR|batch"
```

**Run database migration (via timeline-control):**
```bash
./bin/timeline-control migrate up
./bin/timeline-control migrate version
```

**Test RPC query:**
```bash
./bin/timeline-control health
```

## Configuration

| Flag / Env Var | Description | Default |
|----------------|-------------|---------|
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server URL | required |
| `clickhouse_address` / `CLICKHOUSE_ADDRESS` | ClickHouse server (e.g., `clickhouse.infrastructure:9000`) | required |
| `clickhouse_database` / `CLICKHOUSE_DATABASE` | ClickHouse database name | `default` |
| `migrations_path` / `MIGRATIONS_PATH` | Path to ClickHouse migration files | `./migrations` |
| `homer_api_address` / `HOMER_API_ADDRESS` | Homer SIP analysis API endpoint | optional |
| `homer_auth_token` / `HOMER_AUTH_TOKEN` | Homer API authentication token | optional |
| `gcs_bucket_name` / `GCS_BUCKET_NAME` | GCS bucket for PCAP archival | optional |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics HTTP path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

## Prometheus Metrics

Metrics exposed at `PROMETHEUS_LISTEN_ADDRESS` (default `:2112/metrics`):

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `timeline_manager_receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |
| `timeline_manager_subscribe_batch_insert_time` | Histogram | — | ClickHouse batch insert duration |
| `timeline_manager_subscribe_batch_size` | Histogram | — | Number of events per batch insert |
| `timeline_manager_subscribe_event_dropped_total` | Counter | — | Events dropped because the in-memory event channel was full. Any nonzero value is permanently lost customer timeline data |
| `timeline_manager_subscribe_event_channel_usage_ratio` | Gauge | — | Instantaneous len/cap of the in-memory event channel (0..1). Scrape-sampled; use the histogram for burst-accurate percentiles |
| `timeline_manager_subscribe_event_channel_usage` | Histogram | — | Channel occupancy (len/cap, 0..1) observed at every enqueue attempt. `histogram_quantile(0.99, ...)` over this answers the 2-replica gate (p99 < 0.5) across sub-scrape bursts |

## Deployment (Komodo)

Komodo-managed (VOIP-1349), same mechanism as the other `bin-*-manager` services
(see bin-call-manager for the original pattern, bin-agent-manager for the Tier 1
template this one extends). Deployed via `.circleci/scripts/render-image-tag.sh`
+ `.circleci/scripts/komodo-api-deploy.sh` from `komodo/docker-compose.yml`.

Unlike most `bin-*-manager` services, timeline-manager does not use MySQL/Redis
at all — its Komodo Variables are `RABBITMQ_ADDRESS`, `CLICKHOUSE_ADDRESS`,
`CLICKHOUSE_DATABASE`, `HOMER_API_ADDRESS`, `HOMER_AUTH_TOKEN`. `GCS_BUCKET_NAME`
(PCAP archival) is optional and not currently set in production, so it is
omitted from the Komodo compose file too.
