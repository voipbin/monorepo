# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-api-manager` is a RESTful API gateway for the VoIPBIN platform that handles authentication, routing, and API requests to various backend managers. It's part of a Go monorepo architecture and serves as the public-facing API for the entire VoIPBIN system.

## Build & Development Commands

### Prerequisites
The project requires ZMQ libraries for messaging:
```bash
apt update && apt install -y pkg-config libzmq5 libzmq3-dev libczmq4 libczmq-dev
```

### Build
```bash
# Set up private Go modules access (if needed)
git config --global url."https://<GL_DEPLOY_USER>:<GL_DEPLOY_TOKEN>@gitlab.com".insteadOf "https://gitlab.com"
export GOPRIVATE="gitlab.com/voipbin"

# Download dependencies and build
go mod download
go mod vendor
go build ./cmd/...
```

### Testing
```bash
# Run all tests with coverage
go test -v $(go list ./...)

# Run tests with coverage report
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html
go tool cover -func=cp.out

# Run specific package tests
go test -v ./pkg/servicehandler/...
```

### Linting
```bash
# Run go vet
go vet $(go list ./...)

# Run golangci-lint (requires golangci-lint installed)
golangci-lint run -v --timeout 5m
```

### API Documentation

#### Swagger/OpenAPI Generation
```bash
# Install swag if not already installed
go install github.com/swaggo/swag/cmd/swag@latest

# Format and generate Swagger docs
swag fmt
swag init --parseDependency --parseInternal -g cmd/api-manager/main.go -o docsapi
```

OpenAPI specs are generated using `oapi-codegen` from specs in `openapi/config_server/`.

Access Swagger UI at: `https://api.voipbin.net/swagger/index.html`

### API Schema Validation

**IMPORTANT: This service relies on `bin-openapi-manager` as the source of truth for API contracts.**

When backend services (bin-call-manager, bin-conference-manager, etc.) expose data through this API gateway, the OpenAPI schemas in `bin-openapi-manager` must accurately reflect what those services actually return.

**Validation Workflow:**

When adding or modifying API endpoints:
1. **Identify the Backend Service Model:**
   - Determine which service handles the request (e.g., bin-call-manager for calls)
   - Find the public-facing struct (usually `WebhookMessage` in `models/<resource>/webhook.go`)

2. **Verify OpenAPI Schema:**
   - Check that `bin-openapi-manager/openapi/openapi.yaml` has matching schema
   - Ensure all fields in the Go struct appear in the OpenAPI schema
   - Verify types match (string, array, enums, etc.)

3. **Update if Needed:**
   - If schema is missing or outdated, update `bin-openapi-manager/openapi/openapi.yaml`
   - Regenerate models: `cd ../bin-openapi-manager && go generate ./...`
   - Update this service: `go mod tidy && go mod vendor && go generate ./...`

**Example:**
```
Backend Service:
  bin-call-manager/models/call/webhook.go → WebhookMessage struct (23 fields)

OpenAPI Schema:
  bin-openapi-manager/openapi/openapi.yaml → CallManagerCall (must have same 23 fields)

API Response:
  This service returns the schema defined in bin-openapi-manager
```

**Common Issues:**
- Backend service adds new field → OpenAPI schema not updated → API docs outdated
- Enum values added → OpenAPI enum not updated → Validation errors
- Field renamed → Breaking change → Must coordinate across all services

See `bin-openapi-manager/CLAUDE.md` for detailed schema validation procedures.

## Architecture

### Monorepo Structure
This service is part of a monorepo with local module replacements. The `go.mod` contains `replace` directives for ~20 sibling managers:
- `bin-call-manager`: Call management and routing
- `bin-flow-manager`: Call flow and IVR logic
- `bin-ai-manager`: AI/ML features (STT, TTS, summarization)
- `bin-storage-manager`: File and media storage
- `bin-webhook-manager`: Webhook dispatching
- And many others (agent, billing, campaign, chat, conference, conversation, etc.)

When modifying dependencies, ensure changes are compatible across the monorepo.

### Core Components

