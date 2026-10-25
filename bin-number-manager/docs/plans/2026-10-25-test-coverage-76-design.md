# bin-number-manager unit test coverage to 76% (VOIP-1652)

## 1. Analysis

Ticket review: valid. Baseline at origin/main accc03e00, measured with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` (no failures). Hand-written coverage excludes `gens/`, `mock_*.go` and files whose header has `Code generated ... DO NOT EDIT`; duplicate blocks are merged by max count. Hand-written 1721/2282 = 75.42% (raw 65.89%). 76% needs 1735 covered statements (+14); 77% needs +36. The count can vary by one statement between runs (goroutine timing in `subscribehandler.Run`).

| package | covered/stmts | % | missing |
|---|---|---|---|
| cmd/number-control | 0/323 | 0.0 | 323 |
| cmd/number-manager | 2/118 | 1.7 | 116 |
| pkg/listenhandler | 342/418 | 81.8 | 76 |
| pkg/dbhandler | 204/225 | 90.7 | 21 |
| pkg/subscribehandler | 60/71 | 84.5 | 11 |
| pkg/numberhandler | 510/515 | 99.0 | 5 |
| internal/config | 78/81 | 96.3 | 3 |
| other 7 packages | 525/531 | 98.9 | 6 |

Testability: `cmd/*` is process wiring (DB, Redis, RabbitMQ, cobra) and is out of scope. Marshal and `ToSql` failure branches cannot be reached with real inputs and are out of scope. The remaining reachable misses are error and boundary paths of `listenhandler`, `subscribehandler.Run`, `dbhandler` (a scan failure and the `NumberUpdate` field preparation failure; `rows.Err` and `RowsAffected` branches are not reachable with sqlite), `NewNumberHandler` and `setSerialize`. `internal/config` error branches are not reachable (flags are registered by `bindConfig` itself). The other packages leave a few single statements (request construction errors, a feature-flag mapping branch, a duplicate-candidate `continue`) that are not targeted.

## 2. Plan

New test files only, inside bin-number-manager, no product code change, no go.mod change. File and helper names are checked against the existing tests (`newPrivateDB`, `newClosedDB`, `Test_errorResponse_*`) before writing to avoid collisions.
1. `pkg/listenhandler`: the ID handlers (`processV1NumbersIDGet/Delete/Put/FlowIDsPut`, `processV1NumbersIDMetadataPut`) are called directly with a 3 segment URI (`/v1/numbers`) and must return 400 without touching the number handler (through `processRequest` that URI is routed elsewhere). Business handler errors are mapped through `errorResponse` for Post (both type branches), Delete, Get, Put, FlowIDsPut, Renew, List, Count and AvailableNumbers (both type branches); assertions use literal expected status codes and bodies: typed error gives the `cerrors.ToResponse` status and `DataTypeVoipbinError`, wrapped `dbhandler.ErrNotFound` gives 404, untyped gives 500. Invalid filters return 400 for List. PUT with no fields falls back to `Get`. `processRequest` returns 404 for an unknown URI, and 400 only for the plain error returned by available numbers on invalid filter JSON (number handler not called, gomock strict). `Run` returns the wrapped error when `QueueCreate` fails and starts `ConsumeRPC` on success.
2. `pkg/subscribehandler`: `Run` returns an error for each of `QueueCreate`, `TopicCreateWithKind` and `QueueBind`; the failing pattern is the one reported and no later bind or consume call is made. `processEvent` is called synchronously with an unknown publisher or type and must not touch the number handler; `processEventRun` returns nil.
3. `pkg/dbhandler`: with `newPrivateDB`, a row whose id column is not a valid UUID makes `numberGetFromRow` fail, asserted through `NumberGet` (cache mock returns a miss so the DB path runs), `NumberList` and `NumberGetsByTMRenew` with the error wrapped. If sqlite does not make `ScanRow` fail for such a row, the case is dropped and the existing-numbers scan failure with a NULL number is tried instead. `NumberUpdate` with a field value that cannot be JSON encoded (a channel) returns the prepare error without touching the database.
4. `pkg/numberhandler`: `NewNumberHandler` returns a handler carrying the injected dependencies. `pkg/cachehandler`: `setSerialize` with an unmarshalable value (a channel) returns an error and does not write to the cache. `internal/config` is out of scope: `bindConfig` registers every flag itself before binding, so its error branches cannot be reached with real input.

Goroutine rules: success paths of the two `Run` functions call the consume mock from a goroutine, so the mock `DoAndReturn` closes a channel and the test waits on it with a `select` and `time.After` timeout; no `time.Sleep`. Metric collectors are registered once in package init, so the tests neither re-register collectors nor assert metric values.

Rules: `Test_` names, table driven, gomock controller `mc` with `defer mc.Finish()`, no `t.Parallel()`, synthetic values only, no network.

## 3. Verify

- Hand-written total at least 76.0% (expected about 77%: listenhandler about +30, subscribehandler about +6, dbhandler about +4 to +10, numberhandler +2 to +5, cachehandler +1), measured with the same method as section 1 (exclusions, max count merge); the per-package before/after table goes into the PR body.
- Full verification: `go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `go test -race` on the touched packages, `golangci-lint run`, `scripts/check-test-conventions.sh`. `git diff --stat origin/main...HEAD` is limited to bin-number-manager and shows no go.mod or go.sum change.
- Mutation sample, done in a scratch copy only (the implementation worktree is never changed): flip `len(uriItems) < 4`; drop the `errorResponse` mapping of one handler; stop the `QueueBind` loop at the first error (or continue after it); drop the error wrap of a scan failure in dbhandler. Each must fail at least one new test. Only a summary goes into the PR body, without locations.
