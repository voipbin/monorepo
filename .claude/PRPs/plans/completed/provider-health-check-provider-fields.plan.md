# Plan: Provider Health Fields (Phase 4)

## Summary

Add `health_status` and `health_checked_at` to the `Provider` model in `bin-route-manager`, create the Alembic DB migration to add the two columns to `route_providers`, and wire the hostname-update reset so stale health data is never visible after a config change.

This phase produces no visible API change by itself — the new fields will appear in existing GET/LIST responses as `"health_status": "unknown"` and `"health_checked_at": null` immediately after migration. Phase 5 (goroutine) starts populating real values.

## User Story

As a VoIPbin administrator, I want every provider returned by the API to include a `health_status` field, so I can see whether it has been verified reachable.

## Problem → Solution

`Provider` has no health fields today → add `HealthStatus` (string, `"unknown"` by default) and `HealthCheckedAt` (*time.Time, nil until first check) to the model, the DB schema, the field constants, the WebhookMessage, and the update logic.

## Metadata

- **Complexity**: Low-Medium (model + DB migration, no new service)
- **Source PRD**: `.claude/PRPs/prds/provider-health-check.prd.md`
- **PRD Phase**: Phase 4 — Provider health fields
- **Status**: in-progress
- **Files Changed**: 7 (provider.go, field.go, field_test.go, webhook.go, webhook_test.go, dbhandler/main.go, dbhandler/provider.go, dbhandler/mock_dbhandler.go) + 1 Alembic migration

---

## Mandatory Reading

Files that MUST be read before implementing:

| Priority | File | Why |
|---|---|---|
| P0 | `bin-route-manager/models/provider/provider.go` | Current struct — add fields here |
| P0 | `bin-route-manager/models/provider/field.go` | Field constants — add two new constants |
| P0 | `bin-route-manager/models/provider/webhook.go` | WebhookMessage — add both fields + update ConvertWebhookMessage |
| P0 | `bin-route-manager/pkg/dbhandler/provider.go` | ProviderUpdate pattern to mirror for hostname-reset; ProviderUpdateHealthStatus to add |
| P0 | `bin-route-manager/pkg/dbhandler/main.go` | DBHandler interface — add new method signature |
| P1 | `bin-route-manager/models/provider/field_test.go` | Extend with two new field constant assertions |
| P1 | `bin-route-manager/models/provider/webhook_test.go` | Extend existing test cases with health fields |
| P1 | `bin-route-manager/pkg/dbhandler/mock_dbhandler.go` | Regenerated via `go generate` — do NOT edit manually |
| P2 | `bin-dbscheme-manager/bin-manager/alembic.ini` | Config file for the `alembic revision` command |
| P2 | Recent migration file in `bin-manager/main/versions/` | Template for upgrade()/downgrade() SQL pattern |

---

## Patterns to Mirror

### STRUCT_FIELD_DEFINITION

```go
// SOURCE: bin-route-manager/models/provider/provider.go (existing timestamp pattern)
// Existing:
TMCreate *time.Time `json:"tm_create" db:"tm_create"`
TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`

// Add AFTER the existing fields, BEFORE the timestamp block:
HealthStatus    string     `json:"health_status"    db:"health_status"`    // "unknown" | "healthy" | "unhealthy"
HealthCheckedAt *time.Time `json:"health_checked_at" db:"health_checked_at"` // nil until first check
```

GOTCHA: `HealthStatus` is a plain `string`, NOT a pointer — the DB column has `NOT NULL DEFAULT 'unknown'` so it is always present. `HealthCheckedAt` IS a pointer because the DB column is nullable (`DATETIME(6)` with no default).

GOTCHA: `HealthStatus` has no custom type (unlike `Type`). The valid values are string literals `"unknown"`, `"healthy"`, `"unhealthy"` — define them as constants (see below).

### HEALTH_STATUS_CONSTANTS

```go
// SOURCE: bin-route-manager/models/provider/provider.go — add near TypeSIP
// Health status values
const (
    HealthStatusUnknown   = "unknown"
    HealthStatusHealthy   = "healthy"
    HealthStatusUnhealthy = "unhealthy"
)
```

### FIELD_CONSTANTS

```go
// SOURCE: bin-route-manager/models/provider/field.go (existing pattern)
// Add BEFORE the "filter only" section:
FieldHealthStatus    Field = "health_status"    // health_status
FieldHealthCheckedAt Field = "health_checked_at" // health_checked_at
```

### WEBHOOKMESSAGE_FIELDS

```go
// SOURCE: bin-route-manager/models/provider/webhook.go (existing fields pattern)
// Add to WebhookMessage struct, BEFORE TMCreate/TMUpdate/TMDelete:
HealthStatus    string     `json:"health_status"`
HealthCheckedAt *time.Time `json:"health_checked_at"`

