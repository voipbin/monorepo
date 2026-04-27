# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-number-manager` is a Go microservice for telephone number management in a VoIP system. It handles purchasing, managing, and releasing phone numbers through external providers (Telnyx, Twilio).

**Key Concepts:**
- **Number**: A purchased phone number with assigned call flow and/or message flow IDs.
- **AvailableNumber**: A number available for purchase from a provider, queried by country code.
- **ProviderNumber**: Internal mapping between a `Number` and the provider that owns it.
- **Provider routing**: Both Telnyx and Twilio are supported; provider-specific subhandlers wrap the external APIs.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-number-manager`.

## Common Commands

```bash
# Build both number-manager daemon and number-control CLI
go build -o ./bin/ ./cmd/...

# Run the daemon (requires configuration via flags or env vars)
./bin/number-manager

# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html

# Run a single test
go test -v -run TestName ./pkg/packagename/...

# Lint
golint -set_exit_status $(go list ./...)
golangci-lint run -v --timeout 5m

# Vet
go vet $(go list ./...)

# Generate all mocks (uses go.uber.org/mock via //go:generate directives)
go generate ./pkg/numberhandler/...
go generate ./pkg/dbhandler/...
go generate ./pkg/cachehandler/...
go generate ./pkg/requestexternal/...
go generate ./pkg/numberhandlertelnyx/...
go generate ./pkg/numberhandlertwilio/...
```

## number-control CLI Tool

A command-line tool for managing phone numbers directly via database/cache (bypasses RabbitMQ RPC). **All output is JSON format** (stdout), logs go to stderr.

```bash
# Create a number - returns created number JSON
./bin/number-control number create --customer-id <uuid> --number <phone_number> [--call-flow-id <uuid>] [--message-flow-id <uuid>] [--name] [--detail]

# Register a number (internal use) - returns registered number JSON
./bin/number-control number register --customer-id <uuid> --number <phone_number> [--call-flow-id <uuid>] [--message-flow-id <uuid>] [--name] [--detail]

# Get a number - returns number JSON
./bin/number-control number get --id <uuid>

# Get available numbers for purchase - returns available numbers JSON array
./bin/number-control number get-available --country-code <US|GB|...> [--limit 10]

# List numbers - returns JSON array
./bin/number-control number list --customer-id <uuid> [--limit 100] [--token]

# Update a number - returns updated number JSON
./bin/number-control number update --id <uuid> [--name] [--detail] [--call-flow-id <uuid>] [--message-flow-id <uuid>] [--status active|inactive]

# Delete a number - returns deleted number JSON
./bin/number-control number delete --id <uuid>
```

Uses same environment variables as number-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, `TELNYX_*`, `TWILIO_*`, etc.).

## Architecture

### Service Layer Structure

The service follows a layered architecture with handler separation:

1. **cmd/number-manager/** - Main daemon entry point with configuration via Cobra/Viper (flags and env vars)
2. **cmd/number-control/** - CLI tool for interactive number management operations
3. **pkg/listenhandler/** - RabbitMQ RPC request handler with regex-based URI routing for REST-like API operations
4. **pkg/subscribehandler/** - Event subscriber handling cascading deletions from other services (customer-manager, flow-manager)
5. **pkg/numberhandler/** - Core business logic coordinating provider handlers
6. **pkg/numberhandlertelnyx/** - Telnyx provider-specific implementation
7. **pkg/numberhandlertwilio/** - Twilio provider-specific implementation
8. **pkg/dbhandler/** - Database operations with MySQL and Redis cache coordination
9. **pkg/cachehandler/** - Redis cache operations for number lookups
10. **pkg/requestexternal/** - HTTP client for external provider APIs
11. **models/** - Data structures (number, availablenumber, providernumber)
12. **internal/config/** - Singleton configuration management with Viper

### Inter-Service Communication

- Uses RabbitMQ for message passing between microservices
- Publishes events to `QueueNameNumberEvent` when numbers change
- Subscribes to `QueueNameFlowEvent` and `QueueNameCustomerEvent` for cascading deletions
- **Monorepo structure**: All sibling services are referenced via `replace` directives in go.mod pointing to `../bin-*-manager` directories. When modifying shared dependencies, changes affect all services immediately.

### Key Patterns

- Handler interfaces with mock generation using `go.uber.org/mock` (`//go:generate mockgen`)
- Table-driven tests with `testing` package
- Prometheus metrics exposed at configurable endpoint
- Context propagation through all handler methods
- UUID-based entity identification using `github.com/gofrs/uuid`

### Request Flow

```
RabbitMQ Request → listenhandler (regex routing) → numberhandler → provider handler (telnyx/twilio) → external API
                                                              ↓
                                                          dbhandler → MySQL/Redis
```

## Request Routing

ListenHandler routes RPC requests using regex patterns matching REST-like URIs:

**Numbers API (`/v1/numbers/*`):**
- `POST /v1/numbers` — Create (purchase) number
- `POST /v1/numbers/register` — Register existing number without going through a provider
- `GET /v1/numbers?<filters>` — List numbers (pagination)
- `GET /v1/numbers/<uuid>` — Get number
- `PUT /v1/numbers/<uuid>` — Update number (call/message flow IDs, status)
- `DELETE /v1/numbers/<uuid>` — Release number

**Available Numbers API (`/v1/available_numbers/*`):**
- `GET /v1/available_numbers?country_code=<code>&limit=<n>` — Search provider for purchasable numbers

## Event Subscriptions

SubscribeHandler subscribes to:
- **bin-manager.customer-manager.event**: `customer_deleted` → release all numbers for the deleted customer.
- **bin-manager.flow-manager.event**: `flow_deleted` → unset `call_flow_id`/`message_flow_id` references on affected numbers.

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared utilities (sockhandler, requesthandler, notifyhandler)
- `monorepo/bin-customer-manager`: Customer event models for cascading deletes
- `monorepo/bin-flow-manager`: Flow event models for flow-deletion cleanup

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

### Provider Strategy
The numberhandler delegates to provider-specific sub-handlers (`numberhandlertelnyx`, `numberhandlertwilio`). Each provider implements the same interface for purchase, release, and listing operations.

### Soft Deletes
Records use `tm_delete` timestamp (`"9999-01-01 00:00:00.000000"` for active records).

### Cache Strategy
Redis cache fronts MySQL for number lookups. Mutations invalidate the relevant Redis keys.

## Configuration

Environment variables / flags:

| Flag / Env | Description | Default |
|------------|-------------|---------|
| `database_dsn` / `DATABASE_DSN` | MySQL connection string | required |
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server | required |
| `redis_address` / `REDIS_ADDRESS` | Redis cache | required |
| `redis_password` / `REDIS_PASSWORD` | Redis auth | optional |
| `redis_database` / `REDIS_DATABASE` | Redis DB index | optional |
| `telnyx_connection_id` / `TELNYX_CONNECTION_ID` | Telnyx connection ID | required if Telnyx used |
| `telnyx_profile_id` / `TELNYX_PROFILE_ID` | Telnyx messaging profile | required if Telnyx used |
| `telnyx_token` / `TELNYX_TOKEN` | Telnyx API token | required if Telnyx used |
| `twilio_sid` / `TWILIO_SID` | Twilio account SID | required if Twilio used |
| `twilio_token` / `TWILIO_TOKEN` | Twilio auth token | required if Twilio used |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

## Prometheus Metrics

Service exposes metrics on the configured endpoint (default `:2112/metrics`):
- `number_manager_receive_request_process_time` — histogram of RPC request processing time (labels: type, method)
- `number_manager_subscribe_event_process_time` — histogram of event processing time (labels: publisher, type)
