# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

bin-tts-manager is a Go microservice for text-to-speech (TTS) synthesis in a VoIP system. It provides two modes of operation:
1. **Batch TTS**: Generate and store pre-recorded audio files from text
2. **Real-time Streaming TTS**: Stream synthesized audio directly to live VoIP calls via AudioSocket protocol

The service integrates with multiple TTS providers (Google Cloud TTS, AWS Polly, ElevenLabs) and manages audio delivery through both file storage and real-time streaming.

## Build and Test Commands

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

### Pod-specific Routing

The service supports Kubernetes pod-specific routing:
- Listens on both a shared queue (`bin-manager.tts-manager.request`) and a pod-specific queue (`bin-manager.tts-manager.request.{HOSTNAME}`)
- Uses `POD_IP` environment variable to construct the streaming endpoint address
- Enables direct routing to specific pods for streaming sessions

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

### Configuration

Environment variables / flags:
- `RABBITMQ_ADDRESS` - RabbitMQ connection (default: `amqp://guest:guest@localhost:5672`)
- `PROMETHEUS_ENDPOINT` - Metrics endpoint (default: `/metrics`)
- `PROMETHEUS_LISTEN_ADDRESS` - Metrics server address (default: `:2112`)
- `AWS_ACCESS_KEY`, `AWS_SECRET_KEY` - AWS Polly credentials
- `ELEVENLABS_API_KEY` - ElevenLabs API key for streaming
- `POD_IP` - Pod IP address for streaming endpoint (injected by k8s)
- `HOSTNAME` - Pod hostname for queue routing (injected by k8s)

GCP credentials are managed via Application Default Credentials (ADC) - typically through `GOOGLE_APPLICATION_CREDENTIALS` environment variable or workload identity.

### Deployment Architecture

Multi-container pod deployment:
1. **tts-manager** container: Main Go service listening on port 8080 (AudioSocket) and 2112 (metrics)
2. **http-server** container: Python HTTP server on port 80 serving audio files from shared volume

Shared volume `/shared-data` acts as a bridge for generated audio files between containers.
