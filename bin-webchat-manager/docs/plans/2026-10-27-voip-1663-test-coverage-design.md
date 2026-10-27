# VOIP-1663 bin-webchat-manager test coverage: design

Scope: `_test.go` files only inside bin-webchat-manager. No product code change, no signature or visibility change, no go.mod or go.sum change. Analysis: `2026-10-27-voip-1663-test-coverage-analysis.md`.

## Test units (priority order)

1. `pkg/listenhandler/encode_failure_test.go`: for each of the 15 response-encoding points (widgets 6, sessions 5, messages 4, every handler that marshals a model or a list, including the session end and the widget direct hash regenerate handlers), the mocked handler returns a model whose creation time is in year 10000; the response is 500 with no body and the handler is called with the parsed arguments (strict mocks). A control case with a normal time returns 200 so that the 500 is attributed to the encoding. Table-driven, using the existing test request helpers.
2. `models/{message,session,widget}/webhook_test.go` additions: `CreateWebhookEvent` on a model with a year 10000 time returns an error and nil bytes.
3. `pkg/dbhandler/iteration_failure_test.go` (reuses `testEntities`, `newErrTestHandler`)
   - A small in-test `database/sql/driver` whose rows return a fixed error from `Next`, used through `sql.OpenDB`. For widget, session and message: get returns an error containing "row iteration error" and is not `ErrNotFound`; list returns an error containing "rows iteration error"; no cache write happens (strict cache mock; the get cases first set the cache miss expectation through `expectGetMiss`).
   - An unsupported filter value (a slice) passed to each list function returns an error containing "could not apply filters"; a closed database proves that no statement ran.
   - Widget and session update with an unencodable field value (a slice holding one channel element; an empty slice does not fail) return an error containing "prepare fields failed" and write nothing.
4. `cmd/webchat-control/main_test.go`: `main` with `os.Args` set to `--help` returns, with stdout captured and containing the command name and a config flag; `migrate status` through `initCommand` with `DATABASE_DSN` in the environment set to a nonexistent unix socket path (a missing file under `t.TempDir()`) returns an error that mentions the unix socket, which also proves the environment binding was used and not the default address; no network.
5. `cmd/webchat-manager/main_test.go`: `main` with `--help`, stdout captured through a pipe read in a goroutine, output contains the command name and flag names only (not the default values). Globals (`viper`, the config value, logrus, `os.Args`, `os.Stdout`) are saved and restored with `t.Cleanup`.

`runDaemon` and `initProm` are not tested (see the analysis). New tests do not use `t.Parallel`, follow the repository convention gate (`Test_` names, gomock controller `mc`, no testify), and keep expected values as literals.

## Verify

- Hand-written total >= 90.5% by the method of the stated baseline (about 92% expected); per package before and after table in the PR, measured with the same flags at the final HEAD.
- `go test -race -shuffle=on -count=1 ./...` several times, `golangci-lint run -v --timeout 5m`, `bash scripts/check-test-conventions.sh` (after the commit, with origin/main fetched).
- Mutation sampling in a scratch copy, one change at a time, each must fail a new test: encoding failure answered with 400 instead of 500 (once per file of the three handler files, and once for all fifteen points), the migrate closure given an empty connection string, row iteration error swallowed, filter error ignored, update error wrapped away, `CreateWebhookEvent` error dropped.
- No tautologies: expected values are literals. Synthetic data only, no network, no cost.
- `git diff --stat origin/main...HEAD` lists only bin-webchat-manager `_test.go` and docs files.
