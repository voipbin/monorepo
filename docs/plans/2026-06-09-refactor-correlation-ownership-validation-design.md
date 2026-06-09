# Refactor correlation ownership validation into an error-returning helper

Date: 2026-06-09
Service: bin-ai-manager
Type: Refactor (behavior-preserving) + security-invariant hardening

## 1. Problem Statement

The `get_correlation` LLM tool handler (`bin-ai-manager/pkg/aicallhandler/tool.go`,
`toolHandleGetCorrelation`) performs correlation lookup, ownership validation, and
existence-oracle masking all inline in a single ~75-line function.

The security model is sound today, but the masking string `msgCorrelationNotFound`
is written at **four separate call sites** inside the handler. Every "you cannot
see this" branch must independently remember to emit the exact same byte-identical
string. This is a structural fragility:

- A future edit (renaming, adding a branch, refactoring) can make one path diverge
  from the others, silently turning the tool back into a cross-customer existence
  oracle (IDOR information leak).
- The ownership-validation logic cannot be unit-tested in isolation; only the full
  handler behavior is testable.

There is no functional bug. This is a maintainability + defense-in-depth change:
collapse the four masking sites into one, and isolate the validation logic behind
a single error-returning helper so the security invariant is enforced structurally
rather than by convention.

## 2. Scope

In scope:
- Extract correlation lookup + ownership validation from `toolHandleGetCorrelation`
  into a new helper `resolveCorrelation` that returns `(*tmcorrelation.Correlation, error)`.
- Introduce a single sentinel error `ErrCorrelationNotAccessible` that represents
  every "caller may not see this" outcome (absent / cross-customer /
  ownership-lookup-failure / foreign-id-with-no-activeflow).
- Move the cross-customer Warn log into the helper (validation locus = log locus).
- Reduce the handler so `msgCorrelationNotFound` is emitted at exactly ONE call site.

Out of scope:
- No change to the tool's external behavior. This is behavior-preserving; all
  existing tests in `tool_correlation_test.go` must pass unchanged.
- No Prometheus metric for IDOR attempts (deferred; see Open Questions).
- No change to the "own session, no activeflow" disclosure message.
- No change to timeline-manager, flow-manager, requesthandler, or any other service.
- No change to `pipecat-manager` (it delegates tool execution to ai-manager via
  `AIV1AIcallToolExecute`; it has no correlation code).

## 3. Two-Tier Error Contract (key design decision)

The current handler has TWO distinct rejection behaviors, and the refactor must
preserve both:

| Situation | Current behavior | Why |
|---|---|---|
| timeline RPC itself fails (line 533) | `fillFailed` + "correlation lookup failed" | Resource existence is UNKNOWN; there is no existence to mask. Honestly reporting a tool failure leaks nothing. |
| resource absent | `fillSuccess` + `msgCorrelationNotFound` | Mask. |
| resource exists, cross-customer | `fillSuccess` + `msgCorrelationNotFound` | Mask. Hide that the activeflow / resource exists. |
| resource exists, ownership lookup (flow-manager) fails | `fillSuccess` + `msgCorrelationNotFound` | Mask. Resource was found; do NOT reveal the activeflow exists. |
| resource exists, no activeflow, foreign id | `fillSuccess` + `msgCorrelationNotFound` | Mask. |
| resource exists, no activeflow, own session | `fillSuccess` + "This resource exists but is not linked to any call flow." | Own resource: safe to disclose. |
| owned + has activeflow | `fillSuccess` + correlation summary | Normal success. |

The helper therefore distinguishes two error classes:

1. **`ErrCorrelationNotAccessible`** (sentinel) — caller MUST mask with
   `msgCorrelationNotFound`. Covers: absent, cross-customer, ownership-lookup
   failure, foreign-id-no-activeflow.
2. **Any other (wrapped) error** — caller reports a genuine tool failure via
   `fillFailed`. Covers: timeline RPC failure (existence unknown).

Rationale: masking an infra failure as "no events found" would make the LLM tell
the user a falsehood ("nothing exists") when the truth is "we could not check".
The existing handler already distinguishes these; the refactor keeps the distinction.

The "own session, no activeflow" disclosure is NOT an error: the helper returns
`(corr, nil)` and the handler branches on `corr.ActiveflowID == uuid.Nil` to pick
the disclosure message. This keeps the ownSession-gated disclosure decision in the
handler where the ownSession flag lives.

## 4. Helper Design

