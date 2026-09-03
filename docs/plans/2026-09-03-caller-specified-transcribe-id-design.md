# Design: Caller-specified TranscribeID on start

## Revision note

Rev 2: incorporates round-1 architect review (B1-B5, N1-N5). Corrects
scope from 5 to 7 services, adds the second OpenAPI schema copy, decides
the recording-path collision behavior, adds TOCTOU handling with streaming
rollback, and fixes the UUID wire-format claim.

Rev 3: incorporates round-2 review (B6, B7, N6-N9). Fixes the internal
contradiction in the recording-path decision (one branch was unreachable
given `Start`'s pre-check, and the test the doc specified for it could not
pass), states explicitly that a caller-supplied id is single-use/
non-idempotent, extends the streaming rollback to a mid-loop leg failure,
and corrects a test-file mechanism description.

Rev 4: incorporates round-3 review (B10, N10, N11). Rev 3's fix for B6
overcorrected: reassigning `id` in `Start` before dispatch destroys the
"was this caller-supplied" signal, so `startRecording`'s `id != uuid.Nil`
check was unconditionally true and silently broke the existing
recording-idempotency contract for every no-id caller (including the
mechanical-fix-only callers in scope 5-7). Fixed by tracking
caller-supplied-ness as a separate boolean threaded alongside the resolved
id, not inferred from the resolved id's nilness. RST guidance corrected to
match.

Rev 5: incorporates the two non-blocking notes from the round-4 approval
(N12, N13) — drops a stale parenthetical in the Contract section that
described a now-impossible "id differs from request" case, and threads
`callerSpecifiedID` one step further into `Create` so a duplicate-key
collision on a *server-generated* id (not a caller error) maps to an
internal error instead of the caller-facing "use a different id" message.

