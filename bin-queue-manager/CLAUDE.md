# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-queue-manager` is a microservice in the VoIPbin monorepo that manages call queues and queue calls (calls waiting in queues). It routes calls to available agents using configurable routing methods and integrates with other services via RabbitMQ.

**Key Concepts:**
- **Queue**: A call queue with routing configuration, agent tags, wait/service timeouts, and lists of waiting and serviced queuecalls.
- **Queuecall**: A single call sitting in a queue. Status flow: `initiating` → `waiting` → `connecting` → `service` → `done`/`abandoned`.
- **Routing methods**: How callers are matched to agents (e.g., `random`).
- **Wait timeout**: How long a queuecall may sit in the queue before timing out.
- **Service timeout**: Maximum agent service duration before forced cleanup.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-queue-manager`.

## Common Commands

### Building
```bash
# Build the service and CLI
go build -o ./bin/ ./cmd/...

# Build with Docker
docker build -t queue-manager .
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./pkg/queuehandler
go test ./pkg/queuecallhandler

# Run a specific test
go test ./pkg/queuehandler -run TestQueueHandler_Create

# Run tests with verbose output
go test -v ./pkg/queuehandler

# Run tests with coverage
go test -cover ./...
```

### Code Generation
```bash
# Generate mocks (required after interface changes)
go generate ./...
```

### Running
```bash
# Run with default configuration
go run ./cmd/queue-manager

# Run with environment variables
DATABASE_DSN="user:pass@tcp(localhost:3306)/dbname" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="localhost:6379" \
go run ./cmd/queue-manager
```

## queue-control CLI Tool

A command-line tool for managing queues directly via database/cache (bypasses RabbitMQ RPC). **All output is JSON format** (stdout), logs go to stderr.

```bash
# Create queue - returns created queue JSON
./bin/queue-control queue create --customer_id <uuid> --name <name> [--detail] [--routing_method random] [--tag_ids '<json>'] [--wait_flow_id <uuid>] [--wait_timeout 300000] [--service_timeout 600000]

# Get queue - returns queue JSON
./bin/queue-control queue get --id <uuid>

# List queues - returns JSON array
./bin/queue-control queue list --customer_id <uuid> [--limit 100] [--token]

# Update queue (full) - returns updated queue JSON
./bin/queue-control queue update --id <uuid> --name <name> [--detail] [--routing_method] [--tag_ids '<json>'] [--wait_flow_id <uuid>] [--wait_timeout] [--service_timeout]

# Update queue tag IDs only - returns updated queue JSON
./bin/queue-control queue update-tag-ids --id <uuid> --tag_ids '<json_array>'

# Update queue routing method only - returns updated queue JSON
./bin/queue-control queue update-routing-method --id <uuid> --routing_method <random>

# Update queue execute state - returns updated queue JSON
./bin/queue-control queue update-execute --id <uuid> --execute <run|stop>

