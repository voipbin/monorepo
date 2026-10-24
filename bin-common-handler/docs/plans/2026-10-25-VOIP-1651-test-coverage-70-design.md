# VOIP-1651: raise bin-common-handler unit test coverage to 70%

## 1. Ticket review (adversarial)

Verdict: partially valid.

- Valid: measured on origin/main 99b59c386 with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` the hand-written total is 65.36% (10276 statements, 3560 uncovered), raw total 46.55%, 0 test failures. The numbers in the ticket reproduce.
- Incorrect premise: the ticket says `cmd/` main is part of the hand-written total. bin-common-handler is a library and has no `cmd/` directory, so no `main` needs to be compensated for.
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

Needed: 71% means 7296 covered statements, i.e. about +580 over the 6716 covered today.

Where the gap is:

- `pkg/requesthandler` is 85% of the gap. It is a set of typed RPC client wrappers (`<Svc>V1<Resource><Action>`): build a URI, marshal a body, call `sendRequest<Svc>`, `parseResponse`. Every wrapper is testable with the existing `sockhandler.MockSockHandler`. Whole files with 0% coverage: ai_mcpservers (127 uncovered), rag_rags (108), tts_speakings (108), contact_contact_addresses (104), contact_contacts (89), ai_teams (87), webchat_widget (87), direct_directs (86), contact_addresses (67), webchat_session (67), route_providercalls (62), timeline_analyses (56), webchat_message (56), timeline_sip (47), agent_password (38), contact_tags (31), contact_interactions (30). Together about 1250 statements.
- `pkg/databasehandler/mapping.go` (102 uncovered) holds pure reflection and conversion helpers (`convertValueForDB`, `copy*` scan helpers, `prepareFieldsFromMap`); they have branches for error input that are not exercised.
- `pkg/migrationhandler/command.go` (cobra subcommands, 28 uncovered) takes an `open func() (*Migrator, error)` and can be driven without a database for the error path.

Hard to test (excluded from the plan, listed so the exclusion is explicit):

- `pkg/rabbitmqhandler` (272 uncovered): connection, consume, publish, queue declare need a live broker; a network dependency, not allowed for unit tests.
- `pkg/migrationhandler/migrator.go` (66 uncovered): `New` and `Up/Down` open a real MySQL connection.
- `pkg/sockhandler` (3 statements): constructor wrapper around the broker.

Arithmetic feasibility: the fully uncovered requesthandler files alone provide about 1250 coverable statements, so reaching 71% (+580) needs under half of them. The goal is achievable without the hard-to-test packages.

Overlap with VOIP-1416 (generic raise of packages below 80%): requesthandler, rabbitmqhandler, databasehandler, migrationhandler are below 80% and overlap; this PR does not absorb VOIP-1416.
