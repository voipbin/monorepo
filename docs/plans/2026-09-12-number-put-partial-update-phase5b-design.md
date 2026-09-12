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
    NumberV1NumberUpdate                       NumberV1NumberUpdateFlowID
        |                                           |
  bin-number-manager/pkg/listenhandler/
    models/request/v1_numbers.go: V1DataNumbersIDPut / V1DataNumbersIDFlowIDPut
    v1_numbers.go: processV1NumbersIDPut / processV1NumbersIDFlowIDsPut   <-- ACTUAL FIX SITE
      (both build `fields := map[number.Field]any{...}` directly inline,
      unconditionally; ID PUT variant at v1_numbers.go:155-160, flow_id
      variant at v1_numbers.go:256-259 -- both confirmed via direct read)
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
where `processV1NumbersIDPut` and `processV1NumbersIDFlowIDsPut` build the
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
   pointers with `omitempty` JSON tags. **Correction from Round 1
   review**: only `V1DataNumbersIDPut` (lines 34-39) currently has
   `omitempty` on its plain `uuid.UUID`/`string` fields -- `omitempty` on
   a non-pointer `uuid.UUID` is already a latent bug in the opposite
   direction (it never omits a non-zero UUID from the wire, but also
   never distinguishes "the zero UUID was explicitly sent" from
   "omitted", exactly what pointer conversion fixes). `V1DataNumbersIDFlowIDPut`
   (lines 44-47, `CallFlowID uuid.UUID \`json:"call_flow_id"\`` and
   `MessageFlowID uuid.UUID \`json:"message_flow_id"\``) has **no**
   `omitempty` tags at all today -- adding `omitempty` there when
   converting to pointers is a genuinely new addition, not a
   modification of existing tag behavior.
8. **`bin-number-manager/pkg/listenhandler/v1_numbers.go`**
   (`processV1NumbersIDPut`, `processV1NumbersIDFlowIDsPut`) --
   **THE ACTUAL FIX SITE** (deviation from the standard pattern's step 7,
   since this service has no separate business-handler-level field-map
   construction to fix): the inline `fields := map[number.Field]any{...}`
   literal construction becomes conditional per non-nil pointer, directly
   in the listenhandler function, at both sites (`v1_numbers.go:155-160`
   and `v1_numbers.go:256-259`). `len(fields) == 0` short-circuits to
   this service's existing `Get`-equivalent (confirm the exact call --
   `numberHandler` likely has a `Get(ctx, id)` wrapper; verify and reuse
   it rather than introducing a new one, per this monorepo's established
   discipline of not duplicating `ErrNotFound` translation).
9. **`bin-number-manager/cmd/number-control/main.go`**: confirmed to
   exist and to call `numberHandler.Update` directly with an
   already-built `fields` map (line 385, `runUpdate`/equivalent). **Round
   1 review resolved the open question this doc originally left
   unverified**: this CLI already conditionally builds `fields` only for
   flags the operator explicitly set (`main.go:363-384`, e.g. `if
   viper.IsSet("name")`, `if viper.GetString("call-flow-id") != ""`,
   etc.), and already guards `len(fields) == 0`. **No CLI change is
   needed** -- it already speaks the pointer-shaped, omit-means-unchanged
   contract natively today, unlike Phase 3/4's CLIs which required an
   explicit pointer-wrap fix.
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
- No flag-changed detection added anywhere -- deferred as a named
  follow-up, matching every prior phase's CLI scoping decision
  (오버엔지니어링 지양). Not applicable to `number-control` itself since
  it requires no change (§3 item 9).

## 5. Testing strategy

- **`bin-number-manager/pkg/listenhandler/v1_numbers_test.go`**:
  table-driven cases for `processV1NumbersIDPut` and
  `processV1NumbersIDFlowIDsPut` -- one field set / rest nil per case
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

## 6. Round 1 review disposition

Round 1 (`deleg_81ae9039` task 1) verdict: REQUEST CHANGES. The
load-bearing architectural claim (fix site is the listenhandler's inline
fields-map construction, not `numberHandler.Update`) was independently
verified correct, along with the call-chain, `server/numbers.go`
`FromStringOrNil` sites, and `dbhandler.NumberUpdate`'s `len(fields)==0`
guard. Two factual errors and one self-contradiction were found and
fixed: (1) §3 item 7's claim that both request DTOs already have
`omitempty` was wrong for `V1DataNumbersIDFlowIDPut` (no `omitempty` tags
today) -- corrected to distinguish the two structs' actual current state.
(2) §3 item 9 left the CLI's field-construction behavior as an
unresolved "open question for implementation time" despite this exact
failure mode being the doc's own cited cautionary precedent (Phase 3's
Round 1 REQUEST CHANGES for a missed CLI verification) -- resolved
directly: `cmd/number-control/main.go:363-384` already conditionally
builds fields per explicitly-set flag and already guards
`len(fields)==0`; **no CLI change is needed**, corrected throughout §3
item 9, §4, and the call-chain diagram. (3) Naming inaccuracies fixed:
the flow_id listenhandler function is `processV1NumbersIDFlowIDsPut`
(plural "IDs", not singular), and the RPC client method is
`NumberV1NumberUpdateFlowID` (singular, no trailing "s") -- both
corrected in §2's call-chain diagram and throughout §3/§5. The flow_id
variant's inline fields-map is confirmed at `v1_numbers.go:256-259`
(previously left as "to be confirmed at implementation time").

## 7. Non-goals

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
