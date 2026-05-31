# Design: Central translation of bare RPC status sentinels in bin-api-manager

**Issue:** [#953](https://github.com/voipbin/monorepo/issues/953) — GET/DELETE/accept/reject `/aipromptproposals/{id}` returns `500 INTERNAL` instead of `404 NOT_FOUND` for a nonexistent ID.

**Date:** 2026-06-01
**Branch:** `NOJIRA-Fix-api-manager-bare-status-error-translation`

---

## Problem

All four `aipromptproposals` ID-scoped endpoints (`GET`, `DELETE`, `POST .../accept`, `POST .../reject`) return `500 INTERNAL` instead of `404 NOT_FOUND` when the proposal ID does not exist.

The issue's symptom and expected behavior (404) are correct, but its stated root cause ("the backend does not handle the not-found case gracefully") is **wrong**. The backend handles not-found correctly. The defect is a single gap in the api-manager error translator.

## Root-cause analysis

For a nonexistent ID, every one of these endpoints flows through `serviceHandler.aipromptproposalGet` → RPC `GET /v1/aipromptproposals/<id>`. (accept/reject/delete each perform a permission-check `GET` **first**, so they fail at the GET step before their own RPC is sent.)

The not-found error then propagates:

1. **bin-ai-manager DB** (`aipromptproposalGetFromDB`, `pkg/dbhandler/aipromptproposal.go`) returns `dbhandler.ErrNotFound`. ✅ correct
2. **bin-ai-manager listenhandler** (`errorResponse`, `pkg/listenhandler/main.go:158`) maps `dbhandler.ErrNotFound` → `simpleResponse(404)` — a **bare 404 status code with no typed-error body**. ✅ returns 404
3. **requesthandler `parseResponse`** (`bin-common-handler/pkg/requesthandler/common.go:121`): no typed `VoipbinError` envelope is present (`DataType` is empty), so `getResponseStatusCodeError(404)` returns the sentinel `requesthandler.ErrNotFound` (from `HttpStatusErrorMap`). ✅ still "not found"
4. **bin-api-manager `translateToVoipbinError`** (`server/error_translate.go:36`): **the bug.** Step 2 has an explicit case for `requesthandler.ErrBadRequest` (bare 400) but **no case for `requesthandler.ErrNotFound` (bare 404)**, so the error falls through to the Default branch → `cerrors.Internal(...)` → **500**.

The smoking gun (`error_translate.go:65`): bare-400 was wired up, bare-404 was forgotten.

```go
case stderrors.Is(err, requesthandler.ErrBadRequest):   // 400 handled
    return cerrors.InvalidArgument(...)
// ← no requesthandler.ErrNotFound case → falls to Internal(500)
```

### This is a class of bugs, not a single endpoint

`translateToVoipbinError` is the **single central translator** for every api-manager service error. Any endpoint whose backend returns a bare status via `simpleResponse(...)` (no typed `VoipbinError` envelope) for a status the translator does not explicitly handle will incorrectly degrade to 500.

The same `aipromptproposals` endpoints already exhibit two more latent instances of this bug:

- **accept/reject** return bare **409** (`simpleResponse(409)`, "proposal not completed") → `requesthandler.ErrConflict` → Default → **500** instead of 409.
- **create** returns bare **429** (`simpleResponse(429)`, "rate limit exceeded") → `requesthandler.ErrTooManyRequests` → Default → **500** instead of 429.

## Goal / acceptance criteria

- `GET`, `DELETE`, `POST .../accept`, `POST .../reject` on a nonexistent `aipromptproposals` ID return `404 NOT_FOUND` with the standard api-manager not-found envelope.
- Bare backend statuses `401/402/403/404/409/429/503/500` are translated to the matching client HTTP status instead of collapsing to 500.
- Typed `VoipbinError` envelopes continue to take precedence (no behavior change for already-migrated backends).
- Regression tests cover each sentinel, including the `pkg/errors`-wrapped form used by servicehandler.
- Full verification workflow passes in `bin-api-manager`.

## Approach (chosen: central translator fix, full bare-status set)

Add cases to step 2 of `translateToVoipbinError`, immediately adjacent to the existing `requesthandler.ErrBadRequest` case, mapping each `requesthandler` HTTP-status sentinel to the canonical `cerrors` constructor. The mapping mirrors `cerrors.HTTPStatusFor` (`bin-common-handler/models/errors/rpc.go:54`, the single source of truth for Status→HTTP) in reverse, so a bare backend status round-trips back to the same client HTTP code.

| `requesthandler` sentinel | HTTP | → `cerrors` constructor | resulting `Status` | reason constant |
|---|---|---|---|---|
| `ErrBadRequest` | 400 | `InvalidArgument` | `INVALID_ARGUMENT` | *(already present — unchanged)* |
| `ErrUnauthorized` | 401 | `Unauthenticated` | `UNAUTHENTICATED` | `AUTHENTICATION_REQUIRED` |
| `ErrPaymentRequired` | 402 | `PaymentRequired` | `PAYMENT_REQUIRED` | `PAYMENT_REQUIRED` |
| `ErrForbidden` | 403 | `PermissionDenied` | `PERMISSION_DENIED` | `PERMISSION_DENIED` |
| **`ErrNotFound`** | **404** | **`NotFound`** | **`NOT_FOUND`** | **`RESOURCE_NOT_FOUND`** |
| `ErrConflict` | 409 | `FailedPrecondition` | `FAILED_PRECONDITION` | `STATE_INVALID` |
| `ErrTooManyRequests` | 429 | `ResourceExhausted` | `RESOURCE_EXHAUSTED` | `RATE_LIMIT_EXCEEDED` |
| `ErrServiceUnavailable` | 503 | `Unavailable` | `UNAVAILABLE` | `SERVICE_UNAVAILABLE` |
| `ErrInternal` | 500 | `Internal` | `INTERNAL` | `INTERNAL` |

Notes:
- **Reason/message wording** matches the existing typed-sentinel cases where one exists (e.g. `RESOURCE_NOT_FOUND` / "The requested resource was not found." reuses the exact strings from the `serviceerrors.ErrNotFound` case) so the external envelope is identical whether the not-found originated as a typed sentinel or a bare backend status.
- **409 → `FailedPrecondition`** rather than `AlreadyExists`. Both map to HTTP 409 via `HTTPStatusFor`, but the bare-409 emitters in practice (accept/reject "not completed") are state-precondition failures, and `FailedPrecondition` already has an established api-manager mapping (`STATE_INVALID`). `AlreadyExists` semantics are not represented by a bare status code today.
- **Ordering:** these sentinel cases must run **after** step 1 (typed passthrough) — which they do, since they live in step 2 — so a migrated backend's typed `VoipbinError` is never overridden by a coincidental status-code match.
- `ErrInternal` (500) → `Internal` is explicit and harmless (same outcome as the Default branch) but documents the full closed set and guards against a future Default-branch change.

### Why the full set, not just 404

The bare statuses above are exactly the ones backends emit via `simpleResponse(...)`. Cherry-picking 404 would leave the identical 409/429 bugs on the very same endpoints. A closed set mirroring `HTTPStatusFor` is more principled, self-documenting, and maintainable, and it remains fully consistent with the typed-error migration design (typed envelopes still win via step 1; the Default branch still degrades genuinely-unmatched errors to INTERNAL).

## Components touched

| File | Change |
|---|---|
| `bin-api-manager/server/error_translate.go` | Add 8 sentinel cases (7 new + the existing 400) in step 2 of `translateToVoipbinError`. |
| `bin-api-manager/server/error_translate_test.go` | Add a table-driven test over every `requesthandler.Err*` sentinel → expected `Status`, plus a `pkg/errors`-wrapped variant for the 404 case (mirrors servicehandler's `errors.Wrapf`). |

No backend (`bin-ai-manager`) changes. No `bin-common-handler` changes.

## Error handling / edge cases

- **Wrapped errors:** servicehandler wraps RPC errors with `pkg/errors.Wrapf`. `stderrors.Is` walks the chain (pkg/errors v0.9+ implements `Unwrap`), so the sentinel match works through the wrap. The test suite asserts this explicitly.
- **Typed precedence preserved:** step 1 (`errors.As(&ve)`) runs first; only bare-status sentinels (no typed body) reach step 2's new cases.
- **Unmatched statuses:** any status not in the table (e.g. 405, 410, 422) still falls through to Default → INTERNAL, unchanged. This is acceptable — those are not emitted by current backends; they can be added if/when a backend starts emitting them.
- **Panic safety:** unchanged — the existing `defer recover()` still degrades to INTERNAL.

## Testing strategy

1. **Unit (translator):** extend `error_translate_test.go`:
   - Table: each `requesthandler.Err{BadRequest,Unauthorized,PaymentRequired,Forbidden,NotFound,Conflict,TooManyRequests,ServiceUnavailable,Internal}` → expected `cerrors.Status` and reason.
   - `pkg/errors.Wrapf(requesthandler.ErrNotFound, "...")` → `StatusNotFound` (production wrapping path).
   - Confirm an unmatched status string still yields `StatusInternal`.
2. **Verification workflow** in `bin-api-manager`: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`.
3. **api-validator follow-up (separate, per repo convention):** un-`xfail` the 4 affected tests in `monorepo-monitoring/api-validator` once the fix is deployed. Tracked as a follow-up, not part of this monorepo PR.

## Out of scope

- The dead-code "proposal not found" → 404 branches in ai-manager's accept/reject listenhandlers (unreachable for the not-found case because the api-manager GET fails first). Harmless; left as-is.
- Migrating ai-manager's `errorResponse` to emit typed `NotFound` envelopes. The central fix makes this unnecessary for correctness; it can be a future cleanup.
- Statuses not currently emitted by backends (405/410/422/etc.).

## Rollback

Single-file behavioral change with no schema/state impact. Revert the PR commit to restore prior behavior. Low risk: strictly turns previously-500 responses into their correct 4xx/503 codes.
