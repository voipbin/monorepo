# Billing Manager Metrics and Grafana Dashboard Design

## Problem Statement

The billing-manager service handles billing records, account balances, and payment validation. It had 5 registered metrics plus 2 unregistered ones (billing_create_total, billing_end_total, account_create_total were defined but never called prometheus.MustRegister). It also lacked duration tracking and balance validation metrics, with no Grafana dashboard.

## Approach

Fix 3 unregistered metrics, add 2 new metrics (billing duration, balance check outcomes), and create a Grafana dashboard (5 rows, 15 panels) visualizing billing operations, failed events, accounts, and API performance.

## Existing Metrics (Fixed: 3 were unregistered)

| Metric | Type | Labels | Handler | Status |
|--------|------|--------|---------|--------|
| `billing_create_total` | Counter | reference_type | billinghandler | Fixed (was unregistered) |
| `billing_end_total` | Counter | reference_type | billinghandler | Fixed (was unregistered) |
| `account_create_total` | Counter | none | accounthandler | Fixed (was unregistered) |
| `failed_event_save_total` | Counter | event_type, publisher | failedeventhandler | OK |
| `failed_event_retry_total` | Counter | result | failedeventhandler | OK |
| `failed_event_exhausted_total` | Counter | event_type | failedeventhandler | OK |
| `receive_request_process_time` | Histogram | type, method | listenhandler | OK |
| `receive_subscribe_event_process_time` | Histogram | publisher, type | subscribehandler | OK |

## New Metrics (2)

| Metric | Type | Labels | Location |
|--------|------|--------|----------|
| `billing_duration_seconds` | Histogram | reference_type | billinghandler/db.go UpdateStatusEnd() |
| `account_balance_check_total` | Counter | result | accounthandler/balance.go IsValidBalance() |

## Grafana Dashboard

Location: `monitoring/grafana/dashboards/billing-manager.json`

### Layout: 5 rows, 15 panels

| Row | Title | Panels |
|-----|-------|--------|
| 1 | Overview | Billings Created/min, Billings Ended/min, Failed Events/min, Balance Check Failures/min |
| 2 | Billing Records | Created by Type, Ended by Type, Duration p50/p95, Balance Check Results |
| 3 | Failed Events & Retry | Failed Events by Type, Retry Results, Exhausted by Type |
| 4 | Accounts | Account Creates/min, Balance Check Totals |
| 5 | API & Events | Request Processing Time p95, Event Processing Time p95 |

## Files Changed

- `bin-billing-manager/pkg/billinghandler/main.go` — Added billing_duration_seconds, registered existing metrics with MustRegister
- `bin-billing-manager/pkg/billinghandler/db.go` — Instrumented UpdateStatusEnd() with duration tracking
- `bin-billing-manager/pkg/accounthandler/main.go` — Added account_balance_check_total, registered existing account_create_total with MustRegister
- `bin-billing-manager/pkg/accounthandler/balance.go` — Instrumented IsValidBalance() with balance check counter
- `monitoring/grafana/dashboards/billing-manager.json` — New Grafana dashboard (5 rows, 15 panels)
