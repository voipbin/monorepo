# bin-billing-manager unit test coverage to 80% (VOIP-1657)

## 1. Ticket review and analysis

Verdict: valid. Measured at origin/main 161e9748b (merge base of the branch, after rebasing; the branch had been created on an older base that lacked the earlier billing-manager coverage work, so the first measurement was discarded) with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in the service directory (local vendor not used; no test failures). Hand-written code excludes `gens/`, `mock_*.go` and files whose header has `Code generated ... DO NOT EDIT`; duplicate blocks are merged with the max count.

Result: hand-written 76.49% (3172/4147 statements), raw 66.4%. 80.5% needs at least 3339 covered (4147 x 0.805 = 3338.3), so at least 167 more statements; the internal goal is 3360 (81.0%, +188) as margin. Before the PR is raised, `git fetch origin main` is run and the merge base is checked again; if bin-billing-manager changed on main in between, the tables are re-measured and this document is updated.

Uncovered statements (975): cmd/billing-control 652 of 652, cmd/billing-manager 174 of 174, and 149 outside cmd:

| package | uncovered |
|---|---|
| pkg/dbhandler | 80 |
| pkg/listenhandler | 34 |
| internal/config | 13 |
| pkg/billinghandler | 6 |
| pkg/failedeventhandler | 5 |
| pkg/accounthandler | 3 |
| models/billing, models/account | 3 |
| pkg/paddlehandler, pkg/cachehandler | 4 |
| pkg/subscribehandler | 1 |

Reachability (no signature change, no external connection):
- Unreachable: `main`, `runDaemon`, `run`, `initBillingHandlers` (RabbitMQ connect), the bodies of the `run*` commands after `initHandlers` (they need a live database), config binding errors, `json.Marshal` and squirrel `ToSql` error branches, `New*Handler` wiring constructors, and the `cobra.CheckErr` exit path.
- Reachable outside cmd (about 80 of 149; about 70 are `json.Marshal`, `PrepareFields` and `ToSql` branches): dbhandler row-scan and cache error branches (sqlmock or a broken sqlite handle, as the existing tests do), listenhandler error responses, `config.LoadGlobalConfig`, small branches in models, failedeventhandler and billinghandler.
- Reachable in `cmd/billing-control` (about 395): `initCommand`, the `cmd*` command builders, `resolveUUID`, `printJSON`, and the `initHandlers` failure branch of each `run*` command (the database address is a unix socket path that does not exist, so the connection fails at once). Estimate by function: `initCommand` 119, the 15 `cmd*` builders 214, `resolveUUID` and `printJSON` 12, `run*` failure branches about 2 to 3 each (about 50 in total), 395 in all. Input validation after `initHandlers` is not reachable without a database and is not promised.
- Reachable in `cmd/billing-manager` (about 35): `signalHandler`, `buildFailedEventHandler`, `runSubscribe`, `runListen` with gomock.

Conservative reachable total is about 300 and the middle estimate about 450, against 167 (goal 188) needed, so the reachable ceiling is above the target and no block is needed. Outside cmd alone (about 80) cannot reach 167, so the cmd group is required.

## 2. Plan

Only `*_test.go` files are added or extended inside bin-billing-manager, plus this document. No product code, signature, visibility or dependency change (`go.mod` and `go.sum` stay unchanged; miniredis and go-sqlmock are already present).

Order: (1) outside-cmd error branches (dbhandler, listenhandler, models, config, failedeventhandler, billinghandler), because they test behavior; (2) `cmd/billing-manager` small functions; (3) `cmd/billing-control` builders and failure branches, until the covered count reaches 3360. The PR is raised only at 80.5% or more. If a group yields less than half of its estimate, the remaining groups are re-estimated before continuing, and if the re-estimate cannot reach the target the work stops and is reported. If all groups are done and the total is in the 80.0% to 80.5% band, the remaining reachable branches are added until 80.5%; if none are left, the PR is raised with the exact value recorded. If the total is under 80.0%, the work stops and is reported.

Rules for every test:
- Table-driven, `Test_<function>` names (new tests only; existing names are not renamed), gomock controller named `mc` with `defer mc.Finish()`, synthetic values only, no `time.Sleep`, no network, no parallel tests.
- Exact mock arguments (`gomock.Any()` only for `ctx`), errors asserted with `errors.Is` or the exact response, not just `err != nil`. No self-comparison assertions. Expected values are written as literals, never computed with the product function under test.
- cmd tests: `config.LoadGlobalConfig` is guarded by `sync.Once`, so tests overwrite the process config through `*config.Get() = config.Config{...}` and restore it with `t.Cleanup`; `run*` functions are called directly (not through `Execute`), and the cobra paths that call `os.Exit` (`cobra.CheckErr`) are not invoked. `viper.Reset()` runs in `t.Cleanup`. The unreachable database is a mysql DSN with `unix(/nonexistent/db.sock)` and a short `timeout`, which fails deterministically without any network; the assertion is on the error text `could not connect to the database`. `Execute` and `PersistentPreRunE` are never called (they would rewrite the overwritten config and consume `once`). `config.Bootstrap`/`initLog` changes the logrus formatter and level, so tests restore both in `t.Cleanup`. DSN credentials are obviously fake. Stdout capture restores `os.Stdout` in `t.Cleanup`. No network host appears in tests. Command builder tests assert flag names, defaults (for example the payment-type default) and the subcommand tree as literals, not just `Use` or a non-nil `RunE`; the mutation sample includes renaming a flag.
- `cmd/billing-manager` `signalHandler` closes the package-level `chDone`, so each test replaces `chSigs` and `chDone` with new channels and restores them in `t.Cleanup`, injects the signal with `chSigs <-`, waits with `select` and a `time.After` bound, and calls `signalHandler` once per test, which keeps `-count=20` safe. `runListen` starts `ConsumeRPC` in a goroutine, so the test waits on a channel closed by the mock before the controller finishes; `buildFailedEventHandler` builds the real subscribe handler and never calls its `Run`. `runDaemon` (it calls `signal.Notify`) and `InitPrometheus` beyond its early return (duplicate `http.Handle`) are not called.
- The `internal/config` test that resets `once` restores `once` and the config in `t.Cleanup`.

Verify:
1. Hand-written total at least 80.5% by the measurement above; the PR has the per-package before/after table, hand-written and raw totals, the exclusion method and the command.
2. The final coverage is measured again with the same `-mod=mod` command. `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`, `go vet ./...`, `bash scripts/check-test-conventions.sh` run from the repository root after `git fetch origin main`; `git diff --exit-code go.mod go.sum`.
3. `go test -race -count=20` over the packages with new tests passes.
4. Mutation sample: for each new test group a mutant of the covered code must make the named test fail. The module uses relative `replace` paths, so a mutant is applied inside the worktree and undone with `git checkout -- <file>`, and `git status` is checked clean afterwards.
5. `git diff --stat origin/main...HEAD` lists only files under bin-billing-manager.
