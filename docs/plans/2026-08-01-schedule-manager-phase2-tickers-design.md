# bin-schedule-manager Phase 2 — Migrate in-process tickers into schedule definitions (VOIP-1284)

Status: approved (design review loop: 3 rounds — RC, Approve, Approve)
Ticket: VOIP-1284 — bin-schedule-manager Phase 2
Depends on: VOIP-1283 (bin-schedule-manager Phase 1, merged: PR #1154 + hotfix PR #1155)
Related: VOIP-1282 (campaign-manager / queue-manager investigation — concluded neither needs
scheduler-manager; see §2)

## 1. Problem

Five `time.Ticker`-based periodic jobs run in-process inside three services, with no leader
election, no distributed lock, and no claim mechanism:

| # | Service | Job | Cadence | File:line (2026-08-01) |
|---|---|---|---|---|
| 1 | billing-manager | monthly token top-up | 1h | `cmd/billing-manager/main.go:144-158` |
| 2 | billing-manager | failed-event retry | 60s | `cmd/billing-manager/main.go:205-219` |
| 3 | ai-manager | expired-proposal sweep | 1h | `cmd/ai-manager/main.go:165-177` |
| 4 | customer-manager | unverified-customer cleanup | 15m | `cmd/customer-manager/main.go:100-101` → `pkg/customerhandler/cleanup.go` |
| 5 | customer-manager | frozen-customer expiry | 24h | `cmd/customer-manager/main.go:103-104` → `pkg/customerhandler/expiry.go` |

**This is not a theoretical risk.** All three services already run `replicas: 2` in production
Kubernetes today (`bin-billing-manager/k8s/deployment.yml:8`, `bin-ai-manager/k8s/deployment.yml:8`,
`bin-customer-manager/k8s/deployment.yml:8`). Every hazard below is live.

Two further structural problems independent of replica count:

- **`time.Ticker` fires relative to pod start, not wall-clock.** The 24h customer-frozen-expiry
  ticker only runs 24h after the pod started; a service redeployed more often than daily never
  runs it. The 1h jobs have the same drift, just less severe.
- **Shutdown is broken on 4 of 5.** Only the two billing-manager tickers are wired to `chDone`
  (`main.go:154`, `main.go:215`). The ai-manager and both customer-manager tickers select on
  `context.Background().Done()`, which never fires — dead code, and the goroutine dies mid-tick
  when the process exits rather than draining cleanly.

Sandbox already carries a "replica guard" (`voipbin-cli.py:2266-2311`) that warns operators not
to scale billing-manager/ai-manager above 1 replica — an acknowledgment that this problem exists,
worked around by refusing to exercise the failure mode rather than fixing it. The guard doesn't
even cover customer-manager (undiscovered when it was written) and is silently violated by every
K8s `replicas: 2` deployment.

## 2. VOIP-1282 scope check (why campaign/queue are NOT here)

VOIP-1284's description flagged VOIP-1282 findings as a possible fold-in. Investigation (VOIP-1282,
concluded before this design) found:

- **campaign-manager** dial loop and **queue-manager** wait/service timeouts are both already
  self-driving via one-shot RabbitMQ `x-delayed-message` self-RPC (500ms–5000ms campaign re-arm;
  per-call timeout arm at queue-join/service-start). Neither is broken, dormant, or waiting on an
  external scheduler — their CLAUDE.md/README documentation describing a dependency on "an external
  scheduler" is simply stale (fixed separately, doc-only).
- Both are **fine-grained, per-entity, sub-minute-precision, one-shot** timers. bin-schedule-manager
  is a 5-field UTC cron engine with MySQL rows and DB-CAS claims per slot — a poor fit for
  thousands of concurrent one-shot timers, and pushing either loop through it would be a latency
  and correctness regression, not an improvement.
- Real bugs were found in both (campaign-manager silently swallows re-arm errors at two of four
  call sites, leaving the loop capable of permanent silent death; queue-manager timeouts are lost
  when a call is parked in `connecting` state, and the delayed publish is `amqp.Transient` with no
  delivery confirmation). These are tracked as separate follow-up tickets, not folded into this
  design — they are bug fixes in campaign-manager/queue-manager's existing delayed-RPC mechanism,
  not schedule-manager work.

