# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-campaign-manager` is a Go microservice that manages outbound calling campaigns within a VoIP platform. It orchestrates campaign execution, manages individual calls, and coordinates with other microservices via RabbitMQ message broker.

## Development Commands

### Build
```bash
# Build the service binary
go build -o campaign-manager ./cmd/campaign-manager

# Build with vendor dependencies
go build -mod=vendor -o campaign-manager ./cmd/campaign-manager
```

### Test
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./pkg/campaignhandler/...

# Run a specific test
go test -v -run TestCampaignHandler_Create ./pkg/campaignhandler/

# Run tests with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Code Generation
```bash
# Generate all mocks (uses go.uber.org/mock)
go generate ./...
```

### Running Locally
```bash
# Using environment variables
export DATABASE_DSN="user:password@tcp(localhost:3306)/campaign_db"
export RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672"
export REDIS_ADDRESS="localhost:6379"
export REDIS_DATABASE="1"
export PROMETHEUS_LISTEN_ADDRESS=":2112"

go run ./cmd/campaign-manager
```

## Architecture

### Core Domain Models

**Campaign** (`models/campaign/`): Outbound campaign configuration
- Manages campaign lifecycle (status: Stop/Run/Stopping)
- Two execution types: `TypeCall` (direct calls) or `TypeFlow` (execute flow first)
- Links to: Outplan (dial config), Outdial (target list), Queue (agents), Flow
- End handling: `EndHandleStop` (stop when done) or `EndHandleContinue` (continue to next campaign)

**Campaigncall** (`models/campaigncall/`): Individual call instance within a campaign
- Tracks a single call attempt as part of a campaign
- References either a Call (in call-manager) or Activeflow (in flow-manager)
- Supports up to 5 destination addresses with independent retry counts
- Status: Dialing → Progressing → Done (with result: Success/Fail)

**Outplan** (`models/outplan/`): Dialing configuration/plan
- Defines dial timeout, retry intervals, max try counts
- Shared across multiple campaigns
- Contains source address (caller ID) configuration

### Handler Architecture

All business logic is organized into handler packages with interface-first design:

**CampaignHandler** (`pkg/campaignhandler/`): Campaign lifecycle and execution
- CRUD operations for campaigns
- Status management (Run/Stopping/Stop transitions)
- **Execute() method**: Main campaign execution loop that:
  - Fetches available targets from outdial-manager
  - Checks queue capacity via service_level percentage
  - Creates campaigncalls and either calls (TypeCall) or flows (TypeFlow)
  - Recursively schedules next execution with 500ms delay

**CampaigncallHandler** (`pkg/campaigncallhandler/`): Individual call tracking
- Manages campaigncall lifecycle and status transitions
- Handles reference lookups (by call ID or activeflow ID)
- Processes Done transitions with Success/Fail results

**OutplanHandler** (`pkg/outplanhandler/`): Dialing configuration management

**DBHandler** (`pkg/dbhandler/`): Database abstraction layer
- All MySQL CRUD operations for campaigns, campaigncalls, outplans
- Uses soft deletes (tm_delete timestamp)
- Integrates CacheHandler for Redis caching

**ListenHandler** (`pkg/listenhandler/`): RabbitMQ RPC server
- Processes incoming API requests from "campaign_request" queue
- Routes to appropriate handler based on URI pattern and HTTP method
- Endpoints: `/v1/campaigns`, `/v1/campaigncalls`, `/v1/outplans`

**SubscribeHandler** (`pkg/subscribehandler/`): Event subscriber
- Listens to events from call-manager and flow-manager
- Processes: `call_hungup` and `activeflow_deleted` events
- Updates campaigncall status and triggers campaign state changes

### Event-Driven Communication

**Published Events** (to "campaign_event" queue):
- Campaign: campaign_created, campaign_updated, campaign_deleted, campaign_status_run, campaign_status_stopping, campaign_status_stop
- Campaigncall: campaigncall_created, campaigncall_updated, campaigncall_deleted
- Outplan: outplan_created, outplan_updated, outplan_deleted

**Subscribed Events** (from "campaign_subscribe" queue):
- call-manager: `call_hungup` → marks campaigncall as done
- flow-manager: `activeflow_deleted` → marks campaigncall as done

