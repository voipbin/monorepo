# bin-tag-manager unit test coverage to 80% (VOIP-1657)

## 1. Ticket review

Baseline on origin/main 161e9748b (hand written code only; gens/, mock_*.go and `Code generated ... DO NOT EDIT` files excluded, duplicate blocks merged by max count): 842 / 1080 statements = 77.96%. About 22 more statements are needed for 80.0%, 28 for 80.5%.

| Package | Hand statements | Covered | % |
|---|---|---|---|
| cmd/tag-control | 230 | 130 | 56.5 |
| cmd/tag-manager | 138 | 48 | 34.8 |
| internal/config | 78 | 76 | 97.4 |
| models/tag | 9 | 8 | 88.9 |
| pkg/cachehandler | 39 | 38 | 97.4 |
| pkg/dbhandler | 142 | 128 | 90.1 |
| pkg/listenhandler | 232 | 207 | 89.2 |
| pkg/subscribehandler | 56 | 52 | 92.9 |
| pkg/taghandler | 156 | 155 | 99.4 |

## 2. Scope and reachable ceiling

Only `_test.go` files are added. No product code, signature or go.mod change.

Reachable uncovered statements (no signature change, no live DB, Redis or RabbitMQ):

- taghandler `Get` not-found wrapping: 1.
- listenhandler: `Run` consume-error log 1; `processRequest` error branch (typed error, not-found, legacy 400) 7; `processV1TagsGet` URL parse error 1, filter parse error 2, filter conversion error 2; total 13 (the `List` error path is already covered by existing tests). Two statements in the goroutines of `listenhandler.Run` and `subscribehandler.Run` count only when the goroutine runs before the test ends, so the baseline can read 840 instead of 842; the new tests synchronize on those goroutines.
- subscribehandler: `Run` error returns (topic create, bind) 2, consume-error log 1, `processEvent` handler-error log 1; total 4.
- dbhandler: scan-error branches (`tagGetFromRow`, `tagGetFromDB`, `TagList`; a sqlmock row with a wrong column count) 3, `tagUpdateToCache` error propagation 2 (a `tagGetFromDB` failure and a cache-set failure, one test each); total 5.
- cmd/tag-manager `run`: config init failure (a nil flag from `InitConfig`, not the `Bootstrap` bind errors that are excluded) 3, database connect failure about 9; total about 12.
- cmd/tag-control: `initHandler` and its failure branch for `runCreate`, `runGet`, `runList`, `runUpdate`, `runDelete` with a closed loopback DSN, plus the migrate closure; measured on a scratch copy at +19.

Reachable total is about 54 (1 + 13 + 4 + 5 + 12 + 19), so the ceiling is about 896 / 1080 = 82.9%, above 80%. No block is needed. Without cmd/tag-manager the total is about 42 (884 / 1080 = 81.9%), which alone reaches 80.5%. The estimate was reproduced on scratch copies.

Unreachable (excluded): bind errors in config, json.Marshal failures, ToSql errors, and the `PrepareFields` and `ApplyFields` errors of `TagCreate` and `TagDelete` (fixed input); `TagList` with an unsupported filter value type and `TagUpdate` with a non-marshalable field value are reachable and may be added as small extra cases, `errorResponse` ToResponse failure, `main`, the success paths of `initHandler` and `initTagHandler`, the `run*` bodies of tag-control after a successful `initHandler`, `runService`, the `initProm` listener loop.

## 3. Plan

Conventions (scripts/check-test-conventions.sh): `Test_` names, `reflect.DeepEqual` with `t.Errorf`, gomock controller named `mc`. All inputs are synthetic.

