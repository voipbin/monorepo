# OutboundConfig Design

> **Note:** This document was written before the table naming convention (§7.0 in [docs/conventions/database.md](../conventions/database.md)) was enforced. All references to `outbound_configs` in this document should be read as `call_outbound_configs` — that is the actual table name in production.

**Date:** 2026-05-07
**Branch:** NOJIRA-outbound-config

## Overview

Introduce a new `OutboundConfig` resource that consolidates per-customer outbound call configuration. The first two fields are:

1. `destination_whitelist` — ISO 3166 alpha-2 country codes allowed for outbound PSTN calls.
2. `codecs` — comma-separated preferred codec list for outbound PSTN calls (replaces `Customer.Metadata.OutboundCodecs`).

This also removes `Customer.Metadata.OutboundCodecs` entirely — no migration, no fallback. Nobody uses it yet.

---

## Requirements

| # | Requirement |
|---|---|
| R1 | Whitelist-only mode: outbound PSTN calls are allowed only to countries in the whitelist. |
| R2 | Always-on globally: enforcement requires no per-customer enable flag. Empty whitelist = deny all PSTN calls. **This is an intentional design choice — see Deploy-day Warning below.** |
| R3 | Country-level granularity: entries are ISO 3166 alpha-2 codes (e.g. `us`, `gb`, `kr`), normalised to lowercase on write. |
| R4 | PSTN (`tel`) only: SIP/agent/other address types bypass the whitelist entirely. |
| R5 | Customer self-serve: full REST API; no bootstrap migration at deploy. |
| R6 | `codecs` replaces `Customer.Metadata.OutboundCodecs` with no fallback or migration. Confirmed via codebase grep: `Customer.Metadata.OutboundCodecs` is set nowhere in production flows — the field was added in PR #879 but never populated by any customer or internal process. |
| R7 | Partial update: `PUT /v1/outbound_configs/<id>` applies JSON merge — fields absent from the body are unchanged. Sending `[]` for `destination_whitelist` explicitly sets it to empty (deny all). |

### Deploy-day Warning

At deploy, every customer without an `OutboundConfig` row is blocked from making outbound PSTN calls (zero-value config = empty whitelist = deny all). This is **intentional** — the loud failure mode forces customers to explicitly configure their allowed countries before calling. Ops must communicate this to customers before rolling out. The error message returned is explicit: `"destination country not in outbound whitelist; configure via GET /v1/outbound_configs"`.

---

## Data Model

### Database table: `outbound_configs`

```sql
CREATE TABLE outbound_configs (
    id                    VARCHAR(36)  NOT NULL,
    customer_id           VARCHAR(36)  NOT NULL,
    name                  VARCHAR(255) NOT NULL DEFAULT '',
    detail                TEXT         NOT NULL DEFAULT '',
    destination_whitelist JSON         NOT NULL DEFAULT '[]',   -- ["us","gb","kr"]
    codecs                VARCHAR(255) NOT NULL DEFAULT '',      -- "PCMU,PCMA,G729"; empty = server default
    tm_create             DATETIME(6)  DEFAULT NULL,
    tm_update             DATETIME(6)  DEFAULT NULL,
    tm_delete             DATETIME(6)  DEFAULT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_customer_id (customer_id)
);
```

- `id` is server-generated UUID.
- `customer_id` unique constraint enforces one config per customer.
- `name` / `detail`: user-supplied labels for display and documentation purposes (consistent with all other resources in this monorepo).
- `codecs` max length 255 chars — validated at the application write boundary before DB insert.

### Go internal struct

```go
// models/outboundconfig/outboundconfig.go
type OutboundConfig struct {
    ID                   uuid.UUID  `json:"id"`
    CustomerID           uuid.UUID  `json:"customer_id"`
    Name                 string     `json:"name"`
    Detail               string     `json:"detail"`
    DestinationWhitelist []string   `json:"destination_whitelist"` // ISO 3166 alpha-2
    Codecs               string     `json:"codecs"`
    TMCreate             *time.Time `json:"tm_create"`
    TMUpdate             *time.Time `json:"tm_update"`
    TMDelete             *time.Time `json:"tm_delete"`
}
```

### WebhookMessage (external-facing)

