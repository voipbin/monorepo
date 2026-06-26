# Email Conversation Integration (Outbound-only) — Design

- Status: Draft v2
- Date: 2026-06-26
- Owner: CPO
- Service: bin-conversation-manager (consumer), bin-email-manager (event producer)

## 1. Problem Statement

`bin-conversation-manager` is the unified conversation-thread store for the
platform's messaging channels. Today it manages exactly three channel types:
`message` (SMS/MMS), `line`, and `whatsapp`. Email lives entirely in
`bin-email-manager` and is never surfaced as a conversation.

As a result, an outbound email sent to a customer does not appear in that
customer's conversation history. A contact-center operator (or a customer
querying the conversation API) cannot see "what email did we send this person"
alongside the SMS/LINE/WhatsApp thread with the same person. The unified
conversation model is incomplete for the email channel.

This design adds **email as a conversation channel, outbound-only**: every
email the platform sends is recorded as an `outgoing` message inside a
conversation thread, using the same event-subscription pattern by which SMS
is already absorbed from `bin-message-manager`.

## 2. Scope

### In scope (Phase 1)

- New conversation type `email` and message reference type `email`.
- `bin-conversation-manager` subscribes to `bin-manager.email-manager.event`
  and, on `email_created`, creates/locates a conversation and appends an
  `outgoing` message.
- Email-specific fields (subject) carried on the conversation message.
- Conversation/message lifecycle events published as for every other channel.

### Explicitly out of scope

- **Inbound email.** `bin-email-manager` has no inbound-receive capability
  today (its provider webhooks only carry delivery-status updates for emails
  it sent, never a new customer-originated message). Email is therefore
  **outbound-only** in the conversation model. No `incoming` email messages,
  no `POST /v1/hooks` email path, no `ExecuteMode` flow trigger for email.
  Revisit only if/when email-manager gains an inbound-parse feature.
- **Chat.** `bin-talk-manager` is a separate, internally-scoped chat service
  with a different domain model. It is NOT part of the conversation model and
  is explicitly excluded from this and any related work.
- **Email delivery-status reflection.** SendGrid status transitions
  (`delivered`/`open`/`click`/`bounce` ...) are tracked in email-manager and
  are NOT mirrored onto the conversation message in Phase 1 (see Open
  Questions Q3).

### Why outbound-only is the right Phase-1 cut

Because email has no inbound path, there is no two-way "conversation" to
orchestrate: no agent-vs-flow ownership dispatch, no activeflow trigger, no
signature-verified webhook. The integration collapses to a single concern:
**mirror each sent email into the conversation thread.** This is a much
smaller change than the SMS/LINE/WhatsApp bidirectional integrations and
reuses the existing SMS-absorption machinery almost verbatim.

## 3. Domain Model

### 3.1 New enum values (additive only)

`models/conversation/conversation.go`:

```go
const (
    TypeNone     Type = ""
    TypeMessage  Type = "message" // sms, mms
    TypeLine     Type = "line"
    TypeWhatsApp Type = "whatsapp"
    TypeEmail    Type = "email"   // NEW: outbound email
)
```

`models/message/message.go`:

```go
const (
    ReferenceTypeNone     ReferenceType = ""
    ReferenceTypeMessage  ReferenceType = "message" // sms, mms
    ReferenceTypeLine     ReferenceType = "line"
    ReferenceTypeWhatsApp ReferenceType = "whatsapp"
    ReferenceTypeEmail    ReferenceType = "email"   // NEW
)
```

Direction for email is always `outgoing` in Phase 1. `incoming` is never
produced for email. Email message `status` is **`progressing`** at creation
(see §5 and §3.4): `email_created` is published by email-manager *before* the
provider send is attempted, so the email has not yet been delivered when the
conversation message is created. No `StatusNone`-style hazard is introduced;
the existing message `Status` enum (`progressing`/`done`/`failed`) is reused.

### 3.2 Conversation messages are an immutable send-time fact log

This is the load-bearing modeling decision, and it matches how SMS already
works in this service (verified in code, not assumed):

