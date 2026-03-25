# Generic Direct Hash Design

## Problem Statement

The current direct hash mechanism is extension-only, managed by `bin-registrar-manager`. External SIP callers can reach an extension via `sip:direct.<hash>@sip.voipbin.net`, but no other resource type supports this.

We want to generalize direct hash support to five resource types: **extension**, **conference**, **AI assistant**, **AI team**, and **agent**. This enables external callers to reach any of these resources via a single, opaque, shareable hash.

## Approach

Introduce a new service, `bin-direct-manager`, that owns all direct hash lifecycle operations. Each resource type stores its direct hash locally (denormalized) for fast reads. Call-manager resolves hashes through direct-manager and dispatches to the appropriate resource handler based on the resolved `resource_type`.

## New Service: bin-direct-manager

### Responsibilities

- Own the `direct_directs` table (single source of truth for hash → resource mapping)
- Generate, store, resolve, regenerate, and delete hashes
- Provide RPC APIs for other services to manage hashes
- Cache hash lookups in Redis (hot path for incoming calls)
- Clean up orphaned hashes on customer deletion
- Internal RPC only — no `WebhookMessage` needed. The `direct_hash` string is exposed via each resource's existing `WebhookMessage`

### Database Table: `direct_directs`

| Column | Type | Notes |
|--------|------|-------|
| `id` | binary(16) | PK |
| `customer_id` | binary(16) | Owner |
| `resource_type` | varchar(32) | `extension`, `conference`, `ai`, `ai_team`, `agent` |
| `resource_id` | binary(16) | UUID of the resource |
| `hash` | varchar(255) | Full prefixed hash (e.g., `direct.a3f8b2c1d4e5`), unique |
| `tm_create` | datetime(6) | |
| `tm_update` | datetime(6) | |

**Hard deletes** — no `tm_delete` column. Direct hashes are operational lookup data, not audit records. This keeps the UNIQUE constraints clean (no conflict with soft-deleted rows).

**Indexes:**
- `PRIMARY KEY (id)`
- `UNIQUE INDEX idx_direct_directs_hash (hash)` — for resolution lookups
- `UNIQUE INDEX idx_direct_directs_resource (resource_type, resource_id)` — one hash per resource
- `INDEX idx_direct_directs_customer_id (customer_id)` — for customer-scoped queries and deletion

### Hash Generation

- Algorithm: `crypto/rand.Read` (6 bytes) → `hex.EncodeToString` → 12-character hex string
- Prefix: `DirectPrefix = "direct."` constant defined in `models/direct/direct.go`
- Stored value: `direct.<12-char-hex>` (e.g., `direct.a3f8b2c1d4e5`)
- The full prefixed string is stored in both `direct_directs.hash` and each resource's `direct_hash` column — no stripping or transformation needed at query time
- Collision handling: retry up to 3 times on unique constraint violation

### RPC Endpoints

| Method | URI | Body | Purpose |
|--------|-----|------|---------|
| POST | `/v1/directs` | `{ "customer_id": "...", "resource_type": "...", "resource_id": "..." }` | Create hash, returns full Direct object |
| GET | `/v1/directs/{id}` | — | Get by ID |
| GET | `/v1/directs/by-hash/{hash}` | — | Resolve hash to single Direct object |
| GET | `/v1/directs?page_size=&page_token=` | Body filters (see below) | List with pagination |
| DELETE | `/v1/directs/{id}` | — | Hard delete |
| POST | `/v1/directs/{id}/regenerate` | — | Generate new hash for same resource (POST, not PUT — not idempotent) |

**List endpoint filters** (in request body JSON, following monorepo convention):

```go
type FieldStruct struct {
    CustomerID   uuid.UUID `filter:"customer_id"`
    ResourceType string    `filter:"resource_type"`
    ResourceID   uuid.UUID `filter:"resource_id"`
    Hash         string    `filter:"hash"`
}
```

### Caching

- Redis cache for `GetByHash` only (the hot path for every incoming direct call)
- Cache key: `direct:hash:<full-hash-value>`
- Cache value: full `Direct` struct (JSON)
- Long TTL (hashes rarely change)
- Explicit invalidation on delete and regenerate

### SubscribeHandler

