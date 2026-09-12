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
   `ErrNotFound` translation, confirmed as `Get` at `conference.go:167-181`
   -- reuse it directly rather than a raw db call, per this monorepo's
   established discipline). The post-write `ConferenceV1ConferenceDeleteDelay`
   re-scheduling call (`conference.go:283-286`) reads `res.Timeout` (the
   post-write DB value), which already works correctly for both the
   omitted-timeout case (unchanged stored value) and the
   explicitly-set-timeout case (new value) without modification -- but
   for the all-omitted no-op path, this re-scheduling call must NOT run
   (a no-op PUT should not reset the conference's existing termination
   timer), so the no-op short-circuit must return before reaching this
   logic, which it naturally does by returning early from `h.Get`.
10. **`bin-conference-manager/cmd/conference-control/main.go`** --
    **CONFIRMED to exist and to call `conferenceHandler.Update` directly**
    (`runUpdate`, `main.go:338-382`, direct call at `main.go:367`). Its
    current behavior unconditionally passes `map[string]interface{}{}`
    (an always-empty map, never the operator's actual custom data) and
    zero-value UUIDs for any flow ID flag left unset, on every single
    invocation regardless of which flags the operator passed. **This is
    a footgun under the new pointer contract**: if this CLI is
    pointer-wrapped naively ("wrap every value unconditionally,
    preserving current behavior"), every CLI-driven update -- including
    a name-only or timeout-only edit -- would continue to send a real,
    non-nil `&map[string]interface{}{}` for `data`, which under the new
    semantics means "explicitly clear all custom data," silently wiping
    a conference's custom data on every CLI update. This is the same
    risk class this doc already calls out for the HTTP API's `data: {}`
    vs `data: nil` distinction (§3 item 1) and must not be reintroduced
    at the CLI layer.

    **Confirmed scope (Round 2 review found the original wording
    understated this)**: `cmdUpdate()` (`main.go:327-333`) today has
    **no `--data` flag at all** -- for either `create` or `update`. This
    is therefore not a pure "gate an existing flag" fix; it requires
    **adding a new `--data` flag** to `cmdUpdate()`, of type string
    (JSON-encoded object, consistent with how this CLI already handles
    other JSON-shaped inputs elsewhere in the monorepo), parsed via
    `json.Unmarshal` into `map[string]any` with a clear error message on
    malformed JSON (returned before calling `Update`, not silently
    ignored). The resolution is:
    - Register a new `--data` string flag on `cmdUpdate()` (default:
      unset/empty string, not `"{}"`, so it round-trips correctly through
      `viper.IsSet("data")`).
    - Build `fields` conditionally per flag, gating each field on
      `viper.IsSet(...)` for `name`/`detail`/`data` (matching
      `number-control`'s pattern for its `name`/`detail` flags exactly,
      `cmd/number-control/main.go:365-370`-equivalent) and on
      `viper.GetString(...) != ""` for `pre-flow-id`/`post-flow-id`
      (matching `number-control`'s pattern for its
      `call-flow-id`/`message-flow-id` flags, which use the
      non-empty-string check rather than `IsSet` --
      `cmd/number-control/main.go:371-378`-equivalent). Do not apply a
      single blanket pattern uniformly across all fields; the two
      sub-patterns exist in the reference CLI for a reason (string flags
      with a meaningful non-empty default vs. flags whose zero value is
      never itself a valid intentional input) and must be matched
      per-field-type, not applied identically to every field.
    - Only pass a non-nil `data` pointer when `viper.IsSet("data")` is
      true (after successful JSON parse), and only pass non-nil flow-ID
      pointers when the corresponding flag string is non-empty, not
      unconditionally on every invocation. This is a genuine (small)
      behavior change to the CLI's flag-parsing and flag surface (one
      new flag), not a pure pointer-wrap, and must be implemented and
      tested as such.
11. RST docs (`bin-api-manager/docsdev/source/conference_struct_conference.rst`
    -- filename confirmed to exist at this exact path) updated with an
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
- **`bin-conference-manager/cmd/conference-control/main.go`**
  (`runUpdate`): after adding the new `--data` flag and converting to
  flag-conditional field construction, add/update tests confirming a
  name-only CLI invocation (no `--data` flag passed) does NOT pass a
  non-nil `data` pointer (preventing the accidental clear identified in
  §3 item 10); a `--data '{"key":"value"}'` invocation does pass a
  non-nil pointer with the parsed content; a malformed `--data` JSON
  value returns a clear CLI error before calling `Update`.
- **Revert-and-rerun** (mandatory per roadmap §5): temporarily simulate
  the pre-fix unconditional field-map population in `conference.go`'s
  `Update`, rerun the new partial-update tests, confirm they fail
  (reproducing the silent-wipe bug for `pre_flow_id`/`post_flow_id`),
  then revert and reconfirm green.

## 6. Round 1 review disposition

Round 1 (`deleg_81ae9039` task 0) verdict: REQUEST CHANGES. All call-chain,
signature, `Get`-wrapper, timeout/re-scheduling, and RST-filename claims
were independently verified correct. One MAJOR finding: §3 item 10's
original phrasing conditionally hedged on whether
`cmd/conference-control` exists and did not flag that this CLI
unconditionally sends `map[string]interface{}{}` for `data` on every
invocation, which under the new pointer contract would silently clear
custom conference data on every CLI-driven update (the same risk class
already called out for the HTTP API in §3 item 1). Fixed: §3 item 10
rewritten to state the CLI's existence and direct-call site as confirmed
fact (`main.go:338-382`, call at `main.go:367`), and to require
flag-conditional field construction (mirroring `number-control`'s
`viper.IsSet(...)` pattern) rather than a naive unconditional
pointer-wrap. §5 testing strategy updated with a corresponding CLI test
requirement. §3 item 9's `Get`-wrapper hedge and item 11's RST-filename
hedge were also tightened to state the already-confirmed facts
(`Get`, `conference.go:167-181`; RST file confirmed to exist) rather than
deferring trivially-verifiable facts to implementation time.

## 7. Round 2 review disposition

Round 2 (`deleg_16762672` task 0) verdict: REQUEST CHANGES. All Round 1
factual claims (signature, field-map, `Get` wrapper, timeout/rescheduling,
RST file, `campaigns.go` precedent) re-verified correct on independent
re-derivation. One MAJOR gap found: §3 item 10's fix as worded presupposed
an existing `--data` CLI flag to gate -- **no such flag exists today**
(`cmdUpdate()`/`cmdCreate()` have no `--data` flag at all), so the
one-line "only pass non-nil when `--data` was provided" resolution could
not actually be implemented against current code; the fix requires
**adding** a new flag (type, JSON parsing, error handling), not just
gating an existing one. Also flagged: citing `number-control`'s
`viper.IsSet` as *the* mirrored pattern was imprecise, since
`number-control` actually uses two different sub-patterns
(`viper.IsSet` for `name`/`detail`, `viper.GetString(...) != ""` for its
flow-ID flags) and conference's flow-ID fields are the closer match to
the latter. Fixed: §3 item 10 rewritten to state the new-flag requirement
explicitly (flag registration, JSON unmarshal with error handling,
default value choice) and to cite the correct number-control sub-pattern
per field type rather than a single blanket precedent. §5 testing
strategy updated to test flag-addition + JSON-parse-error behavior, not
just gating.

## 8. Non-goals

- No retroactive data remediation for conferences whose `pre_flow_id`/
  `post_flow_id` may have already been silently wiped by the pre-fix
  behavior in production (mirrors roadmap §6).
- No frontend (square-admin/square-talk) changes bundled into this PR
  unless this design doc identifies a concrete behavior change requiring
  one -- none identified.
- No new PATCH method; stays PUT with optional-field semantics.
