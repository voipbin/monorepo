# bin-transcribe-manager Architecture

## Component Overview

`bin-transcribe-manager` provides real-time speech-to-text transcription for VoIP calls and conferences. It integrates with Google Cloud Speech-to-Text and AWS Transcribe, maintains per-pod in-memory streaming sessions, and uses per-pod RabbitMQ queue routing to direct session-specific RPCs to the owning pod.

```
cmd/transcribe-manager/main.go
    ├── MySQL connection (pkg/dbhandler)
    ├── Redis cache connection
    ├── RabbitMQ connection (sockhandler)
    ├── runServiceListen()     → pkg/listenhandler (shared + per-pod queues)
    ├── runServiceSubscribe()  → pkg/subscribehandler
    ├── runServiceStream()     → pkg/streaminghandler (WebSocket transport)
    └── Prometheus metrics endpoint (:2112)
```

Key packages:

| Package | Role |
|---------|------|
| `pkg/listenhandler` | RabbitMQ RPC routing (shared queue + per-pod queue) |
| `pkg/subscribehandler` | Consumes call-manager and customer-manager events for cleanup |
| `pkg/streaminghandler` | WebSocket connections to Asterisk; in-memory session map |
| `pkg/transcribehandler` | Core business logic — session creation, status transitions |
| `pkg/dbhandler` | MySQL + Redis persistence |
| `pkg/notifyhandler` | Publishes events to `bin-manager.transcribe-manager.event` |
| `models/transcribe` | Transcribe session struct, status enum |
| `internal/config` | Cobra + Viper configuration (singleton pattern) |

## Layer Responsibilities

```
listenhandler        — deserializes RPC, routes by URI+method regex
subscribehandler     — call_hangup → finalize session; customer_deleted → cascade cleanup
    │
    └─ transcribehandler — creates sessions, drives status transitions
            │
            ├─ dbhandler       — MySQL (sessions, transcripts) + Redis cache
            ├─ streaminghandler — in-memory session map (mutex-protected)
            └─ notifyhandler   — publishes events on state changes
```

## Request Routing

This service uses two queues simultaneously:

**Shared queue** `bin-manager.transcribe-manager.request` — requests that any replica can handle:

| Method | URI Pattern | Handler |
|--------|-------------|---------|
| POST | `/v1/transcribes` | `v1TranscribesPost` — start a transcription session |
| GET | `/v1/transcribes?` | `v1TranscribesGet` — list transcription sessions |
| GET | `/v1/transcribes/{uuid}` | `v1TranscribesIDGet` — get session by ID |
| GET | `/v1/transcripts?` | `v1TranscriptsGet` — list transcript segments |

**Per-pod queue** `bin-manager.transcribe-manager.request.<host_id>` — must reach the pod owning the session:

| Method | URI Pattern | Handler |
|--------|-------------|---------|
| GET | `/v1/transcribes/{uuid}/health-check` | Session liveness check |
| POST | `/v1/transcribes/{uuid}/stop` | Stop active streaming session |

The `host_id` is `POD_IP` (from Kubernetes Downward API), stored on the session record. Callers route per-pod RPCs using this value.

See [docs/patterns/per-pod-queues.md](../docs/patterns/per-pod-queues.md) for the canonical per-pod queue pattern (queue naming, identity source, Calico POD_IP recycle limitation).
