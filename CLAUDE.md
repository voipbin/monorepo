# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Scope

**This CLAUDE.md applies to ALL services in this monorepo.**

Whether you're working in `bin-api-manager`, `bin-flow-manager`, `bin-call-manager`, or any other service directory, the guidelines, commands, and architectural patterns described here apply uniformly across the entire monorepo.

**Service-Specific Documentation:**
- Some individual services have their own `CLAUDE.md` files with service-specific details
- **When conflicts arise, the service-specific CLAUDE.md takes precedence** over this root file
- Use the root CLAUDE.md for general monorepo patterns and service-specific CLAUDE.md for implementation details
- If a service's CLAUDE.md specifies different commands, architecture, or workflows, follow the service-specific guidance

## Overview

This is the VoIPbin monorepo - a unified backend codebase for a cloud-native CPaaS (Communication Platform as a Service) platform. The repository contains 30+ Go microservices that collectively provide VoIP, messaging, conferencing, AI integration, and communication workflow orchestration capabilities.

**Key Characteristics:**
- **Monorepo architecture** - All backend services in one repository with local module replacements
- **Microservices communication** - Services communicate via RabbitMQ RPC, not HTTP
- **Shared infrastructure** - Common MySQL database, Redis cache, RabbitMQ message broker
- **Event-driven architecture** - Pub/sub events via RabbitMQ and ZeroMQ
- **Kubernetes deployment** - Services designed for GCP GKE with Prometheus monitoring

## CRITICAL: Before Committing Changes

**⚠️ MANDATORY: ALWAYS run the verification workflow after making ANY code changes and BEFORE committing.**

This applies to ALL changes: code modifications, refactoring, bug fixes, new features, or any other changes. No exceptions.

### Regular Code Changes Workflow

**For normal code changes (bug fixes, features, refactoring), run this workflow BEFORE committing:**

```bash
# Navigate to the service directory where changes were made
cd bin-<service-name>

# Run the verification workflow (NO dependency updates)
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**What this does:**
1. `go mod tidy` - Cleans up go.mod and go.sum files
2. `go mod vendor` - Vendors dependencies for reproducible builds
3. `go generate ./...` - Regenerates mocks and generated code
4. `go test ./...` - Runs all tests to ensure nothing broke
5. `golangci-lint run -v --timeout 5m` - Lints code for quality issues

**This runs AFTER making changes but BEFORE `git commit`.**

### Dependency Update Workflow

**Only when specifically updating dependencies, run this workflow:**

```bash
# Navigate to the service directory
cd bin-<service-name>

# Run the full update workflow (WITH dependency updates)
go get -u ./... && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Why separate workflows?**
- `go get -u ./...` updates ALL dependencies to latest versions
- Mixing dependency updates with feature changes makes PR review harder
- Dependency updates should be separate commits/PRs when possible
- For regular code changes, only update dependencies if needed

**Both workflows are MANDATORY before committing** - Do not skip any step. The monorepo's interdependencies require this to maintain consistency and catch issues early.

### Special Case: Changes to bin-common-handler

**🚨 CRITICAL: If you make ANY changes to `bin-common-handler`, you MUST update ALL 30+ services in the monorepo.**

The `bin-common-handler` is a shared library used by ALL other services. Changes to it require updating every service to maintain consistency across the entire monorepo.

#### What Counts as a "Change to bin-common-handler"

Changes that affect dependent services include:
- ✅ Adding new functions or methods (e.g., `PrepareFields()`, `ScanRow()`)
- ✅ Modifying existing interfaces or function signatures
- ✅ Adding/removing fields in shared models (`models/outline`, `models/identity`)
- ✅ Changing behavior of existing handlers
- ✅ Updating dependencies in bin-common-handler's `go.mod`

#### Complete Update Workflow

**Step 1: Make changes to bin-common-handler**
```bash
cd bin-common-handler

# Make your changes, then run verification
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Step 2: Update ALL services in monorepo**
```bash
# From monorepo root, update all 30+ services
cd /home/pchero/gitvoipbin/monorepo

