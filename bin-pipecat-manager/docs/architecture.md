# bin-pipecat-manager Architecture

## Component Overview

`bin-pipecat-manager` is a hybrid Go/Python service that executes real-time AI voice pipelines. Go owns transport, session lifecycle, and database persistence; Python owns the Pipecat pipeline (STT → LLM → TTS).

```
cmd/pipecat-manager/main.go  (Go, port 8080)
    ├── pkg/dbhandler              (MySQL + Redis cache)
    ├── pkg/cachehandler           (Redis operations)
    ├── pkg/listenhandler          (RabbitMQ RPC router — shared + per-pod queues)
    ├── pkg/httphandler            (HTTP endpoints: WebSocket upgrades + tool callbacks)
    ├── pkg/pipecatcallhandler/
    │     ├── start.go             (session startup sequence)
    │     ├── session.go           (in-memory session registry)
    │     ├── websocket.go         (Asterisk external media + Pipecat client WebSocket)
    │     ├── audiosocket.go       (safety-net resampler for non-16 kHz audio)
    │     ├── pipecatframe.go      (protobuf frame ser/de)
    │     ├── pythonrunner.go      (HTTP client → Python FastAPI)
    │     ├── run.go, runner.go    (pipeline coordination)
    │     └── mock_*.go            (gomock generated mocks)
    └── pkg/toolhandler            (LLM tool callbacks dispatched to bin-ai-manager)

scripts/pipecat/  (Python, port 8000)
    ├── main.py       (FastAPI server — /run and /stop endpoints)
    ├── run.py        (Pipecat pipeline construction and execution)
    ├── tools.py      (LLM function-calling tools)
    ├── task.py       (async task lifecycle)
    └── common.py     (shared utilities)
```

**Supporting binaries:**
- `cmd/pipecat-control/` — CLI for direct DB/cache operations

**Protobuf:**
- `proto/frames.proto` → `models/pipecatframe/frames.pb.go` (Go) + pipecat Python serializer

## Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|----------------|
| Transport (Go) | `pkg/listenhandler` | Shared queue: create/get pipecatcall. Per-pod queue: stop, message-send, ping |
| Transport (Go) | `pkg/httphandler` | WebSocket upgrade for Pipecat Python; tool execution callbacks from Python |
| Session (Go) | `pkg/pipecatcallhandler` | In-memory session registry; audio bridge between Asterisk and Python |
| Pipeline (Python) | `scripts/pipecat/` | STT → LLM → TTS execution via pipecat-ai library |
| Data (Go) | `pkg/dbhandler` | MySQL: pipecatcall records; Redis: session cache |

## Request Routing

ListenHandler routes over two queues:

**Shared queue** `bin-manager.pipecat-manager.request` (standard RabbitMQ, survives pod restarts):

| Pattern | Method | Purpose |
|---------|--------|---------|
| `/v1/pipecatcalls$` | POST | Create pipecatcall — starts a new session |
| `/v1/pipecatcalls/{{UUID}}$` | GET | Get pipecatcall by ID |

**Per-pod queue** `bin-manager.pipecat-manager.request.<POD_IP>` (volatile, auto-deletes on pod death):

| Pattern | Method | Purpose |
|---------|--------|---------|
| `/v1/pipecatcalls/{{UUID}}/stop$` | POST | Terminate pipecatcall (must reach owning pod) |
| `/v1/messages$` | POST | Send message into active session |
| `/v1/ping$` | GET | Sub-second liveness probe (no DB I/O) |

`HostID = POD_IP` (from K8s Downward API) is persisted on `pipecatcall.HostID` so `bin-ai-manager` can route follow-up RPCs to the correct pod. See [docs/patterns/per-pod-queues.md](../../docs/patterns/per-pod-queues.md) and [docs/patterns/per-pod-liveness-preflight.md](../../docs/patterns/per-pod-liveness-preflight.md).

## Session Lifecycle

```
POST /v1/pipecatcalls
    ↓ create DB record
    ↓ RPC to call-manager → get external media URI
    ↓ Go dials Asterisk WebSocket at MediaURI
    ↓ wait for MEDIA_START text message
    ↓ HTTP POST localhost:8000/run → Python starts Pipecat pipeline
    ↓ Python connects back to Go via WebSocket (frames.proto)
    ↓ bidirectional audio + tool call frames
    ↓ ConnAstDone closes on Asterisk disconnect → cleanup
```

Cleanup is also triggered by `POST /v1/pipecatcalls/<id>/stop` (per-pod queue).

## Audio Architecture

- Asterisk ↔ Go: 16 kHz slin16 via WebSocket external media (`chan_websocket`)
- Go ↔ Python: protobuf `AudioRawFrame` over WebSocket
- Python Pipecat: `audio_out_sample_rate=16000` in `PipelineParams` — zero resampling on the hot path
- Safety net: `audiosocketHandler.GetDataSamples()` for unexpected non-16 kHz input

**Unpaced delivery + FLUSH_MEDIA barge-in:** `UnpacedWebsocketClientOutputTransport` sends TTS audio faster than real-time (Asterisk re-times internally). On barge-in, Python sends `TextFrame("FLUSH_MEDIA")` → Go forwards as WebSocket text to Asterisk → instant buffer discard.

## Event Subscriptions

This service does **not** subscribe to external RabbitMQ events. There is no SubscribeHandler.

## Events Published

Exchange: `bin-manager.pipecat-manager.event` — pipecat session lifecycle events consumed by `bin-ai-manager`.
