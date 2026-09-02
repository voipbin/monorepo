# Implementation Plan: VOIP-1444 — reconciliation job for orphaned campaign flows

(Title deliberately does not carry a round number — a stale round-number title was flagged
repeatedly across review rounds. Round tracking lives entirely in "Revision history" below.)

Full technical analysis, all traced source citations, and the extensive round-by-round review
history that led to the final design (mechanism swaps, corrected orphan-detection criterion, scope
narrowing — see that document's own "Revision history" for the full, still-growing round count):
[2026-09-02-voip-1444-orphaned-flow-reconciliation-design.md](2026-09-02-voip-1444-orphaned-flow-reconciliation-design.md).
This plan translates that analysis into an ordered task list; it does not re-derive it.

## Revision history

- **Design round 1** (REQUEST_CHANGES): found the "two PRs" premise wrong (this is one monorepo,
  not two repos — corrected to one PR), no rollout sequencing for the schedule-seed-before-route-
  exists ordering hazard (fixed: seed disabled, enable post-deploy), a structural scan-starvation
  bug in the bounded query (`ORDER BY tm_delete ASC LIMIT n` never advances past the oldest `n`
  candidates once the window holds more than `scanLimit` rows — fixed: `DESC` order, converting
  this into the already-accepted "ages out of window" risk instead of a permanent blind spot, plus
  an explicit saturation signal), undefined success/failure semantics on `ReconcileOrphanedFlows`,
  a punted metric-reuse decision, and a test plan under-sized for an all-customer data mutator. All
  addressed below.

- **Design round 2** (REQUEST_CHANGES): the round-1 `DESC`-order fix was justified with a
  mathematically false claim ("candidates rise toward the front as newer rows age past the
  window") — under `DESC` order a row's rank only ever increases as more deletions accumulate, so
  a row that falls past `scanLimit` never re-enters range; this is a permanent, in-scope coverage
  gap, not the deliberately-accepted window-edge boundary. Corrected: the real safety condition is
  `scanLimit` exceeding deletions-per-cron-interval (not per-window), the saturation signal is now
  documented as actionable (raise `scanLimit` / shorten interval), and a new "Why no cursor/
  watermark instead" section records the deliberate retry-vs-saturation trade-off. Also: the
  proposed `response.*` DTO for the reconcile route was a field-for-field identical-json-tag copy
  of the `ReconcileResult` domain struct — exactly the layering violation the root CLAUDE.md
  forbids (style (A) wearing a DTO costume). Corrected: `ReconcileResult` is now defined as the
  domain type itself (with json tags) under `bin-campaign-manager/models/`, and the listenhandler
  marshals it directly — no `response.*` DTO introduced. Task list renumbered (`bin-campaign-
  manager` 6→7 tasks, `bin-dbscheme-manager` 3→4 tasks under the new numbering — design-round-3
  correction: an earlier version of this entry mis-stated these as "6→9" and "unchanged at 3") to
  reflect splitting the old task 2 into a dedicated `ReconcileResult`-definition task, and the
  rollout-sequencing task now inserts a pre-enable manual smoke-test step
  (`POST /v1/schedules/{id}/execute`) before `schedule-control schedule enable`.

- **Design round 3** (REQUEST_CHANGES): six defects found in round 1/2's own fixes.
  (1) **Unbounded execution, breaking the "structurally impossible race" claim**: this service's
  RPC listenhandler builds every request's context with `context.Background()` — it does not
  propagate the caller's `timeout_ms` as a deadline, so the `ctx.Err()` guard in
  `ReconcileOrphanedFlows` was dead code and a pass could run past the schedule's abandonment
  deadline, letting the reaper release the slot for a second dispatch while the first pass was
  still in flight — exactly the race the design doc claimed was impossible. Fixed: task 3 now
  wraps the pass in its own `context.WithTimeout`, sized strictly below the schedule's
  `timeout_ms` with margin. (2) **Saturation signal is a chronic false alarm across most of the
  plausible rate range**: `len(candidates) == scanLimit` fires whenever in-window candidates
  exceed `scanLimit` — which happens routinely even when every *newly-deleted* campaign is still
  being examined promptly, because old, already-`Skipped` (clean) rows keep re-occupying batch
  slots until they age out of the window. "Shorten the interval" does not clear this (it reduces
  per-interval arrivals, not the per-window population that drives batch fullness). Fixed: kept
  `Saturated` as an honest "batch reached its cap" fact (informational, expected at scale, no
  action implied by itself), and added a second, precisely-actionable signal —
  `RecentSaturated` — computed for free from the already-fetched batch (no extra query, no
  persisted cursor): true when the count of returned rows with `tm_delete` within the most recent
  cron interval itself reaches `scanLimit`, i.e. the actual safety condition
  (deletions-per-interval < `scanLimit`) is being approached or violated. Runbook: `Saturated`
  alone → no action; `RecentSaturated` → raise `scanLimit` and/or shorten the interval (both now
  genuinely help this signal). (3) **"Why no cursor" retry-property overclaim**: the full-window
  retry claim silently required the *stronger*, per-window inequality (`scanLimit` > deletions per
  *window*) that the safety-condition section explicitly disclaims in favor of the weaker
  per-interval one — corrected to state retry holds "for as long as the candidate's rank stays
  within `scanLimit`", not unconditionally for the whole window; the pre-implementation check's
  "full window coverage" phrasing corrected to "at-least-once examination shortly after deletion";
  the "safe-by-construction" claim softened to "the failure mode is visible via `RecentSaturated`",
  since deletion rate is unmeasured at plan time. Also: the rate check now asks for the **peak**
  per-cron-interval deletion rate, not just a daily average (the known trigger customer is a
  bursty automated loop). (4) The design doc still specified the rejected `ASC` order and a stale
  `(cleaned int, err error)` handler signature, and its own revision history lagged the body by a
  full round (missing a Round 8 entry) — both synced. (5) Task 7 omitted `docs/domain.md`, which
  the root CLAUDE.md's service-docs-sync table requires for the new `models/` file introduced in
  task 2 — added. (6) The saturation signal was written to require "increment a `saturated`
  counter" in one section but task 5 defined only two counters — resolved by keeping the
  saturation/rate-risk signal out of Prometheus entirely (log + response field only), removing the
  inconsistency rather than adding a third counter for a mostly-informational signal. All
  addressed below; see the design doc's own revision history for its mirrored entries.

- **Design round 4** (REQUEST_CHANGES): traced round 3's own headline fix against actual
  `bin-schedule-manager` source and found it named the wrong mechanism. (1) **The race-prevention
  argument was still wrong, differently**: round 3 claimed the race was prevented because a
  bounded pass finishes "before the reaper's abandonment deadline", implying the reaper releasing
  a stuck slot was the risk. Tracing `pkg/dispatchhandler/dispatch.go` and
  `pkg/dbhandler/execution.go` shows the reaper never enters into it: `ScheduleClaimAndCreateExecution`
  advances `tm_next_run` to the *next* cron boundary in the same atomic claim transaction, before
  the RPC to `bin-campaign-manager` is even sent — so the same schedule cannot be re-claimed for a
  new **cron** dispatch until that next boundary arrives, regardless of how long (or whether) the
  current pass finishes. The real overlap condition is **pass duration exceeding the cron
  interval itself** (the previous pass still physically running in `bin-campaign-manager` when
  the next cron-triggered RPC arrives) — not a reaper-driven slot release. Fixed: reworded every
  claim below to name the `tm_next_run` CAS as the dispatch-uniqueness guarantee, and
  `reconcilePassTimeout` as bounding pass duration **well below the cron interval** (the true
  concurrency-prevention lever) rather than primarily below `timeout_ms` (`timeout_ms` still
  matters, but for audit-row accuracy — see (5) below — not for preventing overlap). Task 8's
  "cross-check against the reaper's abandonment-deadline formula" is dropped as the safety
  argument (it reduced to an unfalsifiable tautology with `retry_max: 0` and a fixed reap margin)
  and replaced with the actual two constraints: `reconcilePassTimeout` << cron interval, and
  `reconcilePassTimeout` < `timeout_ms` with margin.
  (2) **`RecentSaturated` needed the cron interval, which `bin-campaign-manager` cannot read from
  `bin-schedule-manager`'s database, and a hardcoded Go constant would desynchronize the moment
  someone applied this very section's "shorten the interval" remedy**: fixed by threading
  `recentInterval` through as an explicit request parameter (seeded in the same `target_data` row
  as `cron`, so both live in one migration, one diff, one review) rather than a compiled constant.
  (3) **`RecentSaturated` was specified to be computed inside the RPC loop**, so the self-imposed
  pass-timeout bail-out (fix (1) above) could stop the count short of `scanLimit` even on a
  genuinely saturated batch — a false negative in exactly the overload case the signal exists to
  catch. Fixed: `RecentSaturated` (and `Saturated`) must now be computed in a dedicated,
  RPC-free pass over the fetched slice immediately after the query returns, before any
  `FlowV1FlowGet`/`FlowV1FlowDelete` call. (4) Dropped the "or approached" half of
  `RecentSaturated`'s stated meaning — the derivation is exact (`count == scanLimit` iff true
  interval deletions `>= scanLimit`; `scanLimit - 1` reads as clean, no early-warning margin
  exists) — and disclosed a real, accepted limitation: the signal assumes the previous pass ran
  exactly `recentInterval` ago, so a skipped/delayed/failed prior pass (schedule-manager's own
  single-fire, no-catch-up semantics) can make it under-report. (5) **A pass that hits its own
  timeout returns `err == nil` with partial counts and is recorded as a `success` execution, with
  no signal anywhere that it was cut short**: added a `Partial bool` field to `ReconcileResult`,
  set true on the self-timeout bail-out and logged at `warn` — not a new counter, following the
  same informational-signal pattern as `Saturated`. (6) The typed-not-found `FlowV1FlowGet` branch
  incremented neither `Skipped` nor `Failed`, so the three counters didn't sum to candidates
  examined and the case had nothing to assert in a test — fixed by counting it as `Skipped` (it is
  a legitimately clean end state, same bucket as an already-deleted flow). (7) One more unowned
  "log/metric" phrase survived for the window-edge warning (the same class of defect round 3 fixed
  for the saturation signal) — corrected to "log" only. (8) Misattributed which document's
  revision history contained the task-count arithmetic error round 3 fixed — it was this plan's
  round-2 entry, not the design doc's own body; corrected in the design doc. All addressed below.

- **Design round 5** (REQUEST_CHANGES): traced round 4's fix against `bin-schedule-manager`'s
  claim/dispatch source in full and found round 4 had, for the third consecutive round, named an
  incomplete safety-mechanism set — this time not by omitting a guard but by affirmatively denying
  that a real one mattered. (1) **`ExecutionHasRunning` ("Forbid overlap") is a second, independent
  concurrency guard round 4 never mentioned, and `timeout_ms` is genuinely load-bearing for it, not
  merely "cosmetic for audit accuracy" as round 4 claimed**: the execution row stays `running`
  until `bin-schedule-manager`'s own `SendRequest` wait (bounded by `timeout_ms`) returns — not
  until this pass actually finishes. If `reconcilePassTimeout` were allowed to exceed `timeout_ms`,
  there would be a real window where `bin-schedule-manager` believes the pass is done (so
  `ExecutionHasRunning` no longer blocks a new dispatch) while `bin-campaign-manager` is still
  physically executing it — a manual `/v1/schedules/{id}/execute` fired in that window would
  genuinely race the still-running pass. Fixed: task 3 and task 8 now state both
  `reconcilePassTimeout < timeout_ms` (guard 1: keeps "Forbid overlap" effective) and
  `reconcilePassTimeout << cron interval` (guard 2: keeps the `tm_next_run` CAS's next cron claim
  from arriving mid-pass) as independently load-bearing, with neither cosmetic; reinstated the
  reaper-abandonment-deadline cross-check round 4 dropped as "tautological" — it is not: it bounds
  how long a *crashed* replica's stuck `running` row can suppress the schedule via the overlap
  guard before the reaper frees it, a real and distinct property from guards 1 and 2. Added
  `skipped_overlap` as the documented observable symptom if any of the three constraints is ever
  violated (previously absent from every doc). (2) **The `RecentSaturated` remedy ("shorten the
  interval") can silently violate guard 2**, since `cron` is runtime-mutable (no redeploy) while
  `reconcilePassTimeout` is a compiled constant with no automatic re-check on a live interval
  change — fixed by adding an explicit runbook step (task 7): any interval change must update
  `target_data.recent_interval_sec` in the same change *and* re-verify `reconcilePassTimeout`
  against the new interval. (3) The `Partial` bullet in task 3 had a parenthesis left unclosed by
  round 4's edit, burying the normative instruction inside a justification clause — reformatted.
  (4) The defense-in-depth guard (`if c.TMDelete == nil`) reproduced the exact counter-invariant
  gap round 4 had just fixed for the typed-not-found branch (incremented nothing, logged nothing)
  — fixed to count `Skipped` and log at `warn`. (5) A genuine contradiction from round 4's own
  edits: task 8's docstring claimed `cron`/`recent_interval_sec` "can only be edited apart by
  missing a line in the same diff" while the "Scan order and coverage" section said the opposite
  ("no automated check can catch a mismatch... different systems' config surfaces") — both are
  true at different times (seed-time co-location vs. post-seed independent editing); reworded both
  passages to say so explicitly instead of contradicting each other. (6) Design doc's "VOIP-1448's
  relevance" paragraph still reasoned from schedule-level retries ("the schedule's own
  `retry_max`/timeout settings could still in principle overlap with a next scheduled run") after
  `retry_max: 0` had long since been settled elsewhere — corrected. (7) A test for the pre-loop
  signal computation specified a 0-duration `reconcilePassTimeout`, which would fail the initiating
  query itself before there was any `ReconcileResult` to assert against, testing nothing about the
  property it claimed to cover — corrected to use a timeout that survives the query but expires
  during the RPC loop; also defined `passStart` in task 3, used previously but never declared. (8)
  Softened this document's own "8-round review history" reference to the design doc, now stale
  after four additional design-review rounds, to avoid re-staling on every future round. All
  addressed above and in the design doc.

- **Design round 6** (REQUEST_CHANGES): traced round 5's two-guard model against
  `bin-schedule-manager`'s actual claim/dispatch source and found two further errors, both in
  material round 5 had just added. (1) **`skipped_overlap` is not an execution status, and no
  execution row is ever created when it fires** — the execution status enum is
  `running`/`success`/`failed`/`abandoned`; `skipped_overlap` is a **dispatch-level metric result
  label** (`schedule_manager_dispatch_total{result="skipped_overlap"}`), emitted on a code path
  that returns *before* an execution row would ever be inserted. Worse, it is the wrong signal for
  the constraint it was attached to: a constraint-(a) violation (the actual concurrency-unsafe
  case, at the time of this round labeled the mutual-exclusion constraint) produces an ordinary
  `success` row with no distinguishing signal at all — `skipped_overlap` only ever fires for the
  cadence-degradation case. Fixed everywhere this appeared (task 3, task 7, acceptance criteria,
  verify tasks) to correctly name it a metric label, attach it only to the constraint it actually
  indicates, and state plainly that the real concurrency constraint has no runtime monitor at all —
  only correct static sizing. (2) **Constraint (b) is not an independent concurrency guard —
  round 5's "both are real, independent concurrency-safety requirements" over-corrected round 4's
  opposite mistake**: `ExecutionHasRunning` is checked *before* the `tm_next_run` CAS on every claim
  path, so as long as constraint (a) holds (the execution row stays `running` for the pass's whole
  physical duration), a cron claim arriving mid-pass is short-circuited into an overlap skip — no
  CAS runs, no second execution is ever created. Constraint (b) is therefore a cadence/liveness
  requirement (prevents the schedule from permanently falling behind), not a mutual-exclusion
  mechanism; mutual exclusion rests on guard 1 (`ExecutionHasRunning`) plus constraint (a) alone.
  (Design-round-8 correction to this entry's own historical record: an earlier revision of this
  paragraph used a since-abandoned "(1)/(2)" numbering scheme in place of the "(a)/(b)" labels used
  everywhere else — including two bare, referent-less mentions of "constraint (1)" — round 7's own
  standardization pass missed this entry when relabeling; this paragraph now uses (a)/(b)
  throughout, consistent with the rest of both documents.) Relabeled in several places (not fully
  "throughout", per the design-review-round-7 finding above) to stop implying dropping (b) reopens
  a race. Also fixed in this pass: (3) the reap-
  deadline cross-check (task 8, item 3) had been reinstated in round 5 but attributed the crash
  scenario to a `bin-campaign-manager` replica dying "mid-pass" — wrong; that path is already fully
  handled by `SendRequest`'s own `timeout_ms` bound and never reaches the reaper. The reaper only
  matters for `bin-schedule-manager`'s own dispatch goroutine crashing (its documented
  "abandon-not-drain" case) — corrected the attribution. (4) Constraint (a)'s closure claim
  ("always finishes... before `bin-schedule-manager` ever considers the row non-`running`") did
  not account for the two timeout clocks starting at different points (`timeout_ms` from when
  `bin-schedule-manager` sends the RPC; `reconcilePassTimeout` from when `bin-campaign-manager`
  starts processing it) — added an explicit note that the required margin must cover real
  message-delivery delay, not just processing-time variance. (5) Design doc's "Proposed scope"
  item 3 still named the superseded `a5e6f559299c` migration as the primary precedent, contradicting
  its own earlier-corrected paragraph naming `0c037bf0a362` — synced (no practical column
  difference between the two precedents, but the normative reference was self-contradictory). (6)
  A stale "set `retry_max` consistently with it" phrase in the design doc, left over from before
  `retry_max: 0` was settled two sections earlier in the same document — removed. (7) The
  Alembic-column acceptance criterion covered `timeout_ms > 0` but not the reap-deadline
  cross-check's `timeout_ms + 60s < cron` condition, which only the verify task covered — added to
  the acceptance criterion too. All addressed above and in the design doc.

- **Design round 7** (REQUEST_CHANGES): independently re-derived round 6's model from
  `bin-schedule-manager` source — confirmed correct, and confirmed *stronger* than round 6 itself
  verified (the manual claim path skips the `tm_next_run` CAS entirely, not just deprioritizes it,
  so guard 2 doesn't exist there at all). But round 6's fix was applied incompletely, leaving four
  places carrying the refuted round-5 "two independent concurrency guards" framing, one factual
  error in the sentence defining the mechanism itself, and one live contradiction between this
  document's own acceptance criteria/verify-task section (using labels "(1)"/"(2)") and task 3
  (using "(a)"/"(b)") that left a verify-task line with no literal referent in the task it was
  meant to verify. Fixed: standardized on constraint labels (a)/(b) everywhere, removed the
  (1)/(2) scheme entirely; corrected the acceptance-criteria bullet's "concurrency-safety
  condition" back to "cadence/liveness condition" for constraint (b) in the "Scan order and
  coverage" section, matching task 3; swept the design doc's lead bullet and VOIP-1448 paragraph
  (see the design doc's own revision history). Also found and fixed: the `skipped_overlap` metric
  fires only on the **cron** claim path — a manual execute hitting the same overlap guard returns
  `EXECUTION_IN_PROGRESS` instead, with no metric — added this to task 10's rollout smoke-test step
  (which is itself a manual execute and can legitimately hit this) and to the acceptance criteria;
  the `skipped_overlap` counter's granularity is per-replica, in-memory dedup, so a single stuck
  slot can increment it more than once across replicas — added a one-line caveat to task 7's
  runbook text; and the "`timeout_ms` unset silently produces `0`" claim (present in the design doc
  since its original issue-analysis round 6) doesn't account for `timeout_ms` being `NOT NULL`
  with no `DEFAULT` — under strict `sql_mode`, omitting it fails the migration loudly at
  apply-time; the silent-`0` outcome needs a relaxed `sql_mode`. This claim never made it into this
  plan document directly (only the design doc), so no plan-side fix was needed beyond noting it
  here. All addressed above and in the design doc.

- **Design round 8** (REQUEST_CHANGES): a fresh read of the sections unrelated to
  `bin-schedule-manager` concurrency (explicitly requested after seven straight rounds focused
  there) found one substantive defect: **`campaign_flow_reconcile_failed_total` was defined (task
  5, and the design doc) to cover both the `FlowV1FlowGet`-error branch and the
  `FlowV1FlowDelete`-error branch of task 3, but only the latter branch actually incremented the
  metric** — the former incremented `Failed` in-memory only. Since a `bin-flow-manager` outage or
  an open circuit breaker fails every remaining candidate on exactly the unmetriced branch, the
  counter would read zero during the outage it exists to catch. Fixed: both branches now increment
  the shared counter (task 3, task 6's test list, task 5, and the design doc's Metrics section and
  pseudocode all updated to agree); documented explicitly that the one counter conflates both RPCs
  with no `reason` label. Also found and fixed: this document's own round-6 entry above still used
  a since-abandoned "(1)/(2)" numbering with backwards "[now (1)]"-style annotations and two
  referent-less bare "constraint (1)" mentions — corrected retroactively to (a)/(b), and the
  design doc gained explicit (a)/(b) labels so round 7's "standardized... in both documents" claim
  is now actually true (it previously only held for this plan); the verification checklist named
  only `go test`/`golangci-lint`, a 2-of-5 subset of root CLAUDE.md's mandatory 5-step workflow —
  `go generate` specifically is required here, since tasks 1 and 3 add methods to the
  `DBHandler`/`CampaignHandler` interfaces that `mockgen` must regenerate mocks from before task
  6's tests compile — added the interface-declaration step to tasks 1/3 and the full workflow to
  the acceptance criteria/verify tasks; and task 10's smoke-test step never said how to actually
  fire a manual execute (`schedule-control` has no `execute` subcommand) — added the RPC-client
  path and the prerequisite `schedule get` step to resolve the schedule's UUID. Two non-blocking
  notes also folded in: the "at most `timeout_ms`" mutual-exclusion bound is conditioned on
  `retry_max: 0` (noted in the design doc), and a saturated pass's RPC volume shares
  `bin-campaign-manager`'s process-wide circuit breaker with live traffic (noted as a scope
  caveat). All addressed above and in the design doc.

- **Design round 9** (REQUEST_CHANGES): verified the round-8 metric fix landed in all five places
  it should have, plus found one more place it hadn't reached and four smaller issues from a fresh
  read of sections outside the concurrency-model discussion. The design doc's "Not-found detection
  for `FlowV1FlowGet`" section — the normative text an implementer would actually build the
  error-branch logic from — still described the non-not-found-error case as "logged and skipped",
  which reads as the `Skipped` counter even though the settled semantics reserve `Skipped` for
  legitimately-clean states and this case is `Failed`; also omitted the metric increment entirely.
  Fixed to say `Failed`, with the metric. Also found: the acceptance criteria still named
  `schedule-control` as a way to manually trigger the new route, when task 10 (a round-8 fix)
  already established `schedule-control` has no `execute` subcommand and cannot reach this route
  at all — fixed to name only the `/v1/schedules/{id}/execute` RPC; one referent-less "Constraint
  (1)" survived in this document's own round-6 entry, undercutting round 8's claim to have fixed
  "two" such mentions — relabeled to "Constraint (a)"; the acceptance criteria never mentioned
  `campaign_flow_reconcile_failed_total` at all despite it being the exact thing round 8 fixed —
  added a criterion pinning it explicitly; and the pre-implementation check's "This decides" list
  omitted the `cron` interval, even though task 8 says to seed `cron` "from the pre-implementation
  check" — added it to the list and to the conservative default (`'0 */6 * * *'` /
  `recent_interval_sec = 21600`, matching task 8's own worked example). Non-blocking: retitled the
  design doc from "Issue Analysis" to "Issue Analysis and Design", since design-review rounds 3-8
  folded substantial design content into the same file and the plan already refers to it as "the
  design doc". All addressed above and in the design doc.

- **Design round 10** (APPROVE, first of two required consecutive approvals): a fresh,
  from-scratch read of the full task list as an implementer, the acceptance-criteria-to-task
  mapping, both revision histories for orphaned staleness, and the layering/DTO compliance —
  independently re-verified against `bin-campaign-manager`, `bin-schedule-manager`, and
  `bin-common-handler` source (route-regex collision, `tm_delete` NULL-sentinel convention, the
  two-directory PR scope, the metric registration pattern) — found no defect that would cause an
  implementer to build the wrong thing. Three non-blocking suggestions folded in anyway: task 3
  now declares its package constants (`window`, `scanLimit`, `reconcilePassTimeout`) and the exact
  `CampaignListDeletedSince` call, making it self-contained without cross-referencing the design
  doc; task 7 now states explicitly that `scanLimit` (a compiled constant, needs redeploy) and
  `cron`/`recent_interval_sec` (a live data edit) are not equally fast remedies for
  `RecentSaturated`; and task 7 now notes that `Saturated`/`RecentSaturated`/`Partial` survive in
  `bin-schedule-manager`'s own `schedule_executions.result` column even after
  `bin-campaign-manager`'s local logs rotate, which is what makes the no-third-counter decision
  safe.

## Scope

A periodic, `bin-schedule-manager`-triggered job in `bin-campaign-manager` that finds campaigns
deleted within a recent, bounded time window whose backing flow is still live (not soft-deleted)
and deletes that flow, closing the gap VOIP-1443 made observable but did not close.

**One PR, both directories touched**: `bin-campaign-manager` (the job itself) and
`bin-dbscheme-manager` (the Alembic seed migration registering the schedule, plus the `tm_delete`
index migration). Both are directories inside this single monorepo, not separate repositories —
the "one PR per repo" structural exception does not apply here (corrected from an earlier draft
that wrongly claimed it did). One PR, commit body with both `bin-campaign-manager:` and
`bin-dbscheme-manager:` prefixed bullets, per the standard convention.

Explicitly out of scope (all already filed as separate tickets or explicitly deferred):
- The existing historical flow backlog beyond the window (VOIP-1443's design doc, needs separate
  authorization).
- Prometheus alerting (dropped from VOIP-1444 by 대표님's explicit instruction).
- `bin-flow-manager`'s `Delete()` non-idempotency (VOIP-1448, orthogonal — `bin-schedule-manager`'s
  claim semantics make concurrent duplicate dispatch structurally impossible, so this job's
  correctness never depends on VOIP-1448 landing).

## Pre-implementation check (do this first, before fixing constants)

Get, from production, both: (a) the current count of `campaign_campaigns` rows with
`tm_delete IS NOT NULL`, and (b) the **peak** rate of new campaign deletions within a single
prospective cron interval, not just a daily average (design-round-3 correction: a daily average
can understate the peak — the known trigger customer is a bursty, automated api-validator loop;
rate is what determines whether `scanLimit` and the scan-order fix below actually give
at-least-once examination shortly after each deletion, not "full window coverage" in the sense of
every in-window row being re-examined every pass). This decides:
- The window size and `scanLimit`.
- Whether the `tm_delete` index migration is needed now or can be deferred.
- **The `cron` interval to seed (design-round-9 addition — an earlier draft left this
  undetermined: task 8 says to seed `cron` "from the pre-implementation check", but this list
  didn't actually include it)**: pick an interval such that the peak-rate data point above keeps
  `scanLimit` comfortably above deletions-per-interval (the safety condition in "Scan order and
  coverage" below), and such that `reconcilePassTimeout`/`timeout_ms` (task 3/task 8) fit
  comfortably beneath it. This same value also becomes `target_data.recent_interval_sec` (task 8)
  — the two are seeded together from this one decision.

Default to proceeding with a conservative, safe choice if this data cannot be obtained before
implementation: **window = 7 days, scanLimit = 500, cron = `'0 */6 * * *'` (every 6 hours,
`recent_interval_sec = 21600`), add the index anyway** (small, additive, reversible) rather than
blocking implementation indefinitely — see "Scan order and coverage" below; this default's safety
is not assumed blind, but it is also not "by construction" in the strong sense (design-round-3
correction) — deletion rate is unmeasured at plan time, so what the design actually guarantees is
that a rate violation becomes **visible** via the `RecentSaturated` signal, not that it cannot
happen.

## Scan order and coverage (design-round-1 correction)

**Rejected**: `ORDER BY tm_delete ASC LIMIT scanLimit` (oldest-deleted-first). With no cursor
carried between passes, this query returns the *exact same* oldest `scanLimit` rows on every
single scheduled run once the window holds more candidates than `scanLimit` — rows beyond position
`scanLimit` are never reached until enough of the oldest rows age out of the window on their own.
This is a structural blind spot, not a matter of degree: campaigns deleted while the window is
"full" from this job's perspective can go unprocessed for the rest of their time in the window,
however long that is.

**Adopted**: `ORDER BY tm_delete DESC LIMIT scanLimit` (most-recently-deleted-first). This
guarantees every pass always covers the most recent deletions — matching this job's actual
purpose (catch a leak shortly after it happens).

**Design-round-2 correction — the residual risk is real and permanent, not equivalent to the
window-edge risk**: under `DESC`, a candidate's rank at any later pass is the count of deletions
strictly newer than it, which is monotonically non-decreasing — a row that once falls past
position `scanLimit` **never re-enters range**. It does not "rise toward the front as newer rows
age past the window" (an earlier draft of this section claimed this, incorrectly — every row
newer than it ages out *after* it does, never before). So a burst of churn that pushes a
still-in-window candidate past `scanLimit` makes that specific candidate permanently unreachable
by this job for the rest of its time in the window; it is only rescued by aging out into the
out-of-scope historical backlog, the same *outcome* as the window-edge risk but via a genuinely
different, in-scope trigger (a coverage failure on a row this job was supposed to reach, not a
deliberate scope boundary). Do not conflate the two when reasoning about correctness.

**The actual safety condition, to size against real data**: `scanLimit` must exceed the number of
campaign deletions per cron interval (not per window), with margin — e.g. at a 6-hour interval and
`scanLimit = 500`, this design is safe as long as fewer than ~500 campaigns are deleted platform-
wide per 6 hours. The pre-implementation rate check above exists specifically to confirm this
inequality holds; if it doesn't, raise `scanLimit`, shorten the interval, or both.

**Two distinct signals, not one — design-round-3 correction**: a naive `Saturated =
len(candidates) == scanLimit` conflates two very different situations. Once steady-state in-window
candidates exceed `scanLimit` (which happens routinely at scale, since already-`Skipped`
correctly-clean rows keep re-appearing as candidates every pass until they age out of the window),
`Saturated` is true on **every single pass forever**, even when every newly-deleted campaign is
still being examined on the first pass after its own deletion — i.e. even when the job is working
exactly as intended. Treating that as an actionable alarm produces a chronic false positive with
no real remedy (see below), which is worse than no signal at all.

- **`Saturated`** (`len(candidates) == scanLimit`): kept, but reframed as purely **informational** —
  "this pass's batch reached its cap; the window holds more history than one pass returns, which is
  normal and expected at scale." No action implied by itself. No independent counter — see task 5.
- **`RecentSaturated`** (new in design-round-3, corrected in design-round-4; the actual actionable
  signal): computed from the batch already fetched for `Saturated` — no extra query, no persisted
  cursor. **Design-round-4 correction — must be computed in a dedicated pass over the fetched
  slice, before the RPC loop begins, not interleaved with it**: an earlier draft computed this
  "while iterating the batch" in the same loop that issues `FlowV1FlowGet`/`FlowV1FlowDelete`
  calls, which meant the self-imposed pass-timeout bail-out (task 3) could stop the count short of
  `scanLimit` even when the batch itself was genuinely full — producing a false negative exactly
  in the overload case this signal exists to catch. Since the batch is already `DESC`-ordered by
  `tm_delete` and fully in memory, computing this is a pure O(n) timestamp scan requiring no RPCs —
  do it immediately after the query returns, before touching `bin-flow-manager` at all.

  Count how many returned rows have `tm_delete >= passStart - recentInterval` (see
  **`recentInterval`, a request parameter, design-round-4 fix** below for why this cannot be a
  compiled Go constant). If that count reaches `scanLimit` — meaning even restricting to just this
  interval's own new deletions already fills the entire batch — the actual safety condition
  (deletions-per-interval < `scanLimit`) **is being violated** (design-round-4 correction: drop
  "or approached" from the original wording — the derivation is exact: with `R = min(scanLimit,
  D)` where `D` is true deletions in the interval, `R == scanLimit` if and only if `D >=
  scanLimit`; there is no early-warning margin, `D = scanLimit - 1` reads as clean). This is the
  signal to act on: log a **warning** (not `info`/`debug`) and raise `scanLimit` and/or shorten the
  interval — unlike the plain-`Saturated` case, shortening the interval genuinely helps here,
  because it lowers deletions-per-interval directly (see the `recentInterval` note below for what
  else must change together with the interval). Document this response explicitly in the log
  message and in `docs/operations.md` (task 7).

  **Known limitation, disclosed rather than solved**: this signal assumes the previous pass ran
  exactly `recentInterval` ago. If a pass was skipped, delayed, or failed (per
  `bin-schedule-manager`'s own single-fire catch-up semantics — missed slots are not replayed), the
  true gap since the last successful pass can exceed `recentInterval`, and `RecentSaturated` can
  under-report in that scenario (it measures the wrong, too-short interval and may read `false`
  when the actual gap-since-last-pass already exceeds the safety condition). This is an accepted
  gap, not a claimed guarantee — `Saturated` and the audit trail in `bin-schedule-manager` remain
  the backstop for detecting an unhealthy cadence.

  **`recentInterval` must be a request parameter, not a hardcoded Go constant (design-round-4
  fix)**: `bin-campaign-manager` has no way to read the schedule's `cron` field from
  `bin-schedule-manager`'s own database — a compiled constant would silently drift from the real
  cron interval the moment someone changes it. Fixed: the seed migration's `target_data` carries
  `recent_interval_sec` alongside `cron` (task 8, same migration row **at seed time**), the new
  route accepts it as a request body field, and `ReconcileOrphanedFlows` takes it as an explicit
  parameter (task 3). If the field is missing or non-positive (e.g. an old/malformed dispatch),
  fall back to a conservative default (24 h) and log a warning — never panic or fail the pass over
  a malformed rate-signal input.

  **Design-round-5 correction — this does not eliminate post-seed drift, and the runbook must say
  so**: co-locating the two values in one migration row only protects the *initial* seed. A later
  operational change to `cron` (via `schedule-control` or `PUT /v1/schedules/{id}` — including the
  "shorten the interval" remedy this section itself recommends for a `RecentSaturated` alert) edits
  `schedule_schedules.cron` directly and does **not** touch `target_data.recent_interval_sec`,
  since they're edited through different operational paths once the schedule exists. **The runbook
  for both remedies (raising `scanLimit` and/or shortening the interval) must therefore
  explicitly instruct**: (1) update `target_data.recent_interval_sec` to match the new `cron` in
  the same operational change, and (2) re-verify `reconcilePassTimeout` (task 3) is still well
  below the *new*, shorter interval — `reconcilePassTimeout` is a compiled constant that does not
  move on its own, and a large-enough interval cut could violate the **cadence/liveness condition**
  in task 3's "self-imposed pass timeout" section (constraint (b) — design-round-7 correction: not
  a concurrency-safety condition; violating (b) alone causes chronic `skipped_overlap` cadence
  degradation, not a race, since mutual exclusion rests on constraint (a) alone) without either
  document's own checks catching it, since those checks only run once, at migration-review time.
  Document this as an explicit pre-flight step in `docs/operations.md` (task 7), not just a passing
  mention here.

**Why no cursor/watermark instead** (deliberate trade-off, stated explicitly so a future
maintainer doesn't "optimize" it away — design-round-3 correction to the retry-property claim
below): a cursor-based design (remember the last-processed `tm_delete` per pass, only fetch newer)
would eliminate the plain `Saturated` case entirely (no more re-fetching already-`Skipped` history),
but at a real cost — it would also permanently skip any candidate whose `FlowV1FlowDelete` call
failed on a past pass, since the cursor would have already advanced past it. The current no-cursor,
re-scan-the-whole-window-every-time design instead gives a failed cleanup automatic retry on every
subsequent pass **for as long as the candidate's rank stays within `scanLimit`** — not
unconditionally for its entire time in the window; full-window retry would require the stronger
per-window inequality (`scanLimit` > deletions per *window*) that the safety-condition section
above explicitly does not rely on. In practice, under the per-interval safety condition, a
newly-deleted or newly-failed candidate is retried on every pass for multiple intervals before its
rank could plausibly exceed `scanLimit`, which is worth the re-scan cost at this job's expected
scale (a background hygiene job, not a high-throughput queue) — and is why the
DESC-order-plus-two-signal fix was chosen over a cursor, not because a cursor wasn't considered.

## Tasks — `bin-campaign-manager`

1. **`pkg/dbhandler/campaign.go`**: add `CampaignListDeletedSince(ctx, since time.Time, limit
   uint64) ([]*campaign.Campaign, error)` — direct squirrel query,
   `WHERE tm_delete >= ? ORDER BY tm_delete DESC LIMIT ?`, mirroring this file's existing
   `CampaignList`'s error-wrapping style (no cache interaction, consistent with `CampaignList`
   itself being a fresh scan, not a single-entity cache-backed lookup). **Also add the method to
   the `DBHandler` interface declaration in `pkg/dbhandler/main.go`** (design-round-8 addition: an
   earlier draft only specified the concrete implementation — the interface is what task 6's tests
   mock, and what `//go:generate mockgen ... -source main.go` regenerates
   `mock_dbhandler.go` from; without this, task 3 won't compile against the interface and `go
   generate` won't produce a mock exposing the new method).
2. **`bin-campaign-manager/models/campaign`** (or a new small `models/reconcile` package, matching
   how `bin-timeline-manager`'s analogous multi-field result, `EventListResponse`, lives in a
   *domain* package under `models/`, not under `pkg/listenhandler/models`): add
   `ReconcileResult{Cleaned, Skipped, Failed int; Saturated, RecentSaturated, Partial bool}` (
   `RecentSaturated` is a design-round-3 addition, corrected in design-round-4 — see "Scan order
   and coverage" above for its exact meaning and the `recentInterval` request-parameter fix;
   `Partial` is a design-round-4 addition, set true when the self-imposed pass timeout cuts a pass
   short — see task 3) **with json tags** — this is a
   domain type that also serves as the wire shape (style A per this repo's layering rule), not a
   `response.*` DTO. Design-round-2 correction: an earlier draft proposed a separate
   `pkg/listenhandler/models/response.*` copy of this exact same shape, which the root CLAUDE.md's
   layering rule explicitly forbids ("Do NOT introduce a `response.*` DTO that is a field-for-field,
   identical-json-tag copy of a domain type — that is style (A) wearing a DTO costume"). Since no
   single existing domain type already equals this wire shape, `ReconcileResult` itself becomes
   that domain type (carrying json tags is allowed and is not the same as being a `response.*`
   DTO) — there is nothing to map into a separate struct.
3. **`pkg/campaignhandler`** (new file `reconcile.go`): add
   `ReconcileOrphanedFlows(ctx context.Context, recentIntervalSec int64) (result
   campaign.ReconcileResult, err error)` (or the equivalent type from wherever task 2 places it) —
   `recentIntervalSec` is a design-round-4 addition, threaded in from the request body by task 4.
   **Also add the method to the `CampaignHandler` interface declaration in
   `pkg/campaignhandler/main.go`** (design-round-8 addition, same reasoning as task 1: task 4's
   listenhandler calls this through the interface, and `//go:generate mockgen ... -source main.go`
   must regenerate `mock_campaignhandler.go` with the new method before task 6's listenhandler test
   can mock it). **Package-level constants (design-round-10 addition, for self-containment)**:
   declare `window time.Duration`, `scanLimit uint64`, and `reconcilePassTimeout time.Duration` as
   package constants in this file, sized per the pre-implementation check above; the body calls
   `CampaignListDeletedSince(ctx, passStart.Add(-window), scanLimit)` (task 1) as its first
   statement after the timeout wrap:
   - **Self-imposed pass timeout, required (design-round-3, corrected rounds 4/5, corrected again
     design-round-6)**: `ReconcileOrphanedFlows` must wrap its own body in `passStart := <time this
     call began>; ctx, cancel := context.WithTimeout(ctx, reconcilePassTimeout); defer cancel()`.
     **`reconcilePassTimeout` must satisfy two constraints that play genuinely different roles —
     design-round-6 correction: round 5 wrongly promoted both to co-equal "independent concurrency
     guards"; only (a) actually prevents concurrent execution, (b) is a cadence/liveness
     requirement**:
     - **(a) — the actual mutual-exclusion guarantee: `reconcilePassTimeout` strictly below the
       schedule's seeded `timeout_ms` (task 8), with margin that also absorbs message-delivery
       delay (design-round-6 addition — see below).** Tracing `bin-schedule-manager` source:
       `ExecutionHasRunning` ("Forbid overlap") is checked *before* the `tm_next_run` CAS on every
       claim path (cron and manual alike) and refuses to dispatch while a `running` row exists for
       this schedule. That row is marked `running` until `ExecutionComplete` fires, which happens
       as soon as `SendRequest`'s own `timeout_ms`-bounded wait returns (success or timeout) — **not**
       when this pass actually finishes in `bin-campaign-manager`. If `reconcilePassTimeout` (as
       measured from when `bin-campaign-manager` actually starts processing, not from when
       `bin-schedule-manager` sends the RPC — the two clocks do not start together; the margin
       below `timeout_ms` must absorb whatever message-delivery delay sits between them, not just
       clock skew) could exceed `timeout_ms`, there would be a real window — from `timeout_ms`
       until this pass's own timeout — where `bin-schedule-manager` believes the execution is no
       longer `running` (so `ExecutionHasRunning` no longer blocks anything) while this pass is
       still physically executing. A manual `/v1/schedules/{id}/execute` fired in that window would
       genuinely race a second RPC against the still-running first one. This is the only condition
       under which two calls to this route could ever execute concurrently, and it produces **no
       distinct alert** — the raced execution reads as an ordinary `success` row, exactly like a
       correctly-serialized one (design-round-6 finding: there is no monitorable signal for an
       (a)-violation; correct static sizing, with margin for delivery delay, is the only defense).
     - **(b) — a cadence/liveness requirement, not a concurrency guard (design-round-6
       correction)**: `reconcilePassTimeout` well below the schedule's `cron` interval. Because
       `ExecutionHasRunning` is checked *before* the `tm_next_run` CAS, a cron-triggered claim that
       arrives while the row is still `running` is short-circuited into an overlap skip — the CAS
       never runs, so `tm_next_run` is never advanced ("the slot is late, not lost", per
       `bin-schedule-manager`'s own documentation) — it does **not** create concurrent execution;
       given (a) holds, guard 1 alone already prevents that. What a chronically-overrunning pass
       actually produces is **cadence degradation**: every cron tick after the first observes the
       overlap skip, so the job effectively stops running on schedule at all. Constraint (b) exists
       to keep the schedule's cadence meaningful, not to prevent a race.
     **Runbook implication**: because `cron` is runtime-mutable data (edited via
     `schedule-control`/`PUT /v1/schedules/{id}`, no redeploy) while `reconcilePassTimeout` is a
     compiled Go constant, *any* operational change to the interval — including the
     `RecentSaturated` remedy below — must be checked against constraint (b) before applying it, to
     avoid reintroducing cadence degradation; see "Scan order and coverage" above. Constraint (a)
     is load-bearing independent of both, for a further reason: this service's
     `pkg/listenhandler/main.go` builds every RPC request's context with `context.Background()` —
     it does **not** propagate `timeout_ms` as a deadline — so without this internal bound, nothing
     stops a pass from running arbitrarily long in the first place. The `ctx.Err()` check below is
     only meaningful because of this self-imposed deadline. **Observable symptom, design-round-6
     correction — this is NOT an execution row or status; there is no `skipped_overlap` execution
     status (the enum is `running`/`success`/`failed`/`abandoned`)**: `skipped_overlap` is a
     **dispatch-level metric result label** (`schedule_manager_dispatch_total{result=
     "skipped_overlap"}`, emitted and counted *before* any execution row is ever created for that
     attempt — the claim path returns early on an overlap skip). This metric climbing is the
     observable symptom of a **constraint-(b)** violation (chronic cadence degradation) — it tells
     you nothing about an (a) violation, which produces a normal `success` row with no distinct
     signal at all. Document both facts explicitly in `docs/operations.md` (task 7): watch
     `schedule_manager_dispatch_total{result="skipped_overlap"}` for (b) health, and treat (a) as a
     static-sizing correctness property with no runtime monitor.
   - `err` is non-nil **only** when the initial `CampaignListDeletedSince` query itself fails (the
     pass could not run at all) — never for per-row outcomes, which are counted, not propagated.
   - **Compute `Saturated` and `RecentSaturated` in a dedicated, RPC-free pass over the fetched
     slice immediately after the query returns — design-round-4 fix, before any
     `FlowV1FlowGet`/`FlowV1FlowDelete` call, not interleaved with the RPC loop below.** An earlier
     draft computed `RecentSaturated` "while iterating the batch" in the same loop that issues
     RPCs, which meant the self-imposed pass-timeout bail-out could stop the count short of
     `scanLimit` even on a genuinely saturated batch — a false negative in exactly the overload
     case this signal exists to catch. Since the batch is already `DESC`-ordered and fully in
     memory, this is a pure O(n) timestamp scan, no RPCs involved:
     - `Saturated = uint64(len(candidates)) == scanLimit` — informational only (see "Scan order
       and coverage" above): the window holds more history than one pass returns, normal and
       expected at scale, not itself actionable. (Implementation nit, design-round-11: `len()`
       returns `int`; keep `scanLimit`'s declared type (task 3's package constants) in mind and
       cast explicitly rather than comparing mixed integer types, which doesn't compile in Go.)
     - `RecentSaturated`: count how many candidates have `tm_delete >= passStart -
       recentInterval` (`passStart` is the wall-clock time this call began, captured before the
       query runs; `recentInterval = time.Duration(recentIntervalSec) * time.Second`, falling back
       to a conservative 24h default with a logged warning if `recentIntervalSec <= 0`);
       `RecentSaturated = (uint64(thatCount) == scanLimit)`. See "Scan order and coverage" above
       for the exact meaning, the disclosed under-reporting limitation, and the runbook response.
   - For each candidate campaign (now iterating for the RPC loop proper), check `ctx.Err()` at the
     top of the loop iteration and bail out early if the context is done: set `Partial = true`
     (design-round-4 addition) and log at `warn` — a partial pass with `err == nil` would otherwise
     be silently indistinguishable from a complete one, which is what previously made a timed-out
     pass look like a full `success` in the audit trail — then return `(result, nil)` with the
     accumulated counts so far; the self-imposed pass timeout will cut a slow pass mid-iteration,
     so don't keep issuing RPCs against a dead context.
   - Defensive guard: `if c.TMDelete == nil { Skipped++; continue }` (design-round-5 fix: an
     earlier draft incremented no counter and logged nothing here, reproducing the exact
     counter-invariant gap design-round-4 fixed for the typed-not-found branch — corrected to log
     at `warn` and count it as `Skipped`, same reasoning: nothing needed doing for this row) even
     though the query already filters on `tm_delete >= since` (which excludes live campaigns via
     NULL-comparison semantics) — this is a second, cheap layer of defense against a future query
     change accidentally including live campaigns, given this job mutates data across all
     customers. The warn log (not just a silent `continue`) is what makes a future query regression
     visible instead of merely harmless.
   - `FlowV1FlowGet(campaign.FlowID)`. On typed not-found
     (`stderrors.As(err, &ve) && ve.Status == cerrors.StatusNotFound`), **`Skipped++`**
     (design-round-4 fix: an earlier draft incremented neither counter here, which meant
     `Cleaned+Skipped+Failed` didn't sum to candidates examined and this case had nothing
     assertable in a test — corrected to count it as `Skipped`, the same bucket as an
     already-cleaned flow, since a genuinely absent flow row is an equally legitimate clean end
     state). On any other error, log + increment `campaign_flow_reconcile_failed_total` (task 5) +
     `Failed++` (design-round-8 fix: an earlier draft incremented `Failed` here but **not** the
     metric, leaving the metric defined — task 5 and the design doc — to cover this branch while
     no instruction actually wired it up; this is the systemically important branch to alert on,
     since a `bin-flow-manager` outage or an open circuit breaker on the shared RPC client makes
     every remaining candidate in the pass fail here, and under the previous, unmetriced version
     the counter would read zero while the pass failed on every single row). On success: if
     `f.TMDelete != nil`, `Skipped++` (already clean, not an error). Otherwise, it's a genuine
     orphan: call `FlowV1FlowDelete(campaign.FlowID)`; on error, log + increment
     `campaign_flow_reconcile_failed_total` (task 5) + `Failed++`; on success, `Cleaned++` +
     increment `campaign_flow_reconcile_cleaned_total`. **Both error branches share the same
     `campaign_flow_reconcile_failed_total` counter** (no `reason` label distinguishing
     `FlowV1FlowGet` failures from `FlowV1FlowDelete` failures) — call this out explicitly in
     `docs/operations.md` (task 7) so a reader of the metric knows it conflates the two RPCs.
   - Window-edge warning (per-candidate, orthogonal to the saturation signals): for each orphan
     found whose owning campaign's `tm_delete` is within [some margin, e.g. 10%] of the window's
     outer edge, **log** a warning noting the window may need widening (design-round-4 fix: an
     earlier draft said "log/metric" here — no counter is defined for this, matching the
     no-third-counter decision for `Saturated`/`RecentSaturated`).
4. **`pkg/listenhandler`**: add route `POST /v1/campaigns/flows/reconcile` (new regex + dispatch
   case in `main.go`, new handler file e.g. `flows.go`). Request body: `{"recent_interval_sec":
   int64}` — a new, small `request.*` DTO under this service's existing
   `pkg/listenhandler/models/request` subpackage (this is the transport-DTO-for-inputs case the
   layering rule expects listenhandler to own — not a violation of the task-2 domain-type
   decision, which is about the *response* shape). The handler unmarshals it, calls
   `campaignHandler.ReconcileOrphanedFlows(ctx, req.RecentIntervalSec)` with the unwrapped scalar
   (not the request struct itself, per this repo's "unwrapped parameters, not a wrapped request
   struct" convention), and `json.Marshal`s the returned `ReconcileResult` **directly** — style
   (A), per the correction in task 2, matching every other route in this service (this service's
   `pkg/listenhandler/models/` gains only this one small `request.*` addition; still no
   `response/` subpackage). HTTP status: 200 even if `Failed > 0` or `Partial == true` (the pass
   ran; individual row failures and a self-timeout cutoff are data, not a pass-level error) — a
   non-nil `err` from the handler (query-level failure) is the only case that should produce a
   non-2xx response. Internal-only route, not exposed through `bin-api-manager` (no OpenAPI spec
   change).
5. **`pkg/campaignhandler/main.go`**: add two new plain `Counter`s (not a reuse of VOIP-1443's
   `promCampaignFlowDeleteFailedTotal` — that metric is documented as "failures during campaign
   delete" and reusing it here would collapse two operationally distinct signals: "a live delete
   is failing right now" vs. "the reconciler couldn't clean up a known-orphaned flow" — into one
   series and make that existing doc line inaccurate):
   `campaign_flow_reconcile_cleaned_total` and `campaign_flow_reconcile_failed_total`, following
   the existing `init()` pattern. No third counter for `Saturated`/`RecentSaturated`
   (design-round-3 clarification): that signal is surfaced only via the log warning and the
   `ReconcileResult` response field, kept out of Prometheus since it is informational far more
   often than actionable (see "Scan order and coverage" above).
6. **Tests**: `reconcile_test.go` covering:
   - Genuine orphan (flow `TMDelete` nil) found and cleaned — `Cleaned` increments, metric fires.
   - Already-clean flow (`TMDelete` set) skipped — `Skipped` increments, no `FlowV1FlowDelete`
     call, no metric.
   - `FlowV1FlowGet` typed not-found — **`Skipped` increments** (design-round-4 fix: an earlier
     draft counted this case nowhere, so `Cleaned+Skipped+Failed` didn't sum to candidates
     examined; now the same bucket as an already-clean flow), not counted as `Failed`.
   - A transient `FlowV1FlowGet` error — `Failed` increments, logged, not counted as `Cleaned`,
     **and `campaign_flow_reconcile_failed_total` fires** (design-round-8 fix: an earlier draft
     asserted only the in-memory `Failed` counter here, matching a task-3 draft that incremented
     `Failed` without the metric — corrected so this, the systemically important failure branch, is
     actually observable via Prometheus, not just in the response body and logs).
   - `FlowV1FlowDelete` failure on a genuine orphan — `Failed` increments (not `Cleaned`), and
     `campaign_flow_reconcile_failed_total` fires (same counter as the `FlowV1FlowGet`-error case
     above — no `reason` label distinguishes them).
   - A campaign with `TMDelete == nil` fed directly into the handler (bypassing the query) —
     assert `Skipped` increments (design-round-5 fix: an earlier draft asserted only "no RPCs
     called", which is not assertable as a positive outcome — now the same counter-invariant fix
     applied to the typed-not-found case above) and `FlowV1FlowDelete`/`FlowV1FlowGet` are never
     called (the defense-in-depth guard, tested in isolation from the query's own filtering), and
     that a `warn`-level log line fires (this guard existing at all means a future query regression
     should be loud, not silent).
   - `scanLimit`-sized result set — assert `Saturated == true`; a smaller result set — assert
     `Saturated == false`.
   - `scanLimit`-sized result set where all rows fall within `recentInterval` — assert
     `RecentSaturated == true`; a `scanLimit`-sized result set where most rows are older than
     `recentInterval` — assert `Saturated == true` but `RecentSaturated == false` (this is the case
     that distinguishes the two signals).
   - **Design-round-4 addition, corrected design-round-5**: `Saturated`/`RecentSaturated` are
     computed correctly even when the self-imposed pass timeout fires during the RPC loop (a slow
     fake RPC client on, say, the second candidate) — asserts the two-signal computation genuinely
     happened in its own pre-loop pass over the fetched slice before any RPC was issued, not
     interleaved with the RPC loop (the exact false-negative this task's earlier draft would have
     produced). Design-round-5 correction: do not test this via a 0-duration `reconcilePassTimeout`
     — that would expire before `CampaignListDeletedSince` itself returns, so the pass would fail
     at `err != nil` with no `ReconcileResult` to assert against, testing nothing about the
     ordering property; use a `reconcilePassTimeout` long enough for the query to return but short
     enough to expire partway through the RPC loop.
   - **Design-round-4 addition**: `recentIntervalSec <= 0` (missing/malformed request field) falls
     back to the conservative default and logs a warning, does not panic or fail the pass.
   - Self-imposed pass timeout fires mid-pass (inject a short `reconcilePassTimeout` or a slow fake
     RPC client) — assert early return with the partial counts accumulated so far, `Partial ==
     true` (design-round-4 addition), `err == nil` (this exercises the handler's own
     `context.WithTimeout`, not the RPC-layer context, since the RPC layer does not propagate a
     deadline).
   - Empty candidate set — `{0,0,0,false,false,false}, nil` (design-round-4: now six `ReconcileResult`
     fields including `Partial`), zero RPCs issued.
   - Idempotent re-run: running the same pass twice against the same fixture data cleans nothing
     and deletes nothing the second time.
   - Window-edge warning fires for a near-cutoff candidate.
   - A `dbhandler` test for `CampaignListDeletedSince` against this package's real SQLite
     test-schema harness, confirming the date-range filter, the `DESC` order, and the
     NULL-comparison exclusion of live campaigns.
   - A basic listenhandler test for the new route (request parses, calls through, the marshaled
     `ReconcileResult` shape is correct, 200 status even with `Failed > 0`).
7. **Docs**: `docs/operations.md` — two new metrics, new route, the `Saturated`/`RecentSaturated`
   runbook note from "Scan order and coverage" above, **and**: (i) design-round-6 correction of a
   design-round-5 addition — there is no `skipped_overlap` execution status (the execution status
   enum is `running`/`success`/`failed`/`abandoned`); the real observable signal is the
   **dispatch-level metric** `schedule_manager_dispatch_total{result="skipped_overlap"}` climbing
   for this schedule, which indicates a **constraint-(b)** violation (cadence degradation — see
   task 3) and calls for re-tuning `reconcilePassTimeout` against real pass durations; explicitly
   document that this metric does **not** detect a constraint-(a) violation (the actual
   concurrency-safety condition), which produces an indistinguishable, ordinary `success` row —
   (a) is a static-sizing correctness property with no runtime monitor, not something to watch a
   dashboard for; (ii) any operational change to the schedule's `cron` interval (including the
   `RecentSaturated` remedy) MUST update `target_data.recent_interval_sec` to match in the same
   change, and must re-verify `reconcilePassTimeout` is still well below the new interval before
   applying it (constraint (b), to avoid reintroducing cadence degradation); (iii) design-round-10
   addition — the `RecentSaturated` remedy's two options are not equally fast: raising `scanLimit`
   is a compiled constant and needs a code change + redeploy, while shortening `cron`/updating
   `recent_interval_sec` is a live data edit with no redeploy — state this explicitly so an
   operator under saturation pressure reaches for the immediate remedy first; (iv) design-round-10
   addition — `Saturated`/`RecentSaturated`/`Partial` are not merely logged and dropped: they land
   in the response body, which `bin-schedule-manager` persists verbatim as
   `schedule_executions.result` (`models/execution/execution.go`'s `Result string` field) on every
   pass that returns a response (design-round-11 precision fix: every HTTP 200, including
   `Failed > 0`/`Partial == true` — `result` is populated only inside `ExecutionComplete`'s success
   branch, so a transport-level failure that never reaches this route leaves `result` empty and
   `error` populated instead; that case has no `ReconcileResult` to persist regardless) — this is
   what makes the no-third-counter decision safe despite
   `bin-campaign-manager`'s own logs "rotating within hours" (per VOIP-1443's investigation): the
   durable record survives in `bin-schedule-manager`'s own audit trail even after local logs
   rotate. `docs/architecture.md` (routing-table entry for the new route), and — design-round-3
   addition, missed in the prior revision — `docs/domain.md` (the new `ReconcileResult` type added
   under `models/` in task 2), per this service's own service-docs-sync convention and the root
   CLAUDE.md's `models/.../*.go` → `docs/domain.md` mapping.

## Tasks — `bin-dbscheme-manager` (same PR, different directory — not a separate PR)

8. **New Alembic migration**: seed the `campaign-flow-reconcile` schedule row into
   `schedule_schedules`. Primary model: `0c037bf0a362_schedule_schedules_seed_phase2_ticker_...py`
   (seeds five schedules dispatching into *other* services' request queues — the closer precedent
   for this ticket's cross-service dispatch shape than the self-RPC housekeeping jobs in
   `a5e6f559299c`). Set every column both precedent migrations set: `id` (fresh UUID),
   `customer_id` (nil-UUID platform-job convention), `name: 'campaign-flow-reconcile'`, `detail`,
   `type: 'rpc'`, `cron` (from the pre-implementation check, e.g. `'0 */6 * * *'` — every 6 hours,
   adjust per the rate finding), `target_queue: 'bin-manager.campaign-manager.request'`,
   `target_uri: '/v1/campaigns/flows/reconcile'`, `target_method: 'POST'`, `target_data_type:
   'application/json'`, **`target_data: JSON_OBJECT('recent_interval_sec', N)` where `N` is the
   `cron` field's interval in seconds (design-round-4 addition — e.g. `21600` for the `'0 */6 * *
   *'` example above)** — not an empty `JSON_OBJECT()`: `bin-campaign-manager` has no way to read
   `cron` from `bin-schedule-manager`'s own database, so the interval must be handed to it
   explicitly per request; seeding both in this same migration row means the *initial* values can
   only be edited apart by missing a line in the same diff (design-round-5 clarification: this
   applies at seed time only — see the docstring note below and "Scan order and coverage" above for
   the post-seed runtime-drift risk this does *not* eliminate), `timeout_ms` (design-round-6
   correction of a design-round-5 over-correction: `reconcilePassTimeout` (task 3) strictly below
   `timeout_ms` is the actual mutual-exclusion guarantee — it keeps `bin-schedule-manager`'s "Forbid
   overlap" guard (`ExecutionHasRunning`, checked before the `tm_next_run` CAS on every claim path)
   effective for this pass's entire physical duration; being well below `cron`'s interval is a
   separate, secondary cadence requirement (task 3), not an independent concurrency guard — see
   task 3's full reasoning for why. Size with margin that covers real message-delivery delay
   between `bin-schedule-manager` sending the RPC and `bin-campaign-manager` starting to process
   it, not just RPC latency — e.g. `reconcilePassTimeout = 90s` → `timeout_ms: 120000` as a
   starting estimate, both values confirmed against real per-row
   `FlowV1FlowGet`/`FlowV1FlowDelete` latency, real queue delivery delay, and comfortably below the
   6-hour `cron` interval), `retry_max: 0` (a failed pass is naturally retried by the next
   scheduled run — no need for schedule-level retry), **`enabled: 0`** (rollout ordering, see task
   10 — not `1`, correcting an earlier draft), `tm_next_run: NULL`, `tm_create`/`tm_update:
   UTC_TIMESTAMP(6)`. Include, in the migration's docstring: (a) why `target_data` carries
   `recent_interval_sec` as a bound SQL parameter via `JSON_OBJECT(...)` rather than a raw JSON
   string literal like `'{"recent_interval_sec": 21600}'` — `op.execute()` passes raw SQL through
   `sqlalchemy.text()`, which would parse the literal's bare colon as a bind-parameter marker, per
   `0c037bf0a362`'s own documented reasoning; and that `recent_interval_sec` must be kept in sync
   with `cron` by hand whenever either changes **after** this migration is applied — a post-seed
   edit to `cron` (via `schedule-control` or `PUT /v1/schedules/{id}`) does **not** automatically
   update `target_data`, since the two are edited through entirely different operational paths once
   the schedule exists (design-round-5 correction: an earlier draft of this note contradicted
   itself by also implying they're safely co-located forever because they started in one migration
   row — that co-location protects only the initial seed, not later edits); (b) why this row ships
   `enabled: 0` unlike all five rows in the `0c037bf0a362` precedent (which ship `enabled: 1`) —
   the rollout-ordering reason in task 10 — so a future reader doesn't "fix" it back to `1`.
   - Cross-check (design-round-6 correction of design-round-5's framing — round 4 wrongly dropped
     the reaper-abandonment-deadline comparison as "an unfalsifiable tautology"; round 5 reinstated
     it correctly but attributed the crash scenario to the wrong service; both fixed below):
     1. `reconcilePassTimeout` (task 3) strictly below `timeout_ms`, with margin covering real
        message-delivery delay (not just processing latency) — the actual mutual-exclusion
        guarantee: keeps `bin-schedule-manager`'s "Forbid overlap" (`ExecutionHasRunning`) guard
        covering this pass's entire actual execution. This has no runtime monitor — get the sizing
        right, since a violation produces an ordinary `success` row with no distinguishing signal.
     2. `reconcilePassTimeout` well below the `cron` interval above — a cadence/liveness
        requirement (task 3), not a concurrency guard: keeps the schedule from chronically
        observing dispatch-level `skipped_overlap` skips (see task 3) rather than actually running
        on its intended cadence.
     3. `timeout_ms + 60s` (the reap deadline at `retry_max: 0`) comfortably below the `cron`
        interval — bounds how long a **`bin-schedule-manager` dispatch goroutine itself crashing
        mid-wait** (its documented "abandon-not-drain" shutdown case, not a
        `bin-campaign-manager` crash — design-round-6 correction: an earlier draft of this
        cross-check attributed the crash to the wrong service) can leave this schedule's execution
        row stuck `running` — and hence the schedule stuck observing overlap skips — before the
        reaper reclaims it; at `timeout_ms: 120000` vs. a 6-hour `cron`, this holds with wide
        margin, but re-verify it whenever either constant changes.
9. **`tm_delete` index migration** (include per the plan's conservative default, or confirmed
   necessary by the pre-implementation check): add an index on `campaign_campaigns.tm_delete`, and
   update the corresponding test-schema fixture
   (`bin-campaign-manager/scripts/database_scripts/table_campaigns.sql`) in the same commit, per
   `bin-dbscheme-manager/CLAUDE.md`'s migration+fixture-sync rule.
10. **Rollout sequencing (closes the dispatch-before-route-exists ordering hazard)**: the seed
    migration ships with **`enabled: 0`**. After this PR merges and both the migration is applied
    (by 대표님/ops, per the Alembic AI-authorship boundary below) and the `bin-campaign-manager`
    image carrying the new route is deployed: (a) fire a manual smoke test via
    `POST /v1/schedules/{id}/execute` — **design-round-8 addition, how to actually issue this
    call**: `bin-schedule-manager/cmd/schedule-control` has no `execute` subcommand (only
    `list`/`get`/`enable`/`disable` for schedules); the only path is
    `requesthandler.ScheduleV1ScheduleExecute` (a Go RPC client call) or publishing the equivalent
    RPC directly onto `bin-manager.schedule-manager.request` — the same idiom
    `bin-schedule-manager/docs/operations.md` already documents for its own backup runbook, not a
    new invention. First resolve the schedule's `{id}` (a generated UUID, unknown at
    migration-authoring time) via `schedule-control schedule get campaign-flow-reconcile`. This
    call (works on disabled schedules, per `bin-schedule-manager`'s own documentation, and never
    consumes the cron slot; it dispatches asynchronously and returns the execution row in
    `running` status — design-round-4 fix: poll the execution row rather than expecting a terminal
    status in the response) — confirm the resulting execution row settles
    to status **`success`** (design-round-4 correction: the status enum value is `success`, not
    `succeeded` — an earlier draft used the event name instead of the status value) and the new
    counters moved. **Design-round-7 addition**: if this call instead returns a
    `FailedPrecondition` `EXECUTION_IN_PROGRESS` error, it means a prior execution for this
    schedule is still `running` (the manual path's own overlap guard — distinct from cron's
    `skipped_overlap` metric, and not itself a metric event) — simply wait for it to finish and
    retry, not a failure of this rollout step; (b) only then enable the schedule via
    `schedule-control schedule enable campaign-flow-reconcile` (no redeploy, no second migration
    needed for either step). This is a post-deploy operational sequence, not part of the PR's
    code — call it out explicitly in the PR description so it isn't missed. Symmetric rollback:
    `schedule-control schedule disable campaign-flow-reconcile` if the job needs to be paused for
    any reason, no redeploy needed either direction.
11. **AI-authorship boundary**: create and commit the migration file(s) via `alembic revision`; do
    **not** run `alembic upgrade`/`downgrade` — that requires human authorization per this repo's
    Alembic rule. The enable step in task 10 is likewise a human/ops action, not something this
    session performs.

## Acceptance criteria

- A campaign deleted more than `window` days ago is never touched by this job (out of scope by
  design, not a bug).
- A campaign deleted within the window whose flow was already correctly cleaned up (by
  VOIP-1443's `Delete()` best-effort path succeeding) is left alone — no re-delete, counted as
  `Skipped` in the response only (no dedicated Prometheus counter exists for skips — see task 5).
- A campaign deleted within the window whose flow is still live gets that flow deleted, and
  `campaign_flow_reconcile_cleaned_total` increments.
- A row-level failure on *either* `FlowV1FlowGet` (non-not-found) or `FlowV1FlowDelete` increments
  `campaign_flow_reconcile_failed_total` (design-round-9 addition, pinning the round-8 lesson here
  explicitly: an earlier draft only wired this metric to the `FlowV1FlowDelete` branch, leaving it
  silent during exactly the `bin-flow-manager`-outage scenario it exists to catch) — the one
  counter conflates both RPCs, with no `reason` label distinguishing which one failed.
- A pass that reaches its `scanLimit` is visibly flagged (`Saturated` in the response, log +
  response `RecentSaturated` when the more precise rate-risk condition is met), not silently
  truncated — but plain `Saturated` alone is informational, not itself a defect (see "Scan order
  and coverage").
- A reconciliation pass's own execution is bounded by `ReconcileOrphanedFlows`'s self-imposed
  `context.WithTimeout` (task 3), satisfying two constraints that play different roles —
  design-round-6 correction of design-round-5's framing (design-round-7 fix: standardizing on
  constraint labels (a)/(b) throughout both documents, not a second, conflicting (1)/(2) scheme):
  they are not co-equal "independent concurrency guards"; only one of them prevents concurrent
  execution: **(a)** strictly below the schedule's `timeout_ms`, with margin covering real
  message-delivery delay — this is the actual mutual-exclusion guarantee, keeping
  `bin-schedule-manager`'s "Forbid overlap" (`ExecutionHasRunning`) guard effective for the pass's
  entire physical duration; without it, a manual execute could race a genuinely still-running pass
  once `bin-schedule-manager`'s own RPC wait gives up, and this violation produces an ordinary
  `success` row with **no distinguishing signal** — correct sizing is the only defense; and **(b)**
  well below the schedule's `cron` interval — a cadence/liveness requirement, not a concurrency
  guard, since `ExecutionHasRunning` is checked before the `tm_next_run` CAS on the **cron** claim
  path and a cron claim arriving mid-pass is short-circuited into a dispatch-level
  `skipped_overlap` metric result, not a second execution (design-round-7 correction: this applies
  to the cron path only — a **manual** `/v1/schedules/{id}/execute` that hits the same overlap
  guard does not increment this metric at all; it returns a `FailedPrecondition`
  `EXECUTION_IN_PROGRESS` error instead, since `ScheduleClaimAndCreateExecution` skips the CAS
  entirely for manual triggers). The `schedule_manager_dispatch_total{result="skipped_overlap"}`
  metric climbing is the observable symptom of a constraint-(b) violation (chronic cadence
  degradation) specifically, on the cron path — it says nothing about constraint (a).
- A pass cut short by its own self-imposed timeout sets `Partial = true` in the response and logs
  a warning — it is never silently recorded as a full `success` with no trace it was incomplete.
- The new route can only ever be reached via the seeded schedule (once enabled) or a manual
  `/v1/schedules/{id}/execute` RPC (design-review-round-9 fix: not `schedule-control`, which per
  task 10 has no `execute` subcommand and cannot fire this route at all — only
  `list`/`get`/`enable`/`disable`) — no `bin-api-manager` exposure.
- The schedule ships disabled; enabling it is an explicit, documented post-deploy step, not
  automatic on merge or on migration apply.
- Before the schedule is ever enabled, a manual `POST /v1/schedules/{id}/execute` smoke test has
  been fired and its execution row polled to a terminal **`success`** status with the new counters
  moved — enabling the cron trigger is not the first time this route is exercised in production.
- The full repo-mandated verification workflow (`go mod tidy && go mod vendor && go generate ./...
  && go test ./... && golangci-lint run`, per root CLAUDE.md's "CRITICAL: Verification before
  commit" — design-round-8 fix: an earlier draft of this criterion and the verify tasks below
  named only `go test`/`golangci-lint run`, a 2-of-5 subset; `go generate` in particular is
  load-bearing here, not boilerplate — see tasks 1/3) passes in `bin-campaign-manager`; the new
  Alembic migration is created (not applied) and reviewed for column completeness (all columns
  explicitly set, `timeout_ms > 0`, `enabled: 0`, `target_data.recent_interval_sec` matching
  `cron`, and — design-round-6 addition, previously covered only by the verify task, not this
  criterion — `timeout_ms + 60s` comfortably below `cron`) against this plan's task 8's full
  three-part cross-check.

## Verify tasks

- [ ] `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v
      --timeout 5m` passes in `bin-campaign-manager`, in that order, per root CLAUDE.md's mandatory
      5-step workflow (design-round-8 fix: an earlier draft of this checklist covered only
      `go test`/`golangci-lint`, skipping `go mod tidy`/`vendor` — go.sum drift risk for any new
      import — and `go generate`, which is required here specifically: `mockgen` regenerates
      `mock_dbhandler.go`/`mock_campaignhandler.go` from the `//go:generate` directives in
      `pkg/dbhandler/main.go`/`pkg/campaignhandler/main.go`, and task 6's tests cannot compile
      against the new `CampaignListDeletedSince`/`ReconcileOrphanedFlows` interface methods until
      the mocks are regenerated), including all reconciliation-specific cases in task 6
- [ ] Manual review of the seed migration's column list against task 8's checklist, specifically
      confirming all three constraints in task 8's cross-check: `reconcilePassTimeout` (task 3) is
      strictly below `timeout_ms` with margin for delivery delay, well below the seeded `cron`
      interval, and that `timeout_ms + 60s` (the reap deadline at `retry_max: 0`) is comfortably
      below `cron`; and that `target_data.recent_interval_sec` matches `cron` (design-round-6: the
      reaper-formula comparison was dropped in round 4, reinstated in round 5, and correctly
      re-attributed in round 6 to a `bin-schedule-manager` — not `bin-campaign-manager` — crash
      scenario; it is not the tautology round 4 believed it was)
- [ ] Confirm `docs/operations.md` documents (design-round-6 correction: there is no
      `skipped_overlap` execution status) the `schedule_manager_dispatch_total{result=
      "skipped_overlap"}` **metric** as the observable symptom of a constraint-(b)
      (cadence/liveness) violation only, **on the cron path** (design-round-7: a manual execute
      hitting the same overlap guard returns `EXECUTION_IN_PROGRESS` instead and increments no
      metric — call this out for task 10's smoke-test step, which is itself a manual execute and
      can legitimately hit this if a prior run is still in flight) — explicitly noting the metric
      does NOT detect a constraint-(a) (mutual-exclusion) violation, which is a static-sizing
      property with no runtime monitor — and that the counter's granularity is per-replica,
      in-memory dedup (design-round-7 addition: a single stuck cron slot can increment it more than
      once across replicas, so "climbing" should be read as a health signal, not an exact skip
      count) — and the recent-interval-change runbook step (update `target_data.recent_interval_sec`
      and re-verify `reconcilePassTimeout` together whenever `cron` changes)
- [ ] `git diff` review confirms no unrelated changes, and that both `bin-campaign-manager` and
      `bin-dbscheme-manager` changes are in the same PR
- [ ] Confirm the new route does not appear in any `bin-api-manager`/OpenAPI surface
- [ ] Confirm the PR description explicitly calls out the post-merge, post-deploy sequence: apply
      migration → deploy → manual `/v1/schedules/{id}/execute` smoke test → poll the execution row
      to `success` → `schedule-control schedule enable campaign-flow-reconcile`
