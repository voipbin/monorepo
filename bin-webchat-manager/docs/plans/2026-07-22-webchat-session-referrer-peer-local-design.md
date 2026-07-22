# bin-webchat-manager: Session Referrer + Peer/Local address capture

Status: CLOSED (rounds 0-3, 2 consecutive APPROVE) -- see §8 Revision 1 (round 2 -- fixes revision1-round1 review finding, see docs/plans/2026-07-22-webchat-session-referrer-peer-local-design-review-revision1-round1.md) for a post-closure change to §4.1's type-string decision
Author: Hermes (CPO)
Date: 2026-07-22

## 1. Problem

Following the already-shipped `page_url` feature (PR #1131/#380, merged
2026-07-22), pchero asked for two more pieces of context to be captured on
webchat `Session`:

1. **`referrer`** (`document.referrer`) -- which page the visitor was on
   immediately before arriving at the page that embeds the widget (distinct
   from `page_url`, which is the page the widget is currently embedded on;
   see the already-shipped design doc's §3 for that distinction).
2. **`Peer`/`Local`** (`commonaddress.Address`) on `Session`, mirroring the
   pattern already shipped on `kase.Case`/`interaction.Interaction`
   (PR #1130), but with an explicit new `commonaddress.Type` --
   `TypeWebSession` -- for the Peer role, rather than reusing the existing
   `TypeWebchat` for both roles.

This design doc supersedes the CPO's earlier verbal rejection of a
webchat-Session Peer/Local (recorded in the `page_url` design doc's §6,
"Rejected alternative: Peer/Local on Session"). That rejection was based on
both roles sharing the same `TypeWebchat` value, making Peer a tautology of
the Session's own ID. This design closes that gap by giving Peer a
DIFFERENT type than Local, which changes the calculus (see §4.2).

## 2. Goal / Non-goals

**Goal:**
- Capture `document.referrer` at session-creation time, store it, expose it
  read-only, alongside the already-shipped `page_url`.
- Add `Peer`/`Local` `commonaddress.Address` fields to `Session`, with
  `Peer.Type = TypeWebSession` (new), `Peer.Target = Session.ID`,
  `Local.Type = TypeWebchat` (existing), `Local.Target = Widget.ID`.

**Non-goals:**
- No change to `kase.Case`/`interaction.Interaction`'s EXISTING Peer/Local
  values for webchat-originated Cases/Interactions (those continue to use
  `TypeWebchat` for both roles, unchanged by this doc -- see §5 for why this
  is a deliberate scope boundary, not an oversight).
- No UTM/campaign parsing of `referrer` (same non-goal as `page_url`'s
  design doc §2).
- No retroactive backfill of `Peer`/`Local`/`referrer` for
  already-existing Session rows (all three are nullable/optional; existing
  rows simply have them absent).

## 3. `referrer`: design (mirrors `page_url` exactly)

This is a near-verbatim repeat of the already-shipped `page_url` feature's
design and file list, substituting `document.referrer` for
`window.location.href`. Documented in full here (not just "see the other
doc") per this repo's convention of each design doc being self-contained,
and because the PR-review round-1 finding on `page_url` (the
`javascript:`/`data:` scheme XSS vector, fixed in commit `0d818afa1`)
applies IDENTICALLY here and must not be re-introduced.

### 3.1 Client: `webchat-widget-runtime/client.js`

In `_doStart()`, alongside the already-shipped `page_url`:

```js
const session = await this._fetchJson(this._v1Url('/webchat_sessions'), {
  method: 'POST',
  body: JSON.stringify({
    widget_id: this.resourceId,
    page_url: (typeof window !== 'undefined' && window.location?.href) || undefined,
    referrer: (typeof document !== 'undefined' && document.referrer) || undefined,
  }),
})
```

`document.referrer` is an empty string (not undefined/null) when there is no
referrer (direct navigation, typed URL, bookmark) -- the `|| undefined`
guard converts that empty string to `undefined` so it is omitted from the
JSON body entirely, exactly mirroring how an absent `page_url` is handled
today (rather than sending an explicit empty-string `referrer` that would
need a second "is this meaningfully absent" check server-side).

### 3.2 API contract, backend, DB, square-admin

Every file `page_url` touched gets the identical `referrer` treatment,
side by side:

- `bin-openapi-manager`: `referrer` (string, maxLength 2048, optional) next
  to `page_url` in both the `POST /webchat_sessions` request body and the
  `WebchatManagerSession` response schema.
- `bin-webchat-manager/models/session/session.go`: `Referrer string
  \`json:"referrer,omitempty" db:"referrer"\`` next to `PageURL`.
- `models/session/field.go`: `FieldReferrer`.
- `models/session/webhook.go`: `Referrer` added to `WebhookMessage` +
  `ConvertWebhookMessage()`.
- `pkg/listenhandler/models/request/v1_sessions.go`: `Referrer string` on
  `V1DataSessionsPost`.
- `pkg/listenhandler/v1_sessions.go`: `processV1SessionsPost` threads
  `req.Referrer` into `Create(...)`.
- `pkg/sessionhandler/main.go`/`create.go`: `Create(...)` signature gains
  `referrer string`, stored on the `session.Session{}` literal.
- `bin-common-handler/pkg/requesthandler/main.go` + `webchat_session.go`:
  `WebchatV1SessionCreate` gains `referrer string`, threads into
  `V1DataSessionsPost{..., Referrer: referrer}`.
- `bin-api-manager/server/webchat_sessions.go`: passes `req.Referrer`
  through.
- `bin-api-manager/pkg/servicehandler/webchat_session.go`:
  `WebchatSessionCreate` gains `referrer string`, validated by a NEW
  `validateReferrer` helper -- **identical logic to `validatePageURL`**
  (2048-char cap + http/https scheme allowlist). Not a shared function
  with `validatePageURL` (two near-identical private validators, not a
  premature abstraction) -- mirrors this file's existing pattern of
  one-purpose-built helper per field rather than a generic
  `validateURLField(name, value)` that would need a discriminator param
  for zero actual benefit at this call count (2 call sites).
- `bin-dbscheme-manager`: new Alembic migration
  `webchat_sessions_add_column_referrer`, `ALTER TABLE webchat_sessions ADD
  COLUMN referrer VARCHAR(2048) NULL` (mirrors `04b99363284c` exactly).
- `scripts/database_scripts_test/sessions.sql`: `referrer TEXT` column
  added next to `page_url`.
- square-admin `message_timeline.js`: a second header line, "Came from:
  `<link>`" below the existing "Started from:" line, reusing
  `truncatePageURL`/`isSafePageURL` (renamed to `truncateURL`/`isSafeURL`
  since they are no longer page_url-specific -- see §3.3) rather than
  duplicating both helpers under new names.

### 3.3 Refactor note: rename `truncatePageURL`/`isSafePageURL`

Since `referrer` needs the identical truncate+scheme-guard treatment as
`page_url`, and duplicating both functions under `referrer`-specific names
would be a copy-paste of security-sensitive logic (the exact kind of
duplication that caused the round-1 XSS gap to need a second, independent
fix in JS after the Go side), this design RENAMES the two helpers in
`message_timeline.js`:

- `truncatePageURL(url)` -> `truncateURL(url)`
- `isSafePageURL(url)` -> `isSafeURL(url)`

Both keep their exact existing implementation (only the name changes), and
both call sites (`page_url`'s existing render + `referrer`'s new render)
use the same two functions. This is a pure rename with no behavior change
to the already-shipped `page_url` rendering -- verified by the existing
`message_timeline.test.js` cases for `page_url` continuing to pass
unmodified (they test behavior, not the internal helper names).

## 4. Peer/Local: design

### 4.1 New `commonaddress.Type`

`bin-common-handler/models/address/main.go` gains:

> **[SUPERSEDED BY §8 REVISION 1]** The code snippet and rationale
> immediately below (through the end of §4.1) reflect this design's
> ROUND-1 decision, `"webchat_visitor"`. **That decision was reverted by
> §8 (Revision 1): the shipped string value is `"web_session"`, not
> `"webchat_visitor"`.** The snippet below is kept verbatim as review
> history (it is what rounds 0-3 actually reviewed and approved), NOT as
> the current spec. Jump to §8.3 for the authoritative, current code
> snippet before implementing.

```go
TypeWebSession Type = "webchat_visitor" // target is webchat-manager's Session.ID (the visitor's continuity token)
```

**Round 1 decision (resolves round-0 open question §6.1):** the literal
string value is `"webchat_visitor"`, NOT `"web_session"`. Rationale:

**Naming collision check (mandatory per lessons from the earlier verbal
exchange on this topic):** the literal string `"web_session"` ALREADY
EXISTS in three places in this monorepo today, as an unexported map key,
NOT as a `commonaddress.Type` enum member:

- `bin-flow-manager/pkg/activeflowhandler/actionhandle.go:1265`
- `bin-ai-manager/pkg/aicallhandler/tool.go:453`
- `bin-contact-manager/pkg/contacthandler/interaction.go:66`

All three are entries in a **locally-duplicated `crmIneligiblePeerTypes`
map** (three independent copies, each service's own file, each commented
`// synthetic type; not in commonaddress.Type enum`), used to decide
whether a Call/Conversation peer address type disqualifies a
`case_create` Flow action / AI tool from creating a CRM Case. **Confirmed
by tracing the actual webchat data flow:** these three maps operate ONLY
on `commonaddress.Type` values that `bin-call-manager`/
`bin-conversation-manager` attach to CALL and CONVERSATION-MESSAGE
webhook payloads. Webchat conversations flow through
`bin-conversation-manager` tagged `TypeWebchat` (confirmed at
`bin-conversation-manager/pkg/conversationhandler/event.go:31`,
`pkg/messagehandler/send.go:39,191`, `event_webchat.go:88-89` -- both
self AND peer use `TypeWebchat`), never `"web_session"` -- so a new type
value here would NOT interact with those maps' existing (and,
per pchero's confirmation, correct and intentional) behavior either way.

**Despite the proven non-interaction, this design picks a DIFFERENT
literal (`"webchat_visitor"`) rather than reusing `"web_session"`
anyway.** Reasoning: the non-interaction proof holds only for CODE THAT
EXISTS TODAY. A future engineer grepping the codebase for `"web_session"`
would find it already reserved in three `crmIneligiblePeerTypes` maps and
could reasonably assume this new enum member is either (a) the same
concept being formalized, or (b) in conflict with those maps' existing
semantics -- neither reading is correct, but both are plausible enough to
cost real debugging time. Avoiding the string collision entirely removes
that ambiguity at zero cost (this is a brand-new type with no existing
callers to migrate), which is strictly cheaper than documenting a
non-obvious "these are unrelated despite the identical string" fact for
every future reader. `"webchat_visitor"` was chosen over alternatives
(`"web_visitor"`, `"webchat_session_peer"`) for readability and for
pairing naturally with the existing `"webchat"` (`TypeWebchat`) value
used for Local -- `webchat_visitor` reads as "the visitor side of a
webchat interaction," `webchat` as "the webchat channel itself."

Per §4.1's finding, if Case/Interaction ever adopt `TypeWebSession`
(deferred per §4.3), the three `crmIneligiblePeerTypes` maps' existing
`"web_session"` entries would need separate, explicit re-evaluation at
that time -- they do NOT automatically cover `"webchat_visitor"` (a
different string), so that future work must not assume the existing
blacklist entries transfer.

### 4.2 Why Peer/Local now clears the earlier "zero information" bar

The `page_url` design doc's §6 rejected Session Peer/Local because BOTH
would-be fields used `TypeWebchat`, making `Peer.Target == Session.ID`
(the record's own primary key) a tautology and `Local.Target ==
Widget.ID` a byte-for-byte duplicate of the pre-existing `WidgetID`
column.

This design changes the PEER role's type to the new `TypeWebSession`,
while Local stays `TypeWebchat`:

> **[SUPERSEDED BY §8 REVISION 1]** The code snippet immediately below,
> and every `"webchat_visitor"` mention in this §4.2 (the type-pair
> comparison, the "Web visitor" render-label example), reflect this
> design's ROUND-1 decision. **§8 (Revision 1) reverts the string value
> to `"web_session"`** -- the underlying dispatch argument this section
> makes (Peer type-distinguishable from Local) is UNCHANGED by that
> revert (both `"web_session"` and `"webchat_visitor"` equally
> distinguish Peer from Local's `"webchat"`), only the literal string
> differs. See §8.3 for the current, authoritative string value.

```go
Peer  commonaddress.Address `json:"peer"  db:"peer,json"`  // {Type: TypeWebSession, Target: Session.ID}
Local commonaddress.Address `json:"local" db:"local,json"` // {Type: TypeWebchat,    Target: Widget.ID}
```

This does NOT add new information content in the strict sense either --
`Peer.Target` is still the row's own ID, `Local.Target` is still a copy of
`WidgetID`. What changes is that **Peer is now type-distinguishable from
Local** (`webchat_visitor` vs `webchat`), which matters for exactly one
concrete consumer: **a future cross-channel Peer/Local rendering
component** (as originally floated in the very first exchange on this
topic) that dispatches on `Address.Type` to decide how to render/label an
address (e.g. `tel` -> phone icon, `email` -> envelope icon,
`webchat_visitor` -> "Web visitor" label, vs. today where `webchat`-typed
Local/Peer are visually indistinguishable from each other without also
inspecting which field they came from). This is a real, if narrow,
benefit: format consistency across Voice/SMS/Email/Chat becomes
achievable in a single shared component keyed purely on `Address.Type`,
which was not true when Peer and Local carried the same Type string.

**Open question carried into round 1 review:** is this single benefit
(type-based dispatch for a not-yet-built shared rendering component)
sufficient justification on its own, given the `Target` values still
carry zero NEW information beyond what `Session.ID`/`Session.WidgetID`
already provide? Flagging honestly rather than overselling it.

### 4.3 Scope boundary: Case/Interaction's EXISTING webchat Peer/Local are UNCHANGED

pchero's original phrasing floated unifying Case/Interaction to also use
`TypeWebSession`, then explicitly deferred that in the same conversation
("Session뿐만 아니라 Case/Interaction까지 함께 WebSession으로 통일" was
raised, then the conversation moved to confirming CRM behavior instead of
committing to that expanded scope). **This design does NOT touch
Case/Interaction.** They keep using `TypeWebchat` for webchat-derived
Peer/Local, exactly as shipped in PR #1130. Rationale for deferring, made
explicit rather than silently dropped:

- Unifying would require a DATA migration on already-live
  `contact_cases`/`contact_interactions` rows (peer_type/local_type
  columns, generated-column dependents `open_peer_uk`/`uq_case_open_peer`
  per the `167bebb7c46f` migration's documented dependency chain) --
  strictly higher risk than this design's brand-new, currently-empty
  `webchat_sessions.peer`/`.local` columns.
- It would touch `bin-flow-manager`, `bin-ai-manager`,
  `bin-conversation-manager` call sites that construct webchat
  Peer/Local addresses today (`event.go`, `send.go`,
  `convtitle/build.go`) -- a materially larger blast radius than this
  design's single-service (`bin-webchat-manager`) + gateway
  (`bin-api-manager`) scope.
- No concrete consumer need for the unification was identified in the
  conversation beyond "consistency for its own sake" -- which conflicts
  with pchero's own standing principle (CPO memory) against schema
  changes whose only justification is looking like another table's
  pattern.

If a future need for Case/Interaction to ALSO use `TypeWebSession`
emerges, that is a separate design doc with its own review loop, scoped
explicitly to the data-migration risk above.

### 4.4 Backend implementation

**`bin-webchat-manager/models/session/session.go`**: add both fields
after `WidgetID`/`Status` (mirrors `kase.Case`'s field ordering: identity
fields, then Peer/Local, then everything else):

```go
Peer  commonaddress.Address `json:"peer"  db:"peer,json"`
Local commonaddress.Address `json:"local" db:"local,json"`
```

Unlike `page_url`/`referrer` (`omitempty`, genuinely absent for old rows),
`Peer`/`Local` are **NOT `omitempty`** -- mirrors `kase.Case.Peer`'s
"ALWAYS PRESENT in JSON output" convention (`kase.go`'s comment), since
both are computable unconditionally from data the `Create()` call already
has in hand (`id`, `widgetID`) at construction time, with no "unknown"
case to represent as absence.

**`pkg/sessionhandler/create.go`**: `Create()` already computes byte-identical
values as local variables `self`/`peer` (lines 78-79 of the current file,
used only for the `ConversationV1ConversationCreateAndExecuteFlow` RPC
call) -- this design STORES those same two values on the `Session{}`
literal instead of only passing them to the RPC call:

```go
s := &session.Session{
    Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
    WidgetID: widgetID,
    Status:   session.StatusActive,
    PageURL:  pageURL,
    Referrer: referrer,
    Peer:     commonaddress.Address{Type: commonaddress.TypeWebSession, Target: id.String()},
    Local:    commonaddress.Address{Type: commonaddress.TypeWebchat, Target: widgetID.String()},
}
```

Note the TYPE CHANGE from the current local-variable computation: today's
`peer := commonaddress.Address{Type: commonaddress.TypeWebchat, Target:
id.String()}` (line 79) uses `TypeWebchat` for what becomes the Flow's
"peer" argument. This design's new `Session.Peer` field uses the NEW
`TypeWebSession` instead. **These are two independent uses that must not
be conflated**: the existing `self`/`peer` locals feed
`ConversationV1ConversationCreateAndExecuteFlow` (which creates a
Conversation, unrelated to Session's own stored fields) and are UNCHANGED
by this design (still `TypeWebchat` on both sides, per §4.3's scope
boundary); the NEW `Session.Peer`/`Session.Local` fields are a separate,
new computation using the new type for Peer only. Implementers must not
"deduplicate" these into a single shared local var pair -- they are
deliberately different once this design lands.

**`bin-common-handler/models/address/normalize.go`** and **`validate.go`**:
both files have EXHAUSTIVE switches over `commonaddress.Type`
(`normalize.go:50`'s `case TypeNone, TypeAgent, ..., TypeWebchat:` and
`validate.go:33`'s `case TypeAgent, TypeConference, TypeLine,
TypeExtension, TypeWebchat:`) that must each gain `TypeWebSession`,
matching `TypeWebchat`'s existing treatment (opaque UUID: identity
normalization / UUID-format validation) in both. Missing this update
means `NormalizeTarget`/`ValidateTarget` return `ErrUnknownType`/"unknown
address type" for the new type -- a silent, easy-to-miss omission since
neither Session's own dbhandler nor sessionhandler currently calls either
function on Peer/Local. This design does NOT add a `NormalizeTarget`
call in `sessionhandler/create.go` (the value is already a raw UUID
string with nothing to canonicalize, exactly like `kase.Case`'s existing
`TypeWebchat` peer today), but the switch-exhaustiveness fix is still
required so any OTHER future caller of `NormalizeTarget`/`ValidateTarget`
with `TypeWebSession` does not hit an unexpected error.

**`models/session/field.go`**: `FieldPeer`, `FieldLocal`.

**`models/session/webhook.go`**: `Peer`/`Local` added to `WebhookMessage`
+ `ConvertWebhookMessage()` (no `omitempty` on Peer/Local, matching the
internal model).

**Database (Round 1 decision, resolves round-0 open question §6.3):**
`webchat_sessions` gains `peer JSON NULL`, `local JSON NULL` columns --
**nullable at the DB level**, diverging from the Go/JSON layer's "always
present" contract (§4.4's Go struct has no `omitempty` on `Peer`/`Local`;
every row created through `sessionhandler.Create()` from this point
forward always populates both). This mirrors `kase.Case.Local`'s own
existing precedent (nullable at the DB level via generated columns, while
still JSON-required/non-`omitempty` at the app layer per `kase.go`'s
comment) -- so this is not a novel pattern in this codebase, just applying
an already-established one.

Rationale for choosing nullable over `contact_cases`'s NOT NULL +
three-step backfill approach (`167bebb7c46f`): `id`/`widget_id` are
`BINARY(16)` (confirmed in `sessions.sql`), so a naive backfill via
`HEX(id)` would produce an un-dashed 32-char hex blob
(`550e8400e29b41d4a716446655440000`), NOT a canonical UUID string
(`550e8400-e29b-41d4-a716-446655440000`) that `uuid.FromStringOrNil`
(used by `validateUUID` in `validate.go`) can parse -- a real formatting
gap, not a hypothetical one. Producing a properly-dashed UUID string
purely in SQL requires a `CONCAT_WS`/`SUBSTR` expression with no
precedent elsewhere in this codebase's migrations. Given
`webchat_sessions` is a short-lived, high-churn table (sessions
end/expire; `widget.go`'s `DefaultSessionIdleTimeout` = 1800s / 30 min
confirms in-flight sessions age out on the order of minutes-to-hours,
unlike Case's long-lived CRM records), the correctness value of a
NOT NULL guarantee on rows created BEFORE this migration lands is low:
those rows will have ended/expired long before any code reads
`Peer`/`Local` off them in anger, and the app layer (Go struct with no
`omitempty`) already guarantees every row created AFTER this migration
lands has both fields populated correctly. The migration is therefore a
single, unconditional step with no backfill:

```sql
ALTER TABLE webchat_sessions
    ADD COLUMN peer  JSON NULL AFTER widget_id,
    ADD COLUMN local JSON NULL AFTER peer;
```

No generated column depends on `peer`/`local` here (no `open_peer_uk`-style
dependency chain, unlike Case), so there is no drop-and-recreate ordering
concern either.

## 5. Files touched (implementation checklist)

**bin-common-handler:**
- `models/address/main.go` (`TypeWebSession` constant)
- `models/address/normalize.go` (switch exhaustiveness)
- `models/address/validate.go` (switch exhaustiveness)
- `models/address/main_test.go`/`normalize_test.go`/`validate_test.go` (new type coverage)
- `pkg/requesthandler/main.go` (`WebchatV1SessionCreate` gains `referrer string`)
- `pkg/requesthandler/webchat_session.go`
- `pkg/requesthandler/mock_main.go` (regenerated)

**bin-openapi-manager:**
- `openapi/paths/webchat_sessions/main.yaml` (`referrer` request field)
- `openapi/openapi.yaml` (`WebchatManagerSession`: `referrer`, `peer`, `local`)

**bin-api-manager:**
- `server/webchat_sessions.go`
- `pkg/servicehandler/main.go` (`WebchatSessionCreate` gains `referrer string`)
- `pkg/servicehandler/webchat_session.go` (new `validateReferrer`, mirrors `validatePageURL` incl. scheme allowlist)
- `pkg/servicehandler/webchat_session_test.go`
- `pkg/servicehandler/mock_main.go` (regenerated)
- `docsdev/source/webchat_struct_session.rst` (`referrer`, `peer`, `local` bullets)

**bin-webchat-manager:**
- `models/session/session.go` (`Referrer`, `Peer`, `Local`)
- `models/session/field.go` (`FieldReferrer`, `FieldPeer`, `FieldLocal`)
- `models/session/webhook.go`
- `pkg/listenhandler/models/request/v1_sessions.go` (`Referrer` field)
- `pkg/listenhandler/v1_sessions.go`
- `pkg/sessionhandler/main.go` (`Create(...)` gains `referrer string`; `Peer`/`Local` computed internally, not passed in)
- `pkg/sessionhandler/mock_main.go` (regenerated)
- `pkg/sessionhandler/create.go`
- `pkg/sessionhandler/create_test.go`
- `scripts/database_scripts_test/sessions.sql` (`referrer TEXT`, `peer TEXT` NULL, `local TEXT` NULL -- nullable per §4.4's round-1 decision, JSON-shaped strings per SQLite test-schema convention)

**bin-dbscheme-manager:**
- `bin-manager/main/versions/<new>_webchat_sessions_add_column_referrer.py`
- `bin-manager/main/versions/<new>_webchat_sessions_add_columns_peer_local.py`

**monorepo-javascript (square-admin):**
- `src/webchat-widget-runtime/client.js` (`referrer: document.referrer`)
- `src/webchat-widget-runtime/__tests__/client.test.js`
- `src/views/webchat_widgets/message_timeline.js` (rename `truncatePageURL`/`isSafePageURL` -> `truncateURL`/`isSafeURL`; new "Came from" line)
- `src/views/webchat_widgets/__tests__/message_timeline.test.js`
- `public/webchat-widget-runtime.bundle.js` / `.esm.js` (rebuilt via `npm run build:widget`)

## 6. Round 0 open questions -- resolved in round 1

For audit-trail continuity, round 0 originally left three items open;
this revision resolves them:

1. **§4.1 type-string choice**: RESOLVED (at round 1) -- `"webchat_visitor"`,
   not `"web_session"`. See §4.1's "Round 1 decision" note for full
   rationale (avoids future-reader ambiguity with the three pre-existing
   `crmIneligiblePeerTypes` map entries, at zero migration cost since this
   is a brand-new type with no existing callers). **[SUPERSEDED BY §8
   REVISION 1]: this round-1 resolution was itself later reverted --
   the shipped string value is `"web_session"`. See §8.1-§8.3 for the
   current, authoritative decision and rationale (the three map entries
   this item worried about are deleted by §8.3, removing the premise for
   avoiding the string).**
2. **§4.2 justification sufficiency**: NOT further resolved -- this
   remains an honest, standing caveat rather than a blocking question.
   `Peer`/`Local`'s `Target` values genuinely carry no new information
   beyond what `Session.ID`/`Session.WidgetID` already provide; the sole
   benefit is type-based dispatch for a not-yet-built shared rendering
   component. This is disclosed as-is for pchero's final call, not
   something a design review can resolve on the author's behalf --
   product judgment on whether that single benefit justifies the schema
   addition belongs with pchero, not with this doc's author or reviewer.
3. **§4.4 NOT NULL vs. nullable**: RESOLVED -- nullable at the DB level.
   See §4.4's "Round 1 decision" note (BINARY(16)-to-UUID-string backfill
   formatting has no precedent in this codebase's migrations and is
   disproportionate cost for a short-lived, high-churn table where the
   app layer already guarantees non-empty values on every row created
   going forward).

## 8. Revision 1 (2026-07-22, post-closure, pchero-initiated): string value reverts to `web_session`; three dead `"web_session"` map entries deleted

This design doc closed cleanly after round 3 (2 consecutive APPROVE).
pchero then reviewed the final §4.1 decision and asked two things in the
same follow-up conversation, both accepted as-is:

1. **Delete the dead `"web_session"` entries** in the three
   `crmIneligiblePeerTypes` maps (`bin-flow-manager`, `bin-ai-manager`,
   `bin-contact-manager`) instead of leaving them as inert clutter.
2. **Revert §4.1's type-string decision**: `TypeWebSession`'s underlying
   string value goes back to `"web_session"` (NOT `"webchat_visitor"`,
   which was this design's round-1 decision after round-0 review flagged
   the collision risk).

This section documents the accepted rationale and the concrete updated
spec. Per this repo's post-closure-revision convention, §4.1's original
text is NOT rewritten in place -- the original round-0-through-3 review
history (why `"webchat_visitor"` was chosen, and that the collision-check
proof was independently re-verified three times) has audit value and
stays legible. This section is the CURRENT, authoritative spec; §4.1's
body text is superseded by it.

### 8.1 Rationale

The original concern (§4.1, unchanged in substance) was that a future
reader grepping for `"web_session"` would find it already reserved in
three `crmIneligiblePeerTypes` maps and could mistakenly assume a
collision. pchero's resolution removes the premise entirely: **delete
those three dead map entries** (confirmed dead -- see §8.2) rather than
picking an alternate string to avoid them. With the dead entries gone,
there is nothing left to collide with, and `"web_session"` becomes the
more natural, readable string value for `TypeWebSession` (it reads as
"web session" -- the visitor's session on the web channel -- rather than
`"webchat_visitor"`'s slightly more awkward compound).

### 8.2 Confirming the three map entries are actually dead (re-verified for this revision)

Re-confirmed directly against live source, same three locations §4.1
originally cited:

- `bin-flow-manager/pkg/activeflowhandler/actionhandle.go:1265`
- `bin-ai-manager/pkg/aicallhandler/tool.go:453`
- `bin-contact-manager/pkg/contacthandler/interaction.go:66`

Each is a single map entry, `"web_session": {}, // synthetic type; not in
commonaddress.Type enum`, inside a `crmIneligiblePeerTypes`/
`caseCreateCRMIneligiblePeerTypes` map otherwise populated entirely with
REAL `commonaddress.Type` constants (`TypeNone`, `TypeAgent`, `TypeAI`,
`TypeAITeam`, `TypeConference`, `TypeExtension`, `TypeSIP`). The bare
string literal `"web_session"` is the ONLY non-symbolic entry in all
three maps -- every other entry references an actual enum constant.

These maps are consulted only via each file's local `isCRMEligiblePeer`
(or `bin-ai-manager`'s equivalent), which is called against peer address
types that `deriveEndpointsForCase`/`deriveEndpoints` derive from
**Call**/**Conversation-message** webhook payloads
(`bin-call-manager`/`bin-conversation-manager`'s `Source`/`Destination`
fields). No code path anywhere in the monorepo constructs a
`commonaddress.Address` with `Type: "web_session"` today -- confirmed by
repo-wide search for the literal string outside these three map
definitions and their own test files. **Round-0 revision-review finding
(fixed): the search initially missed one additional test file**,
`bin-contact-manager/pkg/contacthandler/interaction_test.go`'s
`Test_EventConversationMessageCreated` (a "web_session" case at lines
247-266, testing `EventConversationMessageCreated`'s behavior when fed a
`Source.Type: "web_session"` webhook payload) -- this is a test
constructing the string as INPUT to exercise the (now-to-be-deleted)
ineligibility check, not a real production call site, so it does not
change the "no production caller uses this type" conclusion, but it DOES
mean this test must be updated/removed alongside the map deletion (see
§8.4) or it fails with a gomock panic once the entry is gone. Webchat
Conversations specifically use `TypeWebchat` for both self and peer
(`bin-conversation-manager/pkg/conversationhandler/event_webchat.go:88-89`,
unchanged and reconfirmed across every prior round). **The entries are
unreachable dead code in production paths**, not a latent behavior any
REAL caller currently depends on -- deleting them changes zero
PRODUCTION runtime behavior today, but does require updating the two
test files (§8.4) that exercise the deleted entries directly.

### 8.3 Updated spec (supersedes §4.1's text)

`bin-common-handler/models/address/main.go`:

```go
TypeWebSession Type = "web_session" // target is webchat-manager's Session.ID (the visitor's continuity token)
```

Delete the `"web_session"` entry from all three `crmIneligiblePeerTypes`-family
maps:

- `bin-flow-manager/pkg/activeflowhandler/actionhandle.go`: remove line
  1265 (`"web_session": {}, // synthetic type; not in commonaddress.Type enum`)
  from `crmIneligiblePeerTypes`.
- `bin-ai-manager/pkg/aicallhandler/tool.go`: remove the equivalent line
  453 from `caseCreateCRMIneligiblePeerTypes`.
- `bin-contact-manager/pkg/contacthandler/interaction.go`: remove the
  equivalent line 66 from `crmIneligiblePeerTypes`.

Each file's doc-comment on its map (e.g. `interaction.go:34-57`'s
extended comment explaining WHY each entry is CRM-ineligible) does not
need rewriting beyond the removed line itself -- the surrounding
rationale (agent extensions, conference legs, AI resources, PSTN
trunk legs, TypeNone/unknown-direction) is unaffected and still accurate
for the remaining entries.

**New consideration introduced by deleting these entries, not present in
§4.1's original analysis:** once `TypeWebSession = "web_session"` is a
REAL, live enum value (not just a bare string), and now that the
blacklist entries that would have excluded it are gone, if a FUTURE
change ever causes a Call/Conversation-derived peer to legitimately carry
`Type: TypeWebSession` (not the case today, and no such change is
proposed by this design), that peer would now be treated as
CRM-ELIGIBLE by `isCRMEligiblePeer` (since nothing excludes it), where
under the pre-revision state it would have been excluded (the dead
string literal would have matched). This is very unlikely given `Session`
is `bin-webchat-manager`'s own resource and Call/Conversation payloads
come from unrelated services, but is recorded here for completeness: a
future engineer wiring `TypeWebSession` into a Call/Conversation-derived
peer construction path should re-evaluate whether that specific case
warrants CRM-ineligibility, rather than assuming this deletion is
inert forever.

Every other occurrence of `"webchat_visitor"` in §4.1-§4.4/§5/§6 of this
document (the type constant's string value, the "Web visitor" render
label example, the `webchat_visitor`/`webchat` type-pair discussion) is
superseded by `"web_session"` per this revision. The Go symbol name
`commonaddress.TypeWebSession` is UNCHANGED -- only its underlying string
value reverts. §5's file checklist and §4.4's Go/DB field definitions
are otherwise unaffected (same fields, same nullable-at-DB-level
decision, same RPC-threading files) -- this revision changes exactly the
literal string value and adds the three dead-code deletions to the
implementation checklist below.

### 8.4 Updated §5 file checklist addendum

In addition to every file already listed in §5, this revision adds:

**bin-flow-manager:**
- `pkg/activeflowhandler/actionhandle.go` (delete the dead `"web_session"` map entry)
- `pkg/activeflowhandler/actionhandle_case_create_test.go` (if `Test_isCRMEligiblePeer` asserts on the now-removed entry, update it)

**bin-ai-manager:**
- `pkg/aicallhandler/tool.go` (delete the dead `"web_session"` map entry)
- `pkg/aicallhandler/tool_case_create_test.go` (same test-assertion check)

**bin-contact-manager:**
- `pkg/contacthandler/interaction.go` (delete the dead `"web_session"` map entry)
- `pkg/contacthandler/interaction_crm_eligibility_test.go` (this file's
  `Test_isCRMEligiblePeer` explicitly asserts `{"web_session is
  ineligible", commonaddress.Type("web_session"), false}` -- CONFIRMED
  via direct read, line 36 -- this specific test case must be removed or
  updated, since after deletion `isCRMEligiblePeer("web_session")` returns
  `true`, the opposite of what the existing assertion expects)
- `pkg/contacthandler/interaction_test.go` (**round-0 revision-review
  finding, added here**: `Test_EventConversationMessageCreated`'s
  "outgoing web_session message - peer is synthetic web session type -
  projection skipped" case, lines 247-266, constructs a webhook message
  with `Source: commonaddress.Address{Type: "web_session", ...}` and
  asserts `expectInteraction: nil` -- i.e. it relies on
  `isCRMEligiblePeer("web_session")` being `false` to assert NO
  Interaction row gets created (no `InteractionCreate` mock expectation
  set). After §8.3's deletion, `isCRMEligiblePeer("web_session")` returns
  `true`, so `EventConversationMessageCreated` would proceed to call
  `h.db.InteractionCreate` -- which this test's mock has NOT configured
  to expect, producing a gomock unexpected-call panic, not merely a wrong
  assertion. This test case must be removed or rewritten (e.g. renamed to
  a genuinely-ineligible type still in the map, or deleted entirely if it
  was solely testing the now-removed synthetic-type behavior) BEFORE the
  map-entry deletion lands, or `go test ./...` fails immediately.)

### 8.5 Review status of this revision

Per this repo's post-closure-revision convention, Revision 1 gets its own
fresh, narrowly-scoped review loop (2 consecutive APPROVE) rather than
inheriting the original round 0-3 closure -- a string-value change plus a
cross-service dead-code deletion touching three OTHER services' test
files is exactly the kind of change that needs independent
re-verification, not an assumption that the original loop's approval
still covers it. See round-by-round review files named
`2026-07-22-webchat-session-referrer-peer-local-design-review-revision1-round*.md`.
