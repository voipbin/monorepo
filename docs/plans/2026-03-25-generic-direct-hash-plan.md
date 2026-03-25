# Generic Direct Hash Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Introduce `bin-direct-manager` service and generalize direct hash support across extensions, conferences, AI assistants, AI teams, and agents.

**Architecture:** New `bin-direct-manager` service owns the `direct_directs` table and provides RPC APIs for hash CRUD + resolution. Each owning service calls direct-manager on resource create/delete to manage its hash. Call-manager resolves hashes through direct-manager and dispatches by resource type.

**Tech Stack:** Go 1.25, MySQL, Redis, RabbitMQ RPC, Squirrel query builder, gomock, Alembic (Python) for migrations

**Design document:** `docs/plans/2026-03-25-generic-direct-hash-design.md`

---

## Phase 1: Foundation

### Task 1: Alembic Migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<auto>_generic_direct_hash.py`

**Step 1:** Generate the migration file

```bash
cd bin-dbscheme-manager/bin-manager/main
alembic -c alembic.ini revision -m "generic_direct_hash"
```

**Step 2:** Edit the generated migration file

The `upgrade()` function should contain:

```python
def upgrade():
    # Step 1: Create direct_directs table
    op.execute("""
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
        )
    """)

    # Step 2: Alter resource tables — add direct_id and direct_hash
    for table in ['registrar_extensions', 'conference_conferences', 'ai_ais', 'ai_teams', 'agent_agents']:
        op.execute(f"ALTER TABLE {table} ADD COLUMN direct_id binary(16), ADD COLUMN direct_hash varchar(255)")

    # Step 3: Migrate existing registrar_directs data
    # Existing hash values are raw hex — prepend "direct." prefix
    op.execute("""
        INSERT INTO direct_directs (id, customer_id, resource_type, resource_id, hash, tm_create, tm_update)
        SELECT id, customer_id, 'extension', extension_id, CONCAT('direct.', hash), tm_create, tm_update
        FROM registrar_directs
        WHERE tm_delete IS NULL
    """)

    # Step 4: Populate direct_id and direct_hash on registrar_extensions
    op.execute("""
        UPDATE registrar_extensions e
        INNER JOIN direct_directs d ON d.resource_id = e.id AND d.resource_type = 'extension'
        SET e.direct_id = d.id, e.direct_hash = d.hash
    """)

    # Step 5: Drop old table
    op.execute("DROP TABLE registrar_directs")


def downgrade():
    # Recreate registrar_directs
    op.execute("""
        CREATE TABLE registrar_directs (
            id            binary(16),
            customer_id   binary(16),
            extension_id  binary(16),
            hash          varchar(255),
            tm_create     datetime(6),
            tm_update     datetime(6),
            tm_delete     datetime(6),
            PRIMARY KEY(id),
            UNIQUE INDEX idx_registrar_directs_extension_id (extension_id),
            UNIQUE INDEX idx_registrar_directs_hash (hash),
            INDEX idx_registrar_directs_customer_id (customer_id)
        )
    """)

    # Migrate data back — strip "direct." prefix
    op.execute("""
        INSERT INTO registrar_directs (id, customer_id, extension_id, hash, tm_create, tm_update)
        SELECT id, customer_id, resource_id, REPLACE(hash, 'direct.', ''), tm_create, tm_update
        FROM direct_directs
        WHERE resource_type = 'extension'
    """)

    # Remove direct columns from resource tables
    for table in ['registrar_extensions', 'conference_conferences', 'ai_ais', 'ai_teams', 'agent_agents']:
        op.execute(f"ALTER TABLE {table} DROP COLUMN direct_id, DROP COLUMN direct_hash")

    # Drop direct_directs
    op.execute("DROP TABLE direct_directs")
```

**Step 3:** Commit

```bash
git add bin-dbscheme-manager/
git commit -m "NOJIRA-Generic-direct-hash-design

- bin-dbscheme-manager: Add Alembic migration for generic direct hash"
```

