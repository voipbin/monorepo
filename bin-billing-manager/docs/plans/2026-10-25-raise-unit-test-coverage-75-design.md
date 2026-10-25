# bin-billing-manager unit test coverage to 75% (VOIP-1652)

## 1. Ticket review and analysis

Verdict: valid. Measured at origin/main accc03e00 (this document is the only difference) with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in the service directory; no test failures. Hand-written code drops `gens/`, `mock_*.go` and `Code generated ... DO NOT EDIT` files; duplicate blocks are merged with the max count.

Result: hand-written 73.88% (3064/4147 statements), raw 64.2%. 75% needs 3111 covered (47 more), the recommended 76% needs 3152 (88 more). Earlier coverage PRs of this service (VOIP-1647, 1649, 1651) are merged; no open PR touches it.

| package | hand-written % | uncovered |
|---|---|---|
| cmd/billing-control, cmd/billing-manager | 0.0 | 652 + 174 (main wiring, not targeted) |
| pkg/dbhandler | 88.1 | 122 |
| pkg/subscribehandler | 83.9 | 36 |
| pkg/listenhandler | 94.8 | 35 |
| pkg/billinghandler | 93.3 | 19 |
| pkg/accounthandler | 97.6 | 18 |
| internal/config | 84.5 | 13 |
| pkg/failedeventhandler, paddlehandler, cachehandler, models | 91 to 97 | 14 |

Findings from the code (all uncovered statements listed per file):
- Reachable: subscribehandler unmarshal and handler-error returns of 7 files (about 36); billinghandler error returns of `BillingEnd`/`BillingStart` propagation in call, recording, tts events and free-call end (about 15); accounthandler error propagation, unknown plan type and already-processed idempotency returns (about 14); dbhandler exec-failure returns, `AccountSetStatus`, balance and token wrapper functions, `FailedEvent*` exec errors, `billing_paddle.go` (about 40); config `LoadGlobalConfig`/`InitPrometheus` (about 8); constructors (about 8).
- Not reachable with valid input, not targeted: `PrepareFields` and `ToSql` error returns, `json.Marshal` failure returns in listenhandler and paddlehandler, `cmd/*`.
- The realistic reachable total (about 125) exceeds the 88 needed, so the goal is achievable without touching `cmd/`.

## 2. Plan

Only `*_test.go` files are added inside bin-billing-manager (plus this document). No product code, signature, visibility, `go.mod` or `go.sum` change (go-sqlmock is already a dependency).

Order, measuring after each group: (1) subscribehandler, (2) billinghandler and accounthandler, (3) dbhandler, (4) config and constructors only if still below 76%. Stop rule: if all groups are used and the total is 75% or more but below 76%, raise the PR and record the shortfall; if below 75%, stop and report without lowering the target.

Rules for every test: table-driven where the cases share a shape, `Test_<function>` naming, gomock controller named `mc` with `defer mc.Finish()`, synthetic values only, no sleep, no network, no wall clock, no package globals touched. Errors are asserted with `errors.Is` or an exact message fragment against the injected cause, not only `err != nil`. Expectations are re-read from the code when writing.

Cases:
- subscribehandler: for each uncovered `processEvent*` function, invalid JSON returns an error and the billing/account handler mock is not called (no EXPECT set); a handler error is returned wrapped and `errors.Is` finds the injected cause.
- billinghandler: `EventCMCallProgressing`, `EventCMCallHangup`, recording start/stop and tts start/stop return the wrapped error when `BillingStart` or `BillingEnd` fails (db mock error injected one level down); `BillingEnd` for call-extension cost types returns an error when `BillingSetStatusEnd` fails and when the re-read fails; `Get` maps `dbhandler.ErrNotFound` to a not-found error and passes other errors through.
- accounthandler: `IsValidBalanceByCustomerID` and `IsValidResourceLimitByCustomerID` propagate lookup errors; `IsValidBalance` get error; unknown plan type returns the error and no db write is expected; an already-processed Paddle event id returns nil with no balance write; account create db error is logged and the account is still returned.
- dbhandler (sqlmock, db closed in defer, `ExpectationsWereMet` checked): exec failure of `AccountCreate`, `AccountUpdate`, `AccountDelete`, `AccountSetStatus`, `FailedEventCreate/Update/Delete`, `BillingSetStatus` and similar returns the wrapped cause; `AccountSetStatus` success writes status and update time with exact arguments (sqlite fixture); the wrapper functions (`AccountAddBalance`, `AccountSubtractBalance`, `AccountSubtractBalanceWithCheck`, token variants) apply the right sign and check flag (insufficient balance returns `ErrInsufficientBalance` only for the `WithCheck` variants).
- config and constructors: `LoadGlobalConfig` reflects values bound through viper and is idempotent (`once`); `InitPrometheus` with an empty endpoint returns without registering; constructors return non-nil handlers with the injected dependencies.

Verify:
1. Hand-written total at least 76% (75% minimum), per-package before/after table, hand-written and raw totals and the exclusion method in the PR body.
2. `go test ./...` passes, `golangci-lint run` passes, `scripts/check-test-conventions.sh` passes, `git diff --stat origin/main...HEAD` lists only `bin-billing-manager/` files, `go.mod`/`go.sum` unchanged.
3. Mutation sample: for at least five representative tests (one per group), the guarded product line is temporarily changed in a scratch copy and the test must fail; the results are recorded in the card.
