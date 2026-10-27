# bin-talk-manager unit test coverage 90% (VOIP-1663)

## 1. Ticket review

Result: valid. Baseline on origin/main 0bfbe2a1d (`GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`, hand-written code only, excluding mock_*.go, gens/ and `DO NOT EDIT` files): 89.23% (2013 of 2256 statements, 243 uncovered). 90.0% needs 18 more statements, 90.5% needs 29 more.

Most uncovered statements need a live database, RabbitMQ or Redis. Some entry points and error branches are reachable offline, which this plan covers. About 51 statements are reachable, so about 91.5% is the ceiling and the target is not blocked.

## 2. Design

Test files only, no non-test change, no new module.

1. `cmd/talk-manager` entry point. Run `main()` with `os.Args` set to the command name only, the environment `DATABASE_DSN=bad` (the driver rejects it while parsing, before any dial) and a listen address without a port, and the logger exit function replaced by a panic that the test recovers. Checks: exit code 1, a fatal log about the database connection, and no "RabbitMQ initialized" or "Redis initialized" log. A second test, with its own metrics path and listen string so that no log of the first test can satisfy it, calls the metrics listener start with an address without a port and waits on the log hook (bounded) for the start log and for the error log containing "missing port in address".
2. `cmd/talk-control` entry point. Run `main()` with `--help` and a replaced `os.Stdout` (read after the writer is closed). Checks: it returns without exiting, and the output contains the command name and the sub command names (chat, message, participant, reaction, migrate). Reset viper first and reuse the package `TestMain` (`DATABASE_DSN=bad`).
3. `pkg/dbhandler`. The list, update and `MessageCreate` checks use a handler with a nil database handle (any query execution would panic and so prove a missing guard) and the real time helper of the existing tests; the reaction checks use the scripted driver of the existing tests, because they read the stored metadata first:
   - list functions of chat, message and participant: the filter value is `commondb.NotEq{Value: x}` where `x` implements `driver.Valuer` with `Value() (driver.Value, error)` returning an error. Expect that error, no query.
   - update functions of chat and message: an update value of string kind whose `ToSql` returns an error. Expect that error, no query.
   - `MessageCreate`: a reaction `TMCreate` in year 10000 makes the field conversion fail; expect an error containing "year outside of range". The add and remove reaction functions: the stored metadata JSON returned by the scripted driver holds a reaction time with the offset `+24:00`, which decodes but cannot be encoded; expect an error containing "timezone hour outside of range" and no update statement. The error texts are asserted as substrings, so a Go upgrade does not break them.
4. `pkg/reactionhandler` and `models/chat`, `models/participant`, `models/message`: a time in year 10000 (time mock for the reaction add, field value for the models) returns the marshal error of the reaction add and of the webhook events of the three models, asserted by its text.
- Isolation: no `t.Parallel()`. The `main()` tests also stop the signal notification and drain the signal channels in `t.Cleanup`. The default HTTP mux (a fresh mux per test), the logger exit function, hooks, `os.Args`, `os.Stdout` and viper are replaced per test and restored in `t.Cleanup`, so the new tests are safe with `-count=N`. The existing database tests insert fixed ids into a shared in-memory database, so the whole `pkg/dbhandler` package is not repeatable with `-count>1` on main either; that is not changed here. The service config is loaded once per process, so every `main()` test uses the same environment values and none depends on an earlier load. `main()` may run more than once under `-count=N`; each run leaves a signal goroutine that ends on the same channels the existing signal test drains, so they do not change its result. Nothing is dialed, no fixed port, no clock dependence beyond the bounded log wait. Test values are synthetic.

Expected result: about +51 statements, about 91.5% hand-written (target 90.5%).

## 3. Verify

- Measurement: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` from the service directory, then drop `mock_*.go`, `gens/` and files with a `DO NOT EDIT` header, merging equal blocks by their highest count. The hand-written total must be at least 90.5%, and the uncovered list before and after shows every targeted branch turned covered.
- Protection runs: `go test -count=1 -race -shuffle=on ./...` passes, and the new tests of `cmd/*`, `pkg/dbhandler`, `pkg/reactionhandler` and `models/*` pass alone with `-run <new test names> -count=20 -race -shuffle=on`. `go mod tidy` leaves `go.mod` and `go.sum` unchanged.
- Mutation samples in a scratch copy, each must fail a test: replace the fatal log by an error log (run with a short `-timeout`, because the daemon then goes on to connect), remove the database connection error branch, swallow the listener error, remove the help output, drop each query build error return (list, update), drop each marshal error return (message create, reaction add and remove, reaction handler, three models).
- Existing tests pass and `golangci-lint run` is clean. `git diff --stat origin/main...HEAD` lists only `bin-talk-manager/` files.
