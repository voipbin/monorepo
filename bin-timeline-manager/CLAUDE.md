# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-timeline-manager` is a Go service within the VoIPbin monorepo that manages timeline events stored in ClickHouse. It provides event querying capabilities for tracking the history of resources across the platform (calls, flows, conversations, etc.).

This service operates as an event-driven microservice using RabbitMQ for RPC-style communication.

## Architecture

### Service Communication Pattern

This service uses **RabbitMQ for RPC-style communication**:
- **ListenHandler** (`pkg/listenhandler/`): Consumes RPC requests from queue `bin-manager.timeline-manager.request`, processes them, and returns responses

### Core Components

```
cmd/timeline-manager/main.go
    ├── ClickHouse connection (pkg/dbhandler)
    ├── RabbitMQ connection (sockhandler)
    ├── runServiceListen() -> pkg/listenhandler
    └── Prometheus metrics endpoint
```

**Layer Responsibilities:**
- `models/event/`: Domain Event struct (database-compatible types)
- `pkg/listenhandler/models/request/`: API request DTOs (uses ServiceName type)
- `pkg/listenhandler/models/response/`: API response DTOs
- `pkg/eventhandler/`: Business logic for event queries with cursor-based pagination
- `pkg/dbhandler/`: ClickHouse database operations
- `pkg/listenhandler/`: RabbitMQ RPC request routing

### Request Routing

ListenHandler routes requests using regex patterns:
- `POST /v1/events` - List events with filters (publisher, resource ID, event patterns)

### Configuration

Uses **Cobra + Viper** pattern (see `internal/config/`):
- Command-line flags and environment variables
- Required: `rabbitmq_address`, `clickhouse_address`
- Optional: `clickhouse_database` (default: `default`), `prometheus_endpoint`, `prometheus_listen_address`, `migrations_path`

Default values:
- `clickhouse_database`: `default`
- `migrations_path`: `./migrations`

Production values (set in k8s deployment):
- `clickhouse_address`: `clickhouse.infrastructure:9000`

## Development Commands

### Build
```bash
# Build the service binary
go build -o ./bin/timeline-manager ./cmd/timeline-manager/

# Build the CLI tool
go build -o ./bin/timeline-control ./cmd/timeline-control/
```

### Testing
```bash
# Run all tests with coverage
go test -coverprofile cp.out -v $(go list ./...)

# View coverage report
go tool cover -html=cp.out -o cp.html
go tool cover -func=cp.out

# Run tests for a specific package
go test -v ./pkg/eventhandler/...
```

### Code Quality
```bash
# Run golangci-lint
golangci-lint run -v --timeout 5m

# Run go vet
go vet $(go list ./...)
```

### Mock Generation
```bash
# Generate mocks using go:generate directives
go generate ./...

# Mocks are created for interfaces in:
# - pkg/dbhandler/main.go -> mock_main.go
# - pkg/eventhandler/main.go -> mock_main.go
# - pkg/listenhandler/main.go -> mock_main.go
```

## timeline-control CLI Tool

A command-line tool for timeline-manager operations. **All output is JSON format** (stdout), logs go to stderr.

```bash
# Run database migrations
./bin/timeline-control migrate up      # Apply all pending migrations
./bin/timeline-control migrate down    # Roll back last migration
./bin/timeline-control migrate version # Show current migration version

# Display version information
./bin/timeline-control version

# Check health status
./bin/timeline-control health
```

Uses same environment variables as timeline-manager (`CLICKHOUSE_ADDRESS`, `CLICKHOUSE_DATABASE`, `MIGRATIONS_PATH`, etc.).

### Run Locally
```bash
# With environment variables
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
CLICKHOUSE_ADDRESS="127.0.0.1:9000" \
CLICKHOUSE_DATABASE="default" \
PROMETHEUS_ENDPOINT="/metrics" \
PROMETHEUS_LISTEN_ADDRESS=":2112" \
./bin/timeline-manager

# Or with flags
./bin/timeline-manager \
  --rabbitmq_address "amqp://guest:guest@localhost:5672" \
  --clickhouse_address "127.0.0.1:9000" \
  --clickhouse_database "default"
```

## Database Schema

The service uses ClickHouse for event storage. Schema is managed via golang-migrate.

