# Operations: bin-call-manager

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|--------------|------------|
| Calls stuck in `dialing` or `ringing` forever | ARI event delivery stopped (RabbitMQ `asterisk.all.event` queue is not being consumed) | Check RabbitMQ connection; verify `bin-asterisk-proxy` is running and publishing events; check `arieventhandler` log for connection errors |
| `recording_start` returns error immediately | Recording already active on the resource (another `recording_id` is set) | Call `/recording_stop` first; check `recording_id` field on the call/confbridge |
| Calls created but media does not flow | Confbridge or bridge was not created in Asterisk; mismatch between DB state and Asterisk state | Check `bridge_id` on the confbridge; verify Asterisk bridge exists via asterisk-proxy; use `call-control` CLI to inspect DB state |
| `external-media` requests fail with Asterisk error | Asterisk snoop channel creation failed; Asterisk WebSocket port not reachable | Verify `asterisk_ws_port` configuration; check Asterisk logs for snoop channel errors; verify network connectivity between call-manager pod and Asterisk |
| High call create latency | MySQL slow queries on `calls` table; Redis cache miss storm | Check `call_create_total` and `receive_request_process_time` metrics; run `EXPLAIN` on slow queries; verify Redis is reachable |
| Confbridge does not terminate when last call leaves | `no_auto_leave` flag is set, or `conference` type (does not auto-terminate) | Check confbridge `flags` and `type`; send explicit `/terminate` if stuck in `progressing` |
| Call-manager pod restarted and calls are orphaned | Pod crash left calls in `progressing` status with no Asterisk channels | Use `/v1/recovery` endpoint with Homer SIP capture data; or use `call-control call update-status` to force `hangup` status |
| Outbound call fails immediately | All dial routes exhausted; outbound config codec mismatch | Check `call_outbound_whitelist_rejected_total` metric; verify `outbound_config` has valid routes; check route-manager for routing entries |

## Debugging Guide

### Key Log Patterns

```bash
# Trace a specific call by UUID
kubectl logs -n voipbin deploy/bin-call-manager | grep <call-uuid>

# Find ARI event processing errors
kubectl logs -n voipbin deploy/bin-call-manager | grep "arieventhandler" | grep -i "error\|fail"

# Find confbridge join/leave events
kubectl logs -n voipbin deploy/bin-call-manager | grep "confbridge" | grep -E "join|leave|terminate"

# Find recording failures
kubectl logs -n voipbin deploy/bin-call-manager | grep "recording" | grep -i "error\|fail\|failed"

# Find outbound dial route failures
kubectl logs -n voipbin deploy/bin-call-manager | grep "dialroute\|dialfail\|whitelist"
```

### Tracing a Call

1. **Get call state from DB/cache** using the `call-control` CLI:
   ```bash
   ./bin/call-control call get --id <uuid>
   ```

2. **Check the call's channel and bridge IDs** — if `channel_id` is empty but status is `progressing`, the call is likely orphaned.

3. **Check confbridge membership** — if `confbridge_id` is set, fetch the confbridge to see `channel_call_ids`.

4. **Check metrics** for the time window:
   - `call_create_total` — rate of new calls
   - `call_hangup_total` — rate of hangups
   - `ari_event_listen_total{type="ChannelDestroyed"}` — Asterisk channel destruction rate
   - `call_duration_seconds` — histogram of call durations (p99 spike indicates stuck calls)

5. **Check RabbitMQ** for queue depth on `bin-manager.call-manager.request` — if depth is growing, the service is overloaded or stuck.

### call-control CLI

Direct DB/cache tool for emergency inspection or repair (bypasses RabbitMQ):

```bash
# Get call
./bin/call-control call get --id <uuid>

# Force-update call status (use only for orphaned calls)
./bin/call-control call update-status --id <uuid> --status hangup

# Delete call record
./bin/call-control call delete --id <uuid>
```

All output is JSON (stdout); logs go to stderr.

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
| `homer_api_address` | `HOMER_API_ADDRESS` | _(empty)_ | Homer SIP capture API base URL (optional) |
| `homer_auth_token` | `HOMER_AUTH_TOKEN` | _(empty)_ | Homer API authentication token (optional) |
| `homer_whitelist` | `HOMER_WHITELIST` | _(empty)_ | Comma-separated IP whitelist for Homer recovery endpoint |
| `asterisk_ws_port` | `ASTERISK_WS_PORT` | `8088` | Asterisk WebSocket port for ARI/external-media connections |

## Prometheus Metrics

All metric names are prefixed with `call_manager_` at runtime.

| Metric Name | Type | Description |
|-------------|------|-------------|
| `ari_event_listen_process_time` | Histogram | Time to process one ARI event (labels: `asterisk_id`, `type`) |
| `ari_event_listen_total` | Counter | Total ARI events received (labels: `type`, `asterisk_id`) |
| `bridge_create_total` | Counter | Total Asterisk bridges created |
| `bridge_destroy_total` | Counter | Total Asterisk bridges destroyed |
| `call_action_process_time` | Histogram | Time to execute a call action |
| `call_action_total` | Counter | Total call actions executed |
| `call_create_total` | Counter | Total calls created |
| `call_duration_seconds` | Histogram | Duration of completed calls in seconds |
| `call_hangup_total` | Counter | Total calls hung up |
| `call_manager_outbound_config_fetch_error_total` | Counter | Outbound config lookup errors |
| `call_outbound_whitelist_rejected_total` | Counter | Outbound calls rejected by whitelist |
| `channel_create_total` | Counter | Total Asterisk channels created |
| `channel_hangup_total` | Counter | Total Asterisk channels hung up |
| `channel_transport_direction_total` | Counter | Channels by transport direction (labels: `direction`) |
| `confbridge_close_total` | Counter | Total confbridges closed |
| `confbridge_create_total` | Counter | Total confbridges created |
| `confbridge_duration_seconds` | Histogram | Duration of completed confbridges in seconds |
| `confbridge_join_total` | Counter | Total calls joining confbridges |
| `conference_leave_total` | Counter | Total calls leaving confbridges |
| `external_media_start_total` | Counter | Total external media streams started |
| `external_media_stop_total` | Counter | Total external media streams stopped |
| `groupcall_create_total` | Counter | Total group calls created |
| `receive_request_process_time` | Histogram | RPC request processing time (labels: `type`, `method`) |
| `recording_end_total` | Counter | Total recordings ended |
| `recording_start_total` | Counter | Total recordings started |
| `subscribe_event_process_time` | Histogram | Event subscription processing time (labels: `publisher`, `type`) |
