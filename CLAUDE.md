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
   go clean -testcache && \
   go test ./..." \;
```

**What this does for EACH service:**
1. `go mod tidy` - Syncs go.mod with bin-common-handler changes (NOT `go get -u`)
2. `go mod vendor` - Updates vendored bin-common-handler code
3. `go generate ./...` - Regenerates mocks that depend on bin-common-handler interfaces
4. **`go clean -testcache`** - **CRITICAL**: Clears test cache to force actual test execution
5. `go test ./...` - Verifies the service still works with new bin-common-handler

**⚠️ WARNING: ALWAYS use `go clean -testcache` when testing after bin-common-handler changes!**
- Go caches test results, which can hide failures introduced by your changes
- Cached tests show "PASS" even though they never actually re-ran with the new code
- This is the #1 cause of missing test failures in the monorepo

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
- **Type mismatches**: Shared utility changes affect test expectations, not just interfaces
- **Test cache masking failures**: Cached tests can hide breaking changes
- **Inconsistent state**: Half-updated monorepo leads to subtle bugs

#### Verification Checklist

After changing bin-common-handler, verify EACH dependent service:

- [ ] **`go mod tidy`** - Dependencies synced
- [ ] **`go mod vendor`** - Vendored code updated
- [ ] **`go generate ./...`** - Mocks regenerated
- [ ] **`go clean -testcache`** - Test cache cleared ⚠️ NEVER SKIP THIS
- [ ] **`go test ./...`** - All tests pass (actually ran, not cached)
- [ ] **Test expectations updated** - If types/values changed in shared utilities
- [ ] **`golangci-lint run`** - No linting issues
- [ ] **`go build ./...`** - Compiles successfully

**If ANY test fails:**
1. Don't just regenerate mocks and retry
2. Investigate WHAT changed in bin-common-handler
3. Update test EXPECTATIONS to match new behavior
4. Clear cache and re-run: `go clean -testcache && go test ./...`

#### Common Mistakes to Avoid

❌ **DON'T run tests without clearing cache first** - `go test` uses cached results, hiding failures
❌ **DON'T just regenerate mocks and assume tests will pass** - Test EXPECTATIONS need updates for type changes
❌ **DON'T run `go get -u`** - This updates ALL dependencies, not just bin-common-handler
❌ **DON'T skip `go generate`** - Mocks will be stale if interfaces changed
❌ **DON'T skip `go test`** - Won't catch breaking changes until production
❌ **DON'T commit bin-common-handler alone** - All services must be updated together

✅ **DO run `go clean -testcache` before testing** - Force actual test execution, not cached results
✅ **DO update test expectations when shared utilities change types** - Especially for type conversions
✅ **DO run the complete workflow** - `go mod tidy && go mod vendor && go generate && go clean -testcache && go test`
✅ **DO verify tests ACTUALLY ran** (not cached) for all services before committing
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

**If tests fail in multiple services after bin-common-handler changes:**

🚨 **CRITICAL: Test failures are NOT just about regenerating mocks!**

When bin-common-handler shared utilities change (especially type conversions, data transformations, or filter handling), test EXPECTATIONS must be updated to match the new behavior.

**Common scenario (like the UUID filter bug):**
1. You modify a shared utility in bin-common-handler (e.g., database field type conversion)
2. This changes the TYPES of values passed to mocked functions (e.g., string → uuid.UUID)
3. Mock expectations in tests still expect the OLD types
4. Tests fail with "expected call doesn't match" errors
5. **Solution**: Update test expectations to use the new types, NOT just regenerate mocks

**Example - What to fix when type conversions change:**
```go
// ❌ WRONG - Old test expectation (before fix)
expectFilters: map[Field]any{
    FieldCustomerID: "5e4a0680-804e-11ec-8477-2fea5968d85b",  // String
}

// ✅ CORRECT - Updated test expectation (after UUID conversion fix)
expectFilters: map[Field]any{
    FieldCustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),  // UUID type
}

// Alternative: Use gomock.Any() if exact matching is problematic
mockReq.EXPECT().SomeMethod(ctx, gomock.Any()).Return(response, nil)
```

**Systematic approach to fixing test failures:**
1. **Identify the changed behavior** in bin-common-handler
2. **Find all test files** with failing expectations (`grep -r "FieldCustomerID" *_test.go`)
3. **Update test expectations** to match new types/behavior
4. **Re-run with clean cache**: `go clean -testcache && go test ./...`
5. **Verify actual execution**: Check test ran (not cached)

**Common changes requiring test expectation updates:**
- ✅ Type conversions (string → UUID, string → int, etc.)
- ✅ Filter map value types
- ✅ Field name changes or additions
- ✅ Enum value changes
- ✅ Default value changes

**DON'T just regenerate mocks and assume tests will pass!**
- Mock interfaces may be unchanged, but VALUES passed have changed types
- Test expectations must be manually updated to match new value types

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

## Database Migrations with Alembic

### Overview

Database schema changes for VoIPbin services are managed through Alembic migrations in the `bin-dbscheme-manager` service. This is the ONLY way to modify database schemas - never run manual SQL DDL statements against production databases.

### When to Create Migrations

Create an Alembic migration whenever you:
- Add new columns to existing tables
- Create new tables
- Modify column types or constraints
- Add/remove indexes
- Change foreign key relationships

**Common scenarios:**
- Adding fields to a model (e.g., adding `owner_type` column to `storage_files` table)
- Creating tables for new features
- Database refactoring to support new functionality

### Migration Workflow

**Step 1: Navigate to bin-dbscheme-manager**
```bash
cd bin-dbscheme-manager/bin-manager
```

**Step 2: Configure Alembic (first time only)**
```bash
# Copy sample config
cp alembic.ini.sample alembic.ini

