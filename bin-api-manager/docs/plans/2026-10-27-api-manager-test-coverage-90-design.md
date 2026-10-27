# bin-api-manager unit test coverage to 90% (VOIP-1663)

## 1. Ticket review
Verdict: valid. Measured on origin/main with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`; hand-written code (excluding `gens/`, `mock_*.go`, `DO NOT EDIT` files, duplicate blocks merged by max count) is 89.75% of 26176 statements. About 65 more statements reach 90.0%; the target is 90.5% (about 196 statements) to leave margin for CI rounding differences. An open PR that touches only `server/recordings*` exists; those files are not edited here (re-checked with `git fetch` and `git merge-tree` before the PR).

## 2. Analysis
Most uncovered statements are success paths and secondary error paths in `server/` handlers and `pkg/servicehandler/` functions, where existing tests cover only authentication and service-error cases. In several handlers (for example `providercalls`) even argument parsing and validation branches are uncovered. Service startup code in `cmd/` (about 590 statements) needs external connections and is out of scope; excluding it the ceiling is about 97%, far above 90%, so no scope reduction is needed.

## 3. Design
- `server/recordings*` is excluded (touched by another open PR).
- Test-only change inside `bin-api-manager`; no production code, signature or exposure change.
- Order by uncovered statements: server handlers with untested success paths (providercalls, rags, service_agents_conversations, agents, webchat_messages, conversations, contacts), then servicehandler functions with untested permission, not-found and RPC-error branches (serviceagent_file, outdialtargets, speaking, aisummary, serviceagent_aicall, conversation_account, storage_file, tag, message), then small packages (websockhandler, streamhandler, lib/service) only if needed and only where no real socket or external dependency is required.
- Follow the existing sibling pattern of each file: `s2Case` harness for server error envelopes, gomock `MockServiceHandler` / `MockRequestHandler` for success paths, exact argument matching, no sleeps, no network, synthetic IDs only.
- Each test asserts observable output (HTTP status, JSON body, arguments forwarded to the mock, returned error), never a value built by the test itself.

## 4. Verify
- Hand-written total >= 90.5% with the same command and script as the baseline; all existing tests pass; `golangci-lint run` passes.
- `go test -race -count=3 -shuffle=on` over the touched packages passes (isolation check).
- Mutation sample (at least 5): flip a condition in a few covered handlers and confirm a new test fails.
- The standard `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run` flow is also run.
- `git diff --name-only origin/main...HEAD` lists only new `*_test.go` files and this document (no production code, `go.mod`, `go.sum` or `mock_*.go`).
- grep over the added lines finds no `time.Sleep`, network call or real host/account value; each new test asserts forwarded mock arguments and response body (success) or status and reason (error).
- The PR records per-file coverage deltas.
