# Design: Add direct_hash Support to Queue

**Date:** 2026-03-31

## Problem Statement

The queue resource (`queue_queues`) does not support `direct_hash`, unlike the other 5 resources (agent, conference, extension, AI, AI team). This means callers cannot dial `direct.<hash>@sip.voipbin.net` to enter a queue directly. Adding `direct_hash` to queues enables consistent SIP routing and API behavior across all resources.

## Approach

Mirror the existing `direct_hash` pattern used by the other 5 resources. No new abstractions or shared code needed — reuse existing `DirectV1DirectCreate`/`DirectV1DirectRegenerate` RPC calls.

**Auto-creation:** Hash is generated automatically when a queue is created.
**Existing queues:** Columns added as nullable. Existing queues get their hash lazily on first `regenerate` call.

## Changes

### 1. Database Migration (bin-dbscheme-manager)

New Alembic migration to add columns to `queue_queues`:
```sql
ALTER TABLE queue_queues ADD COLUMN direct_id binary(16), ADD COLUMN direct_hash varchar(255);
```
No backfill. Columns are nullable.

### 2. Queue Model (bin-queue-manager/models/queue/)

**queue.go** — Add fields:
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

### 3. DB Handler (bin-queue-manager/pkg/dbhandler/)

No special changes expected — `commondatabasehandler.PrepareFields` and `GetDBFields`/`ScanRow` handle new struct fields automatically via `db:` tags.

### 4. Queue Handler (bin-queue-manager/pkg/queuehandler/)

**New file: `direct_hash.go`** — Add `DirectHashRegenerate()` method following the agent/conference/AI pattern:
- If queue has `DirectID` → call `DirectV1DirectRegenerate()`
- If queue has no `DirectID` → call `DirectV1DirectCreate(ctx, customerID, "queue", queueID)`
- Update queue with new `DirectID` + `DirectHash`
- Return updated queue

**Modify queue creation** — After creating the queue, call `DirectV1DirectCreate()` to auto-generate the hash, then update the queue with `DirectID`/`DirectHash`.

### 5. Listen Handler (bin-queue-manager/pkg/listenhandler/)

**New file: `v1_queues_direct_hash.go`** — Add endpoint:
- `PUT /v1/queues/{id}/direct-hash-regenerate`
- Extract queue ID from URI
- Call `queueHandler.DirectHashRegenerate(ctx, id)`
- Return updated queue as JSON

**Register route** in the router setup.

### 6. Call Manager Routing (bin-call-manager)

**File: `pkg/callhandler/start_incoming_domain_type_sip.go`** — Add `"queue"` case to the resource type switch:
```go
case "queue":
    return h.startIncomingDomainTypeSIPDirectQueue(ctx, cn, d)
```

**New file: `start_incoming_domain_type_sip_direct_queue.go`** — Create temporary flow with `Connect` action targeting the queue, following the extension/conference/AI pattern.

### 7. OpenAPI (bin-openapi-manager)

Add `direct_hash` field (string, optional) to the queue schema in `openapi.yaml`. Regenerate types.

## Files Summary

| Service | File | Action |
|---------|------|--------|
| bin-dbscheme-manager | New migration | Create |
| bin-queue-manager | models/queue/queue.go | Edit |
| bin-queue-manager | models/queue/field.go | Edit |
| bin-queue-manager | models/queue/webhook.go | Edit |
| bin-queue-manager | pkg/queuehandler/direct_hash.go | Create |
| bin-queue-manager | pkg/queuehandler/ (queue creation) | Edit |
| bin-queue-manager | pkg/listenhandler/v1_queues_direct_hash.go | Create |
| bin-queue-manager | pkg/listenhandler/ (router) | Edit |
| bin-call-manager | pkg/callhandler/start_incoming_domain_type_sip.go | Edit |
| bin-call-manager | pkg/callhandler/start_incoming_domain_type_sip_direct_queue.go | Create |
| bin-openapi-manager | openapi/openapi.yaml | Edit |
| bin-api-manager | Regenerate via `go generate` | Regenerate |

## Out of Scope

- Backfill migration for existing queues
- RST documentation updates (can follow up separately)
- API validator tests (follow up)
