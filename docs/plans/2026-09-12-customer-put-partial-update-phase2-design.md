# Phase 2: PUT /customer and PUT /customers/{id} partial-update fix

Status: DRAFT (Round 0)
Author: CPO, per roadmap `docs/plans/2026-09-12-put-partial-update-migration-roadmap.md`
Phase: 2 of 8 (§4, roadmap row #1/#2, both rated **HIGH** risk)
Precedent: PR #1291 (`mcpservers`, merged `81a9a6b98`), PR #1293 (Phase 1,
`extensions`/`trunks`, merged `fcb65062c`) — this doc applies the same
established pattern (roadmap §3) to `bin-customer-manager`, it does not
re-derive it.

## 1. Problem statement

Both `PUT /customer` (self-service, JWT-scoped to the caller's own
customer) and `PUT /customers/{id}` (admin surface, `ProjectSuperAdmin`
only) require all 7 body fields (`name`, `detail`, `email`, `phone_number`,
`address`, `webhook_method`, `webhook_uri`). A client that omits any field
does not get a validation error — every field is a plain (non-pointer)
`string`/`WebhookMethod` value from the OpenAPI-generated JSON body all the
way down to `bin-customer-manager/pkg/customerhandler/db.go`'s
`UpdateBasicInfo`, which **unconditionally** builds the full 7-key SQL field
map (`db.go:180-188`) regardless of what the caller actually sent — the gin
`BindJSON` step silently defaults an omitted JSON field to the Go zero
value (`""`), and that zero value then overwrites the existing row.

The highest-risk field is `webhook_uri`: an omitted `webhook_uri` silently
wipes the customer's outbound webhook target. Every backend event
(call/message/chat/etc.) for that customer then silently stops delivering
with `200 OK` returned to whatever client made the otherwise-unrelated PUT
(e.g. a client that only meant to update `name`). This is the platform's
single-tenant equivalent of the `mcpservers.auth_type` silent-deauth bug
that motivated PR #1291 — same shape (silent, valid-looking zero value),
higher blast radius (customer-wide, not one MCP connector).

Unlike Phase 1 (`bin-registrar-manager`), **the dbhandler layer here is not
already safe** — `CustomerUpdate` (`dbhandler/customer.go:236-240`) does
have the `len(fields) == 0` short-circuit and `squirrel.SetMap`-based
partial-update mechanism, but the **caller** (`customerHandler.UpdateBasicInfo`)
never leaves `fields` empty or partial: it is the one layer in this call
chain that behaves differently from Phase 1's `extensionHandler.Update`/
`trunkHandler.Update`, which already built `fields` conditionally before
Phase 1 touched them. This phase's actual fix site is one layer higher
than Phase 1's was.

## 2. Full call chain (both endpoints)

Two HTTP endpoints converge on the same `bin-customer-manager` business
logic through two different `servicehandler` entry points (different
permission models), then share everything below that:

```
PUT /customer (self)              PUT /customers/{id} (admin)
  bin-api-manager/server/           bin-api-manager/server/
    customer.go: PutCustomer          customers.go: PutCustomersId
        |                                  |
  servicehandler/customer.go:       servicehandler/customer.go:
    CustomerSelfUpdate                CustomerUpdate
    (perm: CustomerAdmin,             (perm: ProjectSuperAdmin,
     scoped to a.CustomerID)           explicit id param)
        \_______________________________/
                      |
     bin-common-handler/pkg/requesthandler/
       customer_customer.go: CustomerV1CustomerUpdate
       (single shared function — both callers converge here)
                      |
     bin-customer-manager/pkg/listenhandler/
       models/request/customers.go: V1DataCustomersIDPut
       v1_customers.go: processV1CustomersIDPut
                      |
     bin-customer-manager/pkg/customerhandler/
       db.go: UpdateBasicInfo   <-- ACTUAL FIX SITE (builds fields map)
                      |
     bin-customer-manager/pkg/dbhandler/
       customer.go: CustomerUpdate   <-- already safe, no change needed
```