find . -maxdepth 2 -name "go.mod" -execdir bash -c \
  "echo 'Updating \$(basename \$(pwd))...' && \
   go mod tidy && \
   go mod vendor && \
   go generate ./... && \
   go test ./..." \;
```

**What this does for EACH service:**
1. `go mod tidy` - Syncs go.mod with bin-common-handler changes (NOT `go get -u`)
2. `go mod vendor` - Updates vendored bin-common-handler code
3. `go generate ./...` - Regenerates mocks that depend on bin-common-handler interfaces
4. `go test ./...` - Verifies the service still works with new bin-common-handler

**Step 3: Verify key services compile**
```bash
# Spot check critical services
cd bin-api-manager && go build ./...
cd ../bin-call-manager && go build ./...
cd ../bin-flow-manager && go build ./...
```

**Step 4: Commit ALL changes together**
```bash
cd /home/pchero/gitvoipbin/monorepo

# Commit bin-common-handler changes
git add bin-common-handler/
git commit -m "feat(common-handler): add new database mapping utilities"

# Commit dependency updates for all services
git add */go.mod */go.sum
git commit -m "chore: update dependencies after bin-common-handler changes"
```

#### Projects Affected

**All services depend on bin-common-handler** (30+ projects):
- bin-api-manager (REST API gateway)
- bin-call-manager (Call routing)
- bin-flow-manager (Flow execution)
- bin-conference-manager (Conferencing)
- bin-ai-manager (AI integration)
- bin-webhook-manager (Webhooks)
- bin-agent-manager (Agent management)
- bin-customer-manager (Customer data)
- And 22+ other services...

#### Why This Is Critical

- **Compilation errors**: Stale vendored dependencies will break builds
- **Runtime failures**: Services may use outdated interfaces/models
- **Test failures**: Mock regeneration required if interfaces changed
- **Inconsistent state**: Half-updated monorepo leads to subtle bugs

#### Common Mistakes to Avoid

❌ **DON'T run `go get -u`** - This updates ALL dependencies, not just bin-common-handler
❌ **DON'T skip `go generate`** - Mocks will be stale if interfaces changed
❌ **DON'T skip `go test`** - Won't catch breaking changes until production
❌ **DON'T commit bin-common-handler alone** - All services must be updated together

✅ **DO run the complete workflow** - `go mod tidy && go mod vendor && go generate && go test`
✅ **DO verify tests pass** for all services before committing
✅ **DO commit bin-common-handler and dependency updates together** (can be separate commits in same PR)

#### Troubleshooting

**If services fail to compile after update:**
```bash
# Clean and retry for specific service
cd bin-<service-name>
rm -rf vendor/
go clean -modcache
go mod tidy && go mod vendor && go build ./...
```

**If tests fail in multiple services:**
- Check if bin-common-handler changes introduced breaking changes
- Review interface modifications
- Verify mock regeneration completed successfully

**Never commit changes to bin-common-handler without updating dependent services.**

### Special Case: Changes to Public-Facing Models and OpenAPI Schemas

**CRITICAL: If you modify public-facing data structures in ANY service, you MUST verify and update the corresponding OpenAPI schemas in `bin-openapi-manager`.**

**What are public-facing models?**
Services expose data to external APIs, webhooks, and events through specific structs:
- `WebhookMessage` structs (e.g., `call.WebhookMessage`, `conference.WebhookMessage`)
- API response models that are returned to external clients
- Any data structure used in RPC responses to `bin-api-manager`

**The Rule:**
When a service defines what data is exposed publicly (via `WebhookMessage` or similar), the corresponding OpenAPI schema in `bin-openapi-manager/openapi/openapi.yaml` MUST accurately reflect all fields.

**Example Mapping:**
```
bin-call-manager/models/call/webhook.go → WebhookMessage struct
    ↓ maps to ↓
bin-openapi-manager/openapi/openapi.yaml → CallManagerCall schema

bin-conference-manager/models/conference/webhook.go → WebhookMessage struct
    ↓ maps to ↓
