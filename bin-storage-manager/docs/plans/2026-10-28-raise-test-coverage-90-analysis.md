# bin-storage-manager: raise unit test coverage to 90% (analysis)

## 1. Ticket review

Verdict: valid. The hand-written coverage (generated mocks and `DO NOT EDIT` files excluded) is 86.0% (2542 statements, 356 uncovered) on the base `origin/main`. Reaching 90% needs about 102 more statements, 90.5% about 115.

Most of the uncovered statements are in `cmd/storage-control` (161) and `cmd/storage-manager` (115). The remaining 80 are spread over `pkg/dbhandler`, `pkg/filehandler`, `pkg/listenhandler`, `internal/config` and small leftovers.

## 2. Why the command packages were not reachable

The run functions of both commands open a database connection, a message broker connection and a migration engine before any handler logic runs. The shared library now provides replaceable factories for these three (`databasehandler.SetConnectFunc`, `sockhandler.SetRabbitFactory`, `migrationhandler.SetEngineFunc`), each returning a restore function. Defaults are unchanged.

## 3. Reachable scope

With the factories, the following run in a unit test without a real broker, database or migration engine: `run`, `createDBHandler`, `runService`, `initStorageHandler`, `initAccountHandler`, `initCache`, `initHandler`, and every `run*` subcommand body of `storage-control`. Needed collaborators:

- database: an in-memory sqlite database created from the existing test schema scripts, returned by the connect factory;
- message broker: the generated `MockSockHandler`;
- cache: an in-process Redis server (`miniredis`, test-only; the version already used by other services);
- object storage: the storage client in no-auth mode through the emulator host variable with an unreachable loopback address, so no request leaves the process.

Roughly 240 of the 276 uncovered command statements are reachable this way (process entry points and signal handling excluded), against the 115 needed; so the package-level work is limited to the two command packages. Other packages are touched only if the measured total stays below 90.5%.

## 4. Constraints

- No product code change. Test files and `go.mod`/`go.sum` (test-only module) only.
- Tests that swap process-global state do not use `t.Parallel` and restore it with `defer`/`t.Cleanup`.
- The configuration singleton, the process environment and the default HTTP mux are shared, so each test sets and restores what it changes and uses unique values.
- No fixed ports, no host temp files outside `t.TempDir`, no signal delivery. The service's own done channel (buffered, filled before the call) ends `run`.
- `run` defines flags on the command it receives, so each test builds a new command. The configuration singleton loads only once per process, so tests assign `*config.Get()` directly and restore it, and do not depend on test order.
- `initProm` registers on the default HTTP mux, so each call uses a unique endpoint, and it listens on a loopback address obtained from a listener that was opened and closed. Its goroutine cannot be stopped and is allowed to live until the process ends.
- The in-memory sqlite database uses one connection. Tests clear `GOOGLE_APPLICATION_CREDENTIALS` with `t.Setenv` so a developer machine does not change the result.
- The existing command tests (input validation, flag definitions, listener and subscriber start) stay as they are; the new tests cover the paths behind the connections. The process entry points, signal registration and the last-resort branches stay uncovered.