- Subscribes to `customer_deleted` events from customer-manager
- On customer deletion: hard-delete all `direct_directs` rows for that customer
- Safety net for orphaned directs when individual resource deletion fails to clean up

### Metrics

Declared inline in `pkg/directhandler/main.go` (following existing convention — no separate `metricshandler` package).

Metric names (namespace `direct_manager`):
- `direct_manager_direct_create_total` — counter
- `direct_manager_direct_delete_total` — counter
- `direct_manager_direct_regenerate_total` — counter
- `direct_manager_hash_resolve_total` — counter

These do not conflict with requesthandler's `request_process_time` and `event_publish_total`.

### Event Types

Defined in `models/direct/event.go`:
- `EventTypeDirectCreated`
- `EventTypeDirectDeleted`
- `EventTypeDirectRegenerated`

### Service Structure

```
bin-direct-manager/
  cmd/direct-manager/main.go              # Daemon entry point
  cmd/direct-control/main.go              # CLI tool for operational troubleshooting
  internal/config/main.go                 # Cobra + Viper config
  models/direct/
    direct.go                             # Direct struct, DirectPrefix constant
    event.go                              # Event type constants
    field.go                              # Field type + constants
    filters.go                            # FieldStruct with filter tags
  pkg/directhandler/
    main.go                               # DirectHandler interface + go:generate mockgen
    mock_main.go                          # Generated mock
    handler.go                            # Business logic
    handler_test.go                       # Tests (gomock, table-driven)
    db.go                                 # DB init helper
    event.go                              # Event publishing helpers
  pkg/dbhandler/
    main.go                               # DBHandler interface + go:generate mockgen
    mock_main.go                          # Generated mock
    db.go                                 # DB init
    direct.go                             # CRUD on direct_directs table
    direct_test.go                        # Tests (gomock, table-driven)
  pkg/listenhandler/
    main.go                               # RPC routing with regex patterns + go:generate mockgen
    mock_main.go                          # Generated mock
    v1_directs.go                         # Endpoint handlers
    v1_directs_test.go                    # Tests (gomock, table-driven)
    models/request/
      main.go
      v1_directs.go                       # Request structs (POST, regenerate)
  pkg/cachehandler/
    main.go                               # CacheHandler interface + go:generate mockgen
    mock_main.go                          # Generated mock
    handler.go                            # Redis cache operations
    handler_test.go                       # Tests (gomock, table-driven)
  pkg/subscribehandler/
    main.go                               # SubscribeHandler interface + go:generate mockgen
    mock_main.go                          # Generated mock
    customer_manager.go                   # customer_deleted event handler
```

### Infrastructure

- `Dockerfile` following standard monorepo pattern
- `k8s/` directory with Kubernetes manifests
- `.circleci/config.yml`: add `bin-direct-manager/.* run-bin-direct-manager true` path filtering
- `go.mod` with replace directives for monorepo dependencies

## Changes to Existing Services

### common-handler

**New files/changes:**
- `models/outline/queuename.go`: add `QueueNameDirectRequest`, `QueueNameDirectEvent`, `QueueNameDirectSubscribe`
- `models/outline/servicename.go`: add `ServiceNameDirectManager`
- `pkg/requesthandler/direct_directs.go`: add `sendRequestDirect` (using `json.RawMessage`), `DirectV1DirectCreate`, `DirectV1DirectGet`, `DirectV1DirectGets`, `DirectV1DirectGetByHash`, `DirectV1DirectDelete`, `DirectV1DirectRegenerate`
- `pkg/requesthandler/main.go`: add methods to `RequestHandler` interface

**Removed:**
- `RegistrarV1ExtensionGetByDirectHash` method
- `RegistrarV1ExtensionDirectGetByHash` method
- Import of `rmextensiondirect` model

### registrar-manager (extension)

**Schema:** Alter `registrar_extensions` — add `direct_id binary(16)`, `direct_hash varchar(255)`

**Model:** Update `Extension` struct — note that `DirectHash` changes from `db:"-"` (programmatically populated via separate query) to `db:"direct_hash"` (read directly from the extensions table). This means `commondatabasehandler.GetDBFields()` will now include `direct_hash` in SELECT lists, which is correct since the column will exist after migration.
```go
DirectID   uuid.UUID `json:"direct_id" db:"direct_id,uuid"`
DirectHash string    `json:"direct_hash" db:"direct_hash"`
```

