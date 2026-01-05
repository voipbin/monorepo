# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

flow-manager is a Go microservice that manages call flows in a VoIP system. It orchestrates call actions (e.g., answer, play, talk, record) by coordinating with other microservices (call-manager, agent-manager, ai-manager, etc.) through RabbitMQ RPC.

## Development Commands

### Building
```bash
# Configure git for private modules
git config --global url."https://${GL_DEPLOY_USER}:${GL_DEPLOY_TOKEN}@gitlab.com".insteadOf "https://gitlab.com"
export GOPRIVATE="gitlab.com/voipbin"

# Vendor dependencies and build
go mod download
go mod vendor
go build ./cmd/...
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

## Architecture

### Core Concepts

**Flow**: A sequence of actions that defines call behavior (e.g., answer → play message → hangup). Flows are templates stored in the database.

**Activeflow**: A running instance of a Flow attached to a specific call/reference. Contains execution state (current action, stack, variables).

**Action**: An atomic operation in a flow. Actions are executed by different managers based on type:
- `call-manager`: answer, play, talk, recording_start, dtmf_receive, hangup, etc.
- `flow-manager`: connect, goto, branch, call, patch, block, etc.
- `ai-manager`: ai_talk, ai_summary, ai_task
- `agent-manager`: agent_call
- Other managers: conference_join, queue_join, transcribe_start, etc.

**Stack**: Activeflows use a stack structure to handle nested flows (e.g., when branching or calling sub-flows). Each stack contains its own action sequence.

### Component Structure

```
cmd/flow-manager/         # Main application entry point
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

## Important Patterns

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

## Testing

- Tests use `go.uber.org/mock` for mocking dependencies
- Most tests follow table-driven pattern with subtests
- Tests cover: CRUD operations, action execution, variable substitution, stack operations, error handling
- 34 test files total across the codebase

## Configuration

Service runs with these primary flags (see README.md for full list):
- `-dbDSN`: MySQL connection string
- `-rabbit_addr`: RabbitMQ address
- `-redis_addr`: Redis address
- `-prom_listen_addr`: Prometheus metrics endpoint

## Monorepo Context

This service is part of a larger monorepo (`monorepo/bin-*`). It depends on:
- `bin-common-handler`: Shared utilities (sockhandler, requesthandler, notifyhandler)
- Other `bin-*-manager` services: Definitions for cross-service communication

All inter-service communication happens via RabbitMQ RPC, not direct HTTP calls.
