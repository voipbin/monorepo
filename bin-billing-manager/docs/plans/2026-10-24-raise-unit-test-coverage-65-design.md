# bin-billing-manager unit test coverage to 65% (VOIP-1649)

## 1. Ticket review and analysis

Verdict: valid. Measured at origin/main bffdd69e3 (the worktree code is identical to it; this document is the only difference) with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in the service directory (local vendor not used; no test failures). Hand-written code excludes `gens/`, `mock_*.go` and files whose header has `Code generated ... DO NOT EDIT`; duplicate blocks are merged with the max count.

Result: hand-written 64.05% (2656/4147 statements), raw 55.62% (2656/4775). 65% needs 2696 covered (at least 40 more statements); the ticket recommends 66% (2738, at least 83 more) as margin for CI differences, and the working goal is 67% (2779, at least 123 more) so that the planned first group alone leaves a margin above that.

| package | hand-written % | covered/total |
|---|---|---|
| cmd/billing-control | 0.0 | 0/652 |
| cmd/billing-manager | 0.0 | 0/174 |
| internal/config | 84.5 | 71/84 |
| models/account | 94.4 | 17/18 |
| models/billing | 56.5 | 13/23 |
| pkg/accounthandler | 93.9 | 705/751 |
| pkg/billinghandler | 93.3 | 266/285 |
| pkg/cachehandler | 0.0 | 0/71 |
| pkg/dbhandler | 65.5 | 672/1026 |
| pkg/failedeventhandler | 95.0 | 96/101 |
| pkg/listenhandler | 84.3 | 571/677 |
| pkg/paddlehandler | 93.4 | 57/61 |
| pkg/subscribehandler | 83.9 | 188/224 |

The previous coverage PR of this service (VOIP-1647) already covered the listenhandler, accounthandler and subscribehandler groups that its design listed; its reserve groups remain. Main findings from the code:

- `pkg/dbhandler/account_paddle.go` has 228 uncovered statements, the largest testable block. `AccountPaddleAddCredit`, `AccountPaddleSubtractCredit` and `AccountPaddleTopUpTokens` use raw `SELECT ... FOR UPDATE`, which sqlite cannot run, so they cannot use the sqlite fixtures of the other dbhandler tests. The package already uses go-sqlmock (in go.mod) for the same pattern in `accountAdjustCreditWithLedger`, so no new dependency is needed. `AccountGetByPaddleSubscriptionID` goes through `AccountList` and can use the sqlite fixtures.
- Each of the three transaction functions has the same branches: begin error, no row (`ErrNotFound`), other read error, update error, insert error, duplicate-key insert error (returns nil, the deferred rollback discards the balance change), commit error, success with a post-commit cache refresh whose result is discarded.
- The `PrepareFields` and `ToSql` error returns of the three functions (about 6 statements) cannot be reached with valid input and are not tested, so the realistic upper bound of this file is about 220 of 228.
- Duplicate-key detection is a substring check on the driver error text for the MySQL form (`Duplicate entry`) and the sqlite form (`UNIQUE constraint failed`) in `pkg/dbhandler/billing.go` (`isDuplicateKeyError`, 0%). `BillingCreate` uses it too (its duplicate branch is uncovered); the sqlite fixture has a unique index on the idempotency key, so a second `BillingCreate` with the same key reaches that branch without sqlmock.
- Not worth the cost: `cmd/*` main and wiring (connect to MySQL, Redis, RabbitMQ). `pkg/cachehandler` (0/71) has real logic (serialization and key rules) on a concrete `*redis.Client`; it could be tested without a new dependency (the exported `*redis.Client` accepts a custom dialer, or a closed port reaches the connection error branches), but the cost is high relative to the other groups, so it stays the last reserve.
- Reserve, testable with existing mocks and fixtures: `models/billing/webhook.go` (0/9, pure field mapping of the externally exposed message), dbhandler `billing.go` error branches (55 missed; the transaction branches of `BillingConsumeAndRecord` and `AccountTopUpTokens` use `FOR UPDATE` and need sqlmock, the rest can use the sqlite fixtures), Paddle webhook handlers in `pkg/listenhandler/v1_hooks_paddle.go` (44 missed) and `v1_accounts.go` (49 missed), `pkg/accounthandler/paddle.go` (33), dbhandler `AccountSetStatus` and balance/token wrappers (about 20), `cmd/billing-control` pure cobra builders.

Behavior that looks questionable is pinned as it is and is not changed (product code is out of scope).

## 2. Plan

Only `*_test.go` files are added or extended inside bin-billing-manager (plus this document). No product code, signature, visibility or dependency change (`go.mod` and `go.sum` stay unchanged).

