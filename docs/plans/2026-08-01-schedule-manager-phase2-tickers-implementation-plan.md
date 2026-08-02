# bin-schedule-manager Phase 2 — Implementation Plan (VOIP-1284)

Status: approved (plan review loop: 6 rounds — RC, RC, Approve, RC, Approve, Approve)
Design: [2026-08-01-schedule-manager-phase2-tickers-design.md](2026-08-01-schedule-manager-phase2-tickers-design.md) (approved, 3-round review)
Branch: `VOIP-1284-Migrate-tickers-to-schedule-manager`
(worktree `.worktrees/VOIP-1284-Migrate-tickers-to-schedule-manager`)

Single monorepo PR (design §11). Sandbox guard removal is a separate, later PR in the sandbox
repo, sequenced after image release (design §9) — not part of this plan.

Commit granularity: one commit per numbered step below; each must pass the full verification
workflow in its own service directory before the next step starts.

## Step 1 — billing-manager: fix `AccountTopUpTokens`, add the top-up endpoint

**Signature change (required — see below):**
`AccountTopUpTokens(ctx, accountID, customerID uuid.UUID, tokenAmount int64, planType string) error`
becomes `AccountTopUpTokens(ctx, accountID, customerID uuid.UUID, tokenAmount int64, planType string) (applied bool, err error)`.
An `error`-only return cannot distinguish "CAS-skip, not due yet" from "actually applied" —
both must resolve to `nil` error — and the design's `TopUpDue` counter logic (§5.1: CAS-skip
counts as neither `processed` nor `failed`) requires that distinction. Three call sites, all
updated in this step:
- `pkg/dbhandler/main.go:31` — interface declaration.
- `cmd/billing-manager/main.go:269` (moving into `pkg/accounthandler/topup.go` in this same step,
  see below) — becomes the primary caller that reads `applied`.
- `cmd/billing-control/main.go:743` (`billing-control topup run`) — already pre-filters candidates
  by `a.TmNextTopUp.After(now)` before calling; on the new CAS, `applied=false` for a filtered-in
  candidate only happens under a genuine race with the scheduled job, which is fine — log at
  debug and continue, do not treat as an error.
- `pkg/accounthandler/event.go:84` (initial top-up on `customer_created`) — `applied=false` here
  is nearly impossible (`TmNextTopUp` is unset on a brand-new account, so `IS NULL` always
  matches) but handle it the same way as the other callers for consistency: not an error.
- `pkg/accounthandler/event_test.go:166,218` — existing `mockDB.EXPECT().AccountTopUpTokens(...).Return(nil)` /
  `.Return(fmt.Errorf(...))` mock expectations must be updated to the two-value return
  (`.Return(true, nil)` / `.Return(false, fmt.Errorf(...))`) or these tests will fail to compile
  once the signature changes — self-evident from `go test ./...` in Step 1's verify block, listed
  here explicitly so it isn't missed while editing.
- Regenerate `pkg/dbhandler/mock_main.go`.

Files:
- `pkg/dbhandler/billing.go` (`AccountTopUpTokens`, L514-598 today): add the CAS WHERE clause to
  the UPDATE (today's plain `UPDATE ... WHERE id = ?` around L543-552) —
  `WHERE id = ? AND (tm_next_topup IS NULL OR tm_next_topup <= ?)`, reusing the `now` local the
  method **already computes** at `billing.go:534` (`now := h.utilHandler.TimeNow()`) as the bound
  value — no new clock call needed inside this method. Check `sql.Result.RowsAffected()`
  immediately after the UPDATE, inside the transaction, before the ledger INSERT block
  (L555-588 today): if 0, `tx.Rollback()` (already deferred, but call it explicitly for clarity)
  and `return false, nil` — normal CAS-skip, not an error. On the UPDATE succeeding, proceed to
  the ledger insert and commit as today, then `return true, nil`.
- `pkg/dbhandler/billing_test.go`: **net-new** test coverage for `AccountTopUpTokens` — this
  function has zero existing test cases in this file today (verified: no case for it exists to
  "update alongside"). Add table cases: due account → `applied=true`, ledger row inserted,
  balance set; not-yet-due account (`tm_next_topup` in the future) → `applied=false, err=nil`, no
  ledger row, no balance change; account not found → `err=ErrNotFound` (existing behavior,
  confirm unchanged).
- **No dbhandler-layer double-fire test in this step.** `AccountTopUpTokens` opens with
  `SELECT ... FOR UPDATE` (`billing.go:524-526`), which SQLite does not support — the shared
  sqlite harness pins `SetMaxOpenConns(1)` (`pkg/dbhandler/main_test.go:21`), which serializes
  writers and would mask a real race even if `FOR UPDATE` didn't error outright first. The
  concurrency proof for this fix is
  therefore **not** a Phase-1-style two-goroutine sqlite test. Instead: (a) a single-threaded
  test that calls `AccountTopUpTokens` twice in a row for the same account with the same `now`,
  asserting the second call returns `applied=false` and exactly one `billing_billings` row exists
  — this proves the CAS logic is correct without needing real concurrency; (b) note in the PR
  description that true concurrent-replica proof for this specific fix relies on the row lock
  (`FOR UPDATE`) plus the CAS predicate together, which cannot be exercised in the sqlite unit
  harness and is not blocking for this PR (the row lock forces serialization at the DB layer
  regardless of the CAS; the CAS is what makes the second serialized writer a no-op instead of a
  second write — this is provable by code inspection plus the sequential test in (a), not by a
  race test).
- `scripts/database_scripts_test/table_billing_billings.sql`: change
  `idx_billing_billings_idempotency_key` from a plain index to a **unique** index
  (`ux_billing_billings_idempotency_key`), matching the real MySQL schema (correctness fix,
  independent of the point above — this makes the sqlite fixture accurately reflect production
  even though the specific double-fire scenario can't be raced in sqlite).
- `pkg/accounthandler/topup.go` (new file): move `runMonthlyTopUp`'s body out of
  `cmd/billing-manager/main.go:240-283` into an exported method, e.g.
  `TopUpDue(ctx context.Context) (processed, failed int, err error)`, on the existing
  `AccountHandler`. Keep the identical pagination shape (500/page, cursor on `tm_create`,
  `FieldDeleted: false` filter, `PlanTokenMap` lookup, skip `tokenAmount <= 0`), but switch the
  `time.Now()` call (today `main.go:243`) to `h.utilHandler.TimeNow()` (the handler already has
  `utilHandler` — verify via `pkg/accounthandler/main.go`'s struct fields) so the new
  `topup_test.go` can control the clock. Per row: call `h.db.AccountTopUpTokens(...)`; `applied
  == true` → increment `processed`; `applied == false, err == nil` → increment neither (CAS-skip,
  matches design §5.1); `err != nil` → increment `failed`, log (as today), `continue` (does not
  abort the loop). Return a non-nil `err` from `TopUpDue` only if the `AccountList` call itself
  fails (nothing could be attempted) — per design §4's partial-failure contract.
- `pkg/accounthandler/topup_test.go` (new): table-driven test for `TopUpDue` with a mocked
  dbhandler — pagination across 2 pages, a due account, a not-due account (mocked `applied=false`),
  a per-row error, an `AccountList` failure. This logic currently has zero test coverage (lived
  untested in `cmd/main.go`).
- `pkg/accounthandler/main.go`: add `TopUpDue(ctx context.Context) (int, int, error)` to the
  `AccountHandler` interface; regenerate `pkg/accounthandler/mock_main.go`.
- `pkg/listenhandler/main.go`: add `regV1AccountsTopUp = regexp.MustCompile("/v1/accounts/top_up$")`
  and a switch case → `processV1AccountsTopUpPost`, placed among the other literal-path
  `/v1/accounts/...` routes per this file's existing ordering.
- `pkg/listenhandler/v1_accounts.go`: new handler func `processV1AccountsTopUpPost` — no request
  body; calls `h.accountHandler.TopUpDue(ctx)`; on handler error → `errorResponse(err)`
  (non-2xx); on success → marshal `response.V1ResponseAccountsTopUp{Processed: n, Failed: n}`
  (style B, bare scalars) → 200.
- `pkg/listenhandler/models/response/account.go`: add `V1ResponseAccountsTopUp{Processed, Failed
  int}` to this **existing** file (it already holds `V1ResponseAccountsIDIsValidBalance` —
  confirmed present at `bin-billing-manager/pkg/listenhandler/models/response/account.go`; do not
  create a new file).
- Routing test (wherever this service's `pkg/listenhandler` routing tests live — check for a
  `main_test.go` or per-route test file pattern and match it): mocked `AccountHandler.TopUpDue`
  EXPECT, assert response body on success, assert non-2xx mapping on handler error.
- `cmd/billing-manager/main.go`: delete the top-up ticker goroutine (`run()`, currently
  L144-158) and `runMonthlyTopUp` (currently L240-283, now moved). Prune now-unused imports in
  this file if `runMonthlyTopUp` was their only user (verify — `time` is still needed by the
  failed-event-retry ticker touched in Step 2, so do not remove it yet).

Verify: full 5-step workflow in `bin-billing-manager`.

## Step 2 — billing-manager: failed-event retry endpoint + main.go restructure

**This step requires restructuring `run()`/`runSubscribe()`, not just adding a route.**
`failedEventHandler` is built *inside* `runSubscribe()` (`cmd/billing-manager/main.go`, current
flow: `runListen()` at line ~134 runs **before** `runSubscribe()` at line ~139, and
`failedHandler` is a local variable constructed inside `runSubscribe()` at ~L194, via a circular
wiring dance: `subHandler` is created first with a `nil` failed-event processor placeholder, then
`failedHandler := failedeventhandler.NewFailedEventHandler(db, subscribehandler.GetEventProcessor(subHandler))`,
then `subscribehandler.SetFailedEventHandler(subHandler, failedHandler)` patches it back onto
`subHandler`). `failedHandler` is never passed to `listenhandler.NewListenHandler` (called inside
`runListen`), and cannot be without restructuring, since it doesn't exist yet at that point in the
current call order.

**Required restructure:** extract the `subHandler`/`failedHandler` circular-construction block
(currently inside `runSubscribe`) into its own function, e.g. `buildFailedEventHandler(db,
sockHandler, accountHandler, billingHandler) (subscribehandler.SubscribeHandler,
failedeventhandler.FailedEventHandler)`, called from `run()` **before** `runListen`. Change
`runListen`'s signature to accept `failedHandler failedeventhandler.FailedEventHandler` and pass
it through to `listenhandler.NewListenHandler`. Change `runSubscribe` to accept the
already-constructed `subHandler` (just calls `.Run()` on it) instead of building it. Existing
`pkg/listenhandler/*_test.go` files construct `&listenHandler{...}` struct literals directly
rather than calling `NewListenHandler` — adding a `failedEventHandler` field to the struct does
not break those literals (unset fields default to nil), so no test call-site update is needed
there; the new route's own test (below) is what actually exercises the new field.

Files:
- `pkg/failedeventhandler/main.go` (`RetryPending`): both the interface declaration and the
  implementation live in this one file (unlike Steps 3-5's services), so no separate
  interface-file edit is needed here — change the signature from `(ctx) error` to
  `(ctx) (retried, succeeded, exhausted int, err error)` in place; `go generate ./...`
  regenerates `mock_main.go` from the same file's `mockgen -source main.go` directive. Internals
  unchanged (no CAS needed per
  design §5.2 — downstream `billing_billings.idempotency_key` uniqueness, now genuinely `UNIQUE`
  in the sqlite fixture per Step 1, already protects the outcome that matters); accumulate the
  three counters across the existing loop body. Return a non-nil `err` only if the initial
  `FailedEventListPendingRetry` call fails.
- `pkg/failedeventhandler/main_test.go`: update existing `RetryPending` test cases to assert the
  three returned counts, not just error-nil.
- `pkg/listenhandler/main.go`: add
  `regV1FailedEventsRetry = regexp.MustCompile("/v1/failed_events/retry$")` + switch case;
  `NewListenHandler`'s signature and the `listenHandler` struct gain a `failedEventHandler
  failedeventhandler.FailedEventHandler` field.
- `pkg/listenhandler/v1_accounts.go` (or a new `v1_failed_events.go` if this file's route
  grouping convention favors a dedicated file per resource — match the existing pattern, e.g.
  `v1_accounts.go` holds account routes, a billing-record-adjacent route may deserve its own
  file): handler `processV1FailedEventsRetryPost` → `h.failedEventHandler.RetryPending(ctx)` →
  `response.V1ResponseFailedEventsRetry{Retried, Succeeded, Exhausted int}` on success (new file
  in `models/response/`, e.g. `failed_events.go`), `errorResponse` on list-fetch failure.
- Routing test for the new route, same shape as Step 1.
- `cmd/billing-manager/main.go`: delete the failed-event-retry ticker goroutine (currently at the
  end of `runSubscribe`, ~L205-219). `chDone` is still needed by `run()`'s own shutdown wait
  regardless of tickers — do not remove it.

Verify: full 5-step workflow in `bin-billing-manager` (second commit in this service directory,
Step 1's changes still present).

## Step 3 — ai-manager: proposal expiry endpoint (no dbhandler fix needed — design §5.3)

Files:
- `pkg/aipromptproposalhandler/sweep.go` (`SweepExpiredProposals`): change signature to
  `(ctx) (expired int, err error)` (verify its exact current signature first — exploration
  reported it as called without inspecting a return value; confirm at the call site in
  `cmd/ai-manager/main.go:171` before assuming `void` vs. an already-present unused `error`).
  `AIPromptProposalUpdateExpired` is already CAS-guarded (`WHERE status='completed'`) — no
  dbhandler change. Per design §4's partial-failure contract: return non-nil `err` only if
  `AIPromptProposalList` fails; count `expired` across the loop; a per-row `AIGet` failure
  (`continue`s today) does not increment `expired` and does not fail the batch.
- `pkg/aipromptproposalhandler/main.go:44`: the `AIPromptProposalHandler` interface declares
  `SweepExpiredProposals(ctx context.Context)` **in a separate file** from the implementation
  above — update this line to the new `(ctx) (expired int, err error)` signature. `go generate
  ./...` (part of the standard verify workflow) regenerates `mock_main.go` from this file's
  `//go:generate mockgen -source main.go` directive once the interface line is edited by hand;
  it does not update itself from the `sweep.go` implementation alone.
- `pkg/aipromptproposalhandler/sweep_test.go`: update to assert the returned count.
- `pkg/listenhandler/main.go`: add
  `regV1AIPromptProposalsExpire = regexp.MustCompile("/v1/aipromptproposals/expire$")` + switch
  case, positioned near the existing `/v1/aipromptproposals/...` routes.
- `pkg/listenhandler/v1_aipromptproposals.go` (confirmed existing file — this is where
  `/v1/aipromptproposals/...` handlers live, not `v1_data_aipromptproposals.go`, which is the
  **request DTO** file under `models/request/`): new handler func
  `processV1AIPromptProposalsExpirePost` → `h.aiPromptProposalHandler.SweepExpiredProposals(ctx)`
  → `response.V1ResponseAIPromptProposalsExpire{Expired: n}` / `errorResponse`. Add the response
  DTO to `pkg/listenhandler/models/response/` (this package **exists** for ai-manager — confirmed
  `bin-ai-manager/pkg/listenhandler/models/response/`; add a new file or extend an existing one
  matching the resource, check current contents first).
- Routing test for the new route.
- `cmd/ai-manager/main.go`: delete the proposal-sweep ticker goroutine (currently L165-177, the
  one with the dead `ctx.Done()` arm where `ctx := context.Background()`). **Do not touch** the
  two startup one-shot sweep calls (`SweepStaleAudits`, `SweepStaleProposals`) — design §6,
  explicitly out of scope, stays per-boot. Prune the now-unused `"time"` import if the deleted
  ticker was its only user in this file (it likely is — verify).

Verify: full 5-step workflow in `bin-ai-manager`.

## Step 4 — customer-manager: unverified cleanup endpoint (no dbhandler fix needed — design §5.4)

Files:
- `pkg/customerhandler/cleanup.go`: export `cleanupUnverified` (or add a thin exported wrapper —
  match whatever the existing private/public split convention in this package looks like; check
  the actual current function name and visibility first) with signature `(ctx) (expired int, err
  error)`. `CustomerUpdate` needs no CAS — soft-delete via `tm_delete` is structurally idempotent
  (design §5.4). Non-nil `err` only on `CustomerList` failure. **Count `expired` only on
  successful `CustomerUpdate` calls** — today's code reports `len(customers)` unconditionally
  regardless of per-row update failures (`cleanup.go`, confirm exact lines); this is a real
  behavior fix, not a mechanical signature change: a row whose `CustomerUpdate` errors must not
  be counted as expired.
- `pkg/customerhandler/main.go:66`: the `CustomerHandler` interface declares
  `RunCleanupUnverified(ctx context.Context)` **in a separate file** from `cleanup.go` — update
  this line to the new exported name and `(ctx) (expired int, err error)` signature. `go generate
  ./...` regenerates `mock_main.go` from this file's `mockgen -source main.go` directive once the
  interface line is hand-edited; it does not follow from editing `cleanup.go` alone.
- `pkg/customerhandler/cleanup_test.go`: update to assert the returned count, including a case
  where a per-row `CustomerUpdate` fails and confirm it is excluded from the count.
- `pkg/listenhandler/main.go`: add
  `regV1CustomersCleanupUnverified = regexp.MustCompile("/v1/customers/cleanup_unverified$")` +
  switch case, placed as a literal-path route **before** `regV1CustomersID`
  (`"/v1/customers/" + regUUID + "$"`, confirmed at `pkg/listenhandler/main.go:60`) in file
  order, matching this file's existing convention for `/v1/customers/signup` and
  `/v1/customers/email_verify` (`regV1CustomersID` cannot match `/cleanup_unverified` since it
  requires a UUID segment, so there's no functional collision either way — placement is purely
  matching the file's established literal-path-first convention).
- `pkg/listenhandler/v1_customers.go` (confirmed existing file — general customer routes; the
  freeze-family routes live in the separate `v1_customers_freeze.go`, which this new pair of
  routes resembles more closely — use whichever of the two this service's convention would put a
  new "customer lifecycle action" route in; default to `v1_customers.go` unless
  `v1_customers_freeze.go`'s existing content suggests otherwise): handler
  `processV1CustomersCleanupUnverifiedPost` → `h.customerHandler.CleanupUnverified(ctx)` (final
  exported name matches whatever Step 4's dbhandler-adjacent change above settles on) →
  `response.V1ResponseCustomersCleanupUnverified{Expired: n}` / `errorResponse`.
- `pkg/listenhandler/models/response/` (**new package** — customer-manager currently has only
  `models/request/`, confirmed no `models/response/` directory exists today): create
  `pkg/listenhandler/models/response/customers.go` holding both this step's and Step 5's response
  DTOs.
- Routing test for the new route.
- `cmd/customer-manager/main.go`: delete the unverified-cleanup ticker call site (currently
  `go customerHandler.RunCleanupUnverified(context.Background())`, ~L100-101). Grep for other
  callers of `RunCleanupUnverified` before deleting the function itself — if none, delete the
  ticker-loop wrapper function in `cleanup.go` too, in this same step (leaving a dead exported
  function whose only caller was just removed is worse than a slightly larger single-step diff).

Verify: full 5-step workflow in `bin-customer-manager`.

## Step 5 — customer-manager: fix `CustomerAnonymizePII`, add the frozen-expiry endpoint

Files:
- `pkg/dbhandler/customer.go` (`CustomerAnonymizePII`, ~L430-484): add
  `AND status = 'frozen' AND tm_delete IS NULL` to the WHERE clause. Return contract unchanged —
  still returns `dbhandler.ErrNotFound` when `RowsAffected == 0` (this now covers both "genuinely
  missing row" and "guard didn't match, a concurrent claimant already won"; the two customer-
  handler call sites interpret which applies, per design §5.5 — the dbhandler contract itself
  does not change).
- `pkg/dbhandler/customer_test.go`: add a case exercising the new guard (row not in `frozen`
  status → `ErrNotFound`, no anonymization applied) alongside existing `CustomerAnonymizePII`
  coverage.
- `pkg/dbhandler/customer_test.go` (new): **double-fire test** at the dbhandler layer — this
  package's sqlite harness works for this method (plain `UPDATE`, no `FOR UPDATE`, unlike Step
  1's billing case) — two goroutines racing `CustomerAnonymizePII` for the same frozen customer
  against one sqlite DB; assert exactly one call returns success (nil error) and the other
  returns `ErrNotFound`, and the row ends in `status=deleted` with anonymized fields exactly
  once (not double-anonymized in a way that would matter, but primarily proving the CAS holds).
  Same shape as Phase 1's `Test_DoubleFire` in `bin-schedule-manager/pkg/dbhandler/execution_test.go`.
- `pkg/customerhandler/expiry.go` (`cleanupFrozenExpired` or equivalent — export per this step,
  confirm exact current name/visibility first): signature `(ctx) (processed int, err error)`. At
  the `CustomerAnonymizePII` call site (currently ~L57-60, log-and-continue on any error): if the
  error `errors.Is(err, dbhandler.ErrNotFound)`, treat as a **normal CAS-skip** — do not log as
  an error, do not increment `processed`, do not call `CustomerGet`/`PublishEvent`, `continue`.
  Any other error still logs and `continue`s without incrementing `processed`, unchanged from
  today's behavior. On success (no error): `CustomerGet` + `PublishEvent(customer_deleted)` as
  today, increment `processed`. Non-nil `err` returned from `cleanupFrozenExpired` itself only if
  `CustomerListFrozenExpired` fails.
- `pkg/customerhandler/main.go:67`: the `CustomerHandler` interface declares
  `RunCleanupFrozenExpired(ctx context.Context)` **in a separate file** from `expiry.go` — update
  this line to the new exported name and `(ctx) (processed int, err error)` signature (this is
  the same file Step 4 already touches for its own interface line — both edits land in the same
  file across Steps 4 and 5). `go generate ./...` regenerates `mock_main.go` once the interface
  line is hand-edited.
- `pkg/customerhandler/expiry_test.go`: update to assert the returned count; add a case for the
  `ErrNotFound`-as-CAS-skip path (not logged as an error, not counted, no event published) — this
  test is at the **handler** layer with a mocked dbhandler, distinct from the dbhandler-layer
  double-fire test above.
- `pkg/customerhandler/freeze_test.go`: add a concurrent-race test for `FreezeAndDelete`
  (`pkg/customerhandler/freeze.go:91` calls `CustomerAnonymizePII`) — two concurrent calls on the
  same customer at the dbhandler layer (or mocked at the handler layer with the second call
  returning `ErrNotFound`, matching how this test file's existing style mocks the dbhandler);
  assert the winner succeeds and publishes `customer_deleted` once, the loser receives
  `dbhandler.ErrNotFound` (design §5.5's real, observable behavior change for this
  out-of-scope-but-affected caller: today both concurrent calls silently succeed and both
  publish; after this fix, the loser gets a visible error and does not publish) and does not
  publish a second event. Review the existing mocked `CustomerAnonymizePII` expectations at
  ~L323/386/428 against the new WHERE clause — no code change needed there since the dbhandler
  signature/contract is unchanged, confirm the mocks still make sense as-is.
- `pkg/listenhandler/main.go`: add
  `regV1CustomersCleanupFrozenExpired = regexp.MustCompile("/v1/customers/cleanup_frozen_expired$")`
  + switch case, same literal-path-first placement as Step 4.
- `pkg/listenhandler/v1_customers.go` (or wherever Step 4 placed the sibling route): handler
  `processV1CustomersCleanupFrozenExpiredPost` →
  `response.V1ResponseCustomersCleanupFrozenExpired{Processed: n}` / `errorResponse`, added to
  the same `models/response/customers.go` file created in Step 4.
- Routing test for the new route.
- `cmd/customer-manager/main.go`: delete the frozen-expiry ticker call site (currently
  `go customerHandler.RunCleanupFrozenExpired(context.Background())`, ~L103-104) and, per the
  same grep-first rule as Step 4, the now-dead `RunCleanupFrozenExpired` ticker-loop wrapper if
  it has no other callers.
- After Steps 4 and 5: check whether `cmd/customer-manager/main.go`'s `"context"` import is now
  unused (it was only referenced by the two deleted `context.Background()` ticker calls) and
  prune if so.

Verify: full 5-step workflow in `bin-customer-manager` (third commit in this service directory,
Step 4's changes still present).

## Step 6 — Seed migration (bin-dbscheme-manager)

```bash
cd bin-dbscheme-manager/bin-manager
cp -n alembic.ini.sample alembic.ini
alembic -c alembic.ini revision -m "schedule_schedules_seed_phase2_ticker_jobs"
alembic -c alembic.ini heads   # exactly ONE head
```

One migration, five `op.execute` INSERTs into `schedule_schedules`, following
`a5e6f559299c_schedule_schedules_seed_platform_jobs.py`'s exact pattern: full column list
(`id, customer_id, name, detail, type, cron, target_queue, target_uri, target_method,
target_data_type, target_data, timeout_ms, retry_max, enabled, tm_next_run, tm_create, tm_update`),
`UNHEX(REPLACE(UUID(),'-',''))` ids, nil-UUID customer_id, `type='rpc'`,
`target_data_type='application/json'`, `target_data=JSON_OBJECT()` for all five (**never** a raw
`'{...}'` string literal — the VOIP-1283 hotfix pitfall: a bare colon inside a string literal is
misparsed as a bind parameter by `sqlalchemy.text()`), `tm_next_run=NULL`,
`tm_create`/`tm_update=UTC_TIMESTAMP(6)`.

| name | cron | target_queue | target_uri | timeout_ms | retry_max | enabled |
|---|---|---|---|---|---|---|
| `billing-monthly-topup` | `0 * * * *` | `bin-manager.billing-manager.request` | `/v1/accounts/top_up` | 300000 | 0 | 1 |
| `billing-failed-event-retry` | `* * * * *` | `bin-manager.billing-manager.request` | `/v1/failed_events/retry` | 60000 | 0 | 1 |
| `ai-proposal-expiry` | `0 * * * *` | `bin-manager.ai-manager.request` | `/v1/aipromptproposals/expire` | 300000 | 0 | 1 |
| `customer-unverified-cleanup` | `*/15 * * * *` | `bin-manager.customer-manager.request` | `/v1/customers/cleanup_unverified` | 60000 | 0 | 1 |
| `customer-frozen-expiry` | `0 4 * * *` | `bin-manager.customer-manager.request` | `/v1/customers/cleanup_frozen_expired` | 600000 | 0 | 1 |

All `target_method='POST'`. Downgrade: `DELETE FROM schedule_schedules WHERE name IN (...) AND
customer_id = UNHEX('00000000000000000000000000000000')`. Note (unlike Phase 1's seed, which ran
before any execution could reference the rows): by the time this downgrade could plausibly run,
`schedule_executions` rows referencing these five schedules may already exist. There is no FK
constraint from `schedule_executions.schedule_id` to `schedule_schedules.id` (confirm against the
Phase 1 DDL), so the plain `DELETE` does not need a corresponding executions cleanup to succeed —
it would simply leave orphaned audit rows pointing at a deleted schedule, which is consistent with
how `schedule_executions` already outlives soft-deleted schedules via the app-level API (`DELETE
/v1/schedules/<id>` is a soft-delete, not a hard delete, and doesn't touch executions either).
No change needed; noted for the downgrade's author.

NEVER run `alembic upgrade`/`downgrade` (repo rule) — create and edit the file only.

Verify: `alembic -c alembic.ini heads` → single head; `python3 -m py_compile` on the new revision;
confirm all five `target_queue` values match `commonoutline.QueueName{Billing,AI,Customer}Request`
string values exactly (they must, to pass `bin-schedule-manager`'s `isValidTargetQueue` allowlist
check — no `bin-common-handler` change is needed per design §3, but a typo here would silently
fail forever since nothing creates these rows through the validated CRUD API path).

## Step 7 — Service docs sync

Per root CLAUDE.md's service-docs-sync table (route/event changes in `pkg/listenhandler/main.go`
and `cmd/*/main.go` require a same-commit `docs/architecture.md` update) for all three touched
services:

- `bin-billing-manager/docs/architecture.md`: routing table gains `/v1/accounts/top_up`,
  `/v1/failed_events/retry`; events section drops the two ticker descriptions.
- `bin-ai-manager/docs/architecture.md`: routing table gains `/v1/aipromptproposals/expire`;
  events section updated (startup sweeps stay documented, ticker sweep entry removed).
- `bin-customer-manager/docs/architecture.md`: routing table gains
  `/v1/customers/cleanup_unverified`, `/v1/customers/cleanup_frozen_expired`; events section
  drops both ticker descriptions.

Re-extract via `bash docs/reference/extractor.sh <service-dir>` per service, then hand-verify the
routing table matches the new `pkg/listenhandler/main.go` exactly (the extractor's whitelist
gaps found during Phase 1 — e.g. it didn't know `schedule`/`execution` as event-package names —
may need the same manual patch-up here if it doesn't recognize `account`/`aipromptproposal`
event types cleanly; verify, don't assume).

`bin-schedule-manager/docs/domain.md` or `CLAUDE.md`: no change required — Phase 2 adds rows to
the existing seeded-schedule pattern, doesn't change the engine; the "Housekeeping dogfoods the
engine" language in `bin-schedule-manager/CLAUDE.md` referred to `execution-retention`/
`database-backup` specifically and doesn't need updating for unrelated services' schedules.

Verify: `bash scripts/check-service-docs.sh` (or trust the PostToolUse hook) shows no warnings for
the three touched services; `make lint-docs` at repo root.

## Step 8 — Full verification + cross-service sanity

- Re-run the full 5-step workflow one final time in all three touched service directories
  (`bin-billing-manager`, `bin-ai-manager`, `bin-customer-manager`) with all steps' changes
  present together, to catch any cross-step interaction — in particular, confirm
  `cmd/billing-manager/main.go` compiles cleanly after both Step 1's and Step 2's edits together
  (Step 2's `run()`/`runListen`/`runSubscribe` restructure is the highest-risk interaction point
  in this plan).
- `bin-dbscheme-manager/bin-manager`: `alembic -c alembic.ini heads` → single head (re-verify
  after Step 7's doc changes haven't touched migrations, this is just a final sanity check).
- No `bin-common-handler` changes in this ticket (design §3) — confirm `git diff` touches nothing
  under `bin-common-handler/`.
- Manual-execute smoke check (design §10): for each of the 5 new schedule rows, once seeded in a
  local/throwaway environment, exercise `POST /v1/schedules/<id>/execute` against
  `bin-schedule-manager` and confirm the response count matches expectations against known fixture
  data — this is a manual verification step for the PR description, not an automated test, since
  it requires a running multi-service environment.

## Step 9 — Code review loop, PR

- Code review loop per session policy: minimum 3 rounds, 2 consecutive approvals, max 30.
- Pre-PR: `git fetch origin main`; `git merge-tree` conflict check; `git log --oneline HEAD..origin/main`.
- Commit/PR body lists affected projects: `bin-billing-manager:`, `bin-ai-manager:`,
  `bin-customer-manager:`, `bin-dbscheme-manager:`, `monorepo:` (docs sync). PR title
  `VOIP-1284-Migrate-tickers-to-schedule-manager`. No AI attribution. Single PR. NO merge without
  explicit user authorization.
- PR description explicitly notes: (a) two dbhandler-layer bug fixes shipped as part of this
  migration (`AccountTopUpTokens` CAS + signature change to `(applied bool, err error)`,
  `CustomerAnonymizePII` status guard) with a one-line explanation of the production impact each
  fixes; (b) the `AccountTopUpTokens` concurrency proof is by code inspection + sequential test,
  not a race test, due to sqlite's lack of `SELECT ... FOR UPDATE` support (Step 1 rationale);
  (c) sandbox guard removal is tracked as a separate follow-up PR in the sandbox repo, sequenced
  after this PR's images are released and pinned (design §9) — not part of this PR.

## Acceptance mapping (design §1/§3 ↔ ticket)

| Criterion | Where proven |
|---|---|
| All 5 jobs fire via bin-schedule-manager, no ticker goroutines remain | Steps 1-6 |
| Billing monthly top-up proven single-fire under 2 replicas | Step 1 sequential CAS test + code-inspection argument (row lock + CAS); not a race test, see Step 1/9 rationale |
| Customer frozen-expiry proven single-fire under 2 replicas | Step 5 dbhandler-layer double-fire test |
| Sandbox replica-guard warnings removed for billing/ai | Explicitly deferred to a separate sandbox-repo PR (design §9) — not this PR |
| Per-job metrics visible; execution audit rows present | Inherited from Phase 1 engine, no new work — confirmed in Step 8 manual smoke check |
| Monorepo conventions: unit tests, docs sync, review loops | Steps 1-9 |

## Explicitly out of scope (this PR)

- campaign-manager / queue-manager (design §2 — VOIP-1282 concluded neither needs schedule-manager)
- `idempotency_key`/`reference_id` mismatch cleanup in billing ledger (design §8, separate ticket)
- Unbounded `CustomerListFrozenExpired` query fix (design §8, separate ticket)
- campaign-manager swallowed re-arm errors, queue-manager lost-timeout gap (design §8, VOIP-1282 follow-ups)
- Stale campaign/queue-manager documentation (design §8, VOIP-1282 follow-up, doc-only)
- ai-manager startup one-shot sweeps (design §6 — stay per-boot, unchanged)
- Sandbox guard removal (design §9 — separate, later, separate-repo PR)
- `FreezeAndDelete`'s call-site behavior itself is unchanged by this PR (only the shared
  dbhandler method it calls gains a guard) — its new observable behavior under a race is a
  documented side effect (§5.5, Step 5), not a scoped feature of this PR