bin-openapi-manager/openapi/openapi.yaml → ConferenceManagerConference schema
```

**Validation Process:**
1. **Before modifying a WebhookMessage or public model:**
   - Note all fields currently in the struct

2. **After making changes:**
   - Identify the corresponding OpenAPI schema in `bin-openapi-manager/openapi/openapi.yaml`
   - Compare fields in the Go struct vs. OpenAPI schema
   - Add any missing fields to the OpenAPI schema
   - Remove deprecated fields from the OpenAPI schema
   - Ensure field types match (string, array, object, etc.)
   - Ensure enums are properly defined

3. **Regenerate OpenAPI code:**
```bash
cd bin-openapi-manager
go generate ./...
```

4. **Update dependent services:**
```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./...
```

**Why this is critical:**
- `bin-openapi-manager` is the source of truth for the public REST API contract
- External API consumers rely on accurate OpenAPI documentation
- API documentation (Swagger UI) must match actual service behavior
- Type mismatches cause runtime errors and API consumer confusion

**Common scenarios requiring validation:**
- Adding new fields to a model
- Changing field types or nullability
- Adding new enum values
- Renaming fields (breaking change - coordinate carefully)
- Removing deprecated fields

**Never commit changes to public-facing models without verifying OpenAPI schemas are synchronized.**

## Git Workflow: Commit Message Format

**CRITICAL: This is a monorepo containing multiple projects. Commit messages MUST specify which projects were affected.**

### Commit Message Structure

**Title (first line):**
```
VOIP-[ticket-number]: Brief description of change
```
or
```
NOJIRA-Underscored_Description
```

**Body (subsequent lines):**
List each affected project with specific changes:
```
- bin-common-handler: Fixed type handling in database mapper
- bin-flow-manager: Updated flow execution to use new types
- bin-call-manager: Refactored call handler to support new interface
- bin-conference-manager: Updated conference creation logic
```

**Complete Example (Narrative Style):**
```
VOIP-1190: Refactor database handlers to use commondatabasehandler pattern

Successfully refactored 22 microservices to adopt standardized commondatabasehandler
pattern from bin-common-handler, improving type safety and code consistency across
the entire monorepo.

- bin-common-handler: Provides PrepareFields(), ScanRow(), ApplyFields(), and
  GetDBFields() utilities for standardized database operations
- bin-ai-manager: Adds db tags to ai, aicall, message, and summary models with
  typed Field constants for type-safe database operations
- bin-ai-manager: Migrates all dbhandler operations to use PrepareFields() for
  INSERT/UPDATE and ScanRow() for result scanning
- bin-call-manager: Adds field.go files for call, confbridge, groupcall, and
  recording models with typed Field constants
- bin-call-manager: Refactors all database queries to use Squirrel SetMap()
  with PrepareFields() instead of Columns().Values()
- bin-conference-manager: Adds ConvertStringMapToFieldMap() helper functions
  for filter conversion from external APIs
... (continue for all affected services)

Test results: All 28 services passing (477 files modified)
```

**Merge Commit Format:**
```
Merge VOIP-[ticket-number]: Brief description

- bin-service-1: What changed
- bin-service-2: What changed
```

### Rules

1. **Always list affected projects** - Even if it's just one project
2. **Be specific** - Describe what changed in each project, not just "updated"
3. **Keep title concise** - Detailed changes go in the body
4. **Use present tense** - "Add feature" not "Added feature"
5. **Use dashes (`-`) for bullet points** - Never use asterisks (`*`)
6. **Add narrative summary** - For large changes, include a summary paragraph before the bullet list
7. **Multiple bullets per service** - Complex changes should have multiple detailed bullets
8. **Include test results** - Add summary line at the end (e.g., "Test results: All 28 services passing")

**Good examples:**
```
VOIP-1234: Add JWT authentication support

- bin-api-manager: Implement JWT middleware and token validation
- bin-customer-manager: Add token generation endpoints
- bin-common-handler: Add JWT utility functions
```

```
NOJIRA-Fix_database_connection_leak

