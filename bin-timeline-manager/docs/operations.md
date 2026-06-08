# bin-timeline-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| Empty query results | ClickHouse not receiving events | Check subscribe handler queue consumption; verify ClickHouse connection |
| `converting String to *ServiceName is unsupported` | Custom type in domain model scanned from ClickHouse | Ensure `models/event/` uses `string` for `Publisher`; convert at handler boundary |
| High `subscribe_batch_insert_time` | ClickHouse write pressure | Check ClickHouse resource utilization; consider tuning batch size |
| SIP analysis returning 503 | Homer API unreachable | Check `homer_api_address` config and Homer service health |
| 27 queues failing to subscribe | RabbitMQ connection dropped | Check RabbitMQ connectivity; service will reconnect automatically |
| Migration failure at startup | Missing `migrations_path` or ClickHouse unavailable | Verify `CLICKHOUSE_ADDRESS` and `MIGRATIONS_PATH` env vars |
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
