# bin-route-manager test coverage 90% design

Goal: raise the hand-written statement coverage (generated code and mocks excluded) from 87.53% to at least 90.5%. Only test files are added. No product code, signature or go.mod change.

## Approach

The start paths in `cmd/route-manager` (`run`, `runService`) and `cmd/route-control` (`initHandler`, `initRouteHandler`, every `runRoute*`) were unreachable without a database, a broker and a migration engine. The merged bin-common-handler seams open them:

- `databasehandler.SetConnectFunc` returns a named in-memory sqlite database (`file:<unique name>?mode=memory&cache=shared`, `SetMaxOpenConns(1)`) loaded with `scripts/database_scripts` (the schema `pkg/dbhandler` tests use). `run` closes it; the `route-control` helper closes it in cleanup.
- `migrationhandler.SetEngineFunc` returns a fake `Engine`: `Version` returns version 0 without error, `Up` returns nil, `Close` returns two nil errors; one case makes `Up` fail.
- `sockhandler.SetRabbitFactory` returns `MockSockHandler`. Expected calls: `Connect` once; `TopicCreateWithKind` once (notifyhandler global topic); `EventPublish` (synchronous) for each successful create, update and delete, recorded and compared by event type; in `runService` also `QueueCreate` and a `ConsumeRPC` call made from a goroutine. Paths that end the process are out of scope.
- `requesthandler.NewRequestHandler` registers prometheus collectors without a guard and panics on the second call in one process. A test helper replaces `prometheus.DefaultRegisterer` with a fresh registry and restores it in cleanup; every seam test uses it.
- Redis is a miniredis instance; the address is set on the config singleton and restored.

## Tests

- `cmd/route-manager`: `run` success (blocks on `chDone`, released by the test), migration failure, cache failure; `runService` with a mock cache and mock sock handler.
- `cmd/route-control`: `initHandler` and `initRouteHandler` success; each command success path with seeded rows, argument-validation errors (these need the seams too, because `initHandler` runs first; the existing unreachable-database tests stay), and one handler error per command (unknown id gives not found for get, update and delete; list errors by dropping the table in the test database). DSN and Redis address are changed through the `config.Get()` pointer, because the config is loaded once. The output is captured, parsed back into `route.Route` and compared field by field (id and timestamps only checked for presence); create, update and delete are also checked by reading the row back.
- Small gaps in other packages only if the total needs them.

## Isolation

- Helpers (sqlite loader through `purse` with a relative path to `scripts/database_scripts`, fresh prometheus registry, miniredis) live in `_test.go` files of each of the two `cmd` packages; no shared non-test package is added.
- Seam tests do not use `t.Parallel`; every restore function runs in `t.Cleanup`.
- Each `run` call, including the existing `Test_run` (so `-count=3` does not register one pattern twice), uses a unique metrics endpoint (a package level atomic counter, starting with `/`) and a listen address with port 0, so there is no fixed port and no duplicate registration on the default mux. Config fields changed by a test are restored.
- The metrics listener goroutine of a port 0 `run` stays until the process exits; this is accepted. `run` calls have a timeout so a hang fails the test; channel waits for the `ConsumeRPC` goroutine have a timeout too. The cache failure case uses a closed miniredis address. The comment of the existing `Test_run` about calling `run` once is updated.
- `chDone` is written by the test right before `run` and drained without blocking in cleanup of every `run` test (a `run` that fails early leaves the token). `runService` tests cancel their context in cleanup so the health check goroutine stops. `chSigs` and real signals are not used.
- The health check interval is long, so the loop never touches the database. The `ConsumeRPC` call from a goroutine is awaited through a channel before the mock controller finishes; all other mock calls are synchronous.
- No host files, no network other than loopback, no real broker. Values are synthetic.

## Verify

- Before: hand-written 2190 of 2502 statements; the two `cmd` packages hold 237 of the 312 uncovered statements, so at least 75 statements are needed for 90.5%; covering most of them gives about 95%.
- `GOFLAGS=-mod=mod go test -count=1 -race -coverprofile` and a second run with `-count=3 -shuffle=on` (global state: prometheus, viper, config, default mux). Hand-written total at least 90.5%, per-package before and after table in the PR.
- Show first that a second `NewRequestHandler` call without the registry helper panics.
- The five steps of the repository verification workflow (`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run -v --timeout 5m`).
- Mutation spot checks: break a branch in `runRouteCreate` argument validation, the migration failure propagation in `run`, `initCache` failure handling, and drop an expected `EventPublish`; the new tests must fail.
- `go mod tidy` leaves no diff; `golangci-lint run`; `git diff --stat origin/main...HEAD` lists only `bin-route-manager/` test files and this document, with no go.mod, go.sum or vendor change.
