# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is the `bin-openapi-manager` service, part of the VoIPbin monorepo. It serves as the **centralized OpenAPI specification repository** for the entire VoIPbin platform. This service does not run as a standalone service but provides the source of truth for API contracts across all manager services.

**Key Purpose:**
- Maintains the comprehensive OpenAPI 3.0 specification (`openapi/openapi.yaml`) that defines all VoIPbin REST APIs
- Generates Go type definitions and models used by other services for type safety and consistency
- Provides a modular structure for organizing API paths across multiple manager services

## Architecture

### OpenAPI Structure

The OpenAPI specification is organized in a modular way:

```
openapi/
├── openapi.yaml           # Main spec (4070 lines) - aggregates all paths and schemas
└── paths/                 # Modular path definitions (~50 directories)
    ├── agents/            # Agent-related endpoints
    │   ├── main.yaml     # GET/POST /agents
    │   ├── id.yaml       # GET/PUT/DELETE /agents/{id}
    │   ├── id_status.yaml
    │   └── ...
    ├── calls/             # Call-related endpoints
    ├── conferences/       # Conference-related endpoints
    ├── files/             # File-related endpoints
    └── ...
```

**Path Organization:**
- Each resource type (agents, calls, conferences, etc.) has its own directory under `paths/`
- `main.yaml` - Collection endpoints (list, create)
- `id.yaml` - Individual resource endpoints (get, update, delete)
- `id_<action>.yaml` - Specific actions on resources (e.g., `id_status.yaml`, `id_recording_start.yaml`)
- Main `openapi.yaml` references these files: `$ref: './paths/agents/main.yaml'`

### Components Section

The `components:` section (line 192+) defines:
- **Parameters**: Common query params like `PageSize`, `PageToken` for pagination
- **Schemas**: Type definitions organized by manager service:
  - `AgentManager*` - Agent types (Agent, AgentStatus, AgentPermission, AgentRingMethod)
  - `AIManager*` - AI engine and call types
  - `CallManager*` - Call, groupcall types
  - `ConferenceManager*` - Conference types
  - `StorageManager*` - File storage types
  - And many more for each manager service

### Code Generation

Generated code is created using **oapi-codegen**:

```
configs/config_model/
├── config.generate.yaml   # oapi-codegen config
└── generate.go           # go:generate directive

gens/models/
└── gen.go                # Generated Go types (auto-generated, do not edit)
```

**Generation Process:**
1. `go generate ./...` triggers `configs/config_model/generate.go`
2. Runs: `go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config config.generate.yaml`
3. Reads: `openapi/openapi.yaml`
4. Generates: `gens/models/gen.go` with Go type definitions
5. Package name: `models` (configured in `config.generate.yaml`)

**Configuration (`config.generate.yaml`):**
```yaml
package: models
generate:
  models: true
output-options:
  skip-prune: true
```

## Common Commands

### Generate Models
```bash
# Generate Go types from OpenAPI spec
go generate ./...

# This creates/updates gens/models/gen.go with:
# - Type definitions for all schemas
# - Enums with constants
# - Validation types (using oapi-codegen/runtime/types)
```

### Validate OpenAPI Spec
```bash
# Install validator (if not already installed)
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

# Validate the OpenAPI specification
oapi-codegen -config configs/config_model/config.generate.yaml openapi/openapi.yaml > /dev/null
```

### Dependencies
```bash
# Download dependencies
go mod download

# Vendor dependencies (creates vendor/ directory)
go mod vendor

# Update dependencies
go get -u ./...
go mod tidy
```

## Monorepo Context

**Critical**: This service is consumed by ALL other manager services in the monorepo:
- `bin-agent-manager`
- `bin-call-manager`
- `bin-conference-manager`
- `bin-customer-manager`
- And 20+ other manager services

Other services import the generated models:
```go
import "monorepo/bin-openapi-manager/gens/models"
```

**Important**: Changes to OpenAPI spec require:
1. Updating `openapi/openapi.yaml` or path files
2. Running `go generate ./...` to regenerate Go types
3. Testing impact on dependent services
4. Coordinating updates across services that use changed types

## Key Implementation Details

### Service-Prefixed Type Names

All generated types are prefixed with their service name to avoid conflicts:
- Agent Manager: `AgentManagerAgent`, `AgentManagerAgentStatus`
- Call Manager: `CallManagerCall`, `CallManagerCallStatus`
- AI Manager: `AIManagerAIEngineType`, `AIManagerAIEngineModel`

Example from generated code:
```go
type AgentManagerAgent struct {
    Id          string
    CustomerId  string
    Username    string
    Status      AgentManagerAgentStatus
    Permission  AgentManagerAgentPermission
    // ...
}

type AgentManagerAgentStatus string
const (
    AgentManagerAgentStatusAvailable AgentManagerAgentStatus = "available"
    AgentManagerAgentStatusAway      AgentManagerAgentStatus = "away"
    AgentManagerAgentStatusBusy      AgentManagerAgentStatus = "busy"
    // ...
)
```

