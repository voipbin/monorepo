# Provider Codec Support — Design

**Date:** 2026-05-09
**Branch:** NOJIRA-Provider-codec-support
**Status:** Revised (v3)

## Problem

Providers (SIP carriers) may only support a restricted set of codecs.
Today, codec preferences can only be configured at the customer (OutboundConfig) level and only apply to SIP-to-SIP calls. PSTN calls — which are routed through a carrier Provider — carry no codec constraint, even when the carrier requires one.

## Goal

Allow operators to specify a preferred codec list on a Provider. When a PSTN call is dialed through that provider, the codec list is injected as a `VBOUT-CODECS` SIP header on the outgoing Asterisk channel, constraining SDP negotiation to the listed codecs **for that specific dial attempt**.

## Non-Goals

- Provider codecs do NOT interact with OutboundConfig codecs. The two apply to disjoint call legs (PSTN via carrier vs. SIP-to-SIP respectively), so there is no precedence conflict and no race condition.
- Codec intersection/negotiation between provider and customer is out of scope.
- Codec names are not validated against a known list — treated as opaque operator-supplied strings, same as `OutboundConfig.Codecs`.

## Key Design Decision: Dial-Time Injection via `dialTarget` Struct

Provider codecs **must not be stored in persisted call metadata**. The reason: `getDialURITel` resolves a provider at dial time, after call creation. Dialroute failover (via `createFailoverChannel` → `updateForRouteFailover`) advances `c.DialrouteID` to a new provider between attempts. If Provider A's codecs were frozen into call metadata and the call failed over to Provider B, Provider A's codec list would silently be applied to Provider B's trunk.

The solution is to propagate the provider's codec list from `getDialURITel` through `getDialURI` to `createChannelOutgoing`, where it is written directly into the per-attempt `channelVariables` map. This scopes codec enforcement to each individual dial attempt's provider.

The mechanism is a `callhandler`-internal `dialTarget` struct that replaces the existing `(string, map[string]string)` pair returned by `getDialURI` and its variants.

## Architecture

### 1. Data Model — `bin-route-manager`

**`models/provider/provider.go`**

Add field:
```go
// Codecs is a comma-separated list of codecs offered to this provider (e.g. "PCMU,PCMA").
// Empty string means no restriction — server default negotiation applies.
Codecs string `json:"codecs" db:"codecs"`
```

**`models/provider/field.go`**

```go
FieldCodecs Field = "codecs"
```

**`models/provider/webhook.go`**

Add `Codecs` to `WebhookMessage` struct AND update `ConvertWebhookMessage()` to copy the field:
```go
res.Codecs = p.Codecs
```

### 2. Database Migration — `bin-dbscheme-manager`

```sql
-- upgrade
ALTER TABLE route_providers
  ADD COLUMN codecs VARCHAR(255) NOT NULL DEFAULT '';
-- Existing rows receive '' (empty = no restriction). No data migration needed.
-- No index needed — codecs is not a query predicate.

-- downgrade
ALTER TABLE route_providers
  DROP COLUMN codecs;
```

Generated via `alembic revision -m "add_codecs_to_route_providers"`. Never hand-crafted.

**Deployment ordering:** The migration MUST be applied before the new code is deployed. The new struct has `db:"codecs"` which causes `SELECT codecs FROM route_providers` — this fails until the column exists.

### 3. DB Handler — `bin-route-manager/pkg/dbhandler`

`ProviderCreate` uses `PrepareFields`/`SetMap` reflecting struct tags — the new `db:"codecs"` tag is automatically included. No change.

`ProviderUpdate` accepts a `fields map[provider.Field]any` — the caller adds `provider.FieldCodecs: codecs` to the map. No change to the DB handler itself.

### 4. Wire DTO — `bin-route-manager/pkg/listenhandler/models/request/providers.go`

Add `Codecs` to both wire DTOs (the actual JSON transported over RabbitMQ RPC):

```go
type V1DataProvidersPost struct {
    // ... existing fields ...
    Codecs string `json:"codecs"`
}

type V1DataProvidersIDPut struct {
    // ... existing fields ...
    Codecs string `json:"codecs"`
}
```

`V1DataProvidersSetupPost` (used by `POST /providers/setup`) is **intentionally NOT updated** — the Setup endpoint does not accept codecs (see section 5).

### 5. Provider Handler — `bin-route-manager/pkg/providerhandler`

**Internal request struct (handler-private, not shared with other packages):**

```go
// createRequest holds all parameters for creating a Provider.
type createRequest struct {
    Type        provider.Type
    Hostname    string
    TechPrefix  string
    TechPostfix string
    TechHeaders map[string]string
    Name        string
    Detail      string
    Codecs      string
}
```

The interface remains positional today — add `codecs string` as the last parameter to `Create` and `Update`. Callers enumerated below. If Provider gains more fields in the future, that is the point to introduce a struct DTO pattern monorepo-wide; now is not the time.

