# CLAUDE.md and docs/ Template Reference

This document defines the expected structure for each service class.
Use it as the generation template during the docs refresh.

---

## Class A — Standard Go RPC Manager

**Examples:** bin-call-manager, bin-flow-manager, bin-conference-manager

### CLAUDE.md (slim index, ~100-200 lines)

```
# <Service Name>

## Overview
2-3 sentences: what business problem this service solves, its primary resource type,
and how it fits in the CPaaS platform.

## Key Concepts
- **<Entity>**: one-line definition
- **<Entity>**: one-line definition
(5-8 concepts max)

## Common Commands
| Command | Purpose |
|---------|---------|
| `cd bin-<name> && go build ./...` | Compile |
| `go test ./...` | Run tests |
| `golangci-lint run -v` | Lint |
| `go generate ./...` | Regenerate mocks/openapi |

## Architecture
→ [docs/architecture.md](docs/architecture.md)

## Domain / Business Logic
→ [docs/domain.md](docs/domain.md)

## Dependencies
→ [docs/dependencies.md](docs/dependencies.md)

## Operations
→ [docs/operations.md](docs/operations.md)

## CRITICAL Rules
(Any service-specific rules that must stay inline — no length cap)
```

### docs/architecture.md

**Required H2 headings (must all be present):**
- `## Component Overview`
- `## Layer Responsibilities`
- `## Request Routing`

**Content per section:**
- **Component Overview**: Mermaid diagram showing main packages and their relationships.
  `cmd/` → `pkg/<domain>handler`, `pkg/listenhandler`, `pkg/subscribehandler`, `pkg/dbhandler`, `pkg/cachehandler`
- **Layer Responsibilities**: Table with columns: Package | Role | Key Types
- **Request Routing**: Table from extractor routing_table. Columns: Route Pattern | Handler Function | Description. Must list every route — no fabrication.

### docs/domain.md

**Required H2 headings (must all be present):**
- `## Domain Entities`
- `## Key Business Rules`

**Optional H2 headings:**
- `## State Machines` (include when service manages stateful resources)

