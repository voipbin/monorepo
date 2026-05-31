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

The `aipromptproposals` resource already exhibits two more latent instances of this bug, on **different** endpoints from the 404 (so they are the same *class*, not the same endpoints):

- **accept / reject** (the ID-scoped endpoints from this issue) return bare **409** (`simpleResponse(409)`, "proposal not completed") → `requesthandler.ErrConflict` → Default → **500** instead of 409. This is reachable for a *valid* proposal acted on in the wrong state (independent of the not-found defect).
- **create** (`POST /aipromptproposals`, a separate endpoint) returns bare **429** (`simpleResponse(429)`, "rate limit exceeded") → `requesthandler.ErrTooManyRequests` → Default → **500** instead of 429. Reached via the rate-limit path, not the not-found path.

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
| `ErrPaymentRequired` | 402 | `PaymentRequired` | `PAYMENT_REQUIRED` | `INSUFFICIENT_BALANCE` |
| `ErrForbidden` | 403 | `PermissionDenied` | `PERMISSION_DENIED` | `PERMISSION_DENIED` |
| **`ErrNotFound`** | **404** | **`NotFound`** | **`NOT_FOUND`** | **`RESOURCE_NOT_FOUND`** |
| `ErrConflict` | 409 | `FailedPrecondition` | `FAILED_PRECONDITION` | `STATE_INVALID` |
| `ErrTooManyRequests` | 429 | `ResourceExhausted` | `RESOURCE_EXHAUSTED` | `RATE_LIMIT_EXCEEDED` |
| `ErrServiceUnavailable` | 503 | `Unavailable` | `UNAVAILABLE` | `SERVICE_UNAVAILABLE` |
| `ErrInternal` | 500 | `Internal` | `INTERNAL` | `INTERNAL` |

Notes:
- **Reason/message wording reuses the existing typed-sentinel cases verbatim where one exists**, so the external envelope is identical whether the error originated as a typed sentinel or a bare backend status. Concretely, each bare-status case reuses the exact reason/message strings already present in `translateToVoipbinError` (`error_translate.go`): 404 → `RESOURCE_NOT_FOUND` ("The requested resource was not found."), 401 → `AUTHENTICATION_REQUIRED`, 403 → `PERMISSION_DENIED`, 402 → `INSUFFICIENT_BALANCE`, 409 → `STATE_INVALID`, 503 → `SERVICE_UNAVAILABLE`, 500 → `INTERNAL`. Only 429 introduces a new reason (`RATE_LIMIT_EXCEEDED`) because no existing case maps to `ResourceExhausted`.
- **402 reuses `INSUFFICIENT_BALANCE`** (not a new `PAYMENT_REQUIRED` reason) to stay consistent with the established 402 mapping at `error_translate.go:78-80` and its test (`error_translate_test.go:76`), so there is a single 402 envelope. No backend emits a bare 402 today, so this is a consistency choice with no live path.
- **409 → `FailedPrecondition`** rather than `AlreadyExists`. Both map to HTTP 409 via `HTTPStatusFor`, but the bare-409 emitters in practice (accept/reject "not completed") are state-precondition failures. This is already the documented contract: the `AlreadyExists` constructor doc (`constructors.go:40-45`) explicitly states "the api-manager translator never emits ALREADY_EXISTS as a fallback mapping — 409 responses default to FAILED_PRECONDITION." Reusing `STATE_INVALID` matches the existing `serviceerrors.ErrStateInvalid` case.
- **Ordering:** these sentinel cases must run **after** step 1 (typed passthrough) — which they do, since they live in step 2 — so a migrated backend's typed `VoipbinError` is never overridden by a coincidental status-code match.
- **`ErrInternal` (500) → `Internal`** yields the same outcome as the Default branch, so to avoid any behavioral divergence the explicit case **must `.Wrap(err)` exactly as the Default branch does** (`error_translate.go:92`). It is included only to document the full closed set and guard against a future Default-branch change; the implementer may instead omit it and rely on Default — either is acceptable as long as the wrap behavior is identical.

### Why the full set, not just 404

The bare statuses above are exactly the ones backends emit via `simpleResponse(...)`. Cherry-picking 404 would leave the identical-class 409 bug (accept/reject) and 429 bug (create) unfixed. A closed set mirroring `HTTPStatusFor` is more principled, self-documenting, and maintainable, and it remains fully consistent with the typed-error migration design (typed envelopes still win via step 1; the Default branch still degrades genuinely-unmatched errors to INTERNAL).

## Components touched

