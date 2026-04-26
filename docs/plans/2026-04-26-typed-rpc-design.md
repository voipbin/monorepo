# Typed RPC Error Envelope — Design

**Status:** Active
**Author:** Sungtae Kim
**Date:** 2026-04-26 (last updated 2026-04-26 with PR1+PR2 combination)
**Related:** Follows the 18-PR `bin-api-manager` structured-error-envelope migration. Eliminates the substring-fallback in the api-manager translator by making upstream managers emit typed errors over RPC.

---

## 1. Goal & non-goals

### Goal

Eliminate `bin-api-manager/server/error_translate.go` section 4 (substring fallback for legacy `fmt.Errorf` strings). After this work, every error reaching api-manager is one of:

1. A typed `*cerrors.VoipbinError` from an upstream manager (translator section 1: `errors.As` passthrough).
2. An api-manager-local sentinel from `pkg/serviceerrors` (translator section 2: `errors.Is` match).
3. A true transport failure: `context.Canceled` or `context.DeadlineExceeded` (translator section 3).
4. An unclassified internal error caught by the section-5 `INTERNAL` safety net.

Section 4 (substring matching against `http.StatusText`-derived strings) is removed.

### Non-goals

- **Replacing every `fmt.Errorf` in the codebase.** Only errors with a user-visible classification get migrated. Examples that get migrated: `NOT_FOUND`, `INVALID_ARGUMENT`, `FAILED_PRECONDITION`, `PAYMENT_REQUIRED`, `PERMISSION_DENIED`, `RESOURCE_EXHAUSTED`, `ALREADY_EXISTS`. Examples that stay as plain `fmt.Errorf` (and ride the default `INTERNAL` path): DB connection failures, JSON marshal errors, programmer-error invariant violations.
- **Manager-to-manager protocol changes** beyond what `cerrors.ToResponse`/`FromResponse` already do. Backward-compatible: a non-migrated client just ignores `DataTypeVoipbinError` and treats the response the same as before.
- **Touching api-manager-local error logic.** Sections 1–3 and 5 of the translator stay as-is.

---

## 2. Current state (PR0 confirmed it)

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
                          section 4 — substring-matches the canned sentinel string
```

**Key insight:** section 4 isn't matching upstream service text. It's matching `http.StatusText` strings synthesized inside `bin-common-handler/pkg/requesthandler/common.go` from status codes.

---

## 3. Target end-state

```
[manager business handler]
   return cerrors.NotFound(commonoutline.ServiceNameXManager, "RESOURCE_NOT_FOUND",
                           "Account not found.").WithRequestID(...)
       │
       ▼
[manager listenHandler]
   errorResponse(err)   ← detects typed VoipbinError, emits via cerrors.ToResponse
       │
       ▼
[requesthandler.parseResponse (PR0)]
   if ve := cerrors.FromResponse(resp); ve != nil { return ve }
       │
       ▼
[error_translate.go section 1]
   errors.As(err, &ve) → return ve     ← passthrough, no string matching
