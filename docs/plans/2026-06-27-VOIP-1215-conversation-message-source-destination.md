# conversation_message: persist source / destination (per-message immutable fact)

- Issue: VOIP-1215
- Parent context: VOIP-1204 (CRM interaction timeline), VOIP-1208 (projection handlers, method B), VOIP-1214 (contract parity audit)
- Class: schema migration + model/handler change + webhook contract + OpenAPI/RST sync
- Service: bin-conversation-manager (+ bin-dbscheme-manager migration, bin-common-handler RPC signature)
- Date: 2026-06-27 (rev 2: persist-to-DB after pchero review; supersedes the
  transient/read-time draft)

## 1. Problem

The `conversation_message` record (and its webhook event) carries the message
body but NOT the remote party. The endpoint pair lives only on the parent
`Conversation` as `Self` / `Peer`. Consumers that need "who was on the other end"
per message (CRM interaction timeline VOIP-1208, timeline-manager, external
customer webhooks) cannot get it from the message/event alone.

The RST docs (`conversation_overview.rst`) already promise a
`participant{type,target}` field that the code never emits (VOIP-1214 seed): the
published contract is currently a lie.

## 2. Decision (revised): persist source/destination as columns

Store `source` and `destination` on `conversation_messages` as durable columns
(typed `commonaddress.Address`, JSON), set once at message creation, and emit
them on the webhook event.

This REVERSES the transient/read-time-derive approach of the first draft. The
reversal is correct because the premise of "derived value, do not store" was
wrong:

- **source/destination are an immutable first-class FACT, not a derived value.**
  They are what the message actually carried (the SMS's `m.Source` /
  `target.Destination`, the WhatsApp webhook's `from` / `display_phone_number`,
  etc.). Per the project principle (immutable facts ARE stored; only
  lookup-derived / mutable values are computed at read time), this belongs in a
  column.
- **The parent's `Self`/`Peer` are immutable** (code-verified §5), so even when
  source/destination are populated from `Conversation.Self/Peer + direction`, the
  stored value can never go stale.
- **The data is already in hand at create time** (§5), so persisting it adds no
  lookup; the earlier "re-fetch conversation at publish time" helper is deleted.

### 2.1 History note (mandatory in PR description)

`conversation_messages.source (json)` previously existed (added 2022-06,
`122b2ba1b2b0`) and was deliberately DROPPED 2025-04 (`7a27decc13da`), which in
the same migration moved the endpoint pair up to the conversation as
`self`/`peer`. Rationale at the time (pchero): a per-message copy of data already
on the conversation was redundant. That judgment was correct then. It changed
because a NEW consumer (the CRM interaction timeline, VOIP-1204) now requires a
per-message immutable record of the endpoints. So this is a requirements-driven
RE-introduction with a different purpose (immutable per-message fact for the CRM
projection), not a flip-flop. State this explicitly in the PR so the migration
history reads coherently.

## 3. Coordinate semantics: absolute (source/destination), not relative (self/peer)

The message columns use the absolute "who-sent / who-received" axis, identical to
`call-manager` and `message-manager` webhooks:

```
source      = the sending party    (always "from")
destination = the receiving party  (always "to")
```

This is direction-independent in MEANING: `source` is always the origin and
`destination` is always the target, regardless of inbound/outbound. The parent
Conversation uses the RELATIVE axis (`Self` = us, `Peer` = remote), which needs
`direction` to know who is the sender. The single fill rule that converts
relative -> absolute:

```
direction = outbound  =>  source = Self,  destination = Peer
direction = inbound    =>  source = Peer,  destination = Self
```

(Conversation `Message.Direction` values are `incoming` / `outgoing`; treat
`outgoing` as outbound, `incoming` as inbound. The enum rename to
inbound/outbound is VOIP-1214, out of scope here.)

This is THE authoritative fill rule for ALL creation paths (§6). Every row's
`source`/`destination` means the same thing, so the append-only column never
carries mixed semantics.

### 3.0a Consumer-side rule: how a reader recovers the remote party

