# VOIP-1678: allow immediate self-delete for initial/expired customers

## 1. Ticket review (re-checked against current main)

Verdict: valid (partially: the HTTP 400 originates in customer-manager, not api-manager).

- `POST /auth/unregister` with `immediate=true` ends in `customerHandler.FreezeAndDelete`, whose first step `Freeze` calls `dbhandler.CustomerFreeze`. That UPDATE is guarded by `status='active'`, so an `initial` or `expired` customer matches 0 rows (`ErrNotFound`) and the listenhandler collapses the error to a bare 400.
- Nothing records the `active` guard as a deliberate exclusion of `initial`. Unverified customers may delete their own account when `immediate=true`.
- No overlapping change in the recent history or open/merged PRs touching `freeze.go`.
- Legacy expired customers (`tm_delete` set) and deleted customers stay rejected: `Freeze` rejects `Status==deleted || TMDelete!=nil` before any DB write. That check stays first and a test asserts no DB write happens.

## 2. Design

Scope: `immediate=true` only. `immediate=false` (30-day grace), admin freeze and `CustomerFreeze` are unchanged (active only).

### 2-1. customer-manager

- `dbhandler`: new `CustomerFreezeAllowUnverified(ctx, id)`: same UPDATE as `CustomerFreeze` but guarded by `status IN ('active','initial','expired') AND tm_delete IS NULL`; 0 rows returns `ErrNotFound`. One private helper builds the UPDATE from the allowed-status list (no duplicated SQL). Interface and mock updated.
- `customerhandler/freeze.go`: `Freeze` becomes a thin wrapper over a private `freeze(ctx, id, allowUnverified bool)`; `FreezeAndDelete` calls `freeze(..., true)`. Order inside `freeze` is unchanged (Get, already-frozen idempotent return, deleted/`TMDelete!=nil` rejection, DB freeze, re-Get, publish `customer_frozen`). The DB method is chosen by the flag only, not by the status read in Get, so there is no read-then-choose gap. The concurrent-freeze refetch is kept.
- Retry: if the freeze succeeds and the anonymization call then fails, the customer stays `frozen`. Calling `POST /auth/unregister` with `immediate=true` again takes the already-frozen path and proceeds to anonymization (the unregister endpoint is exempt from the account-status gate). If nobody retries, the 30-day cleanup anonymizes and deletes the customer automatically. Known limit (same as for active customers today): if anonymization succeeds but the following read or event publish fails, the customer is already `deleted`, a retry is rejected, and the `customer_deleted` cascade is not emitted.
- Events are the same as for an active customer: `customer_frozen` (subscribers: billing, call), then `customer_deleted` after anonymization (13 subscribers: agent, billing, call, contact, direct, flow, number, queue, registrar, schedule, storage, tag, transcribe). Checked: the billing and call frozen handlers list accounts/calls of the customer and act per item, with no precondition on the customer status, so zero accounts or zero calls (typical for unverified customers) is a no-op. The `customer_deleted` cascade is the same one the admin delete already emits for these statuses.

### 2-2. Verification must not be bypassed through Recover

Recover currently restores any `frozen` customer to `active` without looking at `email_verified`. With this change an unverified customer can be `frozen` (briefly inside the immediate call, or longer if a later step failed), so Recover must not turn it into an active account.

- `customerhandler.Recover` rejects a frozen customer whose `email_verified` is false (clear error before any DB write, no event), and `dbhandler.CustomerRecover` adds `email_verified = true` to its UPDATE guard so the rule also holds if the state changes between the read and the write (0 rows returns `ErrNotFound`, which the handler's concurrent-recover refetch treats as failure because the customer is not `active`).
- Effect: a frozen unverified customer can only be finished by `immediate=true` (retry) or the 30-day cleanup. This closes both the millisecond window inside the immediate call and the residual-frozen case, instead of only documenting them.
- Impact on existing data: only frozen customers with `email_verified=false` are affected. No code path leaves an `active` customer unverified (signup creates `initial`; email verification sets `email_verified=true` and `active` together; admin-created customers are created verified), so a customer frozen from `active` is verified and recovers as before. Trigger to revisit: any report of a frozen customer that cannot recover although it was active before freezing. The admin recover endpoint uses the same handler and is affected the same way.

### 2-3. Credential branches (api-manager, no code change)

- `confirmation_phrase`: no re-authentication; works for `initial` and `expired` with an existing token or access key (the account-status gate exempts `/auth/unregister`).
- `password`: `initial` works. `expired` is refused earlier by the login path with 403, which is unchanged. Docs tell expired users to use the confirmation phrase.

### 2-4. Docs

- OpenAPI descriptions of the unregister/recover endpoints are checked for an 'active only' statement; if present they are corrected together with their generated artifacts.
- `auth_overview.rst` (POST /auth/unregister): `immediate=true` accepts `initial`/`expired`; `immediate=false` still requires `active`; legacy and deleted unchanged; 400 cause text updated; password-branch 403 explained for `expired`; DELETE /auth/unregister: an unverified frozen account cannot be recovered. Also `customer_overview.rst` where it describes unregister. Rebuild and add the HTML.
- `bin-customer-manager/docs/domain.md`: lifecycle note.
- No OpenAPI change. Error-code visibility work (bare 400) is out of scope.

## 3. Verification plan (tests, written first and shown failing on current code)

- dbhandler (sqlite): `CustomerFreezeAllowUnverified` freezes active, initial, expired; rejects frozen, deleted, and initial/expired with `tm_delete`. `CustomerFreeze` still rejects initial/expired. A guard-removal mutation (drop `tm_delete IS NULL`) makes the legacy test fail.
- customerhandler `FreezeAndDelete`: initial and expired go freeze, `customer_frozen`, anonymize, `customer_deleted` in this order; legacy expired and deleted return an error with no DB write and no event; already-frozen retry proceeds to anonymize; anonymize failure returns an error without `customer_deleted`; concurrent-freeze refetch.
- customerhandler `Freeze` (immediate=false path): initial/expired still use `CustomerFreeze` (error from DB guard).
- customerhandler `Recover`: frozen + unverified rejected without DB write; frozen + verified recovers as before. dbhandler `CustomerRecover`: frozen + unverified matches 0 rows (guard-removal mutation fails the test), frozen + verified recovers.
- Event order asserted with gomock `InOrder` (frozen, anonymize, deleted); anonymize `ErrNotFound` (lost race) returns an error with no `customer_deleted`.
- Full verification workflow, `scripts/check-*.sh`, CI.

## 4. Out of scope

Cleaning up accumulated expired/initial rows, legacy rows, listenhandler error-code changes.
