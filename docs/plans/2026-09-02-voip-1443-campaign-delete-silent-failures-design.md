# Issue Analysis (revised after round-7 review): campaignHandler.Delete() silent failures

## Revision history

- **Round 1** (REQUEST_CHANGES): original draft wrongly claimed `Delete()` never calls
  `FlowV1FlowDelete`. Corrected — it does, but swallows the error (log only).
- **Round 2** (REQUEST_CHANGES): factual core accepted, but flagged: (a) fix was under-scoped
  ("single-file, additive-only" was false — docs/dashboard/tests are also implicated by this
  repo's own conventions), (b) plain `Counter` under-analyzed vs. a labeled `CounterVec`,
  (c) overselling what a counter accomplishes (no alerting exists in this repo; the counter is a
  lower bound, not a full leak count), (d) a second, more severe silent-failure defect in the same
  function was missed, (e) the "no other resource shares this flow" claim was stated too strongly.
  All incorporated below. **User explicitly approved folding the second defect into this same
  ticket/PR** (same function, 2-line fix, avoids a needless second PR for one file).
- **Round 3** (REQUEST_CHANGES): the Defect 2 downstream wire-response claim ("HTTP 200 with body
  `null`") was technically wrong — traced and corrected to the actual observed body (a non-nil,
  all-zero-value campaign, due to `CampaignV1CampaignDelete` unmarshaling into a non-pointer struct
  value, which is a documented Go JSON no-op for `null`).
- **Round 4** (REQUEST_CHANGES): the "confirmed touch points" list was still incomplete — missed
  that Defect 2's fix makes a new HTTP 409 reachable, which per this repo's OpenAPI-First and
  RST-sync rules requires updating `bin-openapi-manager/openapi/paths/campaigns/id.yaml` and
  `bin-api-manager/docsdev/source/restful_api_errors.rst`. Also corrected two inaccurate "N lines
  away" rhetorical claims and an incomplete `execute.go` FlowID-reference enumeration.
- **Round 5** (REQUEST_CHANGES): the round-4 "corrected" `execute.go` enumeration was itself wrong
  (miscategorized a `Call`-creation call site as a campaigncall-creation site) and a cross-reference
  to a historical design-plan document misattributed a 409 note to the wrong endpoint. Both
  corrected.
- **Round 6** (REQUEST_CHANGES): found the RST-table citation in the round-4 fix was wrong (cited
  409 precedents that live in different sections of the file; the Campaign Reasons section this
  fix touches has zero existing 409 rows today), the `execute.go` safety argument leaned on an
  unsupported "conventions" appeal instead of the actual gating code (which itself has a
  documented error-swallow caveat), a stale revision-history section, and an off-by-two line range
  on the Defect 1 snippet. All corrected.
- **Round 7** (REQUEST_CHANGES): confirmed rounds 1-6's corrections all hold; found one remaining
  gap — the `execute.go` safety note's claim that `isStoppable()` is "the sole gate reachable via
  `campaignStopNow`" missed a fourth, unguarded call site (`status_run.go:66`, a force-stop path
  with no `isStoppable()` check at all). Disclosed below alongside the existing error-swallow
  caveat, with the same "should, not proof" framing, rather than asserting sole-gate reachability
  as fact.
- **Round 8** (APPROVE): first approval. Independently re-verified the round-7 fix, confirmed the
  `campaignStopNow` enumeration is now exhaustive (cross-checked via a second method — grepping
  every write of `StatusStop`, not just the two already-cited functions), and confirmed the aside
  is genuinely non-load-bearing for either defect's fix.
- **Round 9** (REQUEST_CHANGES — consecutive-approval count reset): found that Defect 1's
  `CounterVec{reason}` design, unlike Defect 2's fully-traced fix, asserted a reason-classification
  ("not-found vs. real error") by analogy to other services without tracing whether flow-manager's
  actual error path makes that split mechanically possible. Traced it: it does (see item 3 in the
  fix scope below) — `dbhandler.ErrNotFound` is a distinct, cross-service-traceable sentinel from
  the generic bare-400 fallback, all the way to the `campaign.go` call site. Added the concrete
  `reason` derivation rule so Defect 1 now meets the same "implementable without further research"
  bar Defect 2 already met.

