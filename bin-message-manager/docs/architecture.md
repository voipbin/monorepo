# bin-message-manager — Architecture

## Component Overview

`bin-message-manager` is a Class A Standard Go RPC Manager that handles SMS messaging through external providers (Telnyx, MessageBird). It manages message creation, asynchronous delivery, per-target status tracking, and inbound provider webhooks for delivery updates.

```
cmd/message-manager/main.go
    ├── pkg/cachehandler      (Redis — message lookups)
    ├── pkg/dbhandler         (MySQL via Squirrel query builder)
    ├── pkg/requestexternal   (HTTP clients for Telnyx/MessageBird APIs)
    ├── pkg/messagehandler    (Business logic — send, get, delete, hook)
    └── pkg/listenhandler     (RabbitMQ RPC — messages & hooks API)
```

Supporting binary:
- `cmd/message-control/` — CLI for direct DB/cache operations, bypasses RabbitMQ RPC.

**No SubscribeHandler.** Provider delivery status is received via inbound `POST /v1/hooks` webhooks forwarded through `bin-hook-manager`.

## Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|---------------|
| Entry | `cmd/message-manager` | Cobra + Viper config, dependency wiring, daemon start |
| Listen | `pkg/listenhandler` | RabbitMQ RPC routing; dispatches to messagehandler |
| Business | `pkg/messagehandler` | Message CRUD, billing validation, provider dispatch, webhook processing |
| Provider | `pkg/messagehandler/provider_telnyx.go` | Telnyx SMS API send per target |
| Provider | `pkg/messagehandler/provider_messagebird.go` | MessageBird SMS API send per target |
| External | `pkg/requestexternal` | HTTP client wrapper for provider APIs |
| Data | `pkg/dbhandler` | Squirrel SQL builder queries for messages and targets |
| Cache | `pkg/cachehandler` | Redis caching for message lookups |
| Models | `models/message` | Message, Type, Direction, ProviderName, Status |
| Models | `models/target` | Target (recipient) with delivery status |
| Models | `models/telnyx` | Telnyx webhook payload models |
| Models | `models/messagebird` | MessageBird webhook payload models |

## Request Routing

The `listenhandler` consumes from queue `bin-manager.message-manager.request` and dispatches by regex-matching the request URI:

| Method | URI Pattern | Handler |
|--------|------------|---------|
| POST | `/v1/messages` | Send SMS (balance check → create → async dispatch to provider) |
| GET | `/v1/messages?` | List messages (pagination) |
| GET | `/v1/messages/{uuid}` | Get message details |
| DELETE | `/v1/messages/{uuid}` | Delete message |
| POST | `/v1/hooks` | Process provider webhook (delivery status update) |
