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

## api-control CLI Tool

A command-line tool for API Manager operations. **All output is JSON format** (stdout), logs go to stderr.

```bash
# Display version information
./bin/api-control version

# Check health status of API Manager dependencies (database, cache, RabbitMQ)
./bin/api-control health
```

Uses same environment variables as api-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

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

### Developer Documentation (docsdev)

The `docsdev/` directory contains Sphinx-based developer documentation for the entire VoIPBIN platform. This is the main public-facing documentation for external developers.

#### Documentation Structure

```
docsdev/
├── source/               # RST source files
│   ├── index.rst        # Main table of contents
│   ├── intro*.rst       # Introduction and features
│   ├── quickstart*.rst  # Getting started guides
│   ├── *_overview.rst   # Feature overviews
│   ├── *_tutorial.rst   # Step-by-step tutorials
│   ├── *_struct*.rst    # Data structure references
│   ├── glossary.rst     # Terminology definitions
│   └── _static/         # Images and static assets
├── build/               # Generated HTML (committed to git)
├── conf.py              # Sphinx configuration
└── Makefile             # Build commands
```

#### Building Documentation

**Prerequisites:**
```bash
# Create virtual environment
python3 -m venv .venv_docs
source .venv_docs/bin/activate

# Install Sphinx and required extensions
pip install sphinx sphinx-rtd-theme sphinx-wagtail-theme sphinxcontrib-youtube
```

**Build HTML:**
```bash
cd docsdev
make html

# Or use sphinx-build directly
sphinx-build -M html source build
```

**Output:** Generated HTML files in `build/html/` (committed to git)

**View locally:**
```bash
# Open in browser
open build/html/index.html  # macOS
xdg-open build/html/index.html  # Linux
```

#### AI-Native RST Writing Guidelines

**Target Audience:** Documentation writers, developers, and AI agents (LLMs).
**Goal:** Make RST documentation machine-actionable for AI agents without losing human readability.
**Applicability:** These rules apply when **creating new RST pages or modifying existing ones**. Do not retroactively rewrite pages you are not otherwise touching. When editing an existing page, apply the rules to the sections you are changing.

**Relationship to OpenAPI Rules:** These RST guidelines are the documentation counterpart to the "AI-Native OpenAPI Specification Rules" in `bin-openapi-manager/CLAUDE.md`. When updating both an OpenAPI schema and its corresponding RST documentation, both guideline sets apply.

**Core Philosophy: "Explicit Over Implicit"**

AI models cannot infer context. Every piece of information must be self-contained.

- Human-Native: "Enter the ID to stop it."
- AI-Native: "Enter the `activeflow_id` (UUID) returned from `POST /activeflows` to stop the running flow."

**File Naming:**
- `<resource>.rst` - Main resource file (includes other files)
- `<resource>_overview.rst` - Conceptual overview
- `<resource>_tutorial.rst` - Step-by-step guide
- `<resource>_struct_<type>.rst` - Data structure reference

##### API Endpoint References in Documentation

**Rule: Use user-facing endpoints with FQDNs. Never use admin-only endpoints in user documentation.**

1. **Always use fully qualified URLs** — Write `GET https://api.voipbin.net/v1.0/customer`, not `GET /customer` or `GET /customers`.
2. **Use `/customer` (singular) for normal users** — The `/customer` endpoint returns the authenticated user's own customer info. The `/customers` (plural) and `/customers/{id}` endpoints are admin-only and must NOT appear in user-facing documentation (quickstart, tutorials, overviews).
3. **Use `/customer` for webhook configuration** — Write `PUT https://api.voipbin.net/v1.0/customer` (not `PUT /customers/{id}`) when documenting webhook setup or customer profile updates.

| Context | Correct | Wrong |
|---|---|---|
| Get customer info | ``GET https://api.voipbin.net/v1.0/customer`` | ``GET /customers``, ``GET /customers/{id}`` |
| Update webhook | ``PUT https://api.voipbin.net/v1.0/customer`` | ``PUT /customers/{id}`` |
| Get numbers | ``GET https://api.voipbin.net/v1.0/numbers`` | ``GET /numbers`` |
| Create a call | ``POST https://api.voipbin.net/v1.0/calls`` | ``POST /calls`` |

