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
| Calls orphaned after an Asterisk container died | Calls left in `progressing` status with no Asterisk channels | Call recovery (automatic on container death, or `/v1/recovery`) moves the calls it can. For the rest, the channel health check declares the channel dead about 30 to 40 s after the container dies and runs the hangup for it (fake `ChannelDestroyed`); the call health check finishes a call about 30 to 40 s after its channel ended. If a call is still stuck, see "Stale non-final calls"; as a last resort use `call-control call update-status` to force `hangup` status (the real destroy event still runs the full cleanup for it, because the call has no `tm_hangup`) |
| Outbound call fails immediately | All dial routes exhausted; outbound config codec mismatch | Check `call_outbound_whitelist_rejected_total` metric; verify `outbound_config` has valid routes; check route-manager for routing entries |

## Stale non-final calls

A call must end in `hangup`. Two health checks make sure of it, one per call (`callhandler/health.go`) and one per channel (`channelhandler/health.go`). Each checks every 10 s and acts after more than 2 failed checks, so a stuck call is finished about 30 to 40 s after its channel ended, or about 40 s after its creation when its channel row never appeared.

- A `canceling` or `terminating` call whose channel ended, but whose `ChannelDestroyed` event was dropped (for example the event arrived before `StasisStart` and the channel type was still empty), is finished through `Hangup`: hangup reason `cancel`, `hangup_by` `local`.
- A `dialing` or `canceling` call that never got a channel row (the channel create request failed) is finished as `failed` (`hangup_by` `local`) with the activeflow stopped. The groupcall is not notified: the groupcall caller already updates its counters on the create error.
- A channel that stops answering Asterisk is ended by a fake `ChannelDestroyed` event, which runs `Hangup` for the call channel. The check cannot tell a dead Asterisk from a failing ARI request, so an ARI request outage that lasts longer than about 40 s while the event WebSocket still works can end live calls in the database (the media may continue) and mark their channels deleted; real events that arrive later do not change the finished calls.
- `Hangup` returns at once for a call that has `tm_hangup`, so a late real `ChannelDestroyed` event does not hang up a finished call a second time. A call forced to the `hangup` status with `call-control` has no `tm_hangup` and still gets the full cleanup.

Log lines that show the mechanism: `Exceeded max call health check retry count`, `Exceeded max channel health check retry count`, `The call has no channel. Hanging up the call.`, `The call has hungup already. Skipping.`

Detection query (read-only). After each bin-call-manager or Asterisk deploy, check for calls that are not final after 2 hours (the longest answered call seen in 90 days was 3599 s):

```sql
SELECT status, COUNT(*) FROM call_calls WHERE tm_hangup IS NULL AND status <> 'hangup' AND tm_create < NOW() - INTERVAL 2 HOUR GROUP BY status;
```

A non-zero result means a case the health checks do not cover (a call forced to `hangup` with `call-control` has no `tm_hangup` by design and is excluded by the status condition). There is no periodic job that cleans such rows and no alert; add one only if this query returns rows again after this behavior is deployed. This does not look at `call_groupcalls`, whose call count a failed group member can leave above zero (existing behavior).

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

## Deployment

bin-call-manager is the Komodo-managed deploy pilot (VOIP-1342) — the first
`bin-*-manager` service to move off `voipbin/voipbin`'s `install/`
(`versions.lock`/`ssh-deploy.sh`) mechanism. VOIP-1347 (Tier 1 rollout)
followed with 16 more services on the same pattern; the remaining
`bin-*-manager` services will migrate in future follow-up rollouts.

- **Stack definition:** `bin-call-manager/komodo/docker-compose.yml` (git
  is the source of truth for structure; Komodo only executes it on
  request — no polling/webhook auto-deploy).
- **CI path:** `.circleci/scripts/render-image-tag.sh` substitutes the
  built image tag, then `.circleci/scripts/komodo-api-deploy.sh` pushes
  the file's content to Komodo (`UpdateStack`) and triggers a deploy
  (`DeployStack`) over `https://komodo.voipbin.net`, gated by the
  `bin-call-manager-deploy` job's poll/running checks.
- **Break-glass fallback:** `.circleci/scripts/komodo-api-deploy-ssh-fallback.sh`
  (SSH tunnel to bm-nyc-01, unchanged from VOIP-1341) for use if the public
  HTTPS endpoint itself is unreachable.
- **Full design, cutover sequencing, and rollback procedure:**
  [docs/plans/2026-08-16-komodo-call-manager-cutover-design.md](../docs/plans/2026-08-16-komodo-call-manager-cutover-design.md)
  (in the monorepo root, not this service's own `docs/`).

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
| `homer_whitelist` | `HOMER_WHITELIST` | _(empty)_ | Comma-separated IPs whose capture rows Homer excludes from the recovery query (the Kamailio outer interface). Call recovery reconstructs the dialog from the remaining rows and fails closed when copies of the same message differ, so this must exclude the hops that rewrite Contact or Record-Route |
| `asterisk_ws_port` | `ASTERISK_WS_PORT` | `8088` | Asterisk WebSocket port for ARI/external-media connections |

Call recovery always runs (no on/off setting). A recovered call logs `Switched the call to the recovery channel`. Expected noise during a switch: the old channel's late destroy after a switch (real, or the fake destroy that the channel health check publishes for the old channel), and a recovery leg the remote refused, each log `Could not get the call info from the db` (error), because no call is owned by that channel. The event is not redelivered: the asterisk-proxy event consumer logs a handler error and moves on.

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
