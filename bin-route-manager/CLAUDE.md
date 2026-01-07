# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

bin-route-manager is a Go microservice for call routing in a VoIP system. It manages routing providers and routes, providing route selection logic for outbound calls based on customer configuration and target destinations.

## Build and Test Commands

```bash
# Build the route-manager daemon
go build -o ./bin/ ./cmd/...

# Run the daemon (requires configuration via flags or env vars)
./bin/route-manager

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
go generate ./pkg/routehandler/...
go generate ./pkg/providerhandler/...
go generate ./pkg/dbhandler/...
go generate ./pkg/cachehandler/...
```

## Architecture

### Service Layer Structure

The service follows a layered architecture with handler separation:

1. **cmd/route-manager/** - Main daemon entry point with configuration via Viper/pflag (flags and env vars)
2. **pkg/listenhandler/** - RabbitMQ RPC request handler with regex-based URI routing for REST-like API operations
3. **pkg/routehandler/** - Core route business logic including dialroute selection
4. **pkg/providerhandler/** - Provider management business logic
5. **pkg/dbhandler/** - Database operations with MySQL and Redis cache coordination
6. **pkg/cachehandler/** - Redis cache operations for provider/route lookups
7. **models/provider/** - Provider data structures and types
8. **models/route/** - Route data structures and constants

### Inter-Service Communication

- Uses RabbitMQ for message passing between microservices
- Listens on a configurable queue for RPC-style REST API requests (GET/POST/PUT/DELETE)
- Returns responses via RabbitMQ reply-to mechanism
- **Monorepo structure**: All sibling services are referenced via `replace` directives in go.mod pointing to `../bin-*-manager` directories. When modifying shared dependencies, changes affect all services immediately.

### Key Patterns

- Handler interfaces with mock generation using `go.uber.org/mock` (`//go:generate mockgen`)
- Table-driven tests with `testing` package
- Prometheus metrics exposed at configurable endpoint
- Context propagation through all handler methods
- UUID-based entity identification using `github.com/gofrs/uuid`
- Regex-based URI routing in listenhandler/main.go:processRequest

### Request Flow

```
RabbitMQ Request → listenhandler (regex routing) → route/provider handler → dbhandler → MySQL/Redis
                                                                         ↓
                                                                   cache lookup/update
```

### Configuration

Environment variables / flags (managed via Viper/pflag):
- `DATABASE_DSN` - MySQL connection string (e.g., `user:password@tcp(localhost:3306)/dbname`)
- `RABBITMQ_ADDRESS` - RabbitMQ connection (e.g., `amqp://guest:guest@localhost:5672`)
- `REDIS_ADDRESS`, `REDIS_PASSWORD`, `REDIS_DATABASE` - Redis cache configuration
- `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS` - Metrics endpoint configuration

## Core Concepts

### Providers

Providers represent SIP trunks/gateways for routing outbound calls. Each provider has:
- Type (currently only "sip" is supported)
- Hostname (SIP destination)
- Tech prefix/postfix (dial string manipulation)
- Tech headers (custom SIP headers)

See models/provider/provider.go for the Provider struct.

### Routes

Routes map customer destinations to providers with priority ordering:
- Each route belongs to a customer_id
- Target can be a country code or "all" for default routing
- Priority determines route order (lower = higher priority)
- Basic route customer_id `00000000-0000-0000-0000-000000000001` provides system-wide defaults

See models/route/route.go for the Route struct.

### Dialroute Selection

The dialroute logic (pkg/routehandler/dialroute.go:DialrouteGets) implements a fallback mechanism:
1. First retrieves customer-specific routes for the target destination
2. Then retrieves default routes (CustomerIDBasicRoute) for the same target
3. Merges results, prioritizing customer routes while adding default routes that use providers not already in the customer routes

This allows customers to override specific providers while falling back to system defaults for others.

## API Endpoints

Handled via RabbitMQ RPC with regex routing in pkg/listenhandler/main.go:

### Providers
- `GET /v1/providers?page_size=N&page_token=T` - List providers with pagination
- `POST /v1/providers` - Create provider
- `GET /v1/providers/{id}` - Get single provider
- `PUT /v1/providers/{id}` - Update provider
- `DELETE /v1/providers/{id}` - Delete provider

### Routes
- `GET /v1/routes?page_size=N&page_token=T` - List routes with pagination
- `POST /v1/routes` - Create route
- `GET /v1/routes/{id}` - Get single route
- `PUT /v1/routes/{id}` - Update route
- `DELETE /v1/routes/{id}` - Delete route

### Dialroute
- `GET /v1/dialroutes?customer_id={id}&target={target}` - Get effective routes for dialing (merges customer and default routes)

## Database Schema

Tables are defined in scripts/database_scripts/:
- **table_providers.sql** - Provider configuration
- **table_routes.sql** - Route mappings

Both tables use soft-delete pattern with tm_delete timestamp.
