# bin-transfer-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| Transfer stuck in block state | `attendedUnblock` / `blindUnblock` not called | Check call-manager event delivery; manually cancel via DELETE `/v1/transfers/{id}` |
| Parties stuck on hold after failed transfer | `attendedUnblock` not executed on rollback | Trigger delete/cancel on transfer ID; verify `groupcall_hangup` event was received |
| Confbridge survives transferer hangup (blind) | `FlagNoAutoLeave` not cleared | Check `blindUnblock` execution; verify `groupcall_hangup` event handler ran |
| Transfer not found by groupcall ID | Redis cache miss | Check Redis connectivity; cache is populated at execute phase |
| `groupcall_progressing` not bridging parties | Event not received or wrong groupcall ID | Check subscribe handler is consuming `bin-manager.call-manager.event` |
| RPC timeout on transfer start | `bin-call-manager` unavailable | Check call-manager health and RabbitMQ queue depth |

## Debugging Guide

**Check active transfers:**
```bash
./bin/transfer-control transfer get-by-call --call_id <uuid>
./bin/transfer-control transfer get-by-groupcall --groupcall_id <uuid>
```

**Check subscribe queue depth:**
```bash
kubectl exec -n voipbin deploy/rabbitmq -- rabbitmqctl list_queues name messages | grep -E "call-manager.event|transfer"
```

**Check service logs for state transitions:**
```bash
kubectl logs -n voipbin -l app=transfer-manager --tail=200 | grep -E "ERROR|attended|blind|unblock"
```

**Verify Redis cache entries:**
```bash
kubectl exec -n voipbin deploy/redis -- redis-cli keys "*transfer*"
```

**Force rollback a stuck transfer:**
```bash
# Via RPC — triggers attendedUnblock or blindUnblock depending on type
./bin/transfer-control transfer cancel --id <transfer-uuid>
```

## Configuration

| Flag / Env Var | Description | Default |
|----------------|-------------|---------|
| `database_dsn` / `DATABASE_DSN` | MySQL connection string | required |
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server URL | required |
| `redis_address` / `REDIS_ADDRESS` | Redis server address | required |
| `redis_password` / `REDIS_PASSWORD` | Redis authentication | optional |
| `redis_database` / `REDIS_DATABASE` | Redis DB index | optional |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics HTTP path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

## Prometheus Metrics

Metrics exposed at `PROMETHEUS_LISTEN_ADDRESS` (default `:2112/metrics`):

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `transfer_manager_receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |
| `transfer_manager_receive_subscribe_event_process_time` | Histogram | `publisher`, `type` | Event subscription processing duration |