**Conclusion: VOIP-1284's scope is exactly the five tickers in §1. Nothing from VOIP-1282 is
in scope here.**

## 3. Goals / non-goals

**Goals:**

- All five jobs dispatch through bin-schedule-manager: at-most-once claim, Forbid overlap, audit
  trail, per-job Prometheus metrics — the same guarantees Phase 1 gave `number-renew`.
- Fix the two genuine correctness bugs found during exploration: `AccountTopUpTokens` (§5.1) and
  `CustomerAnonymizePII` (§5.5) are not safe under concurrent execution — both must be fixed as
  part of this migration, since moving an unsafe job onto infrastructure whose entire selling
  point is "safe under N replicas" without fixing the job itself would be a lie of omission.
- Remove the five ticker goroutines and their dead/broken shutdown wiring.
- Remove (or correct) the sandbox replica-guard warning for billing-manager/ai-manager, sequenced
  after the monorepo fixes are released and pinned (§9) — and correct the guard's premise: it was
  never accurate to omit customer-manager. One of the two customer-manager jobs (frozen expiry,
  §5.5) has exactly the same class of double-fire hazard as billing's top-up — a duplicate
  `customer_deleted` cascade into number-manager and billing-manager — and was live at
  `replicas: 2` the whole time the guard existed without covering it. The other customer job
  (unverified cleanup, §5.4) is structurally idempotent and was never at risk, same as the ai
  proposal sweep (§5.3).

**Non-goals:**

- campaign-manager / queue-manager (§2 — explicitly out of scope, tracked separately).
- route-manager SIP OPTIONS health check, pipecat-manager tool-cache refresh, timeline flush,
  transport keepalives — per-instance/per-pod semantics, already evaluated and rejected for
  centralization in the Phase 1 design (`2026-08-01-bin-schedule-manager-design.md` §2/§9.3).
- The two ai-manager **startup** one-shot sweeps (`SweepStaleAudits`, `SweepStaleProposals`) —
  these recover state orphaned by *this pod's own* crashed goroutines; their entire premise is
  "ran once, by this process, right after it started." Centralizing them would fire on a pod that
  never owned the orphaned rows, at a cadence unrelated to restarts. Stays per-boot, unchanged.
  (Direct analogue of the route-manager decision above — same reasoning, different service.)
- `bin-common-handler` changes: none needed. All five target queues
  (`QueueNameBillingRequest`, `QueueNameAIRequest`, `QueueNameCustomerRequest`) are already in
  `QueueNameRequestAll()` from Phase 1. No new typed client methods are required — schedule rows
  dispatch generically via `target_queue`/`target_uri`, exactly like `number-renew`. This keeps
  the 38-service verification blast radius at zero, same as Phase 1's design intent.
- Two bugs found but out of scope for this ticket (filed separately, see §8): the inert
  `idempotency_key`/`reference_id` mismatch in the billing ledger (harmless once the top-up itself
  is made idempotent — the ledger will simply stop producing duplicate rows, so the mismatch
  becomes moot for this job, but the underlying dead code is worth cleaning up on its own); the
  unbounded `CustomerListFrozenExpired` query (no `LIMIT`, unlike every other job in this batch).

## 4. New endpoints (per service, following the Phase 1 `number-renew` / `/v1/executions/prune` pattern)

All five follow the established sweep-endpoint shape: POST, empty or trivial JSON body, no path
parameters, business handler returns `(count int, err error)` instead of today's `error`-only or
`void` signature, listenhandler marshals a small response DTO (style B — bare scalars, no domain
type equivalent, same precedent as `V1ResponseExecutionsPrune`).

