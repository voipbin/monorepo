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

One table-driven runner in `pkg/requesthandler/rpc_table_test.go`:

- A case holds: wrapper name, expected queue (`outline.QueueName...`), expected exact `sock.Request` (URI, Method, DataType, Data), a canned success response body, the call closure, and the independently written expected result.
- Subtests per case:
  1. success: `MockSockHandler.RequestPublish(gomock.Any(), queue, wantRequest)` returns 200 and the body; assert no error and `reflect.DeepEqual` of the hand-written expected result (built from literals, not from the response bytes).
  2. transport error: the mock returns an error; assert the wrapper returns a non-nil error and a nil or zero result.
  3. non-2xx status: the mock returns a 404 response; assert the wrapper returns a non-nil error (exercises `parseResponse` status mapping through the wrapper).
- Failure injection of the marshal step is not used (marshal of these structs cannot fail); wrappers that return an error for an invalid combination of arguments get an explicit case.
- The gomock controller variable is named `mc` and test function names start with `Test_` to satisfy `scripts/check-test-conventions.sh`.

### 3.2 Case files (one per source file, named after the source file)

Wave 1, fully uncovered requesthandler files, largest first: ai_mcpservers, rag_rags, tts_speakings, contact_contact_addresses, contact_contacts, ai_teams, webchat_widget, direct_directs, contact_addresses, webchat_session, route_providercalls, timeline_analyses, webchat_message, timeline_sip, agent_password, contact_tags, contact_interactions, then the remaining small 0% files (ai_ai_prompt_histories, ai_participants, storage_compress, ai_analysis, timeline_aggregated_events, call_channels, flow_action).

Each wrapper gets its exact URI (including query escaping and pagination), method, data type, body JSON and a representative response; wrappers with pointer, optional, or empty-string arguments get one case per branch (for example `mcpServerID == nil` in `AIV1McpOAuthStart`, `reanalyze` true and false, uuid.Nil arguments).

Wave 2 (only if the total is below 71% after wave 1): partial files customer_customer, billing_accounts, call_calls, agent_agents; `databasehandler/mapping.go` conversion and scan error branches (`convertValueForDB` per conversion type, `prepareFieldsFromMap` invalid input, `copy*` with malformed values); `sockhandler` both `NewSockHandler` branches; `migrationhandler/command.go` error paths with an injected `open` function; `rabbitmqhandler` publish and queue paths with the existing test doubles.

### 3.3 Verify items (written now, executed in the coding stage)

1. Hand-written total is at least 71.0% with the same measurement method as section 2; per-package before/after table goes into the PR body together with the raw total and the exclusion method.
2. `go test -count=1 ./...` passes with no existing test changed (`git diff --numstat` shows additions only in `_test.go`, plus docs).
3. `golangci-lint run -v --timeout 5m` and `scripts/check-test-conventions.sh` pass; `go mod tidy` leaves go.mod and go.sum unchanged.
4. Mutation sample (in a scratch copy, never in the worktree): at least five mutations across different files, such as changing a URI segment, swapping a method, dropping a body field, changing the queue, and ignoring an error from `sendRequest`; each must make at least one new test fail. Results go to the card.
5. Meaningfulness: no case compares a value built from the same expression as the wrapper output; expected requests are literal strings; every wrapper test covers success, transport error and non-2xx status.
6. Public repository hygiene: synthetic uuids and strings only; no password-like literals next to addresses; no host, IP, or customer data.
7. `git diff --stat origin/main...HEAD` lists only paths under `bin-common-handler/`.
8. Defects found while testing are reported in the card comment only and are not described in repo files.

### 3.4 Risks

- Large test count slows `go test`: the runner uses mocks only, so it adds well under a second.
- Duplicate test function names: each case file uses the wrapper name as the suffix, checked by compiling the package.
- Coverage drift versus CI: target 71% leaves margin over the 70% floor.
