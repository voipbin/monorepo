# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

bin-transfer-manager is a Go microservice that handles call transfer operations in a VoIP system. It manages both attended transfers (where the transferer can speak to the transferee before completing the transfer) and blind transfers (immediate transfer without consultation).

## Build and Test Commands

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

### Configuration

Environment variables / flags:
- `DATABASE_DSN` - MySQL connection string
- `RABBITMQ_ADDRESS` - RabbitMQ connection (e.g., `amqp://guest:guest@localhost:5672`)
- `REDIS_ADDRESS`, `REDIS_PASSWORD`, `REDIS_DATABASE` - Redis cache
- `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS` - Metrics endpoint

### Dependencies on Other Services

- **bin-call-manager**: For call, groupcall, and confbridge operations (creating groupcalls, managing confbridge flags, call muting/MOH)
- **bin-flow-manager**: For flow information used in outbound call routing
- **bin-common-handler**: Shared models (address, identity, outline) and handlers (requesthandler, notifyhandler, sockhandler)
