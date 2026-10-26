# VOIP-1657 bin-campaign-manager test coverage design

## Goal and scope

Raise the hand-written statement coverage (excluding `mock_*.go`, `gens/` and files with a `Code generated ... DO NOT EDIT` header) from 79.67% (2791 of 3503) to at least 80.5%, which needs 2820 covered statements (3503 x 0.805), at least +29 and planned with margin. Only unit tests are added. No production code, signature, exposure, `go.mod` or `go.sum` change.

Measurement: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in `bin-campaign-manager`, then a hand-written filter script (duplicate blocks keep the maximum count) that prints the per-package figures. Before and after are measured in the same worktree path because the filter reads the generated-file headers from it. The raw total (68.61% before) rises as well but is not the figure claimed.

## Test targets (new `_test.go` files, existing patterns reused)

Planned statements are the reachable blocks listed in the baseline profile; the requirement is the +29 above, so the plan is a margin over the reachable blocks and not a promise for every package.

| Package | Target | Technique | Plan (stmt) |
|---|---|---|---|
| `pkg/listenhandler` | handler error return, malformed URI (`url.Parse` error, returned as an error), `len(uriItems) < 4` branches (400 response), asserted separately, flows reconcile invalid body | existing gomock handlers, direct handler calls, exact status and error assertions | ~20 |
| `pkg/campaigncallhandler` | `Create` DB failures, `Delete` failures and not-done guard, `EventHandle*` error returns, constructor | gomock `DBHandler`/`RequestHandler`, assert returned error and that follow-up calls are not made | ~23 |
| `pkg/campaignhandler` | error returns of create/list/stop paths, `execute` update-failure return path, `eventhandle` not-stoppable and stop-failure branches, reconcile list failure, constructor | gomock | ~30 |
| `pkg/subscribehandler` | `processEventFMActiveflowDeleted` unmarshal error, lookup failure, done, call-reference and handler-error branches, `Run` queue and bind error returns (error paths and the synchronous `processEvent` branches only; the goroutine started by `processEventRun` and `ConsumeMessage` are excluded, with no wait) | gomock `SockHandler` and handlers | ~12 |
| `pkg/outplanhandler` | constructor | direct call, assert non-nil and injected dependencies | 4 |
| `pkg/dbhandler` | `GetCurTimeAdd` (format, and the parsed value lies within a before/after `time.Now()` window with a stated tolerance; no exact-equality comparison); exec, query and scan failures and cache failures | independent `go-sqlmock` `*sql.DB` per test (closed per test, `ExpectationsWereMet` asserted) (already in `go.mod`, used by `ctx_rows_test.go`), mock `CacheHandler`; the shared sqlite test DB is neither touched nor closed | ~30 |
| `cmd/campaign-control` | `resolveUUID`, `printJSON` (including the unmarshalable-value error), `initCommand` structure, `cmd*` flag definitions, `runCreate` input validation (missing or malformed customer, name, outplan and outdial ids only, so the connection step is never reached) | pattern of `bin-tag-manager/cmd/tag-control/main_test.go` | ~40 |
| `cmd/campaign-manager` | `initLog` | restore the logrus formatter and level afterwards | 2 |

Plan total is about 160 statements against the +29 requirement. Branches that cannot be reached without a signature change or an external connection are not targeted.

## Quality and isolation rules

- Error assertions use `errors.Is` or gomock argument matchers, not `==` (CI runs errorlint on added lines, tests included). Where a test pins the current behavior of a branch, it asserts the observable result (return values, calls made and not made) and carries a neutral comment only.
- Every new test asserts returned values, errors, call arguments or state. Constructor tests assert a non-nil result and the injected dependencies. No bare calls and no self-comparison.
- Tests that touch global state (viper including flag bindings, `config.Get()`, host environment keys, `os.Stdout`, logrus) save and restore it in `t.Cleanup` (`viper.Reset`, restore the config copy, unset and restore the environment keys the tests could read). No `t.Parallel()` in those tests.
- `cmd/campaign-manager` has an `init()` that registers flags, sets logging and starts the signal goroutine at test binary load. That is existing behavior and is not changed; the new tests do not send signals or start listeners, and `-race` is part of verification.
- `os.Stdout` capture reads the pipe in a goroutine, closes the write end, waits for the reader and restores `os.Stdout` before the test returns.
- Every test that calls `initCommand` or reads viper keys starts from `viper.Reset` and unsets the host environment keys it reads (viper reads the host environment), so empty-value assertions hold in any environment, and restores everything in `t.Cleanup`.
- Host environment values used by the cmd tests are fixed dummy strings without credentials and without DSN form; the `runCreate` validation tests assert the error is returned before any connection step.
- Added lines also pass `nilerr` and `errorlint`, which apply to test files.
- No network, timers or sleeps. No shared mutable fixtures between tests. Synthetic ids and values only.
- Test names start with `Test_`; gomock controllers are named `mc` (CI convention gate).

## Verify items

1. The hand-written total is at least 80.5%, measured with the same options as the baseline (`-count=1`, default cover mode, no `-race`); the `-race` run is a separate command.
2. `go test -race -count=3 ./...` passes for every package except `pkg/dbhandler`, whose existing tests insert fixed ids into the shared sqlite database and are run with `-count=1` (the new dbhandler tests also pass with `-count=5`), also with the cmd tests run under hostile host environment values (`DATABASE_DSN`, `REDIS_ADDRESS`, `REDIS_PASSWORD`, `REDIS_DATABASE`, `RABBITMQ_ADDRESS`, `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `ID`, `NAME`, `DETAIL`, `TYPE`, `CUSTOMER_ID` set to fixed dummy non-connectable strings).
3. After `git fetch origin main`: `bash scripts/check-test-conventions.sh`, `golangci-lint run -v --timeout 5m` and `golangci-lint run -c <repo-root>/.golangci-errorlint.yml --new-from-merge-base=origin/main` pass; the root verification workflow (`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, lint) leaves `go.mod` and `go.sum` unchanged.
4. `git diff --stat origin/main...HEAD` lists only files under `bin-campaign-manager/`, `git diff --name-only` shows no non-`_test.go` Go file changed, and `git status --porcelain` shows no stray untracked file; the `go.mod`/`go.sum` check runs after the last measurement run.
5. Mutation sample: for at least eight new tests across the packages (at least one each in cmd and dbhandler), invert or remove the guarded branch in a scratch copy and confirm the test fails.