# Edit alembic.ini and set database connection
# sqlalchemy.url = mysql://user:pass@host/voipbin
```

**Step 3: Create migration file**
```bash
# Use descriptive naming: <table>_<action>_<type>_<items>
alembic -c alembic.ini revision -m "storage_files add column owner_type"
```

This creates a new file in `main/versions/` with format: `<hash>_storage_files_add_column_owner_type.py`

**Step 4: Edit migration file**

Edit the generated file to add your SQL changes:
```python
def upgrade():
    op.execute("""
        ALTER TABLE storage_files
        ADD COLUMN owner_type VARCHAR(255) DEFAULT '' AFTER owner_id
    """)

def downgrade():
    op.execute("""
        ALTER TABLE storage_files
        DROP COLUMN owner_type
    """)
```

**Step 5: Test migration locally (recommended)**
```bash
# Apply migration to local database
alembic -c alembic.ini upgrade head

# Verify schema change
mysql -u user -p voipbin -e "DESCRIBE storage_files;"

# Test rollback (optional)
alembic -c alembic.ini downgrade -1
alembic -c alembic.ini upgrade head
```

**Step 6: Commit and push**
```bash
cd /home/pchero/gitvoipbin/monorepo
git add bin-dbscheme-manager/bin-manager/main/versions/<hash>_*.py
git commit -m "feat(dbscheme): add owner_type column to storage_files table"
git push
```

**Step 7: Apply to staging/production**

Connect to VPN first (REQUIRED), then apply migration:
```bash
cd bin-dbscheme-manager/bin-manager
alembic -c alembic.ini upgrade head
```

### Migration Best Practices

1. **Always add rollback logic** - Implement both `upgrade()` and `downgrade()` functions
2. **Use raw SQL** - Migrations use `op.execute()` with raw SQL, not SQLAlchemy DDL
3. **Test locally first** - Never run untested migrations against staging/production
4. **One change per migration** - Keep migrations focused and atomic
5. **Descriptive names** - Follow naming convention: `<table>_<action>_<type>_<items>`
6. **Coordinate with code** - Deploy code changes that depend on schema changes AFTER migration runs

### Common Migration Patterns

**Add column:**
```python
def upgrade():
    op.execute("""
        ALTER TABLE <table_name>
        ADD COLUMN <column_name> <type> DEFAULT <default_value> AFTER <existing_column>
    """)

def downgrade():
    op.execute("""
        ALTER TABLE <table_name>
        DROP COLUMN <column_name>
    """)
```

**Create table:**
```python
def upgrade():
    op.execute("""
        CREATE TABLE <table_name> (
            id BINARY(16) NOT NULL PRIMARY KEY,
            customer_id BINARY(16) NOT NULL,
            name VARCHAR(255) NOT NULL,
            tm_create DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
            INDEX idx_customer_id (customer_id)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    """)

def downgrade():
    op.execute("""DROP TABLE <table_name>""")
```

### Troubleshooting

**Issue: "Target database is not up to date"**
```bash
# Check current migration version
alembic -c alembic.ini current

# View migration history
alembic -c alembic.ini history

# Apply missing migrations
alembic -c alembic.ini upgrade head
```

**Issue: Migration fails to apply**
1. Check database connection in `alembic.ini`
2. Verify VPN connection for remote databases
3. Check SQL syntax in migration file
4. Review error messages for constraint violations

**Issue: Need to rollback migration**
```bash
# Rollback last migration
alembic -c alembic.ini downgrade -1

# Rollback to specific revision
alembic -c alembic.ini downgrade <revision_id>
```

### Important Notes

- VPN connection is MANDATORY for staging/production migrations
- Never modify applied migrations - create new ones instead
- Coordinate schema changes with dependent services
- Test migrations against realistic data volumes
- Document breaking changes in migration docstrings
- See `bin-dbscheme-manager/CLAUDE.md` for service-specific details

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

### Parsing Filters from Request Body

**Pattern:** All list endpoints parse filters from request body (JSON), not URL query parameters.

This pattern was implemented as part of the `commondatabasehandler` refactoring. See `docs/plans/2026-01-14-listenhandler-filter-parsing-implementation-plan.md` for complete implementation details.

**How it works:**

**Step 1: Define FieldStruct in model package**

Create or update `models/<resource>/filters.go`:
```go
package <resource>

