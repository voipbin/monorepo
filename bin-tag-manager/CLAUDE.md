# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

bin-tag-manager is a Go microservice for managing customer tags in a VoIP system. It provides CRUD operations for tags and handles event-driven cascading deletions when customers are removed.

## Build and Test Commands

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

### Configuration

Environment variables / flags:
- `DATABASE_DSN` - MySQL connection string (default: `testid:testpassword@tcp(127.0.0.1:3306)/test`)
- `RABBITMQ_ADDRESS` - RabbitMQ connection (default: `amqp://guest:guest@localhost:5672`)
- `REDIS_ADDRESS` - Redis server address (default: `127.0.0.1:6379`)
- `REDIS_PASSWORD` - Redis password (default: empty)
- `REDIS_DATABASE` - Redis database index (default: `1`)
- `PROMETHEUS_ENDPOINT` - Metrics endpoint path (default: `/metrics`)
- `PROMETHEUS_LISTEN_ADDRESS` - Metrics server address (default: `:2112`)

### API Endpoints (via RabbitMQ RPC)

The service handles REST-like requests through RabbitMQ with URI pattern matching:

- `GET /v1/tags?<params>` - List tags with pagination
- `POST /v1/tags` - Create new tag
- `GET /v1/tags/{tag-id}` - Get specific tag
- `PUT /v1/tags/{tag-id}` - Update tag
- `DELETE /v1/tags/{tag-id}` - Delete tag

### Event Types Published

- `tag_created` - When a new tag is created
- `tag_updated` - When tag information is updated
- `tag_deleted` - When a tag is deleted

### Cache Strategy

- Tags are cached in Redis for fast lookups
- Cache is invalidated on updates and deletions
- Uses `pkg/cachehandler` for all Redis operations