---

### Task 2: Common Handler — Queue Names, Service Names

**Files:**
- Modify: `bin-common-handler/models/outline/queuename.go`
- Modify: `bin-common-handler/models/outline/servicename.go`

**Step 1:** Add queue names to `queuename.go`

Add after the `customer-manager` section (alphabetical order — `d` comes after `c`):

```go
// direct-manager
QueueNameDirectEvent     QueueName = "bin-manager.direct-manager.event"
QueueNameDirectRequest   QueueName = "bin-manager.direct-manager.request"
QueueNameDirectSubscribe QueueName = "bin-manager.direct-manager.subscribe"
```

**Step 2:** Add service name to `servicename.go`

Add alphabetically after `ServiceNameCustomerManager`:

```go
ServiceNameDirectManager ServiceName = "direct-manager"
```

**Step 3:** Commit

---

## Phase 2: bin-direct-manager Service

### Task 3: Direct Model

**Files:**
- Create: `bin-direct-manager/models/direct/direct.go`
- Create: `bin-direct-manager/models/direct/field.go`
- Create: `bin-direct-manager/models/direct/event.go`
- Create: `bin-direct-manager/models/direct/filters.go`

**direct.go** — main struct + DirectPrefix constant:

```go
package direct

import (
    "time"

    "github.com/gofrs/uuid"
    commonidentity "monorepo/bin-common-handler/models/identity"
)

const DirectPrefix = "direct."

type Direct struct {
    commonidentity.Identity

    ResourceType string    `json:"resource_type" db:"resource_type"`
    ResourceID   uuid.UUID `json:"resource_id" db:"resource_id,uuid"`
    Hash         string    `json:"hash" db:"hash"`

    TMCreate *time.Time `json:"tm_create" db:"tm_create"`
    TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
}
```

**field.go:**

```go
package direct

type Field string

const (
    FieldID           Field = "id"
    FieldCustomerID   Field = "customer_id"
    FieldResourceType Field = "resource_type"
    FieldResourceID   Field = "resource_id"
    FieldHash         Field = "hash"
    FieldTMCreate     Field = "tm_create"
    FieldTMUpdate     Field = "tm_update"
)
```

**event.go:**

```go
package direct

const (
    EventTypeDirectCreated     = "direct_created"
    EventTypeDirectDeleted     = "direct_deleted"
    EventTypeDirectRegenerated = "direct_regenerated"
)
```

**filters.go:**

```go
package direct

import "github.com/gofrs/uuid"

type FieldStruct struct {
    CustomerID   uuid.UUID `filter:"customer_id"`
    ResourceType string    `filter:"resource_type"`
    ResourceID   uuid.UUID `filter:"resource_id"`
    Hash         string    `filter:"hash"`
}
```

---

### Task 4: DBHandler

**Files:**
- Create: `bin-direct-manager/pkg/dbhandler/main.go`
- Create: `bin-direct-manager/pkg/dbhandler/db.go`
- Create: `bin-direct-manager/pkg/dbhandler/direct.go`
- Create: `bin-direct-manager/pkg/dbhandler/direct_test.go`

**main.go** — interface definition. Follow `bin-tag-manager/pkg/dbhandler/main.go` pattern:
- `DirectCreate(ctx, *direct.Direct) error`
- `DirectGet(ctx, id uuid.UUID) (*direct.Direct, error)`
- `DirectGetByHash(ctx, hash string) (*direct.Direct, error)`
- `DirectGets(ctx, pageSize uint64, pageToken string, filters map[direct.Field]any) ([]*direct.Direct, error)`
- `DirectDelete(ctx, id uuid.UUID) error` — hard delete
- `DirectUpdate(ctx, id uuid.UUID, fields map[direct.Field]any) error`

