# bin-webchat-manager Design

## Revision Notice (v5 — multi-visitor topic scoping resolved via verified precedent)

**v4's Round-1 review returned CHANGES REQUESTED.** The reviewer correctly
identified a Critical gap: v4 set `Direct.resource_id = Widget.ID`, meaning
every visitor of a widget shared the SAME `resource_id`. Combined with the
platform's 4-part WebSocket topic shape
(`customer_id:<id>:<resource_type>:<resource_id>`), this meant every visitor
subscribing to their widget's topic would receive every OTHER visitor's
messages too — a live cross-visitor data leak, not a hypothetical one. v4
flagged this honestly as an Open Question ("add a `visitor_id` claim to
`/auth/boot`?") but had not verified the fix against real code, and the
reviewer additionally caught that adding a new JWT claim would itself
contradict §13's claim of "no auth-pipeline changes."

**v5 resolves this by reading the actual `validateTopics`/`AuthBoot` code
in `bin-api-manager`, not by guessing.** The fix requires no new JWT claim
and no api-manager auth-pipeline logic change — only a one-line addition to
an existing static mapping table, because the platform already has this
exact "one parent resource shared by many children" pattern in production
for AI voice widgets:

- `bin-api-manager/pkg/websockhandler/etc.go`'s `validateTopics` explicitly
  does **not** check the 4-part topic's `resource_id` segment (`tmps[3]`)
  against `DirectScope.ResourceID` — verbatim code comment: *"resource_id
  (tmps[3]) is NOT validated against DirectScope.ResourceID because they
  refer to different things. DirectScope.ResourceID is the parent resource
  (e.g., AI), while tmps[3] is a child resource (e.g., aicall)."* Only the
  topic's `resource_type` segment (`tmps[2]`) is checked, via
  `a.HasAllowedResourceType(tmps[2])`.
- `bin-api-manager/pkg/servicehandler/boot.go`'s `AuthBoot` looks up
  `directResourceMapping[d.ResourceType]` — a static Go map — to populate
  `DirectScope.AllowedResourceTypes`. For `resource_type=ai` today, this
  maps to `["aicall"]`: one AI agent (the parent, reused across every call)
  can have many `aicall` children (one per call), each independently
  addressable via its own topic, with **zero code needed to keep two
  different `aicall`s from seeing each other's events** — the isolation
  comes entirely from `aicall` IDs being non-guessable, randomly generated
  UUIDs, the same "capability-based" trust model this platform already
  ships in production (verified against `pkg/servicehandler/aicall.go`'s
  `AIcallCreate`/`AIcallGet`/etc., all of which gate on
  `HasAllowedResourceType("aicall")` + `CustomerID` match, never on a
  specific `aicall` ID being pre-registered in the token).

**webchat maps onto this precedent exactly, with `Widget` as the parent and
`Session` as the child** — the identical shape as `ai`/`aicall`. This
replaces v4's flawed "shared resource_id, guess a 5th topic segment" design
entirely:

- `bin-direct-manager`: `resource_type = "webchat_widget"` (as in v4).
- `bin-api-manager/pkg/servicehandler/boot.go`: add ONE entry to
  `directResourceMapping`: `"webchat_widget": {"webchat_session"}`. This is
  the entire api-manager auth-pipeline change — a static table entry, not
  new claim-parsing logic, so §13's "no auth-pipeline changes beyond a
  routing/servicehandler layer" claim now actually holds without the
  internal contradiction the reviewer caught.
- **No `visitor_id` JWT claim, no new claim of any kind.** A visitor's
  identity IS their `Session.ID` — generated server-side by webchat-manager
  on the FIRST message of a conversation (no client-supplied identity input
  at all), returned to the browser in the REST response, and round-tripped
  by the client (e.g. `localStorage`) on subsequent messages and the
  WebSocket subscribe call. This is simpler than v4's proposal, not just a
  fix to it — it needed nothing new from `/auth/boot` at all.
- WebSocket subscription topic: `customer_id:<customer_id>:webchat_session:<Session.ID>`.
  Validated by the EXISTING `validateTopics` logic completely unmodified:
  `tmps[2]="webchat_session"` checked against
  `AllowedResourceTypes=["webchat_session"]`; `tmps[3]=<Session.ID>` is
  never checked against anything, exactly mirroring `aicall` — isolation
  comes from `Session.ID` being an unguessable UUID, not from a
  authorization check on that ID within the WS layer. This is a real
  security property with an honest, already-accepted risk profile (see
  §12), not a bug I'm papering over.
- **Session ownership check still happens — at the REST layer, not the WS
  layer** — mirroring `AIcallCreate`'s existing
  `if a.DirectScope.ResourceID != assistanceID { return ...ErrPermissionDenied }`
  pattern: when a client supplies an existing `session_id` on a subsequent
  message, `bin-api-manager`'s new webchat servicehandler must verify
  `session.WidgetID == a.DirectScope.ResourceID` (the token's own scope)
  AND `session.CustomerID == a.CustomerID` before treating that session as
  addressable. This is the exact same shape as the already-shipped
  `AIcallGet`/`AIcallCreate` checks, copied, not invented.

Everything else from v4 (webchat-manager as a plain Class A RPC manager, no
public HTTP surface, no WebSocket code in webchat-manager, reuse of
existing `/ws` fan-out) is unchanged and was validated by the Round-1
reviewer as directionally correct.

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
| `bin-queue-manager` | Human-agent handoff integration | 2 |
| `bin-conversation-manager` | Possible storage delegation | 2 (conditional) |

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

This document is considered design-complete pending CEO/CTO sign-off on the
remaining Open Questions in §15.