Measurement: the command above plus an aggregation script that drops `gens/`, `mock_*.go` and `Code generated ... DO NOT EDIT` files and merges duplicate blocks with the max count; the command and exclusion rule are written in the PR body with the per-package before/after table, hand-written and raw totals.

Order and stop rule: (1) dbhandler `account_paddle.go`, then measure. About 205 statements are expected (the 228 uncovered minus the 6 unreachable and the statements that stay unreached after the post-commit refresh), which alone passes the 67% goal. If below 67%, add one reserve group at a time and measure after each (listenhandler Paddle webhook handlers, accounthandler `paddle.go`, dbhandler `AccountSetStatus`). If all reserves are used and the total is 65% or more but below 67%, the PR is raised and the shortfall is recorded. If below 65%, stop and report with the remaining uncovered code, without lowering the target.

Rules for every test:
- Table-driven where the cases share a shape, `Test_<function>` naming, gomock controller named `mc` with `defer mc.Finish()`. Synthetic UUIDs and values only. No `time.Sleep`, no wall clock (the util handler mock returns a fixed time), no network, no parallel tests, no package globals touched.
- sqlmock: the db is closed in `defer`; queries are matched with a short distinctive fragment (`regexp.QuoteMeta`), `WithArgs` is exact for ids (`accountID.Bytes()`) and for the update arguments computed by the function (new balance, fixed `now`, and for top-up the first day of the next month at 00:00 UTC), and `ExpectationsWereMet()` is checked. The insert is matched on `INSERT INTO billing_billings` and its argument list is asserted by positional `sqlmock.Argument` matchers or plain `WithArgs` so that the fields that matter (transaction type, reference type, signed amount, balance snapshots, idempotency key) are checked, not only that an insert happened.
- Errors are asserted with `errors.Is` against the injected cause or `ErrNotFound`, not only `err != nil`. The duplicate-key case asserts a nil return, that `Commit` is not expected (sqlmock fails on an unexpected commit), and that a rollback is expected.
- Rollback expectations: every case that fails after `Begin` and before `Commit` (read error, no row, update error, insert error, duplicate key) expects `ExpectRollback()`. The begin-error case has no rollback. The commit-error case must not expect a rollback (`database/sql` treats the already finished transaction as done without calling the driver), so `ExpectationsWereMet` would fail.
- Insert arguments: `SetMap` orders the 22 columns alphabetically (account_id, amount_credit, amount_token, balance_credit_snapshot, balance_token_snapshot, billable_units, cost_type, customer_id, id, idempotency_key, rate_credit_per_unit, rate_token_per_unit, reference_id, reference_type, status, tm_billing_end, tm_billing_start, tm_create, tm_delete, tm_update, transaction_type, usage_duration). A test helper (in the test file) defines that column order once and builds 22 positional `sqlmock.Argument` matchers (one per column, each holding the column name and its expected value) that are passed to `WithArgs`, so all 22 arguments are asserted. Expected values are computed from the test inputs (account id, customer id, idempotency key, amount, fixed `now`, expected new balance) written out in the case, not produced by the code under test. Types as seen by the driver: uuid as `[]byte`, integers as `int64`, times as `time.Time`, enums as `string` (`cost_type` is the empty string for `CostTypeNone`), unset `tm_delete` and `tm_update` as `nil`.
- The post-commit cache refresh is not mocked: its failing query result is discarded by the function, as in the existing ledger tests. This means the refresh itself is not verified by these tests; they assert the transaction only.

