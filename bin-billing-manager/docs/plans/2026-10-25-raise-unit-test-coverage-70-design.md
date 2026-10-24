# VOIP-1651: bin-billing-manager unit test coverage to 70%

## 1. Ticket review (analysis)

- Baseline (origin/main 99b59c386, `GOFLAGS=-mod=mod go test -count=1 -coverprofile`): hand-written 69.64% (2888/4147), raw 60.5%. Reproduced.
- Hand-written excludes `mock_*.go`, `gens/`, and files whose header has `Code generated ... DO NOT EDIT`; duplicate blocks keep the max count.
- Uncovered statements by package: cmd/billing-control 652 and cmd/billing-manager 174 (both `main` only, treated as not testable), pkg/dbhandler 122, pkg/listenhandler 106, pkg/cachehandler 71, pkg/accounthandler 46, pkg/subscribehandler 36, pkg/billinghandler 19, internal/config 13, models/billing 10.
- Reaching 71% needs 57 more covered statements; 70% needs 15. The two `main` packages (826 statements) are left out of the plan; without them the hand-written share is 87.0% (2888/3321), and the other uncovered statements (433) are enough to reach the target.
- Scope: test files only plus go.mod/go.sum (miniredis is not yet in this module; v2.36.1 is already used by other services). No product code change. VOIP-1416 overlap is not asserted here; it is checked and stated in the PR body only.

## 2. Plan

| Target | Uncovered | Approach |
|---|---|---|
| pkg/cachehandler | 71 | miniredis (v2.36.1, already used by other services). Round trip, miss, corrupt JSON, write failure after server close, `Connect` ok and fail, `NewHandler` |
| pkg/listenhandler/v1_hooks_paddle.go | 44 | Table cases for malformed JSON, invalid customer id, bad amount, handler error mapped to 500, refund lookup failure, plan-type validation |
| pkg/listenhandler/v1_accounts.go | 49 | Missing URI segment (400), malformed body (nil response plus error), handler error mapped through `errorResponse` |
| models/billing/webhook.go | 9 | `ConvertWebhookMessage` with every field set to a distinct value, asserted field by field from independent literals, plus `CreateWebhookEvent` JSON keys |
| pkg/accounthandler/paddle.go | up to 33 | Error returns of the Paddle handlers with gomock (db error, cache error) |

The table sums to about 206 statements at most; the realistic gain is about 150 to 175 (about 73 to 74%). Items are added in the order above until the hand-written total is at least 71%.

## 3. Rules for the tests

- No empty calls and no self-comparison; every case asserts a distinct response or error.
- Synthetic values only; no network, no external calls; no `user:pass@host` strings and no password literal next to an address.
- Test names `Test_<Func>_<case>`; gomock controller named `mc`.
- Defects found while testing are not fixed and not described in files; they are reported on the card only.

## 4. Verify

1. Hand-written total at least 71.0% using the reference method; per-package before/after table in the PR body.
2. `go mod tidy` leaves no diff; `go test ./...` and `golangci-lint run` pass.
3. `git diff --stat origin/main...HEAD` lists only files under bin-billing-manager.
4. Mutation sample: revert one guarded statement in a scratch copy for several new tests and confirm each fails.
