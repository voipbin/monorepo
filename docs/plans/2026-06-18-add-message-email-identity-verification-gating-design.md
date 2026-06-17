# Add Customer Identity Verification Gating to Message/Email Send (Issue #989)

Status: Design (hardening-class, minimal-change)
Date: 2026-06-18
Issue: voipbin/monorepo#989
Type: Security hardening (architectural asymmetry fix)

---

## 1. Problem Statement

Outgoing PSTN call origination enforces customer identity verification in
`bin-call-manager` (`callhandler.validateOutgoingCallPermission` /
`ValidateCustomerIdentityVerified`, fail-closed, internal-customer bypass).
The equivalent gating is **absent** in `bin-message-manager` and
`bin-email-manager` send paths.

Consequence: an unverified customer who is blocked from originating PSTN calls
can still send SMS / Email by routing through a flow `message_send` /
`email_send` action. Flow action executors call the resource-manager send RPC
directly, bypassing any gating that exists only at the api-manager service
layer. The verified-only send policy is therefore enforced only for calls.

Evidence (from issue, re-confirmed against `origin/main` 2026-06-18):

```
# bin-call-manager: gating present
bin-call-manager/pkg/callhandler/validate.go
  - ValidateCustomerIdentityVerified(...) / validateOutgoingCallPermission(...)
  - PSTN (TypeTel) destinations require cu.IdentityVerificationStatus == Verified
  - internal customer IDs bypass (cucustomer.IsInternalSystemID)

# bin-message-manager: no gating
grep -rni 'verified|identityverif' bin-message-manager/pkg/ (excluding _test) -> 0

# bin-email-manager: no gating
grep -rni 'verified|identityverif' bin-email-manager/pkg/ (excluding _test) -> 0
```

## 2. Scope

In scope:
- `bin-message-manager`: add customer identity verification gating to
  `messagehandler.Send` (the resource-manager send entry point).
- `bin-email-manager`: add customer identity verification gating to
  `emailhandler.Create` (the resource-manager send entry point — `Create`
  validates balance, persists, then dispatches `Send` in a goroutine).
- Regression unit tests for both: verified passes, unverified rejected,
  internal-customer bypass, fetch-failure fail-open.

