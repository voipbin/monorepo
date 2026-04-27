# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-route-manager` is a Go microservice for call routing in a VoIP system. It manages routing providers and routes, providing route selection logic for outbound calls based on customer configuration and target destinations.

**Key Concepts:**
- **Provider**: SIP trunk/gateway used for outbound call routing (hostname, tech prefix/postfix, custom SIP headers).
- **Route**: Maps a customer destination (country code or `all`) to a provider with a priority order.
- **Dialroute**: Effective merged route list for a (customer, target) pair — customer-specific routes override system defaults from `CustomerIDBasicRoute` (`00000000-0000-0000-0000-000000000001`).

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-route-manager`.

## Common Commands

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

## route-control CLI Tool

A command-line tool for managing routes directly via database/cache (bypasses RabbitMQ RPC). **All output is JSON format** (stdout), logs go to stderr.

```bash
# Create route - returns created route JSON
./bin/route-control route create --customer_id <uuid> --target <target> --provider <provider> [--name] [--detail] [--priority]

# Get route - returns route JSON
./bin/route-control route get --id <uuid>

# List routes - returns JSON array
./bin/route-control route list --customer_id <uuid> [--limit 100] [--token]

# List routes by target destination - returns JSON array
./bin/route-control route list-by-target --customer_id <uuid> --target <target>

# Update route - returns updated route JSON
./bin/route-control route update --id <uuid> [--name] [--detail] [--provider] [--priority]

# Delete route - returns deleted route JSON
./bin/route-control route delete --id <uuid>

# Get effective dial routes - returns merged routes JSON
./bin/route-control route dialroute-list --customer_id <uuid> --target <target>
```

Uses same environment variables as route-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

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

## Request Routing

Handled via RabbitMQ RPC with regex routing in `pkg/listenhandler/main.go`:

**Providers API (`/v1/providers/*`):**
- `GET /v1/providers?page_size=N&page_token=T` — List providers with pagination
- `POST /v1/providers` — Create provider
- `GET /v1/providers/{id}` — Get single provider
- `PUT /v1/providers/{id}` — Update provider
- `DELETE /v1/providers/{id}` — Delete provider

**Routes API (`/v1/routes/*`):**
- `GET /v1/routes?page_size=N&page_token=T` — List routes with pagination
- `POST /v1/routes` — Create route
- `GET /v1/routes/{id}` — Get single route
- `PUT /v1/routes/{id}` — Update route
- `DELETE /v1/routes/{id}` — Delete route

**Dialroute API (`/v1/dialroutes/*`):**
- `GET /v1/dialroutes?customer_id={id}&target={target}` — Get effective routes for dialing (merges customer and default routes)

## Event Subscriptions

This service does not subscribe to external events. There is no SubscribeHandler.

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared handlers (sockhandler, requesthandler, notifyhandler, databasehandler, utilhandler)

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

## Configuration

Environment variables / flags (managed via Viper/pflag):

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
- `route_manager_receive_request_process_time` — histogram of RPC request processing time (labels: type, method)

## Key Implementation Details

### Dialroute Selection
`DialrouteGets` (in `pkg/routehandler/dialroute.go`) implements a fallback merge:
1. First retrieves customer-specific routes for the target destination.
2. Then retrieves default routes (`CustomerIDBasicRoute`) for the same target.
3. Merges results, prioritizing customer routes while adding default routes that use providers not already used by customer routes.

This allows customers to override specific providers while falling back to system defaults for others.

### Provider Model
Providers represent SIP trunks/gateways for routing outbound calls. Each provider has type (currently only `sip`), hostname, tech prefix/postfix (dial string manipulation), and tech headers (custom SIP headers). See `models/provider/provider.go`.

### Route Model
Routes map customer destinations to providers with priority ordering:
- Each route belongs to a `customer_id`
- Target can be a country code or `all` for default routing
- Priority determines route order (lower = higher priority)
- The basic route `customer_id` `00000000-0000-0000-0000-000000000001` provides system-wide defaults

See `models/route/route.go`.

### Soft Deletes
Both `providers` and `routes` tables use the `tm_delete` timestamp pattern.

## Database Schema

Tables are defined in scripts/database_scripts/:
- **table_providers.sql** - Provider configuration
- **table_routes.sql** - Route mappings

Both tables use soft-delete pattern with tm_delete timestamp.