## Symptom (unchanged — the original trigger)

Scheduled `api-validator` run (2026-09-02): 34 failed + 26 errors, concentrated in
campaign-dependent tests, all via `POST /v1.0/campaigns` → 400 with no diagnostic detail. Root
cause of *that* symptom: the shared test customer has hit `bin-flow-manager`'s hard
`maxFlowCount = 10000` per-customer cap (confirmed via live container log:
`Flow hard limit reached for customer. customer_id: 27690e2e-..., count: 10000, limit: 10000`).
**Fixing the existing 10,000-flow backlog for that customer is explicitly out of scope** (data
cleanup, needs separate authorization). This ticket addresses the code defects that make such a
leak both possible and invisible.

## Defect 1 (original scope): flow-delete failure is silently swallowed

`bin-campaign-manager/pkg/campaignhandler/campaign.go`, `Delete()`, lines 147-157 (verified
against current source):

```go
// delete flow
f, err := h.reqHandler.FlowV1FlowDelete(ctx, res.FlowID)
if err != nil {
    // we got an error here, but we've deleted the campaign already.
    // just write the log only.
    log.Errorf("Could not delete the flow. err: %v", err)
} else {
    log.WithField("flow", f).Debugf("Deleted campaign flow. flow_id: %s", f.ID)
}

return res, nil
```

The campaign delete correctly succeeds regardless (a user's delete call should not fail over an
internal cleanup step — that part of the design is sound and unchanged by this fix). But today,
if `FlowV1FlowDelete` fails, the only trace is one log line with no retention guarantee (this
investigation's own log fetches show campaign-manager's logs rotate within hours) and no metric.
Given `maxFlowCount` is a hard per-customer cap, an invisible, unbounded leak of exactly this
shape is a real defect on its own, independent of whether it is provably *the* historical cause of
the current 10,000/10,000 state (unverifiable from current log retention — not claimed as proven).

**Correction from round 2**: the flow IS referenced elsewhere — `campaigncall` records carry the
campaign's `FlowID` (`execute.go:235,314`, `models/campaigncall/webhook.go:56`) as children of the
campaign for dial execution, and `execute.go:344` uses it to create an activeflow. This does not
change the fix's safety: campaigncalls are children of the campaign (require `StatusStop` before
delete, same as the flow), not independent owners of the flow's lifecycle — but the earlier
"no other entity holds it" phrasing overstated the case and is corrected here.

## Defect 2 (added in this revision, per round-2 finding + explicit approval to include)

Same function, lines 128-131:

```go
if c.Status != campaign.StatusStop {
    log.Errorf("The campaign is not stop. status: %s", c.Status)
    return nil, err   // err is nil here — from the successful CampaignGet above
}
```

`err` is `nil` at this point (the preceding `CampaignGet` succeeded). This returns `(nil, nil)`.
Traced the full downstream path (corrected from an earlier, wrong "HTTP 200 with body `null`"
claim caught in review): `listenhandler/campaigns.go` marshals this `nil` campaign to the wire as
`"null"`, but `bin-common-handler/pkg/requesthandler/campaign_campaigns.go`'s
`CampaignV1CampaignDelete` unmarshals that response into a **non-pointer struct value**
(`var res cacampaign.Campaign`) and returns `&res` — `json.Unmarshal([]byte("null"), &res)` on an
already-allocated non-pointer destination is a documented Go no-op, so this produces a **non-nil
pointer to a zero-value `Campaign`**, not a nil pointer. That zero-value struct then flows through
`bin-api-manager`'s `ConvertWebhookMessage()` (no nil-guard needed since it isn't nil) and out to
the client as **HTTP 200 with a JSON body of an all-zero-value campaign** (`id:
"00000000-0000-0000-0000-000000000000"`, empty `status`, etc.) — not literal `null`, but the same
practical effect: a delete attempt on a running campaign gets an HTTP 200 that looks like success
and deletes nothing. This is the same silent-failure class as Defect 1, more severe (client
believes deletion succeeded), and ~16 lines above the code already being touched in the same
`Delete()` function (corrected from an earlier, inaccurate "three lines" claim).