**Exception:** Admin documentation (if it exists separately) may reference `/customers/{id}`.

##### The 5 Commandments

**Rule 1: Data Provenance (Where Do IDs Come From?)**

Never mention an ID parameter without stating its source endpoint.

```
Bad:
  queue_id: The ID of the queue.

Good:
  queue_id: The unique UUID of the queue. Obtained from the ``id`` field of ``GET /queues``.
```

**Rule 2: Strict Typing in Prose**

Use specific formats. Never use vague terms like "text" or "number."

| Vague Term | AI-Native Replacement |
|---|---|
| "The phone number" | "Phone number in E.164 format (e.g., `+821012345678`). Must start with `+`. No dashes or spaces." |
| "The ID" | "UUID string (e.g., `a1b2c3d4-e5f6-7890-abcd-ef1234567890`)" |
| "The timestamp" | "ISO 8601 / RFC 3339 timestamp (e.g., `2026-01-15T10:30:00Z`)" |
| "The status" | "Enum string. One of: `dialing`, `ringing`, `progressing`, `hangup`." |

**Rule 3: AI Hint Admonitions**

Add `.. note:: **AI Implementation Hint**` blocks for normalization rules, decision logic, and common pitfalls.

```rst
.. note:: **AI Implementation Hint**

   If the user provides a local phone number (e.g., ``010-1234-5678``),
   you **MUST** normalize it to E.164 format (``+821012345678``) before
   calling this API.
```

Use for:
- Input normalization (phone formats, timezone conversions)
- Prerequisite checks ("Verify the flow exists via `GET /flows/{id}` before assigning")
- Cost warnings ("This operation deducts credits from the account balance")
- Async behavior ("This returns immediately. Poll `GET /calls/{id}` or use WebSocket for status updates")

Required in overview and tutorial files (at least one per page). In struct files: include only when field usage has non-obvious requirements.

**Rule 4: Explicit State Transitions (Enums)**

Every enum or status field must list all possible values with descriptions. Applies only to resources that have a status or state field.

```
Bad:
  Returns the status of the call.

Good:
  status (enum string):
  * ``dialing``: System is currently dialing the destination.
  * ``ringing``: Destination device is ringing, awaiting answer.
  * ``progressing``: Call answered. Audio is flowing between parties.
  * ``terminating``: System is ending the call.
  * ``canceling``: Originator is canceling before answer (outgoing calls only).
  * ``hangup``: Call ended. Final state — no further changes possible.
```

**Rule 5: Self-Correcting Error Handling**

Provide cause-and-fix pairs for HTTP error codes. This lets AI agents self-heal when an API call fails.

```rst
Troubleshooting
---------------

* **400 Bad Request:**
    * **Cause:** The ``to`` field contains dashes or spaces.
    * **Fix:** Remove all non-numeric characters except the leading ``+``.

* **402 Payment Required:**
    * **Cause:** Insufficient account balance.
    * **Fix:** Check balance via ``GET /billing-accounts``. Prompt user to top up.

* **404 Not Found:**
    * **Cause:** The resource UUID does not exist or belongs to another customer.
    * **Fix:** Verify the UUID was obtained from a recent ``GET`` list call.

* **409 Conflict:**
    * **Cause:** Resource is in an incompatible state for this operation.
    * **Fix:** Check current status via ``GET /resource/{id}`` before retrying.
```

##### Per-File-Type Rules

**`*_overview.rst` Files:**

1. AI Context block at the top of the overview:
   ```rst
   .. note:: **AI Context**

      * **Complexity:** Low | Medium | High
      * **Cost:** Free | Chargeable (credit deduction)
      * **Async:** Yes/No. If yes, state how to track status.
   ```
2. State lifecycle with all enum values (if the resource has a status field)
3. Related documentation cross-refs using `:ref:`
4. At least one AI Implementation Hint (Rule 3)

**`*_struct_*.rst` Files:**

