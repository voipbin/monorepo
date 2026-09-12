# Phase 5b: PUT /numbers/{id} and PUT /numbers/{id}/flow_id partial-update fix

Status: DRAFT (Round 0)
Author: CPO, per roadmap `docs/plans/2026-09-12-put-partial-update-migration-roadmap.md`
Phase: 5 of 8 (§4, roadmap rows #9 and #10, MEDIUM risk)
Precedent: PR #1291 (`81a9a6b98`, mcpservers), and this migration's own
Phase 1-4 (PRs #1293, #1295, #1298, #1299, #1300).

## 1. Problem statement

Two endpoints share the same two highest-risk fields and the same owning
service, so this phase covers both in one PR:

- `PUT /numbers/{id}` requires all 4 body fields (`call_flow_id`,
  `message_flow_id`, `name`, `detail`).
- `PUT /numbers/{id}/flow_id` requires both of its 2 body fields
  (`call_flow_id`, `message_flow_id`) -- a narrower, single-purpose PUT
  that updates only the flow bindings.

`call_flow_id`/`message_flow_id` are the highest-risk fields in both: a
live phone number's inbound call or message routing could be silently
wiped (collapsed to `uuid.Nil` via `uuid.FromStringOrNil` at the HTTP
boundary today) if a client omits one while only intending to change the
other, or update `name`/`detail`. This is functionally the identical bug
shape to Phase 3's `routes/{id}`'s `provider_id`/`target` and Phase 4a's
`wait_flow_id`.

## 2. Call chain (verified via direct code read)

```
PUT /numbers/{id}                          PUT /numbers/{id}/flow_id
  bin-api-manager/server/numbers.go:         bin-api-manager/server/numbers.go:
    PutNumbersId                               PutNumbersIdFlowIds
        |                                           |
  bin-api-manager/pkg/servicehandler/numbers.go:
    NumberUpdate                               NumberUpdateFlowIDs
        |                                           |
  bin-common-handler/pkg/requesthandler/number_number.go:
    NumberV1NumberUpdate                       NumberV1NumberUpdateFlowIDs (verify exact name)
        |                                           |
  bin-number-manager/pkg/listenhandler/
    models/request/v1_numbers.go: V1DataNumbersIDPut / V1DataNumbersIDFlowIDPut
    v1_numbers.go: processV1NumbersIDPut / processV1NumbersIDFlowIDPut   <-- ACTUAL FIX SITE
      (both build `fields := map[number.Field]any{...}` directly inline,
      unconditionally, v1_numbers.go:155-160 for the ID PUT variant --
      confirmed via direct read; the flow_id variant's exact line range
      to be confirmed at implementation time but follows the identical
      inline-fields-map pattern per file structure)
        |
  bin-number-manager/pkg/numberhandler/number.go: Update(ctx, id, fields)
    (thin passthrough, already accepts a pre-built fields map -- no
    change needed here, confirmed at number.go:314-329)
        |
  bin-number-manager/pkg/dbhandler/number.go: NumberUpdate   <-- already safe,
                                                                  no change needed
    (len(fields)==0 guard confirmed at dbhandler/number.go:223-225)
```

**Key structural difference from every prior phase**: this service's
`numberHandler.Update(ctx, id, fields map[number.Field]any)` already
accepts a pre-built field map (not individual typed parameters), so the
business-handler layer itself needs NO signature change. **The actual fix
site is one layer up, in `bin-number-manager/pkg/listenhandler/v1_numbers.go`**,
where `processV1NumbersIDPut` and `processV1NumbersIDFlowIDPut` build the
`fields` map directly inline from the unmarshaled request DTO
(`v1_numbers.go:155-160`, confirmed):

```go
fields := map[number.Field]any{
    number.FieldCallFlowID:    req.CallFlowID,
    number.FieldMessageFlowID: req.MessageFlowID,
    number.FieldName:          req.Name,
    number.FieldDetail:        req.Detail,
}
```

This is unconditional today: every key is always present in the map
regardless of what the caller sent, because `req.CallFlowID` etc. are
plain (non-pointer) values on `V1DataNumbersIDPut`
(`bin-number-manager/pkg/listenhandler/models/request/v1_numbers.go:34-39`,
confirmed) -- an omitted `call_flow_id` in the incoming JSON unmarshals to
`uuid.Nil`, which is then written into `fields` as a real (wrong) value,
identical in shape to every prior phase's bug.

## 3. In scope

Per roadmap §3's established pattern, adapted for this service's
inline-fields-map structure (step 7 below is the deviation from the
standard business-handler fix site; everything else follows the pattern
verbatim):

1. **OpenAPI schema** -- `bin-openapi-manager/openapi/paths/numbers/id.yaml`
   and `bin-openapi-manager/openapi/paths/numbers/id_flow_id.yaml`: remove
   the `required:` block from both (4 fields optional in the first, 2 in
   the second); update each field's `description` to state the
   omitted-means-unchanged contract.
2. **`bin-api-manager/server/numbers.go`** (`PutNumbersId`,
   `PutNumbersIdFlowIds`): stop unconditionally dereferencing
   OpenAPI-generated fields via `uuid.FromStringOrNil` to plain
   `uuid.UUID` values (`numbers.go:189-190` and `numbers.go:231-232`,
   confirmed -- both call sites use the exact silent-collapse-to-Nil
   anti-pattern); pass the generated pointers straight through once the
   schema is optional, with malformed-but-present values returning 400
   via `uuid.FromString` (not `FromStringOrNil`), mirroring Phase 3's
   `provider_id` fix and Phase 4a's `wait_flow_id`/`tag_ids` fix. Same
   same-file precedent as those phases:
   `bin-api-manager/server/campaigns.go:422-431`'s `next_campaign_id`
   handling.
3. **`bin-api-manager/pkg/servicehandler/numbers.go`**: `NumberUpdate` and
   `NumberUpdateFlowIDs` signatures change to accept pointers for
   `callFlowID`/`messageFlowID` (and `name`/`detail` for `NumberUpdate`).
4. **`bin-api-manager/pkg/servicehandler/main.go`**: interface updated;
   mock regenerated.
5. **`bin-common-handler/pkg/requesthandler/number_number.go`** (verify
   exact filename at implementation time): the RPC client methods
   (`NumberV1NumberUpdate` and the flow-ID-update equivalent -- confirm
   exact method name, do not assume) accept and marshal pointers.
6. **`bin-common-handler/pkg/requesthandler/main.go`**: interface
   declaration updated; `mock_main.go` regenerated.
7. **`bin-number-manager/pkg/listenhandler/models/request/v1_numbers.go`**:
   `V1DataNumbersIDPut` and `V1DataNumbersIDFlowIDPut` fields become
   pointers with `omitempty` JSON tags (currently plain `uuid.UUID`/
   `string` with `omitempty`, confirmed at lines 34-47 -- `omitempty` on
   a non-pointer `uuid.UUID` is already a latent bug in the opposite
   direction: it never omits a non-zero UUID from the wire, but also
   never distinguishes "the zero UUID was explicitly sent" from
   "omitted", which is exactly what pointer conversion fixes).
8. **`bin-number-manager/pkg/listenhandler/v1_numbers.go`**
   (`processV1NumbersIDPut`, `processV1NumbersIDFlowIDPut`) --
   **THE ACTUAL FIX SITE** (deviation from the standard pattern's step 7,
   since this service has no separate business-handler-level field-map
   construction to fix): the inline `fields := map[number.Field]any{...}`
   literal construction becomes conditional per non-nil pointer, directly
   in the listenhandler function. `len(fields) == 0` short-circuits to
   this service's existing `Get`-equivalent (confirm the exact call --
   `numberHandler` likely has a `Get(ctx, id)` wrapper; verify and reuse
   it rather than introducing a new one, per this monorepo's established
   discipline of not duplicating `ErrNotFound` translation).
9. **`bin-number-manager/cmd/number-control/main.go`**: confirmed to
   exist and to call `numberHandler.Update` directly with an
   already-built `fields` map (per this service's own CLAUDE.md
   documenting `./bin/number-control number update --id <uuid>
   --call-flow-id <flow-uuid>`) -- **must be updated** to preserve
   current behavior. Verify at implementation time exactly how this CLI
   builds its `fields` map (whether it already conditionally includes
   only flags the user passed, in which case NO CLI change is needed
   since it already speaks the pointer-shaped-map language natively; or
   whether it unconditionally includes all fields regardless of which
   flags were passed, in which case it needs the same
   pointer-wrap-preserving-current-behavior treatment as Phase 2/3/4's
   CLIs). Do not assume either way -- this is the one open question this
   design doc flags for implementation-time verification with a
   concrete, cited answer, per this monorepo's review-loop standard
   (an assumption here previously caused Phase 3's Round 1 REQUEST
   CHANGES for a missed CLI call site).
10. RST docs (`bin-api-manager/docsdev/source/number_struct_number.rst`
    -- verify exact filename) updated with an Implementation Hint note
    (omission=unchanged for all 4/2 fields respectively), clean Sphinx
    rebuild committed.

## 4. Out of scope (explicitly)

- No changes to `PUT /conferences/{id}` (separate PR, sibling doc,
  different service per the one-PR-per-repo rule).
- No changes to number purchase (`POST /numbers`), release, or metadata
  update (`PUT /numbers/{id}/metadata`) -- not in the 34-endpoint
  inventory.
- No flag-changed detection added to the `number-control` CLI if its
  current behavior turns out to require pointer-wrapping (per §3 step 9)
  -- deferred as a named follow-up, matching every prior phase's CLI
  scoping decision (오버엔지니어링 지양).

## 5. Testing strategy

- **`bin-number-manager/pkg/listenhandler/v1_numbers_test.go`**:
  table-driven cases for `processV1NumbersIDPut` and
  `processV1NumbersIDFlowIDPut` -- one field set / rest nil per case
  (assert exact `fields` map contents); all-omitted no-op case for each
  endpoint (assert `numberHandler.Update` is NOT called, `Times(0)`).
  Since the fix site is the listenhandler itself here (not a separate
  business-handler layer), this is where the core partial-update logic
  test coverage lives for this phase, analogous to where Phase 1-4 put
  their business-handler-layer tests.
- **`bin-api-manager/server/numbers_test.go`**: a
  `Test_numbersIDPut_NameOnly`-style regression test and a
  `Test_numbersIDPut_MalformedCallFlowID`-style test (malformed
  `call_flow_id` returns 400, does not reach `NumberUpdate`), mirroring
  Phase 3/4a/5a's HTTP-layer regression test pattern; equivalent tests
  for `PutNumbersIdFlowIds`.
- **`bin-api-manager/pkg/servicehandler/numbers_test.go`** and
  **`bin-common-handler/pkg/requesthandler/number_number_test.go`**:
  updated for the pointer signature change.
- **Revert-and-rerun** (mandatory per roadmap §5): temporarily simulate
  the pre-fix unconditional inline-fields-map construction in
  `v1_numbers.go`, rerun the new partial-update tests, confirm they fail
  (reproducing the silent-wipe-to-`uuid.Nil` bug for
  `call_flow_id`/`message_flow_id`), then revert and reconfirm green.

## 6. Non-goals

- No retroactive data remediation for numbers whose `call_flow_id`/
  `message_flow_id` may have already been silently wiped by the pre-fix
  behavior in production (mirrors roadmap §6).
- No frontend (square-admin/square-talk) changes bundled into this PR
  unless this design doc identifies a concrete behavior change requiring
  one -- none identified.
- No new PATCH method; stays PUT with optional-field semantics.
- No unification of `PUT /numbers/{id}` and `PUT /numbers/{id}/flow_id`
  into a single endpoint -- both stay as separate, independently-scoped
  PUTs; this migration only fixes their partial-update semantics, it
  does not redesign the API surface.
