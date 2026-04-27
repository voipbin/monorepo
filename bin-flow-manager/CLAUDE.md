# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-flow-manager` is a Go microservice that manages call flows in a VoIP system. It orchestrates call actions (e.g., answer, play, talk, record) by coordinating with other microservices (call-manager, agent-manager, ai-manager, etc.) through RabbitMQ RPC.

**Key Concepts:**
- **Flow**: A sequence of actions that defines call behavior (e.g., answer → play message → hangup). Flows are templates stored in the database.
- **Activeflow**: A running instance of a Flow attached to a specific call/reference. Contains execution state (current action, stack, variables).
- **Action**: An atomic operation in a flow, dispatched to the appropriate manager (call-manager, ai-manager, agent-manager, etc.) by type.
- **Stack**: Activeflows use a stack structure to handle nested flows (e.g., when branching or calling sub-flows). Each stack contains its own action sequence.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-flow-manager`.

## Common Commands

### Building
```bash
# Configure git for private modules
git config --global url."https://${GL_DEPLOY_USER}:${GL_DEPLOY_TOKEN}@gitlab.com".insteadOf "https://gitlab.com"
export GOPRIVATE="gitlab.com/voipbin"

# Vendor dependencies and build
go mod download
go mod vendor
go build ./cmd/...

# Build binaries
go build -o ./bin/ ./cmd/...

# Run with default configuration
./flow-manager

# Run with custom config via environment variables
export DATABASE_DSN="user:pass@tcp(host:3306)/db"
export RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672"
./flow-manager

# Run with command-line flags
./flow-manager --database_dsn="user:pass@tcp(host:3306)/db" --rabbitmq_address="amqp://localhost:5672"

# Show all available flags
./flow-manager --help
```

### Testing
```bash
# Run all tests
go test -v $(go list ./...)

# Run tests for specific package
go test -v ./pkg/activeflowhandler/...
```

### Linting
```bash
# Run golint
golint -set_exit_status $(go list ./...)

# Run go vet
go vet $(go list ./...)

# Run golangci-lint (preferred)
golangci-lint run -v --timeout 5m
```

### Generate Mocks
```bash
# Generate all mocks (uses go:generate directives)
go generate ./...
```

## flow-control CLI Tool

A command-line tool for managing flows directly via database/cache (bypasses RabbitMQ RPC). **All output is JSON format** (stdout), logs go to stderr.

```bash
# Create flow - returns created flow JSON
./bin/flow-control flow create --customer_id <uuid> --name <name> [--type flow] [--detail] [--persist] [--actions '<json>'] [--on_complete_flow_id <uuid>]

# Get flow - returns flow JSON
./bin/flow-control flow get --id <uuid>

# List flows - returns JSON array
./bin/flow-control flow list --customer_id <uuid> [--limit 100] [--token] [--type]

# Update flow - returns updated flow JSON
./bin/flow-control flow update --id <uuid> --name <name> [--detail] [--actions '<json>'] [--on_complete_flow_id <uuid>]

# Update flow actions only - returns updated flow JSON
./bin/flow-control flow update-actions --id <uuid> --actions '<json>'

# Delete flow - returns deleted flow JSON
./bin/flow-control flow delete --id <uuid>

# Get specific action from flow - returns action JSON
./bin/flow-control flow action-get --flow_id <uuid> --action_id <uuid>
```

Uses same environment variables as flow-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

## Architecture

### Action Dispatch by Manager

Actions are executed by different managers based on type:
- `call-manager`: answer, play, talk, recording_start, dtmf_receive, hangup, etc.
- `flow-manager`: connect, goto, branch, call, patch, block, etc.
- `ai-manager`: ai_talk, ai_summary, ai_task
- `agent-manager`: agent_call
- Other managers: conference_join, queue_join, transcribe_start, etc.

### Component Structure

```
cmd/flow-manager/         # Main application entry point
internal/config/          # Configuration management (Cobra/Viper)
models/                   # Data models
  action/                 # Action types and definitions
  activeflow/             # Activeflow model and status
  flow/                   # Flow model and types
  stack/                  # Stack model for nested flows
  variable/               # Variable model for flow state
pkg/                      # Business logic packages
  activeflowhandler/      # Activeflow lifecycle and execution
  actionhandler/          # Action validation and parsing
  flowhandler/            # Flow CRUD operations
  listenhandler/          # RabbitMQ RPC request handler (REST-like endpoints)
  subscribehandler/       # Event subscription handler
  variablehandler/        # Variable substitution and management
  stackmaphandler/        # Stack management for nested flows
  dbhandler/              # Database and cache abstraction
  cachehandler/           # Redis cache operations
