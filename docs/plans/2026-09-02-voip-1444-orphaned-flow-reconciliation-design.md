# Issue Analysis and Design: VOIP-1444 — reconciliation job for orphaned campaign flows

(Title deliberately does not carry a round number — see the repeated staleness this caused,
flagged at design-review round 3. Round tracking lives entirely in "Revision history" below.
Design-review-round-9 addition: title updated from "Issue Analysis" alone — the document started
as issue-analysis-only through round 8, but rounds 3-8 of design review folded substantial design
content into the same file ("Proposed scope", "Correct placement", "Scheduling mechanism", etc.),
and the plan document consistently refers to this file as "the design doc".)

## Revision history

- **Round 1** (REQUEST_CHANGES): the proposed self-scheduling mechanism (a self-perpetuating
  delayed RPC chain, bootstrapped once per pod boot) was unsafe under this service's 2-replica,
  `restart: always` deployment.
- **Round 2** (REQUEST_CHANGES): found the original orphan-detection criterion wrong
  (`FlowV1FlowGet` succeeding does not mean the flow is live) and the "zero new DB code" claim
  false. Corrected the criterion to check `flow.TMDelete != nil`; added a new DB-layer query for
  the time-windowed scan.
- **Round 3 / Round 4** (REQUEST_CHANGES both times): spent trying to design an atomic fix to
  `bin-flow-manager`'s `Delete()` to close a 2-replica concurrent-race window created by the
  round-1 fallback mechanism (an in-process `time.Ticker` loop). Each attempt found that fix
  growing in scope (a shared write-path refactor, a return-contract change, a cache-only-flow edge
  case). Filed separately as **VOIP-1448**, scope narrowed to exclude it from this ticket, and the
  residual race disclosed as an accepted trade-off.
- **Round 5** (REQUEST_CHANGES): found that the accepted-trade-off framing was solving the wrong
  problem — the race exists **only because** of the round-1 ticker-loop mechanism choice, and this
  platform already has a purpose-built alternative that eliminates the race entirely:
  **`bin-schedule-manager`** (VOIP-1283), a platform-internal cron scheduler explicitly documented
  as existing to "absorb... future ticker migrations" via Redis-lock + DB-CAS at-most-once claim
  semantics, with an audit trail and an ops kill switch (`schedule-control schedule disable`).
  Using it instead of a bespoke ticker removes the entire "Concurrency: disclosed residual risk"
  section this document previously had to carry — there is no residual risk to disclose, because
  no two replicas can claim the same scheduled run. **This revision replaces the ticker-loop
  mechanism with a `bin-schedule-manager` schedule entry.** VOIP-1448 remains filed and still
  valid (the underlying `bin-flow-manager` `Delete()` non-idempotency is real and pre-existing,
  reachable by any two concurrent `DELETE /v1/flows/{id}` callers in general), but it is now fully
  orthogonal to this ticket rather than something this ticket's design leans on being "small
  enough to accept."
- **Round 6** (REQUEST_CHANGES): the proposed Alembic seed migration for the new schedule row
  omitted several columns the existing platform-job seed migration sets, most critically
  `timeout_ms` — that column has no DB default, and a schedule row seeded with it unset scans as
  Go zero value `0`, which produces an already-expired RPC context on every dispatch, forever
  (confirmed stronger than initially found: `bin-schedule-manager`'s own CRUD validation rejects
  `timeout_ms <= 0` at creation time, so a *seeded* row — this ticket's only path — is the one way
  to introduce this defect). Fixed by explicitly enumerating every column the model migration sets,
  with sizing guidance for `timeout_ms`/`retry_max` against a worst-case pass and a cross-check
  against the reaper's abandonment-deadline formula.
- **Round 7** (APPROVE, first of two required consecutive approvals): independently re-verified
  the round-6 fix and five other claims spread across the document; all held up. Only editorial
  gaps found — a stale title/changelog lag behind the body, the seed-migration column list not
  explicitly covering the cosmetic/loud-failure columns alongside the silent-failure one, and the
  closing open-questions list not yet listing `timeout_ms`/`retry_max` sizing as still-open.
  Swept in this revision.
- **Round 8** (APPROVE, second consecutive — issue-analysis phase closed): independently
  re-verified the round-7 fix; held up. One non-blocking footnote inaccuracy found (a precedent
  column mischaracterized as "non-nullable" when it is schema-nullable-but-always-populated in
  practice) — fixed in this revision. This closed the mandatory 2-consecutive-approval issue-
  analysis gate; the document then moved into the design/plan review stage below.
- **Design-review round 1** (REQUEST_CHANGES, on the paired design+plan documents): found the
  "two PRs" premise wrong (this is one monorepo, not two repos), missing rollout sequencing for
  the schedule-seeded-before-route-exists ordering hazard, a structural scan-starvation bug in the
  originally-proposed `ORDER BY tm_delete ASC LIMIT n` query, undefined success/failure semantics
  on `ReconcileOrphanedFlows`, a punted metric-reuse decision, and an under-sized test plan. All
  addressed in the plan (see the plan's own revision history for the full fix list).
- **Design-review round 2** (REQUEST_CHANGES): the round-1 `DESC`-order fix was justified with a
  mathematically false claim; and the proposed response shape violated the root CLAUDE.md's
  transport-DTO layering rule (a `response.*` DTO that was a field-for-field copy of the domain
  `ReconcileResult` struct). Both corrected in the plan.
- **Design-review round 3** (REQUEST_CHANGES): found six further defects introduced by or
  surviving the round-1/round-2 fixes, the most significant being that this service's RPC
  listenhandler builds requests with `context.Background()` (no deadline propagation), which
  falsified the "structurally impossible race" claim below unless `ReconcileOrphanedFlows` imposes
  its own internal pass-timeout; and that the saturation signal (`len(candidates) == scanLimit`)
  is a chronic false alarm across most of the plausible deletion-rate range, with a "shorten the
  interval" remedy that does not actually clear it. Also found: the "why no cursor" retry-property
  claim overstated (requires the stronger per-window inequality it explicitly disclaims elsewhere),
  a missing `docs/domain.md` update for the new `models/` file, an unowned saturation metric
  mentioned in one section but absent from the metrics task, and an arithmetic error in the
  **plan's** round-2 revision-history entry (corrected there, not in this document's body).
- **Design-review round 4** (REQUEST_CHANGES): traced round 3's own fix against actual
  `bin-schedule-manager` source and found it named the wrong mechanism — the reaper's abandonment
  deadline does not govern re-dispatch; `bin-schedule-manager`'s `tm_next_run` CAS, which advances
  at claim time (before the RPC is even sent), is what prevents a second **dispatch** of the same
  cron slot, independent of pass duration. The real overlap risk is a pass physically outliving the
  cron interval itself. Also found: the `RecentSaturated` signal (round 3) needed the schedule's
  cron interval, which `bin-campaign-manager` cannot read from `bin-schedule-manager`'s database —
  threaded through as an explicit request parameter instead of a compiled constant that would
  desynchronize the moment the interval changed; `RecentSaturated` was specified to be computed
  inside the RPC loop, so the self-imposed pass-timeout bail-out could suppress it exactly when the
  batch was genuinely saturated — moved to a dedicated pre-loop pass; a timed-out partial pass was
  invisible (recorded as a full `success` with no signal) — added a `Partial` result field; the
  typed-not-found branch counted neither `Skipped` nor `Failed`, breaking the counter invariant —
  now counted as `Skipped`; and one more unowned "log/metric" phrase (window-edge warning) survived
  the round-3 sweep of the same class of defect. All addressed in the plan; see the plan's own
  revision history for the full fix list.