- bin-call-manager: Close database connections in defer statements
- bin-conference-manager: Add connection pool timeout handling
```

**Bad examples:**
```
Fixed bug  ❌ (No ticket number, no affected projects)
```

```
VOIP-1234: Updated everything  ❌ (Not specific, no project list)
```

```
VOIP-1234: Add feature
- Updated files  ❌ (Not specific about which projects)
```

## Git Workflow: Branch Management

**CRITICAL: Before making ANY changes or commits, ALWAYS check the current branch first.**

**If the current branch is `main`:**
1. **STOP - DO NOT make commits on main**
2. Ask the user to create a feature branch first
3. Suggest a branch name following this convention: `NOJIRA-Underscored_change_summary`
4. Wait for user confirmation before proceeding with any code changes

**Example prompt when starting work:**
```
You're currently on the main branch. It's recommended to create a feature branch before making changes.

Suggested branch name: NOJIRA-Fix_conference_customer_id

Would you like to:
1. Create and switch to this branch
2. Use a different branch name
3. Work on main anyway (not recommended)
```

**Correct Workflow:**
```bash
# Step 1: ALWAYS check current branch BEFORE making any changes
git branch --show-current

# Step 2: If on main, create feature branch BEFORE any edits
git checkout -b NOJIRA-Descriptive_change_summary

# Step 3: Make your code changes

# Step 4: Run the verification workflow BEFORE committing (from section above)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# Step 5: Commit changes
git add .
git commit -m "Descriptive commit message"

# Step 6: Push to remote
git push -u origin NOJIRA-Descriptive_change_summary
```

**Branch naming convention:**
- Format: `NOJIRA-Underscored_change_or_plan_summary`
- Examples:
  - `NOJIRA-Fix_conference_customer_id`
  - `NOJIRA-Add_user_authentication`
  - `NOJIRA-Refactor_flow_manager_cobra_viper`
- Use underscores (_) to separate words
- Keep it concise but descriptive

**Only proceed with working on main if the user explicitly confirms.**

### CRITICAL: Never Commit Directly to Main

**ALWAYS work on feature branches. NEVER commit directly to `main` without explicit user permission.**

**Before making ANY changes or commits:**
1. Check current branch: `git branch --show-current`
2. If on `main`, create a feature branch FIRST
3. Only commit to main if user explicitly approves

**This applies to ALL changes:**
- Code changes
- Documentation updates (including CLAUDE.md)
- Configuration files
- Any other modifications

**Prohibited workflow:**
```bash
# ❌ WRONG - Committing directly to main
git branch --show-current  # shows: main
# ... make changes to files ...
git add .
git commit -m "some change"  # NEVER DO THIS ON MAIN
```

**Correct workflow:**
```bash
# ✅ CORRECT - Create branch first
git branch --show-current  # shows: main
git checkout -b NOJIRA-Descriptive_change_name  # Create feature branch FIRST
# ... make changes to files ...
git add .
git commit -m "some change"  # Safe - on feature branch
```

**Exception:**
Only commit directly to main when user explicitly says:
- "commit this to main"
- "yes, commit directly to main" (in response to your question)

### CRITICAL: Merging to Main Branch

**NEVER merge any branch to `main` without explicit user permission.**

**Prohibited actions without user approval:**
- `git merge <branch>` while on main
- `git merge <branch> --no-ff` while on main
- Any operation that merges commits into main

**Required workflow:**
1. ✅ Push feature branches to remote: `git push -u origin <branch-name>`
2. ✅ Create pull requests on GitHub for review
3. ❌ **DO NOT** merge to main directly - always ask user first

**If user says "push it" or similar:**
- **ONLY push the current branch** to remote
- **DO NOT assume** this means merge to main
- **ASK explicitly** if merge to main is intended

**Example - What to do:**
```
User: "push it"
Claude: "I'll push the current branch to remote. Should I also merge it to main, or just push the feature branch?"
```

**Only merge to main when the user explicitly says:**
- "merge to main"
- "merge it to main and push"
- "yes, merge to main" (in response to your question)

## Build & Development Commands

### Prerequisites
Some services require system libraries:
```bash
# ZMQ libraries (required by bin-api-manager and other services)
apt update && apt install -y pkg-config libzmq5 libzmq3-dev libczmq4 libczmq-dev
```

### Building Services

Each service follows the same build pattern:

```bash
# Navigate to a service directory
cd bin-<service-name>

