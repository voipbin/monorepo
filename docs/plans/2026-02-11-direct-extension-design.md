# Direct Extension Design

## Problem Statement

Currently, extensions can only receive calls from within the same customer's registrar domain (e.g., extension 1001 calling 1002 via `1002@{customer-id}.registrar.voipbin.net`). External calls from PSTN/trunks go through trunk domain routing, which routes to flows, not directly to extensions.

We need a way for external callers to reach an extension directly via a public SIP URI, bypassing the flow system.

## Approach

Introduce a "direct extension" feature. When enabled on an extension, the system generates a random hash and stores a mapping in a new `registrar_directs` table. The extension becomes reachable at `sip:direct.<hash>@sip.voipbin.net`.

The hash is opaque — it hides the real extension ID and customer ID from external callers.

## Scope

This design covers the registrar-manager side only:
- New data model and database table
- New handler for direct extension CRUD
- Integration with existing extension API (update, get, list, delete)
- New RPC endpoint for hash lookup

Out of scope (follow-up): Call-manager routing for `sip:direct.<hash>@sip.voipbin.net`.

## Data Model

### New Table: `registrar_directs`

```sql
create table registrar_directs(
    -- identity
    id            binary(16),
    customer_id   binary(16),

    extension_id  binary(16),
    hash          varchar(255),

    -- timestamps
    tm_create datetime(6),
    tm_update datetime(6),
    tm_delete datetime(6),

    primary key(id)
);

create unique index idx_registrar_directs_extension_id on registrar_directs(extension_id);
create unique index idx_registrar_directs_hash on registrar_directs(hash);
create index idx_registrar_directs_customer_id on registrar_directs(customer_id);
```

- Unique index on `extension_id` enforces one hash per extension.
- Unique index on `hash` enables fast lookup and prevents collisions.

### New Model: `extensiondirect.ExtensionDirect`

Location: `bin-registrar-manager/models/extensiondirect/extensiondirect.go`

```go
package extensiondirect

import (
    "time"

    commonidentity "monorepo/bin-common-handler/models/identity"

    "github.com/gofrs/uuid"
)

type ExtensionDirect struct {
    commonidentity.Identity

    ExtensionID uuid.UUID  `json:"extension_id" db:"extension_id,uuid"`
    Hash        string     `json:"hash" db:"hash"`

    TMCreate *time.Time `json:"tm_create" db:"tm_create"`
    TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
    TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}
```

### Field Types

Location: `bin-registrar-manager/models/extensiondirect/field.go`

```go
package extensiondirect

type Field string

const (
    FieldID          Field = "id"
    FieldCustomerID  Field = "customer_id"
    FieldExtensionID Field = "extension_id"
    FieldHash        Field = "hash"

    FieldTMCreate Field = "tm_create"
    FieldTMUpdate Field = "tm_update"
    FieldTMDelete Field = "tm_delete"

    // filter only
    FieldDeleted Field = "deleted"
)
```

### Filter Struct

Location: `bin-registrar-manager/models/extensiondirect/filters.go`

```go
package extensiondirect

import "github.com/gofrs/uuid"

type FieldStruct struct {
    ID          uuid.UUID `filter:"id"`
    CustomerID  uuid.UUID `filter:"customer_id"`
    ExtensionID uuid.UUID `filter:"extension_id"`
    Hash        string    `filter:"hash"`
    Deleted     bool      `filter:"deleted"`
}
```

### Extension Model Change

Add `DirectHash` to the existing Extension struct:

```go
type Extension struct {
    commonidentity.Identity

    // ... existing fields ...

    DirectHash string `json:"direct_hash" db:"-"` // populated from registrar_directs table

    // ... timestamps ...
}
```

The `db:"-"` tag means it is not stored in the extensions table. It gets populated at the application level from the `registrar_directs` table.

No Alembic migration needed for the extensions table itself.

## API Changes

### Extension Update (`PUT /extensions/{id}`)

Two new optional fields in the request body:

- `direct` (bool) - `true` to enable, `false` to disable
- `direct_regenerate` (bool) - `true` to regenerate a new hash

Behavior:

| Request | Current State | Result |
|---------|--------------|--------|
| `direct: true` | disabled | Generate hash, create row in `registrar_directs` |
| `direct: true` | enabled | No-op, hash unchanged |
| `direct: false` | enabled | Soft-delete row, `direct_hash` becomes empty |
| `direct: false` | disabled | No-op |
| `direct_regenerate: true` | enabled | Generate new hash, old SIP URI stops working |
| `direct_regenerate: true` | disabled | Error or no-op |

