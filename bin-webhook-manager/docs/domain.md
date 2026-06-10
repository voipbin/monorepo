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

### Per-activeflow additive delivery

In addition to the customer-level destination, an activeflow may declare its OWN webhook destination (`webhook_uri` / `webhook_method`, set at activeflow creation in `bin-flow-manager`, immutable). When `SendWebhookToCustomer` handles an event whose nested payload carries an `activeflow_id`, it ADDITIONALLY resolves the per-activeflow destination via `pkg/activeflowhandler` and delivers there too. This is additive: the customer delivery always happens; the per-activeflow delivery is an extra fan-out and never replaces it. A failure to resolve the per-activeflow destination (Redis down, RPC error) only skips the extra delivery; the customer delivery is unaffected.

The per-activeflow `webhook_uri` is the customer's OWN data, not a cross-tenant secret. It IS carried on flow-manager lifecycle events (`activeflow_created` / `activeflow_updated` / `activeflow_deleted`) and is a documented `GET /activeflows` response field; it can also be fetched over the internal `FlowV1ActiveflowGet` RPC (used by the fallback path). Customers MUST NOT embed secrets or tokens in the `webhook_uri` query string, since it is echoed in the activeflow lifecycle webhook payloads delivered to the customer's own webhook endpoint.

### Per-activeflow webhook cache (`pkg/cachehandler` + `pkg/activeflowhandler`)

The per-activeflow destination is cached in Redis under `webhook:activeflow:{id}` as a single entry that is either:
- **Positive** — a real destination (`uri` set, not deleted); TTL `T_live` (default 24h, a safety net only).
- **Negative tombstone** — no destination configured, the activeflow is deleted, or a transient miss; TTL `T_neg` (default 10m) or `T_transient` (default 5s) for a NotFound. A negative entry prevents a fallback RPC storm for activeflows that do not use a per-activeflow webhook.

**Cache population**: lifecycle events carry `webhook_uri` (Option A), so the cache is pre-populated on `activeflow_created` / `activeflow_updated` directly from the event payload: a POSITIVE entry when `webhook_uri` is set, a NEGATIVE entry when empty, using the event timestamp as the monotonic Tm. The fallback path is the lazy/miss safety net: on a resource event that references an activeflow not yet cached, `activeflowhandler.Get` misses, then a single-flight-coalesced `FlowV1ActiveflowGet` fallback fetches the full Activeflow and backfills the cache (positive or negative per the result). Concurrent misses for the same id issue exactly one fallback RPC.

**Atomic monotonic writes (resurrection guard)**: events for the same activeflow can be processed out of order (per-event goroutines, no RabbitMQ ordering guarantee), so a late `created` must not resurrect a deleted destination. Each entry stores its source `tm` (unix-nano). Cache writes use a Redis Lua compare-and-set that reads the stored `tm` and only overwrites when the incoming `tm` is not older (single round trip, no read-modify-write race). `activeflow_deleted` writes a negative tombstone carrying `tm_delete` rather than a bare delete.

## Cache Invalidation

`pkg/accounthandler` caches customer webhook configs in Redis to avoid repeated RPC calls to `bin-customer-manager`. The cache must be invalidated when:
- `customer_updated` event received — customer may have changed their webhook URI/method
- `customer_deleted` event received — customer no longer exists

Without invalidation, webhook dispatches continue to stale endpoints until cache TTL expires. The `pkg/subscribehandler` drives this invalidation.
