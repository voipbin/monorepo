# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-transfer-manager` is a Go microservice that handles call transfer operations in a VoIP system. It manages both attended transfers (where the transferer can speak to the transferee before completing the transfer) and blind transfers (immediate transfer without consultation).

**Key Concepts:**
- **Attended Transfer**: Transferer speaks to the transferee first; existing bridge participants are placed on MOH and muted while the consult call happens.
- **Blind Transfer**: Immediate handoff — transferer hangs up as soon as the transferee answers.
- **Block / Execute / Unblock phases**: Each transfer type has a block-state setup, an execute step that creates the groupcall to the transferee, and an unblock/rollback step if the transfer fails.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-transfer-manager`.

## Common Commands

```bash
# Build the transfer-manager daemon
go build -o ./bin/transfer-manager ./cmd/transfer-manager

# Build the transfer-control CLI tool
go build -o ./bin/transfer-control ./cmd/transfer-control

# Run the daemon (requires configuration via flags or env vars)
./bin/transfer-manager

# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html

# Run a single test
go test -v -run TestName ./pkg/packagename/...

# Generate all mocks (uses go.uber.org/mock via //go:generate directives)
go generate ./pkg/transferhandler/...
go generate ./pkg/dbhandler/...
go generate ./pkg/cachehandler/...
go generate ./pkg/subscribehandler/...
```

### transfer-control CLI Tool

A command-line tool for managing transfer operations. **All output is JSON format** (stdout), logs go to stderr.

```bash
# Start a transfer service - returns created transfer JSON
./bin/transfer-control transfer service-start --transfer-type <attended|blind> --transferer-call-id <uuid> --transferee-addresses '<json_array>'

# Get transfer by groupcall ID - returns transfer JSON
./bin/transfer-control transfer get-by-groupcall --groupcall_id <uuid>

# Get transfer by transferer call ID - returns transfer JSON
./bin/transfer-control transfer get-by-call --call_id <uuid>
```

Uses same environment variables as transfer-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

## Architecture

### Service Layer Structure

The service follows a layered architecture with handler separation:

1. **cmd/transfer-manager/** - Main daemon entry point with configuration via Viper/pflag (flags and env vars)
2. **pkg/listenhandler/** - RabbitMQ RPC request handler with regex-based URI routing (`/v1/transfers`)
3. **pkg/subscribehandler/** - Event subscriber handling call-manager events (groupcall_progressing, groupcall_hangup, call_hangup)
4. **pkg/transferhandler/** - Core business logic implementing attended and blind transfer workflows
5. **pkg/dbhandler/** - Database operations with MySQL and Redis cache coordination
6. **pkg/cachehandler/** - Redis cache operations for transfer lookups
7. **models/transfer/** - Data structures (Transfer, Type, Webhook)

### Inter-Service Communication

- Uses RabbitMQ for message passing between microservices
- Listens on `QueueNameTransferRequest` for incoming transfer requests
- Subscribes to `QueueNameCallEvent` from call-manager for transfer state management
- Publishes events to `QueueNameTransferEvent` when transfers change state
- **Monorepo structure**: All sibling services are referenced via `replace` directives in go.mod pointing to `../bin-*-manager` directories. When modifying shared dependencies (especially bin-common-handler, bin-call-manager, bin-flow-manager), changes affect all services immediately.

### Transfer Types and Workflows

#### Attended Transfer
1. **Block phase** (`attendedBlock`): Places existing bridge participants on hold with music-on-hold and mutes their input
2. **Execute phase** (`attendedExecute`): Creates new groupcall to transferee while transferer remains connected
3. **Answer handling**: When transferee answers (via `processEventCMGroupcallProgressing`), both parties are bridged together
4. **Completion**: Transferer can complete the transfer or retrieve the call
5. **Rollback** (`attendedUnblock`): If transfer fails, removes hold/mute from bridge participants

#### Blind Transfer
1. **Block phase** (`blindBlock`): Sets `FlagNoAutoLeave` on confbridge to prevent automatic destruction
2. **Execute phase** (`blindExecute`): Immediately hangs up transferer call and creates groupcall to transferee
3. **Completion**: When transferee answers, they're connected to the remaining bridge participants
4. **Rollback** (`blindUnblock`): Removes `FlagNoAutoLeave` if transfer fails

### Key Patterns

- Handler interfaces with mock generation using `go.uber.org/mock` (`//go:generate mockgen`)
- Table-driven tests in `*_test.go` files
- Prometheus metrics exposed at configurable endpoint
- Context propagation through all handler methods
- UUID-based entity identification using `github.com/gofrs/uuid`
- Error wrapping with `github.com/pkg/errors` for stack traces

### Request Flow

```
RabbitMQ Request → listenhandler (regex routing) → transferhandler → call-manager (via requesthandler)
                                                                  ↓
                                                              dbhandler → MySQL/Redis

Call Events → subscribehandler → transferhandler → state transitions
```

## Request Routing

ListenHandler routes RPC requests using regex patterns matching REST-like URIs:

**Transfers API (`/v1/transfers/*`):**
- `POST /v1/transfers` — Start transfer service (`attended` or `blind`)
- `GET /v1/transfers/<uuid>` — Get transfer
- `GET /v1/transfers?<filters>` — List transfers
- `DELETE /v1/transfers/<uuid>` — Cancel transfer / rollback

## Event Subscriptions

SubscribeHandler subscribes to:
- **bin-manager.call-manager.event**: `groupcall_progressing` (transferee answered → bridge parties), `groupcall_hangup` and `call_hangup` (rollback or finalize transfer state).

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-call-manager`: Call, groupcall, and confbridge operations (creating groupcalls, managing confbridge flags, call muting/MOH)
- `monorepo/bin-flow-manager`: Flow information used in outbound call routing
- `monorepo/bin-common-handler`: Shared models (address, identity, outline) and handlers (requesthandler, notifyhandler, sockhandler)

Always run `go mod vendor` after changing dependencies.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock):
- Mock interfaces co-located with handlers (`mock_*.go`)
- Table-driven tests with struct slices
- Tests cover attended/blind transfer state transitions, rollback paths, and error handling

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

### Attended Transfer Block/Unblock
`attendedBlock` places existing bridge participants on hold (MOH) and mutes their input; `attendedUnblock` reverses this on rollback. Failing to unblock leaves the original parties stuck on hold.

### Blind Transfer ConfBridge Flag
`blindBlock` sets `FlagNoAutoLeave` on the confbridge so the bridge survives the transferer hangup. `blindUnblock` clears the flag if the transfer fails.

### Soft Deletes
Records use `tm_delete` timestamp (`"9999-01-01 00:00:00.000000"` for active records).

## Configuration

Environment variables / flags:

| Flag / Env | Description | Default |
|------------|-------------|---------|
| `database_dsn` / `DATABASE_DSN` | MySQL connection string | required |
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server | required |
| `redis_address` / `REDIS_ADDRESS` | Redis cache | required |
| `redis_password` / `REDIS_PASSWORD` | Redis auth | optional |
| `redis_database` / `REDIS_DATABASE` | Redis DB index | optional |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

## Prometheus Metrics

Service exposes metrics on the configured endpoint (default `:2112/metrics`):
- `transfer_manager_receive_request_process_time` — histogram of RPC request processing time (labels: type, method)
- `transfer_manager_subscribe_event_process_time` — histogram of event processing time (labels: publisher, type)
