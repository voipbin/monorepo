# bin-contact-manager — Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| Contact lookup returns 404 for known contact | Redis cache stale or evicted | Check cache connectivity; lookup falls back to DB if cache miss is propagated correctly |
| `customer_deleted` events not triggering cascades | Subscribe queue consumer stopped | Check `subscribehandler` logs; verify RabbitMQ queue binding |
| Tag assignments failing | `bin-tag-manager` RPC timeout | Verify tag-manager is healthy; check RabbitMQ for dead-lettered messages |
| Slow list queries | Missing index on `customer_id` + `tm_delete` | Verify database indexes; check slow query log |
| Duplicate phone number in lookup | Cache inconsistency after partial update | Cache key collision is unlikely; force cache invalidation via contact update |
| RPC requests timing out | DB connection pool exhaustion | Check `DATABASE_DSN` pool settings; monitor active connections |

## Debugging Guide

**Check service health:**
```bash
curl -s http://<pod-ip>:2112/metrics | grep contact_manager
```

**Inspect live contacts (CLI tool — bypasses RabbitMQ):**
```bash
./bin/contact-control contact list --customer-id <uuid> --limit 10
./bin/contact-control contact get --id <contact-uuid>
./bin/contact-control contact lookup --customer-id <uuid> --phone-e164 +15551234567
```

**Check RabbitMQ queue depth:**
```bash
rabbitmqctl list_queues name messages consumers | grep contact
```

**Check Redis cache:**
```bash
redis-cli -n 1 keys "contact:*" | head -20
```

**Database queries:**
```sql
-- Active contacts for a customer
SELECT id, display_name, tm_create FROM contact_contacts
WHERE customer_id = '<uuid>' AND tm_delete IS NULL
LIMIT 20;

-- Phone numbers for a contact
SELECT id, phone_number FROM contact_phone_numbers
WHERE contact_id = '<uuid>' AND tm_delete IS NULL;
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
| `contact_manager_receive_request_process_time` | Histogram | `type`, `method` | RPC request processing latency |
| `contact_manager_receive_subscribe_event_process_time` | Histogram | `publisher`, `type` | Event subscription processing latency |

**Useful PromQL:**
```promql
# P99 RPC latency
histogram_quantile(0.99, rate(contact_manager_receive_request_process_time_bucket[5m]))

# Event processing rate
rate(contact_manager_receive_subscribe_event_process_time_count[5m])
```
