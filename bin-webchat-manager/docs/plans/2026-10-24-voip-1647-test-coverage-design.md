# VOIP-1647 bin-webchat-manager test coverage: plan

Scope: add `_test.go` files only inside bin-webchat-manager. No product code change, no signature or visibility change. Analysis: `2026-10-24-voip-1647-test-coverage-analysis.md`.

## Test units (priority order)

1. `pkg/listenhandler/main_test.go`
   - `errorResponse`: nil error (500), typed `VoipbinError` (status from `cerrors.ToResponse`), typed nil `*VoipbinError` so that `ToResponse` fails (500), `dbhandler.ErrNotFound` (404), wrapped `ErrNotFound` (404), other error (500).
   - `simpleResponse`.
   - `processRequest`: a table of all 15 method and URI pairs plus negative cases with a suffixed or prefixed URI (for example GET on `/v1/widgets/<uuid>/direct-hash-regenerate`, or `/v1/widgets/<uuid>/x`) so that loosening an anchor or regex fails a strict mock, each asserting that exactly the matching mock method is called with the parsed arguments (gomock strict, so a mis-route fails); unknown URI and known URI with a wrong method give 404; untyped handler error gives 400; typed error and `ErrNotFound` pass through `errorResponse`.
   - `Run`: `QueueCreate` error is returned and `ConsumeRPC` is not called; success path calls `ConsumeRPC` once with the queue name, the callback argument captured in `DoAndReturn` and checked non-nil, synchronized by a channel signalled in `DoAndReturn` with a 2 second timeout.
   - `NewListenHandler`: returned value is non-nil, holds the given handlers, and has a non-nil `utilHanlder`.
   - Tests build `&listenHandler{...}` literals; `init()` (prometheus registration) runs once per binary and is never re-run. `utilHanlder` is unused by `v1_*.go`, so nil is safe.
2. `pkg/listenhandler/v1_widgets_test.go`, `v1_sessions_test.go`, `v1_messages_test.go`, per handler function: success (all request fields forwarded exactly, 200, content type, body unmarshals to the mock's return), invalid JSON (400, no handler call), short URI and nil-UUID URI (400, no handler call), unparsable filter body (400) and a UUID-typed filter field given a non-UUID string such as `customer_id: "not-a-uuid"` (400, the only `ConvertFilters` failure; other types pass through), a URI with a control character such as `\x01` (not `\n`, which fails the route regex and gives 404) so that `url.Parse` fails in the three list handlers (they return `nil, err`; through `processRequest` this becomes 400), handler error mapping (typed, `ErrNotFound`, other), list pagination parsing (`page_size`, `page_token`, missing and non-numeric size give 0), marshal failure paths only where a value can make `json.Marshal` fail without product change (otherwise left uncovered and noted).
3. `models/{message,session,widget}/webhook_test.go`: field mapping from the model to `WebhookMessage`, internal fields excluded (message `ActiveflowID`, widget `DirectID`, session `ActiveflowID` and `TMDelete` which has `json:"-"`), `CreateWebhookEvent` JSON keys asserted against literal expected key sets.
4. `internal/config/config_test.go`: `Bootstrap` on a fresh `cobra.Command` with `t.Setenv` values and defaults; per case `viper.Reset()` and `once = sync.Once{}` (same package), logrus formatter and level and `globalConfig` restored with `t.Cleanup`; `LoadGlobalConfig` and `Get`.
5. Fallback only if the measured total is below 61% after 1 to 4: dbhandler cache-hit returns and `ErrNotFound` empty-result returns and `len(fields)==0` returns, using the existing sqlite harness and `MockCacheHandler`; handler package error branches. `cachehandler` with miniredis is the last resort because it adds a test-only module requirement; if used, run the full verification workflow and report the go.mod and go.sum diff.

## Verify

- Hand-written total >= 61% by the measurement method of the stated baseline (exclusion: `mock_*.go`, `gens/`, DO NOT EDIT headers, duplicate blocks merged by max count); before and after per package table in the PR. Listenhandler about 90% is an estimate to be confirmed.
- `go test -race -count=1 ./...`, `golangci-lint run -v --timeout 5m`, `bash scripts/check-test-conventions.sh` (repo root script) (Test_ naming, no testify, gomock controller named `mc`) pass. The script inspects `git diff <merge-base> HEAD`, so it runs after the commit with `origin/main` fetched; before the commit it checks nothing.
- Mutation sampling in a scratch copy, one mutation at a time, a new test must fail for each: wrong status code in an error branch, dropped argument in a forwarded call, `ErrNotFound` mapped to 500, removed nil-UUID guard, a route regex loosened, `errorResponse` fallthrough changed, a webhook field mapping changed, a config default changed. Results are reported in the PR description.
- No tautological tests: expected values are literals, not built from the code under test.
- Synthetic data only, no network, no external cost.
- `git diff --stat origin/main...HEAD` lists only bin-webchat-manager paths and only `_test.go` plus docs files (and go.mod and go.sum only if miniredis is used).

## Out of scope

`cmd/*` (signal, broker, database and cache connections), generated and mock files, product code defects (not fixed here).
