# bin-webchat-manager Design

## 0. Headline Architecture Decision (read this first)

**This design introduces the platform's first unauthenticated public HTTP+WebSocket
ingress and the platform's first anonymous-consumer PII surface.** Every other
`bin-*` REST endpoint requires customer JWT with server-injected `customer_id`.
`POST /v1/webchat/sessions` and the WS upgrade endpoint cannot follow that model
by definition — an anonymous website visitor has no VoIPbin credential. This is
not a routine engineering detail buried among Open Questions; it is an
architecture-boundary decision that should get explicit CEO/CTO sign-off on the
compensating-controls model (origin allow-list + short-lived signed token +
rate limiting, §12) before Phase 1 ships to any environment reachable by real
traffic, including staging. See §12 (Security & Compliance) and §15 (Open
Questions) for the full security/PII discussion this decision requires.

`bin-webchat-manager` is also architecturally a **hybrid service archetype**:
unlike `bin-hook-manager` (stateless HTTP→RabbitMQ thin proxy, no business
logic, no inbound RabbitMQ queue) and unlike a standard Class A RPC manager
(RabbitMQ RPC only, no public HTTP), this service carries both a public
HTTP+WS surface *with* business logic *and* a standard inbound RabbitMQ
`listenhandler` for its authenticated CRUD routes. It does not map cleanly
onto either existing precedent and should be documented as a new service
class, not quietly implied by omission.

## 1. Problem Statement

VoIPbin has no service that manages anonymous web-visitor chat sessions (the
classic "chat widget embedded on a customer's website" use case). The two
services that look adjacent do not fit:

- **bin-conversation-manager** models `Account` as durable *platform
  credentials* (LINE channel secret/token, SMS provider, WhatsApp Meta app
  secret) and `Conversation` as a thread keyed by `(account_id, dialog_id)`
  where both `self` and `peer` are stable platform addresses (phone number,
  LINE user ID). A web visitor has no such stable identity before a session
  starts — there is no phone number, no OAuth identity, often no login at
  all.
- **bin-talk-manager** models `Chat`/`Participant` for *known* entities
  (`owner_type`/`owner_id` referencing an agent, a customer, etc.). It has no
  concept of an anonymous, unauthenticated, expiring session.

What's missing is the layer every CPaaS "web chat widget" product needs
first: issuing a short-lived session identity to an anonymous browser tab,
keeping a WebSocket connection alive against that identity, and enforcing
per-customer widget embed policy (which domains may embed the widget, business
hours, pre-chat form). Without this layer, neither AI-agent auto-response nor
human-agent handoff for web chat can be built, because there is no way to
identify "who is this browser tab" or "which customer's widget is this."

## 2. Scope

### In scope (Phase 1)

- New service `bin-webchat-manager`: visitor session issuance/lifecycle,
  widget embed configuration (`Source`), WebSocket message transport, and
  message persistence (self-contained — see §15 Open Questions for why
  conversation-manager delegation is deferred, not assumed).
- Public HTTPS + WebSocket ingress served directly by webchat-manager (not
  proxied through bin-hook-manager — see §10 rationale).
- Origin/domain allow-list enforcement per `Source`.
- Session issuance, heartbeat, idle timeout, reconnect via session token.
- Flow integration: an incoming webchat message can trigger a `bin-flow-manager`
  activeflow (`reference_type=webchat`), the same mechanism conversation-manager
  uses today for LINE/WhatsApp/SMS. This is what lets Flow decide AI-response
  vs human-agent routing — the service does not hardcode either.
- Outbound message delivery from flow actions / REST API back to the visitor's
  live WebSocket connection.
- Basic delivery events (`webchat_message_created`, `webchat_session_*`) for
  webhook/flow-variable consumption.

### Out of scope (explicitly deferred)

| Item | Phase | Reason |
|---|---|---|
| Human-agent queue handoff (bin-queue-manager integration) | Phase 2 | Requires agent presence/routing UI decisions; Flow can already route to `queue_join` once this session layer exists — no blocking dependency |
| Pre-chat form / lead capture fields | Phase 2 | Product-config surface, not core session mechanics |
| File/image attachments in webchat messages | Phase 2 | Reuses conversation-manager's `medias` pattern once storage question (§15) is settled |
| Typing indicators / read receipts / presence | Phase 2 | Nice-to-have UX, not required for MVP handoff |
| Mobile SDK / push notification parity | Not scoped | Separate product surface (square-talk mobile), unrelated dependency |
| Multi-tab / multi-device session merge | Phase 2 | Requires visitor identity beyond browser-local token (e.g. login binding) |

## 3. Domain Model

### Source

Widget configuration owned by a customer. One customer may have multiple
`Source`s (e.g. separate widgets for marketing site vs support portal).

```go
type Status string

const (
    StatusActive   Status = "active"
    StatusInactive Status = "inactive"
)

type Source struct {
    commonidentity.Identity // ID, CustomerID

    Name   string `json:"name"`
    Status Status `json:"status"`

    AllowedOrigins []string `json:"allowed_origins"` // exact-match origin allow-list, e.g. "https://example.com"

    WelcomeMessage string    `json:"welcome_message,omitempty"`
    FlowID         uuid.UUID `json:"flow_id,omitempty"` // activeflow started on first inbound message; empty = no auto flow

    SessionIdleTimeout int `json:"session_idle_timeout"` // seconds; default 1800 (30m)

    TMCreate string `json:"tm_create"`
    TMUpdate string `json:"tm_update"`
    TMDelete string `json:"tm_delete"`
}
```

Design rule check: no `StatusNone=""`; both values are explicit non-empty
strings.

### Session

A single visitor's chat session, anchored to an in-memory WebSocket connection
on one pod (same anchoring pattern as `bin-pipecat-manager`'s `Pipecatcall`
session — one DB record + one in-process connection).