**Fix**: return a typed error instead of `(nil, nil)`. This exact pattern already exists in the
same file, ~45 lines below — `Get()` (`campaign.go:174-178`) does
`cerrors.NotFound(commonoutline.ServiceNameCampaignManager, "CAMPAIGN_NOT_FOUND", ...).Wrap(err)`
on a lookup failure (corrected from an earlier, inaccurate "three lines away" claim). Copy that
shape (constructor confirmed to exist at
`bin-common-handler/models/errors/constructors.go:50-53`, signature
`FailedPrecondition(domain outline.ServiceName, reason, message string) *VoipbinError`), omitting
`.Wrap()` since there is no underlying `err` to wrap here:
`cerrors.FailedPrecondition(commonoutline.ServiceNameCampaignManager, "CAMPAIGN_NOT_STOPPED", "The campaign must be stopped before it can be deleted.")`.

**Resulting client-visible behavior change**: a typed `FailedPrecondition` error is routed through
`cerrors.ToResponse`/`cerrors.FromResponse` unchanged (verified: `bin-campaign-manager/pkg/listenhandler/main.go`'s
dispatcher already special-cases `*cerrors.VoipbinError` via `errorResponse`, and
`bin-api-manager/server/error_translate.go`'s typed-passthrough branch forwards it as-is — **no
`bin-api-manager` code change is required** for this half of the fix). `models/errors/rpc.go` maps
`StatusFailedPrecondition` to **HTTP 409**. So `DELETE /v1.0/campaigns/{id}` on a non-stopped
campaign changes from **200 + a garbled all-zero-value campaign body** to **409 + a proper error
envelope**. Any consumer currently treating that bogus 200 as success (this includes the
`api-validator` suite itself, out of this repo) will start seeing a 409 — expected and correct,
but worth stating explicitly since it's an observable contract change, not purely additive.

## Revised fix scope (per round-2 feedback)

This is **not** a single-file, additive-only change. Confirmed touch points:

1. **`bin-campaign-manager/pkg/campaignhandler/campaign.go`**:
   - Defect 1: increment a new metric, e.g. `promCampaignFlowDeleteFailedTotal`, on the
     `FlowV1FlowDelete` error branch, labeled by `reason` (see item 3 for exactly how `reason` is
     derived — traced through the actual RPC error path below, not asserted by analogy).
   - Defect 2: return a typed error instead of `(nil, nil)` when status check fails.
2. **`bin-campaign-manager/pkg/campaignhandler/main.go`**: register the new metric, following the
   existing `promCampaignCreateTotal` `init()` + `prometheus.MustRegister` pattern in this file.
