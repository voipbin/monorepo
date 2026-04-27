# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-tts-manager` is a Go microservice for text-to-speech (TTS) synthesis in a VoIP system. It provides two modes of operation:

1. **Batch TTS**: Generate and store pre-recorded audio files from text.
2. **Real-time Streaming TTS**: Stream synthesized audio directly to live VoIP calls via the AudioSocket protocol.

The service integrates with multiple TTS providers (Google Cloud TTS, AWS Polly, ElevenLabs) and manages audio delivery through both file storage and real-time streaming.

**Key Concepts:**
- **Batch TTS file**: Audio file generated from text and stored on a shared volume; served by a Python HTTP sidecar on port 80.
- **Streaming session**: Real-time WebSocket connection to ElevenLabs that pumps audio frames to Asterisk over AudioSocket on port 8080.
- **Per-pod queue routing**: Streaming control RPCs are routed to the pod owning the in-memory session via `bin-manager.tts-manager.request.<HOSTNAME>`.
- **Multi-container pod**: Go service + Python HTTP sidecar share `/shared-data` for file delivery.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-tts-manager`.

## Common Commands

```bash
# Build the tts-manager daemon
go build -o ./bin/ ./cmd/...

# Run the daemon (requires configuration via flags or env vars)
./bin/tts-manager

# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html

# Run a single test
go test -v -run TestName ./pkg/packagename/...

# Generate all mocks (uses go.uber.org/mock via //go:generate directives)
go generate ./pkg/ttshandler/...
go generate ./pkg/streaminghandler/...
go generate ./pkg/audiohandler/...
go generate ./pkg/buckethandler/...
go generate ./pkg/cachehandler/...
```

## tts-control CLI Tool

A command-line tool for TTS operations. **All output is JSON format** (stdout), logs go to stderr.

```bash
# Create a new TTS audio file - returns created TTS JSON
./bin/tts-control tts create --text <text> --language <lang> [--voice] [--gender]
```

Uses same environment variables as tts-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

## Architecture

### Service Layer Structure

The service follows a dual-mode architecture with handler separation:

1. **cmd/tts-manager/** - Main daemon entry point with configuration via Cobra/Viper (flags and env vars)
2. **pkg/listenhandler/** - RabbitMQ RPC request handler routing to appropriate handlers
3. **pkg/ttshandler/** - Batch TTS creation handler (generates audio files from text)
4. **pkg/streaminghandler/** - Real-time streaming handler (WebSocket/AudioSocket protocol)
5. **pkg/audiohandler/** - Multi-provider TTS synthesis (GCP, AWS Polly)
6. **pkg/buckethandler/** - Local file storage handler for generated audio
7. **pkg/cachehandler/** - Redis cache for TTS metadata
8. **pkg/dbhandler/** - Database operations (if applicable)
9. **models/** - Data structures (tts, streaming, message)

### Dual Operation Modes

#### Batch TTS Mode
Creates pre-recorded audio files that are stored locally and served via HTTP:

```
RabbitMQ Request → listenhandler → ttshandler → audiohandler (GCP/AWS) → buckethandler → local storage
                                                                                              ↓
                                                                                     HTTP server (Python)
```

The service runs a Python HTTP server sidecar (on port 80) to serve generated audio files from `/shared-data`.

#### Real-time Streaming Mode
Streams synthesized audio directly to live calls using AudioSocket protocol:

```
Asterisk Call → AudioSocket (TCP:8080) → streaminghandler → ElevenLabs WebSocket → audio frames → AudioSocket
                                                ↑
                                    RabbitMQ control messages (Start/Say/Stop)
```

### Per-pod queue routing

This service uses the per-pod RabbitMQ queue convention. The `HostID` is sourced from `HOSTNAME` (rather than `POD_IP` as in `bin-pipecat-manager`):
- Shared queue: `bin-manager.tts-manager.request` — batch TTS RPCs (`POST /v1/ttses`)
- Per-pod queue: `bin-manager.tts-manager.request.<HOSTNAME>` — streaming session control (Start/Say/Stop)
- `POD_IP` is used to advertise the AudioSocket endpoint that callers (Asterisk) dial into

See [docs/patterns/per-pod-queues.md](../docs/patterns/per-pod-queues.md) for the canonical pattern (queue naming, identity source, limitations).

### TTS Provider Strategy

The audiohandler attempts providers in sequence:
1. **Google Cloud TTS** - Primary provider using Application Default Credentials (ADC) with regional endpoint `eu-texttospeech.googleapis.com:443`
2. **AWS Polly** - Fallback provider using access key/secret key credentials

For real-time streaming:
- **ElevenLabs** - WebSocket-based streaming TTS with keep-alive management

### Inter-Service Communication

- Uses RabbitMQ for message passing between microservices
- Publishes events to `QueueNameTTSEvent` when TTS operations complete
- Listens on `QueueNameTTSRequest` for incoming TTS requests
- **Monorepo structure**: All sibling services are referenced via `replace` directives in go.mod pointing to `../bin-*-manager` directories. When modifying shared dependencies (especially `bin-common-handler`), changes affect all services immediately.

### Key Patterns

- Handler interfaces with mock generation using `go.uber.org/mock` (`//go:generate mockgen`)
- Prometheus metrics exposed at configurable endpoint (default `:2112/metrics`)
- Context propagation through all handler methods
- UUID-based entity identification using `github.com/gofrs/uuid`
- Concurrent connection handling with goroutines and context cancellation
- AudioSocket protocol implementation for real-time audio streaming
- Keep-alive management with configurable intervals and retry backoff

