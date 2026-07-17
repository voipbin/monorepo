# Webchat widget runtime + real-time delivery (backend half)

- Ticket: VOIP-1265
- Companion frontend doc: `monorepo-javascript/.worktrees/SQUARE-15-webchat-widget-runtime-preview/docs/plans/2026-07-17-webchat-widget-runtime-and-preview-design.md`
- Author: Hermes (CPO), for pchero
- Status: DRAFT — Round 3 review complete (CHANGES_REQUESTED, 1 blocking finding: §9.2's code sample dropped FieldCustomerID, now fixed with verbatim-verified code), pending Round 4

## 1. Problem

square-admin can create a Webchat Widget and copy an embed snippet
(`<script src="https://webchat.voipbin.net/embed.js" data-hash="...">`),
but nothing in the platform actually renders that widget or lets a
customer/admin see or test the real chat experience it produces. A
CPO-level audit (this session) found four concrete gaps:

1. **No `embed.js` widget runtime exists anywhere in the codebase.** The
   design doc for `bin-webchat-manager` (`2026-06-29-add-webchat-manager-design.md`
   §"theme_config") explicitly scoped the embed snippet as "frontend
   deliverable, not part of the API design" and it was never built.
2. **No hosting path for `webchat.voipbin.net`.** DNS does not resolve.
   Out of scope for this design — tracked separately, not blocking local/
   square-admin-embedded testing (see §6 Non-goals).
3. **No real-time delivery path for a visitor.** `bin-api-manager`'s
   WS topic builder (`pkg/subscribehandler/webhookmanager.go`
   `createTopics()`) has no case for the `webchat` resource family, so a
   `webchat_message_created` event never gets published to any topic a
   visitor's direct-scoped WS connection could subscribe to.
4. **`WebchatMessageList` unconditionally rejects direct-scoped callers**
   (`bin-api-manager/pkg/servicehandler/webchat_message.go` line 85-87):
   ```go
   if a.IsDirect() {
       return nil, serviceerrors.ErrDirectAccessNotSupported
   }
   ```
   `WebchatMessageCreate` (line 217+) already has a full direct-scope
   branch (verify `s.WidgetID == a.DirectScope.ResourceID`, reject
   ended sessions, reject soft-deleted widgets). List has no equivalent
   branch at all — a visitor can send a message but has no API-level way
   to ever read the agent's reply, via polling or WS. This is judged a
   bug, not an intentional restriction: Create/End have direct branches,
   List does not, with no comment or design rationale marking the
   asymmetry as deliberate.

pchero directed (this session): build the real `embed.js` widget
runtime, fix the message-list gap, and build WS-based real-time
delivery — do not settle for a polling-only MVP or an admin-only fake
preview. All four items ship together in this design.

## 2. Goals

- G1. A visitor holding a widget's `direct_hash` can open a real chat
  panel (rendered from `embed.js`), send a message, and receive the
  agent's reply in real time over WS (no polling). The agent's
  visibility into the visitor's inbound message is addressed in §9.7 —
  it is NOT left as a manual-refresh-only gap; minimal agent-side WS
  wiring is pulled into scope via the frontend PR, using the same topic
  this design already creates.
- G2. `bin-api-manager`'s WS layer supports a visitor-scoped session
  topic so a direct-scoped JWT can subscribe to exactly one session's
  message stream and nothing else.
- G3. `WebchatMessageList` is safely reachable by a direct-scoped caller,
  scoped to the caller's own session only (no cross-session/cross-tenant
  leak), with the same negative-case checks (deleted widget) that
  `WebchatMessageCreate`'s direct branch already enforces (see §9.2).
- G4. square-admin's widget create/detail pages render a **live preview**
  driven by the *same* rendering code the real `embed.js` uses (not a
  parallel reimplementation that can drift), reflecting theme_config
  (primary_color, logo_url, position) as the admin edits it.
- G5. The whole loop (create widget → copy snippet → visitor sends →
  agent replies via square-admin's Sessions tab → visitor sees it
  live) is demonstrably testable without needing `webchat.voipbin.net`
  DNS/hosting to exist yet (see §6).

