# VOIP-1652: bin-contact-manager unit test coverage to 75%

## 1. Ticket review (analysis)

Baseline (origin/main accc03e, `GOFLAGS=-mod=mod go test -count=1 -coverprofile`): hand-written 2837/3831 = 74.05%, raw 64.0%. Generated code is excluded by path (`/gens/`, `mock_*.go`) and by the `Code generated ... DO NOT EDIT` header; duplicate blocks are merged with the maximum count. The ticket figures reproduce exactly. PR #1571 (the VOIP-1651 coverage work, a separate earlier ticket) is already merged and included in the baseline; no open PR touches this service.

Hand-written statements not covered (994 in total): cmd/ 489 (contact-control 325, contact-manager 148, case-control 16), pkg/dbhandler 352, pkg/listenhandler 93, pkg/casehandler 32, pkg/contacthandler 20, others 8 (addresshandler 3, config 2, models/contact 1, cachehandler 1, subscribehandler 1). The core paths of cmd/ need live database and cache connections and cannot be unit tested without a refactor, which is out of scope; its pure helpers could be tested but are not planned.

Reaching 76% needs at least 75 more covered statements (0.76 x 3831 - 2837 = 74.56); 75% needs 37. The target is 90 or more covered statements to leave margin against CI rounding. Without cmd/ there are about 477 candidates in dbhandler, listenhandler and casehandler.

## 2. Plan

Tests only, inside bin-contact-manager. No non-test file changes, no new modules (sqlmock and sqlite are already test dependencies of this service). Candidates and reachable uncovered statements, measured from the profile (upper bounds):

| Group | Functions | Statements |
|---|---|---|
| A. dbhandler, sqlite harness (normal paths) | `BeginTx` success 3, `CaseListAll` normal path about 15, `*Tx` wrappers 4 (`CaseInsertTx`, `CaseGetByIDTx`, `CaseUpdateContactIDTx`, `CaseClearContactIDTx`), `AddressListAllByContactID` normal path about 6, `AddressList` contact_id and type filters and page token 6 (the target filter is already covered) | about 34 |
| B. dbhandler, sqlmock (error paths) | `BeginTx` failure 1, `CaseListAll` query, scan and `rows.Err()` branches about 5, `AddressListAllByContactID` and `AddressList` scan and `rows.Err()` branches about 8, `caseInsertExec` 6, `caseGetByIDExec` 3, `CaseGetOpenByPeer` 4 (both deadlock points), `caseUpdateStatusClosedExec` 5, `caseUpdateContactIDExec` 2, `caseClearContactIDExec` 2, `CaseList` 6 | about 42 |
| C. casehandler | `CaseListAll` 1, `Close` 2, `Continue` 7 | 10 |
| D. required top-ups | `staleRowRepairTx` 18, `OwnershipPeriodsLockAndResolveTx` 14, `AddressUpdateTx` 13, `addressDeleteCompensatingAttempt` 11, listenhandler `v1_cases.go` 30, contacthandler `contact.go` 14 | up to 100 |

A to C total at most about 86 (about 79 realistically, since query-build errors are unreachable), below the 90 statement target, so group D is required. D is worked in order of gain per scenario; if the measured gain stays under 90 after D, the PR body states the achieved value and the reason instead of padding. Group D is reachable with sqlmock (injected exec and query errors, deadlock errno 1213, rows-affected 0) and with the existing handler mocks; each group D function is confirmed by running one scenario and checking in the coverage profile that its uncovered blocks shrink, and the PR table reports the measured gain per group.

Harness rules:
- The sqlite database has a single connection. A test that opens a `*sql.Tx` reads only through that tx and reads through `h.db` only after Commit or Rollback; `t.Cleanup` rolls back any tx still open so a failing test cannot hang later tests.
- The sqlite database is shared and tests run with `-count=2`. Tests generate ids per run (`uuid.NewV4`) or delete their rows in `t.Cleanup`, and assert on the rows they inserted (membership), never on total counts, because `CaseListAll` spans tenants.
- casehandler error branches use `dbhandler.MockDBHandler` returning a `*sql.Tx` obtained from sqlmock (`ExpectBegin`, then `ExpectCommit().WillReturnError` for the commit failure; no `ExpectRollback` there because `database/sql` marks the tx done after a failed Commit, `ExpectRollback` for the insert and carry-over failures). The Begin failure is injected by the mock returning an error.
- Test values are synthetic: reserved phone ranges (`+1555...`) and `example.com` addresses only. No network, no external cost.

## 3. Verify

- Floor: hand-written total at least 75.0% (the ticket requirement). Target: at least 76% and at least 90 newly covered statements. If the floor is missed the card is blocked; if only the target is missed the PR body states the achieved value and the reason. Both measured with `GOFLAGS=-mod=mod go test -count=1 -coverprofile` and the same filter (exclude `/gens/`, `mock_*.go`, `DO NOT EDIT` header files; merge duplicate blocks by maximum count). The PR body gives the command, the filter, the per-package before and after table, and the raw total (64.0% before) next to the hand-written total. The CI threshold calculation is not known, so the margin is kept.
- New tests pass with `-race`, `-count=2` and `-shuffle=on`.
- Existing tests pass, `golangci-lint run` clean, `git diff origin/main...HEAD --stat` limited to bin-contact-manager test files and this document, no go.mod or go.sum change. Service docs sync does not apply (no source change).
- Assertion standard: error tests use `errors.Is` against the injected error or the sentinel (`ErrDeadlock`, `ErrNotFound`), check the function-name wrapping text, and check the returned value is nil; sqlmock tests also call `ExpectationsWereMet` except in cancelled-context cases. State tests assert persisted rows after Commit and after Rollback. Call counts alone are not accepted.
- Mutation table, one row per feature group, run in a scratch copy and recorded in the PR. Each mutation must fail the intended test only:
  - A (sqlite): remove the `WHERE customer_id` condition of `CaseClearContactIDTx` (killed by a case that calls it with the same id and a different customer id and asserts the row is unchanged); drop the contact_id filter and the type filter of `AddressList` (each test inserts at least two addresses with different values so the filter is distinguishable).
  - B (sqlmock): remove the `rows.Err()` check of `CaseListAll` (killed by the sqlmock `RowError` test, not by sqlite); drop the deadlock mapping of `caseUpdateStatusClosedExec`.
  - C (casehandler): remove the rollback defer of `Continue` (killed by the insert-failure and carry-over-failure cases, not by the commit-failure case); remove the commit error propagation and the error propagation of `CaseUpdateStatusClosed` in `Close`.
  - D (top-ups): for the two largest functions, for example `staleRowRepairTx` (remove the rows-affected zero branch) and `OwnershipPeriodsLockAndResolveTx` (remove one error propagation).
  - Reserve candidates if D falls short of the target: `AddressUpdate` (14) and `ContactDeleteByCustomerID` (10).
- Public repository hygiene check on `git diff`: no real addresses, hosts, secrets or personal data.