Confirmed via direct code read (not inferred): `CustomerV1CustomerUpdate`
(`bin-common-handler/pkg/requesthandler/customer_customer.go:124-140`) is a
single shared function called by both `CustomerUpdate` and
`CustomerSelfUpdate` in `bin-api-manager/pkg/servicehandler/customer.go`
(lines 307 and 343 respectively) — there is exactly one wire path from
`bin-api-manager` down to `bin-customer-manager`'s listenhandler regardless
of which HTTP route was hit; only the permission check and the `id` source
(explicit param vs. `a.CustomerID`) differ above that point.

## 3. In scope

Per roadmap §3's established pattern, applied to this chain:

1. **OpenAPI schema** — `bin-openapi-manager/openapi/paths/customer/main.yaml`
   (self, PUT) and `bin-openapi-manager/openapi/paths/customers/id.yaml`
   (admin, PUT): remove `required:` block (all 7 fields become optional);
   update each field's `description` to state the omitted-means-unchanged
   contract, plus a top-level schema `description` per PR #1291's §5
   pattern. `webhook_method`'s `description` gets the extra callout that an
   explicit empty string / `"GET"` etc. is a real value distinct from
   omission (mirrors the `mcpservers.auth_type` precedent — `WebhookMethod`
   has a valid zero-value-shaped member, `WebhookMethodNone = ""`, confirmed
   at `models/customer/customer.go:67`).
