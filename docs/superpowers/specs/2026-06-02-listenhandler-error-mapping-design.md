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

The bug is in the **backend manager listenhandler endpoint layer**, not the edge.
The codebase is **partially migrated in conversation-manager only** (its PUT maps
errors correctly while its GET still swallows); message-manager, number-manager,
and call-manager have no migrated by-id endpoints. See the per-service baseline
table below.

1. The **handler `Get` methods already emit typed not-found errors.** For all
   four issue services, `xHandler.Get(...)` converts `dbhandler.ErrNotFound`
   into a typed `cerrors.NotFound(...)` VoipbinError. Example
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

   **State-changing handler methods (`Delete`/`Update`/`Hangingup`) do NOT emit
   typed not-found.** Verified: `conversationHandler.Update`,
   `messageHandler.Delete`, `numberHandler.Delete/Update/UpdateMetadata`,
   `groupcallHandler.Delete/Hangup/Hangingup` either return the raw db error or
   `errors.Wrap(err, ...)` (pkg/errors, which preserves `Unwrap`). For these,
   not-found correctness depends on the raw `dbhandler.ErrNotFound` **sentinel
   surviving the wrap chain** and being matched by `errorResponse`'s sentinel
   branch (`stderrors.Is`).

2. Some **listenhandler endpoint functions still swallow every error** with
   `return simpleResponse(500), nil`, discarding both the typed not-found and
   the raw sentinel. Confirmed still-swallowing for the issue endpoints, e.g.
   `bin-conversation-manager/pkg/listenhandler/v1_conversations.go`
   (`processV1ConversationsIDGet`):

   ```go
   tmp, err := h.conversationHandler.Get(ctx, id)
   if err != nil {
       return simpleResponse(500), nil   // ← discards typed not-found
   }
   ```

   Meanwhile the PUT sibling (`processV1ConversationsIDPut`,
   `v1_conversations.go:211`) already uses `return errorResponse(err), nil` — the
   exact convention this spec recommends for new sites (map at the call site),
   **not** bare `(nil, err)` + central tail. So the one truly-migrated endpoint
   already validates the recommended mechanism. Phase 1 must **enumerate the
   specific still-swallowing sites per service**, not assume a uniform starting
   point.

   **Per-service baseline (verified):**

   | Service | Migrated by-id sites | Still-swallowing | Central tail |
   |---|---|---|---|
   | conversation-manager | PUT (`errorResponse`-at-site) | GET + others (`simpleResponse(500)`) | wired (backstop) |
   | message-manager | none | all by-id (`simpleResponse(500)`) | wired (backstop) |
   | number-manager | none | all by-id (`simpleResponse(500)`) | wired (backstop) |
   | call-manager (groupcalls) | none | all by-id (`simpleResponse(500)`) | **unwired** (bare `simpleResponse(400)`) |

   Do not look for already-correct GET/PUT siblings in message/number — there are
   none.

