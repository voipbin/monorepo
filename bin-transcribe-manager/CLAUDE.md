# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

transcribe-manager is a speech-to-text service that processes real-time audio transcription for VoIP calls and conferences. It integrates with both Google Cloud Speech-to-Text and AWS Transcribe services, using RabbitMQ for messaging and Redis for caching.

## Key Architecture Concepts

### Monorepo Context
This service is part of a larger monorepo structure (`monorepo/bin-transcribe-manager`). Dependencies use local replace directives in `go.mod` (e.g., `replace monorepo/bin-common-handler => ../bin-common-handler`). When working with shared code, be aware that changes may affect multiple services.

### Message-Based Architecture
The service operates on a request/response pattern using RabbitMQ queues:
- **Listen queues**: `bin-manager.transcribe-manager.request` (shared), `bin-manager.transcribe-manager-<uuid>.request` (instance-specific)
- **Event queue**: `bin-manager.transcribe-manager.event` (outbound notifications)
- **Subscribe queues**: Listens to `bin-manager.call-manager.event` and `bin-manager.customer-manager.event` for lifecycle events

### Three Core Handlers
1. **ListenHandler** (`pkg/listenhandler`): Processes REST-style API requests via RabbitMQ. Routes requests based on URL patterns (e.g., `/v1/transcribes`, `/v1/transcripts`)
2. **SubscribeHandler** (`pkg/subscribehandler`): Listens to events from other services (call hangups, customer deletions) to trigger cleanup
3. **StreamingHandler** (`pkg/streaminghandler`): Manages real-time audio streaming connections using AudioSocket protocol. Maintains active streaming sessions in memory with mutex-protected map

### Dual STT Provider Support
The service supports both Google Cloud Platform and AWS transcription:
- GCP: Uses `speech.Client` with LINEAR16 encoding at 8kHz
- AWS: Uses `transcribestreaming.Client` with PCM encoding at 8kHz
- Configuration is per-streaming session based on customer preferences or language requirements

### Data Models
- **Transcribe** (`models/transcribe`): Represents a transcription session with reference to call/conference/recording, language (BCP47), direction (in/out/both), and status (progressing/done)
- **Streaming** (`models/streaming`): Represents an active audio stream connection with provider-specific client configuration
- **Transcript** (`models/transcript`): Individual transcribed text segments with timestamps and confidence scores

### Database Layer
`pkg/dbhandler` abstracts database operations with a consistent interface. Uses both MySQL (via `database/sql`) and Redis cache. Always uses context for cancellation support.

## Common Development Commands

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
# Lint check (requires golint)
golint -set_exit_status $(go list ./...)

# Vet check
go vet $(go list ./...)
```

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

## Working with the Codebase

### Configuration via Environment Variables
The service uses Cobra and Viper for configuration (see `internal/config/main.go`). Key environment variables:
- `DATABASE_DSN`: MySQL connection string
- `REDIS_ADDRESS`, `REDIS_DATABASE`, `REDIS_PASSWORD`: Redis configuration
- `RABBITMQ_ADDRESS`: RabbitMQ connection string
- `AWS_ACCESS_KEY`, `AWS_SECRET_KEY`: AWS credentials for Transcribe (optional if GCP configured)
- `POD_IP`: Required for AudioSocket listening address (populated by Kubernetes)

All configuration can also be provided via CLI flags. Run `transcribe-manager --help` for details.

**STT Provider Requirements:**
- At least one STT provider must be configured (GCP or AWS)
- GCP: Uses Application Default Credentials (ADC) - can be from service account key, gcloud CLI, GKE metadata server, etc.
- AWS: Requires both `AWS_ACCESS_KEY` and `AWS_SECRET_KEY` environment variables
- If both are configured, GCP is tried first with AWS as fallback
- Service fails to start if neither provider is available

**Configuration Pattern:**
Uses singleton pattern with `config.Get()` for thread-safe access. Configuration is loaded once at startup in the Cobra `PersistentPreRunE` hook.

### Adding New Transcribe Operations
When adding new transcribe-related endpoints:
1. Add URL pattern regex to `pkg/listenhandler/main.go`
2. Implement handler method in `pkg/listenhandler/v1_transcribes.go`
3. Add business logic to `pkg/transcribehandler/transcribe.go`
4. Update database methods in `pkg/dbhandler/transcribe.go` if persistence needed
5. Send notifications via `notifyhandler` for state changes

### Working with Streaming Audio
The streaming handler maintains active connections in `mapStreaming` (mutex-protected). When modifying streaming logic:
- Always lock/unlock `muSteaming` when accessing the map
- Implement proper cleanup in Stop() to prevent resource leaks
- Use context cancellation for graceful shutdown
- AudioSocket protocol expects 8kHz, 16-bit mono PCM

### Event-Driven Cleanup
Subscribe handler methods (e.g., `EventCMCallHangup`, `EventCUCustomerDeleted`) ensure resources are cleaned up when parent entities are deleted. When adding new reference types, implement corresponding event handlers.

### Prometheus Metrics
Metrics are registered in handler init() functions:
- `transcribe_create_total`: Counter for transcription creation by type
- `receive_request_process_time`: Histogram for request processing latency

## CI/CD Pipeline

The GitLab CI pipeline (`.gitlab-ci.yml`) has stages:
1. **ensure**: Download and vendor dependencies
2. **test**: Run golint, go vet, and tests with coverage
3. **build**: Build Docker image and push to registry
4. **release**: Deploy to Kubernetes using kustomize (manual trigger)

## Testing Considerations

- Tests use `go.uber.org/mock` for mocking interfaces
- Database tests may require test database setup (see `scripts/database_scripts_test/`)
- Mock files follow pattern `mock_*.go` in same package as interface
- Use table-driven tests for multiple scenarios
- Always test error paths and edge cases (nil contexts, invalid UUIDs, etc.)

## Important Constraints

- **Language codes**: Must be valid BCP47 format (e.g., "en-US", "ko-KR")
- **Status transitions**: Only allow valid state transitions (see `models/transcribe/transcribe.go:IsUpdatableStatus`)
- **UUID validation**: All IDs must be valid UUIDs; use `uuid.FromString()` with error checking
- **Concurrency**: Streaming map requires mutex protection; database operations use transactions where needed
- **Graceful shutdown**: Use signal handlers and context cancellation to prevent data loss
