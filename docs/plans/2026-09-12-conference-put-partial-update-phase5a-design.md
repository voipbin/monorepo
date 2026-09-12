# Phase 5a: PUT /conferences/{id} partial-update fix

Status: DRAFT (Round 0)
Author: CPO, per roadmap `docs/plans/2026-09-12-put-partial-update-migration-roadmap.md`
Phase: 5 of 8 (§4, roadmap row #8, MEDIUM risk)
Precedent: PR #1291 (`81a9a6b98`, mcpservers), and this migration's own
Phase 1-4 (PRs #1293, #1295, #1298, #1299, #1300).

## 1. Problem statement

`PUT /conferences/{id}` requires all 6 body fields (`name`, `detail`,
`data`, `timeout`, `pre_flow_id`, `post_flow_id`). A client that omits
any field does not get a validation error -- every field is a plain
(non-pointer) value from the OpenAPI-generated JSON body down to
`bin-conference-manager/pkg/conferencehandler/conference.go`'s `Update`,
which **unconditionally** builds the full 6-key SQL field map
(`conference.go:263-270`) regardless of what the caller sent.

The highest-risk fields are `pre_flow_id`/`post_flow_id`: omitting them
(or a client bug that serializes a Go zero-value UUID) silently detaches
a conference's pre/post flow hooks -- a conference that used to run a
greeting flow before participants join, or a wrap-up flow after the
conference ends, loses that behavior with a `200 OK` response and no
error. `data` (an arbitrary JSON object) and `timeout` (an integer,
where `0` is a real, valid value per `conference.go:258-260`'s own
`if timeout > 0 && timeout < 60` normalization logic) share the same
risk shape as prior phases' object/zero-value fields.

## 2. Call chain (verified via direct code read)

```
PUT /conferences/{id}
  bin-api-manager/server/conferences.go: PutConferencesId
        |
  bin-api-manager/pkg/servicehandler/conferences.go: ConferenceUpdate
        |
  bin-common-handler/pkg/requesthandler/conference_conference.go: ConferenceV1ConferenceUpdate
        |
  bin-conference-manager/pkg/listenhandler/
    models/request/v1_conferences.go: V1DataConferencesIDPut
    v1_conferences.go: processV1ConferencesIDPut
        |
  bin-conference-manager/pkg/conferencehandler/conference.go: Update   <-- ACTUAL FIX SITE
    (builds fields map, currently unconditional, conference.go:263-270)
        |
  bin-conference-manager/pkg/dbhandler/conference.go: ConferenceUpdate   <-- already safe,
                                                                             no change needed
```

Confirmed via direct code read: `conferenceHandler.Update`'s signature is
`Update(ctx, id, name string, detail string, data map[string]any, timeout
int, preFlowID uuid.UUID, postFlowID uuid.UUID)` and unconditionally
builds `fields := map[conference.Field]any{...}` (`conference.go:263-270`)
before calling `h.db.ConferenceUpdate`. The function also has a
non-mechanical side effect after the DB write: `if res.Timeout > 0` it
calls `h.reqHandler.ConferenceV1ConferenceDeleteDelay` to (re)schedule the
conference's auto-termination timer (`conference.go:283-286`). This must
continue to fire correctly when `timeout` is omitted (using the
already-current stored value) and when `timeout` is explicitly set to a
new value -- confirmed both cases naturally work once the fix reads
`res.Timeout` (the post-update DB value) rather than the raw incoming
parameter, which the existing code already does (`res.Timeout`, not
`timeout`), so no special-casing is needed here beyond making `timeout`
itself an optional pointer.

## 3. In scope

Per roadmap §3's established pattern, applied to this chain:

1. **OpenAPI schema** -- `bin-openapi-manager/openapi/paths/conferences/id.yaml`:
   remove the `required:` block (all 6 fields become optional); update
   each field's `description` to state the omitted-means-unchanged
   contract. `data` (a `type: object` with `additionalProperties: true`)
   needs an explicit note that an explicit empty object (`{}`) clears
   custom data, distinct from omitting the field, mirroring the
   `tech_headers`/`tag_ids` empty-vs-omit pattern from Phase 4.
   `timeout: 0` is a real value meaning "no auto-termination timer"
   (confirmed: the existing `if res.Timeout > 0` guard at
   `conference.go:283` only schedules the delete-delay when timeout is
   positive), distinct from omission, mirroring Phase 3's `priority: 0`
   and Phase 4a's `wait_timeout: 0` precedent.
2. **`bin-api-manager/server/conferences.go`** (`PutConferencesId`): stop
   unconditionally dereferencing OpenAPI-generated fields to plain
   values; pass the generated pointers straight through once the schema
   is optional. `pre_flow_id`/`post_flow_id` are UUID strings today --
   malformed values must return 400 via `uuid.FromString` (not
   `uuid.FromStringOrNil`), mirroring Phase 3's `provider_id`/Phase 4a's
   `wait_flow_id` fix (`bin-api-manager/server/campaigns.go:422-431`'s
   `next_campaign_id` handling remains the correct same-file precedent
   for the omit-vs-malformed-UUID 3-state pattern).
3. **`bin-api-manager/pkg/servicehandler/conferences.go`**: `ConferenceUpdate`
   signature changes to accept pointers for all 6 fields.
4. **`bin-api-manager/pkg/servicehandler/main.go`**: interface updated;
   mock regenerated.
5. **`bin-common-handler/pkg/requesthandler/conference_conference.go`**:
   `ConferenceV1ConferenceUpdate` accepts and marshals pointers.
6. **`bin-common-handler/pkg/requesthandler/main.go`**: interface
   declaration updated; `mock_main.go` regenerated.
7. **`bin-conference-manager/pkg/listenhandler/models/request/v1_conferences.go`**:
   the PUT wire DTO's fields become pointers with `omitempty` JSON tags.
8. **`bin-conference-manager/pkg/listenhandler/v1_conferences.go`**
   (`processV1ConferencesIDPut`): pass pointers through to
   `conferenceHandler.Update` unchanged in shape.
9. **`bin-conference-manager/pkg/conferencehandler/conference.go`**
   (`Update`) -- **THE ACTUAL FIX SITE**: signature changes from 6 plain
   values to 6 pointers; the `fields := map[conference.Field]any{...}`
   literal (currently unconditional, `conference.go:263-270`) becomes
   built conditionally. The `if timeout > 0 && timeout < 60 { timeout =
   defaultConferenceTimeout }` normalization (`conference.go:258-260`)
   becomes conditional on the timeout pointer being non-nil (an omitted
   timeout should not trigger this normalization against a zero
   placeholder value). `len(fields) == 0` short-circuits to `h.Get(ctx,
   id)` (confirmed `conferenceHandler` has its own `Get` wrapper with
   `ErrNotFound` translation via `h.db.ConferenceGet` -- reuse it
   directly rather than a raw db call, per this monorepo's established
   discipline; verify exact wrapper name at implementation time and cite
   its line number). The post-write `ConferenceV1ConferenceDeleteDelay`
   re-scheduling call (`conference.go:283-286`) reads `res.Timeout` (the
   post-write DB value), which already works correctly for both the
   omitted-timeout case (unchanged stored value) and the
   explicitly-set-timeout case (new value) without modification -- but
   for the all-omitted no-op path, this re-scheduling call must NOT run
   (a no-op PUT should not reset the conference's existing termination
   timer), so the no-op short-circuit must return before reaching this
   logic, which it naturally does by returning early from `h.Get`.
10. **`bin-conference-manager/cmd/conference-control/main.go`** (if this
    CLI binary exists and calls `conferenceHandler.Update` directly --
    confirm at implementation time via `search_files(target='files')` on
    `cmd/`, do not assume absence or presence): if a direct call exists,
    wrap every value in a pointer unconditionally, preserving current
    all-fields-sent behavior, mirroring Phase 2/3/4's CLI treatment.
11. RST docs (`bin-api-manager/docsdev/source/conference_struct_conference.rst`
    -- verify exact filename at implementation time) updated with an
    Implementation Hint note (omission=unchanged, `data: {}` clears
    custom data, `timeout: 0` disables auto-termination distinct from
    omission), clean Sphinx rebuild committed.

## 4. Out of scope (explicitly)

- No changes to `PUT /numbers/{id}` or `PUT /numbers/{id}/flow_id`
  (separate PR, sibling doc, different service per the one-PR-per-repo
  rule).
- No changes to conference creation (`POST /conferences`), which is a
  separate, already-fully-required-by-design endpoint not in scope for
  this migration.
- No changes to `UpdateRecordingID` or other single-purpose conference
  update methods (`conference.go:294+`) -- these are internal,
  system-driven updates, not the customer-facing `PUT /conferences/{id}`
  endpoint, and are not part of the 34-endpoint inventory.
- No flag-changed detection added to any CLI, if one exists -- deferred
  as a named follow-up per roadmap discipline (오버엔지니어링 지양).

## 5. Testing strategy

- **`bin-conference-manager/pkg/conferencehandler/conference_test.go`**:
  table-driven cases for `Update` -- one field set / rest nil (assert
  exact `fields` map contents per field); all-omitted no-op case (assert
  `dbhandler.ConferenceUpdate` is NOT called, `Times(0)`, and
  `ConferenceV1ConferenceDeleteDelay` is also NOT called in this case);
  a `timeout: intPtr(0)` case distinct from `nil`, confirming `0` is
  treated as a real, set value (disables auto-termination) rather than
  omission; a `data: &map[string]any{}` (pointer to an explicit empty
  map) case distinct from `data: nil`, confirming the explicit-clear-vs-
  omit distinction holds for the object field.
- **`bin-conference-manager/pkg/listenhandler/v1_conferences_test.go`**:
  updated for the pointer signature change; one full-fields "normal"
  case is sufficient here since the partial-update logic lives at the
  conferencehandler layer.
- **`bin-api-manager/server/conferences_test.go`**: a new
  `Test_conferencesIDPut_NameOnly`-style regression test (name-only PUT
  does not clobber other fields at the HTTP boundary) and a
  `Test_conferencesIDPut_MalformedPreFlowID`-style test (malformed
  `pre_flow_id`/`post_flow_id` returns 400, does not reach
  `ConferenceUpdate`), mirroring Phase 3/4a's HTTP-layer regression test
  pattern.
- **`bin-api-manager/pkg/servicehandler/conferences_test.go`** and
  **`bin-common-handler/pkg/requesthandler/conference_conference_test.go`**:
  updated for the pointer signature change.
- **Revert-and-rerun** (mandatory per roadmap §5): temporarily simulate
  the pre-fix unconditional field-map population in `conference.go`'s
  `Update`, rerun the new partial-update tests, confirm they fail
  (reproducing the silent-wipe bug for `pre_flow_id`/`post_flow_id`),
  then revert and reconfirm green.

## 6. Non-goals

- No retroactive data remediation for conferences whose `pre_flow_id`/
  `post_flow_id` may have already been silently wiped by the pre-fix
  behavior in production (mirrors roadmap §6).
- No frontend (square-admin/square-talk) changes bundled into this PR
  unless this design doc identifies a concrete behavior change requiring
  one -- none identified.
- No new PATCH method; stays PUT with optional-field semantics.