2. **`bin-api-manager/server/customer.go`** (`PutCustomer`) and
   **`customers.go`** (`PutCustomersId`): stop dereferencing
   `openapi_server.PutCustomerJSONBody`/`PutCustomersIdJSONBody` fields to
   plain values; pass the generated pointers straight through. Confirmed via
   `gen.go:9067-9137`: **the fields are currently generated as plain
   `string`/`CustomerManagerCustomerWebhookMethod` (NOT pointers)**, because
   `required:` is set on all of them in the schema today — step 1 above is a
   precondition for this step, not independent (oapi-codegen only emits a
   pointer for a non-required object property, confirmed by comparing
   `gen.go`'s `PutExtensionsIdJSONBody` (all pointers, Phase 1) against
   today's `PutCustomerJSONBody`/`PutCustomersIdJSONBody` (all plain)).
3. **`bin-api-manager/pkg/servicehandler/customer.go`**: `CustomerUpdate`
   and `CustomerSelfUpdate` signatures change to accept pointers for all 7
   fields (except `id`/`a`, which stay as-is).
4. **`bin-api-manager/pkg/servicehandler/main.go`**: `ServiceHandler`
   interface declarations for both methods updated to match; `mock_main.go`
   regenerated.
5. **`bin-common-handler/pkg/requesthandler/customer_customer.go`**:
   `CustomerV1CustomerUpdate` signature changes to pointers; marshals into
   `V1DataCustomersIDPut` as pointers too (mandatory per roadmap §3 step 4 —
   skipping this reintroduces the bug at the RabbitMQ hop, exactly as the
   mcpservers design doc's §4.4 explains).
6. **`bin-common-handler/pkg/requesthandler/main.go`**: `RequestHandler`
   interface declaration updated; `mock_main.go` regenerated.
7. **`bin-customer-manager/pkg/listenhandler/models/request/customers.go`**:
   `V1DataCustomersIDPut` fields become pointers with `omitempty` JSON tags.
8. **`bin-customer-manager/pkg/listenhandler/v1_customers.go`**
   (`processV1CustomersIDPut`): pass pointers through to
   `customerHandler.UpdateBasicInfo` unchanged in shape (no new logic at
   this layer — mirrors how Phase 1's `v1_extensions.go` stayed a thin
   pass-through and did its own conditional-map-building one layer up, at
   the business handler; Phase 2's shape differs slightly here only because
   Phase 2's actual "build the fields map" logic lives inside
   `customerHandler.UpdateBasicInfo` rather than in the listenhandler
   itself — see step 9).
9. **`bin-customer-manager/pkg/customerhandler/db.go`** (`UpdateBasicInfo`)
   — **THE ACTUAL FIX SITE**: signature changes from 7 plain values to 7
   pointers; the `fields := map[customer.Field]any{...}` literal
   (`db.go:180-188`, currently unconditional) becomes built conditionally,
   `if name != nil { fields[customer.FieldName] = *name }` per field,
   mirroring PR #1291 §4.1 exactly. `len(fields) == 0` short-circuits to
   `h.db.CustomerGet(ctx, id)` directly (there is no separate `Get` method
   on `customerHandler` wrapping a not-found translation the way
   `mcpserverhandler.Get`/`extensionHandler.Get` do — confirmed via
   `customerhandler/main.go`'s interface: `Get` exists as its own method
   too, `db.go` likely has a thin wrapper; the no-op path should call
   whatever the existing `Get`-equivalent path is so `ErrNotFound` mapping
   is reused, not duplicated — verify exact function name during
   implementation and use it, do not hand-roll a second `CustomerGet` call
   site).
10. **`bin-customer-manager/pkg/customerhandler/main.go`**:
    `CustomerHandler` interface's `UpdateBasicInfo` declaration updated;
    `mock_main.go` regenerated.
11. **`bin-customer-manager/cmd/customer-control/main.go`** (`runUpdate`,
    the CLI tool): calls `handler.UpdateBasicInfo` directly with plain
    `viper.GetString(...)` values today (`main.go:193-203`). This call site
    must be updated to pass pointers to keep compiling. **Behavioral note,
    not a defect to fix elsewhere**: the CLI currently has no per-flag
    "was this flag set" tracking — it reads every flag's current value
    (defaulting to `""` for unset string flags, `"POST"` for
    `webhook-method` per its declared default at line 171) and would need
    `cmd.Flags().Changed("name")`-style checks added to actually take
    advantage of the new partial-update semantics from the CLI. **Decision:
    out of scope for this PR** — wrap every value in a pointer unconditionally
    (`&nameVal` etc.), preserving the CLI's exact current all-fields-sent
    behavior. This is safe (no functional regression — the CLI keeps
    behaving exactly as it does today) and avoids scope creep into CLI UX
    redesign; a follow-up ticket can add `--field-was-set` flag-changed
    detection to the CLI if an operator ever asks for it (오버엔지니어링
    지양 — no speculative CLI UX work without a concrete request).
12. **RST docs** (`bin-api-manager/docsdev/source/customer_struct_customer.rst`):
    add an Implementation Hint note stating the new contract, mirroring
    Phase 1's `extension_struct_extension.rst`/`trunk_struct_trunk.rst`
    notes; clean Sphinx rebuild committed in the same commit.
13. **Tests** at every layer per roadmap §3 step 11 / mcpservers §7's
    structure — see §6 below.

## 4. Out of scope

- **`PUT /customer/billing_account_id`** and
  **`PUT /customers/{id}/billing_account_id`** — single-field PUTs, already
  exempt per roadmap §3.2 (row #26/#27). Not touched by this phase.
- **`bin-customer-manager/cmd/customer-control/main.go`'s CLI UX** (adding
  flag-changed detection) — see §3 step 11's explicit deferral.
- **`square-admin` frontend** — not verified in this repo (separate repo,
  per Phase 1's same caveat and the roadmap's own non-goal, roadmap §6).
  Flagged as a residual, unverified-but-low-risk item exactly as Phase 1's
  Round 3 review flagged it for `square-admin`'s extensions/trunks forms —
  a frontend that already sends every field unconditionally is unaffected
  either way.
- **`POST /customers`** (create, roadmap out of scope generally — mirrors
  mcpservers §3's reasoning: a brand-new customer has no prior value to
  preserve, every field either comes from the caller or takes its
  from-scratch default; `Create`'s DTO already uses plain values correctly
  and is not touched).
- **`webhook_method`'s validity is not currently checked at all** (no
  `IsValid()` method exists on `customer.WebhookMethod`, confirmed via
  `models/customer` package search — only 5 string constants are defined,
  no validation function). This phase does not add new validation; it only
  makes the field's *presence* conditional. If 대표님 wants webhook_method
  validated in the future, that is a separate, unrelated enhancement.

## 5. Risk / rollback

Same class of risk as Phase 1 (HIGH per roadmap §2 rows #1/#2): the fix
strictly reduces silent-data-loss surface (a field can no longer be wiped
by omission) and does not change validation strictness for values that ARE
provided. No schema/DB migration involved — purely Go-layer + OpenAPI
description changes. Rollback is a straight revert of the PR if needed.

## 6. Testing strategy

Mirrors Phase 1 exactly, adapted to this call chain's one deviation (fix
site is `customerHandler.UpdateBasicInfo`, not the listenhandler):

- **`bin-customer-manager/pkg/customerhandler/db_test.go`**: table-driven
  cases for `UpdateBasicInfo` — one field set / rest nil (assert exact
  `fields` map contents per field, not just "no error"); all-omitted
  no-op case (assert `dbhandler.CustomerUpdate` is NOT called, `Times(0)`,
  and whatever `Get`-equivalent path takes over IS called); an explicit
  `webhook_method: &WebhookMethodNone` (pointer to zero value) case
  distinct from `webhook_method: nil`, confirming the pointer-to-zero-value
  vs. omitted distinction holds for this enum-shaped field the same way
  Phase 1 proved it for `trunks.AuthTypes`/mcpservers' `auth_type`.
- **`bin-customer-manager/pkg/listenhandler/v1_customers_test.go`**:
  extend/add a table test confirming `processV1CustomersIDPut` passes
  pointers through to `UpdateBasicInfo` unchanged (thin pass-through,
  mechanical assertion).
- **`bin-api-manager/pkg/servicehandler/customer_test.go`**: update
  existing `CustomerUpdate`/`CustomerSelfUpdate` mock call sites for the
  new pointer signatures (mechanical); add at minimum one new case per
  method with a partial body, asserting the right pointers are non-nil vs.
  nil through the permission-check layer.
- **`bin-common-handler/pkg/requesthandler/customer_customer_test.go`**:
  mechanical signature update (pure pass-through layer, per roadmap §3
  step 4/11 — no new test cases required beyond confirming the pointer
  marshal still round-trips correctly, same treatment as Phase 1's
  `registrar_extensions_test.go`/`registrar_trunks_test.go`).
- **`bin-api-manager/server/customer_test.go`** and **`customers_test.go`**
  (HTTP layer — Phase 1's Round 1 review specifically flagged this as the
  layer where the dereference-to-zero-value bug actually lives and is
  invisible to every lower-layer test): add a name-only PUT test for BOTH
  `PutCustomer` and `PutCustomersId`, asserting the service-handler mock is
  called with `name` non-nil and the other 6 pointers nil.
- **`bin-customer-manager/cmd/customer-control/main_test.go`** (if it
  exists — verify during implementation): update `runUpdate`'s
  `UpdateBasicInfo` mock call site for the pointer signature; no new test
  cases needed since §3 step 11 keeps its current all-fields-sent behavior
  unchanged.
- **Revert-and-rerun requirement** (standing practice): for at least the
  `webhook_uri: nil` case (the highest-risk field per §1), confirm the new
  test genuinely FAILS against pre-fix `db.go` (git stash the fix, rerun,
  confirm FAIL, restore) before considering the phase complete.
- Full verification workflow per phase exit criteria (roadmap §5):
  `go mod tidy && go mod vendor && go generate ./... && go test ./... &&
  golangci-lint run` in every touched service
  (`bin-customer-manager`, `bin-common-handler`, `bin-api-manager`).

## 7. Non-goals

- No retroactive remediation for customer rows whose `webhook_uri`/other
  fields may have already been silently wiped in production by the pre-fix
  behavior (mirrors PR #1291 §8 and the roadmap §6 — raise separately to
  대표님 if a concrete customer complaint surfaces).
- No `square-admin` frontend changes bundled into this PR (roadmap §6).
- No new PATCH method / HTTP verb redesign (roadmap §6).
- No `webhook_method` value validation added (§4 above — out of scope,
  unrelated enhancement).
- No CLI flag-changed UX work for `customer-control` (§3 step 11).
