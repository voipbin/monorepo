# bin-registrar-manager — Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| Extension creation fails | Asterisk DB unreachable or schema mismatch | Check `DATABASE_DSN_ASTERISK`; verify `ps_endpoints` table exists |
| SIP registrations not visible | Redis cache stale or Asterisk DB read error | Check `ps_contacts` directly in Asterisk DB; flush Redis cache for that endpoint |
| Trunk domain rejected on create | Domain name fails regex validation | Use only alphanumeric, `-`, `.` chars; max 31 chars; must start with a letter |
| Extension delete leaves Asterisk orphans | Partial delete (DB error mid-transaction) | Manually delete `ps_endpoints`, `ps_aors`, `ps_auths` rows with matching ID in Asterisk DB |
| `customer_deleted` cascade not completing | Subscribe consumer stopped | Check `subscribehandler` logs; verify RabbitMQ queue binding |
| Domain names not set at startup | `domain_name_extension` or `domain_name_trunk` config missing | Set both flags; service will fail to create extensions/trunks without them |
| Contact count mismatch | Redis vs Asterisk DB divergence | Invalidate Redis key for the endpoint; contact list re-reads from DB on miss |

## Debugging Guide

**Check service health:**
```bash
curl -s http://<pod-ip>:2112/metrics | grep registrar_manager
```

**Inspect extensions/trunks (CLI tool — bypasses RabbitMQ):**
```bash
./bin/registrar-control extension list --customer_id <uuid>
./bin/registrar-control extension get --id <extension-uuid>
./bin/registrar-control trunk list --customer_id <uuid>
```

**Check Asterisk tables directly:**
```sql
-- Active PJSIP endpoints
SELECT id, transport, context FROM ps_endpoints WHERE id LIKE '%<extension-uuid>%';

-- Active registrations
SELECT id, uri, expiration_time, endpoint FROM ps_contacts
WHERE endpoint = '<endpoint-id>' ORDER BY expiration_time DESC;

-- AOR status
SELECT id, max_contacts, total_contacts FROM ps_aors WHERE id = '<extension-uuid>';
```

**Check RabbitMQ queue depth:**
```bash
rabbitmqctl list_queues name messages consumers | grep registrar
```

**Check Redis contact cache:**
```bash
redis-cli -n 1 keys "registrar:contact:*" | head -10
```

## Configuration

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--database_dsn_bin` | `DATABASE_DSN_BIN` | required | bin-manager MySQL connection string |
| `--database_dsn_asterisk` | `DATABASE_DSN_ASTERISK` | required | Asterisk MySQL connection string |
| `--rabbitmq_address` | `RABBITMQ_ADDRESS` | required | RabbitMQ server |
| `--redis_address` | `REDIS_ADDRESS` | required | Redis server |
| `--redis_password` | `REDIS_PASSWORD` | `` | Redis auth password |
| `--redis_database` | `REDIS_DATABASE` | `` | Redis DB index |
| `--domain_name_extension` | `DOMAIN_NAME_EXTENSION` | required | Base domain for SIP extensions (e.g., `ext.voipbin.net`) |
| `--domain_name_trunk` | `DOMAIN_NAME_TRUNK` | required | Base domain for SIP trunks |
| `--prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | `/metrics` | Metrics path |
| `--prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `:2112` | Metrics listen address |

## Prometheus Metrics

Metrics are served at `<prometheus_listen_address><prometheus_endpoint>` (default: `:2112/metrics`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `registrar_manager_extension_create_total` | Counter | — | Extensions successfully created |
| `registrar_manager_extension_delete_total` | Counter | — | Extensions successfully deleted |
| `registrar_manager_trunk_create_total` | Counter | — | Trunks successfully created |
| `registrar_manager_trunk_delete_total` | Counter | — | Trunks successfully deleted |
| `registrar_manager_receive_request_process_time` | Histogram | `type`, `method` | RPC request processing latency |
| `registrar_manager_receive_subscribe_event_process_time` | Histogram | `publisher`, `type` | Event subscription processing latency |

**Useful PromQL:**
```promql
# Extension creation rate
rate(registrar_manager_extension_create_total[5m])

# P99 RPC latency
histogram_quantile(0.99, rate(registrar_manager_receive_request_process_time_bucket[5m]))
```
