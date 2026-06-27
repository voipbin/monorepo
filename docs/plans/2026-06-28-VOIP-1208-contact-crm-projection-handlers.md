# VOIP-1208: contact-manager CRM projection handlers

- Issue: VOIP-1208
- Parent: VOIP-1204 (CRM interaction timeline)
- Class: event subscription + DB model + schema migration + handler
- Service: bin-contact-manager (+ bin-dbscheme-manager migration)
- Date: 2026-06-28

## 1. Goal

bin-contact-manager subscribes to two channel-native creation events and appends one
row per event (per fan-out recipient for SMS) to `contact_interactions`. This is
the write side of the CRM timeline; identity resolution (peer → contact_id) is
read-time (VOIP-1209), not projection-time.

## 2. Canonical reference decision (option 가, pchero confirmed)

Subscribe to exactly TWO event sources:

| subscribed event            | reference_type      | reference_id                | queue constant              |
|-----------------------------|---------------------|-----------------------------|-----------------------------|
| `call_created`              | `"call"`            | `call.WebhookMessage.ID`    | `QueueNameCallEvent`        |
| `conversation_message_created` | `"conversation_message"` | `convmsg.WebhookMessage.ID` | `QueueNameConversationEvent` |

**NOT subscribed (and why):**
- `message_created` (message-manager SMS): conversation-manager already subscribes
  to `message_created` and re-emits `conversation_message_created` for every SMS
  thread. Subscribing to both would double-count every inbound SMS. The
  conversation-manager is the canonical unifier for all text channels
  (SMS/LINE/WhatsApp/email). Code confirmed: `grep QueueNameMessageEvent
  bin-conversation-manager/cmd` shows conversation-manager consuming it.
- `aicall_status_initializing`: out of scope until VOIP-1213 wraps web/task-direct
  AIcall into the conversation layer (currently Backlog). Clean coverage gap;
  documented explicitly here so it is not a silent miss.

**Coverage by channel after this PR:**
- Inbound/outbound voice call: ✓ (call_created)
- Inbound/outbound SMS: ✓ (conversation_message_created — SMS flows through
  conversation-manager as reference_type='message' or 'line'/'whatsapp' depending
  on channel)
- LINE / WhatsApp / outbound email: ✓ (conversation_message_created)
- Web/task-direct AIcall: ✗ (VOIP-1213 dependency, not yet wrapped)

## 3. Schema migration (bin-dbscheme-manager, decision A)

Add `local_type` and `local_target` to `contact_interactions` (decision A:
scoped into VOIP-1208 because projection fills them; same work unit).

Current head: `bbcf80d332eb`. New migration chains onto it.

```sql
ALTER TABLE contact_interactions
    ADD COLUMN local_type   VARCHAR(255) NOT NULL DEFAULT '' AFTER peer_target,
    ADD COLUMN local_target VARCHAR(255) NOT NULL DEFAULT '' AFTER local_type;
```

Rationale: local endpoint (our number / LINE official account) is a first-class
immutable fact the event carries. With no-backfill policy, omitting the column
now = permanent data loss. Mirrors conversation-manager (VOIP-1215) which stores
both source/destination. Use case: per-number/per-channel inbound attribution.
- `local_type`/`local_target` are NOT in the idempotency unique
  (same event always has same local; adding local to the unique would widen it
  unnecessarily without dedup benefit).
- No index on local columns (not used as match key or cursor; attribution only).
- downgrade drops both columns.

**contact_interactions final column set (migration complete):**
```
id             BINARY(16)    NOT NULL  PK
customer_id    BINARY(16)    NOT NULL
direction      VARCHAR(255)  NOT NULL DEFAULT ''
peer_type      VARCHAR(255)  NOT NULL DEFAULT ''
peer_target    VARCHAR(255)  NOT NULL DEFAULT ''  -- normalized, match key
local_type     VARCHAR(255)  NOT NULL DEFAULT ''  -- NEW
local_target   VARCHAR(255)  NOT NULL DEFAULT ''  -- NEW
reference_type VARCHAR(255)  NOT NULL DEFAULT ''
reference_id   BINARY(16)    NOT NULL
tm_interaction DATETIME(6)                         -- origin event time (display sort)
tm_create      DATETIME(6)                         -- projection insert time (cursor)
UNIQUE(reference_type, reference_id, peer_target)  -- idempotency
INDEX(customer_id, peer_type, peer_target)         -- read-time peer match
INDEX(customer_id, tm_create)                      -- pagination cursor
```

