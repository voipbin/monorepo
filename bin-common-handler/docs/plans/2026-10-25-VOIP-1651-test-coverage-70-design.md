# VOIP-1651: raise bin-common-handler unit test coverage to 70%

## 1. Ticket review (adversarial)

Verdict: partially valid.

- Valid: measured on origin/main 99b59c386 with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` the hand-written total is 65.36% (10276 statements, 3560 uncovered), raw total 46.55%, 0 test failures. The numbers in the ticket reproduce.
- Incorrect premise: the card states that the `main` under `cmd/` is included in the hand-written total. bin-common-handler is a library and has no `cmd/` directory, so no `main` needs to be compensated for.
- Not a duplicate: no open or merged PR matches VOIP-1651 for this service (`gh pr list --state all --search VOIP-1651` is empty). The recent history of the path is lint cleanup (VOIP-1642) and feature wrappers, none of which add coverage.
- Scope check: only `_test.go` files (and go.mod/go.sum if a test-only module is needed) are changed. No product code is touched.

## 2. Issue analysis

Hand-written baseline (generated code, `mock_*.go`, `gens/` and `Code generated ... DO NOT EDIT` files excluded, duplicate blocks merged by max count):

| package | hand % | statements | uncovered |
|---|---|---|---|
| pkg/requesthandler | 63.4 | 8242 | 3014 |
| pkg/rabbitmqhandler | 47.4 | 517 | 272 |
| pkg/databasehandler | 68.7 | 402 | 126 |
| pkg/migrationhandler | 35.5 | 155 | 100 |
| pkg/notifyhandler | 93.4 | 212 | 14 |
| pkg/utilhandler | 92.0 | 162 | 13 |
| pkg/migrationhandler/migrationcheck | 97.2 | 287 | 8 |
| pkg/circuitbreakerhandler | 94.7 | 152 | 8 |
| pkg/sockhandler | 0.0 | 3 | 3 |
| models/errors | 95.8 | 48 | 2 |
| total | 65.36 | 10276 | 3560 |

The acceptance floor is 70% (+478 statements); 71% is the recommended margin for CI rounding. The numbers were reproduced again at the plan commit (only a docs file differs from 99b59c386). Needed: 71% means 7296 covered statements, i.e. about +580 over the 6716 covered today.

Where the gap is:

- `pkg/requesthandler` is 85% of the gap. It is a set of typed RPC client wrappers (`<Svc>V1<Resource><Action>`): build a URI, marshal a body, call `sendRequest<Svc>`, `parseResponse`. Every wrapper is testable with the existing `sockhandler.MockSockHandler`. Largest whole files with 0% coverage (top 17 of 24 such files, 1364 statements in all): ai_mcpservers (127 uncovered), rag_rags (108), tts_speakings (108), contact_contact_addresses (104), contact_contacts (89), ai_teams (87), webchat_widget (87), direct_directs (86), contact_addresses (67), webchat_session (67), route_providercalls (62), timeline_analyses (56), webchat_message (56), timeline_sip (47), agent_password (38), contact_tags (31), contact_interactions (30). The top 17 together are about 1250 statements.
- `pkg/databasehandler/mapping.go` (102 uncovered) holds pure reflection and conversion helpers (`convertValueForDB`, `copy*` scan helpers, `prepareFieldsFromMap`); they have branches for error input that are not exercised.
- `pkg/migrationhandler/command.go` (cobra subcommands, 28 uncovered) takes an `open func() (*Migrator, error)` and can be driven without a database for the error path.

Hard to test or lower priority:

- `pkg/rabbitmqhandler` (272 uncovered): the package has internal `amqpChannel` and `amqpConnection` interfaces, hand-written test doubles (`mockChannel`, `mockConnection` in `main_test.go`) and an injectable `amqpDial` variable, so publish, queue declare and consume paths can be exercised without a network. Only a successful dial against a real broker is out of reach. It is a secondary candidate if the requesthandler files are not enough.
- `pkg/migrationhandler/migrator.go` (66 uncovered): `New` opens a real MySQL connection, so it is excluded; the thin wrappers after `New` are not worth a mock without product code changes.
- `pkg/sockhandler` (3 statements): `NewSockHandler` only selects a type and `NewRabbit` does not dial, so both branches (RabbitMQ type and the default nil result) are testable at almost no cost and are included.

Arithmetic feasibility: the 24 fully uncovered requesthandler files hold 1364 statements (top 17: about 1250). Covering the success and error-propagation paths of each wrapper reaches roughly 90% of them (about 1230), which is well above the +580 needed (the top 17 alone would need about 46%, which is borderline, so the full set and the margin files below are used). Partially covered files give extra margin if needed: customer_customer (128 uncovered), billing_accounts (86), call_calls (74), agent_agents (67), then databasehandler/mapping.go (102) and rabbitmqhandler.

Overlap with VOIP-1416 (generic raise of packages below 80%): requesthandler, rabbitmqhandler, databasehandler, migrationhandler are below 80% and overlap; this PR does not absorb VOIP-1416.

## 3. Design

Scope: new `_test.go` files in `pkg/requesthandler` first, then (only if needed to clear 71%) `pkg/databasehandler`, `pkg/sockhandler`, `pkg/migrationhandler`, `pkg/rabbitmqhandler`. No product code, signature, or visibility change. No network, no real broker, no database.

### 3.1 Shared test helper (inside a `_test.go` file)

One table-driven runner in `pkg/requesthandler/rpc_table_test.go`. Before writing it, grep the package test files for helper and identifier names (for example `testTime`) so nothing collides.

- A case holds: wrapper name; the expected queue as a plain string (a constant such as `string(outline.QueueNameAIRequest)`, or a computed per-pod queue for the wrappers that build the queue from a pod id, such as `TTSV1SpeakingSay`, `Flush`, `Stop`); the exact expected `sock.Request` (URI, Method, DataType, Data); a canned success response; a call closure normalized to `func(h *requestHandler) (any, error)`; and the expected result.
- Return shapes differ, so the closure adapts them: wrappers returning only `error` (`AgentV1PasswordForgot`, `AgentV1PasswordReset`) return `(nil, err)`; multi-value wrappers (`AIV1McpOAuthStart` returns `(string, string, error)`, `AIV1McpOAuthCallback` returns `(bool, error)`) return a small struct or slice of the values, and the expected result is the same literal shape. On failure the expected result is spelled out per case (nil pointer, nil slice, empty string, zero value), not "nil or zero".
- Subtests per case:
  1. success: `MockSockHandler.RequestPublish(gomock.Any(), queue, wantRequest)` returns 200 with the body; assert no error and `reflect.DeepEqual` against the hand-written expected result. Response JSON and expected struct are written independently by hand.
  2. transport error: the mock returns an error; assert `err != nil`, the error is not `ErrNotFound` (so the case is distinguishable from the status case), and the stated empty result.
  3. non-2xx status: the mock returns a 404 with no typed-error DataType; for wrappers that use `parseResponse` assert `errors.Is(err, ErrNotFound)` and the stated empty result. Each case carries a status-check mode: `sentinel` (default, `errors.Is(err, ErrNotFound)`) or `custom` (non-nil error that is not `ErrNotFound` and whose message contains the status code). This is the check that each wrapper actually calls `parseResponse` and propagates its error. The typed `VoipbinError` branch and the unknown status branch of `parseResponse` belong to `common.go`, which already has its own tests (`common_test.go`), so they are not repeated per wrapper.
- Exceptions that do not use `parseResponse` (confirmed in the design stage by running `rg 'StatusCode|parseResponse'` over all 24 fully uncovered files, 90 wrappers; only these two exist; the wave 2 files are re-checked the same way in the coding stage): `TimelineV1SIPPcapGet` (returns `fmt.Errorf` when `StatusCode >= 300`, decodes a `{"data":"<base64>"}` body, and has separate JSON and base64 failure branches, each pinned by its own test) and `RagV1RagDelete` (treats only 200 as success; a 404 and a 2xx other than 200 both return an error that is not `ErrNotFound`, each pinned). Both use the `custom` mode or dedicated tests.
- Marshal-failure branch: only wrappers that marshal an argument of type `map[...]any` or `any` can reach the early `json.Marshal` error return (for example the `filters`-taking list wrappers, `RagV1RagGets`, `RagV1RagUpdate`, `AIV1McpServerList`; the exact list is fixed in the coding stage with `rg`, including the argument type of `AIV1McpServerCreate` and `Update`). Each such wrapper gets one case that passes a channel value inside the map, asserts an error, the stated empty result, and that `RequestPublish` is never called (no `EXPECT`). Wrappers whose marshaled arguments have fixed types (`uuid.UUID`, `string`, `int`, slices of these, as in `RagV1RagQuery`, `RagV1RagCreate`, `TTSV1SpeakingCreate`) cannot fail to marshal, so that branch is unreachable and is left out of the coverage plan.
- Literal expectations: the queue name (including the per-pod queue of `TTSV1SpeakingSay`, `Flush`, `Stop`) is written as a hand-typed string literal after checking the queue constant value, never rebuilt with the same `fmt.Sprintf` expression as the product code. `RagV1RagDelete` sends `ContentTypeJSON` with nil data; the expected request pins exactly that. At the start of the coding stage, `ls` confirms that no wave 1 target already has a `_test.go` file (contact_* and timeline_* have other test files with different names).
- Matching: `sock.Request` is compared exactly through gomock (`Publisher` is empty because the handler is built without a publisher). `Data` is a `json.RawMessage` written as a hand-typed literal; map-backed bodies (filters, field maps) rely on `encoding/json` sorting map keys, so literals use sorted keys, and struct-backed bodies follow the struct field order. `AIV1McpOAuthStart` pins the body for `mcpServerID == nil` against the real struct tag behavior read from the code, and `AIV1McpOAuthCallback` uses a `state` value that needs query escaping.
- The timeout argument and the context are matched with `gomock.Any()` (as in all existing tests), so timeout constants are not verified; delay wrappers are out of scope (they already have dedicated tests), so every new case uses the direct publish path. `initPrometheus` is already called in `main_test.go`, so new files never call it.
- `cb` is nil in these tests, exactly as in the existing tests (`requestHandler{sock: mockSock}`); `sendRequest` only touches the circuit breaker when it is set, so the transport case is the plain wrapped publish error.
- The gomock controller variable is named `mc`, test names start with `Test_`, and no testify is used (`reflect.DeepEqual` only), which are the three rules of `scripts/check-test-conventions.sh`.
- If a test exposes a real defect, the case is not skipped (`t.Skip` is not used) and the product code is not changed: the observed behavior is pinned with a neutral comment and the defect is reported in the card comment only.
- Same-package (internal) tests are used for all packages, as the existing tests do.

### 3.2 Case files (one per source file, named after the source file)

Wave 1, fully uncovered requesthandler files, largest first: ai_mcpservers, rag_rags, tts_speakings, contact_contact_addresses, contact_contacts, ai_teams, webchat_widget, direct_directs, contact_addresses, webchat_session, route_providercalls, timeline_analyses, webchat_message, timeline_sip, agent_password, contact_tags, contact_interactions, then the remaining small 0% files (ai_ai_prompt_histories, ai_participants, storage_compress, ai_analysis, timeline_aggregated_events, call_channels, flow_action).

Naming: wave 1 files have no existing test file, so they are `<source>_test.go` with `Test_<Wrapper>` names. Wave 2 targets already have a `<source>_test.go`, so they use `<source>_rpc_test.go` with `Test_RPC_<Wrapper>` names so nothing collides; compilation of the package is the collision check.

Each wrapper gets its exact URI (including query escaping and pagination), method, data type, body JSON and a representative response; wrappers with pointer, optional, or empty-string arguments get one case per branch (for example `mcpServerID == nil` in `AIV1McpOAuthStart`, `reanalyze` true and false, uuid.Nil arguments).

Wave 2 (only if the total is below 71% after wave 1): partial files customer_customer, billing_accounts, call_calls, agent_agents; `databasehandler/mapping.go` conversion and scan error branches (`convertValueForDB` per conversion type, `prepareFieldsFromMap` invalid input, `copy*` with malformed values); `sockhandler` both `NewSockHandler` branches; `migrationhandler/command.go` error paths with an injected `open` function; `rabbitmqhandler` publish and queue paths with the existing test doubles.

### 3.3 Verify items (written now, executed in the coding stage)

1. Measurement is reproducible: the hand-written coverage script and the exact command of section 2 (`GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`, generated code, mocks and DO NOT EDIT files excluded, duplicate blocks merged by max count) are kept in the card scratch directory and rerun unchanged. Hand-written total must be at least 71.0%. The per-package before/after table goes into the PR body with the raw total and the exclusion method.
2. Intermediate gate: after wave 1 the hand-written total is measured. Wave 2 is started only if it is below 71%; the expected gain from wave 1 is about 900 to 1200 statements, which is above the +580 needed, and the wave 1 result is recorded in a card comment.
3. Additions only: `git diff --numstat origin/main...HEAD` shows 0 deleted lines for every Go file, and `git diff --name-status` shows only new `_test.go` files plus this document; go.mod and go.sum are unchanged after `go mod tidy`.
4. Quality gates: from `bin-common-handler/`: `go test -count=1 ./...`, `go test -race -count=1 ./pkg/requesthandler/...`, `go vet ./...`, `golangci-lint run -v --timeout 5m`; from the repository root, after `git fetch origin main` (the script needs the merge base): `bash scripts/check-test-conventions.sh`. All must pass. The `go test` time of `pkg/requesthandler` is recorded before and after.
5. Mutation sample, chosen from wave 1 files only (timeout and delay constants are not mutation targets, see 3.1), in a scratch copy only: at least eight mutations across different files, including a URI segment change, a method swap, a dropped body field, a queue change, a `ContentTypeNone` versus `ContentTypeJSON` swap, removing the `parseResponse` call from a wrapper, and ignoring the `sendRequest` error. Each must make at least one new test fail; the results go to the card.
6. Meaningfulness is checked by the reviewers on the diff: expected requests are literal strings, response JSON and expected structs are written separately by hand, and every wrapper test has the success, transport error and status-error cases (sentinel or custom mode as read from the wrapper).
7. Public repository hygiene: only synthetic values (reserved example domains, `203.0.113.x`, `+1555` style numbers, clearly fake secrets like `test-secret-not-real`, newly invented uuids). A grep over the added lines for `AKIA|sk-|BEGIN .*KEY|eyJ|://[^ ]*:[^ ]*@` must return nothing, secret-like literals are assigned in their own statement and never sit in the same literal as an address, and the GitGuardian check must be green.
8. `git diff --stat origin/main...HEAD` lists only paths under `bin-common-handler/`.
9. Defects found while testing go to the card comment only and are not described in repo files.

### 3.4 Risks

- Large test count slows `go test`: the runner uses mocks only; the measured time is recorded (verify item 4).
- Duplicate test function names: new names use the `Test_RPC_<Wrapper>` form (3.2), checked by compiling the package.
- Coverage drift versus CI: target 71% leaves margin over the 70% floor.
