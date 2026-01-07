# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`voip-asterisk-proxy` is a Go service that acts as a bidirectional proxy between Asterisk PBX (via ARI/AMI) and RabbitMQ. It connects to Asterisk's REST Interface (ARI) via WebSocket and Asterisk Manager Interface (AMI) via TCP, forwarding events to RabbitMQ queues and handling RPC requests from RabbitMQ to control Asterisk.

## Key Architecture

### Three-Handler Design Pattern

The service is structured around three core handlers that run concurrently:

1. **EventHandler** (`pkg/eventhandler/`): Receives events from Asterisk
   - Maintains WebSocket connection to Asterisk ARI (`/ari/events`)
   - Receives AMI events from the AMI connection
   - Publishes all events to RabbitMQ queue (`asterisk.all.event` by default)
   - Auto-reconnects on connection failures with 1-second retry delay

2. **ListenHandler** (`pkg/listenhandler/`): Handles RPC requests from RabbitMQ
   - Listens on two queue types:
     - **Permanent queue**: Shared queue for request type (e.g., `asterisk.call.request`)
     - **Volatile queue**: Instance-specific queue using Asterisk ID (e.g., `asterisk.<mac-address>.request`)
   - Routes requests based on URI patterns:
     - `/ari/*` → Proxies HTTP requests to Asterisk ARI
     - `/ami/*` → Sends AMI actions to Asterisk
     - `/proxy/recording_file_move` → Uploads recording files to Google Cloud Storage
   - Returns responses as RabbitMQ RPC replies

3. **ServiceHandler** (`pkg/servicehandler/`): Provides business logic services
   - Currently handles recording file uploads to Google Cloud Storage buckets
   - Uses Application Default Credentials (ADC) for GCP authentication

### Asterisk Identity System

Each proxy instance identifies itself using the MAC address of a configured network interface:
- **Asterisk ID**: MAC address from network interface (e.g., `42:01:0a:a4:0f:d0`)
- **Internal Address**: IPv4 address from the same interface
- Updates Redis key `asterisk.<asterisk-id>.address-internal` every 5 minutes with 24-hour expiry
- In Kubernetes environments, sets pod annotation `asterisk-id` with the MAC address

### Request/Response Flow

**Incoming ARI/AMI requests (RabbitMQ → Asterisk):**
```
RabbitMQ Request → ListenHandler → HTTP/AMI to Asterisk → Response back to RabbitMQ
```

**Outgoing events (Asterisk → RabbitMQ):**
```
Asterisk ARI/AMI Event → EventHandler → RabbitMQ publish
```

### Configuration Management

Uses **Viper + pflag** for configuration:
- Command-line flags take precedence over environment variables
- Each config parameter binds to both a flag (e.g., `--ari_address`) and env var (e.g., `ARI_ADDRESS`)
- See `cmd/asterisk-proxy/init.go` for the complete binding logic

## Common Development Commands

### Building

```bash
# Build the binary
go build -o bin/asterisk-proxy ./cmd/asterisk-proxy

# Using Makefile (delegates to subdirectory Makefile)
make build
```

The Dockerfile builds using a two-stage process:
1. Build stage: Compiles Go binary in `golang:1.25-alpine`
2. Runtime stage: Copies binary to minimal `alpine` image

### Testing

```bash
# Run all tests
go test ./...

# Run tests in a specific package
go test ./pkg/listenhandler

# Run a specific test
go test ./pkg/listenhandler -run Test_ariSendRequestToAsterisk

# Run tests with verbose output
go test -v ./...

# Using Makefile
make test
```

### Generating Mocks

The codebase uses `go.uber.org/mock` for interface mocking:

```bash
# Generate mocks (uses go:generate directive in servicehandler/main.go)
go generate ./...
```

### Running Locally

```bash
./asterisk-proxy \
  --ari_address localhost:8088 \
  --ari_account asterisk:asterisk \
  --ari_application voipbin \
  --ari_subscribe_all true \
  --ami_host 127.0.0.1 \
  --ami_port 5038 \
  --ami_username asterisk \
  --ami_password asterisk \
  --interface_name eth0 \
  --rabbitmq_address amqp://guest:guest@localhost:5672 \
  --rabbitmq_queue_listen asterisk.call.request \
  --redis_address localhost:6379 \
  --redis_database 1
```

Or use environment variables:
```bash
export ARI_ADDRESS=localhost:8088
export RABBITMQ_ADDRESS=amqp://guest:guest@localhost:5672
./asterisk-proxy
```

## Monorepo Context

This service is part of a Go monorepo with replace directives in `go.mod` for local dependencies:
- **bin-common-handler**: Shared handlers for RabbitMQ, notifications, request processing, and common models
  - Provides `sockhandler.SockHandler` for RabbitMQ connections
  - Provides `notifyhandler.NotifyHandler` for event publishing
  - Provides `requesthandler.RequestHandler` for RPC request handling
  - Contains common models in `models/sock` and `models/outline`

When adding imports, use the monorepo module path: `monorepo/<module-name>/...`

## Message Formats

### RabbitMQ Event Message
```json
{
  "type": "ari_event" | "ami_event",
  "data_type": "application/json",
  "data": "{...}"
}
```

### RabbitMQ RPC Request
```json
{
  "uri": "/ari/channels?endpoint=pjsip/test@sippuas&app=test",
  "method": "POST",
  "data": "request body",
  "data_type": "text/plain"
}
```

### RabbitMQ RPC Response
```json
{
  "status_code": 200,
  "data_type": "application/json",
  "data": "{...}"
}
```

## Important Implementation Details

### Connection Resilience
- Both ARI WebSocket and AMI connections implement automatic reconnection with 1-second retry delays
- RabbitMQ connection is managed by `bin-common-handler/pkg/sockhandler`

### Kubernetes Integration
- Can be disabled with `--kubernetes_disabled=true` flag or `KUBERNETES_DISABLED=true` env var
- Uses in-cluster config to patch pod annotations
- Falls back to `POD_NAME` and `POD_NAMESPACE` env vars if in-cluster detection fails
- Implements retry logic with 3 attempts for annotation patching

### Prometheus Metrics
- Metrics exposed on `/metrics` endpoint (default port `:2112`)
- Configurable via `--prometheus_endpoint` and `--prometheus_listen_address`

### Logging
- Uses `sirupsen/logrus` with `joonix` formatter for structured logging
- Default log level: Debug

## Adding New Features

### Adding a New Proxy Endpoint
1. Define request/response structs in `pkg/listenhandler/request/`
2. Add regex pattern in `pkg/listenhandler/main.go` (e.g., `regProxyRecordingFileMove`)
3. Implement handler method in `pkg/listenhandler/proxy_handler.go`
4. Add case to `processRequest()` switch statement in `pkg/listenhandler/main.go`
5. Write tests in `pkg/listenhandler/proxy_handler_test.go`

### Adding Service Logic
1. Add method to `ServiceHandler` interface in `pkg/servicehandler/main.go`
2. Implement method in the same file
3. Regenerate mocks: `go generate ./pkg/servicehandler`
4. Call from appropriate handler (usually `ListenHandler`)