# Configure git for private modules (if needed)
git config --global url."https://${GL_DEPLOY_USER}:${GL_DEPLOY_TOKEN}@gitlab.com".insteadOf "https://gitlab.com"
export GOPRIVATE="gitlab.com/voipbin"

# Download dependencies
go mod download

# Vendor dependencies (optional but recommended for reproducible builds)
go mod vendor

# Build the service
go build ./cmd/...

# Run the service with default configuration
./<service-name>

# Show available configuration flags
./<service-name> --help
```

### Testing

```bash
# From monorepo root: Run tests for all services
find . -name "go.mod" -execdir go test ./... \;

# Test a specific service
cd bin-<service-name>
go test -v ./...

# Run tests with coverage
go test -coverprofile cp.out -v ./...
go tool cover -html=cp.out -o cp.html
go tool cover -func=cp.out

# Test a specific package
go test -v ./pkg/servicehandler/...
```

### Code Generation

Many services use code generation for mocks and API specs:

```bash
# Generate mocks for a specific service
cd bin-<service-name>
go generate ./...

# Generate OpenAPI models (from bin-openapi-manager)
cd bin-openapi-manager
go generate ./...

# Update all services in monorepo (run from root)
ls -d */ | xargs -I {} bash -c "cd '{}' && go get -u ./... && go mod vendor && go generate ./... && go test ./..."
```

### Linting

```bash
# Run go vet
go vet ./...

# Run golangci-lint (recommended)
golangci-lint run -v --timeout 5m
```

## Architecture

### Service Categories

**Core API & Gateway:**
- `bin-api-manager` - External REST API gateway with JWT authentication, Swagger UI at `/swagger/index.html`
- `bin-openapi-manager` - Centralized OpenAPI 3.0 specification repository, generates Go types used by all services

**Call & Media Management:**
- `bin-call-manager` - Inbound/outbound call routing, media control (recording, transcription, DTMF, hold, mute)
- `bin-flow-manager` - Flow execution engine (IVR workflows), manages action sequences for calls
- `bin-conference-manager` - Audio conferencing with recording, transcription, and media streaming
- `bin-transcribe-manager` - Audio transcription services (STT)
- `bin-tts-manager` - Text-to-Speech integration
- `bin-registrar-manager` - SIP registrar (UDP/TCP/WebRTC)
- `voip-asterisk-proxy` - Integration proxy for Asterisk PBX

**AI & Automation:**
- `bin-ai-manager` - AI chatbot integrations, summarization, task extraction
- `bin-pipecat-manager` - AI voice assistant pipeline management

**Queue & Routing:**
- `bin-queue-manager` - Call queueing and distribution logic
- `bin-route-manager` - Routing policies and rules
- `bin-transfer-manager` - Call transfer logic

**Customer & Agent Management:**
- `bin-customer-manager` - Customer accounts and relationships
- `bin-agent-manager` - Agent presence, status, permissions, addresses
- `bin-billing-manager` - Billing accounts, balance tracking, subscription management

**Campaign & Outbound:**
- `bin-campaign-manager` - Outbound dialing campaigns with service level tracking
- `bin-outdial-manager` - Outbound call dialer engine
- `bin-number-manager` - DID and phone number provisioning

**Messaging & Communication:**
- `bin-message-manager` - SMS and messaging
- `bin-email-manager` - Email sending and inbox parsing
- `bin-chat-manager` - Web chat and live chat integration
- `bin-conversation-manager` - Conversation thread management

**Infrastructure & Utilities:**
- `bin-common-handler` - Shared library (RabbitMQ handlers, data models, utilities)
- `bin-storage-manager` - File storage backend (integrates with GCP Cloud Storage)
- `bin-webhook-manager` - Webhook sender for customer notifications
- `bin-hook-manager` - Webhook receivers
- `bin-tag-manager` - Resource labeling and tagging
- `bin-dbscheme-manager` - Database schemas and migrations
- `bin-sentinel-manager` - Monitoring and health checks

### Inter-Service Communication

**RabbitMQ RPC Pattern:**
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

**Queue Naming Convention:**
- Request queues: `bin-manager.<service-name>.request`
- Event queues: `bin-manager.<service-name>.event`
- Delayed exchange: `bin-manager.delay`

**Making Inter-Service Requests:**
Use `bin-common-handler/pkg/requesthandler` which provides typed methods for all services:

```go
import "monorepo/bin-common-handler/pkg/requesthandler"

