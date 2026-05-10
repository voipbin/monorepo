# bin-direct-manager — Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| SIP calls to direct URI failing | Hash not found in cache or DB | Verify direct record exists; check Redis connectivity; confirm hash is correct |
| Cache returning stale hash after regeneration | Redis invalidation missed | Redis key should be deleted on regeneration; manually flush the key: `DEL direct:<hash>` |
| `customer_deleted` not cascading | Subscribe consumer stopped | Check `subscribehandler` logs; verify RabbitMQ queue binding for customer events |
| Hash collision on create | Hash space exhausted (unlikely) | Check for unusually high direct count; inspect creation error logs |
| Slow hash lookup | Redis miss falling through to MySQL | Check Redis availability; verify cache warming logic in `cachehandler` |

## Debugging Guide

**Check service health:**
```bash
curl -s http://<pod-ip>:2112/metrics | grep direct_manager
```

**Inspect direct records (CLI tool — bypasses RabbitMQ):**
```bash
./bin/direct-control direct list --customer-id <uuid> --limit 10
./bin/direct-control direct get --id <direct-uuid>
```

**Check RabbitMQ queue depth:**
```bash
rabbitmqctl list_queues name messages consumers | grep direct
```

**Check Redis cache:**
```bash
redis-cli -n 1 keys "direct:*" | head -10
redis-cli -n 1 get "direct:<hash>"
```

**Database queries:**
```sql
-- Active directs for a customer
SELECT id, resource_type, resource_id, hash FROM direct_manager_direct
WHERE customer_id = '<uuid>' AND tm_delete = '9999-01-01 00:00:00.000000';

-- Look up by hash
SELECT id, customer_id, resource_type, resource_id FROM direct_manager_direct
WHERE hash = '<hash>' AND tm_delete = '9999-01-01 00:00:00.000000';
```

## Configuration

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--database_dsn` | `DATABASE_DSN` | `testid:testpassword@tcp(127.0.0.1:3306)/test` | MySQL connection string |
| `--rabbitmq_address` | `RABBITMQ_ADDRESS` | `amqp://guest:guest@localhost:5672` | RabbitMQ server |
| `--redis_address` | `REDIS_ADDRESS` | `127.0.0.1:6379` | Redis server |
| `--redis_password` | `REDIS_PASSWORD` | `` | Redis auth password |
| `--redis_database` | `REDIS_DATABASE` | `1` | Redis DB index |
| `--prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | `/metrics` | Metrics path |
| `--prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `:2112` | Metrics listen address |

## Prometheus Metrics

Metrics are served at `<prometheus_listen_address><prometheus_endpoint>` (default: `:2112/metrics`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `direct_manager_receive_request_process_time` | Histogram | `type`, `method` | RPC request processing latency |
| `direct_manager_receive_subscribe_event_process_time` | Histogram | `publisher`, `type` | Event subscription processing latency |

**Useful PromQL:**
```promql
# P99 RPC latency
histogram_quantile(0.99, rate(direct_manager_receive_request_process_time_bucket[5m]))

# Event processing latency
histogram_quantile(0.99, rate(direct_manager_receive_subscribe_event_process_time_bucket[5m]))
```
