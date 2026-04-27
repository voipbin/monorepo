# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-pipecat-manager` is a hybrid Go/Python microservice that manages AI-powered voice conversations in VoIPbin. It integrates Pipecat (a Python framework for voice AI pipelines) with Go-based infrastructure, coordinating real-time audio streaming, LLM interactions, STT (Speech-to-Text), and TTS (Text-to-Speech) services.

**Key Concepts:**
- **Pipecatcall** — a single AI voice session: a row in MySQL plus an in-memory session bound to one pod
- **Per-pod ownership** — every pipecatcall is anchored to exactly one pod (`HostID = POD_IP`); follow-up RPCs (message-send, terminate, ping) target a per-pod queue
- **Hybrid Go/Python** — Go owns transport, lifecycle, and DB; Python owns the Pipecat pipeline (STT → LLM → TTS) and connects back over WebSocket
- **End-to-end 16 kHz audio** — Asterisk slin16 ↔ Pipecat with no resampling on the hot path

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-pipecat-manager`.

## Architecture

### Hybrid Go/Python Design

1. **Go service (port 8080)** — main orchestration layer
   - HTTP endpoints for WebSocket connections from Pipecat Python runners
   - RabbitMQ RPC handler for creating/managing pipecat calls
   - WebSocket external media connection to Asterisk for real-time audio streaming
   - Session lifecycle management and state coordination
   - Database and cache operations

2. **Python service (port 8000)** — Pipecat pipeline executor
   - FastAPI server with `/run` and `/stop` endpoints
   - Executes Pipecat AI voice pipelines (STT → LLM → TTS)
   - WebSocket client connecting back to the Go service
   - Async task management for concurrent pipeline executions

### Service Layer Structure

**Go packages:**

```
cmd/pipecat-manager/main.go
    ├── Database (MySQL)
    ├── Cache (Redis via pkg/cachehandler)
    └── run()
        ├── pkg/dbhandler            (MySQL operations with Redis caching)
        ├── pkg/cachehandler         (Redis cache operations)
        ├── pkg/pipecatcallhandler   (Core orchestration logic)
        │     ├── pythonrunner.go    (HTTP client → Python FastAPI)
        │     ├── audiosocket.go     (resampler safety net for non-16 kHz audio)
        │     ├── websocket.go       (WebSocket: Asterisk external media + Pipecat client)
        │     ├── pipecatframe.go    (protobuf frame ser/de)
        │     ├── session.go         (active pipecat call sessions)
        │     ├── start.go           (call startup sequence)
        │     └── run.go, runner.go  (pipeline execution coordination)
        ├── pkg/toolhandler          (LLM tool callbacks)
        ├── pkg/httphandler          (HTTP endpoints for WebSocket upgrades + tool callbacks)
        └── pkg/listenhandler        (RabbitMQ RPC handler with regex-based routing)
