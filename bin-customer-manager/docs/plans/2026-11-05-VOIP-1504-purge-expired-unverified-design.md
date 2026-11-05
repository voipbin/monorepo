# VOIP-1504: purge expired unverified customers after the 30-day retention

## 1. Ticket review (re-checked against origin/main)

Verdict: valid. Every claim in the ticket was re-read in code.

- `CleanupUnverified` (`cleanup.go`) only moves `initial` rows to `expired` after 72 hours. Nothing selects an `expired` row afterwards. `CleanupFrozenExpired` (`expiry.go`) selects `status='frozen'` only.
- `CustomerAnonymizePII` is guarded by `status='frozen' AND tm_delete IS NULL`, so it cannot write an `expired` row. `CustomerFreezeAllowUnverified` does not look at `email_verified`.
- `EmailVerify` (`signup.go`) turns an `expired` row into `active` and clears `tm_delete` (the recovery path). The purge guard repeats `email_verified=false`, so a customer that verified after the list query is not purged.
- `CustomerUpdate` always sets `tm_update`. For a row expired by `CleanupUnverified`, `tm_update` is the expiry time. Other updates move it later (the purge is only delayed).
- `customer_customers` has only `idx_customer_customers_status` (baseline). There is no `tm_update` index. A purge query filters on `status` first, and the number of `expired` rows is small, so no new index (and no migration) is needed.
- `EmailVerifyResend` does not write the customer row, so a resend does not move `tm_update`. A resend therefore does not extend the retention; only another update of the row postpones the purge.
- `customer-unverified-cleanup` (every 15 minutes) calls `POST /v1/customers/cleanup_unverified`. The schedule-manager seed migration only registers the URI; it does not read the response body.

Not refuted. Corrections: none of substance. Out of scope (confirmed): legacy expired rows that already have `tm_delete`, rows that are `deleted` without a replaced email.

## 2. Design

Goal: an `expired` customer with `email_verified=false` whose `tm_update` is older than 30 days is anonymized (`status=deleted`, `tm_delete` set) and `customer_deleted` is published once. The row is kept.

### 2-1. Decisions

1. Do not reuse `FreezeAndDelete`: it publishes `customer_frozen` and treats the anonymize `ErrNotFound` as an error.
2. A single UPDATE anonymizes the row, guarded by `status='expired' AND email_verified=false AND tm_delete IS NULL`. The guard is a compare-and-set: a row that was verified (or already claimed by the other replica) after the list query matches 0 rows and is skipped without an event.
3. Retention is the constant `expiredRetention = 30 * 24 * time.Hour`, not configurable. The candidate query is `status='expired' AND email_verified=false AND tm_delete IS NULL AND tm_update < now-30d`, ordered by `tm_update` ascending, limit 100 per run. At 15-minute runs this clears 9,600 rows a day, far above the backlog. Legacy rows (`tm_delete` set) are excluded by the query and by the guard. Known limit: a row whose anonymize fails persistently with an error other than `ErrNotFound` stays at the front of the ordered list and is retried every run; 100 such rows would block the rest. Failures are logged at error level (alertable), and no skip bookkeeping is added without such a signal. The guard does not re-check `tm_update`, so an unrelated update between list and write does not stop the purge; it only delays rows that were updated before the list.
4. Event: after a successful anonymize the event is published for every claimed row, with no `CustomerGet`. Reason: if `CustomerGet` fails after the anonymize, the row is already `deleted` and is never selected again, so the 13 `customer_deleted` subscribers would never run (the agent subscriber deletes the signup agent, which still carries the email address). Alternatives compared: (i) `CustomerGet` and skip on failure, as `CleanupFrozenExpired` does: the cascade is lost permanently for that row. (ii) `CustomerGet` with a single retry: the usual cause is a database outage that a retry seconds later rarely fixes. (iii) return the updated row from the dbhandler method: MySQL has no `UPDATE ... RETURNING`, so it needs the same read. (iv) `CustomerGet` with a synthesized fallback: two payload paths to build and test, and the fallback path runs only during a database fault. (v) always synthesize from the listed row: one path, no extra read, no loss. Decision: (v). The payload is a copy of the listed row with name and email replaced by the anonymized values, phone number, address and webhook URI cleared, `status=deleted`, and `tm_delete` and `tm_update` set to `utilHandler.TimeNow()` taken in the handler right after the anonymize (within milliseconds of the stored value). The 13 subscribers read only `ID` (see 2-2). The existing two paths are not touched. The test asserts the payload contains none of the original name, email, phone, address or webhook URI (distinct fake values in the listed row) and carries the anonymized values, `status=deleted`, the mocked `TimeNow` as `tm_delete` and `tm_update`, and the unchanged `ID` and `billing_account_id`. Scope note: only those five fields are anonymized, the same set as in `CleanupFrozenExpired`. `TimeNow` is mocked after the anonymize call (InOrder).
5. Schedule: no new job and no migration. `POST /v1/customers/cleanup_unverified` (every 15 minutes) runs the existing expire step and then the purge step, both always, and fails the request if either failed. The response gets an additional `purged` count. The seed row description still reads "Soft-delete unverified customers past the grace period"; the seed migration is immutable and is not changed; `docs/architecture.md` is updated in this change to describe both steps. If one step fails the request returns an error and the other step's count is not in the response, but its work is committed and both steps log their counts. A separate daily job would need a schedule-manager seed migration (strengthened schema verification, an extra deploy step) for no benefit.
6. `EmailVerify` is not changed by this card. Its interaction with the purge is recorded on the ticket, not in this public repository.
7. The retention period is also the period in which a user can recover with `POST /auth/email-verify-resend`. After it the account is anonymized, the address can sign up again (duplicate-email check no longer matches), and the old verification link is rejected (`deleted`).
8. Anonymized fields are exactly the existing set of `CleanupFrozenExpired` (name, email, phone number, address, webhook URI). The helper takes the same field map, so both guards share one definition.
9. The new constant `expiredRetention` (time since the expiry, measured by `tm_update`) is distinct from the existing `gracePeriod` (time since a freeze). Both are 30 days today; the code comment says they are independent.