**direct.go** — table constant `directTable = "direct_directs"`. Follow squirrel + commondatabasehandler pattern from `bin-tag-manager/pkg/dbhandler/tag.go`. Key difference: `DirectDelete` uses `sq.Delete` (hard delete) instead of setting `tm_delete`.

**direct_test.go** — table-driven tests with gomock.

---

### Task 5: CacheHandler

**Files:**
- Create: `bin-direct-manager/pkg/cachehandler/main.go`
- Create: `bin-direct-manager/pkg/cachehandler/handler.go`
- Create: `bin-direct-manager/pkg/cachehandler/handler_test.go`

Follow `bin-tag-manager/pkg/cachehandler/` pattern.

**Key methods:**
- `DirectGetByHash(hash string) (*direct.Direct, error)` — Redis key: `direct:hash:<hash>`
- `DirectSetByHash(hash string, d *direct.Direct) error`
- `DirectDeleteByHash(hash string) error`

---

### Task 6: DirectHandler (Business Logic)

**Files:**
- Create: `bin-direct-manager/pkg/directhandler/main.go`
- Create: `bin-direct-manager/pkg/directhandler/handler.go`
- Create: `bin-direct-manager/pkg/directhandler/event.go`
- Create: `bin-direct-manager/pkg/directhandler/db.go`
- Create: `bin-direct-manager/pkg/directhandler/handler_test.go`

**main.go** — interface:

```go
type DirectHandler interface {
    Create(ctx context.Context, customerID uuid.UUID, resourceType string, resourceID uuid.UUID) (*direct.Direct, error)
    Get(ctx context.Context, id uuid.UUID) (*direct.Direct, error)
    GetByHash(ctx context.Context, hash string) (*direct.Direct, error)
    Gets(ctx context.Context, pageSize uint64, pageToken string, filters map[direct.Field]any) ([]*direct.Direct, error)
    Delete(ctx context.Context, id uuid.UUID) (*direct.Direct, error)
    Regenerate(ctx context.Context, id uuid.UUID) (*direct.Direct, error)

    EventCustomerDeleted(ctx context.Context, customerID uuid.UUID) error
}
```

**handler.go** — key logic:
- `Create`: generate hash using `crypto/rand` (6 bytes → hex → prepend `direct.`), retry 3x on collision, store in DB
- `GetByHash`: check Redis cache first, fall back to DB, populate cache on miss
- `Delete`: get the direct first, hard-delete from DB, invalidate cache
- `Regenerate`: generate new hash, update DB, invalidate old cache entry
- `EventCustomerDeleted`: delete all directs for the customer from DB

---

### Task 7: ListenHandler

**Files:**
- Create: `bin-direct-manager/pkg/listenhandler/main.go`
- Create: `bin-direct-manager/pkg/listenhandler/v1_directs.go`
- Create: `bin-direct-manager/pkg/listenhandler/v1_directs_test.go`
- Create: `bin-direct-manager/pkg/listenhandler/models/request/main.go`
- Create: `bin-direct-manager/pkg/listenhandler/models/request/v1_directs.go`

Follow `bin-tag-manager/pkg/listenhandler/main.go` pattern.

**Regex patterns:**

```go
regV1DirectsPost       = regexp.MustCompile(`/v1/directs$`)
regV1DirectsGet        = regexp.MustCompile(`/v1/directs\?`)
regV1DirectsByHashGet  = regexp.MustCompile(`/v1/directs/by-hash/`)
regV1DirectsIDGet      = regexp.MustCompile(`/v1/directs/[0-9a-f-]+$`)
regV1DirectsIDDelete   = regexp.MustCompile(`/v1/directs/[0-9a-f-]+$`)
regV1DirectsIDRegenerate = regexp.MustCompile(`/v1/directs/[0-9a-f-]+/regenerate$`)
```

**Request structs** in `models/request/v1_directs.go`:

