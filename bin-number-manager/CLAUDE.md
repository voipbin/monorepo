# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

bin-number-manager is a Go microservice for telephone number management in a VoIP system. It handles purchasing, managing, and releasing phone numbers through external providers (Telnyx, Twilio).

## Build and Test Commands

```bash
# Build
go build -o ./bin/ ./cmd/...

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

# Generate mocks (each handler package has its own mock generator)
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

1. **cmd/number-manager/** - Entry point with configuration via Cobra/Viper (flags and env vars)
2. **pkg/listenhandler/** - RabbitMQ RPC request handler for REST-like API operations
3. **pkg/subscribehandler/** - Event subscriber handling events from other services (customer-manager, flow-manager)
4. **pkg/numberhandler/** - Core business logic for number operations
5. **pkg/numberhandlertelnyx/** - Telnyx provider-specific implementation
6. **pkg/numberhandlertwilio/** - Twilio provider-specific implementation
7. **pkg/dbhandler/** - Database operations with MySQL
8. **pkg/cachehandler/** - Redis cache operations for number lookups
9. **pkg/requestexternal/** - HTTP client for external provider APIs
10. **models/** - Data structures (number, availablenumber, providernumber)

### Inter-Service Communication

- Uses RabbitMQ for message passing between microservices
- Publishes events to `QueueNameNumberEvent` when numbers change
- Subscribes to `QueueNameFlowEvent` and `QueueNameCustomerEvent` for cascading deletions
- Part of a monorepo with sibling services referenced via `replace` directives in go.mod

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