Rev 6: incorporates round-5 review (B11). Rev 5's `Create` change broke
`startLive`, `Create`'s other call site, which has no way to supply the
new parameter without itself being threaded (Go methods don't capture a
caller's locals) — the same defect class as B10, recurring because N13's
fix was applied without re-deriving where else the signal needed to reach.
Fixed per the reviewer's recommended Option 2: `Create` and `startLive`
keep their original signatures; the duplicate-key mapping happens once,
inside `Start`, after dispatch, where `callerSpecifiedID` is already in
scope with no threading required. **Approved (round 6, approval #1 of 2).**

Rev 7: incorporates round-6 review (E1-E4), all editorial/documentation
corrections with no logic change — the reviewer's independent
write-once/in-scope/reachable re-derivation passed cleanly. Removes stale
mentions of `Create` changing (E3), clarifies that "unchanged by this
change" refers specifically to `callerSpecifiedID`, not to `startLive`'s
pre-existing `id` parameter addition (E4), and flags two things for
whoever implements the `Start` refactor: the post-dispatch snippet is
illustrative and requires actually hoisting error handling out of the
existing per-branch `switch` (E1), and a pre-existing `errors.Wrapf(nil,
...)` bug in the `default:` case (silently returns `(nil, nil)` for an
unsupported reference type) should be fixed while that code is being
touched anyway, not carried forward (E2).

## Goal

Allow the caller of `POST /v1/transcribes` (and `POST
/service_agents/transcribes`, and the internal `TranscribeV1TranscribeStart`
RPC) to optionally supply the `Transcribe.ID` to use for the new session,
end to end. If omitted (zero UUID), behavior is unchanged: server generates
a random UUID as it does today.

## Motivation

A caller that wants to bind an event-topic subscription
(`transcribe-manager.transcript.<transcribe-id>.#`) *before* the streaming
session starts producing events needs to know the ID in advance, otherwise
there is a race between "receive the Start response with the generated ID"
and "the first transcript/speech event is already published". Pre-declaring
the ID closes that race. (Wiring the actual subscribe-then-start flow is a
separate follow-up; this change only adds the capability to specify the ID.)

## Scope — 7 services

**Feature surface (design decisions live here):**

1. `bin-transcribe-manager` — `transcribehandler.Start`/`startLive`/
   `startRecording` (`Create` itself is unchanged — see the B11 correction
   below), `dbhandler.TranscribeCreate` (duplicate-key classification),
   listenhandler DTO + wiring.
2. `bin-common-handler` — `requesthandler.TranscribeV1TranscribeStart`
   interface + implementation + mock.
3. `bin-openapi-manager` — **both** copies of the create-transcribe request
   schema (`openapi/paths/transcribes/main.yaml` and
   `openapi/paths/service_agents/transcribes.yaml` — these are independent,
   not `$ref`-shared) + regenerated `gens/models/gen.go`.
4. `bin-api-manager` — `servicehandler.TranscribeStart`,
   `ServiceAgentTranscribeStart` (`serviceagent_transcribe.go`), the
   `ServiceHandler` interface (`pkg/servicehandler/main.go`) + its mock,
   `server/transcribes.go`, `server/service_agents_transcribes.go`,
   regenerated `gens/openapi_server/gen.go` and `gens/openapi_redoc/*`, RST
   docs.

**Mechanical compile-fix only (interface signature changed, no new
behavior, pass `uuid.Nil`):**

5. `bin-ai-manager` — 3 call sites: `pkg/summaryhandler/start.go:88,151,252`.
6. `bin-flow-manager` — 2 call sites: `pkg/activeflowhandler/actionhandle.go:676,728`.
7. `bin-conference-manager` — 1 call site: `pkg/conferencehandler/transcribe.go:33`.

All 7 land in **one PR** (single monorepo, and the `bin-common-handler`
interface change breaks every consumer's compile simultaneously — no
intermediate commit would build if split). Considered and rejected: adding
a second, additive RPC method (`TranscribeV1TranscribeStartWithID`) instead
of changing the existing signature, to shrink blast radius to 2 services.
Rejected because it leaves a permanent duplicate method for a one-time
migration cost, which doesn't match this repo's convention of extending an
existing signature (see `Create(ctx, id, ...)` already doing this).

## Contract

`POST /v1/transcribes` and `POST /service_agents/transcribes` requests gain
an optional field:

```json
{
  "id": "3b7e6f2c-...-uuid",  // optional; omitted/zero => server generates one
  "customer_id": "...",
  "reference_type": "call",
  "reference_id": "...",
  "language": "en-US",
  "direction": "both",
  "provider": "gcp"
}
```

OpenAPI: declare `id` as `type: string, format: uuid` — **not** `x-go-type:
string` — so oapi-codegen emits a typed UUID (precedent:
`GetTranscribesParams.ReferenceId` in the same spec), giving free format
validation at the framework's request-binding layer instead of silent
`uuid.FromStringOrNil` coercion. A malformed `id` becomes a 400 from the
generated binder, not a silently-ignored "server generates one" (this was
wrong in rev 1 — the existing `reference_id`/`on_end_flow_id` fields on this
same endpoint use the `x-go-type: string` + `uuid.FromStringOrNil` idiom,
which we are deliberately *not* following for this field, precisely because
silent coercion defeats the pre-declared-ID use case).

Accepted, unchanged behavior: `bin-api-manager/server/transcribes.go`
funnels every request-bind failure (including a malformed `id`) into
`cerrors.InvalidArgument(..., "INVALID_JSON_BODY", "The request body is not
valid JSON.")` — the message is imprecise for a bind failure on an
otherwise-valid JSON body, but this is the existing, uniform behavior for
every field on this endpoint today, not something this change regresses.
Not fixing it here to keep scope bounded.

Response shape is unchanged (`Transcribe` object). The returned `id` always
equals the supplied `id` when one was supplied — after the B6/B10
corrections below, a caller-supplied id is never silently replaced with a
different existing row's id; the recording path errors instead of doing
that substitution.

**Non-idempotency of a caller-supplied id:** `Start`'s pre-check (below)
means a caller-supplied id is single-use — a retried request with the exact
same `id` always returns 409 `TRANSCRIBE_ID_ALREADY_EXISTS`, never the
original result, on both the live and recording paths. This is deliberate:
the pre-check is what makes "this id is now safe to pre-subscribe to" true
the moment the caller receives it, and silently treating a retry as "the
same logical request" would require a separate dedup key we don't have. If
a request's outcome is ambiguous (e.g. client timeout), the caller must
**not** retry with the same id — it should `GET /transcribes/{id}` to check
whether the first attempt landed, and mint a new id only if it did not.

## Validation & error handling

Centralized in `transcribehandler.Start`, before dispatch to
`startLive`/`startRecording`:

```go
// captured BEFORE any reassignment -- this is the one and only signal for
// "did the caller ask for a specific id", and it must survive past the
// point where id itself becomes always-non-nil. See B10 below for why this
// separate boolean exists instead of testing id != uuid.Nil downstream.
callerSpecifiedID := id != uuid.Nil

if !callerSpecifiedID {
    id = h.utilHandler.UUIDCreate()
} else if _, err := h.Get(ctx, id); err == nil {
    // a row with this id already exists (any customer, any status,
    // soft-deleted or not -- Get()/TranscribeGet() do not filter on
    // customer_id or tm_delete)
    return nil, cerrors.AlreadyExists(
        commonoutline.ServiceNameTranscribeManager,
        "TRANSCRIBE_ID_ALREADY_EXISTS",
        "A transcribe with the given id already exists. Use a different id or omit it.",
    )
} else if !stderrors.Is(err, dbhandler.ErrNotFound) {
    return nil, err // defensive; Get() already wraps not-found
}
```

This is independent of, and in addition to, the existing
`TRANSCRIBE_ALREADY_PROGRESSING` dedup guard in `startLive` (same
customer+reference+language) — both checks stay, for different reasons.

`startLive` and `startRecording` no longer call `h.utilHandler.UUIDCreate()`
themselves; they take the resolved `id` as a parameter to use for creation.
`startRecording` additionally takes `callerSpecifiedID bool` — see B10 and
the Recording-path section below for why the resolved `id` alone is not
enough information for it to make the idempotency decision correctly.
Neither `startLive` nor `Create` gains `callerSpecifiedID` (see the B11
correction below — the duplicate-key error mapping that might have
suggested otherwise is handled entirely inside `Start`). `Create`'s
signature is untouched altogether. `startLive` does gain the `id` parameter
described in the Signature-changes section below (it needs the resolved id
to create with, same as before this design existed for `Create`) — the
"untouched" claim here is specifically about `callerSpecifiedID`, not about
every parameter.

**Accepted risk — cross-tenant existence oracle:** the pre-check above is
tenant-agnostic by construction (it must be, to guarantee global PK
uniqueness), so a 409 on this endpoint confirms *some* transcribe with that
id exists, possibly belonging to another customer or soft-deleted. IDs are
unguessable v4 UUIDs (or whatever the caller chooses to submit), so this
does not leak information a client didn't already have modulo brute force,
and severity is low. **Do not scope the existence check to the caller's
`customer_id`** — that would let same-id-different-customer requests both
pass the check and then race each other into `TranscribeCreate`, which is
exactly the TOCTOU case handled below, not something to reintroduce.

### TOCTOU: duplicate-key race between the pre-check and the insert

The read-then-create check above has a race window: two concurrent requests
with the same caller-supplied `id` can both pass the `Get()` check before
either has inserted. (This was a non-issue when ids were always
server-generated v4 UUIDs; caller-supplied ids make it reachable via a
retried or duplicated client request.)

**Correction from rev 5 (B11):** rev 5 threaded `callerSpecifiedID` into
`Create` itself, to condition the error mapping there. That breaks:
`Create` has two call sites — `startLive` (`start.go:237`) and
`startRecording` (`recording.go:31`) — and `startLive` had, correctly, no
reason to carry the boolean *until* `Create` started requiring it. Go
methods don't capture a caller's locals, so `callerSpecifiedID` (declared
in `Start`) is not visible inside `startLive` without adding it as a
parameter there too — which rev 5's own text said `startLive` didn't need.
Threading it three ways (`Start` → `startRecording` → `Create`, plus
`Start` → `startLive` → `Create`) is exactly the kind of multi-site
propagation that produced B10 in the first place.

**Fix: do the mapping in `Start`, after dispatch, where `callerSpecifiedID`
is already in scope and needs no further threading.** `Create`'s signature
is fully unchanged; `startLive` keeps its signature free of
`callerSpecifiedID` (it still gains the plain `id` parameter described in
Signature changes below — that part is unrelated to this fix).

Handled at two layers:

1. **`dbhandler.TranscribeCreate`** classifies a MySQL duplicate-key error
   (errno 1062) and returns a new sentinel `dbhandler.ErrDuplicateID`,
   following the existing `isDuplicateKeyErr` idiom already used in
   `bin-agent-manager/pkg/dbhandler/agent_address.go` (typed
   `*mysql.MySQLError` match, not string-contains on the message, plus a
   `UNIQUE constraint failed` fallback for the sqlite test harness). It
   bubbles up through `Create` → `startLive`/`startRecording` unwrapped
   (`errors.Wrapf` preserves the chain for `stderrors.Is`).
2. **`Start`** checks for it once, after dispatch to `startLive`/
   `startRecording` returns:

   ```go
   res, err := /* dispatch to startLive or startRecording */
   if err != nil {
       if stderrors.Is(err, dbhandler.ErrDuplicateID) {
           if callerSpecifiedID {
               return nil, cerrors.AlreadyExists(
                   commonoutline.ServiceNameTranscribeManager,
                   "TRANSCRIBE_ID_ALREADY_EXISTS",
                   "A transcribe with the given id already exists. Use a different id or omit it.",
               )
           }
           // a server-generated id collided -- not a caller error
           return nil, cerrors.Internal(
               commonoutline.ServiceNameTranscribeManager,
               "TRANSCRIBE_ID_GENERATION_COLLISION",
               "Could not create the transcribe due to an internal id collision.",
           )
       }
       return nil, err
   }
   return res, nil
   ```

   This snippet is illustrative, not a literal patch. The real `Start`
   (`pkg/transcribehandler/start.go:68-87` as of this writing) checks and
   returns *inside each `switch referenceType` case*, wrapping the error
   with `errors.Wrapf` per branch, rather than assigning to a single
   `res, err` and checking once after the switch. Implementing this design
   means hoisting that check to after the switch (semantics unaffected —
   `errors.Wrapf` preserves the `stderrors.Is` chain either way, per point 1
   above) — this is a real refactor of the switch's control flow, not a
   drop-in of the snippet above. **While doing that hoist, do not carry
   forward a pre-existing bug in the `default:` case**
   (`start.go:83-84`): `return nil, errors.Wrapf(err, "unsupported
   reference type...", referenceType)` wraps `err`, which is still `nil` at
   that point (no prior statement in that branch sets it) — `errors.Wrapf`
   returns `nil` when given a `nil` error, so `Start` today silently
   returns `(nil, nil)` for an unsupported reference type instead of an
   error. Fix it to `fmt.Errorf("unsupported reference type. reference_type: %s", referenceType)`
   while restructuring this function, since the line is being touched
   anyway.

   This is the same conditional mapping N13 wanted (caller-supplied
   collision → the caller-facing 409; server-generated collision → 500),
   with a single consumer of `callerSpecifiedID` for this purpose instead of
   three, and it reuses the exact error-construction code already written
   for the pre-check a few lines above it in the same function.

**Streaming-session rollback:** in `startLive`, `streamingHandler.Start` is
called once per direction, in a loop, *before* `h.Create`. Today, a failure
leaks whatever streaming session(s) were already started in that loop —
both when `h.Create` itself fails afterward, and when `direction == both`
and the *second* leg's `streamingHandler.Start` call fails after the first
leg already succeeded (`pkg/transcribehandler/start.go:227-230` returns
immediately on that error today, leaking the first leg). Both are
pre-existing gaps; caller-supplied ids only make the `h.Create` case
newly-*foreseeable* (a duplicate-key error is now a real outcome, not a
theoretical one), but since the compensating-stop helper is being added
anyway, both cases are fixed together as part of this change: on any
`streamingHandler.Start` failure mid-loop, or any `h.Create` error after
the loop, `startLive` calls `h.streamingHandler.Stop` for every
`streamingID` already started, best-effort (log failures, don't mask the
original error), before returning.

## Recording-path collision decision (resolves the rev-1 "advisory id" gap)

`startRecording`'s idempotent lookup (`GetByReferenceIDAndLanguage`) still
runs first. Given the stated motivation (pre-bind a subscription to a known
id before the event can fire), silently returning a *different* existing
id when the caller asked for a specific one would leave that subscription
dead with no error — unacceptable.

**Correction from rev 2 (B6):** the original Option A had a branch —
"`id != uuid.Nil` and `tmp.ID == id`: idempotent return" — that can never
execute. `Start`'s pre-check (previous section) already runs before
`startRecording` is dispatched and rejects with 409 whenever the caller
specified an id and *any* row with that id exists. So by the time
`startRecording` runs with a caller-specified id, no row with that id
exists yet — meaning any row this lookup finds by reference+language
necessarily has a *different* id. That case is therefore unreachable, and
the id is consequently **single-use** here too (see the Contract section's
non-idempotency note): a caller-supplied id always conflicts with an
existing recording-transcribe for the same reference+language, full stop.

**Correction from rev 3 (B10):** rev 3 branched this on `id != uuid.Nil`,
reusing the resolved `id` from `Start`. That is wrong: `Start` reassigns
`id` to a freshly generated UUID whenever the caller did *not* specify one,
so by the time `startRecording` runs, `id != uuid.Nil` is **unconditionally
true** regardless of whether the caller asked for anything. Branching on it
would turn "id omitted" into "id always looks specified", making the
`return tmp, nil` idempotent-return branch dead code for every existing
caller (including all 3 mechanical-fix-only services in scope, none of
which will ever pass an id) and silently breaking the existing, documented
recording-idempotency contract. The fix is to branch on the
`callerSpecifiedID` boolean captured in `Start` *before* the reassignment,
not on the resolved `id`'s nilness:

```go
if tmp, err := h.GetByReferenceIDAndLanguage(ctx, recordingID, language); err == nil {
    if callerSpecifiedID {
        // Start's pre-check guarantees a caller-specified id isn't already
        // in use, so a row found here by reference+language is guaranteed
        // to have a different id -- this is always a conflict, never a
        // match on the id itself.
        return nil, cerrors.FailedPrecondition(
            commonoutline.ServiceNameTranscribeManager,
            "TRANSCRIBE_ALREADY_EXISTS_DIFFERENT_ID",
            "A transcribe for this recording/language already exists with a different id.",
        )
    }
    return tmp, nil // id was not caller-specified: unchanged idempotent return
}
```

## Signature changes

`id` goes immediately after the ambient/auth parameters, matching each
layer's existing convention (not universally "right after ctx" — that only
holds where there is no ambient/auth param ahead of it):

```go
// bin-transcribe-manager/pkg/transcribehandler (no ambient param before domain args)
Start(ctx, id, customerID, activeflowID, onEndFlowID, referenceType, referenceID, language, direction, provider) (*Transcribe, error)

// bin-common-handler/pkg/requesthandler (same shape, mirrors Start)
TranscribeV1TranscribeStart(ctx, id, customerID, activeflowID, onEndFlowID, referenceType, referenceID, language, direction, provider, timeout) (*Transcribe, error)

// bin-api-manager/pkg/servicehandler (auth.AuthIdentity is the ambient param, stays second)
TranscribeStart(ctx, a *auth.AuthIdentity, id, ...) (*WebhookMessage, error)
ServiceAgentTranscribeStart(ctx, a *auth.AuthIdentity, id, ...) (*WebhookMessage, error)
```

`streamingHandler.Start(ctx, customerID, id, ...)` (transcribe_id as its
own 3rd positional param, id-of-parent-not-id-of-self) is pre-existing and
out of scope — not touched by this change.

`startLive` and `startRecording` are unexported helpers on
`transcribeHandler`, not part of the `TranscribeHandler` interface, so their
signatures are free to differ from `Start`'s public shape:

```go
startLive(ctx, id, customerID, activeflowID, onEndFlowID, referenceType, referenceID, language, direction, provider) (*Transcribe, error)
startRecording(ctx, id uuid.UUID, callerSpecifiedID bool, customerID, activeflowID, onEndFlowID, recordingID, language, provider) (*Transcribe, error)
```

**Correction from rev 5 (B11):** rev 5 also put `callerSpecifiedID` on
`Create`'s signature. Reverted — `Create`'s signature is **unchanged by
this design** (it already takes `id`; nothing else about it changes). It
has two call sites, `startLive` and `startRecording`, and adding a
parameter there would have required threading the boolean into `startLive`
too, which the design elsewhere correctly says is unnecessary. The
duplicate-key error mapping that motivated this (N13) is done in `Start`
after dispatch instead — see the TOCTOU section above — where
`callerSpecifiedID` is already in scope with no threading needed.

`callerSpecifiedID` therefore has exactly two readers, neither relying on
an assumed capture: `Start`'s own post-dispatch duplicate-key mapping
(a plain local read — same function scope, no threading needed), and
`startRecording`'s idempotency branch (received as an explicit parameter,
per B10). It is the only new parameter anywhere in this design
that is not part of a public interface.

