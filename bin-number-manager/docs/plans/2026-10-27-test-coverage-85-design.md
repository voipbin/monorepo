# bin-number-manager unit test coverage to 85% (VOIP-1659)

## 1. Analysis

Baseline at origin/main a47f5a2a1, measured with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`. Hand-written coverage excludes `gens/`, `mock_*.go` and `Code generated ... DO NOT EDIT` files; duplicate blocks are merged by max count. Hand-written total is 82.75% (2471 of 2986 statements), raw total 71.6%. 85.0% needs 2539 covered (+68), 85.5% needs 2554 (+83).

Uncovered hand-written statements (515) by file:

| file | uncovered | reachable without a signature change |
|---|---|---|
| cmd/number-control/main.go | 319 of 319 | six cmd* constructors (117 statements), initCommand (about 33 of 36 when a subcommand is executed so that the pre-run hook runs; the flag-binding error handling and the migrator opener are not reached), resolveUUID and resolveString (11), printJSON (4 of 5), and the UUID or string validation failures of runCreate, runGet, runList, runUpdate and runDelete. runGetAvailable, the field check of runUpdate, the virtual branch of runCreate and the init* functions need a connection and stay out |
| cmd/number-manager/main.go | 106 of 108 | initSignal (5), runServiceListen (6), runServiceSubscribe (6) through a mocked sock handler |
| pkg/dbhandler, pkg/listenhandler, pkg/providerhandler, pkg/secretcrypto, others | 90 | not individually verified yet; only the branches that a mock error or real input reaches are planned (see section 2) |

The bulk of the gap is `cmd/number-control`, which has no test file. Earlier lanes treated `cmd/` as untestable; the reachable parts above need no database, Redis or RabbitMQ.

Reachable upper bound is well above 85.5% (constructors 117 plus initCommand 36 plus resolvers 11, printJSON 4, validation failures about 14 and cmd/number-manager 17 add up to about 199 against the 83 needed), so no block is needed.

## 2. Design

Test files only, no product code change, no new module.

cmd/number-control (`main_test.go`):
- `resolveUUID`, `resolveString` and the validation failures of `runCreate`, `runGet`, `runList`, `runUpdate`, `runDelete` are called directly, without cobra: `viper.Reset()` then `viper.Set(...)`, with `t.Cleanup(viper.Reset)`. They fail before `initHandler`, so no connection is made. Assertions check the error text for the missing and the malformed case separately.
- `printJSON` captures `os.Stdout` through `os.Pipe` (restored in cleanup, write end closed before reading, no `t.Parallel()`), unmarshals the output and compares the structure against a literal, and an unmarshalable value (a channel) covers the marshal error.
- `cmd*` constructors: a literal expectation table of flag name, type and default per command (`Lookup(name)`), plus required subcommand names (`number` with its six children, `migrate`). No value is derived from the code under test.
- `initCommand`: one or two tests run `number get` through `Execute` with `SetArgs`, and `SetOut` and `SetErr` set to a discarding writer, no `--id`, and `t.Setenv("ID", "")` because `AutomaticEnv` maps the `id` key to the environment; the expected result is an error that contains `Number ID is required` (substring, the error is wrapped), which proves the pre-run hook and flag binding work together. `config.LoadGlobalConfig` is a `sync.Once` that is not reset by `viper.Reset()`, so no test depends on the loaded config values. logrus level and formatter changed by `initLog` are restored in cleanup. The `migrate` subcommands are never executed (opening the migrator can touch a database).

cmd/number-manager:
- `initSignal`: `chDone` is drained first, a value is pushed with `chSigs <- syscall.SIGTERM` (no real kill), `chDone` is read with `select` and a timeout and must be `true`; `signal.Stop(chSigs)` in cleanup.
- `runServiceListen` and `runServiceSubscribe` with `sockhandler.MockSockHandler`. `Run` starts its consumer goroutine after returning, so the success cases make `ConsumeRPC` and `ConsumeMessage` close a channel in `DoAndReturn` and wait on it with a timeout before the test ends (pattern of `pkg/listenhandler/v1_error_paths_test.go`); `AnyTimes()` is not used. The subscribe success case expects the exact `QueueBind` count with a gomock `Times(2)` (the number of topic patterns); queue and exchange are taken from `commonoutline`; the pattern strings stay with the existing golden test of the subscribe handler. `QueueBind` always has an explicit `Times`. Error cases: `runServiceListen` with a failing `QueueCreate`; `runServiceSubscribe` with a failing `QueueCreate`, `TopicCreateWithKind` and `QueueBind` (no `ConsumeMessage` expectation); each checks the returned error. The `DoAndReturn` functions use the exact typed callback signature of the consume methods, not a variadic one.

Other packages (not needed in the end: the cmd tests alone reached 89.48%, so this item was not carried out), a safety margin only if the cmd tests leave less than 1 point over 85.5%. Planned branches: error returns of `providerhandler` admin functions after a gomock db error (`admin.go` list, row lookup, ensureRow and candidate lookup), the `NewListenHandler` constructor, and the subscribe handler log path after a handler error. Each is read in the code first; a branch that no mock error or input reaches is dropped.

Isolation: no network, no sleeps, no real clock, no real signals; global state (viper, logrus, stdout, signal registration) restored in cleanup; test values are synthetic.

## 3. Verify

- Hand-written total at or above 85.5%, computed by excluding `gens/`, `mock_*` and `Code generated ... DO NOT EDIT` files with duplicate blocks merged by max count (the same rule as the baseline); the PR body lists the command and flags.
- The root verification workflow passes (`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run`). Existing and new tests pass with `-race -count=3`, and with `-shuffle=on` for the whole service; `go vet` and `golangci-lint` are clean; `scripts/check-test-conventions.sh` passes after `git fetch origin main`.
- Mutation spot check, each must fail a named new test (the empty and the malformed UUID cases are separate tests): invert the empty check in `resolveUUID`, drop the malformed-UUID check in `resolveUUID`, and remove the error return of the `Run` result in `runServiceListen`.
- The PR file list stays inside `bin-number-manager/`.