Per monorepo convention, a `WebhookMessage` struct and `ConvertWebhookMessage()` function are created even though all fields are currently safe to expose. This prevents future internal fields from accidentally leaking.

```go
// models/outboundconfig/webhook.go
type WebhookMessage struct {
    ID                   uuid.UUID  `json:"id"`
    CustomerID           uuid.UUID  `json:"customer_id"`
    Name                 string     `json:"name"`
    Detail               string     `json:"detail"`
    DestinationWhitelist []string   `json:"destination_whitelist"`
    Codecs               string     `json:"codecs"`
    TMCreate             *time.Time `json:"tm_create"`
    TMUpdate             *time.Time `json:"tm_update"`
    TMDelete             *time.Time `json:"tm_delete"`
}

func ConvertWebhookMessage(c *OutboundConfig) *WebhookMessage { ... }
```

### Update request struct

To distinguish "field absent (no change)" from "field explicitly set to empty", the update request uses pointer fields:

```go
type UpdateRequest struct {
    Name                 *string   `json:"name,omitempty"`
    Detail               *string   `json:"detail,omitempty"`
    DestinationWhitelist *[]string `json:"destination_whitelist,omitempty"`
    Codecs               *string   `json:"codecs,omitempty"`
}
```

- `nil` pointer → field not present in body → leave unchanged.
- `*[]string` pointing to `[]` → explicitly set to empty (deny all).
- `*string` pointing to `""` → explicitly clear codecs (use server default).

---

## Match Algorithm

`ValidateDestination` receives an already-fetched `*OutboundConfig` (the caller in `CreateCallOutgoing` fetches it once and passes it in — see Enforcement Flow). The function does not fetch the config itself.

1. If `destination.Type != commonaddress.TypeTel` → return `true` (bypass).
2. If customer is an internal system ID (using shared helper `cucustomer.IsInternalSystemID(customerID)`) → return `true` (bypass).
   - Note: `IsInternalSystemID` will be extracted as a shared helper to avoid duplicating the bypass list across `validateOutgoingCallPermission`, `ValidateCustomerIdentityVerified`, and this new check.
3. Parse `destination.Target` via `getCountry()` (private helper in `pkg/callhandler/validate.go`, same package as `ValidateDestination` — no export required) → ISO 3166 alpha-2 (lowercase).
4. If country is empty/unknown → return `false` (fail-closed). A destination whose country cannot be determined cannot be verified against the whitelist.
5. If `config == nil` or `len(config.DestinationWhitelist) == 0` → return `false` (deny all). **`config == nil` and an empty whitelist are semantically equivalent — both deny.** This is the always-on invariant: absence of configuration is not permissive.
6. Check membership: `destinationCountry ∈ config.DestinationWhitelist` → return result.

**Short-circuit in `CreateCallOutgoing`:** The OutboundConfig fetch and `ValidateDestination` call are both skipped when `destination.Type != TypeTel` or `IsInternalSystemID(customerID)`. This prevents unnecessary cache hits and negative-cache pollution on every internal-system outbound call.

### Write-side validation

- Each `destination_whitelist` entry must be a valid ISO 3166 alpha-2 code, validated against a hardcoded constant set in `models/outboundconfig/iso.go` (~250 codes). The `phonenumber` library does not expose a standalone ISO list, so the set is maintained locally as a `map[string]struct{}`.
- Entries normalised to lowercase on write. Duplicate detection runs **after** normalisation (so `["us","US"]` is rejected as a duplicate of `["us","us"]`).
- Duplicate entries within the same request are rejected.
- `codecs` must match the format `^[A-Za-z0-9]+(,[A-Za-z0-9]+)*$` or be empty — rejects free text, semicolons, SQL fragments, and anything that could corrupt SIP headers downstream.
- `codecs` must not contain `\r` or `\n` (header-injection defence — belt-and-suspenders with the regex).
- `codecs` must not exceed 255 characters.

### ISO map drift test

A unit test in `models/outboundconfig/iso_test.go` calls `phonenumber.GetISO3166ByNumber` directly (not `getCountry`, which is package-private to `pkg/callhandler`) against a representative set of E.164 numbers (one per ITU region), and asserts the returned alpha-2 code exists in the exported `outboundconfig.ISOCountryCodes` map. The local ISO set is exported as `var ISOCountryCodes = map[string]struct{}{...}` in `models/outboundconfig/iso.go`. This catches drift between the `phonenumber` library's recognition set and the local validator's set.