```go
type V1DataDirectsPost struct {
    CustomerID   uuid.UUID `json:"customer_id"`
    ResourceType string    `json:"resource_type"`
    ResourceID   uuid.UUID `json:"resource_id"`
}
```

---

### Task 8: SubscribeHandler

**Files:**
- Create: `bin-direct-manager/pkg/subscribehandler/main.go`
- Create: `bin-direct-manager/pkg/subscribehandler/customer_manager.go`

Follow `bin-tag-manager/pkg/subscribehandler/` pattern. Subscribe to `QueueNameCustomerEvent`. On `customer_deleted` event, call `directHandler.EventCustomerDeleted(ctx, customerID)`.

---

### Task 9: Service Entry Point, Config, Dockerfile, go.mod

**Files:**
- Create: `bin-direct-manager/cmd/direct-manager/main.go`
- Create: `bin-direct-manager/cmd/direct-control/main.go`
- Create: `bin-direct-manager/internal/config/config.go`
- Create: `bin-direct-manager/Dockerfile`
- Create: `bin-direct-manager/go.mod`
- Create: `bin-direct-manager/go.sum`

Follow `bin-tag-manager/` as the reference implementation for all files.

**main.go** key differences from tag-manager:
- `serviceName = commonoutline.ServiceNameDirectManager`
- `QueueNameDirectRequest`, `QueueNameDirectEvent`, `QueueNameDirectSubscribe`
- `directHandler` instead of `tagHandler`

**go.mod** — module path `monorepo/bin-direct-manager`, Go 1.25.3, replace directives for all `bin-*` sibling modules. Copy from `bin-tag-manager/go.mod` and adjust module name. Then run `go mod tidy && go mod vendor`.

**Dockerfile** — copy from `bin-tag-manager/Dockerfile`, change binary name to `direct-manager`.

**Step: Verify the service builds and tests pass:**

```bash
cd bin-direct-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step:** Commit Phase 2.

---

## Phase 3: Common Handler — RequestHandler Methods

### Task 10: RequestHandler for Direct Manager

**Files:**
- Create: `bin-common-handler/pkg/requesthandler/direct_directs.go`
- Modify: `bin-common-handler/pkg/requesthandler/main.go` — add interface methods + `sendRequestDirect` + import `dmdirect`

**direct_directs.go:**

Methods to implement (follow `tag_tag.go` pattern):
- `DirectV1DirectCreate(ctx, customerID uuid.UUID, resourceType string, resourceID uuid.UUID) (*dmdirect.Direct, error)`
- `DirectV1DirectGet(ctx, id uuid.UUID) (*dmdirect.Direct, error)`
- `DirectV1DirectGetByHash(ctx, hash string) (*dmdirect.Direct, error)`
- `DirectV1DirectGets(ctx, pageToken string, pageSize uint64, filters map[dmdirect.Field]any) ([]*dmdirect.Direct, error)`
- `DirectV1DirectDelete(ctx, id uuid.UUID) (*dmdirect.Direct, error)`
- `DirectV1DirectRegenerate(ctx, id uuid.UUID) (*dmdirect.Direct, error)`

`sendRequestDirect` — private helper using `json.RawMessage`, sends to `QueueNameDirectRequest`.

**main.go** — add all 6 methods to the `RequestHandler` interface. Add import for `dmdirect "monorepo/bin-direct-manager/models/direct"`.

**Step:** Remove old methods:
- Delete `RegistrarV1ExtensionGetByDirectHash` from `registrar_extensions.go`
- Delete `RegistrarV1ExtensionDirectGetByHash` from `registrar_extensions.go`
- Remove corresponding interface entries in `main.go`
- Remove import of `rmextensiondirect`

**Step:** Verify:

```bash
cd bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step:** Commit Phase 3.

---

## Phase 4: Owning Service Updates

### Task 11: registrar-manager

