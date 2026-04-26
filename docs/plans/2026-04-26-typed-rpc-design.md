# Typed RPC Error Envelope — Design

**Status:** Active
**Author:** Sungtae Kim
**Date:** 2026-04-26
**Related:** Follows the 18-PR `bin-api-manager` structured-error-envelope migration. Eliminates the substring-fallback in the api-manager translator by making upstream managers emit typed errors over RPC.

---

## 1. Goal & non-goals

### Goal

Eliminate `bin-api-manager/server/error_translate.go` section 4 (substring fallback for legacy `fmt.Errorf` strings). After this work, every error reaching api-manager is one of:

1. A typed `*cerrors.VoipbinError` from an upstream manager (translator section 1: `errors.As` passthrough).
2. An api-manager-local sentinel from `pkg/serviceerrors` (translator section 2: `errors.Is` match).
3. A true transport failure: `context.Canceled` or `context.DeadlineExceeded` (translator section 3).
4. An unclassified internal error caught by the section-5 `INTERNAL` safety net.

Section 4 (substring matching against `http.StatusText`-derived strings) is removed. The substring rules in section 4 have always been a leaky abstraction — they match strings that the api-manager's own `requesthandler` synthesized from status codes, not strings that upstream managers actually emit. They are infrastructure pretending to be domain logic, and they swallow real bugs (an internal error containing the word "unavailable" gets misclassified as a SERVICE_UNAVAILABLE response).

### Non-goals

- **Replacing every `fmt.Errorf` in the codebase.** Only errors with a user-visible classification get migrated. Examples that get migrated: `NOT_FOUND`, `INVALID_ARGUMENT`, `FAILED_PRECONDITION`, `PAYMENT_REQUIRED`, `PERMISSION_DENIED`, `RESOURCE_EXHAUSTED`, `ALREADY_EXISTS`. Examples that stay as plain `fmt.Errorf` (and ride the default `INTERNAL` path): DB connection failures, JSON marshal errors, programmer-error invariant violations.
- **Manager-to-manager protocol changes** beyond what `cerrors.ToResponse`/`FromResponse` already do. The wire is backward-compatible: a non-migrated client just ignores `DataTypeVoipbinError` and treats the response the same way it always has.
- **Touching api-manager-local error logic.** Sections 1–3 and 5 of the translator stay as-is.

---

## 2. Current state (confirmed in scoping)

```
[manager business handler]
   fmt.Errorf("not found: ...")
       │
       ▼
[manager listenHandler]
   simpleResponse(404)         ← bare response, no body, no DataType
       │
       ▼
[bin-common-handler/pkg/requesthandler/common.go:parseResponse]
   getResponseStatusCodeError(404)
   → ErrNotFound = errors.New("Not Found")    ← canned http.StatusText sentinel
       │
       ▼
[bin-api-manager/pkg/servicehandler]
   fmt.Errorf("get xyz failed: %w", err)
       │
       ▼
[bin-api-manager/server/error_translate.go]
   strings.ToLower(err.Error()) → contains "not found" → NotFound envelope
                                  ↑
                          section 4 — substring-matches the canned sentinel string,
                          NOT manager-emitted text
```

**Key insight from scoping**: section 4 isn't matching upstream service text. It's matching `http.StatusText` strings synthesized inside `bin-common-handler/pkg/requesthandler/common.go` from status codes. That means the fix lives in `requesthandler`, not just per-manager.

---

## 3. Target end-state

```
[manager business handler]
   return cerrors.NotFound(commonoutline.ServiceNameXManager, "RESOURCE_NOT_FOUND",
                           "Account not found.").WithRequestID(...)
       │
       ▼
[manager listenHandler]
   if errors.As(err, &ve) { return cerrors.ToResponse(ve) }
       ← StatusCode=404, DataType=DataTypeVoipbinError, body=JSON
       │
       ▼
[requesthandler.parseResponse]
   if ve := cerrors.FromResponse(resp); ve != nil { return ve }
       ← typed *VoipbinError flows through
       │
       ▼
[api-manager servicehandler]
   return fmt.Errorf("...: %w", err)   ← still wraps; the cause is typed
       │
       ▼
[error_translate.go section 1]
   errors.As(err, &ve) → return ve     ← passthrough, no string matching
```

Section 4 deleted. Section 5 (default `INTERNAL`) catches the rare case of an un-classified error.

---

## 4. Audit findings (pre-implementation scoping)

### 4.1 `errors.Is`/`errors.Cause` consumers of `requesthandler.Err*` (production-only)

