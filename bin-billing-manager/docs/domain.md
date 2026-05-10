# bin-billing-manager — Domain

## Domain Entities

### Account

Represents the financial account for a customer. One account per customer.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owning customer |
| `type` | string | `admin` (unlimited) or `normal` |
| `plan_type` | string | `unlimited` bypasses balance checks; default is metered |
| `balance` | float64 | Current balance in USD |
| `payment_type` | string | `prepaid` or empty |
| `payment_method` | string | `credit card` or empty |
| `tm_delete` | timestamp | Soft-delete sentinel (`9999-01-01` = active) |

### Billing

A billing record created for each billable event. Immutable after finalization.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owning customer |
| `account_id` | UUID | Associated billing account |
| `reference_id` | UUID | ID of the resource billed (call, message, number) |
| `reference_type` | string | `call`, `sms`, `number`, `number_renew` |
| `status` | string | `progressing`, `end`, `pending`, `finished` |
| `cost_per_unit` | float64 | Rate in USD per billing unit |
| `billing_unit_count` | int | Number of units consumed |
| `cost_total` | float64 | `cost_per_unit × billing_unit_count` |
| `tm_delete` | timestamp | Soft-delete sentinel |

Default unit costs (defined in `models/billing/billing.go`):
- Call: $0.020 per minute
- SMS: $0.008 per message
- Number: $5.00 per number purchase
- Number renew: $5.00 per renewal

### FailedEvent

Persisted retry queue for billing operations that could not be applied downstream. Hard-deleted (not soft-deleted) after successful retry.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `status` | string | `pending`, `processing`, `failed` |
| `data` | JSON | Serialized billing event payload |
| `retry_count` | int | Number of retry attempts |

## Key Business Rules

### Unlimited-Plan Bypass

Accounts with `plan_type = unlimited` always pass balance checks:
```
if account.PlanType == unlimited → IsValidBalance = true (skip deduction)
```
Admin-type accounts inherit this behavior implicitly.

### Balance Deduction Order

For metered accounts, balance is deducted only after the billable event concludes:
1. `call_progressing` → create billing record with status `progressing` (no deduction yet)
2. `call_hangup` → update billing record with final duration → deduct from account balance

For number purchases and SMS, deduction is immediate on event receipt.

### Billing Event Processing

The `subscribehandler` triggers billing for events from:

| Source queue | Event | Action |
|-------------|-------|--------|
| `bin-manager.call-manager.event` | `call_progressing` | Create billing record (`progressing`) |
| `bin-manager.call-manager.event` | `call_hangup` | Finalize billing record, deduct balance |
| `bin-manager.message-manager.event` | `message_created` | Create billing record, deduct balance |
| `bin-manager.number-manager.event` | `number_created` | Create billing record, charge number cost |
| `bin-manager.number-manager.event` | `number_renewed` | Create billing record, charge renewal cost |
| `bin-manager.customer-manager.event` | `customer_created` | Create billing account |
| `bin-manager.customer-manager.event` | `customer_deleted` | Soft-delete billing account |
| `bin-manager.tts-manager.event` | TTS events | Create billing record for TTS usage |
| `bin-manager.email-manager.event` | Email events | Create billing record for email usage |

### Paddle Integration

Paddle webhooks drive subscription lifecycle:
- Transaction completed → top-up account balance
- Subscription activated/upgraded → set plan type on account
- Subscription cancelled → revert plan type to metered

Paddle webhook handler logs follow the **External Event & Webhook Processing Logs** convention (Info level for receipt, processing start, and success; Error on failure) with fields: `event_id`, `transaction_id`, `subscription_id`, `customer_id`, `plan_type`, `amount_micros`, `token_allowance`.

### Access Control

Authorization is enforced by `bin-api-manager`, not by this service. Only users with `CustomerAdmin` permission can access billing/account endpoints. Manager-level users do not have access.

### Soft Deletes

Both `billing_accounts` and `billing_billings` tables use `tm_delete` for soft deletes. Default active sentinel: `9999-01-01 00:00:00.000000`.

### Cache Strategy

Redis caches account lookups. The database is the source of truth. Cache is invalidated on all mutations (balance changes, payment info updates).
