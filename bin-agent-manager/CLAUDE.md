# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is the `bin-agent-manager` service, part of the VoIPbin monorepo. It manages agents (call center personnel) who handle incoming/outgoing calls, including their status, permissions, authentication, and event processing.

**Key Concepts:**
- **Agent**: A person who handles calls with properties like status (available/away/busy/offline/ringing), permissions (customer-level and project-level), ring method (ringall/linear), and associated addresses/tags
- **Ring Method**: Strategy for routing calls to agents - "ringall" (all at once) or "linear" (sequential)
- **Permission System**: Bitwise permission flags organized in levels (project: super admin; customer: agent/admin/manager)

## Architecture

### Service Communication Pattern

This service uses **RabbitMQ for RPC-style communication**:
- **ListenHandler** (`pkg/listenhandler/`): Consumes RPC requests from queue `bin-manager.agent-manager.request`, processes them, and returns responses
- **SubscribeHandler** (`pkg/subscribehandler/`): Subscribes to events from other services (call-manager, customer-manager) to react to external state changes
- **NotifyHandler**: Publishes events to exchange `bin-manager.agent-manager.event` when agent state changes

### Core Components

```
cmd/agent-manager/main.go
    ├── initCache() -> pkg/cachehandler (Redis)
    ├── runServices()
        ├── pkg/dbhandler (MySQL via Squirrel query builder)
        ├── pkg/agenthandler (Business logic layer)
        ├── runServiceListen() -> pkg/listenhandler
        └── runServiceSubscribe() -> pkg/subscribehandler
```

**Layer Responsibilities:**
- `models/agent/`: Core data structures (Agent, Status, Permission, RingMethod)
- `pkg/agenthandler/`: Business logic (Create/Update/Delete agents, event handling)
- `pkg/dbhandler/`: Database operations using Squirrel SQL builder
- `pkg/cachehandler/`: Redis caching for agent lookups
- `pkg/listenhandler/`: RabbitMQ RPC request routing (REST-like paths: `/v1/agents`, `/v1/login`)
- `pkg/subscribehandler/`: Event consumption from other services

### Request Routing

ListenHandler routes requests using regex patterns matching REST-like URIs:
- `POST /v1/agents` - Create agent
- `GET /v1/agents?<filters>` - List agents (pagination via page_size/page_token)
- `GET /v1/agents/<uuid>` - Get agent
- `PUT /v1/agents/<uuid>` - Update agent basic info
- `DELETE /v1/agents/<uuid>` - Delete agent
- `PUT /v1/agents/<uuid>/status` - Update agent status
- `PUT /v1/agents/<uuid>/addresses` - Update agent SIP/WebRTC addresses
- `PUT /v1/agents/<uuid>/tag_ids` - Update agent skill tags
- `POST /v1/login` - Authenticate agent

### Event Subscriptions

SubscribeHandler processes events from:
- **call-manager**: `groupcall_created`, `groupcall_progressing` - Updates agent status when calls are routed
- **customer-manager**: `customer_created`, `customer_deleted` - Creates/deletes guest agent on customer lifecycle

### Configuration

Uses **Cobra + Viper** pattern (see `internal/config/`):
- Command-line flags and environment variables (e.g., `--rabbitmq_address` or `RABBITMQ_ADDRESS`)
- Config loaded once via `sync.Once` in `LoadGlobalConfig()`
- Required: `database_dsn`, `rabbitmq_address`, `redis_address`, `prometheus_endpoint`

## Common Commands

### Build
```bash
# From monorepo root (expects parent directory context for replacements)
cd /path/to/monorepo/bin-agent-manager
go build -o bin/agent-manager ./cmd/agent-manager
go build -o bin/agent-control ./cmd/agent-control
```

### Test
```bash
# Run all tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html
go tool cover -func=cp.out

# Run tests for specific package
go test -v ./pkg/agenthandler/...

# Run single test
go test -v ./pkg/agenthandler -run Test_Gets
```

### Generate Mocks
```bash
# Generate all mocks (uses go:generate directives)
go generate ./...

# Mocks are created via mockgen for interfaces in:
# - pkg/agenthandler/main.go -> mock_main.go
# - pkg/dbhandler/main.go -> mock_main.go
# - pkg/listenhandler/main.go -> mock_main.go
# - pkg/subscribehandler/main.go -> mock_main.go
# - pkg/cachehandler/main.go -> mock_main.go
```

### Lint
```bash
# Run golangci-lint (CI uses golangci/golangci-lint:latest image)
golangci-lint run -v --timeout 5m

# Run vet
go vet $(go list ./...)
```

### Run Locally
```bash
# With environment variables
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/voipbin" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
REDIS_DATABASE=1 \
PROMETHEUS_ENDPOINT="/metrics" \
PROMETHEUS_LISTEN_ADDRESS=":2112" \
./bin/agent-manager

# Or with flags
./bin/agent-manager \
  --database_dsn "user:pass@tcp(127.0.0.1:3306)/voipbin" \
  --rabbitmq_address "amqp://guest:guest@localhost:5672" \
  --redis_address "127.0.0.1:6379"
```

### Docker
```bash
# Build (expects monorepo root context)
docker build -f Dockerfile -t agent-manager:latest ../..

# CI builds from monorepo root with:
# docker build --tag $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA .
```

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared handlers (sockhandler, requesthandler, notifyhandler, databasehandler)
- `monorepo/bin-call-manager`: Models for groupcall events
- `monorepo/bin-customer-manager`: Models for customer events
- `monorepo/bin-registrar-manager`: Models for SIP extension

**Important**: Builds and Docker images assume parent monorepo directory context is available.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock):
- Mock interfaces are generated in same package as interface definition (e.g., `pkg/dbhandler/mock_main.go`)
- Table-driven tests with struct slices defining test cases
- Context passed to all handler methods

Example test structure:
```go
tests := []struct {
    name    string
    // inputs
    // expected results
}{
    {"normal", ...},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        mc := gomock.NewController(t)
        defer mc.Finish()
        // Setup mocks and test
    })
}
```

## Key Implementation Details

### Permission Checking
Permissions use bitwise operations. Check permissions with:
```go
agent.HasPermission(agent.PermissionCustomerAdmin)
```

### Database Queries
Use Squirrel SQL builder (not raw SQL):
```go
import sq "github.com/Masterminds/squirrel"

sq.Select("*").From("agents").
    Where(sq.Eq{"id": id}).
    Where(sq.Eq{"tm_delete": DefaultTimeStamp})
```

### Soft Deletes
Records use `tm_delete` timestamp. Default value `"9999-01-01 00:00:00.000000"` indicates active records.

### Cache Strategy
Redis cache is used for agent lookups by ID. Database is source of truth; cache updates on mutations.

### Guest Agent
A special guest agent exists with ID `d819c626-0284-4df8-99d6-d03e1c6fba88`. Created automatically when customers are created.

## CI/CD

GitLab CI pipeline (`.gitlab-ci.yml`):
1. **ensure**: `go mod download && go mod vendor`
2. **test**: `golangci-lint`, `go vet`, `go test`
3. **build**: Docker build and push to registry
4. **release**: Deploy to k8s using kustomize (manual trigger)

## Prometheus Metrics

Service exposes metrics on configured endpoint (default `:2112/metrics`):
- `receive_request_process_time` - Histogram of RPC request processing time (labels: type, method)
- `receive_subscribe_event_process_time` - Histogram of event processing time (labels: publisher, type)