**Known edge case:** `getCountry` returns `""` for numbers the `phonenumber` library cannot parse (malformed E.164, national format without country code). The whitelist check treats these as fail-closed (return `false`). This is consistent with the billing path (`ValidateCustomerBalance` already calls `getCountry` and tolerates empty — it passes empty to billing which is a separate concern). Ops must be aware that destinations with unparseable numbers will be blocked even if the customer has a populated whitelist.

### Shared bypass helper

```go
// bin-customer-manager/models/customer/ids.go
func IsInternalSystemID(id uuid.UUID) bool {
    return id == IDCallManager || id == IDAIManager || id == IDSystem || id == IDBasicRoute
}
```

This helper lives in `bin-customer-manager/models/customer/` — **NOT in `bin-common-handler`** (single-service consumer rule; call-manager already imports customer models). This replaces the three duplicated bypass-ID lists in call-manager.

---

## Enforcement Flow

`CreateCallOutgoing` fetches `OutboundConfig` **once** before both codec embedding and destination validation:

```
CreateCallOutgoing(ctx, customerID, destination, ...)
    │
    ├─ [existing] fetch customer info (cu)
    ├─ [existing] embedRTPDebug(metadata, cu.Metadata.RTPDebug)
    ├─ [existing] validateOutgoingCallPermission(ctx, cu, destination)
    ├─ [existing] ValidateCustomerBalance(ctx, ...)
    │
    ├─ [NEW] config = outboundConfigHandler.GetByCustomerID(ctx, customerID)
    │          (Redis cache → DB → nil if not found)
    ├─ [NEW] metadata = embedCodecs(metadata, config)           ← config.Codecs replaces cu.Metadata.OutboundCodecs
    ├─ [NEW] ValidateDestination(ctx, customerID, config, destination)
    │          └─ applies algorithm above; returns bool
    │
    └─ getDialroutes → create call
```

`ValidateDestination` function signature change:

```go
// Before:
func (h *callHandler) ValidateDestination(ctx context.Context, customerID uuid.UUID, destination commonaddress.Address) bool

// After:
func (h *callHandler) ValidateDestination(ctx context.Context, customerID uuid.UUID, config *outboundconfig.OutboundConfig, destination commonaddress.Address) bool
```

`outboundconfighandler` is injected into `callHandler` at construction time (same pattern as all other handler dependencies in `pkg/callhandler/main.go`).

### Cache

`outboundconfighandler` adds its cache operations as methods on the existing `pkg/cachehandler` (same pattern used by all other handlers in call-manager — no separate Redis client).

