# VOIP-1652: bin-contact-manager unit test coverage to 75%

## 1. Ticket review (analysis)

Baseline (origin/main accc03e, `GOFLAGS=-mod=mod go test -count=1 -coverprofile`): hand-written 2837/3831 = 74.05%, raw 64.0%. Generated code is excluded by path (`/gens/`, `mock_*.go`) and by the `Code generated ... DO NOT EDIT` header; duplicate blocks are merged with the maximum count. The ticket figures reproduce exactly. PR #1571 (the VOIP-1651 coverage work, a separate earlier ticket) is already merged and included in the baseline; no open PR touches this service.

Hand-written statements not covered (994 in total): cmd/ 489 (contact-control 325, contact-manager 148, case-control 16), pkg/dbhandler 352, pkg/listenhandler 93, pkg/casehandler 32, pkg/contacthandler 20, others 8 (addresshandler 3, config 2, models/contact 1, cachehandler 1, subscribehandler 1). The core paths of cmd/ need live database and cache connections and cannot be unit tested without a refactor, which is out of scope; its pure helpers could be tested but are not planned.

Reaching 76% needs at least 75 more covered statements (0.76 x 3831 - 2837 = 74.56); 75% needs 37. The target is 90 or more covered statements to leave margin against CI rounding. Without cmd/ there are about 477 candidates in dbhandler, listenhandler and casehandler.

## 2. Plan

Tests only, inside bin-contact-manager. No non-test file changes, no new modules (sqlmock and sqlite are already test dependencies of this service).

1. dbhandler, sqlite in-memory harness: `CaseInsertTx`, `CaseGetByIDTx`, `CaseUpdateContactIDTx`, `CaseClearContactIDTx` with commit and rollback (rollback must leave the row unchanged), `CaseListAll` across tenants, `CaseGetByID` not found, `AddressListAllByContactID` (including rows of another contact), `AddressList` filters (contact_id, type, page token).
2. dbhandler, sqlmock harness (pattern of `ctx_rows_test.go`): begin and commit failures of the Address write paths, `BeginTx` failure, exec and query failures of the Case update and list functions, deadlock mapping to `ErrDeadlock` in `CaseGetOpenByPeer` and `CaseUpdateStatusClosed`, rows-affected failure, invalid page token in `CaseList`.
3. casehandler: `CaseListAll` delegation and error propagation, `Close` and `Continue` error branches (begin, commit, carry-over failure, rollback on error).
4. The named functions in 1 to 3 add up to roughly 77 uncovered statements, close to the minimum, so the address write paths (stale-row repair, AddressUpdateTx, ownership resolution) and the listenhandler and contacthandler error branches from the profile are treated as required, not optional, to reach the 90 statement target.

Test values are synthetic. No network, no external cost.

## 3. Verify

- Hand-written total at least 76% with the same measurement; per-package before and after table in the PR body.
- The PR body records the measurement and filter method so the figure can be reproduced.
- Existing tests pass, `golangci-lint run` clean, `git diff origin/main...HEAD --stat` limited to bin-contact-manager test files and this document, no go.mod or go.sum change.
- Mutation sample: break a rollback, an error wrap, and the deadlock mapping in a scratch copy and confirm the new tests fail.
- Tests assert behavior (persisted state after commit and rollback, sentinel error mapping, error wrapping), not call counts alone.
