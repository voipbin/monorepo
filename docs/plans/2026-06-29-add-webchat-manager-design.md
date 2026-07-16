# bin-webchat-manager Design

## Revision Notice (v8 — Widget theming/appearance customization)

**pchero asked whether the widget's appearance can be customized the way
Intercom's is** ("브랜드 컬러/로고/위치 등을 위젯 대시보드에서 바꿀 수 있나").
v7 and earlier gave `Widget` exactly three customer-facing knobs
(`WelcomeMessage`, `FlowID`, `SessionIdleTimeout`) — none of them visual.
Matching Intercom-class appearance customization requires new schema, not
just new UI, so this is a real design revision, not a pure frontend task.

**v8 adds a `ThemeConfig` field to `Widget`**, following the exact
platform precedent already used elsewhere for "small bag of typed,
customer-editable settings that don't deserve their own table":
`bin-conversation-manager`'s `Account.ProviderData` (a `json.RawMessage`
column holding per-account-type settings, e.g. WhatsApp's
`phone_number_id`/`app_secret`). Concretely:

- `Widget.ThemeConfig` — a new, optional, typed struct (NOT a raw
  `json.RawMessage` blob, since unlike `Account.ProviderData` there is
  only ONE widget "type," so there's no per-type-shape problem to solve
  with an untyped column): `PrimaryColor`, `LogoURL`, `Position`
  (`bottom_right`/`bottom_left`), `OfflineMessage`. Stored as a single
  JSON column (`theme_config` in the DDL) for the same reason
  `Metadata`-style fields are already stored this way elsewhere in the
  platform (avoids a wide, sparsely-populated table; all fields are
  optional cosmetic overrides with sane defaults when absent).
- `WidgetHandler.Create`/`Update` (§5) gain a `themeConfig
  *widget.ThemeConfig` parameter (nilable — omitting it entirely keeps
  the existing default appearance, matching how `WelcomeMessage`/`FlowID`
  are already optional).
- `POST`/`PUT /v1/webchat/widgets` (§7) gain an optional `theme_config`
  body field with the same four sub-fields.
- **No new endpoint, no new service, no new auth surface.** This is
  additive to the existing Widget CRUD — everything else in v7 (session
  creation, message send, rate limiting, the Round 4 findings) is
  unchanged.
- The widget-embed snippet (frontend deliverable, not part of the API
  design) reads `GET /v1/webchat/widgets/{id}`'s `theme_config` at load
  time and applies it to the floating bubble/panel it renders — see the
  companion `voipbin-webchat-widget-snippet.js` reference implementation.

**Explicitly out of scope for v8** (same "Phase 2" bucket as other
deferred cosmetic/product-config items in §2): custom CSS injection,
multiple themes per widget (e.g. dark/light), avatar per-agent (vs. one
logo for the whole widget), font selection. `ThemeConfig`'s four fields
are chosen to cover the highest-value 80% (Intercom's own "Brand" tab is
similarly minimal: primary color, avatar/logo, widget position) without
opening an arbitrary CSS-injection attack surface on a page rendered
inside a customer's own site.

## Revision Notice (v7 — explicit session creation, decoupled from first message)

**pchero identified a UX/API-shape gap after re-verifying the direct-hash →
JWT flow against real code**: v5/v6 create a `Session` implicitly, as a
side effect of the visitor's first message (`SessionHandler.Create` was
called from inside the message-send path whenever no `session_id` was
supplied). This works, but has two real costs:

1. **No way to show `Widget.WelcomeMessage` before the visitor types
   anything.** A `Session` (and therefore a `session_id` to scope a WS
   subscription to) doesn't exist until the visitor sends their first
   message — so the widget can't display a welcome message or open its
   WebSocket subscription at load time, only after the first keystroke.
2. **`session_id` was an optional field with two meanings** (absent = create;
   present = continue) baked into a single endpoint, requiring a branch in
   both the API contract and every caller's mental model.

**v7 makes Session creation an explicit, separate step**, decoupled from
sending the first message — mirroring how `aicall` creation is already a
distinct RPC from sending an `aimessage` to that call. Concretely:

- New endpoint: `POST /v1/webchat/sessions` (direct token) — creates a
  `Session` immediately on widget load, before the visitor types anything.
  Returns `{ session_id, welcome_message }` so the widget can render the
  welcome message and open its `/ws` subscription right away.
- `POST /v1/webchat/sessions/messages` is replaced by
  `POST /v1/webchat/sessions/{id}/messages` (direct token) — `session_id`
  is now a required path parameter, never an optional body field. The
  "create vs continue" branch in `SessionHandler`/the servicehandler is
  eliminated entirely; every message-send call is now a `Get` + ownership
  check + `MessageSend`, full stop.
- `SessionHandler.Create` no longer takes an implicit trigger from message
  content — it is invoked directly by the new session-create endpoint.
- The Round-1/Round-2 rate-limiting decision (§15) is extended: the new
  `POST /v1/webchat/sessions` endpoint gets the SAME IP-based
  `middleware.RateLimit(20, 40)` applied to the sessions route group,
  since it is now a widget-load-time call an attacker could otherwise
  spam to create unbounded empty Sessions. **Round 4 reviewer finding**:
  since `POST /v1/webchat/sessions/{id}/messages` now serves BOTH the
  visitor's direct-token path AND the agent/dashboard's customer-JWT path
  (§7), this IP-based limiter must be scoped to the direct-token-
  authenticated branch only — applying it to the whole route group risks
  rate-limiting legitimate agent/dashboard traffic (e.g. several agents
  behind one office NAT) under a budget sized for anonymous visitor abuse.
- Widget-level Flow triggering (§9) still fires on the FIRST INBOUND
  MESSAGE, not on Session creation — an empty Session with no message
  should not itself start an activeflow. This is unchanged in spirit from
  v6, just re-anchored to the (now separate) message-send step rather
  than being conflated with session creation. **Round 4 reviewer finding
  (Medium, accepted as a tracked risk, not silently assumed safe)**: the
  "first inbound message" check inside `MessageSend` is a read-then-act
  race if not done under row-level locking or single-writer serialization
  per session — unlike the session-creation double-fire race (§4, which
  only duplicates a cosmetic Session row), a double-fired first message
  racing this check could cause BOTH calls to observe "zero prior
  messages" and both trigger `FlowV1ActiveflowCreate`, producing a
  visible, customer-facing double-response (duplicate AI reply or
  duplicate queue-join). This must be resolved with either a serialized
  message-count check (e.g. `SELECT ... FOR UPDATE` or an equivalent
  single-writer guarantee per session) at implementation time — not
  treated as equivalent in severity to the session-row-duplication case
  it superficially resembles.
- **Round 4 reviewer finding (Medium): the `webchat_session_created`
  webhook's meaning changes under v7** — pre-v7 it only fired when a
  visitor sent an actual message (a real-engagement signal); v7 fires it
  on every widget page load, before any message exists. Page-view bots,
  crawlers, and link/preview-fetchers now each spawn a persistent Session
  row and a webhook event. Any downstream consumer of
  `webchat_session_created` that previously read it as "someone started
  chatting" will see a materially higher false-positive rate. This is a
  genuine behavior change, not the same accepted concern as the v5
  double-`Create` race (which was scoped to message-triggered creation,
  not page-load-triggered creation) — flagged here explicitly so
  downstream webhook consumers are not silently surprised by the
  broadened blast radius.

**Direct-hash → JWT generation (`AuthBoot`) was independently re-verified
against the live `bin-api-manager/pkg/servicehandler/boot.go` source in
this pass and found to need NO changes**: the hash-format check, the
`DirectV1DirectGetByHash` lookup, the `Customer.Status == StatusActive`
check, the `directResourceMapping` lookup, and the `DirectScope`/JWT
construction are all exactly as previously documented. The only
already-known gap (`AuthBoot` checks Customer liveness, not Widget
liveness) is the same gap already recorded and resolved in §15 (Widget
status check added to `SessionHandler.Create`) — re-verification found
nothing new to fix there.

