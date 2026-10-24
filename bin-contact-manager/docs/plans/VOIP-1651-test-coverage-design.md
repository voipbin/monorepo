# VOIP-1651 bin-contact-manager test coverage

## Goal
Target: 2721 of 3831 statements or more (0.71 x 3831 = 2720.01), that is +156 statements over 2565. Hand-written coverage (excluding `gens/`, `mock_*.go`, `Code generated ... DO NOT EDIT` files, same-block duplicates merged by max count) from 66.95% (2565/3831) to at least 71%. Only `_test.go` files change.

## Baseline by package (hand-written, before)
| package | before | uncovered stmts |
|---|---|---|
| cmd/* (3 mains) | 0.0 | 489 |
| pkg/listenhandler | 69.1 | 365 |
| pkg/dbhandler | 75.4 | 352 |
| pkg/casehandler | 85.3 | 32 |
| pkg/contacthandler | 93.3 | 20 |
| config, models, cachehandler, subscribehandler, addresshandler | 0 to 100 | 8 (addresshandler 3) |

`cmd/*` start paths need real RabbitMQ, MySQL and Redis, so they are excluded from the plan (pure helpers there are optional, only if margin is short). Outside `cmd/*`, about 76.8% ((3342-777)/3342) is already covered, and listenhandler alone has about 315 reachable uncovered statements, which is about twice the 156 needed, so 71% is reachable without them. Aim for about 200 new statements in listenhandler as margin.

The uncovered listenhandler statements are a mix of error paths, unreached routes and success paths (all of `v1_interactions.go`, the `Assign` handler, `processRequest` routes). About 50 statements are `json.Marshal` failure branches that cannot be triggered with normal domain types and are not targeted. The current status code mapping of each handler is asserted as it is.

## Plan
Test rules checked by `scripts/check-test-conventions.sh` on added lines: `Test_` prefixed names, `reflect.DeepEqual` with `t.Errorf` (no testify), gomock controller variable named `mc`.

1. `pkg/listenhandler` (mock based, same harness as `v1_cases_test.go`): error and boundary paths of handlers that only have the happy path tested.
   - `v1_cases.go`: malformed id (400), malformed body (400), missing `customer_id` (400), downstream error mapped by `errorResponse`, and the whole `Assign` handler.
   - `v1_interactions.go`: `customer_id` missing, body absent, malformed JSON, non string `customer_id`; filter count 0, 1, 2 and 3, downstream error, success.
   - `v1_contacts_addresses.go` (`processV1ContactsAddressesGet`, `processV1ContactsAddressesIDDelete`, Post error branches), `v1_contact_addresses.go`: short URI (400), malformed body, downstream error, success. Interactions success asserts the argument order passed to `InteractionList` (customer id, page size, token, peer type, peer target, contact id, address id), with one success case for each of the three filter modes (peer, contact_id, address_id); malformed identifiers and partial peer filters assert the current behavior. Input validation boundaries (filter count 0, 1, 2, 3, unparsable URI, non numeric page size, malformed identifiers) assert the current response of each handler; exact expectations live in the test code.
   - `main.go` `processRequest` error conversion: a mock error returned through a route that propagates handler errors is converted to a mapped response for a `VoipbinError`, a mapped response for `dbhandler.ErrNotFound`, and 400 for any other error.
   - `main.go` `processRequest`: a routing table test that calls `processRequest` and asserts per route which mock handler method was invoked (each URI and method reaches the matching mock method), plus method mismatch and unknown route 404. The table test confirms that a representative URI for each route reaches the intended handler. Each case expects one distinct mock method and strict gomock rejects any other call, so a mis-wired case fails.
2. `pkg/dbhandler` (conditional default reserve, started when the interim measurement after step 1 is below 72%; measured after each step): `kase.go` transaction variants and error branches through the existing go-sqlmock helper pattern (`BeginTx`, `CaseInsertTx`, `CaseGetByIDTx`, `CaseListAll`, `CaseClearContactIDTx`, `CaseUpdateContactIDTx`). Further candidates by uncovered size: `address_ownership_write.go` (98), `address.go` (79), `address_ownership.go` (44), `contact.go` (43).
3. `pkg/addresshandler` and `casehandler.CaseListAll`: delegation tests only when they assert argument pass-through and error propagation.

## Verify
- Every new test asserts a concrete status code, body, mock call arguments or error text. No bare calls, no self comparison.
- gomock strict expectations: for validation failures no downstream call is expected, so reaching it fails the test.
- Mutation sample in a scratch copy (original worktree verified unchanged by `git diff`): 3 to 5 guards, including one filter count validation check, one customer id nil check, one error response mapping, one `processRequest` route mis-wiring (one case calling the wrong handler) and, only when plan step 2 is executed, one dbhandler Tx error propagation (otherwise replaced by another listenhandler mutation so that 3 to 5 are always run). Only the count (N of N detected) is reported, not the guard locations.
- Every error path test asserts the status code and that the downstream mock is not called. Each dbhandler sqlmock test asserts `ExpectationsWereMet`.
- Measurement command (run in a scratch copy only, since `-mod=mod` may touch go.mod and go.sum): `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` then the hand-written filter (drop `/gens/`, `mock_*`, `Code generated` + `DO NOT EDIT` header files, merge duplicate blocks by max count), first reproduced as 2565/3831 before any change.
- Full workflow: `go mod tidy` (no go.mod or go.sum diff expected), `go mod vendor`, `go generate ./...`, `go test -race -count=1 ./...`, `golangci-lint run -v --timeout 5m`, `git fetch origin main` then `bash scripts/check-test-conventions.sh` (run from the repository root), re-measure with the same method. Target 71% or more, interim check after each plan step.
- `git diff --stat origin/main...HEAD` lists only `bin-contact-manager/` files, and `git diff --name-only origin/main...HEAD | grep '\.go$' | grep -v '_test.go$'` is empty (this design document is the only non Go file).
- Test values are synthetic. No network, no external cost.

## Out of scope
Product code behavior changes, `cmd/*` mains, other packages below 80% coverage beyond the files above.

## Result
Plan step 1 alone raised the hand-written total from 66.95% to 74.16% (2841/3831), so the conditional dbhandler step and the optional step 3 were not executed.