// Add to ConvertWebhookMessage() return literal:
HealthStatus:    h.HealthStatus,
HealthCheckedAt: h.HealthCheckedAt,
```

### PROVIDER_CREATE_DEFAULT

```go
// SOURCE: bin-route-manager/pkg/dbhandler/provider.go ProviderCreate()
// In providerhandler/provider.go Create(), the provider struct literal must set:
HealthStatus: provider.HealthStatusUnknown,
// HealthCheckedAt: nil (zero value — omit, pointer defaults to nil)
```

GOTCHA: `ProviderCreate` already uses `commondatabasehandler.PrepareFields(p)` which reads all `db:` tagged fields. Since `HealthStatus` has a `db:"health_status"` tag, it will be included in the INSERT automatically with value `"unknown"`. No additional code change needed in `dbhandler/provider.go:ProviderCreate`.

### PROVIDER_UPDATE_HOSTNAME_RESET

```go
// SOURCE: bin-route-manager/pkg/providerhandler/provider.go Update()
// Current Update() builds the field map — extend it to reset health when hostname is present:

fields := map[provider.Field]any{
    provider.FieldType:        providerType,
    provider.FieldHostname:    hostname,
    provider.FieldTechPrefix:  techPrefix,
    provider.FieldTechPostfix: techPostfix,
    provider.FieldTechHeaders: techHeaders,
    provider.FieldName:        name,
    provider.FieldDetail:      detail,
    // Reset health status when hostname changes
    provider.FieldHealthStatus:    provider.HealthStatusUnknown,
    provider.FieldHealthCheckedAt: nil,
}
```

MIRROR: The existing `ProviderUpdate()` in `dbhandler/provider.go` uses `commondatabasehandler.PrepareFields(fields)` on the `map[provider.Field]any`. Setting `FieldHealthCheckedAt: nil` will produce `health_checked_at = NULL` in the SQL — this is the correct behavior.

GOTCHA: The hostname reset always fires on Update(), even if the hostname didn't actually change. This is intentional and matches the PRD: "Reset health_status to unknown on provider hostname update." The implementation always includes `FieldHostname` in the update map, so the reset is unconditional on any update call — this is the safest and simplest approach for v1.

### PROVIDER_UPDATE_HEALTH_STATUS

```go
// New dbhandler method — add to bin-route-manager/pkg/dbhandler/provider.go

// ProviderUpdateHealthStatus updates the health_status and health_checked_at fields.
// Used by the background health check goroutine (Phase 5).
func (h *handler) ProviderUpdateHealthStatus(ctx context.Context, id uuid.UUID, status string, checkedAt *time.Time) error {
    fields := map[provider.Field]any{
        provider.FieldHealthStatus:    status,
        provider.FieldHealthCheckedAt: checkedAt,
    }

    return h.ProviderUpdate(ctx, id, fields)
}
```

MIRROR: `ProviderDelete` and `ProviderUpdate` both use `map[provider.Field]any` + `commondatabasehandler.PrepareFields`. Use the same pattern.

IMPORTS: `time` package is already imported in `dbhandler/provider.go` via the `provider` package's `*time.Time` usage, but the `time` stdlib import must be explicitly added to `provider.go` if not already present.

### DBHANDLER_INTERFACE

```go
// SOURCE: bin-route-manager/pkg/dbhandler/main.go — add to DBHandler interface

// provider
ProviderCreate(ctx context.Context, c *provider.Provider) error
ProviderGet(ctx context.Context, id uuid.UUID) (*provider.Provider, error)
ProviderList(ctx context.Context, token string, limit uint64, filters map[provider.Field]any) ([]*provider.Provider, error)
ProviderDelete(ctx context.Context, id uuid.UUID) error
ProviderUpdate(ctx context.Context, id uuid.UUID, fields map[provider.Field]any) error
ProviderUpdateHealthStatus(ctx context.Context, id uuid.UUID, status string, checkedAt *time.Time) error  // ADD THIS
```

IMPORTS: Add `"time"` to imports in `main.go` once the method signature references `*time.Time`.

---

## DB Migration

### CRITICAL: Never Manually Create Migration Files

**ALWAYS use `alembic revision` to generate the migration file. NEVER create a `.py` file by hand or use a placeholder revision ID.**

The migration file must be created with:

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-dbscheme-manager/bin-manager
alembic -c alembic.ini revision -m "route_providers add column health_status health_checked_at"
```