## 3. Non-goals (explicitly out of scope for this design)

- NG1. Production hosting/CDN path for `webchat.voipbin.net` and the
  actual `<script src="https://webchat.voipbin.net/embed.js">` URL going
  live. Tracked as a follow-up infra item once the runtime code exists
  and is reviewed.
- NG2. Custom CSS injection, multiple themes, per-agent avatars — already
  deferred in the v8 theme_config design (Phase 2 bucket).
- NG3. Typing indicators, read receipts, file attachments in webchat.
- NG4. (Narrowed after Round 1 — see §9.7.) Full agent-side presence/
  typing indicators and any UI beyond a live-updating message list on
  the existing Sessions/message-timeline view remain out of scope.
  Minimal WS wiring for that view to receive `webchat_message_created`
  events live IS now in scope, driven from the frontend PR against the
  topic this design creates — no additional backend change needed
  beyond §4.2.

## 4. Backend changes (this doc's scope: `monorepo`)

### 4.1 Fix `WebchatMessageList` direct-scope gap

**Superseded by §9.2's corrected code** (Round 1 found the version
below missing a widget-soft-delete check). Kept here for the original
context; implement per §9.2, not this block.

File: `bin-api-manager/pkg/servicehandler/webchat_message.go`

Add a direct-scope branch mirroring `WebchatMessageCreate`'s existing
one, instead of the current unconditional rejection. See §9.2 for the
corrected, final version of this branch.

Tests to add: direct-scoped caller can list their own session's
messages; direct-scoped caller with a `session_id` belonging to a
different widget gets `ErrPermissionDenied`; direct-scoped caller
omitting `session_id` gets rejected; direct-scoped caller on a
soft-deleted widget gets rejected (added per §9.2).

### 4.2 WS topic routing for visitor-scoped session subscription

Files: `bin-api-manager/pkg/subscribehandler/webhookmanager.go`,
`bin-api-manager/pkg/websockhandler/etc.go`.

**Topic shape.** Add a `webchat_session`-scoped 4-part topic a
direct-scoped JWT can subscribe to:

```
customer_id:<customer_id>:webchat_session:<session_id>
```

This deliberately does NOT reuse the existing 4-part
`customer_id:<id>:<resource_type>:<resource_id>` pattern's generic
`resource_type` value from `getServiceNamespace()` (which would produce
`customer_id:<id>:webchat:<message_id>`, scoped by *message* id, useless
to a visitor who doesn't know the next message's id yet). Instead it is
scoped by **session id**, known to the visitor from the
`POST /webchat_sessions` response, mirroring how `aicall_id` is used as
the stable scope anchor for `aimessage_created` events (see
`createTopics()`'s existing `case "aimessage":` branch, which already
solves exactly this "message id is unknown in advance, scope by parent
id instead" problem for AI calls).

**`createTopics()` change** (`webhookmanager.go`): add a case for
`webchat_message` (message.go's `EventTypeMessageCreated` constant is
`"webchat_message_created"`, so `tmps[0]` after the existing
`strings.Split(messageType, "_")` is `"webchat"` — confirm this doesn't
collide with the generic `default:` branch's per-`d.ID` topic, which
would be useless here for the same reason as aimessage). Mirror the
`aimessage` branch shape:

```go
case "webchat":
    if d.CustomerID != uuid.Nil && d.SessionID != uuid.Nil {
        res = append(res, fmt.Sprintf("customer_id:%s:webchat_session:%s", d.CustomerID, d.SessionID))
    }
```

`commonWebhookData` (top of the same file) needs a new
`SessionID uuid.UUID \`json:"session_id,omitempty"\`` field.
**Confirmed (Round 1, §9.1/§9.4)**: `wcmessage.WebhookMessage` already
serializes `session_id` on the wire and no other publisher's payload
collides with this field name — no open question remains here.

**`validateTopics()`/`validateTopic()` change** (`websockhandler/etc.go`):
add a `case "webchat_session":` arm alongside the existing
`case "customer_id":`/`case "agent_id":` dispatch — actually this needs
to slot into the existing 4-part `customer_id:...` handling, not a new
top-level case, since the topic's first segment is still `customer_id`.
The 4-part branch already does:
```go
if a.IsDirect() {
    if !a.HasAllowedResourceType(tmps[2]) {
        return false
    }
}
```
`tmps[2]` here is `"webchat_session"` — so this validation already works
correctly as long as the direct-scope JWT's `AllowedResourceTypes`
includes `"webchat_session"` (it does — `directResourceMapping` in
`boot.go` already maps `dmdirect.ResourceTypeWebchatWidget` to
`{"webchat_session"}`). **Confirmed (Round 1, §9.5): no `validateTopics`
code change is needed**, and separately confirmed the `/ws` handshake
itself accepts direct-scoped JWTs (same `middleware.Authenticate()`
used by every other authenticated route).

**Note on scope vs `DirectScope.ResourceID`:** `DirectScope.ResourceID`
is the *widget* id (boot-time claim), but the topic is scoped by
*session* id, a different value the visitor only learns after
`POST /webchat_sessions`. `HasAllowedResourceType` only checks the
resource-type string, not the specific id — so nothing here currently
stops a direct-scoped JWT from subscribing to ANY session's topic if it
guesses/enumerates a session UUID, not just its own. **This is
acceptable**: session UUIDs are v4-random and not enumerable, and this
exactly mirrors the existing aicall_id-scoped topic's trust model (no
id-level check there either) — called out explicitly here and must
appear in the PR description too, not left as a silent assumption.