Three production sites depend on the legacy sentinels. They will silently regress when their upstream starts emitting typed errors, because `*VoipbinError` is a different concrete type and won't match `errors.Is(err, requesthandler.ErrNotFound)`.

| File | Pattern | Behavior at risk |
|------|---------|------------------|
| `bin-call-manager/pkg/channelhandler/hangup.go:46` | `errors.Cause(errHangup) == requesthandler.ErrNotFound` | "Channel already gone" silently treated as success. Regression: the typed 404 falls through to error propagation, hangup looks like a real failure. |
| `bin-api-manager/pkg/servicehandler/provider.go:212` | `errors.Is(err, commonrequesthandler.ErrUnprocessableEntity)` | "Carrier rejected key" → `CARRIER_CREDENTIALS_REJECTED`. Regression: typed 422 falls through, returns generic error to client. |
| `bin-api-manager/server/providers.go:213` | Same as above (HTTP-handler-layer duplicate guard) | Same. |

**Mitigation**: PR0 patches all three with parallel typed-error guards alongside the legacy check. Both branches stay until the relevant upstream is migrated. Cleanup of legacy guards happens in the per-manager PRs as the upstream is migrated.

### 4.2 Manager scope

All 28 managers with a `pkg/listenhandler/` (RPC entry point) are in scope:

```
agent, ai, billing (pilot), call, campaign, conference, contact,
conversation, customer, direct, email, flow, message, number, outdial,
pipecat, queue, rag, registrar, route, storage, tag, talk, timeline,
transcribe, transfer, tts, webhook
```

Excluded: `dbscheme-manager`, `openapi-manager`, `sentinel-manager` (no listenhandler / not RPC entry points).

### 4.3 api-validator framework

`monorepo-monitoring/api-validator/tests/helpers/assertions.py:assert_error_envelope()` already supports the per-domain assertion needed for end-to-end verification (`domain="billing-manager"` etc.). All 10 canonical statuses are in `CANONICAL_STATUSES`. Existing per-resource error scenarios in `tests/scenarios/test_*_errors.py` are the targets for adding domain assertions per migrated manager.

No framework work needed. Per-manager PR adds scenarios.

---

## 5. Migration plan (30 PRs)

### PR0 — Wire unwrap path `[bin-common-handler + 3 callers]`

**Scope:**

1. **`bin-common-handler/pkg/requesthandler/common.go:parseResponse`** — add typed-error detection ahead of the legacy `HttpStatusErrorMap` path:

    ```go
    func parseResponse(resp *sock.Response, out any) error {
        if resp == nil {
            return nil
        }

        // NEW: typed VoipbinError takes precedence over canned sentinel.
        if ve := cerrors.FromResponse(resp); ve != nil {
            return ve
        }

        if errStatus := getResponseStatusCodeError(resp.StatusCode); errStatus != nil {
            return errStatus
        }
        // ... existing path unchanged
    }
    ```

    Behavior: if upstream sets `DataType=DataTypeVoipbinError`, the typed `*VoipbinError` flows up; otherwise legacy path fires. Wire-compatible with all non-migrated managers.

2. **Patch 3 production callers** that depend on the legacy sentinel match. Add a parallel typed guard:

    ```go
    // call-manager hangup.go:
    var ve *cerrors.VoipbinError
    if errors.As(errHangup, &ve) && ve.Status == cerrors.StatusNotFound {
        return nil
    }
    if errors.Cause(errHangup) == requesthandler.ErrNotFound {
        return nil
    }
    ```

3. **Unit tests** in `bin-common-handler/pkg/requesthandler/`:
   - `parseResponse` with `DataType=DataTypeVoipbinError` and valid body → returns `*VoipbinError`.
   - `parseResponse` with status 404 and no DataType → returns legacy `ErrNotFound`.
   - `parseResponse` with `DataType=DataTypeVoipbinError` and malformed JSON → falls through to legacy path.
   - `parseResponse` with 200 status → no error, unchanged behavior.

**Verify:** all 28+ services that depend on `bin-common-handler`. Mostly mechanical (no service-side code changes; just rebuild + test). Use the standard verification:
`go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`

**Branch:** `NOJIRA-typed-rpc-pr0-wire-unwrap`

---

### PR1 — Pilot listenHandler integration `[bin-billing-manager]`

**Scope:**

- Add `errorResponse(err error) *sock.Response` helper in `bin-billing-manager/pkg/listenhandler/`:

    ```go
    func errorResponse(err error) *sock.Response {
        var ve *cerrors.VoipbinError
        if errors.As(err, &ve) {
            if resp, e := cerrors.ToResponse(ve); e == nil {
                return resp
            }
        }
        return simpleResponse(http.StatusInternalServerError)
    }
    ```

