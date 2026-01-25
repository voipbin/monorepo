# CLAUDE.md Service Template

> **Purpose:** Standardized template for service-specific CLAUDE.md files. Copy this template when creating documentation for a new service.

## Required Sections

Every service CLAUDE.md should include these 10 sections:

1. Overview
2. Architecture
3. Request Routing (API endpoints)
4. Event Subscriptions
5. Common Commands
6. Monorepo Context
7. Testing Patterns
8. Key Implementation Details
9. Configuration
10. Prometheus Metrics

---

## Template

Copy everything below this line into your service's `CLAUDE.md`:

---

```markdown
# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-<service>-manager` is [brief description of service purpose]. It [main responsibilities].

**Key Concepts:**
- **Concept1**: Brief explanation
- **Concept2**: Brief explanation
- **Concept3**: Brief explanation

## Architecture

### Service Communication Pattern

This service uses **RabbitMQ for event-driven RPC communication**:
- **ListenHandler** (`pkg/listenhandler/`): Consumes RPC requests from queue `bin-manager.<service>-manager.request`
- **SubscribeHandler** (`pkg/subscribehandler/`): Subscribes to events from other services
- **NotifyHandler**: Publishes events to `bin-manager.<service>-manager.event` exchange

### Core Components

```
cmd/<service>-manager/main.go
    ├── Database (MySQL)
    ├── Cache (Redis via pkg/cachehandler) [if applicable]
    └── run()
        ├── pkg/dbhandler (MySQL operations)
        ├── Domain Handlers:
        │   ├── pkg/<domain>handler (Business logic)
        │   └── [additional handlers]
        ├── runSubscribe() -> pkg/subscribehandler
        └── runRequestListen() -> pkg/listenhandler
```

**Layer Responsibilities:**
- `models/`: Data structures
- `pkg/*handler/`: Domain-specific business logic
- `pkg/dbhandler/`: Database operations (Squirrel query builder)
- `pkg/listenhandler/`: RabbitMQ RPC request routing
- `pkg/subscribehandler/`: Event consumption [if applicable]

## Request Routing

ListenHandler routes requests using regex patterns matching REST-like URIs:

**Resource API (`/v1/<resources>/*`)**:
- `POST /v1/<resources>` - Create resource
- `GET /v1/<resources>?<filters>` - List resources (pagination)
- `GET /v1/<resources>/<uuid>` - Get resource details
- `POST /v1/<resources>/<uuid>` - Update resource
- `DELETE /v1/<resources>/<uuid>` - Delete resource

[Add additional endpoints as needed]

## Event Subscriptions

SubscribeHandler subscribes to these RabbitMQ queues:
- **bin-manager.<other>-manager.event**: [What events and why]

Processes events including:
- **EventType1**: What it does
- **EventType2**: What it does

[If no subscriptions, write: "This service does not subscribe to external events."]

## Common Commands

### Build
```bash
# Build from service directory
go build -o bin/<service>-manager ./cmd/<service>-manager

# Build with vendor
export GOPRIVATE="gitlab.com/voipbin"
go mod vendor
go build ./cmd/...
```

### Test
```bash
# Run all tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -func=cp.out

# Run tests for specific package
go test -v ./pkg/<domain>handler/...
```

### Generate Mocks
```bash
go generate ./...
```

### Lint
```bash
golangci-lint run -v --timeout 5m
```

### Run Locally
```bash
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/bin_manager" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
[additional env vars] \
./bin/<service>-manager
```

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared handlers (sockhandler, requesthandler, notifyhandler)
- [List other dependencies]

**Important**: Always run `go mod vendor` after changing dependencies.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock):
- Mock interfaces generated in same package as interface definition
- Table-driven tests with struct slices
- Context passed to all handler methods
- Tests co-located with implementation: `<package>/<feature>_test.go`

Example test structure:
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

## Key Implementation Details

### Important Pattern 1
Brief explanation of key implementation detail.

### Important Pattern 2
Brief explanation of another key detail.

### Database Operations
- Uses Squirrel query builder
- Soft deletes with `tm_delete` timestamp (`"9999-01-01 00:00:00.000000"` for active records)
- UUID fields require `,uuid` db tag

[Add 3-5 key implementation details specific to this service]

## Configuration

Uses **Viper + pflag** pattern (see `cmd/<service>-manager/init.go`):

| Flag/Env | Description | Default |
|----------|-------------|---------|
| `database_dsn` / `DATABASE_DSN` | MySQL connection string | required |
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server | required |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |
| [service-specific configs] | | |

## Prometheus Metrics

Service exposes metrics on configured endpoint (default `:2112/metrics`):
- `<service>_manager_receive_request_process_time` - Histogram of RPC request processing time
- `<service>_manager_subscribe_event_process_time` - Histogram of event processing time [if applicable]
- [service-specific metrics]
```

---

## Section Guidelines

### Overview
- 2-3 sentences describing what the service does
- List 3-5 key concepts/entities the service manages
- Keep it high-level

### Architecture
- Show the component hierarchy
- Explain layer responsibilities
- Include ASCII diagram of component structure

### Request Routing
- List ALL endpoints handled by ListenHandler
- Group by resource type
- Include HTTP method and URI pattern

### Event Subscriptions
- List queues the service subscribes to
- Explain what events are processed
- Say "none" if service doesn't subscribe

### Common Commands
- Build, test, generate, lint commands
- Local run command with environment variables
- Keep it copy-paste ready

### Monorepo Context
- List dependencies from go.mod replace directives
- Highlight shared handler usage

### Testing Patterns
- Show example test structure
- Mention mock generation
- Reference table-driven test pattern

### Key Implementation Details
- 3-5 service-specific patterns
- Database patterns if unique
- Cache strategies
- Business logic rules

### Configuration
- Table of all config flags
- Include environment variable names
- Mark required vs optional

### Prometheus Metrics
- List all exposed metrics
- Include metric type (counter, histogram, gauge)
- Describe labels

## See Also

- [Code Quality Standards](code-quality-standards.md) - Logging, naming conventions
- [Common Workflows](common-workflows.md) - Adding endpoints, handlers
- [RabbitMQ Queues Reference](rabbitmq-queues-reference.md) - Queue naming
- [Error Handling Patterns](error-handling-patterns.md) - Error responses