3. The **shared `errorResponse` helper exists in all four issue services**
   (identical to flow-manager's). It maps:
   `*cerrors.VoipbinError` → `cerrors.ToResponse` (true status);
   `dbhandler.ErrNotFound` → 404; everything else → **500**.

4. **Central dispatch tails differ:**

   | Service | Central tail routes `VoipbinError`+`ErrNotFound`→`errorResponse`? | `errorResponse` helper exists? | Tail default |
   |---|---|---|---|
   | conversation-manager | yes | yes | `simpleResponse(400)` |
   | message-manager | yes | yes | `simpleResponse(400)` |
   | number-manager | yes | yes | `simpleResponse(400)` |
   | **call-manager** (groupcalls) | **no** (bare `simpleResponse(400)`) | **yes (unwired)** | `simpleResponse(400)` |
   | flow-manager (reference) | yes | yes | `simpleResponse(400)` |

   Note: call-manager's gap is that its tail **does not call the existing
   `errorResponse` helper** — not that the helper is absent.

The api-manager edge translator (`server/error_translate.go`, fixed in #954)
already converts typed VoipbinError and bare `requesthandler.Err*` statuses to
the correct client HTTP code. **No edge change is required** — the information
is destroyed at the backend before it ever reaches the edge.

## Scope

- **Breadth:** All four issue managers **plus a monorepo-wide audit** of all 37
  services for the same swallow pattern.
- **Error classes:** **Full typed-error fidelity** — wherever a handler returns
  a typed `cerrors.VoipbinError` (404/403/409/429/503/etc.), or a raw
  `dbhandler.ErrNotFound` sentinel, stop swallowing it so the client receives
  the true status. Genuine internal errors (response marshal failures, untyped
  DB/connection errors) correctly resolve to `500`. Request-parse / body-
  unmarshal failures correctly resolve to `400`.

## Design — Canonical Listenhandler Error Convention

The shared `errorResponse(err)` helper is the single canonical mapper. Reference
(flow-manager, present verbatim in all four issue services):

```go
func errorResponse(err error) *sock.Response {
    if err == nil { return simpleResponse(http.StatusInternalServerError) }
    var ve *cerrors.VoipbinError
    if stderrors.As(err, &ve) {
        resp, e := cerrors.ToResponse(ve)
        if e == nil { return resp }
        return simpleResponse(http.StatusInternalServerError)
    }
    if stderrors.Is(err, dbhandler.ErrNotFound) {   // matches through errors.Wrap
        return simpleResponse(http.StatusNotFound)
    }
    return simpleResponse(http.StatusInternalServerError)   // internal → 500
}
```

Every `bin-*-manager` listenhandler converges on:

1. **Handler-call errors are mapped at the call site** with the shared helper:
   replace `return simpleResponse(500), nil` with `return errorResponse(err), nil`.
   This yields not-found→404 (typed *or* surviving sentinel), other typed
   codes→true status, and genuine internal errors→500 — with **no change to the
   central tail and no regression of parse-class errors**.
2. **Request-parse failures stay local & client-class:** bad URI split and
   request-body `json.Unmarshal` failures keep `return simpleResponse(400), nil`.
   Response `json.Marshal` failures keep `simpleResponse(500)`.
3. **Central dispatch tail is a defensive backstop, left unchanged** (default
   `simpleResponse(400)` in all four services). Because every handler-call error
   is mapped at the site, the tail is not relied upon for handler errors — so
   **call-manager's unwired tail does not need to be wired for the #955 fix**.
   Wiring it (route `*cerrors.VoipbinError` + `dbhandler.ErrNotFound` through the
   existing `errorResponse` helper) is deferred to optional separate work; see
   the call-manager note in Phase 1.

### Why call the helper at the site (not bare `(nil, err)` + central tail)

Returning bare `(nil, err)` defers mapping to the central tail, whose **default
is `400`**. A propagated *genuine internal* handler error (e.g. a DB connection
failure on a GET) would then become `400` — a server fault mislabeled as a
client error, and a regression from today's `500`. Calling `errorResponse(err)`
at the site maps internal→`500` correctly. This also avoids the central-default
flip considered and rejected below, and matches the de-facto pattern of the one
already-migrated endpoint (conversation PUT, `v1_conversations.go:211`).

### Rejected: flipping the central default 400→500

Flipping the central tail default to `500` would regress legitimate client-class
errors. Multiple endpoints return bare `(nil, err)` for request-body parse
failures (e.g. `requesthandler.GetFilteredItems` JSON unmarshal, field-type
conversion) that resolve to `400` via the default branch today; a `500` default
silently turns each into a server error. It also contradicts the flow-manager
reference, whose default is `400`. Therefore the central default is **kept at
400**, and correct internal→500 mapping is achieved at the call site via
`errorResponse`.

### Rejected: edge-only (api-manager)

Not viable — the backend emits a bare HTTP 500 with no surviving "not found"
signal; the edge cannot reconstruct it.

## Not-Found Path Verification (per state-changing method)

Because `Delete`/`Update`/`Hangingup` do not emit typed not-found, each must be
verified to surface the raw `dbhandler.ErrNotFound` sentinel (preserved through
`errors.Wrap`) so `errorResponse` maps it to 404:

- **GET (all four):** `Get` emits typed `cerrors.NotFound` → 404 directly. ✔
- **conversation PUT (`Update`):** confirm the update path performs a get/lookup
  that returns `ErrNotFound` for a missing id (not a silent 0-row update).
- **message DELETE, number DELETE/UPDATE:** `numberHandler.Delete` re-fetches via
  `Get` first → ErrNotFound naturally; confirm `messageHandler.Delete` surfaces
  the sentinel rather than returning success on 0 rows.
- **groupcall DELETE:** returns the already-deleted record with `200` when
  `TMDelete != nil` (soft-delete); for a never-existed id, confirm it surfaces
  ErrNotFound.
- **groupcall `/hangup` (`Hangingup`) — highest risk:** path is
  `Hangingup → UpdateStatus → GroupcallSetStatus` (SQL UPDATE, which returns nil
  on 0 rows) then `GroupcallGet` (returns wrapped `ErrNotFound`). Must verify the
  re-fetch actually runs and the wrapped sentinel propagates; otherwise hangup of
  a non-existent groupcall returns `200`/`500` instead of `404`. If the path does
  not re-fetch, add an explicit existence check.

## Delete / Soft-Delete Semantics (canonical rule)

- **Never-existed id → 404.** This is what #955 and api-validator assert, and the
  only behavior this program guarantees for DELETE.
- **Already-deleted id → per-service, must not regress.** Behavior varies today:
  `groupcallHandler.Delete` has an explicit `if gc.TMDelete != nil { return gc }`
  idempotency short-circuit (→ 200); `messageHandler.Delete` / `numberHandler.Delete`
  have no such short-circuit and currently return 200 with a re-deleted record as
  a side effect. This program does **not** unify that behavior — it only requires
  that the de-swallow not change the already-deleted response. Unifying
  idempotent-delete semantics is optional follow-up.
- Each affected DELETE endpoint is checked so a **never-existed** id returns 404,
  and the corresponding api-validator assertions are confirmed to encode that.

## Phasing

Each phase is its own spec → plan → implement → PR cycle. Dependency is linear:
**Phase 0 → Phase 1 → Phase 2 → Phase 3+**.

### Phase 0 — Establish the convention (docs, no behavior change)
- Write the canonical convention to
  `docs/conventions/listenhandler-error-mapping.md`, referencing flow-manager as
  the template and `errorResponse` as the canonical mapper. Single source of
  truth for later phases.

### Phase 1 — The 4 issue managers (closes #955)
Full typed-error fidelity for all by-ID and state-changing endpoints (not only
the 9 named):
- **conversation / message / number:** enumerate still-swallowing sites; replace
  `return simpleResponse(500), nil` with `return errorResponse(err), nil`; verify
  each state-changing method's not-found path (section above); add/extend
  `error_response_test.go`.
- **call-manager:** apply `errorResponse`-at-site to the groupcall by-id
  endpoints (GET/DELETE/hangup and any by-id siblings) exactly like the other
  three; verify the `Hangingup` 0-row path. **Do NOT wire the central tail in
  this PR** — the tail is currently bare `simpleResponse(400)`, and routing
  `dbhandler.ErrNotFound` through it would silently flip ~30 unrelated by-id call
  endpoints (Hold/Mute/MOH/Play/Silence/etc., which return bare
  `(nil, errors.Wrap(...))`) from 400→404 with no enumeration or test coverage.
  Tail-wiring for call-manager is tracked as optional separate work, with its own
  enumeration + tests, so the groupcall fix stays a reviewable, revertible diff.
- Re-run api-validator against the 9 endpoints → green.
- **One PR per manager** (4 PRs) for reviewable, revertible diffs.

### Phase 2 — Monorepo audit (discovery, no code change)
- Sweep all 37 services. For each listenhandler endpoint, classify every
  `simpleResponse(500)` / swallow as (a) swallowed handler error → **fix**, or
  (b) legitimate local internal/parse error → **leave**.
- **Classify cache-miss error types per service:** services that read from Redis
  first (e.g. call-manager) may surface not-found as a representation other than
  the MySQL `dbhandler.ErrNotFound` sentinel; the `ErrNotFound`→404 net assumes
  the DB sentinel reaches the mapper. Record any cache-path not-found types that
  need explicit handling.
- Output: tracking checklist in `docs/plans/` (per service: tail wired?,
  endpoints to fix, handler emits typed not-found vs raw sentinel?, cache-miss
  error type).

### Phase 3..N — Remaining services (batched)
- All services get the same at-site treatment (site changes + tests). Group only
  by incidental effort; for tail-unwired services, wiring the `errorResponse`
  backstop into the tail is optional and, where it would silently reclassify
  unrelated bare-`(nil, err)` endpoints, must be its own enumerated+tested change.
- One PR per service (or small batch). Each independently verifiable.
- Services whose handler layer returns only the *raw* `dbhandler.ErrNotFound`
  sentinel are still fixed via `errorResponse`'s sentinel branch; upgrading them
  to typed `cerrors.NotFound` is optional follow-up, not required for the 404 fix.

## Per-Service Procedure (repeatable unit)

1. **Tail check (informational, not required for the fix)** — note whether
   `main.go`'s `if err != nil` tail routes `*cerrors.VoipbinError` and
   `dbhandler.ErrNotFound` through `errorResponse`. Because every handler-call
   error is mapped at the site (step 3), the fix does not depend on the tail. Do
   **not** wire an unwired tail as part of a de-swallow PR if doing so would
   silently reclassify unrelated bare-`(nil, err)` endpoints (e.g. call-manager);
   track tail-wiring as separate work with its own enumeration + tests.
   (`errorResponse` already exists in all four issue services.)
2. **Enumerate swallow sites** — grep `simpleResponse(500)` in the listenhandler
   endpoint files; for each, decide handler-call error (→ fix) vs response-marshal
   failure (→ leave as 500).
3. **De-swallow** — replace handler-call `return simpleResponse(500), nil` with
   `return errorResponse(err), nil`. Leave parse→`400` and marshal→`500` intact.
4. **Not-found path verification** — for each state-changing method, confirm the
   raw `dbhandler.ErrNotFound` sentinel (or typed `cerrors.NotFound`) reaches
   `errorResponse`; add an existence check where a 0-row update would otherwise
   succeed silently (e.g. groupcall `Hangingup`).
5. **Tests** — two layers:
   - *Helper-level* (`error_response_test.go`, already exists): asserts
     `errorResponse` maps typed VoipbinError (404/403/409/429/503), raw
     `dbhandler.ErrNotFound` (through `errors.Wrap`), internal→500, and nil. Add
     any missing codes here once — this is where full-fidelity mapping is proven.
   - *Per-endpoint* (`*_test.go`): assert only the codes the endpoint's handler
     **actually emits**. For the #955 endpoints that is not-found→404 (and, e.g.,
     conversation PUT's `InvalidArgument`→400 for a bad agent id). Do NOT add
     synthetic 409/403/429/503 per-endpoint tests for paths that never produce
     those codes.
6. **Verify** — in the service dir:
   `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`.

## Testing & Verification Story

- **Unit (per service):** helper-level `error_response_test.go` proves the full
  code matrix (404/403/409/429/503/500/400/nil) once; per-endpoint listenhandler
  tests with a mocked handler assert only the codes that endpoint emits
  (primarily not-found→404, plus the parse→400 / marshal→500 locals). This is the
  regression guard that would have caught #955. Avoid synthetic per-endpoint
  tests for codes a path never produces.
- **Edge integration:** translator path covered by #954; add one Phase-1 sanity
  test asserting a backend 404 round-trips to client 404.
- **End-to-end:** api-validator (`monorepo-monitoring/api-validator/`) already
  exercises the 9 endpoints every 6h; Phase 1 turns them green. Add matching
  read-only api-validator tests for any newly touched endpoints (no
  cost-sensitive operations per repo policy).
- **Per-PR DoD:** 5-step verification workflow passes; listenhandler tests assert
  the new status codes; PR body lists each manager with `bin-<service>:` prefixes.

## Definition of Done (program)

- All four issue managers return 404 for non-existent UUIDs (GET and
  state-changing verbs) and true status for other typed errors; api-validator's 9
  tests pass.
- Convention documented in `docs/conventions/`.
- Monorepo audit checklist produced (including cache-miss classification);
  remaining services fixed in batched PRs.
- Every PR passes the mandatory 5-step verification workflow.
