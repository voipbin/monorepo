# Phase 3: PUT /routes/{id} partial-update fix

Status: DRAFT (Round 0)
Author: CPO, per roadmap `docs/plans/2026-09-12-put-partial-update-migration-roadmap.md`
Phase: 3 of 8 (§4, roadmap row #5, rated **HIGH** risk)
Precedent: PR #1291 (`mcpservers`, merged `81a9a6b98`), PR #1293 (Phase 1,
`extensions`/`trunks`, merged `fcb65062c`), PR #1295 (Phase 2, `customer`/
`customers`, merged `fc263447a`) — this doc applies the same established
pattern (roadmap §3) to `bin-route-manager`, it does not re-derive it.

## 1. Problem statement

`PUT /routes/{id}` requires all 5 body fields (`name`, `detail`,
`provider_id`, `priority`, `target`). A client that omits any field does
not get a validation error — every field is a plain (non-pointer)
`string`/`uuid.UUID`/`int` value from the OpenAPI-generated JSON body all
the way down to `bin-route-manager/pkg/routehandler/route.go`'s `Update`,
which **unconditionally** builds the full 5-key SQL field map
(`route.go:200-206`) regardless of what the caller actually sent.

The highest-risk fields are `provider_id` and `target`: an omitted
`provider_id` silently rewrites the route's carrier binding to
`uuid.Nil` (the Go zero value for `uuid.UUID`), and an omitted `target`
silently wipes the route's destination-country/catch-all match. Both
silently misroute **live outbound call traffic** for every call matching
that route, with `200 OK` returned to whatever client made the otherwise-
unrelated PUT (e.g. a client that only meant to rename the route or bump
its `priority`). This is the platform's routing-table equivalent of the
`extensions.password` / `customer.webhook_uri` silent-wipe bugs already
fixed in Phase 1/2 — same shape (silent, valid-looking zero value),
comparable blast radius (all traffic through one route, versus one
extension or one customer's webhooks).

Unlike Phase 1 (`bin-registrar-manager`) but **like Phase 2**
(`bin-customer-manager`), the dbhandler layer here is already safe —
`RouteUpdate` (`dbhandler/route.go:244-247`) already has the
`len(fields) == 0` short-circuit and `squirrel.SetMap`-based partial-update
mechanism. The caller, `routeHandler.Update`, is the layer that behaves
differently: it never leaves `fields` empty or partial today. This phase's
actual fix site is `routeHandler.Update`, matching Phase 2's shape (fix
site one layer above dbhandler, not at the listenhandler).

## 2. Full call chain

```
PUT /routes/{id}
  bin-api-manager/server/routes.go: PutRoutesId
        |
  bin-api-manager/pkg/servicehandler/route.go: RouteUpdate
    (perm: check TBD -- verify at implementation time, follows the
     same a.IsDirect() + hasPermission() pattern as Phase 1/2)
        |
  bin-common-handler/pkg/requesthandler/route_routes.go: RouteV1RouteUpdate
        |
  bin-route-manager/pkg/listenhandler/
    models/request/routes.go: V1DataRoutesIDPut
    v1_routes.go: v1RoutesIDPut
        |
  bin-route-manager/pkg/routehandler/route.go: Update   <-- ACTUAL FIX SITE
    (builds fields map, currently unconditional)
        |
  bin-route-manager/pkg/dbhandler/route.go: RouteUpdate   <-- already safe,
                                                                no change needed
    (len(fields)==0 guard + squirrel SetMap, confirmed at route.go:244-247)
```

Confirmed via direct code read: `routehandler.Update`'s signature is
`Update(ctx, id, name string, detail string, providerID uuid.UUID, priority
int, target string)` (`route.go:186`) and unconditionally builds
`fields := map[route.Field]any{route.FieldName: name, ...}`
(`route.go:200-206`) before calling `h.db.RouteUpdate`. `dbhandler.RouteUpdate`
already has `if len(fields) == 0 { return nil }` (`dbhandler/route.go:245-247`)
followed by `squirrel.Update(routesTable).SetMap(tmpFields)` — no dbhandler
change needed, exactly like Phase 1 and Phase 2.

