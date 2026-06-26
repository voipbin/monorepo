# bin-conversation-manager Domain Model

## Core Concepts

### Account
Platform-specific credentials for sending messages. Stored in `conversation_accounts` table.

- `type`: `sms` | `line` | `whatsapp`
- `secret` — LINE channel secret, SMS provider credential, or WhatsApp webhook verify token (write-only)
- `token` — LINE channel access token or Meta system user access token (write-only)
- `provider_data` — platform-specific JSON blob; for `whatsapp` contains `phone_number_id` (Meta phone number ID) and `app_secret` (HMAC signing key for inbound webhook validation). Returned in API responses.

### Conversation
A communication thread between two parties. Persisted in `conversation_conversations` table.

- `type`: `message` (SMS/MMS) | `line` (LINE) | `whatsapp` (WhatsApp)
- `dialog_id` — external platform conversation identifier (LINE chatroom ID, WhatsApp recipient phone, SMS thread ID)
- `self` — platform address of the VoIPbin-side participant (e.g., phone number, LINE user ID)
- `peer` — platform address of the customer/end user

A conversation is uniquely identified by `(account_id, dialog_id)`.

### Message
Individual message within a conversation. Persisted in `conversation_messages` table.

- `direction`: `incoming` | `outgoing`
- `status`: `progressing` | `done` | `failed`
- `reference_type` — source resource type: `message` (SMS/MMS) | `line` | `whatsapp`
- `transaction_id` — external platform message ID for dedup
- `medias` — JSON array of media attachments

### Media
Media attachment metadata for messages. Stored in `conversation_medias` table.

## Message Flows

### Incoming Message (LINE / WhatsApp webhook)
```
LINE / WhatsApp platform → POST /v1/hooks
    → conversationhandler.Hook() (parse account_id from URI, select handler by account type)
    → linehandler.Hook() / whatsapphandler.Hook() (verify signature, parse payload)
    → find or create Conversation by (account_id, dialog_id)
    → create Message record (direction=incoming)
    → publish conversation_created + message_created events
    → execute-mode dispatch (agent: no-op; flow: start activeflow via account.MessageFlowID)
```

WhatsApp note: inbound signature is validated as HMAC-SHA256 (`X-Hub-Signature-256`, `sha256=` prefix) keyed on `provider_data.app_secret`. The handler is fail-closed: an empty `app_secret` rejects the request.

### Outgoing Message (API)
```
Caller → POST /v1/messages/create
    → messagehandler.CreateSend()
    → create Message record (direction=outgoing, status=progressing)
    → linehandler.Send() / whatsapphandler.Send() / smshandler.Send()
    → update Message status to done or failed
    → publish message_created event
```

### Incoming SMS/MMS (event subscription)
```
message-manager publishes message_created event
    → subscribehandler processes event
    → conversationhandler.Event()
    → find or create Conversation by phone numbers
    → create Message record (direction=incoming)
    → publish conversation_created + message_created events
    → execute-mode dispatch (agent: no-op; flow: start activeflow via number.MessageFlowID)
```

## Flow Integration Variables

Conversations can set flow context variables for integration with `bin-flow-manager`:

| Variable | Value |
|----------|-------|
| `voipbin.conversation.self.*` | Self party: name, detail, target, type |
| `voipbin.conversation.peer.*` | Peer party: name, detail, target, type |
| `voipbin.conversation.id` | Conversation UUID |
| `voipbin.conversation.owner_id` | Owner UUID |
| `voipbin.conversation.message.text` | Most recent message text |

## Database Schema

Tables (schemas in `scripts/database_scripts_test/`):

| Table | Purpose |
|-------|---------|
| `conversation_accounts` | Platform credentials (type, secret, token) |
| `conversation_conversations` | Conversation threads (type, dialog_id, self, peer) |
| `conversation_messages` | Individual messages (direction, status, text, medias JSON) |
| `conversation_medias` | Media attachment metadata |

**Soft deletes:** `tm_delete = "9999-01-01 00:00:00.000000"` for active records.

**Cache:** Redis cache for account and conversation lookups by ID. DB is source of truth; cache invalidated on mutations.
