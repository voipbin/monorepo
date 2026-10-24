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

Raw total including generated code and mocks is 42.1%. 60% needs 1370 covered statements, 61% needs 1393 (+294). The 61% target only leaves a margin for CI rounding differences. The five other packages are already near or above 60%, so they are not targets.

Testability findings:
- `cmd/*/main.go`: process wiring (DB, Redis, RabbitMQ, cobra). Not unit-testable without external services. Excluded from the plan.
- `pkg/requestexternal/telnyx.go`: URLs are hardcoded, but the HTTP client is the package variable `telnyxHTTPClient`. An in-package test can replace its `Transport` with a fake `http.RoundTripper`. No network access and no product code change are needed.
- `numberhandlertelnyx`: depends on the `RequestExternal` interface which has a gomock mock. Directly testable.
- `numberhandlertwilio`: all three methods (`GetAvailableNumbers`, `CreateNumber`, `ReleaseNumber`) only log and return an `unimplemented` error. They do not call `RequestExternal`, so no mock is needed.
- `cachehandler`: uses `*redis.Client`. `miniredis` is not in this module's go.mod; a fake via go-redis `Options.Dialer` is possible but is not planned because the other packages already give enough margin. It is the first fallback if the total is under 61%.

## 2. Plan

New test files only, inside bin-number-manager:
1. `pkg/requestexternal/telnyx_test.go`: for each of the 7 methods, success (request method, path, query, auth header and body are asserted by the fake transport), non-200 status, transport error, malformed JSON, and the input guards (empty country code, limit clamp to 10, size default 10, empty phone number list).
2. `pkg/numberhandlertelnyx/*_test.go`: purchase (order error, lookup error, success conversion), release and update tags (error propagation, token and provider reference id passed), available numbers (error, conversion).
3. `pkg/numberhandlertwilio/*_test.go`: the three unimplemented methods must return an error.
4. If the total is still under 61%, add error-path cases to `numberhandler` and `listenhandler` found by `go tool cover -func`.

Estimated gain: about 270 + 70 + 25 = 365 statements, giving about 64%.

## 3. Verify

- Hand-written total >= 61.0% by the same method; all existing tests pass; `golangci-lint run` passes; `scripts/check-test-conventions.sh` passes.
- Mutation sample: change a status check, a clamp bound and an error wrap in a scratch copy and confirm the new tests fail.
- `git diff --stat origin/main...HEAD` shows only files under bin-number-manager/.