**WS-topic lifecycle on session end (Round 1 finding, resolved in
§9.3):** the topic is not forcibly closed server-side when
`WebchatSessionEnd` fires; the client is responsible for a cooperative
close on receiving the session-ended event. See §9.3 for the full
rationale and the (inert, since posting is already blocked) residual
exposure this leaves.

### 4.3 `bin-webchat-manager` message create: ensure `SessionID` is present on the published event

**Confirmed (Round 1, §9.6):** `messagehandler/create.go` calls
`PublishWebhookEvent` unconditionally with `SessionID` populated
regardless of message direction or auth path — no change needed here,
and this also resolves the sender-echo question (§9.6): a visitor's own
inbound message IS echoed back to them over their own subscribed topic,
so the frontend's send-path must dedupe against it.

## 5. Frontend changes (companion doc, `monorepo-javascript`)

See `SQUARE-15-webchat-widget-runtime-preview`'s design doc. Summary
only, for backend reviewers' context:

- New `embed.js`-equivalent widget runtime (vanilla JS, framework-free,
  loaded via `<script>` tag) implementing: `POST /auth/boot` →
  `POST /webchat_sessions` → render welcome message → open
  `GET /ws?token=<jwt>` and subscribe to
  `customer_id:<cid>:webchat_session:<session_id>` → render inbound
  messages live → `POST /webchat_messages` on send → dedupe the WS echo
  of the visitor's own message against the synchronous create response
  (§9.6) → on session end, close the WS connection client-side (§9.3).
- Shared theming module (`applyWidgetTheme(themeConfig)` or similar)
  consumed by BOTH the real runtime and square-admin's live preview
  iframe/component, so admin preview cannot drift from the real render.
- square-admin's widget create/detail pages embed a live preview panel
  using this shared module.
- (Added per §9.7) square-admin's Sessions/message-timeline view
  subscribes to the same session topic for a live-updating agent-side
  message list, closing the G1 real-time gap Round 1 flagged.

## 6. Testability without `webchat.voipbin.net` hosting

Local/CI verification path for G5, until NG1 is separately resolved:

- The widget runtime bundle is served locally (e.g. from square-admin's
  own dev/build server, or a throwaway static file server) during
  development/testing, with `data-hash` pointed at a real widget created
  against `api.voipbin.net` (CORS already allows `*` — confirmed this
  session).