### 2-2. Subscriber check (customer_deleted)

All 13 subscribers were read (handler function and its subscribe wrapper). None reads a customer field other than `ID`, so the synthesized event of decision 4 is sufficient with the ID and the anonymized values.

| Service | Action on `customer_deleted` | Unverified customer without resources |
|---|---|---|
| agent | list agents of the customer (limit 1000), delete each | deletes the signup agent (intended) |
| billing | list accounts of the customer, delete each | deletes the account created for the customer |
| call | call, groupcall and confbridge handlers act per item found for the customer; also soft-deletes the customer outbound config if one exists | no items |
| contact | delete contacts and stars by customer id | no rows |
| direct | list directs by customer id, delete each | no items |
| flow | flow and activeflow handlers act per item found | no items |
| number | list numbers, release each, errors skipped | no items |
| queue | queue and queuecall handlers act per item found | no items |
| registrar | trunk, extension handlers per item; customer domain `DeleteByCustomerID` | no items; domain delete is a keyed delete |
| schedule | per-item delete for the customer's schedules | no items |
| storage | file handler per item, then the account handler fetches the account and deletes it; an error from either is wrapped and logged by the subscriber, and a file handler error skips the account step | deletes the account |
| tag | per-item delete | no items |
| transcribe | per-item delete | no items |

Subscriber errors are logged by each subscribe handler. At most 100 events per 15-minute run are emitted.

Timeout: the schedule row for `customer-unverified-cleanup` has a 60-second timeout. One run now does the expire step (at most 100 rows) and the purge step (at most 100 rows, one UPDATE with its cache refresh and one publish each), which is a few hundred short operations. If a run times out, the schedule-manager records a failed execution, the work that finished stays committed, and the next run (15 minutes later) continues; the CAS guard makes a repeated attempt safe. No timeout change is made. Trigger to revisit: failed executions of this schedule in the execution audit.

A resend at day 29 yields a 24-hour token whose validity can extend past the purge; the purge wins and the token is then rejected as `deleted`. A resend does not move `tm_update`, so it does not extend the retention.

### 2-3. Code changes

