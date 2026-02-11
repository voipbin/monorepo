# AI Manager Metrics and Grafana Dashboard Design

## Problem Statement

The ai-manager service orchestrates AI-powered conversations, managing AIcall sessions, LLM message processing, tool execution, and summary generation. It has 6 existing custom metrics but is missing session lifecycle metrics (end/duration), tool execution tracking, and summary metrics. It also has no Grafana dashboard.

## Approach

Add 5 new Prometheus metrics to fill gaps in session lifecycle, tool execution, and summary tracking. Create a Grafana dashboard (5 rows, 17 panels) visualizing both existing and new metrics.

## Existing Metrics (6)

| Metric | Type | Labels | Handler |
|--------|------|--------|---------|
| `ai_create_total` | Counter | engine_type | aihandler |
| `aicall_create_total` | Counter | reference_type | aicallhandler |
| `aicall_init_process_time` | Histogram | engine_type | aicallhandler |
| `aicall_message_process_time` | Histogram | engine_type | aicallhandler |
| `message_create_total` | Counter | engine_type | messagehandler |
| `message_process_time` | Histogram | engine_type | messagehandler |

## New Metrics (5)

| Metric | Type | Labels | Location |
|--------|------|--------|----------|
| `aicall_end_total` | Counter | reference_type | aicallhandler/db.go UpdateStatus() at StatusTerminated |
| `aicall_duration_seconds` | Histogram | reference_type | aicallhandler/db.go UpdateStatus() at StatusTerminated |
| `aicall_tool_execute_total` | Counter | tool_name | aicallhandler/tool.go ToolHandle() |
| `summary_start_total` | Counter | reference_type | summaryhandler/start.go Start() |
| `summary_done_total` | Counter | reference_type | summaryhandler/db.go UpdateStatusDone() and Create() with StatusDone |

### Notes

- `aicall_duration_seconds` uses TMCreate from the AIcall model, measured at StatusTerminated transition.
- `summary_done_total` fires both in UpdateStatusDone() (for call/conference async completions) and in Create() when status is directly set to Done (for transcribe/recording synchronous completions).
- Tool execution counter tracks all tool names including connect_call, send_email, send_message, stop_service, etc.

## Grafana Dashboard

Location: `monitoring/grafana/dashboards/ai-manager.json`

### Layout: 5 rows, 17 panels

| Row | Title | Panels |
|-----|-------|--------|
| 1 | Overview | AIcalls/min, Messages/min, Tool Executions/min, Summaries/min |
| 2 | AIcalls | Created by Reference Type, Ended by Reference Type, Duration p50/p95, Init Time p95 |
| 3 | Messages & Tools | Messages by Engine Type, Message Process Time p95, Tool Executions by Tool Name, AIcall Message Process Time p95 |
| 4 | Summaries & AI Config | Summary Starts, Summary Completions, AI Configs by Engine Type |
| 5 | API & Events | Request Processing Time p95, Event Processing Time p95 |

## Files Changed

- `bin-ai-manager/pkg/aicallhandler/main.go` — Added `aicall_end_total`, `aicall_duration_seconds`, `aicall_tool_execute_total`
- `bin-ai-manager/pkg/aicallhandler/db.go` — Instrumented UpdateStatus() at StatusTerminated with end counter and duration
- `bin-ai-manager/pkg/aicallhandler/tool.go` — Instrumented ToolHandle() with tool execution counter
- `bin-ai-manager/pkg/summaryhandler/main.go` — Added `summary_start_total` and `summary_done_total` counters
- `bin-ai-manager/pkg/summaryhandler/start.go` — Instrumented Start()
- `bin-ai-manager/pkg/summaryhandler/db.go` — Instrumented UpdateStatusDone() and Create()
- `monitoring/grafana/dashboards/ai-manager.json` — New Grafana dashboard (5 rows, 17 panels)
