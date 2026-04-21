# Provider Call

> **Revision note (2026-04-21):** This PRD was revised after PR [#793](https://github.com/voipbin/monorepo/pull/793) (`NOJIRA-call-metadata-route-provider-ids`, commit `2ebdf0b4e`) shipped the underlying metadata mechanism. Option A (general metadata pass-through on `CallV1CallsCreate`) was chosen over Option B (dedicated RPC) with improvements: **plural** `route_provider_ids` array (failover semantics), typed `MetadataKey` registry with runtime validation, and synthetic-route ID == provider ID. The `ProviderCall` entity has been **reinstated** as a first-class record in `bin-route-manager` — it captures the admin's request info and the IDs of the created calls/groupcalls, and is the response shape of the admin endpoint.

## Problem Statement

Project admins cannot verify a provider's end-to-end call routing until real customer traffic flows through it. New-provider onboarding, routing debugging, and post-config-change verification all rely on waiting for a production call to fail. The underlying plumbing to force a call through a specific provider now exists (PR #793), but there is still no external API surface exposing it to admins.

## Evidence

- `Call.Metadata` now carries typed keys via `bin-call-manager/models/call/metadata.go`. `MetadataKeyRouteProviderIDs = "route_provider_ids"` is declared (lines 10-17), registered in `ValidMetadataKeys` (lines 28-31), and enforced at the call-manager listen handler (HTTP 400 on unknown keys).
- `bin-common-handler/pkg/requesthandler/call_calls.go` — `CallV1CallsCreate` and `CallV1CallCreateWithID` now accept `metadata map[string]interface{}`. `bin-flow-manager`, `bin-queue-manager`, `bin-campaign-manager`, and `bin-api-manager`'s customer-facing paths pass `nil`.
- `bin-common-handler/pkg/requesthandler/route_dialroutes.go` — `RouteV1DialrouteList` accepts `targetProviderIDs []uuid.UUID`. When non-empty, `bin-route-manager` returns `len(ids)` synthetic routes in array order (synthetic route ID == provider ID) and validates provider existence.
- `bin-call-manager/pkg/callhandler/outgoing_call.go` `getDialroutes()` extracts `route_provider_ids` from `Call.Metadata`, parses strings → `[]uuid.UUID`, and forwards to `RouteV1DialrouteList`.
- Design + plan published: `docs/plans/2026-04-21-call-metadata-route-provider-ids-design.md`, `docs/plans/2026-04-21-call-metadata-route-provider-ids-plan.md`.
- Remaining gap: no admin endpoint exists on `bin-api-manager` to populate `route_provider_ids`, so the mechanism is unreachable from outside the monorepo today.

## Proposed Solution

Add an admin-only endpoint `POST /v1/providercalls` on `bin-api-manager`. The handler:

1. Checks `PermissionProjectSuperAdmin`.
2. Takes `customer_id` from the auth context (`a.CustomerID`) — not from the body.
3. Validates `provider_id` (from body) exists.
4. Builds metadata **server-side**: `{"route_provider_ids": [<provider_id from body>], "skip_source_validation": true}`.
5. Calls `CallV1CallsCreate(..., metadata=...)` and collects resulting `calls` / `groupcalls` IDs.
6. Calls `RouteV1ProviderCallCreate(...)` to persist a `ProviderCall` record in `bin-route-manager` capturing: admin's request info + created call IDs + created groupcall IDs.
7. Returns the `ProviderCall.WebhookMessage`.

New work in this PRD:
- `ProviderCall` entity in `bin-route-manager` (model, Alembic migration, dbhandler, RabbitMQ handlers, RPC wrappers in `bin-common-handler`).
- `skip_source_validation` metadata key in `bin-call-manager` (Phase 1).
- Admin endpoint + OpenAPI in `bin-api-manager`.
- RST docs + api-validator tests.

The synthetic-route flow and metadata pass-through plumbing are already live (PR #793).

## Motivation

Giving project admins an API to force a call through a specific provider gives them a direct observation tool for onboarding, post-config-change verification, and routing debugging — eliminating the wait for real customer traffic. The endpoint **triggers** a real outbound call; it does not produce a pass/fail verdict. Admins observe the resulting `Call` (via the existing Call API, webhooks, or downstream observability) and apply their own judgment.

The feature succeeds when admins adopt it for provider-verification tasks in place of waiting for customer complaints. That adoption is the signal; there is no automated measure of correctness because correctness is the admin's interpretation.

## What We're NOT Building (v1)

- Admin console UI — lives in a separate repo and will be coordinated separately.
- Automatic provider health scoring / auto-disable on failed tests.
- Customer-tier-user access — admin-only feature.
- Scheduled / recurring provider tests.
- Cost controls or per-call spending caps beyond existing outbound rate limits.
- Automated pass/fail interpretation of the call outcome. This endpoint triggers a call; admins observe the result themselves.
- Multi-provider failover testing via the admin endpoint (N-element array). The underlying mechanism supports it; v1 exposes a 1-element array only.
- Storing inline `actions` on the `ProviderCall` record. Only `flow_id` is stored; inline actions are captured on the created `Call` records and can be retrieved via `CallIDs`.
- Reverse lookup on `Call.Metadata` back to a `ProviderCall.ID`. V1 does not add a `provider_call_id` metadata key; reverse-direction is via `ProviderCall.CallIDs`.

## Success Metrics

The endpoint is a tool, not a verdict generator. There is no pass/fail result to measure. The only quantitative metric is operational health.

| Metric | Target | How Measured |
|--------|--------|--------------|
| Endpoint availability | ≥ 99.9% (matches existing `bin-api-manager` SLO) | Existing Prometheus / API monitoring |

Call outcomes (status, hangup reason, duration, routing trace) are observed by admins via the existing `Call` API and webhook flow and are not aggregated into a feature-level metric.

## Open Questions

All resolved. See the Decisions Log for the full list with alternatives and rationale.

---

## Users & Context

**Primary User**
- **Who**: Platform project admin (agent carrying `PermissionProjectSuperAdmin`). Cross-customer scope — can see and test every provider in the system.
- **Current behavior**: After configuring a provider or adjusting routing, waits for real traffic and watches dashboards for failures.
- **Trigger**: (a) onboarding a new provider, (b) debugging a routing issue, (c) verifying after a config change.
- **Success state**: One API call → triggers the call → observes the resulting `Call` via `GET /v1/calls/{id}` → moves on within a minute. The endpoint itself does not interpret the outcome.

**Job to Be Done**
> When the admin needs to verify a provider, I want to trigger a real call through that specific provider with admin-chosen source and destination, so I can confirm the provider is working without waiting for production traffic.

**Non-Users**
All customer-tier roles (`PermissionCustomerAdmin`, `PermissionCustomerManager`, `PermissionCustomerAgent`). End-users are not in scope. Only `PermissionProjectSuperAdmin` can invoke this endpoint.

---

## Solution Detail

### Core Capabilities (MoSCoW)

| Priority | Capability | Status |
|----------|------------|--------|
| Must | `POST /v1/providercalls` (admin-only): populates `Metadata.route_provider_ids=[provider_id]` and `Metadata.skip_source_validation=true`, calls `CallV1CallsCreate`, persists a `ProviderCall`, returns `ProviderCall.WebhookMessage` | **To build** |
| Must | `ProviderCall` model, Alembic migration, dbhandler, RabbitMQ handlers in `bin-route-manager` | **To build** |
| Must | `RouteV1ProviderCallCreate` / `RouteV1ProviderCallGet` / `RouteV1ProviderCallGets` / `RouteV1ProviderCallDelete` RPC wrappers in `bin-common-handler` | **To build** |
| Must | `skip_source_validation` typed metadata key + runtime registry + bypass in `getValidatedSourceForOutgoingCall` | **To build** |
| Must | OpenAPI schema: four endpoints (POST / GET list / GET detail / DELETE) under `/providercalls` + `RouteManagerProviderCall` schema | **To build** |
| Must | RST doc updates (admin endpoint + ProviderCall struct + extended internal-metadata note) | **To build** |
| Must | api-validator tests (read-only — cost-safety rule forbids creating real calls in the validator) | **To build** |
| Must | `targetProviderIDs` on `DialrouteList`, synthetic-route generation, `route_provider_ids` key + registry, `metadata` param on `CallV1CallsCreate` / `CallV1CallCreateWithID`, `getDialroutes` extracts `route_provider_ids` | ✅ Done (PR #793) |
| Should | `GET /v1/providercalls`, `GET /v1/providercalls/{id}`, `DELETE /v1/providercalls/{id}` for retrieval and cleanup of past `ProviderCall` records | **To build** |
| Won't (v1) | Synchronous SIP-response-code pass/fail in response body | Call lifecycle is async; admin polls `GET /v1/calls/{id}` |
| Won't (v1) | `actions` inline replay from the `ProviderCall` record | Captured on Call records; derive via `CallIDs` if needed |
| Won't (v1) | UI | External repo |

### MVP Scope

`POST /v1/providercalls` request body: **required** `provider_id` plus the customer call API shape — `source`, `destinations`, optional `flow_id` / `actions` / `anonymous`. Gated by `PermissionProjectSuperAdmin`. `customer_id` is derived from the auth context (JWT or accesskey) — not part of the request body. Handler builds metadata `{"route_provider_ids": [<provider_id from body>], "skip_source_validation": true}` **server-side**, calls `CallV1CallsCreate` with `a.CustomerID`, then calls `RouteV1ProviderCallCreate`. Returns the `ProviderCall.WebhookMessage`. Admin polls `GET /v1/calls/{id}` for per-call outcome; `GET /v1/providercalls/{id}` for the `ProviderCall` record.

### ProviderCall (new entity in `bin-route-manager`)

```go
type ProviderCall struct {
    ID           uuid.UUID           `db:"id,uuid"`

    // Requested
    CustomerID   uuid.UUID           `db:"customer_id,uuid"`
    ProviderID   uuid.UUID           `db:"provider_id,uuid"`
    FlowID       uuid.UUID           `db:"flow_id,uuid"`     // uuid.Nil when not provided
    Source       *commonaddress.Address `db:"source,json"`
    Destinations []commonaddress.Address `db:"destinations,json"`
    Anonymous    string              `db:"anonymous"`

    // Created
    CallIDs      []uuid.UUID         `db:"call_ids,json"`
    GroupcallIDs []uuid.UUID         `db:"groupcall_ids,json"`

    TMCreate     string              `db:"tm_create"`
    TMUpdate     string              `db:"tm_update"`
    TMDelete     string              `db:"tm_delete"`
}
```

`WebhookMessage` variant exposes all fields above (nothing infrastructure-only). `Actions` are not stored on the record — they are captured on each created `Call` and accessible via `CallIDs`. (Re-examine if admins report wanting replay of inline actions.)

**Deletion semantics**: soft-delete via `tm_delete`. `DELETE /v1/providercalls/{id}` returns the deleted `ProviderCall.WebhookMessage` (consistent with the monorepo convention used by Call / Conference / Agent deletes).

**`GET /v1/providercalls` filters**:
- Pagination: `page_size`, `page_token` (standard monorepo pattern).
- Scope: implicitly `customer_id = a.CustomerID` (admin's own customer, from auth). Every `ProviderCall` belongs to its creator's customer, so listing is naturally scoped to the admin's account.
- Optional: `provider_id` (narrow to a single provider).
- Dialect: URL query params for scalar filters; follow the monorepo's `Parsing Filters from Request Body` pattern only if pagination/filter surface grows.

### User Flow

1. Admin authenticates with `PermissionProjectSuperAdmin`.
2. Admin calls `POST /v1/providercalls` with required `provider_id` plus the customer call API shape (`source`, `destinations`, optional `flow_id`/`actions`/`anonymous`). `customer_id` comes from the auth context, not the body.
3. `bin-api-manager` handler verifies permission; validates that `provider_id` exists (via route-manager); builds `metadata = {"route_provider_ids": [provider_id], "skip_source_validation": true}` server-side.
4. Handler calls `CallV1CallsCreate(ctx, a.CustomerID, flowID, uuid.Nil, source, destinations, false, false, anonymous, metadata)`; receives `([]*Call, []*Groupcall, error)`.
5. `bin-call-manager` persists each Call with Metadata. `getDialroutes` extracts `route_provider_ids` and calls `RouteV1DialrouteList(..., targetProviderIDs=[provider_id])`. `getValidatedSourceForOutgoingCall` sees `skip_source_validation` and preserves the admin-supplied source.
6. `bin-route-manager` validates the provider exists and returns a synthetic dialroute (ID == provider_id). `bin-call-manager` dials through.
7. Handler extracts `call_ids` and `groupcall_ids` from the create result.
8. Handler calls `RouteV1ProviderCallCreate(ctx, a.CustomerID, providerID, flowID, source, destinations, anonymous, callIDs, groupcallIDs)`; receives the `ProviderCall` record.
9. API responds with `ProviderCall.WebhookMessage`.
10. Admin polls `GET /v1/calls/{id}` for per-call progress and `GET /v1/providercalls/{id}` for the `ProviderCall` record.

---

## Technical Approach

**Feasibility**: MEDIUM. Core call-forcing plumbing is merged (PR #793), but this PRD adds a new entity (`ProviderCall`) + Alembic migration + RPC wrappers + a call-manager metadata key.

**Architecture Notes**

- New DB table `provider_call` in `bin-route-manager`. Alembic migration via `bin-dbscheme-manager` (file generated via `alembic revision`, never hand-rolled). Follow monorepo conventions: `db:` tags, UUID tag (`,uuid`), JSON tag (`,json`) for `source` / `destinations` / `call_ids` / `groupcall_ids`, `Field` type for type-safe updates, empty-slice init on list functions.
- `ProviderCall` model files: `bin-route-manager/models/providercall/providercall.go`, `field.go`, `webhook.go`. Business handler in `pkg/providercallhandler/`. dbhandler methods added to `pkg/dbhandler/providercall.go` and `DBHandler` interface. RabbitMQ routes in the listen handler.
- `bin-common-handler` gets new wrappers: `RouteV1ProviderCallCreate`, `RouteV1ProviderCallGet`, `RouteV1ProviderCallGets`, `RouteV1ProviderCallDelete`. Additive — no existing RPC signatures change.
- `bin-call-manager` gets one new metadata constant (`MetadataKeySkipSourceValidation`) + one branch in `getValidatedSourceForOutgoingCall`. No schema change.
- `bin-api-manager` orchestrates: check permission → read `customer_id` from auth → validate `provider_id` (via route-manager) → build metadata → `CallV1CallsCreate` → `RouteV1ProviderCallCreate` → return `ProviderCall.WebhookMessage`.
- Services touched: `bin-call-manager`, `bin-route-manager`, `bin-dbscheme-manager`, `bin-common-handler`, `bin-api-manager`, `bin-openapi-manager`, `bin-api-manager/docsdev/`, `monorepo-monitoring/api-validator/`.
- Admin permission check uses the existing pattern: `h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionProjectSuperAdmin)`.
- **Customer attribution**: `customer_id` is derived from the auth context (`a.CustomerID` — the project admin's own customer, carried in the JWT or accesskey). Not accepted in the request body. `CallV1CallsCreate` is invoked with `a.CustomerID`. No additional customer-validation RPC needed — auth middleware already ensures the agent/customer is valid.
- Metadata is built **server-side only** — the endpoint must not accept a `metadata` field in its request body. This preserves the trust invariant from the design doc: customer-derived input never flows into `Metadata`.
- **Response unpacking**: `CallV1CallsCreate` returns `([]*Call, []*Groupcall, error)`. Handler collects the `[]uuid.UUID` of each slice's IDs and passes them to `RouteV1ProviderCallCreate`. If `CallV1CallsCreate` fails, abort before creating the `ProviderCall` record (no orphaned records).
- **Source-validation bypass**: handler also sets `Metadata[MetadataKeySkipSourceValidation] = true` so `bin-call-manager`'s `getValidatedSourceForOutgoingCall` (in `pkg/callhandler/outgoing_call.go`) preserves the admin-supplied source verbatim instead of silently falling back to the customer's `DefaultOutgoingSourceNumberID` when the source isn't owned by the customer. This is required because providers commonly reject INVITEs whose `From` / `P-Asserted-Identity` doesn't match a pre-allowed caller ID — the admin typically supplies a source matching that allowlist, not a number owned by an arbitrary customer.
- **ProviderCall failure semantics**: if the `Call` creation succeeds but the `ProviderCall` persistence fails, the Call records exist without a `ProviderCall` record. Handler should log an error and return 500 with guidance to retrieve via `GET /v1/calls` (auth-scoped to the admin's customer). Worth calling out as a risk; v1 accepts this trade-off rather than compensating with a rollback of the created calls.
- OpenAPI: add a new `ProviderCall` tag (separate from the existing `Provider` tag) grouping the four paths. Add a new `RouteManagerProviderCall` schema. No changes to `CallManagerCall`.
- RST: new top-level `providercall_*.rst` pages per Phase 5 (overview + tutorial + struct). RST rebuild per monorepo rule (`rm -rf build && python3 -m sphinx -M html source build` → force-add `build/`).
- api-validator: read-only tests only (e.g., verify `POST` returns 403 for non-admin, verify `POST` with invalid provider_id returns an expected error). Do not create real calls per the cost-safety rule.

**Technical Risks**

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Admin triggers a provider call to an unintended destination (billable) | L | Admin explicitly supplies destination; documented behavior; existing outbound rate-limit applies |
| `route_provider_ids` visible in the targeted customer's `Call.WebhookMessage` / webhook stream | L | Customer sees a Call with internal provider UUIDs. Provider UUIDs are not secrets (already retrievable via `/v1/providers`). If concealment becomes a requirement later, strip the `route_provider_ids` key from `WebhookMessage` (not `Call`) |
| Admin supplies a non-existent or invalid `provider_id` | M | Validate the provider exists (via `bin-route-manager` RPC) before invoking `CallV1CallsCreate`; return 404 / 400 if absent |
| `Call` creation succeeds but `ProviderCall` persistence fails — orphaned Calls without a `ProviderCall` record | L | Admin can still find the Calls via `GET /v1/calls` (auth-scoped to the admin's customer); v1 logs the failure and returns 500. Revisit if orphans become a practical problem (compensating delete of the Call would complicate the happy path) |
| Alembic migration for `provider_call` table not applied before api-manager deploy | M | Standard monorepo policy: migration file in the PR; user applies `alembic upgrade` manually with VPN access; api-manager deploy should follow the migration |
| OpenAPI-spec change not regenerated into `bin-api-manager` server code | M | Standard monorepo rule: regenerate **both** `bin-openapi-manager` and `bin-api-manager` after schema edits |
| RST build output not committed (gitignored) | M | Standard rule: `git add -f bin-api-manager/docsdev/build/` |
| api-validator accidentally creates a real call through a real carrier | M | Rule-enforced: only read-only / negative-path tests for this endpoint; no positive-path call creation |

---

## Implementation Phases

<!--
  STATUS: pending | in-progress | complete
  PARALLEL: phases that can run concurrently
  DEPENDS: phases that must complete first
  PRP: link to generated plan file once created
-->

| # | Phase | Description | Status | Parallel | Depends | PRP Plan |
|---|-------|-------------|--------|----------|---------|----------|
| 0 | Metadata plumbing (route-manager, call-manager, common-handler) | `route_provider_ids` metadata key, `targetProviderIDs` on DialrouteList, synthetic routes, metadata pass-through on `CallV1CallsCreate` | ✅ complete | - | - | PR #793 |
| 1 | `bin-call-manager` source-validation bypass | New `MetadataKeySkipSourceValidation` constant, add to `ValidMetadataKeys` registry, early-return in `getValidatedSourceForOutgoingCall` when set | pending | with 2 | 0 | - |
| 2 | `bin-dbscheme-manager` migration + `bin-route-manager` `ProviderCall` entity | Alembic migration for `provider_call` table; model (`models/providercall/providercall.go`, `field.go`, `webhook.go`); `pkg/dbhandler/providercall.go`; business handler (`pkg/providercallhandler/`); listen-handler routes (`POST /v1/providercalls`, `GET /v1/providercalls`, `GET /v1/providercalls/{id}`, `DELETE /v1/providercalls/{id}`) | pending | with 1 | 0 | - |
| 3 | `bin-common-handler` RPC wrappers | `RouteV1ProviderCallCreate`, `RouteV1ProviderCallGet`, `RouteV1ProviderCallGets`, `RouteV1ProviderCallDelete`; regenerate mocks; verify all 30+ services build | pending | - | 2 | - |
| 4 | `bin-api-manager` endpoint + OpenAPI | `POST /v1/providercalls` handler (checks `PermissionProjectSuperAdmin`, takes `customer_id` from auth context, validates `provider_id`, builds both metadata keys, calls `CallV1CallsCreate`, calls `RouteV1ProviderCallCreate`, returns `ProviderCall.WebhookMessage`); `GET /v1/providercalls`, `GET /v1/providercalls/{id}`, `DELETE /v1/providercalls/{id}`; OpenAPI additions (`RouteManagerProviderCall` schema + four paths); regenerate both `bin-openapi-manager` and `bin-api-manager` | pending | - | 1, 3 | - |
| 5 | RST docs | New admin endpoint tutorial + new struct page for `ProviderCall` in `bin-api-manager/docsdev/source/`; extend the internal-metadata note to cover `skip_source_validation`; clean rebuild HTML; force-add `build/` | pending | with 6 | 4 | - |
| 6 | api-validator tests | Read-only / negative-path tests for `POST /v1/providercalls` and `GET /v1/providercalls[/{id}]` in `monorepo-monitoring/api-validator/` | pending | with 5 | 4 | - |

### Phase Details

**Phase 1: `bin-call-manager` source-validation bypass**
- Goal: Let server-side trusted code opt-out of the silent source-fallback behavior so admin-supplied source numbers reach the provider verbatim.
- Scope:
  - `models/call/metadata.go` — declare `MetadataKeySkipSourceValidation` constant; add to `ValidMetadataKeys`; document as **creation-time only**, **server-side only**.
  - `pkg/callhandler/outgoing_call.go` — in `getValidatedSourceForOutgoingCall`, check the metadata key early; when true, return the supplied source without ownership lookup or default-number fallback (keep the minimal E.164 sanity check so downstream callers don't get an empty source).
  - Unit tests: metadata-absent (existing behavior), metadata-true + owned source, metadata-true + unowned source, metadata-true + malformed source.
- Success signal: `bin-call-manager` verification workflow green; existing source-validation tests still pass.

**Phase 2: `bin-dbscheme-manager` migration + `bin-route-manager` `ProviderCall` entity**
- Goal: First-class entity that wraps a provider-call request plus the resulting call / groupcall IDs.
- Scope:
  - `bin-dbscheme-manager` — new Alembic migration generated via `alembic revision` (never hand-rolled) creating a `provider_call` table with columns for all `ProviderCall` fields. Implement both `upgrade()` and `downgrade()`; `tm_delete` default `"9999-01-01 00:00:00.000000"` for active rows.
  - `bin-route-manager/models/providercall/providercall.go`, `field.go`, `webhook.go` — model + typed `Field` enum + external-facing `WebhookMessage` variant.
  - `bin-route-manager/pkg/dbhandler/providercall.go` — squirrel-based CRUD using `commondatabasehandler.PrepareFields` / `ScanRow`; added to the `DBHandler` interface; empty-slice init on list.
  - `bin-route-manager/pkg/providercallhandler/` — business handler.
  - `bin-route-manager/pkg/listenhandler/` — RabbitMQ routes `POST /v1/providercalls`, `GET /v1/providercalls`, `GET /v1/providercalls/{id}`, `DELETE /v1/providercalls/{id}`.
- Success signal: `bin-route-manager` verification workflow green; dbhandler and handler unit tests pass.

**Phase 3: `bin-common-handler` RPC wrappers**
- Goal: Expose the new route-manager RPCs to all consumers.
- Scope: `bin-common-handler/pkg/requesthandler/` — add `RouteV1ProviderCallCreate`, `RouteV1ProviderCallGet`, `RouteV1ProviderCallGets`, `RouteV1ProviderCallDelete`; regenerate mocks.
- Blast radius: additive-only signatures, but the full verification workflow must run on `bin-common-handler` AND every service that imports it (30+). Most of the wall-clock is the cross-service verification, not the wrapper code itself.
- Success signal: full verification workflow (`go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run`) passes on `bin-common-handler` AND every service that imports it.

**Operator step (manual, between Phase 2 and Phase 4)**
- Apply the Alembic migration from Phase 2 to the target environments: `alembic upgrade head` in `bin-dbscheme-manager` (requires VPN + manual authorization per monorepo policy). Phase 4's end-to-end verification depends on the `provider_call` table existing at runtime. Phases 2 and 3 can still land as code without the migration being applied; only Phase 4 verification is blocked.

**Phase 4: `bin-api-manager` endpoint + OpenAPI**
- Goal: Public HTTP surface for provider calls and retrieval.
- Concurrency note: Phase 4 implementation can be written in parallel with Phase 3 by mocking the new `RouteV1ProviderCall*` RPCs; only Phase 4 **verification** (and merge) requires Phase 3 landing first so the mocks line up with real signatures.
- Scope:
  - `pkg/servicehandler/` — new methods for `ProviderCallCreate`, `ProviderCallGet`, `ProviderCallGets`, `ProviderCallDelete`. The create method checks `PermissionProjectSuperAdmin`, takes `customer_id` from `a.CustomerID`, validates `provider_id` (from body), builds metadata with **both** keys, calls `CallV1CallsCreate`, then `RouteV1ProviderCallCreate`, returns `ProviderCall.WebhookMessage`. All Get/Delete methods also require `PermissionProjectSuperAdmin`.
  - `server/` — route registrations.
  - `bin-openapi-manager/openapi/openapi.yaml` — new paths `/providercalls` (GET list, POST create) and `/providercalls/{id}` (GET, DELETE); modular path files under `paths/providercalls/` (`main.yaml`, `id.yaml`). New `RouteManagerProviderCall` schema that mirrors the `WebhookMessage`. Follow AI-Native spec rules (realistic examples, `format: uuid` on ID fields, `minItems: 1` on `destinations`, etc.).
  - Regenerate `go generate ./...` in both `bin-openapi-manager` and `bin-api-manager`.
- Success signal: endpoints callable in a local dev environment; unit tests pass for permission, `provider_id` validation, metadata construction, and providercall orchestration; both services' verification workflows green.

**Phase 5: RST docs**
- Goal: Public docs reflect the new admin endpoint and the `ProviderCall` struct.
- Scope: edit `bin-api-manager/docsdev/source/` — new top-level `providercall_*.rst` pages (overview + tutorial + struct) plus an index entry; extend the internal-metadata note (added by PR #793) to mention `skip_source_validation`. RST struct docs must match `ProviderCall.WebhookMessage` field-for-field.
- Success signal: `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build` succeeds with no errors; `git add -f bin-api-manager/docsdev/build/` committed alongside source changes.

**Phase 6: api-validator tests**
- Goal: Regression coverage without incurring real-call cost.
- Scope: `monorepo-monitoring/api-validator/` — tests that verify:
  - Non-admin gets 403 on POST / GET list / GET by ID / DELETE.
  - POST with missing, malformed, unknown, or soft-deleted `provider_id` returns the expected error (soft-deleted requires that `ProviderGet` filters `tm_delete`, which the synthetic-route flow relies on).
  - GET list / GET by ID return the expected schema.
  - DELETE on an unknown ID returns the expected error; DELETE response shape matches `ProviderCall.WebhookMessage`.
  - OpenAPI schema conformance for request/response shape.
- Explicitly out: any test that creates a real call (per `monorepo/CLAUDE.md` cost-safety rule).
- Success signal: api-validator CI green.

### Parallelism Notes

- Phase 0 is done.
- Phases 1 and 2 are independent and run concurrently (bin-call-manager vs. bin-route-manager + migration).
- Phase 3 (common-handler wrappers) must follow Phase 2.
- Phase 4 (api-manager) depends on both Phase 1 (metadata key must exist in `ValidMetadataKeys` or `bin-call-manager` listen handler rejects the request with HTTP 400) and Phase 3 (RPC wrappers must exist).
- Phases 5 and 6 run concurrently after Phase 4.

---

## Decisions Log

| Decision | Choice | Alternatives | Rationale |
|----------|--------|--------------|-----------|
| Call fidelity | Real call | Signaling-only INVITE probe via `kamailio-proxy` | User explicitly chose real-call path |
| Provider override plumbing | `bin-route-manager` synthetic routes via `targetProviderIDs` on `DialrouteList` | Thread `providerID` as a dedicated parameter through call-manager | Implemented in PR #793 |
| api-manager → call-manager signal | Option A: general `metadata` pass-through on `CallV1CallsCreate` with typed `MetadataKey` registry and runtime validation | Option B: dedicated `CallV1CallsCreateForProviderTest` RPC | Implemented in PR #793; general mechanism reusable for future internal features (e.g., `rtp_debug` already uses it) |
| Metadata array shape | Plural `route_provider_ids` array (failover semantics, try in order) | Singular `test_call_provider_id` | Implemented; also future-proofs failover testing |
| Synthetic route identity | Synthetic `Route.ID == ProviderID` | Generate throwaway UUID | Implemented; human-traceable; compatible with call-manager's failover tracker |
| Audit trail + response | **Reinstated:** dedicated `ProviderCall` entity in `bin-route-manager` persists the admin's request info plus the resulting `call_ids` / `groupcall_ids`. Response of `POST /v1/providercalls` returns `ProviderCall.WebhookMessage` | Return raw `{calls, groupcalls}` and rely on metadata-tagged Call records only | User decision: the response should be a first-class ProviderCall with requested info + created IDs |
| Response shape | `ProviderCall.WebhookMessage` (IDs of created Calls/Groupcalls + request summary); admin polls `GET /v1/calls/{id}` per call | Synchronous pass/fail + SIP code | Matches async call lifecycle; ProviderCall.WebhookMessage is the atomic return type |
| Permission level | `PermissionProjectSuperAdmin` only | `PermissionCustomerAdmin` (customer-scoped) | User constraint: platform admin must see/control every provider across customers |
| Provider ownership check | None — admin can target any provider in the system | Restrict to providers owned by the admin's customer | Admin is platform-level by design; no customer boundary applies |
| Access to the endpoint | Admin-only | Customer-tier self-serve | User constraint |
| CDR / billing tagging of provider calls | Deferred; `Metadata` mechanism already supports it | Add a tagging key now | User: "no need now" |
| Endpoint shape | `POST /v1/providercalls` (flat resource) | `POST /v1/providers/{provider_id}/calls` (nested); verb-style path | Matches monorepo convention for compound resources (`/v1/campaigncalls`, `/v1/conferencecalls`); ProviderCall is a first-class top-level resource |
| Customer attribution | Derived from auth context (`a.CustomerID` — project admin's own customer via JWT/accesskey); not in the request body | Required `customer_id` in body so admin can attribute to an arbitrary customer | Auth context already carries this; keeps body minimal; avoids a customer-existence RPC on every request. Revisit if admins need to bill provider calls to other customers |
| Request body shape | Required `provider_id` + customer call API shape (`source`, `destinations`, optional `flow_id` / `actions` / `anonymous`). No `customer_id` (auth-derived) | `{source, destination}` minimal; include `customer_id` explicitly | Consistency with the existing customer call API; minimal body; auth-derived customer |
| Source validation bypass | New metadata key `skip_source_validation` (bool); set server-side by the admin endpoint; `bin-call-manager/getValidatedSourceForOutgoingCall` honors it and preserves the admin-supplied source verbatim | Endpoint-only bypass (would require call-manager to know which endpoint originated the call); always-bypass for every call (unsafe) | Follows the PR #793 metadata pattern; typed `MetadataKey` constant; runtime-registry enforced; reusable by any future internal "trusted source" scenario |
| Kill-switch / runtime toggle | None | Config flag to disable endpoint at runtime | `PermissionProjectSuperAdmin` gate is sufficient; revisit only if abuse observed |

---

## Research Summary

**Market Context**
- Commercial CPaaS platforms (Twilio, Telnyx, Vonage) expose per-provider / per-route call-placement tooling via admin dashboards. Admin-only + real-outbound is the established pattern.
- VoIP admin tooling distinguishes reachability (SIP OPTIONS) from call-through (full INVITE). VoIPbin already has the former from completed health-check work; PR #793 + this PRD deliver the latter.

**Technical Context (current state)**
- Call creation path: `bin-api-manager/pkg/servicehandler/call.go` → `CallV1CallsCreate` (`bin-common-handler/pkg/requesthandler/call_calls.go`) → `bin-call-manager` → `DialrouteList` (`bin-route-manager/pkg/routehandler/dialroute.go`).
- `Call.Metadata` is a JSON column with `WebhookMessage` visibility. Typed keys and registry in `bin-call-manager/models/call/metadata.go`.
- `bin-common-handler` RPC signatures updated (PR #793): `CallV1CallsCreate`, `CallV1CallCreateWithID` carry `metadata`; `RouteV1DialrouteList` carries `targetProviderIDs`.
- Callers passing `nil` metadata today: `bin-flow-manager`, `bin-queue-manager`, `bin-campaign-manager`, and `bin-api-manager`'s customer-facing paths.
- `voip-kamailio-proxy` (OPTIONS-based health checks) is out of scope for this feature.
- Admin console is a separate repo — no UI work in this PRD.

---

*Generated: 2026-04-21 (revised after PR #793)*
*Status: DRAFT — ready for implementation planning on Phase 1*
