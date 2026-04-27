# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-outdial-manager` is a Go microservice for managing outbound dialing campaigns in a VoIP system. It manages outdials (campaign containers), outdial targets (individual call targets), and target call tracking.

**Key Concepts:**
- **Outdial**: Container for an outbound dialing campaign — belongs to a customer and a campaign, holds custom JSON data.
- **OutdialTarget**: Individual call target within an outdial; up to 5 destination numbers with independent try counts; status `idle` / `processing` / `done`.
- **OutdialTargetCall**: Call attempt tracking for a single target.
- Used by `bin-campaign-manager` to fetch available targets and update target status during campaign execution.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-outdial-manager`.

## Common Commands

```bash
# Build the daemon
go build -o ./bin/ ./cmd/...

# Run the daemon (requires configuration via flags or env vars)
./bin/outdial-manager

# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html

# Run a single test
go test -v -run TestName ./pkg/packagename/...

# Generate all mocks (uses go.uber.org/mock via //go:generate directives)
go generate ./pkg/dbhandler/...
go generate ./pkg/cachehandler/...
go generate ./pkg/outdialhandler/...
go generate ./pkg/outdialtargethandler/...

# Lint
golint -set_exit_status $(go list ./...)
golangci-lint run -v --timeout 5m

# Vet
go vet $(go list ./...)

# Docker build (from monorepo root)
docker build -t outdial-manager -f bin-outdial-manager/Dockerfile .
```

## outdial-control CLI Tool

A command-line tool for managing outdials directly via database/cache (bypasses RabbitMQ RPC). **All output is JSON format** (stdout), logs go to stderr.

```bash
# Create outdial - returns created outdial JSON
./bin/outdial-control outdial create --customer_id <uuid> [--name] [--detail] [--data '<json>'] [--campaign_id]

# Get outdial - returns outdial JSON
./bin/outdial-control outdial get --id <uuid>

# List outdials - returns JSON array
./bin/outdial-control outdial list --customer_id <uuid> [--limit 100] [--token]

# Update outdial basic info - returns updated outdial JSON
./bin/outdial-control outdial update-basic-info --id <uuid> [--name] [--detail]

# Update outdial campaign ID - returns updated outdial JSON
./bin/outdial-control outdial update-campaign-id --id <uuid> --campaign_id <uuid>

# Update outdial custom data - returns updated outdial JSON
./bin/outdial-control outdial update-data --id <uuid> --data '<json>'

# Delete outdial - returns deleted outdial JSON
./bin/outdial-control outdial delete --id <uuid>
```

Uses same environment variables as outdial-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

## Architecture

### Service Layer Structure

The service follows a layered architecture with handler separation:

1. **cmd/outdial-manager/** - Main daemon entry point with configuration via Viper/pflag (flags and env vars)
2. **pkg/listenhandler/** - RabbitMQ RPC request handler with regex-based URI routing for REST-like API operations
3. **pkg/outdialhandler/** - Core business logic for outdial management
4. **pkg/outdialtargethandler/** - Core business logic for outdial target management
5. **pkg/outdialtargetcallhandler/** - Core business logic for outdial target call tracking
6. **pkg/dbhandler/** - Database operations with MySQL and Redis cache coordination
7. **pkg/cachehandler/** - Redis cache operations
8. **models/** - Data structures (outdial, outdialtarget, outdialtargetcall)

### Inter-Service Communication

- Uses RabbitMQ for message passing between microservices
- Listens on `QueueNameOutdialRequest` for RPC requests
- Publishes webhook events to `QueueNameOutdialEvent` when entities change
- **Monorepo structure**: All sibling services are referenced via `replace` directives in go.mod pointing to `../bin-*-manager` directories. When modifying shared dependencies in `bin-common-handler`, changes affect all services immediately.

### Key Patterns

- Handler interfaces with mock generation using `go.uber.org/mock` (`//go:generate mockgen`)
- Table-driven tests with `testing` package using SQLite in-memory databases
- Test database setup via SQL scripts in `scripts/database_scripts/` loaded using `github.com/smotes/purse`
- Prometheus metrics exposed at configurable endpoint
- Context propagation through all handler methods
- UUID-based entity identification using `github.com/gofrs/uuid`
- Webhook event publishing after entity mutations (create, update, delete)