### Extension Response

All extension GET/LIST responses include a new read-only field:

- `direct_hash` (string) - the hash value when direct is enabled, empty string when disabled

The client constructs the full SIP URI: `sip:direct.<hash>@sip.voipbin.net`.

### Request Model Update

Update `bin-registrar-manager/pkg/listenhandler/models/request/extensions.go`:

```go
type V1DataExtensionsIDPut struct {
    Name             string `json:"name"`
    Detail           string `json:"detail"`
    Password         string `json:"password"`
    Direct           *bool  `json:"direct,omitempty"`
    DirectRegenerate *bool  `json:"direct_regenerate,omitempty"`
}
```

Using `*bool` so we can distinguish between "not provided" (nil) and "explicitly set to false".

### OpenAPI Spec Updates

- Add `direct_hash` (string, read-only) to `RegistrarManagerExtension` schema
- Add `direct` (bool) and `direct_regenerate` (bool) to extension update request schema

## New Handler: `extensiondirecthandler`

Location: `bin-registrar-manager/pkg/extensiondirecthandler/`

```go
type ExtensionDirectHandler interface {
    Create(ctx context.Context, customerID, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error)
    Delete(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error)
    Get(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error)
    GetByExtensionID(ctx context.Context, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error)
    GetByExtensionIDs(ctx context.Context, extensionIDs []uuid.UUID) ([]*extensiondirect.ExtensionDirect, error)
    GetByHash(ctx context.Context, hash string) (*extensiondirect.ExtensionDirect, error)
    Regenerate(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error)
}
```

- `Create` generates a random hash string (e.g., 12-char hex) and inserts a row.
- `Regenerate` generates a new hash and updates the existing row.
- Hash generation uses `crypto/rand` for secure random strings.

## New DB Handler Methods

Added to `pkg/dbhandler/main.go` interface:

```go
ExtensionDirectCreate(ctx context.Context, ed *extensiondirect.ExtensionDirect) error
ExtensionDirectDelete(ctx context.Context, id uuid.UUID) error
ExtensionDirectGet(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error)
ExtensionDirectGetByExtensionID(ctx context.Context, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error)
ExtensionDirectGetByExtensionIDs(ctx context.Context, extensionIDs []uuid.UUID) ([]*extensiondirect.ExtensionDirect, error)
ExtensionDirectGetByHash(ctx context.Context, hash string) (*extensiondirect.ExtensionDirect, error)
ExtensionDirectUpdate(ctx context.Context, id uuid.UUID, fields map[extensiondirect.Field]any) error
```

## Integration with Existing Extension Handlers

### Dependency Injection

The `extensionHandler` struct gains a new dependency:

```go
type extensionHandler struct {
    utilHandler            utilhandler.UtilHandler
    reqHandler             requesthandler.RequestHandler
    dbAst                  dbhandler.DBHandler
    dbBin                  dbhandler.DBHandler
    notifyHandler          notifyhandler.NotifyHandler
    extensionDirectHandler extensiondirecthandler.ExtensionDirectHandler
}
```

The `NewExtensionHandler` constructor adds the new parameter. The service initialization in `cmd/registrar-manager/main.go` creates the `extensionDirectHandler` and passes it in.

### extensionhandler.Update()

When extension update receives `direct` or `direct_regenerate` fields:
- `direct: true` -> call `extensionDirectHandler.Create(customerID, extensionID)`
- `direct: false` -> look up by extension ID, call `extensionDirectHandler.Delete(id)`
- `direct_regenerate: true` -> look up by extension ID, call `extensionDirectHandler.Regenerate(id)`

### extensionhandler.Get()

After fetching the extension, fetch the direct record in the handler layer:

```go
func (h *extensionHandler) Get(ctx context.Context, id uuid.UUID) (*extension.Extension, error) {
    ext, err := h.dbBin.ExtensionGet(ctx, id)
    if err != nil {
        return nil, err
    }

    direct, err := h.extensionDirectHandler.GetByExtensionID(ctx, ext.ID)
    if err == nil && direct != nil {
        ext.DirectHash = direct.Hash
    }

    return ext, nil
}
```

### extensionhandler.List()