- Replace `simpleResponse(4xx)` call sites in the dispatcher with `errorResponse(err)` where the error originates from a business handler.
- **No business-logic changes.** Existing `fmt.Errorf` paths still produce a 500 via the helper's fallback.

**Verify:** bin-billing-manager + bin-api-manager.

**Branch:** `NOJIRA-typed-rpc-pr1-billing-listenhandler`

---

### PR2 — Pilot business-handler migration `[bin-billing-manager]`

**Scope:** migrate user-visible errors in `accounthandler/`, `billinghandler/`. Mapping:

| Current | New |
|---------|-----|
| `fmt.Errorf("invalid status: %s", status)` | `cerrors.InvalidArgument(svc, "INVALID_STATUS", "...")` |
| `fmt.Errorf("unsupported billing type")` | `cerrors.InvalidArgument(svc, "UNSUPPORTED_BILLING_TYPE", "...")` |
| `fmt.Errorf("insufficient balance")` (production sites) | `cerrors.PaymentRequired(svc, "INSUFFICIENT_BALANCE", "...")` |
| Account/billing not-found from DB | `cerrors.NotFound(svc, "RESOURCE_NOT_FOUND", "...")` |
| DB connection / marshal failures | **stay as `fmt.Errorf`** → `INTERNAL` via default |

Update tests asserting on error strings to use `errors.Is` / `errors.As`.

**End-to-end check via api-validator:** trigger `INSUFFICIENT_BALANCE` and confirm:
- HTTP 402.
- Body shape `{"error": {"status": "PAYMENT_REQUIRED", "reason": "INSUFFICIENT_BALANCE", "domain": "billing-manager", ...}}`.
- `domain` field reads `billing-manager`, **not** `api-manager`. This proves the typed envelope flowed end-to-end and didn't get re-synthesized by api-manager's translator.

**Verify:** bin-billing-manager + bin-api-manager + monorepo-monitoring api-validator (add error-envelope assertions for billing endpoints).

**Branch:** `NOJIRA-typed-rpc-pr2-billing-handlers`

---

### PR3..N — Replicate per remaining manager `[one PR per manager]`

Each follows the compressed pattern (listenHandler integration + business-handler migration in one PR), since the pilot proves the wiring.

**Order** (priority by api-manager exposure × error-semantic complexity):

| Tier | PR | Manager | Branch |
|------|-----|---------|--------|
| 1 | PR3 | bin-call-manager | `NOJIRA-typed-rpc-pr3-call` |
| 1 | PR4 | bin-flow-manager | `NOJIRA-typed-rpc-pr4-flow` |
| 1 | PR5 | bin-conference-manager | `NOJIRA-typed-rpc-pr5-conference` |
| 1 | PR6 | bin-conversation-manager | `NOJIRA-typed-rpc-pr6-conversation` |
| 1 | PR7 | bin-customer-manager | `NOJIRA-typed-rpc-pr7-customer` |
| 1 | PR8 | bin-agent-manager | `NOJIRA-typed-rpc-pr8-agent` |
| 2 | PR9 | bin-number-manager | `NOJIRA-typed-rpc-pr9-number` |
| 2 | PR10 | bin-message-manager | `NOJIRA-typed-rpc-pr10-message` |
| 2 | PR11 | bin-email-manager | `NOJIRA-typed-rpc-pr11-email` |
| 2 | PR12 | bin-campaign-manager | `NOJIRA-typed-rpc-pr12-campaign` |
| 2 | PR13 | bin-outdial-manager | `NOJIRA-typed-rpc-pr13-outdial` |
| 2 | PR14 | bin-queue-manager | `NOJIRA-typed-rpc-pr14-queue` |
| 2 | PR15 | bin-ai-manager | `NOJIRA-typed-rpc-pr15-ai` |
| 3 | PR16 | bin-rag-manager | `NOJIRA-typed-rpc-pr16-rag` |
| 3 | PR17 | bin-tts-manager | `NOJIRA-typed-rpc-pr17-tts` |
| 3 | PR18 | bin-transcribe-manager | `NOJIRA-typed-rpc-pr18-transcribe` |
| 3 | PR19 | bin-storage-manager | `NOJIRA-typed-rpc-pr19-storage` |
| 3 | PR20 | bin-pipecat-manager | `NOJIRA-typed-rpc-pr20-pipecat` |
| 4 | PR21 | bin-webhook-manager | `NOJIRA-typed-rpc-pr21-webhook` |
| 4 | PR22 | bin-hook-manager | `NOJIRA-typed-rpc-pr22-hook` |
| 4 | PR23 | bin-timeline-manager | `NOJIRA-typed-rpc-pr23-timeline` |
| 4 | PR24 | bin-tag-manager | `NOJIRA-typed-rpc-pr24-tag` |
| 4 | PR25 | bin-contact-manager | `NOJIRA-typed-rpc-pr25-contact` |
| 4 | PR26 | bin-direct-manager | `NOJIRA-typed-rpc-pr26-direct` |
| 4 | PR27 | bin-talk-manager | `NOJIRA-typed-rpc-pr27-talk` |
| 4 | PR28 | bin-transfer-manager | `NOJIRA-typed-rpc-pr28-transfer` |
| 4 | PR29 | bin-route-manager | `NOJIRA-typed-rpc-pr29-route` |
| 4 | PR30 | bin-registrar-manager | `NOJIRA-typed-rpc-pr30-registrar` |

