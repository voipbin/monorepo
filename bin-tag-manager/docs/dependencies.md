# bin-tag-manager Dependencies

## Upstream Services (consumed via RabbitMQ RPC)

| Service | Purpose |
|---------|---------|
| `bin-billing-manager` | Billing checks for tag operations |
| `bin-customer-manager` | Customer validation; source of `customer_deleted` events |

## Events Subscribed

| Queue | Event | Handler |
|-------|-------|---------|
| `bin-manager.customer-manager.event` | `customer_deleted` | Delete all tags for the customer; emit `tag_deleted` per tag |

## Events Published

| Queue | Events |
|-------|--------|
| `bin-manager.tag-manager.event` | `tag_created`, `tag_updated`, `tag_deleted` |

## Infrastructure Dependencies

| Dependency | Use |
|-----------|-----|
| RabbitMQ | RPC request queue and event pub/sub |
| MySQL | Durable tag records with soft-delete |
| Redis | Tag cache; invalidated on every mutation |

## Monorepo Module Dependencies

Key local imports:
- `monorepo/bin-common-handler` — sockhandler, requesthandler, notifyhandler
- `monorepo/bin-customer-manager` — customer event model for SubscribeHandler

## Reverse Dependencies

Services that use tags to categorize their own resources:
- `bin-contact-manager` — tags applied to contacts
- `bin-queue-manager` — tags applied to queues
- `bin-campaign-manager` — tags applied to campaigns
- Any service that calls `POST /v1/tags` or reads tags via the RPC API
