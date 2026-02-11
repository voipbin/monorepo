# Flow Manager Metrics & Grafana Dashboard Design

## Problem Statement

Flow-manager currently has only 4 metrics (2 service-specific, 2 shared from bin-common-handler).
These cover basic request/response timing but miss the flow execution engine, activeflow lifecycle,
action throughput, error breakdown, and service dependency visibility. There is also no Grafana
dashboard to visualize any of these.

## Goals

1. Add 9 new Prometheus metrics covering operational health and business insight
2. Create a Grafana dashboard JSON provisioning file with 16 panels across 5 rows
3. No customer_id labels (keep cardinality low)

## Existing Metrics (4)

| Metric | Type | Labels | Location |
|--------|------|--------|----------|
| `flow_manager_action_exeucte_duration` | Histogram | `type` | activeflowhandler/main.go |
| `flow_manager_receive_request_process_time` | Histogram | `type`, `method` | listenhandler/main.go |
| `flow_manager_request_process_time` (shared) | Histogram | `target`, `resource`, `method` | bin-common-handler requesthandler |
| `flow_manager_event_publish_total` (shared) | Counter | `event_type` | bin-common-handler requesthandler |

## New Metrics (9)

### Operational Health

| # | Metric Name | Type | Labels | Purpose | Instrumentation Point |
|---|-------------|------|--------|---------|----------------------|
| 1 | `activeflow_created_total` | Counter | `reference_type` | Track activeflow creation rate | `activeflowhandler.Create()` |
| 2 | `activeflow_ended_total` | Counter | `reference_type` | Track activeflow termination rate | `activeflowhandler.Stop()` |
| 3 | `activeflow_running` | Gauge | `reference_type` | Currently running activeflows | Inc in Create(), Dec in Stop() |
| 4 | `action_executed_total` | Counter | `type` | Action throughput by type | `executeAction()` |
| 5 | `action_error_total` | Counter | `type` | Fatal action errors that stop the flow | `executeAction()` on error |
| 6 | `activeflow_duration_seconds` | Histogram | `reference_type` | Total activeflow lifetime | `activeflowhandler.Stop()` |

### Business Insight

| # | Metric Name | Type | Labels | Purpose | Instrumentation Point |
|---|-------------|------|--------|---------|----------------------|
| 7 | `action_dispatch_total` | Counter | `target`, `type` | Actions dispatched to external services | `executeAction()` for non-flow actions |
| 8 | `activeflow_execute_iterations` | Histogram | `reference_type` | Actions per execute loop iteration | `ExecuteNextAction()` loop exit |
| 9 | `flow_crud_total` | Counter | `operation` | Flow template CRUD ops | listenhandler CRUD endpoints |

### Metric Details

**Namespace:** `flow_manager` (all metrics)

**Histogram Buckets:**
- `activeflow_duration_seconds`: [0.1, 0.5, 1, 5, 10, 30, 60, 120, 300, 600] (100ms to 10min)
- `activeflow_execute_iterations`: [1, 2, 5, 10, 20, 50, 100, 200, 500, 1000]

**`activeflow_ended_total`:** The `reason` label was dropped during implementation because
all stop paths go through a single `Stop()` method, making it non-trivial to distinguish the cause
without changing the interface. The `reference_type` label is sufficient for operational monitoring.

**`action_error_total`:** Only tracks fatal errors that stop the flow. Non-critical actions
(email_send, webhook_send, conversation_send, etc.) swallow errors and continue the flow,
so they are not counted.

**`flow_crud_total` operation values:**
- `create`, `update`, `delete`

### Collision Check

None of these metric names conflict with the shared metrics from
`bin-common-handler/pkg/requesthandler/main.go` (`request_process_time`, `event_publish_total`).

## Implementation Plan

### File Changes

**1. `bin-flow-manager/pkg/activeflowhandler/main.go`** — Register new metrics

Add metric variables alongside the existing `actionExecuteDuration`:
- `activeflowCreatedTotal` (CounterVec)
- `activeflowEndedTotal` (CounterVec)
- `activeflowRunning` (GaugeVec)
- `actionExecutedTotal` (CounterVec)
- `actionErrorTotal` (CounterVec)
- `activeflowDurationSeconds` (HistogramVec)
- `actionDispatchTotal` (CounterVec)
- `activeflowExecuteIterations` (HistogramVec)

Register them in the `init()` function or alongside the existing `prometheus.MustRegister` calls.

**2. `bin-flow-manager/pkg/activeflowhandler/activeflow.go`** — Instrument Create/Stop

In `Create()`:
- Increment `activeflowCreatedTotal` with reference_type label
- Increment `activeflowRunning` gauge

