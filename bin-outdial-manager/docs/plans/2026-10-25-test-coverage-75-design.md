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

Hard to test (not attempted): `main`, `runDaemon`, `initSignal`, `initProm`, `createDBHandler`, `run` of `cmd/outdial-manager` and `initHandler`, `initCache`, `initOutdialHandler` and the `run*` bodies of `cmd/outdial-control` after `initHandler`: they connect to a real database, Redis and RabbitMQ. Not testable without a product code change: `PrepareFields` of `*Create` and the `ToSql` branches in dbhandler (inputs cannot make them fail). They stay uncovered.

Testable gaps:
- dbhandler (about 60 of the 89): `Update` `PrepareFields` errors (value that cannot be marshaled), `Exec`/`Query` error returns of Create, Update, Delete, List, Get for the three entities (sqlmock, helper `newMockHandler` already exists), `ErrNotFound` on empty result, scan error on a row with a wrong column set, cache-write failure and cache-read hit/miss branches of `*UpdateToCache`, `*SetToCache`, `*Get`, the by-activeflow/by-reference call getters, empty `token` default in the three List functions, empty `fields` early return in the three Update functions.
- cmd/outdial-control (about 165): `resolveUUID` (empty, invalid, valid), `printJSON` (marshal error with a channel value, success), the seven `cmd*` constructors (use, flags, defaults), `initCommand` (sub-commands registered, persistent infrastructure flags bound), `runCreate` validation errors that return before `initHandler` (bad customer id, bad campaign id, empty name).
- cmd/outdial-manager: `runListen` with the mock sock handler (11 statements: the QueueCreate error case, and the success case where `Run` starts `ConsumeRPC` through a channel-synchronised mock; `runListen` returns nil in both).
- listenhandler: `Run` (QueueCreate error; success starts ConsumeRPC with the queue name; ConsumeRPC error is logged), `NewListenHandler` wiring (`errorResponse(nil)` is already covered by `error_response_test.go`; the remaining 2 statements of `errorResponse`, the `ToResponse` failure branch, are practically unreachable for a valid typed error and are excluded), the invalid-body branch of each `v1*` not yet covered.
- constructors `NewOutdialHandler`, `NewOutdialTargetHandler`, `NewOutdialTargetCallHandler`: the returned object carries the given dependencies (asserted through the unexported fields).
- `Bootstrap` error branch needs a failing flag lookup and is excluded; `models/outdial` `ConvertStringMapToFieldMap` error branch (2 statements) is covered by a known key with a value that cannot be converted (for example `campaign_id` set to a non-UUID string); an unknown key is passed through and does not reach it. `Bootstrap` branches at the flag/env binding calls are excluded because the keys are fixed constants.

Estimated (statement counts of the cover profile blocks): cmd/outdial-control about 174 by the listed items (about 165 after leaving out the body of the `migrationhandler.New` closure in `initCommand`, which needs a database and is not run) (`initCommand` 39, `resolveUUID` 7, `printJSON` 5, the seven `cmd*` constructors 114 [create 24, get 12, list 18, update-basic 18, campaign 15, data 15, delete 12], `runCreate` validation before `initHandler` about 9). listenhandler `Run` (including the error branch of the `ConsumeRPC` goroutine, synchronised by a channel), `NewListenHandler` about 13; three constructors 12; `runListen` 11. These alone give about 200 against +76 (75.0%) and +96 (76%). dbhandler is additional margin, not a requirement: of its 89 uncovered statements about 60 are reachable with sqlmock (the `*Update` `PrepareFields` branches via a map value that cannot be marshaled, such as a slice holding a channel; exec/query errors; `ErrNotFound`; scan and `RowError`; `ApplyFields` errors with an unsupported filter value; cache branches) and about 29 are not (the `PrepareFields` branch of `*Create` and `*Delete`, which get fixed inputs, and the `ToSql` branches). Further reserve: listenhandler invalid-body branches (about 39), `processRequest` default 404 branch (4), `initSignal` driven by a signal sent to its channel, `config.LoadGlobalConfig` run once in one test (about 5). All figures in the PR table are hand-written figures; raw figures are reported separately (raw dbhandler 68.9 and cachehandler 55.9 differ from the hand-written 83.9 and 100).

