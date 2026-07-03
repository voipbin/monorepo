# Design: Standardize oapi-codegen binding-error envelope (ETC-3)

## Problem

`bin-api-manager` exposes two different error body shapes for the same
class of failure (400 Bad Request):

1. **Handler-level errors** (business logic, e.g. `TRANSCRIBE_NOT_FOUND`)
   go through `server/error.go:abortWithError` → `lib/apierror.EnvelopeFor`,
   producing the standard envelope:
   ```json
   {"error": {"status": "INVALID_ARGUMENT", "reason": "...", "message": "...", "request_id": "..."}}
   ```
2. **oapi-codegen parameter-binding errors** (malformed `id`, missing
   required query arg, bad UUID, etc.) never reach handler code. They
   are caught inside the generated `ServerInterfaceWrapper.*` methods in
   `gens/openapi_server/gen.go`, which call `siw.ErrorHandler(c, err, statusCode)`.
   Because `cmd/api-manager/main.go:245` calls
   `openapi_server.RegisterHandlers(v1, appServer)` (no options), the
   generated default `ErrorHandler` is used:
   ```go
   func(c *gin.Context, err error, statusCode int) {
       c.JSON(statusCode, gin.H{"msg": err.Error()})
   }
   ```
   producing `{"msg": "Invalid format for parameter id: ..."}` — no
   `status`/`reason`/`message`/`request_id`, and no `X-Request-Id`
   correlation (middleware.RequestID() output is unaffected, but the
   body carries nothing to look it up by).

Both paths return HTTP 400 today, so this is a **response body schema
inconsistency**, not a status-code bug. It affects every typed
query/path parameter across the platform (`openapi_types.UUID`, integer
page_size, etc.) — discovered originally on `/transcribes` but present
on all ~30 resource groups.

## Goal

Every 4xx/5xx response from `bin-api-manager` v1.0 routes uses the
single envelope shape defined by `lib/apierror.EnvelopeFor`, including
responses that originate from oapi-codegen's own parameter binding,
before any handler code runs.

## Non-goals

- Changing the envelope schema itself (`lib/apierror.EnvelopeFor` stays
  as-is).
- Changing HTTP status codes (400 stays 400).
- Auth-layer errors (`middleware/authenticate.go`) already use
  `apierror.EnvelopeFor` — untouched.