| Service | Route | Business change | Response |
|---|---|---|---|
| billing-manager | `POST /v1/accounts/top_up` | `runMonthlyTopUp` moves from `cmd/billing-manager/main.go` into `pkg/accounthandler` (or `pkg/billinghandler`, whichever owns `Account`), returns `(processed, failed int, err error)`. **Also fixes the CAS bug, §5.1.** | `{"processed": n, "failed": n}` |
| billing-manager | `POST /v1/failed_events/retry` | `failedeventhandler.RetryPending` changes signature to return `(retried, succeeded, exhausted int, err error)` | `{"retried": n, "succeeded": n, "exhausted": n}` |
| ai-manager | `POST /v1/aipromptproposals/expire` | `aipromptproposalhandler.SweepExpiredProposals` changes signature to return `(expired int, err error)` | `{"expired": n}` |
| customer-manager | `POST /v1/customers/cleanup_unverified` | `customerhandler.cleanupUnverified` exported (or a thin exported wrapper added) returning `(expired int, err error)` | `{"expired": n}` |
| customer-manager | `POST /v1/customers/cleanup_frozen_expired` | `customerhandler.cleanupFrozenExpired` exported, returns `(processed int, err error)`. **Fixes the double-publish bug, §5.5.** | `{"processed": n}` |