## 4. peer / local extraction rule

Absolute coords (source/destination from event) → relative coords (peer/local):

```
incoming: peer = source,      local = destination
outgoing: peer = destination, local = source
```

Both call and conversation_message use `"incoming"`/`"outgoing"` direction values
(verified in code: `call.DirectionIncoming = "incoming"`,
`convmsg.DirectionIncoming = "incoming"`).

peer_type  = peer.Type   (commonaddress.Address.Type, e.g. "tel"/"line"/"whatsapp")
peer_target = NormalizeTarget(peer.Type, peer.Target)  ← MUST use this, bit-identical to contact_addresses.target
local_type  = local.Type
local_target = NormalizeTarget(local.Type, local.Target) — error on unknown type: log + store raw target, do not drop the row

**Unknown direction guard:** if Direction == "" (DirectionNond/empty), log a warning
and store peer_type/peer_target = "", local_type/local_target = "" (zero values).
Do NOT drop the row; the reference fact is still valid. This is a graceful fallback
for any future malformed event.

## 5. New code to create (nothing pre-exists)

### 5.1 models/interaction/interaction.go (new package)

```go
package interaction

import (
    "github.com/gofrs/uuid"
    commonaddress "monorepo/bin-common-handler/models/address"
    "time"
)

// Interaction is an immutable append-only fact in the CRM interaction timeline.
// It records that a channel-level event (call or conversation message) touched a
// particular remote endpoint (peer). Identity resolution (peer → contact) is done
// at read time.
type Interaction struct {
    ID         uuid.UUID `json:"id"          db:"id,uuid"`
    CustomerID uuid.UUID `json:"customer_id"  db:"customer_id,uuid"`

    Direction string `json:"direction"  db:"direction"`  // "incoming" / "outgoing"

    // Remote endpoint (the peer's address — match key for contact resolution).
    // peer_target is stored normalized via commonaddress.NormalizeTarget.
    PeerType   string `json:"peer_type"   db:"peer_type"`
    PeerTarget string `json:"peer_target" db:"peer_target"`

    // Our local endpoint (for attribution: which number/account received/sent).
    LocalType   string `json:"local_type"   db:"local_type"`
    LocalTarget string `json:"local_target" db:"local_target"`

    // Origin channel record. State and body are fetched at read time.
    ReferenceType string    `json:"reference_type" db:"reference_type"`
    ReferenceID   uuid.UUID `json:"reference_id"   db:"reference_id,uuid"`

    TMInteraction *time.Time `json:"tm_interaction" db:"tm_interaction"` // origin event time
    TMCreate      *time.Time `json:"tm_create"      db:"tm_create"`      // projection insert time
}
```

### 5.2 pkg/dbhandler/main.go — add InteractionCreate

```go
InteractionCreate(ctx context.Context, i *interaction.Interaction) error
```

Implementation: squirrel INSERT, idempotency duplicate is a no-op (not an error).
On `errno 1062` (duplicate key), return nil (idempotent).

### 5.3 pkg/dbhandler/interaction.go (new file)

Implements `InteractionCreate`. Uses squirrel, no cache (append-only facts).

### 5.4 pkg/contacthandler/main.go — add event handlers

```go
EventCallCreated(ctx context.Context, m *callwebhook.WebhookMessage) error
EventConversationMessageCreated(ctx context.Context, m *convmsgwebhook.WebhookMessage) error
```

Implementation pattern (both):
1. Extract peer/local from source/destination + direction (§4 rule).
2. NormalizeTarget for peer_target (and local_target). On error: log, use raw target.
3. Build Interaction{} with UUIDCreate() for ID, TimeNow() for TMCreate.
   TMInteraction = event.TMCreate. **nil guard required**: if event.TMCreate == nil
   (can happen for call events due to omitempty), set TMInteraction = nil (store as
   NULL — the column is nullable). Do NOT dereference a nil pointer.
4. Call db.InteractionCreate(ctx, &i). On duplicate: silently return nil.

### 5.5 pkg/subscribehandler/ — two new handler files

**subscribehandler/callmanager.go:**
```go
publisherCallManager = string(commonoutline.ServiceNameCallManager)

func (h *subscribeHandler) processEventCallManagerCallCreated(ctx context.Context, m *sock.Event) error {
    var payload callwebhook.WebhookMessage
    if err := json.Unmarshal(m.Data, &payload); err != nil {
        return err
    }
    return h.contactHandler.EventCallCreated(ctx, &payload)
}
```

