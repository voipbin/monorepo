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

**CRITICAL: Run verification BEFORE every commit, not just once per branch.**

"Changed" means compared to the `main` branch, NOT compared to your previous commit. If your branch has modified bin-common-handler at any point (even in earlier commits), you MUST run the full verification for ALL services before EVERY subsequent commit.

**If you changed bin-common-handler (compared to main):** Run the full verification workflow for ALL 30+ services.

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

### Git Worktrees

**CRITICAL: MANDATORY RULE - NO EXCEPTIONS**

**NEVER edit, create, or modify ANY files directly in the main repository (`~/gitvoipbin/monorepo`).**

This includes:
- Code changes
- Config file changes (including `.circleci/`)
- Documentation changes (including `docs/`)
- Design documents
- ANY file modifications

**BEFORE making any changes, you MUST:**

1. **Check current directory:**
   ```bash
   pwd
   # If output is ~/gitvoipbin/monorepo → STOP, create worktree first
   # If output is ~/gitvoipbin/monorepo-worktrees/<branch-name> → OK to proceed
   ```

2. **Create a worktree (if not already in one):**
   ```bash
   cd ~/gitvoipbin/monorepo
   git worktree add ~/gitvoipbin/monorepo-worktrees/NOJIRA-feature-name -b NOJIRA-feature-name
   cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-feature-name
   ```

3. **Work in the worktree:**
   - All file edits, document creation, and commits happen here
   - Main repository stays clean on `main` branch

4. **When done, remove the worktree:**
   ```bash
   cd ~/gitvoipbin/monorepo
   git worktree remove ~/gitvoipbin/monorepo-worktrees/NOJIRA-feature-name
   ```

**Worktree location:** `~/gitvoipbin/monorepo-worktrees/`

**Why worktrees:**
- Keeps main repository (`~/gitvoipbin/monorepo`) clean and always on `main` branch
- Allows parallel work on multiple features
- Easy to abandon work without affecting main workspace
- Clear separation between exploration and implementation

**If you accidentally edited files in main repository:**
1. Do NOT commit
2. Stash or discard changes: `git stash` or `git checkout .`
3. Create worktree and apply changes there

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

**CRITICAL: Before creating a PR or merging, ALWAYS pull the latest `main` and check for conflicts.**

This is mandatory — no exceptions. Follow this sequence:
1. **Fetch latest main:** `git fetch origin main`
2. **Check for conflicts:** `git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"`
3. **Review what changed on main:** `git log --oneline HEAD..origin/main`
4. **If conflicts exist:** Rebase or merge main into your branch, resolve conflicts, and re-run the full verification workflow before proceeding.
5. **If no conflicts:** Proceed with PR creation or merge.

**For detailed branch strategies and merge workflows, see [git-workflow-guide.md](docs/git-workflow-guide.md)**

### Design & Implementation Workflow

**CRITICAL: For non-trivial changes, ALWAYS write a design document before implementing.**

This applies to: new features, architectural changes, multi-file modifications, and any work that benefits from upfront planning. Simple fixes (typos, single-line bug fixes) do not require a design document.

**Workflow:**

1. **Brainstorm and validate the design verbally** — Understand the current state, ask clarifying questions, and agree on the approach with the user.
2. **Create a worktree** — All files (design docs and code) are written in the worktree, never in the main repository.
   ```bash
   cd ~/gitvoipbin/monorepo
   git worktree add ~/gitvoipbin/monorepo-worktrees/NOJIRA-feature-name -b NOJIRA-feature-name
   cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-feature-name
   ```
3. **Write the design document** in the worktree:
   - Location: `docs/plans/YYYY-MM-DD-<topic>-design.md`
   - Include: problem statement, approach, files to change, and any trade-offs discussed
4. **Implement the changes** in the same worktree.
5. **Run verification workflow** before committing.
6. **Commit everything together** (design doc + implementation), push, and create PR.

**Design document location:** `docs/plans/YYYY-MM-DD-<topic>-design.md` (always in the worktree)

**Why design docs:**
- Serves as a reference during implementation
- Provides context for future developers
- Documents decisions and trade-offs
- Lives in the repo alongside the code it describes

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

### bin-common-handler Admission Rule

**CRITICAL: A package may only live in `bin-common-handler` if it is used by 3 or more services.**

- Single-consumer or dual-consumer packages belong in the consuming service(s).
- If a package's usage later grows to 3+ services, it can be promoted to `bin-common-handler`.
- Internal plumbing packages (e.g., `rabbitmqhandler` wrapped by `sockhandler`) are exempt since they serve the shared library itself.

**Why:** `bin-common-handler` is a globally shared library — every change triggers verification across 30+ services. Keeping it lean reduces blast radius and maintenance burden.

### Database Handling Rules

**CRITICAL: All database operations MUST follow these conventions. No exceptions.**

