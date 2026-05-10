# bin-outdial-manager — Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| Available targets query returns 0 when targets exist | Status filter mismatch or try-count thresholds too low | Check target statuses in DB; verify request includes correct `try_count_N` params |
| Targets stuck in `processing` | Campaign manager crashed mid-dial without resetting status | Manually update status via CLI tool or API `PUT /v1/outdialtargets/{id}/status` |
| Outdial created but campaign never dials | `campaign_id` not set on outdial | Use `PUT /v1/outdials/{id}/campaign_id` to associate with campaign |
| RPC requests timing out | MySQL connection pool exhaustion | Check `DATABASE_DSN` pool settings; monitor DB connections |
| Event publishing failing | RabbitMQ connectivity issue | Check RabbitMQ logs; verify exchange `bin-manager.outdial-manager.event` exists |

## Debugging Guide

**Check service health:**
```bash
curl -s http://<pod-ip>:2112/metrics | grep outdial_manager
```

**Inspect outdials (CLI tool — bypasses RabbitMQ):**
```bash
./bin/outdial-control outdial list --customer_id <uuid> --limit 10
./bin/outdial-control outdial get --id <outdial-uuid>
```

**Check RabbitMQ queue depth:**
```bash
rabbitmqctl list_queues name messages consumers | grep outdial
```

**Database queries:**
```sql
-- All outdials for a customer
SELECT id, name, campaign_id, tm_create FROM outdial_manager_outdial
WHERE customer_id = '<uuid>' AND tm_delete = '9999-01-01 00:00:00.000000';

-- Targets by status
SELECT id, status, try_count_0, destination_0 FROM outdial_manager_outdialtarget
WHERE outdial_id = '<uuid>' AND tm_delete = '9999-01-01 00:00:00.000000'
ORDER BY tm_create;

-- Stuck processing targets
SELECT id, tm_update FROM outdial_manager_outdialtarget
WHERE status = 'processing' AND tm_update < NOW() - INTERVAL 30 MINUTE
  AND tm_delete = '9999-01-01 00:00:00.000000';
```

**Reset a stuck target:**
```bash
# Via API (through RabbitMQ)
curl -X PUT .../v1/outdialtargets/<id>/status -d '{"status": "idle"}'
```

## Configuration

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--database_dsn` | `DATABASE_DSN` | required | MySQL connection string |
| `--rabbitmq_address` | `RABBITMQ_ADDRESS` | required | RabbitMQ server (`amqp://user:pass@host:port`) |
| `--redis_address` | `REDIS_ADDRESS` | required | Redis cache address |
| `--redis_password` | `REDIS_PASSWORD` | `` | Redis auth password |
| `--redis_database` | `REDIS_DATABASE` | `` | Redis DB index |
| `--prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | `/metrics` | Metrics path |
| `--prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `:2112` | Metrics listen address |

## Prometheus Metrics

Metrics are served at `<prometheus_listen_address><prometheus_endpoint>` (default: `:2112/metrics`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `outdial_manager_receive_request_process_time` | Histogram | `type`, `method` | RPC request processing latency |

**Useful PromQL:**
```promql
# P99 RPC latency
histogram_quantile(0.99, rate(outdial_manager_receive_request_process_time_bucket[5m]))

# Request rate
rate(outdial_manager_receive_request_process_time_count[5m])
```
