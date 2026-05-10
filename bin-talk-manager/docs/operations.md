# bin-talk-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| 500 on message create | Parent message not found or in different chat | Check `parent_id` validity; verify chat IDs match |
| Reactions not updating | Concurrent writes causing JSON parse errors | Check MySQL logs for JSON_ARRAY_APPEND failures; restart if stuck |
| Participant add returns 409 | UPSERT collision with deleted UNIQUE index | Check for orphaned unique constraint violations in DB |
| RPC timeout | RabbitMQ queue backed up | Check queue depth on `bin-manager.talk-manager.request` |
| Redis connection refused | Cache unavailable | Service degrades gracefully; check `REDIS_ADDRESS` env |
| `tm_create` MySQL error | Timestamp format mismatch | Ensure `utilHandler.TimeGetCurTime()` is used, not `time.Now().UTC().Format()` |

## Debugging Guide

**List active RPC queue depth:**
```bash
kubectl exec -n voipbin deploy/rabbitmq -- rabbitmqctl list_queues name messages | grep talk-manager
```

**Check recent errors in service logs:**
```bash
kubectl logs -n voipbin -l app=talk-manager --tail=100 | grep -E "ERROR|Could not"
```

**Verify database tables:**
```bash
kubectl exec -n voipbin deploy/talk-manager -- mysql -u root -p -e "SELECT COUNT(*) FROM talk_chats; SELECT COUNT(*) FROM talk_messages;"
```

**Check reaction JSON corruption:**
```sql
SELECT id, metadata FROM talk_messages WHERE JSON_VALID(metadata) = 0;
```

**Test RPC with talk-control CLI:**
```bash
./bin/talk-control chat list --customer-id <uuid>
./bin/talk-control message list --chat-id <uuid>
```

## Configuration

| Flag / Env Var | Description | Default |
|----------------|-------------|---------|
| `database_dsn` / `DATABASE_DSN` | MySQL connection string | required |
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server URL | required |
| `redis_address` / `REDIS_ADDRESS` | Redis server address | optional |
| `redis_password` / `REDIS_PASSWORD` | Redis authentication | optional |
| `redis_database` / `REDIS_DATABASE` | Redis DB index | `1` |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics HTTP path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

## Prometheus Metrics

Metrics exposed at `PROMETHEUS_LISTEN_ADDRESS` (default `:2112/metrics`):

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `talk_manager_receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |
