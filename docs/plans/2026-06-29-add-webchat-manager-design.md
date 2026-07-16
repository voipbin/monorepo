# bin-webchat-manager Design

## Revision Notice (v6 — conversation-manager integration via message-manager pattern)

**pchero (CEO/CTO) directed a further integration requirement**: webchat
conversations must also surface in `bin-conversation-manager`'s unified
Conversation/Message view (the same place SMS/LINE/WhatsApp threads live),
so agents get one timeline across every channel, and any existing
Contact/Case linkage (`Conversation.Metadata.ContactCaseID`) applies to
webchat automatically. Three follow-up questions were resolved in discussion
before writing this revision:

1. **Should `bin-webchat-manager` disappear and conversation-manager own
   Widget/Session/Message directly (full delegation)?** No — pchero
   directed that webchat-manager must keep owning messages itself, the same
   way `bin-message-manager` owns SMS `Message`/`Target` records
   independently of conversation-manager.
2. **Should conversation-manager get its own webchat-specific columns
   (e.g. a `DirectID` field on `Account`)?** No — verified against the real
   `Account`/`Conversation`/`Message` structs and the SMS ingestion path
   (`conversationhandler.eventSMS`, `execute_mode.go`'s
   `runExecuteModeFlowMessage`) that **`Account` is not universal**: SMS has
   no Account at all (it resolves via `bin-number-manager`'s `Number`
   instead), and Conversation identity for SMS comes from `(self, peer)`
   address pairs, not `(account_id, dialog_id)`. Webchat's cardinality
   (`Widget` = fixed VoIPbin-side identity reused across many visitors;
   each visitor's Session = a fresh, per-conversation identity) matches
   SMS's `(self=our number, peer=caller's number)` shape far more closely
   than LINE/WhatsApp's `(account_id, dialog_id)` shape. **Conclusion:
   webchat does NOT get a conversation-manager `Account` row at all,
   mirroring SMS exactly** — not the earlier-considered "Account per
   Widget" design.
3. **How is the double-Flow-trigger problem resolved** (both
   `bin-webchat-manager`, via `Widget.FlowID`, and conversation-manager's
   `execute_mode.go`, via a hypothetical per-channel flow source, would
   otherwise independently create an activeflow for the same inbound
   message)? Three options were discussed (webchat-manager triggers /
   conversation-manager triggers / hand off based on `ExecuteModeAgent` vs
   `ExecuteModeFlow`). **pchero selected: `bin-webchat-manager` keeps the
   ONLY Flow-triggering responsibility for the real-time visitor-facing
   path (AI response, welcome message, etc.); `bin-conversation-manager`'s
   webchat ingestion path NEVER triggers a Flow** — mirroring how
   `bin-message-manager` itself never triggers a Flow (only
   conversation-manager's SMS `execute_mode.go` does, and there is no
   third layer above conversation-manager to double-trigger against). For
   webchat, webchat-manager occupies the position message-manager occupies
   for SMS, but retains the Flow-triggering role that, for SMS, uniquely
   belongs to conversation-manager — because for webchat, real-time
   visitor-facing response requires no perceptible additional latency hop,
   which was the deciding product requirement.

**Everything from v5 is unchanged**: `bin-webchat-manager` remains a plain
Class A RabbitMQ RPC manager owning `Widget`/`Session`/`Message` in its own
tables; the direct-token identity model (`Widget`=parent, `Session`=child,
mirroring `ai`/`aicall`) is unchanged; the WebSocket delivery path via
api-manager's existing `/ws` and `pkg/subscribehandler`/`pkg/websockhandler`
is unchanged; Round 1/Round 2 review findings (§15) are unchanged and still
open. **This revision is purely additive**: a new one-way integration from
webchat-manager into conversation-manager, described in §16.

## 1. Problem Statement

VoIPbin has no service that manages anonymous web-visitor chat widget
sessions. The two services that look adjacent do not fit:

- **bin-conversation-manager** models `Account` as durable platform
  credentials and `Conversation` as a thread keyed by `(account_id, dialog_id)`
  with stable platform addresses. A web visitor has none of that before a
  session starts.
- **bin-talk-manager** models `Chat`/`Participant` for already-known entities.
  It has no concept of an anonymous, unauthenticated, expiring identity.

What's missing is a Widget/Session/Message domain model for widget
configuration and conversation state, wired onto the platform's EXISTING
parent/child direct-token pattern (proven in production for AI voice
widgets via `ai`/`aicall`) rather than a webchat-specific mechanism.

## 2. Scope

### In scope (Phase 1)

- New service `bin-webchat-manager`: a standard Class A RabbitMQ RPC manager
  owning three entities — `Widget` (widget config, the parent), `Session`
  (a visitor's conversation instance, the child — doubles as visitor
  identity, see Revision Notice), `Message`.
- `bin-direct-manager`: add `resource_type = "webchat_widget"`.
- `bin-api-manager`: add ONE entry to the existing `directResourceMapping`
  table in `pkg/servicehandler/boot.go`: `"webchat_widget": {"webchat_session"}`.
  No new JWT claims, no changes to `validateTopics`, no changes to
  `pkg/websockhandler` at all.
- `bin-api-manager`: new REST routes `/v1/webchat/widgets` (customer-JWT,
  admin CRUD) and `/v1/webchat/sessions/*` (mix of customer-JWT and
  direct-token-scoped operations, with the ownership check described in
  the Revision Notice).
- Flow integration: incoming webchat message can trigger a
  `bin-flow-manager` activeflow (`reference_type=webchat`).
- Outbound message delivery: webchat-manager publishes an event; the
  EXISTING `pkg/subscribehandler` -> `pkg/websockhandler` pipeline fans it
  out to whichever browser is subscribed to `customer_id:<id>:webchat_session:<Session.ID>`.

### Out of scope (explicitly deferred)

| Item | Phase | Reason |
|---|---|---|
| Human-agent queue handoff (bin-queue-manager integration) | Phase 2 | Flow can already route to `queue_join` once this session layer exists |
| Pre-chat form / lead capture fields | Phase 2 | Product-config surface |
| File/image attachments | Phase 2 | Reuses conversation-manager's `medias` pattern |
| Typing indicators / read receipts / presence | Phase 2 | Not required for MVP |
| Mobile SDK / push notification parity | Not scoped | Separate product surface |
| Direct-hash regeneration/rotation policy for a live widget | Phase 2 | Regenerate already exists platform-wide |
| Multi-tab session merge (same visitor, two tabs, two Session.IDs) | Phase 2 | Same tradeoff already accepted for `aicall`; each tab that doesn't share `localStorage` gets its own conversation. Not a new problem introduced by webchat. |

## 3. Domain Model

### Widget

Widget configuration owned by a customer. One customer may have multiple
`Widget`s. Each `Widget` has exactly one direct hash (1:1), the platform's
existing PARENT resource in the direct-token model.

```go
type Status string

const (
    StatusActive   Status = "active"
    StatusInactive Status = "inactive"
)

type Widget struct {
    commonidentity.Identity // ID, CustomerID

    Name   string `json:"name"`
    Status Status `json:"status"`

    // DirectID references the bin-direct-manager record
    // (resource_type=webchat_widget, resource_id=Widget.ID). Populated by
    // webchat-manager calling DirectV1Create immediately after Widget
    // creation. See §10 for the create/delete failure-mode handling.
    DirectID uuid.UUID `json:"direct_id,omitempty"`

    WelcomeMessage string    `json:"welcome_message,omitempty"`
    FlowID         uuid.UUID `json:"flow_id,omitempty"` // activeflow started on first inbound message; empty = no auto flow

    SessionIdleTimeout int `json:"session_idle_timeout"` // seconds; default 1800 (30m)

    TMCreate string `json:"tm_create"`
    TMUpdate string `json:"tm_update"`
    TMDelete string `json:"tm_delete"`
}
```

No `AllowedOrigins`/origin allow-list field. Same as v4: origin restriction
would be a `bin-direct-manager`-wide feature request applying uniformly to
all six resource types, not a webchat-specific bolt-on (Open Question §15).

Design rule check: no `StatusNone=""`.

### Session

The platform's existing CHILD resource in the direct-token model
(`ai`/`aicall`'s exact shape). **`Session.ID` doubles as the visitor's
identity** — it is what the widget round-trips as a continuity token, and
what the WebSocket topic is scoped by. No separate `VisitorID` field exists
(v4 had one with an unresolved source-of-truth question; v5 removes the
need for it entirely).

```go
type SessionStatus string

const (
    SessionStatusActive SessionStatus = "active" // created on first inbound message, or freshly created
    SessionStatusEnded  SessionStatus = "ended"   // idle timeout elapsed, or explicitly closed
)

type Session struct {
    commonidentity.Identity // ID, CustomerID — Identity.ID IS the visitor's continuity token

    WidgetID uuid.UUID     `json:"widget_id"`
    Status   SessionStatus `json:"status"`

    ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`

    TMLastActivity string `json:"tm_last_activity"`
    TMCreate       string `json:"tm_create"`
    TMUpdate       string `json:"tm_update"`
    TMEnd          string `json:"tm_end,omitempty"` // lifecycle marker, distinct from tm_delete
    TMDelete       string `json:"-"`                // standard soft-delete sentinel
}
```

Status lifecycle: two states, unchanged from v4.

```
(created on first inbound message, no session_id supplied by client) --> active
active --(idle_timeout elapsed, or explicit close)--------------------> ended
```

`ended` is terminal; a subsequent message with no `session_id` (or an
`ended` one) creates a NEW `Session` — i.e., a new conversation, with its
own new unguessable ID, which is the correct behavior (matches how a new
`aicall` is created for a new call against the same `ai` direct hash).

**No reconnect race, no remote-close protocol, no per-pod queue** —
unchanged from v4, still eliminated by construction.

### Message

Unchanged from v4.

```go
type MessageDirection string

const (
    MessageDirectionInbound  MessageDirection = "inbound"  // visitor -> VoIPbin
    MessageDirectionOutbound MessageDirection = "outbound" // VoIPbin -> visitor
)

type MessageStatus string

const (
    MessageStatusSent      MessageStatus = "sent"
    MessageStatusDelivered MessageStatus = "delivered" // best-effort — see §10
    MessageStatusFailed    MessageStatus = "failed"    // event publish itself failed (rare; RabbitMQ down)
)

type Message struct {
    commonidentity.Identity // ID, CustomerID

    SessionID uuid.UUID        `json:"session_id"`
    Direction MessageDirection `json:"direction"`
    Status    MessageStatus    `json:"status"`

    // SenderID: agent user ID for an agent-typed outbound reply; empty for
    // flow/AI-originated or inbound messages. (Namespace note, Round-1
    // reviewer Low finding: this is always an Agent ID when set, never a
    // visitor identity — visitors are identified by SessionID, not by a
    // SenderID on their own messages.)
    SenderID uuid.UUID `json:"sender_id,omitempty"`

    ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`

    Text string `json:"text"`

    TMCreate string `json:"tm_create"`
    TMDelete string `json:"tm_delete"`
}
```

## 4. Database Schema

```sql
CREATE TABLE webchat_widgets (
    id                    BINARY(16)   NOT NULL,
    customer_id           BINARY(16)   NOT NULL,

    name                  VARCHAR(255) NOT NULL,
    status                VARCHAR(16)  NOT NULL,
    direct_id             BINARY(16),

    welcome_message       TEXT,
    flow_id               BINARY(16),
    session_idle_timeout  INT          NOT NULL DEFAULT 1800,

    tm_create             DATETIME(6)  NOT NULL,
    tm_update             DATETIME(6)  NOT NULL,
    tm_delete             DATETIME(6)  NOT NULL DEFAULT '9999-01-01 00:00:00.000000',

    PRIMARY KEY (id),
    INDEX idx_webchat_widgets_customer_id_tm_create (customer_id, tm_create),
    INDEX idx_webchat_widgets_customer_id_tm_delete (customer_id, tm_delete)
);

CREATE TABLE webchat_sessions (
    id                BINARY(16)   NOT NULL,
    customer_id       BINARY(16)   NOT NULL,
    widget_id         BINARY(16)   NOT NULL,

    status            VARCHAR(16)  NOT NULL,
    activeflow_id     BINARY(16),

    tm_last_activity  DATETIME(6)  NOT NULL,
    tm_create         DATETIME(6)  NOT NULL,
    tm_update         DATETIME(6)  NOT NULL,
    tm_end            DATETIME(6)  NOT NULL DEFAULT '9999-01-01 00:00:00.000000',
    tm_delete         DATETIME(6)  NOT NULL DEFAULT '9999-01-01 00:00:00.000000',

    PRIMARY KEY (id),
    INDEX idx_webchat_sessions_customer_id_tm_create (customer_id, tm_create),
    INDEX idx_webchat_sessions_customer_id_tm_delete (customer_id, tm_delete),
    INDEX idx_webchat_sessions_widget_id_status (widget_id, status),
    INDEX idx_webchat_sessions_status_tm_last_activity (status, tm_last_activity)
);

CREATE TABLE webchat_messages (
    id             BINARY(16)   NOT NULL,
    customer_id    BINARY(16)   NOT NULL,
    session_id     BINARY(16)   NOT NULL,

    direction      VARCHAR(16)  NOT NULL,
    status         VARCHAR(16)  NOT NULL,
    text           VARCHAR(4000) NOT NULL,
    sender_id      BINARY(16),
    activeflow_id  BINARY(16),

    tm_create      DATETIME(6)  NOT NULL,
    tm_delete      DATETIME(6)  NOT NULL DEFAULT '9999-01-01 00:00:00.000000',

    PRIMARY KEY (id),
    INDEX idx_webchat_messages_session_id_tm_create (session_id, tm_create),
    INDEX idx_webchat_messages_customer_id_tm_create (customer_id, tm_create)
);
```

Changed vs v4: `webchat_sessions.visitor_id` column removed entirely (no
longer needed — `Session.ID` IS the visitor's continuity token); the
`(widget_id, visitor_id)` index is replaced with `(widget_id, status)`,
useful for admin/dashboard queries ("how many active sessions does this
widget have right now").

**GetOrCreate concurrency (Round-1 reviewer High finding, now resolved by
construction, not by adding a DB constraint):** a duplicate-session race
required a *shared* key (`visitor_id`) two concurrent requests could
collide on. Since v5 has no such shared key — every "create" call that
doesn't supply an existing `session_id` unconditionally creates a brand
new `Session` row with a fresh UUID — there is no `GetOrCreate` race to
guard against. The operation is now cleanly split into two: `Create`
(first message, no input ID, always succeeds with a new row) and `Get` +
ownership check (subsequent messages, client-supplied `session_id`). A
double-fired first message (e.g. a duplicate page-load beacon) legitimately
creates two `Session`s — a known, accepted, low-severity product
consequence (two started conversations instead of one), not a data
integrity bug requiring transactional protection.

## 5. Handler Interface

```go
type WidgetHandler interface {
    Create(ctx context.Context, customerID uuid.UUID, name string, welcomeMessage string, flowID uuid.UUID, idleTimeout int) (*widget.Widget, error)
    Get(ctx context.Context, id uuid.UUID) (*widget.Widget, error)
    List(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]*widget.Widget, error)
    Update(ctx context.Context, id uuid.UUID, name string, welcomeMessage string, flowID uuid.UUID, idleTimeout int) (*widget.Widget, error)
    Delete(ctx context.Context, id uuid.UUID) (*widget.Widget, error)
}

type SessionHandler interface {
    // Create makes a brand new Session for a Widget (first message, no
    // client-supplied session_id). Always succeeds with a fresh UUID — no
    // uniqueness constraint to violate, no race to guard (see §4).
    Create(ctx context.Context, widgetID uuid.UUID) (*session.Session, error)

    // Get + ownership check for a client-supplied session_id on a
    // subsequent message. Callers (api-manager's servicehandler) MUST
    // additionally verify session.WidgetID == token's DirectScope.ResourceID
    // and session.CustomerID == token's CustomerID before trusting this —
    // mirrors AIcallCreate's existing `ResourceID != assistanceID` check.
    Get(ctx context.Context, id uuid.UUID) (*session.Session, error)

    MessageSend(ctx context.Context, sessionID uuid.UUID, text string) (*message.Message, error)
    MessageDeliver(ctx context.Context, sessionID uuid.UUID, text string, senderID uuid.UUID) (*message.Message, error)
    Close(ctx context.Context, sessionID uuid.UUID) error
}
```

### Core flow: visitor boots, chats, gets a Flow-routed reply

```
1. Widget snippet embeds Widget's direct hash (obtained at Widget creation).
2. Visitor's browser: POST https://api.voipbin.net/auth/boot { direct_hash }
   -> EXISTING code, ONE new directResourceMapping entry
      ("webchat_widget" -> ["webchat_session"]). Returns JWT:
      { token, type:"direct", resource_type:"webchat_widget",
        resource_id:<Widget.ID>, customer_id, expire }
3. Visitor's FIRST message (no session_id known yet):
   POST /v1/webchat/sessions/messages?token=<jwt> { text: "hello" }
   -> api-manager verifies a.HasAllowedResourceType("webchat_session")
   -> calls SessionHandler.Create(widgetID = a.DirectScope.ResourceID)
   -> calls SessionHandler.MessageSend(session.ID, text)
   -> response includes { session_id, message_id, status }
   -> IF Widget.FlowID set: FlowV1ActiveflowCreate(reference_type=webchat, reference_id=session.ID)
4. Browser stores session_id (e.g. localStorage), then:
   GET https://api.voipbin.net/ws?token=<jwt>
   -> EXISTING code, UNCHANGED. Subscribes to topic:
      customer_id:<id>:webchat_session:<session_id>
   -> validateTopics checks HasAllowedResourceType("webchat_session") only;
      the session_id segment is never re-validated against anything — same
      as the aicall precedent.
5. Visitor's SUBSEQUENT messages include the stored session_id:
   POST /v1/webchat/sessions/messages?token=<jwt> { session_id, text }
   -> api-manager calls SessionHandler.Get(session_id), then verifies
      session.WidgetID == a.DirectScope.ResourceID AND
      session.CustomerID == a.CustomerID (ownership check, mirrors
      AIcallCreate) before calling MessageSend.
6. Flow executes actions (ai_talk / queue_join / webchat_message_send).
   -> webchat_message_send calls MessageDeliver().
   -> webchat-manager publishes webchat_message_created (outbound).
7. api-manager's EXISTING pkg/subscribehandler/websockhandler fans the
   event out to any browser subscribed to
   customer_id:<id>:webchat_session:<session_id>. Only the browser that
   knows this specific session_id can ever be subscribed to it (the ID is
   an unguessable UUID never exposed to other visitors) — this is what
   prevents cross-visitor leakage, verified against real code, not assumed.
```

## 6. LLM Logic

Not applicable — Flow's `ai_talk` action against bin-ai-manager. Unchanged.

## 7. REST API

| Method | Path | Auth | Purpose |
|---|---|---|---|
| POST | `/v1/webchat/widgets` | customer JWT | Create a Widget (also issues a direct hash) |
| GET | `/v1/webchat/widgets` | customer JWT | List Widgets |
| GET | `/v1/webchat/widgets/{id}` | customer JWT | Get a Widget (includes the widget snippet / hash) |
| PUT | `/v1/webchat/widgets/{id}` | customer JWT | Update a Widget |
| DELETE | `/v1/webchat/widgets/{id}` | customer JWT | Delete a Widget (cascades: revoke direct hash) |
| POST | `/v1/webchat/sessions/messages` | direct token (`resource_type=webchat_widget`, allows `webchat_session`) | Visitor sends a message; `session_id` optional (first message creates a new Session; supplying an existing one continues it, subject to the ownership check in §5) |
| GET | `/v1/webchat/sessions/{id}/messages` | customer JWT | List a session's messages (agent/dashboard view) |
| POST | `/v1/webchat/sessions/{id}/messages` | customer JWT | Agent/API-authored outbound message |
| POST | `/v1/webchat/sessions/{id}/close` | customer JWT | Explicitly end a session |

No public/unauthenticated routes. Unchanged from v4.

Example: visitor sends first message

```json
POST /v1/webchat/sessions/messages?token=<direct-jwt>
{
  "text": "Hi, I have a question about pricing"
}

200 OK
{
  "session_id": "7a1b...",
  "message_id": "5e9c...",
  "status": "sent"
}
```

Example: visitor sends a follow-up message (same conversation)

```json
POST /v1/webchat/sessions/messages?token=<direct-jwt>
{
  "session_id": "7a1b...",
  "text": "Do you offer a free trial?"
}
```

## 8. Webhook Events

| Event | Trigger | Payload |
|---|---|---|
| `webchat_session_created` | `SessionHandler.Create()` | `Session` |
| `webchat_session_ended` | `Close()` or idle sweep | `Session` |
| `webchat_message_created` | Message persisted (either direction) | `Message` |

Unchanged from v4.

## 9. Flow Variable Integration

| Variable | Value |
|---|---|
| `voipbin.webchat.session.id` | Session UUID |
| `voipbin.webchat.session.widget_id` | Widget UUID |
| `voipbin.webchat.message.text` | Most recent inbound message text |

Unchanged from v4.

## 10. RabbitMQ Integration & Delivery Model

**Standard RPC surface only**: `bin-manager.webchat-manager.request`.
Unchanged from v4 — no per-pod queues, no HostID, no in-memory session map.

**Delivery is fire-and-forget from webchat-manager's perspective.**
Unchanged from v4 — `MessageDeliver` persists + publishes an event; whether
a browser is connected is api-manager's `pkg/websockhandler` concern,
invisible to webchat-manager. `MessageStatus.Delivered` semantics are
correspondingly weaker than "confirmed socket write" (Open Question §15,
unchanged from v4).

**Direct hash lifecycle ties to Widget lifecycle.** Unchanged from v4:
`WidgetHandler.Create` calls `DirectV1Create` synchronously;
`WidgetHandler.Delete` calls `DirectV1Delete`. The partial-failure gap
(§15) is unchanged.

**Idle sweep**: unchanged from v4 — ticker scans
`status = 'active' AND tm_last_activity < now() - idle_timeout`,
transitions to `ended`, publishes `webchat_session_ended`.

## 11. Observability

Unchanged from v4:

- Prometheus counter: `webchat_manager_session_total{status}`
- Prometheus counter: `webchat_manager_message_total{direction,status}`
- Prometheus histogram: `webchat_manager_request_process_time`
- Standard trace ID propagation, `context.WithTimeout`+`recover()` goroutine convention
- Check `initPrometheus()` for duplicate-registration panic

## 12. Security & Compliance

- **No unauthenticated surface introduced** — every route gated by an
  existing credential type. Unchanged from v4.
- **Cross-visitor isolation is capability-based (unguessable `Session.ID`),
  not access-control-based (no per-session ACL check at the WS layer).**
  This is an HONEST characterization of the security model, not a euphemism
  for a gap: it is the identical model the platform already runs in
  production for `aicall` (verified against `validateTopics` and
  `AIcallGet`/`AIcallCreate`), so this design carries no NEW risk profile
  versus what api-manager already accepts elsewhere. The practical
  implication: a `Session.ID` must never be logged to a place a different
  customer's operators could read, must never be predictable/sequential
  (standard UUIDv4 generation, already the platform convention), and any
  future feature that lets an agent "share a session link" must treat that
  link as bearer-token-equivalent.
- **Rate limiting / abuse control for direct-token issuance is a
  pre-existing platform concern** (already reachable via AI-widget direct
  hashes today) — Open Question (§15) to confirm current state, not new to
  webchat. Round-1 reviewer correctly noted a text-chat widget likely
  invites higher-frequency automated `/auth/boot` calls than a voice widget
  — this elevates the urgency of confirming the answer, without changing
  the recommendation itself.
- **Input validation on visitor-submitted text**: `text` capped at 4000
  characters. Unchanged from v4.
- **No PII required to boot or to start a session** — `Session.ID` is a
  randomly generated UUID, not derived from any visitor-supplied identity
  claim (there is none — see Revision Notice).
- **Message content MAY contain visitor-volunteered PII.** Unchanged from
  v4 — standard `tm_delete` soft-delete convention; retention window is an
  Open Question (§15).
- **Message content is customer-visible business data**, scoped by
  `customer_id`. Unchanged.
- **No external LLM involvement directly** — via Flow's `ai_talk` action
  only. Unchanged.

## 13. Affected Services

| Service | Change | Phase |
|---|---|---|
| `bin-webchat-manager` (new) | New Class A RPC service: Widget/Session/Message | 1 |
| `bin-direct-manager` | Add `resource_type = "webchat_widget"` to the existing enum | 1 |
| `bin-flow-manager` | Add `reference_type=webchat`; add `webchat_message_send` action | 1 |
| `bin-api-manager` | New REST routes; **exactly one line added to the existing `directResourceMapping` table** (`"webchat_widget": {"webchat_session"}`) in `pkg/servicehandler/boot.go` — no JWT claim changes, no `pkg/websockhandler` changes | 1 |
| `bin-openapi-manager` | New OpenAPI paths | 1 |
| `bin-webhook-manager` | No change — new event keys only | 1 |
| `bin-conversation-manager` | **New in v6**: subscribe to `webchat_message_created`; add `conversation.TypeWebchat`; `eventWebchat()` handler mirroring `eventSMS()`; `execute_mode.go` gets a no-op webchat branch (never triggers Flow) — see §16 | 1 |
| `bin-queue-manager` | Human-agent handoff integration | 2 |

## 14. Implementation Order

1. `bin-direct-manager`: add `resource_type = "webchat_widget"` to the enum.
2. `bin-api-manager/pkg/servicehandler/boot.go`: add the single
   `directResourceMapping` entry. Write a unit test asserting
   `AuthBoot` against a `webchat_widget` hash returns
   `AllowedResourceTypes=["webchat_session"]`, mirroring the existing
   `ai`->`["aicall"]` test coverage.
3. `bin-webchat-manager` scaffold (cmd/, models/widget, models/session,
   models/message, dbhandler, standard Class A listenhandler).
4. `widgethandler.Create/Get/List/Update/Delete`, wired to
   `DirectV1Create`/`DirectV1Delete` (§10's two-phase-commit question
   resolved here, not deferred).
5. `sessionhandler.Create/Get/MessageSend/MessageDeliver/Close` + idle sweep.
6. `bin-api-manager`: new `pkg/servicehandler/webchat_widget.go` and
   `webchat_session.go`. The session-message handler implements the
   ownership check from §5 (`session.WidgetID == a.DirectScope.ResourceID`,
   `session.CustomerID == a.CustomerID`) for any client-supplied
   `session_id`, mirroring `AIcallCreate`'s existing pattern verbatim.
7. `bin-flow-manager`: `reference_type=webchat` + `webchat_message_send`
   action.
8. First-inbound-message flow trigger wiring.
9. Webhook events.
10. OpenAPI spec + REST routes.
11. End-to-end test: hash issuance -> `/auth/boot` -> first message (new
    session) -> `/ws` subscribe to that exact `session_id` topic -> second
    message with the stored `session_id` -> flow-triggered reply -> event
    fan-out to the subscribed browser -> negative test: a SECOND browser
    session (different `Session.ID`, same `Widget`) does NOT receive the
    first session's messages, proving the isolation claim in §12 rather
    than assuming it.
12. `bin-conversation-manager`: add `conversation.TypeWebchat` to the
    `Type` enum (models/conversation). Add `publisherWebchatManager`
    constant + a `webchat_message_created` case to
    `pkg/subscribehandler/main.go`'s `processEvent` switch, mirroring the
    existing `publisherMessageManager` case exactly.
13. `bin-conversation-manager`: new `pkg/subscribehandler/webchatmanager.go`
    (`processEventWebchatMessageMessageCreated`) + new
    `conversationhandler.eventWebchat()`, mirroring `eventSMS()`'s shape:
    resolve/create `Conversation` by `(Self, Peer)` address pair (NOT
    `(account_id, dialog_id)` — see §16), create the `Message` record with
    `ReferenceType=webchat`, `ReferenceID=<webchat-manager Message.ID>`,
    publish `conversation_created`/`message_created`. **Explicit note (Round
    3 reviewer finding): `conversationHandler.Event(ctx, referenceType,
    data)` in `event.go` is a SECOND, distinct switch from
    `subscribehandler.processEvent`'s outer switch — today it only has
    `case conversation.TypeMessage: return h.eventSMS(...)`, and its
    `default` branch returns a HARD ERROR (`"reference type handler not
    found"`), not a silent no-op (unlike `execute_mode.go`'s dispatcher, see
    step 14). This step MUST add `case conversation.TypeWebchat: return
    h.eventWebchat(ctx, data)` to `Event()`'s switch alongside the existing
    `TypeMessage` case. Skipping this — e.g. by wiring
    `processEventWebchatMessageMessageCreated` straight to `eventWebchat()`
    and bypassing `Event()` entirely — silently diverges from the mirrored
    SMS pattern; wiring through `Event()` but forgetting this case produces
    a hard error logged on every single webchat message.**
14. `bin-conversation-manager`: add `conversation.TypeWebchat` as a
    no-op case in `execute_mode.go`'s `runExecuteModeFlow` dispatch —
    **Round 3 reviewer verified this is actually unnecessary: the existing
    switch's `default` branch already logs and returns `nil` for any
    unrecognized `cv.Type`, and this no-op behavior is already covered by
    an existing test (`Test_runExecuteModeFlow_unsupportedTypeIsNoop`).
    `TypeWebchat` inherits correct no-op behavior automatically with ZERO
    new code at this layer** — this step is now a verification-only item
    (confirm the existing test still passes with a webchat `Conversation`
    fixture), not an implementation item. The reviewer additionally
    confirmed `MessageEventSent` (the outbound path) never calls
    `getExecuteMode`/`runExecuteModeFlow` at all, so outbound webchat
    messages carry zero flow-trigger risk by construction, independent of
    this step.
15. End-to-end test (conversation-manager side): a webchat message flowing
    through webchat-manager produces exactly ONE activeflow execution
    (webchat-manager's), and a corresponding Conversation/Message row in
    conversation-manager with zero activeflow executions triggered from
    that side — this is the test that directly verifies §16's core claim.

## 15. Open Questions

| Question | Recommendation | Decision owner |
|---|---|---|
| Rate limiting / abuse control on `/auth/boot` and the session-message endpoint | Confirm current state before assuming either "already covered" or "needs new work"; the reviewer correctly notes a text widget likely sees higher automated-abuse volume than the existing voice-widget use case | CEO/CTO, pre-implementation |
| `MessageStatus.Delivered` semantics are weaker than a confirmed socket write — acceptable, or does it need a delivery-ack loop back to webchat-manager? | Ship without an ack loop in Phase 1 (matches every other channel's actual guarantee level) | CEO/CTO |
| Widget/Direct two-phase-commit gap: `Widget.Create` succeeding but the paired `Direct.Create` failing (or vice versa on delete) | `Widget.DirectID` nullable; a Widget with `DirectID = NULL` is "provisioning incomplete," excluded from the widget-snippet response until resolved, rather than distributed-transaction machinery | Engineering default, confirm no objection |
| Should `webchat_widget` direct hashes support rotation the same way other resource types do (`POST /v1/directs/{id}/regenerate`)? | Yes by default — platform-wide feature, not webchat-specific work | Engineering default, confirm no objection |
| Message content retention window for visitor-volunteered PII | Align with whatever policy exists for conversation-manager messages today, or surface as a pre-existing platform gap if none exists | CEO/CTO |
| Widget JS SDK / embeddable snippet delivery, including exactly how/when the client persists and re-sends `session_id` (localStorage key naming, expiry alignment with `idle_timeout`) | Out of scope for this backend design doc; needs its own frontend design once this API is stable, but flag the `session_id` persistence contract explicitly in that follow-up doc since it's the client-side half of this design's security model | Product, Phase 2 planning |
| **WS-subscribe vs REST-send ownership asymmetry (Round-2 reviewer finding).** The REST message-send path checks `session.WidgetID == a.DirectScope.ResourceID`; the WS subscribe path has NO equivalent check — `validateTopics` only verifies `resource_type` match, never that the specific `session_id` in the topic belongs to the token's own `Widget`. A token scoped to Widget A can subscribe to `customer_id:<id>:webchat_session:<any-UUID>`, including a session under Widget B (same customer) or one whose WidgetID doesn't match the token at all, if that UUID is somehow known (leaked referrer, shared/misconfigured proxy, a future "share this session link" feature). This is the exact same mechanism `aicall` already accepts in production — not a new hole introduced by this design — but it is a real, asymmetric guarantee (write is ownership-checked, read-via-subscribe is not) that §12 previously only alluded to inside the "capability-based" framing rather than stating plainly. | Accept as an inherited, platform-wide characteristic of the direct-token WS model (same as `aicall` today), not a webchat-specific defect requiring new engineering; state it explicitly in customer-facing security documentation once a UI ever exposes a "session link" to agents, since that is the point at which this asymmetry becomes actionable rather than theoretical | CEO/CTO, informational — no code change requested |
| **Widget status/liveness not checked at Session-create time (Round-2 reviewer finding).** `AuthBoot` (verified source) only checks `Customer.Status == StatusActive`; it does not check the resource (`Widget`) itself. A visitor holding a valid direct-token JWT for a `Widget` that has since been set `StatusInactive` (or soft-deleted) can still successfully call `SessionHandler.Create` and start a new conversation against a widget the customer believes is disabled. | `SessionHandler.Create` should reject with a clear error if `Widget.Status != StatusActive` or `Widget.TMDelete` is set (soft-deleted); this is a small, uncontroversial addition to Implementation Order step 5, not a design change | Engineering default, confirm no objection |
| **Idle sweep / `Close()` racing with an in-flight `MessageSend` (Round-2 reviewer finding).** No version/lock discussion: if the idle sweep (or an explicit agent `Close()`) transitions a `Session` to `ended` between `SessionHandler.Get()` returning `active` and `MessageSend()` committing, a message can be persisted against an already-`ended` session with no re-check. Lower likelihood with the default 1800s idle timeout, but a concurrent explicit `Close()` mid-message is plausible. | Phase 1: accept as a low-severity product consequence (a message arrives just as/after a session closes — comparable to the double-`Create` race already accepted for the no-`session_id` case), OR add a lightweight status re-check inside `MessageSend`'s transaction if trivial to implement. Either is acceptable; flag the decision explicitly rather than leaving it silently unaddressed | Engineering default, confirm no objection |
| **`DELETE /v1/webchat/widgets/{id}` is silent on existing active Sessions under that Widget (Round-2 reviewer finding).** Per the verified `validateTopics` mechanism, an in-flight visitor's WS subscription would continue to function after the parent `Widget` is deleted (topic validation only checks `resource_type`, never `Widget` liveness) — this is a direct, intentional consequence of the capability-based design, not a bug, but the document should say so explicitly rather than leave it implicit. | State explicitly: deleting a `Widget` revokes future session issuance (the direct hash) but does NOT retroactively terminate already-`active` Sessions or their live WS subscriptions; if the product wants "delete = kill live conversations too," that requires an explicit cascade in `WidgetHandler.Delete` (bulk `Close()` of active Sessions), which is not in Phase 1 scope by default | CEO/CTO, confirm desired behavior before ship |

## 16. Conversation-Manager Integration (new in v6)

### 16.1 Why (and why not full delegation)

pchero required that webchat conversations surface in
`bin-conversation-manager`'s unified Conversation/Message view alongside
SMS/LINE/WhatsApp, so agents get a single cross-channel timeline and any
existing Contact/Case linkage (`Conversation.Metadata.ContactCaseID`, used
by `bin-contact-manager`) applies to webchat automatically without new
contact-manager work.

The integration shape is **not** "conversation-manager owns webchat data" —
that was considered and rejected. Instead it mirrors the platform's
existing precedent for exactly this situation:
**`bin-message-manager` owns SMS `Message`/`Target` records completely
independently, and conversation-manager consumes a `message_created` event
to build its own, separate `Conversation`/`Message` view.**
`bin-webchat-manager` occupies the same position message-manager occupies
for SMS. Concretely:

- `bin-webchat-manager` keeps its own `webchat_widgets`/`webchat_sessions`/
  `webchat_messages` tables (v5, unchanged) as the source of truth for
  real-time visitor-facing delivery (direct-token auth, WS fan-out, Flow
  triggering).
- `bin-conversation-manager` keeps its own `conversation_conversations`/
  `conversation_messages` tables (unchanged schema) as the source of truth
  for the cross-channel agent-facing timeline.
- The two are connected by ONE event subscription, one direction only:
  webchat-manager → conversation-manager. Conversation-manager never calls
  back into webchat-manager.

### 16.2 Why webchat does NOT get a `conversation-manager` `Account` row

This was the main design fork in v6 and is worth stating explicitly because
the earlier direction of this conversation (before this revision) assumed
the opposite.

Verified against the real `Account`/`Conversation` structs and the SMS
ingestion path:

- `Account` is not a universal "one per channel type" concept.
  **SMS has no `Account` at all.** `conversationhandler.eventSMS()` and
  `execute_mode.go`'s `runExecuteModeFlowMessage` resolve the Flow source
  via `bin-number-manager`'s `Number.MessageFlowID` (looked up by
  `cv.Self.Target`, the phone number), not via any `Account` row.
  `Conversation` identity for SMS is `ConversationGetBySelfAndPeer(self,
  peer)` — an address-pair lookup — not `(account_id, dialog_id)`.
- LINE/WhatsApp DO use `Account`, because their `Conversation` identity is
  genuinely `(account_id, dialog_id)`: the `account_id` is load-bearing for
  conversation lookup itself, not just credential storage.
- Webchat's cardinality is: `Widget` = one fixed VoIPbin-side identity,
  reused across every visitor who ever chats with it (structurally
  identical to "our phone number" in the SMS case). Each visitor's
  `Session` = a fresh, per-conversation identity that exists once and is
  never reused (structurally identical to "the caller's phone number" in
  the SMS case, except unpredictable/random instead of a real telephone
  number). **This is the SMS shape, not the LINE/WhatsApp shape.**

Conclusion: webchat gets **no `Account` row in conversation-manager at
all**. `Conversation.Self`/`Conversation.Peer` carry the full identity, the
same way SMS's `Self`/`Peer` do.

### 16.3 Address type and Conversation identity

`bin-common-handler/models/address` gets one new `Type`:

```go
TypeWebchat Type = "webchat" // opaque identifier (Widget.ID or Session.ID); no sub-form to canonicalize, same class as TypeLine/TypeAgent/TypeConference
```

`NormalizeTarget`/`ValidateTarget` require the corresponding `case
TypeWebchat` added alongside the existing `TypeNone, TypeAgent, TypeAI,
TypeAITeam, TypeConference, TypeExtension, TypeLine` "opaque identifier"
group in both functions (`address/normalize.go`, `address/validate.go`) —
UUID validation, no telephone/email canonicalization, mirroring exactly how
`TypeLine` is handled today.

`Conversation` fields for a webchat thread:

```go
Type:     conversation.TypeWebchat
DialogID: ""                                              // unused for address-pair-identified types, same as SMS
Self:     commonaddress.Address{Type: TypeWebchat, Target: <Widget.ID>}
Peer:     commonaddress.Address{Type: TypeWebchat, Target: <Session.ID>}
```

Lookup/creation uses `ConversationGetBySelfAndPeer(self, peer)`, the
existing SMS code path — no new lookup method needed.

### 16.4 New webchat-manager → conversation-manager event flow

```
1. Visitor sends a message (or Flow delivers one via webchat_message_send).
2. bin-webchat-manager persists to webchat_messages, publishes
   webchat_message_created (unchanged from v5 — this event already existed
   for §8's webhook/flow-variable purposes; conversation-manager becomes a
   second, independent subscriber of the same event).
3. bin-conversation-manager's pkg/subscribehandler (NEW):
   - subscribeTargets gains bin-manager.webchat-manager.event
   - processEvent's switch gains:
     case m.Publisher == publisherWebchatManager && m.Type == "webchat_message_created":
         err = h.processEventWebchatMessageMessageCreated(ctx, m)
   mirroring the existing publisherMessageManager case verbatim.
4. processEventWebchatMessageMessageCreated -> conversationHandler.eventWebchat():
   - unmarshal the webchat-manager Message payload (session_id, widget_id,
     direction, text, tm_create)
   - self := Address{Type: TypeWebchat, Target: widget_id}
   - peer := Address{Type: TypeWebchat, Target: session_id}
   - cv, err := ConversationGetBySelfAndPeer(self, peer); if not found,
     ConversationCreate(...) — same shape as eventSMS's Conversation
     resolution.
   - create the conversation-manager Message record:
     ReferenceType: message.ReferenceTypeWebchat  (NEW enum value)
     ReferenceID:   <webchat-manager Message.ID>  (cross-service pointer,
                    same pattern as ReferenceTypeMessage -> message-manager
                    Message.ID today)
     Direction:     incoming | outgoing  (mapped from webchat's inbound/outbound)
     Source/Destination: derived from Self/Peer + Direction, same helper
                    (deriveEndpoints) SMS/LINE/WhatsApp already use.
   - publish conversation_created (if new) + message_created (conversation-
     manager's own event, distinct from webchat-manager's).
5. execute_mode.go's runExecuteModeFlow dispatch on cv.Type == TypeWebchat:
   returns nil immediately, no Account/Number lookup, no
   FlowV1ActiveflowCreate call — see §16.5.
```

### 16.5 No double Flow trigger — explicit design decision, not an oversight

Both webchat-manager and conversation-manager COULD independently trigger
an activeflow for the same inbound visitor message (webchat-manager via
`Widget.FlowID`, conversation-manager via a hypothetical webchat flow
source). This was identified explicitly and pchero decided:

**`bin-webchat-manager` retains exclusive Flow-triggering responsibility
for the real-time visitor-facing path. `bin-conversation-manager`'s webchat
ingestion path (§16.4) NEVER calls `FlowV1ActiveflowCreate`.**

Rationale, and how this differs from every other channel:

- For SMS/LINE/WhatsApp, the *only* Flow-triggering layer in the entire
  inbound path is conversation-manager (`execute_mode.go`). The
  originating service (`bin-message-manager` for SMS; the LINE/WhatsApp
  webhook arrives directly at conversation-manager, no separate service in
  between) never triggers a Flow itself — so there is no double-trigger
  risk for those channels by construction, not by an explicit prevention
  rule.
- Webchat is structurally different: `bin-webchat-manager` is NOT a dumb
  ingestion layer like message-manager is for SMS. It has a *product
  requirement* v5 already established (§2, §5) that a visitor's first
  message must produce a real-time response (AI reply, welcome message)
  with no perceptible extra latency — the entire reason `Widget.FlowID`
  triggering lives in webchat-manager, at the point of first contact,
  rather than one event-subscription hop away in conversation-manager.
  Moving that responsibility to conversation-manager (mirroring SMS exactly)
  was considered and rejected specifically because it would add a
  publish-then-consume round trip to the critical real-time path for no
  benefit — conversation-manager's timeline is an agent-facing read model,
  not the visitor-facing response path.
- Concretely, this means:
  - `bin-flow-manager`'s `reference_type=webchat` activeflow (triggered by
    webchat-manager, §9) is the ONLY activeflow a webchat message ever
    produces automatically.
  - conversation-manager's `Conversation` for a webchat thread MUST NOT
    carry a flow-triggering identity the way `Account.MessageFlowID` or
    `Number.MessageFlowID` do for other channels — there is deliberately no
    such field to look up, because there is no `Account` row at all
    (§16.2) and `execute_mode.go`'s webchat branch is a hard no-op, not a
    "look up flow id, usually nil" branch.
  - If a human agent later takes ownership of a webchat `Conversation`
    (`OwnerType=agent`), `getExecuteMode` already returns
    `ExecuteModeAgent` for ANY conversation type purely from
    `cv.OwnerType`/`cv.OwnerID` — this requires no webchat-specific code,
    it's the existing dispatch working unmodified. The `ExecuteModeFlow`
    branch (which would otherwise attempt a Flow trigger) is what must be
    hard no-op'd for `TypeWebchat` specifically, precisely because it's
    the one branch that could double-trigger.

### 16.6 Message field mapping (webchat-manager → conversation-manager)

| conversation-manager `Message` field | Derived from |
|---|---|
| `ConversationID` | Resolved/created `Conversation.ID` (§16.4) |
| `Direction` | `incoming` if webchat `direction=inbound`, `outgoing` if `direction=outbound` |
| `ReferenceType` | New enum value `ReferenceTypeWebchat` |
| `ReferenceID` | webchat-manager's `Message.ID` (cross-service pointer, read-only from conversation-manager's side) |
| `Source`/`Destination` | Derived from `Self`/`Peer` + `Direction` via the existing `deriveEndpoints` helper — no new logic |
| `Text` | webchat `Message.Text`, copied verbatim |
| `Medias` | Empty (Phase 1 webchat has no attachments, §2 Out of scope) |

This is a **one-way, eventually-consistent copy**, not a live reference.
conversation-manager's copy of a webchat message can never be edited back
into webchat-manager; conversation-manager is a read model for the agent
timeline, matching how it already treats SMS/LINE/WhatsApp messages.

### 16.7 Open Questions (v6 additions)

| Question | Recommendation | Decision owner |
|---|---|---|
| If conversation-manager's event consumption lags or fails (RabbitMQ backlog, transient DB error), does the agent-facing timeline silently miss messages, and is there a reconciliation/replay mechanism? | Same limitation SMS/LINE/WhatsApp already accept today (subscribehandler has no dead-letter/replay story documented) — not a webchat-specific gap to solve in Phase 1, but worth confirming this is an accepted platform-wide limitation rather than an oversight | CEO/CTO, confirm no objection |
| Does `bin-contact-manager`'s existing Case-linking logic (`Conversation.Metadata.ContactCaseID`) need ANY webchat-specific change, or does it work unmodified because it operates on `Conversation`, not on channel type? | Expected to work unmodified — `ContactCaseID` linking is conversation-type-agnostic per the existing code; verify with a contact-manager owner as a implementation-time sanity check, not a design blocker | Engineering default, confirm during implementation |
| Should `Conversation.Peer.TargetName` be populated with anything human-readable for a webchat visitor (LINE populates a display name; a webchat visitor has none by default)? | Leave empty in Phase 1; if/when a pre-chat form (§2, deferred) collects a visitor name, that becomes the natural place to populate it | Product, Phase 2 |
| Retry/ordering: could conversation-manager ever process a `webchat_message_created` event for message N+1 before message N (out-of-order delivery)? | Same question already applies to SMS/LINE/WhatsApp today (RabbitMQ does not guarantee cross-message ordering to a single consumer group under retries) — not a new webchat-specific risk; confirm existing platform behavior/assumption rather than solving uniquely for webchat | Engineering default, confirm no objection |

---

## Review Summary

**v1 → v2 → v3**: Two independent review rounds (CHANGES REQUESTED →
APPROVE WITH COMMENTS) against a now-abandoned self-hosted-WebSocket
architecture. Superseded entirely.

**v3 → v4**: Full architecture pivot — reuse `bin-api-manager`'s existing
`/ws` and `bin-direct-manager` + `POST /auth/boot` instead of
webchat-manager hosting its own ingress and minting its own tokens.

**v4 → v5 (this revision).** Round 1 review of v4 returned **CHANGES
REQUESTED**, correctly identifying: (a) Critical — `Direct.resource_id =
Widget.ID` is shared across all visitors of a widget, and combined with the
platform's 4-part topic shape, this meant every visitor would receive
every other visitor's messages (a live data leak, not a hypothetical); (b)
High — the fix (\"maybe a 5th topic segment?\") was speculative, not
verified against the real `validateTopics` code; (c) Medium — adding a new
`visitor_id` JWT claim would itself contradict §13's \"no auth-pipeline
changes\" claim.

v5 resolves all three by reading the actual code instead of guessing: the
platform already has an established parent/child direct-token pattern in
production for AI voice widgets (`ai` parent, `aicall` child,
`directResourceMapping` static table, `validateTopics`'s explicit
non-validation of the topic's `resource_id` segment). Applying that exact,
verified pattern to `Widget`/`Session` eliminates the need for any new JWT
claim, any `pkg/websockhandler` change, or any topic-shape extension — the
only api-manager change becomes a single new entry in an existing static
map. `Session.ID` (server-generated, unguessable, never derived from any
client-supplied identity claim) replaces v4's `VisitorID` field entirely,
removing the "source of truth TBD" schema gap the Round-1 reviewer flagged
at the DDL level. The `GetOrCreate` concurrency race the reviewer flagged
is also eliminated by construction (split into unconditional `Create` +
ownership-checked `Get`, since there is no longer a shared key two
concurrent requests could race on).

**Round 2 review of v5** returned **APPROVE WITH COMMENTS**. The reviewer
independently traced the fix against the same verified `validateTopics`/
`AuthBoot`/`AIcallCreate` source code quoted in the Revision Notice above and
confirmed it holds: 4-part topic validation genuinely never checks
`resource_id`, the `directResourceMapping` addition is genuinely a pure
data-table change, and the REST-layer ownership check is a correct,
non-superficial adaptation of `AIcallCreate`'s existing pattern (not just a
plausible-sounding copy). No Critical or High finding remained, and no new
issue was introduced by the v4→v5 edit itself. Four Medium-severity gaps
were identified and folded into §15 above (WS-subscribe/REST-send ownership
asymmetry, Widget status check at Session-create time, idle-sweep/Close()
race with in-flight MessageSend, and DELETE-cascade behavior toward live
sessions) — none require another design round; all are implementable
directly as part of Implementation Order steps 4-6.

**v5 → v6 (this revision).** Purely additive — §1-§15 unchanged except
§13/§14 gaining `bin-conversation-manager` rows/steps. Adds a new
integration (§16) surfacing webchat conversations in conversation-manager's
cross-channel agent timeline, requested directly by pchero after the v5
review closed. Mirrors the platform's existing `bin-message-manager` → SMS
pattern rather than inventing a new one: webchat-manager keeps owning its
own data and publishes an event; conversation-manager independently builds
its own `Conversation`/`Message` view from that event, exactly as it
already does for SMS via `eventSMS`/`ConversationGetBySelfAndPeer`.
Verified against real code that `Account` is not a universal concept (SMS
has none — Flow source resolves via `bin-number-manager`'s `Number`
instead) and that webchat's cardinality (`Widget`=fixed self,
`Session`=per-visit peer) matches SMS's `(self, peer)` address-pair shape,
not LINE/WhatsApp's `(account_id, dialog_id)` shape — so webchat gets no
`Account` row in conversation-manager at all. The double-Flow-trigger risk
this integration would otherwise introduce (both webchat-manager and
conversation-manager independently triggering an activeflow for the same
message) is resolved by an explicit CEO/CTO decision, not left ambiguous:
webchat-manager keeps exclusive Flow-triggering responsibility for the
real-time visitor-facing path; conversation-manager's webchat ingestion
path is a hard no-op for Flow triggering, matching how message-manager
itself never triggers a Flow for SMS.

This document has not yet been through a review round in its v6 form.
Pending a fresh Step 4 review focused on §16.

**v6 → v6 (Round 3, this revision, §16-only).** Verdict: **APPROVE.** The
reviewer independently verified every factual code claim in §16 against the
live monorepo rather than trusting the doc's prose: `eventSMS`,
`ConversationGetBySelfAndPeer`, `NormalizeTarget`'s opaque-identifier case
list, and the `message.ReferenceType` enum all match §16's description
field-for-field. §16.5's no-double-Flow-trigger claim turned out to be
**stronger than documented**: `runExecuteModeFlow`'s dispatch already
no-ops on any unrecognized `cv.Type` (verified via an existing test,
`Test_runExecuteModeFlow_unsupportedTypeIsNoop`), so `TypeWebchat` inherits
correct behavior with zero new code at that layer — Implementation Order
step 14 is now a verification-only item, not new code. The reviewer also
confirmed the outbound path (`MessageEventSent`) never calls
`runExecuteModeFlow` at all, closing off a flow-trigger risk on that side
by construction. One real, previously-unflagged gap was found and folded
into Implementation Order step 13: `conversationHandler.Event()` in
`event.go` is a SECOND, distinct switch from `subscribehandler.processEvent`,
and its `default` branch returns a HARD ERROR rather than a no-op — an
implementer following §16.4's narrative literally could plausibly miss
adding the required `case conversation.TypeWebchat` there, producing an
error logged on every webchat message. This is a documentation-completeness
fix, not a design flaw, and required no further review round.

This document is considered design-complete pending CEO/CTO sign-off on the
remaining Open Questions in §15 and §16.7. §16 has passed its Round 3
review (APPROVE); no further review round is pending.