```

### Communication Architecture

- **RabbitMQ RPC**: Primary communication method. Other services send REST-like requests to `bin-manager.flow-manager.request` queue.
- **Event Publishing**: Activeflow events published to `bin-manager.flow-manager.event` queue.
- **Event Subscription**: Subscribes to `bin-manager.customer.event` for customer-related events.
- **Delayed Messages**: Uses `bin-manager.delay` exchange for time-delayed operations.

## Request Routing

ListenHandler routes requests using regex patterns matching REST-like URIs (see `pkg/listenhandler/`):

**Flows API (`/v1/flows/*`):**
- `POST /v1/flows` — Create flow
- `GET /v1/flows?<filters>` — List flows (pagination)
- `GET /v1/flows/<uuid>` — Get flow details
- `PUT /v1/flows/<uuid>` — Update flow
- `PUT /v1/flows/<uuid>/actions` — Update flow actions
- `DELETE /v1/flows/<uuid>` — Delete flow

**Activeflows API (`/v1/activeflows/*`):**
- `POST /v1/activeflows` — Create activeflow
- `GET /v1/activeflows?<filters>` — List activeflows
- `GET /v1/activeflows/<uuid>` — Get activeflow
- `POST /v1/activeflows/<uuid>/execute` — Execute next action
- `POST /v1/activeflows/<uuid>/stop` — Stop activeflow
- `POST /v1/activeflows/<uuid>/push` — Push actions onto stack
- `POST /v1/activeflows/<uuid>/variables` — Set activeflow variables

**Variables API (`/v1/variables/*`):**
- `GET /v1/variables/<uuid>` — Get variables for activeflow
- `POST /v1/variables/<uuid>` — Set variables

## Event Subscriptions

SubscribeHandler subscribes to:
- **bin-manager.customer-manager.event**: Customer lifecycle events (e.g., customer deletion to clean up flows).

### Handler Pattern

Each handler follows a consistent pattern:
1. Interface definition in `main.go`
2. Implementation struct with dependencies (db, cache, other handlers)
3. Mock generation via `//go:generate mockgen`
4. Tests in `*_test.go` files using mocks

Example dependency chain: `listenhandler` → `flowhandler` + `activeflowhandler` → `actionhandler` + `variablehandler` → `dbhandler`

### Database and Caching

- **Database**: MySQL (via go-sql-driver/mysql)
- **Cache**: Redis for activeflow state and temporary data
- **Query Builder**: Uses Masterminds/squirrel for SQL query construction
- DBHandler provides unified interface to both database and cache

## Key Implementation Details

### Action Execution Flow
1. Request arrives at listenhandler (e.g., `/v1/activeflows/{id}/execute`)
2. activeflowhandler retrieves activeflow from cache/db
3. Current action is determined based on stack and cursor position
4. actionhandler validates action type and options
5. variablehandler performs variable substitution on action options
6. Action is dispatched to appropriate manager via RabbitMQ
7. Activeflow state is updated and cached

### Flow Control Actions
- `goto`: Jump to specific action by ID
- `branch`: Conditional branching based on variables
- `patch`: Modify activeflow state (add/remove actions)
- `block`: Pause execution until manual continue
- `call`: Create new outgoing call with new flow

### Variable System
Variables follow format: `${variable.key}` or `${voipbin.activeflow.id}`. Built-in variables:
- `voipbin.activeflow.*`: Activeflow metadata
- `voipbin.call.*`: Call information (from call-manager)
- Custom variables stored per activeflow

### Stack Management
Stacks enable nested flow execution. When pushing actions (e.g., via `patch_flow`), a new stack is created. When a stack completes, it pops back to the parent stack.

### Variable Substitution

Variable substitution happens automatically before actions are dispatched to target managers. The variablehandler replaces all `${variable.key}` placeholders with actual values from activeflow state and call data.

## Configuration

Configuration is managed via the `internal/config` package using Cobra and Viper.

**Configuration precedence (highest to lowest):**
1. Command-line flags (e.g., `--database_dsn`)
2. Environment variables (e.g., `DATABASE_DSN`)
3. Default values

| Flag / Env | Description | Default |
|------------|-------------|---------|
| `database_dsn` / `DATABASE_DSN` | MySQL connection string | required |
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server address | required |
| `redis_address` / `REDIS_ADDRESS` | Redis server address | required |
| `redis_password` / `REDIS_PASSWORD` | Redis password | optional |
| `redis_database` / `REDIS_DATABASE` | Redis database index | optional |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

**Access config in code:**
```go
import "monorepo/bin-flow-manager/internal/config"

dbDSN := config.Get().DatabaseDSN
redisAddr := config.Get().RedisAddress
```

## Testing Patterns

- Tests use `go.uber.org/mock` for mocking dependencies
- Most tests follow the table-driven pattern with subtests
- Tests cover: CRUD operations, action execution, variable substitution, stack operations, error handling
- 34 test files total across the codebase

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

## Monorepo Context

This service is part of a larger monorepo (`monorepo/bin-*`). It depends on:
- `bin-common-handler`: Shared utilities (sockhandler, requesthandler, notifyhandler)
- Other `bin-*-manager` services: Definitions for cross-service communication

All inter-service communication happens via RabbitMQ RPC, not direct HTTP calls.

## Prometheus Metrics

Service exposes metrics on the configured endpoint (default `:2112/metrics`):
- `flow_manager_receive_request_process_time` — histogram of RPC request processing time (labels: type, method)
- `flow_manager_subscribe_event_process_time` — histogram of event processing time (labels: publisher, type)