## 3. In scope

Per roadmap §3's established pattern, applied to this chain, with the two
non-string field types (`provider_id uuid.UUID`, `priority int`) requiring
the same pointer treatment Phase 1 already proved out for
`trunks.AuthTypes` (`*[]sipauth.AuthType`, a non-string pointer type):

1. **OpenAPI schema** — `bin-openapi-manager/openapi/paths/routes/id.yaml`:
   remove the `required:` block (all 5 fields become optional); update
   each field's `description` to state the omitted-means-unchanged
   contract, plus a top-level schema `description` per the established
   pattern. Example wording, matching Phase 1/2's committed style
   (`extensions/id.yaml`, `customer/main.yaml`):
   ```yaml
   requestBody:
     required: true
     content:
       application/json:
         schema:
           type: object
           description: >
             Every field is optional. Omitting a field leaves its current
             value unchanged; a full resend of every field is NOT required
             to update a single field (e.g. PUT { "priority": 5 } only
             changes the route's priority and leaves name/detail/
             provider_id/target exactly as they were).
           properties:
             name:
               type: string
               description: Omit to leave the current name unchanged.
             detail:
               type: string
               description: Omit to leave the current detail unchanged.
             provider_id:
               type: string
               description: >
                 Omit to leave the current provider_id unchanged. If
                 present, must be a valid UUID referencing an existing
                 provider.
             priority:
               type: integer
               description: >
                 Omit to leave the current priority unchanged. Lower
                 values are higher priority; 0 is a valid, meaningful
                 value distinct from omission.
             target:
               type: string
               description: Omit to leave the current target unchanged.
   ```
2. **`bin-api-manager/server/routes.go`** (`PutRoutesId`): stop
   unconditionally converting `req.ProviderId` (a `*string` per the
   OpenAPI-generated `PutRoutesIdJSONBody`, once schema is optional) via
   `uuid.FromStringOrNil` to a plain `uuid.UUID` when the field was never
   sent; pass a nil-safe `*uuid.UUID` through instead (nil when
   `req.ProviderId == nil`, else the parsed value — an explicitly-sent but
   unparseable ID string should still surface as an error, not silently
   become `uuid.Nil`; see §3.1 below for this edge case). `req.Name`,
   `req.Detail`, `req.Target` become `*string` automatically once the
   schema is optional (oapi-codegen pointer generation, same as Phase
   1/2). `req.Priority` becomes `*int` automatically.
3. **`bin-api-manager/pkg/servicehandler/route.go`**: `RouteUpdate`
   signature changes to accept pointers for all 5 mutable fields (`name
   *string`, `detail *string`, `providerID *uuid.UUID`, `priority *int`,
   `target *string`).
4. **`bin-api-manager/pkg/servicehandler/main.go`**: `ServiceHandler`
   interface declaration updated; `mock_main.go` regenerated.
5. **`bin-common-handler/pkg/requesthandler/route_routes.go`**:
   `RouteV1RouteUpdate` signature changes to pointers; marshals into
   `V1DataRoutesIDPut` as pointers too (mandatory per roadmap §3 step 4).
6. **`bin-common-handler/pkg/requesthandler/main.go`**: `RequestHandler`
   interface declaration updated; `mock_main.go` regenerated.
7. **`bin-route-manager/pkg/listenhandler/models/request/routes.go`**:
   `V1DataRoutesIDPut` fields become pointers with `omitempty` JSON tags.
   `V1DataRoutesPost` (create) is untouched — see §4.
8. **`bin-route-manager/pkg/listenhandler/v1_routes.go`**
   (`v1RoutesIDPut`): pass pointers through to `routeHandler.Update`
   unchanged in shape (thin pass-through, same as Phase 2's
   `processV1CustomersIDPut`).
