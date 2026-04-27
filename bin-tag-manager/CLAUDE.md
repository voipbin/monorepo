# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-tag-manager` is a Go microservice for managing customer tags in a VoIP system. It provides CRUD operations for tags and handles event-driven cascading deletions when customers are removed.

**Key Concepts:**
- **Tag**: A customer-scoped label (name, optional detail) used by other services (e.g., `bin-contact-manager`, `bin-queue-manager`) to categorize records.
- **Cascading deletes**: When a customer is deleted, all of that customer's tags are removed automatically.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-tag-manager`.

## Common Commands

```bash
# Build the service and CLI
go build -o ./bin/ ./cmd/...

# Run the daemon (requires configuration via flags or env vars)
./bin/tag-manager

# Run the CLI tool
./bin/tag-control

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
go generate ./pkg/listenhandler/...
go generate ./pkg/subscribehandler/...
go generate ./pkg/taghandler/...
go generate ./pkg/dbhandler/...
go generate ./pkg/cachehandler/...
```

## tag-control CLI Tool

A command-line tool for managing tags directly via database/cache (bypasses RabbitMQ RPC). **All output is JSON format** (stdout), logs go to stderr.

```bash
# Create tag - returns created tag JSON
./bin/tag-control tag create --customer_id <uuid> --name <name> [--detail <detail>]

# Get tag - returns tag JSON
./bin/tag-control tag get --id <uuid>

# List tags - returns JSON array
./bin/tag-control tag list --customer_id <uuid> [--limit 100] [--token]

# Update tag - returns updated tag JSON
./bin/tag-control tag update --id <uuid> --name <name> [--detail <detail>]

# Delete tag - returns deleted tag JSON
./bin/tag-control tag delete --id <uuid>
```

Uses same environment variables as tag-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

## Architecture

### Service Layer Structure

The service follows a layered architecture with handler separation:

1. **cmd/tag-manager/** - Main daemon entry point with configuration via pflag/Viper (flags and env vars)
2. **pkg/listenhandler/** - RabbitMQ RPC request handler with regex-based URI routing for REST-like API operations
3. **pkg/subscribehandler/** - Event subscriber handling cascading deletions from customer-manager
4. **pkg/taghandler/** - Core business logic for tag CRUD operations and event publishing
5. **pkg/dbhandler/** - Database operations with MySQL and Redis cache coordination
6. **pkg/cachehandler/** - Redis cache operations for tag lookups
7. **models/tag/** - Data structures (Tag, event types, webhook)

### Inter-Service Communication

- Uses RabbitMQ for message passing between microservices
- Listens on `QueueNameTagRequest` for RPC requests
- Publishes events to `QueueNameTagEvent` when tags change (created, updated, deleted)
- Subscribes to `QueueNameCustomerEvent` for cascading deletions when customers are removed
- **Monorepo structure**: All sibling services are referenced via `replace` directives in go.mod pointing to `../bin-*-manager` directories. When modifying shared dependencies, changes affect all services immediately.

### Key Patterns

- Handler interfaces with mock generation using `go.uber.org/mock` (`//go:generate mockgen`)
- Table-driven tests with `testing` package
- Prometheus metrics exposed at configurable endpoint (default `:2112/metrics`)
- Context propagation through all handler methods
- UUID-based entity identification using `github.com/gofrs/uuid`

### Request Flow

```
RabbitMQ Request → listenhandler (regex routing) → taghandler → dbhandler → MySQL/Redis
                                                              ↓
                                                          notifyhandler → RabbitMQ event publish

Event Flow:
RabbitMQ Event → subscribehandler → taghandler → cleanup operations
```

## Request Routing

The service handles REST-like requests through RabbitMQ with URI pattern matching:

**Tags API (`/v1/tags/*`):**
- `GET /v1/tags?<params>` — List tags with pagination
- `POST /v1/tags` — Create new tag
- `GET /v1/tags/{tag-id}` — Get specific tag
- `PUT /v1/tags/{tag-id}` — Update tag
- `DELETE /v1/tags/{tag-id}` — Delete tag

**Events Published:**
- `tag_created` — when a new tag is created
- `tag_updated` — when tag information is updated
- `tag_deleted` — when a tag is deleted

## Event Subscriptions

SubscribeHandler subscribes to:
- **bin-manager.customer-manager.event**: `customer_deleted` → cascading deletion of tags owned by the deleted customer.

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared utilities (sockhandler, requesthandler, notifyhandler)
- `monorepo/bin-customer-manager`: Customer event models for cascading deletes

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

### Cache Strategy
Tags are cached in Redis for fast lookups; cache is invalidated on updates and deletions. Uses `pkg/cachehandler` for all Redis operations.

### Soft Deletes
Records use `tm_delete` timestamp (`"9999-01-01 00:00:00.000000"` for active records).

## Configuration

Environment variables / flags:

| Flag / Env | Description | Default |
|------------|-------------|---------|
| `database_dsn` / `DATABASE_DSN` | MySQL connection string | `testid:testpassword@tcp(127.0.0.1:3306)/test` |
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server | `amqp://guest:guest@localhost:5672` |
| `redis_address` / `REDIS_ADDRESS` | Redis server | `127.0.0.1:6379` |
| `redis_password` / `REDIS_PASSWORD` | Redis password | empty |
| `redis_database` / `REDIS_DATABASE` | Redis DB index | `1` |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

## Prometheus Metrics

Service exposes metrics on the configured endpoint (default `:2112/metrics`):
- `tag_manager_receive_request_process_time` — histogram of RPC request processing time (labels: type, method)
- `tag_manager_subscribe_event_process_time` — histogram of event processing time (labels: publisher, type)