### Pagination Pattern

API uses cursor-based pagination with standard parameters:
- `page_size` (query param) - Number of items per page
- `page_token` (query param) - Cursor for next page (typically `tm_create` timestamp)

Responses include:
```yaml
CommonPagination:
  properties:
    next_page_token: string
    result: array
```

### File Structure Best Practices

When modifying the OpenAPI spec:
1. **New Resource Type**: Create new directory under `paths/` with `main.yaml` and `id.yaml`
2. **New Endpoint**: Add new YAML file in appropriate directory (e.g., `id_custom_action.yaml`)
3. **Reference in Main**: Add path reference in `openapi/openapi.yaml` paths section
4. **Schema Changes**: Update appropriate schema in `components/schemas` section
5. **Regenerate**: Always run `go generate ./...` after changes

### AI-Native OpenAPI Specification Rules (Mandatory)

These rules apply to all **NEW or MODIFIED** fields. They ensure our API spec is readable by AI agents (like Claude/ChatGPT) while maintaining compatibility with `oapi-codegen`.

#### Rule 1: Use `oneOf` for Polymorphism

Never use `additionalProperties: true` for fields whose structure depends on a sibling `type` field. Explicitly list allowed schemas using `oneOf`.

**Why:** AI agents hallucinate fields when they see a generic `type: object` without strict options.

**Caution:** Changing a field to `oneOf` changes the generated Go type (e.g., from `map[string]interface{}` to a specific struct wrapper). Always run `go generate ./...` in **both** `bin-openapi-manager` and `bin-api-manager`, then verify builds.

Bad (ambiguous):
```yaml
option:
  type: object
  additionalProperties: true
  description: "See FlowManagerActionOptionTalk if type is talk."
```

Good (AI-native):
```yaml
option:
  oneOf:
    - $ref: '#/components/schemas/FlowManagerActionOptionTalk'
    - $ref: '#/components/schemas/FlowManagerActionOptionPlay'
    - $ref: '#/components/schemas/FlowManagerActionOptionHangup'
```

#### Rule 2: Strict Structured Strings vs Safe Free Text

Distinguish between identifier/protocol strings and human-readable text.

**Why:** AI needs to know if a string follows a strict protocol or is just a label.

**Type A — Structured Strings (must have `format:` or `pattern:`):**
- UUIDs: `format: uuid`
- Timestamps: `format: date-time`
- Phone numbers: `pattern: "^\+[1-9]\d{1,14}$"`
- Enums: use `enum` keyword (do not just describe options in text)

**Type B — Free Text (must have `example:`):**
- Fields like `name`, `description`, `detail`
- Exempt from `format:`, but must provide a realistic `example:`
- Add `maxLength:` if the database has a column constraint

Bad:
```yaml
id:
  type: string
tm_create:
  type: string
number:
  type: string
```

Good:
```yaml
id:
  type: string
  format: uuid
  example: "550e8400-e29b-41d4-a716-446655440000"
tm_create:
  type: string
  format: date-time
  example: "2026-02-17T12:00:00Z"
number:
  type: string
  pattern: "^\\+[1-9]\\d{1,14}$"
  example: "+821012345678"
  description: "E.164 format. Must start with '+'."
name:
  type: string
  example: "My Campaign"
```

#### Rule 3: Provenance in Descriptions

Every ID field referencing another resource must explicitly state where it comes from.

**Why:** AI agents are stateless and don't know foreign key relationships or execution order.

**Pattern:** `"The [ID Name] returned from the [Endpoint Path] response."`

Bad:
```yaml
activeflow_id:
  type: string
  description: "Activeflow ID"
```

Good:
```yaml
activeflow_id:
  type: string
  format: uuid
  description: "The unique identifier of the active flow. Returned from the `POST /activeflows` response."
```

#### Rule 4: Mandatory Realistic Examples

Every new or modified leaf property must have an `example:` with real-looking data.

**Why:** Examples are the most effective "few-shot prompt" for LLMs to understand data formats.

**Constraints:**
- NEVER use: `"string"`, `"text"`, `null`, `"user_id"`
- ALWAYS use realistic values: `"+821012345678"`, `"active"`, `"550e8400-e29b-41d4-a716-446655440000"`

```yaml
email:
  type: string
  format: email
  example: "support@voipbin.net"
status:
  type: string
  enum: ["queued", "sending"]
  example: "queued"
```

#### Rule 5: Explicit Array Constraints

Arrays that logically require at least one item must enforce it with `minItems`.

**Why:** AI agents often send empty lists, causing silent failures or validation errors.

```yaml
destinations:
  type: array
  items:
    $ref: '#/components/schemas/CommonAddress'
  minItems: 1
  description: "List of target addresses. Must contain at least one destination."
```

### Schema Validation Against Service Models

**CRITICAL: OpenAPI schemas must accurately reflect the public-facing models defined in each service.**

**The Validation Rule:**
When services expose data publicly (via webhooks, API responses, or events), they define this through specific Go structs. The OpenAPI schemas in this repository MUST match those structs field-for-field.

