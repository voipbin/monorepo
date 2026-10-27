# bin-storage-manager: raise unit test coverage to 90% (design)

Follows the analysis in `2026-10-28-raise-test-coverage-90-analysis.md`. Tests only; no product code change.

## Test helpers (test files in each command package)

- `newTestEnv(t)`: `viper.Reset()`, then `t.Setenv` for every setting the command reads (database, Redis, broker address, project, buckets, metrics address; unrelated developer values are overwritten) and clears `GOOGLE_APPLICATION_CREDENTIALS`. Starts an in-process Redis server (`miniredis` v2.36.1, test-only, same version as other modules) and sets `STORAGE_EMULATOR_HOST` to a loopback address that was obtained from a listener and closed again, so a request fails at once. It then consumes the once-only `config.LoadGlobalConfig()` first so later calls cannot overwrite the test's values in any test order (`-shuffle`), and then assigns the needed fields to `config.Get()`; the previous values are restored in cleanup.
- Connect factory: an in-memory sqlite database with a unique name per test (one connection; closed in cleanup, also when the code under test fails after the connect), schema from `scripts/database_scripts_test`.
- Broker factory: a `MockSockHandler` that allows `Connect`, the publish calls, the queue create/bind calls and the consume calls (the consume calls run in goroutines, so they are matched with `AnyTimes` and signal a channel the test waits on), and always returns nil from the topic declaration calls (a failure there ends the process). No broker error case is built for the constructors.
- Engine factory: a small fake `migrationhandler.Engine` with configurable errors.
- The default metrics registerer is replaced by one that accepts a repeated registration, because the request handler registers its collectors at every construction and panics on the second one.
- Factories are restored in reverse order; no `t.Parallel`. Flags of the `run*` bodies are set with `viper.Set` and cleared by `viper.Reset` in cleanup (the bodies are called directly, without the root command's pre-run). Command output is captured with `captureStdout`. The done channel is filled only by tests that expect `run` to finish and is drained in cleanup.

## Cases

cmd/storage-manager:
- `createDBHandler`: connect error; engine open error; engine `Up` error; cache connect error (Redis server closed); success returns a usable handler.
- `runService`: file handler error (credential path missing in `t.TempDir`); cache connect error; listener start error; subscriber start error; success with both consumers started (wait on channels).
- `run`: database failure and service failure return the error; success ends through the done channel. Each call uses a new command and a metrics endpoint taken from an atomic counter.
- `initProm`: a free loopback address (listener opened and closed) serves the endpoint; when the address is held, a log hook waits for the first failure, the holder is released, and the endpoint is polled with a bounded deadline until it serves. A serving goroutine stays until the process ends (it is allowed to leak); the test only guarantees a unique endpoint, a loopback address, and that the response comes from its own handler (checked by a unique endpoint path) so another process on a reused port cannot satisfy it.

cmd/storage-control:
- `initCache`, `initStorageHandler`, `initAccountHandler`, `initHandler`: connect, cache and file handler errors, and success.
- `run*` bodies with success and handler-error cases where the result can be produced offline: file get, list, delete; account create, get, list, delete; recording delete. Recording get and file create reach the object store for their result, so only their error paths behind the connections are tested. The object store requests of delete fail at once on the closed loopback address and each handler call is bounded by a deadline. Success data is created through the cache or the sqlite database; JSON output is checked field by field.

If the measured total is below 90.5%, the next candidates are the remaining error branches of `pkg/dbhandler` and `pkg/filehandler` that need no new module.

## Verify (completion criteria)

1. Hand-written total >= 90.5% with `GOFLAGS=-mod=mod go test -count=1 -coverprofile`, generated files excluded, same flags for the before/after table.
2. `go test -race -count=3 -shuffle=on ./cmd/...` and `go test -race -shuffle=on ./...` pass, each new test passes alone with `-run`, and the only listeners left are the metrics servers of `initProm` (loopback, process lifetime).
3. The root verification workflow runs completely (`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test`, `golangci-lint`); `go mod tidy` adds only test-only modules (miniredis and its dependencies); no product file changes; `go vet`, `golangci-lint` clean.
4. Mutation sample, at least one per function group (`createDBHandler`, `runService`, `run`, `initProm`, `init*`, `run*`): invert a branch, drop an error return, change an order or a JSON field in a scratch copy; the new tests must fail each time.
5. Assertions compare against independently written expectations (fixture values, error identities), not values produced by the code under test.