#### 1. Model Structs with `db:` Tags (MANDATORY)
Every database table MUST have a corresponding Go model struct with `db:` tags.
- Location: `models/<entity>/<entity>.go`
- UUID fields MUST use `db:"column_name,uuid"` tag
- JSON fields MUST use `db:"column_name,json"` tag
- NO loose variables for scanning rows (`var id []byte` is FORBIDDEN)

#### 2. Squirrel Query Builder (MANDATORY)
All SQL queries MUST be built using the squirrel query builder (`sq.Select`, `sq.Insert`, `sq.Update`, `sq.Delete`).
- Raw SQL strings are FORBIDDEN except for computed expressions that squirrel cannot express (e.g., `cost_per_unit * ?`)
- String concatenation for table/column names is FORBIDDEN
- When raw SQL is unavoidable, document WHY with a comment

#### 3. commondatabasehandler Utilities (MANDATORY)
- **INSERT**: Use `commondatabasehandler.PrepareFields(struct)` → `sq.Insert(table).SetMap(fields)`
- **SELECT**: Use `commondatabasehandler.GetDBFields(struct)` → `sq.Select(cols...)` → `commondatabasehandler.ScanRow(rows, &struct)`
- **UPDATE**: Use `sq.Update(table).SetMap(preparedFields).Where(...)`
- Manual `rows.Scan(&var1, &var2, ...)` is FORBIDDEN — always use `ScanRow`

#### 4. DB Operations in `dbhandler/` Package Only (MANDATORY)
- All database operations MUST live in `pkg/dbhandler/<entity>.go`
- All methods MUST be part of the `DBHandler` interface in `pkg/dbhandler/main.go`
- NO direct `*sql.DB` access outside of `dbhandler/` — handlers receive `DBHandler` interface only
- Business logic packages (billinghandler, accounthandler, etc.) MUST NOT import `database/sql`

#### 5. Field Type for Updates (MANDATORY)
- Each model MUST define a `Field` type in `models/<entity>/field.go`
- Update methods accept `map[<entity>.Field]any` for type-safe field updates

### Debug Logging for Retrieved Data

**CRITICAL: Always add debug logs when retrieving data from other services or databases.**

When fetching data (calls, channels, agents, customers, etc.), add a debug log immediately after successful retrieval. This helps trace request flow and debug issues in production.

**Pattern:**
```go
// After retrieving data, log it with the object and key identifier
call, err := h.callGet(ctx, callID)
if err != nil {
    log.Infof("Could not get call: %v", err)
    return nil, fmt.Errorf("call not found")
}
log.WithField("call", call).Debugf("Retrieved call info. call_id: %s", call.ID)

ch, err := h.reqHandler.CallV1ChannelGet(ctx, call.ChannelID)
if err != nil {
    log.Errorf("Could not get channel: %v", err)
    return nil, fmt.Errorf("no data available")
}
log.WithField("channel", ch).Debugf("Retrieved channel info. channel_id: %s", ch.ID)
```

**Guidelines:**
- Use `Debug` level (not Info) to avoid log spam in production
- Include the full object in `WithField` for detailed inspection
- Include the key identifier in the message for quick scanning
- Add logs after EVERY successful data retrieval, not just errors

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

#### Model/Struct Changes Require OpenAPI Updates

**CRITICAL: Most API-facing structs are tightly coupled with OpenAPI specs. When changing struct definitions, you MUST also update the corresponding OpenAPI schema.**

This applies to any struct that:
- Is returned by API endpoints (response models)
- Is accepted by API endpoints (request models)
- Is embedded in other API-facing structs

**When modifying struct fields:**
1. **Update the Go struct** in the service's `models/` directory
2. **Update the OpenAPI schema** in `bin-openapi-manager/openapi/openapi.yaml`
3. **Regenerate OpenAPI types** with `go generate ./...` in `bin-openapi-manager`
4. **Regenerate API server code** with `go generate ./...` in `bin-api-manager` (it generates server code FROM the openapi.yaml)
5. **Run verification** for the service, `bin-openapi-manager`, AND `bin-api-manager`

**Why api-manager regeneration is required:**
`bin-api-manager` uses `go generate` to create server code directly from `bin-openapi-manager/openapi/openapi.yaml`. The generated code lives in `bin-api-manager/gens/openapi_server/gen.go`. If you update the OpenAPI spec but don't regenerate api-manager, the API server will use stale types.

**Example:**
```go
// Changing this in bin-talk-manager/models/message/message.go:
type Media struct {
    AgentID uuid.UUID `json:"agent_id,omitempty"`  // Changed from Agent amagent.Agent
}

// Requires updating bin-openapi-manager/openapi/openapi.yaml:
TalkManagerMedia:
  properties:
    agent_id:           # Changed from agent: $ref AgentManagerAgent
      type: string
      format: uuid

// Then regenerate BOTH:
// cd bin-openapi-manager && go generate ./...
// cd bin-api-manager && go generate ./...
```

**Why this matters:**
- Clients (web apps, mobile apps) depend on the OpenAPI spec for type generation
- The API server uses generated types from the spec
- Mismatched specs cause runtime errors or silent data loss
- API documentation becomes incorrect

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
