# Call Metadata: `route_provider_ids` for Internal Provider Override

**Date:** 2026-04-21
**Status:** Design approved, pending implementation plan
**Related:** `provider-test-call.prd.md` (replaces PRD Option B — dedicated RPC wrapper + singular `test_call_provider_id` — with Option A: general metadata pass-through + plural array)

## Problem

`bin-call-manager`'s `Call` model already has a `Metadata map[string]interface{}` field, currently used only for `rtp_debug`. The monorepo needs a general-purpose, internal-only mechanism for services to tag calls with information that influences downstream behavior — starting with the "provider test call" admin feature, which needs to override normal dialroute selection with a specific ordered list of provider IDs.

## Goals

1. Add a typed metadata key `route_provider_ids` (array of UUID strings) that `bin-call-manager` forwards to `bin-route-manager` during dialroute resolution.
2. `bin-route-manager` returns synthetic dialroutes in the order of the array (failover semantics: try provider 0, fall back to provider 1, etc.) — bypassing normal customer/default route merging when the override is present.
3. Provide a general metadata pass-through on `CallV1CallsCreate` so future internal features can reuse the pattern without new RPCs per feature.
4. Keep metadata strictly internal — no customer-facing write path.

## Non-Goals

- Customer-initiated metadata (no new customer API field).
- Runtime mutation of metadata after call creation.
- Migrating `rtp_debug` — it already uses this pattern and stays as-is.

## Architecture

Reuse the existing `Call.Metadata` field. No schema change, no new table.

```
admin → POST /v1/providers/{provider_id}/calls
  → bin-api-manager servicehandler (admin permission check only)
  → CallV1CallsCreate(ctx, ..., metadata={"route_provider_ids": ["<id>"]})
  → bin-call-manager persists Call with Metadata
  → bin-call-manager.getDialroutes reads metadata → targetProviderIDs
  → RouteV1DialrouteList(filters, targetProviderIDs)
  → bin-route-manager returns synthetic Routes in array order
  → bin-call-manager dials provider 0, fails over to provider 1, ...
```

### Trust model

| Path | Can set metadata? |
|---|---|
| Customer `POST /v1/calls` | **No** — field not exposed on public schema |
| Admin `POST /v1/providers/{id}/calls` | **No from client** — api-manager builds metadata server-side |
| Internal RPC `CallV1CallsCreate` | **Yes** — only internal services call this |

## Components

### `bin-call-manager`

- **`models/call/metadata.go`** — add:
  ```go
  const MetadataKeyRouteProviderIDs MetadataKey = "route_provider_ids"
  ```
- **`pkg/listenhandler`** (call create handler) — accept optional `metadata` field in the request body.
- **`pkg/callhandler`** (call creation) — merge caller-supplied metadata into `Call.Metadata` before insert, preserving already-set internal keys like `rtp_debug`.
- **`pkg/callhandler/outgoing_call.go`** — in `getDialroutes()`, extract `route_provider_ids` from `call.Metadata`, parse strings → `[]uuid.UUID`, pass to `RouteV1DialrouteList`.

### `bin-route-manager`

- **`pkg/routehandler/dialroute.go`** — extend `DialrouteList`: when `targetProviderIDs` non-empty, return `len(ids)` synthetic `Route` entries in array order; skip normal customer/default merge. When empty/absent, behavior unchanged.

### `bin-common-handler`

- **`pkg/requesthandler/call_calls.go`** — extend `CallV1CallsCreate` signature with `metadata map[string]interface{}`.
- **`pkg/requesthandler/route_dialroutes.go`** — extend `RouteV1DialrouteList` signature with `targetProviderIDs []uuid.UUID`.
- Regenerate mocks; update all callers across the 30+ services.

### `bin-api-manager`

- Add endpoint **`POST /v1/providers/{provider_id}/calls`** (replaces PRD's `POST /v1/providers/{id}/test-call`):
  - Admin permission check only (no customer/provider ownership check).
  - Server-side: `metadata = {"route_provider_ids": [provider_id]}` (array of one for v1).
  - Calls `CallV1CallsCreate` with metadata.
- OpenAPI spec update for the new admin endpoint. `Call.Metadata` is already in `WebhookMessage` — no schema change to the Call response.

### Docs

- **RST**: add a note under call-manager docs that `metadata` is internally populated and not part of the customer call-creation API.
- **`bin-api-manager/docsdev/source/`**: document the admin endpoint. Rebuild HTML, force-add `build/`.

## Data Flow & Edge Cases

| Case | Behavior |
|---|---|
| `route_provider_ids` absent or empty array | Today's behavior — normal dialroute merging |
| Array with 1 UUID | 1 synthetic route |
| Array with N UUIDs | N synthetic routes in order; normal failover applies |
| Invalid UUID string in array | Fail in api-manager before call creation (HTTP 400) |
| Unknown provider ID | route-manager returns error → call hangs up with `HangupReasonFailed` |
| All providers fail | Normal dialroute exhaustion → hangup |
| Metadata visible in webhook | Acceptable — admin is the owner of their test call |
| Concurrent writes | N/A — metadata set once at creation, read-only thereafter |

## Testing

**Unit:**
- `metadata.go` typed constant
- `getDialroutes()` correctly extracts and converts `route_provider_ids`
- route-manager synthetic route generator preserves array order and count
- route-manager error path on unknown provider

**Integration:**
- api-manager `POST /v1/providers/{id}/calls` rejects non-admin (403)
- End-to-end: admin call → dialroute uses specified provider

**api-validator:**
- Read-only tests for the new endpoint (no call creation — call-to-external-numbers cost policy).

## Security

- Admin permission check in api-manager.
- `MetadataKey` typed constants prevent string-key typos.
- No public write path for metadata.
- `Call.Metadata` exposed in `WebhookMessage` — admin-owned calls only see their own metadata.

## Trade-offs

| Decision | Alternative | Why |
|---|---|---|
| Option A (general metadata pass-through) | Option B from PRD (dedicated `CallV1CallsCreateForProviderTest` RPC) | Reusable for future internal features; higher one-time blast radius across 30+ services but zero per-feature cost thereafter |
| Plural `route_provider_ids` array | Singular `test_call_provider_id` | Supports failover testing in a single admin call; matches existing dialroute mental model |
| Failover semantics (return N synthetic routes in order) | Allowlist filter on normal merge | Simpler route-manager logic; mirrors existing failover behavior; explicit ordering is useful for testing |
| Admin permission only (no ownership check) | Per-customer provider ownership | Admin is testing, not customer-bound; simpler |

## Blast Radius

- `CallV1CallsCreate` signature change: every service that mocks it needs a mock regenerate + test fixture update (~15 services).
- `RouteV1DialrouteList` signature change: all callers must pass `nil`/empty for `targetProviderIDs` unless overriding (~3 services).
- Full verification workflow required on every affected service.

## Out of Scope (follow-ups)

- Mutating metadata on existing calls (future: `POST /v1/calls/{id}/metadata`).
- Customer-facing metadata keys (future: whitelist mechanism).
- Metadata-driven feature flags beyond routing.