**Create flow:**
1. Generate extension UUID
2. Call `DirectV1DirectCreate(ctx, customerID, "extension", extensionID)` → get DirectID + DirectHash
3. Insert extension with DirectID + DirectHash populated
4. If step 2 fails → don't create extension
5. If step 3 fails → call `DirectV1DirectDelete` to clean up orphaned direct

**Delete flow:**
- Call `DirectV1DirectDelete(ctx, directID)` to hard-delete the direct
- If RPC fails → log error, continue with extension deletion

**Removed:**
- `pkg/extensiondirecthandler/` package (entire)
- `models/extensiondirect/` package (entire)
- Listen handler endpoints: `/v1/extension-directs`, `/v1/extensions/by-direct-hash/{hash}`
- DB handler methods for `registrar_directs` table

### conference-manager (conference)

**Schema:** Alter `conference_conferences` — add `direct_id binary(16)`, `direct_hash varchar(255)`

**Model:** Update `Conference` struct with `DirectID` and `DirectHash` (same pattern as extension).

**Create/Delete:** Same ordering and error handling as registrar-manager.

### ai-manager (AI assistant)

**Schema:** Alter `ai_ais` — add `direct_id binary(16)`, `direct_hash varchar(255)`

**Model:** Update `AI` struct with `DirectID` and `DirectHash`.

**Create/Delete:** Same pattern.

### ai-manager (AI team)

**Schema:** Alter `ai_teams` — add `direct_id binary(16)`, `direct_hash varchar(255)`

**Model:** Update `Team` struct with `DirectID` and `DirectHash`.

**Create/Delete:** Same pattern.

### agent-manager (agent)

**Schema:** Alter `agent_agents` — add `direct_id binary(16)`, `direct_hash varchar(255)`

**Model:** Update `Agent` struct with `DirectID` and `DirectHash`.

**Create/Delete:** Same pattern.

### Regenerate Flow (all services)

When regenerating a hash for any resource:
1. Owning service calls `DirectV1DirectRegenerate(ctx, directID)` → gets new Direct with updated hash
2. Owning service updates its own resource row with the new `DirectHash`
3. direct-manager invalidates the Redis cache for the old hash

## Call-Manager Routing Changes

### Current Flow (extension-only)
```
SIP destination: "direct.<hash>"
  → strip "direct." prefix
  → call registrar-manager: ExtensionGetByDirectHash(hash)
  → get Extension
  → create Connect flow
```

### New Flow (generic)
```
SIP destination: "direct.<hash>" (e.g., "direct.a3f8b2c1d4e5")
  → detect "direct." prefix (using dmdirect.DirectPrefix)
  → call direct-manager: DirectV1DirectGetByHash(ctx, "direct.a3f8b2c1d4e5")
  → get Direct { resource_type, resource_id, customer_id }
  → switch on resource_type:
      "extension"  → call registrar-manager ExtensionGet → Connect flow
      "conference" → call conference-manager ConferenceGet → ConferenceJoin flow
      "agent"      → call agent-manager AgentGet → Connect flow
      "ai"         → call ai-manager AIGet → AI flow
      "ai_team"    → call ai-manager TeamGet → AI Team flow
```

### Files Changed

`pkg/callhandler/start_incoming_domain_type_sip.go`:
- Replace `startIncomingDomainTypeSIPDirectExtension()` with generic `startIncomingDomainTypeSIPDirect()`
- No prefix stripping before lookup — destination number matches stored hash exactly (call-manager still checks for the prefix to enter the direct routing path)
- Dispatch by `resource_type` reuses existing flow creation patterns from `start_incoming_domain_type_registrar.go`

### Error Handling

- Direct hash not found → hang up with `ChannelCauseNoRouteDestination`
- Resolved resource not found (deleted after hash creation) → hang up with `ChannelCauseNoRouteDestination`
- direct-manager unreachable → hang up with `ChannelCauseNoRouteDestination`

## Migration

### Alembic Migration (single migration in bin-dbscheme-manager)

