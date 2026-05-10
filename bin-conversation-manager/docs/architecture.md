# bin-conversation-manager Architecture

## Component Overview

`bin-conversation-manager` manages multi-channel conversation threads (SMS/MMS, LINE messaging) and their messages. It handles bidirectional communication with external platforms and maintains conversation state across interactions.

```
cmd/conversation-manager/main.go
    ├── pkg/dbhandler            (MySQL + Redis cache via Squirrel SQL builder)
    ├── pkg/cachehandler         (Redis operations)
    ├── pkg/accounthandler       (messaging platform credentials)
    ├── pkg/conversationhandler  (conversation business logic + webhook processing)
    ├── pkg/messagehandler       (message create/send/status)
    ├── pkg/linehandler          (LINE Bot SDK v7 integration)
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
| Domain | `pkg/conversationhandler` | Create/get/update conversations; process platform webhooks; event handling |
| Domain | `pkg/messagehandler` | Create message records; dispatch outbound messages; status tracking |
| Domain | `pkg/accounthandler` | CRUD for platform credentials (LINE channel secret/token, SMS credentials) |
| Platform | `pkg/linehandler` | LINE Bot SDK: webhook signature verification, message send/receive |
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
| `GET/PUT/DELETE /v1/conversations/<uuid>` | Get / update / delete conversation |
| `POST /v1/conversations` | Create conversation |
| `POST /v1/hooks` | Incoming platform webhook (LINE, etc.) |
| `GET /v1/messages?` | List messages (paginated) |
| `GET /v1/messages/<uuid>` | Get message |
| `POST /v1/messages` | Create message record |
| `POST /v1/messages/create` | Create and send message |

## Event Subscriptions

SubscribeHandler (`pkg/subscribehandler/`) consumes:

| Queue | Event | Action |
|-------|-------|--------|
| `bin-manager.message-manager.event` | `message_created` | Create or update conversation record; create inbound message; publish conversation/message events |

## Events Published

Exchange: `bin-manager.conversation-manager.event`

| Event type | Trigger |
|-----------|---------|
| `conversation.EventTypeConversationCreated` | New conversation thread created |
| `conversation.EventTypeConversationUpdated` | Conversation metadata updated |
| `message.EventTypeMessageCreated` | New message added to conversation |
| `message.EventTypeMessageDeleted` | Message deleted |
| `message.EventTypeMessageUpdated` | Message status updated |

## External Webhook Processing

The service receives inbound webhooks from external platforms via the `POST /v1/hooks` endpoint:

- **LINE**: Webhook from LINE Bot platform; `pkg/linehandler/hook.go` verifies signature, parses payload, creates conversation and message records
- Hook URI pattern in LINE config: `/v1.0/conversation/accounts/<account_id>`