- `bin-conversation-manager`'s `subscribehandler` subscribes to **only**
  `message_created` (`pkg/subscribehandler/main.go:138`). It does NOT
  subscribe to `message_updated` or `message_deleted`; all other events hit
  the `default: // ignore the event` arm.
- Therefore an SMS message body is **snapshotted** into
  `conversation_messages` at `message_created` time and is never updated or
  cascade-deleted afterward. If the source message in `bin-message-manager`
  changes or is deleted, the conversation copy is unaffected.
- Email follows the identical contract: the conversation message is an
  **append-only record of "we sent this email at time T"**, not a live mirror
  of `email_emails`. Consequences, stated explicitly so there is no ambiguity:
  - The subject/body copy is an intentional point-in-time snapshot, not a
    derived value to be looked up at read time. (The derived-value-rejection
    rule targets mutable lookup-able state, not immutable event snapshots;
    SMS `text` is copied on the same basis.)
  - `bin-conversation-manager` does **NOT** subscribe to `email_updated` or
    `email_deleted`. Source-email deletion does NOT cascade to the
    conversation log, exactly as SMS deletion does not. PII retention of the
    conversation log is governed by conversation-manager's own retention
    policy, not by the source email's lifecycle (see §13).

### 3.3 Subject field on Message

Email carries a `subject` that SMS/LINE/WhatsApp do not. Two options:

- **Option A (recommended):** add a nullable `Subject string` column to
  `conversation_messages`, populated only for email messages, empty for all
  others. Read-time consumers see it directly.
- Option B: pack subject into the existing `text` body with a separator.
  Rejected: lossy, breaks body rendering, not queryable.

Recommended struct addition (`models/message/message.go`):

```go
type Message struct {
    // ... existing fields ...
    Text    string        `json:"text,omitempty"    db:"text"`
    Subject string        `json:"subject,omitempty" db:"subject"` // NEW: email only
    Medias  []media.Media `json:"medias,omitempty"  db:"medias,json"`
    // ...
}
```

Body (`Content`) maps to `Text`. Attachments are NOT mapped in Phase 1
(email attachments are storage references, not inline media; see Q2).

### 3.4 No Account record for email

LINE/WhatsApp require a `conversation_accounts` row (platform credentials).
SMS does not (it keys off the number). Email follows the SMS model:
**no account row.** The email's own `source`/`destination` addresses
(already `commonaddress.Address` in the email model) map directly onto
conversation `self`/`peer`. No new credential storage, no `account.TypeEmail`.

## 4. Conversation/Message identity

A conversation is uniquely `(account_id, dialog_id)`. For email, mirroring
the SMS approach (`account_id = uuid.Nil`, `dialog_id = ""`), the thread is
located by `GetOrCreateBySelfAndPeer` on the normalized `(self, peer)` email
address pair:

- `self` = email `Source` address (the platform-side sender), type `email`.
- `peer` = email `Destination` address (the customer), type `email`.
- `dialog_id` = `""` (no external thread id; same as SMS).

`commonaddress.TypeEmail` already exists and already has normalize/validate
support (lowercasing, trim), so the existing `NormalizeTarget` authority
applies with no new normalization code.

### Multi-destination fan-out

An email may have multiple `Destinations`. Each destination yields its own
`(self, peer)` conversation and its own outgoing message, exactly as
`MessageEventSent` already loops over `m.Targets` for SMS. One email to N
recipients produces N conversation messages across N threads.

`Email.Source` is a pointer (`*commonaddress.Address`) and may be nil on a
deserialized event. `EmailEventSent` MUST nil-guard `Source` and skip (log
and return nil) when it is nil, rather than dereferencing. `Destinations` is
a value slice and needs no such guard.

The email model has a flat `Destinations []Address` with no To/CC/BCC
distinction. Modeling each recipient as an independent per-peer fact is the
only sound choice given that data, and it matches the contact-center unit of
interest ("the conversation with this person"). CC/BCC semantics are
explicitly not represented. If email-manager later adds CC/BCC, this
flattening is intentional and would need a follow-up design.