```go
type SessionStatus string

const (
    SessionStatusPending SessionStatus = "pending" // issued, no WS connection yet
    SessionStatusActive  SessionStatus = "active"   // WS connected
    SessionStatusIdle    SessionStatus = "idle"      // WS disconnected, token still valid, waiting reconnect
    SessionStatusEnded   SessionStatus = "ended"      // expired, closed by visitor, or closed by agent/flow
)

type Session struct {
    commonidentity.Identity // ID, CustomerID

    SourceID uuid.UUID     `json:"source_id"`
    Status   SessionStatus `json:"status"`

    VisitorID  uuid.UUID `json:"visitor_id"`  // stable per browser (from token, not PII)
    OriginHost string    `json:"origin_host"` // validated at WS upgrade

    ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`

    HostID string `json:"-"` // POD_IP; per-pod WS routing, mirrors pipecatcall.HostID

    TMLastActivity string `json:"tm_last_activity"`
    TMCreate        string `json:"tm_create"`
    TMUpdate        string `json:"tm_update"`
    TMEnd           string `json:"tm_end,omitempty"`   // session-lifecycle marker (NOT the delete marker — see tm_delete below)
    TMDelete        string `json:"-"`                  // standard soft-delete sentinel; excluded from JSON like every other bin-* entity
}
```

**`tm_end` vs `tm_delete` — two distinct sentinels, do not conflate.**
`tm_end` marks when a `Session` transitioned to `ended` (a *lifecycle* event —
every session eventually gets a real `tm_end`, this is normal, expected data).
`tm_delete` is the platform's standard soft-delete sentinel
(`DEFAULT '9999-01-01 00:00:00.000000'`) used for GDPR/retention purges and
generic "exclude deleted rows" tooling across every `bin-*` table. `Session`
was missing `tm_delete` in the original draft — every entity table in this
codebase carries it, and `webchat_sessions` is no exception (reviewer finding,
High severity — see Review Summary at the end of this document).

Status lifecycle:

```
pending --(WS upgrade succeeds, origin validated)--> active
active  --(WS closed, token not expired)-----------> idle
idle    --(WS reconnect with valid token)------------> active
idle    --(idle_timeout elapsed)---------------------> ended
active  --(visitor/agent/flow explicit close)---------> ended
pending --(no WS upgrade within 60s)------------------> ended
```

`ended` is terminal. A new `Session` (new token) must be issued after `ended`
— sessions are not resurrected, matching the "expiring, non-durable identity"
nature of an anonymous visitor.

**Reconnect / duplicate-session handling (single-writer enforcement).**
`Upgrade()` is the only path that transitions a session to `active` and sets
`HostID`. To prevent a stale-`HostID` split-brain (e.g. a visitor's old tab
still holds a WS to pod A while a new tab reconnects with the same token to
pod B), `Upgrade()` performs a conditional update:
`UPDATE webchat_sessions SET host_id=?, status='active' WHERE id=? AND status IN ('pending','idle')`.
If the affected-row count is 0 (session was already `active` on another pod),
the new `Upgrade()` call force-closes the *previous* connection first — it
sends an explicit `MessageDeliver`-style RPC to the old `HostID`'s per-pod
queue telling that pod to close its local WS handle for this `sessionID` —
then retries the conditional update. This guarantees at most one live WS per
`Session` at any time and prevents messages from being routed to a queue for
a `HostID` that no longer holds the connection.

**Remote-close failure/timeout behavior (Round 2 reviewer finding, Medium).**
Per-pod queues are volatile and auto-deleted when their pod dies (same
lifecycle as bin-pipecat-manager's). If the old `HostID`'s pod has already
crashed, the remote-close RPC has no consumer and will time out rather than
succeed or explicitly fail. This is NOT an error state — it means the old
connection is already gone. The protocol treats RPC-timeout identically to
RPC-success for this specific call: after a short bounded wait (one RPC
timeout, no retry loop), `Upgrade()` proceeds directly to the conditional
update regardless of whether the remote-close ack was received. A hung/dead
old pod must never block a visitor's legitimate reconnect.

**Pending-session timeout (the "no WS upgrade within 60s" transition).** This
is enforced by the same idle-sweep ticker described in §10, not a separate
per-session timer — the sweep query additionally matches
`status='pending' AND tm_create < now() - 60s`. This is now made explicit in
§10 rather than left as an undocumented mechanism (reviewer finding, Medium).

### Message

```go
type MessageDirection string

const (
    MessageDirectionInbound  MessageDirection = "inbound"  // visitor -> VoIPbin
    MessageDirectionOutbound MessageDirection = "outbound" // VoIPbin -> visitor
)

type MessageStatus string

const (
    MessageStatusSent      MessageStatus = "sent"
    MessageStatusDelivered MessageStatus = "delivered" // ack'd by live WS
    MessageStatusFailed    MessageStatus = "failed"    // no live connection, dropped after retry window
)

