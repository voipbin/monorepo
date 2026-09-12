# Phase 4b: PUT /providers/{id} partial-update fix

Status: DRAFT (Round 0)
Author: CPO, per roadmap `docs/plans/2026-09-12-put-partial-update-migration-roadmap.md`
Phase: 4 of 8 (§4, roadmap row #6, rated **MEDIUM** risk, hybrid — `codecs`
already optional) — provider half of the two-service Phase 4 split (queues
+ providers land as two separate PRs per the roadmap's one-PR-per-repo-boundary
rule; this doc covers `bin-route-manager` only, see the sibling doc
`2026-09-12-queue-put-partial-update-phase4a-design.md` for
`bin-queue-manager`'s `queues`).
Precedent: PR #1291 (`mcpservers`), PR #1293 (Phase 1), PR #1295 (Phase 2),
PR #1298 (Phase 3, `routes` — also `bin-route-manager`, closest same-service
precedent) — this doc applies the same established pattern (roadmap §3), it
does not re-derive it.

## 1. Problem statement

`PUT /providers/{id}` requires 6 of 7 body fields (`type`, `hostname`,
`tech_prefix`, `tech_postfix`, `tech_headers`, `name`, `detail`). `codecs`
is optional **at the OpenAPI/generated-Go-layer only**
(`PutProvidersIdJSONBody.Codecs *string`, confirmed at
`bin-api-manager/gens/openapi_server/gen.go:9636`) — but this is a
false signal about the actual runtime contract. **`codecs` IS silently
wiped by omission today**, the exact bug class this phase exists to fix,
confirmed by tracing the full chain:
`bin-api-manager/server/providers.go:289-292` collapses the pointer to a
plain value BEFORE calling the service handler --
`codecs := ""; if req.Codecs != nil { codecs = *req.Codecs }` -- so
`providerHandler.Update` never sees "was `codecs` present," only ever a
`string` that is already `""` for both "omitted" and "explicitly cleared."
`Update` then unconditionally calls `validateCodecs("")` (returns `("",
nil)`, confirmed at `validate.go:8,18-19`) and sets
`fields[provider.FieldCodecs] = ""` unconditionally (`provider.go:197`).
A PUT that omits `codecs` today silently clears any existing codecs value
to server-default negotiation -- **`codecs` is the 7th field this phase
must fix, not a reference pattern the other 6 already follow.** §3 below
treats all 7 fields uniformly; nothing about `codecs` is out of scope or
deferred.

For the other 6 fields, the same bug shape applies via the same
mechanism: every field is a plain (non-pointer) value from the
OpenAPI-generated JSON body down to
`bin-route-manager/pkg/providerhandler/provider.go`'s `Update`, which
**unconditionally** builds the full 8-key SQL field map (`provider.go:189-198`,
8 not 7 because `Update` also conditionally adds `FieldHealthStatus`/
`FieldHealthCheckedAt` — see §1.1) regardless of what the caller sent.

Per roadmap §2 row #6, `hostname`/`tech_prefix`/`tech_postfix` are the
highest-risk fields (routing-determining, comparable to `routes/{id}`'s
`provider_id`/`target`, Phase 3's already-fixed HIGH-risk fields) — an
omitted `hostname` silently rewrites the provider's SIP gateway target to
`""`, breaking every outbound call routed through that provider; omitted
`tech_prefix`/`tech_postfix` silently wipe SIP dial-string prefixes/
postfixes required by some carriers' trunk configs. Rated MEDIUM (not
HIGH like `routes/{id}`) per the roadmap's own reasoning: narrower blast
radius (one provider's traffic, not the whole route table's dispatch
logic) — this phase doesn't re-litigate that risk-tier call, it inherits
it from the roadmap.

### 1.1 `Update`'s existing hostname-change detection interacts with partial update

`providerHandler.Update` (`provider.go:182-204`) already fetches the
current provider row (`current, err := h.Get(ctx, id)`) BEFORE building
the fields map, specifically to compare `current.Hostname != hostname`
and conditionally reset `FieldHealthStatus`/`FieldHealthCheckedAt` to
`Unknown`/`nil` when the hostname is actually changing (health-check
state is hostname-scoped; an unrelated field update — e.g. renaming the
provider — should not trigger a health-status reset).

