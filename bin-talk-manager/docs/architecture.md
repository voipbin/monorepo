# bin-talk-manager Architecture

## Component Overview

`bin-talk-manager` manages chat sessions, messages, participants, and reactions for the VoIPbin platform. It exposes a REST-style RPC interface over RabbitMQ and persists data in MySQL with Redis caching.

```
cmd/talk-manager/main.go
    ├── MySQL connection (pkg/dbhandler)
    ├── Redis cache connection
    ├── RabbitMQ connection (sockhandler)
    ├── runServiceListen()  → pkg/listenhandler
    └── Prometheus metrics endpoint (:2112)
```

Key packages:

| Package | Role |
|---------|------|
| `pkg/listenhandler` | RabbitMQ RPC routing via regex patterns |
| `pkg/chathandler` | Chat CRUD and business logic |
| `pkg/messagehandler` | Message creation with threading validation |
| `pkg/participanthandler` | Chat membership UPSERT operations |
| `pkg/reactionhandler` | Atomic emoji reaction management |
| `pkg/dbhandler` | MySQL + Redis persistence |
| `pkg/notifyhandler` (shared) | Event publishing to `bin-manager.talk-manager.event` and, since VOIP-1405, dual publishing to the global topic exchange `bin-manager.event` |

`cmd/talk-manager` constructs its NotifyHandler with `commonnotify.WithGlobalTopicPublish()`, so every event is published twice: once to the per-service fanout exchange `bin-manager.talk-manager.event` (unchanged, still the system of record) and once to the global topic exchange `bin-manager.event` with the routing key `talk-manager.<resource>.<chat-id>.<action>`. A topic publish failure never propagates to the caller and never affects the fanout publish.

`cmd/talk-control` deliberately does NOT get the option in this change: it constructs its NotifyHandler with an empty `queueEvent` and a nil `reqHandler`, a pre-existing defect tracked as a separate ticket (VOIP-1405 §7). The option is to be added there only after that defect is fixed. See [docs/domain.md](domain.md) for the per-event routing keys and the monorepo `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md` for the schema.

## Layer Responsibilities

```
listenhandler           — deserializes RabbitMQ RPC, routes by URI+method regex
    │
    ├─ chathandler      — validates inputs, orchestrates DB + notify
    ├─ messagehandler   — threading validation, parent-chat consistency check
    ├─ participanthandler — UPSERT membership, member_count updates
    └─ reactionhandler  — atomic JSON_ARRAY_APPEND/REMOVE in MySQL
            │
            └─ dbhandler — raw SQL on MySQL; Redis cache for lookups
```

Rules:
- Domain handlers own all business rules; `listenhandler` only routes and serializes.
- `dbhandler` operates exclusively on database types; conversion is done in domain handlers.
- Events are published by domain handlers after successful DB writes via `notifyhandler`.

## Request Routing

Requests arrive on queue `bin-manager.talk-manager.request`. The listenhandler matches URI + HTTP method with `regexp.MustCompile` patterns:

| Method | URI Pattern | Handler |
|--------|-------------|---------|
| POST | `/v1/chats` | `v1ChatsPost` |
| GET | `/v1/chats` | `v1ChatsGet` |
| GET | `/v1/chats/{id}` | `v1ChatsIDGet` |
| PUT | `/v1/chats/{id}` | `v1ChatsIDPut` |
| DELETE | `/v1/chats/{id}` | `v1ChatsIDDelete` |
| POST | `/v1/chats/{id}/participants` | `v1ChatsIDParticipantsPost` |
| GET | `/v1/chats/{id}/participants` | `v1ChatsIDParticipantsGet` |
| DELETE | `/v1/chats/{id}/participants/{pid}` | `v1ChatsIDParticipantsIDDelete` |
| GET | `/v1/participants` | `v1ParticipantsGet` |
| POST | `/v1/messages` | `v1MessagesPost` |
| GET | `/v1/messages` | `v1MessagesGet` |
| GET | `/v1/messages/{id}` | `v1MessagesIDGet` |
| DELETE | `/v1/messages/{id}` | `v1MessagesIDDelete` |
| POST | `/v1/messages/{id}/reactions` | `v1MessagesIDReactionsPost` |
| DELETE | `/v1/messages/{id}/reactions` | `v1MessagesIDReactionsDelete` |

No per-pod queue routing — all replicas handle any request.