**Content per section:**
- **Domain Entities**: One subsection (###) per main entity. Fields from models/ (but refer to RST docs for full field lists — don't restate them here).
- **Key Business Rules**: Numbered list of invariants and constraints that enforce correct behavior. Source from domain logic in pkg/<domain>handler/.
- **State Machines**: Mermaid stateDiagram-v2 for each stateful resource lifecycle.

### docs/dependencies.md

**Generated deterministically from extractor JSON using the template at `docs/reference/dependencies.md.tmpl`.**
**Never hand-written or LLM-generated.**

### docs/operations.md

**Required H2 headings (must all be present):**
- `## Common Failure Modes`
- `## Debugging Guide`
- `## Configuration`
- `## Prometheus Metrics`

**Content per section:**
- **Common Failure Modes**: Table. Columns: Symptom | Likely Cause | Resolution. Minimum 4 rows.
- **Debugging Guide**: Log grep patterns, key log messages to look for, how to trace a request.
- **Configuration**: Table from extractor config_vars. Columns: Flag | Env Var | Default | Description.
- **Prometheus Metrics**: Table from extractor metrics. Columns: Metric Name | Type | Description.

---

## Class A2 — Event-Driven Worker (no inbound RPC)

**Examples:** bin-sentinel-manager, bin-hook-manager

Same structure as Class A **except:**
- `docs/architecture.md`: Replace `## Request Routing` with `## Execution Model`
  - **Execution Model**: What triggers this service (events, scheduled polling, webhooks), what it does when triggered, what it produces/emits.
- No routing table (this service has no listenhandler)

---

## Class A+sub — Go RPC + Embedded Native Daemon

**Examples:** voip-asterisk-proxy, voip-kamailio-proxy, voip-rtpengine-proxy

Same structure as Class A **plus:**
- `docs/subsystems.md` (REQUIRED for all A+sub services)

### docs/subsystems.md

**Required H2 headings:**
- `## Native Daemon Overview`
- `## Configuration`
- `## Deployment Notes`

**Content:**
- **Native Daemon Overview**: What the embedded daemon is (Asterisk/Kamailio/RTPEngine), version, how the Go service manages its lifecycle (subprocess, sidecar, etc.)
- **Configuration**: Config files, environment variables, and ports the daemon needs.
- **Deployment Notes**: Docker setup, dependency on the native daemon being present, startup order.

---

## Class B — HTTP/REST API Gateway

**Examples:** bin-api-manager

### docs/ structure (differs from Class A):
- `docs/architecture.md` — Required H2: `## Component Overview`, `## Middleware Stack`, `## Backend Services`
- `docs/routing.md` — Maps every REST endpoint to its backend RPC service. Required H2: one per domain group (Auth, Calls, Flows, Agents, Billing, Numbers, etc.)
- `docs/auth.md` — JWT validation, customer scoping, permission model. Required H2: `## Authentication Flow`, `## Authorization Model`
- `docs/operations.md` — Same as Class A

### CLAUDE.md

Same slim-index format as Class A. Keep any RST sync CRITICAL rule inline.

---

## Class C — Shared Library

**Examples:** bin-common-handler

### docs/ structure:
- `docs/architecture.md` — Required H2: `## Package Overview`, `## Package Responsibilities`
- `docs/usage.md` — How consuming services use this library, import patterns, gotchas. Required H2: `## Import Guidelines`, `## Common Patterns`

### CLAUDE.md

Keep the bin-common-handler admission rule inline (CRITICAL — 3+ services requirement).

---

## Class D — Python/Alembic Migrations Manager

**Examples:** bin-dbscheme-manager

### docs/ structure:
- `docs/migrations.md` — How to create migrations. Required H2: `## Creating a Migration`, `## Naming Conventions`, `## Example Migration`
- `docs/schema-ownership.md` — Table: table name → owning service. Required H2: `## Table Ownership`
- `docs/operations.md` — Environment access matrix. Required H2: `## Environment Access`, `## Emergency Rollback`, `## Common Failures`

### CLAUDE.md

**CRITICAL rules must stay inline (no length cap):**
- Alembic upgrade/downgrade prohibition
- Manual revision ID prohibition
- Human-authorization requirement for schema changes

---

## Class E — OpenAPI Codegen (no runtime)

**Examples:** bin-openapi-manager

### docs/ structure:
- `docs/architecture.md` — Required H2: `## Codegen Pipeline`, `## Output Artifacts`
- `docs/usage.md` — How consuming services use the generated types. Required H2: `## Consuming Generated Types`, `## Regeneration`

### CLAUDE.md

Brief. Link to docs/architecture.md and docs/usage.md. Note that this service has no runtime — it only generates code.

---

## Consistency Guard — Required H2 Headings

This is the machine-checkable minimum for each file type. Every file MUST contain all headings for its class.

### architecture.md
- Class A: `## Component Overview`, `## Layer Responsibilities`, `## Request Routing`
- Class A2: `## Component Overview`, `## Layer Responsibilities`, `## Execution Model`
- Class A+sub: same as Class A plus `## Subsystem Overview` (in subsystems.md, not architecture.md)
- Class B: `## Component Overview`, `## Middleware Stack`, `## Backend Services`
- Class C: `## Package Overview`, `## Package Responsibilities`
- Class D: (no architecture.md)
- Class E: `## Codegen Pipeline`, `## Output Artifacts`

### domain.md
- Class A, A2, A+sub: `## Domain Entities`, `## Key Business Rules`
- Class B, C, D, E: (no domain.md)

### operations.md
- Class A, A2, A+sub, B: `## Common Failure Modes`, `## Debugging Guide`, `## Configuration`, `## Prometheus Metrics`
- Class C, D (class-specific ops): see class-specific definitions above
- Class E: (no operations.md)
