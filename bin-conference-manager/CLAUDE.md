# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-conference-manager` is a service in the VoIPbin monorepo that manages conference sessions and conference participants (conferencecalls). It coordinates with `bin-call-manager` to create and manage conference bridges (confbridges), handles conference lifecycle operations including recording and transcription, and tracks participant join/leave events.

**Key Concepts:**
- **Conference**: A conference session that coordinates multiple participants, with properties including type (conference/connect/queue), status lifecycle, recording/transcribe state, and pre/post flow execution
- **Conferencecall**: A participant in a conference - typically references a call from call-manager that has joined the conference
- **Conference Types**:
  - `conference`: Standard multi-party conference (3+ participants)
  - `connect`: Auto-terminates when only 1 participant remains
  - `queue`: Special conference type for queue operations
- **Confbridge**: The underlying Asterisk conference bridge managed by call-manager

## Architecture

### Service Communication Pattern

This service uses **RabbitMQ for RPC-style communication**:
- **ListenHandler** (`pkg/listenhandler/`): Consumes RPC requests from queue `bin-manager.conference-manager.request`, processes them, and returns responses
- **SubscribeHandler** (`pkg/subscribehandler/`): Subscribes to events from call-manager (confbridge join/leave events)
- **NotifyHandler**: Publishes conference lifecycle events to exchange `bin-manager.conference-manager.event`

### Core Components

```
cmd/conference-manager/main.go
    ├── Database (MySQL)
    ├── Cache (Redis via pkg/cachehandler)
    └── run()
        ├── pkg/dbhandler (MySQL operations via Squirrel)
        ├── pkg/conferencehandler (Conference business logic)
        ├── pkg/conferencecallhandler (Conference participant management)
        ├── runListen() -> pkg/listenhandler
        └── runSubscribe() -> pkg/subscribehandler
```

**Layer Responsibilities:**
- `models/conference/`: Conference data structures (Conference, Type, Status, Event, Webhook)
- `models/conferencecall/`: Conference participant structures (Conferencecall, ReferenceType, Status, Event, Webhook)
- `pkg/conferencehandler/`: Conference business logic (create/update/delete conferences, recording, transcription, termination)
- `pkg/conferencecallhandler/`: Participant management (create/join/terminate conferencecalls, health checks)
- `pkg/dbhandler/`: Database operations using Squirrel SQL builder
- `pkg/cachehandler/`: Redis caching for conference lookups
- `pkg/listenhandler/`: RabbitMQ RPC request routing (REST-like paths)
- `pkg/subscribehandler/`: Event consumption from call-manager

### Request Routing

ListenHandler routes requests using regex patterns matching REST-like URIs:

**Conferences API (`/v1/conferences/*`)**:
- `POST /v1/conferences` - Create conference
- `GET /v1/conferences?<filters>` - List conferences (pagination via page_size/page_token)
- `GET /v1/conferences/<uuid>` - Get conference details
- `PUT /v1/conferences/<uuid>` - Update conference basic info
- `DELETE /v1/conferences/<uuid>` - Delete conference
- `PUT /v1/conferences/<uuid>/recording_id` - Update recording ID
- `POST /v1/conferences/<uuid>/recording_start` - Start conference recording
- `POST /v1/conferences/<uuid>/recording_stop` - Stop conference recording
- `POST /v1/conferences/<uuid>/stop` - Stop/terminate conference
- `POST /v1/conferences/<uuid>/transcribe_start` - Start conference transcription
- `POST /v1/conferences/<uuid>/transcribe_stop` - Stop conference transcription

**Conferencecalls API (`/v1/conferencecalls/*`)**:
- `GET /v1/conferencecalls?<filters>` - List conference participants
- `GET /v1/conferencecalls/<uuid>` - Get participant details
- `DELETE /v1/conferencecalls/<uuid>` - Remove participant from conference
- `POST /v1/conferencecalls/<uuid>/health-check` - Health check for participant

**Services API**:
- `POST /v1/services/type/conferencecall` - Create conferencecall service (used by flow-manager)

### Event Subscriptions

SubscribeHandler subscribes to these RabbitMQ queues:
- **bin-manager.call-manager.event**: Conference bridge join/leave events

Processes events including:
- **confbridge_joined**: When a call joins a confbridge, updates conferencecall status to `joined`
- **confbridge_leaved**: When a call leaves a confbridge, terminates the conferencecall

### Conference-Call Relationship

The service manages a two-layer architecture:
1. **Conference layer** (`pkg/conferencehandler`): High-level conference coordination
   - Creates underlying confbridge via call-manager RPC
   - Tracks conference metadata (name, detail, data, timeout)
   - Manages conference-level recording and transcription
   - Executes pre/post flows

2. **Conferencecall layer** (`pkg/conferencecallhandler`): Individual participants
   - Each conferencecall references a call from call-manager
   - Tracks participant status: `joining` → `joined` → `leaving` → `leaved`
   - Performs health checks on participants
   - Auto-terminates based on conference type rules

### Configuration

Uses **Viper + pflag** pattern (see `cmd/conference-manager/init.go`):
- Command-line flags and environment variables (e.g., `--rabbitmq_address` or `RABBITMQ_ADDRESS`)
- Configuration parameters:
  - `database_dsn`: MySQL connection string (default: `testid:testpassword@tcp(127.0.0.1:3306)/test`)
  - `rabbitmq_address`: RabbitMQ server address (default: `amqp://guest:guest@localhost:5672`)
  - `redis_address`, `redis_password`, `redis_database`: Redis connection (default: `127.0.0.1:6379`, `""`, `1`)
  - `prometheus_endpoint`, `prometheus_listen_address`: Metrics endpoint (default: `/metrics`, `:2112`)