Per-function cases (expectations re-read from the code when writing):
- `AccountPaddleAddCredit`: success (credit +amount, billing row top-up/paddle credit purchase, snapshots, commit); begin error; no row gives `ErrNotFound`; read error wrapped; update error wrapped and no insert; insert error wrapped; duplicate-key insert error (both message forms: MySQL "Duplicate entry" and sqlite "UNIQUE constraint failed") returns nil with rollback; commit error wrapped.
- `AccountPaddleSubtractCredit`: the same cases; success asserts credit minus amount (negative result allowed), refund type, negative billing amount.
- `AccountPaddleTopUpTokens`: the same cases; success asserts token reset to the amount, next top-up at the first of the next month (the success case uses a fixed non-December `now` with a day other than 1 and a non-zero time of day, for example the 17th at 10:30:45, so that a result mixing in the day or the time of day is caught; a second case uses a December `now` and expects the first of January of the next year at 00:00 UTC), the txn type passed in is stored, credit snapshot kept, token snapshot is the new amount; the existing token balance and the new token amount are different values, so a snapshot taken from the wrong one is caught.
- `AccountGetByPaddleSubscriptionID` (sqlite fixtures): found; list error from a separate connection created for the case (an independent sqlmock whose query returns an injected error; the shared `dbTest` is never closed) is returned wrapped (`errors.Is` on the injected error, not `ErrNotFound`); not found gives `ErrNotFound`; a soft-deleted account with the same subscription id is not returned (the deleted account has a newer `tm_create` than the active one, so removing the filter changes the result of `Limit 1` ordered by `tm_create desc`). `AccountList` with an empty token calls `TimeGetCurTime` on the util mock, so the case sets a fixed time string in the format used by `ctx_rows_test.go` (`"2026-01-01 00:00:00.000000"`) that is later than every fixture `tm_create`; creating the fixtures and deleting one with `AccountDelete` need their own `TimeNow` and cache expectations, stated explicitly in the case. Fixture ids, customer ids (the existing list tests filter by customer id), subscription ids and idempotency keys are synthetic and unique across the package, because the sqlite in-memory database is shared by all tests of the process.
- `isDuplicateKeyError`: nil, MySQL text, sqlite text, unrelated error. `BillingCreate` twice with the same idempotency key and different ids on the sqlite fixture (a same-id second call would hit the primary key instead of the idempotency index): the second call returns nil, calls `TimeNow` only and not `BillingSet`, and the first call expects `TimeNow` once and, after the insert, the refresh that reads the row back and calls `cache.BillingSet` once with the stored billing (exact argument), the second call expects `TimeNow` once and no `BillingSet`; a re-read by idempotency key (`BillingGetByIdempotencyKey`) returns the first call's id and fields, so the second call did not overwrite it.

Verify:
1. Hand-written total at least 67% by the measurement above (65% minimum per the stop rule), with the per-package before/after table.
2. Root CLAUDE.md verification: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`, plus `go vet ./...`, `bash scripts/check-test-conventions.sh` (from the repository root; it checks committed added lines) and `git diff --exit-code go.mod go.sum` (also right after the coverage measurement, because `-mod=mod` can touch `go.sum`).
3. Flakiness: `for i in $(seq 20); do go test -race -count=1 ./pkg/dbhandler || break; done` passes (a new process per iteration, because `-count=N` in one process reuses the shared sqlite database and the existing tests use fixed ids; that limitation is in the existing tests and is not changed). The new sqlmock tests also pass with `-run 'Test_AccountPaddle|Test_isDuplicateKeyError' -race -count=20`.
4. Mutation sample on a scratch copy (never the worktree); each mutant must compile, the original passes, and the named test must fail:

| mutation | file | test that must fail |
|---|---|---|
| `+` to `-` in the new credit of `AccountPaddleAddCredit` | pkg/dbhandler/account_paddle.go | the success case of Test_AccountPaddleAddCredit |
| remove the duplicate-key branch (treat as error) in `AccountPaddleSubtractCredit` | pkg/dbhandler/account_paddle.go | the duplicate-key case of Test_AccountPaddleSubtractCredit |
| `ErrNotFound` mapping removed in `AccountPaddleTopUpTokens` | pkg/dbhandler/account_paddle.go | the no-row case of Test_AccountPaddleTopUpTokens |
| `BalanceTokenSnapshot` set to the current token balance instead of the new amount | pkg/dbhandler/account_paddle.go | the insert-argument matcher in the success case of Test_AccountPaddleTopUpTokens |
| the passed txn type replaced by a constant | pkg/dbhandler/account_paddle.go | the case that passes the adjustment type in Test_AccountPaddleTopUpTokens |
| `newBalance := currentCredit - amountMicros` to `+` in `AccountPaddleSubtractCredit` | pkg/dbhandler/account_paddle.go | the success case of Test_AccountPaddleSubtractCredit |
| `1` to `now.Day()` in the `time.Date` call | pkg/dbhandler/account_paddle.go | the success case of Test_AccountPaddleTopUpTokens |
| `Month()+1` to `Month()+2` | pkg/dbhandler/account_paddle.go | the success case (a non-December month) and the December case of Test_AccountPaddleTopUpTokens |
| refund billing amount sign flipped | pkg/dbhandler/account_paddle.go | the success case of Test_AccountPaddleSubtractCredit |
| `_ = tx.Commit()` instead of returning its error | pkg/dbhandler/account_paddle.go | the commit error case of Test_AccountPaddleAddCredit |
| `deleted` filter removed in `AccountGetByPaddleSubscriptionID` | pkg/dbhandler/account_paddle.go | the soft-deleted case of Test_AccountGetByPaddleSubscriptionID |
| `UNIQUE constraint failed` removed from `isDuplicateKeyError` | pkg/dbhandler/billing.go | Test_isDuplicateKeyError |

   The results (mutant, failing test name) are attached to the PR body. Each new test file is read to confirm every test asserts on a returned value, a mock argument or a call count.
5. `git diff --stat origin/main...HEAD` lists only files under bin-billing-manager, and only `_test.go` files plus this document.