**subscribehandler/conversationmanager.go:**
```go
publisherConversationManager = string(commonoutline.ServiceNameConversationManager)

func (h *subscribeHandler) processEventConversationManagerMessageCreated(ctx context.Context, m *sock.Event) error {
    var payload convmsgwebhook.WebhookMessage
    if err := json.Unmarshal(m.Data, &payload); err != nil {
        return err
    }
    return h.contactHandler.EventConversationMessageCreated(ctx, &payload)
}
```

**subscribehandler/main.go — add cases to switch:**
```go
case m.Publisher == publisherCallManager && m.Type == call.EventTypeCallCreated:
    err = h.processEventCallManagerCallCreated(ctx, m)
case m.Publisher == publisherConversationManager && m.Type == string(convmsg.EventTypeMessageCreated):
    err = h.processEventConversationManagerMessageCreated(ctx, m)
```

### 5.6 cmd/contact-manager/main.go — add queues to subscribeTargets

```go
subscribeTargets := []string{
    string(commonoutline.QueueNameCustomerEvent),
    string(commonoutline.QueueNameCallEvent),           // NEW: "bin-manager.call-manager.event"
    string(commonoutline.QueueNameConversationEvent),   // NEW: "bin-manager.conversation-manager.event"
}
```

### 5.7 scripts/database_scripts_test/contacts.sql — add contact_interactions DDL

Match the final column set (§3) for unit test DB.

## 6. Idempotency

`UNIQUE(reference_type, reference_id, peer_target)` is the DB-level dedup guard.
- `reference_id NOT NULL` (code-verified: every projected channel has an origin id).
- `peer_target` normalized → bit-identical to contact_addresses.target.
- On 1062 (duplicate): return nil, do NOT error. This is the at-least-once delivery
  absorber.
- `local_type/local_target` NOT in the unique (same event = same local, no
  dedup benefit; adding would widen the unique unnecessarily).

## 7. Imports needed (new in bin-contact-manager)

- `callwebhook "monorepo/bin-call-manager/models/call"` (WebhookMessage is in the `call` package)
- `convmsg "monorepo/bin-conversation-manager/models/message"` (WebhookMessage + EventTypeMessageCreated)
- `call "monorepo/bin-call-manager/models/call"` (EventTypeCallCreated, DirectionIncoming, DirectionOutgoing)
- `commonaddress "monorepo/bin-common-handler/models/address"` (NormalizeTarget)
- `commonoutline "monorepo/bin-common-handler/models/outline"` (QueueNameCallEvent, QueueNameConversationEvent, ServiceNameCallManager, ServiceNameConversationManager)

Note: bin-contact-manager.go.mod already depends on bin-common-handler. Need to
verify bin-call-manager and bin-conversation-manager are in go.mod; if not, add.

## 8. Test plan

- `InteractionCreate`: normal insert, duplicate key → no-op (verify 1062 absorbed).
- `EventCallCreated`: incoming call → peer=source, local=destination; outgoing →
  peer=destination, local=source; unknown direction → zero peer/local, row still
  created. nil TMCreate → TMInteraction stored as nil (no panic).
- `EventConversationMessageCreated`: incoming/outgoing cases, NormalizeTarget applied,
  reference_type="conversation_message", reference_id=message ID.
- subscribehandler: publisher+type routing directs to correct processEvent function.
  json.Unmarshal error returns error (not silently ignored).
- NormalizeTarget error path: log + use raw target, row created (not dropped).

## 9. Implementation checklist (mandatory steps)

- [ ] `alembic revision` to generate migration (never hand-pick ID), fill upgrade/downgrade
- [ ] `bin-contact-manager/scripts/database_scripts_test/contacts.sql` — add contact_interactions DDL with all columns including local_type/local_target
- [ ] `go generate ./...` in bin-contact-manager after modifying DBHandler and ContactHandler interfaces (regenerates mock_main.go in both dbhandler and contacthandler packages)
- [ ] Full verification workflow: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`

## 10. Out of scope

- aicall_status_initializing subscription (VOIP-1213 dependency).
- Identity resolution peer → contact_id (VOIP-1209 read API).
- message_created direct subscription (deliberately excluded, see §2).
- Backfill of historical interactions (VOIP-1204 §7.2 decision: no backfill).
- contact_interactions DELETE / GET / LIST (append-only in v1; read-time query is
  VOIP-1209's job).
