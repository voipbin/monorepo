# bin-number-manager unit test coverage to 60% (VOIP-1647)

## 1. Analysis

Baseline at origin/main d68e695b5, measured with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`.
Hand-written coverage excludes `gens/`, `mock_*.go` and files whose header has `Code generated ... DO NOT EDIT`; duplicate blocks are merged by max count.

| package | covered/stmts | % |
|---|---|---|
| cmd/number-control | 0/323 | 0.0 |
| cmd/number-manager | 2/118 | 1.7 |
| pkg/requestexternal | 0/305 | 0.0 |
| pkg/numberhandler | 395/515 | 76.7 |
| pkg/listenhandler | 342/418 | 81.8 |
| pkg/dbhandler | 180/225 | 80.0 |
| pkg/numberhandlertelnyx | 0/72 | 0.0 |
| pkg/cachehandler | 0/70 | 0.0 |
| pkg/numberhandlertwilio | 0/28 | 0.0 |
| other 5 packages (internal/config, models/availablenumber, models/number, requestexternal/models/telnyx, subscribehandler) | 180/208 | 86.5 |
| total (hand-written) | 1099/2282 | 48.2 |

Raw total including generated code and mocks is 42.1%. 60% needs 1370 covered statements, 61% needs 1393 (+294). The 61% target only leaves a margin for CI rounding differences. The five other packages are 86.5% in total (`models/number` alone is 52.6%), so they are not targets. Percentages in the table exclude generated code and mocks, so they differ from the per-package `go test` output.

Testability findings:
- `cmd/*/main.go`: process wiring (DB, Redis, RabbitMQ, cobra). Not unit-testable without external services. Excluded from the plan.
- `pkg/requestexternal/telnyx.go`: URLs are hardcoded, but the HTTP client is the package variable `telnyxHTTPClient`. An in-package test can replace its `Transport` with a fake `http.RoundTripper`. No network access and no product code change are needed.
- `numberhandlertelnyx`: depends on the `RequestExternal` interface which has a gomock mock. Directly testable.
- `numberhandlertwilio`: all three methods (`GetAvailableNumbers`, `CreateNumber`, `ReleaseNumber`) only log and return an `unimplemented` error. They do not call `RequestExternal`, so no mock is needed.
- `cachehandler`: uses `*redis.Client`. `miniredis` is not in this module's go.mod; a fake via go-redis `Options.Dialer` is possible but is not planned because the other packages already give enough margin. It is the last resort, after the fallback in section 2.

## 2. Plan

New test files only, inside bin-number-manager:
1. `pkg/requestexternal/telnyx_test.go`: for each of the 7 methods, success (request method, path, query, Authorization header, Content-Type for POST/PATCH and body are asserted by the fake transport), non-200 status, transport error, malformed JSON, `io.ReadAll` failure, the input guards (empty country code, limit boundaries 0/10/11 clamp to 10, size 0 defaults to 10, empty `Data` is an error for `GetByNumber`), each optional query branch on and off (`locality`, `administrativeArea`, `tag`, `number`), a control character in the URL input (`"\x7f"`) to reach the `http.NewRequestWithContext` error of the 6 methods that put input into the URL, GET/DELETE without a body, `IDUpdate` with nil `data` (body `null`), and `GetByNumber` returning the first of several items. Also `NewRequestExternal`.
   Mutation targets: `!= 200` of at least 3 methods, limit clamp (`> 10` to `> 11`; `>= 10` is equivalent and excluded), empty-result guard, Authorization header, Content-Type.
2. `pkg/numberhandlertelnyx/*_test.go`: purchase (order error, lookup error, success conversion), release and update tags (error propagation, token and provider reference id passed, exact `data` map), available numbers (error, conversion), and `NewNumberHandler` (returned type and injected token, connection id, profile id). Handler tests build the struct directly and inject the gomock `RequestExternal`, because `NewNumberHandler` creates the real one.
3. `pkg/numberhandlertwilio/*_test.go`: the three unimplemented methods must return an error; `NewNumberHandler` injects sid and token.
4. Fallback, only if the total is under 61% after 1 to 3: error-path cases for `numberhandler` and `listenhandler` found by `go tool cover -func`. `cachehandler` stays out unless that is still not enough.

Rules for the new tests: `Test_` function names, table driven, gomock controller named `mc` with `defer mc.Finish()`, mock order results non-nil in `NumberPurchase` cases that pass the order step, no testify, no `t.Parallel()`, fake tokens and reserved numbers (`+15550100xxx`) only.
The fake transport replaces the `Transport` field of the shared `*http.Client` `telnyxHTTPClient` (original value is nil) through one helper that restores it with `t.Cleanup`. A `TestMain` installs a fail-closed transport (any request fails the test) for the whole package, each case overrides it, and each case asserts the number of requests that reached the fake; fake responses always have a non-nil `Body`, and an unexpected request fails the test, so nothing can reach the real host. The `IDUpdate` marshal failure (`chan` value) is covered. Only two statements are unreachable without a product code change (the `NewRequestWithContext` error and the marshal error of `NumberOrdersPost`) and they are not targeted. The multi-item success case of `TelnyxPhoneNumbersGet` asserts that each element is a distinct value; this pins the current loop variable semantics and is not a mutation target. The control character goes into `countryCode`, `number`, `id`, and `tag` or `number` for `TelnyxPhoneNumbersGet`. The non-200 case is included for `GetByNumber` too (its debug log runs before the status check), and a `(nil, nil)` lookup in `NumberPurchase` is intentionally not tested (product code would panic). `GetAvailableNumbers` matcher arguments are exact (`""` locality and area, limit), and `NumberUpdateTags` uses a non-empty tags slice with the exact `data` map. Tests do not touch the global prometheus registry.

Estimated gain: telnyx.go up to about 298 of 300 reachable (plan uses about 270), requestexternal/main.go 5, numberhandlertelnyx about 70, numberhandlertwilio about 28 including constructors and init. Total about 365 to 390, giving 64% to 65%. The method tests alone give about 340 which is still above the +294 needed.

## 3. Verify

- Run the 5 verification steps of CLAUDE.md (`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run -v --timeout 5m`), `go test -race -shuffle=on -count=2 ./...`, and `git fetch origin main` then `scripts/check-test-conventions.sh` after the test files are committed (it only inspects committed added lines; confirm the output lists the new test files and not "no changed test files"). `go.mod`/`go.sum` must not change, also after the coverage run with `GOFLAGS=-mod=mod`.
- No network: `go test ./pkg/requestexternal/...` passes inside a network-less namespace (`unshare -rn`), and fails when the fake transport is not installed.
- Hand-written total >= 61.0% by the section 1 command and an aggregation script kept in scratch (its command and exclusion rules are written in the PR body); the per-package before/after table goes into the PR body.
- Mutation sample: mutate the items listed above in a scratch copy and confirm the new tests fail.
- `git diff --stat origin/main...HEAD` shows only files under bin-number-manager/.
