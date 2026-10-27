# VOIP-1659 bin-webchat-manager test coverage

## 1. Ticket review (verdict: valid)

- Base: `origin/main` a47f5a2a1 (the worktree was reset to it; the card cited 320acbfc2).
- Re-measured with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`: all tests pass, hand-written 84.7% (1479 statements), raw 71.6%. Matches the stated baseline.
- Exclusion rule: files under `gens/`, `mock_*.go`, and files whose header holds `Code generated ... DO NOT EDIT`; duplicate blocks merged by max count. Only `mock_*.go` files match here.
- Target 85% needs 6 more statements, 85.5% needs 13 more.

## 2. Analysis

Uncovered hand-written code (statements are approximate, from `go tool cover -func`):

| Area | Uncovered | Reachable without signature change |
|---|---|---|
| `pkg/cachehandler` (`handler.go`, `main.go`) | 67 of 67 | yes, via a miniredis-backed `*redis.Client` (the `handler` struct is built in-package); `Connect` failure by closing the server |
| `cmd/webchat-control` | 6: `main` (4), flag and bootstrap error branches (2) | `initCommand` is already tested in `main_test.go` (structure, success path); the remaining branches are unreachable (`log.Fatalf`, `BindPFlags`/`Bootstrap` errors); the `migrate` DB closure runs only when a migrate subcommand executes, so it is left out |
| `models/*/webhook.go` `CreateWebhookEvent` | 1 each | only the `json.Marshal` failure branch, unreachable, so none |
| `pkg/dbhandler` error branches | 31 | partly (cache errors through `MockCacheHandler`, closed sqlite handle) |
| `cmd/webchat-manager` `main`, `runDaemon`, `initProm`, `run` | 86 | no: signals, RabbitMQ, DB and Redis connections |
| `internal/config` `Bootstrap` | 1 | no: flag binding error |
| `pkg/listenhandler` | 32 (`v1_widgets.go` 12, `v1_sessions.go` 10, `v1_messages.go` 8, `main.go` 2) | mostly error branches, reachable through the existing gomock harness; spare pool if the margin is needed |

Reachable ceiling: cachehandler alone adds about 60 statements, so covering all 67 gives about 89.2%, above 85% without any cmd refactoring. No block is needed.

Dependency: `github.com/alicebob/miniredis/v2` is not in this service's `go.mod`. 22 other services pin v2.36.1 (one uses v2.38.0) with the same go-redis v8.11.5; v2.36.1 is chosen. Adding the same version as a test-only requirement is within the card's allowance; the test must be the only importer, and the expected `go.mod`/`go.sum` diff is the `miniredis/v2` requirement, its `yuin/gopher-lua` indirect requirement, and the matching `go.sum` lines; `go mod tidy` must leave nothing beyond that. Covering all 67 statements needs the error branches too (closed-server Get and Set failures), not only the happy paths.

Risks: new tests must not depend on time or external state. miniredis is started with `miniredis.RunT` (auto-assigned port, cleaned up by the test), TTL is checked through `mr.TTL` and never by waiting or `FastForward`. `pkg/dbhandler` tests share one in-memory sqlite state and fail under `-count=3` with unique-constraint errors (existing behavior), so no test is added there; `-count=1` is the measurement flag. Get paths pass a concrete type pointer as `getSerialize` does, and cover a missing key (`redis.Nil`) and a malformed JSON value. `config` has global state (`once`, global viper), so any config-touching test reuses the existing `resetGlobals` pattern.

## 3. Design

One new file `pkg/cachehandler/handler_test.go` (package `cachehandler`) plus `go.mod`/`go.sum` (miniredis v2.36.1, test-only). No product code change.

Helper (test file only): `newTestHandler(t)` starts `miniredis.RunT(t)` (auto-assigned port) and returns `&handler{Cache: redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})}` and the server. `MaxRetries: -1` and a `context.WithTimeout` on closed-server calls bound the delay. Closing the server early is allowed (the `RunT` cleanup closes again without error; checked by `-race`). The password stays an empty string.

Test data: synthetic fixed UUIDs, fixed UTC `time.Time` literals (never `time.Now`, because monotonic and Location differ after a round trip), and the entity fields relevant to the round trip filled with different non-zero values, ThemeConfig only partly (Identity ID and CustomerID, names, status, direct id, hash, flow ids, theme config, session and message fields). The expected value is built separately from the stored value. `Session.TMDelete` is tagged `json:"-"`, so it is left nil on both sides.

Tests, table-driven where the shape repeats:

- `Test_NewHandler_Connect`: `NewHandler(mr.Addr(), "", 3)` asserted through a `*handler` type assertion (Addr, Password, DB) and `Connect` succeeds; after `mr.Close()` `Connect` returns an error.
- `Test_Set_Get` (widget, session, message): set, assert the key (`webchat:widget:<id>`, `webchat:session:<id>`, `webchat:message:<id>`) exists with a 24h TTL through `mr.TTL`, read back and `reflect.DeepEqual` against the separately built expected value.
- `Test_Set_overwrite`: a second Set with a changed value replaces the first (catches a set-if-absent regression).
- Get miss: unknown id returns `redis.Nil` and a nil result for each type.
- Get malformed: `mr.Set(key, "{not json")` returns a JSON error and a nil result for each type (`null` is not used, it decodes without error).
- Set and Get failure on a closed server for each type.
- `Test_setSerialize_marshalError`: a channel value returns an error and writes no key.

Verify:

1. `GOFLAGS=-mod=mod go test -count=1 -race ./pkg/cachehandler` and `-count=3` pass (own server per test, no shared state, no sleep).
2. Service total from the card command `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` with `mock_*.go` and generated files excluded, merged by max count: at least 85.5% and cachehandler at least 95%.
3. Mutations in a scratch copy only, one at a time, each must fail at least one test (record the failing test name per mutation): the three key prefixes, TTL 24h to 1h, Set to SetNX, drop the DB index or address assignment in `NewHandler`, ignore the `Connect` error, drop the Unmarshal error return in `getSerialize`, drop the Marshal error return and the Set error return in `setSerialize`. After the run the real worktree `git diff` shows no product file change.
4. `go mod tidy` then `git diff -- go.mod go.sum` shows only: `miniredis/v2 v2.36.1`, `yuin/gopher-lua v1.1.1 // indirect`, and their `go.sum` lines. Any other version change (for example go-redis) fails the check. Then `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run -v --timeout 5m`, and `scripts/check-test-conventions.sh` are clean.
5. `git diff --stat origin/main...HEAD` lists only `bin-webchat-manager/` files.

## 4. Out of scope

Product code behavior, signatures, and `cmd/webchat-manager` refactoring.