**Callers of `Create` and `Update`:**

| File | Action |
|---|---|
| `bin-route-manager/pkg/providerhandler/provider.go` | Update implementation — set `Codecs` on create, add `FieldCodecs` on update |
| `bin-route-manager/pkg/listenhandler/v1_providers.go` | Parse `Codecs` from wire DTO, pass to handler |
| `bin-common-handler/pkg/requesthandler/route_providers.go` | Add `codecs string` param, pass to wire DTO |
| `bin-api-manager/server/providers.go` | Add `codecs` field from parsed request, pass to `RouteV1ProviderCreate/Update` |

**`Setup()`** does not accept codecs — codec requirements for a newly-onboarded carrier are not known at onboarding time. Operators set codecs afterward via `PUT /providers/{id}`. This is documented in the RST tutorial for Setup.

### 6. Validation — `bin-route-manager/pkg/providerhandler/validate.go`

Codec string validation (inline — not shared with `outboundconfighandler` because the 3+ consumer rule is not yet met; if a third consumer appears, promote to `bin-common-handler`):

- Empty string → valid (no restriction)
- Max length → 255 characters (matches DB column)
- Must not contain `\r` or `\n` (CRLF injection defence; aligns with `setChannelVariableCodecs` guard)
- Allowed characters → alphanumeric, `,`, `-`, `_`, `.` — reject `(` and `)` to future-proof Asterisk function-call assembly
- Whitespace around commas is trimmed before storage
- Double commas (empty list elements) are rejected rather than silently collapsed — operator input should not be silently rewritten
- Duplicate codec names are not rejected — harmless to the SIP header

Validated at the route-manager listenhandler boundary before calling the handler.

### 7. API Layer — `bin-route-manager/pkg/listenhandler`

Parse the optional `codecs` field from `POST /providers` and `PUT /providers/{id}` request bodies via the updated wire DTOs and forward to the handler.

### 8. OpenAPI Spec — `bin-openapi-manager`

Add optional `codecs` string field to:
- `Provider` response schema
- `ProviderCreateRequest` body schema
- `ProviderUpdateRequest` body schema

Description: `"Comma-separated codec list offered to this provider (e.g. \"PCMU,PCMA\"). Empty means server-default negotiation. Applied to outgoing PSTN dial attempts via this provider; has no effect on SIP-to-SIP traffic."`

### 9. Call Flow — `bin-call-manager`

**`dialTarget` struct (internal to `pkg/callhandler`):**

```go
// dialTarget carries per-attempt dial parameters resolved at channel-creation time.
type dialTarget struct {
    URI         string
    TechHeaders map[string]string
    Codecs      string    // provider codec list; empty = no constraint
    ProviderID  uuid.UUID // for logging; uuid.Nil for non-provider paths
}
```

**Signature changes (all within `pkg/callhandler/outgoing_call.go`):**

```go
// Before:
func (h *callHandler) getDialURITel(ctx, c) (string, map[string]string, error)
func (h *callHandler) getDialURISIP(ctx, c) (string, map[string]string, error)
func (h *callHandler) getDialURISIPDirect(ctx, c) (string, map[string]string, error)
func (h *callHandler) getDialURI(ctx, c) (string, map[string]string, error)

// After:
func (h *callHandler) getDialURITel(ctx, c) (*dialTarget, error)
func (h *callHandler) getDialURISIP(ctx, c) (*dialTarget, error)
func (h *callHandler) getDialURISIPDirect(ctx, c) (*dialTarget, error)
func (h *callHandler) getDialURI(ctx, c) (*dialTarget, error)
```

`getDialURITel` populates all fields: `URI`, `TechHeaders: pr.TechHeaders`, `Codecs: pr.Codecs`, `ProviderID: pr.ID`.
`getDialURISIP` and `getDialURISIPDirect` return `URI` only; `TechHeaders`, `Codecs`, `ProviderID` are zero values.

**`createChannelOutgoing` (updated):**

```go
target, err := h.getDialURI(ctx, c)
// ... error handling ...

channelVariables := map[string]string{}
techApplied, techSkipped := mergeTechHeaders(channelVariables, target.TechHeaders, log)

transport := getDestinationTransport(target.URI)
setChannelVariableTransport(channelVariables, transport)
anonymous := c.Data[call.DataTypeAnonymous] == "true"
if err := setChannelVariablesCallerID(channelVariables, c, anonymous); err != nil { ... }
setChannelVariableCodecs(channelVariables, c.Metadata) // customer/SIP codec — unchanged
setProviderCodecs(channelVariables, target.Codecs)     // provider/PSTN codec — NEW

if target.Codecs != "" {
    log.Debugf("Provider codec applied for dial attempt. provider_id: %s, codecs: %s",
        target.ProviderID, target.Codecs)
}
```

**`pkg/callhandler/codec.go`** — Add:

