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

-- Addresses (phone + email) for a contact
SELECT id, type, target, is_primary FROM contact_addresses
WHERE contact_id = '<uuid>';
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
| `contact_manager_case_getorcreate_deadlock_retry_total` | Counter | (none) | VOIP-1232: total GetOrCreate attempts restarted after a MySQL deadlock (errno 1213) on the case get-or-create transaction. Non-zero-but-flat is expected under concurrent same-peer traffic; a sustained climb suggests the in-process peer lock isn't preventing the race (e.g. multi-replica deployment -- see the topology caveat below) |
| `contact_manager_case_getorcreate_deadlock_exhausted_total` | Counter | (none) | VOIP-1232: total GetOrCreate calls that exhausted all `maxDeadlockRetries` attempts and gave up. Each occurrence means one `conversation_message_created`/`call_created` event's CRM projection was silently dropped (ack-before-process pipeline, no DLQ yet -- see VOIP-1233) |
| `contact_manager_case_getorcreate_peer_lock_timeout_total` | Counter | (none) | VOIP-1232: total GetOrCreate calls that failed to acquire the per-peer in-process serialization lock within `peerLockTimeout` (5s). Each occurrence also drops the triggering event (same caveat as above) |
| `contact_manager_case_peer_lock_map_size` | Gauge | (none) | VOIP-1232: current number of distinct peer-tuple entries held in the in-process GetOrCreate serialization lock map. No eviction is implemented yet; monitor for unbounded growth (expected to stay in the thousands-to-low-tens-of-thousands range for a busy tenant; escalate if it grows without bound) |

**Useful PromQL:**
```promql
# P99 RPC latency
histogram_quantile(0.99, rate(contact_manager_receive_request_process_time_bucket[5m]))

# Event processing rate
rate(contact_manager_receive_subscribe_event_process_time_count[5m])

# VOIP-1232: deadlock retry rate (per-minute)
rate(contact_manager_case_getorcreate_deadlock_retry_total[5m]) * 60

# VOIP-1232: silently-dropped-event rate (deadlock exhaustion + peer-lock timeout combined)
rate(contact_manager_case_getorcreate_deadlock_exhausted_total[5m]) * 60
  + rate(contact_manager_case_getorcreate_peer_lock_timeout_total[5m]) * 60
```

**VOIP-1232 topology caveat:** the peer-tuple serialization lock described
above is **in-process only**, not a distributed lock. It fully prevents
the same-peer-tuple concurrent-INSERT race under contact-manager's
**current single-replica production config** (`replicas: 1`, no HPA --
confirmed in both `k8s/deployment.yml` and the install repo's
`k8s/backend/services/contact-manager.yaml`). If contact-manager is ever
scaled to 2+ replicas, RabbitMQ's plain shared-queue competing-consumers
model gives no per-peer-tuple pod affinity, so this lock would provide
**zero** cross-pod protection and the fix degrades to relying on the
DB-level deadlock retry alone (still correct, just less effective at
preventing the deadlock in the first place). A Redis-based distributed
lock (contact-manager already depends on Redis for contact-body caching)
is the correct upgrade path if/when replicas are increased -- not built
preemptively here.
