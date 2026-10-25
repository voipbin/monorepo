# bin-schedule-manager test coverage plan (VOIP-1652)

## Goal
Raise hand-written statement coverage (excluding gens/, mock_*.go, DO NOT EDIT files) from 74.9% (1522/2032) to at least 76% (1545/2032, +23). Tests only; no production code change.

## Baseline (origin/main accc03e00)
cmd/schedule-control 0/167, cmd/schedule-manager 2/132, config 78/81, backuphandler 125/144, cachehandler 78/79, dbhandler 321/393, dispatchhandler 267/289, listenhandler 313/375, schedulehandler 270/302, subscribehandler 54/56, models 14/14. The two cmd main packages are out of scope (process entry points).

## Approach
dbhandler has 72 uncovered statements; an estimated 25 are reachable by test (to be confirmed by an interim measurement after the dbhandler tests), about 45 are not (query build, row scan, commit failures). Reachable set alone leaves a margin of only 2-4, so listenhandler and schedulehandler tests are a required buffer, added whenever the measured total is below 1545 (checked with the measurement command below).
1. `isDuplicateKeyErr` table: errno 1062 (plain, wrapped), other errno, message containing "1062" only, sqlite UNIQUE text, nil.
2. Closed database, every DB-backed method: error contains that method's own marker (ScheduleGet reports `scheduleGetFromDB`; it reads the cache first, so its mock expects one cache miss; all other methods have a cache mock with no expectation, so any cache call fails). Also ApplyFields errors in the two List methods via a `[]string` filter value (unsupported type).
3. Empty-schema database: BeginTx succeeds, statements fail. Cron trigger fails in the claim update (`scheduleClaimSlot` marker); a non-cron trigger skips the claim and returns the unwrapped insert error, asserted as the original `no such table` text and `isDuplicateKeyErr == false`.
4. Isolation: databases for items 2 and 3 are private sqlite (`file::memory:` without `cache=shared`, `SetMaxOpenConns(1)`), never the shared `dbTest`, which is never closed or reassigned (TestMain owns it).
5. listenhandler: per route, filter parse or convert failure maps to 400, handler error maps to 500, and ErrNotFound on update and delete maps to 404, using the existing mocks. The unreachable json.Marshal failure branches are not targeted.
6. schedulehandler: validation branches and db errors wrapped and returned, with no later step called. Not needed once the measured total passed 1545 after items 1-5 (77.0%), so it is not part of this change.

## Verify (no test code in this stage)
- Measurement: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in the service directory, then the hand-written filter (drop gens/, mock_*.go, DO NOT EDIT; duplicate blocks use max count). Result must be >= 76.0% (>= 1545).
- Standard steps: `go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run`; `go test -race` on touched packages.
- Assertions check a specific marker, status code or call absence, never only `err != nil`.
- Mutation samples (scratch copy), each must fail a new test: drop the `scheduleClaimSlot` error wrap in dbhandler/execution.go; invert `isDuplicateKeyErr` errno comparison in dbhandler/main.go; make a new listenhandler route return 500 instead of the mapped 404 for ErrNotFound and confirm the new route test fails (the existing `errorResponse` unit test does not count).
- `git diff --stat origin/main...HEAD` lists only bin-schedule-manager test files and this document; go.mod and go.sum unchanged.
