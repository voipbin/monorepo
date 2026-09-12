# Phase 6b design: PUT partial-update for teams

Status: DRAFT (Round 0)
Author: CPO, per roadmap `docs/plans/2026-09-12-put-partial-update-migration-roadmap.md` §4 Phase 6.
Reference pattern: `docs/plans/2026-09-12-mcp-server-put-partial-update-design.md` (PR #1291,
merged `81a9a6b98`); same technical pattern applied by every prior phase
(1-5) — see roadmap §3 for the canonical 11-step pattern, not re-derived
here except where this endpoint's cross-field validation forces a
deviation (§3 below).

Separate PR from Phase 6a (campaigns/outplans) per this monorepo's
one-PR-per-repo-boundary rule: `teams` is owned by `bin-ai-manager`, a
different service.

## 1. Scope

`PUT /teams/{id}` — `name`, `detail`, `start_member_id`, `members`
(4 fields `required`; `parameter` is already optional, making this
endpoint hybrid like `providers`/`flows` per roadmap §1 — confirmed
directly against `bin-openapi-manager/openapi/paths/teams/id.yaml:107-111`,
`required: [name, detail, start_member_id, members]`, `parameter` absent
from that list). Roadmap §2 row 13 rates this MEDIUM: `members`/
`start_member_id` carry wipe risk on the team's roster/entry-point graph.

## 2. Call chain (verified against current source)

1. `bin-openapi-manager/openapi/paths/teams/id.yaml:61-111` — `put`
   requires `name`/`detail`/`start_member_id`/`members`.