## No DB/schema change

`id` is already the primary key column and `TranscribeCreate` already
accepts any UUID — no migration needed.

## Test coverage to add/update

- `bin-transcribe-manager/pkg/transcribehandler/start_test.go`: id omitted
  (unchanged), id provided & free (used as-is), id provided & already
  exists via pre-check (409 `TRANSCRIBE_ID_ALREADY_EXISTS`), id provided &
  wins the TOCTOU race against the pre-check (mock `Get` returns not-found
  then mock `TranscribeCreate` returns `ErrDuplicateID` → still 409, and the
  streaming(s) started for that attempt are stopped), id omitted & the
  freshly-generated id somehow collides (`ErrDuplicateID` with
  `callerSpecifiedID == false` → `cerrors.Internal`, not the caller-facing
  409 — exercises `Start`'s post-dispatch mapping), plus mechanical
  signature updates to every existing `Start(...)` call. `Create`'s own
  signature is unchanged (B11 correction), so its direct callers in
  `transcribe_test.go` need no update.
- `recording_test.go`: id omitted + existing row (unchanged idempotent
  return); id provided + existing row for that reference+language (always
  `TRANSCRIBE_ALREADY_EXISTS_DIFFERENT_ID` — the "matching id" case is
  unreachable per the correction above and is *not* a test case).
- `bin-transcribe-manager/pkg/dbhandler/transcribe_test.go`: duplicate-id
  insert returns `ErrDuplicateID`.
- `v1_transcribes_test.go`: request body with/without `id`; malformed `id`
  string → 400. (This is `bin-transcribe-manager`'s internal RPC listener —
  there is no generated binder here, just `json.Unmarshal` into
  `request.V1DataTranscribesPost` at `pkg/listenhandler/v1_transcribes.go:34`
  followed by `simpleResponse(400)` on error; the 400 comes from the typed
  `uuid.UUID` field failing to unmarshal, not from any oapi-codegen binder —
  that mechanism is `bin-api-manager`-only, see below.)
- `bin-common-handler/pkg/requesthandler/transcribe_transcribes_test.go`:
  signature update.
- `bin-api-manager/pkg/servicehandler/transcribe_test.go`,
  `serviceagent_transcribe_test.go`: signature/behavior updates.
- `bin-api-manager/server/transcribes_test.go` (currently has no
  `PostTranscribes` test — add one, including the malformed-`id` 400 case),
  `server/service_agents_transcribes_test.go`.
- Mechanical signature-only updates: `bin-ai-manager/pkg/summaryhandler/{start_test.go,service_test.go}`,
  `bin-flow-manager/pkg/activeflowhandler/actionhandle_test.go`,
  `bin-conference-manager/pkg/conferencehandler/transcribe_test.go`.

## Docs to update

- `bin-transcribe-manager/docs/domain.md` — `id` is caller-settable
  (optional); new `TRANSCRIBE_ID_ALREADY_EXISTS` /
  `TRANSCRIBE_ALREADY_EXISTS_DIFFERENT_ID` errors.
- `bin-transcribe-manager/CLAUDE.md` — Critical Implementation Notes: ids
  are no longer guaranteed-unguessable server-generated v4 UUIDs; a caller
  can set a transcribe id equal to another resource's id, which makes
  log/timeline correlation by "looks like a call id" unreliable going
  forward.
- `bin-api-manager/docsdev/source/transcribe_*.rst` and the
  service-agent-transcribe equivalent — document the optional `id` field on
  both endpoints: (1) omitted → unchanged behavior, server generates one;
  (2) supplied → single-use, non-idempotent (a retry with the same `id`
  always 409s — see the Contract section's `GET /transcribes/{id}`
  guidance, this is the user-facing half of the change and belongs here,
  not only in this design doc); (3) on the recording path specifically,
  supplying `id` when a transcribe already exists for that
  recording+language is *always* a conflict (`TRANSCRIBE_ALREADY_EXISTS_DIFFERENT_ID`)
  — it is never silently ignored in favor of the existing row, unlike the
  no-`id` case which keeps today's idempotent-return behavior unchanged.
  Rebuild HTML per root CLAUDE.md RST-sync rule.

## Verification

Full 5-step workflow (`go mod tidy && go mod vendor && go generate ./... &&
go test ./... && golangci-lint run -v --timeout 5m`) in all 7 touched
services. `bin-openapi-manager`'s `go generate ./...` must run, and its
output be vendored into `bin-api-manager`, before `bin-api-manager`'s own
`go generate ./...`.

## Out of scope (explicitly deferred by the user)

- Actually wiring "subscribe to the topic before Start" on any caller.
- Any change to Transcript/Speech/Streaming event subscription mechanics.