## 2. Design (test plan)

New `_test.go` files only; no product code, go.mod or go.sum change; names `Test_<Func>`, gomock controller `mc`, synthetic values only, no network, no shared mutable state except viper in `cmd/outdial-control` (each test resets the keys it sets with `viper.Reset()` in `t.Cleanup`).

| file | verify |
|---|---|
| pkg/dbhandler/error_paths_test.go | per function with sqlmock: exec error is returned wrapped (`errors.Is` to the injected error) and no cache call happens (mock cache without EXPECT); query error likewise; empty result gives `ErrNotFound` for the three `*GetFromDB`; wrong column set gives scan error naming the function; `*Get` returns the cache value without any SQL when the cache hits, falls back to SQL on a miss and still returns the row when the cache write fails; `*Update` with a value that cannot be marshaled (a slice holding a channel) returns the prepare-fields error with no SQL; `*List` with an unsupported filter value returns the apply-filters error with no SQL; a `RowError` injected into the rows returns the row error and not `ErrNotFound`; `*Update` with empty fields returns nil with no SQL and no cache call; List with empty token asks `TimeGetCurTime` and binds that value (checked through `WithArgs`); Create/Update/Delete return nil when the cache refresh fails |
| cmd/outdial-control/main_test.go | `resolveUUID`: empty gives "is required" naming the label, invalid string gives "invalid format" quoting the value, valid returns the same UUID; `printJSON` returns the marshal error for an unmarshalable value and nil for a struct; `cmd*` constructors: `Use`, registered flag names and defaults (`limit` 100) are asserted; `initCommand`: sub-commands `outdial` (7 children by name) and `migrate` exist, and `--database_dsn` etc. are persistent flags; `runCreate` returns a wrapped error for each invalid input and does not reach `initHandler` (no network) |
| cmd/outdial-manager/main_test.go | `runListen` with `sockhandler.NewMockSockHandler`: `QueueCreate` error gives nil and `ConsumeRPC` is not called |
| pkg/listenhandler/run_test.go | `Run`: `QueueCreate("<queue>", "normal")` error is returned wrapped; success returns nil and `ConsumeRPC` is called with the queue, consumer name and prefetch (synchronised by a channel, no sleep); `NewListenHandler` returns a handler carrying the three dependencies |
| pkg/listenhandler/ invalid body | for each `v1*` function not yet covered, a malformed JSON body gives a non-2xx status through `processRequest` and the business handler mock has no EXPECT; the default 404 branch for an unknown URI |
| models/outdial/convert_test.go | `ConvertStringMapToFieldMap` returns an error for `campaign_id` set to a non-UUID string and passes an unknown key through unchanged |
| pkg/*handler/main_test.go (3 files) | the constructors return objects whose dependency fields are the given mocks (field by field, not whole-struct equality with a copy built the same way) |

Verify steps: (1) hand-written total at least 76% with `$S/hand.py` (aggregation as in section 1; per-package before/after table in the PR); (2) `go test -race -count=3 ./...`, `golangci-lint run`, `scripts/check-test-conventions.sh` clean; (3) `go mod tidy && go mod vendor && go generate ./...` leaves `git status` without go.mod, go.sum or generated changes; `git diff --stat origin/main...HEAD` lists only files under bin-outdial-manager; (4) mutation sample in a scratch copy, each must fail a named new test: drop the `ErrNotFound` return in one `*GetFromDB`; swap the cache-hit check in one `*Get`; change `resolveUUID` invalid-format message condition (return nil error on invalid); change the `limit` default; skip the empty-fields early return in one Update; pass the wrong queue type to `QueueCreate` in `Run`; (5) no test only calls without asserting a result, none compares a value with itself.
