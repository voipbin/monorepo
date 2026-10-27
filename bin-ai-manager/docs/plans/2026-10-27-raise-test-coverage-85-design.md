# bin-ai-manager: raise hand-written unit test coverage to 85%

## 1. Ticket review

- Baseline re-measured on origin/main a47f5a2a1 (same command as the ticket): hand-written 81.907% (13029 of 15907 statements), raw 75.0%, zero failing tests. The number matches the ticket.
- Needed: 492 more covered statements for 85.0%, 571 for 85.5%. The target is 85.5% or more (about 600 statements) to leave margin for CI rounding.
- No open PR touches the service. The only PR the ticket named as conflicting (VOIP-1559, #1642) is already merged and is part of the baseline.
- Measurement: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in `bin-ai-manager/`. Hand-written excludes `gens/`, `mock_*.go` and files with a `Code generated ... DO NOT EDIT` header; duplicate blocks keep the maximum count.

## 2. Reachable ceiling

Uncovered statements by package: cmd/ai-control 546, aicallhandler 443, dbhandler 354, cmd/ai-manager 246, engine_openai_handler 229, listenhandler 186, aipromptproposalhandler 90, subscribehandler 87, summaryhandler 80, engine_dialogflow_handler 71, aiaudithandler 66, aihandler 50, mcpserverhandler 45, others below 45 each.

Treated as unreachable without a signature change: all of `cmd/ai-manager` (needs live connections), the `run*`, `init*Handler` and `main` of `cmd/ai-control`, the dialogflow client calls, the three dbhandler functions that use `FOR UPDATE` or `ON DUPLICATE KEY` (the sqlite harness cannot run them), the external LLM `Evaluate` calls, constructors that open external connections, and `json.Marshal` failure branches. These total about 800 statements. Of the remaining about 2080, a further 30% is discounted for branches that turn out to be always-nil or constant based. The resulting ceiling is about 91%, far above the 85.5% target, so the work proceeds without escalation.

## 3. Constraints

- Test files only, inside `bin-ai-manager/`. No product code or signature change and no `go.sum` change. `go.mod` may change only by the `// indirect` marker of a monorepo module that a test imports directly (see the scope section).
- Tests do not depend on wall clock, ordering, shared state or the network. dbhandler tests use the existing sqlite harness (`resetTestDB`, no `t.Parallel()`); Redis state is per-test miniredis; the OpenAI client tests use a loopback `httptest` server only; goroutine results are awaited on channels with a context timeout, never a sleep.
- Global state: `initCommand` calls `config.Bootstrap` and `PersistentPreRunE` uses the global viper instance. cmd tests never call `Execute`, so `PersistentPreRunE`, `config.LoadGlobalConfig` and `ConfigureLocalModels` are not reached. The only global state touched is the global viper instance (`viper.Set`, `AutomaticEnv` from `Bootstrap`) and the logger level and formatter set by `Bootstrap`; tests restore the logger level and formatter first and then call `viper.Reset()` in `t.Cleanup`, use `t.Setenv` for any environment, and do not use `t.Parallel()`.
- Detached goroutines (`aiaudithandler.Create` and `aipromptproposalhandler.Create` start a background job; the subscribehandler methods start `go h.aicallHandler.EventCM*`, `go h.processEvent(m)` and similar, and `EventCMCallHangup` starts two): the mock that the job calls last closes a done channel from `DoAndReturn`; the test waits on it with a `select` and a context timeout, and registers a `t.Cleanup` wait before the gomock controller finishes, so no mock call can happen after `Finish`. When a handler starts several goroutines, there is one done channel per goroutine. Job paths that call an external LLM `Evaluate` are outside the target (see section 2) and are not counted in the G4 estimate; the G4 estimate (~170) already excludes them.
- Streaming tests: the SSE handler writes its events, flushes and returns; `t.Cleanup` closes the server; every request context has a timeout; tests drain both output channels until they close (the handler's sends block and ignore the context, so an undrained channel would leak the goroutine) and assert they are closed after the stream ends; the context-cancel test runs against a server that withholds its response.
- Conventions (`scripts/check-test-conventions.sh`): `Test_` prefix, gomock controller named `mc`.
- Package overlap with the general lift of packages under 80% (VOIP-1416) is noted in the PR body only.

## 4. Design

The gains in the table assume about 70% of each group's uncovered statements are reached (the sum of the groups' uncovered statements is about 1100; 70% is about 800 against a need of 571 for 85.5%). G6 is therefore likely to be needed and is part of the plan, not a spare. Work proceeds in order and stops once the measured total reaches 85.6% with all tests green. A later group is skipped if an earlier one already gives the margin.

| Group | Scope | Approach | Expected gain |
|---|---|---|---|
| G1 | engine_openai_handler: `Send`, `SendOnce`, `MessageSend`, `StreamingSend`, `streamingResponseHandle*`, `NewEngineOpenaiHandlerWithConfig` | handler built against a loopback `httptest` server; JSON and SSE bodies written by hand. `SendOnce` and `StreamingSend` get error-status paths; `Send` and `MessageSend` (which retry through a backoff that ignores the context, up to a minute on a failing server) get success-path tests only, their error paths are left out. The stream tests assert the decoded text and, for tool calls, only the `ID` and `Type` that the handler fills; a missing field is reported on the card per verify item 6, not asserted | ~180 |
| G2 | cmd/ai-control: `initCommand`, `cmd*` constructors, `resolveUUID` | `Execute` is never called: the commands set no required-flag validation and `RunE` opens the database and Redis first, and `PersistentPreRunE` would run `LoadGlobalConfig` (a `sync.Once`) and mutate global model and logger state. Tests build the command tree (`initCommand()` and each `cmd*()`) and assert the subcommand names. The flag contract is tested through the keys the `run*` functions read: each `cmd*` gets `ParseFlags` with sample arguments, then `viper.BindPFlags(cmd.Flags())`, and the test asserts `viper.GetString` and `viper.GetInt` return the parsed values under the keys `run*` reads (`id`, `customer-id`, `limit`, ...); `resolveUUID` is called directly for empty, malformed and valid values. Commands that share keys are separate subtests with `viper.Reset()` between them. Only defaults that encode a contract are asserted, not product constants. Straight-line flag registration is counted as ~100 of the ~277. Lowest value per statement: done last among G1 to G5 and skipped if the margin is already reached | ~100 |
| G3 | aicallhandler: `tool.go` (including `toolHandleSearchKnowledge` through `resolveAI` and the RAG request, and invalid-argument paths), `tool_describeaction.go`, `listen*`, `event.go` (`EventCMConfbridgeJoined`, `EventCMConfbridgeLeaved`), `db.go` | gomock; table tests for invalid JSON arguments, dependency errors and success | ~170 |
| G4 | aipromptproposalhandler (`Get`, `List`, `Delete`, `Reject`, uncovered `Accept`), subscribehandler (`callmanager.go`, `conversationmanager.go`, `transcribemanager.go`, `pipecat_message.go`), summaryhandler, aiaudithandler (without the `runAuditJob` path that calls the external evaluator), aihandler, mcpserverhandler, mcptoolhandler, messagehandler | gomock; every handler that starts a goroutine is awaited as described in section 3 | ~170 |
| G5 | models pure functions (`IsValid`, `IsExpired`, `ConvertWebhookMessage`, `CreateWebhookEvent`, `ConvertResolvedAI`), internal/config `Set*ForTest` setters | direct calls with synthetic values, asserting the returned value or the stored config | ~70 |
| G6 (reserve) | dbhandler `ai.go`, `aiaudit.go`, `aicall.go`, `message.go`, `summary.go`, `mcpserver.go`, `aipromptproposal.go`, `team.go`, `mcpoauthstate.go`, except the three excluded functions | existing sqlite harness | ~120 |

### Test rules

- Every test asserts an observable result (returned value, error identity, stored row, written key, RPC status and body, exact arguments of a mock call). No bare call tests and no test that builds a value and compares it to itself.
- Priority: error paths, boundary values, validation and permission branches, then the success path.
- No `t.Skip`. Synthetic values only (fixed uuids, `example.com` hosts, fake keys assigned in a separate statement, no `user:pass@host` strings).
- Test files are named after the source file (`<source>_test.go`, or `<source>_coverage_test.go` when the existing file would grow past reason). The existing tests are not modified.

### Verify items

0. Final numbers are taken from the baseline command (`GOFLAGS=-mod=mod`); the vendor-based CI run is expected to agree within rounding and the PR states which was used. If the total is still under 85.0% after G6, the card is blocked with the remaining uncovered inventory instead of lowering the target.
1. After each group the measured total and per-package numbers are recomputed with the baseline command and script; the PR table lists them before and after and states the number of skipped tests added (zero).
2. Repository verification workflow (`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run -v --timeout 5m`) with `go.sum` unchanged and `go.mod` limited to that marker.
3. Flake check on the new tests: `go test -race -count=5 -shuffle=on` on the touched packages (this includes `cmd/ai-control`, to show its tests do not depend on order or on viper state), plus the same with `-cpu 1,4` for the OpenAI streaming tests.
4. Mutation sample on a scratch copy: at least eight small product-code mutations spread over G1 to G4 (inverted condition, dropped error return, wrong status or path, swapped argument, off-by-one); each must fail at least one new test. A table goes into the PR body.
5. `git diff --stat origin/main...HEAD` shows only files under `bin-ai-manager/`, no non-test `.go` file; `scripts/check-test-conventions.sh` passes; the diff is grepped for email addresses, IP addresses, tokens and `user:pass@` patterns.
6. Defects noticed while writing tests are not fixed; tests that pin such behavior carry a neutral comment and the finding goes into the card only.

### Scope actually executed

G1, G3, G4 and G5 were executed and the measured hand-written total reached 87.80% (13966 of 15907 statements, raw 80.4%) with the baseline command, above the 85.6% stop line. G2 and G6 were not started, as the stop rule allows. Beyond the plan, `aiaudithandler` and `aipromptproposalhandler` tests also reach the job paths through a mocked evaluator. `go.mod` changes by one line only (`bin-rag-manager`, already a local replace used by other services, moves from indirect to direct because a test imports its response type); `go.sum` is unchanged.
