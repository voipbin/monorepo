# bin-webhook-manager Architecture

## Component Overview

`bin-webhook-manager` manages webhook subscription configuration (CRUD) and dispatches outbound HTTP notifications to customer-configured endpoints. It is the outbound delivery side of the webhook system; `bin-hook-manager` is the inbound receive side.

```
cmd/webhook-manager/main.go
    ├── pkg/dbhandler          (MySQL + Redis cache)
    ├── pkg/cachehandler       (Redis operations; incl. per-activeflow webhook cache)
    ├── pkg/listenhandler      (RabbitMQ RPC router)
    ├── pkg/subscribehandler   (event consumer from customer-manager + flow-manager)
    ├── pkg/webhookhandler     (core webhook delivery logic)
    ├── pkg/accounthandler     (customer webhook config: URI + method)
    ├── pkg/activeflowhandler  (per-activeflow webhook resolver: cache + fallback RPC)
    └── models/                (webhook, account, activeflow, event data structures)
```

**Supporting binaries:**
- `cmd/webhook-control/` — CLI for triggering webhook deliveries

## Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|----------------|
| Transport | `pkg/listenhandler` | Receives RPC requests; routes by URI regex |
| Transport | `pkg/subscribehandler` | Consumes customer-manager events (cache invalidation) and flow-manager activeflow events (per-activeflow webhook cache) |
| Transport | notifyhandler (bin-common-handler) | Publishes `webhook_published` events |
| Domain | `pkg/webhookhandler` | Webhook delivery — resolves destination, dispatches HTTP, publishes event |
| Domain | `pkg/accounthandler` | Retrieves and caches customer webhook config (URI, method) from customer-manager |
| Domain | `pkg/activeflowhandler` | Resolves the optional per-activeflow webhook destination: Redis cache lookup, single-flight `FlowV1ActiveflowGet` fallback on miss, monotonic cache backfill |
| Data | `pkg/dbhandler` | MySQL for webhook records |
| Data | `pkg/cachehandler` | Redis cache for account webhook config and per-activeflow webhook (positive/negative tombstone, atomic monotonic writes) |

## Request Routing

ListenHandler routes over `bin-manager.webhook-manager.request`:

| Pattern | Purpose |
|---------|---------|
| `POST /v1/webhooks` (send-to-customer) | Resolve customer's saved URI/method config and dispatch webhook |
| `POST /v1/webhooks` (send-to-uri) | Dispatch to a caller-specified URI/method override |
| `/v1/webhook_destinations` | Webhook destination CRUD |

## Event Subscriptions

SubscribeHandler consumes:

| Queue | Event | Action |
|-------|-------|--------|
| `bin-manager.customer-manager.event` | `customer_updated`, `customer_deleted` | Invalidate `pkg/accounthandler` Redis cache so next dispatch uses current URI/method |
| `bin-manager.flow-manager.event` | `activeflow_created`, `activeflow_updated` | Pre-populate the per-activeflow webhook cache from the event payload (Option A: the event carries `webhook_uri` / `webhook_method`): a POSITIVE entry when `webhook_uri` is set, a NEGATIVE entry when empty, using the event timestamp as the monotonic Tm. The fallback path remains the lazy/miss safety net |
| `bin-manager.flow-manager.event` | `activeflow_deleted` | Write a negative tombstone (carrying `tm_delete`) to the per-activeflow webhook cache so a deleted destination is not resurrected |

## Events Published

Exchange: `bin-manager.webhook-manager.event`

| Event | Trigger |
|-------|---------|
| `webhook_published` | After successfully queuing a webhook for delivery (both send modes) |

## Request Flow

```
RabbitMQ RPC request
    → listenhandler (regex route)
    → webhookhandler.SendToCustomer() or SendToURI()
        → accounthandler.GetWebhookConfig()  (Redis cache → customer-manager)
        → HTTP delivery to customer endpoint
        → notifyhandler.Publish(webhook_published)
        → dbhandler.Update()
```