#### Main Application Flow (`cmd/api-manager/main.go`)
1. **Initialization**: Connects to MySQL database, Redis cache, and RabbitMQ
2. **Handler Setup**: Creates instances of:
   - `sockHandler`: RabbitMQ communication
   - `dbhandler`: Database and cache operations
   - `serviceHandler`: Main business logic orchestrator
   - `streamHandler`: Audio streaming via audiosocket
   - `websockHandler`: WebSocket connections
   - `zmqPubHandler`: ZMQ pub/sub messaging
3. **Goroutines**: Runs three concurrent services:
   - HTTP API server (Gin router)
   - Message subscription handler
   - Audio stream listener

#### Package Structure

**`pkg/` packages** (core infrastructure):
- `cachehandler`: Redis operations
- `dbhandler`: Database queries and transactions
- `servicehandler`: Main business logic coordinator - orchestrates requests to other managers via RabbitMQ
- `streamhandler`: Real-time audio streaming (audiosocket protocol)
- `websockhandler`: WebSocket connection management
- `subscribehandler`: RabbitMQ message subscription
- `zmqpubhandler`/`zmqsubhandler`: ZMQ messaging for events
- `mediahandler`: Media file handling

**`lib/` packages** (HTTP layer):
- `middleware`: Authentication middleware (JWT validation)
- `service`: HTTP endpoint handlers (auth, ping)

**`models/`**: Request/response data structures and DTOs

**`gens/openapi_server`**: Auto-generated OpenAPI server code

### Communication Patterns

#### Inter-Service Communication
The API manager doesn't implement business logic directly. Instead:
1. HTTP requests arrive at `lib/service` handlers
2. Handlers call `pkg/servicehandler` methods
3. ServiceHandler sends RabbitMQ requests to appropriate managers (call-manager, flow-manager, etc.)
4. Responses are returned asynchronously via RabbitMQ

Example: Creating a conference call involves sending a request to `bin-conference-manager` via the `requestHandler`.

#### Message Queue Architecture
- **RabbitMQ**: Primary async communication bus
- **ZMQ**: Pub/sub events (webhooks, agent events)
- Queue naming: `bin-manager.<service>.request` pattern

### Configuration

Runtime configuration via CLI flags (defined in `main.go`):
- `-dsn`: MySQL connection string
- `-rabbit_addr`: RabbitMQ address
- `-redis_addr`: Redis address
- `-jwt_key`: JWT signing key
- `-gcp_project_id`, `-gcp_bucket_name`: Google Cloud Storage
- `-ssl_cert_base64`, `-ssl_private_base64`: SSL certificates (base64 encoded)

### Authentication & Security
- JWT-based authentication (see `lib/middleware/authenticate.go`)
- Middleware validates tokens on protected endpoints
- Login endpoint: `/auth/login` generates JWT tokens

### Testing Patterns
- Mock generation using `go:generate` with `mockgen`
- Tests co-located with source files (`*_test.go`)
- Mock files: `mock_*.go` in respective packages

## Common Workflows

### Adding a New API Endpoint
1. Define OpenAPI spec in `openapi/config_server/`
2. Regenerate code: run `oapi-codegen` or update generation scripts
3. Implement handler in `lib/service/` or register with generated router
4. Add business logic in `pkg/servicehandler/` if calling other managers
5. Update Swagger annotations and regenerate docs

### Modifying Inter-Service Communication
When adding calls to a new manager:
1. Import the manager's models from the monorepo (e.g., `monorepo/bin-X-manager/models/...`)
2. Use `serviceHandler.requestHandler` to send RabbitMQ requests
3. Follow the existing pattern in `pkg/servicehandler/main.go`

### Working with Database Changes
Database operations go through `pkg/dbhandler/`. This wraps both MySQL and Redis caching. Don't bypass this abstraction.

## Key Dependencies
- **gin-gonic/gin**: HTTP router and middleware
- **swaggo**: Swagger/OpenAPI documentation
- **go-redis/redis**: Redis client
- **rabbitmq/amqp091-go**: RabbitMQ client (via bin-common-handler)
- **pebbe/zmq4**: ZeroMQ bindings
- **golang-jwt/jwt**: JWT authentication
- **cloud.google.com/go/storage**: GCP storage integration

## Notes
- Base64 encoding is used for SSL certificates to allow passing via environment variables/flags
- The service uses structured logging via `logrus` with field-based context
- All HTTP handlers should use `c.ClientIP()` for request tracking
- Audio streaming uses the audiosocket protocol on port 9000 by default
