# bin-number-manager unit test coverage to 76% (VOIP-1652)

## 1. Analysis

Ticket review: valid. Baseline at origin/main accc03e00, measured with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` (no failures). Hand-written coverage excludes `gens/`, `mock_*.go` and files whose header has `Code generated ... DO NOT EDIT`; duplicate blocks are merged by max count. Hand-written 1721/2282 = 75.42% (raw 65.89%). 76% needs 1735 covered statements (+14); 77% needs +37.

| package | covered/stmts | % | missing |
|---|---|---|---|
| cmd/number-control | 0/323 | 0.0 | 323 |
| cmd/number-manager | 2/118 | 1.7 | 116 |
| pkg/listenhandler | 342/418 | 81.8 | 76 |
| pkg/dbhandler | 204/225 | 90.7 | 21 |
| pkg/subscribehandler | 60/71 | 84.5 | 11 |
| pkg/numberhandler | 510/515 | 99.0 | 5 |
| internal/config | 78/81 | 96.3 | 3 |
| other 7 packages | 449/451 | 99.6 | 2 |

Testability: `cmd/*` is process wiring (DB, Redis, RabbitMQ, cobra) and is out of scope. Marshal and `ToSql` failure branches cannot be reached with real inputs and are out of scope. The remaining reachable misses are error and boundary paths of `listenhandler`, `subscribehandler.Run`, `dbhandler`, `config` and `NewNumberHandler`.

## 2. Plan

New test files only, inside bin-number-manager, no product code change, no go.mod change.
1. `pkg/listenhandler`: URI too short (`/v1/numbers`, 3 segments) returns 400 for ID Get/Delete/Put/FlowIDsPut/Metadata; business handler errors are mapped through `errorResponse` (typed, wrapped not-found, untyped) for Post, Delete, Get, Put, FlowIDsPut, Renew, List, Count, AvailableNumbers (both type branches); invalid filters return 400 for List; PUT with no fields falls back to `Get`; `processRequest` returns 404 for an unknown URI and 400 for a handler that returns a plain error (invalid filter JSON of available numbers); `Run` returns the wrapped error when `QueueCreate` fails and starts `ConsumeRPC` on success (synchronized by a channel).
2. `pkg/subscribehandler`: `Run` error for each of `QueueCreate`, `TopicCreateWithKind`, `QueueBind` (the failing pattern is the one reported, later calls are not made); `processEventRun` dispatches; `processEvent` ignores an unknown publisher/type without touching the number handler.
3. `pkg/dbhandler`: rows whose id column is not a valid UUID make `numberGetFromRow` fail, asserted through `NumberGet` (DB path), `NumberList` and `NumberListByTMRenew`; scan failure of the existing-numbers query if the sqlite test schema allows a NULL number.
4. `internal/config`: `bindConfig` with a command that lacks the flags returns an error and `Bootstrap` propagates it. `pkg/numberhandler`: `NewNumberHandler` returns a handler carrying the injected dependencies. `pkg/cachehandler`: `setSerialize` with an unmarshalable value returns an error and does not write.

Rules: `Test_` names, table driven, gomock controller `mc` with `defer mc.Finish()`, no `t.Parallel()`, synthetic values only, no network.

## 3. Verify

- Hand-written total at least 76.0% (target about 77%), per-package before/after table in the PR body.
- Existing tests unchanged and passing, `golangci-lint run` clean, `scripts/check-test-conventions.sh` clean, `git diff --stat origin/main...HEAD` limited to bin-number-manager.
- Mutation sample: flip `len(uriItems) < 4`, drop the `errorResponse` mapping of one handler, and swap the `QueueBind` error wrap; each must fail at least one new test.
