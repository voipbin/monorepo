# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-transcribe-manager` is a speech-to-text service that processes real-time audio transcription for VoIP calls and conferences. It integrates with both Google Cloud Speech-to-Text and AWS Transcribe, using RabbitMQ for messaging and Redis for caching.

**Key Concepts:**
- **Transcribe**: A transcription session with reference to a call/conference/recording, language (BCP47), direction (in/out/both), and status (`progressing`/`done`).
- **Streaming**: An active audio stream connection with provider-specific client configuration; sessions live in memory keyed by UUID and are anchored to one pod.
- **Transcript**: An individual transcribed text segment with timestamps and confidence scores.
- **Per-pod queue routing**: Operations targeting an active streaming session are routed via the per-pod queue convention so they reach the pod owning the in-memory state.
- **Dual provider support**: GCP (`speech.Client`, LINEAR16 8 kHz) and AWS (`transcribestreaming.Client`, PCM 8 kHz). Per-session selection.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-transcribe-manager`.

## Architecture

### Service Communication Pattern

This service uses **RabbitMQ for RPC-style communication**:
- **ListenHandler** (`pkg/listenhandler/`): Consumes from two queues:
  - `bin-manager.transcribe-manager.request` (shared) — RPC requests that don't need to reach a specific pod (`POST /v1/transcribes`, `GET /v1/transcribes/<id>`).
  - `bin-manager.transcribe-manager.request.<host_id>` (volatile, per-pod) — operations targeting an in-memory streaming session.
- **SubscribeHandler** (`pkg/subscribehandler/`): Consumes events from `bin-manager.call-manager.event` and `bin-manager.customer-manager.event` for lifecycle cleanup.
- **StreamingHandler** (`pkg/streaminghandler/`): Manages real-time audio streaming connections via WebSocket transport. Dials out to Asterisk's `chan_websocket` endpoint per session. Maintains active streaming sessions in memory with a mutex-protected map.
- **NotifyHandler**: Publishes events to `bin-manager.transcribe-manager.event`.

### Per-pod queue routing

Like `bin-pipecat-manager`, this service uses the per-pod RabbitMQ queue convention. The `HostID` is `POD_IP` from the Kubernetes Downward API and is persisted on the streaming session so consumer services can route follow-up RPCs.

See [docs/patterns/per-pod-queues.md](../docs/patterns/per-pod-queues.md) for the canonical pattern (queue naming, identity source, limitations including Calico POD_IP recycle).

### Database Layer
`pkg/dbhandler` abstracts database operations with a consistent interface. Uses both MySQL (via `database/sql`) and Redis cache. Always uses context for cancellation support.

## Common Commands

### Building
```bash
# Build the service
go build -o ./bin/transcribe-manager ./cmd/transcribe-manager

# Docker build (from monorepo root)
docker build -t transcribe-manager .
```

### Testing
```bash
# Run all tests with coverage
go test -v ./...

# Run tests with coverage report
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html

# Run tests for a specific package
go test -v ./pkg/transcribehandler
go test -v ./pkg/streaminghandler
```

### Linting
```bash
# Comprehensive lint check (requires golangci-lint)
golangci-lint run -v --timeout 5m

# Legacy lint check (requires golint)
golint -set_exit_status $(go list ./...)

# Vet check
go vet $(go list ./...)
```

## transcribe-control CLI Tool

A command-line tool for managing transcription sessions directly via database/cache (bypasses RabbitMQ RPC). **All output is JSON format** (stdout), logs go to stderr.

```bash
# Start a new transcription session - returns created session JSON
./bin/transcribe-control transcribe start --reference_type <type> --reference_id <uuid> --language <lang> [--customer_id]

# Stop a transcription session - returns stopped session JSON
./bin/transcribe-control transcribe stop --id <uuid>

# Get transcription session - returns session JSON
./bin/transcribe-control transcribe get --id <uuid>

# Get transcription session by reference - returns session JSON
./bin/transcribe-control transcribe get-by-reference --reference_id <uuid> --language <lang>

# List transcription sessions - returns JSON array
./bin/transcribe-control transcribe list --customer_id <uuid> [--limit 100] [--token]

# Delete transcription session - returns deleted session JSON
./bin/transcribe-control transcribe delete --id <uuid>
```

Uses same environment variables as transcribe-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

### Dependencies
```bash
# Download dependencies
go mod download

# Vendor dependencies (required for CI/CD)
go mod vendor
```

### Generate Mocks
The codebase uses `go:generate` directives with mockgen. To regenerate mocks:
```bash
# Generate all mocks
go generate ./...