**Files:**
- Modify: `bin-registrar-manager/models/extension/extension.go` — change `DirectHash` from `db:"-"` to `db:"direct_hash"`, add `DirectID uuid.UUID db:"direct_id,uuid"`
- Modify: `bin-registrar-manager/models/extension/webhook.go` — add `DirectHash` to `WebhookMessage`
- Modify: `bin-registrar-manager/pkg/extensionhandler/main.go` — remove `DirectEnable`, `DirectDisable`, `DirectRegenerate`, `GetDirectByHash`, `GetByDirectHash` methods. Remove `extensionDirectHandler` dependency.
- Modify: `bin-registrar-manager/pkg/extensionhandler/extension.go` — update `Create()` to call `reqHandler.DirectV1DirectCreate()` before insert. Update `Delete()` to call `reqHandler.DirectV1DirectDelete()`. Remove direct hash population from `Get()`/`List()`.
- Delete: `bin-registrar-manager/pkg/extensiondirecthandler/` (entire package)
- Delete: `bin-registrar-manager/models/extensiondirect/` (entire package)
- Modify: `bin-registrar-manager/pkg/listenhandler/main.go` — remove `/v1/extension-directs` and `/v1/extensions/by-direct-hash/` routes
- Delete: `bin-registrar-manager/pkg/listenhandler/v1_extension_directs.go`
- Modify: `bin-registrar-manager/pkg/dbhandler/main.go` — remove `ExtensionDirect*` methods from interface
- Delete: `bin-registrar-manager/pkg/dbhandler/extension_direct.go`
- Update tests accordingly

**Key code change in extension Create():**

```go
func (h *extensionHandler) Create(ctx context.Context, ...) (*extension.Extension, error) {
    // Generate extension ID
    id := uuid.Must(uuid.NewV4())

    // Create direct hash first
    d, err := h.reqHandler.DirectV1DirectCreate(ctx, customerID, "extension", id)
    if err != nil {
        return nil, fmt.Errorf("could not create direct hash: %w", err)
    }

    ext := &extension.Extension{
        // ... existing fields
        DirectID:   d.ID,
        DirectHash: d.Hash,
    }

    // Insert extension with direct hash populated
    if err := h.db.ExtensionCreate(ctx, ext); err != nil {
        // Cleanup orphaned direct
        if _, errDel := h.reqHandler.DirectV1DirectDelete(ctx, d.ID); errDel != nil {
            log.Errorf("Could not delete orphaned direct. err: %v", errDel)
        }
        return nil, err
    }
    // ...
}
```

**Step:** Verify:

```bash
cd bin-registrar-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

---

### Task 12: conference-manager

**Files:**
- Modify: `bin-conference-manager/models/conference/conference.go` — add `DirectID`, `DirectHash`
- Modify: `bin-conference-manager/models/conference/webhook.go` — add `DirectHash` to `WebhookMessage`
- Modify: `bin-conference-manager/pkg/conferencehandler/conference.go` — update `Create()` and `Delete()` to call direct-manager
- Update tests

Same pattern as registrar-manager Task 11.

**Step:** Verify conference-manager.

---

### Task 13: ai-manager (AI + Team)

**Files:**
- Modify: `bin-ai-manager/models/ai/main.go` — add `DirectID`, `DirectHash`
- Modify: `bin-ai-manager/models/team/main.go` — add `DirectID`, `DirectHash`
- Modify AI and Team webhook.go files — add `DirectHash`
- Modify AI and Team handler Create/Delete methods
- Update tests

**Step:** Verify ai-manager.

---

### Task 14: agent-manager

**Files:**
- Modify: `bin-agent-manager/models/agent/agent.go` — add `DirectID`, `DirectHash`
- Modify: `bin-agent-manager/models/agent/webhook.go` — add `DirectHash` to `WebhookMessage`
- Modify: `bin-agent-manager/pkg/agenthandler/agent.go` — update `Create()` and `Delete()` to call direct-manager
- Update tests

**Step:** Verify agent-manager.

**Step:** Commit Phase 4 (all owning service updates together or individually).

---

## Phase 5: Call-Manager Routing

### Task 15: Update call-manager direct routing

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip.go`
- Modify: `bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip_test.go`

