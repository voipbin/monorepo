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

The code knows a fixed set of providers (`providerhandler` registry: `telnyx`, `twilio`). A provider has a row in `number_providers` only after an admin changed it; without a row it uses the registry defaults (telnyx: enabled, priority 10; twilio: disabled, priority 20, and it cannot purchase) and the environment credentials. Adding a provider means adding a registry entry and an implementation of the provider interface (search, purchase, release, tag update); no migration is needed.

Purchase (`POST /v1/numbers`) after the balance check:
1. The usable providers that can purchase are tried in ascending `priority` (ties by name). A provider is usable when it is enabled and has a secret; for a DB secret the required `config` keys must exist too.
2. Fallback to the next provider happens only when the provider definitively refused the order (`ErrPurchaseRejected`: for Telnyx HTTP 400, 401, 403, 404, 422 or an order whose status is `failure`). Any other failure (network error, timeout, 408, 409, 429, 5xx, a failed lookup after the order) may have created an order, so the purchase stops with `ErrPurchaseUnconfirmed` and the log carries `PURCHASE_UNCONFIRMED`. An operator checks the provider, and may disable the provider to route around it.
3. A fallback only helps when the next provider holds the same number; the available number search is Telnyx only.
4. The number is recorded with the provider that sold it. If the registration fails, the number is released at that provider (`ROLLBACK_RELEASE_FAILED` in the log when the release fails too).

Release (`DELETE /v1/numbers/{uuid}`, renewal without balance, customer deletion) uses the number's own `provider_name`, whether or not that provider is enabled. A provider that has no usable credential makes the release fail and keeps the record. During a renewal run, a number that could not be released is left out of the rest of that run.

### Provider Settings (`number_providers`)

One row per provider that an admin changed: `provider_name` (key), `enabled`, `priority` (lower is tried first), `config` (non-secret values such as the Telnyx `connection_id` and `profile_id`), `secret_ciphertext` with its `key_version` (AES-256-GCM, never returned), and the audit columns `updated_by`, `tm_update`, `secret_updated_by`, `tm_secret_update`. The admin view adds `has_secret`, `credential_source` (`db`, `env`, `none`, `db_undecryptable`) and `usable`. The row `config` is used only while the row holds a secret; with the environment credentials it is ignored.

The public admin API (`/number_providers`) returns `provider.WebhookMessage` (`models/provider/webhook.go`), produced by `Info.ConvertWebhookMessage()`: the same fields as the admin view, never a secret or a ciphertext.

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
