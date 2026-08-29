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

SubscribeHandler consumes from the global topic exchange `bin-manager.event` (VOIP-1406):
the subscribe queue is bound with one pattern per handled event pair
(`pkg/subscribehandler` `topicPatterns`, pinned byte-for-byte by
`pkg/subscribehandler/binding_golden_test.go`):

| Binding pattern | Event | Action |
|-----------------|-------|--------|
| `customer-manager.customer.*.created` | `customer_created` | Refresh `pkg/accounthandler` Redis cache so next dispatch uses current URI/method |
| `customer-manager.customer.*.updated` | `customer_updated` | Refresh `pkg/accounthandler` Redis cache so next dispatch uses current URI/method |
| `flow-manager.activeflow.*.created` | `activeflow_created` | Pre-populate the per-activeflow webhook cache from the event payload (Option A: the event carries `webhook_uri` / `webhook_method`): a POSITIVE entry when `webhook_uri` is set, a NEGATIVE entry when empty, using the event timestamp as the monotonic Tm. The fallback path remains the lazy/miss safety net |
| `flow-manager.activeflow.*.updated` | `activeflow_updated` | Same per-activeflow cache pre-population as `activeflow_created` |
| `flow-manager.activeflow.*.deleted` | `activeflow_deleted` | Write a negative tombstone (carrying `tm_delete`) to the per-activeflow webhook cache so a deleted destination is not resurrected |

As of VOIP-1407, this topic-pattern binding is the **sole intake mechanism** — the old
per-service fanout `QueueSubscribe` calls (to `bin-manager.customer-manager.event` and
`bin-manager.flow-manager.event`) and the fanout-unbind step that used to follow a
successful topic bind have both been removed. A declare or bind failure is now fatal;
there is no fanout fallback left to degrade to.

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