This command generates a new `.py` file in `main/versions/` with a unique auto-generated revision ID and the correct `down_revision` pointing to the current head. After running the command, open the generated file and add the SQL in `upgrade()` and `downgrade()`.

### Migration SQL Content

After `alembic revision` generates the file, fill in the functions:

```python
def upgrade():
    op.execute("""
        ALTER TABLE route_providers
        ADD COLUMN health_status VARCHAR(64) NOT NULL DEFAULT 'unknown'
    """)
    op.execute("""
        ALTER TABLE route_providers
        ADD COLUMN health_checked_at DATETIME(6)
    """)


def downgrade():
    op.execute("""
        ALTER TABLE route_providers
        DROP COLUMN health_status
    """)
    op.execute("""
        ALTER TABLE route_providers
        DROP COLUMN health_checked_at
    """)
```

GOTCHA: `health_status VARCHAR(64) NOT NULL DEFAULT 'unknown'` — the `NOT NULL DEFAULT` means existing rows automatically get `'unknown'` when the migration runs. No backfill needed.

GOTCHA: `health_checked_at DATETIME(6)` with no DEFAULT — MySQL allows NULL for columns with no explicit DEFAULT; existing rows get NULL. This matches `HealthCheckedAt *time.Time` (nil pointer).

GOTCHA: The table name is `route_providers` (not `providers`). This was established by the `354328132eeb_all_add_table_namespace.py` migration: `rename table providers to route_providers`. Verify with `grep route_providers` before running.

GOTCHA: Run `alembic -c alembic.ini heads` AFTER generating the revision to confirm exactly one head. Multiple heads indicate a chain error.

---

## Step-by-Step Tasks

### Task 1: Add fields to Provider struct

**File**: `bin-route-manager/models/provider/provider.go`

MIRROR: Follow the exact `json:"..."` + `db:"..."` tag pattern of existing fields.

Add `HealthStatus` and `HealthCheckedAt` fields after `Detail` and before the timestamp block. Add string constants `HealthStatusUnknown`, `HealthStatusHealthy`, `HealthStatusUnhealthy` near `TypeSIP`.

```go
// Add constants (near TypeSIP const block)
const (
    HealthStatusUnknown   = "unknown"
    HealthStatusHealthy   = "healthy"
    HealthStatusUnhealthy = "unhealthy"
)

// Add fields to Provider struct (after Detail, before // timestamp)
HealthStatus    string     `json:"health_status"     db:"health_status"`
HealthCheckedAt *time.Time `json:"health_checked_at" db:"health_checked_at"`
```

VALIDATE: `go build ./models/provider/...` — no errors.

---

### Task 2: Add Field constants

**File**: `bin-route-manager/models/provider/field.go`

Add two new constants. Insert them after `FieldDetail` and before the `// filter only` comment.

```go
FieldHealthStatus    Field = "health_status"    // health_status
FieldHealthCheckedAt Field = "health_checked_at" // health_checked_at
```

VALIDATE: `go build ./models/provider/...` — no errors.

---

### Task 3: Update field_test.go

**File**: `bin-route-manager/models/provider/field_test.go`

Add two new test cases to `TestFieldConstants`:

```go
{"field_health_status",     FieldHealthStatus,    "health_status"},
{"field_health_checked_at", FieldHealthCheckedAt, "health_checked_at"},
```

VALIDATE: `go test ./models/provider/...` — all tests pass.

---

### Task 4: Update WebhookMessage and ConvertWebhookMessage

**File**: `bin-route-manager/models/provider/webhook.go`

Add two fields to `WebhookMessage` struct (before `TMCreate`):

```go
HealthStatus    string     `json:"health_status"`
HealthCheckedAt *time.Time `json:"health_checked_at"`
```

Update `ConvertWebhookMessage()` to include:

```go
HealthStatus:    h.HealthStatus,
HealthCheckedAt: h.HealthCheckedAt,
```

No new imports needed — `time` is already imported.

VALIDATE: `go build ./models/provider/...` — no errors.

---

### Task 5: Update webhook_test.go

**File**: `bin-route-manager/models/provider/webhook_test.go`

Extend the "all fields populated" test case to include health fields in both `provider` input and `want` output:

```go
// In provider input:
HealthStatus:    HealthStatusHealthy,
HealthCheckedAt: timePtr(time.Date(2023, 1, 3, 12, 0, 0, 0, time.UTC)),

// In want output:
HealthStatus:    HealthStatusHealthy,
HealthCheckedAt: timePtr(time.Date(2023, 1, 3, 12, 0, 0, 0, time.UTC)),
```