## 5. Inbound flow (event subscription)

`bin-conversation-manager` already runs a `subscribehandler` consuming
`QueueNameMessageEvent`. Add `QueueNameEmailEvent`
(`bin-manager.email-manager.event`) to `subscribeTargets`, plus a new
`publisherEmailManager = "email-manager"` const and a dedicated routing
branch in `subscribehandler/main.go` (the existing
`(publisher, type)`-tuple switch). Email is NOT routed through the existing
`Event`/`eventSMS` direction dispatcher: that path switches on
`mm.Direction`, and the email model has no `Direction` field. A new dedicated
branch calls `EmailEventSent` directly.

```
email-manager publishes email_created (status is always StatusInitiated;
    fired BEFORE the provider send is attempted)
    → conversation subscribehandler matches (publisher=email-manager,
      type=email_created)
    → conversationhandler.EmailEventSent(ctx, &email)
    → if email.Source == nil: log + return nil   // nil-guard
    → self = *email.Source
    → for each destination in email.Destinations:
        → txID = email.ID.String() + ":" + normalized(destination)   // dedup key
        → if h.db.MessageGetsByTransactionID(ctx, txID, ...) non-empty: skip  // idempotent
        → cv = GetOrCreateBySelfAndPeer(customerID, TypeEmail, "", self, destination)
        → messageHandler.Create(
              id            = uuid.Nil,           // auto-generate PK (NOT email.ID; see dedup)
              customerID    = email.CustomerID,
              conversationID= cv.ID,
              direction     = outgoing,
              status        = progressing,        // email not yet sent at email_created
              referenceType = ReferenceTypeEmail,
              referenceID   = email.ID,           // points at the source email
              transactionID = txID,               // composite (email.ID, peer) dedup key
              subject       = email.Subject,      // requires Create signature extension
              text          = email.Content,
              medias        = []                  // attachments not mapped (Q2)
          )
        → publish conversation_created (if new) + message_created
```

No `ExecuteMode` dispatch is invoked for email (outbound-only: nothing to
trigger a flow on). This is the key simplification versus the SMS path.

> Implementation note: the current `messageHandler.Create` signature
> (`pkg/messagehandler/db.go`) is
> `(ctx, id, customerID, conversationID, direction, status, referenceType,
> referenceID, transactionID, text, medias)` — it has **no `subject`
> parameter**. The implementation must extend `Create` (and the dbhandler
> `MessageCreate` mapping) to accept and persist `subject`. This is listed in
> the Implementation Order.

### Idempotency / dedup

The existing SMS path dedups by **primary-key id**: `MessageEventSent` calls
`messageHandler.Get(ctx, m.ID)` and only creates when absent, passing the
source message id as the conversation message's PK so the second delivery
collides on the PK. That single-key scheme is unsafe for email, because one
`email.ID` fans out to N destinations; reusing `email.ID` as the dedup key
would create only the first recipient's message and silently skip the rest.

Email therefore dedups on a **composite transaction id**
`transaction_id = email.ID + ":" + normalized(peer)`, which is unique per
(email, recipient). The conversation message PK is auto-generated
(`id = uuid.Nil`), and `reference_id` remains `email.ID` (it points at the
source email, shared across the fan-out). Re-delivery of the same
`email_created` event is idempotent: each `(email.ID, peer)` lookup finds the
existing row and skips creation. (No SMS twin-dedup / RPC-vs-event double
path concern: email is single-sourced from the email-manager event.)

The lookup method (`MessageGetsByTransactionID`) and the
`idx_conversation_messages_transaction_id` index both already exist in
`bin-conversation-manager` (`pkg/dbhandler/message.go:204` on the `DBHandler`
interface, `scripts/database_scripts_test/table_conversation_messages.sql:28`),
so dedup adds no new dbhandler query method and no new index. The method lives
on the **dbhandler** layer (reached via `conversationHandler.db`), not on the
`MessageHandler` interface; `EmailEventSent` calls `h.db.MessageGetsByTransactionID`
directly (message *creation* still goes through `h.messageHandler.Create`).
SMS currently writes an empty `transaction_id`; email is the first channel to
populate it, so there is no collision risk with existing rows.

