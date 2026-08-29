# bin-tag-manager Architecture

## Component Overview

`bin-tag-manager` is a lightweight CRUD microservice for managing customer-scoped tags. Tags are labels that other services (contacts, queues, campaigns) attach to resources for categorization and filtering. The service handles tag lifecycle operations and event-driven cascading deletes when customers are removed.

```
cmd/tag-manager/        — Daemon entry point (pflag/Viper)
cmd/tag-control/        — Admin CLI (JSON output, bypasses RabbitMQ)
pkg/listenhandler/      — RabbitMQ RPC request router (regex URI dispatch)
pkg/subscribehandler/   — Event subscriber (customer_deleted cascading delete)
pkg/taghandler/         — Core business logic and event publishing
pkg/dbhandler/          — MySQL + Redis cache coordination
pkg/cachehandler/       — Redis cache operations for tag lookups
models/tag/             — Data structures (Tag, event types, WebhookMessage)
```

## Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|---------------|
| Entry | `cmd/tag-manager` | Configuration; starts ListenHandler and SubscribeHandler |
| Transport | `pkg/listenhandler` | Consumes `bin-manager.tag-manager.request`; regex-routes to taghandler |
| Events | `pkg/subscribehandler` | Consumes customer events via a pattern binding on the global topic exchange `bin-manager.event` (sole intake mechanism since VOIP-1407); cascades customer deletes |
| Business logic | `pkg/taghandler` | CRUD operations; publishes `tag_created`, `tag_updated`, `tag_deleted` events |
| Persistence | `pkg/dbhandler` | MySQL writes with soft-delete (`tm_delete`); Redis cache invalidation |
| Cache | `pkg/cachehandler` | Redis reads for fast tag lookups |

### Soft Deletes

Active records have `tm_delete = "9999-01-01 00:00:00.000000"`. A delete operation sets `tm_delete` to the current timestamp. Queries filter on this sentinel value.

## Request Routing

Requests arrive on the RabbitMQ queue `bin-manager.tag-manager.request`. The `listenhandler` dispatches by matching the URI against compiled regexes.

| Pattern | Operations |
|---------|-----------|
| `/v1/tags$` | POST (create) |
| `/v1/tags?(.*)$` | GET (list with filters/pagination) |
| `/v1/tags/<uuid>$` | GET, PUT (update), DELETE |

Request flow:

```
RabbitMQ → listenhandler (regex dispatch)
               |
               v
           taghandler
           |         |
       dbhandler   notifyhandler
       (MySQL/     (RabbitMQ event
        Redis)      publish)

Event flow:
RabbitMQ → subscribehandler → taghandler → bulk delete
```

### Events Subscribed

SubscribeHandler (`pkg/subscribehandler/`) consumes from the queue `bin-manager.tag-manager.subscribe`. Since VOIP-1406 the queue is bound to the **global topic exchange `bin-manager.event`** with one pattern per dispatched (publisher, event-type) pair — 1 pattern total, pinned byte-for-byte by the binding golden test (`pkg/subscribehandler/binding_golden_test.go`):

| Pattern | Purpose |
|---------|---------|
| `customer-manager.customer.*.deleted` | Customer deletion — cascading bulk tag delete |

As of VOIP-1407 this topic-pattern binding is the **sole intake mechanism**; the old per-service fanout subscription (`QueueSubscribe` to `bin-manager.customer-manager.event`) has been removed from `Run()` entirely, along with the fanout-unbind step that used to follow a successful topic bind.

### Events Published

Tag state changes emit events on `bin-manager.tag-manager.event`:

| Event | Trigger |
|-------|---------|
| `tag_created` | Successful POST |
| `tag_updated` | Successful PUT |
| `tag_deleted` | Successful DELETE or cascading customer delete |

All three carry a `*tag.Tag` payload.

### Global topic exchange (VOIP-1404 / VOIP-1405)

Both NotifyHandler construction sites — `cmd/tag-manager/main.go` and `cmd/tag-control/main.go` —
construct their NotifyHandler with `notifyhandler.WithGlobalTopicPublish()`. Every event is
therefore published twice: once to the per-service fanout exchange `bin-manager.tag-manager.event`
(unchanged, still the system of record) and once to the global topic exchange `bin-manager.event`
with the routing key `tag-manager.tag.<tag-id>.<action>`. The two cmds must stay in lockstep on
this option — enabling it in only one would leave consumers with gaps depending on which process
published. A topic publish failure never propagates to the caller and never affects the fanout
publish.

`*tag.Tag` carries an explicit `EventSubscriptionID()` returning the tag's own `id`
(VOIP-1419): a tag is an independent persistent resource addressed by its own `id`.
The golden table pinning every key is
`models/tag/routingkey_golden_test.go`; the schema is defined in monorepo
`docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md`.
