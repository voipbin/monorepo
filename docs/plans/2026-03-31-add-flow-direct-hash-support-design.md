# Design: Add direct_hash Support to Flow

**Date:** 2026-03-31

## Problem Statement

The flow resource (`flow_flows`) does not support `direct_hash`. This means callers cannot dial `direct.<hash>@sip.voipbin.net` to trigger a flow directly. Adding `direct_hash` to flows enables customers to test flows via SIP and share direct SIP addresses that execute custom action sequences.

## Approach

Mirror the existing `direct_hash` pattern used by other resources (queue, agent, conference, etc.). Reuse existing `DirectV1DirectCreate`/`DirectV1DirectRegenerate` RPC calls.

**Auto-creation:** Hash is generated automatically when a flow is created, but only for `Type=TypeFlow` and `Persist=true`. Other flow types (conference, queue, campaign, transfer) are sub-flows that shouldn't be directly routable. Ephemeral/cached flows don't need direct hashes.

**Existing flows:** Columns added as nullable. Existing flows get their hash lazily on first `regenerate` call.

**Call routing:** When the call-manager resolves a direct hash to a flow, it calls `startCallTypeFlow()` directly — no temporary flow or prepended actions needed. The flow's action sequence executes as-is.

**Deletion cleanup:** Flow deletion (soft-delete) performs best-effort direct hash cleanup via `DirectV1DirectDelete()`, following the queue pattern.

## Changes

### 1. Database Migration (bin-dbscheme-manager)

New Alembic migration to add columns to `flow_flows`:
```sql
ALTER TABLE flow_flows ADD COLUMN direct_id binary(16), ADD COLUMN direct_hash varchar(255);
```
No backfill. Columns are nullable.

### 2. Flow Model (bin-flow-manager/models/flow/)

**flow.go** — Add fields:
```go
DirectID   uuid.UUID `json:"direct_id" db:"direct_id,uuid"`
DirectHash string    `json:"direct_hash" db:"direct_hash"`
```

**field.go** — Add constants:
```go
FieldDirectID   Field = "direct_id"
FieldDirectHash Field = "direct_hash"
```

**webhook.go** — Add to WebhookMessage:
```go
DirectHash string `json:"direct_hash,omitempty"`
```

### 3. DB Handler (bin-flow-manager/pkg/dbhandler/)

No special changes — `commondatabasehandler.PrepareFields` and `GetDBFields`/`ScanRow` handle new struct fields automatically via `db:` tags.

### 4. Flow Handler (bin-flow-manager/pkg/flowhandler/)

**New file: `direct_hash.go`** — Add `DirectHashRegenerate()` method:
- If flow has `DirectID` → call `DirectV1DirectRegenerate()`
- If flow has no `DirectID` → call `DirectV1DirectCreate(ctx, customerID, "flow", flowID)`
- Update flow with new `DirectID` + `DirectHash`
- Return updated flow

**Modify flow creation (`db.go`)** — After generating the flow ID, for `Type=TypeFlow && Persist=true`:
1. Call `DirectV1DirectCreate(ctx, customerID, "flow", flowID)` to get hash
2. Store `DirectID` and `DirectHash` in the flow struct before DB insert
3. If flow DB insert fails, clean up orphaned hash via `DirectV1DirectDelete()`

**Modify flow deletion** — Before soft-delete, if `DirectID != uuid.Nil`:
- Best-effort `DirectV1DirectDelete(ctx, directID)` — log error but don't block deletion

### 5. Listen Handler (bin-flow-manager/pkg/listenhandler/)

**New file: `v1_flows_direct_hash.go`** — Add endpoint:
- `PUT /v1/flows/{id}/direct-hash-regenerate`
- Extract flow ID from URI
- Call `flowHandler.DirectHashRegenerate(ctx, id)`
- Return updated flow as JSON

**Register route** in the router setup.

### 6. Call Manager Routing (bin-call-manager)

**File: `pkg/callhandler/start_incoming_domain_type_sip.go`** — Add `"flow"` case to the resource type switch:
```go
case "flow":
    return h.startIncomingDomainTypeSIPDirectFlow(ctx, cn, d, source)
```

**New file: `start_incoming_domain_type_sip_direct_flow.go`** — Handler that:
1. Fetches the flow via `FlowV1FlowGet(ctx, resourceID)` to validate it exists
2. Calls `startCallTypeFlow(ctx, cn, customerID, flowID, source, destination, nil)` directly
3. No temporary flow creation, no prepended actions — flow executes as-is

### 7. OpenAPI (bin-openapi-manager)

Add `direct_hash` field (string, optional) to the flow schema in `openapi.yaml`. Regenerate types in both `bin-openapi-manager` and `bin-api-manager`.

### 8. RST Documentation (bin-api-manager/docsdev/source/)

Update direct hash documentation to include "flow" as a supported resource type. Update any resource type tables or examples.

## Files Summary

| Service | File | Action |
|---------|------|--------|
| bin-dbscheme-manager | New migration | Create |
| bin-flow-manager | models/flow/flow.go | Edit |
| bin-flow-manager | models/flow/field.go | Edit |
| bin-flow-manager | models/flow/webhook.go | Edit |
| bin-flow-manager | pkg/flowhandler/direct_hash.go | Create |
| bin-flow-manager | pkg/flowhandler/db.go (Create + Delete) | Edit |
| bin-flow-manager | pkg/listenhandler/v1_flows_direct_hash.go | Create |
| bin-flow-manager | pkg/listenhandler/ (router) | Edit |
| bin-call-manager | pkg/callhandler/start_incoming_domain_type_sip.go | Edit |
| bin-call-manager | pkg/callhandler/start_incoming_domain_type_sip_direct_flow.go | Create |
| bin-openapi-manager | openapi/openapi.yaml | Edit |
| bin-api-manager | Regenerate via `go generate` | Regenerate |
| bin-api-manager | docsdev/source/ (RST docs) | Edit |

## Out of Scope

- Backfill migration for existing flows
- API validator tests (follow up)