## 6. Handler Interface

`pkg/conversationhandler/main.go` (additive):

```go
// EmailEventSent records a sent email as an outgoing conversation message.
EmailEventSent(ctx context.Context, e *emmemail.Email) error
```

`pkg/subscribehandler/` gains an email-manager event branch analogous to the
existing message-manager branch, unmarshalling into the email-manager
`email` model and dispatching on `email_created`.

## 7. REST API

No new endpoints. Email conversations and messages are read through the
existing conversation/message list/get endpoints. New filter value:

- `GET /v1/conversations?type=email` — list email conversations.
- `GET /v1/messages?conversation_id=<uuid>` — unchanged; returns email
  messages with `reference_type=email`, `direction=outgoing`, populated
  `subject`.

## 8. Webhook Events

No new event types. The existing conversation/message lifecycle events
(`conversation_created`, `conversation_updated`, `message_created`,
`message_updated`, `message_deleted`) fire for email exactly as for other
channels. Customer webhook payloads gain `reference_type=email` and the
`subject` field on the message object.

## 9. Flow Variable Integration

Not applicable in Phase 1. Email is outbound-only and triggers no activeflow,
so no `voipbin.conversation.*` flow variables are produced for the email
path. (Existing variables continue to work for SMS/LINE/WhatsApp.)

## 10. RabbitMQ Integration

- **New subscription:** `bin-conversation-manager` adds
  `bin-manager.email-manager.event` to its subscribe targets.
- **No new RPC calls.** Phase 1 is purely event-driven absorption.
- email-manager is unchanged (it already publishes `email_created`).

## 11. Database Schema

`conversation_messages`: add one column.

```sql
ALTER TABLE conversation_messages
    ADD COLUMN subject VARCHAR(255) NOT NULL DEFAULT '' AFTER text;
```

- `subject` is empty for all non-email messages; populated for email.
- `VARCHAR(255)`: confirm against the source `email_emails.subject` column
  width during implementation. If the source is wider (e.g. `TEXT`), widen
  this to match so the snapshot is not silently truncated.
- No new table. No index change required: dedup reuses the existing
  `idx_conversation_messages_transaction_id` index; email messages are read
  by the existing `(conversation_id)` / `(customer_id, tm_create)` indexes.
- Alembic migration in the conversation-manager schema repo;
  `tm_delete DEFAULT '9999-01-01 00:00:00.000000'` convention already on the
  table and is not altered.

> Modeling note (shared-table column): `subject` is a channel-specific field
> on the shared `conversation_messages` table, empty for 3 of 4 channel types.
> This mirrors how `text`/`medias` already live on the shared row and keeps
> the message a single self-contained record (no side-table join to render a
> thread). A per-channel side table was considered and rejected as
> over-engineering for one column. If future channels add many idiosyncratic
> fields, revisit with a side table or a typed JSON `metadata` column.

## 12. Observability

Reuse existing subscribehandler Prometheus metrics. Add label coverage so
email events are counted under the existing
`receive_request_process_time` / event-processing counters with an
`email_created` dimension. No new metric family required; if event-type
cardinality matters, add a `conversation_email_event_total` counter
(by result: ok/error) in Phase 1.

Trace ID is propagated from the subscribed event context into
`EmailEventSent` (no goroutine fan-out; synchronous processing per event).

## 13. Security & Compliance

- **Customer ownership:** `email.CustomerID` is the authority; the created
  conversation and message inherit it. No cross-customer leakage path is
  introduced (email-manager already scopes by customer).