---

### Final PR — Remove translator section 4 `[bin-api-manager]`

After all in-scope managers are migrated AND api-validator error-envelope tests run green for at least one CI cycle:

- Delete section 4 from `error_translate.go`.
- Update `error_translate_test.go`: drop section-4 cases, beef up sections 1–3 + 5 if any gaps.
- Keep section 5 (default `INTERNAL`) as the safety net.

If a regression appears post-merge, revert is one PR.

**Branch:** `NOJIRA-typed-rpc-final-kill-section4`

---

## 6. Per-manager migration template (PR3..N)

For each manager:

1. **Scope** — `grep -rn "fmt.Errorf" pkg/ | grep -v _test.go` and classify each site:
   - User-visible (NOT_FOUND, INVALID_ARGUMENT, etc.) → migrate to typed.
   - Internal/transient → leave as `fmt.Errorf`.
2. **listenHandler** — copy the `errorResponse` helper from billing-manager. Replace `simpleResponse(4xx)` call sites where the error originates from a migrated business handler.
3. **Business handlers** — migrate classified sites to typed constructors. Use the manager's own `commonoutline.ServiceName*` for the `domain` field.
4. **Tests** — switch string-assertions to `errors.Is`/`errors.As`.
5. **api-validator** — add at least one scenario that asserts `domain=<service-name>` for one error path.
6. **Verify** — full workflow on the manager + bin-api-manager + relevant api-validator scenarios.
7. **Code review** — review-and-fix loop until clean (HIGH+ severity issues addressed).
8. **Squash merge** with explicit user authorization.

---

## 7. Risks & mitigations

| Risk | Mitigation |
|------|------------|
| PR0 changes `parseResponse` behavior subtly for callers using legacy sentinels on responses that *will be* migrated post-PR0 | Pre-implementation audit identified 3 sites; PR0 patches each with a parallel typed guard. Both branches stay until upstream is migrated. |
| Breaking changes in error-message text for clients matching on it | Out of scope. Clients should be using HTTP status code and envelope `reason`, not message strings. If api-validator catches text-matching, fix the test. |
| Accidentally migrating an internal error to a user-visible classification | Code-review checkpoint per PR. Heuristic: if an operator (not the user) needs to debug it, it's `INTERNAL`. |
| 28-service blast-radius PR0 verification fatigue | PR0 is a no-op for service code; verification is `go test`/lint passing, not "review every diff." |
| api-validator coverage gap masks a regression | Each per-manager PR adds at least one domain-asserting scenario. Stricter coverage tracked across the rollout. |
| Pre-existing `errors.Is(err, requesthandler.Err*)` callers in pending feature work (worktrees) regress when those features land post-PR0 | Worktree work is not in scope. Feature owners must update their callers using the same dual-guard pattern when their features merge. |

---

## 8. Rollback

Every PR is independently revertable:

- **PR0** — revert restores legacy-only path. Migrated managers' typed responses become canned sentinels again — degraded but functional.
- **PR1..N** — revert restores `fmt.Errorf` + `simpleResponse(4xx)` path. Translator's section 4 still catches it (until the final PR lands).
- **Final PR** — revert restores section 4.

No schema changes, no migrations, no irreversible state.

---

## 9. Verification strategy

- **PR0**: full monorepo verify (28+ services). Mechanical — `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` per service. Plus new unit tests in `bin-common-handler/pkg/requesthandler` covering the typed-passthrough and legacy-fallback paths. Plus `bin-call-manager` and `bin-api-manager` verification for the 3 patched callers.
- **PR1..N**: per-manager verify + bin-api-manager verify + api-validator scenarios for at least one error path with `domain=<service-name>` assertion.
- **Final PR**: bin-api-manager verify + full api-validator error-envelope suite green.

Code review (review-and-fix loop) runs before every merge — no shortcuts based on PR size.