1. Type for every field (UUID, String(E.164), enum, ISO 8601, Boolean, Integer, Array, Object)
2. Provenance for every ID/reference field ("Obtained from `GET /endpoint`") (Rule 1)
3. Required/Optional marker for request body structs
4. Enum values listed inline or via `:ref:` to a dedicated section (Rule 4)
5. AI Implementation Hint only when field usage has non-obvious requirements

Example field description format:
```
* ``flow_id`` (UUID, Optional): The flow to execute when the call is answered.
  Obtained from the ``id`` field of ``GET /flows``.
  Set to ``00000000-0000-0000-0000-000000000000`` if no flow is assigned.
```

**`*_tutorial.rst` Files:**

1. Prerequisites block listing what IDs/resources are needed and how to obtain them:
   ```rst
   Prerequisites
   +++++++++++++

   Before creating a call, you need:

   * A source phone number (E.164 format). Obtain one via ``GET /numbers``.
   * A destination phone number (E.164 format) or extension.
   * (Optional) A flow ID (UUID). Create one via ``POST /flows`` or obtain from ``GET /flows``.
   ```
2. Complete request AND response examples for every operation
3. At least one AI Implementation Hint for common gotchas (Rule 3)
4. Response field annotations — comment key fields (e.g., `// Save this as call_id`)

**`*_troubleshooting.rst` Files:**

1. Debugging tools — list relevant API endpoints for diagnosis
2. Symptom -> Cause -> Fix pattern for each issue (Rule 5)
3. HTTP error code reference table
4. Diagnostic steps with actual API calls to run

##### Quick Checklist

Before considering any RST documentation page complete, verify:

- [ ] Every ID field states its source endpoint (Rule 1)
- [ ] Every field has an explicit type: UUID, E.164, enum, ISO 8601, etc. (Rule 2)
- [ ] Overview and tutorial pages have at least one `.. note:: **AI Implementation Hint**` (Rule 3)
- [ ] Every enum/status lists all possible values with descriptions (Rule 4)
- [ ] Request struct fields are marked Required or Optional (Rule 2)
- [ ] Error scenarios include cause + fix pairs (Rule 5)
- [ ] Cross-references use `:ref:` to link related documentation
- [ ] Code examples include both request AND response

##### RST Formatting Rules

- Use `.. code::` for code blocks (not `.. code-block::`)
- Reference other sections: `:ref:\`link-target\``
- Use `**bold**` for emphasis, not `*italic*`
- Keep lines under 120 characters when practical
- Use "VoIPBIN" consistently (not "Voipbin" or "voipbin")
- Always provide content in sections — no empty stubs like "Common used."
- Check for common typos: "Ovewview", "Acesskey", "comming", "existed"
- Grammar: "The doesn't affect" → "This doesn't affect"

#### Documentation Maintenance

**CRITICAL: When modifying any RST file in `docsdev/source/`, you MUST rebuild the HTML and commit both the RST sources and the built HTML together.**

The built HTML in `docsdev/build/` is tracked in git and deployed to production. If you commit RST changes without rebuilding, the live docs will be out of sync.

**Required steps when updating docs:**
1. Edit RST files in `source/`
2. Rebuild HTML: `cd docsdev && python3 -m sphinx -M html source build`
3. Verify changes locally in `build/html/`
4. Stage built files: `git add -f bin-api-manager/docsdev/build/` (force-add required because root `.gitignore` contains `build/`)
5. Commit both RST source files and `build/` directory together

**Recent improvements:**
- Fixed empty `common_overview.rst` (was only "Common used.")
- Populated `glossary.rst` with 25+ VoIPBIN terms
- Expanded `restful_api.rst` with API conventions and HTTP status codes
- Fixed typos in `sdk_overview.rst` and `flow_tutorial_basic.rst`

**Key sections to maintain:**
- **Glossary** (`glossary.rst`) - Keep terminology up to date
- **Common Overview** (`common_overview.rst`) - Document shared patterns
- **RESTful API** (`restful_api.rst`) - API conventions and error codes
- **Quickstart** (`quickstart*.rst`) - First-run experience for developers

#### Live Documentation