- Per-parameter structured `details` (e.g. "which field, which
  constraint") — out of scope; `message` continues to carry the raw
  oapi-codegen text. Tracked as a possible follow-up, not blocking.

## Design

### Round-3 revision note (scope narrowing)

Rounds 1–2 pursued a "reuse `INVALID_ID` for ID-shaped binding params"
policy to maximize consistency with existing handler-level UUID
validation. Each review round found a new edge case in the naming
heuristic used to classify a binding failure as "ID-shaped": hyphenated
param names (`billing-id` truncates to `billing`, missing the `_id`
suffix check), non-suffixed ID params (`rid` on
`DELETE /interactions/{id}/resolutions/{rid}`), and a semantic
conflation between "parameter missing" and "parameter malformed" that
a name-only heuristic cannot distinguish (a required `chat_id` that's
simply absent was being reported as "not a valid UUID", which is
factually wrong).

**Decision (CPO call, 2026-07-04): narrow scope.** ETC-3's actual
problem is the envelope *shape* mismatch (`{"msg": ...}` vs the
standard `{"error": {...}}` envelope) — not full reason-code
unification with the pre-existing (and separately inconsistent, see
`speakings.go:100`'s `INVALID_ARGUMENT`-for-`reference_id` outlier)
handler-level `INVALID_ID` convention. Chasing full reuse turned a
bounded envelope-consistency fix into an open-ended parameter-naming
classification problem, and each attempt to patch the classifier
introduced a fresh correctness bug rather than converging. Reusing
`INVALID_ID` via a name-based heuristic is deferred to a **separate
follow-up ticket** scoped explicitly to auditing every ID-shaped
parameter name across the whole OpenAPI spec (a enumerable, one-time
audit — not a runtime heuristic that must handle every future
parameter name pattern).

This revision uses a **single new reason** for every oapi-codegen
binding failure, distinguished only by **message text** (not by a
different reason code), based on which of the two fixed oapi-codegen
error formats fired — a distinction the error text already encodes
unambiguously, with no naming heuristic required.

### New reason codes

Add two reasons to the generic/cross-cutting reason catalog
(`docsdev/source/restful_api_errors.rst`) for wrapper-level
(oapi-codegen) binding failures, distinguished purely by which of the
two **fixed, mutually-exclusive** oapi-codegen error-format strings
fired (verified against all 433 `siw.ErrorHandler(` call sites in
`gen.go` — no third format exists, and the two known prefixes never
overlap; see "Message construction" for the invariant this relies on):

- `INVALID_REQUEST_PARAMETER` — the parameter was present but its
  value failed to bind (malformed UUID, out-of-range int, etc).
  Matches `"Invalid format for parameter <name>: %w"`.
- `MISSING_REQUEST_PARAMETER` — a required parameter was absent
  entirely. Matches `"Query argument <name> is required, but not
  found"` (and, forward-compatibly, any future
  `"<Kind> parameter <name> is required, but not found"` variant —
  see the vendored oapi-codegen gin template, which also defines a
  `"Header parameter %s is required, but not found"` format not
  currently reachable because no header parameter is required by the
  spec today).

Both reasons carry `StatusInvalidArgument` (400), matching current
behavior. This is **not** a name-based classification — round-1/2's
bug class (hyphenated names, non-suffixed ID params like `rid`,
missing-vs-malformed conflation via name-only heuristics) cannot recur
here because the split is purely a prefix match on oapi-codegen's own
fixed, controlled error-format strings, never on the parameter name
itself. Splitting into two reasons matters because
`restful_api_errors.rst`'s own AI Context note instructs clients to
"branch on `reason` for self-healing behavior" — a client that got a
`MISSING_REQUEST_PARAMETER` should simply add the parameter, while
`INVALID_REQUEST_PARAMETER` means the value itself needs correcting;
collapsing both into message-text-only differentiation (as an earlier
draft of this design did) would have forced clients to parse message
strings, which the docs elsewhere explicitly discourage.

The existing `INVALID_ARGUMENT` row's wording is left unchanged
(round-1 flagged a wording overlap with `INVALID_ARGUMENT`'s
"path/query parameter" phrase; with this narrower scope the overlap is
addressed by adding a one-line clarifying note to the `INVALID_ARGUMENT`
row instead of rewording it, since `INVALID_ARGUMENT` genuinely does
still cover handler-level path/query validation for parameters typed
as `string` in the OpenAPI spec — e.g. `providers.go`'s manual UUID
parse — and removing that phrase would make the existing row
inaccurate).

This intentionally does **not** attempt byte-for-byte reason-code
parity with handler-level `INVALID_ID`/`INVALID_ARGUMENT` call sites.
That pre-existing platform-wide inconsistency (documented as a known
gap below) is out of scope for this ticket.

### Known gap (explicitly out of scope, tracked for follow-up)

The platform today already has three different reason/message
combinations for "the same UUID-format mistake" depending on which
layer catches it and how the OpenAPI spec typed that parameter:
handler-level manual parse (`INVALID_ID`, message varies by call site
— some include the param name, most say "The provided id..."),
`speakings.go:100`'s `reference_id` outlier (`INVALID_ARGUMENT`), and
now (with this design) the new wrapper-level `INVALID_REQUEST_PARAMETER`.
This ticket adds a well-defined additional case (wrapper-binding
failures) without reducing the existing three-way inconsistency. A
full audit-and-unify pass across all `INVALID_ID`/`INVALID_ARGUMENT`
call sites is tracked as a separate follow-up ticket, not blocking
this fix.

### Message construction

oapi-codegen's binding error text follows two fixed, mutually-exclusive
formats emitted by the generated wrapper code (verified against all
433 `siw.ErrorHandler(` call sites in `gen.go`: 428 use the first
format, 5 use the second, zero use anything else):
- `"Invalid format for parameter <name>: %w"` → malformed value →
  `INVALID_REQUEST_PARAMETER`.
- `"Query argument <name> is required, but not found"` → missing
  required parameter → `MISSING_REQUEST_PARAMETER`.

**Invariant this relies on (must hold for the branch order below to
stay correct):** the two prefixes are mutually exclusive — no
malformed-value error's `%w`-wrapped inner text can itself start with
`"Invalid format for parameter "` after a different outer prefix, and
none of the wrapped inner error strings from `google/uuid` or
oapi-codegen's own `bindstring.go`/`bindparam.go` (checked: "invalid
UUID format", "value overflows...", "error unmarshaling...") contain
the substring `"is required, but not found"`. Because `HasPrefix` is
checked before `Contains` below, even if this invariant were ever
violated by a future oapi-codegen version, the malformed-value case
would still win (its own outer text always starts with the first
prefix) — the ordering is a deliberate defense-in-depth, not just
happenstance. `TestBindingErrorReason_PrefixOverlap_MalformedWins`
below pins this down with an explicit regression test.

```go
func classifyBindingError(errText string) (reason, message string) {
    switch {
    case strings.HasPrefix(errText, "Invalid format for parameter "):
        if name, ok := extractBindingParamName(errText); ok {
            return "INVALID_REQUEST_PARAMETER", fmt.Sprintf("The parameter %q has an invalid format.", name)
        }
        return "INVALID_REQUEST_PARAMETER", "One or more request parameters have an invalid format."
    case strings.Contains(errText, "is required, but not found"):
        if name, ok := extractBindingParamName(errText); ok {
            return "MISSING_REQUEST_PARAMETER", fmt.Sprintf("The parameter %q is required.", name)
        }
        return "MISSING_REQUEST_PARAMETER", "A required request parameter is missing."
    default:
        return "INVALID_REQUEST_PARAMETER", "One or more request parameters are invalid."
    }
}

var reBindingParamName = regexp.MustCompile(`(?:Invalid format for parameter |Query argument |Header parameter )([\w-]+)`)

func extractBindingParamName(errText string) (string, bool) {
    m := reBindingParamName.FindStringSubmatch(errText)
    if len(m) != 2 {
        return "", false
    }
    return m[1], true
}
```

Note: `[\w-]+` (not `\w+`) so hyphenated parameter names (e.g.
`billing-id`) are captured whole for the message text — this feeds
only the message string, never the reason-code decision (round-2's
bug was specifically that a name-derived value drove reason-code
routing; here it's cosmetic-only, so any extraction miss just degrades
to the generic per-reason fallback message). The regex also matches a
`"Header parameter "` prefix even though no route currently triggers
it (see reason-code note above) — forward-compatible so a future
header-required parameter still gets a named message instead of
silently falling back to the generic one.

The raw `err.Error()` (with oapi-codegen/Go-stdlib internal wording)
is never placed in the public `message` field — kept curated per the
round-1 finding — and remains available server-side only via
`.Wrap(err)` (see Implementation below).

### Implementation

1. **New function** `server.BindingErrorHandler(c *gin.Context, err error, statusCode int)`
   in `bin-api-manager/server/error.go`:
   ```go
   var reBindingParamName = regexp.MustCompile(`(?:Invalid format for parameter |Query argument |Header parameter )([\w-]+)`)

   func BindingErrorHandler(c *gin.Context, err error, statusCode int) {
       // Defensive guard: abortWithError derives the actual HTTP status
       // solely from e.Status, not from this statusCode argument. Every
       // current call site in gen.go passes http.StatusBadRequest (verified
       // by grepping all 433 siw.ErrorHandler( call sites — this branch is
       // therefore unreachable in production today), but this is generated
       // code we don't control — if a future `go generate` starts passing
       // something else (e.g. 500 on an internal binding panic), fail safe
       // to an INTERNAL envelope instead of silently mislabeling a
       // server-side condition as a 400 client error.
       if statusCode != http.StatusBadRequest {
           logrus.WithFields(logrus.Fields{
               "func":        "BindingErrorHandler",
               "status_code": statusCode,
               "error":       err.Error(),
           }).Warn("oapi-codegen binding error handler invoked with unexpected status code; falling back to INTERNAL.")
           abortWithError(c, cerrors.Internal(commonoutline.ServiceNameAPIManager, "INTERNAL", "An internal error occurred.").Wrap(err))
           return
       }

       reason, message := classifyBindingError(err.Error())
       e := cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, reason, message).Wrap(err)
       abortWithError(c, e)
   }

   func classifyBindingError(errText string) (reason, message string) { /* see "Message construction" above */ }
   func extractBindingParamName(errText string) (string, bool) { /* see "Message construction" above */ }
   ```
   - Reuses the existing `abortWithError` helper so request_id wiring,
     HTTP-status mapping, and the domain-stripping boundary all stay
     centralized in one place (no duplicate envelope-construction
     logic).
   - `.Wrap(err)` keeps the raw oapi-codegen error text available to
     server-side logs via `VoipbinError.Error()` (which includes
     `Cause`) without ever serializing it into the JSON body — same
     pattern already used by `server/error_translate.go` for backend
     RPC errors. Traced: `EnvelopeFor` only reads `Status`/`Reason`/
     `Message`/`Details` off the struct and never calls `.Error()` or
     touches `Cause` — confirmed no leak path exists.
   - No parameter-name-based branching decides the *reason code*
     anymore — eliminates the entire class of naming-heuristic bugs
     found in rounds 1–2 (hyphenated names, non-suffixed ID params
     like `rid`, missing-vs-malformed conflation). The
     missing-vs-malformed distinction is reason-code-level (round-3
     finding) but is driven purely by oapi-codegen's own fixed prefix
     strings, not by any parameter name.

2. **Wire it up** in `cmd/api-manager/main.go`:
   ```go
   v1 := app.Group("v1.0")
   v1.Use(middleware.Authenticate())
   openapi_server.RegisterHandlersWithOptions(v1, appServer, openapi_server.GinServerOptions{
       ErrorHandler: server.BindingErrorHandler,
   })
   ```
   Replaces the no-options `RegisterHandlers` call. `GinServerOptions`
   also has `BaseURL` and `Middlewares` fields — left as zero-value
   (matching current behavior; `v1.Use(middleware.Authenticate())` is
   already applied to the whole group before this call, so it is not
   duplicated into `Middlewares`).

3. **RST doc update** — in `docsdev/source/restful_api_errors.rst`:
   - Add **both** new reasons to the Generic/Cross-cutting table:
     - `INVALID_REQUEST_PARAMETER` — malformed value for a request
       parameter caught during wrapper-level (oapi-codegen) binding,
       before any handler code ran.
     - `MISSING_REQUEST_PARAMETER` — a required request parameter was
       absent entirely, caught during the same wrapper-level binding
       stage. (Round-4 review found this row was missing from the
       original checklist even though the "New reason codes" section
       above always specified both reasons go into the catalog — this
       step must add both, not just the first.)
   - Add a one-line clarifying note to the existing `INVALID_ARGUMENT`
     row: it continues to cover handler-level body/path/query
     validation for parameters typed as `string` in the OpenAPI spec
     (e.g. `providers.go`'s manual UUID parse) — distinct from the two
     new wrapper-level reasons above, which fire only for
     `openapi_types`-typed parameters caught by generated binding code
     before the handler runs.
   - Rebuild: `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build`.
   - Force-add the rebuilt HTML: `git add -f bin-api-manager/docsdev/build/`
     (root `.gitignore` excludes `build/`; this step was missing from
     the round-1 draft per review feedback — commit RST source and
     `build/` together).

### Test plan

All new tests live in `bin-api-manager/server/error_test.go` (existing
file — this service already has `TestAbortWithErrorSetsStatusAndBody`
etc. there per `server/error.go`'s doc comment).

- `TestBindingErrorHandler_InvalidFormat_UsesInvalidReason` — call
  `BindingErrorHandler` with `fmt.Errorf("Invalid format for parameter id: invalid UUID length")`
  and a gin test context; assert `reason == "INVALID_REQUEST_PARAMETER"`,
  curated message `The parameter "id" has an invalid format.` (not the
  raw error text), no `domain` key, `request_id` present.
- `TestBindingErrorHandler_HyphenatedParamName_ExtractsWhole` — same
  with `parameter billing-id` to confirm `[\w-]+` captures the full
  hyphenated name into the message (`"billing-id"`), regression test
  for the round-2-found truncation bug.
- `TestBindingErrorHandler_MissingRequiredParam_UsesMissingReason` —
  input `"Query argument aicall_id is required, but not found"`;
  assert `reason == "MISSING_REQUEST_PARAMETER"` and message is
  `The parameter "aicall_id" is required.` — regression test for the
  round-2-found missing-vs-malformed conflation, now promoted to a
  reason-code-level distinction per round-3.
- `TestBindingErrorReason_PrefixOverlap_MalformedWins` — construct a
  synthetic error text where the `%w`-wrapped inner error deliberately
  contains the substring `"is required, but not found"` (e.g.
  `fmt.Errorf("Invalid format for parameter id: %w", errors.New("value is required, but not found in enum"))`)
  and assert the outer `"Invalid format for parameter "` prefix still
  wins → `INVALID_REQUEST_PARAMETER`, not `MISSING_REQUEST_PARAMETER`.
  This pins down the round-3-flagged prefix-exclusivity invariant with
  an explicit test instead of only a code comment.
- `TestBindingErrorHandler_UnparseableErrorText_FallsBackGeneric` — an
  error string that matches neither fixed format; assert graceful
  fallback to `INVALID_REQUEST_PARAMETER` with the generic
  parameterless message (no panic, no empty-string param name leak).
- `TestBindingErrorHandler_UnexpectedStatusCode_FallsBackToInternal` —
  call with `statusCode = http.StatusInternalServerError`; assert the
  response is the `INTERNAL` envelope, not `INVALID_ARGUMENT` — this is
  the round-1-flagged defensive-guard behavior and must have explicit
  coverage, not just a code comment. Documented as currently
  unreachable in production (all 433 real call sites pass 400) but
  kept as a regression guard against a future `go generate`.
- **Regression check for existing route tests**: `server/*_test.go`
  (e.g. `messages_test.go`, `customer_test.go`) call
  `openapi_server.RegisterHandlers(r, h)` directly (not through
  `main.go`), so they keep using the **default** oapi-codegen
  ErrorHandler (`{"msg": ...}`) unless individually updated. These
  tests exercise handler-level logic, not binding failures, so they
  are unaffected by this change — confirmed no test asserts on the
  `{"msg": ...}` shape for a binding failure (grepped whole repo,
  zero hits besides `gen.go`'s own definition). No test file changes
  needed there.
- **One true end-to-end regression test** — new file
  `bin-api-manager/server/binding_error_integration_test.go`: boots a
  gin router via `openapi_server.RegisterHandlersWithOptions` (mirroring
  `cmd/api-manager/main.go`'s actual call, not the other test files'
  bare `RegisterHandlers`) with `ErrorHandler: BindingErrorHandler`,
  fires a request with a malformed UUID path param against a real
  generated route (e.g. `GET /transcribes/{id}` with `id=not-a-uuid`),
  and asserts the full HTTP response body matches the standard
  envelope end-to-end — proving the wiring in `main.go`, not just the
  function in isolation.

### Verification workflow

Per root CLAUDE.md: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` in `bin-api-manager`.

### Rollout / risk

- Purely additive at the wire level for well-behaved clients (still
  400, still JSON body) — only clients pattern-matching on the exact
  `{"msg": ...}` shape (undocumented, never published in RST) would
  break. Given `restful_api_errors.rst` already documents the
  `{"error": {...}}` shape as canonical and never mentions `{"msg":
  ...}`, this fix aligns behavior with the already-published contract
  rather than introducing a new contract.
- No DB/schema/migration involved. No RPC change. Single-service
  (`bin-api-manager`) change.

## Review history

**Round 1** (2026-07-04, 2 parallel independent subagents):
- Fact-verification reviewer: APPROVED, 1 IMPORTANT (RST `INVALID_ARGUMENT`
  wording collision) + 2 NICE-TO-HAVE.
- Adversarial reviewer: CHANGES_REQUESTED, 3 IMPORTANT:
  1. New `INVALID_REQUEST_PARAMETER` reason creates a *fresh*
     inconsistency vs. existing handler-level `INVALID_ID` for the
     identical UUID-format mistake, undermining the stated
     standardization goal.
  2. Raw `err.Error()` in the public `message` field breaks the
     codebase-wide "curated static message, raw cause server-side only"
     convention (see `server/error_translate.go`).
  3. `statusCode` argument silently ignored with only a comment as
     mitigation — a future `go generate` regen passing a non-400 status
     would silently mislabel a server error as a 400 client error.
  Plus 3 NICE-TO-HAVE (missing `git add -f` step, SDK enum impact
  unverifiable in this repo, missing concrete test file names).

**Resolution applied above**: reason-code policy now reuses `INVALID_ID`
for ID-shaped params (bucket policy in "Reason-code policy"); messages
are curated static strings with the raw error only reachable via
`.Wrap(err)` server-side; `statusCode != http.StatusBadRequest` now
fails safe to an `INTERNAL` envelope with an explicit log warning and
a dedicated test; `git add -f` step and concrete test file paths were
added to the RST/test-plan sections above.

**Round 2** (2026-07-04, 2 parallel independent subagents, fresh
context, re-verifying round-1's fixes against the codebase — not
just checking them off):
- Fact-verification reviewer: CHANGES_REQUESTED, 1 BLOCKER + 1
  IMPORTANT: the `\w+` regex truncates hyphenated parameter names
  (`billing-id` → captures `billing`, missing `_id` suffix → wrongly
  routed to the new reason instead of `INVALID_ID`); a non-suffixed ID
  parameter (`rid` on `/interactions/{id}/resolutions/{rid}`) is
  ID-shaped but the naming heuristic can't detect it.
- Adversarial reviewer: CHANGES_REQUESTED, 1 BLOCKER + 1 IMPORTANT: the
  name-only classification conflates "parameter missing" with
  "parameter malformed" (a genuinely absent `chat_id` was reported as
  "not a valid UUID", which is factually wrong); the ID/non-ID buckets
  constructed messages inconsistently (one included the param name,
  the other didn't), reintroducing round-1's "internal detail leaks
  into client-visible inconsistency" concern in a new form.

**Resolution applied above (round 3): scope narrowing.** Rather than
patching the naming heuristic again (each round-1/2 fix pattern
produced a new edge case rather than converging), the design now uses
a **single reason** (`INVALID_REQUEST_PARAMETER`) for all binding
failures, with message text branching only on which of the two fixed
oapi-codegen error-format strings fired (malformed vs. missing) — a
distinction the error text encodes unambiguously with no name
inspection required. Full reuse of `INVALID_ID` for ID-shaped
parameters (and the broader pre-existing platform inconsistency
between `INVALID_ID`/`INVALID_ARGUMENT` call sites) is explicitly
deferred to a follow-up ticket (see "Known gap" above), since it is
an open-ended parameter-naming classification problem, not a bounded
envelope-shape fix.

**Round 3** (2026-07-04, 2 parallel independent subagents, fresh
context): Fact-verification reviewer APPROVED (no defects found — the
regex fix, `.Wrap()` no-leak path, and RST claims all re-verified
against the live codebase). Adversarial reviewer CHANGES_REQUESTED, 2
IMPORTANT:
1. Collapsing "malformed" and "missing" into one reason code and
   distinguishing only by message text contradicts
   `restful_api_errors.rst`'s own stated client-integration contract
   ("Clients should branch on `reason` for self-healing behavior").
   Crucially, this distinction is **not** reached via a name-based
   heuristic (the actual source of every round-1/2 bug) — it's a safe
   fixed-prefix match on oapi-codegen's own two error formats. There
   was no need to give this distinction up when narrowing scope; it
   could safely become two reason codes without reintroducing any
   naming-heuristic risk.
2. The `HasPrefix`-before-`Contains` switch ordering being safe (the
   two oapi-codegen prefixes are mutually exclusive) was not
   documented as an explicit invariant, so a future refactor could
   silently break it without any test catching a message-format
   overlap.

**Resolution applied below (round 4): promote missing/malformed to two
reason codes.** Both are still detected via the same safe fixed-prefix
match already used for message text (not a name heuristic), so this
change carries none of the risk that caused rounds 1–2 to fail.

**Round 4** (2026-07-04, 2 parallel independent subagents): both
CHANGES_REQUESTED, converging on the **same single IMPORTANT/BLOCKER**
finding — the "Implementation §3 RST doc update" checklist still had
stale round-3 wording that only instructed adding
`INVALID_REQUEST_PARAMETER` to the catalog, omitting
`MISSING_REQUEST_PARAMETER` even though "New reason codes" above always
specified both. Both reviewers independently confirmed the core design
(prefix-exclusivity invariant, `.Wrap(err)` no-leak path, open reason
enum — RST explicitly documents "Reason codes are append-only",
`ErrorBody.Reason` is a plain string with no closed enum anywhere in
the OpenAPI spec, `cerrors.InvalidArgument`'s `reason` parameter is a
free string with no coupling to the `Status` name) as sound with no
further blocking issues. Minor nice-to-haves noted: a stale
"non-ID-shaped parameters" phrase left over from the abandoned
name-heuristic approach (cosmetic only), an off-by-a-few count of
gen.go call sites (429 vs. the doc's 433 — immaterial to the logic,
corrected below), and a suggestion (out of this ticket's scope) to add
a Prometheus counter given `MISSING_REQUEST_PARAMETER` only fires on
5 of 433 real call sites.

**Resolution applied below (round 4 fix)**: "RST doc update" step now
lists both reasons explicitly; stale heuristic-era phrasing removed;
call-site count corrected to the exact grep result.