3. **Metric shape — `CounterVec{reason}`, not a plain `Counter`** (round-2 correction), with the
   `reason` value concretely derivable from `FlowV1FlowDelete`'s actual error (round-9 correction
   — a prior revision asserted this classification by analogy without tracing whether flow-manager
   even distinguishes the two cases; it does):
   - `bin-flow-manager/pkg/flowhandler/db.go`'s `Delete()` returns `dbhandler.ErrNotFound`
     (unwrapped) when the flow is already gone (from `flowGetFromDB`, `dbhandler/flows.go:149`).
   - `bin-flow-manager/pkg/listenhandler/v1_flows.go`'s `v1FlowsIDDelete` propagates that error
     unchanged to the dispatcher.
   - `bin-flow-manager/pkg/listenhandler/main.go`'s `errorResponse()` special-cases
     `stderrors.Is(err, dbhandler.ErrNotFound)` → `simpleResponse(404)` (a bare-status response,
     distinct from the generic `simpleResponse(400)` "legacy" fallback used for everything else).
   - Bare status codes round-trip through `bin-common-handler/pkg/requesthandler`'s
     `HttpStatusErrorMap` as *distinct* sentinels — `ErrNotFound` (404) is a different `error` value
     from `ErrBadRequest` (400) (`bin-common-handler/pkg/requesthandler/common.go:20,24`).
   - **Therefore**, at the `campaign.go` call site, `FlowV1FlowDelete`'s error is mechanically
     classifiable: `stderrors.Is(err, requesthandler.ErrNotFound)` → `reason="not_found"` (benign,
     e.g. idempotent retry on an already-deleted flow); anything else (including
     `requesthandler.ErrBadRequest` and any typed `*cerrors.VoipbinError`) → `reason="error"` (a
     real failure — the actual leak-candidate bucket). This two-value label is what the fix should
     implement; it does not need, and this ticket does not propose, any change to flow-manager's
     error semantics to make the split possible — it already exists.
   - Precedent for the labeled-`CounterVec` shape itself: `bin-storage-manager/pkg/filehandler/signing.go`'s
     `promDownloadURIFailureTotal` (`reason` label); `bin-billing-manager/pkg/failedeventhandler`.
     **Note**: this monorepo has no existing idiom for "best-effort secondary cleanup on delete,
     instrumented" specifically — `bin-flow-manager`'s own analogous best-effort cleanup
     (`DirectV1DirectDelete` on flow delete, `db.go:255-260`) has no metric at all. This fix
     establishes that idiom rather than following one; using the labeled shape (now traced, not
     just precedented) is the correct choice given that ambiguity.
4. **`bin-campaign-manager/docs/operations.md`**: add the new counter to the existing "Prometheus
   Metrics" table (root CLAUDE.md's service-docs-sync rule; both `bin-storage-manager` and
   `bin-ai-manager` did this when adding failure counters — an established, followed convention,
   not optional).
5. **`monitoring/grafana/dashboards/campaign-manager.json`**: add a panel for the new metric.
   Precedent is mixed (ai-manager dashboarded 2 of 3 new counters; storage-manager dashboarded
   neither), so this is a "should" not a hard requirement — will add it since the metric's entire
   stated value is operator visibility, and a metric absent from the service's own dashboard
   doesn't deliver that.