2. `bin-api-manager/server/teams.go:180` `PutTeamsId` — binds
   `PutTeamsIdJSONBody`; converts `req.StartMemberId` (a plain `string`)
   via `uuid.FromStringOrNil` (line 211, **not** `uuid.FromString`,
   meaning a malformed UUID today silently becomes `uuid.Nil` rather than
   a 400 — same pre-existing-bug shape roadmap Phase 5's `server/
   conferences.go`/`server/numbers.go` fixed via `uuid.FromString`; this
   phase inherits the same fix per §3 item 2); converts
   `req.Members` via `convertOpenAPIMembers(...)`; builds `parameter` from
   `req.Parameter` only `if req.Parameter != nil` (already-optional field,
   line 214-217); calls `h.serviceHandler.TeamUpdate(ctx, a, target,
   req.Name, req.Detail, startMemberID, members, parameter)` (line 219).
3. `bin-api-manager/pkg/servicehandler/team.go:247` `TeamUpdate` —
   `a.IsDirect()` check, `teamGet` (for `CustomerID`), `hasPermission`,
   THEN `h.reqHandler.AIV1TeamUpdate(ctx, id, name, detail, startMemberID,
   members, parameter)` (line 281) — pure pass-through, no validation at
   this layer (confirmed reading the full function body).
4. `bin-common-handler/pkg/requesthandler/ai_teams.go:139` `AIV1TeamUpdate`
   — marshals into `amrequest.V1DataTeamsIDPut{Name, Detail,
   StartMemberID, Members, Parameter}` and RPCs to ai-manager.
5. `bin-ai-manager/pkg/listenhandler/v1_teams.go:161`
   `processV1TeamsIDPut` — reads `request.V1DataTeamsIDPut`, calls
   `h.teamHandler.Update(ctx, id, req.Name, req.Detail,
   req.StartMemberID, req.Members, req.Parameter)` (line 184-192), pure
   pass-through.
6. `bin-ai-manager/pkg/teamhandler/handler.go:156` `Update` — **fix site**.
   Currently: (a) calls `validateTeam(startMemberID, members)` (line 162,
   **unconditionally, using the raw incoming values**), (b) calls
   `h.validateNoInsightMembers(ctx, members)` (line 168), (c) builds an
   unconditional 5-key `fields := map[team.Field]any{...}` (lines
   172-178), (d) calls `h.db.TeamUpdate(ctx, id, fields)` (line 180,
   **already accepts a pre-built field map** — same shape as
   Phase 5b's `numberHandler`, NOT the campaigns/queues/providers/
   conferences shape where the dbhandler wrapper takes plain values).
7. `bin-ai-manager/pkg/dbhandler/team.go:196` `TeamUpdate(ctx, id,
   fields map[team.Field]any)` — **note: NO `len(fields) == 0` guard
   exists here** (confirmed via direct read, lines 195-225) unlike every
   other dbhandler `*Update(ctx, id, fields)` method in this migration
   (`CampaignUpdate`, `OutplanUpdate`, `NumberUpdate` all have one). It
   unconditionally sets `tm_update` and issues a SQL UPDATE even for an
   empty `fields` map (a harmless no-op UPDATE with only `tm_update`
   changing, but NOT a true no-op — it still bumps `tm_update` and would
   NOT match the "no DB write, no webhook" contract this migration
   establishes elsewhere). **This gap must be closed at the business-
   handler fix site (step 6) via a `len(fields) == 0` short-circuit to
   `Get`, exactly like every other phase — the dbhandler-level guard is a
   defense-in-depth nicety other phases added but is not itself required
   for correctness as long as the caller never invokes `TeamUpdate` with
   an empty map. This phase's design deliberately does NOT add the
   dbhandler-level guard as a mandatory fix (keeps the diff minimal,
   matches the "business handler owns the no-op decision" pattern used in
   numbers/Phase 5b) — call out this choice explicitly in the PR
   description so reviewers don't assume it was missed.**
8. CLI: no `team-control` or similar CLI tool exists under
   `bin-ai-manager/cmd` for teams (confirmed: `search_files` for `team`
   inside `bin-ai-manager/cmd` returned zero matches; `ai-control` exists
   but only has AI-resource subcommands, not team subcommands, confirmed
   by reading `cmd/ai-control/main.go`'s command tree) — **no CLI changes
   needed for this phase.**

## 3. The cross-field validation problem (this endpoint's key deviation from the established pattern)

Unlike every endpoint fixed in Phases 1-5, `teams.Update`'s `validateTeam`
function performs **cross-field validation between `start_member_id` and
`members`**: rule 2 requires `start_member_id` to reference an ID present
in the `members` array (validation.go line 40-42), and rules 3/4/5/7-11
validate `members`' internal structure. Converting both fields to pointers
naively and validating only the fields present in the request body would
be wrong in two ways:

- If a client omits `members` (wants to change only `name`/`detail`) but
  the validation only runs against the omitted-therefore-nil `members`,
  either the fix skips validation entirely (silently allows detaching
  `start_member_id`'s implicit invariant with the *existing* stored
  `members`, if `start_member_id` is simultaneously changed) or crashes on
  a nil-vs-populated mismatch.
- If a client sends `start_member_id` alone (e.g. "switch the team's entry
  point to a different existing member") without resending `members`,
  the validation must check the *new* `start_member_id` against the
  *currently stored* `members`, not an empty/nil slice.

**Design decision**: `teamHandler.Update`'s fix-site logic must, when
either `startMemberID` or `members` is non-nil (i.e. the caller is
changing at least one of the two coupled fields), fetch the current team
via `h.Get(ctx, id)` first, then construct the **effective merged
values** (`effectiveStartMemberID := startMemberID` if non-nil else
`current.StartMemberID`; `effectiveMembers := members` if non-nil else
`current.Members`) and run `validateTeam(effectiveStartMemberID,
effectiveMembers)` and `h.validateNoInsightMembers(ctx,
effectiveMembers)` against those merged values — never against a nil or
stale value. If **both** `startMemberID` and `members` are nil (caller is
only touching `name`/`detail`/`parameter`), skip `validateTeam`/
`validateNoInsightMembers` entirely (there is nothing to validate — the
roster is untouched) — this is consistent with every other phase's
"validation becomes conditional on non-nil" rule (roadmap §3 item 7) but
generalized to a *pair* of fields instead of one, because that pair has
an interdependency the others don't.

This means `teamHandler.Update`'s fix site needs one extra DB round-trip
(`Get`) compared to the mechanical pattern **whenever `start_member_id` or
`members` is being changed** (not on every call — a `name`-only update
still takes the cheap path of skipping straight to the conditional
field-map build with no extra `Get`, since `validateTeam` is not invoked
in that case). This extra `Get` is unavoidable given the cross-field
invariant and is not over-engineering — it is the correct generalization
of "validate against the true current+incoming merged state," the same
principle roadmap §3 item 7 already establishes for single-field
validation ("any field-specific validation... also becomes conditional on
non-nil"), just applied to a 2-field interdependent group.

**Note on where this Get sits in the overall call chain**: this is an
*additional* round trip within `bin-ai-manager`, on top of (not instead
of) the `teamGet` already performed one layer up, in
`bin-api-manager/pkg/servicehandler/team.go:270`, for permission/
`CustomerID` purposes (§2 step 3). By the time `teamHandler.Update` runs,
the resource's existence has already been confirmed once at the
api-manager layer — but `ai-manager` cannot rely on that cross-service
check (no shared cache/transaction guarantee between the two services),
so this second, ai-manager-local `Get` is a genuinely separate fetch, not
a redundant re-check of the same round trip. A request that changes
`start_member_id` or `members` therefore reads the team row from the DB
twice total across the full call chain (once at api-manager, once here)
even before the mandatory post-write re-fetch (`h.db.TeamGet` after a
successful `TeamUpdate`) — three reads plus one write in the worst case.
This is an accepted cost of correctness for this one cross-field-validated
endpoint, not introduced elsewhere in the migration.

**Note on stale-snapshot / concurrent-update behavior**: this
read-validate-write sequence has no transaction or optimistic-lock guard,
matching every other phase's dbhandler write pattern in this migration
(no phase introduces row-level locking) — two concurrent `PUT`s that each
set only `start_member_id` (to two different, individually-valid target
members) could both validate against the same stale `current.Members`
snapshot and both succeed, with the second write simply overwriting the
first (last-write-wins on `start_member_id`, not a torn/corrupted state).
This is a pre-existing, accepted architectural limitation shared by every
write path in this monorepo's RPC-hop design (no phase in this migration
adds concurrency control), not a new risk this phase introduces — flagged
here only because this is the first phase where the validate step reads
a *different* field (`members`) than the one being written in isolation
(`start_member_id`), making the staleness window conceptually visible in
a way single-field phases don't surface. No action item; documented as an
accepted limitation consistent with the rest of the platform.

**Note on a possible confusing 400 for a `start_member_id`-only update**:
if the team's currently-stored `members` is empty (e.g. a row that
predates rule 10, or was written by a lower-level path that bypasses
`validateTeam`), a client sending only `start_member_id` will have that
request rejected by `validateTeam`'s rule 10 ("members list must not be
empty") even though the client never touched `members` at all. This is
correct behavior (the merged state genuinely fails validation and must
not be persisted), but is worth flagging in the RST doc (§4 item 8) so a
caller debugging an unexpected 400 on a `start_member_id`-only PUT knows
to check the team's current `members` state, not just the request body
they sent.

Concretely, the fix-site logic becomes:

```go
func (h *teamHandler) Update(ctx context.Context, id uuid.UUID, name, detail *string, startMemberID *uuid.UUID, members *[]team.Member, parameter *map[string]any) (*team.Team, error) {
    fields := map[team.Field]any{}

    if startMemberID != nil || members != nil {
        // NOTE: h.Get already maps a dbhandler.ErrNotFound to a typed
        // *cerrors.VoipbinError (confirmed at handler.go:85-99). errors.Wrap
        // (github.com/pkg/errors) is transparent to errors.As's unwrap-chain
        // walk (its withMessage/withStack wrappers implement Unwrap()), so a
        // caller further up the stack doing errors.As(err, &voipbinErr) still
        // finds the typed NotFound error through this extra Wrap layer, and
        // the request correctly surfaces as 404, not a generic 500. This is
        // verified (not assumed) — see the dedicated
        // Test_Update_NotFoundDuringMergedValidation case in §4 item 9,
        // which asserts the HTTP-layer 404 end-to-end, not just that an
        // error is returned.
        current, err := h.Get(ctx, id)
        if err != nil {
            return nil, errors.Wrap(err, "could not get current team for validation")
        }
        effectiveStartMemberID := current.StartMemberID
        if startMemberID != nil {
            effectiveStartMemberID = *startMemberID
        }
        effectiveMembers := current.Members
        if members != nil {
            effectiveMembers = *members
        }
        if err := validateTeam(effectiveStartMemberID, effectiveMembers); err != nil {
            return nil, errors.Wrap(err, "validation failed")
        }
        if err := h.validateNoInsightMembers(ctx, effectiveMembers); err != nil {
            return nil, err
        }
        if startMemberID != nil {
            fields[team.FieldStartMemberID] = *startMemberID
        }
        if members != nil {
            fields[team.FieldMembers] = *members
        }
    }
    if name != nil {
        fields[team.FieldName] = *name
    }
    if detail != nil {
        fields[team.FieldDetail] = *detail
    }
    if parameter != nil {
        fields[team.FieldParameter] = *parameter
    }

    if len(fields) == 0 {
        return h.Get(ctx, id)
    }

    if err := h.db.TeamUpdate(ctx, id, fields); err != nil {
        return nil, errors.Wrapf(err, "could not update team")
    }
    res, err := h.db.TeamGet(ctx, id)
    if err != nil {
        return nil, errors.Wrapf(err, "could not get updated team")
    }
    h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, team.EventTypeUpdated, res)
    return res, nil
}
```

(Illustrative — exact variable naming/error wrapping to match this file's
existing conventions at implementation time; the load-bearing point is
the merged-effective-value validation, not this exact code shape.)

## 4. Fixes to apply

1. **OpenAPI** (`paths/teams/id.yaml`): remove `required:` block; update
   `name`/`detail`/`start_member_id`/`members` descriptions to state
   "Omit to leave unchanged." (`parameter` already has no `required`
   entry and needs no schema change, only its own description could
   optionally clarify the same contract for consistency — non-blocking.)
2. **`bin-api-manager/server/teams.go` `PutTeamsId`**: switch
   `req.StartMemberId` parsing from `uuid.FromStringOrNil` to
   `uuid.FromString` with a 400 (`INVALID_START_MEMBER_ID`) on parse
   error, mirroring the `pre_flow_id`/`post_flow_id`/`call_flow_id`/
   `message_flow_id` precedent from Phase 5 — **but only when
   `req.StartMemberId` is a non-nil pointer with a non-empty string**;
   if the field is omitted (nil/absent), skip parsing entirely and pass
   `nil` through. Pass `req.Name`/`req.Detail`/`req.Members`/
   `req.Parameter` straight through as their regenerated pointer types.
3. **`bin-api-manager/pkg/servicehandler/team.go` `TeamUpdate`**: accept
   pointers for `name`/`detail`/`startMemberID`/`members`/`parameter`,
   pass through unchanged (no validation at this layer today, none
   added).
4. **`bin-common-handler/pkg/requesthandler/ai_teams.go`
   `AIV1TeamUpdate`**: accept pointers, marshal into
   `V1DataTeamsIDPut` as pointers.
5. **`bin-ai-manager/pkg/listenhandler/models/request/teams.go`
   `V1DataTeamsIDPut`**: fields become pointers with `omitempty`.
6. **`bin-ai-manager/pkg/listenhandler/v1_teams.go`
   `processV1TeamsIDPut`**: pass pointers through unchanged.
7. **`bin-ai-manager/pkg/teamhandler/handler.go` `Update`** (fix site):
   per §3's merged-validation logic; no-op (`len(fields)==0`)
   short-circuits to `h.Get(ctx, id)` (reusing the existing `Get` at line
   85, which already maps `ErrNotFound` correctly — confirmed by reading
   it).
8. RST doc update (`bin-api-manager/docsdev/source/team_struct_team.rst`
   or equivalent, exact filename confirmed at implementation time) with
   an Implementation Hint describing both the general omit-semantics and
   the cross-field `start_member_id`/`members` merged-validation
   behavior specifically (this is the one place in the whole migration
   where a reader needs the extra behavioral note, since it's not just
   "omit to leave unchanged" but "omit one, the other is still validated
   against the merged state"). Explicitly document the possible-confusing-
   400 case from §3 ("a `start_member_id`-only PUT can still fail
   validation if the team's currently-stored `members` is empty or
   otherwise invalid — check the team's current state, not just your
   request body, when debugging a 400 here").
9. Tests per roadmap §3 item 11, PLUS §3's new cases specific to this
   endpoint's cross-field logic:
   - `start_member_id` omitted, `members` provided (validates against new
     members + *existing* start_member_id) — both a case where the
     existing start_member_id IS in the new members list (success) and
     where it is NOT (expect validation error).
   - `members` omitted, `start_member_id` provided — both existing-
     members-contains-it (success) and does-not-contain-it (error) cases.
   - both omitted (name/detail-only update) — assert `validateTeam`/
     `validateNoInsightMembers` are NOT called (e.g. via a test double or
     by asserting `h.Get` is called exactly once — the no-op-check `Get`
     — not twice, since the merged-validation `Get` should be skipped
     entirely in this case; if implemented as written in §3's pseudocode,
     a name-only update never triggers the extra `Get` at all since
     `startMemberID != nil || members != nil` is false).
   - all fields omitted, true no-op — asserts `TeamUpdate` (db) is
     `.Times(0)` and only `Get` is called once.
   - malformed `start_member_id` string via the HTTP layer → 400,
     asserting `servicehandler.TeamUpdate` is never reached (mirrors
     Phase 5's `Test_conferencesIDPUT_MalformedFlowID`/
     `Test_numbersIDPUT_MalformedFlowID` pattern).
   - **`Test_Update_NotFoundDuringMergedValidation`**: the merged-
     validation `Get` (triggered by a non-nil `startMemberID` or
     `members`) returns `dbhandler.ErrNotFound`; assert the error
     propagates as a typed not-found error through `errors.Wrap`'s
     unwrap-transparent chain, all the way to an HTTP 404 at the
     `bin-api-manager/server/teams_test.go` layer — this pins the
     currently-incidental-but-verified-correct behavior described in
     §3's inline code comment, rather than leaving it to work by luck of
     `pkg/errors`' `Unwrap()` semantics.
   - **`Test_Update_StartMemberIDOnly_EmptyStoredMembers`**: currently-
     stored `members` is empty, request supplies only `start_member_id` —
     assert the request fails with a validation error (rule 10), pinning
     the §3-documented "confusing 400" behavior as intentional, not a
     regression.
   - revert-and-rerun proof for at least the "members omitted, effective
     validation still runs against current members" case, since this is
     the phase's genuinely novel logic (not just a mechanical port of the
     established pattern) and the highest-value regression to guard.

## 5. Non-goals

- `validateTeam`/`validateNoInsightMembers`'s internal rules (1-11) are
  not modified — only how their inputs are derived (raw request values vs.
  merged-with-current values) changes.
- No change to `Create`'s validation call (`Create` already receives full,
  non-partial values by construction — a POST body has no "omitted means
  unchanged" concept, there is no "current" team yet).
- No dbhandler-level `len(fields) == 0` guard added to
  `bin-ai-manager/pkg/dbhandler/team.go`'s `TeamUpdate` — deliberately
  deferred per §2 step 7's reasoning (business-handler-owned no-op
  decision is sufficient and consistent with Phase 5b's numbers pattern);
  flagged as an optional low-priority follow-up, not scheduled.
- No CLI changes (no team CLI subcommand exists today, per §2 step 8).

## 6. Risk / testing notes

- This is the one phase-6 endpoint where an incorrect implementation could
  produce a **worse** bug than the one being fixed: naively skipping
  `validateTeam` whenever either field is nil (rather than merging with
  current state) would let a client detach `start_member_id` from the
  team's actual member roster silently, exactly the kind of state
  corruption this whole migration exists to prevent — this is why §3's
  merged-validation logic is treated as the core deliverable of this
  design doc, not a footnote.
- `parameter` (already-optional today) requires no behavior change; a
  client already relies on `nil`/absent `parameter` meaning "no custom
  parameters" or "leave unchanged" (whichever the current `if
  req.Parameter != nil` branch at `server/teams.go:215-217` implies) —
  confirm at implementation time which of these two the current behavior
  actually is (a `nil` parameter passed to `AIV1TeamUpdate` today already
  reaches `teamHandler.Update` as a `nil` map, which under the *current*
  unconditional-fields-map code sets `team.FieldParameter` to a literal
  `nil` value in the SQL update, i.e. **clears** it every time it's
  omitted, not "leaves unchanged" — this is itself a pre-existing bug of
  the exact class this migration fixes, and this phase's fix
  automatically resolves it as a side effect of making `parameter` a true
  `*map[string]any` pointer with nil-means-omitted semantics; call this
  out explicitly as a secondary bug fix in the PR description, since it's
  a real behavior change beyond what the OpenAPI `required` list alone
  would suggest — `parameter` was never in that list, only affected via
  the fields-map construction bug).

## 7. Round 1 review disposition

Independent review (`deleg_095290d1` task 2) verdict: REQUEST CHANGES.
Findings and fixes:

| # | Finding | Severity | Fix location |
|---|---|---|---|
| 1 | The doc never traced or verified how an `ErrNotFound` from the merged-validation `Get` surfaces to the caller (404 vs. generic 500); it happens to work today because `pkg/errors.Wrap` is transparent to `errors.As`'s unwrap-chain walk and `teamHandler.Get` already maps `ErrNotFound` to a typed `*VoipbinError`, but the doc asserted nothing about this | **MAJOR** | §3's pseudocode now carries an inline comment explaining and citing the verified unwrap-transparency mechanism; a dedicated test (`Test_Update_NotFoundDuringMergedValidation`, §4 item 9) pins the end-to-end 404 behavior rather than leaving it to incidental correctness |
| 2 | §4 item 9's test list omitted a test for the not-found-during-merged-validation-`Get` path | MINOR | Added `Test_Update_NotFoundDuringMergedValidation` to §4 item 9 |
| 3 | Doc didn't address the case where currently-stored `members` is empty/malformed, causing a `start_member_id`-only update to fail rule 10 in a way that may confuse the caller; RST note (§4 item 8) didn't mention it either | MINOR | §3 gained a dedicated "Note on a possible confusing 400" paragraph; §4 item 8's RST guidance now explicitly calls this out; §4 item 9 gained `Test_Update_StartMemberIDOnly_EmptyStoredMembers` to pin it as intentional |
| 4 | Doc didn't clarify that the extra `Get` in `teamHandler.Update` is additional to the already-existing `teamGet` at the api-manager servicehandler layer (line 270) — the team is fetched from DB at least twice per request when `start_member_id`/`members` change | MINOR | §3 gained a "Note on where this Get sits in the overall call chain" paragraph quantifying the total read/write cost (up to 3 reads + 1 write) |
| 5 | No mention of a TOCTOU race between the merged-validation `Get` and the later write (no transaction/optimistic lock) | MINOR | §3 gained a "Note on stale-snapshot / concurrent-update behavior" paragraph documenting this as an accepted, pre-existing, platform-wide limitation (not unique to or newly introduced by this phase) |

Not changed (reviewer confirmed correct as originally written): all §2
call-chain file paths/line numbers/function signatures; the
`dbhandler/team.go` `TeamUpdate` no-guard claim; `validateTeam`'s rules
and the `start_member_id`/`members` cross-field dependency
characterization; the core §3 design decision itself (merged-effective-
value validation via a conditionally-triggered extra `Get`) — confirmed
architecturally sound, not over-engineering; the absence of any team CLI
subcommand; the `parameter`-always-cleared-on-omission pre-existing bug
claim in §6; the illustrative code's consistency with the file's real
struct fields, interface signatures, and error-wrapping conventions.

