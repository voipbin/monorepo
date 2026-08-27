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
| `pkg/notifyhandler` | Publishes events to the fanout exchange `bin-manager.transcribe-manager.event` and, since VOIP-1404, dual-publishes the same payload to the global topic exchange `bin-manager.event` |
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

Both `cmd/transcribe-manager` and `cmd/transcribe-control` construct their NotifyHandler with `notifyhandler.WithGlobalTopicPublish()`, so every event is published twice: once to the per-service fanout exchange `bin-manager.transcribe-manager.event` (unchanged, still the system of record) and once to the global topic exchange `bin-manager.event` with the routing key `transcribe-manager.<resource>.<transcribe-id>.<action>`. The two cmds must stay in lockstep on this option — enabling it in only one would leave consumers with gaps depending on which process published. A topic publish failure never propagates to the caller and never affects the fanout publish. See [docs/domain.md](domain.md) for the per-event routing keys and `docs/plans/` (monorepo `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md`) for the schema.

## Request Routing

This service uses two queues simultaneously:

**Shared queue** `bin-manager.transcribe-manager.request` — requests that any replica can handle:

| Method | URI Pattern | Handler |
|--------|-------------|---------|
| POST | `/v1/transcribes` | `v1TranscribesPost` — start a transcription session |
| GET | `/v1/transcribes?` | `v1TranscribesGet` — list transcription sessions |
| GET | `/v1/transcribes/{uuid}` | `v1TranscribesIDGet` — get session by ID |
| GET | `/v1/transcripts?` | `v1TranscriptsGet` — list transcript segments |

**Per-pod queue** `bin-manager.transcribe-manager-<host_id>.request` — must reach the pod owning the session:

| Method | URI Pattern | Handler |
|--------|-------------|---------|
| GET | `/v1/transcribes/{uuid}/health-check` | Session liveness check |
| POST | `/v1/transcribes/{uuid}/stop` | Stop active streaming session |

The `host_id` is a random UUID (`uuid.Must(uuid.NewV4())`) generated fresh in `cmd/transcribe-manager/main.go` each time the process starts — it is not `POD_IP` or any other pod identity. It is stored on the session record so callers can route per-pod RPCs using this value, but it changes on every process restart, not just when the underlying pod IP changes.

See [docs/patterns/per-pod-queues.md](../docs/patterns/per-pod-queues.md) for the general per-pod queue pattern (naming convention, volatile queue declaration). That doc's example identity source is `POD_IP`; this service instead uses a process-lifetime random UUID as its identity source.