1. taghandler: `Get` with a not-found db error returns a typed not-found error that still wraps `dbhandler.ErrNotFound`; other errors pass through unchanged.
2. listenhandler: `processRequest` table: a handler returning a typed error maps to its status, a not-found error to 404, a plain error to 400 (a URI such as `/%zz/v1/tags?a=b` makes `processV1TagsGet` return a parse error, because the route regex is not anchored at the start and an invalid escape in the path part fails `url.Parse` (one in the query part does not); the typed and not-found cases come from the mocked tagHandler on the id routes). `processV1TagsGet` bad filter body and unknown filter field give 400 with the tag handler never called. `Run` with `ConsumeRPC` returning an error is logged and `Run` still returns nil; the mock signals a channel from `DoAndReturn` and the test waits on it with a timeout in `select`.
3. subscribehandler: `Run` with `TopicCreateWithKind` error and `QueueBind` error returns the wrapped error and does not start consuming; `ConsumeMessage` error does not fail `Run` (same channel and timeout synchronization); `processEvent` is called directly (synchronous) for a customer_deleted event whose handler fails: it does not panic and the tag handler mock is called once.
4. dbhandler (sqlmock): a row with a wrongly typed value gives a scan error from `tagGetFromDB` and `TagList`; `tagUpdateToCache` returns the cache-set error when the cache mock fails.
5. cmd/tag-manager: `run` with an empty `cobra.Command` returns the wrapped config error; `run` with a fresh `cobra.Command` per call that defines all 7 flags and a closed loopback DSN (`u:p@tcp(127.0.0.1:<port>)/d`, fake values; the port comes from `net.Listen("127.0.0.1:0")` closed before use, as the existing `Test_initCache` does) returns the database ping error. The package global `rootCmd` is not used. `initProm` registers on the default mux and panics on a duplicate pattern, so each call uses a unique endpoint from a package level `atomic.Int64` counter (`/m-<n>`) and the listen address `127.0.0.1:0` (never a fixed port). Host env keys (`DATABASE_DSN`, `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS` and the other config keys) are fixed with `t.Setenv` and `viper.Reset()` runs in cleanup, as `internal/config` tests do. `InitConfig` overwrites the global config, so cleanup also restores `*config.Get()` to its saved value. The `initProm` listener goroutine cannot be stopped and leaks by design; unique endpoints and `127.0.0.1:0` keep it safe under `-race` and `-count=N`.
6. cmd/tag-control: `create`, `get`, `list`, `update`, `delete` through `initCommand` (arguments set with `cmd.SetArgs`) with a closed loopback DSN return `failed to initialize handlers`; `migrate status` returns an error. Same env and viper isolation as item 5 and the existing tests in that package; `LoadGlobalConfig` runs once per process and ignores later flag values, so the closed DSN is injected with `config.InitConfig` (the existing `injectConfig` helper) before each case and the same DSN is passed as a flag, which makes the result the same whether the once fired earlier or not. `runCreate` needs a valid customer id and name because it validates before `initHandler`; the other four fail in `initHandler` without arguments. Cleanup restores the saved config.

## 4. Verify items

- Measurement: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=<scratch>/cp.out ./...`, hand total excluding generated files; per package before and after table. Target at least 870 / 1080 (80.5%).
- Existing tests pass; `golangci-lint run -v --timeout 5m` clean; `go mod tidy`, `go generate ./...` produce no diff; `scripts/check-test-conventions.sh` clean; `go test -race -count=3 -shuffle=on` passes on touched packages.
- Mutation sample: remove one guarded branch in a scratch copy (for example the not-found wrapping, the 400 default mapping, the bind-error return) and confirm the matching new test fails.
- A pre-measurement check compares the per-package gain with the estimate (tag-control about +19, tag-manager at least +12, others about +23) before the final run.
- `git fetch origin main` runs before `scripts/check-test-conventions.sh`; tests that use `t.Setenv` never call `t.Parallel`; the two goroutine statements in item 2 and 3 are covered by waiting on the channel before the test returns.
- `go mod vendor` is part of the verification run; `git diff --name-only origin/main...HEAD` lists only bin-tag-manager test files and this document; a grep over the diff finds no real IP, credential or DSN with a real password; the tests are run with nothing listening on 3306, 6379 and 5672 and still pass.
- `go test -race -count=3 -shuffle=on ./...` is also run for the whole module. The existing dbhandler tests share one in-memory sqlite database and are not repeatable with `-count` above 1 (this predates the change); for that package `-count=1 -shuffle=on` runs over the whole package and `-count=5` over the new tests only.
- `go test -race -count=3 -shuffle=on` is run on cmd/tag-manager and cmd/tag-control explicitly; the new tests never touch a fixed port (2112, 3306, 6379) and use only fake credentials and loopback addresses.
- No empty call tests and no self comparison; every test asserts concrete values or error text.
