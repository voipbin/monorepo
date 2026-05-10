# Operations: bin-conference-manager

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|--------------|------------|
| Conference stuck in `progressing` with no participants | call-manager confbridge terminated but conference was not stopped; missed call-manager event | Check `subscribehandler` logs for call-manager events; manually call `POST /v1/conferences/{id}/stop` if confirmed empty |
| `recording_start` fails | call-manager returned error (confbridge not found, or recording already active) | Verify `confbridge_id` on the conference; check call-manager logs; verify only one recording per conference at a time |
| Conferencecall stays in `initiating` | Call-manager confbridge join event was not received; call failed to join before event was published | Check call-manager logs for confbridge join errors; check subscribehandler connectivity to RabbitMQ |
| Conference creation succeeds but no audio | call-manager failed to create confbridge (Asterisk issue); `confbridge_id` is empty on the conference | Verify call-manager is healthy; check Asterisk bridge creation logs; confirm confbridge_id is set after conference creation |
| `transcribe_start` returns error | Transcription service (bin-transcribe-manager) unavailable or quota exceeded | Check transcribe-manager health; verify customer quota; check conference for active `transcribe_id` |
| High latency on conference list requests | Large number of conferences per customer; missing index on `customer_id` | Check `receive_request_process_time` metric; inspect slow query logs |

## Debugging Guide

### Key Log Patterns

```bash
# Trace all operations for a specific conference UUID
kubectl logs -n voipbin deploy/bin-conference-manager | grep <conference-uuid>

# Find participant join/leave events from call-manager
kubectl logs -n voipbin deploy/bin-conference-manager | grep "subscribehandler" | grep -E "join|leave|confbridge"

# Find recording errors
kubectl logs -n voipbin deploy/bin-conference-manager | grep "recording" | grep -i "error\|fail"

# Find conference stop/termination events
kubectl logs -n voipbin deploy/bin-conference-manager | grep "conferencehandler" | grep -E "stop|terminat"
```

### Tracing a Conference

1. **Get conference state via API**:
   ```bash
   curl -H "Authorization: Bearer <token>" https://api.voipbin.net/v1/conferences/<uuid>
   ```

2. **Check participant list** — GET `/v1/conferencecalls?conference_id=<uuid>` to see all participants and their statuses.

3. **Correlate with call-manager** — the `confbridge_id` on the conference corresponds to a confbridge in call-manager. Check that confbridge's `channel_call_ids` for the actual Asterisk channels.

4. **Check events** using `conference_create_total` and `conference_close_total` metrics — a growing gap indicates conferences not being cleaned up.

### Metrics Checklist

```bash
# Conference creation rate
kubectl exec -n voipbin deploy/bin-conference-manager -- curl -s localhost:9090/metrics | grep conference_create_total

# Conference close rate (should roughly track create rate)
kubectl exec -n voipbin deploy/bin-conference-manager -- curl -s localhost:9090/metrics | grep conference_close_total

# Participant join rate
kubectl exec -n voipbin deploy/bin-conference-manager -- curl -s localhost:9090/metrics | grep conference_join_total
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

## Prometheus Metrics

All metric names are prefixed with `conference_manager_` at runtime.

| Metric Name | Type | Description |
|-------------|------|-------------|
| `conferencecall_total` | Counter | Total conferencecalls created |
| `conference_close_total` | Counter | Total conferences closed (stopped) |
| `conference_create_total` | Counter | Total conferences created |
| `conference_join_total` | Counter | Total participant join events processed |
| `receive_request_process_time` | Histogram | RPC request processing time (labels: `type`, `method`) |
| `receive_subscribe_event_process_time` | Histogram | Event subscription processing time (labels: `publisher`, `type`) |