Add a new test case for the initial state (health_status = unknown, health_checked_at = nil):

```go
{
    name: "initial state - unknown health",
    provider: &Provider{
        ID:           uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
        Type:         TypeSIP,
        Hostname:     "sip.new.com",
        HealthStatus: HealthStatusUnknown,
        // HealthCheckedAt: nil (zero value)
    },
    want: &WebhookMessage{
        ID:           uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
        Type:         TypeSIP,
        Hostname:     "sip.new.com",
        HealthStatus: HealthStatusUnknown,
        // HealthCheckedAt: nil
    },
},
```

Add assertions in the test loop for the new fields:

```go
if got.HealthStatus != tt.want.HealthStatus {
    t.Errorf("HealthStatus = %v, want %v", got.HealthStatus, tt.want.HealthStatus)
}
if (got.HealthCheckedAt == nil) != (tt.want.HealthCheckedAt == nil) {
    t.Errorf("HealthCheckedAt nil-ness = %v, want %v", got.HealthCheckedAt == nil, tt.want.HealthCheckedAt == nil)
} else if got.HealthCheckedAt != nil && !got.HealthCheckedAt.Equal(*tt.want.HealthCheckedAt) {
    t.Errorf("HealthCheckedAt = %v, want %v", got.HealthCheckedAt, tt.want.HealthCheckedAt)
}
```

VALIDATE: `go test ./models/provider/...` — all tests pass.

---

### Task 6: Update providerhandler Create() to set initial HealthStatus

**File**: `bin-route-manager/pkg/providerhandler/provider.go`

In `Create()`, the `provider.Provider` literal must set `HealthStatus` to `"unknown"`:

```go
p := &provider.Provider{
    ID:           id,
    Type:         providerType,
    Hostname:     hostname,
    TechPrefix:   techPrefix,
    TechPostfix:  techPostfix,
    TechHeaders:  techHeaders,
    Name:         name,
    Detail:       detail,
    HealthStatus: provider.HealthStatusUnknown, // ADD
}
```

GOTCHA: Without this, `HealthStatus` would be an empty string `""` which is NOT a valid health status value. The DB column has `DEFAULT 'unknown'` so the DB will still store `'unknown'`, but the in-memory struct would be wrong until the next read.

VALIDATE: `go build ./pkg/providerhandler/...` — no errors.

---

### Task 7: Update providerhandler Update() to reset health on any update

**File**: `bin-route-manager/pkg/providerhandler/provider.go`

In `Update()`, add `FieldHealthStatus` and `FieldHealthCheckedAt` to the update field map:

```go
fields := map[provider.Field]any{
    provider.FieldType:           providerType,
    provider.FieldHostname:       hostname,
    provider.FieldTechPrefix:     techPrefix,
    provider.FieldTechPostfix:    techPostfix,
    provider.FieldTechHeaders:    techHeaders,
    provider.FieldName:           name,
    provider.FieldDetail:         detail,
    provider.FieldHealthStatus:    provider.HealthStatusUnknown, // ADD
    provider.FieldHealthCheckedAt: nil,                          // ADD — resets to NULL
}
```

GOTCHA: `commondatabasehandler.PrepareFields` with a nil value in the map produces `health_checked_at = NULL` in the SQL. This is the intended behavior — reset to no-check state.

VALIDATE: `go build ./pkg/providerhandler/...` — no errors.

---

### Task 8: Add ProviderUpdateHealthStatus to dbhandler interface

**File**: `bin-route-manager/pkg/dbhandler/main.go`

Add import of `"time"` at the top (if not already present). Add the new method to the `DBHandler` interface:

```go
ProviderUpdateHealthStatus(ctx context.Context, id uuid.UUID, status string, checkedAt *time.Time) error
```

IMPORTS: Add `"time"` to the import block in `main.go`.

VALIDATE: `go build ./pkg/dbhandler/...` fails until Task 9 is complete (implementation must exist).

---

### Task 9: Implement ProviderUpdateHealthStatus in dbhandler

**File**: `bin-route-manager/pkg/dbhandler/provider.go`

Add the implementation after `ProviderUpdate`. No new imports needed beyond what is already present (`context`, `github.com/gofrs/uuid`, `monorepo/bin-route-manager/models/provider`). The `time` package is needed — add it to the import block.

