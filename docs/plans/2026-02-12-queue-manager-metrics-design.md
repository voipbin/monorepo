# Queue Manager Metrics & Grafana Dashboard Design

## Date: 2026-02-12

## Problem Statement

The queue-manager service had no custom Prometheus metrics or Grafana dashboard, limiting visibility into queuecall lifecycle, wait times, abandonment rates, and overall queue performance.

## Approach

Add 4 new Prometheus metrics to the queuecallhandler package and create a Grafana dashboard with 5 rows and 14 panels.

## New Metrics

| Metric | Type | Location | Description |
|--------|------|----------|-------------|
| `queue_manager_queuecall_create_total` | Counter | queuecallhandler | Total queuecalls created |
| `queue_manager_queuecall_done_total` | Counter | queuecallhandler | Total queuecalls completed successfully |
| `queue_manager_queuecall_abandoned_total` | Counter | queuecallhandler | Total queuecalls abandoned |
| `queue_manager_queuecall_waiting_duration_seconds` | Histogram | queuecallhandler | Wait time before service (buckets: 1,5,10,30,60,120,300,600) |

## Shared Metrics (from bin-common-handler/requesthandler)

- `queue_manager_request_process_time` — RPC request processing duration
- `queue_manager_event_publish_total` — Published events by type

## Instrumentation Points

- **Create()** in `db.go` — increments `queuecall_create_total` after successful creation
- **UpdateStatusService()** in `db.go` — observes `queuecall_waiting_duration_seconds` with duration from TMCreate to now
- **UpdateStatusDone()** in `db.go` — increments `queuecall_done_total`
- **UpdateStatusAbandoned()** in `db.go` — increments `queuecall_abandoned_total`

## Grafana Dashboard

File: `monitoring/grafana/dashboards/queue-manager.json`

### Rows & Panels

1. **Overview** (4 stat panels) — Total created, done, abandoned, avg waiting duration
2. **Queuecall Lifecycle** (4 panels) — Create rate, done vs abandoned rate, success rate, abandonment rate
3. **Waiting Duration** (3 panels) — Average wait, percentiles (p50/p90/p99), histogram distribution
4. **API & RPC Performance** (2 panels) — RPC processing time percentiles, request rate
5. **Events** (1 panel) — Event publish rate by type

## Files Changed

- `bin-queue-manager/pkg/queuecallhandler/main.go` — Added prometheus import, 4 metric variables, init() registration
- `bin-queue-manager/pkg/queuecallhandler/db.go` — Instrumented Create, UpdateStatusService, UpdateStatusDone, UpdateStatusAbandoned
- `monitoring/grafana/dashboards/queue-manager.json` — New dashboard (5 rows, 14 panels)