**Example Mapping:**
```
Service Model (what services actually send):
  bin-call-manager/models/call/webhook.go → WebhookMessage struct

OpenAPI Schema (what this repo documents):
  openapi/openapi.yaml → CallManagerCall schema
```

**How to Validate:**

1. **Identify the Service Model:**
   - Look for `WebhookMessage` structs in service `models/` directories
   - Example: `bin-call-manager/models/call/webhook.go`
   - Example: `bin-conference-manager/models/conference/webhook.go`
   - Example: `bin-agent-manager/models/agent/webhook.go`

2. **Locate the Corresponding OpenAPI Schema:**
   - Search `openapi/openapi.yaml` for the schema name
   - Naming pattern: `[ServiceName][ResourceType]`
   - Examples:
     - `CallManagerCall` for call-manager's Call
     - `ConferenceManagerConference` for conference-manager's Conference
     - `AgentManagerAgent` for agent-manager's Agent

3. **Field-by-Field Comparison:**
   ```
   Go Struct Field          →  OpenAPI Schema Field
   ───────────────────────────────────────────────────
   FlowID uuid.UUID         →  flow_id: type: string
   Status Status            →  status: $ref: '#/components/schemas/CallManagerCallStatus'
   ChainedCallIDs []uuid.UUID → chained_call_ids: type: array, items: type: string
   TMCreate string          →  tm_create: type: string
   ```

4. **Check for Missing Fields:**
   - Every field in the Go `WebhookMessage` struct should have a corresponding property in the OpenAPI schema
   - Embedded structs (like `commonidentity.Identity`) should have their fields expanded in the schema

5. **Verify Type Correctness:**
   - `string` types in Go → `type: string` in OpenAPI
   - `uuid.UUID` → `type: string` (with optional `format: uuid`)
   - `[]Type` slices → `type: array` with `items` definition
   - Enums → `$ref` to corresponding enum schema
   - Nested structs → `$ref` to corresponding schema

**Common Validation Scenarios:**

**Adding a New Field to a Service Model:**
```go
// In bin-call-manager/models/call/webhook.go
type WebhookMessage struct {
    // ... existing fields ...
    NewField string `json:"new_field,omitempty"`  // ← New field added
}
```

Must update OpenAPI schema:
```yaml
# In openapi/openapi.yaml under CallManagerCall
properties:
  # ... existing properties ...
  new_field:                    # ← Add this
    type: string
    description: Description of new field
```

**Adding a New Enum Value:**
```go
// In bin-conference-manager/models/conference/status.go
const (
    StatusActive  Status = "active"
    StatusPaused  Status = "paused"   // ← New enum value
    StatusEnded   Status = "ended"
)
```

Must update OpenAPI enum:
```yaml
ConferenceManagerConferenceStatus:
  type: string
  enum:
    - active
    - paused    # ← Add this
    - ended
  x-enum-varnames:
    - ConferenceManagerConferenceStatusActive
    - ConferenceManagerConferenceStatusPaused    # ← Add this
    - ConferenceManagerConferenceStatusEnded
```

**Service Model Locations by Manager:**
- **bin-call-manager**: `models/call/webhook.go`, `models/groupcall/webhook.go`, `models/recording/webhook.go`
- **bin-conference-manager**: `models/conference/webhook.go`, `models/conferencecall/webhook.go`
- **bin-agent-manager**: `models/agent/webhook.go`
- **bin-flow-manager**: `models/flow/webhook.go`, `models/activeflow/webhook.go`
- Pattern: Most services follow `models/<resource>/webhook.go`

**After Making Schema Changes:**
1. Validate OpenAPI spec: `oapi-codegen -config configs/config_model/config.generate.yaml openapi/openapi.yaml > /dev/null`
2. Regenerate models: `go generate ./...`
3. Test in dependent services (especially `bin-api-manager`)
4. Update documentation if public API changes

**Why This Matters:**
- API documentation at https://api.voipbin.net/docs/ must be accurate
- External API consumers depend on correct schema definitions
- Type mismatches cause runtime errors and customer confusion
- Swagger UI generation relies on accurate schemas

### External Documentation

OpenAPI spec includes external documentation links:
```yaml
externalDocs:
  description: Find more about agents
  url: https://api.voipbin.net/docs/agent.html
```

API is documented at: https://api.voipbin.net/docs/

## Dependencies

Key Go modules:
- `github.com/oapi-codegen/runtime` - Runtime types for generated code (Date, UUID, Email, File)
- `github.com/oapi-codegen/oapi-codegen/v2` - Code generator tool
- `github.com/google/uuid` - UUID handling

## Notes

- This repository contains NO runnable service code - only OpenAPI specs and generated types
- The `vendor/` directory is gitignored but can be recreated with `go mod vendor`
- Generated file `gens/models/gen.go` should not be manually edited - always regenerate
- All API endpoints follow REST conventions with standard HTTP verbs (GET, POST, PUT, DELETE)