After fetching extensions, fetch direct records for each. Use a batch method to avoid N+1 queries:

```go
func (h *extensionHandler) List(ctx context.Context, ...) ([]*extension.Extension, error) {
    exts, err := h.dbBin.ExtensionList(ctx, ...)
    if err != nil {
        return nil, err
    }

    // Collect extension IDs
    extIDs := make([]uuid.UUID, len(exts))
    for i, ext := range exts {
        extIDs[i] = ext.ID
    }

    // Batch fetch direct records
    directs, _ := h.extensionDirectHandler.GetByExtensionIDs(ctx, extIDs)
    directMap := make(map[uuid.UUID]string)
    for _, d := range directs {
        directMap[d.ExtensionID] = d.Hash
    }

    // Populate DirectHash
    for _, ext := range exts {
        if hash, ok := directMap[ext.ID]; ok {
            ext.DirectHash = hash
        }
    }

    return exts, nil
}
```

This requires an additional method on the handler and DB layer:
- `ExtensionDirectHandler.GetByExtensionIDs(ctx, extensionIDs []uuid.UUID) ([]*extensiondirect.ExtensionDirect, error)`
- `DBHandler.ExtensionDirectGetByExtensionIDs(ctx, extensionIDs []uuid.UUID) ([]*extensiondirect.ExtensionDirect, error)`

### extensionhandler.Delete()

When deleting an extension, also delete its direct record if one exists.

## New RPC Endpoint

Exposed via listen handler for future call-manager use:

- `GET /v1/extension-directs?hash=<hash>` - lookup by hash, returns `ExtensionDirect`

New request handler method in `bin-common-handler`:
- `RegistrarV1ExtensionDirectGetByHash(ctx, hash)` - cross-service RPC call

## Hash Generation

- Random 12-character hex string using `crypto/rand`
- Example: `a3f8b2c1d4e5`
- 48 bits of entropy (12 hex chars)
- Collision handling: retry up to 3 times on duplicate key error from the unique index, then return error
- Regeneratable: user can request a new hash, invalidating the old SIP URI

```go
func generateHash() (string, error) {
    b := make([]byte, 6) // 6 bytes = 12 hex chars
    _, err := crypto_rand.Read(b)
    if err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil
}
```

## Files to Create/Modify

### New Files
- `bin-registrar-manager/models/extensiondirect/extensiondirect.go` - model struct
- `bin-registrar-manager/models/extensiondirect/field.go` - field types
- `bin-registrar-manager/models/extensiondirect/filters.go` - filter struct for RPC query parsing
- `bin-registrar-manager/models/extensiondirect/webhook.go` - webhook message
- `bin-registrar-manager/models/extensiondirect/event.go` - event types
- `bin-registrar-manager/pkg/extensiondirecthandler/main.go` - handler interface
- `bin-registrar-manager/pkg/extensiondirecthandler/handler.go` - handler implementation
- `bin-registrar-manager/pkg/dbhandler/extension_direct.go` - DB operations
- `bin-registrar-manager/pkg/listenhandler/v1_extension_directs.go` - RPC endpoint
- `bin-dbscheme-manager/bin-manager/main/versions/<auto>_add_table_registrar_directs.py` - migration (hash auto-generated by `alembic revision`)

### Modified Files
- `bin-registrar-manager/models/extension/extension.go` - add `DirectHash` field
- `bin-registrar-manager/models/extension/webhook.go` - add `DirectHash` field
- `bin-registrar-manager/pkg/extensionhandler/main.go` - add ExtensionDirectHandler to struct and constructor
- `bin-registrar-manager/pkg/extensionhandler/extension.go` - integrate direct logic in Update/Get/List/Delete
- `bin-registrar-manager/pkg/dbhandler/main.go` - add ExtensionDirect DB methods to interface
- `bin-registrar-manager/pkg/listenhandler/main.go` - add regex pattern for new endpoint
- `bin-registrar-manager/pkg/listenhandler/models/request/extensions.go` - add `Direct` and `DirectRegenerate` fields to `V1DataExtensionsIDPut`
- `bin-registrar-manager/cmd/registrar-manager/main.go` - create and inject extensiondirecthandler
- `bin-openapi-manager/openapi/openapi.yaml` - add `direct_hash` to schema, update request schema
- `bin-common-handler/pkg/requesthandler/registrar_extensions.go` - add `RegistrarV1ExtensionDirectGetByHash`