Out of scope:
- AI tool `create_call` action-assembly feature (#994, tracked separately).
- Any change to `bin-call-manager` (its gating is the reference, untouched).
- api-manager servicehandler-layer gating (the flow-action path bypasses it;
  resource-manager is the canonical enforcement point — see §6).
- SNS / conversation-manager send paths (not in this issue; the same gap may
  exist but is out of #989 scope — noted as Open Question).
- Prometheus metrics for rejections (Phase 2 / Open Question).

## 3. Canonical Enforcement Point

Each resource manager enforces verification in its own send handler. This
matches the call-manager precedent and is the only point the flow-action path
cannot bypass:

```
flow action executor (actionHandleMessageSend / email_send)
  -> MessageV1MessageSend RPC      -> message-manager listenhandler -> messagehandler.Send   [GATE HERE]
  -> EmailV1EmailSend  RPC (Create) -> email-manager  listenhandler -> emailhandler.Create   [GATE HERE]
```

Placing the gate at the api-manager service layer would NOT cover the flow
executor path, which hits the resource-manager RPC directly.

## 4. Fix (before / after)

### 4.1 Shared semantics (mirror of call-manager)

A small per-manager helper that returns a bare `bool`:

- Returns `true` (allow) when:
  - the customer is an internal/system customer (`cucustomer.IsInternalSystemID`), OR
  - the customer cannot be fetched (customer-manager unavailable -> fail-open,
    matching `ValidateCustomerStatusOutgoing`/`ValidateCustomerIdentityVerified`), OR
  - the customer fetch returns `(nil, nil)` (no error, no customer -> fail-open,
    same defensive branch call-manager has as `if cu == nil { return true }`).
- Returns `false` (reject) when:
  - the customer is fetched (non-nil) and
    `cu.IdentityVerificationStatus != cucustomer.IdentityVerificationStatusVerified`.

The `cu == nil` check is MANDATORY before dereferencing `cu.IdentityVerificationStatus`.
`CustomerV1CustomerGet` can in principle return `(nil, nil)` for a not-found-
without-error shape; omitting the guard is both a parity break with call-manager
and a nil-pointer panic risk (review round 1, HIGH).

Difference from call-manager (deliberate, decided with CEO/CTO): **no
destination-type branch.** call-manager only gates PSTN (`TypeTel`) because
non-PSTN call legs are internal. SMS and Email are by nature external, billable
sends, so every send is gated. Internal/system traffic is covered by the
internal-customer bypass, not by a type branch.

**Scope is identity verification ONLY (deliberate narrowing).** call-manager
pairs identity verification (`ValidateCustomerIdentityVerified`) with a separate
account-status check (`validateOutgoingCallPermission` rejects when
`cu.Status != StatusActive`). This fix mirrors ONLY the identity-verification
half, because issue #989 is scoped to identity-verification gating. A
suspended-but-previously-verified customer would still pass this gate. Adding an
account-status check to message/email send is a separate concern (neither
message-manager nor email-manager checks customer status today) and is recorded
as Open Question #5 rather than silently expanded into this PR.

**Fail-open rationale (corrected, review round 1 HIGH).** The gate fails open
when customer-manager is unavailable. The justification is **availability with a
bounded blast radius** (the open window lasts only as long as the customer-
manager outage), matching the call-manager precedent. It is NOT "the billing
balance check is a second enforcement layer" — identity-verification status and
account balance are independent (a prepaid unverified customer can hold ample
balance), so the balance check does NOT backstop this gate. The fail-open branch
MUST log at WARN with `customer_id` so an outage that widens the gate is visible
in logs even before the Phase-2 Prometheus counter (Open Question #2) ships.

### 4.2 message-manager

`bin-message-manager/pkg/messagehandler/validate.go` (new file):

```go
package messagehandler

// validateCustomerIdentityVerified returns true if the customer is allowed to
// send a message. Internal/system customers bypass. If the customer cannot be
// fetched (customer-manager unavailable), fail open. Otherwise the customer's
// identity must be verified.
func (h *messageHandler) validateCustomerIdentityVerified(ctx context.Context, customerID uuid.UUID) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":        "validateCustomerIdentityVerified",
		"customer_id": customerID,
	})

	if cucustomer.IsInternalSystemID(customerID) {
		return true
	}

	cu, err := h.reqHandler.CustomerV1CustomerGet(ctx, customerID)
	if err != nil {
		// fail open: customer-manager unavailable. Bounded blast radius (outage
		// window only). Logged at WARN for outage visibility.
		log.Warnf("Could not get customer info, failing open. err: %v", err)
		return true
	}
	if cu == nil {
		// defensive parity with call-manager: (nil, nil) not-found shape.
		log.Warnf("Customer not found, failing open. customer_id: %s", customerID)
		return true
	}

	if cu.IdentityVerificationStatus != cucustomer.IdentityVerificationStatusVerified {
		log.Infof("Customer identity not verified. Rejecting message send. status: %s", cu.IdentityVerificationStatus)
		return false
	}

	return true
}
```

(The `cu == nil` and `err != nil` branches both fail open and both log at WARN;
shown separately here for clarity. The email-manager helper is byte-identical
except for the reject log wording "message send" -> "email send".)

`messagehandler.Send` (gate inserted before the balance check):

```go
func (h *messageHandler) Send(ctx context.Context, id, customerID uuid.UUID, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*message.Message, error) {
	...
	// gate: customer identity verification (fail-closed)
	if !h.validateCustomerIdentityVerified(ctx, customerID) {
		return nil, fmt.Errorf("customer identity verification required to send message")
	}

	// check the balance
	...
}
```

### 4.3 email-manager

`bin-email-manager/pkg/emailhandler/validate.go` (new file): same helper shape
on `*emailHandler`, returning bool with the same three branches.

`emailhandler.Create` (gate inserted before the balance check, after
destination validation):

```go
func (h *emailHandler) Create(ctx context.Context, customerID, activeflowID uuid.UUID, destinations []commonaddress.Address, subject, content string, attachments []email.Attachment) (*email.Email, error) {
	// validate destinations
	...
	// gate: customer identity verification (fail-closed)
	if !h.validateCustomerIdentityVerified(ctx, customerID) {
		return nil, fmt.Errorf("customer identity verification required to send email")
	}
	// validate balance before sending
	...
}
```

Note: the gate goes BEFORE the balance check so an unverified customer is
rejected with a clear reason and without consuming a billing round-trip. The
order is cosmetic for correctness (both reject), but verification-first reads
as the stronger policy.

**Error-construction style (review round 1 LOW).** Both new gates use
`fmt.Errorf` for the fresh sentinel error (no wrapped cause to carry), so the
two files agree. `errors.Wrap` from `github.com/pkg/errors` is the convention
only when wrapping an underlying error; here there is none. The message-manager
gate uses `fmt.Errorf("customer identity verification required to send message")`
and the email-manager gate uses
`fmt.Errorf("customer identity verification required to send email")`.

## 5. Impact Analysis

| Scenario | Customer | Before | After |
|---|---|---|---|
| Verified customer sends SMS/Email | verified | allowed | allowed (unchanged) |
| Unverified customer sends SMS/Email | not verified | **allowed (bug)** | **rejected (error returned)** |
| Internal/system customer (call-manager, ai-manager, system, basic-route) | bypass | allowed | allowed (bypass) |
| customer-manager unavailable | fetch fails | allowed | allowed (fail-open) |
| Flow `message_send` / `email_send` action by unverified customer | not verified | **allowed (bug)** | **rejected** |

Blast radius: two send entry points. No schema change, no API contract change,
no new RPC, no migration. Error is returned to the RPC caller exactly like the
existing insufficient-balance error path, so api-manager / flow-manager error
surfacing is already wired.

## 6. Error Propagation Path

- message-manager: `Send` returns `error` -> `listenhandler/messages.go:86`
  (`h.messageHandler.Send(...)`) -> RPC error response -> api-manager edge /
  flow-manager action executor surfaces it. Identical to the existing
  `insufficient balance` return at `send.go:48`.
- email-manager: `Create` returns `error` -> `listenhandler/v1_emails.go:77`
  (`h.emailHandler.Create(...)`) -> RPC error response. Identical to the
  existing `insufficient balance` return at `email.go:42`.

No resource is created before the gate (message targets are built in-memory but
not persisted until `Create`; email is not persisted until after the gate), so
there is no leak/cleanup concern.

## 6.1 Entry-point single-choke-point verification (review round 1, code-verified)

A code-verifying reviewer traced every outbound dispatch path in both services
and confirmed the gate sits on the SOLE choke point in each:

- **message-manager:** the only path that reaches a provider `SendMessage` is
  the goroutine *inside* `messagehandler.Send`. `messagehandler.Create` only
  persists (it does not dispatch). The inbound webhook path (`hook.go`) calls
  `Create` with `DirectionInbound` and never dispatches. The listenhandler
  exposes only `POST /v1/messages -> Send`. No `resend`/`retry`/`bulk`/
  `scheduled`/`campaign` RPC route exists. Gating `Send` covers the whole
  service.
- **email-manager:** the only caller of `emailHandler.Send` (the goroutine that
  hits the engines) is `Create` at `email.go:51`. The listenhandler exposes only
  `POST /v1/emails -> Create`. Gating `Create` covers the whole service.
- **Scope of "gap closed":** this fix closes the gap for all sends originated
  through the message-manager / email-manager RPC (which includes the flow
  `message_send` / `email_send` action path and the api-manager REST path). It
  does NOT cover hypothetical originators in OTHER services (SNS /
  conversation-manager) that might dispatch without funneling through these two
  RPCs. That cross-service question is Open Question #1.

## 7. Test Strategy

Both services use gomock + table-driven tests. Add to each send test file
(`messagehandler/send_test.go`, `emailhandler/email_test.go`, or a new
`validate_test.go`):

| Case | CustomerV1CustomerGet mock | Expected |
|---|---|---|
| verified | returns customer with `IdentityVerificationStatusVerified` | send proceeds (balance check reached) |
| unverified | returns customer with non-verified status | error returned, balance check NOT called |
| internal customer | (not called — bypass) | send proceeds, `CustomerV1CustomerGet` NOT called |
| fetch failure | returns error | send proceeds (fail-open), balance check reached |
| nil-nil (not-found) | returns `(nil, nil)` | send proceeds (fail-open), no panic, balance check reached |
| suspended but verified | returns customer `Status != StatusActive`, `IdentityVerificationStatusVerified` | send proceeds (locks in the deliberate identity-only narrowing, OQ#5) |

Key assertions:
- unverified case: assert `CustomerV1CustomerGet` IS called and the downstream
  `BillingV1AccountIsValidBalanceByCustomerID` / `Create` is NOT called
  (gomock no-EXPECT), and the returned error is non-nil.
- internal case: assert `CustomerV1CustomerGet` is NOT called (bypass short-
  circuits before the RPC) AND that the balance check IS reached (bypass allows,
  does not abort the send).
- nil-nil case: assert no panic and that the send proceeds (guards the
  mandatory `cu == nil` branch, review round 1 HIGH).
- email-manager: assert the destination-validation rejection happens BEFORE the
  gate (invalid destination returns its own error without calling
  `CustomerV1CustomerGet`).
- Use one of `cucustomer.IDSystem` etc. for the internal-customer row.

## 8. Affected Services

| Service | Change | Phase |
|---|---|---|
| bin-message-manager | new `validate.go` helper + gate in `Send`; regression tests | 1 |
| bin-email-manager | new `validate.go` helper + gate in `Create`; regression tests | 1 |

## 9. Implementation Order

1. message-manager: add `validate.go`, wire gate into `Send`, add tests.
2. email-manager: add `validate.go`, wire gate into `Create`, add tests.
3. Full verification workflow (`go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run`) in EACH service.
4. Commit, push, PR.

## 10. Open Questions

| # | Question | Recommendation | Owner / Priority |
|---|---|---|---|
| 1 | Does the same gap exist in SNS / conversation-manager send? | Likely yes; out of #989 scope. File a follow-up if confirmed. | CEO/CTO, Phase 2 |
| 2 | Add Prometheus counter for verification rejections (mirror `promCallOutboundWhitelistRejectedTotal`)? | Defer; not required for the security fix. | CEO/CTO, Phase 2 |
| 3 | Should email gate run before or after balance check? | Before (verification-first reads stronger; saves a billing round-trip). | Settled in design |
| 4 | Extract the helper into a shared `bin-common-handler` / customer-model function instead of duplicating per manager? | Keep per-manager duplication for v1 (call-manager already duplicates its own; cross-service shared helper is a larger refactor). | Settled: minimal-change |
| 5 | Add account-status (`cu.Status == StatusActive`) check to message/email send, mirroring call-manager's `validateOutgoingCallPermission`? | Defer. #989 is scoped to identity verification; neither manager checks status today. A suspended-verified customer still passes this gate. File a follow-up if status-gating on send is desired. | CEO/CTO, Phase 2 |
| 6 | Unexport `emailHandler.Send` (latent ungated dispatch surface — only caller today is `Create`'s goroutine)? | Defer. No live bypass (sole caller is post-gate). Unexporting touches mocks/tests; out of minimal-change scope. | Phase 2 |

## 12. Review Summary (v1 -> v2)

Two independent fresh reviewers (one pure-reasoning, one code-verifying against
the actual repo) both returned CHANGES REQUESTED on v1. Changes applied in v2:

- **[HIGH] Added mandatory `cu == nil` fail-open branch** before dereferencing
  `cu.IdentityVerificationStatus`. v1 only handled `err != nil`; a `(nil, nil)`
  RPC shape would have panicked and broke parity with call-manager's
  `if cu == nil { return true }`. (§4.1)
- **[HIGH] Corrected the fail-open rationale.** v1 claimed "billing balance is a
  second enforcement layer"; this is false (verification and balance are
  independent). v2 states the real justification: availability with a bounded
  outage-window blast radius, matching call-manager. Fail-open branch now logs
  at WARN. (§4.1)
- **[MEDIUM] Documented the deliberate account-status narrowing.** call-manager
  pairs identity verification with a `StatusActive` check; this fix mirrors only
  the identity-verification half (issue scope). Recorded as Open Question #5
  instead of silently expanding. (§4.1, OQ#5)
- **[code-verified] Added §6.1** documenting that `Send`/`Create` are the SOLE
  outbound dispatch choke points in each service (reviewer traced all callers of
  provider `SendMessage` / engine `Send`), and scoped the "gap closed" claim to
  message/email-manager-RPC-originated sends. (§6.1)
- **[LOW] Standardized error construction** on `fmt.Errorf` for both gates (no
  wrapped cause, so `errors.Wrap` does not apply). (§4.3)
- **[LOW] Added test cases:** `(nil, nil)` no-panic fail-open; internal-customer
  still reaches balance check; email destination-validation-before-gate
  ordering. (§7)
- **Deferred (Phase 2, in Open Questions):** Prometheus rejection/fail-open
  counter (#2), shared-helper extraction (#4), account-status check (#5),
  unexporting `emailHandler.Send` (#6).

Verdict after v2: both HIGH findings resolved, all MEDIUM findings either
resolved or explicitly deferred with rationale. Ready for re-review.

## 11. Checklist (hardening-class subset)

- [x] No new DB table / migration / schema change
- [x] No API contract change
- [x] fail-closed on unverified, fail-open on fetch failure (call-manager parity)
- [x] internal-customer bypass via `cucustomer.IsInternalSystemID`
- [x] gate placed at resource-manager (flow-action path cannot bypass)
- [x] regression tests cover verified / unverified / internal / fetch-failure
- [x] no destination-type branch (deliberate divergence from call-manager, CEO/CTO approved)