- **Design-review round 5** (REQUEST_CHANGES): traced round 4's fix against
  `bin-schedule-manager`'s claim/dispatch source in full and found round 4 had, for the third
  consecutive round, named an incomplete safety-mechanism set — this time by affirmatively denying
  a real guard mattered rather than by omitting it. `ExecutionHasRunning` ("Forbid overlap") is a
  second, independent concurrency guard neither round 3 nor round 4 accounted for, and it makes
  `timeout_ms` genuinely load-bearing (round 4 called it merely "cosmetic, for audit accuracy"):
  the execution row stays `running` — and so blocks a new dispatch — only until
  `bin-schedule-manager`'s own `SendRequest` wait (bounded by `timeout_ms`) returns, not until this
  pass physically finishes; if `reconcilePassTimeout` could exceed `timeout_ms`, a manual execute
  fired in the resulting gap would genuinely race a still-running pass. Fixed: both
  `reconcilePassTimeout < timeout_ms` and `reconcilePassTimeout << cron interval` are now stated as
  independently load-bearing; the reaper-abandonment-deadline cross-check round 4 dropped as "an
  unfalsifiable tautology" was reinstated, correctly framed as bounding crash-recovery time (a real,
  separate property) rather than as the concurrency-prevention mechanism itself; `skipped_overlap`
  added as the documented observable symptom. Also found and fixed: the `RecentSaturated` "shorten
  the interval" remedy could silently violate the cron-interval guard, since `cron` is
  runtime-mutable while `reconcilePassTimeout` is a compiled constant with no automatic re-check —
  added an explicit two-step runbook requirement; a genuine self-contradiction between two sections
  about whether `cron`/`recent_interval_sec` can drift apart (both were right, about different
  times — seed-time co-location vs. post-seed independent editing — reworded to say so); stale
  `retry_max`-based reasoning in the "VOIP-1448's relevance" paragraph, left over from before
  `retry_max: 0` was settled; and a test for the pre-loop signal computation that used a
  0-duration timeout, testing nothing (it would fail the query itself before producing a
  `ReconcileResult`). All addressed in the plan; see the plan's own revision history for the full
  fix list.
- **Design-review round 6** (REQUEST_CHANGES): traced round 5's two-guard model against
  `bin-schedule-manager`'s actual source and found two further errors in material round 5 had just
  added. **`skipped_overlap` is not an execution status and no execution row exists when it
  fires** — the execution status enum is `running`/`success`/`failed`/`abandoned`; `skipped_overlap`
  is a dispatch-level metric result label emitted on a path that returns before any execution row
  is created, and it is the wrong signal besides — it only ever fires for the cadence-degradation
  case below, never for the actual concurrency-unsafe case, which produces an ordinary `success`
  row with no distinguishing signal. Fixed everywhere this appeared. **Guard 2 (the `tm_next_run`
  CAS) is not an independent concurrency guard, contrary to round 5's "both are real, independent
  concurrency-safety requirements"**: `ExecutionHasRunning` is checked before the CAS on every
  claim path, so as long as guard 1 holds, a cron claim mid-pass is short-circuited into an
  overlap skip before the CAS ever runs — no second execution is created either way. Guard 2's
  real role is cadence/liveness (prevents the schedule from permanently falling behind), not
  mutual exclusion, which rests on guard 1 alone. Relabeled in several places (design-review-round-7
  finding: not actually "throughout" — this document's own lead-in bullet and one other paragraph
  still carried the refuted round-5 framing after this revision; fully swept in round 7). Also
  fixed: the reap-
  deadline cross-check (task 8) had been reinstated correctly in round 5 but attributed the crash
  scenario to a `bin-campaign-manager` replica dying "mid-pass" — wrong; that path is already
  fully handled by `SendRequest`'s own `timeout_ms` bound and never reaches the reaper, which only
  matters for `bin-schedule-manager`'s own dispatch goroutine crashing (its "abandon-not-drain"
  case) — corrected the attribution; the mutual-exclusion closure claim didn't account for the two
  timeout clocks starting at different points (`timeout_ms` from send, `reconcilePassTimeout` from
  receive) — added an explicit note that the margin must cover message-delivery delay, not just
  processing-time variance; this document's own "Proposed scope" item 3 still named the superseded
  `a5e6f559299c` migration as primary precedent, contradicting its own earlier-corrected paragraph
  — synced to `0c037bf0a362`; a stale "set `retry_max` consistently with it" phrase, left over from
  before `retry_max: 0` was settled two sections earlier in the same document — removed; and the
  plan's Alembic-column acceptance criterion covered `timeout_ms > 0` but not the reap-deadline
  cross-check condition, which only its verify task covered — added to the acceptance criterion.
  All addressed in the plan; see the plan's own revision history for the full fix list.
- **Design-review round 7** (REQUEST_CHANGES): independently re-derived round 6's corrected mental
  model from `bin-schedule-manager` source and confirmed it holds — `ExecutionHasRunning` genuinely
  runs before the CAS on both the cron **and** manual claim paths (stronger than round 6 verified:
  `ScheduleClaimAndCreateExecution` skips the CAS *entirely* for manual triggers, so guard 2/the
  CAS does not exist at all on that path), and an execution row leaves `running` only via
  `ExecutionComplete` (success and timeout both funnel through the same call) or the reaper. But
  round 6's fix was applied incompletely: this document's own "What this buys over the ticker" lead
  bullet still opened with "provides two independent concurrency guards, not one... Both are real"
  — the exact framing round 6 refuted, with the correction buried 15 lines below where a skimming
  reader would miss it — rewritten so the lead itself states the corrected model. The "VOIP-1448's
  relevance" paragraph still said "the two concurrency guards above" — corrected to name guard 1
  and correct sizing only. The sentence defining the mutual-exclusion mechanism itself contained a
  factual error: "lasts exactly `timeout_ms`... regardless of how long the pass actually runs" is
  the precise inverse of the property being secured (`ExecutionComplete` fires as soon as
  `SendRequest` returns, whether early on success or at the `timeout_ms` ceiling on timeout, so the
  row's `running` duration is *at most* `timeout_ms`, tracking real pass duration up to that point)
  — corrected. Two new findings: the `skipped_overlap` metric is cron-path-only — a manual execute
  hitting the same overlap guard returns `EXECUTION_IN_PROGRESS` with no metric at all, which
  matters because task 10's own rollout smoke test is a manual execute and can legitimately hit
  this if a prior run is in flight — documented the expected behavior there; and the "`timeout_ms`
  unset silently produces `0`" claim (from the original issue-analysis round 6, carried through
  every revision since) doesn't account for `timeout_ms` being `NOT NULL` with no `DEFAULT` — under
  the strict `sql_mode` most modern deployments run, an `INSERT` omitting it is rejected loudly at
  migration-apply time; the silent-`0` outcome requires a relaxed `sql_mode` some legacy configs
  still use — hedged the claim without changing the underlying guidance (set every column
  explicitly regardless). All addressed in the plan; see the plan's own revision history for the
  full fix list.
