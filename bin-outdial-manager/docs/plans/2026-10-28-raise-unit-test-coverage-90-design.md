# bin-outdial-manager unit test coverage to 90% (VOIP-1663)

## Baseline
Baseline on origin/main at the start of this work: hand-written statements 2021, covered 1761 (87.14%). The hand-written rule excludes `gens/`, `mock_*.go` and files with a `DO NOT EDIT` header. Most of the remaining gap is in `cmd/outdial-control` and `cmd/outdial-manager`, which were out of reach until `bin-common-handler` gained the test seams `sockhandler.SetRabbitFactory`, `databasehandler.SetConnectFunc` and `migrationhandler.SetEngineFunc`. No product code changes; only `_test.go` files, this plan, and the test-only `miniredis` module in `go.mod`/`go.sum` are added.

## Goal
Hand-written coverage of 90.5% or more (at least 1829 of 2021 statements, +68). The denominator stays 2021.

## Test targets
1. `cmd/outdial-control`: `initHandler`, `initCache`, `initOutdialHandler` success and cache-failure paths; the seven `run*` commands against an in-memory sqlite database (success output as JSON, handler errors such as unknown id, list filters); `initCommand` through `Execute` with explicit args (global flag binding in `PersistentPreRunE`); the `migrate status` factory with a fake migration engine.
2. `cmd/outdial-manager`: `createDBHandler` (migration failure, cache failure, success); `run` with a mock sock handler; `runDaemon` (database failure, and success that runs until a stop value is fed to the existing `chSigs` channel); `initSignal` by feeding the same channel; `initProm` with a unique path and loopback port 0, plus the listen-error branch through an address that is already bound.
3. No other package needs new tests (the other packages are already above 95%, the cmd packages hold the gap); the remaining uncovered statements are branches that cannot fail with valid input and process entry points.

## Isolation
- Every test that uses a seam restores it with `defer`/`t.Cleanup` and does not call `t.Parallel`.
- `requesthandler.NewRequestHandler` registers Prometheus metrics without a guard, so each such test swaps `prometheus.DefaultRegisterer` for a fresh registry and restores it.
- `config.Get()` is copied and restored; `LoadGlobalConfig` runs once per process, so tests set every key through viper and the global config directly. Global viper is reset before and after.
- `initProm` registers on `http.DefaultServeMux`: each test uses a unique path and `127.0.0.1:0`. The serving goroutine lives until process exit. The listen-error branch is observed with a logrus hook, not a sleep.
- `initSignal`: no real signal is sent. Tests put a value into `chSigs`, drain `chDone`, and call `signal.Stop(chSigs)`; every test, including the error paths, releases the goroutine it started so that a later test never loses an injected value to a stale one; the order independence is checked with `-shuffle -count`.
- sqlite: one private in-memory database per test (a single connection, so it is not shared), `SetMaxOpenConns(1)`, schema loaded from `scripts/database_scripts`, closed on cleanup.
- Redis: `miniredis` (already used by other services at the same version, imported only from tests). A closed loopback address covers the cache-failure path.
- Sock: the publish paths run in fire-and-forget goroutines, so command tests use a small fake sock handler that counts calls, and the test waits for the expected calls before restoring anything. The manager `run` test uses the gomock sock handler and waits on `ConsumeRPC` through a channel before the controller finishes.
- Values are synthetic: fake IDs, loopback addresses, `test-user`. No host files, fixed ports or real signals.

## Verify
- Hand-written total at least 90.5% with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`.
- `go test -race -count=3 -shuffle=on ./...` passes.
- Mutation sample: a wrong JSON field in the list command, a changed error wrap in one `run*` command, a dropped flag binding in `PersistentPreRunE` and skipping the migration-error return in `createDBHandler` make the new tests fail. The registration of the OS signals is not asserted, because no real signal is sent.
- `go mod tidy` leaves only miniredis and its indirect dependencies as additions in `go.mod`/`go.sum`; `go mod vendor`, `go generate ./...`, `golangci-lint run` and `scripts/check-test-conventions.sh` (run from the repository root) pass; `git diff --stat origin/main...HEAD` shows only `bin-outdial-manager/` files.
