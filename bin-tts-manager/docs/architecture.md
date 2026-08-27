# bin-tts-manager Architecture

## Component Overview

`bin-tts-manager` provides text-to-speech synthesis in two modes: batch (pre-recorded file generation) and real-time streaming (live audio pumped to Asterisk via AudioSocket). It is a multi-container pod — a Go service and a Python HTTP sidecar share a `/shared-data` volume.

```
Pod: tts-manager
├── Container: tts-manager (Go)
│   ├── RabbitMQ listener (pkg/listenhandler)
│   ├── AudioSocket TCP server :8080 (pkg/streaminghandler)
│   └── Prometheus metrics :2112
└── Container: http-server (Python)
    └── HTTP file server :80  →  /shared-data
```

Key packages:

| Package | Role |
|---------|------|
| `pkg/listenhandler` | RabbitMQ RPC routing (batch + streaming control) |
| `pkg/ttshandler` | Batch TTS creation — synthesize and store |
| `pkg/streaminghandler` | Real-time AudioSocket + ElevenLabs WebSocket session management |
| `pkg/audiohandler` | Multi-provider TTS synthesis (GCP Cloud TTS, AWS Polly) |
| `pkg/buckethandler` | Local file storage for generated audio in `/shared-data` |
| `pkg/cachehandler` | Redis cache for TTS metadata |
| `pkg/dbhandler` | MySQL persistence |
| `models/tts` | Batch TTS structs |
| `models/streaming` | Streaming session structs |
| `pkg/notifyhandler` (shared) | Publishes events to the fanout exchange `bin-manager.tts-manager.event` and, since VOIP-1405, dual-publishes the same payload to the global topic exchange `bin-manager.event` |

`cmd/tts-manager` constructs its NotifyHandler with `notifyhandler.WithGlobalTopicPublish()`, so every event is published twice: once to the per-service fanout exchange `bin-manager.tts-manager.event` (unchanged, still the system of record) and once to the global topic exchange `bin-manager.event` with the routing key `tts-manager.<resource>.<subscription-id>.<action>`. A topic publish failure never propagates to the caller and never affects the fanout publish.

`cmd/tts-control` deliberately does NOT get the option: its NotifyHandler is injected only into `pkg/ttshandler`, which has zero publish sites, so there is no live publish path to mirror (VOIP-1405 §1.1; the dead dependency itself is tracked as a separate cleanup). See [docs/domain.md](domain.md) for the per-event routing keys and the monorepo `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md` for the schema.

## Layer Responsibilities

```
listenhandler       — routes RPC by URI+method regex; batch on shared queue, streaming on per-pod queue
    │
    ├─ ttshandler   — batch: synthesize text → audiohandler → buckethandler → store file
    │       │
    │       └─ audiohandler — provider selection: GCP (primary) → AWS Polly (fallback)
    │
    └─ streaminghandler — per-session goroutines; ElevenLabs WebSocket → AudioSocket TCP frames
            │
            └─ in-memory session map (mutex-protected)
```

## Request Routing

This service uses two queues simultaneously:

**Shared queue** `bin-manager.tts-manager.request` — batch TTS RPCs (any replica can handle):

| Method | URI Pattern | Handler |
|--------|-------------|---------|
| POST | `/v1/speeches` | `v1SpeechesPost` — synthesize speech (batch) |
| GET/POST | `/v1/speakings` | `v1SpeakingsGet/Post` — batch TTS operations |
| GET | `/v1/speakings/{uuid}` | `v1SpeakingsIDGet` — get TTS session |
| POST | `/v1/speakings/{uuid}/say` | `v1SpeakingsIDSayPost` |
| POST | `/v1/speakings/{uuid}/flush` | `v1SpeakingsIDFlushPost` |
| POST | `/v1/speakings/{uuid}/stop` | `v1SpeakingsIDStopPost` |

**Per-pod queue** `bin-manager.tts-manager.request.<HOSTNAME>` — streaming control (must reach the pod owning the session):

| Method | URI Pattern | Handler |
|--------|-------------|---------|
| POST/GET | `/v1/streamings` | `v1StreamingsPost/Get` |
| GET | `/v1/streamings/{uuid}` | `v1StreamingsIDGet` |
| POST | `/v1/streamings/{uuid}/say_add` | Append text chunk to session |
| POST | `/v1/streamings/{uuid}/say_init` | Initialize streaming TTS |
| POST | `/v1/streamings/{uuid}/say_stop` | Stop streaming TTS |
| POST | `/v1/streamings/{uuid}/say_finish` | Finish streaming TTS |

Asterisk dials into the Go service on TCP port 8080 (AudioSocket protocol) for media delivery to active streaming sessions.

See [docs/patterns/per-pod-queues.md](../docs/patterns/per-pod-queues.md) for the canonical per-pod queue pattern. Note: `bin-tts-manager` uses `HOSTNAME` as `HostID`, while `bin-transcribe-manager` uses a random UUID generated fresh at each process start (not `POD_IP`).