# Generate mocks for specific package
cd pkg/streaminghandler && go generate
```

## Request Routing

ListenHandler routes RPC requests using regex patterns matching REST-like URIs:

**Transcribes API (`/v1/transcribes/*`):**
- `POST /v1/transcribes` — Start a transcription session. Served on the **shared** queue.
- `GET /v1/transcribes?<filters>` — List transcriptions. Served on the **shared** queue.
- `GET /v1/transcribes/<uuid>` — Get a transcription session. Served on the **shared** queue.
- `POST /v1/transcribes/<uuid>/stop` — Stop a session. Served on the **per-pod** queue (operates on in-memory streaming state).
- `DELETE /v1/transcribes/<uuid>` — Delete a session.

**Transcripts API (`/v1/transcripts/*`):**
- `GET /v1/transcripts?<filters>` — List transcript segments
- `GET /v1/transcripts/<uuid>` — Get a transcript segment

## Event Subscriptions

SubscribeHandler subscribes to:
- **bin-manager.call-manager.event**: `call_hangup` — finalizes any associated transcription session (`EventCMCallHangup`).
- **bin-manager.customer-manager.event**: `customer_deleted` — cascading cleanup of the customer's transcribes (`EventCUCustomerDeleted`).

## Monorepo Context

This service is part of a larger monorepo. Dependencies use local `replace` directives in `go.mod` (e.g., `replace monorepo/bin-common-handler => ../bin-common-handler`). Key local dependencies:
- `monorepo/bin-common-handler`: Shared utilities (sockhandler, requesthandler, notifyhandler)
- `monorepo/bin-call-manager`: External media models for streaming setup

Always run `go mod vendor` after changing dependencies.

## Testing Patterns

- Tests use `go.uber.org/mock` for mocking interfaces
- Database tests may require test database setup (see `scripts/database_scripts_test/`)
- Mock files follow `mock_*.go` in the same package as the interface
- Use table-driven tests for multiple scenarios; always test error paths and edge cases (nil contexts, invalid UUIDs, etc.)
- **Test struct initialization**: Use explicit field names — `{name: "test", input: "value", expectedRes: result}`, not positional.
- **Test function comments**: Test names should be self-documenting. Use inline comments only where behavior is non-obvious.

```go
tests := []struct {
    name      string
    input     InputType
    mockSetup func(*MockHandler)
    expectRes ResultType
    expectErr bool
}{
    {name: "success case", input: input1, mockSetup: setupMock1, expectRes: expected1, expectErr: false},
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

### STT Provider Selection
At startup, all providers with valid credentials are initialized; at least one must be available. Default order: `gcp` → `aws`. Callers may pass `provider` ("gcp" or "aws") to try a specific provider first, with fallback to the default order.

### Streaming Audio Handling
The streaming handler maintains active connections in `mapStreaming` (mutex-protected via `muSteaming`):
- Always lock/unlock when accessing the map
- Implement proper cleanup in `Stop()` to prevent leaks (WebSocket close + external media stop)
- Use context cancellation for graceful shutdown
- WebSocket transport delivers raw 8 kHz, 16-bit mono signed linear PCM (slin) binary frames
- Service dials out to Asterisk via `MediaURI` returned from `ExternalMediaStart` (connection type `server`, transport `websocket`, encapsulation `none`)

### Language Codes
Must be valid BCP47 format (e.g., `en-US`, `ko-KR`).

### Status Transitions
Only allow valid state transitions — see `models/transcribe/transcribe.go:IsUpdatableStatus`.

### Adding New Transcribe Operations
1. Add URL pattern regex to `pkg/listenhandler/main.go`
2. Implement handler method in `pkg/listenhandler/v1_transcribes.go`
3. Add business logic to `pkg/transcribehandler/transcribe.go`
4. Update database methods in `pkg/dbhandler/transcribe.go` if persistence needed
5. Send notifications via `notifyhandler` for state changes

## Configuration

Service uses Cobra and Viper (see `internal/config/main.go`). Configuration uses singleton pattern with `config.Get()` for thread-safe access; loaded once in the Cobra `PersistentPreRunE` hook.

| Flag / Env | Description | Default |
|------------|-------------|---------|
| `database_dsn` / `DATABASE_DSN` | MySQL connection string | required |
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server | required |
| `redis_address` / `REDIS_ADDRESS` | Redis cache | required |
| `redis_password` / `REDIS_PASSWORD` | Redis auth | optional |
| `redis_database` / `REDIS_DATABASE` | Redis DB index | optional |
| `aws_access_key` / `AWS_ACCESS_KEY` | AWS Transcribe credentials | optional (if GCP configured) |
| `aws_secret_key` / `AWS_SECRET_KEY` | AWS Transcribe credentials | optional (if GCP configured) |
| `POD_IP` | **Required.** Used as `HostID` for per-pod queue routing. Set via K8s Downward API (`status.podIP`). | required |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

GCP authentication uses Application Default Credentials (service account key, `gcloud` CLI, GKE metadata server). At least one STT provider (GCP or AWS) must be configured.

## Prometheus Metrics

Metrics are registered in handler `init()` functions:
- `transcribe_create_total` — counter for transcription creation (label: `type`)
- `receive_request_process_time` — histogram for request processing latency

## CI/CD Pipeline

The GitLab CI pipeline (`.gitlab-ci.yml`) has stages:
1. **ensure**: Download and vendor dependencies
2. **test**: Run golint, go vet, and tests with coverage
3. **build**: Build Docker image and push to registry
4. **release**: Deploy to Kubernetes using kustomize (manual trigger)
