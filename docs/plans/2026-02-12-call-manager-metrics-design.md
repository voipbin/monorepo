# Call Manager Metrics and Grafana Dashboard Design

## Problem Statement

The call-manager service is the core telephony operations hub handling calls, channels, conference bridges, recordings, and external media. It already has 15 existing custom metrics but is missing duration histograms, recording metrics, and external media metrics. It also has no Grafana dashboard for visualization.

## Approach

Add 6 new Prometheus metrics to fill the gaps in existing instrumentation, plus a comprehensive Grafana dashboard that visualizes both existing and new metrics. Same approach as flow-manager and tts-manager: operational health + business insight, no customer_id labels, JSON provisioning file.

## Existing Metrics (15)

| Metric | Type | Labels | Handler |
|--------|------|--------|---------|
| `call_create_total` | Counter | direction, type | callhandler |
| `call_hangup_total` | Counter | direction, type, reason | callhandler |
| `call_action_total` | Counter | type | callhandler |
| `call_action_process_time` | Histogram | type | callhandler |
| `conference_leave_total` | Counter | type | callhandler |
| `channel_create_total` | Counter | direction, tech | channelhandler |
| `channel_hangup_total` | Counter | direction, type, reason | channelhandler |
| `channel_transport_direction_total` | Counter | transport, direction | channelhandler |
| `confbridge_create_total` | Counter | none | confbridgehandler |
| `confbridge_close_total` | Counter | none | confbridgehandler |
| `confbridge_join_total` | Counter | none | confbridgehandler |
| `bridge_create_total` | Counter | reference_type | bridgehandler |
| `bridge_destroy_total` | Counter | reference_type | bridgehandler |
| `groupcall_create_total` | Counter | none | groupcallhandler |
| `subscribe_event_process_time` | Histogram | publisher, type | subscribehandler |

## New Metrics (6)

| Metric | Type | Labels | Location |
|--------|------|--------|----------|
| `call_duration_seconds` | Histogram | direction, type | callhandler/db.go UpdateHangupInfo() |
| `confbridge_duration_seconds` | Histogram | type | confbridgehandler/db.go UpdateStatus() |
| `recording_start_total` | Counter | reference_type | recordinghandler/start.go Start() |
| `recording_end_total` | Counter | reference_type | recordinghandler/stop.go Stopped() |
| `external_media_start_total` | Counter | reference_type, encapsulation | externalmediahandler/start.go Start() |
| `external_media_stop_total` | Counter | reference_type | externalmediahandler/stop.go Stop() |

### Notes

- `call_duration_seconds` uses TMCreate and TMHangup timestamps from the call model (database-persisted).
- `confbridge_duration_seconds` uses TMCreate from the confbridge model, measured at StatusTerminated.
- Recording and external media metrics are simple counters since these operations are initiated by other services.

## Grafana Dashboard

Location: `monitoring/grafana/dashboards/call-manager.json`

### Layout: 5 rows, 18 panels

| Row | Title | Panels |
|-----|-------|--------|
| 1 | Overview | Calls/min, Active Channels, Conference Bridges/min, ARI Events/min |
| 2 | Calls | Created by Direction, Hangup by Reason, Duration p50/p95, Actions |
| 3 | Conference & Recording | Confbridge Created/Closed, Confbridge Duration, Recording Start/End, External Media |
| 4 | Channels & Infrastructure | Channels by Tech, Groupcall Created, Bridge Create/Destroy |
| 5 | API & Events | Request Processing Time p95, ARI Event Processing Time p95, Events Published |

## Files Changed

- `bin-call-manager/pkg/callhandler/main.go` — Added `call_duration_seconds` histogram
- `bin-call-manager/pkg/callhandler/db.go` — Instrumented UpdateHangupInfo() with duration tracking
- `bin-call-manager/pkg/confbridgehandler/main.go` — Added `confbridge_duration_seconds` histogram
- `bin-call-manager/pkg/confbridgehandler/db.go` — Instrumented UpdateStatus() with duration tracking
- `bin-call-manager/pkg/recordinghandler/main.go` — Added `recording_start_total` and `recording_end_total` counters
- `bin-call-manager/pkg/recordinghandler/start.go` — Instrumented Start()
- `bin-call-manager/pkg/recordinghandler/stop.go` — Instrumented Stopped()
- `bin-call-manager/pkg/externalmediahandler/main.go` — Added `external_media_start_total` and `external_media_stop_total` counters
- `bin-call-manager/pkg/externalmediahandler/start.go` — Instrumented Start()
- `bin-call-manager/pkg/externalmediahandler/stop.go` — Instrumented Stop()
- `monitoring/grafana/dashboards/call-manager.json` — New Grafana dashboard (5 rows, 18 panels)