9. **`bin-route-manager/pkg/routehandler/route.go`** (`Update`) — **THE
   ACTUAL FIX SITE**: signature changes from plain values to pointers
   (`name *string`, `detail *string`, `providerID *uuid.UUID`, `priority
   *int`, `target *string`); the `fields := map[route.Field]any{...}`
   literal (`route.go:200-206`, currently unconditional) becomes built
   conditionally, `if name != nil { fields[route.FieldName] = *name }`
   per field, mirroring PR #1291 §4.1 / Phase 1 / Phase 2 exactly.
   `len(fields) == 0` short-circuits to `h.Get(ctx, id)` (confirmed
   `routeHandler.Get` exists as its own method — verify its `ErrNotFound`
   mapping shape at implementation time and reuse it, do not duplicate,
   same discipline as Phase 1/2).
10. **`bin-route-manager/pkg/routehandler/main.go`** (interface file,
    exact name TBD, verify at implementation time): `RouteHandler`
    interface's `Update` declaration updated; `mock_main.go` regenerated.
11. **`bin-route-manager/cmd/route-control/main.go`** (`runRouteUpdate`,
    the CLI tool): calls `handler.Update` directly with plain
    `viper.GetString`/`viper.GetInt` values today (`main.go:334-342`),
    after its own required-flag checks (`name`/`provider-id`/`target` are
    all validated non-empty before the call, lines 319-332). This call
    site must be updated to pass pointers to keep compiling. Mirrors
    Phase 2's `customer-control` CLI treatment exactly: wrap every value
    in a pointer unconditionally (`&name`, `&detail`, `&providerID`,
    `&priorityVal`, `&target`), preserving the CLI's exact current
    all-fields-required, all-fields-sent behavior. No CLI UX change (no
    flag-changed detection) in this phase — out of scope, same rationale
    as Phase 2 §3 step 11 (오버엔지니어링 지양, no speculative CLI UX work
    without a concrete request). A follow-up ticket can add partial-update
    support to the CLI if an operator ever asks for it.
12. **RST docs** (`bin-api-manager/docsdev/source/route_struct_route.rst`
    or equivalent — verify exact filename at implementation time): add an
    Implementation Hint note stating the new contract and specifically
    calling out that `provider_id`/`target` omission-vs-explicit-set
    matters for live traffic routing (mirrors the risk callout already in
    the roadmap's row #5), mirroring Phase 1/2's RST note pattern; clean
    Sphinx rebuild committed in the same commit.
13. **Tests** at every layer per roadmap §3 step 11 — see §5 below.

### 3.1 `provider_id`'s malformed-but-present string is a validation error, not a silent nil

`req.ProviderId` becomes `*string` once optional. Three states must be
distinguished, not collapsed to two:

- **omitted** (`req.ProviderId == nil`): pass `nil` through as
  `*uuid.UUID`, meaning "leave provider_id unchanged" — the fix this
  phase delivers.
- **present and a valid UUID string**: parse and pass a non-nil
  `*uuid.UUID` pointing at the parsed value — "set provider_id to this".
- **present but an unparseable string** (e.g. `"not-a-uuid"`): this is a
  client error and must be rejected with 400, NOT silently converted to
  `uuid.Nil` and treated as "leave unchanged" or "set to the zero UUID".
  `uuid.FromStringOrNil` returns `uuid.Nil` on parse failure with no error
  signal, which is exactly the ambiguity this fix is designed to
  eliminate elsewhere — implementation must use `uuid.FromString` (which
  returns `(uuid.UUID, error)`) when `req.ProviderId != nil`, and surface
  a `cerrors.InvalidArgument` on parse failure. **Confirmed existing
  precedent for this exact pattern**: `bin-api-manager/server/campaigns.go`'s
  `PutCampaignsIdNextCampaignId` (lines 414-431) already implements
  "empty/omitted is a valid domain value, non-empty-but-unparseable is a
  400" for `next_campaign_id` -- use that function's shape (parse only
  when present, `cerrors.InvalidArgument` with code `INVALID_ARGUMENT` on
  parse failure) as the implementation reference, not `PostRoutes` in the
  same file (`routes.go:90`), which uses unguarded
  `uuid.FromStringOrNil` and is the anti-pattern this section exists to
  avoid repeating.

