# bin-talk-manager Domain

## Domain Entities

### Chat

A chat session linking multiple participants. Stored in `talk_chats`.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owning tenant |
| `type` | enum | `direct` (1:1 private), `group` (multi-user private), `talk` (public channel) |
| `name` | string | Display name (optional for direct) |
| `detail` | string | Free-text description |
| `member_count` | int | Cached count of active participants |
| `tm_create` | timestamp | Creation time |
| `tm_update` | timestamp | Last update time |
| `tm_delete` | timestamp | Soft-delete marker |

### Participant

Membership record linking an owner (agent, customer, etc.) to a chat. Stored in `talk_participants`.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owning tenant |
| `chat_id` | UUID | FK to chat |
| `owner_type` | string | Type of participant entity |
| `owner_id` | UUID | ID of participant entity |
| `tm_joined` | timestamp | When joined (or most recent re-join) |

Hard delete — no `tm_delete` column. UNIQUE constraint on `(chat_id, owner_type, owner_id)` enables UPSERT for re-join.

### Message

A chat message with optional threading and emoji reactions. Stored in `talk_messages`.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owning tenant |
| `chat_id` | UUID | FK to chat |
| `owner_type` | string | Who sent the message |
| `owner_id` | UUID | Sender ID |
| `type` | enum | `normal` or `system` |
| `text` | string | Message body |
| `parent_id` | UUID | Optional: reply-to message ID |
| `metadata` | JSON | Reaction array (atomic JSON operations) |
| `medias` | JSON | Attachment array (address, agent, file, link types) |
| `tm_create` | timestamp | Creation time |
| `tm_update` | timestamp | Last update time |
| `tm_delete` | timestamp | Soft-delete marker |

### Pin

A personal state of an owner (an agent): the owner pinned a chat or a message. Stored in `talk_pins`.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owning tenant |
| `owner_type` | string | Who pinned (`agent`) |
| `owner_id` | UUID | Owner ID |
| `chat_id` | UUID | FK to chat. For a chat pin the chat itself; for a message pin the chat of the message. Filled by the server |
| `target_type` | enum | `chat` or `message` |
| `target_id` | UUID | Chat ID or message ID, depending on `target_type` |
| `tm_create` | timestamp | When pinned |

Hard delete — no `tm_delete` column. UNIQUE constraint on `(owner_type, owner_id, target_type, target_id)`: pinning again returns the existing pin. A message has no foreign key (messages are soft-deleted); the list filters deleted messages out.

### Mention

The index row of one agent mention in one message. Stored in `talk_mentions`; it is never exposed itself, the mention list returns the mentioning messages.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owning tenant |
| `chat_id` | UUID | FK to chat (`ON DELETE CASCADE`) |
| `message_id` | UUID | FK to message (`ON DELETE CASCADE`) |
| `agent_id` | UUID | The mentioned agent |
| `tm_create` | timestamp | The message's creation time (used as the page token) |

UNIQUE on `(message_id, agent_id)`; the list index is `(customer_id, agent_id, tm_create)`.

## Key Business Rules

### Agent Mentions

`messagehandler.MessageCreate` scans the text of a `normal` message for `[@display name](mention:<agent-id>)` (the id must be a UUID, so `mention:contact:<id>` never matches). Distinct agents are kept in order of appearance, the author is dropped, and at most 50 are recorded. The message row and its mention rows are written by `dbhandler.MessageCreateWithMentions` in one transaction: if a mention insert fails the message is not stored and the creation fails. A message without an agent mention, and every `system` message, uses the plain `MessageCreate`. The mentioned id is not checked against the agent service. Messages created before the feature was deployed have no mention rows (no backfill). Mentions are extracted only when a message is created: messages have no edit API today, and `dbhandler.MessageUpdate` (which could change `text`) has no caller outside tests, so a future text-editing feature has to re-extract the mentions of the message.

`GET /v1/mentions` (`customer_id`, `agent_id` in the body, `page_size`/`page_token` in the query) joins the mention rows to their messages and chats and returns the messages newest first, applying the access rule in SQL: the message and the chat are not deleted, the chat belongs to the customer, and the chat is a `talk` chat or the agent is an `agent` participant of it.

### Threading Validation