```go
// ErrCorrelationNotAccessible is the single sentinel covering every outcome where
// the caller may not see the correlation: genuinely absent, exists-but-cross-customer,
// ownership-lookup failure, and foreign-id-with-no-activeflow. Callers MUST collapse
// it to the byte-identical msgCorrelationNotFound so the tool cannot be used as a
// cross-customer existence oracle. It deliberately does NOT distinguish the four
// causes — that is the security property.
//
// Use stdlib errors.New (aliased stderrors), NOT github.com/pkg/errors.New: a
// sentinel needs no stack trace, and identity comparison via stderrors.Is is all
// that matters.
var ErrCorrelationNotAccessible = stderrors.New("correlation not accessible")

// resolveCorrelation fetches the correlation graph for resourceID and validates
// that the caller (callerCustomerID) owns it.
//
// Returns:
//   - (corr, nil)                          : access granted. corr is non-nil.
//                                            corr.ActiveflowID may be uuid.Nil only
//                                            when ownSession is true (caller's own
//                                            unlinked resource).
//   - (corr, ErrCorrelationNotAccessible)  : caller may not see this. corr is
//                                            non-nil but MUST NOT be exposed; the
//                                            caller masks and ignores corr entirely.
//   - (nil, <wrapped err>)                 : transient/infra failure (e.g. timeline
//                                            RPC down). corr is nil. Existence is
//                                            unknown; caller reports a tool failure.
//
// CONTRACT: corr is meaningful ONLY when err == nil. On any non-nil error the caller
// MUST NOT dereference corr (it is nil for the transient case). The handler enforces
// this by returning inside the error block before reading corr.
//
// ownSession must be true iff the caller did not supply a resource_id (target is the
// caller's own activeflow, owned by definition).
func (h *aicallHandler) resolveCorrelation(
    ctx context.Context,
    callerCustomerID uuid.UUID,
    resourceID uuid.UUID,
    ownSession bool,
) (*tmcorrelation.Correlation, error) {
    log := logrus.WithFields(logrus.Fields{
        "func":        "resolveCorrelation",
        "customer_id": callerCustomerID,
        "resource_id": resourceID,
    })

    corr, err := h.reqHandler.TimelineV1CorrelationGet(ctx, resourceID)
    if err != nil {
        // Existence unknown -> genuine tool failure, NOT a mask. corr is nil.
        return nil, errors.Wrap(err, "correlation lookup failed")
    }

    if !corr.ResourceFound {
        return corr, ErrCorrelationNotAccessible
    }

    // No activeflow: no anchor to validate ownership against. Disclose only for the
    // caller's own session; mask a supplied foreign id.
    if corr.ActiveflowID == uuid.Nil {
        if ownSession {
            return corr, nil
        }
        return corr, ErrCorrelationNotAccessible
    }

    // Has activeflow: validate ownership via flow-manager (timeline has no customer_id).
    af, err := h.reqHandler.FlowV1ActiveflowGet(ctx, corr.ActiveflowID)
    if err != nil {
        // Mask the lookup failure: do not reveal that the activeflow exists.
        log.Warnf("Could not verify correlation ownership. resource_id: %s, err: %v", resourceID, err)
        return corr, ErrCorrelationNotAccessible
    }
    if af.CustomerID != callerCustomerID {
        log.Warnf("Cross-customer correlation attempt blocked. session_customer: %s, resource_owner: %s, resource_id: %s",
            callerCustomerID, af.CustomerID, resourceID)
        return corr, ErrCorrelationNotAccessible
    }

    return corr, nil
}
```

