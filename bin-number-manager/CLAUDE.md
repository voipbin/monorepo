# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

bin-number-manager is a Go microservice for telephone number management in a VoIP system. It handles purchasing, managing, and releasing phone numbers through external providers (Telnyx, Twilio).

## Build and Test Commands

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

### Configuration

Environment variables / flags:
- `DATABASE_DSN` - MySQL connection string
- `RABBITMQ_ADDRESS` - RabbitMQ connection
- `REDIS_ADDRESS`, `REDIS_PASSWORD`, `REDIS_DATABASE` - Redis cache
- `TELNYX_CONNECTION_ID`, `TELNYX_PROFILE_ID`, `TELNYX_TOKEN` - Telnyx credentials
- `TWILIO_SID`, `TWILIO_TOKEN` - Twilio credentials
- `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS` - Metrics endpoint

### number-control CLI Tool

A command-line tool for managing phone numbers. **All output is JSON format** (stdout), logs go to stderr.

```bash
# Create number - returns created number JSON
./bin/number-control number create --customer_id <uuid> --number "+15551234567" [--call_flow_id <uuid>] [--message_flow_id <uuid>] [--name] [--detail]

# Register number (manual registration without provider) - returns registered number JSON
./bin/number-control number register --customer_id <uuid> --number "+15551234567" [--call_flow_id <uuid>] [--message_flow_id <uuid>] [--name] [--detail]

# Get number - returns number JSON
./bin/number-control number get --id <uuid>

# List numbers - returns JSON array
./bin/number-control number list --customer_id <uuid> [--limit 100] [--token]

# Delete number - returns deleted number JSON
./bin/number-control number delete --id <uuid>
```

Uses same environment variables as number-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).
