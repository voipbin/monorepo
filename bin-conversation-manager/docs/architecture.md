# bin-conversation-manager Architecture

## Component Overview

`bin-conversation-manager` manages multi-channel conversation threads (SMS/MMS, LINE, WhatsApp) and their messages. It handles bidirectional communication with external platforms and maintains conversation state across interactions.

```
cmd/conversation-manager/main.go
    ├── pkg/dbhandler            (MySQL + Redis cache via Squirrel SQL builder)
    ├── pkg/cachehandler         (Redis operations)
    ├── pkg/accounthandler       (messaging platform credentials)
    ├── pkg/conversationhandler  (conversation business logic + webhook processing + execute-mode dispatch)
    ├── pkg/messagehandler       (message create/send/status)
    ├── pkg/linehandler          (LINE Bot SDK v7 integration)
    ├── pkg/whatsapphandler      (WhatsApp Meta Cloud API integration)
    ├── pkg/smshandler           (SMS/MMS via message-manager RPC)
    ├── pkg/listenhandler        (RabbitMQ RPC router)
    └── pkg/subscribehandler     (event consumer from message-manager)
```

**Supporting binaries:**
- `cmd/conversation-control/` — CLI tool for direct DB/cache operations

## Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|----------------|
| Transport | `pkg/listenhandler` | Receives RPC requests from `bin-manager.conversation-manager.request`; routes by URI regex |
| Transport | `pkg/subscribehandler` | Consumes `message_created` events from message-manager (inbound SMS/MMS) |
| Transport | notifyhandler (bin-common-handler) | Publishes conversation/message events to exchange |
| Domain | `pkg/conversationhandler` | Create/get/update conversations; process platform webhooks; execute-mode dispatch (agent vs flow); event handling |
| Domain | `pkg/messagehandler` | Create message records; dispatch outbound messages; status tracking |
| Domain | `pkg/accounthandler` | CRUD for platform credentials (LINE channel secret/token, SMS credentials, WhatsApp Meta credentials) |
| Platform | `pkg/linehandler` | LINE Bot SDK: webhook signature verification, message send/receive |
| Platform | `pkg/whatsapphandler` | WhatsApp Meta Cloud API: HMAC webhook validation, hub challenge verification, message send/receive |
| Platform | `pkg/smshandler` | SMS/MMS sending via RPC to message-manager |
| Data | `pkg/dbhandler` | MySQL CRUD using Squirrel SQL builder |
| Data | `pkg/cachehandler` | Redis cache for account/conversation lookups |

## Request Routing

ListenHandler routes over `bin-manager.conversation-manager.request`:

| Pattern | Purpose |
|---------|---------|
| `GET /v1/accounts?` | List accounts (paginated) |
| `GET/PUT/DELETE /v1/accounts/<uuid>` | Get / update / delete account |
| `POST /v1/accounts` | Create messaging platform account |
| `GET /v1/conversations?` | List conversations (paginated) |
| `GET/PUT /v1/conversations/<uuid>` | Get / update conversation |
| `POST /v1/conversations` | Create conversation |
| `GET /v1/hooks` | Meta hub challenge verification (WhatsApp) |
| `POST /v1/hooks` | Incoming platform webhook (LINE, WhatsApp, etc.) |
| `GET /v1/messages?` | List messages (paginated) |
| `POST /v1/messages` | Create message record |
| `POST /v1/messages/create` | Create and send message |

## Event Subscriptions

SubscribeHandler (`pkg/subscribehandler/`) consumes:

| Queue | Event | Action |
|-------|-------|--------|
| `bin-manager.message-manager.event` | `message_created` | Create or update conversation record; create incoming message; publish conversation/message events |

## Events Published

Exchange: `bin-manager.conversation-manager.event`

| Event type | Trigger |
|-----------|---------|
| `account.EventTypeAccountCreated` | New platform account created |
| `account.EventTypeAccountUpdated` | Account credentials/metadata updated |
| `account.EventTypeAccountDeleted` | Account deleted |
| `conversation.EventTypeConversationCreated` | New conversation thread created |
| `conversation.EventTypeConversationUpdated` | Conversation metadata updated |
| `message.EventTypeMessageCreated` | New message added to conversation |
| `message.EventTypeMessageDeleted` | Message deleted |
| `message.EventTypeMessageUpdated` | Message status updated |

## Inbound Dispatch: Execute Mode

Every inbound message (LINE/WhatsApp webhook or SMS event) is dispatched by `conversationhandler` based on the conversation's owner snapshot (`pkg/conversationhandler/execute_mode.go`). The snapshot loaded by the inbound handler is authoritative: the dispatch path resolves the mode from the in-hand snapshot and never re-fetches the conversation (per-type flow runners fetch only the account or number, never the conversation).

| Mode | Condition | Action |
|------|-----------|--------|
| `ExecuteModeAgent` | `OwnerType == agent` and `OwnerID != uuid.Nil` | No flow triggered. The agent UI receives the new message via the `message_created` event filtered on `OwnerID`. Logging only. |
| `ExecuteModeFlow` | otherwise | Resolve the per-type flow source and start an activeflow. LINE/WhatsApp use `account.MessageFlowID`; SMS uses `number.MessageFlowID` resolved from `self.target`. A `uuid.Nil` flow id is a no-op. |

`ExecuteModeNone` is reserved (not currently produced by `getExecuteMode`). The flow integration creates an activeflow (`reference_type=conversation`), injects conversation variables, and executes it.

## External Webhook Processing

The service receives inbound webhooks from external platforms via the `POST /v1/hooks` endpoint. The account type (resolved from the `account_id` in the URI) selects the platform handler:

- **LINE**: `pkg/linehandler/hook.go` verifies the channel signature, parses the payload, and produces conversation/message records.
- **WhatsApp**: `pkg/whatsapphandler/hook.go` validates the Meta `X-Hub-Signature-256` HMAC (using `app_secret` from `provider_data`), parses the Cloud API payload, and produces conversation/message records. `GET /v1/hooks` answers the Meta hub challenge (`hub.mode`/`hub.verify_token`/`hub.challenge`) via `whatsapphandler.VerifyWebhook`. On signature failure the payload is discarded without persistence, and the endpoint still returns 200 so Meta does not retry a forged request (duplicate valid-signature deliveries are deduplicated by transaction/wamid, not by the 200).
- Hook URI pattern in the platform account config: `/v1.0/conversation/accounts/<account_id>`
