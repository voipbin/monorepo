# Listenhandler Error Mapping — Not-Found→500 Fix and Full Typed-Error Fidelity

**Date:** 2026-06-02
**Issue:** [#955](https://github.com/voipbin/monorepo/issues/955)
**Branch:** `NOJIRA-Fix-listenhandler-error-mapping`

## Problem

Several resource endpoints return `500 INTERNAL` when a valid-format but
non-existent UUID is passed in the path (e.g.
`GET /v1.0/conversations/00000000-0000-0000-0000-000000000099`). The correct
response is `404 NOT_FOUND`. UUID *format* validation already works (malformed
strings return `400`).

Endpoints named in #955 (9 total):

| Endpoint | Method | Expected | Actual |
|---|---|---|---|
| `/conversations/{id}` | GET | 404 | 500 |
| `/conversations/{id}` | PUT | 404 | 500 |
| `/groupcalls/{id}` | GET | 404 | 500 |
| `/groupcalls/{id}` | DELETE | 404 | 500 |
| `/groupcalls/{id}/hangup` | POST | 404 | 500 |
| `/messages/{id}` | GET | 404 | 500 |
| `/messages/{id}` | DELETE | 404 | 500 |
| `/numbers/{id}` | GET | 404 | 500 |
| `/numbers/{id}` | DELETE | 404 | 500 |

These 9 are a **subset** of a widespread pattern; the same swallow exists across
many more endpoints in these and other managers.

## Root Cause

The bug is in the **backend manager listenhandler dispatch layer**, not the edge.

1. The **handler layers already emit typed errors.** For all four issue
   services, `xHandler.Get(...)` converts `dbhandler.ErrNotFound` into a typed
   `cerrors.NotFound(...)` VoipbinError. Example
   (`bin-conversation-manager/pkg/conversationhandler`):

   ```go
   func (h *conversationHandler) Get(ctx, id) (*conversation.Conversation, error) {
       res, err := h.db.ConversationGet(ctx, id)
       if err != nil {
           if stderrors.Is(err, dbhandler.ErrNotFound) {
               return nil, cerrors.NotFound(
                   commonoutline.ServiceNameConversationManager,
                   "CONVERSATION_NOT_FOUND", "The conversation was not found.").Wrap(err)
           }
           return nil, err
       }
       return res, nil
   }
   ```

2. The **listenhandler endpoint functions swallow every error** before it can
   reach the dispatcher
   (`bin-conversation-manager/pkg/listenhandler/v1_conversations.go`):

   ```go
   tmp, err := h.conversationHandler.Get(ctx, id)
   if err != nil {
       log.Debugf("...err: %v", err)
       return simpleResponse(500), nil   // ← discards the typed not-found
   }
   ```

3. The **central dispatch tail already routes errors correctly** in 3 of the 4
   issue services. Only `call-manager` still uses the legacy
   `simpleResponse(400)` tail with no `errorResponse` helper.

| Service | Central tail maps `VoipbinError`/`ErrNotFound`→404? | `errorResponse` helper |
|---|---|---|
| conversation-manager | yes | present |
| message-manager | yes | present |
| number-manager | yes | present |
| **call-manager** (groupcalls) | no (legacy `simpleResponse(400)`) | none |
| flow-manager (reference) | yes | present |

The api-manager edge translator (`server/error_translate.go`, fixed in #954)
already converts typed VoipbinError and bare `requesthandler.Err*` statuses to
the correct client HTTP code. **No edge change is required** — the information
is destroyed at the backend before it ever reaches the edge.

## Scope

- **Breadth:** All four issue managers **plus a monorepo-wide audit** of all 37
  services for the same swallow pattern.
- **Error classes:** **Full typed-error fidelity** — wherever a handler returns
  a typed `cerrors.VoipbinError` (404/403/409/429/etc.), stop swallowing it so
  the client receives the true status. Genuine internal errors (response
  marshal failures, untyped DB errors) correctly remain `500`. Request-parse /
  body-unmarshal failures correctly remain `400`.

## Design — Canonical Listenhandler Error Convention

Every `bin-*-manager` listenhandler converges on:

1. **Endpoint functions propagate handler errors.** Replace
   `return simpleResponse(500), nil` (where the swallowed value is a
   *handler-layer call* error) with `return nil, err`.
2. **Request-parse failures stay local & client-class.** Bad URI split and
   request-body `json.Unmarshal` failures keep `return simpleResponse(400), nil`.
   Response `json.Marshal` failures keep `simpleResponse(500)`. These are *not*
   propagated.
3. **Central dispatch tail** (in `main.go`) maps the propagated error:
   - `*cerrors.VoipbinError` → `errorResponse(err)` (→ `cerrors.ToResponse`, true status)
   - `dbhandler.ErrNotFound` → 404 (defensive net for handlers still returning the raw sentinel)
   - **default → 500** (handler-call errors are server-side; a deliberate flip
     from the legacy `400` default present in several services)
4. **`errorResponse` helper** present in every service (port flow-manager's
   verbatim where missing).

Reference `errorResponse` (flow-manager):

```go
func errorResponse(err error) *sock.Response {
    if err == nil { return simpleResponse(http.StatusInternalServerError) }
    var ve *cerrors.VoipbinError
    if stderrors.As(err, &ve) {
        resp, e := cerrors.ToResponse(ve)
        if e == nil { return resp }
        return simpleResponse(http.StatusInternalServerError)
    }
    if stderrors.Is(err, dbhandler.ErrNotFound) {
        return simpleResponse(http.StatusNotFound)
    }
    return simpleResponse(http.StatusInternalServerError)
}
```

### Key correctness note — central-default flip (400→500)

In conversation/message/number the central default is currently `400`. Handler-
call errors are server-side and must default to `500`, so the default is flipped
to `simpleResponse(500)`. Before flipping, each service is audited to confirm no
endpoint relies on the legacy `400` default by propagating a *client-class*
error as bare `(nil, err)` (parse failures should already be local `400`). Any
such case is made local before the flip.

### Rejected approaches

- **Local `errorResponse(err)` at each swallow site** (no central-tail change):
  works and keeps internal at 500, but scatters mapping calls across every call
  site and diverges from the central-tail convention used by 4 of 5 reference
  services.
- **Edge-only (api-manager):** not viable — the backend emits a bare HTTP 500
  with no surviving "not found" signal; the edge cannot reconstruct it.

## Phasing

Each phase is its own spec → plan → implement → PR cycle. Dependency is linear:
**Phase 0 → Phase 1 → Phase 2 → Phase 3+**.

### Phase 0 — Establish the convention (docs, no behavior change)
- Write the canonical convention to `docs/conventions/listenhandler-error-mapping.md`,
  referencing flow-manager as the template. Single source of truth for later phases.

### Phase 1 — The 4 issue managers (closes #955)
Full typed-error fidelity for all by-ID and state-changing endpoints (not only
the 9 named):
- **conversation / message / number:** central infra present → de-swallow,
  propagate `(nil, err)`, flip central default 400→500 after per-service audit,
  add/extend `error_response_test.go`.
- **call-manager:** additionally port the `errorResponse` helper and upgrade the
  central dispatch tail before propagating.
- Re-run api-validator against the 9 endpoints → green.
- **One PR per manager** (4 PRs) for reviewable, revertible diffs.

### Phase 2 — Monorepo audit (discovery, no code change)
- Sweep all 37 services. For each listenhandler, classify every
  `simpleResponse(500)` / swallow as (a) swallowed handler error → **fix**, or
  (b) legitimate local internal/parse error → **leave**.
- Output: tracking checklist in `docs/plans/` (per service: infra present?,
  endpoints to fix, handler emits typed errors?).

### Phase 3..N — Remaining services (batched)
- Group by effort: **infra-present** (propagate + tests) vs **infra-missing**
  (port `errorResponse` + upgrade dispatch first).
- One PR per service (or small batch). Each independently verifiable.
- Services whose handler layer still returns *raw* `dbhandler.ErrNotFound` are
  caught by the central `ErrNotFound`→404 branch, so they are fixed without a
  forced handler-layer rewrite; upgrading them to typed errors is optional
  follow-up.

## Per-Service Procedure (repeatable unit)

1. **Infra check** — does `errorResponse` exist and does the `main.go` tail
   route `*cerrors.VoipbinError` and `dbhandler.ErrNotFound`? If missing, port
   flow-manager's helper and upgrade the tail.
2. **Default-flip audit** — grep endpoints returning bare `(nil, err)`; confirm
   none rely on the legacy `400` default for a client-class error. Make any such
   case local before flipping the default to 500.
3. **De-swallow** — replace handler-call `return simpleResponse(500), nil` with
   `return nil, err`. Leave parse→`400` and marshal→`500` untouched.
4. **Handler-layer spot check** — confirm `xHandler.Get/Delete/Update` emit
   typed `cerrors.NotFound` (or at least raw `dbhandler.ErrNotFound`).
5. **Tests** — extend `pkg/listenhandler/*_test.go` / `error_response_test.go`:
   not-found→404; (full fidelity) invalid-state→409, permission→403, etc.;
   malformed/parse→400; marshal failure→500.
6. **Verify** — in the service dir:
   `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`.

## Testing & Verification Story

- **Unit (per service):** listenhandler tests with a mocked handler returning
  typed errors, asserting `sock.Response.StatusCode`. This is the regression
  guard that would have caught #955.
- **Edge integration:** translator path covered by #954; add one Phase-1 sanity
  test asserting a backend 404 round-trips to client 404.
- **End-to-end:** api-validator
  (`monorepo-monitoring/api-validator/`) already exercises the 9 endpoints every
  6h; Phase 1 turns them green. Add matching read-only api-validator tests for
  any newly touched endpoints (no cost-sensitive operations).
- **Per-PR DoD:** 5-step verification workflow passes; listenhandler tests
  assert the new status codes; PR body lists each manager with `bin-<service>:`
  prefixes.

## Definition of Done (program)

- All four issue managers return 404 for non-existent UUIDs and true status for
  other typed errors; api-validator's 9 tests pass.
- Convention documented in `docs/conventions/`.
- Monorepo audit checklist produced; remaining services fixed in batched PRs.
- Every PR passes the mandatory 5-step verification workflow.