import "github.com/gofrs/uuid"

type FieldStruct struct {
    CustomerID uuid.UUID `filter:"customer_id"`
    Deleted    bool      `filter:"deleted"`
    Name       string    `filter:"name"`
    // ... other filterable fields
}
```

**Step 2: Parse filters in listenhandler**

In `pkg/listenhandler/v1_<resource>s.go`:
```go
func (h *listenHandler) processV1<Resource>sGet(ctx context.Context, m sock.Request) (sock.Response, error) {
    // Parse pagination from URL (unchanged)
    u, err := url.Parse(m.URI)
    // ...

    // Parse filters from request body (NEW)
    tmpFilters, err := h.utilHandler.ParseFiltersFromRequestBody(m.Data)
    if err != nil {
        log.Errorf("Could not parse filters. err: %v", err)
        return simpleResponse(400), nil
    }

    // Convert to typed filters using FieldStruct
    typedFilters, err := h.utilHandler.ConvertFilters[<resource>.FieldStruct, <resource>.Field](
        <resource>.FieldStruct{},
        tmpFilters,
    )
    if err != nil {
        log.Errorf("Could not convert filters. err: %v", err)
        return simpleResponse(400), nil
    }

    // Use typedFilters with dbhandler
    items, err := h.<resource>Handler.<Resource>GetAll(ctx, typedFilters, pageOpts)
    // ...
}
```

**Step 3: Make requests with body filters**

External API calls (via `bin-api-manager`):
```bash
curl -X GET https://api.voipbin.net/v1/conversations \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
    "deleted": false,
    "limit": 50
  }'
```

Internal RPC calls (via `requesthandler`):
```go
// requesthandler already sends filters in body automatically
conversations, err := reqHandler.ConversationV1ConversationGetAll(ctx, filters, pageOpts)
```

**Why this pattern:**

1. **Consistency** - Matches how `requesthandler` has always sent filters (in body)
2. **Type safety** - FieldStruct enables compile-time validation and type conversion
3. **Complex filters** - Body JSON supports nested structures, URL params don't
4. **No URL length limits** - Avoids issues with long filter strings

**Migration notes:**

- Old pattern: Filters in URL query params (e.g., `?customer_id=xxx&deleted=false`)
- New pattern: Filters in request body JSON
- Pagination still in URL: `?page=1&limit=50`

For full implementation details, see:
- `bin-common-handler/pkg/utilhandler/filters.go` - Generic filter parsing functions
- `docs/plans/2026-01-14-listenhandler-filter-parsing-implementation-plan.md` - Complete implementation guide

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

### Common Gotchas

#### UUID Fields and DB Tags

**CRITICAL: UUID fields MUST use the `,uuid` db tag for proper type conversion.**

When adding `db:` struct tags to model fields, UUID fields require special handling:

```go
// ✅ CORRECT - UUID field with uuid tag
type Model struct {
    ID         uuid.UUID `db:"id,uuid"`
    CustomerID uuid.UUID `db:"customer_id,uuid"`
    Name       string    `db:"name"`
}

// ❌ WRONG - Missing uuid tag
type Model struct {
    ID         uuid.UUID `db:"id"`           // Will cause string-to-UUID conversion issues
    CustomerID uuid.UUID `db:"customer_id"`  // Will cause filter parsing errors
}
```

**Why this matters:**

1. **Database queries fail silently** - Filters with UUID fields without `,uuid` tags are passed as strings instead of binary values, causing no database matches
2. **Type conversion errors** - `commondatabasehandler.PrepareFields()` needs the `,uuid` tag to convert `uuid.UUID` → binary for MySQL
3. **API bugs** - List endpoints return empty results even when data exists

**Example bug:**
```go
// Bug: conversation model missing uuid tags
type Conversation struct {
    CustomerID uuid.UUID `db:"customer_id"`  // Missing ,uuid tag
}

// Result: GET /v1/conversations?customer_id=<uuid> returns []
// Because filter is passed as string, not binary
```

**How to fix:**
1. Add `,uuid` tag to ALL uuid.UUID fields in model structs
2. Regenerate mocks: `go generate ./...`
3. Update tests: If tests mock database queries, verify UUID values are `uuid.UUID` type, not strings
4. Run verification workflow: `go mod tidy && go mod vendor && go generate ./... && go clean -testcache && go test ./...`

**Always verify UUID fields have `,uuid` tags when:**
- Adding new models
- Refactoring to use `commondatabasehandler` pattern
- Debugging empty API list responses
- Reviewing pull requests with model changes

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
