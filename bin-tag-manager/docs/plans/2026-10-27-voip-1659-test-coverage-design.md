# VOIP-1659 bin-tag-manager test coverage

## 1. Ticket review

Baseline on origin/main faf84788b (VOIP-1647 and VOIP-1657 already merged): hand-written coverage 84.07% (908 of 1080 statements). The floor is 85.0% (918) and the target of this work is 85.5% or more (924).

## 2. Uncovered statements (172)

| Group | Statements | Plan |
|---|---|---|
| Outside `cmd/`: marshal, squirrel, PrepareFields/ApplyFields, flag/env bind, `ToResponse` failures | 25 | Not targeted: the failure cannot be produced without changing product code |
| tag-control `initTagHandler` | 24 | Planned. `rabbitmqhandler` `connect` only needs a successful AMQP dial, so a small in-process server that speaks the AMQP 0-9-1 handshake and answers channel open and exchange declare is enough. `initTagHandler(nil, nil)` works because the constructors only store the database and cache |
| `Execute` line of both `main` functions | 3 | Planned. `os.Args` is set to `--help` |
| tag-manager `initProm` listener-failure branch | 3 | Planned. Invalid listen address |
| tag-control `runCreate/Get/List/Update/Delete` after `initHandler` | 48 | Reachable only with a fake MySQL, Redis and broker together; skipped, cost above benefit |
| tag-manager `runService` | 44 | Needs no database (nil is passed), only a fuller fake broker (queue and topic declare, bind, consume); skipped, cost above benefit |
| tag-manager `run` after the database ping | 15 | Needs a fake MySQL; skipped, cost above benefit |
| Error branches of both `main` (`Fatalf`, `os.Exit`), `<-chDone`, `initProm` `break`, remaining small blocks | rest | Not reachable or not worth a test |

Planned total: 908 + 30 = 938 of 1080 (86.85%), measured on a scratch copy of the plan. Without the `initTagHandler` test the other additions reach only 914 (84.6%), so the fake broker is required. No product code change is needed.

## 3. Plan (test files only)

1. Fake broker (tag-control test file): a loopback TCP listener that speaks the AMQP 0-9-1 subset the client needs (protocol header, connection start, tune with heartbeat 0, open, channel open, exchange declare, close methods) and records the declared exchange name and kind. The address is `amqp://127.0.0.1:<port>` with no credentials. It binds to port 0, is started once per process and never closed: the client keeps `reconnector` and `healthChecker` goroutines for the life of the process (the health check interval is 30 seconds), so closing the listener would only produce retry noise.
2. tag-control `Test_initTagHandler`: the call (and the broker address in `config.Get()`, restored in `t.Cleanup`) is wrapped in a package-level `sync.Once` (the request handler registers prometheus collectors without a guard, so a second real call in one process panics) so `-count`, `-shuffle` and `-run` all work, and each test run asserts on the cached result: non-nil handler, no error, the declared exchange has the global topic name and the topic kind. The call runs in a goroutine that does not touch `t`; it returns its result (or a recovered panic) through a channel and the test waits with a deadline. `logrus` `ExitFunc` is replaced for the test (restored in `t.Cleanup`) so that a `Fatalf` in the notify handler becomes a recorded test failure instead of ending the test binary.
3. Both packages: `Test_main_help` with `os.Args` set to `--help` (restored in `t.Cleanup`) and stdout captured; the assertion is that the output contains `Usage:` and the command name (cobra reads `os.Stdout` at call time; `main` gives no access to the command, so the swap is needed).
4. tag-manager `Test_initProm_listenFails`: a listen address that always fails to parse, a new endpoint per run (counter approach of `run_test.go`, because `initProm` registers on `http.DefaultServeMux`), a logrus hook that is removed afterwards (hooks are saved and restored with `ReplaceHooks`) and sends to a buffered channel without blocking, and a bounded wait of at least 5 seconds for two error logs, which proves the retry. The retry goroutine lives until the process ends.
5. Test code follows the existing helpers and the secret hygiene rule (no credential literal in any address).

## 4. Verify

- Hand-written total at least 85.5%, measured with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` and the exclusion of mock, gens and DO NOT EDIT files.
- Existing tests and lint pass; `go test -race -shuffle=on -count=2 ./cmd/...`; each new test also passes alone with `-run`.
- Fatal path: a scratch run where the fake broker answers the exchange declare with an error must end as a test failure, not as a dead test binary.
- Mutation sample in a scratch copy, each must fail a new test: drop `WithGlobalTopicPublish` (declared exchange name changes), change the exchange kind, remove `sockHandler.Connect()` (a recovered panic in the goroutine becomes a test failure), remove the `Execute` call in `main`, remove the `continue` in `initProm`.
- `git diff --name-only origin/main...HEAD` lists only this document and `bin-tag-manager/**/*_test.go`; no `go.mod`, `go.sum` or vendor change.
