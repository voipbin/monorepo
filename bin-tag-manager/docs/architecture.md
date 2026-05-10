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
| Events | `pkg/subscribehandler` | Subscribes to `bin-manager.customer-manager.event`; cascades customer deletes |
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

### Events Published

Tag state changes emit events on `bin-manager.tag-manager.event`:

| Event | Trigger |
|-------|---------|
| `tag_created` | Successful POST |
| `tag_updated` | Successful PUT |
| `tag_deleted` | Successful DELETE or cascading customer delete |
