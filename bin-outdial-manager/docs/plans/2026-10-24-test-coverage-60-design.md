# bin-outdial-manager unit test coverage 60% (VOIP-1647)

## 1. Ticket review and analysis

Ticket review: valid. Re-measured on origin/main d68e695b5 with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`: hand-written 57.37% (1156/2015 statements), raw 49.74%. All tests pass. Hand-written = raw minus `mock_*.go`, `/gens/` and files whose header (before `package`) has `Code generated ... DO NOT EDIT`; duplicate blocks merged with max count. Need about +54 statements for 60.0%, about +74 for 61%.

Baseline per package (hand-written):

| package | before | uncovered stmts |
|---|---|---|
| cmd/outdial-control | 0.0 (0/295) | 295 |
| cmd/outdial-manager | 0.9 (1/114) | 113 |
| internal/config | 86.9 | 8 |
| models/* | 88.5 / 88.9 | 4 |
| pkg/cachehandler | 0.0 (0/95) | 95 |
| pkg/dbhandler | 69.9 (383/548) | 165 |
| pkg/listenhandler | 81.0 (489/604) | 115 |
| pkg/outdialhandler | 78.8 (119/151) | 32 |
| pkg/outdialtargethandler | 74.1 (80/108) | 28 |
| pkg/outdialtargetcallhandler | 0.0 (0/4) | 4 |

Hard to test: the integration wiring of `cmd/outdial-manager` (main, signal, RabbitMQ, Redis and DB wiring) and the `New*Handler` / `NewListenHandler` / `Run` constructors that dial real infrastructure. They are not tested. The pure helpers `resolveUUID` and `printJSON` in `cmd/outdial-control` (viper and stdout only) are cheap candidates but not needed if the target is met.

Testable gaps (cover profile of d68e695b5, uncovered statements):
- cachehandler (95): `getSerialize`, `setSerialize`, 10 Set/Get methods, `Connect`, `NewHandler`, all 0%. It uses `*redis.Client` directly and go.mod has no miniredis, so a client built with `redis.Options.Dialer` returning one end of `net.Pipe` served by a minimal RESP fake (PING, GET, SET) is used. miniredis exists in other modules but adding it changes go.mod and go.sum, which is outside the test-only scope, so it is not used. Fallback if the fake proves unreliable: a client that fails at connect, which only covers error returns.
- dbhandler `OutdialTargetGetAvailable` success path and `outdialTargetGetFromRowForAvailable` (about 50 statements; function is 63.2%, the rows-error path is already covered by sqlmock in `ctx_rows_test.go`). The query uses `HAVING` without `GROUP BY`; this was probed with a throwaway test against the `dbTest` sqlite database of the package tests (go-sqlite3 as pinned in go.mod) and fails there (`a GROUP BY clause is required before HAVING`), so sqlite seeding is not possible and sqlmock (already in go.mod) with the 24 column rows is used. sqlmock verifies the scan mapping and the bound arguments (`WithArgs`), not the SQL threshold semantics; that limitation, and that the query is MySQL-only so sqlite cannot guard it, is stated in the PR.
- dbhandler `*UpdateToCache` and `*SetToCache` (66.7% each, 3 entities, 6 statements) and `*GetFromDB` / `*GetFromCache` 75 to 77%: small contribution, covered through the existing mock cache.
- outdialhandler `Get` 50%, `Delete` 71.4%, `List`/`Create`/`Update*` error returns (28 statements); outdialtargethandler `Get`, `Delete`, `UpdateProgressing` 63.6%, `GetAvailable` 77.8%, `UpdateStatus` 75% (24 statements).
- listenhandler invalid-JSON and handler-error branches of `v1Outdials*` (65) and `v1Outdialtargets*` (24), `processRequest` 85.7%, `errorResponse` 84.6%.

Estimated contribution (statements): cachehandler up to 95 (excluding `NewHandler` about 85), handler error paths about 45, GetAvailable about 50, listenhandler about 60, dbhandler cache about 15. Total about 250 against the 74 needed for 61%, so cachehandler alone almost reaches the target and the rest is margin. Measurement: aggregate the raw profile (drop `mock_*`, `/gens/`, `DO NOT EDIT` header files, merge duplicate blocks by max count); the procedure is written in the PR body.

## 2. Design (test plan)

New test files only, under bin-outdial-manager. No change to non-test .go files, go.mod or go.sum. Names `Test_<Func>` (check-test-conventions), gomock controller named `mc`, synthetic values only, fake addresses only in the reserved 203.0.113.0/24 range, each test builds its own fake (no shared state, safe under `-race -count=3`).

### 2.1 Files and verify items

| file | verify |
|---|---|
| pkg/cachehandler/handler_test.go | Fake redis (see 2.2). For each of the 5 key families Set stores the exact key `outdial:<id>`, `outdialtarget:<id>`, `outdialtargetcall:<id>`, `outdialtargetcall_activeflowid:<activeflow id>`, `outdialtargetcall_referenceid:<reference id>`; the stored value is unmarshaled and its identifying fields compared to the input (not only Set then Get symmetry); SET carries `EX 86400` (asserted from the arguments the fake recorded); Get returns the entity set before; Get of a missing key returns `redis.Nil`; Get of non-JSON stored value returns an unmarshal error; Set against a fake that answers `-ERR` returns the error; `Connect` ping OK and failing; `NewHandler` returns a handler whose `client.Options()` carries the given Addr, Password, DB (no dial at construction) |
| pkg/outdialhandler/outdial_error_test.go | Get: `dbhandler.ErrNotFound` becomes `*cerrors.VoipbinError` with reason `OUTDIAL_NOT_FOUND`, `errors.Is(err, dbhandler.ErrNotFound)` stays true (Wrap), other errors pass through unchanged; Delete: delete error returned and OutdialGet not called and no webhook, get-after-delete error returned and no webhook; Create, UpdateBasicInfo, UpdateCampaignID, UpdateData, List: DB error returned, no webhook (mocks without EXPECT for the skipped calls) |
| pkg/outdialtargethandler/outdialtarget_error_test.go | same shape: Get maps ErrNotFound to `OUTDIAL_TARGET_NOT_FOUND` with Wrap, others pass through; Delete delete-error and get-error; UpdateProgressing and UpdateStatus: update error stops before re-fetch (re-fetch not expected), re-fetch error returned; GetAvailable and GetsByOutdialID pass the five try counts (all different, 1 to 5), outdial id and limit to the DB in order and return DB errors |
| pkg/dbhandler/outdialtarget_available_test.go | GetAvailable with sqlmock (query matched with `regexp.QuoteMeta` of `outdialTargetSelectAvailable` or `QueryMatcherEqual`, reuse helper of `ctx_rows_test.go`): prepare error; query error; `WithArgs(tc0..tc4, outdialID.Bytes(), limit)` with distinct values 1 to 5 so an argument reorder fails; rows with 24 columns, each column a distinct value, are mapped to the target (id, outdial id, name, detail, data, status, try counts 0 to 4, destinations 0 to 4 parsed, TMCreate and TMUpdate set and nil variants, TMDelete); NULL columns keep zero values; invalid destination JSON in each of the five positions returns an error; scan error from a wrong column count returns an error; `rows.Err` is already covered. SQL threshold semantics (`try_count < ?`) cannot be verified with sqlmock and the query is MySQL-only, so sqlite cannot guard it; stated in the PR |
| pkg/dbhandler cache tests | with the mock cache and sqlmock: `OutdialTargetGet`, `OutdialGet`, `OutdialTargetCallGet` (and by activeflow and reference id) return the DB value and nil when the cache write fails (`_ =` path); `Update` and `Delete` return nil when the cache refresh fails; a cache hit returns without any DB expectation; each for the three entities, listing the function and the mock expectation in the test table |
| pkg/listenhandler/*_error_test.go | Through `h.processRequest` (as the existing tests do). The `v1*` functions return `(nil, err)` for malformed JSON and the status code is decided by `processRequest`: a generic error gives 400, `dbhandler.ErrNotFound` gives 404, a `*cerrors.VoipbinError` gives that error's code (NotFound reason gives 404). Cases per endpoint: malformed JSON body gives 400 and the business handler mock has no EXPECT (a call would fail the test); handler generic error gives 400; handler `ErrNotFound` gives 404; handler typed NotFound gives 404. `v1OutdialsGet` additionally: malformed filters JSON and `ConvertStringMapToFieldMap` failure give 400. These tests characterize the current (legacy 400) behavior and are not a statement that 400 is intended |
| optional, only if the target is not met | `resolveUUID` and `printJSON` in `cmd/outdial-control` |

### 2.2 Fake redis

A per-test `net.Pipe` RESP server: each dial starts a goroutine that parses RESP arrays (PING answers `+PONG`, GET answers bulk or `$-1`, SET records key, value and arguments and answers `+OK`, forced error mode answers `-ERR`). Client options: `MaxRetries: -1`, short `DialTimeout`, `Dialer` returning the client end. `t.Cleanup` closes the client and the pipes so no goroutine leaks. go-redis v8 sends no HELLO handshake. If the fake cannot be made stable, cachehandler is dropped and the target is reached from the other rows: handler errors (about 45), GetAvailable (about 50), listenhandler (about 60) and dbhandler cache (about 15) give about 170 against the 74 needed for 61%.

### 2.3 Verify

1. Hand-written total at least 61% by this one-liner run in the service directory: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=$S/cp.out ./... && python3 $S/hand.py $S/cp.out` (`hand.py` aggregates blocks by max count, drops `mock_*`, `/gens/` and `DO NOT EDIT` header files; its source is pasted in the PR body). Per-package before and after table in the PR body.
2. Existing tests pass, also `go test -race -count=3 ./...`; `golangci-lint run` clean; `scripts/check-test-conventions.sh` clean.
3. Mutation sample in a scratch worktree, each must fail a named new test (test names recorded in the PR body): NotFound reason string in outdialhandler Get; swap tryCount1 and tryCount2 in `QueryContext` of GetAvailable; swap two destination scan targets in `outdialTargetGetFromRowForAvailable`; change a cache key prefix; change the cache TTL from 24h; swap two try count arguments in outdialtargethandler GetAvailable; call `OutdialGet` before the delete error return in outdialhandler Delete; map `ErrNotFound` to 400 in `processRequest`.
4. No test only calls without asserting the result or compares a value with itself; checked by the review checklist and the list of assertions in the PR body.
5. After `go mod tidy && go mod vendor && go generate ./...`, `git status` shows no go.mod, go.sum or generated file changes. `git diff --stat origin/main...HEAD` lists only new test files and this document under bin-outdial-manager.

