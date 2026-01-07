# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

bin-outdial-manager is a Go microservice for managing outbound dialing campaigns in a VoIP system. It manages outdials (campaign containers), outdial targets (individual call targets), and target call tracking.

## Build and Test Commands

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

### API Structure

The listenhandler uses regex-based routing to handle REST-like RPC requests:

**Outdials:**
- `POST /v1/outdials` - Create outdial
- `GET /v1/outdials?customer_id=<uuid>` - List outdials by customer
- `GET /v1/outdials/<outdial-id>` - Get outdial
- `PUT /v1/outdials/<outdial-id>` - Update outdial basic info
- `DELETE /v1/outdials/<outdial-id>` - Delete outdial
- `PUT /v1/outdials/<outdial-id>/campaign_id` - Update campaign ID
- `PUT /v1/outdials/<outdial-id>/data` - Update custom data
- `GET /v1/outdials/<outdial-id>/available?try_count_0=N&...&limit=N` - Get available targets
- `POST /v1/outdials/<outdial-id>/targets` - Create target
- `GET /v1/outdials/<outdial-id>/targets?page_size=N&page_token=T` - List targets

**Outdial Targets:**
- `GET /v1/outdialtargets/<target-id>` - Get target
- `DELETE /v1/outdialtargets/<target-id>` - Delete target
- `POST /v1/outdialtargets/<target-id>/progressing` - Mark target as in progress
- `PUT /v1/outdialtargets/<target-id>/status` - Update target status

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

### Configuration

Environment variables / flags:
- `DATABASE_DSN` - MySQL connection string
- `RABBITMQ_ADDRESS` - RabbitMQ connection (amqp://user:pass@host:port)
- `REDIS_ADDRESS`, `REDIS_PASSWORD`, `REDIS_DATABASE` - Redis cache
- `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS` - Metrics endpoint

### Testing

Tests use SQLite in-memory databases with schema loaded from `scripts/database_scripts/*.sql`. The `TestMain` function in `pkg/dbhandler/main_test.go` sets up the shared test database using `github.com/smotes/purse` to load SQL files.