**Everything else from v6 is unchanged**: `bin-webchat-manager` remains a
plain Class A RabbitMQ RPC manager owning `Widget`/`Session`/`Message`;
the direct-token identity model (`Widget`=parent, `Session`=child,
mirroring `ai`/`aicall`) is unchanged; the WebSocket delivery path is
unchanged; the conversation-manager integration (§16) is unchanged;
Round 1/2/3 review findings are unchanged and still hold. **This revision
only touches the session-creation/message-send API shape** (§5, §7) and
the corresponding Implementation Order steps (§14).

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

// WidgetPosition controls where the floating bubble/panel renders on the
// customer's page. v8.
type WidgetPosition string

const (
    WidgetPositionBottomRight WidgetPosition = "bottom_right" // default
    WidgetPositionBottomLeft  WidgetPosition = "bottom_left"
)

// ThemeConfig holds cosmetic, customer-editable widget appearance
// settings. All fields are optional; a nil ThemeConfig or empty field
// falls back to the platform default (blue bubble, no logo, bottom-right,
// no offline message). v8 — see Revision Notice for scope rationale.
type ThemeConfig struct {
    PrimaryColor   string         `json:"primary_color,omitempty"`   // hex, e.g. "#2563eb"; validated server-side
    LogoURL        string         `json:"logo_url,omitempty"`        // https URL only
    Position       WidgetPosition `json:"position,omitempty"`        // default: bottom_right
    OfflineMessage string         `json:"offline_message,omitempty"` // shown when Widget.Status != StatusActive
}

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

    // ThemeConfig: cosmetic appearance overrides (v8). Nil = all defaults.
    ThemeConfig *ThemeConfig `json:"theme_config,omitempty"`

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
    theme_config          JSON,

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
    Create(ctx context.Context, customerID uuid.UUID, name string, welcomeMessage string, flowID uuid.UUID, idleTimeout int, themeConfig *widget.ThemeConfig) (*widget.Widget, error)
    Get(ctx context.Context, id uuid.UUID) (*widget.Widget, error)
    List(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]*widget.Widget, error)
    Update(ctx context.Context, id uuid.UUID, name string, welcomeMessage string, flowID uuid.UUID, idleTimeout int, themeConfig *widget.ThemeConfig) (*widget.Widget, error)
    Delete(ctx context.Context, id uuid.UUID) (*widget.Widget, error)
}

type SessionHandler interface {
    // Create makes a brand new Session for a Widget. Invoked directly by
    // the new POST /v1/webchat/sessions endpoint (v7) — explicit, not an
    // implicit side effect of the first message. Always succeeds with a
    // fresh UUID — no uniqueness constraint to violate, no race to guard
    // (see §4). Rejects if Widget.Status != StatusActive or
    // Widget.TMDelete is set (§15 Widget-liveness check).
    Create(ctx context.Context, widgetID uuid.UUID) (*session.Session, error)

    // Get + ownership check for a client-supplied session_id. Callers
    // (api-manager's servicehandler) MUST additionally verify
    // session.WidgetID == token's DirectScope.ResourceID and
    // session.CustomerID == token's CustomerID before trusting this —
    // mirrors AIcallCreate's existing `ResourceID != assistanceID` check.
    // v7: this is now the ONLY path into MessageSend; there is no longer
    // a "create if absent" branch here (see Revision Notice).
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
3. Widget load, BEFORE the visitor types anything (v7, explicit step):
   POST /v1/webchat/sessions?token=<jwt>  { } (no body needed)
   -> api-manager verifies a.HasAllowedResourceType("webchat_session")
   -> calls SessionHandler.Create(widgetID = a.DirectScope.ResourceID)
      (rejects if Widget.Status != StatusActive or Widget.TMDelete set)
   -> response: { session_id, welcome_message }
   -> Browser stores session_id (e.g. localStorage) and can immediately
      render Widget.WelcomeMessage and open its /ws subscription (step 4)
      without waiting for the visitor to type anything.
4. Browser: GET https://api.voipbin.net/ws?token=<jwt>
   -> EXISTING code, UNCHANGED. Subscribes to topic:
      customer_id:<id>:webchat_session:<session_id>
   -> validateTopics checks HasAllowedResourceType("webchat_session") only;
      the session_id segment is never re-validated against anything — same
      as the aicall precedent.
5. Visitor's FIRST message (session_id is now ALWAYS known — required path
   param, never an optional body field, v7):
   POST /v1/webchat/sessions/{session_id}/messages?token=<jwt> { text: "hello" }
   -> api-manager calls SessionHandler.Get(session_id), then verifies
      session.WidgetID == a.DirectScope.ResourceID AND
      session.CustomerID == a.CustomerID (ownership check, mirrors
      AIcallCreate) before calling MessageSend.
   -> response includes { message_id, status }
   -> IF this is the FIRST inbound message on this Session AND
      Widget.FlowID set: FlowV1ActiveflowCreate(reference_type=webchat,
      reference_id=session.ID). An empty Session with no message never
      triggers a Flow on its own (Revision Notice) — the trigger condition
      is anchored to "first inbound message," not "session creation."
