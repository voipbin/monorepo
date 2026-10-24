# VOIP-1647 bin-webchat-manager test coverage: issue analysis

## 1. Scope review (verdict: valid)

- Base: `origin/main` d68e695b5, the worktree was fast-forwarded to it (it was one commit behind).
- Baseline re-measured with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`: all tests pass, hand-written 568/1479 = 38.4%, raw 568/1749 = 32.5%. Both match the stated baseline.
- Exclusion rule: files under `gens/`, `mock_*.go`, and files whose header (before `package`) holds `Code generated ... DO NOT EDIT`; duplicate blocks merged by max count. Only `mock_*.go` files match in this service.
- No open or merged PR with VOIP-1647 in the title; recent history of the service shows no test-coverage work.

## 2. Measured baseline per package (covered/statements)

| Package | Covered | Total | % |
|---|---|---|---|
| cmd/webchat-control | 0 | 13 | 0.0 |
| cmd/webchat-manager | 1 | 98 | 1.0 |
| internal/config | 0 | 76 | 0.0 |
| models/message | 1 | 10 | 10.0 |
| models/session | 0 | 9 | 0.0 |
| models/widget | 0 | 9 | 0.0 |
| pkg/cachehandler | 0 | 67 | 0.0 |
| pkg/dbhandler | 363 | 447 | 81.2 |
| pkg/listenhandler | 0 | 483 | 0.0 |
| pkg/messagehandler | 40 | 52 | 76.9 |
| pkg/sessionhandler | 77 | 95 | 81.1 |
| pkg/widgethandler | 86 | 120 | 71.7 |

## 3. Findings

- `pkg/listenhandler` is 483 statements (33% of the hand-written total) with zero tests. It depends only on interfaces that already have generated mocks (`sockhandler`, `widgethandler`, `sessionhandler`, `messagehandler`), so it is fully testable with gomock, no network. Error paths: bad JSON (400), invalid or short URI (400), bad filters (400), handler error mapped through `errorResponse` (404 for `dbhandler.ErrNotFound`, typed `VoipbinError` status, else 500), `processRequest` routing and 404 default, legacy 400 for untyped errors.
- Reaching 60% needs about 320 more statements. listenhandler at about 90% gives about 430, so it alone is sufficient; reach 61%+ with margin through the cheap pure-function packages.
- `models/*/webhook.go` (27 statements): `ConvertWebhookMessage` and `CreateWebhookEvent`, pure functions; externally visible shape, worth asserting (for example internal fields stay out of the payload).
- `internal/config` (76): `Bootstrap` / `LoadGlobalConfig` / `Get` with cobra and viper, testable with environment variables; verifies defaults and env binding.
- `pkg/cachehandler` (67): `h.Cache` is a `*redis.Client` used in two helpers (`getSerialize`, `setSerialize`) plus `Connect`. It is testable with `github.com/alicebob/miniredis/v2`, which other services in this monorepo already use in tests (for example bin-sentinel-manager). bin-webchat-manager's `go.mod` does not list it, so using it adds a test-only module requirement (go.mod and go.sum inside this service). This is optional and is used only if the margin needs it; the planned tests reach the target without it, so it is not part of this change.
- `pkg/listenhandler` risks: `init()` registers a prometheus histogram globally (done once per test binary, tests must not re-register or call init again, they only construct `&listenHandler{}` literals); `Run` starts a goroutine that calls `ConsumeRPC`, so the test synchronizes with a channel signalled from the mock `DoAndReturn` and waits with a timeout; `processRequest` uses `time.Now` only for a metric, no fixed clock needed; the handlers take UUIDs from the URI, so tests use fixed synthetic UUIDs.
- `pkg/dbhandler` (84 uncovered) is tested with a real in-memory sqlite `*sql.DB` (`file::memory:?cache=shared`, schema from `scripts/database_scripts_test/*.sql`); go.mod has no sqlmock. Of the 84 uncovered statements about 69 are error branches and about 15 are not: `NewHandler` body (4, main.go 78-84, never called), cache-hit success returns (message 94 and 137, session 97 and 140, widget 95 and 138), empty-result `ErrNotFound` returns (message 122, session 125, widget 123) and the `len(fields)==0` early returns of `SessionUpdate` and `WidgetUpdate` (session 205, widget 203). Cache-error branches (`return err` after `cache.*`) are reachable with `MockCacheHandler`. Build, exec, scan and `row.Err` branches are reachable only by closing the sqlite handle or dropping a table, so only part of the 69 is realistically coverable. Together with messagehandler (12), sessionhandler (18) and widgethandler (34) uncovered statements this is the fallback pool, and its realistic ceiling is below the raw 84 plus 64.
- `internal/config` (76) has global state: `Bootstrap` calls `logrus.SetFormatter` and `SetLevel` and binds the global viper; `LoadGlobalConfig` is guarded by the package-level `once sync.Once`, so it runs once per test binary and several env cases would see only the first value. A test must call `viper.Reset()` and reassign `once = sync.Once{}` (same package) per case, or be limited to one case; logrus formatter and level are restored after each test. `Bootstrap` also assigns `globalConfig` directly (not through `once`), so the `once` reset affects only `LoadGlobalConfig`; the `BindPFlag` and `BindEnv` error branches are practically unreachable.
- `pkg/cachehandler`: `handler.Cache` is a concrete `*redis.Client`. With miniredis a test can build `&handler{Cache: redis.NewClient(&redis.Options{Addr: mr.Addr()})}` or call `NewHandler(mr.Addr(), "", 0)`; `Connect()` failure is reachable by closing miniredis. Using miniredis needs go.mod and go.sum changes in this service, the full verification workflow (`go mod tidy`, `go mod vendor`, `go generate`, test, lint), and a check that miniredis v2.36.1 (used by bin-sentinel-manager with the same go-redis v8.11.5) resolves here.
- `errorResponse` has four branches: nil error, typed `VoipbinError` converted by `cerrors.ToResponse`, typed error where `ToResponse` itself fails, and `ErrNotFound`, plus the 500 default. `NewListenHandler` creates the real `utilhandler.NewUtilHandler()`; the `utilHanlder` field is not referenced in `v1_*.go` (grep finds none), so struct literals with a nil `utilHanlder` are safe in the handler tests.
- The listenhandler estimate (about 90% of 483) assumes every branch of the three `v1_*.go` files is reachable through mocks; the estimate is an assumption until measured.
- Hard to test and left out of scope: `cmd/*` main and `runDaemon` (signal, RabbitMQ, DB and Redis connections).
- Percentages above are after the exclusion rule, so they differ from the raw `go test` output (for example dbhandler 67.1% raw, because generated and mock code is counted). `migrations` has no statements and `listenhandler/models/request` has no executable statements (only types), so neither moves the total.
- Out of scope: product code behavior changes.
