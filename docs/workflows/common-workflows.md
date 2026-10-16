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
4. Coordinate deployment - shared changes affect multiple services

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

## Database Migrations

### Overview

Two mechanisms change database schemas, one per database:

- `bin_manager`: every application table is owned by one service and changes only through that service's `migrations/` directory (golang-migrate, applied when the service starts, version table `<prefix>_schema_migrations`). The rules and the procedure are in `docs/workflows/service-owned-migrations.md`; the services and their tables are listed in `bin-dbscheme-manager/docs/schema-ownership.md`.
- `asterisk`: the `asterisk_config` stream in `bin-dbscheme-manager` keeps using Alembic (see [Alembic for the asterisk database](#alembic-for-the-asterisk-database) below).

The `bin-manager` Alembic stream in `bin-dbscheme-manager` is frozen read-only history: never add a revision to it. The directory stays because the install scripts still run that chain to catch up existing installations and older pins, and `scripts/verify-service-migrations.sh` builds its Alembic head (E1) reference from it. The migration hooks (`.githooks/pre-commit`, `.claude/settings.json`, `.claude/scripts/check-migration-*.sh`) behave as before and still check that tree.

Never run manual SQL DDL statements against shared databases.

### AI Security Boundaries

**What AI CAN do:**
- ✅ Add service migration files (`migrations/NNNNNN_<description>.up.sql` and `.down.sql`) and Alembic revision files for `asterisk_config`
- ✅ Read and explain migration file contents
- ✅ Run the structure and verify checks, which use scratch databases only
- ✅ Commit migration files to git

**What AI MUST NEVER do:**
- 🚫 Run a service's migration command (`<name>-control migrate` with `up`, `down` or `force`; for `bin-timeline-manager` also `migrate-mysql`) against anything other than a local throwaway database
- 🚫 Run `alembic upgrade` or `alembic downgrade` for either stream (`asterisk_config` or the frozen `bin-manager`) against anything other than a local throwaway database
- 🚫 Add a revision to the frozen `bin-manager` Alembic stream
- 🚫 Execute any SQL that modifies a database schema

**Why:** Schema changes on shared databases can lose data or stop services, so they need human review before they reach those databases.

### Changing a `bin_manager` Table

Follow `docs/workflows/service-owned-migrations.md`; this section does not repeat it. In short: add a new numbered file pair in the owning service's `migrations/` directory, one statement per file, never edit a merged file, run `scripts/check-service-migrations.sh <service-dir>` and the verify script, and state the impact analysis in the PR. The owning service's `CLAUDE.md` and `docs/operations.md` describe its version table and recovery commands.

### Alembic for the asterisk Database

Only the `asterisk_config` stream accepts new Alembic revisions. The full procedure is in `bin-dbscheme-manager/CLAUDE.md` and `bin-dbscheme-manager/docs/migrations.md`.

```bash
cd bin-dbscheme-manager/asterisk_config
cp alembic.ini.sample alembic.ini   # first time only, points at a local database
alembic -c alembic.ini revision -m "ps_endpoints add column example_flag"
```

Edit the generated file in `config/versions/` with raw SQL in both `upgrade()` and `downgrade()` (`op.execute(...)`), one change per revision, and never modify a merged revision.

### Troubleshooting

**Service-owned migrations (`bin_manager`):** a service that fails to migrate exits at startup with the error in its log. `<name>-control migrate status` shows the applied version, the image version and the dirty flag. The meaning of each error and the `force` recovery are in the owning service's `docs/operations.md` or `CLAUDE.md`; a dirty version is fixed by checking whether the statement was applied and forcing to `N` or `N-1`. A rollback of a deployed service is the previous image: the migration helper skips a database version that is ahead of the image.

**Alembic (`asterisk`):** `alembic -c alembic.ini current`, `history` and `heads` are read-only status checks; a stream must have exactly one head. Rollback with `alembic -c alembic.ini downgrade -1` is a human-only step. See `bin-dbscheme-manager/docs/operations.md`.
