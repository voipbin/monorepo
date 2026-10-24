# VOIP-1651: bin-billing-manager unit test coverage to 70%

## 1. Ticket review (analysis)

- Baseline (origin/main 99b59c386, `GOFLAGS=-mod=mod go test -count=1 -coverprofile`): hand-written 69.64% (2888/4147), raw 60.5%. Reproduced.
- Hand-written excludes `mock_*.go`, `gens/`, and files whose header has `Code generated ... DO NOT EDIT`; duplicate blocks keep the max count.
- Uncovered statements by package: cmd/billing-control 652 and cmd/billing-manager 174 (both `main` only, treated as not testable), pkg/dbhandler 122, pkg/listenhandler 106, pkg/cachehandler 71, pkg/accounthandler 46, pkg/subscribehandler 36, pkg/billinghandler 19, internal/config 13, models/billing 10.
- Reaching 71% needs 57 more covered statements; 70% needs 15. The two `main` packages (826 statements) are left out of the plan; without them the hand-written share is 87.0% (2888/3321), and the other uncovered statements (433) are enough to reach the target.
- Scope: test files only plus go.mod/go.sum (miniredis is not yet in this module; v2.36.1 is already used by other services). No product code change. Overlap with the general coverage uplift of packages below 80% is stated in the PR body only.

## 2. Plan

| Target | Uncovered | Approach |
|---|---|---|
| pkg/cachehandler | 71 | miniredis v2.36.1 with `miniredis.RunT(t)` and go-redis/v8, same shape as bin-customer-manager's cachehandler test. Get hit, miss, corrupt JSON; Set failure after server close; `Connect` ok and fail (closed server address); `NewHandler` |
| pkg/listenhandler/v1_hooks_paddle.go | 44 | Cases per handler: malformed envelope, malformed data, invalid customer_id, unknown plan_type, amount parse failure (400), each accounthandler error mapped to 500, refund lookup failure (500), refund with several adjustments (sum) and negative amounts (absolute value), exact gomock arguments |
| pkg/listenhandler/v1_accounts.go | 49 | Per handler: URI too short (400), malformed body, accounthandler error mapped by `errorResponse`. Expected value per case follows the source of that handler |
| models/billing/webhook.go | 9 | Every field set to a distinct value; expected `WebhookMessage` written as an independent literal; `CreateWebhookEvent` decoded into a map and keys and values asserted |
| pkg/accounthandler/paddle.go | up to 33 | Per Paddle handler: idempotency check error other than not found, already processed event skipped, account lookup failure, plan-type update failure, status update failure, credit and token top-up db errors, refund subtract failure; errors checked with `errors.Is`. Unknown-plan branches that an earlier step already short-circuits are left uncovered |

The table sums to about 206 statements at most; the realistic gain is about 150 to 175 (about 73 to 74%). Items are added in the order above until the hand-written total is at least 71%. Branches that cannot be reached from a test (for example the second of two consecutive cache writes failing after the first succeeded) are left uncovered.

The expected value of each case follows the source of its handler. Branches that only guard `json.Marshal` failure of a plain struct are unreachable from a test and stay uncovered, so the Uncovered column above is an upper bound.

## 3. Rules for the tests

- No empty calls and no self-comparison; expected values are independent literals, never derived from the input fields under test.
- Cache tests: Get paths use directly seeded keys; Set paths assert the primary key, its stored content and its TTL. Fixture ids are distinct UUIDs.
- Synthetic values only; no network, no external calls. Redis address comes from `s.Addr()` at run time, password argument is an empty string, no DSN with credentials, no email, phone or token-shaped strings (no `pdl_` or API key prefixes).
- Test names `Test_<Func>_<case>`; gomock controller named `mc`; where a table-driven test already exists (hooks, accounts) new cases are added to that table or a new function with a distinct name is used. A failed `Connect` asserts only that an error is returned.

## 4. Verify

1. Hand-written total at least 71.0%: `GOFLAGS=-mod=mod go test -count=1 -coverprofile` then aggregate the profile excluding `mock_*.go`, `/gens/` and files whose header has `Code generated` and `DO NOT EDIT`, keeping the max count for duplicate blocks (baseline 69.64%, 2888/4147); per-package before/after table in the PR body.
2. `go mod tidy` run twice leaves no further diff; go.mod/go.sum change only by miniredis and its transitive entries; `go list -deps ./cmd/...` does not contain miniredis (test-only import).
3. `go test -race -count=1 ./...` and `golangci-lint run` pass.
4. `git diff --stat origin/main...HEAD` lists only files under bin-billing-manager.
5. Mutation sample in a scratch `git archive` copy (never the worktree), at least these six, each with the test that fails recorded: cache primary key prefix change, `getSerialize` error return removed, one `ConvertWebhookMessage` field dropped, refund absolute value removed, a Paddle handler 500 mapped to 200, the 24h TTL of `setSerialize` removed (asserted with the miniredis TTL of the stored key).
