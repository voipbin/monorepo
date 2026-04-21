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
| Internal RPC `CallV1CallsCreate` / `CallV1CallCreateWithID` | **Yes** — only internal services call this |

### Trust invariants (MANDATORY)

The following invariants MUST hold for every caller of `CallV1CallsCreate` and `CallV1CallCreateWithID`:

1. **No customer-derived input may flow into the `metadata` param.** Values coming from flow actions, API request bodies, queue callback payloads, or any other caller-controlled source are explicitly forbidden as metadata. Only server-side-derived values (e.g., a validated provider ID read from a trusted admin request, an internal feature flag) are permitted.
2. **Only typed `MetadataKey` constants are permitted as keys.** Declared in `bin-call-manager/models/call/metadata.go`. String literals at call sites are forbidden.
3. **Default is `nil`.** Services that have no reason to set metadata must pass `nil`. `bin-flow-manager`, `bin-queue-manager`, `bin-campaign-manager`, and the customer-facing `bin-api-manager` paths all pass `nil` in v1.

These invariants exist because `map[string]interface{}` is a loose contract. Forwarding caller-controlled values risks privilege escalation (e.g., a customer crafting a `route_provider_ids` value to bypass their assigned routes). The invariants establish the server-side-only boundary.

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
- **Synthetic route ID = provider ID.** To avoid generating throwaway UUIDs while preserving call-manager's failover tracking (which matches by `route.ID == c.DialrouteID`), synthetic routes use the `ProviderID` as their `Route.ID`. Each synthetic route is still uniquely identifiable, and the ID is human-traceable (the "route" being attempted is literally "provider X"). Duplicate provider IDs in the array are pathological input and are accepted as-is (the second duplicate becomes unreachable by the failover tracker, which is acceptable).
- **Provider existence validation.** Before generating synthetic routes, verify each provider ID exists (not soft-deleted) via `providerHandler.Get`. If any is missing, return an error immediately so the call fails fast rather than hanging up mid-dial.

### `bin-common-handler`

- **`pkg/requesthandler/call_calls.go`** — extend **both** `CallV1CallsCreate` and `CallV1CallCreateWithID` with `metadata map[string]interface{}`. Parity is required so groupcall fan-out and campaign calls can carry metadata when future features need it (v1 passes `nil`).
- **`pkg/requesthandler/route_dialroutes.go`** — extend `RouteV1DialrouteList` signature with `targetProviderIDs []uuid.UUID`.
- Regenerate mocks; update callers in the 7 affected services (enumerated below).

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

**Affected services (7):**
- `bin-common-handler` — signature sources (extends `CallV1CallsCreate`, `CallV1CallCreateWithID`, `RouteV1DialrouteList`).
- `bin-route-manager` — implements the synthetic-route override and provider-existence validation.
- `bin-call-manager` — new metadata key constant, metadata persistence at creation, metadata → `targetProviderIDs` extraction in `getDialroutes`, also updates `groupcallhandler` which calls `CallV1CallCreateWithID`.
- `bin-flow-manager` — caller of `CallV1CallsCreate` (2 sites); pass `nil` metadata.
- `bin-queue-manager` — caller of `CallV1CallsCreate`; pass `nil` metadata.
- `bin-api-manager` — caller of `CallV1CallsCreate`; pass `nil` metadata. (Admin endpoint is a separate feature.)
- `bin-campaign-manager` — caller of `CallV1CallCreateWithID`; pass `nil` metadata.

Full verification workflow required on every affected service.

## Out of Scope (follow-ups)

- Mutating metadata on existing calls (future: `POST /v1/calls/{id}/metadata`).
- Customer-facing metadata keys (future: whitelist mechanism).
- Metadata-driven feature flags beyond routing.