The event carries the absolute pair (`source`/`destination`) plus `direction`,
NOT a `peer`/`self` field. A consumer that needs the REMOTE party (e.g. the CRM
projection VOIP-1208 keying on the customer's endpoint) recovers it by the
inverse of §3:

```
direction = inbound (incoming)   =>  remote (peer) = source       (local = destination)
direction = outbound (outgoing)  =>  remote (peer) = destination   (local = source)
```

This inverse rule is part of the published contract and MUST be documented in the
RST/OpenAPI so VOIP-1208 (and any external consumer) extracts the peer
deterministically. VOIP-1208 itself is out of scope for THIS ticket, so the only
place to record the contract is here + the docs.

### 3.1 Why `commonaddress.Address` (value, JSON)

- Structured, not a bare string: the CRM match key is `(peer_type, peer_target)`.
  `commonaddress.Address` carries `{type, target, ...}`, so the channel
  discriminator survives (a bare phone string could not distinguish `tel` vs
  `line`). Required for the `contact_addresses` IN-match.
- Value type (not pointer): mirrors call-manager's `Source`/`Destination`; both
  endpoints always structurally exist on a conversation message. JSON storage
  matches the existing `conversation.Self`/`Peer` columns (`db:"self,json"`).
- Pre-existing LINE caveat (§7): LINE inbound leaves `Self.Target == ""`. The
  Address struct is still stored; only the Self-side `target` is blank. The CRM
  reads the REMOTE party (Peer side), which is populated on every channel, so
  this does not affect the CRM use. Documented, not introduced here.

## 4. Schema migration (bin-dbscheme-manager)

Add two JSON columns to `conversation_messages`. Generate via
`alembic revision` (never hand-pick the revision id); current single head is
`ac5d4e18060c`, so the new migration chains onto it.

```sql
ALTER TABLE conversation_messages
  ADD COLUMN source      JSON AFTER reference_id,
  ADD COLUMN destination JSON AFTER source;
```

- JSON (nullable), matching `conversation_conversations.self/peer` convention.
- No backfill: from cutover, new messages carry the endpoints; historical rows
  stay NULL (consistent with VOIP-1204 M2 "no retroactive backfill"). The CRM
  projection (VOIP-1208) only consumes go-forward events, so NULL history is
  acceptable.
- downgrade drops both columns.
- Update the in-repo test schema `scripts/database_scripts_test/table_conversation_messages.sql` to match (used by tests).

## 5. Code-verified facts the design relies on

- **Self/Peer are immutable after create.** The external PUT whitelist
  (`listenhandler/v1_conversations.go`) allows only
  `OwnerType/OwnerID/Name/Detail/AccountID`; `self`/`peer` are NOT updatable.
  No production path writes `FieldSelf`/`FieldPeer` after `ConversationCreate`
  (`conversationhandler/db.go`); the only writer is a test. So a value copied
  from Self/Peer at message-create time can never go stale.
- **Data is in hand at every create path** (§6 table).
- **alembic single head** = `ac5d4e18060c`.
- **RPC blast radius is closed to two callers, both inside conversation-manager**
  (`linehandler/hook.go:172`, `whatsapphandler/hook.go:155`). No external service
  calls `ConversationV1MessageCreate` (ai-manager uses a different RPC,
  `ConversationV1MessageSend`). So extending the create signature touches only
  bin-common-handler (RPC + mock) and bin-conversation-manager.

## 6. Fill rule per creation path (the MAJOR-1 closure)

source/destination are derived from `(cv.Self, cv.Peer, direction)` by the §3
rule, in a single SHARED helper, and passed INTO `Create` by the caller. They are
NOT derived inside `Create` (Create receives only `conversationID`, not the
conversation, so deriving inside would force a redundant `ConversationGet` — the
exact re-fetch we are avoiding). Every Create call site already holds `cv`
(code-verified below), so the caller derives and passes the pair; Create just
stores it.

Shared helper (one definition, used by every call site so the rule cannot be
applied inconsistently):

```go
// deriveEndpoints maps the conversation's relative Self/Peer to the message's
// absolute source/destination by direction. Single authority for the §3 rule.
func deriveEndpoints(cv *conversation.Conversation, dir message.Direction) (source, destination commonaddress.Address) {
    switch dir {
    case message.DirectionOutgoing: // outbound: we are the sender
        return cv.Self, cv.Peer
    case message.DirectionIncoming: // inbound: remote is the sender
        return cv.Peer, cv.Self
    default: // DirectionNond ("") / unknown: do NOT guess. Leave zero + log.
        return commonaddress.Address{}, commonaddress.Address{}
    }
}
```

`DirectionNond` ("") guard: an unknown direction must NOT be silently coerced to
outbound. Return zero endpoints and log; the row is still created (direction
itself is the upstream defect, not a reason to drop the message), but we never
write a wrong-meaning source/destination into the append-only column.

Every Create call site holds `cv` (verified):

| path | file | direction source | cv in hand |
|------|------|------------------|------------|
| SMS inbound  | `conversationhandler/message.go` MessageEventReceived | `incoming` (literal) | yes (`GetOrCreateBySelfAndPeer`) |
| SMS outbound (echo path) | `conversationhandler/message.go` MessageEventSent | `outgoing` (literal) | yes (`GetOrCreateBySelfAndPeer`) |
| SMS outbound | `messagehandler/send.go` sendSMS | `outgoing` (literal) | yes (`Send(cv,...)`) |
| WhatsApp outbound | `messagehandler/send.go` sendWhatsApp | `outgoing` (literal) | yes (`Send(cv,...)`) |
| LINE outbound | `messagehandler/send.go` sendLine | `outgoing` (literal) | yes (`Send(cv,...)`) |
| Email outbound | `conversationhandler/email.go` | `outgoing` (literal) | yes |
| WhatsApp/LINE inbound (RPC) | `whatsapphandler/hook.go`, `linehandler/hook.go` -> `ConversationV1MessageCreate` | `incoming` (literal) | yes (hook holds cv before the RPC) |
| API send (RPC) | `listenhandler/v1_messages.go` -> `ConversationV1MessageCreate` | `req.Direction` (caller-provided, variable) | NO — handler gets only `req.ConversationID` |

Path-classification corrections (vs the prior draft, which mis-grouped these):

- `conversationhandler/message.go` has BOTH directions: `MessageEventReceived`
  (inbound) AND `MessageEventSent` (outbound echo), each holding `cv`.
- `send.go` `Send(cv, ...)` already receives `cv` and is the OUTBOUND path for
  SMS/WhatsApp/LINE (all `DirectionOutgoing`). It calls `Create` with `cv.ID`
  today; it will additionally derive endpoints from the `cv` it already has.

**RPC path mechanics (corrected — the prior draft was wrong here).** The
`ConversationV1MessageCreate` RPC handler (`listenhandler/v1_messages.go`
`processV1MessagesCreatePost`) does NOT load the conversation; it forwards
`req.ConversationID` straight into `Create`. So the handler does NOT hold `cv` and
deriving endpoints handler-side would require the very `ConversationGet` re-fetch
we are avoiding. Therefore endpoints for the RPC path are derived by the RPC
CALLERS, who DO hold `cv`:

- `whatsapphandler/hook.go` and `linehandler/hook.go` already have `cv` before
  calling the RPC. They call `deriveEndpoints(cv, dir)` and pass `source` /
  `destination` as fields on the RPC request DTO.
- The RPC handler then just forwards those two fields into `Create` (store only).
  No conversation load on the handler side.

This keeps the rule in ONE helper (`deriveEndpoints`), always invoked where `cv`
is in hand (the hook for the RPC path, the call site for the direct path), and
honors the no-re-fetch principle on every path. The RPC request DTO therefore
gains `source` / `destination` fields (§7).

**RPC caller census (verified).** `ConversationV1MessageCreate` has exactly TWO
callers in the monorepo: `whatsapphandler/hook.go` and `linehandler/hook.go`
(both inside conversation-manager, both holding `cv`). No external service and no
other in-repo path calls it; `processV1MessagesCreatePost` is effectively the
RPC entry point for those two hooks. So there is NO "API-send caller without cv"
in practice. If a future external caller of this RPC appears WITHOUT cv, the
DTO's `source`/`destination` simply arrive zero and the handler stores zero (same
graceful-degradation as the unknown-direction guard); that is acceptable and does
not require the handler to load the conversation. Document this so a future
caller knows it owns the derive step.

