# bin-api-manager unit test coverage (VOIP-1639)

## Ticket review
Baseline reproduced (and re-measured by an independent reviewer) at origin/main 3ce92781f with
`go test -count=1 -coverprofile=cp.out ./...`: 24.2% (16037/66313 statements). gens/openapi_server/gen.go is 37133
statements, mock_*.go 3222, hand-written 25958 at 61.8%. A 40% all-statement goal is unreachable (100% of
hand-written code is 39.1%), so the orchestrator redefined acceptance:
- hand-written coverage (gen.go and mock_*.go excluded; denominator keeps cmd/) >= 70%, i.e. >= +2135 statements;
  internal target 72% (about +2650) for margin
- pkg/cachehandler, pkg/ratelimithandler, pkg/websockhandler each above 18.3 / 25.0 / 33.2 as reported by `go test -cover`
  (includes mocks). The PR reports both measurements separately, and the filter script is the one used for the baseline.

## Scope
Test files only; product code, go.mod and go.sum unchanged. No network (only 127.0.0.1:0 httptest and miniredis), no
DB, PSTN, SMS or Email. Not targeted (rebase onto origin/main before the PR and re-measure the baseline): cmd/api-manager main (517 stmts) and cmd/api-control (73), kept in the
denominator; gen.go; mocks; websockhandler pingPeriod ticker branch (const 10s, not injectable; only the ctx-cancel
branch is tested); websockhandler media_stream.go / run_external_media.go (external media socket dependency) 
setSerialize/getSerialize are unused helpers (nolint); tested only through their real error contract, kept minimal.

## Plan (uncovered hand-written statements -> target gain)
Existing tests are read first; only gaps are added (no duplicate of subscription teardown, cache expired/missing/corrupt).
- infra: cachehandler (26) NewHandler/Connect/serialize errors via miniredis closed; ratelimithandler Allow redis error
  and Remaining decrement (range checks for RetryAfter, SetTime+FastForward for window); websockhandler (322, of which 119 are the excluded media files; about 203 targeted)
  receive*, subscriptionHandleMessage gaps, subscriptionRun via httptest+gorilla Dial with MockBroker/MockSubHandler
  whose RunWithMutex blocks on ctx (DoAndReturn) and Terminate expected; sync by channels, never time.Sleep.
- servicehandler (3578 uncov) in 3 file groups, server (5163 uncov) in 4 file groups (assignments by uncovered size:
  server: S1..S4 about 1290 uncovered each (5163 total); servicehandler: H1..H3 about 1190 each (3578 total); every
  file of both packages belongs to exactly one group), each target >= 40% of the group's uncovered
  statements (>= +3490 overall, above the +2650 need), 0% files first (cheapest), then auth/permission/error paths.
  Fallback if a group falls short: streamhandler (111), subscribehandler (28), other small packages.
  media_stream.go and run_external_media.go are out of scope (excluded).
Per resource: direct-access rejection, permission denied, cross-tenant, RPC error propagation, success;
server: 401, bad JSON, bad id, service error mapping, success with the exact response body.

## Parallel work rules
Each worker owns disjoint source files and creates NEW `*_test.go` files only (no edit to existing tests), with
file-unique helper names (prefix per group) to avoid duplicate symbols. After merging all work: gofmt, go vet,
full `go test ./...`, `go test -race -count=5` for websockhandler, golangci-lint.

## Verify
- Conventions for added lines only (scripts/check-test-conventions.sh checks added lines, needs `git fetch origin main`; existing files keep testify): `Test_` names,
  `reflect.DeepEqual` + `t.Errorf`, no testify, gomock controller named `mc`.
- gomock.Any() only for ctx; exact RPC arguments, exact status/error codes. No test without an assertion; no
  self-comparison. Guard tests assert the specific error; negative control sampled (3-5 per group) in a scratch copy.
- Stability: `go test -race -count=5` for websockhandler; every wait bounded by a timeout.
- Secrets/IP check on added lines (grep IP, token, real phone patterns); only synthetic values (555 numbers, example.com).
- `git diff --stat -- . ':!*_test.go' ':!bin-api-manager/docs/plans'` (run from the repository root) is empty; go generate leaves git status clean.
- Before/after table per package in the PR body; commit and PR title = branch name VOIP-1639-Raise-api-manager-test-coverage.

## Result (measured after the work, same method as the baseline)
Hand-written coverage 61.8% (16038/25958) -> 89.6% (23267/25958). All-statement 24.2% -> about 35.1%.
`go test -cover`: cachehandler 18.3 -> 61.7, ratelimithandler 25.0 -> 31.2, websockhandler 33.2 -> 69.0,
servicehandler 49.0 -> 69.1, server 60.5 -> 94.1. Ratelimit window expiry used Period=1h with range checks
(no SetTime/FastForward). pkg/mediahandler, cmd/api-manager, cmd/api-control, media_stream.go and
run_external_media.go stay at 0%.
