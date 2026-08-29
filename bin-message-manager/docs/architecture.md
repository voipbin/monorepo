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

## Event Publishing

Both `cmd/message-manager` and `cmd/message-control` construct their NotifyHandler with `notifyhandler.WithGlobalTopicPublish()`, so every event is published to the global topic exchange `bin-manager.event` with the routing key `message-manager.message.<message-id>.<action>`. **As of VOIP-1407, this is the sole publish path** — the previous per-service fanout exchange `bin-manager.message-manager.event` is no longer published to, and (per the operational runbook in `docs/reference/rabbitmq-queues-reference.md`) will eventually be deleted from the broker. No code in this service changed for VOIP-1407; the behavior change (dual publish → topic-only) comes entirely from `bin-common-handler/pkg/notifyhandler`'s shared library update. The two cmds must stay in lockstep on this option — enabling it in only one would leave consumers with gaps depending on which process published. A topic publish failure now propagates to the caller as an error (previously it was swallowed silently).

The third routing-key segment is the *subscription address* — the id consumers bind to. `message.Message` deliberately carries no `eventtopic.SubscriptionIdentifier` override: unlike the same-named stream-fragment types in ai/conversation/talk/webchat/tts, this service's `Message` is the SMS resource itself, so its own id is the address and the default JSON `id` extraction covers it. `models/message/routingkey_golden_test.go` pins the exact key of every published event type (`message_created`, `message_updated`, `message_deleted`) plus the deliberate absence of that override. See the monorepo `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md` for the schema and `docs/plans/2026-08-27-voip-1405-topic-publisher-rollout-design.md` §2.4 for the address mapping.

## Request Routing

The `listenhandler` consumes from queue `bin-manager.message-manager.request` and dispatches by regex-matching the request URI:

| Method | URI Pattern | Handler |
|--------|------------|---------|
| POST | `/v1/messages` | Send SMS (balance check → create → async dispatch to provider) |
| GET | `/v1/messages?` | List messages (pagination) |
| GET | `/v1/messages/{uuid}` | Get message details |
| DELETE | `/v1/messages/{uuid}` | Delete message |
| POST | `/v1/hooks` | Process provider webhook (delivery status update) |
