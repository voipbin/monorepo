# VOIP-1647 bin-message-manager unit test coverage to 60%

## 1. Ticket review
Valid. Baseline on origin/main d68e695b5: hand-written 56.7% (780/1375 statements), raw 52.5%. Hand-written excludes `gens/`, `mock_*.go` and `Code generated ... DO NOT EDIT` files; duplicate blocks are merged with max count.

## 2. Analysis
Target: 61% or more (about 59 more statements). Largest uncovered hand-written code:

| Area | Missed stmts | Testable without product change |
|---|---|---|
| cmd/message-control, cmd/message-manager | 364 | Partly. Wiring (runService, runListen, initHandler) connects DB, Redis, RabbitMQ and is excluded. Pure helpers and flag validation before initHandler are testable but depend on global viper/cobra state, so optional |
| pkg/requestexternal/telnyx.go | 43 | Yes: uses http.Client{} (DefaultTransport), swap the transport in the test |
| pkg/cachehandler | 43 | Yes: miniredis (test-only dependency, already used by other services at the same version) |
| pkg/messagehandler/message.go | 33 | Yes: gomock dbhandler |
| pkg/listenhandler/messages.go, main.go, hooks.go | 35 | Yes: error branches (bad URI, bad filters, handler errors) |
| pkg/messagehandler/db.go, hook.go, send.go | 34 | Yes: gomock error paths |
| pkg/dbhandler/message.go | 19 | Yes in principle, but out of scope (see section 3) |
| internal/config, messagehandlermessagebird/send.go, models/message/webhook.go | 8 | Minor, only if needed |
| pkg/requestexternal/messagebird.go | 16 (already 82%, existing test covers the canceled context; the rest is after client.Do) | No: builds its own http.Transport and fixed URL. Excluded |

Achievable: about 190 reachable statements (telnyx has 2 to 3 unreachable ones: json.Marshal and NewRequestWithContext failures), 59 needed.

miniredis: v2.36.1 (used by bin-contact-manager, bin-ai-manager and others; only bin-api-manager uses v2.38.0). go.mod and go.sum of bin-message-manager change accordingly.


## 3. Plan
Test files only (added to existing `*_test.go` files where one exists), plus go.mod/go.sum for miniredis. No product code change. New test names carry a scenario suffix (e.g. `Test_Get_notFound`) because `Test_Get`, `Test_List`, `Test_Delete`, `Test_Create` already exist in the package. Uncovered lines are mapped with `go tool cover -html` before writing each test. dbhandler, config and messagehandlermessagebird gaps are out of scope, like cmd/ and messagebird.go.
1. telnyx_test.go: fake RoundTripper that fails the test on an unexpected host. Checks method, URL, headers (Bearer plus a fake token), JSON body. Status boundaries 199, 200, 299, 300, 404, 500: every non-2xx case asserts the error text (`status code N` and the body); the 199 and 300 bodies are valid JSON (`{}`) so a `> 300` mutation cannot be caught by an unmarshal error instead, malformed JSON body (status 200, the only status that reaches Unmarshal), transport error, body read error (Body whose Read fails). http.DefaultTransport is replaced and restored by t.Cleanup registered in the same function; no t.Parallel. httptest is not usable because the host is fixed.
2. cachehandler with miniredis (t.Cleanup(mr.Close)): Connect ok, Connect fail against a closed miniredis address, MessageSet then MessageGet round trip, key `message:<id>` and TTL 24h via mr.TTL, expiry via mr.FastForward then Get fails, missing key, corrupt cached JSON, optional: delSerialize then Get fails (unused in production, low value; the nolint comment stays). Marshal failure in setSerialize is unreachable.
3. messagehandler (message_test.go, db_test.go, hook_test.go, send_test.go):
   - Create: id generated when uuid.Nil, db create error, get-after-create error, webhook published.
   - List ok and error; Get ok, ErrNotFound becomes typed NotFound (errors.As), other error passes through; Delete ok, db error, get-after-delete error; dbUpdateTargets update error and get error.
   - Hook: unknown uri, uri with an unexpected format (error), bad JSON, a message without a recipient, sent and finalized events ignored (nil, nil), number list error, no number, Create error, success.
   - Send error paths pass the identity gate with an internal system customer ID or a verified CustomerV1CustomerGet expectation (stated per test). Cases: balance error, insufficient balance, Create error. A nil source is not an error path (the code only guards the dereference and continues to Create and the goroutine), so it is a success-path case synchronized with a channel or WaitGroup, if added. The identity rejection case already exists (`Test_Send_unverified`). These return before the goroutine starts, so no sleep is needed. The goroutine success path, if extended, is synchronized with a channel or WaitGroup, never a longer sleep; provider iteration order is random so those mock calls use AnyTimes.
4. listenhandler: the uncovered lines are identified first. Planned cases: GET list with a bad URI (the error is returned to processRequest and mapped to 400), bad filters 400, handler error mapped, ok; POST bad JSON 400, handler error; ID GET/DELETE with a short URI 400 (by calling the handlers directly); error mapping (NotFound is already covered, not rewritten); hooks bad JSON 400, hook error still 200; Run with QueueCreate failure, added as a case of the existing TestRun table (queueError/expectError fields). Branches that no route can reach and the `ToResponse` failure branch of errorResponse are accepted as uncovered.

## 4. Verify
- Standard workflow: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`; tidy and vendor must pass after miniredis is added.
- Mode note: coverage uses `-mod=mod` (needs miniredis in the module cache); the offline run uses the default mode with the tidied go.mod/go.sum.
- Race and order: `go test -race -shuffle=on -count=3 ./...` passes (existing Send tests leave goroutines running, so interference is measured once).
- Measurement: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=$TMPDIR/cp.out ./...` (outside the worktree), then the hand-written total with the same exclusion (gens/, mock_*.go, DO NOT EDIT header, duplicate blocks merged by max count) via a script that is not committed; the PR body describes the method, and the same method reproduces the 56.7% baseline on origin/main. The result must be 61% or more. Per-package before/after table in the PR.
- Mutation sample (scratch copy only, sibling module replace paths symlinked; never the worktree), at least 6, each mapped to the test that fails: Telnyx status boundary; Get NotFound mapping removed; Send `!valid` check removed; hook guard order changed; cache TTL 24h to 1h; processRequest error post-processing status changed.
- Offline: the Telnyx fake RoundTripper fails on any unexpected host, so no request leaves the process; where the environment permits, tests also pass under `unshare -rn sh -c 'ip link set lo up && go test ...'` (loopback is needed by miniredis); new tests contain no `time.Sleep` (grep on added lines).
- Hygiene: fixtures use fictitious numbers, `203.0.113.x`, fake tokens; no self-comparing assertions (the mutation sample is the evidence); `git fetch origin main` then `scripts/check-test-conventions.sh` clean (Test_ prefix, gomock controller `mc`, no testify).
- Scope: `git diff --name-only origin/main...HEAD` lists only `bin-message-manager/**/*_test.go`, go.mod, go.sum and docs/plans; no non-test `.go` file changed (`git diff --name-only origin/main...HEAD | grep '\.go$' | grep -v _test.go` is empty).