```go
// ProviderUpdateHealthStatus updates the health_status and health_checked_at fields for a provider.
// Called by the background health check goroutine (Phase 5).
func (h *handler) ProviderUpdateHealthStatus(ctx context.Context, id uuid.UUID, status string, checkedAt *time.Time) error {
    fields := map[provider.Field]any{
        provider.FieldHealthStatus:    status,
        provider.FieldHealthCheckedAt: checkedAt,
    }
    return h.ProviderUpdate(ctx, id, fields)
}
```

IMPORTS: Add `"time"` to the import block in `provider.go` if not already there.

VALIDATE: `go build ./pkg/dbhandler/...` — no errors.

---

### Task 10: Regenerate the dbhandler mock

**Command** (from `bin-route-manager/` directory):

```bash
go generate ./pkg/dbhandler/...
```

This regenerates `mock_dbhandler.go` with the new `ProviderUpdateHealthStatus` method. The `//go:generate mockgen` directive in `main.go` handles this automatically.

GOTCHA: Never manually edit `mock_dbhandler.go`. Always regenerate via `go generate`.

VALIDATE: `go build ./pkg/dbhandler/...` and `go test ./pkg/dbhandler/...` — no errors.

---

### Task 11: Create the Alembic migration

**CRITICAL: Use `alembic revision` — NEVER create the file manually.**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-dbscheme-manager/bin-manager
alembic -c alembic.ini revision -m "route_providers add column health_status health_checked_at"
```

This creates a new file in `main/versions/`. Open the generated file and populate `upgrade()` and `downgrade()` as specified in the "Migration SQL Content" section above.

After editing:
```bash
alembic -c alembic.ini heads
# Must show exactly ONE head — the new revision
```

VALIDATE: Confirm the file exists in `main/versions/`, has a unique non-placeholder revision ID, and `alembic -c alembic.ini heads` shows exactly one head.

---

### Task 12: Run full verification for bin-route-manager

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-voip-kamailio-proxy/bin-route-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

GOTCHA: `go generate ./...` regenerates all mocks including `mock_dbhandler.go`. If Task 10 was done manually, this step will re-run it anyway — that's fine.

GOTCHA: If `go test ./...` fails in `pkg/dbhandler/` with "missing method ProviderUpdateHealthStatus on mock", the mock was not regenerated. Run `go generate ./pkg/dbhandler/...` explicitly and re-run tests.

---

## Acceptance Criteria

- [ ] `Provider` struct has `HealthStatus string` with `db:"health_status"` and `json:"health_status"` tags
- [ ] `Provider` struct has `HealthCheckedAt *time.Time` with `db:"health_checked_at"` and `json:"health_checked_at"` tags
- [ ] Constants `HealthStatusUnknown`, `HealthStatusHealthy`, `HealthStatusUnhealthy` defined in `models/provider/provider.go`
- [ ] `FieldHealthStatus` and `FieldHealthCheckedAt` constants exist in `field.go`
- [ ] `field_test.go` asserts both new constants have correct string values
- [ ] `WebhookMessage` struct includes both health fields
- [ ] `ConvertWebhookMessage()` maps both health fields from `Provider` to `WebhookMessage`
- [ ] `webhook_test.go` tests health fields in ConvertWebhookMessage (including nil HealthCheckedAt case)
- [ ] `providerhandler.Create()` sets `HealthStatus: provider.HealthStatusUnknown`
- [ ] `providerhandler.Update()` always sets `FieldHealthStatus: provider.HealthStatusUnknown` and `FieldHealthCheckedAt: nil`
- [ ] `DBHandler` interface includes `ProviderUpdateHealthStatus(ctx, id, status, checkedAt) error`
- [ ] `dbhandler/provider.go` implements `ProviderUpdateHealthStatus` using `ProviderUpdate` internally
- [ ] `mock_dbhandler.go` is regenerated and includes `ProviderUpdateHealthStatus`
- [ ] Alembic migration file created with `alembic revision` command (not manually)
- [ ] Migration file has `upgrade()` that adds `health_status VARCHAR(64) NOT NULL DEFAULT 'unknown'` and `health_checked_at DATETIME(6)` to `route_providers`
- [ ] Migration file has `downgrade()` that drops both columns
- [ ] `alembic -c alembic.ini heads` shows exactly one head
- [ ] `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` passes in `bin-route-manager`
- [ ] No existing tests broken

## Out of Scope for This Phase

- Background health check goroutine (Phase 5)
- `KamailioV1ProviderHealthCheck` RPC call (Phase 3)
- OpenAPI schema update (Phase 6)
- RST documentation update (Phase 7)
- Populating real `healthy`/`unhealthy` values (requires Phase 5 goroutine)

---

*Plan created: 2026-04-20*
*PRD: `.claude/PRPs/prds/provider-health-check.prd.md` Phase 4*