6. Visitor's SUBSEQUENT messages (same shape as step 5, same endpoint,
   same session_id in the path — there is no longer a distinct "first
   message" vs "follow-up message" API shape, only a distinct FLOW-TRIGGER
   condition inside webchat-manager):
   POST /v1/webchat/sessions/{session_id}/messages?token=<jwt> { text }
7. Flow executes actions (ai_talk / queue_join / webchat_message_send).
   -> webchat_message_send calls MessageDeliver().
   -> webchat-manager publishes webchat_message_created (outbound).
8. api-manager's EXISTING pkg/subscribehandler/websockhandler fans the
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
| POST | `/v1/webchat/widgets` | customer JWT | Create a Widget (also issues a direct hash). Body may include an optional `theme_config` object (v8: `primary_color`, `logo_url`, `position`, `offline_message` — all optional). |
| GET | `/v1/webchat/widgets` | customer JWT | List Widgets |
| GET | `/v1/webchat/widgets/{id}` | customer JWT | Get a Widget (includes the widget snippet / hash and its `theme_config`) |
| PUT | `/v1/webchat/widgets/{id}` | customer JWT | Update a Widget, including `theme_config` (v8) |
| DELETE | `/v1/webchat/widgets/{id}` | customer JWT | Delete a Widget (cascades: revoke direct hash) |
| POST | `/v1/webchat/sessions` | direct token (`resource_type=webchat_widget`, allows `webchat_session`) | **New in v7.** Visitor's browser calls this at widget-load time, before typing anything. Always creates a brand-new Session — no "get or create" branch. Returns `{ session_id, welcome_message }` so the widget can render the welcome message and open its `/ws` subscription immediately. |
| POST | `/v1/webchat/sessions/{id}/messages` | direct token (`resource_type=webchat_widget`, allows `webchat_session`) | **v7: replaces the old `POST /v1/webchat/sessions/messages`.** `session_id` is now a required path parameter, not an optional body field — every call is `Get` + ownership check + `MessageSend`, whether it's the visitor's first message or a follow-up. |
| GET | `/v1/webchat/sessions/{id}/messages` | customer JWT | List a session's messages (agent/dashboard view) |
| POST | `/v1/webchat/sessions/{id}/messages` | customer JWT | Agent/API-authored outbound message (same path as the visitor's direct-token route above; auth type on the request distinguishes caller identity, mirroring how `/aicalls` already serves both `IsAgent()`/`IsAccesskey()` and `IsDirect()` callers on one path) |
| POST | `/v1/webchat/sessions/{id}/close` | customer JWT | Explicitly end a session |

No public/unauthenticated routes. Unchanged from v4.

Example: widget load — create Session before the visitor types anything (v7)

```json
POST /v1/webchat/sessions?token=<direct-jwt>
{}

200 OK
{
  "session_id": "7a1b...",
  "welcome_message": "Hi! How can we help you today?"
}
```

Example: visitor sends their first message (session_id now always known)

```json
POST /v1/webchat/sessions/7a1b.../messages?token=<direct-jwt>
{
  "text": "Hi, I have a question about pricing"
}

200 OK
{
  "message_id": "5e9c...",
  "status": "sent"
}
```

Example: visitor sends a follow-up message (same conversation, same endpoint)

```json
POST /v1/webchat/sessions/7a1b.../messages?token=<direct-jwt>
{
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
   `Create` (v7) is now invoked ONLY by the new explicit session-creation
   endpoint (step 6 below), never implicitly from a message-send call.
6. `bin-api-manager`: new `pkg/servicehandler/webchat_widget.go` and
   `webchat_session.go`. **v7 API shape**: `SessionCreate` (new,
   `POST /v1/webchat/sessions`, always creates, no branch) and
   `SessionMessageSend` (`POST /v1/webchat/sessions/{id}/messages`,
   `session_id` is a required path parameter — implements the ownership
   check from §5, `session.WidgetID == a.DirectScope.ResourceID` +
   `session.CustomerID == a.CustomerID`, mirroring `AIcallCreate`'s
   existing pattern verbatim). Add `middleware.RateLimit(20, 40)` to the
   new `POST /v1/webchat/sessions` route AND to the direct-token-
   authenticated BRANCH of `/v1/webchat/sessions/{id}/messages` ONLY
   (§15 rate-limiting decision, extended in v7 to cover session creation
   too, since it is now a widget-load-time call reachable without sending
   any message). **Round 4 reviewer finding: do NOT apply this limiter to
   the whole `/v1/webchat/sessions` gin route group**, since that group
   also serves the customer-JWT agent/dashboard branch of the same
   `{id}/messages` path — an IP-based limiter sized for anonymous visitor
   abuse would incorrectly throttle legitimate agent traffic (e.g.
   several agents behind one office NAT). Scope the middleware to the
   direct-token branch specifically, or apply it inside the handler after
   auth-type dispatch, not via a blanket route-group `.Use(...)`.
7. `bin-flow-manager`: `reference_type=webchat` + `webchat_message_send`
   action.
8. First-inbound-message flow trigger wiring — anchored to the first
   inbound `Message` on a Session (via `sessionhandler.MessageSend`'s
   internal message-count check), NOT to `Session` creation. An empty
   Session with no messages never triggers a Flow (v7 Revision Notice).
   **Round 4 reviewer finding (Medium, must be resolved here, not
   deferred): the message-count check inside `MessageSend` MUST be
   serialized per session** (e.g. `SELECT ... FOR UPDATE` on the Session
   row, or an equivalent single-writer guarantee) before this step is
   considered complete. Unlike the session-creation double-fire race
   (§4, cosmetic — an extra Session row), a double-fired first message
   racing this unguarded check could cause BOTH concurrent calls to
   observe "zero prior messages" and both call
   `FlowV1ActiveflowCreate`, producing a visible, customer-facing
   double-response (duplicate AI reply or duplicate queue-join). This is
   a correctness requirement for this step, not an accepted Phase-1 risk.
9. Webhook events.
10. OpenAPI spec + REST routes.
11. End-to-end test (v7 flow): hash issuance -> `/auth/boot` -> explicit
    `POST /v1/webchat/sessions` (session created, welcome_message
    returned, NO Flow triggered yet) -> `/ws` subscribe to that exact
    `session_id` topic -> first message via
    `POST /v1/webchat/sessions/{id}/messages` (Flow triggers here, on
    first inbound message, not on session creation) -> second message,
    same endpoint, same session_id -> flow-triggered reply -> event
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

**All CEO/CTO decisions below were confirmed on 2026-07-16. This section
now records decisions, not open items — retained under the original
heading for traceability.**

| Question | Decision | Rationale / Implementation note |
|---|---|---|
| Rate limiting / abuse control on `/auth/boot` and the session endpoints | **Confirmed.** `/auth/boot` already inherits the existing `middleware.RateLimit` applied to the whole `/auth` route group in `cmd/api-manager/main.go` (IP-based, 10 req/s, burst 20) — no new code needed there. **v7**: BOTH new/changed session routes — `POST /v1/webchat/sessions` (session creation, now a widget-load-time call) and the direct-token-authenticated branch of `POST /v1/webchat/sessions/{id}/messages` (message send) — get the SAME middleware applied fresh: IP-based, 20 req/s, burst 40. **Round 4 reviewer finding: this limiter must be scoped to the direct-token branch only, NOT the whole route group**, since `{id}/messages` also serves customer-JWT agent/dashboard traffic that should not share an anonymous-visitor-sized IP budget. Per-session rate limiting (as opposed to per-IP) is explicitly deferred to Phase 2 — the existing `middleware.RateLimit` is IP-keyed only and would need generalizing to key on an arbitrary identifier (session_id) to support it; not required for Phase 1. | Implementation: apply `middleware.RateLimit(20, 40)` to the `POST /v1/webchat/sessions` route and to the direct-token branch of `{id}/messages` specifically (post-auth-dispatch, not a blanket route-group `.Use(...)`), mirroring the existing `/auth` group's usage in spirit but scoped correctly for a dual-auth-type path. |
| `MessageStatus.Delivered` semantics are weaker than a confirmed socket write | **Confirmed: ship without a delivery-ack loop in Phase 1.** Matches every other channel's actual guarantee level. | No implementation change from v5. |
| Widget/Direct two-phase-commit gap | **Confirmed: `Widget.DirectID` nullable; a Widget with `DirectID = NULL` is "provisioning incomplete," excluded from the widget-snippet response until resolved.** | Engineering default, as proposed. |
| Should `webchat_widget` direct hashes support rotation (`POST /v1/directs/{id}/regenerate`)? | **Confirmed: yes**, via the existing generic regenerate endpoint — no webchat-specific rotation logic. | No new code beyond `resource_type` enum registration (§13). |
| Message content retention window for visitor-volunteered PII | **Confirmed: no separate PII policy for webchat.** Standard `tm_delete` soft-delete convention applies, same as every other channel; no webchat-specific retention rule is introduced. | No implementation change. |
| WS-subscribe vs REST-send ownership asymmetry (Round-2 reviewer finding) | **Confirmed: accept as-is, informational only.** Same characteristic `aicall` already carries in production. Flag in customer-facing security documentation only if/when a "share this session link" feature is ever built. | No code change. |
| Widget status/liveness not checked at Session-create time (Round-2 reviewer finding) | **Confirmed: add the check.** `SessionHandler.Create` rejects if `Widget.Status != StatusActive` or `Widget.TMDelete` is set. | Small addition to Implementation Order step 5. |
| Idle sweep / `Close()` racing with an in-flight `MessageSend` (Round-2 reviewer finding) | **Confirmed: accept as a low-severity Phase 1 product consequence, no additional code.** A message arriving in the narrow window between idle-sweep's status transition and `MessageSend`'s commit may attach to an already-`ended` session — rare (default 1800s idle timeout), non-corrupting, and not worth a transactional re-check in Phase 1. | No implementation change. |
| `DELETE /v1/webchat/widgets/{id}` is silent on existing active Sessions under that Widget (Round-2 reviewer finding) | **Confirmed: keep it silent — live sessions are NOT force-terminated on Widget delete.** Deleting a Widget revokes future session issuance (the direct hash) only; already-`active` Sessions and their live WS subscriptions continue functioning until they naturally idle-timeout or are explicitly closed. | Document explicitly in `WidgetHandler.Delete`'s docstring/API docs so this is not mistaken for an oversight. |

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

**All CEO/CTO decisions below were confirmed on 2026-07-16.**

| Question | Decision | Rationale / Implementation note |
|---|---|---|
| If conversation-manager's event consumption lags or fails, does the agent-facing timeline silently miss messages? | **Confirmed: accepted as an existing platform-wide limitation, not a webchat-specific gap to solve.** SMS/LINE/WhatsApp already carry this same limitation today (no dead-letter/replay story documented for `subscribehandler`). | No implementation change. |
| Does `bin-contact-manager`'s Case-linking logic (`Conversation.Metadata.ContactCaseID`) need any webchat-specific change? | **Confirmed: expected to work unmodified**, since `ContactCaseID` linking operates on `Conversation`, not on channel type. | Verify as an implementation-time sanity check, not a design blocker. |
| Should `Conversation.Peer.TargetName` be populated for a webchat visitor? | **Confirmed: leave empty in Phase 1.** | If a pre-chat form (Phase 2) ever collects a visitor name, that becomes the natural place to populate it. |
| Could conversation-manager ever process a `webchat_message_created` event out of order? | **Confirmed: accepted as an existing platform-wide limitation, not a new webchat-specific risk.** SMS/LINE/WhatsApp already carry the same RabbitMQ ordering characteristic. | No implementation change. |

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

**All Open Questions in §15 and §16.7 were confirmed by pchero (CEO/CTO) on
2026-07-16.** This design document is now fully approved and design-complete.
No further sign-off is pending. Notable decisions from this pass:
`/auth/boot` reuses the existing IP-based `middleware.RateLimit` (10 req/s,
burst 20) unmodified; `/v1/webchat/sessions/messages` gets the same
middleware applied fresh at 20 req/s, burst 40 (per-session rate limiting
deferred to Phase 2); no delivery-ack loop, no webchat-specific PII
retention policy, and Widget deletion does NOT force-terminate already-active
Sessions (explicitly confirmed, not merely defaulted). Ready to proceed to
implementation per §14's Implementation Order.

**v6 → v7 (this revision).** Following re-verification of the direct-hash →
JWT boot flow against the live `AuthBoot` source (found accurate, no
changes needed), pchero identified an API-shape gap: implicit Session
creation as a side effect of the first message meant the widget could not
render `Widget.WelcomeMessage` or open its `/ws` subscription until AFTER
the visitor typed something. v7 makes Session creation an explicit,
separate step (`POST /v1/webchat/sessions`, always creates, no branch),
decoupled from message sending (`POST /v1/webchat/sessions/{id}/messages`,
`session_id` now a required path parameter, never an optional body field).
This eliminates the "create vs continue" branch that previously existed in
both the API contract and `SessionHandler`/servicehandler implementations.
Flow-triggering (§9) remains anchored to the FIRST INBOUND MESSAGE, not to
Session creation — an empty Session with no message never starts an
activeflow on its own. The existing rate-limiting decision (§15) is
extended to cover the new session-creation endpoint, since it is now a
widget-load-time call reachable without sending any message. This revision
touches only §5 and §7 (and the corresponding Implementation Order steps
in §14); §16's conversation-manager integration, its Round 3 approval, and
all other Open-Question decisions are unaffected and still hold. This
document is design-complete pending a fresh, narrowly-scoped review of the
v7 API-shape change if one is desired before implementation begins.

**Round 4 review of v7 (§5/§7/§14 API-shape change only)** returned
**APPROVE WITH COMMENTS**. The reviewer independently verified the "mirrors
`/aicalls`" claim in §7 against the real `PostAicalls`/`AIcallCreate`
source and confirmed the shared-path dual-auth pattern (direct token =
visitor, customer JWT = agent, one handler, one path) is precedented, not
fabricated. No objection to the endpoint split itself — it correctly
solves the stated welcome-message/early-`/ws`-subscribe problem. Three
findings were folded directly into the Revision Notice, §14, and §15
above: (1) Medium — the first-inbound-message Flow-trigger check inside
`MessageSend` needs an explicit per-session serialization guarantee (e.g.
row-level locking), since unlike the already-accepted session-creation
double-fire race, a race here would double-trigger a customer-facing Flow
response, not just duplicate a cosmetic row; (2) Medium — the
`webchat_session_created` webhook's meaning changes from "someone started
chatting" to "someone loaded the widget," which downstream webhook
consumers should be told about explicitly rather than left to discover;
(3) Minor — the new IP-based rate limiter must be scoped to the
direct-token-authenticated branch of `{id}/messages` only, since that path
now also serves customer-JWT agent/dashboard traffic that must not share
an anonymous-visitor-sized budget. None of the three require
re-architecting the v7 endpoint split; all are implementable directly
within the existing Implementation Order steps. This document is now
design-complete through v7, Round 4 reviewed, with no further review round
pending.