## 7. Signature change: use a params struct (MAJOR-2 closure)

`messageHandler.Create` already takes ~11 positional args and the RPC mirrors it;
adding two more `commonaddress.Address` positionals invites mis-positioned
arguments (there is already a `subject` "" positional drift in
`listenhandler/v1_messages.go`). Convert to a params struct:

```go
type MessageCreateArgs struct {
    ID             uuid.UUID
    CustomerID     uuid.UUID
    ConversationID uuid.UUID
    Direction      message.Direction
    Status         message.Status
    ReferenceType  message.ReferenceType
    ReferenceID    uuid.UUID
    TransactionID  string
    Text           string
    Subject        string
    Medias         []media.Media
    // new — caller-derived absolute endpoints (via deriveEndpoints, §6).
    // These are INPUTS to Create (the caller fills them); Create does not
    // re-derive (it has no cv). Single source of the values = the shared helper.
    Source         commonaddress.Address
    Destination    commonaddress.Address
}
```

Apply the same shape to the `ConversationV1MessageCreate` RPC request DTO
(`requesthandler` + `models/request`), regenerate the mock. If a full struct
refactor of the RPC is deemed too broad for this ticket, the minimum is to add
the two fields to the RPC request DTO (not as positional RPC args) and keep the
internal handler on a struct. Decide and state which in the PR; do NOT add bare
positionals.