Note: the helper carries its own `logrus.WithFields` (it has no access to the
handler's logger). The Warn message strings and their structured fields
(`resource_id`, both customer IDs) are preserved verbatim from the current code so
existing log-based greps/alerting do not break.

## 5. Handler After Refactor

CRITICAL: every branch inside the `if err != nil` block MUST `return`. If the mask
or failure branch falls through, `corr.ActiveflowID` is read on the error path —
which (a) for a cross-customer error exposes a foreign activeflow summary (IDOR),
and (b) for the transient case dereferences a nil `corr` (panic). The explicit
returns below are load-bearing, not stylistic.

The `resourceID == uuid.Nil -> "no resource_id available"` pre-guard MUST be kept
BEFORE calling the helper (own session with no active session). Dropping it diverges
behavior (mask instead of honest failure) and issues a needless RPC on uuid.Nil.

```go
func (h *aicallHandler) toolHandleGetCorrelation(ctx context.Context, c *aicall.AIcall, tc *message.ToolCall) *messageContent {
    log := logrus.WithFields(logrus.Fields{
        "func":      "toolHandleGetCorrelation",
        "aicall_id": c.ID,
    })
    log.Debugf("handling tool get_correlation.")

    res := newToolResult(tc.ID)

    var args struct {
        ResourceID string `json:"resource_id"`
    }
    _ = json.Unmarshal([]byte(tc.Function.Arguments), &args)

    ownSession := args.ResourceID == ""
    resourceID := c.ActiveflowID
    if !ownSession {
        parsed, err := uuid.FromString(args.ResourceID)
        if err != nil {
            fillFailed(res, fmt.Errorf("invalid resource_id"))
            return res
        }
        resourceID = parsed
    }
    if resourceID == uuid.Nil {
        // Pre-guard: own session but no active session. Honest failure, not a mask.
        fillFailed(res, fmt.Errorf("no resource_id available"))
        return res
    }

    corr, err := h.resolveCorrelation(ctx, c.CustomerID, resourceID, ownSession)
    if err != nil {
        if stderrors.Is(err, ErrCorrelationNotAccessible) {
            // Single masking site for ALL not-accessible paths. MUST return.
            fillSuccess(res, "correlation", resourceID.String(), msgCorrelationNotFound)
            return res
        }
        // Transient/infra failure: existence unknown, report tool failure. MUST return.
        log.Errorf("Correlation lookup failed. err: %v", err)
        fillFailed(res, fmt.Errorf("correlation lookup failed"))
        return res
    }

    // err == nil below: corr is safe to read.
    // Own-session unlinked resource gets the disclosure message.
    if corr.ActiveflowID == uuid.Nil {
        fillSuccess(res, "correlation", resourceID.String(), "This resource exists but is not linked to any call flow.")
        return res
    }

    fillSuccess(res, "correlation", corr.ActiveflowID.String(), formatCorrelationSummary(corr))
    return res
}
```

`msgCorrelationNotFound` now appears at exactly ONE call site in the handler.

## 6. Behavior Preservation Matrix

Every existing test case maps 1:1 to identical output:

| Existing test | resolveCorrelation returns | Handler output | Unchanged? |
|---|---|---|---|
| own session - has activeflow + resources | (corr, nil) | summary | ✓ |
| supplied owned id - confirmed | (corr, nil) | summary | ✓ |
| cross-customer blocked | (corr, ErrCorrelationNotAccessible) | mask | ✓ |
| resource not found | (corr, ErrCorrelationNotAccessible) | mask | ✓ |
| own session no activeflow | (corr, nil) [ActiveflowID nil] | disclosure | ✓ |
| foreign id no activeflow | (corr, ErrCorrelationNotAccessible) | mask | ✓ |
| ownership lookup error | (corr, ErrCorrelationNotAccessible) | mask | ✓ |
| invalid resource_id | (helper not called) | fillFailed "invalid resource_id" | ✓ |
| correlation lookup error | (nil, wrapped err) | fillFailed "correlation lookup failed" | ✓ |

The `maskingInvariant` test continues to pass: all four mask paths still emit
`msgCorrelationNotFound`, now through a single handler site (strictly stronger).

## 7. Tests

Keep all existing `tool_correlation_test.go` cases unchanged (they exercise the
handler end-to-end and prove behavior preservation).

Add a focused table test for the helper in isolation, asserting the error class
with `errors.Is`:

```go
func Test_resolveCorrelation(t *testing.T) {
    // cases: granted-own, granted-supplied, granted-own-no-activeflow,
    //        absent, cross-customer, ownership-lookup-fail, foreign-no-activeflow,
    //        timeline-rpc-fail
    // assert: errors.Is(err, ErrCorrelationNotAccessible) for the four mask cases,
    //         err == nil for the three granted cases,
    //         err != nil && !errors.Is(...) for the timeline-rpc-fail case.
}
```

This directly locks the two-tier error contract (Section 3) at the helper level,
which the handler-level tests cannot express as precisely. Include a NEGATIVE
assertion for the transient case: `stderrors.Is(err, ErrCorrelationNotAccessible)`
MUST be false when the timeline RPC fails (the error is a wrapped transient, not
the sentinel). This locks the bare-vs-wrapped discrimination so a future change to
`Wrap` semantics cannot silently turn an infra error into a mask.

Additionally, keep/extend a handler-level regression case proving a cross-customer
`resource_id` NEVER yields a summary and a transient failure NEVER panics — i.e.
the `if err` block is terminal. The existing `maskingInvariant` test plus the
cross-customer case in `Test_toolHandleGetCorrelation` already cover the non-panic
+ mask outcome; verify they still hold after the refactor.

## 8. Security & Compliance

- IDOR / existence-oracle protection is preserved and structurally strengthened:
  the masking string now has a single emission point, so divergence is no longer
  possible by editing one branch.
- Cross-customer attempt logging is retained (moved into the helper).
- No PII handling change. No external LLM call change. No new data exposure.

## 9. Affected Services

| Service | Change | Phase |
|---|---|---|
| bin-ai-manager | Extract `resolveCorrelation` helper + `ErrCorrelationNotAccessible` sentinel; collapse masking to one site; add helper unit test | 1 |

Single-file production change (`pkg/aicallhandler/tool.go`) plus a test file.
No DB migration, no OpenAPI change, no RST docs (internal RPC-only tool).

## 10. Implementation Order

1. Add `ErrCorrelationNotAccessible` sentinel via stdlib `stderrors.New` (no stack trace needed for a sentinel).
2. Add `resolveCorrelation` helper with the two-tier contract + its own `logrus.WithFields` + moved Warn logs (verbatim strings/fields).
3. Rewrite `toolHandleGetCorrelation` to call the helper and mask at one site; keep the `uuid.Nil` pre-guard; ensure BOTH `if err` branches return.
4. Add `Test_resolveCorrelation` table test (incl. negative `!stderrors.Is` assertion for the transient case).
5. Run full verification in `bin-ai-manager`:
   `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`.
6. Confirm existing `Test_toolHandleGetCorrelation` and `_maskingInvariant` pass unchanged.

## 11. Open Questions

| Question | Recommendation | Owner |
|---|---|---|
| Add a Prometheus counter for cross-customer attempts (`...correlation_access_total{result}`)? | Defer to a Phase 2 if security monitoring/alerting is wanted. Out of scope here to keep the change minimal. | CEO/CTO |
| Should timeline-RPC failure also be masked (vs honest tool failure)? | No. Keep current behavior: existence is unknown, masking would make the LLM assert a falsehood. Honest failure leaks nothing. | CPO (resolved) |
| `errors.Wrap` vs `fmt.Errorf("%w")` for the transient case | Use `github.com/pkg/errors` (already the file's convention) for the wrapped transient error; sentinel via `errors.New`. | CPO (resolved) |

## 12. Implementation Note: errors package aliasing

The file imports `github.com/pkg/errors` as `errors`. Sentinel comparison needs
stdlib `errors.Is`. The same package already established the convention in
`db.go`: import stdlib as `stderrors` and call `stderrors.Is(...)`, keeping
`github.com/pkg/errors` as the default `errors` for `Wrap`/`Wrapf`/`New`.

- Sentinel: `var ErrCorrelationNotAccessible = stderrors.New(...)` (stdlib; no stack trace for a sentinel).
- Handler check: `stderrors.Is(err, ErrCorrelationNotAccessible)`.
- `github.com/pkg/errors` v0.9+ `Wrap` implements `Unwrap()`, so `stderrors.Is`
  traverses the chain correctly. The sentinel is returned bare (not wrapped), so
  the comparison is direct regardless.

Follow `db.go` line 176 (`stderrors.Is(err, dbhandler.ErrNotFound)`) as the precedent.

## 13. Review Summary

### Round 1 (two independent reviewers)

Reviewer A: APPROVE (security invariant preserved + structurally strengthened;
9-case byte-identical mapping confirmed incl. the ResourceID field). Implementation
must-dos: helper needs its own logger; preserve exact Warn strings/fields.

Reviewer B: CHANGES REQUESTED. Critical finding: the design pseudocode elided the
`return`s inside the `if err` block. If a branch falls through, a cross-customer
error reaches the success-summary path (foreign activeflow disclosure / IDOR) and
the transient case dereferences a nil `corr` (panic). Also flagged the dropped
`uuid.Nil` pre-guard and the moved Warn log fields.

### v1 -> v2 changes applied

- (Critical, R-B) Handler `if err` block: both branches now have explicit, load-bearing
  `return`s; added a CRITICAL note above the handler explaining the IDOR/panic risk of
  fall-through. Added a "corr is meaningful ONLY when err == nil" contract to the helper doc.
- (Medium, R-B) Kept the `resourceID == uuid.Nil -> "no resource_id available"` pre-guard
  explicitly before the helper call.
- (Medium, R-A) Helper constructs its own `logrus.WithFields`; documented.
- (Low, R-A/B) Warn message strings + fields (resource_id, both customer IDs) preserved verbatim.
- (Low, R-A) Sentinel switched to stdlib `stderrors.New` (no stack trace for a sentinel).
- (Info, R-A) `Test_resolveCorrelation` gains a negative `!stderrors.Is` assertion for the
  transient case; handler-level cross-customer/transient regression coverage reaffirmed.

### Round 2 verdict

APPROVE. The Critical (fall-through IDOR + nil-corr panic) is fully resolved by the
explicit returns and the err==nil deref guard; masking stays byte-identical across
all three "cannot see" paths; transient path neither panics nor leaks; sentinel
routing is sound. Confirmed independently: timeline-manager signals absence via
`ResourceFound=false` (not via err), so the not-found-vs-transient split does not
re-open an existence oracle (verified in bin-timeline-manager eventhandler/correlation.go).
Fail-closed on flow-manager error is intentional.