- `dbhandler`: private helper `customerAnonymize(ctx, id, caller, anonName, anonEmail, guard)` holds the UPDATE that is now inside `CustomerAnonymizePII` (same fields, one `TimeNow()` call, same SQL, same metric label `anonymize_pii` shared by both callers on purpose, error prefix taken from `caller`). `CustomerAnonymizePII` calls it with its existing guard and caller name, so its behavior and error text are unchanged. New `CustomerAnonymizeExpiredUnverified(ctx, id, anonName, anonEmail)` calls it with the guard `status='expired' AND email_verified=false AND tm_delete IS NULL`. New `CustomerListExpiredUnverified(ctx, updatedBefore time.Time, limit uint64)` (same query style as `CustomerListFrozenExpired`, `ORDER BY tm_update ASC`). Interface and mock updated.
- `customerhandler`: new `PurgeExpiredUnverified(ctx) (int, error)` in `purge.go`. The cutoff is `h.utilHandler.TimeNow().Add(-expiredRetention)` so tests can fix the clock (`TimeNow` is called first for the cutoff, then once per claimed row after the anonymize, for the event payload). Anonymized values use the existing format (`deleted_user_<id first 8>`, `deleted_<id first 8>@removed.voipbin.net`). A list failure returns the error; `ErrNotFound` from the anonymize is a debug-level skip; any other anonymize error is logged at error level and skipped; the event is published only after a successful anonymize, built from a value copy (`cp := *c`) of the listed row (decision 4). Logs do not print the email (new code only; the existing two paths keep their current log lines, untouched). Interface (`main.go`) and mock updated.
- `listenhandler/models/response/customers.go`: `V1ResponseCustomersCleanupUnverified` gets a `Purged int` field with the json tag `purged`. `processV1CustomersCleanupUnverifiedPost` runs `CleanupUnverified` and then `PurgeExpiredUnverified` always, logs both errors, and returns `errorResponse` of the first non-nil error (expire first). On success the body is `{"expired": n, "purged": m}`.
- Mocks regenerated with `go generate`: `customerhandler/mock_main.go`, `dbhandler/mock_main.go` (the listenhandler uses `MockCustomerHandler` only). No other service uses these interfaces.
- Docs: `docs/domain.md` (lifecycle, retention), `docs/architecture.md` (sweep endpoint row, staged with the change). docsdev RST: `customer_overview.rst` (status diagram `expired -> deleted` after 30 days, status table row), `customer_struct_customer.rst` (`expired` description), `auth_overview.rst` (a sentence under the state diagram near the signup section, the resend recovery text, the `ACCOUNT_EXPIRED` row and the expired-customer note that says the row keeps blocking sign-up), `quickstart_signup.rst` (the 72-hour and recovery sentences), `restful_api_errors.rst` (the `ACCOUNT_EXPIRED` row: a resend works until the purge, 30 days after expiry). The HTML is rebuilt with a clean build and added with `git add -f`. `docs/operations.md` has no config change. `docs/.docs-gen` is not regenerated (no route, event or config change).

## 3. Verification plan (tests written first)

RED: stubs for the new methods are added first (`return 0, nil` / `ErrNotFound`) so the tests compile and fail on assertions, not on compilation.

dbhandler (sqlite, shared database: assert membership and relative order of the rows the test created, UTC `time.Time` values only, `tm_update` seeded by raw UPDATE like the existing `tm_delete` seeding because `CustomerCreate` leaves it NULL):
- `CustomerAnonymizeExpiredUnverified`: expired+unverified is anonymized (name, email, phone, address, webhook URI, `status=deleted`, `tm_delete`, `tm_update`); expired+verified, expired with `tm_delete`, initial, active, frozen, deleted match 0 rows (`ErrNotFound`); the error prefix names the new method.
- `CustomerListExpiredUnverified`: a row 1 second before the cutoff is listed; a row exactly at the cutoff and a newer row are not; verified rows, rows with `tm_delete` and other statuses are not; ascending `tm_update` order of the created rows; the limit is honored (seed more rows than the limit with the oldest `tm_update`).
- Both new methods are added to the existing dbhandler error-path tables (`error_paths_test.go`, `sqlmock_test.go`): build, exec and rows errors with the method name in the message.
- `CustomerAnonymizePII` and its error-path tests are unchanged and must still pass.
- Mutations (scratch copy): drop `email_verified=false`, drop `tm_delete IS NULL`, drop `status='expired'`, `Lt` to `Lte`, drop `ORDER BY`, drop the limit, give the new method the frozen guard. Each must fail a test.

customerhandler (gomock, fixed clock):
- the cutoff passed to the list is exactly `TimeNow - 30 days` (the 29 days 23 hours / 30 days boundary is asserted by the dbhandler test above and by this exact cutoff value);
- processes listed rows, publishes `customer_deleted` once per successful anonymize and only then (InOrder: `TimeNow`, list, then per row anonymize, `TimeNow`, publish; no `CustomerGet`);
- `ErrNotFound` and any other anonymize error skip with no event and the next row is processed; the count excludes skipped rows;
- the published payload has the anonymized values, `status=deleted`, `tm_delete` and `tm_update` equal to the mocked `TimeNow`, the unchanged `ID` and `billing_account_id`, and none of the original name, email, phone, address or webhook URI (distinct fake values in the listed row); the listed slice element is not modified;
- list error returns the error; empty list returns 0.
- Mutations: publish before anonymize, publish on `ErrNotFound`, call `CustomerGet`. Each must fail.

listenhandler: both steps run even when the first fails; one failing step yields an error response (expire error first); both failing returns the expire error; success body is `{"expired":n,"purged":m}`. The existing cleanup test is updated for the new expectation.

Whole workflow: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`, the repo `scripts/check-*.sh` (including `check-test-conventions.sh` and `check-structure-conventions.sh`, run for real), a clean docsdev rebuild, CI.

## 4. Out of scope

Legacy `expired` rows with `tm_delete` (a separate operations DB card), `deleted` rows whose email was not replaced, api-validator changes, a daily-job variant, `EmailVerify` status guard.