| File | Change |
|---|---|
| `bin-api-manager/server/error_translate.go` | Add 8 sentinel cases (7 new + the existing 400) in step 2 of `translateToVoipbinError`. |
| `bin-api-manager/server/error_translate_test.go` | Add a table-driven test over every `requesthandler.Err*` sentinel → expected `Status`/reason, plus a `pkg/errors`-wrapped variant for the 404 case (mirrors servicehandler's `errors.Wrapf`). |
| `bin-api-manager/server/error_test.go` | Add one edge-level test driving `abortWithServiceError` through a real `gin` recorder for a wrapped `requesthandler.ErrNotFound`, asserting `w.Code == 404` — covers the full status round-trip via `HTTPStatusFor`, not just the translator's `Status` field. |

No backend (`bin-ai-manager`) changes. No `bin-common-handler` changes.

## Error handling / edge cases

- **Wrapped errors:** servicehandler wraps RPC errors with `pkg/errors.Wrapf`. `stderrors.Is` walks the chain (pkg/errors v0.9+ implements `Unwrap`), so the sentinel match works through the wrap. The test suite asserts this explicitly.
- **Typed precedence preserved:** step 1 (`errors.As(&ve)`) runs first; only bare-status sentinels (no typed body) reach step 2's new cases.
- **Unmatched statuses:** any status not in the table (e.g. 405, 410, 422) still falls through to Default → INTERNAL, unchanged. This is acceptable — those are not emitted by current backends; they can be added if/when a backend starts emitting them.
- **Bare 404 is also emitted by the no-handler-found default route**, not only by resource-not-found. ai-manager's `processRequest` returns a bare `simpleResponse(404)` for an unmatched RPC route (`pkg/listenhandler/main.go:507`). After this change, a missing RPC route surfaces to the client as `404 NOT_FOUND` instead of `500 INTERNAL`. This is an accepted trade-off: a missing route is a deploy/wiring bug that should be caught before production, and 404-vs-500 for it is a minor cosmetic difference. Reviewers should be aware the blast radius of the 404 mapping is not strictly limited to resource-not-found.
- **No existing edge test asserts bare-status→500.** Verified: the only 500-asserting edge tests (`error_test.go:84`, `:116`) use a `nil` error and a generic `fmt.Errorf` — neither is a `requesthandler.Err*` sentinel, so neither changes behavior. All `serviceerrors.ErrNotFound` test sites use the typed-sentinel path (step-2 `serviceerrors.ErrNotFound`, unchanged).
- **Panic safety:** unchanged — the existing `defer recover()` still degrades to INTERNAL.

## Testing strategy

1. **Unit (translator):** extend `error_translate_test.go`:
   - Table: each `requesthandler.Err{BadRequest,Unauthorized,PaymentRequired,Forbidden,NotFound,Conflict,TooManyRequests,ServiceUnavailable,Internal}` → expected `cerrors.Status` and reason (reasons per the mapping table above).
   - `pkg/errors.Wrapf(requesthandler.ErrNotFound, "...")` → `StatusNotFound` (production wrapping path).
   - Confirm an unmatched status string still yields `StatusInternal`.
2. **Edge (HTTP status round-trip):** extend `error_test.go` — drive `abortWithServiceError(c, pkgerrors.Wrapf(requesthandler.ErrNotFound, "..."))` through a `httptest`/`gin` recorder and assert `w.Code == 404`. This guards the boundary the client actually observes (`translateToVoipbinError` → `abortWithError` → `HTTPStatusFor`), which the translator-unit test alone does not cover.
3. **Verification workflow** in `bin-api-manager`: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`.
4. **api-validator follow-up (separate, per repo convention):** un-`xfail` the 4 affected tests in `monorepo-monitoring/api-validator` once the fix is deployed. Tracked as a follow-up, not part of this monorepo PR.

## Out of scope

- The "proposal not found" → 404 branches in ai-manager's accept/reject listenhandlers. These are **redundant for the common api-manager path** (the api-manager permission-check GET fails first), but they are **not dead code**: they still fire for a TOCTOU race (proposal deleted between the GET and the accept/reject RPC) or any direct/non-api-manager RPC caller. Left as-is — do **not** treat them as removable.
- Migrating ai-manager's `errorResponse` to emit typed `NotFound` envelopes. The central fix makes this unnecessary for correctness; it can be a future cleanup.
- Statuses not currently emitted by backends (405/410/422/etc.).

## Rollback

Single-file behavioral change with no schema/state impact. Revert the PR commit to restore prior behavior. Low risk: strictly turns previously-500 responses into their correct 4xx/503 codes.