When creating a reply message (`parent_id` set):
1. Parent message must exist in the database (even if soft-deleted).
2. Parent message must belong to the same `chat_id` — prevents cross-chat threading attacks.
3. Soft-deleted parents are intentionally allowed to preserve thread structure. UI should render deleted parents as placeholders (e.g., "Message deleted").

### Atomic Reaction Operations

Reactions live in `talk_messages.metadata` as a JSON array. To prevent lost updates under concurrent reaction writes, the service uses atomic MySQL JSON operations:

- **Add**: `JSON_ARRAY_APPEND` inside a single `UPDATE` statement.
- **Remove**: `JSON_REMOVE` with `JSON_SEARCH` path extraction in a single `UPDATE` statement.

No application-level read-modify-write is used for reactions.

### Participant Re-join (UPSERT)

Adding a participant who already exists updates `tm_joined` rather than creating a duplicate. The UNIQUE constraint on `(chat_id, owner_type, owner_id)` enables this:
- MySQL production uses `ON DUPLICATE KEY UPDATE`.
- SQLite tests use `ON CONFLICT DO UPDATE SET`.

Both achieve the same semantics — never create duplicate memberships.

### Pin Visibility (rule R)

A pin is personal: only its owner sees it. The list shows a pin only if the chat of the pin is not deleted, belongs to the calling customer, and is a `talk` chat or a chat in which the owner is an `agent` participant; a message pin also needs a message that is not deleted and belongs to the chat. A pin whose target is hidden stays stored and reappears when the owner joins the chat again. Creating a pin applies the same conditions (a non-participant of a direct or group chat is refused). Deleting a pin needs only ownership, so a pin that is no longer visible can still be removed. api-manager checks `canAccessChat` before it creates a pin; rule R is deliberately stricter than `canAccessChat` on two axes: a deleted chat and a customer mismatch.

### Events Published

Every event below is published to the per-service fanout exchange `bin-manager.talk-manager.event`, and — since VOIP-1405 — also to the global topic exchange `bin-manager.event` with the routing key `talk-manager.<resource>.<chat-id>.<action>`. The third key segment is the *subscription address*: it is always the chat-id, across all three resource namespaces, so one conversation is followed with `talk-manager.chat.<id>.#`, `talk-manager.chatmessage.<id>.#`, and `talk-manager.chatparticipant.<id>.#`.

| Event | Data | Trigger | Topic routing key |
|-------|------|---------|-------------------|
| `chat.EventTypeChatCreated` | `*chat.Chat` | Chat successfully created | `talk-manager.chat.<chat-id>.created` |
| `chat.EventTypeChatUpdated` | `*chat.Chat` | Chat updated | `talk-manager.chat.<chat-id>.updated` |
| `chat.EventTypeChatDeleted` | `*chat.Chat` | Chat soft-deleted | `talk-manager.chat.<chat-id>.deleted` |
| `message.EventTypeMessageCreated` | `*message.Message` | Message successfully created | `talk-manager.chatmessage.<chat-id>.created` |
| `message.EventTypeMessageDeleted` | `*message.Message` | Message soft-deleted | `talk-manager.chatmessage.<chat-id>.deleted` |
| `message.EventTypeMessageReactionUpdated` | `*message.Message` | Reaction added or removed | `talk-manager.chatmessage.<chat-id>.reaction_updated` |
| `participant.EventParticipantAdded` | `*participant.Participant` | Participant added to a chat | `talk-manager.chatparticipant.<chat-id>.added` |
| `participant.EventParticipantRemoved` | `*participant.Participant` | Participant removed from a chat | `talk-manager.chatparticipant.<chat-id>.removed` |

`*message.Message` and `*participant.Participant` both implement `eventtopic.SubscriptionIdentifier` (pointer receiver) returning `ChatID`. Their own ids are stable and persisted, but each id first appears in that resource's own `created`/`added` event, so nobody can pre-bind to it, while every real consumption pattern follows a conversation; single-item retrieval stays available over RPC. `Chat` needs no override — its own id already is the subscription address. The golden routing-key table in `models/chat/routingkey_golden_test.go` pins the whole set.

### Timestamp Convention

All timestamps use `utilHandler.TimeGetCurTime()` which returns `YYYY-MM-DD HH:MM:SS.microseconds` format. Never use `time.Now().UTC().Format()` directly — it produces ISO 8601 format that MySQL rejects.
