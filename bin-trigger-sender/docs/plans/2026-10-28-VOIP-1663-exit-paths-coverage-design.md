# bin-trigger-sender coverage to 90% (VOIP-1663)

## 1. Ticket review

Partially valid. Baseline is 59 of 66 statements (89.39%), a single package. The 7 uncovered statements (3 + 2 + 2) are the three `os.Exit(1)` blocks in `main`; nothing else is uncovered. 90% needs 60 statements.

A scratch measurement (Go 1.25) shows that the coverage of a child process started from the test binary itself (`os.Args[0] -test.run=...`) is merged into the parent `-coverprofile`; the three scenarios below reach 66 of 66. No product code change is needed.

## 2. Scope

Test-only: a new `exit_test.go` in `cmd/bin-trigger-sender`. No product, signature or exported-surface change.

## 3. Tests

Names follow `check-test-conventions.sh`: `Test_main_exit_helper` (gated child) and `Test_main_exit_codes` (parent), plain `t.Errorf` checks, no testify, no `t.Parallel()`.

The helper reads a scenario variable, skips when it is unset (no recursion, `-shuffle` safe), and calls `main` through the existing `runMain`. The parent starts `os.Args[0] -test.run=^Test_main_exit_helper$` with `exec.CommandContext` (30 s timeout, `WaitDelay` set), builds the child environment explicitly from `os.Environ()` (inheriting `GOCOVERDIR`, which the merge needs) plus the scenario variable, passes no coverage flags, and captures stderr. A timeout kill gives a signal exit, which the status check rejects.

Every scenario uses `-rabbit_addr amqp://%zz` (fails in URI parsing, no network) unless that flag is the one omitted, so a removed exit falls through to `run` and prints `Failed:` instead of silently exiting 1 anyway. Assertions: exit status exactly 1 and stderr containing the message below; for the two early-exit scenarios stderr must also not contain `Failed:`.

| scenario | other arguments | stderr must contain |
|---|---|---|
| missing required flag (three cases: rabbit_addr, queue, uri each omitted) | the remaining two valid | `required: -rabbit_addr, -queue, -uri` |
| invalid `-data` | all required flags, data `{` | `invalid JSON in -data flag` |
| run failure | all required flags, no data | `Failed:` and `dial RabbitMQ` |

Without the exit in the run-failure block, the child would end `main` with status 0, which the status check catches.

Statements: 66 of 66 (100%). No broker and no network are used.

## 4. Verify items

1. `GOFLAGS=-mod=mod go test -count=1 -race -coverprofile` passes; total at least 90.5%, and the CI coverage log shows the merged value.
2. `-count=20 -race -shuffle=on` passes.
3. Mutation sample on a scratch copy: `os.Exit(1)` removed from each of the three blocks, exit code changed to 0 and to 2, required-flag condition `||` changed to `&&`, `-data` condition `&&` changed to `||`, each error message changed; each must fail a test.
4. Standard five steps and `scripts/check-test-conventions.sh`; diff only under `bin-trigger-sender/`.
5. No real host, credential or customer data.
