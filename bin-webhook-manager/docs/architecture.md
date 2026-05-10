# bin-webhook-manager Architecture

## Component Overview

`bin-webhook-manager` manages webhook subscription configuration (CRUD) and dispatches outbound HTTP notifications to customer-configured endpoints. It is the outbound delivery side of the webhook system; `bin-hook-manager` is the inbound receive side.

```
cmd/webhook-manager/main.go
    ├── pkg/dbhandler          (MySQL + Redis cache)
    ├── pkg/cachehandler       (Redis operations)
    ├── pkg/listenhandler      (RabbitMQ RPC router)
    ├── pkg/subscribehandler   (event consumer from customer-manager)
    ├── pkg/webhookhandler     (core webhook delivery logic)
    ├── pkg/accounthandler     (customer webhook config: URI + method)
    └── models/                (webhook, account, event data structures)
```

**Supporting binaries:**
- `cmd/webhook-control/` — CLI for triggering webhook deliveries

## Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|----------------|
| Transport | `pkg/listenhandler` | Receives RPC requests; routes by URI regex |
| Transport | `pkg/subscribehandler` | Consumes customer-manager events to invalidate cache |
| Transport | notifyhandler (bin-common-handler) | Publishes `webhook_published` events |
| Domain | `pkg/webhookhandler` | Webhook delivery — resolves destination, dispatches HTTP, publishes event |
| Domain | `pkg/accounthandler` | Retrieves and caches customer webhook config (URI, method) from customer-manager |
| Data | `pkg/dbhandler` | MySQL for webhook records |
| Data | `pkg/cachehandler` | Redis cache for account webhook configuration |

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