The documentation is published at:
- **Developer Docs**: https://docs.voipbin.net/ (this documentation)
- **API Reference (ReDoc)**: https://api.voipbin.net/redoc/index.html
- **API Reference (Swagger)**: https://api.voipbin.net/swagger/index.html

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

## Authentication & Authorization

### Authentication & Authorization Pattern

**CRITICAL: Authentication and authorization logic belongs ONLY in bin-api-manager, NOT in internal services.**

**Architecture:**

```
External Client → bin-api-manager → bin-<service>-manager
                  (Auth Layer)      (Business Logic)
```

**bin-api-manager Responsibilities:**
- ✅ Validate JWT tokens
- ✅ Extract customer_id from JWT
- ✅ Make RPC requests to internal services
- ✅ Check authorization (resource.CustomerID == JWT customer_id)
- ✅ Return appropriate HTTP status codes

**Internal Service Responsibilities (bin-billing-manager, bin-call-manager, etc.):**
- ✅ Process RPC requests
- ✅ Execute business logic
- ✅ Access database
- ✅ Return data or errors
- ❌ NO JWT validation
- ❌ NO customer_id authentication checks
- ❌ NO authorization logic

**Example Flow:**

```
1. Client → GET /v1/billings/550e8400-... with JWT
2. bin-api-manager:
   - Validates JWT ✓
   - Extracts customer_id: 6a93f71e-...
   - Calls: reqHandler.BillingV1BillingGet(ctx, billingID)
3. bin-billing-manager:
   - Receives RPC: GET /v1/billings/550e8400-...
   - Queries: BillingGet(ctx, billingID)
   - Returns billing record (no customer_id check)
4. bin-api-manager:
   - Receives billing: { customer_id: 6a93f71e-..., ... }
   - Checks: billing.CustomerID == JWT customer_id ✓
   - Returns: 200 OK with billing data
```

**Authorization Check Pattern:**

```go
// ✅ CORRECT - Authorization in api-manager servicehandler
func (h *serviceHandler) BillingGet(ctx context.Context, a *amagent.Agent, billingID uuid.UUID) (*bmbilling.WebhookMessage, error) {
    // 1. Fetch resource
    billing, _ := h.billingGet(ctx, billingID)

    // 2. Check permission
    if !h.hasPermission(ctx, a, billing.CustomerID, amagent.PermissionCustomerAdmin) {
        return nil, fmt.Errorf("user has no permission")
    }

    // 3. Return resource
    return billing.ConvertWebhookMessage(), nil
}

// ❌ WRONG - Authorization in internal service
func (h *listenHandler) processV1BillingGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
    customerID := ctx.Value("customer_id")  // NO! Don't do this
    billing, _ := h.billingHandler.Get(ctx, billingID)

    if billing.CustomerID != customerID {  // NO! Don't do this
        return simpleResponse(404), nil
    }
    // ...
}
```

**Why This Matters:**
- Single source of truth for authentication/authorization
- Internal services remain simple and reusable
- Easier to test business logic independently
- Clear separation of concerns
- Internal services can trust the API gateway

### Permission Requirements

**General Pattern for resource access in servicehandler:**

```go
// Private helper - fetches resource
func (h *serviceHandler) resourceGet(ctx context.Context, resourceID uuid.UUID) (*Resource, error) {
    res, err := h.reqHandler.ServiceV1ResourceGet(ctx, resourceID)
    if err != nil {
        return nil, err
    }
    return res, nil
}

// Public method - checks permission
func (h *serviceHandler) ResourceGet(ctx context.Context, a *amagent.Agent, resourceID uuid.UUID) (*WebhookMessage, error) {
    // 1. Fetch resource
    r, err := h.resourceGet(ctx, resourceID)
    if err != nil {
        return nil, err
    }

    // 2. Check permission
    if !h.hasPermission(ctx, a, r.CustomerID, amagent.PermissionCustomerAdmin) {
        return nil, fmt.Errorf("user has no permission")
    }

    // 3. Convert and return
    return r.ConvertWebhookMessage(), nil
}
```

**Resource-Specific Permission Requirements:**

