# Operations: bin-campaign-manager

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|--------------|------------|
| Campaign in `run` status but no calls being made | Service level throttle too low (0 agents available in queue); outdial list exhausted; outplan `max_try_count` reached | Check queue agent availability; verify outplan has dials remaining; check `campaign_execute_total` metric for execution attempts |
| Campaign stuck in `stopping` | In-progress calls not completing; call-manager not sending hangup events | Check subscribehandler logs for call-manager events; verify active calls in call-manager; manually update campaign status if needed |
| Campaigncalls created but calls not dialing | call-manager call creation failing (route not found, outbound config issue, insufficient balance) | Check call-manager logs for dial failures; verify outplan source number exists in routing config; check billing balance |
| High retry rate per campaigncall | All destinations are busy/no-answer; network issues; time-of-day restrictions | Check outplan `dial_timeout` and `try_interval`; review destination number validity; check call-manager for dial result patterns |
| Service level not throttling correctly | queue_id not set or queue has no agents; service_level calculation issue | Verify campaign has `queue_id` set; check queue-manager agent availability; review `service_level` value (0-100 percentage) |
| Campaign execute total not incrementing | Campaign execute endpoint not being called by scheduler; campaign status is `stop` | Verify the execution scheduler is running and calling `POST /v1/campaigns/{id}/execute` periodically; verify campaign status is `run` |

## Debugging Guide

### Key Log Patterns

```bash
# Trace all operations for a specific campaign UUID
kubectl logs -n voipbin deploy/bin-campaign-manager | grep <campaign-uuid>

# Find campaign execution attempts
kubectl logs -n voipbin deploy/bin-campaign-manager | grep "campaignhandler" | grep -E "execute|status|run|stop"

# Find call outcome events from call-manager
kubectl logs -n voipbin deploy/bin-campaign-manager | grep "subscribehandler" | grep -E "hangup|done|call"

# Find campaigncall retry attempts
kubectl logs -n voipbin deploy/bin-campaign-manager | grep "campaigncallhandler" | grep -E "retry|try|dial"
```

### Tracing a Campaign Execution Issue

1. **Get campaign state via API**:
   ```bash
   curl -H "Authorization: Bearer <token>" https://api.voipbin.net/v1/campaigns/<uuid>
   ```

2. **Check campaigncalls** — GET `/v1/campaigncalls?campaign_id=<uuid>` to see all call attempts and their statuses.

3. **Check the outplan** — verify outplan has valid `source`, `dial_timeout`, and that `dials` list is not empty.

4. **Check service level** — if queue_id is set, verify queue has available agents. Service level = 0 means no calls will be made.

5. **Check metrics** — `campaign_execute_total` should increment each time execute is called; `campaigncall_create_total` increments per call attempt; compare these rates to identify where execution is stalling.

### campaign-control CLI

The `campaign-control` binary provides direct DB/cache access for inspection:

```bash
# Get campaign details
./bin/campaign-control campaign get --id <uuid>

# Get all campaigncalls for a campaign
./bin/campaign-control campaigncall list --campaign-id <uuid>
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

## Prometheus Metrics

All metric names are prefixed with `campaign_manager_` at runtime.

| Metric Name | Type | Description |
|-------------|------|-------------|
| `campaigncall_create_total` | Counter | Total campaigncalls (call attempts) created |
| `campaigncall_done_total` | Counter | Total campaigncalls completed (any outcome) |
| `campaign_create_total` | Counter | Total campaigns created |
| `campaign_execute_total` | Counter | Total campaign execute calls (each execution loop trigger) |
| `campaign_status_run_total` | Counter | Total campaigns transitioned to `run` status |
| `campaign_status_stop_total` | Counter | Total campaigns transitioned to `stop` status |
| `receive_request_process_time` | Histogram | RPC request processing time (labels: `type`, `method`) |
| `receive_subscribe_event_process_time` | Histogram | Event subscription processing time (labels: `publisher`, `type`) |