**Step 1:** Create `direct_directs` table
```sql
CREATE TABLE direct_directs (
    id            binary(16) NOT NULL,
    customer_id   binary(16) NOT NULL,
    resource_type varchar(32) NOT NULL,
    resource_id   binary(16) NOT NULL,
    hash          varchar(255) NOT NULL,
    tm_create     datetime(6),
    tm_update     datetime(6),

    PRIMARY KEY(id),
    UNIQUE INDEX idx_direct_directs_hash (hash),
    UNIQUE INDEX idx_direct_directs_resource (resource_type, resource_id),
    INDEX idx_direct_directs_customer_id (customer_id)
);
```

**Step 2:** Alter resource tables
```sql
ALTER TABLE registrar_extensions ADD COLUMN direct_id binary(16), ADD COLUMN direct_hash varchar(255);
ALTER TABLE conference_conferences ADD COLUMN direct_id binary(16), ADD COLUMN direct_hash varchar(255);
ALTER TABLE ai_ais ADD COLUMN direct_id binary(16), ADD COLUMN direct_hash varchar(255);
ALTER TABLE ai_teams ADD COLUMN direct_id binary(16), ADD COLUMN direct_hash varchar(255);
ALTER TABLE agent_agents ADD COLUMN direct_id binary(16), ADD COLUMN direct_hash varchar(255);
```

**Step 3:** Migrate existing data from `registrar_directs`

Note: existing `registrar_directs.hash` values are raw hex without the `direct.` prefix. The migration must prepend it.

```sql
INSERT INTO direct_directs (id, customer_id, resource_type, resource_id, hash, tm_create, tm_update)
SELECT id, customer_id, 'extension', extension_id, CONCAT('direct.', hash), tm_create, tm_update
FROM registrar_directs
WHERE tm_delete IS NULL;  -- registrar_directs uses NULL for active records

UPDATE registrar_extensions e
INNER JOIN direct_directs d ON d.resource_id = e.id AND d.resource_type = 'extension'
SET e.direct_id = d.id, e.direct_hash = d.hash;
```

**Step 4:** Drop old table
```sql
DROP TABLE registrar_directs;
```

### Application-Level Backfill

After deploying `direct-manager` and updated services, run a one-time backfill to create direct hashes for all existing resources that don't have one. This can be done via `direct-control` CLI or a script that:
1. Lists all extensions/conferences/AIs/teams/agents without a `direct_hash`
2. Calls `DirectV1DirectCreate` for each
3. Updates the resource row with the returned DirectID + DirectHash

## OpenAPI Changes

**Add `direct_hash` field to 5 resource WebhookMessage schemas:**
- `RegistrarManagerExtension`: add `direct_hash` (string)
- `ConferenceManagerConference`: add `direct_hash` (string)
- `AIManagerAI`: add `direct_hash` (string)
- `AIManagerTeam`: add `direct_hash` (string)
- `AgentManagerAgent`: add `direct_hash` (string)

**Remove:**
- Extension direct-related endpoint schemas (replaced by direct-manager internal RPC)

**Regenerate:**
```bash
cd bin-openapi-manager && go generate ./...
cd bin-api-manager && go generate ./...
```

## Documentation Fix

### coding-conventions.md Section 10.3

Replace the current misleading example (shows pagination in JSON body struct) with the correct pattern:

```go
// ✅ CORRECT — pagination from URL, filters from request body
func (h *listenHandler) processV1AgentsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
    u, err := url.Parse(m.URI)

    // Pagination from URL
    tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
    pageSize := uint64(tmpSize)
    pageToken := u.Query().Get(PageToken)

    // Filters from request body
    tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
    filters, err := utilhandler.ConvertFilters[agent.FieldStruct, agent.Field](agent.FieldStruct{}, tmpFilters)

    tmp, err := h.agentHandler.List(ctx, pageSize, pageToken, filters)
}

// ❌ WRONG — never parse filters from URL query parameters
customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))  // Will be uuid.Nil!
```

Filter fields are defined via `FieldStruct` with `filter:` tags in `models/<resource>/filters.go`. For the complete implementation guide, see [common-workflows.md](common-workflows.md#parsing-filters-from-request-body).
