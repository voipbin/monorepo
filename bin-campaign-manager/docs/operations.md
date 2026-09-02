# Operations: bin-campaign-manager

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|--------------|------------|
| Campaign in `run` status but no calls being made | Service level throttle too low (0 agents available in queue); outdial list exhausted; outplan `max_try_count` reached | Check queue agent availability; verify outplan has dials remaining; check `campaign_execute_total` metric for execution attempts |
| Campaign stuck in `stopping` | In-progress calls not completing; call-manager not sending hangup events | Check subscribehandler logs for call-manager events; verify active calls in call-manager; manually update campaign status if needed |
| Campaigncalls created but calls not dialing | call-manager call creation failing (route not found, outbound config issue, insufficient balance) | Check call-manager logs for dial failures; verify outplan source number exists in routing config; check billing balance |
| High retry rate per campaigncall | All destinations are busy/no-answer; network issues; time-of-day restrictions | Check outplan `dial_timeout` and `try_interval`; review destination number validity; check call-manager for dial result patterns |
| Service level not throttling correctly | queue_id not set or queue has no agents; service_level calculation issue | Verify campaign has `queue_id` set; check queue-manager agent availability; review `service_level` value (0-100 percentage) |
| Campaign execute total not incrementing | The self-scheduling execute chain stalled (campaign-manager's consumer was down when the last delayed RPC fired, or the delayed message was lost); campaign status is `stop` | Check campaign-manager pod health and RabbitMQ delayed-exchange health; verify campaign status is `run`; call `POST /v1/campaigns/{id}/execute` manually to restart the chain |

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

## Deployment

bin-campaign-manager deploys via Komodo (VOIP-1347 Tier 1 rollout, following the
VOIP-1342/bin-call-manager pilot pattern) instead of the older SSH +
`versions.lock` (`ssh-deploy.sh`) path.

- **Stack definition:** `bin-campaign-manager/komodo/docker-compose.yml` (git is
  the source of truth for structure; Komodo only executes it on
  request).
- **CI path:** `.circleci/scripts/render-image-tag.sh` substitutes
  the built image tag, then `.circleci/scripts/komodo-api-deploy.sh`
  pushes the file's content to Komodo and triggers a deploy, gated
  by the `bin-campaign-manager-deploy` job's poll/running checks.
- **Full design and cutover procedure:**
  [docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md](../../docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md)
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

## Prometheus Metrics

All metric names are prefixed with `campaign_manager_` at runtime.

| Metric Name | Type | Description |
|-------------|------|-------------|
| `campaigncall_create_total` | Counter | Total campaigncalls (call attempts) created |
| `campaigncall_done_total` | Counter | Total campaigncalls completed (any outcome) |
| `campaign_create_total` | Counter | Total campaigns created |
| `campaign_execute_total` | Counter | Total campaign execute calls (each execution loop trigger) |
| `campaign_flow_delete_failed_total` | CounterVec | Total failures to delete a campaign's backing flow during campaign delete (labels: `reason` = `not_found` \| `error`). Best-effort cleanup; a failure here does not fail the campaign delete but leaves the flow orphaned, counting against bin-flow-manager's per-customer flow cap. |
| `campaign_flow_reconcile_cleaned_total` | Counter | (VOIP-1444) Total orphaned flows cleaned up by the reconciliation job. See [Orphaned Flow Reconciliation](#orphaned-flow-reconciliation-voip-1444) below. |
| `campaign_flow_reconcile_failed_total` | Counter | (VOIP-1444) Total row-level reconciliation failures — a non-not-found `FlowV1FlowGet` error **or** a `FlowV1FlowDelete` error on a genuine orphan. No `reason` label distinguishes which RPC failed; a `bin-flow-manager` outage or open circuit breaker fails every remaining candidate in the pass on the `FlowV1FlowGet` branch, so this counter is the primary outage signal, not an edge case. |
| `campaign_status_run_total` | Counter | Total campaigns transitioned to `run` status |
| `campaign_status_stop_total` | Counter | Total campaigns transitioned to `stop` status |
| `receive_request_process_time` | Histogram | RPC request processing time (labels: `type`, `method`) |
| `receive_subscribe_event_process_time` | Histogram | Event subscription processing time (labels: `publisher`, `type`) |

## Orphaned Flow Reconciliation (VOIP-1444)

### What it does

`POST /v1/campaigns/flows/reconcile` (internal-only, not exposed through
`bin-api-manager`) scans campaigns deleted within a recent, bounded time
window (`window`, `scanLimit` — compiled constants in
`pkg/campaignhandler/reconcile.go`) and deletes the backing flow of any
whose owning campaign is deleted but whose flow is still live — closing the
gap left when `Delete()`'s best-effort `FlowV1FlowDelete` call (VOIP-1443)
fails silently. It is dispatched by the `bin-schedule-manager`
`campaign-flow-reconcile` schedule (seeded via a `bin-dbscheme-manager`
Alembic migration, shipped **disabled** at seed time — see the migration's
own docstring and the PR's rollout-sequencing note for the manual smoke
test and enable step) on its `cron` interval, or via a manual
`POST /v1/schedules/{id}/execute` RPC (there is no `schedule-control
execute` subcommand — `schedule-control` only supports
`list`/`get`/`enable`/`disable` for schedules).

Request body: `{"recent_interval_sec": <int64>}` — the schedule's own
`cron` interval, in seconds, threaded through explicitly because
`bin-campaign-manager` has no way to read `bin-schedule-manager`'s own
`cron` field from its database. A missing or non-positive value falls back
to a conservative 24h default and logs a warning.

The route always returns HTTP 200 for a pass that ran, even with
`failed > 0` or `partial == true` in the body — those are data about the
pass, not a pass-level error. A non-2xx response means the initial
`CampaignListDeletedSince` query itself failed (the pass never ran).

### `saturated` vs `recent_saturated` — two different signals, only one is actionable

- **`saturated`** (`len(candidates) == scanLimit`): purely informational.
  This fires routinely once the window's total candidate population
  exceeds `scanLimit` — including when every newly-deleted campaign is
  still being examined on the very next pass after its own deletion, i.e.
  when the job is working exactly as intended. Do not alert on this alone.
- **`recent_saturated`**: the actionable rate-risk signal. True when the
  count of candidates deleted within `recent_interval_sec` of this pass
  starting already fills `scanLimit` on its own — meaning the actual
  safety condition (`scanLimit` > deletions per cron interval) is being
  violated, not merely approached (the derivation is exact; there is no
  early-warning margin). **Remedy**: raise `scanLimit` and/or shorten the
  schedule's `cron` interval. These two remedies are **not equally fast**:
  raising `scanLimit` is a compiled Go constant and needs a code change
  plus redeploy; shortening `cron` (and `recent_interval_sec`, see below)
  is a live data edit via `schedule-control`/`PUT /v1/schedules/{id}`, no
  redeploy — reach for the live edit first under time pressure.
- Both `saturated` and `recent_saturated` land in the response body only —
  there is deliberately no third Prometheus counter for either signal (see
  "Durability" below for why this is safe despite this service's own logs
  rotating within hours).

### Changing the schedule's `cron` interval — a two-step operational change

`target_data.recent_interval_sec` is seeded from the same migration row as
`cron` (kept in sync at seed time only). A later, post-seed edit to `cron`
(via `schedule-control` or `PUT /v1/schedules/{id}`, including the
`recent_saturated` remedy above) does **not** automatically update
`target_data.recent_interval_sec` — they are edited through different
operational paths once the schedule exists. Any operational change to the
interval **must**:

1. Update `target_data.recent_interval_sec` to match the new `cron` in the
   same change.
2. Re-verify `reconcilePassTimeout` (`pkg/campaignhandler/reconcile.go`) is
   still well below the new, shorter interval before applying it —
   `reconcilePassTimeout` is a compiled constant that does not move on its
   own, and a large-enough interval cut could reintroduce the cadence
   degradation described below.

### `reconcilePassTimeout` sizing — two independently load-bearing constraints

`ReconcileOrphanedFlows` bounds its own execution with an internal
`context.WithTimeout(reconcilePassTimeout)` — required because this
service's RPC listenhandler builds every request's context with
`context.Background()` (no deadline propagation from
`bin-schedule-manager`'s `timeout_ms`). `reconcilePassTimeout` must satisfy:

- **Constraint (a) — the actual mutual-exclusion guarantee**:
  `reconcilePassTimeout` strictly below the schedule's seeded `timeout_ms`,
  with margin covering real message-delivery delay (not just processing
  latency — the two timeouts' clocks start at different points).
  `bin-schedule-manager`'s "Forbid overlap" guard (`ExecutionHasRunning`,
  checked before the `tm_next_run` CAS on every claim path) keeps an
  execution row `running` only until `ExecutionComplete` fires, which
  happens as soon as `SendRequest`'s own `timeout_ms`-bounded wait returns
  — not when this pass actually finishes. If `reconcilePassTimeout` could
  exceed `timeout_ms`, there is a real window where `bin-schedule-manager`
  believes the pass is done while it is still physically executing, and a
  manual execute fired in that window would genuinely race it. **This
  violation has no runtime monitor** — it produces an ordinary `success`
  execution row, indistinguishable from a correctly-serialized one.
  Correct static sizing is the only defense.
- **Constraint (b) — a cadence/liveness requirement, not a concurrency
  guard**: `reconcilePassTimeout` well below the schedule's `cron`
  interval. Because guard 1 above is checked before the `tm_next_run` CAS,
  a cron-triggered claim that arrives while the row is still `running` is
  short-circuited into a dispatch-level overlap skip — it does **not**
  create a second execution (that's already prevented by constraint (a)
  holding); it just means the schedule falls behind. **Observable
  symptom**: the dispatch-level metric
  `schedule_manager_dispatch_total{result="skipped_overlap"}` climbing for
  this schedule indicates a constraint-(b) violation (chronic cadence
  degradation) specifically on the **cron** dispatch path — a manual
  `/v1/schedules/{id}/execute` hitting the same overlap guard returns a
  `FailedPrecondition` `EXECUTION_IN_PROGRESS` error instead, with no
  metric at all (relevant for a rollout smoke test that races a still-
  running prior pass: just retry). This metric says nothing about a
  constraint-(a) violation, which has no runtime monitor at all. The
  counter's granularity is also per-replica, in-memory dedup — a single
  stuck cron slot can increment it more than once across replicas, so read
  "climbing" as a health signal, not an exact skip count.

### Durability

`saturated`, `recent_saturated`, and `partial` are not merely logged and
dropped: they land in the response body, which `bin-schedule-manager`
persists verbatim as `schedule_executions.result`
(`models/execution/execution.go`'s `Result string` field) on every pass
that returns an HTTP response — including `failed > 0` / `partial == true`
passes (`result` is populated only inside `ExecutionComplete`'s success
branch; a transport-level failure that never reaches this route leaves
`result` empty and `error` populated instead, with no `ReconcileResult` to
persist regardless). This is what makes the decision not to add a third
Prometheus counter for these signals safe, despite this service's own logs
rotating within hours (per VOIP-1443's investigation): the durable record
survives in `bin-schedule-manager`'s own audit trail.