# Delete queue - returns deleted queue JSON
./bin/queue-control queue delete --id <uuid>
```

Uses same environment variables as queue-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

## Architecture

### Core Domain Models

- **Queue** (`models/queue/queue.go`): Represents a call queue with routing configuration, agent tags, wait/service timeouts, and lists of waiting/serviced queuecalls
- **Queuecall** (`models/queuecall/queuecall.go`): Represents a call in a queue with status tracking (initiating → waiting → connecting → service → done/abandoned), timeouts, and duration metrics

### Handler Architecture

The service uses a layered handler architecture where handlers are dependency-injected interfaces:

1. **listenhandler** (`pkg/listenhandler/`): HTTP-like request router over RabbitMQ
   - Listens to `QueueNameQueueRequest` queue for RPC-style requests
   - Routes requests using regex patterns (e.g., `/v1/queues`, `/v1/queuecalls/{id}`)
   - Delegates to queuehandler and queuecallhandler based on URI patterns

2. **subscribehandler** (`pkg/subscribehandler/`): Event consumer from other services
   - Subscribes to events from: call-manager, agent-manager, conference-manager, customer-manager
   - Processes events like call hangup, confbridge joined/leaved, customer deleted
   - Subscribes to `QueueNameCallEvent`, `QueueNameAgentEvent`, `QueueNameConferenceEvent`

3. **queuehandler** (`pkg/queuehandler/`): Business logic for Queue operations
   - CRUD operations for queues
   - Executes queue logic (matching agents to waiting calls)
   - Manages queue execution state (run/stop)
   - See `QueueHandler` interface in `pkg/queuehandler/main.go:28`

4. **queuecallhandler** (`pkg/queuecallhandler/`): Business logic for Queuecall operations
   - CRUD operations for queuecalls
   - Manages queuecall lifecycle and status transitions
   - Handles timeouts (wait timeout, service timeout)
   - Executes agent assignment to queuecalls
   - Health checks for queuecalls
   - See `QueuecallHandler` interface in `pkg/queuecallhandler/main.go:33`

5. **dbhandler** (`pkg/dbhandler/`): Database operations
   - Abstracts MySQL database access using Squirrel query builder
   - Uses `go-sql-driver/mysql` for MySQL connections
   - Interface defined in `pkg/dbhandler/main.go`

6. **cachehandler** (`pkg/cachehandler/`): Redis caching layer
   - Wraps Redis operations using `go-redis/redis/v8`
   - Used by dbhandler for caching database queries

### Communication Pattern

**RabbitMQ-based RPC/Events:**
- Uses `bin-common-handler/pkg/sockhandler` for RabbitMQ abstraction
- **RPC Requests**: Processed by listenhandler via `ConsumeRPC` (synchronous request/response)
- **Events**: Published via `notifyhandler` to `QueueNameQueueEvent` queue
- **Subscriptions**: Consumed by subscribehandler via `ConsumeMessage` (asynchronous)

### Initialization Flow

1. `cmd/queue-manager/init.go`: Parses flags/env vars using Viper and pflags
   - Configuration: database DSN, RabbitMQ address, Redis address, Prometheus settings
2. `cmd/queue-manager/main.go:main()`: Connects to database and cache
3. `run()`: Creates handlers (dbhandler → queuehandler → queuecallhandler)
4. `runListen()`: Starts RPC request listener
5. `runSubscribe()`: Starts event subscription handlers

## Request Routing

ListenHandler routes RPC requests using regex patterns matching REST-like URIs:

**Queues API (`/v1/queues/*`):**
- `POST /v1/queues` — Create queue
- `GET /v1/queues?<filters>` — List queues (pagination)
- `GET /v1/queues/<uuid>` — Get queue
- `PUT /v1/queues/<uuid>` — Update queue
- `PUT /v1/queues/<uuid>/tag_ids` — Update queue tag IDs
- `PUT /v1/queues/<uuid>/routing_method` — Update routing method
- `PUT /v1/queues/<uuid>/execute` — Update execute state (`run`/`stop`)
- `DELETE /v1/queues/<uuid>` — Delete queue

**Queuecalls API (`/v1/queuecalls/*`):**
- `POST /v1/queuecalls` — Create queuecall (place a call into a queue)
- `GET /v1/queuecalls?<filters>` — List queuecalls
- `GET /v1/queuecalls/<uuid>` — Get queuecall
- `POST /v1/queuecalls/<uuid>/health-check` — Health-check the queuecall
- `DELETE /v1/queuecalls/<uuid>` — Remove a queuecall

## Event Subscriptions

SubscribeHandler subscribes to:
- **bin-manager.call-manager.event**: `call_hangup`, `confbridge_joined`, `confbridge_leaved` — drives queuecall state transitions.
- **bin-manager.agent-manager.event**: agent status changes — re-evaluates queue routing.
- **bin-manager.conference-manager.event**: confbridge state events for queue-type conferences.
- **bin-manager.customer-manager.event**: `customer_deleted` — cleans up queues owned by the deleted customer.

## Monorepo Context

This service is part of a Go monorepo with `replace` directives in `go.mod` for local dependencies:
- `monorepo/bin-common-handler`: Shared utilities (sockhandler, requesthandler, notifyhandler)
- `monorepo/bin-call-manager`: Call management models/events
- `monorepo/bin-agent-manager`: Agent models
- `monorepo/bin-customer-manager`: Customer models/events
- `monorepo/bin-conference-manager`: Confbridge models for queue-type conferences

When modifying code that affects other managers, check for cross-service impacts via event contracts.

## Testing Patterns

- Tests use `go.uber.org/mock` for mocking interfaces
- Generate mocks with `//go:generate mockgen` directives (see handler `main.go` files)
- Test files follow `*_test.go` naming convention
- Mock files are named `mock_main.go` in each package
- Database tests may use `scripts/database_scripts_test/` for test fixtures

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

### Queuecall Status Lifecycle
`initiating` → `waiting` → `connecting` → `service` → `done` / `abandoned`. Transitions are driven by RPC entrypoints and `bin-call-manager` events.

### Timeouts
Each queue has a `wait_timeout` (max time waiting before abandonment) and `service_timeout` (max agent service time). Both are enforced by background timer logic in `queuecallhandler`.

### Soft Deletes
Records use `tm_delete` timestamp (`"9999-01-01 00:00:00.000000"` for active records).

## Configuration

All configuration uses environment variables or flags (via Viper):

| Flag / Env | Description | Default |
|------------|-------------|---------|
| `database_dsn` / `DATABASE_DSN` | MySQL connection string | required |
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server | required |
| `redis_address` / `REDIS_ADDRESS` | Redis cache | required |
| `redis_password` / `REDIS_PASSWORD` | Redis auth | optional |
| `redis_database` / `REDIS_DATABASE` | Redis DB index | optional |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

See `cmd/queue-manager/init.go` for the authoritative list and defaults.

## Prometheus Metrics

Service exposes metrics on the configured endpoint (default `:2112/metrics`):
- `queue_manager_receive_request_process_time` — histogram of RPC request processing time (labels: type, method)
- `queue_manager_subscribe_event_process_time` — histogram of event processing time (labels: publisher, type)