## 8. Model + webhook

- `models/message/message.go`: add
  `Source commonaddress.Address \`json:"source,omitempty" db:"source,json"\``
  and the same for `Destination` (NOW persisted — real `db` tag, unlike the
  scrapped transient approach).
- `models/message/webhook.go`: add `Source`/`Destination` to `WebhookMessage`
  and copy them in `ConvertWebhookMessage()`.
- `MessageCreate` dbhandler INSERT and `MessageGet` SELECT pick up the two new
  columns automatically: the dbhandler is reflection-driven (PrepareFields /
  GetDBFields / ScanRow read the `db` struct tags), so adding `db:"source,json"`
  / `db:"destination,json"` to the model is sufficient for INSERT, SELECT, and
  scan. No manual column list to edit. A `Field` enum constant
  (`FieldSource`/`FieldDestination`) is only needed if source/destination are
  used in a filter or `MessageUpdate` (not required for this ticket; they are
  set once at create); add them only if a filter use appears.

Event payload (example, inbound SMS):

```json
{
  "type": "conversation_message_created",
  "data": {
    "id": "8f1d2c3a-...-aabb",
    "customer_id": "1111...-3333",
    "conversation_id": "conv-4444...-6666",
    "direction": "incoming",
    "status": "done",
    "reference_type": "message",
    "reference_id": "msg-7777...-9999",
    "source":      { "type": "tel", "target": "+155****0000" },
    "destination": { "type": "tel", "target": "+155****9999" },
    "text": "Thanks for the update!",
    "medias": [],
    "tm_create": "2026-06-27T10:30:00.000000"
  }
}
```

## 9. Contract parity (same PR, per root CLAUDE.md + VOIP-1214)

1. OpenAPI: add `source`/`destination` to the conversation message schema in
   `bin-openapi-manager/openapi/openapi.yaml`; regenerate.
2. RST: fix the conversation message docs to the REAL shape.
   - `conversation_overview.rst` example is wrong on five axes
     (`conversation_message_received`, nested `data.message.participant`,
     `channel`, `direction:inbound`); replace with the actual flat
     `conversation_message_created` shape incl. the new source/destination.
   - `conversation_struct_message.rst` lists the message struct and currently
     omits source/destination; add both (with the §3.0a consumer-side
     direction->peer note so readers know how to read the pair).
   Rebuild HTML, force-add `build/`.
3. Event NAME stays `conversation_message_created` (fix the doc, do not adopt the
   doc's erroneous `_received`).

## 10. Backward compatibility

- Schema: additive columns, nullable, no backfill -> safe; old rows read NULL.
- Webhook: additive fields on an existing event -> non-breaking for consumers
  that ignore unknown fields. `omitempty` on a value struct does NOT omit a zero
  Address (serializes as `{}`); harmless and identical to call-manager. Do not
  claim it keeps zero payloads lean.
- RPC: signature/DTO change is internal (two in-repo callers + mock); no external
  consumer.

## 11. Scope

In: source/destination columns + migration, model + webhook + dbhandler,
MessageCreate params-struct + RPC DTO, fill rule in Create, OpenAPI + RST sync,
tests.

Out: direction enum rename (VOIP-1214). The broader parity sweep (VOIP-1214). CRM
projection consumption (VOIP-1208). LINE Self.Target backfill (pre-existing; only
document).

## 12. Test plan

- Unit (helper): `deriveEndpoints` — outgoing maps (Self->source, Peer->dest);
  incoming maps (Peer->source, Self->dest); `DirectionNond`/unknown returns zero
  endpoints (NOT silently outbound) and is logged.
- Unit (call sites): each Create call site passes the helper result into the
  params struct (assert the published/persisted Message carries the §3-correct
  pair for both an inbound and an outbound case).
- Unit: `ConvertWebhookMessage` copies Source/Destination.
- DB round-trip: create with source/destination -> MessageGet returns identical
  Address values (proves the columns are actually persisted and scanned). Force a
  cache miss so the DB scan path is exercised, not just the cache.
- Handler: Create / UpdateStatus / Delete publish events carrying the endpoints
  (mock notifyHandler).
- Migration: apply on a local throwaway DB; verify columns exist; downgrade drops
  them. (NEVER run against staging/prod.)
- Full verification workflow in bin-conversation-manager AND bin-common-handler
  (RPC signature change): go mod tidy/vendor/generate/test + golangci-lint.
- bin-dbscheme-manager: build-pipeline round trip (MariaDB build -> mysqldump ->
  MySQL 8.0 import) for the new migration.