### Streaming Session Management

The streaminghandler maintains concurrent streaming sessions:
- Each connection gets its own goroutine with context cancellation
- Keep-alive pings sent every 30 seconds via AudioSocket protocol
- Sessions identified by UUID extracted from initial AudioSocket handshake
- Vendor-specific handlers (ElevenLabs) manage WebSocket lifecycle
- Message queuing for say operations (SayInit, SayAdd, SayStop, SayFinish)

## Request Routing

ListenHandler routes RPC requests using regex patterns matching REST-like URIs:

**TTSes API (`/v1/ttses/*`):**
- `POST /v1/ttses` — Generate batch TTS audio file (returns the audio URL). Served on the **shared** queue.
- `GET /v1/ttses/<uuid>` — Get TTS metadata. Served on the **shared** queue.

**Streaming control (`/v1/streamings/*`):**
- `POST /v1/streamings/<uuid>/start` — Initialize streaming session.
- `POST /v1/streamings/<uuid>/say` — Send a text chunk to the session.
- `POST /v1/streamings/<uuid>/stop` — Stop the streaming session.

Streaming control endpoints are served on the **per-pod** queue.

In addition, the service listens on TCP port 8080 for the AudioSocket protocol — this is how Asterisk delivers media frames to the active streaming session.

## Event Subscriptions

This service does not subscribe to RabbitMQ events. There is no SubscribeHandler — TTS is invoked synchronously via RPC.

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared utilities (sockhandler, requesthandler, notifyhandler)
- `monorepo/bin-call-manager`: External media models (for streaming setup)

Always run `go mod vendor` after changing dependencies.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock):
- Mock interfaces co-located with handlers (`mock_*.go`)
- Table-driven tests with struct slices

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

### Streaming Session Management
The streaminghandler maintains concurrent streaming sessions:
- Each connection gets its own goroutine with context cancellation
- Keep-alive pings sent every 30 seconds via the AudioSocket protocol
- Sessions identified by UUID extracted from the initial AudioSocket handshake
- Vendor-specific handlers (ElevenLabs) manage WebSocket lifecycle
- Message queuing for say operations (`SayInit`, `SayAdd`, `SayStop`, `SayFinish`)

### Multi-Container Deployment
Pod deployment runs two containers:
1. **tts-manager** (Go): main service, listens on port 8080 (AudioSocket) and 2112 (metrics)
2. **http-server** (Python): serves generated audio files from `/shared-data` on port 80

The shared volume `/shared-data` is the bridge between containers — the Go service writes audio files; the Python HTTP server serves them.

### GCP Regional Endpoint
GCP TTS uses the regional endpoint `eu-texttospeech.googleapis.com:443` for lower latency in the EU region.

## Configuration

Environment variables / flags:

| Flag / Env | Description | Default |
|------------|-------------|---------|
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server | `amqp://guest:guest@localhost:5672` |
| `aws_access_key` / `AWS_ACCESS_KEY` | AWS Polly credentials | required if AWS used |
| `aws_secret_key` / `AWS_SECRET_KEY` | AWS Polly credentials | required if AWS used |
| `elevenlabs_api_key` / `ELEVENLABS_API_KEY` | ElevenLabs API key for streaming | required for streaming |
| `POD_IP` | **Required.** Used to advertise the AudioSocket endpoint. Injected by k8s. | required |
| `HOSTNAME` | **Required.** Used as `HostID` for per-pod queue routing. Injected by k8s. | required |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

GCP credentials are managed via Application Default Credentials (ADC) — typically through `GOOGLE_APPLICATION_CREDENTIALS` or workload identity.

## Prometheus Metrics

Service exposes metrics on the configured endpoint (default `:2112/metrics`):
- `tts_manager_receive_request_process_time` — histogram of RPC request processing time (labels: type, method)
