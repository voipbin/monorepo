# bin-outdial-manager unit test coverage 75% (VOIP-1652)

## 1. Ticket review and analysis

Ticket review: valid. Re-measured on origin/main accc03e00 with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`: hand-written 71.25% (1440/2021 statements), raw 61.80%. All tests pass. Hand-written = raw minus `mock_*.go`, `/gens/` and files whose header (before `package`) has `Code generated ... DO NOT EDIT`; duplicate blocks merged with max count. Need +76 statements for 75.0%, +96 for 76%. The branch started on an unrelated commit and was reset to origin/main before any change. Recent history of the path: VOIP-1647 (coverage 60%) and VOIP-1648 (hardening) are merged; no open PR touches this service.

Baseline per package (hand-written, uncovered statements):

| package | before | uncovered |
|---|---|---|
| cmd/outdial-control | 0.0 (0/295) | 295 |
| cmd/outdial-manager | 0.9 (1/114) | 113 |
| internal/config | 86.9 | 8 |
| models/outdial, outdialtarget | 88.5 / 88.9 | 4 |
| pkg/cachehandler | 100 | 0 |
| pkg/dbhandler | 83.9 (465/554) | 89 |
| pkg/listenhandler | 90.1 (544/604) | 60 |
| pkg/outdialhandler | 97.4 | 4 |
| pkg/outdialtargethandler | 96.3 | 4 |
| pkg/outdialtargetcallhandler | 0.0 (0/4) | 4 |

Hard to test (not attempted): `main`, `runDaemon`, `initSignal`, `initProm`, `createDBHandler`, `run` of `cmd/outdial-manager` and `initHandler`, `initCache`, `initOutdialHandler` and the `run*` bodies of `cmd/outdial-control` after `initHandler`: they connect to a real database, Redis and RabbitMQ. Not testable without a product code change: `PrepareFields`/`ToSql` error branches in dbhandler (inputs cannot make them fail). They stay uncovered.

Testable gaps:
- dbhandler (about 60 of the 89): `Exec`/`Query` error returns of Create, Update, Delete, List, Get for the three entities (sqlmock, helper `newMockHandler` already exists), `ErrNotFound` on empty result, scan error on a row with a wrong column set, cache-write failure and cache-read hit/miss branches of `*UpdateToCache`, `*SetToCache`, `*Get`, the by-activeflow/by-reference call getters, empty `token` default in the three List functions, empty `fields` early return in the three Update functions.
- cmd/outdial-control (about 70): `resolveUUID` (empty, invalid, valid), `printJSON` (marshal error with a channel value, success), the seven `cmd*` constructors (use, flags, defaults), `initCommand` (sub-commands registered, persistent infrastructure flags bound), `runCreate` validation errors that return before `initHandler` (bad customer id, bad campaign id, empty name).
- cmd/outdial-manager: `runListen` with the mock sock handler (QueueCreate error is logged and nil is returned).
- listenhandler: `Run` (QueueCreate error; success starts ConsumeRPC with the queue name), `errorResponse(nil)`, `NewListenHandler` wiring, the invalid-body branch of each `v1*` not yet covered.
- constructors `NewOutdialHandler`, `NewOutdialTargetHandler`, `NewOutdialTargetCallHandler`: the returned object carries the given dependencies (asserted through the unexported fields).
- `Bootstrap` error branch needs a failing flag lookup and is excluded; `models/outdial` `ConvertStringMapToFieldMap` error branch (2 statements) is covered by an unknown-key input.

Estimated (hand counts from the cover profile): items that do not depend on dbhandler give about 102 (cmd/outdial-control 71: initCommand 13, resolveUUID 6, printJSON 5, seven cmd* constructors 38, runCreate validation 9; listenhandler Run, NewListenHandler, errorResponse(nil) 17; three constructors 12; runListen 5), which already clears 75.0% (+76) and 76% (+96) by a thin margin only. dbhandler adds the margin: of its 89 uncovered statements about 64 are reachable with sqlmock (outdial about 18, outdialtarget about 18, outdialtargetcall about 28, including the by-reference and by-activeflow getters, `ApplyFields` errors with an unsupported filter value, and `RowError` for the rows/row error branches) and about 25 are not (the `PrepareFields`/`ToSql` branches, 8 each in outdial and outdialtarget). Further reserve: listenhandler invalid-body branches (about 39), the `processRequest` default 404 branch (4), `initSignal` driven by a signal sent to its channel (no network), and `config.LoadGlobalConfig` executed once in one test (about 5; it is a sync.Once singleton, so it is tested once and its values are asserted from viper keys set by the test). All figures in the PR table are hand-written figures; raw figures are reported separately (raw dbhandler 68.9 and cachehandler 55.9 differ from the hand-written 83.9 and 100).

## 2. Design (test plan)

New `_test.go` files only; no product code, go.mod or go.sum change; names `Test_<Func>`, gomock controller `mc`, synthetic values only, no network, no shared mutable state except viper in `cmd/outdial-control` (each test resets the keys it sets with `viper.Reset()` in `t.Cleanup`).

| file | verify |
|---|---|
| pkg/dbhandler/error_paths_test.go | per function with sqlmock: exec error is returned wrapped (`errors.Is` to the injected error) and no cache call happens (mock cache without EXPECT); query error likewise; empty result gives `ErrNotFound` for the three `*GetFromDB`; wrong column set gives scan error naming the function; `*Get` returns the cache value without any SQL when the cache hits, falls back to SQL on a miss and still returns the row when the cache write fails; `*Update` with empty fields returns nil with no SQL and no cache call; List with empty token asks `TimeGetCurTime` and binds that value (checked through `WithArgs`); Create/Update/Delete return nil when the cache refresh fails |
| cmd/outdial-control/main_test.go | `resolveUUID`: empty gives "is required" naming the label, invalid string gives "invalid format" quoting the value, valid returns the same UUID; `printJSON` returns the marshal error for an unmarshalable value and nil for a struct; `cmd*` constructors: `Use`, registered flag names and defaults (`limit` 100) are asserted; `initCommand`: sub-commands `outdial` (7 children by name) and `migrate` exist, and `--database_dsn` etc. are persistent flags; `runCreate` returns a wrapped error for each invalid input and does not reach `initHandler` (no network) |
| cmd/outdial-manager/main_test.go | `runListen` with `sockhandler.NewMockSockHandler`: `QueueCreate` error gives nil and `ConsumeRPC` is not called |
| pkg/listenhandler/run_test.go | `Run`: `QueueCreate("<queue>", "normal")` error is returned wrapped; success returns nil and `ConsumeRPC` is called with the queue, consumer name and prefetch (synchronised by a channel, no sleep); `errorResponse(nil)` gives 500; `NewListenHandler` returns a handler carrying the three dependencies |
| pkg/*handler/main_test.go (3 files) | the constructors return objects whose dependency fields are the given mocks (field by field, not whole-struct equality with a copy built the same way) |

Verify steps: (1) hand-written total at least 76% with `$S/hand.py` (aggregation as in section 1; per-package before/after table in the PR); (2) `go test -race -count=3 ./...`, `golangci-lint run`, `scripts/check-test-conventions.sh` clean; (3) `go mod tidy && go mod vendor && go generate ./...` leaves `git status` without go.mod, go.sum or generated changes; `git diff --stat origin/main...HEAD` lists only files under bin-outdial-manager; (4) mutation sample in a scratch copy, each must fail a named new test: drop the `ErrNotFound` return in one `*GetFromDB`; swap the cache-hit check in one `*Get`; change `resolveUUID` invalid-format message condition (return nil error on invalid); change the `limit` default; skip the empty-fields early return in one Update; pass the wrong queue type to `QueueCreate` in `Run`; (5) no test only calls without asserting a result, none compares a value with itself.