## Common Commands

### Build
```bash
# Build from service directory
go build -o bin/conference-manager ./cmd/conference-manager

# Build with vendor (requires monorepo context)
go mod vendor
go build ./cmd/...

# Build using Docker (from monorepo root)
docker build -t conference-manager:latest -f bin-conference-manager/Dockerfile .
```

### Test
```bash
# Run all tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html
go tool cover -func=cp.out

# Run tests for specific package
go test -v ./pkg/conferencehandler/...

# Run single test
go test -v ./pkg/conferencehandler -run Test_Create
```

### Generate Mocks
```bash
# Generate all mocks (uses go:generate directives)
go generate ./...

# Mocks are created via mockgen for interfaces in:
# - pkg/conferencehandler/main.go -> mock_main.go
# - pkg/conferencecallhandler/main.go -> mock_main.go
# - pkg/dbhandler/main.go -> mock_dbhandler_dbhandler.go
# - pkg/cachehandler/main.go -> mock_cachehandler.go
# - pkg/subscribehandler/main.go -> mock_main.go
```

### Lint
```bash
# Run vet
go vet $(go list ./...)

# Run golangci-lint (if available)
golangci-lint run -v --timeout 5m
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
./bin/conference-manager

# Or with flags
./bin/conference-manager \
  --database_dsn "user:pass@tcp(127.0.0.1:3306)/voipbin" \
  --rabbitmq_address "amqp://guest:guest@localhost:5672" \
  --redis_address "127.0.0.1:6379" \
  --redis_database 1
```

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared handlers (sockhandler, requesthandler, notifyhandler, databasehandler, utilhandler)
- `monorepo/bin-call-manager`: Confbridge models and RPC interfaces
- `monorepo/bin-flow-manager`: Flow execution integration
- `monorepo/bin-transcribe-manager`: Transcription service integration

**Important**: Builds and Docker images assume parent monorepo directory context is available. The service uses Squirrel SQL builder for database queries (unlike call-manager which uses raw SQL).

## Testing Patterns

Tests use **gomock** (go.uber.org/mock):
- Mock interfaces are generated in same package as interface definition
- Table-driven tests with struct slices defining test cases
- Context passed to all handler methods
- Tests co-located with implementation: `pkg/<package>/<feature>_test.go`

Example test structure:
```go
tests := []struct {
    name    string
    id      uuid.UUID
    // expected results
}{
    {"normal", uuid.FromStringOrNil("..."), ...},
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

### Conference Status Flow
Conferences progress through statuses:
- `starting`: Conference is being initialized
- `progressing`: Conference is active with participants
- `terminating`: Conference is shutting down
- `terminated`: Conference has ended

### Conferencecall Status Flow
Conference participants progress through statuses:
- `joining`: Participant is being added to conference
- `joined`: Participant successfully joined
- `leaving`: Participant is being removed
- `leaved`: Participant has left

### Conference Type Behavior
- **TypeConference**: Standard conference, remains active with any number of participants
- **TypeConnect**: Auto-terminates when only 1 participant remains (useful for 1:1 bridged calls)
- **TypeQueue**: Special queue-type conference behavior

### Database Queries
Use Squirrel SQL builder (not raw SQL):
```go
import sq "github.com/Masterminds/squirrel"

sq.Select("*").From("conferences").
    Where(sq.Eq{"id": id}).
    Where(sq.Eq{"tm_delete": defaultTimeStamp})
```

### Soft Deletes
Records use `tm_delete` timestamp. Default value `"9999-01-01 00:00:00.000000"` indicates active records.

### Cache Strategy
Redis cache is used for conference lookups by ID. Database is source of truth; cache updates on mutations.

### Recording and Transcription
- Conferences can have multiple recordings (`recording_ids[]`) but track primary `recording_id`
- Similar pattern for transcription: `transcribe_id` and `transcribe_ids[]`
- Recording/transcription operations delegate to call-manager confbridge APIs

### Health Checks
Conferencecalls perform periodic health checks on referenced calls:
- Default delay: 5 seconds
- Default max retries: 2
- Max conferencecall duration: 24 hours (auto-cleanup)

### Flow Integration
Conferences support pre-flow and post-flow execution:
- `pre_flow_id`: Flow executed before conference starts
- `post_flow_id`: Flow executed after conference terminates

## Prometheus Metrics

Service exposes metrics on configured endpoint (default `:2112/metrics`):
- `conference_manager_receive_request_process_time` - Histogram of RPC request processing time (labels: type, method)
- `conference_manager_conferencecall_total` - Counter of conferencecalls by reference_type and status

## Handler Dependency Chain

Understanding the handler initialization order (see `cmd/conference-manager/main.go:76-105`):

```
dbhandler (depends on: sqlDB, cache)
  ├── conferenceHandler (depends on: requestHandler, notifyHandler, db)
  └── conferencecallHandler (depends on: requestHandler, notifyHandler, db, conferenceHandler)
      ├── listenHandler (depends on: sockHandler, conferenceHandler, conferencecallHandler)
      └── subscribeHandler (depends on: sockHandler, conferenceHandler, conferencecallHandler)
```

When modifying handlers, respect this dependency order to avoid circular dependencies.
