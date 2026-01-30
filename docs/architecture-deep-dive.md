# Architecture Deep Dive

> **Quick Reference:** For architecture summary, see [CLAUDE.md](../CLAUDE.md)

This document provides detailed architectural information about the VoIPbin monorepo's 30+ microservices, their communication patterns, and system design.

## Service Categories

### Core API & Gateway

- `bin-api-manager` - External REST API gateway with JWT authentication, Swagger UI at `/swagger/index.html`
- `bin-openapi-manager` - Centralized OpenAPI 3.0 specification repository, generates Go types used by all services

### Call & Media Management

- `bin-call-manager` - Inbound/outbound call routing, media control (recording, transcription, DTMF, hold, mute)
- `bin-flow-manager` - Flow execution engine (IVR workflows), manages action sequences for calls
- `bin-conference-manager` - Audio conferencing with recording, transcription, and media streaming
- `bin-transcribe-manager` - Audio transcription services (STT)
- `bin-tts-manager` - Text-to-Speech integration
- `bin-registrar-manager` - SIP registrar (UDP/TCP/WebRTC)
- `voip-asterisk-proxy` - Integration proxy for Asterisk PBX

### AI & Automation

- `bin-ai-manager` - AI chatbot integrations, summarization, task extraction
- `bin-pipecat-manager` - AI voice assistant pipeline management

### Queue & Routing

- `bin-queue-manager` - Call queueing and distribution logic
- `bin-route-manager` - Routing policies and rules
- `bin-transfer-manager` - Call transfer logic

### Customer & Agent Management

- `bin-customer-manager` - Customer accounts and relationships
- `bin-agent-manager` - Agent presence, status, permissions, addresses
- `bin-billing-manager` - Billing accounts, balance tracking, subscription management

### Campaign & Outbound

- `bin-campaign-manager` - Outbound dialing campaigns with service level tracking
- `bin-outdial-manager` - Outbound call dialer engine
- `bin-number-manager` - DID and phone number provisioning

### Messaging & Communication

- `bin-message-manager` - SMS and messaging
- `bin-email-manager` - Email sending and inbox parsing
- `bin-talk-manager` - Web chat and live chat integration
- `bin-conversation-manager` - Conversation thread management

### Infrastructure & Utilities

- `bin-common-handler` - Shared library (RabbitMQ handlers, data models, utilities)
- `bin-storage-manager` - File storage backend (integrates with GCP Cloud Storage)
- `bin-webhook-manager` - Webhook sender for customer notifications
- `bin-hook-manager` - Webhook receivers
- `bin-tag-manager` - Resource labeling and tagging
- `bin-dbscheme-manager` - Database schemas and migrations
- `bin-sentinel-manager` - Monitoring and health checks

## Inter-Service Communication

### RabbitMQ RPC Pattern

Services communicate using RabbitMQ request/response pattern, not direct HTTP:

```go
// Request format (defined in bin-common-handler/models/sock)
type Request struct {
    URI        string      // e.g., "/v1/calls"
    Method     string      // "GET", "POST", "PUT", "DELETE"
    Publisher  string      // Sending service name
    DataType   string      // "application/json"
    Data       interface{} // Request payload
}

// Response format
type Response struct {
    StatusCode int         // HTTP-style status code
    DataType   string      // "application/json"
    Data       string      // JSON response
}
```

### Queue Naming Convention

- Request queues: `bin-manager.<service-name>.request`
- Event queues: `bin-manager.<service-name>.event`
- Delayed exchange: `bin-manager.delay`

### Making Inter-Service Requests

Use `bin-common-handler/pkg/requesthandler` which provides typed methods for all services:

```go
import "monorepo/bin-common-handler/pkg/requesthandler"

// Example: Creating a call via call-manager
reqHandler := requesthandler.New(sockHandler)
call, err := reqHandler.CallV1CallCreate(context.Background(), createReq)
```

### Event Publishing

Use `bin-common-handler/pkg/notifyhandler` for publishing events:

```go
import "monorepo/bin-common-handler/pkg/notifyhandler"

notifyHandler := notifyhandler.New(sockHandler)
notifyHandler.PublishEvent(event)
```

## Configuration Management

### Configuration Precedence

Most services use **Cobra + Viper** for configuration (see `internal/config` packages). Configuration precedence:

1. Command-line flags (highest priority)
2. Environment variables
3. Default values

### Common Configuration Patterns

```bash
# Via command-line flags
./service-name \
  --database_dsn="user:pass@tcp(host:3306)/db" \
  --rabbitmq_address="amqp://guest:guest@localhost:5672" \
  --redis_address="localhost:6379" \
  --redis_database=1 \
  --prometheus_endpoint="/metrics" \
  --prometheus_listen_address=":2112"

# Via environment variables
export DATABASE_DSN="user:pass@tcp(host:3306)/db"
export RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672"
./service-name
```

### Common Configuration Fields

- Database: `--database_dsn` / `DATABASE_DSN`
- RabbitMQ: `--rabbitmq_address` / `RABBITMQ_ADDRESS`
- Redis: `--redis_address`, `--redis_password`, `--redis_database`
- Prometheus: `--prometheus_endpoint`, `--prometheus_listen_address`
- Queue names: `--rabbit_queue_listen`, `--rabbit_queue_event`

## Package Organization Pattern

### Standard Structure

Services follow a consistent structure:

```
bin-<service-name>/
├── cmd/<service-name>/     # Main entry point
│   ├── main.go            # Application initialization
│   └── init.go            # Flag/config setup (some services)
├── internal/              # Private packages
│   └── config/           # Configuration management (Cobra/Viper)
├── models/               # Data models and types
├── pkg/                  # Business logic packages
│   ├── dbhandler/       # Database and cache operations
│   ├── cachehandler/    # Redis operations
│   ├── listenhandler/   # RabbitMQ request handler
│   ├── subscribehandler/ # Event subscription handler
│   └── <domain>handler/ # Domain-specific handlers
├── gens/                # Generated code (OpenAPI, mocks)
├── openapi/            # OpenAPI specs (for api-manager)
├── k8s/                # Kubernetes manifests
├── vendor/             # Vendored dependencies
├── go.mod              # Module definition with replace directives
└── README.md
```

### Handler Pattern

Each handler follows:
1. Interface definition in `main.go` or package file
2. Implementation struct with injected dependencies
3. Mock generation via `//go:generate mockgen`
4. Tests in `*_test.go` using table-driven tests

## Database & Caching

### MySQL

**MySQL** - Shared database for persistent storage
- Query builder: `github.com/Masterminds/squirrel`
- Access via `pkg/dbhandler` abstractions in each service

### Redis

**Redis** - Distributed cache for:
- Activeflow state (flow-manager)
- Call state and temporary data
- Agent presence information
- Rate limiting and throttling

### Pattern

Always use `dbhandler` packages - they provide unified interface to both database and cache.

## Testing Patterns

### Mock Generation

```go
//go:generate mockgen -package packagename -destination ./mock_main.go -source main.go -build_flags=-mod=mod
```

### Test Structure

- Tests co-located with source: `*_test.go`
- Table-driven tests with subtests
- Mocks: `go.uber.org/mock`
- 34+ test files in flow-manager, similar counts in other services

### Running Tests

```bash
go test -v ./...
go test -v ./pkg/specifichandler/...
```

## CircleCI Continuous Integration

Path filtering enables selective service testing:
- `.circleci/config.yml` - Path filter setup
- `.circleci/config_work.yml` - Actual build jobs
- Only changed services are tested on each commit