type Message struct {
    commonidentity.Identity // ID, CustomerID

    SessionID uuid.UUID        `json:"session_id"`
    Direction MessageDirection `json:"direction"`
    Status    MessageStatus    `json:"status"`

    // SenderID identifies WHO sent an outbound message: an agent user ID for
    // an agent-typed reply via POST /sessions/{id}/messages, or the zero UUID
    // for a flow/AI-originated MessageDeliver() call. Inbound messages leave
    // this empty (the visitor has no VoIPbin identity to record). Needed for
    // Phase-2 agent-handoff audit and to answer "who said this" in any
    // dashboard, without which outbound messages are anonymous even to the
    // customer's own team (reviewer finding, Medium).
    SenderID uuid.UUID `json:"sender_id,omitempty"`

    // ActiveflowID pins this message to the activeflow execution that
    // produced/consumed it (a session's ActiveflowID can only be set once
    // per §3, but recording it per-message lets a later audit/replay
    // correlate a message to a specific flow run without re-deriving it from
    // the session, and future-proofs against ever allowing multiple
    // activeflows per session).
    ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`

    Text string `json:"text"`

    TMCreate string `json:"tm_create"`
    TMDelete string `json:"tm_delete"`
}
```

## 4. Database Schema

```sql
CREATE TABLE webchat_sources (
    id                    BINARY(16)   NOT NULL,
    customer_id           BINARY(16)   NOT NULL,

    name                  VARCHAR(255) NOT NULL,
    status                VARCHAR(16)  NOT NULL,

    allowed_origins       JSON         NOT NULL,
    welcome_message       TEXT,
    flow_id               BINARY(16),
    session_idle_timeout  INT          NOT NULL DEFAULT 1800,

    tm_create             DATETIME(6)  NOT NULL,
    tm_update             DATETIME(6)  NOT NULL,
    tm_delete             DATETIME(6)  NOT NULL DEFAULT '9999-01-01 00:00:00.000000',

    PRIMARY KEY (id),
    INDEX idx_webchat_sources_customer_id_tm_create (customer_id, tm_create),
    INDEX idx_webchat_sources_customer_id_tm_delete (customer_id, tm_delete)
);

CREATE TABLE webchat_sessions (
    id                BINARY(16)   NOT NULL,
    customer_id       BINARY(16)   NOT NULL,
    source_id         BINARY(16)   NOT NULL,

    status            VARCHAR(16)  NOT NULL,
    visitor_id        BINARY(16)   NOT NULL,
    origin_host       VARCHAR(255) NOT NULL,
    activeflow_id     BINARY(16),
    host_id           VARCHAR(64),

    tm_last_activity  DATETIME(6)  NOT NULL,
    tm_create         DATETIME(6)  NOT NULL,
    tm_update         DATETIME(6)  NOT NULL,
    tm_end            DATETIME(6)  NOT NULL DEFAULT '9999-01-01 00:00:00.000000', -- lifecycle marker, see §3
    tm_delete         DATETIME(6)  NOT NULL DEFAULT '9999-01-01 00:00:00.000000', -- standard soft-delete sentinel (was MISSING in v1 — reviewer High finding)

    PRIMARY KEY (id),
    INDEX idx_webchat_sessions_customer_id_tm_create (customer_id, tm_create),
    INDEX idx_webchat_sessions_customer_id_tm_delete (customer_id, tm_delete),
    INDEX idx_webchat_sessions_source_id_status (source_id, status),
    INDEX idx_webchat_sessions_status_tm_last_activity (status, tm_last_activity),
    INDEX idx_webchat_sessions_status_tm_create (status, tm_create) -- backs the pending-session 60s timeout sweep
);

CREATE TABLE webchat_messages (
    id             BINARY(16)   NOT NULL,
    customer_id    BINARY(16)   NOT NULL,
    session_id     BINARY(16)   NOT NULL,

    direction      VARCHAR(16)  NOT NULL,
    status         VARCHAR(16)  NOT NULL,
    text           VARCHAR(4000) NOT NULL, -- capped; see §12 input validation
    sender_id      BINARY(16),             -- agent user ID for agent-sent outbound messages; NULL for flow/AI-originated or inbound
    activeflow_id  BINARY(16),             -- flow execution that produced/consumed this message, if any

    tm_create      DATETIME(6)  NOT NULL,
    tm_delete      DATETIME(6)  NOT NULL DEFAULT '9999-01-01 00:00:00.000000',

    PRIMARY KEY (id),
    INDEX idx_webchat_messages_session_id_tm_create (session_id, tm_create),
    INDEX idx_webchat_messages_customer_id_tm_create (customer_id, tm_create)
);
```

`idx_webchat_sessions_status_tm_last_activity` backs the active/idle
idle-timeout sweep; `idx_webchat_sessions_status_tm_create` backs the separate
`pending` 60-second upgrade-timeout sweep (§3, §10). Both sweeps share one
ticker goroutine but query different predicates.

## 5. Handler Interface

```go
type SourceHandler interface {
    Create(ctx context.Context, customerID uuid.UUID, name string, allowedOrigins []string, welcomeMessage string, flowID uuid.UUID, idleTimeout int) (*source.Source, error)
    Get(ctx context.Context, id uuid.UUID) (*source.Source, error)
    List(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]*source.Source, error)
    Update(ctx context.Context, id uuid.UUID, name string, allowedOrigins []string, welcomeMessage string, flowID uuid.UUID, idleTimeout int) (*source.Source, error)
    Delete(ctx context.Context, id uuid.UUID) (*source.Source, error)
}

type SessionHandler interface {
    // Issue creates a pending Session + short-lived signed token before any WS
    // connection exists. Called from POST /v1/sessions (public, origin-checked).
    Issue(ctx context.Context, sourceID uuid.UUID, originHeader string) (*session.Session, token string, error)

    // Upgrade validates the token (and origin again) and promotes the session
    // to active, anchoring it to this pod's in-memory connection map.
    Upgrade(ctx context.Context, token string, conn *websocket.Conn) (*session.Session, error)

    // MessageSend accepts an inbound visitor message off the WS, persists it,
    // publishes the event, and — on the FIRST inbound message of a session —
    // starts the Source's activeflow if configured.
    MessageSend(ctx context.Context, sessionID uuid.UUID, text string) (*message.Message, error)

    // MessageDeliver pushes an outbound message (from REST/flow action) to the
    // visitor's live WS connection if active; marks status=failed if no live
    // connection is found within the retry window (see §10).
    MessageDeliver(ctx context.Context, sessionID uuid.UUID, text string) (*message.Message, error)

    // Close ends a session explicitly (visitor tab closed, agent ended chat,
    // flow action "webchat_close").
    Close(ctx context.Context, sessionID uuid.UUID) error

    // sweepIdle runs on a ticker; ended = idle sessions past idle_timeout.
    // internal, not exported via RPC.
}
```

### Core flow: inbound message → flow trigger

```
1. Visitor browser: POST /v1/sessions {source_id}  (public, no auth)
   -> validate Origin header against Source.AllowedOrigins (exact match)
   -> SessionHandler.Issue() creates Session{status=pending}, returns signed token
2. Browser opens WS /v1/sessions/ws?token=<token>
   -> SessionHandler.Upgrade() re-validates token + origin, session -> active
   -> if Source.WelcomeMessage set, MessageDeliver() sends it immediately
3. Visitor sends a message over WS
   -> SessionHandler.MessageSend() persists Message{direction=inbound}
   -> publish webchat_message_created event
   -> IF this is the session's first inbound message AND Source.FlowID is set:
        FlowV1ActiveflowCreate(reference_type=webchat, reference_id=session.ID)
        Session.ActiveflowID = returned ID
   -> ELSE IF Session.ActiveflowID already set: forward as a flow variable
        update (voipbin.webchat.message.text) so a "wait for input" style
        action can consume it — mirrors conversation-manager's message-forward
        pattern
4. Flow executes actions (e.g. ai_talk / queue_join / webchat_message_send)
   -> outbound actions call MessageDeliver() -> pushed to the visitor's live WS
```

## 6. LLM Logic

Not applicable directly in this service. AI response generation is Flow's
responsibility (`ai_talk` action against bin-ai-manager), consistent with the
architecture decision in §2 that webchat-manager stays a neutral transport +
session layer and does not hardcode AI-vs-human routing.

## 7. REST API

All endpoints under `/v1/webchat/`. `sources` and non-public `sessions`
endpoints follow standard JWT customer-scoped auth (via api-manager). The
visitor-facing session issuance and WS endpoints are **public, unauthenticated,
origin-validated** — a deliberate divergence documented in §10.

| Method | Path | Auth | Purpose |
|---|---|---|---|
| POST | `/v1/webchat/sources` | customer JWT | Create a Source (widget config) |
| GET | `/v1/webchat/sources` | customer JWT | List Sources |
| GET | `/v1/webchat/sources/{id}` | customer JWT | Get a Source |
| PUT | `/v1/webchat/sources/{id}` | customer JWT | Update a Source |
| DELETE | `/v1/webchat/sources/{id}` | customer JWT | Delete a Source |
| POST | `/v1/webchat/sessions` | **public, Origin-checked** | Issue a visitor session + token |
| GET | `/v1/webchat/sessions/ws` | **public, token+Origin-checked** | WebSocket upgrade |
| GET | `/v1/webchat/sessions/{id}/messages` | customer JWT | List a session's messages (agent/dashboard view) |
| POST | `/v1/webchat/sessions/{id}/messages` | customer JWT | Send an outbound message (agent/API-triggered, not just flow) |
| POST | `/v1/webchat/sessions/{id}/close` | customer JWT | Explicitly end a session |

Example: issue session

```json
POST /v1/webchat/sessions
Origin: https://example.com
{
  "source_id": "d1a2..."
}

200 OK
{
  "session_id": "5e9c...",
  "token": "eyJhbGciOi...",
  "expires_in": 1800
}
```

Example: create Source

```json
POST /v1/webchat/sources
{
  "name": "Support widget",
  "allowed_origins": ["https://example.com", "https://app.example.com"],
  "welcome_message": "Hi, how can we help?",
  "flow_id": "9f3b...",
  "session_idle_timeout": 1800
}
```

List endpoints follow the monorepo convention: `page_token` (cursor,
`tm_create` of last item) + `page_size` (default 100, max 1000).

## 8. Webhook Events

| Event | Trigger | WebhookMessage payload |
|---|---|---|
| `webchat_session_created` | `Issue()` succeeds | `Session` |
| `webchat_session_ended` | `Close()` or idle sweep transitions to `ended` | `Session` |
| `webchat_message_created` | Inbound or outbound message persisted | `Message` |

Follows the shared `WebhookMessage` envelope (`bin-common-handler/models/webhook`,
`event.go`) — see `voipbin-webhook-contract-parity-audit` skill for the
canonical shape; no new envelope needed.

## 9. Flow Variable Integration

| Variable | Value |
|---|---|
| `voipbin.webchat.session.id` | Session UUID |
| `voipbin.webchat.session.source_id` | Source UUID |
| `voipbin.webchat.message.text` | Most recent inbound message text |

Mirrors conversation-manager's `voipbin.conversation.*` variable convention
(§ conversation-manager domain.md) so Flow authors already familiar with
conversation flows do not need to learn a new variable naming scheme.

New action type `webchat_message_send` (analogous to `message_send`) lets a
Flow push an outbound message back to the visitor — the mechanism a Flow uses
regardless of whether the branch decided on AI response or human handoff.

## 10. RabbitMQ Integration & WebSocket Ingress

**Standard RPC surface**: `bin-manager.webchat-manager.request` queue for
Source CRUD and non-public Session operations, following the Class A RPC
manager pattern used by every other `bin-*` service.

**WebSocket is NOT proxied through bin-hook-manager.** hook-manager is
explicitly a stateless HTTP→RabbitMQ thin proxy with no inbound RabbitMQ queue
and no persistent-connection tracking (per its CLAUDE.md: "no business
logic... transforms HTTP payloads into RabbitMQ messages"). A WebSocket
connection is inherently stateful and must be anchored to one pod for the
life of the session — this is architecturally identical to how
bin-pipecat-manager anchors an Asterisk audio WebSocket to one pod via
`HostID = POD_IP` and per-pod queues (`bin-manager.pipecat-manager.request.<POD_IP>`).
webchat-manager reuses that exact pattern, including the **in-memory
per-pod session registry** the earlier draft omitted (reviewer High finding):

- Each pod holds `mapWebchatSession map[uuid.UUID]*Session` guarded by
  `muWebchatSession sync.Mutex`, mirroring `pipecatcallHandler.mapPipecatcallSession`
  / `muPipecatcallSession` exactly. The in-memory `Session` struct embeds the
  live `*websocket.Conn`, a `Ctx context.Context` + `Cancel context.CancelFunc`
  for teardown, and the DB-backed `Session` fields needed to avoid a DB round
  trip on every inbound message.
- `Upgrade()` registers the new entry in this map AFTER a successful
  conditional DB update (see §3's single-writer enforcement) and closes any
  previously-registered local connection for the same `sessionID` if one
  exists (covers the same-pod double-tab case; the cross-pod case is handled
  by the DB-level conditional update + explicit remote-close RPC in §3).
- `MessageDeliver()` looks up `sessionID` in the local map first (same-pod
  fast path). If the RPC call originates on a different pod (the common case
  — a flow action or REST call has no reason to land on the visitor's pod),
  it is routed via `bin-common-handler/pkg/requesthandler` to the per-pod
  queue `bin-manager.webchat-manager.request.<HostID>`, and the RECEIVING
  pod does the local map lookup + WS write.
- `Session.HostID` is set to `POD_IP` at `Upgrade()` time and cleared
  (DB-level) when the local entry is removed on WS close/`Close()`/idle
  sweep — this is what lets a *different* pod know where to route
  `MessageDeliver`.
- If `HostID`'s pod is gone (rolling deploy, crash) — the receiving pod's
  local map has no entry for a `sessionID` addressed to it, or the per-pod
  queue itself is gone (K8s deletes the queue with the pod, matching
  pipecat-manager's volatile per-pod queue semantics) — `MessageDeliver` fails
  fast and the message is marked `status=failed`. No cross-pod session
  migration in Phase 1 (§15 Open Questions); this is an accepted MVP
  limitation, not silent data loss, and — per the reviewer's finding — should
  be understood explicitly as "every rolling deploy interrupts live visitor
  chats," a real operational/business tradeoff, not just an engineering
  footnote.

**Idle & pending sweeps**: a single ticker goroutine runs two queries each
tick:
1. `status IN ('active','idle') AND tm_last_activity < now() - idle_timeout`
   → transition to `ended` (covers both the "WS closed, never reconnected"
   and the "WS open but silent" cases — see the note below on what bumps
   `tm_last_activity`).
2. `status = 'pending' AND tm_create < now() - 60s` → transition to `ended`
   (the "issued a token, visitor never opened the WS" case).

Both transitions publish `webchat_session_ended` and, if a local map entry
still exists on the owning pod (case 1 with a live-but-silent WS), the sweep
issues a remote-close via the per-pod queue so the socket is not orphaned.

`tm_last_activity` is bumped on **any** WS traffic — inbound message, pong
frame in response to a periodic server-initiated ping, or an explicit
client-side heartbeat frame — not only on inbound chat messages. This closes
the ambiguity the reviewer flagged: a session with an open, healthy, but
conversationally-quiet WebSocket (visitor reading, not typing) is kept alive
by the ping/pong heartbeat and is not force-ended by the sweep. Only a
genuinely dead connection (no pong response) or a truly abandoned tab ages
past `idle_timeout`.

**Origin/domain validation** happens twice: once at `POST /sessions` (issue),
once at WS upgrade — a token stolen and replayed from a different origin is
rejected at the second check even if the first check was satisfied honestly.

**Origin matching is exact by default in Phase 1**, which the reviewer
correctly flagged as potentially unworkable for real customers who want
`*.example.com` wildcard coverage or must tolerate `Origin: null` (sandboxed
iframe embeds, some redirect flows). Rather than silently ship exact-match
and discover this at the first customer onboarding, it is promoted from an
implicit omission to an explicit Open Question (§15) with a recommended
default.

## 11. Observability

- Prometheus counter: `webchat_manager_session_total{status}` (issued /
  activated / ended) — mirrors `sentinel_manager_pod_state_change_total`
  labeling convention.
- Prometheus histogram: `webchat_manager_message_delivery_duration_seconds`
  — time from `MessageDeliver()` call to WS write ack.
- Prometheus gauge: `webchat_manager_active_sessions{host_id}` — per-pod live
  session count (backed by `len(mapWebchatSession)` per pod), useful for
  capacity planning before adding cross-pod migration in a later phase.
- Prometheus counter: `webchat_manager_public_endpoint_requests_total{endpoint,result}`
  — request volume and reject-reason breakdown (`origin_denied`,
  `rate_limited`, `token_invalid`, `ok`) specifically for the two public
  endpoints, since these need dedicated abuse-monitoring distinct from every
  other (JWT-gated) endpoint's metrics.
- Trace ID propagated through the idle-sweep goroutine and the flow-trigger
  goroutine via `context.WithTimeout(context.Background(), Xs)` + explicit
  trace field in logrus, per the goroutine convention in this skill.
- Check `bin-common-handler/pkg/requesthandler/main.go#initPrometheus()`
  before naming new metrics to avoid duplicate-registration panic at startup.

## 12. Security & Compliance

- **Public unauthenticated surface is intentional and bounded** — see §0 for
  why this is a headline architecture decision, not a routine detail. `POST
  /v1/webchat/sessions` and the WS upgrade endpoint accept no JWT — that is
  the nature of an embeddable widget hit by anonymous browsers. The
  compensating controls are the origin allow-list (exact match, not
  wildcard/suffix, to avoid subdomain-takeover bypass — see the wildcard
  Open Question below) and a short-lived signed token (HMAC, server secret,
  embeds `session_id` + `source_id` + `exp`) so a leaked token cannot be
  replayed indefinitely or reused against a different `Source`.
- **Rate limiting MUST land before the public endpoint is reachable in any
  environment, including staging** — this is a correction to the original
  draft, which sequenced rate limiting as Implementation Order step 9 (after
  load testing). The reviewer correctly flagged that a public unauthenticated
  POST endpoint with no rate limiting is a live DoS/spam/cost vector from the
  moment it is deployed anywhere reachable, not just at public GA. Revised
  Implementation Order (§14) moves basic per-IP + per-Source rate limiting to
  step 4, immediately after the public endpoints exist and before any
  staged/staging rollout.
- **Input validation on visitor-submitted text**: `text` is capped at 4000
  characters at both the WS handler and the DB column level (`VARCHAR(4000)`,
  §4) — matching the existing platform convention for LLM-facing message caps
  (see the 4000-rune whole-message cap precedent in this skill's LLM-tool
  references). Content is stored as opaque text and is NOT executed/rendered
  as HTML/Markdown anywhere in this service; any future agent-dashboard or
  widget UI that renders message content is responsible for output-encoding
  at render time (XSS is a rendering-layer concern, not a storage-layer one,
  but is called out explicitly here because this is the platform's first
  anonymous-input text surface — omitting the note would leave it implicit).
- **No PII required to start a session.** `VisitorID` is a random UUID
  generated at issue time, not derived from IP/fingerprint — avoids
  inadvertently collecting tracking-grade identifiers by default. If a
  customer's Flow later collects name/email via a pre-chat form (Phase 2),
  that is an explicit, visible, opt-in step, not implicit fingerprinting.
- **Message content MAY contain visitor-volunteered PII** (name, email, order
  number typed into the chat) even though the session itself requires none.
  This is the platform's first anonymous-consumer-facing data surface, and
  the original draft did not address retention/export/deletion. Phase 1
  ships with: (a) `webchat_messages`/`webchat_sessions` participate in the
  standard `tm_delete` soft-delete convention (§4) so they are covered by
  whatever platform-wide retention/purge tooling already exists for other
  `bin-*` tables — no bespoke mechanism invented here; (b) a documented
  default retention window is an Open Question (§15), not silently
  unbounded; (c) no cross-customer aggregation or analytics use of message
  content is in scope for Phase 1.
- **Message content is customer-visible business data**, not shared across
  customers — every query is scoped by `customer_id`, matching every other
  `bin-*` service's ownership convention.
- **No external LLM involvement in this service directly** — LLM exposure
  (if any) happens in the Flow's `ai_talk` action against bin-ai-manager,
  which already has its own PII/GDPR handling for conversation content.

## 13. Affected Services

| Service | Change | Phase |
|---|---|---|
| `bin-webchat-manager` (new) | New service: Source, Session, Message, WebSocket ingress | 1 |
| `bin-flow-manager` | Add `reference_type=webchat` to Activeflow enum; add `webchat_message_send` action type | 1 |
| `bin-api-manager` | New REST routes proxying to webchat-manager (`/v1/webchat/*`); public routes bypass JWT middleware | 1 |
| `bin-openapi-manager` | New OpenAPI paths for `/webchat/sources`, `/webchat/sessions` | 1 |
| `bin-webhook-manager` | No change — reuses existing `WebhookMessage` dispatch, new event keys only | 1 |
| `bin-queue-manager` | Human-agent handoff integration (Flow calls `queue_join` with webchat session context) | 2 |
| `bin-conversation-manager` | Possible storage delegation if Open Question below resolves that way | 2 (conditional) |

## 14. Implementation Order

1. `bin-webchat-manager` scaffold (cmd/, models/source, models/session, models/message, dbhandler, cachehandler for token validation)
2. `sourcehandler` + REST CRUD (customer-JWT-gated) + OpenAPI spec
3. `sessionhandler.Issue` + token signing/verification + public REST endpoint (no WS yet) — testable via REST alone
4. **Rate limiting on the public issue endpoint** (per-IP burst + per-Source daily cap, §12) — moved ahead of WS work per reviewer finding; the public endpoint must not be reachable unthrottled at any point, including internal staging
5. WebSocket upgrade + in-memory per-pod session registry (§10) + per-pod HostID routing + `MessageSend`/`MessageDeliver` + single-writer reconnect enforcement (§3)
6. Idle-sweep + pending-timeout sweep goroutine (§10) + `webchat_session_ended` event
7. `bin-flow-manager`: `reference_type=webchat` + `webchat_message_send` action
8. First-inbound-message flow trigger wiring (webchat-manager → flow-manager `FlowV1ActiveflowCreate`)
9. Webhook events (`webchat_session_created/ended`, `webchat_message_created`)
10. Load test: concurrent WS connections per pod, idle-sweep correctness under restart, reconnect-race scenario (§3)

## 15. Open Questions

| Question | Recommendation | Decision owner |
|---|---|---|
| Rate limit shape for the public `POST /sessions` endpoint (per-IP? per-Source daily cap? both?) | Per-Source daily cap (customer controls their own exposure) + a global per-IP burst limit as a floor — needs a concrete number, not a placeholder. Per §12/§14 this now blocks Phase 1 entirely, not just "before public launch." | CEO/CTO, before Phase 1 lands in ANY reachable environment |
| Should message storage live in webchat-manager (this design) or delegate to bin-conversation-manager? | Keep self-contained for Phase 1 — conversation-manager's `Account.secret/token` schema assumes durable platform credentials, and retrofitting an anonymous/credential-less type is itself a design change to an existing service that should get its own review, not be smuggled into this one. Revisit in Phase 2 if a unified "all conversations in one place" dashboard becomes a real requirement. | CEO/CTO, pre-Phase-2 |
| Cross-pod session migration (pod restart during an active WS session) | Not in Phase 1 — visitor reconnects with the same token, a `pending→active` re-`Upgrade()` on a new pod picks up existing `Session`/`Message` rows since those are in MySQL, not in-memory. Only the *live socket* is lost, not the conversation history. This means every rolling deploy visibly interrupts live chats (visitor sees the connection drop, must have client-side reconnect logic) — an explicit product/ops tradeoff, not a hidden one. Acceptable for MVP; document as a known limitation, not silently absorbed. | CEO/CTO, sign-off on MVP limitation |
| Token TTL vs `session_idle_timeout` relationship | Token TTL should exceed idle_timeout (e.g. TTL = idle_timeout + 5m grace) so a reconnect attempt just past idle boundary still authenticates, even though the session itself will be swept to `ended` and a fresh `Issue()` is required. Additionally: when a live `active` session's token TTL is reached mid-conversation, the socket is NOT force-closed — the token is only re-checked at `Upgrade()`/reconnect time, never against an already-`active` live connection, so an ongoing chat is never interrupted purely by token expiry. | Engineering default, confirm no objection |
| Origin allow-list: exact-match only, or support wildcard subdomains (`*.example.com`) and `Origin: null` (sandboxed iframes)? | Ship exact-match list in Phase 1 (simplest, safest against subdomain-takeover) but allow a `*.` prefix convention in `AllowedOrigins` entries interpreted as suffix-match at validation time (`*.example.com` matches `app.example.com` but not `example.com` itself, unless both are listed). Treat `Origin: null` as always-rejected in Phase 1 — no customer request for it yet, and it materially weakens the compensating control; revisit if it blocks a real onboarding. | CEO/CTO or Product — confirm wildcard syntax is acceptable before it ships in the OpenAPI spec (breaking to change later) |
| Message content retention window for visitor-volunteered PII | Default retention aligned with whatever policy exists for `bin-conversation-manager` messages today (reuse the existing platform policy rather than inventing a webchat-specific one); if no explicit platform-wide retention policy exists yet, this surfaces that gap rather than being webchat-specific | CEO/CTO — may be a pre-existing gap this design just makes visible |
| Widget JS SDK / embeddable snippet delivery | Out of scope for this backend design doc entirely — needs its own square-talk/frontend design once this API is stable | Product, Phase 2 planning |
| **Real-time agent-facing delivery path for inbound visitor messages.** `webchat_message_created` (§8) is a customer-integration webhook (fires to the customer's own HTTPS endpoint per platform convention), not a push channel to a human agent's live browser session. Phase 1 has no analogue to bin-queue-manager/bin-agent-manager's real-time UI update path — a customer without their own webhook consumer can only see inbound messages by polling `GET /sessions/{id}/messages`. (Round 2 reviewer finding, High — flagged as a Phase-1 UX limitation, not silently absorbed.) | Acceptable as an explicit MVP limitation IF Phase 1 ships before any agent-handoff UI exists (per §2, human-agent handoff is already Phase 2). If Phase 1 and an agent dashboard ship together, this must be resolved first — recommend deferring only if the two truly ship separately. | CEO/CTO, confirm Phase 1/2 sequencing assumption holds |
| **`VisitorID` stability across token renewal.** §3 describes `VisitorID` as "stable per browser," but no mechanism is specified for how a renewed session (after token TTL expiry, mid-visit) receives the SAME `VisitorID` rather than a fresh random one. Default `idle_timeout` (1800s) means many real browsing sessions will span a renewal. (Round 2 reviewer finding, Medium — affects the wire format, cheaper to fix now than post-implementation.) | Add an optional `visitor_id` field to `POST /v1/webchat/sessions`; the widget's client-side storage (e.g. `localStorage`) persists the `VisitorID` returned in the first `Issue()` response and resends it on subsequent calls. Server trusts it as a hint only (still validates the Source/origin independently) — it is a continuity aid, not an auth credential. | Engineering default, confirm no objection — must land in the OpenAPI spec before Phase 1 implementation begins to avoid a breaking change later |
| **CORS story for the public `POST /v1/webchat/sessions` endpoint.** Origin validation (§10) is a server-side allow-list check, not a CORS header. A cross-origin fetch from an embedded widget also needs `Access-Control-Allow-Origin` set dynamically to the validated origin (a static `*` would defeat the allow-list; a static single value breaks customers with multiple `AllowedOrigins`). Not addressed anywhere in §7/§10 despite the otherwise careful origin-validation treatment. (Round 2 reviewer finding, Medium.) | `bin-api-manager`'s Gin router echoes back the validated `Origin` request header as `Access-Control-Allow-Origin` (never a wildcard) on successful origin-allow-list checks for these two public routes only; standard CORS preflight (`OPTIONS`) handling added alongside. | Engineering default, confirm no objection |
| **`Source.FlowID` mid-session mutation race.** If an admin updates `Source.FlowID` via `PUT /sources/{id}` between session issuance and the visitor's first message, §5's flow diagram implies the FlowID is read fresh at first-message time (not snapshotted at `Issue()`), so an admin's edit can retroactively change in-flight routing. (Round 2 reviewer finding, Low — documentation-completeness only.) | Confirm fresh-read-at-message-time is the intended behavior (it is the simplest to implement and matches "config changes take effect immediately" as the default assumption elsewhere in the platform); state this explicitly in §5 rather than leaving it implied by tense. | Engineering default, confirm no objection |

---

## Review Summary

**v1 → v2 (this revision).** v1 was sent to an independent design reviewer
(delegate_task, doc-only review, no codebase access) per the
`voipbin-backend-feature-design` skill Step 4. Verdict: **CHANGES REQUESTED**.
All Critical and High findings are addressed in v2:

| Finding | Severity | v2 resolution |
|---|---|---|
| Public unauthenticated ingress is a precedent-setting architecture decision buried in an Open Question | Critical | Promoted to new §0 "Headline Architecture Decision," called out as requiring explicit CEO/CTO sign-off before any reachable deployment |
| No in-memory session registry / mutex design specified (how does `MessageDeliver` find the live `net.Conn`?) | High | §10 now specifies `mapWebchatSession` + `muWebchatSession`, mirroring pipecat-manager's exact pattern |
| `webchat_sessions` missing `tm_delete` — breaks platform soft-delete invariant | High | Added to §3 struct and §4 schema; `tm_end` vs `tm_delete` distinction documented explicitly |
| Reconnect race / duplicate session (two tabs, same token, different pods) | High | §3 adds single-writer conditional-update + remote-close-old-connection protocol |
| Rate limiting sequenced last in Implementation Order while endpoint is public from day one | High | §12/§14 rate limiting moved to step 4, explicitly gates any reachable deployment, not just public GA |
| No data-retention/PII policy for visitor-volunteered message content | High | §12 adds explicit PII discussion; retention window promoted to §15 Open Question |
| Idle sweep could force-end a live-but-quiet WS (ambiguous `tm_last_activity` semantics) | Medium | §10 specifies ping/pong heartbeat bumps `tm_last_activity`; only genuinely dead connections age out |
| Pending-session 60s timeout mechanism undocumented | Medium | §10 clarifies it is the same sweep ticker, second query predicate; §4 adds the backing index |
| Origin allow-list exact-match may be unworkable for wildcard-subdomain customers / `Origin: null` iframes | Medium | Promoted from silent omission to explicit §15 Open Question with a recommended default |
| No `sender_id` on `Message` — can't audit who sent an agent-typed outbound message | Medium | Added `SenderID`/`ActiveflowID` fields to `Message` (§3, §4) |
| webchat-manager is a hybrid service archetype (public HTTP+WS AND RabbitMQ RPC) not matching hook-manager or a standard Class A manager | Medium | Called out explicitly in new §0 rather than left implicit |
| Flow deletion/FlowID invalidation after a Source references it | Medium | Not fixed in v2 — accepted as a Phase-2 hardening item; the first-inbound-message trigger will fail closed (activeflow creation errors surface as a failed session start, logged, not silently swallowed), which is the same failure mode conversation-manager already has for an invalid `MessageFlowID` |
| `bin-webhook-manager` "no change" claim not verified against a possible event-type registry/whitelist | Low | Not independently verified in this revision — flagged for confirmation during Step 1 implementation recon against the actual webhook-manager source, not assumed safe |

Deferred to Phase 2 without a v2 design change (explicitly, not silently):
VisitorID cross-session stability for analytics/CRM correlation (Low,
reviewer finding) and the `StatusNone=""` build-time discipline reminder (Low,
no code exists yet to audit).

This document is now ready for a fresh review round before implementation
begins, per the skill's re-review requirement after a material post-approval
edit.

**v2 → v3 (this revision).** v2 was sent to a Round 2 independent reviewer,
who additionally verified the Round 1 fixes against REAL source code (not
just re-reading the prose) — confirmed the `bin-pipecat-manager` per-pod
session pattern citation, the `bin-flow-manager` `ReferenceType` enum values,
and the `bin-hook-manager` "no listenhandler" rationale all hold against the
actual codebase. Verdict: **APPROVE WITH COMMENTS**. No Critical or High
regression was introduced by the v2 edits themselves. Remaining findings
(none blocking Phase 1 approval, all addressed in v3):

| Finding | Severity | v3 resolution |
|---|---|---|
| Remote-close RPC in the reconnect protocol has no stated failure/timeout behavior | Medium | §3 adds explicit "timeout = treat as already-gone, proceed" rule — a dead old pod must never block a legitimate reconnect |
| No real-time agent-facing delivery path for inbound messages (only customer webhook + poll) | High | Added to §15 Open Questions with an explicit Phase 1/2 sequencing dependency — acceptable ONLY if Phase 1 ships before any agent-handoff UI, which matches §2's existing Phase 2 placement of human-agent handoff |
| `VisitorID` stability mechanism across token renewal unspecified | Medium | Added to §15 with a recommended client-supplied-hint mechanism, flagged as needing to land in the OpenAPI spec before implementation (wire-format decision, expensive to change later) |
| No CORS/`Access-Control-Allow-Origin` story for the public POST endpoint | Medium | Added to §15 with a recommended dynamic-echo-validated-origin mechanism (never a static wildcard) |
| `Source.FlowID` mid-session mutation race (fresh-read vs snapshot) | Low | Added to §15; recommend confirming fresh-read-at-message-time as the explicit intended behavior |
| `SenderID`/`ActiveflowID` `omitempty` on `uuid.UUID` doesn't suppress the zero UUID in JSON (pre-existing platform-wide quirk, not unique to this design — confirmed via grep against `bin-pipecat-manager/models/message/main.go`) | Low | Documentation nit only, not fixed — matches existing platform convention, not a design defect to deviate on unilaterally |
| §0's "hybrid archetype, doesn't map cleanly onto either precedent" slightly overstates novelty vs. §10's actual composition of two existing patterns | Low | Wording nit, left as-is — defensible framing for documentation purposes (no other service combines both halves in one binary today) |

This document has cleared two independent review rounds (CHANGES REQUESTED →
APPROVE WITH COMMENTS) with all Critical/High items resolved and verified
against real source code where a precedent was cited. Ready for
implementation, pending the CEO/CTO sign-offs flagged in §0 and §15 (public
ingress precedent, rate-limit shape, Phase 1/2 sequencing for agent delivery,
message retention window, origin wildcard policy).
