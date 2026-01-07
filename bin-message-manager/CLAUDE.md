# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is the `bin-message-manager` service, part of the VoIPbin monorepo. It handles SMS messaging through external providers (Telnyx, MessageBird), including message creation, delivery tracking, provider webhooks, and billing integration.

**Key Concepts:**
- **Message**: SMS with source/destination addresses, provider info, text content, and targets (recipients)
- **Provider**: External SMS gateway (Telnyx, MessageBird) with authentication tokens and webhooks
- **Target**: Individual recipient with delivery status (queued/sent/delivered/failed)
- **Hook**: Webhook endpoint receiving provider delivery status updates

## Architecture

### Service Communication Pattern

This service uses **RabbitMQ for RPC-style communication**:
- **ListenHandler** (`pkg/listenhandler/`): Consumes RPC requests from queue `bin-manager.message-manager.request`, processes them, and returns responses
- **NotifyHandler**: Publishes events to exchange `bin-manager.message-manager.event` when message state changes
- External webhooks are forwarded to `/v1/hooks` endpoint via RabbitMQ

### Core Components

```
cmd/message-manager/main.go
    ├── initVariable() -> Configuration (Cobra + Viper)
    ├── run()
        ├── pkg/cachehandler (Redis)
        ├── pkg/dbhandler (MySQL via Squirrel query builder)
        ├── pkg/requestexternal (Telnyx/MessageBird API clients)
        ├── pkg/messagehandler (Business logic layer)
        └── pkg/listenhandler (RabbitMQ RPC request router)
```

**Layer Responsibilities:**
- `models/message/`: Core data structures (Message, Type, Direction, ProviderName)
- `models/telnyx/`, `models/messagebird/`: Provider-specific payload models
- `models/target/`: Recipient tracking with delivery status
- `pkg/messagehandler/`: Business logic (Send/Get/Delete messages, provider routing, hook processing)
- `pkg/dbhandler/`: Database operations using Squirrel SQL builder
- `pkg/cachehandler/`: Redis caching for message lookups
- `pkg/requestexternal/`: HTTP clients for provider APIs (Telnyx, MessageBird)
- `pkg/listenhandler/`: RabbitMQ RPC request routing (REST-like paths: `/v1/messages`, `/v1/hooks`)

### Request Routing

ListenHandler routes requests using regex patterns matching REST-like URIs:
- `POST /v1/messages` - Send message (checks billing balance, creates message, dispatches to provider)
- `GET /v1/messages?<filters>` - List messages (pagination via page_size/page_token)
- `GET /v1/messages/<uuid>` - Get message details
- `DELETE /v1/messages/<uuid>` - Delete message
- `POST /v1/hooks` - Process provider webhook (delivery status updates)

### Provider Architecture

Messages are sent asynchronously via goroutine after creation:
1. Balance validation via `billing-manager` RPC call
2. Message record created in database
3. Provider selected (currently defaults to MessageBird, falls back to Telnyx)
4. Message dispatched to provider API concurrently per target
5. Target statuses updated in database upon response

**Provider Handlers:**
- `pkg/messagehandler/provider_telnyx.go`: Telnyx SMS API integration
- `pkg/messagehandler/provider_messagebird.go`: MessageBird SMS API integration
- Each provider handler sends to multiple targets concurrently using sync.WaitGroup

**Webhook Processing:**
- Webhooks POST to `/v1/hooks` with `received_uri` and `received_data`
- URI suffix determines provider (e.g., `/telnyx` → Telnyx webhook)
- Payload parsed to extract delivery status and update message targets

### Configuration

Uses **Cobra + Viper** pattern (`cmd/message-manager/init.go`):
- Command-line flags and environment variables (e.g., `--rabbitmq_address` or `RABBITMQ_ADDRESS`)
- Required: `database_dsn`, `rabbitmq_address`, `redis_address`, `authtoken_messagebird`, `authtoken_telnyx`
- Prometheus metrics endpoint configurable (default `:2112/metrics`)

## Common Commands

### Build
```bash
# From monorepo root (expects parent directory context for replacements)
cd /path/to/monorepo/bin-message-manager
go build -o bin/message-manager ./cmd/message-manager
```

