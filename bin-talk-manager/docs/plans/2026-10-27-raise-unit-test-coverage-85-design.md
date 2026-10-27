# bin-talk-manager unit test coverage 85% (VOIP-1659)

## 1. Ticket review

Result: valid. Baseline on origin/main a47f5a2a1 (`GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`, hand-written code only, excluding mock_*.go and `DO NOT EDIT` files): 81.03% (1828 of 2256 statements). 85.0% needs 90 more statements, 85.5% needs 101 more.

Uncovered statements: 428. Of these, 250 are in `cmd/talk-control` (the `run*` functions after `initHandlers()`), 137 in `cmd/talk-manager` (`main`, `runDaemon`, which need real connections) and 41 elsewhere (marshal, query-builder and config-binding error branches that cannot be reached).

Only `cmd/talk-control` is reachable. `initHandlers` connects to the database, RabbitMQ and Redis directly, so no `run*` function can get past it in a test.

## 2. Design

One non-test change, the only non-test change: in `cmd/talk-control/main.go`, `func initHandlers() (...)` becomes `var initHandlers = func() (...)` with the same body. Behavior is unchanged and the call sites are unchanged.

Tests (all in `cmd/talk-control/main_test.go` or new `_test.go` files in the same package):
- A helper replaces `initHandlers` with one that returns gomock chat, message, participant and reaction handlers, and restores it with `t.Cleanup`.
- Per `run*` function: success path (the exact arguments passed to the mock handler and the printed JSON), handler error path, and the flag validation errors that come after `initHandlers` (id, customer, chat, owner and emoji flags, invalid UUID, missing required value).
- Assertions use `gomock.Eq` values (no `gomock.Any()`), including pointer arguments (flag omitted gives nil, flag given as empty string gives a pointer to an empty string), fixed arguments (empty participant and media lists), the `size` default of 100 and an explicit `--size`, `token` forwarding, and the fixed output of `participant remove`. The environment cleanup list in `execute` is extended with the remaining flag names.
- Isolation: no `t.Parallel()` (the hook, `os.Stdout` and viper are global; this is what keeps `-race` safe), the viper state is reset per test, nothing touches the network, the clock or the filesystem, and the mock values are fixed synthetic UUIDs.

Expected result: about 66 of the 250 statements stay uncovered (`main`, the body of `initHandlers`, two `initCommand` error branches), so about 184 become reachable and the hand-written total is about 89%. The ceiling is therefore above 85% and the card is not blocked again.

## 3. Verify

- `go test -count=1 -race ./cmd/talk-control/` passes, and the hand-written total is at least 85.5% with the measurement above.
- Mutation samples in a scratch copy of `main.go`; the new tests must fail each time.
- Existing tests pass and `golangci-lint run` is clean. `git diff --stat origin/main...HEAD` lists only `bin-talk-manager/` files.
