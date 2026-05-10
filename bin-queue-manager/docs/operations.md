# Operations: bin-queue-manager

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|--------------|------------|
| Queuecalls stuck in `waiting` with available agents | Agent tag mismatch (queue tags ≠ agent tags); agent-manager events not reaching queue-manager; execute loop not running | Verify queue `tag_ids` matches at least one agent's tags; check subscribehandler logs for agent-manager events; trigger `POST /v1/queues/{id}/execute` manually |
| Queuecall stays in `connecting` forever | Agent did not answer (conference join failed); call-manager confbridge error | Check conference-manager logs; verify agent address is reachable; check `queuecall_done_total` vs `queuecall_create_total` for anomalies |
| Wait timeout not firing | Scheduler not calling `timeout_wait`; queuecall already progressed to `service` | Verify the scheduling mechanism (flow action or external scheduler) is running; check queuecall status directly |
| Queuecall immediately abandoned after creation | Caller hung up before connecting; wait_timeout set very low | Check call-manager events for call hangup before queue entry; review queue `wait_timeout` configuration |
| Conference not created for queuecall | conference-manager unavailable; call-manager confbridge quota exceeded | Check conference-manager health; check `queuecall_create_total` vs `conference_join_total` in conference-manager |
| High queuecall abandon rate | Too few available agents; wait_timeout too short; routing method not finding matches | Check agent availability dashboard; review `queuecall_abandoned_total` vs `queuecall_waiting_duration_seconds`; consider increasing wait_timeout or adding agents |

## Debugging Guide

### Key Log Patterns

```bash
# Trace all operations for a specific queuecall UUID
kubectl logs -n voipbin deploy/bin-queue-manager | grep <queuecall-uuid>

# Find routing execution attempts
kubectl logs -n voipbin deploy/bin-queue-manager | grep "queuehandler" | grep -E "execute|routing|agent"

# Find timeout events
kubectl logs -n voipbin deploy/bin-queue-manager | grep -E "timeout_wait|timeout_service"

# Find abandon events
kubectl logs -n voipbin deploy/bin-queue-manager | grep "abandon" | grep -i "error\|kick"
```

### Tracing a Stuck Queuecall

1. **Get queuecall state via API**:
   ```bash
   curl -H "Authorization: Bearer <token>" https://api.voipbin.net/v1/queuecalls/<uuid>
   ```

2. **Check the queue's eligible agents** — GET `/v1/queues/{queue_id}/agents` to see which agents are currently available and match the queue's tags.

3. **Check the conference** — if `conference_id` is set, check conference-manager for the conference status and participants.

4. **Check wait duration** — `queuecall_waiting_duration_seconds` histogram shows how long callers are waiting. A high p99 indicates persistent routing failures.

5. **Manually kick a stuck queuecall** via `POST /v1/queuecalls/{id}/kick` if the caller has already hung up but the queuecall was not cleaned up.

### Metrics Checklist

```bash
# Abandon rate
kubectl exec -n voipbin deploy/bin-queue-manager -- curl -s localhost:9090/metrics | grep queuecall_abandoned_total

# Create vs done ratio (health indicator)
kubectl exec -n voipbin deploy/bin-queue-manager -- curl -s localhost:9090/metrics | grep -E "queuecall_create_total|queuecall_done_total"

# Waiting duration distribution
kubectl exec -n voipbin deploy/bin-queue-manager -- curl -s localhost:9090/metrics | grep queuecall_waiting_duration_seconds
```

## Configuration

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `database_dsn` | `DATABASE_DSN` | _(required)_ | MySQL DSN (`user:pass@tcp(host:port)/db`) |
| `prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | _(empty)_ | HTTP path for Prometheus metrics scrape |
| `prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | _(empty)_ | Listen address for metrics HTTP server |
| `rabbitmq_address` | `RABBITMQ_ADDRESS` | _(required)_ | RabbitMQ server address (amqp URL) |
| `redis_address` | `REDIS_ADDRESS` | _(required)_ | Redis server address (`host:port`) |
| `redis_password` | `REDIS_PASSWORD` | _(empty)_ | Redis password (optional) |
| `redis_database` | `REDIS_DATABASE` | `0` | Redis logical database index |

## Prometheus Metrics

All metric names are prefixed with `queue_manager_` at runtime.

| Metric Name | Type | Description |
|-------------|------|-------------|
| `queuecall_abandoned_total` | Counter | Total queuecalls abandoned (wait timeout or caller hangup) |
| `queuecall_create_total` | Counter | Total queuecalls created |
| `queuecall_done_total` | Counter | Total queuecalls successfully completed |
| `queuecall_waiting_duration_seconds` | Histogram | Time queuecalls spent in `waiting` state before connecting or abandoning |
| `receive_request_process_time` | Histogram | RPC request processing time (labels: `type`, `method`) |
| `receive_subscribe_event_process_time` | Histogram | Event subscription processing time (labels: `publisher`, `type`) |
