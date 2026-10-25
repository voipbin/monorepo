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
- Reachable (classified by reading each uncovered block): subscribehandler unmarshal and handler-error returns of 7 files (about 36); billinghandler error propagation of `BillingStart`/`BillingEnd` in call, recording and tts events and the free-call end path (about 10), plus `Get` not-found mapping and constructors; accounthandler error propagation, unknown plan type for the plan-change functions that take the plan as a parameter (`paddle.go` 126 and 176; the free-plan lookup at 229 uses a constant key and is not reachable), already-processed idempotency returns and the number-valid success branch (about 13); dbhandler about 84 of 122 (exec failures, row scan failures, `ErrNotFound`, `rows.Err`, customer id parse failure, begin/commit/read failures of the ledger transactions, the unwrapped wrapper functions, `AccountSetStatus`), through sqlmock and the sqlite fixtures; the `PrepareFields` error of the `*Update` functions (a field value that cannot be serialized) and the `ApplyFields` error of `AccountList`/`BillingList` (an unsupported filter value type); listenhandler short-URI 400 of the Paddle portal session; paddlehandler request-build, transport and body-read failures through `httptest`; billinghandler duration observation when both billing times are set.
- Not reachable with valid input, not targeted: `PrepareFields` error of the `*Create` functions and of the ledger transactions (9 statements: Create 3, ledger 6; the 3 `*Update` ones are reachable as above) with a struct input, most `ToSql` error returns (24), config `LoadGlobalConfig` and `InitPrometheus` (they use a `sync.Once` global, the default HTTP mux and a real listener, which the test rules exclude) and the `bindConfig` error returns (reachable only with a nil flag), `json.Marshal` failure returns of listenhandler responses and of the paddlehandler request (no field can fail to serialize), `cmd/*`.
- Summing the per-group estimates gives roughly 160 reachable statements, well above the 88 needed (the estimates are approximate; the stop rule below is what decides), so the goal is achievable without touching `cmd/`; the plan below picks the cheapest groups first.

## 2. Plan

Only `*_test.go` files are added inside bin-billing-manager (plus this document). No product code, signature, visibility, `go.mod` or `go.sum` change (go-sqlmock is already a dependency).

Order, measuring after each group: (1) subscribehandler, (2) billinghandler and accounthandler, (3) dbhandler, (4) paddlehandler httptest cases and listenhandler short-URI cases only if still below 76%. Stop rule: if all groups are used and the total is 75% or more but below 76%, raise the PR and record the shortfall; if below 75%, stop and report without lowering the target.

Rules for every test: table-driven where the cases share a shape, `Test_<function>` naming, gomock controller named `mc` with `defer mc.Finish()`, synthetic values only, no sleep, no external network (loopback httptest only), no wall clock, no package globals touched. Errors are asserted with `errors.Is` or an exact message fragment against the injected cause, not only `err != nil`. Expectations are re-read from the code when writing.

Cases:
- subscribehandler: for each uncovered `processEvent*` function, invalid JSON returns an error and the billing/account handler mock is not called (no EXPECT set); a handler error is returned wrapped and `errors.Is` finds the injected cause.
- billinghandler: `EventCMCallProgressing`, `EventCMCallHangup`, recording start/stop and tts start/stop return the wrapped error when `BillingStart` or `BillingEnd` fails (db mock error injected one level down); `BillingEnd` for call-extension cost types returns an error when `BillingSetStatusEnd` fails and when the re-read fails; `Get` maps `dbhandler.ErrNotFound` to a not-found error and passes other errors through.
- accounthandler: `IsValidBalanceByCustomerID` and `IsValidResourceLimitByCustomerID` propagate lookup errors; `IsValidBalance` get error; unknown plan type returns the error and no db write is expected; an already-processed Paddle event id returns nil with no balance write; account create db error is logged and the account is still returned.
- dbhandler (sqlmock, db closed in defer, `ExpectationsWereMet` checked): exec failure of `AccountCreate`, `AccountUpdate`, `AccountDelete`, `AccountSetStatus`, `FailedEventCreate/Update/Delete`, `BillingSetStatus` and similar returns the wrapped cause; `AccountSetStatus` success writes status and update time with exact arguments (sqlite fixture); the wrapper functions (`AccountAddBalance`, `AccountSubtractBalance`, `AccountSubtractBalanceWithCheck`, token variants) apply the right sign and check flag (insufficient balance returns `ErrInsufficientBalance` only for the `WithCheck` variants).
- constructors: `NewBillingHandler`, `NewAccountHandler`, `NewListenHandler`, `NewFailedEventHandler` return non-nil handlers carrying the injected dependencies (checked through the unexported fields, since the tests are in-package).
- paddlehandler: invalid base URL, transport failure and a body read failure through `httptest` (loopback only, no external network) return the wrapped error; listenhandler: a too-short Paddle portal URI returns 400 without calling the account handler.

Verify:
1. Hand-written total at least 76% (75% minimum), per-package before/after table, hand-written and raw totals and the exclusion method in the PR body.
2. `go test ./...` passes, `golangci-lint run` passes, `scripts/check-test-conventions.sh` passes, `git diff --stat origin/main...HEAD` lists only `bin-billing-manager/` files, `go.mod`/`go.sum` unchanged.
3. Mutation sample: for at least five representative tests (one per group), the guarded product line is temporarily changed in a scratch copy and the test must fail; the results are recorded in the card.
