# bin-ai-manager: raise hand-written unit test coverage to 90%

## 1. Ticket review

- Baseline re-measured on origin/main 0bfbe2a1d (same command as the ticket): hand-written 87.80% (13966 of 15907 statements), zero failing tests. The number matches the ticket. No newer commit touched the service since the ticket was written.
- Needed: 351 more covered statements for 90.0%, 430 for 90.5%. The target is 90.5% or more (430 statements) to leave margin for CI rounding.
- Measurement: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in `bin-ai-manager/`. Hand-written excludes `gens/`, `mock_*.go` and files with a `Code generated ... DO NOT EDIT` header; duplicate blocks keep the maximum count.
- Uncovered statements by package: cmd/ai-control 546, dbhandler 354, aicallhandler 260, cmd/ai-manager 246, listenhandler 186, engine_dialogflow_handler 71, builderhandler 43, geminiaudithandler 35, mcpoauthhandler 32, builderhandler/eval 25, geminiproposalhandler 20, internal/config 19, teamhandler 18, cachehandler 18, others below 10 each.

## 2. Reachable ceiling

Statements that need a live database, cache or broker connection, an external LLM or cloud client, or a server loop are outside the target, as are always-nil error branches and `json.Marshal` failures. They make up about half of the uncovered statements (the whole of `cmd/ai-manager`, the bodies of the `cmd/ai-control` run functions that follow the connection opening, the external client packages and a few queries the sqlite harness cannot run, and error branches that follow query building or field preparation, about 80 statements in dbhandler). The rest, about 900 statements, is reachable by unit tests without a signature change, which gives a conservative ceiling of about 93% (13966 + 900 of 15907) against the 90.5% target, so the work proceeds without escalation.

## 3. Constraints

- Test files only, inside `bin-ai-manager/`. No product code or signature change. `go.mod` and `go.sum` are not changed unless a test needs a module that is already used by another service (miniredis is already in `go.mod`).
- Tests do not depend on wall clock, ordering, shared state or the network, and none uses `t.Parallel()`. Goroutine results are awaited on channels with a timeout, never a sleep. Secret-like fixtures are plainly synthetic (`test-key-0001`), never a real token format, and are assigned in a separate statement.
- dbhandler: the package-wide shared handle (`dbTest`) is never closed. Failure paths use a handle that the test opens and closes itself (a private sqlite database per test). A closed handle fails only the first database call of a function, so later branches (row scan, re-read after write, cache refresh after write) use a new failing driver stub added in the new test file under a different driver name (the existing stub only fails row iteration, and a driver name can be registered once), a private database whose table or column is dropped, and cache mock errors. The private sqlite database uses `:memory:` with `SetMaxOpenConns(1)` or a unique name per test, never the shared-cache name of the package handle. Closed-handle errors are not a sentinel, so assertions are "an error is returned and nothing was written or cached", plus `errors.Is` where a sentinel exists, never driver message text. The schema of the private database comes from a helper in the new test file that loads the same `scripts/database_scripts_test` files as `TestMain`. New tests use no package-level mutable state (the one-time registration of the stateless stub driver is the only exception) (sentinel variables, shared database names). Existing `resetTestDB` isolation is kept.
- `cmd/ai-control`: `Execute` is never called, and the `run*` functions are out of scope (each opens the database and cache before anything else). Reached are `initCommand` (the `PersistentPreRunE` hook is not invoked, because `config.LoadGlobalConfig` is a `sync.Once` over process-wide state), each `cmd*` constructor, and `resolveUUID`. The logrus level and formatter that `config.Bootstrap` changes are saved first and restored in `t.Cleanup`, and `viper.Reset()` runs in `t.Cleanup` (the global viper instance cannot be saved and restored); environment goes through `t.Setenv`, and tests that assert defaults blank the variables that `AutomaticEnv` could read from the developer shell; every test builds its own command objects.
- Allowed assertions for commands are contracts that the `run*` functions actually read: the subcommand hierarchy, the flag keys with their defaults and types, and the values `viper` returns after `ParseFlags` and `BindPFlags` for the keys `run*` reads. A test that only repeats a flag name string passed to the constructor is not written.
- Conventions (`scripts/check-test-conventions.sh`): `Test_` prefix, gomock controller named `mc`.
- Overlap with the separate general lift of packages under 80% is noted in the PR body only. The PR body names functions and mutation kinds, never file and line positions or lists of unreachable branches.

## 4. Design

