# Conference Manager Metrics & Grafana Dashboard Design

## Date: 2026-02-12

## Problem Statement

The conference-manager service already has 4 custom Prometheus metrics but lacked a Grafana dashboard for visualization.

## Existing Metrics

### conferencehandler

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `conference_manager_conference_create_total` | CounterVec | type | Total conferences created by type |
| `conference_manager_conference_close_total` | CounterVec | type | Total conferences closed by type |
| `conference_manager_conference_join_total` | CounterVec | type | Total conference joins by type (defined but not yet instrumented) |

### conferencecallhandler

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `conference_manager_conferencecall_total` | CounterVec | reference_type, status | Total conferencecalls by reference type and status |

## Shared Metrics (from bin-common-handler/requesthandler)

- `conference_manager_request_process_time` — RPC request processing duration
- `conference_manager_event_publish_total` — Published events by type

## Grafana Dashboard

File: `monitoring/grafana/dashboards/conference-manager.json`

### Rows & Panels

1. **Overview** (3 stat panels) — Total conferences created, closed, conferencecalls
2. **Conference Lifecycle** (2 panels) — Create rate by type, close rate by type
3. **Conferencecall Lifecycle** (2 panels) — Rate by reference type, rate by status
4. **API & RPC Performance** (2 panels) — RPC processing time percentiles, request rate
5. **Events** (1 panel) — Event publish rate by type

## Files Changed

- `monitoring/grafana/dashboards/conference-manager.json` — New dashboard (5 rows, 10 panels)