### Test
```bash
# Run all tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html
go tool cover -func=cp.out

# Run tests for specific package
go test -v ./pkg/messagehandler/...

# Run single test
go test -v ./pkg/messagehandler -run Test_Send
```

### Generate Mocks
```bash
# Generate all mocks (uses go:generate directives)
go generate ./...

# Mocks are created via mockgen for interfaces in:
# - pkg/messagehandler/main.go -> mock_main.go
# - pkg/messagehandler/provider_telnyx.go -> mock_provider_telnyx.go
# - pkg/messagehandler/provider_messagebird.go -> mock_provider_messagebird.go
# - pkg/dbhandler/main.go -> mock_main.go
# - pkg/cachehandler/main.go -> mock_main.go
# - pkg/requestexternal/main.go -> mock_main.go
```

### Lint
```bash
# Run golangci-lint
golangci-lint run -v --timeout 5m

# Run vet
go vet $(go list ./...)
```

### Run Locally
```bash
# With environment variables
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/voipbin" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
REDIS_DATABASE=1 \
AUTHTOKEN_MESSAGEBIRD="your-messagebird-token" \
AUTHTOKEN_TELNYX="your-telnyx-token" \
PROMETHEUS_ENDPOINT="/metrics" \
PROMETHEUS_LISTEN_ADDRESS=":2112" \
./bin/message-manager

# Or with flags
./bin/message-manager \
  --database_dsn "user:pass@tcp(127.0.0.1:3306)/voipbin" \
  --rabbitmq_address "amqp://guest:guest@localhost:5672" \
  --redis_address "127.0.0.1:6379" \
  --authtoken_messagebird "your-token" \
  --authtoken_telnyx "your-token"
```

### Docker
```bash
# Build (expects monorepo root context)
docker build -f Dockerfile -t message-manager:latest ../..
```

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared handlers (sockhandler, requesthandler, notifyhandler, databasehandler)
- `monorepo/bin-billing-manager`: Billing validation (balance checks for SMS)
- `monorepo/bin-number-manager`: Phone number management
- `monorepo/bin-hook-manager`: Webhook delivery

**Important**: Builds and Docker images assume parent monorepo directory context is available.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock):
- Mock interfaces are generated in same package as interface definition (e.g., `pkg/dbhandler/mock_main.go`)
- Table-driven tests with struct slices defining test cases
- Context passed to all handler methods

Example test structure:
```go
tests := []struct {
    name    string
    // inputs
    // expected results
}{
    {"normal", ...},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        mc := gomock.NewController(t)
        defer mc.Finish()
        // Setup mocks and test
    })
}
```

## Key Implementation Details

### Message Sending Flow

1. `messageHandler.Send()` validates billing balance via RPC to billing-manager
2. Message created with status `queued` for all targets
3. Provider handler runs in goroutine (non-blocking)
4. Each target sent concurrently via sync.WaitGroup
5. Target statuses updated in database after provider response
6. Prometheus metrics incremented per provider

### Provider Selection

Currently hardcoded to prefer MessageBird, fallback to Telnyx:
```go
// pkg/messagehandler/send.go
handlers := map[message.ProviderName]func(...){
    message.ProviderNameTelnyx:      h.messageHandlerTelnyx.SendMessage,
    message.ProviderNameMessagebird: h.messageHandlerMessagebird.SendMessage,
}
```

### Database Queries

Use Squirrel SQL builder (not raw SQL):
```go
import sq "github.com/Masterminds/squirrel"

sq.Select("*").From("messages").
    Where(sq.Eq{"id": id}).
    Where(sq.Eq{"tm_delete": DefaultTimeStamp})
```

### Soft Deletes

Records use `tm_delete` timestamp. Default value `"9999-01-01 00:00:00.000000"` indicates active records.

### Webhook URI Pattern

Webhook URIs must end with provider suffix:
- `/your/webhook/path/telnyx` → Telnyx webhook handler
- Provider determined by URI suffix matching in `pkg/messagehandler/hook.go`

## Prometheus Metrics

Service exposes metrics on configured endpoint (default `:2112/metrics`):
- `receive_request_process_time` - Histogram of RPC request processing time (labels: type, method)
- `telnyx_number_send_total` - Counter of Telnyx SMS sent (labels: type)
- `messagebird_number_send_total` - Counter of MessageBird SMS sent (labels: type)