Route regex, dispatch-switch placement, and request/response DTO naming follow each service's
own local convention exactly (billing uses uppercase DTO suffixes, ai/customer use mixed-case —
verified in exploration; do not normalize across services, match the file you're editing).
`bin-schedule-manager`'s dispatcher treats 2xx as success and non-2xx as failure
(`dispatchhandler/dispatch.go`), so every new handler must return a real HTTP-style status via the
existing `errorResponse`/`simpleResponse` helpers — a handler that always returns 200 regardless of
outcome would make the execution audit trail lie.

**Partial-failure contract.** All five jobs today are log-and-continue loops over a candidate
list: a single row erroring does not abort the batch (e.g. `expiry.go:57-60`,
`cleanup.go`, `sweep.go:47-62` — the ai sweep also swallows the list-fetch error itself at
`sweep.go:49-51`, returning silently). The migrated handlers must not reproduce silent
swallowing under a green (200) audit row, which the stated goal above explicitly rules out. Rule
for all five: the handler returns non-2xx (mapped via `errorResponse`) if and only if the
**list/fetch call itself fails** (nothing could be attempted — a genuine job failure); per-row
processing errors within a successful list are counted and logged as before, and returned in the
response body as a `failed`/equivalent counter (as `billing-monthly-topup` already does) so the
count is visible without failing the whole run over one bad row. Apply this counter field
uniformly: `{"processed": n, "failed": n}` shape for the two jobs that iterate a candidate list end
to end (top-up, frozen-expiry), `{"expired": n}` / `{"retried": n, "succeeded": n, "exhausted": n}`
where the existing terminology is clearer, per the table above — but every handler must propagate
a non-2xx when its own list-fetch fails, closing the ai-sweep's current silent-failure gap as a
side effect of this migration.

## 5. Per-job correctness analysis

### 5.1 billing-manager: monthly top-up — **not safe today, must be fixed**

`AccountTopUpTokens` (`pkg/dbhandler/billing.go:514-598`) takes a `SELECT ... FOR UPDATE` row lock
but then unconditionally overwrites `balance_token = tokenAmount` (assignment, not increment) and
advances `tm_next_topup` by wall-clock arithmetic rather than reading and incrementing the existing
value. Net effect under two concurrent replicas processing the same due account:

- **Balance**: harmless by accident — both writes set the same absolute value, so the final
  balance is correct regardless of ordering.
- **Ledger**: not harmless — two `billing_billings` rows are inserted (each `+tokenAmount`), one
  per replica. The intended dedup key is `idempotency_key`, which is set to a fresh random UUID
  per call (`billing.go:558`), not the deterministic `reference_id = UUIDv5(accountID, "topup:"+yearMonth)`
  computed two lines later (`billing.go:566`) — the unique index is on `idempotency_key`
  (`ux_billing_billings_idempotency_key`), so the deterministic value never collides and dedup
  never engages. Ledger sum drifts from wallet balance over time.
- **Sequential (not just concurrent) hazard**: if replica A tops up and the customer burns tokens
  before replica B's stale in-memory read of `tm_next_topup` is superseded, B can silently
  re-grant a full month's allowance mid-cycle.

**Fix required as part of this ticket** (in `AccountTopUpTokens`, scoped to the UPDATE statement):
add a CAS guard so the update only applies when due —
`WHERE id = ? AND (tm_next_topup IS NULL OR tm_next_topup <= ?)`, using the method's own internal
`h.utilHandler.TimeNow()` value (the existing `now` local at `billing.go:534`; no new parameter,
no signature change) as the bound value. Check `RowsAffected` **immediately after the UPDATE,
inside the same transaction, before the ledger INSERT**: if 0, `tx.Rollback()` and `return nil`
(the method's existing signature is `error`-only) — the account was already topped up this cycle
by a concurrent claimant, this is a normal outcome, not an error, and critically the ledger
INSERT and `tx.Commit()` at
`billing.go:580-588`/`598` must never execute on this path (this is the entire point of the fix —
a guard that still lets the ledger row through duplicates the effect the CAS exists to prevent).
Mirrors the `ScheduleClaimAndCreateExecution` CAS-then-conditional-insert pattern from Phase 1
(`bin-schedule-manager/pkg/dbhandler/execution.go`). Fixing the `idempotency_key`/`reference_id`
mismatch is explicitly **not** bundled here (§3 non-goals, §8) — once the UPDATE is CAS-guarded,
the ledger simply stops producing the duplicate rows that made the mismatch matter for this job;
the mismatch is dead code worth cleaning up on its own schedule, not a blocker for Phase 2.

A CAS-skip (`nil` — no error, no effect) is neither a processed row nor a failure; the handler's
`(processed, failed int)` counters must not increment either on that outcome (same "neither
processed nor failed" treatment as the `CustomerAnonymizePII` guard-miss case in §5.5, for
symmetry).

`billing-control topup run` (`cmd/billing-control/main.go:716-752`) and the initial top-up on
`customer_created` (`pkg/accounthandler/event.go:84`) both call `AccountTopUpTokens` directly and
get the same safety for free with no call-site change — the fix is entirely inside the dbhandler
method. `billing-control` needs no endpoint change since it already calls the dbhandler directly;
`event.go:84`'s hardening is a side benefit (duplicate `customer_created` delivery would currently
double-grant the signup bonus) worth noting, not a separate task.

### 5.2 billing-manager: failed-event retry — **partially safe, cadence risk**

`FailedEventUpdate` has no CAS guard (`WHERE status='pending'` or similar), so two replicas can
both re-process the same due row: whichever billing insert races second is protected by the
`billing_billings.idempotency_key` unique index (real, correctly wired here — this is the
downstream call the top-up bug's ledger insert should have used), but retry-count bookkeeping is
not — a duplicate increment burns the 5-retry budget at 2x, and the delete race (winner deletes,
loser's update finds 0 rows, silently no-ops) is cosmetic, not corrupting.

**No dbhandler fix required** — downstream idempotency already covers the outcome that matters
(no duplicate billing effects). Only the *retry accounting* is imprecise under double-fire, which
Forbid overlap eliminates at the source (this job never overlaps itself under schedule-manager,
so the question is moot post-migration).

Cadence note: 60s is the tightest cadence in this batch and the tightest the scheduler will carry
anywhere (`SCHEDULE_TICK_INTERVAL_SEC` default 10s already assumes multi-minute jobs are typical).
At 60s this generates 1440 execution audit rows/day for one job alone, against
`SCHEDULE_EXECUTION_RETENTION_DAYS=90` (~130k rows for this job's lifetime retention window) —
not a correctness problem, but worth a deliberate choice. Design decision: keep 60s (`* * * * *`)
to preserve current behavior/responsiveness — a customer's billing event that failed to process
should recover within a minute, not five. Revisit only if `execution_rows` growth becomes an
operational concern (already monitored per Phase 1 §5.7).

### 5.3 ai-manager: proposal sweep — **already safe**

`AIPromptProposalUpdateExpired` is CAS-guarded
(`WHERE id = ? AND tm_delete IS NULL AND status = 'completed'`, `pkg/dbhandler/aipromptproposal.go:147-161`).
A second concurrent (or overlapping) run affects 0 rows on the second pass. No dbhandler change
needed. Safe to migrate as-is.

### 5.4 customer-manager: unverified cleanup — **already safe**

`CustomerUpdate` sets `tm_delete`, which removes the row from the `deleted=false` filter every
subsequent list query uses — a second concurrent pass simply won't see rows the first pass already
flipped (last-writer-wins on the timestamp value is harmless, there's no side effect to double).
No dbhandler change needed.

Correction to the original ticket description: this job does **not** delete or cascade — it is a
bare soft-delete + status flip (`FieldStatus: StatusExpired`, `FieldTMDelete: now`) via
`CustomerUpdate`, no event publish, no downstream fan-out. "Destructive deletes" in the VOIP-1284
inventory table overstated this one; §5.5 below is the job that actually publishes a cascading
event.

### 5.5 customer-manager: frozen expiry — **not safe today, must be fixed**

`CustomerAnonymizePII` has no status guard (`WHERE id = ?` only) — so unlike §5.4, this job
**does** have a real double-fire consequence: two concurrent replicas both anonymize successfully
(each sees `RowsAffected=1`, since there's no CAS to lose), both re-fetch, and **both call
`notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerDeleted, res)`**. Per
`bin-customer-manager/CLAUDE.md`, `customer_deleted` triggers cascading cleanup in
`bin-number-manager` and `bin-billing-manager` — a duplicate cascade fan-out into two other
services.

**Fix required as part of this ticket**: add a status guard to `CustomerAnonymizePII`'s WHERE
clause (`AND status = 'frozen' AND tm_delete IS NULL`, matching the read-path filter in
`CustomerListFrozenExpired`), check `RowsAffected`, and skip the event publish when 0 rows were
affected (already-processed by a concurrent claimant).

**Second caller, same fix, different error-semantics handling.** `CustomerAnonymizePII` has a
second call site: `pkg/customerhandler/freeze.go:91` (`FreezeAndDelete`, which also publishes
`customer_deleted` at `freeze.go:104`). The added guard is compatible there too — `Freeze` runs
before `FreezeAndDelete` in the normal flow, so the row is already `frozen` when this method is
reached — but the two callers must diverge on what a 0-rows result means, because the method
**already** returns `dbhandler.ErrNotFound` today when `RowsAffected == 0`
(`customer.go:474-476`), and "guard didn't match" and "genuinely missing row" become
indistinguishable once the status guard is added:

- **`expiry.go` (the scheduled sweep, this ticket's concern):** a 0-rows result is an *expected,
  routine* outcome under concurrent replicas — count it as neither a failure nor a processed row
  in the new `(processed int, err error)` return, do not log it as an error (today's
  `expiry.go:57-60` logs-and-continues on any error from this call; post-fix that log line would
  fire on every race loss and must be demoted to a no-op or debug-level trace).
- **`FreezeAndDelete` (API-driven, out of scope for this ticket, call site untouched):** a 0-rows
  result here still surfaces as `ErrNotFound` to the caller, same as today — the dbhandler's
  return contract is not changed, only the WHERE clause is tightened, so this call site's code is
  unmodified. **This is, however, a real behavior change for that caller under the specific race
  scenario**, worth stating plainly rather than filed as a non-regression: today, two concurrent
  `FreezeAndDelete` calls on the same customer both silently succeed and both publish
  `customer_deleted` (the same duplicate-cascade bug §5.5 fixes for the scheduled sweep); after
  this change, the loser instead receives `ErrNotFound`. That is strictly better — a visible,
  already-defined error replaces a silent duplicate side effect — but it is a behavior change an
  API caller could observe, not a no-op, and should be called out as such when this fix ships
  rather than only described as "must not regress."

`pkg/customerhandler/freeze_test.go` (mocks `CustomerAnonymizePII` at lines ~323/386/428) needs
its expectations reviewed against the guard addition; §10 adds an explicit concurrent-race test
for `FreezeAndDelete` (not just a review of existing mock expectations) asserting the loser
receives `ErrNotFound` and does not publish a second `customer_deleted` event.

The unbounded `CustomerListFrozenExpired` query (no `LIMIT`, `pkg/dbhandler/customer.go:383-427`)
is real but is a capacity/latency concern, not a correctness one for this ticket — tracked
separately (§8), not blocking.

## 6. Startup-sweep decision (ai-manager)

`SweepStaleAudits` / `SweepStaleProposals` (both 5-minute staleness threshold, both CAS-guarded on
`status='progressing'`) stay exactly as they are: synchronous calls in `run()` before
`runListen`/`runSubscribe`. See §3 non-goals for the reasoning. Documented here so a future reader
of this design doesn't wonder why two of the four ai-manager sweep-shaped functions were migrated
and two weren't.

## 7. Seed schedules

Five new rows in `schedule_schedules`, seeded by an Alembic migration in `bin-dbscheme-manager`
following the exact pattern of `a5e6f559299c_schedule_schedules_seed_platform_jobs.py` (raw SQL via
`op.execute`, `JSON_OBJECT()` for `target_data` — **never a raw `'{"k":v}'` string literal**, per
the VOIP-1283 hotfix: `sqlalchemy.text()` misparses a bare colon as a bind parameter and breaks
`alembic upgrade`; `UNHEX(REPLACE(UUID(),'-',''))` ids, nil-UUID customer_id for platform jobs,
`name` is the stable cross-environment handle).

| name | cron (UTC) | target | data | timeout_ms | retry_max | enabled |
|---|---|---|---|---|---|---|
| `billing-monthly-topup` | `0 * * * *` | billing-manager `/v1/accounts/top_up` POST | `{}` | 300000 | 0 | 1 |
| `billing-failed-event-retry` | `* * * * *` | billing-manager `/v1/failed_events/retry` POST | `{}` | 60000 | 0 | 1 |
| `ai-proposal-expiry` | `0 * * * *` | ai-manager `/v1/aipromptproposals/expire` POST | `{}` | 300000 | 0 | 1 |
| `customer-unverified-cleanup` | `*/15 * * * *` | customer-manager `/v1/customers/cleanup_unverified` POST | `{}` | 60000 | 0 | 1 |
| `customer-frozen-expiry` | `0 4 * * *` | customer-manager `/v1/customers/cleanup_frozen_expired` POST | `{}` | 600000 | 0 | 1 |

Cadence matches current ticker intervals exactly (no behavior change beyond fixing the
pod-start-relative drift): **`billing-monthly-topup` is hourly (`0 * * * *`), matching the
existing 1h ticker** — not daily. This matters beyond consistency: `tm_next_topup` lands exactly
on `00:00:00 UTC` of the 1st of the month (`billing.go:540`, and identically in
`account_paddle.go:221`), so a once-daily schedule would have zero margin against that boundary —
any clock skew or a failed run (`retry_max: 0`) would push a free-plan customer's monthly top-up
out by up to 24h instead of ≤1h. Hourly preserves the current recovery window. `0 4 * * *` for
frozen-expiry avoids collision with the three Phase 1 platform jobs at 01:00/02:00/02:30.
`retry_max: 0` throughout, matching Phase 1's `number-renew` precedent — a failed run waits for
the next natural slot rather than immediate in-run retry; `billing-failed-event-retry`'s own
5-attempt internal retry budget already provides retry semantics at the business-logic layer, a
second retry layer at the dispatch layer would be redundant.

`timeout_ms` set generously above each job's expected runtime bound from exploration, per job:
`billing-monthly-topup` paginates *all* accounts at 500/page with no upper bound (300000ms);
`billing-failed-event-retry` and `customer-unverified-cleanup` are single-page, ≤ a few hundred
rows (60000ms each); `ai-proposal-expiry` is a single page of ≤1000 rows with one cached `AIGet`
per candidate (300000ms); `customer-frozen-expiry` is the one unbounded query in this batch
(no `LIMIT` in `CustomerListFrozenExpired`, §8) and gets the largest budget, 600000ms/10min, as a
circuit-breaker until the separate unbounded-query fix lands.

**Abandoned-run / overlap-guard interaction for the 60s job.** `billing-failed-event-retry`'s
`timeout_ms` (60000) equals its own cron period. Per Phase 1's reaper formula
(`tm_deadline = tm_start + timeout_ms×(retry_max+1) + 5000×retry_max + 60000`ms margin — the
`5000×retry_max` term is 0 here since every Phase 2 job seeds `retry_max: 0`, but is included for
completeness since a future job with retries would need it; `bin-schedule-manager` design §5.6),
an abandoned run of this job blocks the Forbid overlap guard for ~2 scheduled slots (~2 minutes)
before the reaper marks it `abandoned` and the next tick can claim — a wider window than the
60000ms number alone suggests. Acceptable (matches the existing service's own tolerance for a
stuck retry loop) but worth having on record rather than discovered during an incident.

## 8. Explicitly deferred (separate follow-up tickets, not this ticket)

- `idempotency_key`/`reference_id` mismatch in `AccountTopUpTokens`'s ledger insert (dead
  deterministic-key code; moot for double-fire once §5.1's CAS fix lands, but worth its own
  cleanup).
- Unbounded `CustomerListFrozenExpired` query — add pagination/LIMIT.
- campaign-manager swallowed re-arm errors (`execute.go:84,108`) — VOIP-1282 follow-up.
- queue-manager lost-timeout-in-`connecting`-state gap, transient delayed-publish durability —
  VOIP-1282 follow-up.
- Stale documentation in campaign-manager/queue-manager CLAUDE.md/README/operations.md claiming a
  dependency on an external scheduler — VOIP-1282 follow-up (doc-only).

## 9. Sandbox changes (separate repository — sequenced after monorepo release)

The sandbox is a **different repository** (`/home/pchero/gitvoipbin/sandbox`) that pins monorepo
service images by digest (`_create_initial_version_pins` / `versions.lock`). §11's single-PR
discipline applies to the *monorepo* change only; the sandbox change is a separate PR in a
separate repo and must not land first — removing the guard before the fixed billing-manager /
customer-manager images are built, released, and pinned in the sandbox would silently re-expose
the exact hazard the guard exists to catch. Ordering:

1. Monorepo PR (this design) merges and the fixed images are built/released.
2. Sandbox `versions.lock` is bumped to pin the fixed images (routine version-bump PR).
3. Only then, in a follow-up sandbox PR: remove `_check_ticker_replica_guard`
   (`scripts/voipbin-cli.py:2266-2311`) and its two call sites (`cmd_start`, lines 2328/2358) —
   the hazard it warns about no longer exists once steps 1-2 land (both
   money-affecting/cascade-affecting jobs are fixed at the dbhandler layer *and* run through
   Forbid-overlap dispatch; the three already-safe jobs never needed the guard).
4. Delete or rewrite `scripts/tests/test_ticker_replica_guard.py` (4 cases, G1–G4) in the same
   sandbox PR.
5. Update `docs/plans/2026-07-05-production-grade-horizontal-scale-design.md` §1.4 (the guard's
   origin) and its Phase-2 roadmap line (224) — supersede with a pointer to this design doc rather
   than the deferred "redsync lock, when an operator requests N>1" plan, which this ticket makes
   moot.

## 10. Testing strategy

- Each fixed dbhandler method (`AccountTopUpTokens`, `CustomerAnonymizePII`) gets a concurrent
  double-fire test at the dbhandler layer, same shape as Phase 1's
  `Test_DoubleFire`/`Test_KillNineStateMachine` in `bin-schedule-manager/pkg/dbhandler/execution_test.go`:
  two "replica" instances racing the same due row against one sqlite/mysql-fixture DB, asserting
  exactly one side effect (one ledger row / one event publish).
- Each new signature change (`(count, err)`/`(processed, failed int, err error)` instead of
  `(err)`/`void`) needs its existing unit test updated to assert the returned count, not just
  error-nil. Existing test files to update: `cleanup_test.go`, `expiry_test.go`,
  `freeze_test.go` (mocks `CustomerAnonymizePII` at ~L323/386/428 — review expectations against
  the new status guard even though `FreezeAndDelete` itself is untouched, §5.5) in
  customer-manager; `sweep_test.go` in ai-manager; `failedeventhandler/main_test.go` in
  billing-manager. `runMonthlyTopUp` currently has **no test** at all (lives in
  `cmd/billing-manager/main.go`, untested) — moving it into a handler package is a net
  test-coverage win, not just a refactor; new tests required as part of this move.
- `bin-billing-manager/scripts/database_scripts_test/table_billing_billings.sql` currently
  declares `idx_billing_billings_idempotency_key` as **non-unique**, while the real MySQL schema
  has it `UNIQUE` (`ux_billing_billings_idempotency_key`). Fix the sqlite fixture to match —
  without it, the double-fire test for `AccountTopUpTokens` (below) cannot actually observe the
  duplicate-ledger-row failure mode it's meant to catch pre-fix, and would pass even without the
  CAS guard.
- Concurrent-race test for `CustomerAnonymizePII`'s two callers: one at the dbhandler layer (two
  concurrent claimants racing the scheduled sweep's read-then-anonymize, asserting exactly one
  `customer_deleted` publish — same shape as Phase 1's `Test_DoubleFire`), and one specifically for
  `FreezeAndDelete` (§5.5) asserting the race loser receives `ErrNotFound` and no duplicate event.
- New listenhandler routing tests per new route (5), following the Phase 1
  `bin-schedule-manager/pkg/listenhandler` pattern (mocked handler `EXPECT`, no `gomock.Any()` on
  parsed payloads).
- Migration: standard Alembic `alembic revision` flow, `alembic heads` single-head check, matching
  sqlite test fixtures if `scripts/database_scripts_test/` conventions in the three touched
  services require it (they don't seed data today — schedule seeding lives entirely in
  bin-dbscheme-manager, per Phase 1 precedent). Seed rows use the full column set from
  `a5e6f559299c_schedule_schedules_seed_platform_jobs.py` (`id, customer_id, name, detail, type,
  cron, target_queue, target_uri, target_method, target_data_type, target_data, timeout_ms,
  retry_max, enabled, tm_next_run, tm_create, tm_update`), not just the subset shown in §7's
  table — `type: 'rpc'`, `target_data_type: 'application/json'`, `tm_next_run: NULL`.
- Per root CLAUDE.md's service-docs-sync rule: adding routes to `pkg/listenhandler/main.go` and
  removing goroutines from `cmd/*/main.go` obligates updating `docs/architecture.md` (routing
  table + events section) for all three touched services in the same commit
  (`bash docs/reference/extractor.sh <service-dir>` to re-extract).
- End-to-end: verify each new schedule fires via bin-schedule-manager's existing manual-execute
  endpoint (`POST /v1/schedules/<id>/execute`) against a fixture with known due rows, asserting the
  count response and downstream side effect.

## 11. Rollout / risk

- Additive at the schedule-manager layer (new rows only, no engine change) — the risk surface is
  entirely in the three services being edited (billing/ai/customer-manager) and is bounded to:
  removing five goroutines, adding five endpoints, and two dbhandler CAS fixes.
- The two CAS fixes are the only behavior changes with production consequence beyond
  "job now runs on a schedule instead of a ticker" — both are strictly safety-improving (a
  double-fire that used to silently corrupt ledger/cascade data now no-ops instead).
- Seeded `enabled: 1` for all five (unlike Phase 1's `database-backup`, which shipped disabled) —
  these are direct replacements for jobs already running unconditionally in production; there is
  no "opt-in later" step analogous to self-hosted backup enablement.
- Cutover: land the endpoints + dbhandler fixes + seed migration in one monorepo PR (mirrors
  Phase 1's single-PR discipline), remove the ticker goroutines in the same PR (no transition
  period running both — that would double-fire by construction). Verify via manual-execute before
  relying on the cron schedule in any environment that matters. Sandbox guard removal is a
  separate, later PR in the sandbox repo (§9).