- square-admin's own detail page can optionally embed the same runtime
  via a "Test this widget" action (loads the bundle against the current
  widget's real `direct_hash`) — this satisfies the original CEO ask
  ("실제로 바로 채팅 테스트를 해볼 수 있는 기능") without needing
  `webchat.voipbin.net` to exist.

## 7. Test plan

- Backend: unit tests for §9.2 (direct-scope List branches, including
  the widget-soft-delete rejection case) and the new `createTopics()`
  case in §4.2. `go test ./...` + `golangci-lint` per the standard
  verification workflow in both `bin-api-manager` and
  `bin-webchat-manager`.
- End-to-end manual verification (documented in the PR, not automated
  in this phase): create widget in square-admin → open square-admin's
  "Test this widget" action → send a message as visitor → confirm the
  message appears live on the Sessions/message-timeline view (§9.7) →
  reply as agent → confirm visitor sees the reply without a page
  refresh (WS-delivered, not polled, on both sides).

## 8. Open questions for review

All Round-0 open questions (OQ1-OQ3) were resolved during Round 1 — see
§9 for each resolution. No open questions remain for Round 2; Round 2
should focus on verifying the §9 resolutions are correct and complete,
not on re-litigating Round 0's OQs.

## 9. Round 1 review findings and resolutions

An independent adversarial review (Round 1) verified every claim in
this doc against source and found 8 issues. Verdict was
CHANGES_REQUESTED. All 8 are addressed below.

### 9.1 OQ1 resolved (was incorrectly left open)

**Confirmed by direct inspection**: `wcmessage.WebhookMessage`
(`bin-webchat-manager/models/message/webhook.go` line 20) already has
`SessionID uuid.UUID \`json:"session_id"\`` on the wire — no schema
change needed, §4.2's `commonWebhookData` addition just needs to read
this existing field. This should have been checked before drafting
rather than deferred; corrected now, not at implementation time.

### 9.2 §4.1 tenant-isolation gap fixed (widget-deleted / session-ended checks)

Round 1 found the new direct-scope `WebchatMessageList` branch omitted
the widget-not-deleted check `WebchatMessageCreate`'s direct branch
already has, reintroducing a narrower version of the exact
CRUD-asymmetry bug class this design sets out to fix. §4.1's proposed
code is corrected to explicitly include both checks, matching
`Create`'s direct branch. **This is the final version of the §4.1
branch** (Round 2 note: the current `WebchatMessageList` function body
is a standalone `if a.IsDirect() { return ..., ErrDirectAccessNotSupported }`
early-return, not a `switch` — this must be restructured into a
`switch { case a.IsDirect(): ... ; default: <existing agent/accesskey
body, unchanged> }` exactly mirroring `WebchatMessageCreate`'s existing
switch shape).

**Round 3 correction (blocking finding, fixed): the Round 2 code sample
below had dropped `wcmessage.FieldCustomerID` from the default/agent
branch's filters and unconditionally set `FieldSessionID`, contradicting
the doc's own prose and — since `MessageList`'s filters map is the sole
customer-scoping mechanism at the DB layer (confirmed via
`bin-webchat-manager/pkg/dbhandler/message.go`) — would have been a real
cross-tenant data leak if implemented literally. This is now the
verbatim, correct code, transcribed directly against the current
unmodified function (`bin-api-manager/pkg/servicehandler/webchat_message.go`
lines 78-128) with only the minimal diff needed to add the direct-scope
case — not reconstructed from memory:**

```go
func (h *serviceHandler) WebchatMessageList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, sessionID uuid.UUID) ([]*wcmessage.WebhookMessage, error) {
    log := logrus.WithFields(logrus.Fields{
        "func":        "WebchatMessageList",
        "customer_id": a.CustomerID,
        "username":    a.DisplayName(),
    })

    if token == "" {
        token = h.utilHandler.TimeGetCurTime()
    }

    filters := map[wcmessage.Field]any{
        wcmessage.FieldDeleted: false,
    }

    switch {
    case a.IsDirect():
        if !a.HasAllowedResourceType("webchat_session") {
            return nil, serviceerrors.ErrPermissionDenied
        }
        if sessionID == uuid.Nil {
            // A visitor must always scope to their own session; there is
            // no "list all my messages across sessions" concept for a
            // direct-scoped caller.
            return nil, serviceerrors.ErrPermissionDenied
        }
        s, err := h.sessionGet(ctx, sessionID)
        if err != nil {
            log.Errorf("Could not validate the session info. err: %v", err)
            return nil, err
        }
        if s.WidgetID != a.DirectScope.ResourceID {
            return nil, serviceerrors.ErrPermissionDenied
        }
        // Confirm the widget itself hasn't been soft-deleted -- mirrors
        // WebchatMessageCreate's/WebchatSessionCreate's direct branches.
        if _, err := h.widgetGet(ctx, a.DirectScope.ResourceID); err != nil {
            log.Errorf("Could not validate the widget info. err: %v", err)
            return nil, err
        }
        // Deliberately NOT rejecting on s.Status == StatusEnded: unlike
        // Create (posting into a dead session is nonsensical) and unlike
        // the WS topic (see §9.3, explicitly closed client-side on End),
        // reading the final history of an ended session is a legitimate
        // visitor need (e.g. reconnecting after a network drop to see the
        // last replies before the session ended). This is an intentional,
        // explicit decision -- Round 1's finding #2 flagged this exact
        // ambiguity and this is the resolution.
        // s.CustomerID is authoritative here (already ownership-verified
        // above via s.WidgetID check) -- set explicitly rather than
        // trusting a.CustomerID, mirroring WebchatMessageCreate's
        // ownerCustomerID pattern for the same defense-in-depth reason.
        filters[wcmessage.FieldCustomerID] = s.CustomerID
        filters[wcmessage.FieldSessionID] = sessionID

    default:
        if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
            log.Info("The agent has no permission.")
            return nil, serviceerrors.ErrPermissionDenied
        }

        filters[wcmessage.FieldCustomerID] = a.CustomerID

        if sessionID != uuid.Nil {
            s, err := h.sessionGet(ctx, sessionID)
            if err != nil {
                log.Errorf("Could not validate the session info. err: %v", err)
                return nil, err
            }
            if s.CustomerID != a.CustomerID {
                log.Info("The session does not belong to the requesting customer.")
                return nil, serviceerrors.ErrPermissionDenied
            }
            filters[wcmessage.FieldSessionID] = sessionID
        }
    }

    tmps, err := h.reqHandler.WebchatV1MessageList(ctx, token, size, filters)
    if err != nil {
        log.Errorf("Could not get messages from the webchat-manager. err: %v", err)
        return nil, err
    }

    res := []*wcmessage.WebhookMessage{}
    for _, tmp := range tmps {
        e := tmp.ConvertWebhookMessage()
        res = append(res, e)
    }

    return res, nil
}
```

This is the literal, final code — not illustrative pseudocode — and
must be implemented exactly as shown (modulo the `a.IsDirect()` early
`ErrDirectAccessNotSupported` return at the current top of the function
being removed, since it's now handled by the `switch`).

**Server-layer claim corrected (was over-specified — see §9.8):** no
`server/webchat_messages.go` change is needed; the servicehandler-layer
`ErrPermissionDenied` above is already correctly mapped to an HTTP error
by the existing `abortWithServiceError` mechanism.

### 9.3 §4.2 WS-topic lifecycle addressed (session end / widget delete) — CORRECTED IN ROUND 2

**Round 1's original resolution below was wrong and has been retracted.**
Round 2 review verified by direct source inspection that the claim
"`WebchatSessionEnd` publishes its own event... already true today" is
**false**: `bin-webchat-manager/pkg/sessionhandler/db.go`'s `End()` only
calls `db.SessionUpdate`/`db.SessionGet`, and `sessionHandler`
(`pkg/sessionhandler/main.go`) has **no `notifyHandler` field at all**
(unlike `messageHandler`, which does) — there is no session-end event,
no event-type constant, and no publish call anywhere in the codebase
(confirmed via repo-wide grep for `session_ended`/`EventTypeSessionEnd*`:
zero matches). This was a fabricated "confirmed" claim in the original
Round 1 fix — caught by Round 2's adversarial re-verification, not
self-caught, which is itself worth noting for how this doc's claims get
checked going forward.

**Corrected resolution: this IS a real backend change, added to this
PR's scope, not "no change needed."**

1. Inject `notifyHandler notifyhandler.NotifyHandler` into
   `sessionHandler` (`pkg/sessionhandler/main.go`), mirroring
   `messageHandler`'s existing field — `NewSessionHandler`'s signature
   gains a `notifyHandler` parameter.
2. `cmd/webchat-manager/main.go` line 123's
   `sessionhandler.NewSessionHandler(reqHandler, db)` call is updated to
   `sessionhandler.NewSessionHandler(reqHandler, notifyHandler, db)` —
   the `notifyHandler` value already exists at that point in `main.go`
   (constructed at line 121, three lines before `messageHandler`'s own
   construction at line 124, which already consumes it) — no new
   `notifyHandler` construction needed, just passing the existing one
   to a second constructor.
3. `session` package (`models/session/`) already has a complete
   `WebhookMessage`/`ConvertWebhookMessage`/`CreateWebhookEvent` set
   (`webhook.go`, pre-existing, unused by any publish call today) — add
   a new `EventTypeSessionEnded = "webchat_session_ended"` constant
   (mirroring `message.EventTypeMessageCreated`'s pattern in
   `models/message/event.go`) to a new `models/session/event.go`.
4. `sessionHandler.End()` (`pkg/sessionhandler/db.go`), after the
   existing `SessionUpdate`/`SessionGet` calls succeed, adds:
   ```go
   if h.notifyHandler != nil {
       h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, session.EventTypeSessionEnded, res)
   }
   ```
   mirroring `messagehandler/create.go`'s existing call shape exactly,
   including its `if h.notifyHandler != nil` nil-guard (kept for
   consistency with the existing codebase style, even though the
   constructor will always supply a real handler in production).
5. §4.2's `createTopics()` case needs a **second** `case` (or an
   extended condition on the existing `case "webchat":`) for
   `webchat_session_ended`, since `strings.Split("webchat_session_ended", "_")[0]`
   is also `"webchat"` — same topic-string output
   (`customer_id:<cid>:webchat_session:<session_id>`) as the message-
   created case, which is exactly what's wanted here (same topic,
   different event type on it) — **but `commonWebhookData` needs `ID`
   itself to resolve to the session id for this event** (unlike the
   message-created case, where `d.SessionID` is a separate field from
   `d.ID`/the message's own id) — `Session.WebhookMessage.Identity.ID`
   IS the session id for this event, so the topic-building code must
   branch on which event type it's handling to pick the right field
   (`d.SessionID` for `webchat_message_created`, `d.ID` for
   `webchat_session_ended`). This is a real implementation subtlety,
   called out explicitly here so it isn't missed.

The frontend's `client.js` (companion doc, updated) listens for this
NOW-REAL event and closes its own WS connection client-side upon
receipt. No backend enforcement is added to forcibly sever the WS
connection at the socket layer (the underlying `sockhandler`/gorilla-
websocket layer used platform-wide has no per-topic forced-disconnect
primitive today — adding one is a larger cross-cutting change, still
out of scope). This remains a client-side cooperative close, not a
socket-level server-side guarantee — documented as an explicit accepted
limitation: **a visitor who ignores the session-ended event and keeps
their WS connection open would continue receiving any messages
published to that session's topic after End** — but in practice this
requires a subsequent message create on an ended session, which §9.2/
existing `Create`'s direct branch already rejects — so no new
*messages* CAN be published to an ended session's topic post-End; the
remaining exposure is the (now real, single) session-ended event itself
plus theoretical future events, not an open-ended leak.

**Test plan addition**: `sessionHandler.End()` unit test asserting
`PublishWebhookEvent` is called with the correct event type and
customer id (mock `notifyHandler`, mirroring `messagehandler/create_test.go`'s
existing pattern). `createTopics()` unit test for the new
`webchat_session_ended` case.

### 9.4 §4.2 `SessionID` field addition — confirmed safe

Round 1 flagged (low-risk) the shared `commonWebhookData` struct gaining
a `SessionID` field. Confirmed: `AIcallID`/`ChatID` are the only
existing fields, both `omitempty` UUID, following the identical pattern;
no other publisher's payload has ever populated a colliding `session_id`
JSON key (grep confirms `session_id` is unique to `webchat` message/
session payloads platform-wide). No change needed beyond what §4.2
already specified.

### 9.5 WS handshake auth path — confirmed, not assumed

Round 1 correctly flagged that the original doc never verified a
direct-scope JWT is even accepted at the `/ws` upgrade step (only that
post-auth topic validation would pass one through). **Now verified**:
`GetWs` (`bin-api-manager/server/ws.go`) sits under the same
`v1.Use(middleware.Authenticate())` group (`cmd/api-manager/main.go`
line 256) as every other `/v1.0/*` route including
`POST /webchat_sessions` and `POST /webchat_messages`, both of which
already accept direct-scope callers today. `middleware.Authenticate()`
(`lib/middleware/authenticate.go` line 96-114) has an explicit
`case string(auth.TypeDirect):` branch building `auth.NewDirectIdentity`
from the JWT's `direct` claim — this is the exact same code path already
exercised by the existing `POST /auth/boot` → direct-scope-JWT flow. No
gap exists; §4.2 proceeds as designed.

### 9.6 Sender-echo question resolved (not left as shared open question)

Round 1 pointed out this doc owns the authoritative answer and should
not defer it to the frontend. **Resolved**: `messagehandler/create.go`
calls `h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID,
message.EventTypeMessageCreated, res)` unconditionally, regardless of
`direction` or which auth path (agent vs. direct) created the message,
and §4.2's new topic is scoped by session id, not by sender/recipient —
there is exactly one topic per session, subscribed to by whichever
party (visitor and/or agent, per §9.7) is listening. **Therefore: yes,
a visitor's own inbound message IS echoed back to them over their own
subscribed topic.** Companion frontend doc's `client.js` design must
dedupe on the message `id` returned by the synchronous
`POST /webchat_messages` response against the async WS delivery of the
same id — this is now a confirmed requirement, not an open question.

### 9.7 G1 wording corrected + NG4 narrowed (real-time claim scoped accurately, gap closed)

Round 1 noted G1's original "receive the agent's reply in real time...
no polling loop" described only the visitor's half of the round-trip,
since the original NG4 kept the agent side on manual refresh — and the
companion frontend doc's independent Round 1 review flagged the SAME
gap from the other side, additionally noting square-admin's own
CLAUDE.md bans manual-refresh-button UX on detail pages, meaning the
original scope cut wasn't just incomplete, it was unworkable within the
product's own UI conventions.

**Resolution: NG4 is narrowed, not just re-worded.** Minimal agent-side
WS wiring for the Sessions/message-timeline view (subscribing to the
same `customer_id:<cid>:webchat_session:<session_id>` topic this design
already creates) is pulled into scope, implemented in the frontend PR.
This requires **zero additional backend change** — the topic is
symmetric by construction (§4.2 scopes it by session, not by
sender/recipient role), so an agent-side subscriber works identically
to the visitor-side subscriber already designed. G1 and NG4 above are
updated to reflect this; G5's E2E test plan (§7) now confirms live
delivery on both sides, not just the visitor's.

### 9.8 Server-layer 400 requirement — corrected (removed, not needed)

Round 1 flagged this claim was unverified. **Confirmed**:
`bin-api-manager/server/webchat_messages.go`'s `GetWebchatMessages`
(line 15-64) currently passes `sessionID` through as `uuid.Nil` when
the query param is absent, with no server-layer validation — the
"required for direct callers" check does NOT exist today and is not
added by this doc's servicehandler-layer fix in §9.2 either (§9.2's
`sessionID == uuid.Nil` check happens inside `WebchatMessageList`
itself, not in the `server/` HTTP layer). This is sufficient: the
`ErrPermissionDenied` returned by the servicehandler is correctly
mapped to an HTTP error by `abortWithServiceError` (existing, unchanged
mechanism) — so the original doc's claim of a required "server-layer
400" was over-specified; the actual, simpler mechanism is a
servicehandler-layer permission denial, already covered by §9.2's code.
No `server/webchat_messages.go` change is needed.
