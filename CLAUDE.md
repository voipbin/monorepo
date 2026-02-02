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

**This runs AFTER making changes but BEFORE `git commit`.**

### Dependency Update Workflow

**Only when specifically updating dependencies:**

```bash
cd bin-<service-name>

# Run the full update workflow (WITH dependency updates)
go get -u ./... && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

### Special Cases

#### When to Run Verification and Where

**If you changed bin-common-handler:** Run the full verification workflow for ALL 30+ services.

Since all services depend on bin-common-handler, ANY change to it can affect every service. Dependencies propagate through the monorepo and services may need go.mod/go.sum updates even if you didn't touch their code directly.

```bash
# Run full verification for ALL services after bin-common-handler changes
for dir in bin-*/; do
  if [ -f "$dir/go.mod" ]; then
    echo "=== $dir ===" && \
    (cd "$dir" && \
      go mod tidy && \
      go mod vendor && \
      go generate ./... && \
      go test ./... && \
      golangci-lint run -v --timeout 5m) || echo "FAILED: $dir"
  fi
done
```

**If you changed specific services (NOT bin-common-handler):** Run the full verification workflow only for those services.

```bash
# Run full verification only for the changed service
cd bin-<service-name>
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

#### Full Verification Workflow Steps

The full verification workflow consists of 5 steps that MUST all be run:

| Step | Command | Purpose |
|------|---------|---------|
| 1 | `go mod tidy` | Sync go.mod/go.sum with imports, remove unused deps, add missing ones |
| 2 | `go mod vendor` | Copy dependencies to vendor/ for reproducible builds |
| 3 | `go generate ./...` | Run code generators (mocks, OpenAPI types, etc.) |
| 4 | `go test ./...` | Run all unit tests |
| 5 | `golangci-lint run -v --timeout 5m` | Run static analysis and linting |

**Do NOT skip any steps.** Each step can catch different issues.

**Changes to public-facing models:** Update OpenAPI schemas in bin-openapi-manager.

**For complete workflows, troubleshooting, and detailed explanations, see:**
- [verification-workflows.md](docs/verification-workflows.md) - Complete verification details
- [special-cases.md](docs/special-cases.md) - bin-common-handler updates, OpenAPI sync

## Git Workflow

### Commit Message Format

**CRITICAL: Commit title MUST match the branch name exactly.**

**Title (first line):**
```
VOIP-[ticket-number]-brief-description-of-change
```
or (when no JIRA ticket)
```
NOJIRA-brief-description-of-change
```

**Body (subsequent lines):**
List each affected project with specific changes:
```
- bin-common-handler: Fixed type handling in database mapper
- bin-flow-manager: Updated flow execution to use new types
- bin-call-manager: Refactored call handler to support new interface
```

**Complete Example:**
```
NOJIRA-add-claude-md-reorganization-design

Created comprehensive design document to reorganize CLAUDE.md documentation structure
across the monorepo.

- docs: Define clear boundaries between root and service-specific CLAUDE.md files
- docs: Establish decision framework for where to document new information
- docs: Plan content migration from root to service-specific files
```

**Rules:**
1. Always list affected projects
2. Be specific about what changed in each project
3. Keep title concise
4. Use present tense
5. Use dashes (`-`) for bullet points
6. Add narrative summary for significant changes
7. Do NOT include "Co-Authored-By" lines in commit messages or PR descriptions

**For extended examples, branch management, and merge rules, see [git-workflow-guide.md](docs/git-workflow-guide.md)**

### Branch Management

**CRITICAL: Before making ANY changes or commits, ALWAYS check the current branch first.**

**If the current branch is `main`:**
1. **STOP - DO NOT make commits on main**
2. Ask the user to create a feature branch first
3. Suggest a branch name: `NOJIRA-brief-description` or `VOIP-1234-brief-description`
4. Wait for user confirmation before proceeding

**Correct Workflow:**
```bash
# Step 1: ALWAYS check current branch BEFORE making any changes
git branch --show-current

# Step 2: If on main, create feature branch BEFORE any edits
git checkout -b NOJIRA-descriptive-change-summary

# Step 3: Make your code changes

# Step 4: Run verification workflow BEFORE committing
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# Step 5: Commit changes (title matches branch name)
git add .
git commit -m "NOJIRA-descriptive-change-summary

- project-name: What changed"

# Step 6: Push to remote
git push -u origin NOJIRA-descriptive-change-summary
```

**NEVER commit directly to `main` without explicit user permission.**

**NEVER merge any branch to `main` without explicit user permission.**

**For detailed branch strategies and merge workflows, see [git-workflow-guide.md](docs/git-workflow-guide.md)**

## Build & Development

**For complete build commands, testing patterns, code generation, and linting, see [development-guide.md](docs/development-guide.md)**

Quick references:
- Build: `cd bin-<service-name> && go build ./cmd/...`
- Test: `go test -v ./...`
- Generate: `go generate ./...`
- Lint: `golangci-lint run -v --timeout 5m`

## Database Migrations with Alembic

**CRITICAL: Database schema changes ONLY through Alembic migrations in `bin-dbscheme-manager`.**

**What AI CAN do:**
- ✅ Create migration files (`alembic -c alembic.ini revision -m "..."`)
- ✅ Edit migration files to add SQL in upgrade()/downgrade() functions
- ✅ Commit migration files to git

**What AI MUST NEVER do:**
- 🚫 Run `alembic upgrade` (applies migrations to database)
- 🚫 Run `alembic downgrade` (rolls back database changes)
- 🚫 Execute any SQL that modifies database schema

**Why:** Database changes are irreversible and require human authorization, testing, and VPN access.