**Key changes:**

Replace `startIncomingDomainTypeSIPDirectExtension()` with generic `startIncomingDomainTypeSIPDirect()`:

```go
func (h *callHandler) startIncomingDomainTypeSIPDirect(ctx context.Context, cn *channel.Channel, hash string) error {
    log := logrus.WithFields(logrus.Fields{
        "func": "startIncomingDomainTypeSIPDirect",
        "hash": hash,
    })

    // Resolve hash via direct-manager
    d, err := h.reqHandler.DirectV1DirectGetByHash(ctx, hash)
    if err != nil {
        log.Errorf("Could not resolve direct hash. err: %v", err)
        _ = h.channelHangup(ctx, cn.ID, channel.ChannelCauseNoRouteDestination)
        return err
    }
    log.WithField("direct", d).Debugf("Resolved direct hash. resource_type: %s, resource_id: %s", d.ResourceType, d.ResourceID)

    switch d.ResourceType {
    case "extension":
        return h.startIncomingDomainTypeSIPDirectExtension(ctx, cn, d)
    case "conference":
        return h.startIncomingDomainTypeSIPDirectConference(ctx, cn, d)
    case "agent":
        return h.startIncomingDomainTypeSIPDirectAgent(ctx, cn, d)
    case "ai":
        return h.startIncomingDomainTypeSIPDirectAI(ctx, cn, d)
    case "ai_team":
        return h.startIncomingDomainTypeSIPDirectAITeam(ctx, cn, d)
    default:
        log.Errorf("Unknown resource type. resource_type: %s", d.ResourceType)
        _ = h.channelHangup(ctx, cn.ID, channel.ChannelCauseNoRouteDestination)
        return fmt.Errorf("unknown resource type: %s", d.ResourceType)
    }
}
```

Each per-type handler follows existing patterns in `start_incoming_domain_type_registrar.go` (e.g., conference uses `ConferenceJoin` action, agent uses `Connect` action).

Update the main dispatch in `startIncomingDomainTypeSIP()`:

```go
// Old:
if strings.HasPrefix(cn.DestinationNumber, directExtensionPrefix) {
    hash := strings.TrimPrefix(cn.DestinationNumber, directExtensionPrefix)
    return h.startIncomingDomainTypeSIPDirectExtension(ctx, cn, hash)
}

// New — no prefix stripping:
if strings.HasPrefix(cn.DestinationNumber, dmdirect.DirectPrefix) {
    return h.startIncomingDomainTypeSIPDirect(ctx, cn, cn.DestinationNumber)
}
```

Remove the old `directExtensionPrefix` constant.

**Step:** Verify call-manager.
**Step:** Commit Phase 5.

---

## Phase 6: OpenAPI

### Task 16: OpenAPI Schema Updates

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` — add `direct_hash` to 5 resource schemas
- Remove: extension-direct endpoint schemas (if any)

Add to each resource schema:

```yaml
direct_hash:
  type: string
  description: Direct hash for SIP direct dialing
```

**Step:** Regenerate:

```bash
cd bin-openapi-manager && go generate ./...
cd bin-api-manager && go generate ./...
```

**Step:** Verify both services.
**Step:** Commit Phase 6.

---

## Phase 7: Final Verification

### Task 17: Cross-Service Verification

Run the full verification workflow for every changed service:

```bash
for svc in bin-direct-manager bin-common-handler bin-registrar-manager bin-conference-manager bin-ai-manager bin-agent-manager bin-call-manager bin-openapi-manager bin-api-manager; do
    echo "=== Verifying $svc ==="
    cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Generic-direct-hash-design/$svc
    go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
done
```

### Task 18: Final Commit and PR

Push branch and create PR following the monorepo commit conventions.