```go
// setProviderCodecs writes the VBOUT-CODECS channel variable for PSTN dial
// attempts. It is separate from setChannelVariableCodecs (which reads from
// call metadata for SIP paths) so that provider codecs stay scoped to a
// single dial attempt and do not enter persisted call metadata.
//
// Precedence: setProviderCodecs runs after setChannelVariableCodecs in
// createChannelOutgoing, so provider codecs win on map-key collision.
// The normal case has no collision: OutboundConfig codecs are only embedded
// into call metadata for TypeSIP destinations (not TypeTel), so
// setChannelVariableCodecs writes nothing for PSTN calls. However, an operator
// could set MetadataKeyCodecs directly in per-call metadata — if they do,
// provider codecs deliberately overwrite that value, because the provider's
// accepted codec list is the authoritative constraint for the carrier trunk.
func setProviderCodecs(variables map[string]string, codecs string) {
    if codecs == "" {
        return
    }
    if strings.ContainsAny(codecs, "\r\n") {
        return
    }
    variables["PJSIP_HEADER(add,"+common.SIPHeaderCodecs+")"] = codecs
}
```

### 10. Request Handler — `bin-common-handler/pkg/requesthandler`

Add `codecs string` parameter to `RouteV1ProviderCreate` and `RouteV1ProviderUpdate` (positional, consistent with today's style). Set `data.Codecs = codecs` on the wire DTOs.

### 11. Cache Rollout Behavior — `bin-route-manager/pkg/cachehandler`

Cache entries are JSON. Go JSON unmarshal tolerates missing fields — existing cached entries (without `codecs`) deserialize with `Codecs = ""` (no restriction). Safe and correct during the rollout window. No cache flush needed. Cache invalidation on `ProviderUpdate` already exists in `dbhandler.providerUpdateToCache` — the updated struct (with `Codecs`) will be cached on the next write. Verify this path covers update operations during implementation.

## Testing

- **Unit:** `setProviderCodecs` — codec set, empty, CRLF injection (same shape as `Test_embedCodecs` in `codec_test.go`)
- **Unit:** `embedCodecs` guard — PSTN (`TypeTel`) continues to be skipped at the OutboundConfig embed site
- **Unit:** provider handler `Create`/`Update` with codecs field populated and empty
- **Unit:** `ValidateCodecString` — valid, empty, too long, CRLF, double comma, `(`, `)`, whitespace trimming
- **Integration:** `getDialURITel` returns `dialTarget.Codecs` → `createChannelOutgoing` writes `PJSIP_HEADER(add,VBOUT-CODECS)` into channel variables
- **Failover:** Provider A (codecs=PCMU) dial fails → `createFailoverChannel` advances to Provider B (codecs=OPUS) → `createChannelOutgoing` called again → assert channel variables for second attempt contain OPUS, NOT PCMU (i.e., no bleed from previous attempt)
- **No-codec smoke:** when `pr.Codecs == ""`, assert `PJSIP_HEADER(add,VBOUT-CODECS)` is absent from channel variables
- **SIP path zero-value:** assert `getDialURISIP` and `getDialURISIPDirect` return `dialTarget` with `Codecs == ""` and `ProviderID == uuid.Nil`
- **Key collision / precedence:** PSTN call with per-call `MetadataKeyCodecs="OPUS"` set in metadata AND provider `Codecs="PCMU"` → assert final channel variable is `PCMU` (provider wins)
- **API (bin-api-manager):** `POST /providers` and `PUT /providers/{id}` with and without `codecs`; `GET /providers/{id}` returns `codecs`
- **api-validator:** Tests in `~/gitvoipbin/monorepo-monitoring/api-validator/` for `POST /providers` (with codecs) and `PUT /providers/{id}` (update codecs); read-only `GET` coverage included

## Documentation

RST files to update in `bin-api-manager/docsdev/source/`:
- `provider_struct_*.rst` — add `codecs` to WebhookMessage field table
- `provider_overview.rst` — describe the new field and its PSTN-only semantics
- `provider_tutorial.rst` — example: create provider, then set codecs via `PUT`; note `POST /providers/setup` does not accept codecs

Mandatory rebuild after RST edits:
```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
git add -f bin-api-manager/docsdev/build/
```

## Affected Services

| Service | Change |
|---|---|
| `bin-route-manager` | Model, field constants, webhook + `ConvertWebhookMessage`, wire DTOs, listen handler, provider handler + validate |
| `bin-call-manager` | `dialTarget` struct, `getDialURI*` signatures, `setProviderCodecs`, `createChannelOutgoing` |
| `bin-openapi-manager` | OpenAPI spec — Provider schemas |
| `bin-api-manager` | Generated types, `PostProviders`/`PutProvidersId` handlers, RST docs + rebuild |
| `bin-common-handler` | `RouteV1ProviderCreate`/`Update` — add `codecs string` param |
| `bin-dbscheme-manager` | Alembic migration |
| `monitoring/api-validator` | New provider tests |