### Service Dependencies

**Monorepo Services** (via requesthandler RPC):
- `bin-call-manager`: Creates calls, queries call status
- `bin-flow-manager`: Creates flows and activeflows, executes flows
- `bin-outdial-manager`: Manages target lists, provides available targets, updates target status
- `bin-queue-manager`: Provides queue info and available agent counts
- `bin-agent-manager`: Agent availability queries

**External Dependencies**:
- MySQL: Stores campaigns, campaigncalls, outplans (tables: `campaign_campaigns`, `campaign_campaigncalls`, `campaign_outplans`)
- Redis: Caches frequently accessed objects
- RabbitMQ: Message broker for RPC and pub-sub patterns

### Campaign Execution Flow

```
1. Campaign created with OutplanID, OutdialID, optional QueueID
2. Campaign status updated to "Run" → triggers Execute()
3. Execute() loop:
   a. Get next available target from outdial-manager
   b. Check queue capacity (service_level constraint)
   c. Create campaigncall
   d. If TypeCall: create call via call-manager
      If TypeFlow: create activeflow via flow-manager
   e. Wait for external event (call_hungup or activeflow_deleted)
   f. Reschedule Execute() after 500ms delay
4. External event received → campaigncall marked as Done
5. If no more targets or manual stop → Campaign transitions to Stop
```

### Key Design Patterns

**Interface-Based Design**: All handlers are interfaces with mock implementations for testing (generated via `go generate`)

**Soft Deletes**: All models use `tm_delete` timestamp (default: "9999-01-01 00:00:000"). Deleted records are marked with actual deletion time.

**Service Level Queuing**: Campaigns can limit concurrent dialing based on queue agent availability:
```
agent_capacity = (available_agents × service_level) / 100
dialing_allowed = current_dialing_count < agent_capacity
```

**Recursive Execution**: Campaign execute uses goroutine-based recursive scheduling rather than persistent job queue

**Multi-Destination Retry Logic**: Each campaigncall supports 5 destination addresses with independent retry tracking (destination_0 through destination_4, try_count_0 through try_count_4)

## Configuration

Configuration via Viper (command-line flags or environment variables):

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--rabbitmq_address` | `RABBITMQ_ADDRESS` | `amqp://guest:guest@localhost:5672` | RabbitMQ connection string |
| `--database_dsn` | `DATABASE_DSN` | `testid:testpassword@tcp(127.0.0.1:3306)/test` | MySQL DSN |
| `--redis_address` | `REDIS_ADDRESS` | `127.0.0.1:6379` | Redis server address |
| `--redis_password` | `REDIS_PASSWORD` | (empty) | Redis authentication password |
| `--redis_database` | `REDIS_DATABASE` | `1` | Redis database index |
| `--prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `:2112` | Prometheus metrics endpoint |
| `--prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | `/metrics` | Metrics URL path |

## Monorepo Context

This service is part of a larger VoIP platform monorepo at `/home/pchero/gitvoipbin/monorepo/`. Dependencies on other `bin-*-manager` services are replaced via `go.mod` replace directives pointing to sibling directories.

When making changes that affect other services, coordinate changes across:
- `bin-call-manager`: Call creation and management
- `bin-flow-manager`: Flow execution logic
- `bin-outdial-manager`: Target list management
- `bin-queue-manager`: Queue and agent tracking
- `bin-common-handler`: Shared models and utilities

## Testing Considerations

- All handlers have interface definitions for mockability
- Generate mocks via `go generate ./...` before running tests
- Tests should not require external services (use mocks)
- Database tests use in-memory SQLite when possible (see `dbhandler/main_test.go`)
- Event handling tests verify proper event routing and handler invocation

## Important Files

- `cmd/campaign-manager/init.go`: Configuration initialization and Prometheus setup
- `cmd/campaign-manager/main.go`: Service entry point and handler orchestration
- `pkg/campaignhandler/execute.go`: Core campaign execution logic
- `pkg/listenhandler/campaigns.go`: Campaign API endpoint handlers
- `pkg/subscribehandler/callmanager.go`: Call event processing
- `pkg/subscribehandler/flowmanager.go`: Flow event processing