- Key: `outbound_config:{customerID}`
- TTL: 30 minutes (consistent with route-manager's cache TTL convention).
- **Write-through on POST (create)**: after DB insert, set cache entry — overwrites any existing negative-cache sentinel.
- **Invalidate on PUT (update)**: delete cache key after DB update.
- **Negative caching**: if DB returns not-found, cache a sentinel value (a dedicated struct field `{Exists: false}` or a reserved JSON `{"_not_found":true}`) with a 1-minute TTL. The `Get` method distinguishes: sentinel → return `nil, nil` (no row, no error); real row → return `*OutboundConfig, nil`. POST overwrites the sentinel on first create. This avoids ambiguity between "cache miss" and "cached not-found."
- Customer deletion (soft-delete via `tm_delete`): no special cache invalidation — the 30-minute TTL naturally expires the entry; deleted configs are filtered at the handler level.

### Codec embed precedence

`embedCodecs` preserves the existing per-call override priority:

```
per-call metadata[MetadataKeyCodecs] (set by caller) > OutboundConfig.Codecs > server default (empty)
```

Concretely: if `metadata[call.MetadataKeyCodecs]` is already set, `embedCodecs` does not overwrite it. This matches the existing `embedCustomerCodecs` behaviour and must not regress. The per-call override test is included in the codec test suite (see §Testing).

### Error Contract (RPC)

`ValidateDestination` returning `false` causes `CreateCallOutgoing` to return a sentinel error constant:

```go
// models/outboundconfig/errors.go
var ErrDestinationNotWhitelisted = errors.New("outbound destination country not whitelisted")
```

`bin-call-manager` returns this sentinel; it does not embed any public API URL (call-manager must not know the api-manager URL surface).

`bin-api-manager` inspects the error string on its side, maps `ErrDestinationNotWhitelisted` to **400 Bad Request**, and renders the user-facing message there:

> *"Destination country is not in your outbound whitelist. Add it via PUT /v1/outbound_configs/{id}."*

This keeps API-URL references out of the RPC layer and lets api-manager control user-facing copy independently.

---

## API Surface

### `bin-call-manager` (internal RabbitMQ RPC)

```
POST /v1/outbound_configs                      create (409 if already exists for customer)
GET  /v1/outbound_configs?customer_id=<uuid>   list (paginated envelope; returns 0 or 1 item)
GET  /v1/outbound_configs/<uuid>               get one
PUT  /v1/outbound_configs/<uuid>               partial update (JSON merge via UpdateRequest pointer fields)
```

- **POST** returns `409 Conflict` if a config already exists for the customer (unique key enforced). Server generates the `id` UUID (same convention as all other resources).
- **GET list** follows the standard paginated envelope: `{"result": [...], "next_page_token": ""}` — consistent with `/v1/calls`, `/v1/agents`, etc. In practice it returns at most 1 item.
- No DELETE endpoint — the record is permanent; customers zero out fields via PUT.

### `bin-api-manager` (public REST, JWT-scoped to caller's customer)

Same four endpoints, proxied through api-manager.

**IDOR prevention:** For non-admin callers, `bin-api-manager` **MUST ignore any caller-supplied `customer_id` query parameter and substitute the customer ID extracted from the caller's JWT**. This prevents a customer from querying or modifying another customer's `OutboundConfig` by supplying a different UUID. Follows the same pattern already enforced for `/v1/calls`, `/v1/numbers`, etc.

---

## Error Semantics & Observability

- **Blocked by whitelist:** `400 Bad Request`, message: `"destination country not in outbound whitelist; configure via GET /v1/outbound_configs"`.
- **Prometheus counter:** `call_outbound_whitelist_rejected_total{destination_country="us"}` — labeled by destination country (low cardinality) rather than customer ID (high cardinality).
- **Timeline event:** `call.outbound_whitelist_rejected` published to `bin-timeline-manager` on block (carries `customer_id`, `destination_country`, `call_id`). This is a fire-and-forget publish on the existing timeline RabbitMQ topic — no schema change required in `bin-timeline-manager`.
- **No webhook event** for blocked attempts — no call resource exists yet to attach it to. Known UX gap: customers relying only on webhooks will receive a 400 API error from their originating request and nothing further. RST tutorial must call this out explicitly with guidance to use timeline events for audit.

---

## Deployment Runbook

The `outbound_configs` table must exist before any call-manager pod attempts to query it. Deploy in this order:

1. **Merge the migration commit** (`bin-dbscheme-manager` change).
2. **Human runs `alembic upgrade head`** (with VPN connected, inside `bin-dbscheme-manager/bin-manager/`). AI must not run this step — see root CLAUDE.md.
3. **Deploy call-manager** (and optionally customer-manager — order between the two does not matter; see below).
4. **Communicate to customers** — all existing customers with no `OutboundConfig` row will have outbound PSTN calls rejected from this point. This is the intentional loud-failure behaviour from R5.

If step 2 is skipped and call-manager pods start first, they will crash-loop on the missing table. Ops must ensure the migration ran before scaling up call-manager.

## Deployment Ordering (between call-manager and customer-manager)

Both services can be deployed independently in any order because:

- If **call-manager deploys first**: it switches to reading `OutboundConfig.Codecs`; customer-manager still serves `OutboundCodecs` (which call-manager no longer reads). Codecs are blank for all customers during the window → server default. Acceptable since nobody uses codecs yet.
- If **customer-manager deploys first**: it stops returning `OutboundCodecs`; call-manager still references `cu.Metadata.OutboundCodecs` which now returns `""`. `embedCustomerCodecs` already handles empty string gracefully (no-op). Safe.

Single coordination requirement: the Alembic migration creating `outbound_configs` must run before any call-manager pod starts trying to query it.

---

## Services Touched

| Service | Change |
|---|---|
| `bin-dbscheme-manager` | New Alembic migration creating `outbound_configs` table (must deploy first) |
| `bin-call-manager` | New `models/outboundconfig/` (struct + webhook + ISO constants); new `pkg/outboundconfighandler/` (CRUD + cache); new listenhandler routes; fill `ValidateDestination` (new signature); update `CreateCallOutgoing` to fetch config once; inject handler in constructor |
| `bin-customer-manager` | Remove `OutboundCodecs` from `models/customer/metadata.go` + tests; extract `IsInternalSystemID` helper |
| `bin-openapi-manager` | Remove `outbound_codecs` from Customer schema; add `OutboundConfig` + `OutboundConfigCreateRequest` + `OutboundConfigUpdateRequest` schemas and endpoint specs |
| `bin-api-manager` | Regenerate `gens/openapi_server/gen.go` via `go generate`; update Customer fixtures to drop `outbound_codecs`; add OutboundConfig proxy routes + handler tests |
| `bin-api-manager/docsdev` | Remove `outbound_codecs` from `customer_overview.rst` + `customer_struct_customer.rst`; add `outbound_config_overview.rst`, `outbound_config_tutorial.rst`, `outbound_config_struct.rst`; clean rebuild + `git add -f build/` |
| `monitoring/api-validator` | New OutboundConfig CRUD E2E tests |

**Not touched:** `bin-route-manager`, `bin-message-manager`, `bin-timeline-manager` (fire-and-forget topic publish, no schema change), `bin-common-handler` (single consumer rule applies).

Each touched service must run the full verification workflow independently: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`.

---

## Testing Strategy

### `bin-call-manager` unit tests

- `pkg/outboundconfighandler/`:
  - `GetByCustomerID`: cache hit, cache miss → DB found (write-through cache), cache miss → DB not found (negative cache entry), DB unavailable (fail-closed).
  - `Create`: success, conflict on duplicate customer_id.
  - `UpdateFields`: pointer-based partial merge (nil field = unchanged, empty slice = set to []).
  - Write-side validation: invalid ISO code rejected, unknown ISO code rejected, duplicate entries rejected, codecs with `\r\n` rejected, codecs >255 chars rejected.
- `pkg/callhandler/validate_test.go`:
  - Allowed country → `true`.
  - Blocked country → `false`.
  - Empty whitelist → `false` (deny all).
  - `config == nil` (no row) → `false`.
  - Non-tel destination → `true` (bypass).
  - Internal customer ID → `true` (bypass via `IsInternalSystemID`).
  - `getCountry` returns empty → `false` (fail-closed).
- `pkg/callhandler/codec_test.go`:
  - Codec embed uses `config.Codecs`; empty config → no embed (server default).
  - Per-call metadata already set → `embedCodecs` does not overwrite (precedence test — must not regress).
- `models/outboundconfig/iso_test.go`:
  - ISO drift test: for a representative E.164 set (one per ITU region), assert `getCountry()` result is present in the local ISO map.

### `bin-customer-manager` unit tests

- Remove all `outbound_codecs` field assertions from `models/customer/metadata_test.go` and listenhandler JSON fixtures.
- Add `IsInternalSystemID` unit tests.

### `bin-api-manager` unit tests

- Drop `"outbound_codecs":""` from all Customer handler expected-JSON fixtures (~14 files).
- New OutboundConfig handler tests: create (201), create-conflict (409), get (200), list (200), partial update whitelist only, partial update codecs only, partial update both.

### `monitoring/api-validator` E2E

```
POST /v1/outbound_configs                              → 201; assert id + fields
GET  /v1/outbound_configs/<id>                         → 200; read back matches create
GET  /v1/outbound_configs?customer_id=<id>             → 200; singleton result
PUT  /v1/outbound_configs/<id>  {destination_whitelist:["us"]}  → 200; only whitelist changed
PUT  /v1/outbound_configs/<id>  {codecs:"PCMU"}         → 200; only codecs changed
POST /v1/outbound_configs       (duplicate customer)    → 409
```

No live-call enforcement test (real outbound PSTN call excluded per cost rules). Virtual-number enforcement path test deferred to implementation phase.
