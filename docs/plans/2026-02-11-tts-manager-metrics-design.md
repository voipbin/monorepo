# TTS Manager Metrics and Grafana Dashboard Design

## Problem Statement

The tts-manager service handles two modes of text-to-speech: batch TTS (GCP/AWS with file caching) and real-time streaming (ElevenLabs WebSocket). Currently, it has only basic infrastructure metrics (`hash_process_time`, `receive_request_process_time`) and no visibility into speech synthesis performance, streaming session lifecycle, or cache effectiveness.

## Approach

Add 10 new Prometheus metrics covering both batch TTS and streaming operations, plus a Grafana dashboard for visualization. Same approach as the flow-manager metrics work: operational health + business insight, no customer_id labels, JSON provisioning file.

## Metrics

### Batch TTS (ttshandler)

| Metric | Type | Labels | Location |
|--------|------|--------|----------|
| `speech_request_total` | Counter | `result` (cache_hit/created/error) | ttshandler/tts.go Create() |
| `speech_create_duration_seconds` | Histogram | none | ttshandler/tts.go Create() (cache misses only) |
| `speech_language_total` | Counter | `language`, `gender` | ttshandler/tts.go Create() |

### Streaming (streaminghandler)

| Metric | Type | Labels | Location |
|--------|------|--------|----------|
| `streaming_created_total` | Counter | `vendor` | streaminghandler/streaming.go Create() |
| `streaming_ended_total` | Counter | `vendor` | streaminghandler/streaming.go Delete() |
| `streaming_active` | Gauge | `vendor` | streaminghandler/streaming.go Create()/Delete() |
| `streaming_duration_seconds` | Histogram | `vendor` | streaminghandler/streaming.go Delete() |
| `streaming_message_total` | Counter | none | streaminghandler/say.go SayInit() |
| `streaming_error_total` | Counter | `vendor` | streaminghandler/stop.go Stop(), start.go Start() |
| `streaming_language_total` | Counter | `language`, `gender` | streaminghandler/streaming.go Create() |

### Notes

- `streaming_active` gauge resets to 0 on service restart and does not reflect persistent state.
- `streaming_created_total` uses `vendor=unknown` at creation time because the vendor is not yet determined until the streamer is initialized.
- `speech_create_duration_seconds` only measures cache misses (actual audio creation latency).
- Added `CreatedAt` field to the `streaming.Streaming` model to track session duration.

## Grafana Dashboard

Location: `monitoring/grafana/dashboards/tts-manager.json`

### Layout: 4 rows, 13 panels

| Row | Title | Panels |
|-----|-------|--------|
| 1 | Overview | Active Streams, Speech Requests/min, Cache Hit Rate %, Streaming Errors/min |
| 2 | Batch TTS | Speech Requests by Result, Speech Create Duration p95, Language Usage |
| 3 | Streaming Sessions | Sessions Created, Duration p50/p95, Errors |
| 4 | Throughput & API | Messages/min, Request Processing Time p95, Events Published |

### Existing metrics reused in dashboard
- `tts_manager_receive_request_process_time` (from listenhandler)
- `tts_manager_event_publish_total` (from requesthandler/bin-common-handler)

## Files Changed

- `bin-tts-manager/models/streaming/streaming.go` — Added `CreatedAt` field
- `bin-tts-manager/pkg/ttshandler/main.go` — Registered 3 new metrics
- `bin-tts-manager/pkg/ttshandler/tts.go` — Instrumented Create()
- `bin-tts-manager/pkg/streaminghandler/main.go` — Registered 7 new metrics
- `bin-tts-manager/pkg/streaminghandler/streaming.go` — Instrumented Create() and Delete()
- `bin-tts-manager/pkg/streaminghandler/start.go` — Instrumented Start() error path
- `bin-tts-manager/pkg/streaminghandler/stop.go` — Instrumented Stop() with error tracking
- `bin-tts-manager/pkg/streaminghandler/say.go` — Instrumented SayInit()
- `bin-tts-manager/pkg/streaminghandler/streaming_test.go` — Updated test for dynamic CreatedAt
- `monitoring/grafana/dashboards/tts-manager.json` — New Grafana dashboard
