# Campaign Manager Metrics & Grafana Dashboard Design

## Date: 2026-02-12

## Problem Statement

The campaign-manager service had no custom Prometheus metrics or Grafana dashboard, limiting visibility into campaign lifecycle, campaigncall outcomes, and execution loop activity.

## Approach

Add 6 new Prometheus metrics across campaignhandler and campaigncallhandler packages, and create a Grafana dashboard with 5 rows and 13 panels.

## New Metrics

### campaignhandler

| Metric | Type | Description |
|--------|------|-------------|
| `campaign_manager_campaign_create_total` | Counter | Total campaigns created |
| `campaign_manager_campaign_status_run_total` | Counter | Total campaigns set to run status |
| `campaign_manager_campaign_status_stop_total` | Counter | Total campaigns stopped |
| `campaign_manager_campaign_execute_total` | Counter | Total campaign execution loops |

### campaigncallhandler

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `campaign_manager_campaigncall_create_total` | CounterVec | reference_type | Total campaigncalls created by type (call/flow) |
| `campaign_manager_campaigncall_done_total` | CounterVec | result | Total campaigncalls completed by result (success/fail) |

## Shared Metrics (from bin-common-handler/requesthandler)

- `campaign_manager_request_process_time` — RPC request processing duration
- `campaign_manager_event_publish_total` — Published events by type

## Instrumentation Points

- **campaignhandler/campaign.go Create()** — increments `campaign_create_total`
- **campaignhandler/status_run.go campaignRun()** — increments `campaign_status_run_total`
- **campaignhandler/status_stop.go campaignStopNow()** — increments `campaign_status_stop_total`
- **campaignhandler/execute.go Execute()** — increments `campaign_execute_total`
- **campaigncallhandler/campaigncall.go Create()** — increments `campaigncall_create_total` with reference_type label
- **campaigncallhandler/status.go Done()** — increments `campaigncall_done_total` with result label

## Grafana Dashboard

File: `monitoring/grafana/dashboards/campaign-manager.json`

### Rows & Panels

1. **Overview** (4 stat panels) — Campaigns created, started, stopped, execute loops
2. **Campaign Lifecycle** (3 panels) — Create rate, run vs stop rate, execute loop rate
3. **Campaigncall Lifecycle** (3 panels) — Create rate by reference type, done rate by result, success rate
4. **API & RPC Performance** (2 panels) — RPC processing time percentiles, request rate
5. **Events** (1 panel) — Event publish rate by type

## Files Changed

- `bin-campaign-manager/pkg/campaignhandler/main.go` — Added prometheus import, 4 metric vars, init()
- `bin-campaign-manager/pkg/campaignhandler/campaign.go` — Instrumented Create()
- `bin-campaign-manager/pkg/campaignhandler/status_run.go` — Instrumented campaignRun()
- `bin-campaign-manager/pkg/campaignhandler/status_stop.go` — Instrumented campaignStopNow()
- `bin-campaign-manager/pkg/campaignhandler/execute.go` — Instrumented Execute()
- `bin-campaign-manager/pkg/campaigncallhandler/main.go` — Added prometheus import, 2 metric vars, init()
- `bin-campaign-manager/pkg/campaigncallhandler/campaigncall.go` — Instrumented Create()
- `bin-campaign-manager/pkg/campaigncallhandler/status.go` — Instrumented Done()
- `monitoring/grafana/dashboards/campaign-manager.json` — New dashboard