**Billing and Billing Account resources:**
- Require CustomerAdmin permission ONLY (no Manager access)

```go
// Admin only - no Manager access
if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
    return nil, fmt.Errorf("user has no permission")
}
```

## Git Workflow

### Use Git Worktrees for Feature Development

**CRITICAL: Never edit files directly in the main repository directory for feature work.**

```bash
# Create a worktree for your feature
git worktree add ../worktrees/NOJIRA-feature-name -b NOJIRA-feature-name

# Work in the worktree
cd ../worktrees/NOJIRA-feature-name

# When done, remove the worktree
git worktree remove ../worktrees/NOJIRA-feature-name
```

**Why worktrees:**
- Keeps main repository clean and always on `main` branch
- Allows parallel work on multiple features
- Easy to abandon work without affecting main workspace
- Clear separation between exploration and implementation

## Common Workflows

### Adding a New API Endpoint

**CRITICAL: Always follow the OpenAPI-first workflow. Never modify api-manager code directly without updating OpenAPI specs first.**

**Correct workflow:**
1. **Update OpenAPI spec first** in `bin-openapi-manager/openapi/` (paths, schemas, etc.)
2. **Regenerate code**: `cd ../bin-openapi-manager && go generate ./...`
3. **Update api-manager dependencies**: `go mod tidy && go mod vendor`
4. **Implement handler** in `server/` using the generated types
5. **Add business logic** in `pkg/servicehandler/` if calling other managers
6. **Update Swagger annotations** and regenerate docs if needed

**Why OpenAPI-first matters:**
- Single source of truth for API contracts
- Generated types ensure consistency between spec and implementation
- Prevents drift between documentation and actual API behavior
- Makes API changes reviewable before implementation

### Modifying Inter-Service Communication
When adding calls to a new manager:
1. Import the manager's models from the monorepo (e.g., `monorepo/bin-X-manager/models/...`)
2. Use `serviceHandler.requestHandler` to send RabbitMQ requests
3. Follow the existing pattern in `pkg/servicehandler/main.go`

### Before Adding New Validation Functions

**CRITICAL: Always check for existing validation helpers before creating new ones.**

The `pkg/servicehandler/` directory contains private helper functions for fetching and validating resources:
- `callGet()`, `groupcallGet()`, `queuecallGet()`, `conferencecallGet()`, `activeflowGet()`, etc.
- `hasPermission()` for ownership/permission validation

**When implementing new API endpoints that need resource validation:**
1. Check if a private helper already exists for that resource type
2. Reuse existing helpers instead of duplicating fetch/validation logic
3. Follow the two-level pattern: private helper (fetch) + public method (permission check)

**Example - Reusing existing validation:**
```go
func (h *serviceHandler) TimelineCallEventsGet(ctx context.Context, a *amagent.Agent, callID uuid.UUID) (*TimelineResponse, error) {
    // Reuse existing callGet helper
    c, err := h.callGet(ctx, callID)
    if err != nil {
        return nil, err
    }

    // Reuse existing hasPermission
    if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
        return nil, fmt.Errorf("user has no permission")
    }

    // Continue with timeline-specific logic...
}
```

### Use Constants for Service Names

**CRITICAL: Always use `commonoutline.ServiceName*` constants instead of string literals for service names.**

```go
import commonoutline "monorepo/bin-common-handler/models/outline"

// ✅ CORRECT - Use constants
res, err := h.reqHandler.TimelineV1EventList(ctx, commonoutline.ServiceNameCallManager, ...)

// ❌ WRONG - String literals
res, err := h.reqHandler.TimelineV1EventList(ctx, "call-manager", ...)
```

**Available constants (defined in `bin-common-handler/models/outline/servicename.go`):**
- `ServiceNameCallManager` → "call-manager"
- `ServiceNameFlowManager` → "flow-manager"
- `ServiceNameQueueManager` → "queue-manager"
- `ServiceNameConferenceManager` → "conference-manager"
- And many others...

**Why this matters:**
- Prevents typos in service names
- Makes refactoring easier (rename in one place)
- Provides compile-time checking

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
