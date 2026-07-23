# webchat Conversation/Case self/peer address-type unification (TypeWebSession)

Status: DRAFT (round 0)
Author: Hermes (CPO)
Date: 2026-07-23

## 1. Problem

pchero (CEO/CTO) noticed that webchat `Conversation`s are created with
`Self.Type == Peer.Type == TypeWebchat` (`"webchat"`), when the intent -- already
partially shipped for `webchat_sessions.Session` on 2026-07-22 (design doc
`bin-webchat-manager/docs/plans/2026-07-22-webchat-session-referrer-peer-local-design.md`)
-- is for the two roles to carry DIFFERENT `commonaddress.Type` values:

- **Self** (VoIPbin's own endpoint) = `TypeWebchat` (`"webchat"`), Target = Widget.ID
- **Peer** (the visitor) = `TypeWebSession` (`"web_session"`), Target = Session.ID

`Session.Peer`/`Session.Local` already do this correctly (shipped, merged). But
every OTHER call site that constructs a webchat self/peer pair still hardcodes
`TypeWebchat` on BOTH sides -- not because the underlying APIs can't accept two
different types (they always could; every self/peer parameter pair in this
codebase is two fully independent `commonaddress.Address` values), but because
each caller independently re-derived the pair with a copy-paste bug instead of
reusing the already-correct `Session.Peer`/`Session.Local` (or, for
`case_message.go`, an already-correct-but-unused `Case.Local` field).

This is a **caller-side hardcoding bug in three call sites**, not an API
limitation. No RPC signature changes.

## 2. Goal / Non-goals

**Goal:** every webchat-originated `Conversation.Peer.Type` becomes
`TypeWebSession` (Target = Session.ID), `Conversation.Self.Type` stays
`TypeWebchat` (Target = Widget.ID), matching `Session.Peer`/`Session.Local`
exactly. Case-linking (`case_message.go`) is fixed to stop forcing self/peer to
match, so it stops producing spurious duplicate Conversations on every
agent-initiated webchat reply -- this is folded into the same design because it
shares the identical root cause (a caller ignoring an already-correct value it
has in hand) and the identical fix shape (stop hardcoding, read the value that's
already there).

**Non-goals (deferred, per 2026-07-22 design doc §4.3's already-established
precedent):**
- No CODE change to how `kase.Case.Peer`/`.Local` are constructed. This
  design does not touch `casehandler.Create`/`getOrCreate`/`insertWithRetry`,
  nor `actionHandleCaseCreate`'s `call`-reference branch
  (`deriveEndpointsForCase`). **Important correction (added in round 1
  review):** this is NOT the same as "Case's stored values are unaffected."
  `actionHandleCaseCreate`'s `conversation`-reference branch
  (`bin-flow-manager/pkg/activeflowhandler/actionhandle.go:1338`, `peer, self
  = cv.Peer, cv.Self`) and the identical AI-tool path
  (`bin-ai-manager/pkg/aicallhandler/tool.go:503`) read a Conversation's
  CURRENT `Peer`/`Self` and copy them verbatim into a new/reused Case. Since
  §4.B changes what a webchat Conversation's `Peer.Type` IS, every NEW Case
  created via `case_create` against a webchat Conversation, from deploy
  onward, automatically gets `Local.Type = TypeWebchat` / `Peer.Type =
  TypeWebSession` too -- not because this design edits Case-construction
  code, but as a direct, unavoidable consequence of §4.B. See §5.1 (new) for
  the cutover implication this creates for `uq_case_open_peer` dedup, folded
  into the same accept-the-blip decision as §5's Conversation-side risk.
  This is NOT a scope violation to "fix" (there is no code to change here --
  `actionHandleCaseCreate` correctly mirrors whatever the Conversation
  currently holds, exactly as designed on 2026-07-07); it is a disclosed,
  accepted side effect of §4.B, and this doc's non-goal boundary is now
  worded to be accurate about it rather than implying Case is fully inert.
- `kase.Case` schema/dbhandler code itself (columns, `uq_case_open_peer`
  generated-column dependency chain) is unchanged -- the migration-risk
  argument from the 2026-07-22 doc still holds for NOT proactively
  backfilling or restructuring Case storage; only the VALUES flowing into
  the existing `Peer`/`Local` columns for NEW conversation-triggered Cases
  change, as a pass-through of §4.B, not a schema or code change here.
- No retroactive backfill of already-created Conversation rows whose
  `peer.type = webchat` predates this change (see §6 cutover handling instead).
- No change to non-webchat channels (tel/line/whatsapp/email) -- their
  self.Type == peer.Type invariant is correct and untouched.
- No change to `bin-openapi-manager`'s `CommonAddress.type` enum (it already
  lacks BOTH `webchat` and `web_session` -- a pre-existing, unrelated drift;
  `Type` is generated as `*string` so no runtime validation is gated on this,
  confirmed no `OapiRequestValidator` middleware in use).

