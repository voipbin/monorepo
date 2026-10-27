# bin-common-handler test injection seams

Service start paths call `sockhandler.NewSockHandler`, `databasehandler.Connect` and `migrationhandler.Run`/`New`, which open real connections. A service test cannot run the code after them without a broker, a database and a migration engine. The workaround today is a per-service loopback fake broker plus a real database, which is heavy and not shared. This change adds replaceable factories inside the existing public functions. Production behavior and signatures do not change.

## API (additions only)

- sockhandler: `SetRabbitFactory(f func(serverURI string) SockHandler) (restore func())`. `NewSockHandler` calls the factory for `sock.TypeRabbitMQ`; other types still return nil. The default wraps `rabbitmqhandler.NewRabbit` (it returns the `Rabbit` interface, whose method set equals `SockHandler`).
- databasehandler: `SetConnectFunc(f func(dsn string) (*sql.DB, error)) (restore func())`. `Connect` calls it; the default is the current `sql.Open` plus ping.
- migrationhandler: `SetEngineFunc(f func(dsn string, fsys fs.FS, versionTable string) (Engine, uint, error)) (restore func())`. `New` calls it and wraps the returned engine and max version in a `Migrator`. `Run` and the recovery command open closures both go through `New`, so both paths are covered.
- `Engine` is a new exported interface: `Version() (uint, bool, error)`, `Up() error`, `Steps(int) error`, `Force(int) error`, `Close() (source error, database error)`. It is the subset of `*migrate.Migrate` already used by `Migrator`, whose field `m` takes this type. `Migrator.Close` keeps joining both errors. No mock is generated and no existing interface changes. A nil engine from a factory is unsupported (first use panics).
- A nil argument to a setter installs the default. `restore` puts back the factory that was active before the call; call restores in LIFO order (`defer` or `t.Cleanup`).

## Rules

- Each factory is held in a `sync/atomic` pointer written only by its setter, for tests, so production reads stay race free and a factory may call a setter or `New` without deadlock. Each default is an unexported function (`defaultRabbitFactory`, `defaultConnect`, `defaultEngine`), so no default calls itself. A test that sets a factory must not run in parallel with another test of the same package that expects the default.
- Limits: the seams open the start path up to the first process exit or blocking wait inside a service's own run function (for example waiting for a termination signal). The `*sql.DB` comes from the service test (for example a sql mock), which also owns any query expectations of the start path. Commands that connect more than once need a connect function that branches on the dsn. Connections outside these three functions (other databases, cloud clients) are not covered.
- Redis needs no seam: the service cache handlers ping the configured address, which a miniredis instance satisfies. Service configuration singletons are in-package and each service test resets its own.
- The setters are exported because the consumers are other modules; the admission rule is met by the 3 or more services that start through these functions.

## Verification

- Default preservation: default factory yields the same concrete type and behavior; unknown sock type still nil; setter nil and restore return to the default; nested restores; a factory that re-enters its setter; concurrent set and read under `-race`.
- Existing tests do not read `Migrator.m` except `&Migrator{}` with a nil field (`grep -rn 'Migrator{' bin-common-handler --include=*.go` shows only `migrator.go` and `migrator_test.go`). Existing `Test_NewSockHandler` and the integration tests pass.
- Default factories are exercised without a server through error paths (bad dsn, unreachable address, empty migration directory), so coverage does not depend on `MIGRATOR_TEST_DSN`. A fake `Engine` drives `Run`, `Up` (ahead, dirty, nil version, no change), `Down`, `Status`, `Force` and `Close` with both errors.
- Smoke proof outside the PR: a scratch copy of one service run function started through the three seams with a fake broker, a sql mock, a fake engine and miniredis; the result is reported separately.
- No schema or SQL change; the impact items of the migration document are not applicable. `HELPER_TESTS=1 ALEMBIC_VENV=<venv> scripts/verify-service-migrations.sh bin-tag-manager` (Docker required) runs on both engines without SKIP lines, because `migrationhandler` changes; one service suffices since call signatures do not change.
- Repository workflow (`go mod tidy`, `go mod vendor`, `go generate`, `go test -race`, `golangci-lint`) in `bin-common-handler`; module coverage stays at or above 90% (91.8% before this change; the module figure, not per package).
- `go build ./...` and `go vet ./...` in every module whose go.mod has a `replace` line for `bin-common-handler` (`grep -l '=> ../bin-common-handler' */go.mod`).