```

Section 4 deleted. Section 5 (default `INTERNAL`) catches the rare un-classified case.

---

## 4. PR0 audit findings (recap)

### 4.1 Legacy sentinel callers (3 production sites — fixed in PR0)

| File | Pattern | Mitigation in PR0 |
|------|---------|------------------|
| `bin-call-manager/pkg/channelhandler/hangup.go:46` | `errors.Cause(errHangup) == requesthandler.ErrNotFound` | Added parallel typed-error guard. |
| `bin-api-manager/pkg/servicehandler/provider.go` | `errors.Is(err, ErrUnprocessableEntity)` | Added typed-error pass-through. |
| `bin-api-manager/server/providers.go` | Same as above | Added typed-error pass-through. |

### 4.2 Manager scope — all 28 with `pkg/listenhandler/`

```
agent, ai, billing (pilot), call, campaign, conference, contact,
conversation, customer, direct, email, flow, message, number, outdial,
pipecat, queue, rag, registrar, route, storage, tag, talk, timeline,
transcribe, transfer, tts, webhook
```

Excluded: `dbscheme-manager`, `openapi-manager`, `sentinel-manager` (no listenhandler / not RPC entry points).

### 4.3 api-validator framework — ready

`monorepo-monitoring/api-validator/tests/helpers/assertions.py:assert_error_envelope()` accepts `domain="..."` for per-service assertions. No framework work needed.

---

## 5. Migration plan (29 PRs)

### PR0 — Wire unwrap path `[bin-common-handler + 3 callers]` ✅ MERGED 2026-04-26

`parseResponse` now returns `*VoipbinError` ahead of the legacy `HttpStatusErrorMap` path. Logs warn when `DataType=DataTypeVoipbinError` but unmarshal fails. Three production callers patched with parallel typed-error guards. Squash-merged at commit `9fb2ee8e`.

---

### PR1 — Pilot: bin-billing-manager (combined listenHandler + business-handler migration)

**Combined scope** (originally PR1 + PR2 in the v1 plan; combined to avoid the 404→500 regression that would result from migrating only the listenHandler). The pilot ships both layers in one PR so semantics are preserved end-to-end.

**Changes:**

1. **`bin-billing-manager/pkg/listenhandler/main.go`** — add `errorResponse(err)` helper:
   - If `errors.As(err, *cerrors.VoipbinError)` → return `cerrors.ToResponse(ve)`.
   - Else if `errors.Is(err, dbhandler.ErrNotFound)` → return `simpleResponse(404)` (preserves legacy not-found-via-DB-sentinel behavior).
   - Else → return `simpleResponse(500)`.

2. **`bin-billing-manager/pkg/listenhandler/v1_*.go`** — replace business-handler error sites:
   - `simpleResponse(404)` (the kludge that returned 404 for ANY error) → `errorResponse(err)` so the helper picks the right code.
   - JSON-marshal failures stay as `simpleResponse(500)` (programmer error, not user-visible).
   - Paddle-webhook handlers (`v1_hooks_paddle.go`, `v1_accounts_paddle.go`) are out of scope.

3. **Business handlers — typed errors for the user-visible `fmt.Errorf` sites:**
   - `accounthandler/account.go:97` — `invalid status: %s` → `cerrors.InvalidArgument`.
   - `accounthandler/balance.go:130` — `unsupported billing type` → `cerrors.InvalidArgument`.
   - `accounthandler/resource_limit.go:62` — `unknown plan type` → `cerrors.InvalidArgument`.
   - `accounthandler/resource_limit.go:105` — `unsupported resource type` → `cerrors.InvalidArgument`.
   - `billinghandler/billing.go:85` — `unsupported reference type` → `cerrors.InvalidArgument`.
   - `billinghandler/billing.go:158` — `unsupported billing reference type` → `cerrors.InvalidArgument`.
   - DB-failure-wrapping `fmt.Errorf` sites stay as plain errors (INTERNAL via default).

4. **Tests** — update string-asserts in `accounthandler/*_test.go`, `billinghandler/*_test.go`, `listenhandler/v1_*_test.go` to use `errors.Is` / `errors.As`.

5. **api-validator** — extend `monorepo-monitoring/api-validator/tests/scenarios/test_billing_e2e.py` (or `test_accesskey_errors.py` if billing is hard to trigger) with a scenario that asserts `domain="billing-manager"` for at least one user-visible error path.

**Verify:** bin-billing-manager + bin-api-manager.

**Branch:** `NOJIRA-typed-rpc-pr1-billing-pilot`

---

### PR2..N-1 — Replicate per remaining manager `[one PR per manager]`

Each follows the combined pilot pattern.

**Order** (priority by api-manager exposure × error-semantic complexity):

| Tier | PR | Manager | Branch |
|------|-----|---------|--------|
| 1 | PR2 | bin-call-manager | `NOJIRA-typed-rpc-pr2-call` |
| 1 | PR3 | bin-flow-manager | `NOJIRA-typed-rpc-pr3-flow` |
| 1 | PR4 | bin-conference-manager | `NOJIRA-typed-rpc-pr4-conference` |
| 1 | PR5 | bin-conversation-manager | `NOJIRA-typed-rpc-pr5-conversation` |
| 1 | PR6 | bin-customer-manager | `NOJIRA-typed-rpc-pr6-customer` |
| 1 | PR7 | bin-agent-manager | `NOJIRA-typed-rpc-pr7-agent` |
| 2 | PR8 | bin-number-manager | `NOJIRA-typed-rpc-pr8-number` |
| 2 | PR9 | bin-message-manager | `NOJIRA-typed-rpc-pr9-message` |
| 2 | PR10 | bin-email-manager | `NOJIRA-typed-rpc-pr10-email` |
| 2 | PR11 | bin-campaign-manager | `NOJIRA-typed-rpc-pr11-campaign` |
| 2 | PR12 | bin-outdial-manager | `NOJIRA-typed-rpc-pr12-outdial` |
| 2 | PR13 | bin-queue-manager | `NOJIRA-typed-rpc-pr13-queue` |
| 2 | PR14 | bin-ai-manager | `NOJIRA-typed-rpc-pr14-ai` |
| 3 | PR15 | bin-rag-manager | `NOJIRA-typed-rpc-pr15-rag` |
| 3 | PR16 | bin-tts-manager | `NOJIRA-typed-rpc-pr16-tts` |
| 3 | PR17 | bin-transcribe-manager | `NOJIRA-typed-rpc-pr17-transcribe` |
| 3 | PR18 | bin-storage-manager | `NOJIRA-typed-rpc-pr18-storage` |
| 3 | PR19 | bin-pipecat-manager | `NOJIRA-typed-rpc-pr19-pipecat` |
| 4 | PR20 | bin-webhook-manager | `NOJIRA-typed-rpc-pr20-webhook` |
| 4 | PR21 | bin-hook-manager | `NOJIRA-typed-rpc-pr21-hook` |
| 4 | PR22 | bin-timeline-manager | `NOJIRA-typed-rpc-pr22-timeline` |
| 4 | PR23 | bin-tag-manager | `NOJIRA-typed-rpc-pr23-tag` |
| 4 | PR24 | bin-contact-manager | `NOJIRA-typed-rpc-pr24-contact` |
| 4 | PR25 | bin-direct-manager | `NOJIRA-typed-rpc-pr25-direct` |
| 4 | PR26 | bin-talk-manager | `NOJIRA-typed-rpc-pr26-talk` |
| 4 | PR27 | bin-transfer-manager | `NOJIRA-typed-rpc-pr27-transfer` |
| 4 | PR28 | bin-route-manager | `NOJIRA-typed-rpc-pr28-route` |
| 4 | PR29 | bin-registrar-manager | `NOJIRA-typed-rpc-pr29-registrar` |

---

### Final PR (PR30) — Remove translator section 4 `[bin-api-manager]`

After all in-scope managers are migrated AND api-validator error-envelope tests run green for at least one CI cycle:

- Delete section 4 from `error_translate.go`.
- Update `error_translate_test.go`: drop section-4 cases.
- Keep section 5 (default `INTERNAL`) as the safety net.

**Branch:** `NOJIRA-typed-rpc-final-kill-section4`

---

## 6. Per-manager migration template (PR2..PR29)

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
| listenHandler-only migration would regress 404 → 500 | Combined pilot pattern: ship listenHandler integration + business-handler migration together. The `errorResponse` helper also detects legacy `dbhandler.ErrNotFound` to preserve not-found behavior for non-migrated DB code paths. |
| Breaking changes in error-message text for clients matching on it | Out of scope. Clients should be using HTTP status code and envelope `reason`. |
| Accidentally migrating an internal error to a user-visible classification | Code-review checkpoint per PR. Heuristic: if an operator (not the user) needs to debug it, it's `INTERNAL`. |
| api-validator coverage gap masks a regression | Each per-manager PR adds at least one domain-asserting scenario. |

---

## 8. Rollback

Every PR is independently revertable:

- **PR0** — revert restores legacy-only path. Migrated managers' typed responses become canned sentinels again — degraded but functional.
- **PR1..N-1** — revert restores `fmt.Errorf` + `simpleResponse(4xx)` path. Translator's section 4 still catches it (until the final PR lands).
- **Final PR** — revert restores section 4.

No schema changes, no migrations, no irreversible state.

---

## 9. Verification strategy

- **PR0**: full monorepo verify (28+ services). Mechanical — no service-side code changes.
- **PR1..N-1**: per-manager verify + bin-api-manager verify + api-validator scenarios for at least one error path with `domain=<service-name>` assertion.
- **Final PR**: bin-api-manager verify + full api-validator error-envelope suite green.

Code review (review-and-fix loop) runs before every merge — no shortcuts based on PR size.