### Request Flow

```
RabbitMQ Request → listenhandler (regex routing) → outdial/target handlers → dbhandler → MySQL/Redis
                                                                                    ↓
                                                                        webhook event published
```

## Request Routing

ListenHandler uses regex-based routing to handle REST-like RPC requests:

**Outdials API (`/v1/outdials/*`):**
- `POST /v1/outdials` — Create outdial
- `GET /v1/outdials?customer_id=<uuid>` — List outdials by customer
- `GET /v1/outdials/<outdial-id>` — Get outdial
- `PUT /v1/outdials/<outdial-id>` — Update outdial basic info
- `DELETE /v1/outdials/<outdial-id>` — Delete outdial
- `PUT /v1/outdials/<outdial-id>/campaign_id` — Update campaign ID
- `PUT /v1/outdials/<outdial-id>/data` — Update custom data
- `GET /v1/outdials/<outdial-id>/available?try_count_0=N&...&limit=N` — Get available targets
- `POST /v1/outdials/<outdial-id>/targets` — Create target
- `GET /v1/outdials/<outdial-id>/targets?page_size=N&page_token=T` — List targets

**Outdial Targets API (`/v1/outdialtargets/*`):**
- `GET /v1/outdialtargets/<target-id>` — Get target
- `DELETE /v1/outdialtargets/<target-id>` — Delete target
- `POST /v1/outdialtargets/<target-id>/progressing` — Mark target as in progress
- `PUT /v1/outdialtargets/<target-id>/status` — Update target status

## Event Subscriptions

This service does not subscribe to external events. There is no SubscribeHandler.

### Data Models

**Outdial** - Container for outbound dialing campaign:
- Belongs to customer and campaign
- Contains name, detail, and custom JSON data
- Tracks create/update/delete timestamps

**OutdialTarget** - Individual call target within an outdial:
- Up to 5 destination numbers (destination0-4) with corresponding try counts
- Status tracking (idle, processing, done)
- Custom JSON data per target

**OutdialTargetCall** - Call attempt tracking for a target

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared handlers (sockhandler, requesthandler, notifyhandler, databasehandler, utilhandler)
- `monorepo/bin-campaign-manager`: Campaign event models (consumed via the campaign manager's reverse direction)

Always run `go mod vendor` after changing dependencies.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock) plus SQLite in-memory databases:
- Mock interfaces co-located with handlers
- The `TestMain` in `pkg/dbhandler/main_test.go` sets up a shared SQLite test DB using `github.com/smotes/purse` to load schema from `scripts/database_scripts/*.sql`
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

### Multi-Destination Retry Tracking
Each `OutdialTarget` carries 5 destination addresses (`destination_0` through `destination_4`) with independent try-count tracking. The `available` endpoint allows the campaign manager to filter targets by retry count thresholds.

### Soft Deletes
Records use the `tm_delete` timestamp (`"9999-01-01 00:00:00.000000"` for active records).

## Configuration

| Flag / Env | Description | Default |
|------------|-------------|---------|
| `database_dsn` / `DATABASE_DSN` | MySQL connection string | required |
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server (`amqp://user:pass@host:port`) | required |
| `redis_address` / `REDIS_ADDRESS` | Redis cache | required |
| `redis_password` / `REDIS_PASSWORD` | Redis auth | optional |
| `redis_database` / `REDIS_DATABASE` | Redis DB index | optional |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

## Prometheus Metrics

Service exposes metrics on the configured endpoint (default `:2112/metrics`):
- `outdial_manager_receive_request_process_time` — histogram of RPC request processing time (labels: type, method)