### 2.4 Implementation notes from review

- The fake compares the SET TTL argument case-insensitively (go-redis sends `ex` and `86400`).
- dbhandler cache-miss paths run `*GetFromDB` again from `*UpdateToCache`; the sqlmock table lists the expected number of identical queries per case.
- `outdialTargetGetFromRowForAvailable` converts `id` and `outdial_id` with `uuid.FromBytes([]byte(string))`, so the mocked row values must be exactly 16 bytes (use `uuid.Bytes()`), otherwise the mapped IDs are zero and the assertion is meaningless.
- Activeflow and reference key tests use different `ID`, `ActiveflowID`, `ReferenceID` so a key built from the wrong field fails.
- The input that makes `ConvertStringMapToFieldMap` fail is confirmed against the code before the case is written.
- In the PR body, "generic error gives 400" is described as the `processRequest` default branch, not as `errorResponse` (which gives 500 for a generic error).
- `v1OutdialsIDAvailableGet` success is already covered by an existing test, so only its malformed JSON case is added. Each mutation records the name of the new test that fails alone.

## 3. Result notes

- The dbhandler cache-branch tests of 2.1 were not written: the hand-written total reached 71.2% without them, so they would only add code beyond the target.
- `go test -count=3` fails in `pkg/dbhandler` before and after this change (existing tests insert fixed ids into a shared in-memory sqlite); the new packages pass `-race -count=3`, and `-race -count=1 ./...` passes.
- All nine mutations of 2.3 were killed by the new tests only (names recorded in the PR body).