## 3. Root cause -- three independent call sites, one bug pattern

| # | File:lines | What it does today | Already-correct value it ignores |
|---|---|---|---|
| A | `bin-webchat-manager/pkg/sessionhandler/create.go:81-82` | Builds `self`/`peer` locals from scratch (`TypeWebchat` both) for the `ConversationV1ConversationCreateAndExecuteFlow` RPC call (SessionFlowID trigger) | `s.Local`/`s.Peer` (lines 50-51), computed 30 lines earlier for the `Session{}` struct literal, already correct |
| B | `bin-conversation-manager/pkg/conversationhandler/event_webchat.go:88-89` (inbound) and `:160-161` (outbound) | Builds `self`/`peer` locals from `wm.WidgetID`/`wm.SessionID` (`TypeWebchat` both) | Nothing pre-computed to reuse here (event payload doesn't carry a `Session`) -- but the SAME fix (`TypeWebSession` for peer) is a two-token change |
| C | `bin-api-manager/pkg/servicehandler/case_message.go:165-172` | Builds `selfAddr`/`peerAddr` both from `c.Peer.Type` (a single value) | `c.Local.Type` for self (the `kase.Case.Local` field, already populated by every Case writer -- `casehandler.Create`/`insertWithRetry`/`actionHandleCaseCreate`) |

Site C is architecturally distinct from A/B (it derives from a `Case`, not a
`Session`), but the underlying defect is identical in shape: a value that is
ALREADY correctly available (`c.Local`) is being ignored in favor of copying
`c.Peer.Type` onto both sides. §2's non-goal boundary matters here: `c.Local`
itself is still `TypeWebchat` (unchanged by this design, since Case storage is
out of scope) -- but `TypeWebchat` IS the correct value for Conversation.Self,
so reading it is sufficient; the fix does not require Case's stored value to
change at all.

## 4. Fix

### 4.A `bin-webchat-manager/pkg/sessionhandler/create.go`

```go
// before (lines 81-82):
self := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: widgetID.String()}
peer := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: id.String()}

// after: reuse the already-computed, already-correct Session fields
self := s.Local
peer := s.Peer
```

`s.Local`/`s.Peer` are populated at lines 50-51 (`Local: {TypeWebchat,
widgetID}`, `Peer: {TypeWebSession, id}`) -- byte-identical Target values to
what the old code computed inline, only the Peer role's Type changes. No other
line in this function needs to change.

### 4.B `bin-conversation-manager/pkg/conversationhandler/event_webchat.go`

Both `messageEventReceivedWebchat` (inbound, lines 88-89) and
`messageEventSentWebchat` (outbound, lines 160-161):

```go
// before:
self := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: wm.WidgetID.String()}
peer := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: wm.SessionID.String()}

// after:
self := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: wm.WidgetID.String()}
peer := commonaddress.Address{Type: commonaddress.TypeWebSession, Target: wm.SessionID.String()}
```

Two-token change, both call sites, identical edit.

### 4.C `bin-api-manager/pkg/servicehandler/case_message.go`

```go
// before (lines 165-172):
selfAddr := commonaddress.Address{
    Type:   c.Peer.Type,
    Target: source,
}
peerAddr := commonaddress.Address{
    Type:   c.Peer.Type,
    Target: destination,
}

// after:
selfAddr := commonaddress.Address{
    Type:   c.Local.Type,
    Target: source,
}
peerAddr := commonaddress.Address{
    Type:   c.Peer.Type,
    Target: destination,
}
```

This ALSO fixes the pre-existing, channel-agnostic bug the 2026-07-23 review
loop surfaced (before this design doc's scope was set): for tel/line/whatsapp/
email Cases, `c.Local.Type` and `c.Peer.Type` were already the SAME value
today (both were forced to the channel type), so this change is a no-op for
those channels TODAY -- but it stops being a no-op the moment ANY channel has
Local.Type != Peer.Type, which is exactly webchat's case today and may not be
the last such case. **Scope correction (round 1 review):** the claim "reading
`c.Local.Type` is strictly more correct for every channel" is verified only
for the channels actually exercised today (webchat, and tel/line/whatsapp/
email via the `call`-reference branch, where `Local.Type == Peer.Type`
because the underlying `Call.Source`/`Call.Destination` addresses happen to
share the same Type for these channels -- a property of the upstream Call
data, not something `deriveEndpointsForCase` itself constructs or enforces:
that function only reorders `source`/`dest` into `peer`/`self` based on
`direction`, with no Type-equality logic of its own). It is NOT a proven general
guarantee against a hypothetical future Local.Type != Peer.Type Case outside
webchat's `conversation`-reference path -- if one existed today with mismatched
types not backed by a real Conversation row, `case_message.go`'s
`ConversationV1ConversationGetOrCreateBySelfAndPeer` call could construct a
self/peer pair matching no existing Conversation and create a spurious one
(the exact failure mode the OLD comment warned about, now scoped narrower
but not eliminated in the abstract). No such case is known to exist today
(grep-confirmed: `call`-reference and `conversation`-reference are the only
two branches in `actionHandleCaseCreate`, both verified type-consistent), so
this is accepted as a theoretical, not live, residual risk.

The 157-164-line comment block (self/peer-must-match rationale) becomes
stale and must be rewritten to explain the corrected invariant: self and peer
each independently mirror the Case's own Local/Peer, which are NOT required to
share a Type -- conversation-manager's own webchat construction (§4.B) already
proves self.Type != peer.Type is a supported, correct shape, not a special
case to route around.

The comment's original safety concern (`ConversationGetBySelfAndPeer`'s exact
Type match) is still valid and still satisfied: `c.Local.Type`/`c.Peer.Type`
are read from the SAME Conversation-derived Case (§1338 of
`actionhandle.go`'s `actionHandleCaseCreate`, `peer, self = cv.Peer,
cv.Self`), so they already match whatever the originating Conversation's
actual self/peer types are -- the lookup will hit correctly by construction,
not by the removed hardcoded-equality trick.

### Other direct consumers checked (round 1 review additions)

- `bin-conversation-manager/internal/convtitle/build.go`: `channelLabel`
  switches on `conversation.Type` (unaffected), `humanReadableTarget`
  switches on `Address.Type` but has no `TypeWebSession` case -- falls to
  `default: false` (opaque target), which is the CORRECT behavior for a
  visitor session UUID (matches `TypeWebchat`'s existing opaque treatment).
  No code change needed here, but noting it was checked.
- `monorepo-javascript` square-admin: `views/conversation_conversations/
  detail.js` renders Conversation `Peer.Type`/`Peer.Target` as opaque
  read-only strings, no `"webchat"` string-equality branch -- unaffected.
  **However**, per §2's corrected non-goal, NEW Cases created via
  `case_create` against a webchat Conversation will carry `Peer.Type =
  "web_session"` going forward; `views/contact_cases/contact_cases_list.js`
  (or equivalent Case list/detail view) that displays `kase.peer.type` as
  plain text will start showing `"web_session"` instead of `"webchat"` for
  those NEW Cases (old Cases keep showing `"webchat"`, matching their stored
  value). This is a display-string change only (no branching logic on that
  view was found), same severity class as §5's Conversation-side cutover
  cosmetic; no code change required, disclosed here for completeness.

No change required. This function maps `peer.Type` (unaffected by §4.C, still
`c.Peer.Type` = `TypeWebchat` for webchat Cases per §2's non-goal) to
`conversation.Type`. It already has no `TypeWebchat` case and falls through to
`TypeMessage` -- a PRE-EXISTING, separate defect (flagged in the prior
analysis session, 2026-07-23 05:39 thread) that is explicitly OUT OF SCOPE
here: fixing it changes which `conversation.Type` webchat Case-replies
resolve to, a materially different blast radius than this design's pure
self/peer Type fix. Filed as a follow-up, not blocking this design.

## 5. Cutover risk: existing Conversation rows AND conversation-triggered Cases (DECIDED: no fallback, accept the blip)

### 5.1 Case-side cutover (new, round 1 review finding)

Per §2's corrected non-goal: `uq_case_open_peer` dedups on `(customer_id,
peer_type, peer_target, reference_type, status='open')`. A webchat visitor
with an open Case created BEFORE this deploy has `peer_type = "webchat"`
stored. If that same visitor's Conversation continues past cutover and a
`case_create` Flow action or AI tool fires again against it (`peer, self =
cv.Peer, cv.Self` now yields `peer_type = "web_session"`), the dedup lookup
against the OLD `peer_type = "webchat"` row misses, and a second, parallel
open Case is created for the same visitor. This is the identical class of
risk as §5's Conversation-side blip, one hop downstream, and is accepted
under the same decision (§10): no fallback, no backfill. Operationally, a
double-Case for a cutover-era visitor is a one-time, self-resolving
inconvenience (both Cases eventually time out / get manually merged by an
agent), not a data-loss or security issue.

### 5.2 Conversation-side cutover (original)

`ConversationGetBySelfAndPeer`'s SQL (`bin-conversation-manager/pkg/dbhandler/
conversation.go:155-164`) is an exact-match lookup on both
`self.type`/`peer.type`. Every webchat Conversation created BEFORE this
change's deploy has `peer.type = "webchat"` stored; after deploy, callers look
up with `peer.type = "web_session"`. This mismatches on the very next message
for every conversation that was active at cutover, producing:

- A miss -> a brand-new Conversation is created (`GetOrCreateBySelfAndPeer`'s
  create-on-miss path, `db.go:90-109`).
- `owner_id` resets to `{OwnerTypeNone, uuid.Nil}` (`Create` never sets Owner).
- `Metadata.ContactCaseID` resets to zero-value (`Create` never sets
  Metadata) -- breaks the case_id hint on the next message
  (`caseIDHint(cv)` in `event_webchat.go`).
- A `conversation_created` webhook fires again for what is, from the
  customer's perspective, a continuation of an existing thread.

**pchero's decision (2026-07-23):** no fallback query. Webchat's current
production traffic volume is low enough that a one-time cutover blip (new
Conversation created on the next message for any thread active at deploy
time, with the owner/metadata/webhook-noise side effects described above) is
acceptable. Deploy §4's fix as-is; `bin-conversation-manager/pkg/dbhandler/
conversation.go` is NOT touched by this design. If webchat traffic grows to a
point where this class of cutover ever recurs (e.g. a future similar
Type-value change), revisit the fallback-query pattern as its own design at
that time -- not preemptively built here for a one-time event.

## 6. Flow variable and webhook payload value change (DECIDED: deploy without advance notice)

`voipbin.conversation.peer.type` (`variable.go:45`) and
`WebhookMessage.Peer.Type` (`models/conversation/webhook.go:27`) both surface
the raw `Peer.Type` value to customer-authored Flow scripts and webhook
consumers. This design changes that value from `"webchat"` to `"web_session"`
for every NEW webchat conversation going forward, and (per §5.2's decision)
also for cutover-era conversations the moment they receive their first
post-deploy message and miss the exact-match lookup (a fresh Conversation is
created with the new value at that point -- not a fallback finding the old
value, since no fallback exists). If any customer's Flow condition branches
on `voipbin.conversation.peer.type == "webchat"`, or any webhook consumer
string-matches `peer.type`, that logic silently breaks at deploy time.

**pchero's decision (2026-07-23):** deploy without advance customer
communication. Webchat's current customer traffic is low enough that this
disclosed risk (any customer Flow condition or webhook consumer literally
matching `peer.type == "webchat"` would silently stop matching post-deploy)
is accepted as-is. No comms action item created.

## 7. Files touched

**bin-webchat-manager:**
- `pkg/sessionhandler/create.go` (§4.A)
- `pkg/sessionhandler/create_test.go` (update expected self/peer types)

**bin-conversation-manager:**
- `pkg/conversationhandler/event_webchat.go` (§4.B, both functions)
- `pkg/conversationhandler/event_webchat_test.go` (update expected self/peer types)
- `pkg/conversationhandler/create_and_execute_flow_test.go` (round 1 review
  addition: lines 48-49, 129-130, 180-181, 234-235 hardcode `TypeWebchat` for
  what becomes a `TypeWebSession`-typed peer -- must update)

**bin-api-manager:**
- `pkg/servicehandler/case_message.go` (§4.C, incl. stale comment rewrite)
- `pkg/servicehandler/case_message_test.go` (update expected self.Type; add a
  regression test asserting `selfAddr.Type == c.Local.Type`, not `c.Peer.Type`,
  mirroring the existing `Test_CaseMessageSend_SelfAndPeerTypeMatch_WhatsApp`
  pattern)

## 8. Testing plan

- Unit: update every existing test asserting `self.Type == TypeWebchat &&
  peer.Type == TypeWebchat` for webchat conversations (`create_test.go`,
  `create_and_execute_flow_test.go`, `event_webchat_test.go`,
  `case_message_test.go`) to the new expected pair. (Round 3 review
  correction: `execute_mode_test.go`'s `TypeWebchat` occurrences at lines
  481/509/525/544 are all `cv.Self` fixture values for
  `runExecuteModeFlowWebchat`, a function that reads only `cv.Self.Target`
  and never references `cv.Peer` -- confirmed via `execute_mode.go:137-150`.
  That file is unrelated to this design and needs no change; it was
  incorrectly listed as a Round 1 finding and has been removed from §7.)
- New regression test (mirrors `Test_GetOrCreateBySelfAndPeer_NormalizesSelfPeer`'s
  table-driven shape): webchat self/peer construction produces
  `{TypeWebchat, widgetID}` / `{TypeWebSession, sessionID}` at all three call
  sites (A/B/C), not `{TypeWebchat, x}` / `{TypeWebchat, y}`.
- No new integration/E2E test needed -- webchat's existing
  `getorcreate_*_test.go` suite in `bin-contact-manager/pkg/casehandler`
  already covers Case-side get-or-create behavior and is unaffected (§2
  non-goal: Case storage unchanged).

## 9. Rollout

Single PR across `bin-webchat-manager`, `bin-conversation-manager`,
`bin-api-manager` (three services, one logical change, matches this repo's
existing convention of grouping directly-coupled cross-service diffs into one
PR rather than three sequenced ones -- there is no safe intermediate state
where only one of the three sites is fixed, since a Session-triggered Flow
(§4.A) and an inbound/outbound message (§4.B) both feed the SAME Conversation
row and must agree on Peer.Type simultaneously). No feature flag: this is a
correctness fix with no user-facing toggle to gate.

## 10. Decisions log

1. §5 fallback query: **excluded** (pchero, 2026-07-23) -- one-time cutover
   blip accepted, scope minimized.
2. §6 Flow-variable/webhook-payload value change: **no advance customer
   communication** (pchero, 2026-07-23) -- webchat traffic low enough to
   accept as-is.
