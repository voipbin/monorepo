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

### Freeze / Recover Lifecycle

Customers can be frozen (preventing resource creation) and recovered:
- `freeze` — sets customer status to `frozen`; blocks new resource creation across all services
- `recover` — restores customer to `active`
- `freeze_and_delete` — freezes and schedules eventual hard deletion (used for GDPR/account closure)

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
