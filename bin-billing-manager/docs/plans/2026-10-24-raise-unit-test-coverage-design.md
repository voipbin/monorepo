# bin-billing-manager unit test coverage to 60% (VOIP-1647)

## 1. Ticket review and analysis

Verdict: valid. Measured at origin/main d68e695b5 with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in the service directory (local vendor not used; no test failures). Hand-written code excludes `gens/`, `mock_*.go` and files whose header has `Code generated ... DO NOT EDIT`; duplicate blocks are merged with the max count.

Result: hand-written 57.58% (2388/4147 statements), raw 50.01%. Reaching 61% needs 2530 covered, so at least 142 more statements.

| package | hand-written % | covered/total |
|---|---|---|
| cmd/billing-control | 0.0 | 0/652 |
| cmd/billing-manager | 0.0 | 0/174 |
| internal/config | 84.5 | 71/84 |
| models/account | 94.4 | 17/18 |
| models/billing | 56.5 | 13/23 |
| pkg/accounthandler | 84.2 | 632/751 |
| pkg/billinghandler | 93.3 | 266/285 |
| pkg/cachehandler | 0.0 | 0/71 |
| pkg/dbhandler | 65.5 | 672/1026 |
| pkg/failedeventhandler | 95.0 | 96/101 |
| pkg/listenhandler | 60.6 | 410/677 |
| pkg/paddlehandler | 93.4 | 57/61 |
| pkg/subscribehandler | 68.8 | 154/224 |

Excluded: `main`, `runDaemon` and `run` in `cmd/billing-manager`, the `run*` command bodies, `initHandlers`, `initDB`, `initCache`, `initBillingHandlers` and `main` in `cmd/billing-control` (they connect to MySQL, Redis or RabbitMQ). The `New*Handler` constructors are pure wiring and are left out by choice (an assertion would only compare the struct with itself). `pkg/cachehandler` (71) is excluded on cost: `handler.Cache` is a `*redis.Client`, there is no in-process Redis in go.mod and adding a dependency is out of scope.

Testable and uncovered (gomock mocks of the existing interfaces, no new dependency; counts are statements from the cover profile):
- listenhandler (about 137 in 5 handlers, 17 in `processRequest` routing): `processV1AccountsGet`, `processV1AccountsIsValidBalanceByCustomerIDPost`, `processV1AccountsIsValidResourceLimitByCustomerIDPost`, `processV1AccountsIDPaddlePortalSessionPost`, `processV1BillingGet`, plus error branches of `processV1BillingsGet` and the Paddle hook handlers.
- accounthandler (about 68): `UpdateBasicInfo`, `UpdatePaymentInfo`, `EventCUCustomerFrozen`, `EventCUCustomerRecovered`, `GetByPaddleSubscriptionID`, `SetStatus` invalid status.
- subscribehandler (about 28): `processEventCUCustomerFrozen`, `processEventCUCustomerRecovered`; also `GetEventProcessor`, `SetFailedEventHandler` (4) and the error branches of the other processors.
- cmd/billing-control (about 360, pure cobra command builders `initCommand` (119) and the `cmdXxx()` functions, `resolveUUID` (7), `printJSON` (5)): testable in package main without I/O; reserve.
- cmd/billing-manager (about 38, take interfaces only): `buildFailedEventHandler` (16), `runSubscribe` (7), `runListen` (12), `signalHandler` (3); reserve.
- listenhandler `Run` (11): `sockHandler.QueueCreate` and `ConsumeRPC` are interface calls, mockable like `subscribehandler.Run`.
- small items: `models/billing/webhook.go` `ConvertWebhookMessage`, `CreateWebhookEvent` (about 9), `config.LoadGlobalConfig` (5), `dbhandler.isDuplicateKeyError` (4), dbhandler balance/token wrappers (6).
- dbhandler (reserve): `account_paddle.go` is 228 statements in `AccountGetByPaddleSubscriptionID`, `AccountPaddleAddCredit`, `AccountPaddleSubtractCredit`, `AccountPaddleTopUpTokens`; `AccountSetStatus` (14). go-sqlmock is already in go.mod.

Estimated gain from the first three groups: about 250 statements (about 63.6%). This is a lower bound: it leaves out the error and Paddle hook branches (about 50 to 70) and all reserve groups.

Existing behavior is covered as it is and is not changed (product code is out of scope).

## 2. Plan