- **Design-review round 8** (REQUEST_CHANGES): spot-checked the now-thrice-confirmed mutual-
  exclusion model (consistent everywhere it's stated) and did a fresh read of every section
  unrelated to `bin-schedule-manager` concurrency — this surfaced the round's one substantive
  defect. **`campaign_flow_reconcile_failed_total` was defined (this section, item 5) to cover
  both a `FlowV1FlowGet` error and a `FlowV1FlowDelete` error, but the plan only ever wired the
  metric increment into the `FlowV1FlowDelete` branch** — the `FlowV1FlowGet`-error branch
  incremented the in-memory `Failed` field only. This is the branch a `bin-flow-manager` outage or
  an open circuit breaker actually fails on, for every remaining candidate in a pass, so the metric
  would read zero during exactly the outage it exists to surface. Fixed: both branches now
  increment the one shared counter (still no `reason` label distinguishing them — documented
  explicitly rather than silently conflated). Also fixed: this document's own precedent-citation
  for the mutual-exclusion "at most `timeout_ms`" claim didn't note it's conditioned on
  `retry_max: 0` (added a qualification, since the general formula loops retries within one
  execution row); the plan's round-6 revision-history entry still used a since-abandoned "(1)/(2)"
  numbering with backwards "[now (1)]"-style annotations and two referent-less "constraint (1)"
  mentions, which round 7's "relabeled throughout" claim missed — corrected to (a)/(b)
  retroactively in that entry, and this document gained explicit (a)/(b) labels alongside "guard
  1"/"guard 2" so the round-7 claim that both documents use consistent labels is actually true; the
  plan's verification checklist named only `go test`/`golangci-lint`, a 2-of-5 subset of the root
  CLAUDE.md's mandatory workflow, and `go generate` specifically is load-bearing here (task 6's
  tests mock interfaces tasks 1/3 must also update, which `go generate` regenerates from) — fixed;
  and task 10's mandatory smoke-test step never said how to actually fire a manual execute (no
  `schedule-control execute` subcommand exists) — added the RPC-client path and the
  `schedule get`-first step to resolve the schedule's UUID. All addressed in the plan; see the
  plan's own revision history for the full fix list.
- **Design-review round 9** (REQUEST_CHANGES): confirmed the round-8 metric fix landed everywhere
  it should have in this document (pseudocode, Metrics section) but found the "Not-found detection
  for `FlowV1FlowGet`" section — separate, normative text an implementer reads to build the actual
  error branch — still said the non-not-found-error case is "logged and skipped", contradicting the
  settled `Skipped`-means-legitimately-clean semantics and omitting the metric increment; fixed.
  Also found: this document's own precedent-citation for the "at most `timeout_ms`" claim didn't
  carry the same (a)/(b) constraint labels the plan uses, so round 7's "standardized... in both
  documents" claim only partly held (round 8 already fixed most of this; round 9 confirmed no
  further gaps here); the plan's acceptance criteria named `schedule-control` as a manual-trigger
  path for the new route, contradicting task 10's own established fact that it has no `execute`
  subcommand; the plan's round-6 entry still had one referent-less "Constraint (1)"; the plan's
  acceptance criteria never pinned `campaign_flow_reconcile_failed_total` explicitly; and the
  plan's pre-implementation check never actually decided the `cron` interval it's cited elsewhere
  as producing. All fixed in the plan. Non-blocking: retitled this document "Issue Analysis and
  Design" to match how the plan refers to it, since design-review rounds folded substantial design
  content into what started as an issue-analysis-only file.
