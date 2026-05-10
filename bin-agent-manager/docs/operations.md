# Operations: bin-agent-manager

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|--------------|------------|
| Agent status stuck in `ringing` after call ends | call-manager event was not received (RabbitMQ delivery failure or subscribehandler crash) | Check subscribehandler logs; verify RabbitMQ queue `bin-manager.call-manager.event` has no backlog; manually update agent status via PUT `/v1/agents/{id}/status` |
| Login returns 401 despite correct credentials | Password hash mismatch after migration; agent account disabled | Verify agent exists and is active; check for bcrypt version compatibility; reset password via admin flow |
| Password reset email not sent | `password_reset_base_url` config not set, or email service (bin-email-manager) unavailable | Verify `RESET_BASE_URL` is configured; check email-manager health; inspect logs for RPC errors to email-manager |
| Queue not routing to agent | Agent tags do not match queue tag filter; agent status not `available` | Verify agent `tag_ids` match the queue's required tags; check agent status is `available`; check `rpc_call_total` for queue-manager → agent-manager calls |
| `get_by_customer_id_address` returning not-found | Agent address was updated or deleted; SIP URI normalization mismatch | Verify the SIP URI format matches exactly what is stored; check for trailing parameters or port differences |
| High latency on agent list | Large number of agents per customer; no index on `customer_id` + `status` | Check `db_operation_duration` metric; inspect slow queries; verify composite index exists |

## Debugging Guide

### Key Log Patterns

```bash
# Trace all operations for a specific agent UUID
kubectl logs -n voipbin deploy/bin-agent-manager | grep <agent-uuid>

# Find status update events from call-manager
kubectl logs -n voipbin deploy/bin-agent-manager | grep "subscribehandler" | grep -E "status|call|ringing"

# Find login failures
kubectl logs -n voipbin deploy/bin-agent-manager | grep "login" | grep -i "error\|fail\|unauthorized"

# Find password reset operations
kubectl logs -n voipbin deploy/bin-agent-manager | grep "password" | grep -i "reset\|forgot"
```

### Tracing an Agent Status Issue

1. **Get agent state via API**:
   ```bash
   curl -H "Authorization: Bearer <token>" https://api.voipbin.net/v1/agents/<uuid>
   ```

2. **Check the agent's current status** — if stuck in `ringing` or `busy`, check call-manager for any active calls referencing this agent.

3. **Check RabbitMQ event delivery** — if call-manager events are not arriving, the subscribehandler will not update agent status. Check `receive_subscribe_event_process_time` metric.

4. **Check the `login_total` metric** — a spike in failed logins may indicate a credential migration issue or brute-force attempts.

5. **Verify Redis cache consistency** — cache holds agent state for fast lookups; if cache is stale, reads from the API may return incorrect status.

### Metrics Checklist

```bash
# Login rate (success/fail)
kubectl exec -n voipbin deploy/bin-agent-manager -- curl -s localhost:9090/metrics | grep login_total

# RPC call errors (agent-manager calling other services)
kubectl exec -n voipbin deploy/bin-agent-manager -- curl -s localhost:9090/metrics | grep rpc_call_total

# Database operation duration (identify slow queries)
kubectl exec -n voipbin deploy/bin-agent-manager -- curl -s localhost:9090/metrics | grep db_operation_duration
```

## Configuration

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `rabbitmq_address` | `RABBITMQ_ADDRESS` | _(required)_ | RabbitMQ server address (amqp URL) |
| `prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | _(empty)_ | HTTP path for Prometheus metrics scrape |
| `prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | _(empty)_ | Listen address for metrics HTTP server |
| `database_dsn` | `DATABASE_DSN` | _(required)_ | MySQL DSN (`user:pass@tcp(host:port)/db`) |
| `redis_address` | `REDIS_ADDRESS` | _(required)_ | Redis server address (`host:port`) |
| `redis_password` | `REDIS_PASSWORD` | _(empty)_ | Redis password (optional) |
| `redis_database` | `REDIS_DATABASE` | `0` | Redis logical database index |
| `password_reset_base_url` | `PASSWORD_RESET_BASE_URL` | _(empty)_ | Base URL for password reset links in emails (required for password reset to work) |

## Prometheus Metrics

All metric names are prefixed with `agent_manager_` at runtime.

| Metric Name | Type | Description |
|-------------|------|-------------|
| `cache_operation_total` | Counter | Total Redis cache operations (labels: `type`, `result`) |
| `db_operation_duration` | Histogram | Database operation duration in seconds (labels: `type`) |
| `db_operation_total` | Counter | Total database operations (labels: `type`, `result`) |
| `login_total` | Counter | Total login attempts (labels: `result`) |
| `password_reset_total` | Counter | Total password reset requests (labels: `type`) |
| `receive_request_process_time` | Histogram | RPC request processing time (labels: `type`, `method`) |
| `receive_subscribe_event_process_time` | Histogram | Event subscription processing time (labels: `publisher`, `type`) |
| `rpc_call_duration` | Histogram | Outbound RPC call duration in seconds (labels: `target`) |
| `rpc_call_total` | Counter | Total outbound RPC calls (labels: `target`, `result`) |