## 4. Out of scope

- **`POST /routes`** (create) — roadmap out of scope generally, same
  reasoning as every other phase's Create exclusion: a brand-new route has
  no prior value to preserve, every field is caller-supplied or take a
  from-scratch default. `V1DataRoutesPost`'s plain-value DTO and
  `routeHandler.Create` are not touched.
- **`square-admin` frontend** — not verified in this repo (separate repo),
  same caveat as Phase 1/2 (flagged, low-risk residual: a frontend that
  already sends every field unconditionally is unaffected either way).
- **`PUT /providers/{id}`** — separate endpoint (roadmap row #6, MEDIUM
  risk, hybrid: `codecs` already optional), scheduled as its own Phase 4
  work per the roadmap. Not bundled into this phase even though it shares
  `bin-route-manager` with `routes`, per the roadmap's explicit phase
  boundaries (§4) — Phase 3 is `routes` only.
- **Route/provider domain validation beyond presence** (e.g. does
  `provider_id` reference an existing, non-deleted provider row) — if such
  validation exists today in `routeHandler.Update` or `Create`, this phase
  preserves it unchanged, only gating it on the pointer being non-nil
  (mirrors roadmap §3 step 7's "any field-specific validation... becomes
  conditional on non-nil").

## 5. Testing strategy

Mirrors Phase 1/2, adapted to this call chain's fix site
(`routeHandler.Update`, matching Phase 2's shape) and its two non-string
pointer types:

- **`bin-route-manager/pkg/routehandler/route_test.go`**: table-driven
  cases for `Update` — one field set / rest nil (assert exact `fields`
  map contents per field, not just "no error"); all-omitted no-op case
  (assert `dbhandler.RouteUpdate` is NOT called, `Times(0)`, and whatever
  `Get`-equivalent path takes over IS called instead); a
  `priority: intPtr(0)` (explicit pointer to the zero value) case distinct
  from `priority: nil`, confirming the pointer-to-zero-value-is-a-real-value
  distinction holds for an `int` field the same way Phase 1/2 proved it
  for enum/string fields (0 is a legitimate, high-priority route ranking
  per this service's "lower integer = higher priority" convention —
  confirmed in `bin-route-manager/CLAUDE.md`'s Key Facts — so this is not
  a hypothetical edge case, it is the single most routing-consequential
  value this field can take); a `provider_id`-only case confirming
  `target`/`priority`/`name`/`detail` stay unchanged.
- **`bin-route-manager/pkg/listenhandler/v1_routes_test.go`**:
  extend/add a table test confirming `v1RoutesIDPut` passes pointers
  through to `routeHandler.Update` unchanged (thin pass-through,
  mechanical assertion, matching the existing `Test_v1RoutesIDPut`
  structure at `v1_routes_test.go:268`).
- **`bin-api-manager/pkg/servicehandler/route_test.go`**: update existing
  `RouteUpdate` mock call sites for the new pointer signature (mechanical);
  add at minimum one new case with a partial body, asserting the right
  pointers are non-nil vs. nil through the permission-check layer.
- **`bin-common-handler/pkg/requesthandler/route_routes_test.go`**:
  mechanical signature update (pure pass-through layer per roadmap §3 step
  4/11 — no new test cases required beyond confirming the pointer marshal
  round-trips correctly).
- **`bin-api-manager/server/routes_test.go`** (HTTP layer — the layer
  Phase 1's Round 1 review flagged as where the dereference-to-zero-value
  bug actually lives and is invisible to every lower-layer test): add a
  name-only PUT test asserting the service-handler mock is called with
  `name` non-nil and `detail`/`providerID`/`priority`/`target` all nil;
  add a SEPARATE test for §3.1's malformed-`provider_id` case, asserting
  a 400 is returned and `RouteUpdate` is never called (`Times(0)`) — this
  is the one field-specific edge case unique to this phase (no equivalent
  existed in Phase 1's string/enum fields or Phase 2's all-string-or-enum
  fields), so it needs its own dedicated regression test, not just
  coverage-by-implication from the name-only case.
- **`bin-route-manager/cmd/route-control/main.go`** (if a test file
  exists — verify during implementation, no `main_test.go` was confirmed
  present at design time): update `runRouteUpdate`'s `handler.Update`
  mock call site for the pointer signature; no new test cases needed
  since §3 step 11 keeps its current all-fields-sent behavior unchanged
  (mirrors Phase 2's identical CLI treatment).
- **Revert-and-rerun requirement** (standing practice): for at least the
  `provider_id: nil` and `target: nil` cases (the two highest-risk fields
  per §1), confirm the new tests genuinely FAIL against pre-fix
  `route.go` (git stash the fix, rerun, confirm FAIL, restore) before
  considering the phase complete.
- Full verification workflow per phase exit criteria (roadmap §5):
  `go mod tidy && go mod vendor && go generate ./... && go test ./... &&
  golangci-lint run` in every touched service (`bin-route-manager`,
  `bin-common-handler`, `bin-api-manager`).

## 6. Non-goals

- No retroactive remediation for route rows whose `provider_id`/`target`
  may have already been silently wiped in production by the pre-fix
  behavior (mirrors PR #1291 §8 and the roadmap §6 — raise separately to
  대표님 if a concrete customer complaint or traffic-misroute incident
  surfaces).
- No `square-admin` frontend changes bundled into this PR (roadmap §6).
- No new PATCH method / HTTP verb redesign (roadmap §6).
- No changes to `PUT /providers/{id}` (separate Phase 4 per roadmap §4).
- No new domain-level validation beyond what already exists in
  `routeHandler.Update`/`Create` (§4 above).

## 7. Round 1 review disposition

Independent review verdict: REQUEST CHANGES. Findings and fixes:

| # | Finding | Severity | Fix location |
|---|---|---|---|
| 1 | §2/§3/§5 omitted `bin-route-manager/cmd/route-control/main.go`'s `runRouteUpdate` -- a real call site of `routeHandler.Update` (`main.go:334-342`) that would fail to compile once the signature changes to pointers, but was never mentioned as in-scope | IMPORTANT | New §3 step 11 added, documenting the CLI call site and the "wrap unconditionally in pointers, preserve current behavior" scoping decision (mirrors Phase 2's `customer-control` treatment exactly); §5 also updated with a corresponding test-verification note |
| 2 | §3.1 cited `server/routes.go`'s `PostRoutes` as a same-file precedent for the "empty is valid, malformed is a 400" pattern, but `PostRoutes` (`routes.go:90`) actually uses unguarded `uuid.FromStringOrNil` -- the exact anti-pattern this section exists to avoid, not a correct precedent | MINOR (citation accuracy, not a logic defect) | §3.1 corrected to cite the actual verified precedent, `bin-api-manager/server/campaigns.go`'s `PutCampaignsIdNextCampaignId` (lines 414-431), which correctly implements "empty/omitted valid, non-empty-unparseable is 400" |
| 3 | §3 step 1's OpenAPI description guidance had no concrete example text, unlike Phase 1/2's own design docs which showed the actual YAML | NICE-TO-HAVE | §3 step 1 now includes a full example YAML block matching Phase 1/2's committed style |

Not changed (reviewer confirmed correct as originally written): the full
6-layer call chain in §2 (routehandler/dbhandler/listenhandler/
requesthandler/servicehandler/server, all line numbers verified against
current code); the `routeHandler.Get`/`ErrNotFound` reuse claim; the
Phase 2 precedent parallel (fix site one layer above dbhandler); §3.1's
core three-state requirement (omitted/valid/malformed) and its
implementation guidance shape; §4's out-of-scope boundaries; §5's test
strategy for the `priority: intPtr(0)` and `provider_id`/`target`
revert-and-rerun cases; §6's non-goals.