```

**Models:**
- `models/pipecatcall/` — data structures for pipecat sessions
- `models/pipecatframe/` — protobuf frame definitions (generated from `proto/frames.proto`)
- `models/message/` — message structures for pipecat interactions

**Python scripts:**
- `scripts/pipecat/main.py` — FastAPI server entry point
- `scripts/pipecat/run.py` — Pipecat pipeline construction and execution
- `scripts/pipecat/tools.py` — LLM function-calling tools (`connect_call`, `send_email`, etc.)
- `scripts/pipecat/task.py` — async task lifecycle management
- `scripts/pipecat/common.py` — shared utilities

### Inter-Service Communication

- **RabbitMQ:**
  - Listens on `bin-manager.pipecat-manager.request` (shared, normal queue) — call creation, lookup
  - Listens on `bin-manager.pipecat-manager.request.<POD_IP>` (volatile, per-pod queue) — message-send, terminate, ping
  - Publishes events to `bin-manager.pipecat-manager.event`
- **HTTP/WebSocket:**
  - Go → Python: HTTP POST to `localhost:8000/run` and `/stop`
  - Python → Go: WebSocket client to `<POD_IP>:8080` for bidirectional frame streaming
- **WebSocket external media:** Go dials Asterisk's `chan_websocket` endpoint for 16 kHz slin16 audio streaming
- **Monorepo:** sibling services referenced via `replace` directives in `go.mod` pointing to `../bin-*-manager`

## Request Routing

ListenHandler routes RPC requests using regex patterns matching REST-like URIs (see `pkg/listenhandler/main.go`):

**Pipecatcall API (`/v1/pipecatcalls/*`):**
- `POST /v1/pipecatcalls` — create a pipecatcall (start a session). Served on the **shared** queue.
- `GET /v1/pipecatcalls/<uuid>` — get a pipecatcall by ID. Served on the **shared** queue.
- `POST /v1/pipecatcalls/<uuid>/stop` — terminate a pipecatcall. Served on the **per-pod** queue.

**Message API (`/v1/messages`):**
- `POST /v1/messages` — send a message into an active pipecatcall. Served on the **per-pod** queue.

**Liveness preflight (`/v1/ping`):**
- `GET /v1/ping` — sub-second liveness probe. Served on the **per-pod** queue. Returns process identity (`HostID`, `Timestamp`) only — no DB I/O. See [docs/patterns/per-pod-liveness-preflight.md](../docs/patterns/per-pod-liveness-preflight.md).

### Per-pod queue routing

This service uses the per-pod RabbitMQ queue convention. Operations that must reach the specific pod owning the in-memory session are routed to `<service>.request.<host_id>` (declared volatile so it auto-deletes on pod death). The `HostID` is `POD_IP` from the K8s Downward API and is persisted on `pipecatcall.HostID` so consumer services (`bin-ai-manager`) can route follow-up RPCs.

See [docs/patterns/per-pod-queues.md](../docs/patterns/per-pod-queues.md) for the canonical pattern (queue naming, identity source, limitations including Calico POD_IP recycle) and [docs/patterns/per-pod-liveness-preflight.md](../docs/patterns/per-pod-liveness-preflight.md) for the ping-before-RPC pattern that this service originated.

## Event Subscriptions

This service does not subscribe to external events. There is no SubscribeHandler.

## Common Commands

### Build and Test
```bash
# Build the Go daemon
go build -o ./bin/ ./cmd/...

# Run the daemon (requires configuration via env vars)
./bin/pipecat-manager

# Run all Go tests
go test ./...

# Tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html

# Single test
go test -v -run TestName ./pkg/packagename/...

# Generate mocks (uses go.uber.org/mock via //go:generate)
go generate ./pkg/pipecatcallhandler/...
go generate ./pkg/dbhandler/...
go generate ./pkg/cachehandler/...

# Build Docker image (includes both Go and Python components)
docker build -t pipecat-manager -f Dockerfile ../

# Generate protobuf (only when modifying proto/frames.proto)
protoc --go_out=. --go_opt=paths=source_relative proto/frames.proto
```

### `pipecat-control` CLI tool

A command-line tool for managing pipecat calls. **All output is JSON format on stdout**; logs go to stderr.

> Requires the `soxr` system library for audio processing.

```bash
# Get a pipecatcall — returns pipecatcall JSON
./bin/pipecat-control pipecatcall get --id <uuid>

# Start a new pipecatcall — returns created pipecatcall JSON
./bin/pipecat-control pipecatcall start --reference_type <type> --reference_id <uuid> [--customer_id]

# Terminate a pipecatcall — returns terminated pipecatcall JSON
./bin/pipecat-control pipecatcall terminate --id <uuid>

# Send a message to a pipecatcall
./bin/pipecat-control pipecatcall send-message --id <uuid> --message <text>
```

`pipecat-control` reuses the same environment variables as `pipecat-manager` (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

### Python component

The Python FastAPI service (`scripts/pipecat/main.py`) runs on port 8000 and must be running for the Go service to function:

```bash
cd scripts/pipecat

# Install dependencies
pip install -r requirements.txt

# Run the API server
python main.py
# or
uvicorn main:app --host 0.0.0.0 --port 8000
```

> The full pre-commit verification workflow (`go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`) and its rationale live in the root [CLAUDE.md](../CLAUDE.md#critical-verification-before-commit) and [docs/workflows/verification-workflows.md](../docs/workflows/verification-workflows.md).

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler` — shared handlers (sockhandler, requesthandler, notifyhandler, **circuit breaker**)
- `monorepo/bin-call-manager` — external media models
- `monorepo/bin-ai-manager` — AI call integration (consumes the per-pod queue + ping pattern that originated here)

Always run `go mod vendor` after changing dependencies.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock):
- Mock interfaces co-located with the interface definition (`mock_*.go`)
- Table-driven tests with struct slices
- Context passed to all handler methods

```go
tests := []struct {
    name      string
    input     InputType
    mockSetup func(*MockHandler)
    expectRes ResultType
    expectErr bool
}{
    {"success case", input1, setupMock1, expected1, false},
    {"error case", input2, setupMock2, nil, true},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        mc := gomock.NewController(t)
        defer mc.Finish()
        // test implementation
    })
}
```

## Key Implementation Details

### Audio Sample Rates (CRITICAL)
- Asterisk / WebSocket external media: 16 kHz slin16 (signed linear 16-bit PCM)
- Pipecat pipeline: `audio_out_sample_rate=16000` in `PipelineParams` — TTS generates 16 kHz natively, matching Asterisk end-to-end with **zero resampling**
- **Pipecat defaults to 24 kHz output.** Without the explicit 16 kHz setting, Go's per-chunk resampler creates boundary artifacts (robotic audio). Always keep `audio_out_sample_rate=16000`.
- Safety-net resampling via `audiosocketHandler.GetDataSamples()` exists for non-16 kHz input but is rarely used.

### Unpaced Audio Delivery + FLUSH_MEDIA Barge-in
`UnpacedWebsocketClientOutputTransport` no-ops `_write_audio_sleep()` to deliver TTS audio faster than real-time — Asterisk's `chan_websocket` handles re-timing internally, eliminating audio gaps from asyncio contention. On barge-in, the transport sends a `TextFrame("FLUSH_MEDIA")` which Go forwards as a WebSocket text message to Asterisk, instantly discarding all queued audio. This combines gap-free playback with instant interruption.

### `ConnAstDone` Pattern
The `runAsteriskReceivedMediaHandle` goroutine closes the `ConnAstDone` channel on Asterisk WebSocket disconnect. The lifecycle monitor waits on `ConnAstDone` and triggers cleanup, ensuring the Python pipeline and DB record are torn down on the actual hangup.

### Session Lifecycle
1. `POST /v1/pipecatcalls` creates DB record
2. External media created via `call-manager` RPC, returns `em.MediaURI`
3. Go dials Asterisk WebSocket at `em.MediaURI`, waits for `MEDIA_START` text message
4. Python runner started via HTTP POST to `localhost:8000/run`; Python connects back to Go via WebSocket
5. `ConnAstDone` closes when Asterisk WebSocket disconnects → cleanup
6. Cleanup also runs on `POST /v1/pipecatcalls/<id>/stop` (per-pod queue)

### Tool Execution Flow
1. Python Pipecat LLM emits a function call
2. Python sends HTTP request to Go `httpHandler.RunnerToolHandle`
3. Go makes RPC request to `bin-ai-manager`
4. Response returned to Python → Pipecat → LLM context

### Protobuf Frame Protocol
All WebSocket messages between Go and Python use protobuf-serialized frames. Generated code: `models/pipecatframe/frames.pb.go` (Go); Python uses `pipecat.serializers.protobuf`. Frame types: `TextFrame`, `AudioRawFrame`, `TranscriptionFrame`, `MessageFrame`.

### Pipecat Integration
- **Framework:** `pipecat-ai` Python library
- **LLM providers:** OpenAI (`openai.gpt-4o`), Grok (`grok.grok-3`, `grok.grok-3-mini`), Gemini (`gemini.gemini-2.5-flash`, `gemini.gemini-1.5-pro`)
- **STT providers:** Deepgram, Whisper
- **TTS providers:** Cartesia, ElevenLabs, Google
- **RTVI protocol** for runtime control and tool calling
- **VAD:** Silero Voice Activity Detection for turn-taking

## Configuration

Uses **Viper + pflag** (see `cmd/pipecat-manager/init.go`):

| Flag / Env | Description | Default |
|------------|-------------|---------|
| `database_dsn` / `DATABASE_DSN` | MySQL connection string | required |
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server (e.g., `amqp://guest:guest@localhost:5672`) | required |
| `redis_address` / `REDIS_ADDRESS` | Redis cache | required |
| `redis_password` / `REDIS_PASSWORD` | Redis auth | optional |
| `redis_database` / `REDIS_DATABASE` | Redis DB index | optional |
| `POD_IP` | **Required.** Used as `HostID` for per-pod queue + Python WebSocket callback target. Set via K8s Downward API (`status.podIP`). | required |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

Python environment variables (in `.env` or exported):
- LLM provider API keys: `OPENAI_API_KEY`, `XAI_API_KEY` (Grok), `GOOGLE_API_KEY` (Gemini), `ANTHROPIC_API_KEY`, etc.
- STT provider keys: `DEEPGRAM_API_KEY`
- TTS provider keys: `CARTESIA_API_KEY`, `ELEVENLABS_API_KEY`, `GOOGLE_API_KEY`

## Prometheus Metrics

Service exposes metrics on the configured endpoint (default `:2112/metrics`):
- `pipecat_manager_receive_request_process_time` — histogram of RPC request processing time, labels `{type, method}` (registered in `pkg/listenhandler/main.go`)

This service also benefits from the per-target circuit-breaker metrics that `bin-common-handler/pkg/requesthandler` registers under the `pipecat_manager_*` namespace (see [docs/patterns/circuit-breaker.md](../docs/patterns/circuit-breaker.md)).

> **Gotcha:** before adding new metrics to a `metricshandler` package here, check `bin-common-handler/pkg/requesthandler/main.go` `initPrometheus()` for existing metric names. Duplicate names cause `prometheus.MustRegister` to panic at startup. See [docs/workflows/common-gotchas.md](../docs/workflows/common-gotchas.md) (Prometheus Metric Name Conflicts).
