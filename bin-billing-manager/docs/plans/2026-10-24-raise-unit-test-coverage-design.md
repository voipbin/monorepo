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

Behavior that looks questionable is pinned as it is and is not changed (product code is out of scope).

## 2. Plan

Only `*_test.go` files are added or extended inside bin-billing-manager (plus this document). No product code, signature, visibility or dependency change (`go.mod` and `go.sum` stay unchanged).

Measurement: run `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in the service directory and aggregate with a script that drops `gens/`, `mock_*.go` and `Code generated ... DO NOT EDIT` files and merges duplicate blocks with the max count. The command and the exclusion rule are written in the PR body.

Order and stop rule: (1) listenhandler, (2) accounthandler, (3) subscribehandler, then measure. The ticket requires 60%; the working goal is 61% as margin against CI rounding and the one-statement run-to-run variation of `subscribehandler.Run`. If below 61%, add one reserve group at a time and measure after each: `listenhandler.Run` and Paddle hook error branches, `models/billing/webhook.go`, dbhandler `AccountSetStatus` and `account_paddle.go` with go-sqlmock, cmd/billing-control builders, cmd/billing-manager small functions. If all reserves are used and the total is 60% or more but below 61%, the PR is raised (the ticket requirement is met) and the shortfall against the 61% margin is recorded in the PR body. If the total is still below 60%, the work stops and is reported with the achieved value and the remaining uncovered code, without lowering the target.

Rules for every test:
- Table-driven, `Test_<function>` naming, gomock controller named `mc` with `defer mc.Finish()`. New tests in existing files must also use the `Test_` prefix.
- Synthetic UUIDs and values only. No `time.Sleep`, no wall clock, no network, no parallel tests, no package globals touched. Tests build the handler with a struct literal and call `processRequest` or `processEvent`, which also checks routing.
- `gomock.Any()` is allowed only for `ctx`, with one exception: the callback argument of `ConsumeRPC` in the `Run` test (a func value cannot be compared); there `DoAndReturn` asserts it is non-nil, and `QueueCreate` (queue name and `"normal"`) and the other `ConsumeRPC` arguments (queue, consumer name and the literal flag and count values) are asserted exactly. Every other argument (ids, filters map, size, token, field map, status) is asserted exactly, and call counts and order are explicit (`Times`, `gomock.InOrder` where order matters). Typed arguments are written with the exact type (`uint64(1000)` for the list size, `map[account.Field]any` with `bool` for `deleted`, `uuid.UUID` for ids). Errors are asserted with `errors.Is` or the exact response, not just `err != nil`.
- Calling level: HTTP-style handlers are called through `h.processRequest` (what production does), so a handler returning `(nil, err)` shows up as the default 400 response with no error. The direct handler call (`h.processV1...`) is used only for branches that `processRequest` masks or cannot reach. The expected return of each such branch is fixed per handler from the code before writing the case: `json.Unmarshal` failure (by-customer validators, update and balance handlers) and the `url.Parse` failure of the two list handlers (`processV1AccountsGet`, `processV1BillingsGet`) return `(nil, err)` (400 through `processRequest`); filter parse and filter conversion failures of the list handlers return `(simpleResponse(400), nil)`; the `url.Parse` failure, path-length and nil-UUID branches of `processV1BillingGet` and the short-URI branches return `(simpleResponse(400), nil)`. The billing routing regex has no left anchor, so a URI such as `/x/v1/billings/<uuid>` (path length 4) and `/%zz/v1/billings/<uuid>` (`url.Parse` failure) reach `processV1BillingGet` through `processRequest`; those 400 cases use `processRequest`. Branches the regex makes unreachable (for example `len(uriItems)<4` in the portal handler) are not tested.

Per-function cases (all expectations to be re-read from the code in the implementation):
- listenhandler `processV1AccountsGet`: success with size, token and a typed filter map; filters are read from the request body (page size and token from the query); success asserts the converted typed map (`uuid.UUID`, `bool`); broken filter JSON gives 400 and the handler is not called; a non-UUID `customer_id` gives 400 (`ConvertFilters` error); an unknown filter key is silently dropped and the handler is called without it (separate case); handler `ErrNotFound` gives 404; other handler error gives 500.
- listenhandler `processV1AccountsGet` also gets the `url.Parse` failure case (`/%zz/v1/accounts?...`, 400 through `processRequest`). When the existing `Test_processV1BillingsGet` is extended, the normal row is fixed so that the filter is real and asserted exactly: its current `customer_id` is only in the query string, but the handler reads filters from the body, so the row either puts `customer_id` in `Data` with an exact `uuid.UUID` expectation or drops it from the query and expects an empty map.
- listenhandler `processV1BillingsGet` error branches: `url.Parse` failure (direct call `(nil, err)`, `processRequest` 400); broken filter JSON 400; filter conversion failure 400; handler `ErrNotFound` 404; other handler error 500.
- listenhandler `processV1BillingGet`: success; a non-UUID id is not routed and gives 404 through `processRequest`; nil UUID gives 400 through `processRequest`; the path-length 400 (`/v1/billings/<uuid-A>/v1/billings/<uuid-B>`, handler not called; `/x/v1/billings/<uuid>` would be an equivalent case because the nil UUID check also returns 400, so it is not used for the length check) and `url.Parse` 400 (`/%zz/v1/billings/<uuid>`) cases are reached through `processRequest` because the regex is not left-anchored; handler error gives the mapped error response (404 and 500 cases).
- listenhandler `processV1AccountsIDPaddlePortalSessionPost`: nil UUID gives 400; handler error gives 400 (current behavior, it does not use `errorResponse`); success returns `{"url":...}`.
- listenhandler by-customer validators (balance, resource limit): success with valid true and false, with exact arguments (customer id, billing type, country, count / resource type); handler error mapped (404 and 500); unmarshal error via `processRequest` gives 400 and by direct call gives `(nil, err)`; an empty or non-UUID `customer_id` is passed to the handler as `uuid.Nil` without a 400 (current behavior pinned).
- listenhandler routing: each of the five new handlers is reached through `processRequest` with the right method and URI; unregistered method or URI gives 404.
- accounthandler `UpdateBasicInfo`, `UpdatePaymentInfo`: success (db update with the exact field map, then get, then returned account); update failure returns an error that `errors.Is` the cause (the message is wrapped twice, so the text is not compared) and no get is called; get-after-update failure returns an error that `errors.Is` the cause, and a `dbhandler.ErrNotFound` there becomes the not-found common error (`errors.As`, `BILLING_ACCOUNT_NOT_FOUND`). `GetByPaddleSubscriptionID` is a one-line passthrough: one case each for the exact id argument with the same returned pointer, and the same error (`errors.Is`, not wrapped), kept as a minimum case for the statement count.
- accounthandler `SetStatus`: invalid status returns the `INVALID_ACCOUNT_STATUS` error (checked with `errors.As` to the common error type) and `AccountSetStatus` is not called; db failure; get failure; success publishes `account.EventTypeAccountUpdated` with the fetched account.
- accounthandler `EventCUCustomerFrozen`: list filter map exactly `{customer_id, deleted:false}` with size 1000 and empty token; list error is wrapped and returned and no status call is made; N accounts give N `AccountSetStatus(..., StatusFrozen)` calls; a failing status call for one account does not stop the next and the function returns nil; empty list returns nil. `EventCUCustomerRecovered`: filter map exactly `{customer_id, status:frozen}` (no deleted key), same cases with `StatusActive`.
- subscribehandler `processEventCUCustomerFrozen` and `Recovered`: invalid JSON returns an error and the handler is not called; handler error is propagated unchanged; success passes the decoded customer. `GetEventProcessor` and `SetFailedEventHandler` are covered only through behavior: the function returned by `GetEventProcessor` handles a customer_frozen event and the mock `EventCUCustomerFrozen` is called; for `SetFailedEventHandler` the handler is built as a `&subscribeHandler{...}` literal (it satisfies `SubscribeHandler`, so the type assertion in `GetEventProcessor` holds), the mock failed-event handler is injected, and `processEventRun` is called with an event whose processing fails, expecting `Save(ctx, event, the processing error unchanged)`. The handler starts with a different mock that has no expectations, so a no-op `SetFailedEventHandler` fails on that mock instead of panicking on a nil field. No struct field round trip. The other subscribehandler cases go through `processEvent` (routing plus handler), not the private functions.
- reserve Paddle hook error branches (the uncovered return lines of each function are listed from the cover profile when this group starts; `handlePaddleTransactionCompleted`, `handlePaddleSubscriptionCreated`, `handlePaddleSubscriptionCanceled`, `handlePaddleTransactionRefunded` in `v1_hooks_paddle.go`): each one gets one case with exact response and mock arguments.
- reserve `listenhandler.Run`: `QueueCreate` error is returned and `ConsumeRPC` is not called (`Times(0)` stated explicitly); success returns nil and `ConsumeRPC` is called once with the created queue and the consumer name; a `ConsumeRPC` error does not change the return value (it is only logged in the goroutine). The goroutine call is synchronized with a channel closed in `DoAndReturn` and waited for by the test (the wait is a `select` with a long `time.After` used only as a fail-safe so a regression fails instead of hanging; it is not a timing assertion) (`Times(1)`, never `AnyTimes`), so no mock call can happen after the test ends. This applies to the new tests only; the existing `AnyTimes` tests are not touched.
- reserve dbhandler go-sqlmock: queries are matched with `regexp.QuoteMeta` of a short distinctive fragment, `WithArgs` is exact, and `ExpectationsWereMet()` is checked; no copy of the full query as the assertion.

Verify:
1. Hand-written total at least 61% by the measurement above (or 60% or more with the shortfall against 61% recorded in the PR body, per the stop rule); the PR has the per-package before/after table, the hand-written and raw totals, and the exclusion method.
2. Root CLAUDE.md verification (run `git diff --exit-code go.mod go.sum` right after the coverage measurement as well, because `-mod=mod` can touch `go.sum`): `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`, plus `go vet ./...`, `bash scripts/check-test-conventions.sh`, and `git diff --exit-code go.mod go.sum`.
3. Flakiness: `go test -race -count=20 ./pkg/listenhandler ./pkg/subscribehandler ./pkg/accounthandler` passes.
4. Mutation sample on a scratch copy (never the worktree); each mutant must compile, the original passes, and the named test must fail:

| mutation | file | test that must fail |
|---|---|---|
| portal handler error 400 to 500 (the block right after `PaddleCreatePortalSession`, not the uri or nil UUID checks) | pkg/listenhandler/v1_accounts_paddle.go | Test_processV1AccountsIDPaddlePortalSessionPost |
| remove `deleted:false` from the filter in `EventCUCustomerFrozen` only (not the identical one in `EventCUCustomerDeleted`) | pkg/accounthandler/event.go | Test_EventCUCustomerFrozen |
| `return errSet` after a failed `AccountSetStatus` in Frozen | pkg/accounthandler/event.go | Test_EventCUCustomerFrozen |
| `StatusActive` instead of `StatusFrozen` in Frozen | pkg/accounthandler/event.go | Test_EventCUCustomerFrozen |
| drop the unmarshal error return in `processV1AccountsIsValidBalanceByCustomerIDPost` | pkg/listenhandler/v1_accounts.go | direct-call case of its test |
| `StatusFrozen` instead of `StatusActive` in Recovered | pkg/accounthandler/event.go | Test_EventCUCustomerRecovered |
| swap the Frozen and Recovered routing cases in `processEvent` | pkg/subscribehandler/main.go | the customer_frozen and customer_recovered cases of the subscribehandler tests |
| make `SetFailedEventHandler` a no-op | pkg/subscribehandler/main.go | the failed-event `Save` case |
| `GetEventProcessor` returns a different function | pkg/subscribehandler/main.go | the `GetEventProcessor` case |
| drop the `QueueCreate` error return in `Run` | pkg/listenhandler/main.go | the `QueueCreate` error case of the Run test (a stray goroutine panic log after the failure is acceptable; the test failing is the judgment) |
| drop the nil-UUID check in `processV1BillingGet` | pkg/listenhandler/v1_billing.go | the nil UUID case of Test_processV1BillingGet |
| remove `status:frozen` from the Recovered list filter | pkg/accounthandler/event.go | Test_EventCUCustomerRecovered (exact filter map) |
| remove `return nil, errors.Wrap(...)` of the error branch in `UpdateBasicInfo` (the public wrapper) | pkg/accounthandler/account.go | Test_UpdateBasicInfo |
| same for `UpdatePaymentInfo` | pkg/accounthandler/account.go | Test_UpdatePaymentInfo |
| `processV1AccountsGet` url.Parse failure returns `simpleResponse(500), nil` instead of `(nil, err)` | pkg/listenhandler/v1_accounts.go | the url.Parse case of Test_processV1AccountsGet |
| by-customer balance validator returns 400 when the customer id is `uuid.Nil` | pkg/listenhandler/v1_accounts.go | the pinned non-UUID customer_id case |
| remove the `len(pathParts) != 3` check in `processV1BillingGet` | pkg/listenhandler/v1_billing.go | the double-path 400 case (`/v1/billings/<uuid-A>/v1/billings/<uuid-B>`) |
| remove the `ErrNotFound` to 404 mapping in `errorResponse` | pkg/listenhandler/main.go | the 404 case of `processV1AccountsGet` (the existing `Test_errorResponse_legacyNotFound` also fails; both are expected) |
| skip the invalid status check in `SetStatus` | pkg/accounthandler/account.go | Test_SetStatus invalid status |

   The results (mutant, failing test name) are attached to the PR body. Beyond the sample, each new test file is read to confirm every test asserts on a returned value, a mock argument or a call count, not only that the call happened.
5. `git diff --stat origin/main...HEAD` lists only files under bin-billing-manager, and only `_test.go` files plus this document.
