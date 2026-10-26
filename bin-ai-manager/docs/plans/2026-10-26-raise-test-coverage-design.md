# bin-ai-manager: raise hand-written unit test coverage to 80%

## 1. Analysis

### 1.1 Ticket review

- Baseline re-measured on origin/main a03bccd8 (same command as the ticket): hand-written 77.83% (12339 of 15854 statements). The number matches the ticket. No open PR touches the service. Recent merged work on the path is VOIP-1655 (dbhandler test isolation) and VOIP-1653 (builderhandler sentinel flake); neither changes the premise.
- Needed: 345 more covered statements for 80.0%, 424 for 80.5%.
- Measurement: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in `bin-ai-manager/`. Hand-written excludes `gens/`, `mock_*.go` and files with a `Code generated ... DO NOT EDIT` header; duplicate blocks keep the maximum count.

### 1.2 Reachable ceiling

The uncovered statements are concentrated in packages that can be exercised without any external connection:

| Package | uncovered (before) | how it is reached |
|---|---|---|
| pkg/dbhandler | 548 | in-memory sqlite test database already used by the package tests |
| pkg/listenhandler | 513 | gomock handlers, RPC request in, RPC response out |
| pkg/cachehandler | 138 | in-process miniredis (already a dependency of this module) |
| pkg/aipromptproposalhandler | 90 | gomock |
| pkg/subscribehandler | 87 | gomock |
| pkg/aiaudithandler, pkg/geminiaudithandler, pkg/geminiproposalhandler | ~120 | gomock and fakes |
| models/*, internal/config | ~110 | pure functions and setters |
| cmd/ai-control | 546 | only the command constructors (`initCommand`, `cmd*`) and pure helpers; the `run*` and `init*Handler` paths need live connections and are left out |

The table lists the main contributors only; packages not listed (aicallhandler, engine_openai_handler, engine_dialogflow_handler, summaryhandler and others) also have uncovered statements and are candidates when a listed package is exhausted (the single file `aicallhandler/tool.go` alone holds about 127 uncovered statements).

Branch types treated as unreachable and left out of the target: config binding errors, always-nil error returns, JSON marshal failures, query builder `ToSql` errors, `PrepareFields` and `ApplyFields` errors, random source failures, constant-based branches, server run goroutines, constructors that need an external connection, and `main`.

The reachable figure is the sum over the packages in the table above (cmd/ai-control excluded), measured after removing blocks whose source is one of the unreachable types; it is on the order of 1100 or more against a need of 424 (the sum of the listed packages before removal is about 1140; the figure after removal depends on a keyword heuristic and is used only as a sanity bound, the conclusion does not rest on it). Even after removing those and all of `cmd/ai-manager` and the connection-bound part of `cmd/ai-control`, the remaining reachable uncovered statements in the packages above are well over 1000, against a need of 424. The ceiling is therefore above 80% and the work proceeds without escalation. The target is 81% or more (about 500 statements) to leave margin for CI rounding.

### 1.3 Constraints that shape the work

- Test files only, inside `bin-ai-manager/`. No change to product code, signatures or exported names.
- `go.mod` and `go.sum` stay unchanged (miniredis and sqlite are already present).
- Tests must not depend on wall clock, ordering or shared state: dbhandler tests reset the shared sqlite database through the existing `resetTestDB`; time is injected through the mocked util handler; Redis state lives in a per-test miniredis.
- Test conventions (`scripts/check-test-conventions.sh`): `Test_` prefix, gomock controller named `mc`.
- Overlap with the general lift of packages under 80% (VOIP-1416) (general lift of packages under 80%): dbhandler, listenhandler, cachehandler, aipromptproposalhandler, subscribehandler, geminiaudithandler, geminiproposalhandler and engine_* are under 80% here and may overlap; noted only, not absorbed.

## 2. Design

### 2.1 Target and budget

Goal: hand-written total of 81% or more (at least 503 newly covered statements; 80.5% needs 424). Work proceeds in the order below and stops once the measured total passes 81% with all tests green; a later group is skipped if an earlier one already delivers the margin.

| Group | Package and files | Approach | Expected gain |
|---|---|---|---|
| G1 | pkg/dbhandler: `aipromptproposal.go` (all except `AIAcceptProposal`), `team.go`, `aiaudit.go` (all except `AIAuditUpsert`, whose `ON DUPLICATE KEY UPDATE` statement the sqlite harness cannot run), `mcpoauthstate.go`, `summary.go` (delete), `ai.go` (all except `AIActivateInsight`; both excluded functions use `SELECT ... FOR UPDATE`, which the sqlite harness cannot run and which the existing tests already skip) | new `*_test.go` files using the existing sqlite harness and `resetTestDB`; mocked util handler supplies time and ids | ~200 |
| G2 | pkg/cachehandler: `handler.go`, `team.go`, `listen.go` | per-test miniredis, same setup helper as `listen_test.go`; assertions cover key, TTL and value round trip, and the serialize error paths (closed server, corrupt stored JSON) | ~100 (almost all reachable through miniredis; only the marshal failure is not) |
| G3 | pkg/listenhandler: `v1_aiaudits.go`, `v1_services.go`, `v1_teams_direct_hash.go`, `v1_aipromptproposals.go`, `v1_mcpservers.go`, `v1_teams.go`, `v1_summaries.go`, `v1_ais.go` | table tests with gomock handlers: invalid JSON, invalid id, handler error, success | ~200 |
| G4 | pkg/aipromptproposalhandler (`Get`, `List`, `Delete` in `main.go`; `Reject` and the uncovered part of `Accept` in `accept.go`), pkg/subscribehandler (`callmanager.go`, `conversationmanager.go`, `transcribemanager.go`, `pipecat_message.go`) | gomock; subscribe handlers launch work in goroutines, so expectations in new tests are synchronized with a channel, not a sleep (existing tests that sleep are left untouched) | ~100 |
| G5 | models/* pure functions (`IsValid`, `IsExpired`, `ConvertWebhookMessage`, `CreateWebhookEvent`), internal/config test setters (not `InitPrometheus`, to avoid duplicate registry registration), cmd/ai-control constructors | direct calls with synthetic values; the `cmd*` constructors add little, most of the gain comes from models and pure helpers | ~100 (models ~80, config `Set*ForTest` setters ~10, the `cmd*` constructors contribute almost nothing) |
| G6 (reserve) | pkg/aicallhandler (`tool.go`, `tool_describeaction.go`), pkg/aiaudithandler, pkg/geminiaudithandler | only if the total is still under 81% | reserve |

### 2.2 Test rules

- Every test asserts an observable result (returned value, error identity, stored row, written cache key, RPC response status and body, the exact arguments of a mock call). No bare call tests and no tests that build a value and compare it to itself.
- Priority inside each file: error paths, boundary values (empty id, nil uuid, zero size, expired state), validation and permission branches, then the success path.
- No `t.Skip` in new tests: a skipped test does not count toward coverage and hides unreachable code.
- Isolation: dbhandler tests share one in-memory sqlite database, so new dbhandler tests never use `t.Parallel()` and call `resetTestDB` on their first line; no test reads the wall clock for an expectation; Redis state is per test; no network, no external service, no random data without a fixed seed.
- Naming: `Test_<Function>_<case>`; gomock controller named `mc`; test files are named after the source file (`<source>_test.go`, or `<source>_coverage_test.go` where the source's test file already exists and would otherwise grow past reason).
- Synthetic values only (fixed uuids, `example.com` style hosts, fake keys assigned in separate statements, no `user:pass@host` strings).

### 2.3 Verification (verify items, no test code in this stage)

1. After each group, `go test -v` output is checked for SKIP lines in the new tests, and the PR table also lists per-file uncovered counts before and after (for example `aipromptproposal.go`), so a group that cannot reach a function is noticed early; the PR table states that the PR adds zero skipped tests. After each group: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=... ./...` and the hand-written total from the same script as the baseline; the per-package before and after table is built from it.
2. Full verification workflow of the repository (`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run -v --timeout 5m`) with `go.mod` and `go.sum` unchanged.
3. Flake check on the new tests: `go test -race -count=5 -shuffle=on` on the touched packages.
4. Mutation sample: mutations are chosen only inside code the new tests reach (not the `FOR UPDATE` functions), the group G4 mutations include a changed argument of a goroutine call, the G2 mutations include a removed cache write and a changed TTL, and the G1 mutations include an inverted soft-delete condition. In a scratch copy (not the worktree), apply at least six small product-code mutations spread over the groups (inverted condition, dropped error return, wrong status code, removed cache write, off-by-one on a boundary, swapped argument) and confirm first that the unmutated copy passes, then that at least one new test fails for each mutation; a table of mutation, group and failing test is recorded in the PR description.
5. `git diff --stat origin/main...HEAD` shows only files under `bin-ai-manager/`, no non-test `.go` file, no `go.mod` or `go.sum` change; `scripts/check-test-conventions.sh` passes; `git diff origin/main...HEAD` is grepped for email addresses, IP addresses, tokens and `user:pass@` patterns.
6. Defects noticed while writing tests are not fixed; a neutral comment is placed on tests that pin such behavior and the finding is reported in the tracker only.

### 2.4 Scope actually executed

The goal was reached with part of G1 (`aipromptproposal.go`, `team.go`, `mcpoauthstate.go`, `summary.go` delete, `AIAuditDelete`), G2, and the error paths of G3. `ai.go`, the remaining `aiaudit.go` functions, the success paths of G3, G4, G5 and the G6 reserve were not started, as the stop rule in 2.1 allows.