Only `*_test.go` files are added or extended inside bin-billing-manager (plus this document). No product code, signature, visibility or dependency change (`go.mod` and `go.sum` stay unchanged).

Measurement: run `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in the service directory and aggregate with a script that drops `gens/`, `mock_*.go` and `Code generated ... DO NOT EDIT` files and merges duplicate blocks with the max count. The command and the exclusion rule are written in the PR body.

Order and stop rule: (1) listenhandler, (2) accounthandler, (3) subscribehandler, then measure. The ticket requires 60%; the working goal is 61% as margin against CI rounding and the one-statement run-to-run variation of `subscribehandler.Run`. If below 61%, add one reserve group at a time and measure after each: `listenhandler.Run` and Paddle hook error branches, `models/billing/webhook.go`, dbhandler `AccountSetStatus` and `account_paddle.go` with go-sqlmock, cmd/billing-control builders, cmd/billing-manager small functions. If all reserves are used and the total is 60% or more but below 61%, the PR is raised (the ticket requirement is met) and the shortfall against the 61% margin is recorded in the PR body. If the total is still below 60%, the work stops and is reported with the achieved value and the remaining uncovered code, without lowering the target.

Rules for every test:
- Table-driven, `Test_<function>` naming, gomock controller named `mc` with `defer mc.Finish()`. New tests in existing files must also use the `Test_` prefix.
- Synthetic UUIDs and values only. No `time.Sleep`, no wall clock, no network, no parallel tests, no package globals touched. Tests build the handler with a struct literal and call `processRequest` or `processEvent`, which also checks routing.
- `gomock.Any()` is allowed only for `ctx`, with one exception: the callback argument of `ConsumeRPC` in the `Run` test (a func value cannot be compared); there `DoAndReturn` asserts it is non-nil, and `QueueCreate` (queue name and `"normal"`) and the other `ConsumeRPC` arguments (queue, consumer name and the literal flag and count values) are asserted exactly. Every other argument (ids, filters map, size, token, field map, status) is asserted exactly, and call counts and order are explicit (`Times`, `gomock.InOrder` where order matters). Typed arguments are written with the exact type (`uint64(1000)` for the list size, `map[account.Field]any` with `bool` for `deleted`, `uuid.UUID` for ids). Errors are asserted with `errors.Is` or the exact response, not just `err != nil`.
- Calling level: handlers are called through `h.processRequest` (what production does). A direct handler call (`h.processV1...`) is used only for branches that `processRequest` cannot reach. The expected return of each branch is fixed from the code before writing the case. Branches that routing makes unreachable are not tested.

Cases per function (expectations are re-read from the code in the implementation):
- listenhandler: the account list, billing list, billing get, portal session, by-customer validator handlers and their routing, each with success, invalid input and handler error cases, asserted through `processRequest` with exact mock arguments.
- accounthandler: `UpdateBasicInfo`, `UpdatePaymentInfo`, `GetByPaddleSubscriptionID`, `SetStatus`, `EventCUCustomerFrozen` and `EventCUCustomerRecovered`, with success and error cases and exact filter maps, status values and call counts.
- subscribehandler: the customer frozen and recovered processors, `GetEventProcessor` and `SetFailedEventHandler`, covered through `processEvent` and `processEventRun`.
- reserve groups (Paddle hook error branches, `listenhandler.Run`, dbhandler with go-sqlmock): one case per uncovered branch with exact arguments, added one group at a time only if needed.

Verify:
1. Hand-written total at least 61% by the measurement above (or 60% or more with the shortfall against 61% recorded in the PR body, per the stop rule); the PR has the per-package before/after table, the hand-written and raw totals, and the exclusion method.
2. Root CLAUDE.md verification (run `git diff --exit-code go.mod go.sum` right after the coverage measurement as well, because `-mod=mod` can touch `go.sum`): `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`, plus `go vet ./...`, `bash scripts/check-test-conventions.sh`, and `git diff --exit-code go.mod go.sum`.
3. Flakiness: `go test -race -count=20 ./pkg/listenhandler ./pkg/subscribehandler ./pkg/accounthandler` passes.
4. Mutation sample on a scratch copy (never the worktree): for each of the new test groups, a mutant of the covered code must compile and make the named test fail; the results are attached to the PR body.
5. `git diff --stat origin/main...HEAD` lists only files under bin-billing-manager, and only `_test.go` files plus this document.