This existing logic must be adapted, not removed, when `hostname` becomes
a pointer: the comparison becomes `if hostname != nil && current.Hostname
!= *hostname` — i.e., the health-reset trigger fires only when the caller
explicitly supplied a new, different hostname; an omitted `hostname`
(nil) must NOT trigger a health-status reset, since nothing about the
hostname changed. This is the one piece of pre-existing business logic in
this call chain that requires a genuine code change beyond "convert to
pointer, gate on nil" (contrast with Phase 1's finding that `extensionHandler`/
`trunkHandler`'s business logic needed zero changes) — call this out
explicitly to a reviewer so it isn't missed as "just another field."

The `current, err := h.Get(ctx, id)` fetch itself is unconditional today
and remains unconditional after this fix (it's needed regardless of which
fields are present, both for the hostname-comparison above and as the
natural response for an all-omitted no-op PUT — see §3 step 9).

### 1.2 The all-omitted no-op path must NOT publish `provider_updated`

Today, `providerHandler.Update` always writes (even a no-op-in-data-terms
full-body PUT that resends identical values still calls
`h.db.ProviderUpdate` and, on success, unconditionally calls
`h.notifyHandler.PublishEvent(ctx, provider.EventTypeProviderUpdated,
res)` at `provider.go:216`. After this fix, the all-omitted case (§3 step
9's `len(fields) == 0` short-circuit, returning `current` directly) skips
`h.db.ProviderUpdate` entirely and therefore must also skip the
`PublishEvent` call — an empty-body PUT is a client no-op, not a state
change, and should not emit a `provider_updated` webhook event. This
matches Phase 3's `routes/{id}` precedent (`routehandler.Update`'s
`len(fields) == 0 → return h.Get(ctx, id)` branch, which likewise never
reaches its own `PublishEvent` call). **This is a real, if narrow,
behavior change from today's semantics for one specific case (a PUT that
resends the exact-current values in full, or — after this fix — omits
every field) and should be called out explicitly to reviewers, not left
implicit.** No known webhook consumer depends on receiving
`provider_updated` for a true no-op; if one is discovered, that is a
separate question to raise with 대표님, not a reason to block this phase
(mirrors the roadmap §6 non-goal discipline: no retroactive/consumer-side
accommodation without a concrete signal).

## 2. Full call chain

```
PUT /providers/{id}
  bin-api-manager/server/providers.go: PutProvidersId (verify exact
    filename/function name at implementation time)
        |
  bin-api-manager/pkg/servicehandler/provider.go: ProviderUpdate
        |
  bin-common-handler/pkg/requesthandler/route_providers.go: RouteV1ProviderUpdate
    (confirmed via route_providers_test.go's Test_RouteV1ProviderUpdate)
        |
  bin-route-manager/pkg/listenhandler/
    models/request/providers.go: V1DataProvidersIDPut (verify exact name)
    v1_providers.go: v1ProvidersIDPut (confirmed call site at line 169)
        |
  bin-route-manager/pkg/providerhandler/provider.go: Update   <-- ACTUAL
                                                                    FIX SITE
    (builds fields map, currently unconditional, provider.go:141-198;
     ALSO contains the hostname-change health-reset logic, §1.1)
        |
  bin-route-manager/pkg/dbhandler/provider.go: ProviderUpdate   <-- already
                                                                      safe,
                                                                      no change
                                                                      needed
    (len(fields)==0 guard + squirrel SetMap, confirmed at provider.go:254-257)
```

Confirmed via direct code read: `providerHandler.Update`'s signature is
`Update(ctx, id, providerType provider.Type, hostname string, techPrefix
string, techPostfix string, techHeaders map[string]string, name string,
detail string, codecs string)` (`provider.go:142-153`) and unconditionally
builds `fields := map[provider.Field]any{...}` (`provider.go:189-198`)
before calling `h.db.ProviderUpdate`.

## 3. In scope

Per roadmap §3's established pattern, applied to this chain, with the one
genuine business-logic adaptation from §1.1:

1. **OpenAPI schema** — `bin-openapi-manager/openapi/paths/providers/id.yaml`:
   remove the `required:` block (the 6 currently-required fields become
   optional; `codecs`'s existing description, which already correctly
   documents the omitted-vs-empty-string distinction this migration wants
   everywhere else, is preserved as the in-file reference for the other 6
   fields' new descriptions -- note this is a description-only reference;
   §1's finding is that the Go call chain does NOT yet honor that
   description for `codecs`, which this phase fixes). `tech_headers` (a
   `type: object`, i.e. `map[string]string`) needs the same explicit-empty-object-vs-omit
   note as `tag_ids` gets in the sibling queue doc.
2. **`bin-api-manager/server/providers.go`**: stop unconditionally
   dereferencing OpenAPI-generated fields to plain values for ALL 7
   fields, including `codecs` -- remove the
   `codecs := ""; if req.Codecs != nil { codecs = *req.Codecs }`
   collapse-to-value pattern (§1) and pass `req.Codecs` (already `*string`
   at this layer) straight through unchanged; the other 6 fields need the
   same treatment once their OpenAPI types become pointers. `type` (a
   `RouteManagerProviderType` enum) needs the same local-type-to-domain-type
   pointer rebind pattern Phase 1's `AuthTypes` and Phase 3's
   `WebhookMethod`-adjacent conversions established (Go does not allow
   direct `*A`-to-`*B` conversion even for same-underlying-type enums --
   allocate a new pointer, don't try to cast). Confirmed via
   `gen.go:9634-9646`: `Codecs` is already `*string,omitempty` at this
   layer today (the only field oapi-codegen already generated as a
   pointer, since it was already non-required in the schema) -- this is
   live proof the pointer-generation mechanism works as the roadmap's §3
   step 1 claims; the other 6 fields will generate the same way once
   `required:` is removed from them in step 1 above.
3. **`bin-api-manager/pkg/servicehandler/provider.go`**: `ProviderUpdate`
   signature changes to accept pointers for all 7 fields, including
   `codecs` (not optional per §1 -- `codecs` genuinely needs the same
   pointer treatment as the other 6, this is not a judgment call to defer).
4. **`bin-api-manager/pkg/servicehandler/main.go`** interface update, plus
   `RouteManagerProviderType` pointer-generation is independently
   confirmed against generated code, not assumed from the roadmap's
   general claim: `Codecs *string,omitempty` (`gen.go:9636`) is the live
   proof-of-pattern that a non-required object property generates as a
   pointer in this codebase's oapi-codegen setup — the other 6 fields
   (including `Type RouteManagerProviderType`) will generate identically
   once `required:` is removed from them;
   `mock_main.go` regenerated.
5. **`bin-common-handler/pkg/requesthandler/route_providers.go`**:
   `RouteV1ProviderUpdate` signature changes to pointers; marshals into
   the wire DTO as pointers.
6. **`bin-common-handler/pkg/requesthandler/main.go`**: interface
   updated; `mock_main.go` regenerated.
7. **`bin-route-manager/pkg/listenhandler/models/request/providers.go`**:
   the PUT wire DTO's fields become pointers with `omitempty` JSON tags.
8. **`bin-route-manager/pkg/listenhandler/v1_providers.go`**
   (`v1ProvidersIDPut`, confirmed call site at line 169): pass pointers
   through unchanged in shape.
9. **`bin-route-manager/pkg/providerhandler/provider.go`** (`Update`) —
   **THE ACTUAL FIX SITE**: signature changes to pointers for all 7
   fields (including `codecs`, per §1); `validateCodecs(*codecs)` call
   becomes conditional on the `codecs` pointer being non-nil (an omitted
   `codecs` should skip validation entirely, not validate an empty
   string — this is a genuine behavior change from today's
   always-validate-empty-string, and it is correct: today's "always
   validate" is only reachable because `server/providers.go` already
   collapsed nil to `""` before this function ever saw it, so removing
   that collapse and gating on nil here is the fix, not a new risk;
   confirm this doesn't change behavior for callers who currently send
   `codecs: ""` explicitly, which should still validate and still mean
   "clear/default", exactly as it does today — the difference is only
   for the omitted case); the `fields := map[provider.Field]any{...}`
   literal becomes built conditionally per non-nil pointer; the
   `current.Hostname != hostname` comparison (§1.1) becomes `if hostname
   != nil && current.Hostname != *hostname`; `len(fields) == 0`
   short-circuits to returning `current` directly (it's already fetched,
   per §1.1 — no second `Get` call needed, mirroring Phase 1's
   `v1_trunks.go` no-op-avoids-double-fetch pattern rather than Phase 1's
   `v1_extensions.go` always-refetch pattern, since `current` is already
   in hand here). **This no-op path also must NOT call
   `h.notifyHandler.PublishEvent(ctx, provider.EventTypeProviderUpdated,
   res)` — see §1.2, a genuine behavior change from today that must be
   called out to reviewers, not silently introduced.**
10. **`bin-route-manager/pkg/providerhandler/main.go`** (interface file,
    exact name TBD): `ProviderHandler` interface's `Update` declaration
    updated; `mock_main.go` regenerated.
11. **CLI call sites**: confirmed `bin-route-manager/cmd/route-control/main.go`
    has NO `provider`-prefixed subcommand (`initCommand`'s full command
    list is `route`/`dialroute` only; no `provider` subcommand exists to
    call `providerHandler.Update`) — this is a closed, verified
    non-finding, not an open implementation-time grep. No CLI change
    needed for this phase.
12. **RST docs** (`bin-api-manager/docsdev/source/provider_struct_provider.rst`
    or equivalent — verify exact filename): add an Implementation Hint
    note stating the new contract, specifically calling out (a) the
    `hostname`-change health-reset behavior from §1.1 (an omitted
    `hostname` does NOT reset health status; an explicitly-changed one
    does), and (b) the routing-risk callout for `hostname`/`tech_prefix`/
    `tech_postfix` (mirrors roadmap row #6); clean Sphinx rebuild
    committed in the same commit.
13. **Tests** at every layer per roadmap §3 step 11 — see §5 below.

## 4. Out of scope

- **`PUT /queues/{id}`** — separate endpoint, separate service
  (`bin-queue-manager`), lands as its own PR — see sibling design doc
  `2026-09-12-queue-put-partial-update-phase4a-design.md`.
- **`square-admin` frontend** — not verified in this repo, same caveat as
  every prior phase.
- **`POST /providers`** (create) and **`POST /providers/setup`** (the
  "provider setup" legacy/CARRIER_CREDENTIALS_REJECTED path referenced in
  `bin-api-manager/pkg/servicehandler/provider.go:221-236`) — roadmap out
  of scope generally; Create has no prior value to preserve. Verify at
  implementation time that "provider setup" truly is a Create-shaped
  operation and not a hidden second Update path before finalizing this
  exclusion.
- **Provider health-check mechanics beyond the reset-trigger condition
  fixed in §1.1/§3 step 9** — this phase does not change how health
  checks run, only when a hostname-change-triggered reset fires.
- **Provider/route domain validation beyond presence** (e.g. does this
  provider's `hostname` actually resolve, are `tech_prefix`/`tech_postfix`
  validated against a carrier-specific format) — if such validation
  exists today, this phase preserves it unchanged, only gating it on the
  relevant pointer being non-nil.

## 5. Testing strategy

Mirrors Phase 1-3, adapted to this call chain's fix site
(`providerHandler.Update`) and its two service-specific wrinkles
(hostname-change health-reset logic, `map[string]string` field type for
`tech_headers`):

- **`bin-route-manager/pkg/providerhandler/provider_test.go`**:
  table-driven cases for `Update` — one field set / rest nil (assert
  exact `fields` map contents per field); all-omitted no-op case (assert
  `dbhandler.ProviderUpdate` is NOT called, `Times(0)`, and `current` --
  already fetched -- is returned directly per §3 step 9); **the
  health-reset interaction specifically** (§1.1) needs at least 3 cases:
  (a) `hostname: nil`, current hostname unchanged in DB → health fields
  NOT reset, NOT present in the fields map; (b) `hostname:
  &sameHostnameAsCurrent` (explicitly re-sent but unchanged value) →
  verify against actual current behavior whether re-sending the same
  value today triggers a reset (the `!=` comparison suggests no, but
  confirm since this is exactly the kind of subtle behavior a partial-update
  refactor could accidentally flip) and preserve whatever that behavior
  is; (c) `hostname: &differentHostname` → health fields ARE reset,
  present in the fields map as `Unknown`/`nil`. This is the single most
  important test group in this phase — a bug here would either (i) never
  reset health status when it should (stale health data survives a real
  hostname change) or (ii) reset it on every unrelated field update
  (spurious health-check churn), and neither failure mode is caught by
  a generic "does partial update work" test — it requires this specific,
  service-domain-aware test group.
- **`bin-route-manager/pkg/listenhandler/v1_providers_test.go`**: thin
  pass-through test, mechanical.
- **`bin-api-manager/pkg/servicehandler/provider_test.go`**: mechanical
  signature update; one new partial-body case.
- **`bin-common-handler/pkg/requesthandler/route_providers_test.go`**:
  mechanical signature update (existing `Test_RouteV1ProviderUpdate`
  confirmed present, needs pointer-arg update).
- **`bin-api-manager/server/providers_test.go`** (HTTP layer): name-only
  (or `hostname`-only, given this field's centrality) PUT test asserting
  every other pointer stays nil; a `type` enum pointer-rebind test
  (mirrors Phase 1's `AuthTypes` HTTP-layer test) confirming the local
  OpenAPI type correctly converts to the domain `provider.Type` pointer.
- **CLI test file** (if applicable per §3 step 11): mechanical update
  only if a provider CLI call site is found.
- **Revert-and-rerun requirement**: for at least `hostname: nil` (highest
  routing risk per §1) AND the health-reset-not-triggered-on-unrelated-update
  case (§1.1's subtle behavior), confirm the new tests genuinely FAIL
  against pre-fix `provider.go` before considering the phase complete.
- Full verification workflow per phase exit criteria (roadmap §5):
  `go mod tidy && go mod vendor && go generate ./... && go test ./... &&
  golangci-lint run` in every touched service (`bin-route-manager`,
  `bin-common-handler`, `bin-api-manager`).

## 6. Non-goals

- No retroactive remediation for provider rows whose `hostname`/
  `tech_prefix`/`tech_postfix` may have already been silently wiped in
  production by the pre-fix behavior (mirrors PR #1291 §8 and roadmap §6).
- No `square-admin` frontend changes bundled into this PR (roadmap §6).
- No new PATCH method / HTTP verb redesign (roadmap §6).
- No changes to `PUT /queues/{id}` (separate PR, sibling doc).
- No changes to provider health-check mechanics beyond the reset-trigger
  fix in §1.1 (this is a preservation of existing behavior under the new
  pointer semantics, not a health-check feature change).
- No new domain-level validation beyond what already exists in
  `providerHandler.Update`/`Create` (§4 above).

## 7. Round 1 review disposition

Independent review verdict: REQUEST CHANGES. Findings and fixes:

| # | Finding | Severity | Fix location |
|---|---|---|---|
| 1 | §1 claimed `codecs` "is already optional" and its existing behavior was a correct reference pattern; actual trace of `server/providers.go:289-292` (`codecs := ""; if req.Codecs != nil { codecs = *req.Codecs }`) shows `codecs` is silently collapsed to `""` on omission before ever reaching `providerHandler.Update`, exactly the bug class this phase fixes — `codecs` is the 7th field needing the pointer fix, not already correct | CRITICAL | §1 rewritten to state the confirmed bug; §3 steps 1-3 and step 9 updated to treat `codecs` identically to the other 6 fields (pointer, conditional validation, conditional field-map inclusion) |
| 2 | §3 step 11 (CLI call sites) presented as an open implementation-time grep despite being trivially resolvable now | MINOR | §3 step 11 rewritten to state the confirmed non-finding: `route-control` has no `provider` subcommand |
| 3 | The no-op short-circuit (§3 step 9) silently changes whether `provider_updated` is published for a since-nothing-changed PUT, without calling this out anywhere | MAJOR | New §1.2 added, explicitly documenting this behavior change and justifying it against Phase 3 precedent; §3 step 9 cross-references it |
| 4 | §3 step 2's `type` pointer-rebind claim leaned on the roadmap's general oapi-codegen claim without citing live generated code | MINOR | §3 steps 2 and 4 now cite `gen.go:9634-9646`'s `Codecs *string,omitempty` as the live proof-of-pattern |

Not changed (reviewer confirmed correct as originally written): §1.1's
hostname-change/health-reset analysis; §2's full call chain and all cited
line numbers; the established-pattern citations (Phase 1 `AuthTypes`,
Phase 3 no-op-avoids-double-fetch); §4/§6's scope boundaries.