- **Design-review round 10** (APPROVE, first of two required consecutive approvals): a fresh,
  cold read of the plan's full task list, the acceptance-criteria/verify-task mapping, both
  revision histories, and the layering/DTO split, independently re-verified against source. No
  defect found. Three non-blocking suggestions were folded into the plan (package constants and
  the exact DB-query call declared self-containedly in task 3; the redeploy-vs-live-edit
  distinction between the two `RecentSaturated` remedies; and a note that
  `bin-schedule-manager/models/execution/execution.go`'s `Result` column durably persists
  `Saturated`/`RecentSaturated`/`Partial` beyond `bin-campaign-manager`'s own log retention,
  strengthening the no-third-counter decision's justification).

## Ticket validity check

**VOIP-1444** ("bin-campaign-manager: reconciliation job to clean up orphaned flows from deleted
campaigns") is a real, still-open follow-up filed during VOIP-1443. Scope was explicitly narrowed
by 대표님 on 2026-09-02: the per-customer Prometheus alert-rule half of the original proposal is
dropped; **only** the reconciliation job is in scope. `bin-flow-manager` remains untouched by this
ticket (VOIP-1448 covers that separately, orthogonally).

**Re-verified against current code**: the underlying defect this job compensates for is still
present. `bin-campaign-manager/pkg/campaignhandler/campaign.go`'s `Delete()` (post-VOIP-1443)
still has a best-effort `FlowV1FlowDelete` call that can fail and leave the flow orphaned —
VOIP-1443 made that failure *observable* (metric + log fields), not impossible. A reconciliation
job is the correct complementary fix.

## Proceeding-now judgment

- The direct trigger (the shared api-validator customer hitting `bin-flow-manager`'s
  `maxFlowCount=10000`) is still relevant: VOIP-1443 stops *new* leaks from being silent, but does
  nothing to prevent the *next* leak from quietly re-accumulating with no automatic cleanup.
- Contained to `bin-campaign-manager` (the reconciliation logic itself) plus one seeded row in
  `bin-schedule-manager`'s existing scheduling table — no new service, no new locking
  infrastructure, no `bin-flow-manager` change.
- Explicitly separate from, and does **not** retroactively clean up, the existing historical
  10,000-flow backlog for the shared test customer (out of scope, needs separate authorization per
  VOIP-1443's design doc) — this job only prevents *future* accumulation.

## Correct placement: bin-campaign-manager, not bin-flow-manager

`bin-flow-manager` has no way to know which of its flows are "owned" by a campaign, let alone
which owning campaign has since been deleted — flow ownership is one-directional
(`campaign.FlowID` points at the flow; nothing points back). `bin-campaign-manager` is the only
service that holds the fact "this flow belongs to campaign X, and campaign X is deleted." The
reconciliation logic lives in `bin-campaign-manager`, calling out to `bin-flow-manager` only via
the existing, unmodified `FlowV1FlowGet`/`FlowV1FlowDelete` RPCs. Jira summary already corrected
to "bin-campaign-manager:" to match.

## How an orphan is identified

- `campaign.go` `Delete()` soft-deletes: `bin-campaign-manager/pkg/dbhandler/campaign.go`'s
  `CampaignDelete()` (line 141) sets `tm_delete`, does not remove the row. The deleted campaign's
  `FlowID` remains readable indefinitely.
- **Criterion**: a flow is orphaned iff its owning campaign is deleted (`tm_delete` set) **and**
  the flow's own `TMDelete` is still `nil` — not merely "iff `FlowV1FlowGet` succeeds"
  (`bin-flow-manager`'s `FlowGet` query has no `tm_delete IS NULL` predicate, so it returns
  already-soft-deleted flows successfully too — confirmed independently: `flowHandler.Delete()`,
  `pkg/flowhandler/db.go:262-272`, calls `FlowGet` *after* soft-deleting specifically to return the
  now-deleted record, which would be impossible if `FlowGet` filtered deleted rows).
  `flow.TMDelete` (`bin-flow-manager/models/flow/flow.go:34`, json tag `tm_delete`, no
  `omitempty`) crosses the RPC boundary on every `Flow` response (`bin-flow-manager/pkg/listenhandler/v1_flows.go`'s
  `v1FlowsIDGet` marshals the domain `*flow.Flow` directly, no response-DTO stripping), so the
  reconciliation job can and must inspect it directly:
  (pseudocode below sketches the core branching only — see the plan's task 3 for the exact
  `ReconcileResult` bookkeeping, including the design-review-round-4 fix that counts a typed
  not-found as `Skipped`, not an uncounted no-op, and the design-review-round-8 fix below):
  ```go
  f, err := h.reqHandler.FlowV1FlowGet(ctx, campaign.FlowID)
  if err != nil {
      if isTypedNotFound(err) {
          result.Skipped++ // flow row genuinely doesn't exist -- a legitimately clean state
      } else {
          result.Failed++ // + campaign_flow_reconcile_failed_total (design-review-round-8 fix:
          // an earlier draft counted this only in-memory, leaving the metric silent on exactly
          // the systemic-outage case it exists to catch -- a bin-flow-manager outage or open
          // circuit breaker fails every remaining candidate on this branch)
      }
      continue
  }
  if f.TMDelete != nil {
      result.Skipped++ // already cleaned up (by an earlier pass, or a successful Delete() call) — not an orphan
      continue
  }
  // flow is live, owning campaign is deleted -> genuinely orphaned
  if _, err := h.reqHandler.FlowV1FlowDelete(ctx, campaign.FlowID); err != nil {
      result.Failed++ // log + campaign_flow_reconcile_failed_total (new, dedicated counter -- see "Metrics" below)
      continue
  }
  result.Cleaned++ // + campaign_flow_reconcile_cleaned_total
  ```
- Finding the *candidate* deleted campaigns uses `campaignHandler.List()`'s existing filter
  passthrough (`campaign.go:207`) with the `"deleted": true` special filter key
  (`bin-common-handler/pkg/databasehandler/main.go:76-85`, maps to `tm_delete IS NOT NULL`) for the
  boolean predicate — but see "Proposed scope" below for why the *time-windowed* version needs new
  DB-layer code (this generic filter mechanism has no range-predicate support).

## Not-found detection for `FlowV1FlowGet`

VOIP-1443's `reason` classification on `FlowV1FlowDelete` checks
`stderrors.Is(err, requesthandler.ErrNotFound)` because `bin-flow-manager`'s `Delete()` path
returns the flow-delete's `dbhandler.ErrNotFound` **unwrapped**, degrading to a bare 404 at the
dispatcher. `Get()` is different: `bin-flow-manager/pkg/flowhandler/db.go`'s `Get()` (lines 44-64)
explicitly wraps not-found in a **typed** error (`cerrors.NotFound(...).Wrap(err)`, lines 53-59).
So `FlowV1FlowGet`'s not-found case arrives as a typed `*cerrors.VoipbinError` with
`Status == StatusNotFound`. The reconciliation job's error-branch check must use
`stderrors.As(err, &ve) && ve.Status == cerrors.StatusNotFound`, not
`stderrors.Is(err, requesthandler.ErrNotFound)` (that sentinel is for the different, Delete-side,
untyped-error path). Any other error (RPC timeout, backend error) is counted as **`Failed`**, not
`Skipped` (design-review-round-9 fix: an earlier draft of this sentence said "logged and skipped",
which reads as the `Skipped` counter — the settled semantics, per "Metrics" above and the plan's
task 3, reserve `Skipped` for legitimately-clean end states like a typed not-found; a non-not-found
error is a failure, counted in `campaign_flow_reconcile_failed_total` (see "Metrics" above), not
silently treated as "clean").

## Scheduling mechanism: `bin-schedule-manager`, not a bespoke ticker loop (round 5 replacement)

**Rejected**: an in-process `time.Ticker` loop inside `bin-campaign-manager` (rounds 1-4's
approach). It works, but it re-invents scheduling infrastructure this platform already has, and it
is what forced rounds 3-4 into designing (and eventually deferring) a `bin-flow-manager`
concurrency fix that this job wouldn't need at all under a claim-based scheduler.

**Adopted**: `bin-schedule-manager` (VOIP-1283) — this platform's existing DB-driven dispatch
engine, whose own documentation states its charter as absorbing "future ticker migrations" into
one engine with Redis-lock + DB-CAS at-most-once claim semantics
(`bin-schedule-manager/CLAUDE.md`). It already runs in production (`replicas: 2`, same as
`bin-campaign-manager`) specifically *because* "Redis lock + DB CAS claim make concurrent replicas
safe." Concretely:

1. A new `schedule_schedules` row (seeded via a new Alembic migration in `bin-dbscheme-manager` —
   a directory in this same monorepo, not a separate repository; see the implementation plan for
   why this lands in the same PR as the `bin-campaign-manager` changes, not a separate one).
   Design-round-1 correction: the closer precedent is
   `bin-dbscheme-manager/bin-manager/main/versions/0c037bf0a362_schedule_schedules_seed_phase2_ticker_...py`,
   which already seeds five schedules dispatching into *other* services' request queues —
   directly answering "is cross-service dispatch via `bin-schedule-manager` sanctioned?" — rather
   than `a5e6f559299c`'s self-RPC housekeeping jobs, which this analysis originally (and still
   correctly, just less precisely) cited. Column set:
   `name: campaign-flow-reconcile`, `type: rpc`, `cron: <interval, e.g. "0 */6 * * *">`,
   `target_queue: bin-manager.campaign-manager.request` (already in the allowlist —
   `bin-common-handler/models/outline/queuename.go:53,206`), `target_uri:
   /v1/campaigns/flows/reconcile`, `target_method: POST`, nil-customer (platform job), plus every
   other column both precedent migrations set (see the implementation plan for the full list,
   including the `enabled: 0`-at-seed rollout decision).
2. A new `bin-campaign-manager` listenhandler route (`POST /v1/campaigns/flows/reconcile`,
   internal-only — not exposed through `bin-api-manager`), accepting a small request body
   (`recent_interval_sec` — see the plan's `RecentSaturated` design and design-review-round-4 for
   why this must be an explicit parameter, not something the handler infers) and calling
   `campaignHandler.ReconcileOrphanedFlows(ctx, recentIntervalSec)`, returning its result,
   following the same RPC-route shape every other internal `/v1/*` endpoint in this service
   already uses.

What this buys over the ticker, concretely:
- **`bin-schedule-manager` has two dispatch guards, but only one of them provides mutual exclusion
  (design-review-round-7 correction of a design-review-round-5 over-correction, itself correcting a
  design-review-round-4 under-correction)** — the two are not co-equal "independent concurrency
  guards"; one (`ExecutionHasRunning`) is the entire mutual-exclusion mechanism, the other (the
  `tm_next_run` CAS) is a cadence/liveness mechanism that never gets a chance to matter for safety
  because the first guard already short-circuits it:
  1. **"Forbid overlap"** (`ExecutionHasRunning`) — **the actual mutual-exclusion guarantee**: the
     dispatcher refuses to claim/dispatch a schedule while a `running` execution row already
     exists for it (on the **cron** path this produces a `skipped_overlap` **dispatch metric**
     result — not an execution status, since no execution row is ever created for a skipped
     attempt; the **manual** `/v1/schedules/{id}/execute` path hits the same guard but returns a
     `FailedPrecondition` `EXECUTION_IN_PROGRESS` error instead, with no metric at all — see the
     plan's task 10 for the operational implication). That row is marked `running` until
     `bin-schedule-manager`'s own `SendRequest` wait — bounded by the schedule's `timeout_ms` —
     returns, **not** until the pass actually finishes
     inside `bin-campaign-manager`. This guard is only as good as `timeout_ms` tracking real pass
     duration: if `ReconcileOrphanedFlows`'s own internal timeout could exceed `timeout_ms`, there
     would be a real window where `bin-schedule-manager` believes the pass is done (guard no longer
     blocking) while it is still physically executing — a manual `/v1/schedules/{id}/execute`
     fired in that window would genuinely race the still-running pass.
  2. **The `tm_next_run` CAS**: the claim transaction (`ScheduleClaimAndCreateExecution`) advances
     `tm_next_run` to the next cron boundary *before* the RPC to `bin-campaign-manager` is even
     sent, so the same schedule cannot be re-claimed for a new **cron** dispatch until that
     boundary.
  **Design-review-round-6 correction to a design-review-round-5 over-correction**: guard 1 is
  checked *before* the `tm_next_run` CAS on every claim path (cron and manual alike). This means
  guard 2 does **not** independently prevent concurrent execution — given guard 1 holds (the
  execution row stays `running` for the pass's entire physical duration), a cron claim arriving
  mid-pass is short-circuited into a dispatch-level overlap skip before the CAS ever runs; no
  second execution is ever created either way. What a chronically-overrunning pass produces via
  guard 2's absence is **cadence degradation** (the schedule permanently falls behind, observing
  overlap skips forever), not a race. Round 5 called both guards "independent concurrency-safety
  requirements" — guard 2 is real and required, but as a cadence/liveness property, not a mutual-
  exclusion one. Round 4 correctly identified that the schedule-manager reaper's abandonment
  deadline does not itself govern re-dispatch (it only affects whether a stuck row eventually reads
  `abandoned` instead of `running` forever, freeing the schedule for guard-1 purposes) — but round
  4 then over-corrected by treating `timeout_ms` as irrelevant to concurrency, when in fact
  `timeout_ms` is precisely what determines how long guard 1 stays effective. `bin-campaign-manager`'s
  RPC listenhandler builds every request's context with `context.Background()`
  (`pkg/listenhandler/main.go`) — it does **not** propagate the caller's `timeout_ms` as a context
  deadline, so nothing bounds a pass's duration at the RPC layer by itself. **Mutual exclusion
  rests on guard 1 alone, and requires `ReconcileOrphanedFlows`'s self-imposed
  `context.WithTimeout` to stay strictly below `timeout_ms`, with margin covering real
  message-delivery delay (the two timeouts' clocks start at different points) — not just
  processing-time variance.** This constraint has no runtime monitor: a violation produces an
  ordinary `success` row, indistinguishable from a correctly-serialized one. Separately,
  `ReconcileOrphanedFlows`'s timeout must also stay well below the schedule's cron interval, to
  avoid the cadence-degradation failure mode above — see the plan's task 3 and task 8's corrected
  (three-part) cross-check for the concrete sizing, including the reap-deadline bound on how long
  a *`bin-schedule-manager` dispatch goroutine itself* crashing (its "abandon-not-drain" case, not
  a `bin-campaign-manager` crash) can leave the schedule stuck under guard 1. No further
  concurrency section is needed in this design *given the mutual-exclusion bound is implemented*,
  not on dispatch semantics alone.
- **A durable audit trail** — every reconciliation pass gets an `execution` row in
  `bin-schedule-manager`'s own tables, addressing a real gap this session already knows about
  (`bin-campaign-manager`'s own logs "rotate within hours," per VOIP-1443's investigation).
- **An ops kill switch without a redeploy** — `schedule-control schedule disable
  campaign-flow-reconcile` — meaningful for a job that scans and mutates data across all
  customers.
- **The reconciliation interval becomes data** (the schedule's `cron` field), not a compiled Go
  constant — the *mechanism* for changing it later (edit a DB row, no redeploy) is settled by this
  choice; the *concrete cron value* to seed initially is still an open question, listed below.

Cost is comparable to or lower than the ticker approach: one Alembic seed row plus one
listenhandler route, versus a `Run(ctx)` goroutine, `main.go` wiring, and (per rounds 1-4) an
entire concurrency-safety writeup. The one trade-off, already documented and accepted for
`bin-schedule-manager`'s own housekeeping jobs, is that each pass occupies one of
`bin-campaign-manager`'s listenhandler workers for its duration
(`bin-schedule-manager/docs/architecture.md` documents and accepts the identical trade-off for its
own self-RPC jobs) — negligible for an infrequent (hourly-or-slower) background job.

**VOIP-1448's relevance is now fully orthogonal, not load-bearing for this ticket.** With
claim-based scheduling and correctly-sized `reconcilePassTimeout` (design-review-round-7 fix: not
"the two concurrency guards above" — mutual exclusion rests on guard 1 alone, per the correction
above; per the plan's task 3), this job cannot race itself across replicas. **Design-review-round-5 correction**: an earlier draft of this paragraph
reasoned from "the schedule's own `retry_max`/timeout settings could still in principle overlap
with a next scheduled run" — stale, since `retry_max: 0` (no schedule-level retry at all) has been
settled since round 6 and is restated in the plan's task 8; there is no schedule-level retry path
left to overlap with anything. VOIP-1448 (making `bin-flow-manager`'s `Delete()` idempotent)
remains a real, worthwhile fix for the general case (any two concurrent `DELETE /v1/flows/{id}`
callers system-wide, unrelated to this job's own concurrency model) but is no longer something
this ticket's correctness depends on.

## Proposed scope

1. **`bin-campaign-manager/pkg/campaignhandler`** (new file, e.g. `reconcile.go`): a
   `ReconcileOrphanedFlows(ctx context.Context, recentIntervalSec int64) (result
   campaign.ReconcileResult, err error)` method implementing the detection logic above, with a
   bounded per-pass scan limit (see point 3). `recentIntervalSec` is a design-review-round-4
   addition, threaded in from the request body — see "the actual rate-risk signal" below for why
   the handler cannot infer this itself. `ReconcileResult` is a small domain struct (`Cleaned`,
   `Skipped`, `Failed` counts, a `Saturated` flag, a design-review-round-3 addition covering the
   actual rate-risk signal (`RecentSaturated`), and a design-review-round-4 addition (`Partial`,
   set when the self-imposed pass timeout cuts a pass short) — see the plan's "Tasks —
   bin-campaign-manager" section for the exact fields and the corrected success/failure semantics;
   an earlier draft of this document under-specified this as a bare `cleaned int`). The method must
   bound its own execution with an internal `context.WithTimeout`, sized well below the schedule's
   *cron interval* — see "self-imposed pass timeout" below; it cannot rely on the caller's RPC
   context carrying a deadline.
2. **New DB-layer query**: the generic `ApplyFields` filter mechanism
   (`bin-common-handler/pkg/databasehandler/main.go:61-110`) supports only equality plus the
   special boolean `"deleted"` key — no range predicates, so "deleted within the last N days"
   (`tm_delete >= cutoff`) cannot be expressed through it. Separately, `CampaignList`'s existing
   time-token pagination is on `tm_create` (`dbhandler/campaign.go:210-211`), the wrong axis for a
   delete-time window. **New, small, single-consumer method** in
   `bin-campaign-manager/pkg/dbhandler`, e.g.
   `CampaignListDeletedSince(ctx, since time.Time, limit uint64) ([]*campaign.Campaign, error)`, a
   direct squirrel query (`WHERE tm_delete >= ? ORDER BY tm_delete DESC LIMIT ?` — `DESC`, not
   `ASC`, per the plan's design-round-1 fix: `ASC` with a fixed `LIMIT` and no cursor is a
   structural scan-starvation bug once in-window candidates exceed the limit, since it returns the
   same oldest rows forever; `DESC` guarantees every newly-deleted campaign is examined within
   `scanLimit`/rate passes of its own deletion — see the plan's "Scan order and coverage" section
   for the full analysis, including the design-review-round-3 correction to the safety condition
   and the saturation-signal definition. The comparison against the nullable `tm_delete` column
   naturally excludes live campaigns via standard SQL NULL-comparison semantics, no separate
   `IS NOT NULL` needed) — self-contained in this service's
   own dbhandler (no `bin-common-handler` change; a single-consumer query correctly stays local per
   this repo's `bin-common-handler` admission rule). Range-predicate precedent for this shape
   exists elsewhere in the monorepo (`bin-call-manager/pkg/dbhandler/channel.go`'s
   `ChannelGetsForRecovery` uses `squirrel.Gt`/`Lt` on a `*time.Time` column that, while nullable
   in schema, is always populated by `ChannelCreate` in practice — this query's reliance on
   NULL-comparison semantics for a column that is *genuinely, routinely* nullable (`tm_delete`,
   unset for every live campaign) is a step further than that precedent, though the same
   underlying squirrel/SQL mechanics apply).
   **Index note**: `campaign_campaigns` currently has indexes on `customer_id`, `flow_id`,
   `outplan_id`, `outdial_id`, `queue_id` — none on `tm_delete`
   (`bin-campaign-manager/scripts/database_scripts/table_campaigns.sql`). Without one, this query
   full-scans the table on every scheduled run. Whether that's acceptable depends on the row-count
   data point below; if not, an Alembic migration adding an index on `tm_delete` should be included
   in this PR's scope — and per `bin-dbscheme-manager/CLAUDE.md`, must also update the test schema
   fixture (`bin-campaign-manager/scripts/database_scripts/table_campaigns.sql`) alongside the
   migration, not just the migration itself.
3. **Scheduling** (replaces the rejected ticker-loop design — see above):
   - New Alembic seed migration in `bin-dbscheme-manager` adding the `campaign-flow-reconcile`
     schedule row. **Primary model: `0c037bf0a362_schedule_schedules_seed_phase2_ticker_...py`**
     (design-review-round-6 fix: an earlier draft of this paragraph named
     `a5e6f559299c_..._seed_platform_jobs.py` here, contradicting this document's own earlier,
     already-corrected "closer precedent" paragraph above — `0c037bf0a362` is the right reference,
     since it seeds cross-service dispatches like this one rather than self-RPC housekeeping jobs;
     the two migrations happen to set an identical column list, so this was a citation
     inconsistency, not a substantive gap). **Must set every column both precedent migrations
     set**, not just the ones needed to identify the schedule — `name`, `detail`, `type: 'rpc'`, `cron`,
     `target_queue`, `target_uri`, `target_method`, `target_data_type: 'application/json'`,
     **`target_data: JSON_OBJECT('recent_interval_sec', N)` (design-review-round-4 correction: not
     an empty object — the route does take an input, the cron interval in seconds, since
     `bin-campaign-manager` has no way to read the schedule's own `cron` field; see the plan's
     "Scan order and coverage" section. Design-review-round-5 clarification: this seed-time
     co-location keeps the *initial* values in sync, but a later, post-seed edit to `cron` via
     `schedule-control`/`PUT /v1/schedules/{id}` does not automatically update `target_data` — see
     the plan's runbook note for the operational step this requires)**, **`enabled: 0`**
     (design-round-1 correction: not `1` — see the plan's rollout-sequencing task for why the
     schedule must ship disabled until the route is deployed), and critically `timeout_ms`: the
     column is declared `INT NOT NULL` with **no DEFAULT** (design-review-round-7 correction to
     the original round-6 finding: the failure mode if a migration omits it depends on the
     database's `sql_mode` — under the strict mode most modern deployments run, MariaDB rejects the
     `INSERT` outright at migration-apply time (Error 1364), a **loud**, immediate failure, not a
     silent one; under a relaxed `sql_mode` some legacy configs still use, the omission could
     instead silently insert `0`, which `requesthandler.SendRequest` would turn into an
     already-expired `context.WithTimeout` on every dispatch, forever, making the schedule silently
     non-functional from the moment it's seeded). Either way the fix is identical — set every
     column explicitly, never rely on a database default for a column this consequential — but the
     column should not be filed under "guaranteed silent failure" in the taxonomy below; its actual
     failure mode is environment-dependent. Size `timeout_ms` comfortably above
     `ReconcileOrphanedFlows`'s own self-imposed
     `reconcilePassTimeout` (design-review-round-4: this, not a from-scratch RPC-latency estimate,
     is the actual worst-case pass duration once the handler self-bounds it — and, design-review-
     round-5: `reconcilePassTimeout` being strictly below `timeout_ms` is not just for a tidy
     estimate, it's what keeps `bin-schedule-manager`'s own "Forbid overlap" guard effective for
     the pass's entire physical duration — see "What this buys over the ticker" above; margin must
     cover real message-delivery delay, not just processing-time variance), `retry_max: 0` — this
     column is not left open here (design-review-round-6 fix: an earlier draft still said "set
     `retry_max` consistently with it" at this point, stale — `retry_max: 0` is settled two
     sections earlier in this same document, and is the value the reap-deadline formula below
     assumes). **Design-review-round-5 correction to a round-4 over-correction, itself corrected
     design-review-round-6**: an earlier draft of this paragraph dropped the cross-check against
     `bin-schedule-manager`'s reaper deadline formula entirely, calling it "an unfalsifiable
     tautology" — reinstated: with `retry_max: 0` the reap deadline is `timeout_ms + 60s`, a real
     and useful bound on how long a **`bin-schedule-manager` dispatch goroutine itself crashing**
     (its documented "abandon-not-drain" shutdown case — not a `bin-campaign-manager` crash, which
     round 5 wrongly named here and which is already handled by `SendRequest`'s own `timeout_ms`
     bound without ever reaching the reaper) can leave this schedule's stuck `running` row
     suppressing it (via the "Forbid overlap" guard) before the reaper frees it — genuinely
     distinct from, and in addition to, the constraints above. The full set of constraints is now:
     `reconcilePassTimeout` strictly below `timeout_ms` (the actual mutual-exclusion guarantee —
     see "What this buys over the ticker" above for why guard 2/the `tm_next_run` CAS is a cadence
     requirement, not an independent concurrency one), `reconcilePassTimeout` well below the
     schedule's **cron interval** (cadence/liveness), and `timeout_ms + 60s` comfortably below the
     cron interval (bounds crash-recovery time for a `bin-schedule-manager`-side crash). (The
     remaining columns the model migration sets — `id`, `customer_id`
     (nil-UUID, platform-job convention), `tm_next_run: NULL`, `tm_create`/`tm_update:
     UTC_TIMESTAMP(6)` — are cosmetic or loud-failure if gotten wrong (design-review-round-7
     correction: not necessarily contrasted with `timeout_ms` as "silent" — see above,
     `timeout_ms`'s own failure mode depends on `sql_mode` and can also be loud), but should still
     be set for consistency with the precedent rows rather than left to be discovered ad hoc during
     implementation.)
   - New `bin-campaign-manager` listenhandler route `POST /v1/campaigns/flows/reconcile` calling
     `ReconcileOrphanedFlows`.
   - `ReconcileOrphanedFlows` itself queries `CampaignListDeletedSince(ctx, now - window,
     scanLimit)` — bounded both by a time window (e.g. last 7 days — see sizing note below) and an
     explicit per-pass row limit (e.g. 500) so one scheduled run can never issue an unbounded
     number of RPCs.
4. **Bounded, time-windowed, and per-pass-limited scope, not a full historical backfill** — the
   load-bearing scope decision:
   - The entire *historical* backlog beyond the window is explicitly out of scope (separate,
     authorization-gated data cleanup, per VOIP-1443's design doc).
   - **Window-sizing — flagged as needing two concrete data points before finalizing, not resolved
     here**: this analysis could not obtain a live count of `campaign_campaigns` rows with
     `tm_delete IS NOT NULL`, nor the deletion rate, from this session (no production DB query
     access available). Check both once before fixing the window/scanLimit constants and the index
     question above — the safety condition (see the plan's "Scan order and coverage" section, as
     corrected in design-review-round-3) depends on the **peak** deletions in any single cron
     interval, not the daily average; the known trigger customer is an automated api-validator
     loop, i.e. bursty, so a daily average alone could understate the peak.
   - **Window-edge risk, disclosed**: a leak caused by a `bin-flow-manager` outage lasting longer
     than the chosen window would age out of the scan and silently merge into the out-of-scope
     historical backlog. Mitigation to implement regardless of window size: **log** (design-review-
     round-4 fix: no counter is defined for this, matching the no-third-counter decision for
     `Saturated`/`RecentSaturated`/`Partial`) a warning when a found orphan's owning campaign's
     `tm_delete` is close to the window cutoff.
   - **Scope note**: this job scans and mutates data across **all customers** — a platform-internal
     hygiene job, not scoped to one customer. `bin-schedule-manager`'s audit trail and kill switch
     (see above) are a meaningful mitigation for the operational risk of an all-customer background
     mutator, beyond what the rejected ticker-loop design would have had. Design-review-round-8
     note (non-blocking): a saturated pass issues up to `scanLimit` sequential
     `FlowV1FlowGet`/`FlowV1FlowDelete` RPC pairs against `bin-manager.flow-manager.request`,
     sharing `bin-campaign-manager`'s process-wide circuit breaker with the live campaign
     create/update/delete paths — acceptable at an hourly-or-slower cadence, but worth keeping in
     mind if `scanLimit` is later raised substantially.
5. **Metrics** (resolved in plan, design round 2): two new, dedicated plain `Counter`s —
   `campaign_flow_reconcile_cleaned_total` (incremented per genuinely-orphaned flow cleaned) and
   `campaign_flow_reconcile_failed_total` (incremented per row-level failure during reconciliation:
   **both** a non-not-found `FlowV1FlowGet` error **and** a `FlowV1FlowDelete` failure on a genuine
   orphan share this one counter, with no `reason` label distinguishing which RPC failed —
   design-review-round-8 fix: an earlier draft of the plan only wired the metric up for the
   `FlowV1FlowDelete` branch, leaving the `FlowV1FlowGet`-error branch — the one a
   `bin-flow-manager` outage or an open circuit breaker actually fails on, for every remaining
   candidate in the pass — invisible to Prometheus despite this section always having defined the
   metric to cover both). Neither reuses VOIP-1443's `promCampaignFlowDeleteFailedTotal{reason=...}`
   — that metric is
   documented as "failures during a live campaign delete"; reusing it here would conflate two
   operationally distinct signals ("a delete is failing right now" vs. "the reconciler couldn't
   clean up a known-orphaned flow") into one series and make VOIP-1443's own doc line inaccurate.
   No separate counter for `Saturated`/`RecentSaturated`/`Partial` (design-review-round-3/4
   clarification) — each is surfaced only via a log warning and its own field on the response, kept
   out of Prometheus to avoid a counter per signal that is informational far more often than it is
   actionable (see the plan's "Scan order and coverage" section for the corrected framing).
6. **Tests** (expanded significantly in the plan per design-review-round-3/4 — this is a sketch of
   the core cases, see the plan's task 6 for the full matrix including `Saturated`/`RecentSaturated`
   assertions computed correctly independent of the pass-timeout bail-out, the `Partial` field on a
   self-timed-out pass, the defense-in-depth guard tested in isolation, and an idempotent-rerun
   case): unit tests for `ReconcileOrphanedFlows` covering: a genuine orphan (flow `TMDelete` nil)
   found and cleaned; a flow already correctly deleted (`TMDelete` set) correctly skipped and NOT
   re-deleted; `FlowV1FlowGet` returning typed not-found (flow row doesn't exist at all) counted as
   `Skipped` (design-review-round-4 fix: not an uncounted no-op); a transient `FlowV1FlowGet` error
   counted as `Failed` and logged, not counted as cleaned; the per-pass scan limit is respected; a
   warning fires for an orphan near the window edge. A basic listenhandler-level test for the new
   `/v1/campaigns/flows/reconcile` route (request parses, calls through to the handler, returns a
   sane response).
7. **Docs**: `bin-campaign-manager/docs/operations.md` (new metrics, new route,
   `Saturated`/`RecentSaturated` runbook note), `bin-campaign-manager/docs/architecture.md`
   (routing-table entry for the new
   route), and — added in design-review-round-3, missed in the prior revision —
   `bin-campaign-manager/docs/domain.md` (the new `ReconcileResult` domain type under `models/`,
   per the root CLAUDE.md's service-docs-sync table: `models/.../*.go` changes map to
   `docs/domain.md`).
8. **Jira housekeeping**: VOIP-1444's summary already corrected to "bin-campaign-manager:"; the
   `bin-flow-manager` idempotency question filed separately (and now fully orthogonal) as
   VOIP-1448.

## What this explicitly does not do

- Does not clean up the pre-existing historical backlog beyond the bounded recent window — a full
  historical sweep is a separate, explicitly-deferred, authorization-gated data operation.
- Does not add Prometheus alerting (explicitly dropped from VOIP-1444's scope by 대표님's
  instruction) or any new alerting infrastructure — `bin-schedule-manager`'s existing audit trail
  and kill switch are operational tooling already in place, not new alerting.
- Does not touch `bin-flow-manager` at all — the `Delete()` idempotency question is filed as
  VOIP-1448, fully orthogonal to this ticket's correctness once claim-based scheduling removes the
  concurrency motivation for touching it.
- Does not add any new locking infrastructure of its own — `bin-schedule-manager`'s existing claim
  mechanism is reused as-is, nothing new to build or maintain in `bin-campaign-manager`.
- Does not change `Delete()`'s existing best-effort semantics from VOIP-1443 in
  `bin-campaign-manager` (campaign delete still succeeds even if flow cleanup fails) — this job is
  a downstream safety net, not a change to that contract.

## Confidence and remaining open questions for design/plan phase

High confidence in: the corrected orphan-detection criterion (`TMDelete` check, independently
re-derived and cross-checked against `Delete()`'s own behavior), the not-found-detection idiom for
`FlowV1FlowGet`, the `bin-schedule-manager` mechanism choice for *dispatch* (guarantees at most one
dispatch of a given scheduled slot, eliminating the concurrency question this document spent three
rounds on for that half of the problem), the service placement, and keeping VOIP-1448 as a
separate, now-orthogonal ticket.

Design-review-round-3 correction to the above, itself corrected in design-review-round-4, itself
corrected in design-review-round-5, itself corrected in design-review-round-6: dispatch-uniqueness
alone does not make overlap "structurally impossible" — that also requires
`ReconcileOrphanedFlows` to bound its own execution. Round 3 mis-stated the bound as "below
`timeout_ms`, to finish before the reaper's abandonment deadline" — tracing the actual
`bin-schedule-manager` dispatch/claim code shows the reaper does not govern re-dispatch at all
(round 3 was right to move away from it), but round 4 then over-corrected by declaring `timeout_ms`
cosmetic and the cron interval the *only* real bound; round 5 over-corrected the other way,
declaring both "independent" mutual-exclusion guards. The accurate picture, per round 6, corrected
once more in round 7: **guard 1**, "Forbid overlap" (`ExecutionHasRunning`), is checked *before*
the `tm_next_run` CAS on every claim path and stays effective only as long as the execution row
remains `running` — which lasts **at most `timeout_ms`** from the moment `bin-schedule-manager`
sends the RPC, given this schedule's `retry_max: 0` (design-review-round-8 qualification: at
`retry_max > 0` the window would be up to `(retry_max+1)*timeout_ms + retry_max*backoff`, since
`bin-schedule-manager` loops retries within one execution row before completing it — not this
ticket's configuration, but worth noting the bound is conditioned on `retry_max: 0`, not universal)
(design-review-round-7 correction: not "exactly `timeout_ms`, regardless of how long
the pass actually runs" — an earlier draft of this sentence stated the precise inverse of the
property being secured; `ExecutionComplete` fires as soon as `SendRequest` returns, whether that's
because the pass finished early with a real response or because the `timeout_ms` wait itself
expired, so the row's `running` duration tracks the pass's actual duration up to that ceiling, not
a fixed window) — so `reconcilePassTimeout` must stay strictly below `timeout_ms`, with margin for
real delivery delay (**this is constraint (a)**, using the same label the plan uses —
design-review-round-8 addition, for cross-document consistency), for this guard to cover the
pass's real duration. This is the **entire** mutual-exclusion mechanism; it has no runtime monitor
(a violation reads as an ordinary `success` row). **Guard 2**, the `tm_next_run` CAS, guarantees
at-most-once **cron** dispatch per boundary, but because guard 1 already short-circuits any claim
attempt while the row is `running`, guard 2 never gets a chance to matter for mutual exclusion in
the first place — its role is **cadence/liveness**: `reconcilePassTimeout` well below the cron
interval (**constraint (b)**) is what keeps an overrunning pass from making every subsequent cron
tick observe a dispatch-level overlap skip
(`schedule_manager_dispatch_total{result="skipped_overlap"}`) forever — a cadence failure, not a
race. The corrected,
load-bearing requirement is: `ReconcileOrphanedFlows`'s self-imposed `context.WithTimeout` must be
sized strictly below `timeout_ms` (safety) *and* well below the schedule's cron interval (cadence)
— see the plan's task 3 and task 8's corrected (three-part, reaper-formula reinstated and
correctly attributed) cross-check.

Open for design/plan, not resolved here: the concrete window/scanLimit constants and the
`tm_delete` index decision (both pending the deleted-row-count **and** deletion-rate data points —
peak per-cron-interval rate, not just a daily average, per the design-review-round-3 correction
above), and the exact cron interval for the new schedule (which now also fixes
`recentIntervalSec`'s seeded value and `reconcilePassTimeout`/`timeout_ms`'s sizing against both
guards above — design-review-round-4/5). Newly open per design-review-round-5: the operational
runbook must treat any future change to the seeded `cron` interval (including the `RecentSaturated`
remedy) as a two-step change — update `target_data.recent_interval_sec` to match, and re-verify
`reconcilePassTimeout` against the new interval — since neither is automatically re-checked once
the schedule is live; see the plan's "Scan order and coverage" and task 7. The metric-reuse
question above is no longer open — resolved in the plan (design round 1) as two new dedicated
counters, no reuse of VOIP-1443's metric, with no third counter added for the
saturation/rate-risk/partial-pass signals (design-review-round-3/4 clarification — log + response
fields only).