6. **Tests**: `campaign_test.go`'s existing `Test_Delete` only covers the happy path
   (`FlowV1FlowDelete(...).Return(&fmflow.Flow{}, nil)`). Add: (a) a case where
   `FlowV1FlowDelete` returns an error — assert the campaign delete still succeeds (200/no error
   returned) AND the counter increments (pattern: `testutil.ToFloat64`, matching
   `bin-ai-manager/pkg/messagehandler/event_test.go`'s before/after delta assertion), (b) a case
   for Defect 2 — campaign not in `StatusStop` — assert a non-nil typed error is now returned
   instead of `(nil, nil)`.
7. **`bin-openapi-manager/openapi/paths/campaigns/id.yaml`**: the `delete:` operation currently
   declares only `200/400/401/403/404/500`. Defect 2's fix makes a `409` reachable and must be
   declared — `'409': $ref: '#/components/responses/Conflict'`, following the existing convention
   for other DELETE endpoints that can return a state-conflict (e.g. `openapi/paths/calls/id.yaml`).
   The `Conflict` response component already exists in `openapi.yaml`. Requires `go generate ./...`
   in `bin-openapi-manager` **then** `bin-api-manager` (OpenAPI-First rule), touching the checked-in
   `gens/` artifacts in both services.
8. **`bin-api-manager/docsdev/source/restful_api_errors.rst`**: the "Campaign Reasons" section
   (lines ~434-454) currently has exactly three rows — `CAMPAIGN_NOT_FOUND`, `CAMPAIGNCALL_NOT_FOUND`,
   `OUTPLAN_NOT_FOUND` — **all 404, zero existing 409s**. `CAMPAIGN_NOT_STOPPED` (409) would be the
   **first** 409 row in this section (`CALL_ALREADY_HANGUP` and `FLOW_STATE_INVALID` are 409
   precedents, but they live in the separate *Call Reasons* and *Flow Reasons* sections — useful
   only as a cross-section formatting template, not as same-table neighbors, corrected from an
   earlier draft that implied otherwise). This section's intro prose (line ~437) currently reads
   "Campaign state-transition operations are idempotent today — for example, stopping an
   already-stopped campaign returns success (no-op)." Explicit decision for the implementer: this
   sentence does **not** need to change — `DELETE` is not a state-transition endpoint, and stopping
   an already-stopped campaign still no-ops; the new 409 is additive to the section, not a
   contradiction of that sentence, but it should be double-checked at implementation time rather
   than assumed. This is a root-CLAUDE.md CRITICAL RST-sync path — requires the mandatory clean
   rebuild (`rm -rf build && python3 -m sphinx -M html source build`) and `git add -f
   docsdev/build/` after editing the `.rst` source.
   (Cross-reference, corrected: `docs/plans/2026-04-25-api-error-pr10-campaigns-outbound-plan.md`
   classifies `DeleteCampaignsId` under its generic 400/401/403/404/500 write-with-ID bucket and
   states elsewhere that PR10 added "no 409 anywhere." Its only forward-looking 409 note is scoped
   to state-transition endpoints like `PUT .../status`, not `DELETE` — an earlier draft of this
   document misattributed that note to DELETE specifically. This fix is a new addition prompted by
   the newly-identified Defect 2, not the fulfillment of an existing "may add" flag on DELETE. The
   OpenAPI-spec-and-RST-must-move-together point stands regardless of that correction.)

Note (corrected — an earlier revision of this enumeration was itself wrong): `execute.go` carries
the campaign's `FlowID` into **campaigncall** creation at exactly two sites — lines 235 and 314
(both pass `c.FlowID` into `h.campaigncallHandler.Create(...)`) — plus
`models/campaigncall/webhook.go:56`. Line 256 is a **different** resource: it passes `c.FlowID`
into `h.reqHandler.CallV1CallCreateWithID(...)`, creating a `Call` in `bin-call-manager`, not a
campaigncall. This means a live `Call` can independently carry a copy of the campaign's `FlowID`
on a separate lifecycle. Defect 1's safety conclusion still holds *in the normal case*, backed by
actual gating code rather than an appeal to convention: `Delete()` requires `campaign.StatusStop`
(`campaign.go:128`), and `isStoppable()` (`status_stop.go:129-159`) — reached via `campaignStopNow`
from `status_stop.go:38-40` and `eventhandle.go:18-23,41-46` — checks
`ListOngoingByCampaignID(...)` and refuses to allow the Stop transition while campaigncalls (and
by extension their associated live Calls) are ongoing. **Two caveats, disclosed rather than
hidden, neither of which blocks this fix**:
(a) that same function logs-and-falls-through on a lookup error
(`status_stop.go:149-152` — `if err != nil { log.Errorf(...) }` with no `return false`), so if
`ListOngoingByCampaignID` itself errors, `isStoppable()` returns `true` regardless of any actually-
ongoing campaigncalls/Calls; (b) `campaignStopNow` also has a fourth call site,
`status_run.go:66`, reached when `CampaignV1CampaignExecute` fails right after a campaign
transitions to `StatusRun` — this path force-stops with **no `isStoppable()` check at all**. In
practice this fires immediately after the run-transition, before the async dial-execution loop has
had a chance to create any campaigncalls, so the realistic window for a live Call to exist there is
expected to be empty — but that is an inference about timing, not a code-enforced guarantee, so it
is disclosed rather than assumed away. Neither caveat blocks this fix: Defect 1's change is
metric-only and does not alter flow-delete behavior; Defect 2's fix only changes the not-yet-
stopped rejection path, not any Stop-transition gate. The "no live Call should exist by Delete()
time" claim is a should, not a proof — stated as such throughout, not as a hard guarantee.

## Honest statement of what this fix does and does not accomplish (round-2 correction)

- Makes the *error-return* subset of flow-delete failures queryable via Prometheus and persistent
  beyond log retention (today: zero retention beyond a rotating log line).
- Does **not** catch every leak path: if the process crashes between `CampaignDelete` (line 134)
  and `FlowV1FlowDelete` (line 148), the flow leaks with no error return and no metric increment.
  The counter is a **lower bound** on leaked flows, not a complete count.
- This repo has **no alerting rules** anywhere (`monitoring/grafana/dashboards/` — zero
  `PrometheusRule` manifests, zero alert conditions across all dashboards). A dashboard panel
  alone means "an operator might notice," not an automated control against a slow, unbounded
  leak — which is exactly how the current incident reached 10,000 before being noticed at all.
  **Follow-up recommendation, filed as VOIP-1444 (not blocking this PR, per the "log issues,
  don't expand scope" convention)**: a periodic reconciliation check (or alert threshold) for
  per-customer flow count approaching `maxFlowCount` — tracked in that ticket, not implemented
  here.

## Why proceeding now is warranted

- Both defects are confirmed by direct, repeated source reads (this is the eighth revision across
  seven independent review rounds, each re-verifying against the live file, not carried forward on
  trust).
- Fix is still low-risk: additive metric + one corrected error return in an already-identified
  function, no change to campaign-manager's core delete success semantics for the already-stopped
  case, full verification workflow (`go mod tidy/vendor/generate/test/golangci-lint`) applies as
  normal for this service.
- Scope now matches what this repo's own conventions require (docs sync, dashboard, tests) rather
  than under-claiming "single file" as round 2 correctly caught.

## Design review addendum (post issue-analysis, design-phase round 1)

The issue-analysis above went through 11 fact-checking rounds. A separate design-phase review
(judging solution shape, not re-deriving facts) found the solution shapes themselves correct but
flagged four completeness gaps, addressed here. Round 2 of the design review then found the
round-1 fixes substantive but incomplete on two of the four (marked below); those are corrected
in this version.

1. **Actionability of the new metric — real but time-limited (round-2 correction).** A
   `CounterVec{reason}` tells an operator *that* and *roughly how* a flow leaked, and adding
   `flow_id` (`res.FlowID`) and `customer_id` (`res.CustomerID`) to the existing error log
   (`campaign.go:152`, `log.Errorf("Could not delete the flow. err: %v", err)` —
   `campaign_id` is already in that log's fields via the function-entry
   `logrus.WithFields`) makes the log line identify *which* flow leaked. **Round-2 correction**:
   this only helps within this service's hours-long log-retention window (stated earlier in this
   document, Symptom section) — it does not durably identify leaked flows after logs rotate,
   and this repo has no alerting to guarantee anyone reads the log within that window. The counter
   is the durable signal that *a* leak occurred; the log is a short-lived aid for identifying
   *which* flow, useful only if someone is actively watching when it happens. The actual durable
   identification path for flows that are only discovered later is the reconciliation follow-up
   (**VOIP-1444**, filed — see the "Filed follow-ups" list below).
2. **No denominator is the right call, but the metric needed zero-initialization to make that
   true (round-2 correction).** `campaign_manager_campaign_flow_delete_failed_total` has no
   matching "attempts" counter, so no failures-per-*attempt* ratio is computable (a `rate()` over
   time is still available from the counter alone). This is intentional: the decision variable
   here is "is this ever nonzero" (any `reason="error"` increment is a leak candidate), not "what
   fraction of deletes fail" — no scenario in this incident's history needed the ratio. **However**,
   a failure-only `CounterVec` that has never incremented is indistinguishable from a metric that
   was never registered or a service that isn't being scraped — both read as "No data" on the
   dashboard panel this fix adds (fix-scope item 5), which defeats that panel's stated purpose.
   **Added to fix scope**: pre-initialize both label series at registration time
   (`promCampaignFlowDeleteFailedTotal.WithLabelValues("not_found")` and
   `.WithLabelValues("error")`, immediately after `MustRegister` in `main.go`'s `init()`) so both
   read `0` from process start, not "No data." This is standard practice for failure-only
   `CounterVec`s, and it is what makes skipping a success counter *safe* — the actual reason to
   skip one remains what's stated above (no decision here depends on a ratio), not the weaker
   "scope inflation" argument, which is dropped.
3. **Consumer impact of the Defect 2 contract change — follow-up now filed, not just mentioned
   (round-2 correction).** `CampaignV1CampaignDelete` has exactly one production caller
   (`bin-api-manager/pkg/servicehandler/campaigns.go:190`, confirmed by reading the call graph —
   no internal RPC/subscribe cascade depends on this delete path), so the fix's blast radius is
   the single REST endpoint, not a wider internal ripple. The real first-party consumer of that
   endpoint is the **admin console** (`admin.voipbin.net`, separate repo, out of scope for this
   PR's code). Today, clicking Delete on a running campaign there silently "succeeds" per the API
   response and leaves the campaign in the list (the exact bug this fix removes); after this
   change the same click will receive a 409 — how the console currently renders a non-2xx response
   on this action is unverified from this repo and is exactly what the follow-up below should
   confirm, not assumed here. Decision: **no code change to the admin console is bundled into
   this PR** — the new error is strictly more correct than the old silent no-op, and no correct
   frontend behavior could have depended on the broken response. Filed as **VOIP-1445** (admin
   console: handle 409 CAMPAIGN_NOT_STOPPED on campaign delete UX) — not a blocking dependency of
   this PR, but a real ticket, not just a mention in this design doc.
4. **Rollout/rollback, including the known first-party consumer this ticket started from
   (round-2 addition).** No deploy-order coupling: `bin-api-manager` has no OpenAPI
   response-validating middleware, so the spec change (fix-scope item 7) is documentation-only at
   runtime — `bin-campaign-manager` and `bin-api-manager` can ship in either order, and the 409
   behavior is correct as soon as campaign-manager alone is deployed. Rollback granularity is the
   whole squashed PR (this repo's merge policy); reverting also reverts the metric, which is
   harmless since nothing depends on it. The two doc/dashboard-regeneration steps (OpenAPI
   `gens/` regen in two services, RST `docsdev/build/` force-add) are the most conflict-prone
   parts of this change against any concurrent PR touching the same spec or docs tree — do these
   last, immediately before PR creation, after the mandatory `git fetch origin main` +
   `merge-tree` conflict check, not mid-branch where they'd need redoing on rebase.
   **Post-deploy verification** (added to the plan's post-deploy verification task): (a) confirm both
   `promCampaignFlowDeleteFailedTotal` series read `0`, not "No data," proving the metric is
   actually registered and scraped; (b) `api-validator`'s own campaign-delete-on-non-stopped
   assertions (if any exist) will start observing 409 instead of the previous bogus 200 — this is
   the exact monitoring system whose failing run opened this ticket, so its results shifting after
   this deploy is expected, not a regression, and should be checked rather than rediscovered cold
   by whoever reads the next scheduled run's output. **Caveat on (b)**: this repo's shared test
   customer is currently pinned at the `maxFlowCount` cap (Symptom section above), so
   campaign-creation tests — and by extension any campaign-delete test that depends on first
   creating a campaign — may not even reach this code path until that separate, out-of-scope
   backlog is cleared. (a) and the 409-count trend below are the verifications that don't depend
   on that backlog being resolved. **Rollback trigger**: revert (not
   "forward-fix") if a first-party client is found depending on the old 200 response within the
   days after deploy (checked via the two verifications above and DELETE `/v1.0/campaigns/{id}`
   409-count trend); otherwise no action needed.

**Filed follow-ups** (per this repo's Jira-not-GitHub convention, not left as prose in this doc):
- **VOIP-1444** — bin-flow-manager: alert/reconcile per-customer flow count approaching
  `maxFlowCount`. The actual re-occurrence-prevention mechanism; this PR's metric is diagnostic,
  not preventive.
- **VOIP-1445** — admin console: handle 409 `CAMPAIGN_NOT_STOPPED` on campaign delete UX.
