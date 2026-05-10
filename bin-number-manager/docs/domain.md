# bin-number-manager — Domain

## Domain Entities

### Number

A purchased phone number owned by a customer, with routing configuration for inbound calls and messages.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owning customer |
| `number` | string | E.164 formatted phone number (e.g. `+12025551234`) |
| `name` | string | Human-readable label |
| `detail` | string | Description |
| `status` | string | `active`, `inactive` |
| `call_flow_id` | UUID | Flow to invoke when an inbound call arrives on this number |
| `message_flow_id` | UUID | Flow to invoke when an inbound SMS arrives on this number |
| `provider_name` | string | `telnyx` or `twilio` |
| `tm_delete` | timestamp | Soft-delete sentinel (`9999-01-01` = active) |

### AvailableNumber

A phone number available for purchase from a provider. Returned by search queries; not persisted.

| Field | Type | Description |
|-------|------|-------------|
| `number` | string | E.164 phone number |
| `country_code` | string | ISO country code (e.g. `US`, `GB`) |
| `provider_name` | string | Source provider |

### ProviderNumber

Internal mapping between a Number (by UUID) and the provider's own reference for the number. Used for release operations.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `number_id` | UUID | References `numbers.id` |
| `provider_name` | string | `telnyx` or `twilio` |
| `provider_number_id` | string | Provider's internal ID for the number |

## Key Business Rules

### Provider Strategy

Numbers are purchased from either Telnyx or Twilio. The provider is selected at purchase time based on configuration. Both providers implement the same interface:
- `Purchase(ctx, number, customerID)` — buy the number from the provider
- `Release(ctx, providerNumberID)` — release number back to provider
- `ListAvailable(ctx, countryCode, limit)` — search for purchasable numbers

### Number Lifecycle

```
[available at provider]
        ↓ POST /v1/numbers
    active (stored in DB, ProviderNumber record created)
        ↓ PUT /v1/numbers/{uuid} with status=inactive
    inactive
        ↓ DELETE /v1/numbers/{uuid}
    [released to provider] (soft-deleted in DB)
```

### Billing Integration

Before purchasing a number, `numberhandler` validates that the customer has sufficient balance via RPC to `bin-billing-manager`. After successful purchase, it publishes a `number.EventTypeNumberCreated` event so billing-manager can charge the account.

On renewal (`POST /v1/numbers/renew`), the same balance check and charge cycle repeats.

### Flow Routing

- `call_flow_id`: when an inbound call arrives on this number (via Kamailio/SIP), the call-manager uses this flow ID to route the call.
- `message_flow_id`: when an inbound SMS arrives, message routing uses this flow ID.
- Both can be updated independently via `PUT /v1/numbers/{uuid}/flow_ids`.

### Cascading Deletions (SubscribeHandler)

The `subscribehandler` consumes events to maintain referential integrity:

| Queue | Event | Action |
|-------|-------|--------|
| `bin-manager.customer-manager.event` | `customer_deleted` | Release all numbers owned by the customer |
| `bin-manager.flow-manager.event` | `flow_deleted` | Clear `call_flow_id` / `message_flow_id` on affected numbers |

### Soft Deletes

Numbers and ProviderNumbers use `tm_delete` for soft deletes. Default active sentinel: `9999-01-01 00:00:00.000000`. Released numbers are soft-deleted; the provider's own number is released via API before the soft-delete.

### Cache Strategy

Redis caches number lookups by UUID. Mutations invalidate the relevant Redis keys. Cache-first for reads; MySQL is source of truth.

### Event Publishing

| Event | Trigger |
|-------|---------|
| `number.EventTypeNumberCreated` | Number purchased successfully |
| `number.EventTypeNumberDeleted` | Number released/deleted |

These events are consumed by `bin-billing-manager` to apply charges.