// Example: Creating a call via call-manager
reqHandler := requesthandler.New(sockHandler)
call, err := reqHandler.CallV1CallCreate(context.Background(), createReq)
```

**Event Publishing:**
Use `bin-common-handler/pkg/notifyhandler` for publishing events:

```go
import "monorepo/bin-common-handler/pkg/notifyhandler"

notifyHandler := notifyhandler.New(sockHandler)
notifyHandler.PublishEvent(event)
```

### Configuration Management

Most services use **Cobra + Viper** for configuration (see `internal/config` packages). Configuration precedence:

1. Command-line flags (highest priority)
2. Environment variables
3. Default values

**Common configuration patterns:**

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

**Common configuration fields:**
- Database: `--database_dsn` / `DATABASE_DSN`
- RabbitMQ: `--rabbitmq_address` / `RABBITMQ_ADDRESS`
- Redis: `--redis_address`, `--redis_password`, `--redis_database`
- Prometheus: `--prometheus_endpoint`, `--prometheus_listen_address`
- Queue names: `--rabbit_queue_listen`, `--rabbit_queue_event`

### Package Organization Pattern

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

**Handler Pattern:**
Each handler follows:
1. Interface definition in `main.go` or package file
2. Implementation struct with injected dependencies
3. Mock generation via `//go:generate mockgen`
4. Tests in `*_test.go` using table-driven tests

### Database & Caching

**MySQL** - Shared database for persistent storage
- Query builder: `github.com/Masterminds/squirrel`
- Access via `pkg/dbhandler` abstractions in each service

**Redis** - Distributed cache for:
- Activeflow state (flow-manager)
- Call state and temporary data
- Agent presence information
- Rate limiting and throttling

**Pattern:** Always use `dbhandler` packages - they provide unified interface to both database and cache.

### Testing Patterns

**Mock Generation:**
```go
//go:generate mockgen -package packagename -destination ./mock_main.go -source main.go -build_flags=-mod=mod
```

**Test Structure:**
- Tests co-located with source: `*_test.go`
- Table-driven tests with subtests
- Mocks: `go.uber.org/mock`
- 34+ test files in flow-manager, similar counts in other services

**Running Tests:**
```bash
go test -v ./...
go test -v ./pkg/specifichandler/...
```

### CircleCI Continuous Integration

Path filtering enables selective service testing:
- `.circleci/config.yml` - Path filter setup
- `.circleci/config_work.yml` - Actual build jobs
- Only changed services are tested on each commit

## Common Workflows

### Adding a New API Endpoint

1. Define OpenAPI spec in `bin-openapi-manager/openapi/paths/<resource>/`
2. Regenerate models: `cd bin-openapi-manager && go generate ./...`
3. Add handler in target service's `pkg/listenhandler/`
4. Implement business logic in appropriate domain handler
5. Register endpoint in `bin-api-manager` if public-facing
6. Update Swagger docs in api-manager: `swag init`

### Creating a New Flow Action

1. Define action type in `bin-flow-manager/models/action/`
2. Add action handler in `bin-flow-manager/pkg/actionhandler/`
3. Implement execution logic in target service (e.g., call-manager)
4. Update flow-manager's request handler to route to new service
5. Add tests for action validation and execution

### Adding a New Manager Service