- **PII:** email subject and body are snapshotted verbatim into
  `conversation_messages` as an immutable send-time fact (§3.2). This creates
  a **second, independent persistence surface** with its own retention
  lifecycle: it is NOT cascade-deleted when the source `email_emails` row is
  deleted (conversation-manager does not subscribe to `email_deleted`, exactly
  as it does not subscribe to `message_deleted` for SMS). This is the same
  immutable-log contract SMS already has. Implications to acknowledge:
  - No new external system or LLM sees this data; it stays inside the
    platform DB. There is no new external egress.
  - There IS a new internal retention surface. A GDPR/erasure request that
    purges `email_emails` will NOT automatically purge the conversation copy;
    erasure must also target `conversation_messages` (by `reference_id`).
    This is a deliberate consequence of the immutable-log model and must be
    covered by the platform's data-retention/erasure procedure.
- No credentials are stored (no account row for email).

## 14. Affected Services

| Service | Change | Phase |
|---------|--------|-------|
| bin-conversation-manager | New `email` type + reference type; subscribe to email events; `EmailEventSent` handler; `subject` column + model field | 1 |
| bin-email-manager | None (already publishes `email_created`) | 1 |
| bin-api-manager / OpenAPI | Document `type=email` and message `subject` field in conversation/message schemas | 1 |
| docs (RST) | Note email as an outbound conversation channel | 1 |

## 15. Implementation Order

1. Add `TypeEmail` / `ReferenceTypeEmail` enum values.
2. Alembic migration: `subject` column on `conversation_messages`; add
   `Subject` to the `Message` model + dbhandler `messageGetFromRow` /
   `MessageCreate` mapping.
3. Extend `messageHandler.Create` (and its interface in
   `pkg/messagehandler/main.go`) with a `subject` parameter; update all
   existing callers (SMS/LINE/WhatsApp pass `""`). Regenerate the
   messagehandler mock.
4. subscribehandler: add `QueueNameEmailEvent` to `subscribeTargets`
   (`cmd/conversation-manager/main.go`), add `publisherEmailManager` const,
   and add a dedicated `(email-manager, email_created)` routing branch that
   calls `EmailEventSent` (NOT the `Event`/`eventSMS` direction dispatcher).
5. `EmailEventSent` handler: nil-guard `Source`; per-destination loop;
   composite `transaction_id` dedup via `MessageGetsByTransactionID`;
   locate-or-create conversation; create outgoing message at
   `status=progressing`; publish `conversation_created` (if new) +
   `message_created`.
6. Unit tests (table-driven, gomock): new conversation, existing
   conversation, multi-destination fan-out (N messages across N threads),
   duplicate-event dedup (re-delivery is a no-op), nil-`Source` skip.
7. OpenAPI + RST documentation sync (`type=email`, message `subject` field).

## 16. Open Questions

| # | Question | Resolution | Owner |
|---|----------|------------|-------|
| Q1 | Should an email with N destinations create N separate threads, or one thread? | **Decided: N threads** (one per peer), matching SMS `MessageEventSent` fan-out. Email model has flat `Destinations` with no CC/BCC, and per-peer is the contact-center unit. Dedup keyed on composite `(email.ID, peer)`. | CEO/CTO |
| Q2 | Should email attachments be mirrored into the conversation message `medias`? | Defer. Email attachments are storage references with a different model than inline message media; map in a later phase if needed. | CPO |
| Q3 | What `status` should the email conversation message carry? | **Decided (A): `progressing`, fixed.** `email_created` fires before the provider send, so `done` would assert an unverified delivery. The message stays `progressing` in Phase 1 (conversation-manager does not subscribe to `email_updated`). Reflecting delivery transitions (done/failed on `email_updated`) is Phase 2. | CEO/CTO |
| Q4 | Confirm there is genuinely no inbound email path. | Confirmed by code: email-manager has no subscribehandler and its provider webhooks carry only delivery-status updates, not customer-originated messages. | CPO |
| Q5 | Lifecycle: should `email_deleted`/`email_updated` cascade to the conversation copy? | **Decided: no cascade.** The conversation message is an immutable send-time fact log (§3.2), identical to SMS (which ignores `message_updated`/`message_deleted`). GDPR erasure is handled by the platform retention procedure targeting `conversation_messages` by `reference_id` (§13), not by event cascade. | CEO/CTO |
```