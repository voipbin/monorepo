# bin-customer-manager — Domain

## Domain Entities

### Customer

The top-level tenant organization in VoIPbin. All resources in the platform are scoped to a customer.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `email` | string | Unique login email |
| `name` | string | Display name |
| `detail` | string | Description |
| `phone_number` | string | Contact phone |
| `address` | string | Mailing address |
| `webhook_method` | string | HTTP method for webhook delivery (GET/POST/PUT/DELETE) |
| `webhook_uri` | string | URL for outbound webhook events |
| `billing_account_id` | UUID | Linked billing account in bin-billing-manager |
| `tm_delete` | timestamp | Soft-delete sentinel (`9999-01-01` = active) |

### AccessKey

API credentials scoped to a customer. Used by `bin-api-manager` to authenticate inbound API requests.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key (also the token value) |
| `customer_id` | UUID | Owning customer |
| `name` | string | Human-readable label |
| `detail` | string | Description |
| `expire` | duration | TTL; `0` means non-expiring |
| `tm_delete` | timestamp | Soft-delete sentinel |

## Key Business Rules

### Email Uniqueness

Customer email must be unique across all active customers. The `customerhandler.Create()` method checks for conflicts before inserting.

### Username Conflict Check

During customer creation, `customerhandler` makes an RPC call to `bin-agent-manager` to ensure the customer email does not conflict with an existing agent username. If the agent-manager is unavailable, creation fails.

### Billing Account Linkage

A customer may be created without a billing account. The `PUT /v1/customers/{uuid}/billing_account_id` endpoint links the billing account after the fact. This is typically done by the signup flow immediately after billing-manager creates the account.

### Email Verification

The `POST /v1/customers/signup` flow:
1. Creates customer with `status = unverified`
2. Sends verification email to `email_verify_base_url` (configured via flag)
3. `POST /v1/customers/email_verify` confirms the token and marks customer `status = active`

### Unverified Signup Lifecycle

A signup that never verifies its email address follows this path, driven by the `cleanup_unverified` sweep (every 15 minutes):

1. `initial` for 72 hours (`unverifiedMaxAge`), the window in which the verification link can be used or re-sent.
2. `expired`. The row is kept live (no `tm_delete`). The customer can still recover with `POST /v1/customers/email_verify_resend` and the new link.
3. After `expiredRetention` (30 days, counted from the last update of the row, which is normally the expiry time; any later update of the row postpones the purge) the purge step anonymizes the row in one UPDATE: name, email, phone number, address and webhook URI are replaced or cleared, `status = deleted`, `tm_delete` and `tm_update` are set. The row is not removed. `customer_deleted` is published once, built from the listed row with the anonymized values, and the other services clean up their data for the customer. After this the email address can be used for a new signup and an old verification link is rejected.

The purge only selects `status = expired AND email_verified = false AND tm_delete IS NULL`, and the UPDATE repeats the same condition, so a customer that verified after the list query, a legacy expired row that already has `tm_delete`, and a row claimed by another replica are skipped without an event. Up to 100 customers are purged per run; a row whose anonymize fails with an unexpected error is logged and retried on the next run. The five anonymized fields are the same set that `CleanupFrozenExpired` anonymizes.

### Freeze / Recover Lifecycle

Customers can be frozen (preventing resource creation) and recovered:
- `freeze` — sets customer status to `frozen`; blocks new resource creation across all services
- `recover` — restores customer to `active`; refused for a frozen customer whose email is not verified
- `freeze_and_delete` — freezes and immediately anonymizes and deletes (used for account closure); also accepts an unverified (`initial` or `expired`) customer without `tm_delete`, whereas `freeze` accepts `active` only

### Soft Deletes

Active records have `tm_delete = "9999-01-01 00:00:00.000000"`. Deletion sets `tm_delete` to the current timestamp. Queries always filter by `tm_delete = DefaultTimestamp`.

### Cache Strategy

Cache-first pattern for all reads:
1. Check Redis by entity ID
2. On miss, query MySQL
3. Write-through on mutations; cache invalidated immediately

### Pagination

Cursor-based pagination using `tm_create` as the page token. Query pattern:
```sql
WHERE tm_create < :page_token ORDER BY tm_create DESC LIMIT :page_size
```

### Webhook Publishing

All customer and access key mutations publish events to exchange `bin-manager.customer-manager.event`:
- `customer_created`, `customer_updated`, `customer_deleted`
- `accesskey_created`, `accesskey_updated`, `accesskey_deleted`

These events are consumed by `bin-number-manager` (cascading deletions) and `bin-billing-manager` (account lifecycle).

### Cross-Service Impact

Because `customer_id` is the primary isolation boundary in VoIPbin, a customer deletion cascades through:
- `bin-number-manager`: releases all phone numbers for the customer
- `bin-billing-manager`: soft-deletes the billing account
- All other services clean up their resources on `customer_deleted` events