1. Create directory: `bin-<name>-manager/`
2. Copy structure from existing service (e.g., tag-manager)
3. Update `bin-api-manager/go.mod` with replace directive
4. Add RabbitMQ queue names to `bin-common-handler/models/outline/`
5. Add request methods to `bin-common-handler/pkg/requesthandler/`
6. Update `.circleci/config.yml` path mappings

### Modifying Shared Models

When changing `bin-common-handler` or `bin-openapi-manager`:

1. Make changes in the shared package
2. Regenerate if needed: `go generate ./...`
3. Test impact on dependent services
4. Update all affected services' vendor directories if using vendoring
5. Coordinate deployment - shared changes affect multiple services

### Working with Flow-Manager

**Flow Execution Pattern:**
1. Flow = template with action sequence
2. Activeflow = running instance attached to call
3. Actions dispatched to appropriate managers via RabbitMQ
4. Stack-based execution for nested flows (branch, call actions)

**Variable Substitution:**
Variables use format `${variable.key}`:
- `${voipbin.activeflow.id}` - Built-in activeflow metadata
- `${voipbin.call.caller_id}` - Call information from call-manager
- Custom variables stored per activeflow

## Key Dependencies

**All Services:**
- `github.com/go-sql-driver/mysql` - MySQL driver
- `github.com/go-redis/redis/v8` - Redis client
- `github.com/rabbitmq/amqp091-go` - RabbitMQ client
- `github.com/sirupsen/logrus` - Structured logging
- `github.com/prometheus/client_golang` - Prometheus metrics
- `go.uber.org/mock` - Mock generation for testing

**Common Tools:**
- `github.com/Masterminds/squirrel` - SQL query builder
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management
- `github.com/gofrs/uuid` - UUID generation

**API Gateway Specific:**
- `github.com/gin-gonic/gin` - HTTP router
- `github.com/swaggo/swag` - Swagger documentation
- `github.com/oapi-codegen/oapi-codegen` - OpenAPI code generation
- `github.com/golang-jwt/jwt` - JWT authentication
- `github.com/pebbe/zmq4` - ZeroMQ bindings

**Cloud Integration:**
- `cloud.google.com/go/storage` - GCP Cloud Storage

## Deployment

**Kubernetes:**
- Each service has `k8s/` directory with manifests
- Prometheus metrics exposed on configured port (default `:2112` on `/metrics`)
- Dockerfiles for containerization

**Infrastructure Requirements:**
- GCP GKE cluster (recommended)
- MySQL database
- Redis cluster
- RabbitMQ cluster
- Asterisk/RTPEngine for media (external to this repo)
- Public domain with TLS

## Important Notes

### Monorepo-Specific Practices

1. **Always use replace directives** - All `monorepo/bin-*` imports use local paths in `go.mod`
2. **Coordinate breaking changes** - Changes to shared packages affect multiple services
3. **Test holistically** - Inter-service changes require testing communication flow
4. **Update go.mod carefully** - Adding dependencies may affect all services

### Communication Patterns

1. **Never use HTTP between services** - Always use RabbitMQ RPC
2. **Use typed request methods** - Don't construct `sock.Request` manually, use `requesthandler`
3. **Handle async responses** - RabbitMQ RPC is asynchronous
4. **Publish events for notifications** - Use `notifyhandler.PublishEvent()` for pub/sub

### Code Quality

1. **Generate mocks** - Run `go generate ./...` after interface changes
2. **Write table-driven tests** - Follow existing test patterns
3. **Use structured logging** - Include context fields with `logrus.WithFields()`
4. **Handle errors properly** - Wrap errors with `github.com/pkg/errors`

### Security Considerations

1. **JWT authentication** - bin-api-manager validates all external requests
2. **No secrets in code** - Use environment variables or CLI flags
3. **Base64 for certificates** - SSL certs passed as base64 strings in config
4. **Validate input** - Always validate data at service boundaries

## Resources

- Admin Console: https://admin.voipbin.net/
- Agent Interface: https://talk.voipbin.net/
- API Documentation: https://api.voipbin.net/docs/
- Project Site: http://voipbin.net/
- Architecture Diagram: `architecture_overview_all.png` in repo root