**Events Table:**
```sql
CREATE TABLE events (
    timestamp DateTime64(3),
    publisher String,
    type String,
    resource_id UUID,
    data String
) ENGINE = MergeTree()
ORDER BY (publisher, resource_id, timestamp)
```

**Migrations:**
- Located in `migrations/` directory
- Run with `timeline-control migrate up`
- Format: `NNNNNN_description.up.sql` and `NNNNNN_description.down.sql`

## Key Data Models

### Model Organization Pattern

This service follows a layered model pattern (similar to flow-manager):

- **Domain models** (`models/event/`): Database-compatible types using basic Go types
- **API DTOs** (`pkg/listenhandler/models/`): Request/response types with richer types like `ServiceName`
- **Type conversion**: Happens in `pkg/eventhandler/` at the boundary between API and database layers

### Event (`models/event/event.go`)
Domain model representing a timeline event (uses `string` for ClickHouse compatibility):
- **Timestamp**: When the event occurred (`time.Time`)
- **Publisher**: Service name as `string` (e.g., `"flow-manager"`, `"call-manager"`)
- **EventType**: Type of event (e.g., `"activeflow_created"`, `"call_hangup"`)
- **DataType**: MIME type of the data payload
- **Data**: JSON payload with event details (`json.RawMessage`)

### V1DataEventsPost Request (`pkg/listenhandler/models/request/event.go`)
API request DTO for listing events:
- **Publisher**: Required - `ServiceName` type from common-handler
- **ID**: Required - filter by resource ID (`uuid.UUID`)
- **Events**: Required - list of event type patterns (supports wildcards like `activeflow_*`)
- **PageSize**: Optional - number of results per page (default: 100, max: 1000)
- **PageToken**: Optional - cursor for pagination

### V1DataEventsPost Response (`pkg/listenhandler/models/response/event.go`)
API response DTO:
- **Result**: List of Event pointers
- **NextPageToken**: Cursor for next page (omitted if no more results)

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared handlers (sockhandler, requesthandler, notifyhandler)

**Important**: Builds and Docker images assume parent monorepo directory context is available.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock):
- Mock interfaces are generated in same package as interface definition
- Table-driven tests with struct slices defining test cases
- Context passed to all handler methods

Example test structure:
```go
tests := []struct {
    name    string
    req     *request.V1DataEventsPost
    wantErr bool
}{
    {"valid request", validReq, false},
    {"missing publisher", invalidReq, true},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        mc := gomock.NewController(t)
        defer mc.Finish()
        // Setup mocks and test
    })
}
```

## Prometheus Metrics

Service exposes metrics on configured endpoint (default `:2112/metrics`):
- `receive_request_process_time` - Histogram of RPC request processing time (labels: type, method)

## Common Gotchas

### ClickHouse Type Conversion

**CRITICAL: ClickHouse driver cannot convert String columns to custom Go types.**

The ClickHouse Go driver only supports basic Go types for column scanning. Custom types like `ServiceName` will cause runtime errors:

```
converting String to *outline.ServiceName is unsupported
```

**Solution:** Use basic Go types (`string`, `int`, `time.Time`, etc.) in domain models that are scanned from ClickHouse. Convert to/from richer types at the handler layer boundary.

```go
// ✅ CORRECT - Domain model uses string for ClickHouse compatibility
type Event struct {
    Publisher string `json:"publisher"`  // string for ClickHouse
}

// ✅ CORRECT - API DTO uses ServiceName for type safety
type V1DataEventsPost struct {
    Publisher commonoutline.ServiceName `json:"publisher"`
}

// ✅ CORRECT - Convert at handler boundary
events, err := h.db.EventList(ctx, string(req.Publisher), ...)
```

```go
// ❌ WRONG - Custom type in domain model
type Event struct {
    Publisher commonoutline.ServiceName `json:"publisher"`  // Will fail!
}
```

## Kubernetes Deployment

Deployment files in `k8s/`:
- `deployment.yml`: 2 replicas, 30m CPU, 30Mi memory limits
- `service.yml`: ClusterIP service
- `kustomization.yml`: Kustomize configuration

Environment variables in deployment:
- `CLICKHOUSE_ADDRESS`: `clickhouse.infrastructure:9000`
- `CLICKHOUSE_DATABASE`: `default`
- `RABBITMQ_ADDRESS`: Set via kustomize substitution
