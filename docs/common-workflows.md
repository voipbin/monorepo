# Common Workflows

> **Quick Reference:** For workflow overview, see [CLAUDE.md](../CLAUDE.md#common-workflows)

This document provides step-by-step implementation guides for common development tasks across the VoIPbin monorepo.

## Adding a New API Endpoint

1. Define OpenAPI spec in `bin-openapi-manager/openapi/paths/<resource>/`
2. Regenerate models: `cd bin-openapi-manager && go generate ./...`
3. Add handler in target service's `pkg/listenhandler/`
4. Implement business logic in appropriate domain handler
5. Register endpoint in `bin-api-manager` if public-facing
6. Update Swagger docs in api-manager: `swag init`

## Creating a New Flow Action

1. Define action type in `bin-flow-manager/models/action/`
2. Add action handler in `bin-flow-manager/pkg/actionhandler/`
3. Implement execution logic in target service (e.g., call-manager)
4. Update flow-manager's request handler to route to new service
5. Add tests for action validation and execution

## Adding a New Manager Service

1. Create directory: `bin-<name>-manager/`
2. Copy structure from existing service (e.g., tag-manager)
3. Update `bin-api-manager/go.mod` with replace directive
4. Add RabbitMQ queue names to `bin-common-handler/models/outline/`
5. Add request methods to `bin-common-handler/pkg/requesthandler/`
6. Update `.circleci/config.yml` path mappings

## Modifying Shared Models

When changing `bin-common-handler` or `bin-openapi-manager`:

1. Make changes in the shared package
2. Regenerate if needed: `go generate ./...`
3. Test impact on dependent services
4. Update all affected services' vendor directories if using vendoring
5. Coordinate deployment - shared changes affect multiple services

**Note:** For flow execution patterns and variable substitution, see `bin-flow-manager/CLAUDE.md`.

## Parsing Filters from Request Body

**Note:** This is a monorepo-wide pattern for all services that expose list endpoints (GET /v1/resources). Implemented using `bin-common-handler/pkg/utilhandler`. See individual service CLAUDE.md files for service-specific filter field definitions.

### Pattern

**All list endpoints parse filters from request body (JSON), not URL query parameters.**

This pattern was implemented as part of the `commondatabasehandler` refactoring. See `docs/plans/2026-01-14-listenhandler-filter-parsing-implementation-plan.md` for complete implementation details.

### Why This Is Critical

**🚨 CRITICAL RULE: NEVER parse filter data from URL query parameters**

**ONLY pagination parameters** (`page_size`, `page_token`) should be parsed from URL query parameters:
```go
// ✅ CORRECT - Parse pagination from URL
tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
pageSize := uint64(tmpSize)
pageToken := u.Query().Get(PageToken)

// ✅ CORRECT - Parse filters from request body
tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
typedFilters, err := utilhandler.ConvertFilters[resource.FieldStruct, resource.Field](...)
```

```go
// ❌ WRONG - Never parse filter data from URL
aicallID := uuid.FromStringOrNil(u.Query().Get("aicall_id"))  // BUG!
customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))  // BUG!

// These will be uuid.Nil because requesthandler sends them in body, not URL
// This causes empty query results and is a common bug pattern
```

**Why this is critical:**
- `bin-common-handler/pkg/requesthandler` sends **ALL filters in request body**, not URL
- Parsing from URL gets `uuid.Nil` or empty strings → database queries return empty results
- This bug was found in `bin-ai-manager/v1_messages.go` (see `docs/plans/2026-01-16-fix-aimessages-empty-list-bug-design.md`)

### Reference Implementation

**Reference implementation:** `bin-agent-manager/pkg/listenhandler/v1_agents.go:20-53`

### How It Works

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

### Why This Pattern

1. **Consistency** - Matches how `requesthandler` has always sent filters (in body)
2. **Type safety** - FieldStruct enables compile-time validation and type conversion
3. **Complex filters** - Body JSON supports nested structures, URL params don't
4. **No URL length limits** - Avoids issues with long filter strings

### Migration Notes

- Old pattern: Filters in URL query params (e.g., `?customer_id=xxx&deleted=false`)
- New pattern: Filters in request body JSON
- Pagination still in URL: `?page=1&limit=50`

For full implementation details, see:
- `bin-common-handler/pkg/utilhandler/filters.go` - Generic filter parsing functions
- `docs/plans/2026-01-14-listenhandler-filter-parsing-implementation-plan.md` - Complete implementation guide

## Database Migrations with Alembic

### Overview

Database schema changes for VoIPbin services are managed through Alembic migrations in the `bin-dbscheme-manager` service. This is the ONLY way to modify database schemas - never run manual SQL DDL statements against production databases.

### AI Security Boundaries

**⚠️ CRITICAL: AI Security Boundaries**

**What AI CAN do:**
- ✅ Create migration files (`alembic -c alembic.ini revision -m "..."`)
- ✅ Edit migration files to add SQL in upgrade()/downgrade() functions
- ✅ Read and explain migration file contents
- ✅ Commit migration files to git

**What AI MUST NEVER do:**
- 🚫 Run `alembic upgrade` (applies migrations to database)
- 🚫 Run `alembic downgrade` (rolls back database changes)
- 🚫 Execute any SQL that modifies database schema
- 🚫 Apply migrations automatically

**Why:** Database schema changes are irreversible operations requiring:
- Human review and explicit authorization
- Testing on development environment first
- VPN access to production database
- Coordination with deployment schedule
- Risk of data loss or service outages

AI can assist with creating and reviewing migration files, but database execution requires human control.

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

**Step 5: Commit and push** (AI can do this)
```bash
cd /home/pchero/gitvoipbin/monorepo
git add bin-dbscheme-manager/bin-manager/main/versions/<hash>_*.py
git commit -m "feat(dbscheme): add owner_type column to storage_files table"
git push
```

**Step 6: (HUMAN ONLY) Test migration locally**
```bash
# 🚫 AI MUST NOT execute these commands

# Apply migration to local database
alembic -c alembic.ini upgrade head

# Verify schema change
mysql -u user -p voipbin -e "DESCRIBE storage_files;"

# Test rollback (optional)
alembic -c alembic.ini downgrade -1
alembic -c alembic.ini upgrade head
```

**Step 7: (HUMAN ONLY) Apply to staging/production**

Connect to VPN first (REQUIRED), then apply migration:
```bash
# 🚫 AI MUST NOT execute these commands

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