**For complete migration workflows, patterns, and troubleshooting, see [common-workflows.md#database-migrations-with-alembic](docs/common-workflows.md#database-migrations-with-alembic)**

## Architecture

**For service categories, inter-service communication, configuration, and deployment, see [architecture-deep-dive.md](docs/architecture-deep-dive.md)**

Key points:
- 30+ Go microservices for VoIP, messaging, AI, and communication workflows
- RabbitMQ RPC for inter-service communication (not HTTP)
- Shared MySQL database, Redis cache, RabbitMQ broker
- Kubernetes deployment on GCP GKE

## API Design Principles

### Atomic API Responses

**CRITICAL: All API endpoints MUST return atomic data - single resource types without combining data from other services.**

**The Rule:**
API responses should contain ONLY the requested resource type, without including related data from other services or resources. Clients must make separate requests if they need related information.

**Why:**
- Maintains clear service boundaries
- Keeps APIs simple and predictable
- Prevents tight coupling between services

**Examples:**

✅ **CORRECT - Atomic Response:**
```
GET /v1/billings/{billing-id}
Returns: BillingManagerBilling (just the billing record)
```

❌ **WRONG - Combined Response:**
```
GET /v1/billings/{billing-id}
Returns: Billing + Account + Reference Resource (Don't include related resources)
```

**For exceptions, patterns, and authentication details, see [reference.md#api-design-principles](docs/reference.md#api-design-principles)**

## Common Workflows

**For complete workflows and examples, see [common-workflows.md](docs/common-workflows.md)**

Topics covered:
- Adding a New API Endpoint
- Creating a New Flow Action
- Adding a New Manager Service
- Modifying Shared Models
- Parsing Filters from Request Body
- Database Migrations with Alembic

## Code Quality

Follow these standards:
- Generate mocks after interface changes (`go generate ./...`)
- Write table-driven tests
- Use function-scoped logging pattern
- Follow Go naming conventions (List not Gets)
- Handle errors properly

**For detailed standards, logging examples, and naming conventions, see [code-quality-standards.md](docs/code-quality-standards.md)**

### Common Gotchas

#### Updating Shared Library Function Signatures

**CRITICAL: When updating function signatures in bin-common-handler, you MUST account for ALL call patterns across the monorepo.**

Services use different import aliases and call formats:
```go
// Some services use single-line with one alias
notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, queueName, serviceName, "")

// Other services use multi-line with different alias
notifyHandler := commonnotify.NewNotifyHandler(
    sockHandler,
    reqHandler,
    queueName,
    serviceName,
)
```

**When updating function signatures:**
1. **Search for ALL import aliases** - Use `grep -r "notifyhandler\|commonnotify" --include="*.go"` to find all aliases
2. **Check for multi-line calls** - Simple sed patterns only match single-line; multi-line calls will be missed
3. **Run full verification for ALL services** - Don't rely on pattern matching; build errors will catch missed updates
4. **Prefer manual verification** - After bulk updates, manually review files with different patterns

**Example failure mode:**
```bash
# This sed command ONLY matches single-line patterns:
sed -i 's/notifyhandler.NewNotifyHandler(\([^)]*\))/notifyhandler.NewNotifyHandler(\1, "")/g'

# It MISSES multi-line patterns like:
commonnotify.NewNotifyHandler(
    sockHandler,
    reqHandler,
    queueName,
    serviceName,  # Missing 5th param!
)
```

**Safe approach:** After any bin-common-handler signature change, run the full verification workflow on ALL 30+ services to catch any missed updates.

#### UUID Fields and DB Tags

**CRITICAL: UUID fields MUST use the `,uuid` db tag for proper type conversion.**

```go
// ✅ CORRECT - UUID field with uuid tag
type Model struct {
    ID         uuid.UUID `db:"id,uuid"`
    CustomerID uuid.UUID `db:"customer_id,uuid"`
}

// ❌ WRONG - Missing uuid tag
type Model struct {
    ID         uuid.UUID `db:"id"`  // Will cause conversion issues
}
```

**For complete gotcha explanations and troubleshooting, see [code-quality-standards.md#common-gotchas](docs/code-quality-standards.md#common-gotchas)**

## Where to Document New Information

Use this decision tree when adding new documentation:

**Does this apply to ALL services in the monorepo?**
- **YES** → Add to Root CLAUDE.md
  - Examples:
    - New verification step required for all services
    - New git workflow rule
    - New shared pattern from bin-common-handler
    - Changes to RabbitMQ communication pattern

**NO → Does it affect 2+ services?**
  - **YES → Is it a workflow spanning services?**
    - **YES** → Add to Root CLAUDE.md "Common Workflows"
      - Example: "Adding a New API Endpoint" (touches openapi-manager + api-manager + target service)
    - **NO → Is it a shared pattern/gotcha?**
      - **YES** → Add to Root CLAUDE.md "Shared Patterns" with note
        - Example: "Parsing Filters from Request Body"
        - **Note format:** Explain what the pattern is for, which services use it, when to apply it
      - **NO** → Add to each affected service's CLAUDE.md
        - Example: How conference-manager and call-manager handle recordings

  - **NO** → Add to Service-Specific CLAUDE.md
    - Examples:
      - Service-specific API endpoints
      - Handler patterns unique to this service
      - Service-specific testing approaches
      - Domain-specific implementation details

## Reference

**For dependencies, deployment info, security considerations, and resources, see [reference.md](docs/reference.md)**

Quick links:
- Admin Console: https://admin.voipbin.net/
- Agent Interface: https://talk.voipbin.net/
- API Documentation: https://api.voipbin.net/docs/
- Project Site: http://voipbin.net/
