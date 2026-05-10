# bin-webhook-manager Domain Model

## Core Concepts

### Webhook
An outbound HTTP notification dispatched to a customer-configured URI when a VoIPbin event occurs.

Key fields:
- `customer_id` — owning customer
- `uri` — destination URL
- `method` — HTTP method (POST, PUT, etc.)
- `data` — event payload (JSON)
- `data_type` — MIME type of payload (e.g. `application/json`)

### Customer Webhook Configuration
Each customer has a saved `webhook_method` and `webhook_uri` in `bin-customer-manager`. `bin-webhook-manager` reads this configuration via `pkg/accounthandler` and caches it in Redis.

### Webhook Destination
A named endpoint configuration that can be associated with webhook subscriptions.

## Delivery Modes

### SendWebhookToCustomer
Resolves the customer's saved webhook URI and method from `pkg/accounthandler`, then dispatches.

Use case: standard event notification where the customer's configured webhook endpoint receives all events.

### SendWebhookToURI
Caller provides the destination URI and method directly, bypassing the customer's saved config.

Use case: flow-manager or other internal services that need to deliver to a specific override endpoint.

Both modes publish a `webhook_published` event after dispatch.

## Cache Invalidation

`pkg/accounthandler` caches customer webhook configs in Redis to avoid repeated RPC calls to `bin-customer-manager`. The cache must be invalidated when:
- `customer_updated` event received — customer may have changed their webhook URI/method
- `customer_deleted` event received — customer no longer exists

Without invalidation, webhook dispatches continue to stale endpoints until cache TTL expires. The `pkg/subscribehandler` drives this invalidation.