In `Stop()`:
- Increment `activeflowEndedTotal` with reference_type and reason labels
- Decrement `activeflowRunning` gauge
- Observe `activeflowDurationSeconds` (compute from TMCreate to now)

**3. `bin-flow-manager/pkg/activeflowhandler/execute.go`** — Instrument execution loop

In `executeAction()`:
- Increment `actionExecutedTotal` with type label
- On error: increment `actionErrorTotal` with type label
- For non-flow actions (dispatched to other services): increment `actionDispatchTotal` with target and type labels

In `ExecuteNextAction()`:
- Count loop iterations, observe `activeflowExecuteIterations` at loop exit

**4. `bin-flow-manager/pkg/listenhandler/` (v1_flow*.go files)** — Instrument CRUD

In flow create/update/delete handlers:
- Increment `flowCRUDTotal` with operation label

Register `flowCRUDTotal` metric in `listenhandler/main.go`.

**5. `monitoring/grafana/dashboards/flow-manager.json`** — Grafana dashboard

New file with the dashboard JSON provisioning definition. This establishes the standard
location for all Grafana dashboards in the monorepo: `monitoring/grafana/dashboards/<service-name>.json`.

### Grafana Dashboard Layout

**Dashboard Title:** "Flow Manager"
**Refresh:** 30s
**Time Range:** Last 1 hour

#### Row 1 — Overview (4 stat panels)

| Panel | Type | Query |
|-------|------|-------|
| Active Flows | Stat | `sum(flow_manager_activeflow_running)` |
| Activeflows Created/min | Stat | `sum(rate(flow_manager_activeflow_created_total[5m])) * 60` |
| Actions Executed/min | Stat | `sum(rate(flow_manager_action_executed_total[5m])) * 60` |
| Action Error Rate % | Stat | `sum(rate(flow_manager_action_error_total[5m])) / sum(rate(flow_manager_action_executed_total[5m])) * 100` |

#### Row 2 — Activeflow Lifecycle (3 panels)

| Panel | Type | Query |
|-------|------|-------|
| Activeflows Created (by ref type) | Time Series | `sum by (reference_type) (rate(flow_manager_activeflow_created_total[5m]))` |
| Activeflows Ended (by ref type) | Time Series | `sum by (reference_type) (rate(flow_manager_activeflow_ended_total[5m]))` |
| Activeflow Duration (p50/p95/p99) | Time Series | `histogram_quantile(0.5/0.95/0.99, sum by (le) (rate(flow_manager_activeflow_duration_seconds_bucket[5m])))` |

#### Row 3 — Action Performance (3 panels)

| Panel | Type | Query |
|-------|------|-------|
| Action Execution Rate (by type) | Time Series | `sum by (type) (rate(flow_manager_action_executed_total[5m]))` |
| Action Duration p95 (by type) | Time Series | `histogram_quantile(0.95, sum by (le, type) (rate(flow_manager_action_exeucte_duration_bucket[5m])))` |
| Action Errors (by type) | Time Series | `sum by (type) (rate(flow_manager_action_error_total[5m]))` |

#### Row 4 — Service Dependencies (3 panels)

| Panel | Type | Query |
|-------|------|-------|
| Dispatches to External Services | Time Series | `sum by (target) (rate(flow_manager_action_dispatch_total[5m]))` |
| Outbound RPC Latency p95 | Time Series | `histogram_quantile(0.95, sum by (le, target) (rate(flow_manager_request_process_time_bucket[5m])))` |
| Execute Loop Iterations (p50/p95) | Time Series | `histogram_quantile(0.5/0.95, sum by (le) (rate(flow_manager_activeflow_execute_iterations_bucket[5m])))` |

#### Row 5 — API & Flow Templates (3 panels)

| Panel | Type | Query |
|-------|------|-------|
| Request Processing Time p95 | Time Series | `histogram_quantile(0.95, sum by (le, type) (rate(flow_manager_receive_request_process_time_bucket[5m])))` |
| Flow CRUD Operations | Time Series | `sum by (operation) (rate(flow_manager_flow_crud_total[5m]))` |
| Events Published | Time Series | `sum by (event_type) (rate(flow_manager_event_publish_total[5m]))` |

## Testing

- Unit tests for metric registration (verify no panics on duplicate registration)
- Verify metrics appear on `/metrics` endpoint after service startup
- Run `go test ./...` and `golangci-lint run -v --timeout 5m` for bin-flow-manager

## Risks / Notes

- The existing metric `action_exeucte_duration` has a typo ("exeucte" instead of "execute").
  We will NOT rename it to avoid breaking existing dashboards/alerts. New metrics use correct spelling.
- The `activeflow_running` gauge may drift if the service restarts with activeflows still in the database.
  This is acceptable — the gauge tracks in-process activeflows, not database state.
- All new metric names are verified to not conflict with bin-common-handler shared metrics.
