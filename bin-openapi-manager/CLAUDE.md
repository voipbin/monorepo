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
