# VOIP-1659: bin-contact-manager unit test coverage to 85%

Baseline (origin/main 320acbfc2): hand-written 3198/3831 = 83.48%. Floor 85.0% (3257), recommended 85.5% (3276); the completion target is 85.5% or more, aiming for about 3300. 489 of the 633 uncovered statements are in the three `cmd/` main packages, which have no test file. Tests only, inside bin-contact-manager, no product code, signature or go.mod change.

## Plan

| Group | Scope (reachable without a database, Redis, RabbitMQ or an open port) |
|---|---|
| A. contact-control | `initCommand` tree (use strings, subcommands, `migrate`), the six `cmd*` constructors (flag names, defaults, required flags), `resolveUUID`, `printJSON`, `runCreate` argument validation, the wrapped `initHandler` failure of every `run*`, `migrate status` failing in DSN parsing |
| B. case-control, contact-manager | case-control `initCommand` and `PersistentPreRunE`; contact-manager `init()` state (flags and defaults), the one-shot signal path, `runListen` and `runSubscribe` with the generated sockhandler mock |
| C. pkg | `addresshandler` delegation and constructor, `casehandler` error propagation of the db calls |

Not tested (needs a live connection or a port): `main`, `run`, `runService`, `initProm`, `initCache`, `initContactHandler`, everything after a successful `initHandler`, and the config bind error branches.

## Harness

- The only database value is an obviously invalid string that fails in `sql.Open` DSN parsing, so no connection is attempted (a well formed DSN would ping the network and is never used). `TestMain` of the two `-control` packages sets `DATABASE_DSN`, `REDIS_ADDRESS` and `RABBITMQ_ADDRESS` once, because `config.LoadGlobalConfig` runs once per process and `viper.Reset()` does not re-arm it; those three values come only from there, and flags are passed only for per-command options. Tests build a fresh command per case and drive it through `Execute()` with a non-nil `SetArgs` slice and silenced usage. `resolveUUID` is reached only through `Execute()` with explicit flags (no `viper.Set`, `viper.Reset` or `t.Parallel()`), the `case-control` root `PersistentPreRunE` (not runnable through `Execute()`) is called directly, and `printJSON` is called directly.
- `printJSON` output is read concurrently from a pipe that replaces `os.Stdout` and is restored in `t.Cleanup`. `init()` starts the only signal goroutine, so one test sends one value through `chSigs` and waits on `chDone` with a timeout, guarded by a process-wide `sync.Once` (reruns assert nothing there). Mock consumer goroutines report through a channel that the test waits on with a bounded `select`; `runListen` is asserted to return nil even when `QueueCreate` fails. `init()` defaults are asserted only for non-secret values (flag exists and type, `/metrics`, `:2112`, `1`), never the credential-shaped ones.

## Verify

- Assertion standard: wrapped error texts or `errors.Is`, flag names and defaults, command trees and return values; call counts alone, empty calls and self-comparison are not accepted.
- Hand-written total at least 85.5% with the standard filter (the plain CI total includes mocks and is informational); the new tests pass with `-race -shuffle=on -count=3` (stricter than CI); `bash scripts/check-test-conventions.sh` after committing and fetching `origin/main` (names `Test_...`, no testify, `reflect.DeepEqual` with `t.Errorf`, gomock controller `mc`), the verification workflow in the root CLAUDE.md and `golangci-lint` are clean; the diff is limited to bin-contact-manager test files and this document. Test values are synthetic.