Groups are worked in the table order D1, A1, L1, S1, C1 (D1, A1 and L1 carry most of the gain; C1 is last because it touches process-wide state) and the work stops once the measured total reaches 90.5% with margin (90.6% or more; the target is 90.5%, and the extra 0.1 point absorbs rounding differences between the local and the CI measurement) and all tests are green; C1 is skipped if the stop rule is met earlier. If the total is still under 90.5% after all groups, the work stops and the remaining uncovered inventory goes to the requester for a decision instead of lowering the target.

| Group | Scope | Approach | Expected gain |
|---|---|---|---|
| D1 | dbhandler: create, get, list, update, delete of ai, aicall, message, summary, team, mcpserver, mcpoauthstate, aiprompthistory, aipromptproposal, aiaudit, participant | existing sqlite harness for not-found and filter paths; per-test private closed handle, failing-rows stub, dropped table or column, and cache mock errors for the failure paths described in section 3 | ~200 |
| A1 | aicallhandler: `listen*`, `tool*`, `start`, `send`, `service`, `mcp_tool`, `chat`, `event` uncovered branches | gomock; invalid arguments, dependency errors, rollback paths | ~150 |
| L1 | listenhandler: per-resource handlers (ais, aicalls, aiaudits, aipromptproposals, mcpservers, services, summaries, teams, messages, participants, builder) | gomock; table tests for malformed JSON bodies, bad ids, handler errors and success status and body | ~130 |
| S1 | small packages: geminiaudithandler (prompt and response parsing), geminiproposalhandler, mcpoauthhandler, internal/config, teamhandler, cachehandler, actioncatalog, builderhandler and eval | direct calls with synthetic values, miniredis for the cache, gomock for dependencies. The dialogflow package is excluded (its remaining statements call the external client) | ~100 |
| C1 | cmd/ai-control: `initCommand`, `cmd*` constructors, `resolveUUID` | command tree and flag-contract assertions as in section 3 | ~130 |

### Test rules

- Every test asserts an observable result (returned value, error identity, stored row, written key, RPC status and body, exact arguments of a mock call). No bare call tests and no test that builds a value and compares it to itself.
- Priority: error paths, boundary values, validation and permission branches, then the success path.
- No `t.Skip`. Synthetic values only (fixed uuids, `example.com` hosts, no `user:pass@host` strings).
- New tests always go into new files whose names end in `_cov90_test.go` (the suffix does not exist in the tree, so no existing file is touched or shadowed). Existing tests are not modified.

### Verify items

1. After each group the measured total and per-package numbers are recomputed with the baseline command (`GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`); the PR table lists them before and after, with the number of skipped tests added (zero).
2. Repository verification workflow (`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run -v --timeout 5m`) with `go.mod` and `go.sum` unchanged (the baseline command uses `GOFLAGS=-mod=mod`, so the diff is checked after it). The PR body states the hand-written calculation: files under `gens/`, `mock_*.go` and files with a `Code generated ... DO NOT EDIT` header are dropped, and duplicate blocks keep the maximum count.
3. Flake check on the new tests: `go test -race -count=5 -shuffle=on` on the touched packages (cgo is available on the build host; if it were not, the run is repeated without `-race` and the card says so) with the shuffle seed recorded; the new test functions also run alone through `-run '^(A|B)$'` with their names listed, and the full package runs in shuffled order with the existing tests, to show no dependency on state left by other tests; one run with the network namespace unshared and only the loopback interface brought up (`unshare -rn sh -c 'ip link set lo up && go test ...'`) when the host allows it, to show no outside network is needed (loopback servers and miniredis still work). Where the host forbids user namespaces, the new test files are instead checked by grep for dial, listen on non-loopback addresses and external hostnames.
4. Mutation sample on a scratch copy: at least eight small product-code mutations, at least one per executed group and eight or more in total (inverted condition, dropped error return in a dbhandler function, wrong status, swapped argument, off-by-one, a changed flag default or command hierarchy in `cmd/ai-control`); each must fail at least one new test of that group. The PR body table lists function name, mutation kind and result only.
5. Scope check by script: `git diff --name-only origin/main...HEAD` has nothing outside `bin-ai-manager/`, every changed `.go` file ends in `_test.go`, existing test files are unchanged, and the count of test functions in the base commit has not dropped. `scripts/check-test-conventions.sh` passes. The diff is grepped for email addresses, IP addresses, tokens and `user:pass@` patterns.
6. Defects noticed while writing tests are not fixed; tests that pin such behavior carry a neutral comment and the finding goes into the card only.

### Scope actually executed

D1, L1, S1 (the gemini handlers, teamhandler, cachehandler and actioncatalog) and C1 were executed; A1 and the remaining S1 packages were not started, as the stop rule allows. The measured hand-written total is recorded in the PR body. `go.mod` and `go.sum` are unchanged.
