# Customer Account Deletion (Unregister) Design

## Problem Statement

VoIPbin currently has no way for customers to self-service delete their account. The existing `DELETE /v1/customers/{id}` performs an immediate soft delete, which is admin-only and offers no recovery path. We need a graceful account deletion flow that protects customers from accidental data loss while preventing billing leakage.

## Approach

**Hybrid: Customer-Manager Orchestrated + Multi-Layer Enforcement**

- `bin-customer-manager` owns the account lifecycle state
- `bin-api-manager` enforces the freeze at the HTTP layer (403 DELETION_SCHEDULED)
- `bin-call-manager` enforces the freeze at the call layer (reject new calls, terminate active calls)
- `bin-billing-manager` enforces the freeze at the billing layer (block charges, pause subscriptions)
- Event-driven cascading cleanup via RabbitMQ after grace period expiry

## 1. Customer State Model

New `Status` field with three states:

```
active  ──(delete request)──>  frozen  ──(30 days expire)──>  deleted
                                  │
                                  └──(recovery request)──>  active
```

**State transitions:**
- `active` → `frozen`: Customer or admin triggers deletion with password confirmation
- `frozen` → `active`: Customer calls recovery endpoint within 30 days
- `frozen` → `deleted`: Cron job after 30-day grace period. Soft deletes all resources, anonymizes PII

**New fields on customer model:**
- `Status` (string: `active`, `frozen`, `deleted`)
- `TMDeletionScheduled` (*time.Time: when deletion was requested, used to calculate grace period expiry)

## 2. API Design

### Customer Self-Service Endpoints

**Schedule Deletion (Freeze)**
```
POST /auth/unregister
```
- **Auth**: Customer Bearer token or API key (customer identified from token, no `{id}` needed)
- **Request body**:
  ```json
  {
    "password": "user_password",
    "confirmation_phrase": "DELETE"
  }
  ```
  - `password` (optional): Required for password-based accounts.
  - `confirmation_phrase` (optional): Required for SSO users AND API key users. Must match `"DELETE"`.
  - Exactly one of `password` or `confirmation_phrase` must be provided.
- **Validation logic**:
  1. If the account has a password → `password` field required, validated against stored hash.
  2. If the account is SSO or request is authenticated via API key → `confirmation_phrase` field required, must equal `"DELETE"`.
  3. If neither field is provided, or validation fails → `400 Bad Request`.
- **Response** (200): Customer object with `status: "frozen"` and `tm_deletion_scheduled` set
- **Idempotent**: Calling again on an already-frozen account returns the current state

**Cancel Deletion (Recover)**
```
DELETE /auth/unregister
```
- **Auth**: Bearer token or API key only (no request body — avoids DELETE-with-body proxy issues)
- **Response** (200): Customer object with `status: "active"` and `tm_deletion_scheduled` cleared
- **Error**: `404` if account is not in `frozen` state

### Admin Endpoints

**Schedule Deletion (Freeze)**
```
POST /v1/customers/{id}/freeze
```
- **Auth**: Admin JWT
- **Request body**: None required (admin authority is sufficient)
- **Response** (200): Customer object with `status: "frozen"` and `tm_deletion_scheduled` set

**Cancel Deletion (Recover)**
```
POST /v1/customers/{id}/recover
```
- **Auth**: Admin JWT
- **Request body**: None required (admin authority is sufficient)
- **Response** (200): Customer object with `status: "active"` and `tm_deletion_scheduled` cleared

**Immediate Force-Delete — Updated**
```
DELETE /v1/customers/{id}
```
- Remains as admin-only immediate soft delete (bypasses grace period, for fraud/abuse cases)
- **Must also set `status='deleted'`** — the existing `CustomerDelete` DB method currently only sets `tm_delete`. After the migration adds the `status` column, this method must be updated to also set `status='deleted'` to maintain consistency between the two fields.

### Frozen Account Error Response

All non-allowed endpoints return:
```
HTTP 403
{
  "error": "DELETION_SCHEDULED",
  "message": "Account deletion scheduled",
  "deletion_scheduled_at": "2026-02-16T12:00:00Z",
  "deletion_effective_at": "2026-03-18T12:00:00Z",
  "recovery_endpoint": "DELETE /auth/unregister"
}
```

### Allowed Endpoints for Frozen Accounts

- `DELETE /auth/unregister` (self-service recovery)
- `POST /v1/customers/{id}/recover` (admin recovery)
- `GET /v1/customers/{id}` (view account status)
- Authentication/login (needed to call recovery)

Everything else returns `403 DELETION_SCHEDULED`.

## 3. Event Flow & Cascading Cleanup

### Phase 1: Immediate Freeze

```
POST /auth/unregister  (self-service)  or  POST /v1/customers/{id}/freeze  (admin)
    │
    ├─ customer-manager: set status=frozen, tm_deletion_scheduled=now()
    ├─ customer-manager: publish "customer_frozen" event via RabbitMQ
    │
    ├─ bin-call-manager: receives "customer_frozen" event
    │   ├─ Hangup all active calls (query by customer_id, call HangingUp() for each)
    │   ├─ Hangup cascades to chained calls, conferences, bridges
    │   └─ Reject any new calls for frozen customers at call setup
    │
    ├─ bin-billing-manager: receives "customer_frozen" event
    │   ├─ Set billing account status='frozen' (does NOT set tm_delete)
    │   ├─ Reject any new billing records (call charges, number fees, etc.)
    │   └─ Pause recurring subscription charges
    │
    └─ bin-api-manager: checks customer status on every request
        └─ returns 403 DELETION_SCHEDULED
```

**New event type**: `customer_frozen`

Three enforcement layers prevent billing leakage:
1. **API gateway** — blocks all new API requests
2. **call-manager** — terminates active calls and rejects new SIP calls at call setup
3. **billing-manager** — blocks charges and pauses subscriptions

### Phase 2: Recovery (customer cancels within 30 days)

```
DELETE /auth/unregister  (or)  POST /v1/customers/{id}/recover
    │
    ├─ customer-manager: set status=active, clear tm_deletion_scheduled
    ├─ customer-manager: publish "customer_recovered" event via RabbitMQ
    │
    ├─ bin-call-manager: receives "customer_recovered" event
    │   └─ Resume accepting new calls for customer
    │
    ├─ bin-billing-manager: receives "customer_recovered" event
    │   ├─ Set billing account status='active' (restore from frozen)
    │   └─ Resume recurring subscription charges
    │
    └─ All other services resume normal operation (nothing was deleted)
```

**New event type**: `customer_recovered`

Since nothing was deleted during the freeze, recovery is just flipping the status back across all layers.

### Phase 3: Expiry (30 days elapsed, no recovery)

Cron job in customer-manager runs daily:

```
Find customers WHERE status='frozen' AND tm_deletion_scheduled < now() - 30 days

For each expired customer:
    │
    ├─ Step 1: Anonymize PII
    │   ├─ name → "deleted_user_{short_id}"
    │   ├─ email → "deleted_{short_id}@removed.voipbin.net"
    │   ├─ phone_number → ""
    │   ├─ address → ""
    │   └─ webhook_uri → ""
    │
    ├─ Step 2: Set status=deleted, tm_delete=now()
    │
    └─ Step 3: Publish "customer_deleted" event (existing event type)
        │
        └─ Dependent services cascade soft-delete:
            ├─ bin-tag-manager (already implemented)
            ├─ bin-agent-manager
            ├─ bin-number-manager
            ├─ bin-flow-manager
            ├─ bin-campaign-manager
            ├─ bin-queue-manager
            ├─ bin-contact-manager
            ├─ bin-accesskey-manager
            ├─ bin-conversation-manager
            ├─ bin-call-manager
            ├─ bin-conference-manager
            ├─ bin-route-manager
            ├─ bin-storage-manager
            └─ bin-billing-manager (set status='deleted' and tm_delete on billing accounts)
```

## 4. Key Design Decisions

1. **Three enforcement layers** — API gateway (blocks HTTP), call-manager (blocks/terminates calls), billing-manager (blocks charges). Defense in depth without touching Kamailio.

2. **`customer_deleted` event reused** for Phase 3 expiry. Services that already handle this event (like tag-manager) work without changes for final cleanup.

3. **Three event types**: `customer_frozen` (new), `customer_recovered` (new), `customer_deleted` (existing). Only call-manager and billing-manager need to subscribe to the new events.

4. **Cron job is idempotent** — each customer processed once (status changes from `frozen` to `deleted`).

5. **PII anonymization instead of hard delete** — preserves referential integrity in billing/CDR tables while satisfying privacy requirements.

6. **Existing `DELETE /v1/customers/{id}` updated** — admin force-delete path preserved for fraud/abuse cases. Must also set `status='deleted'` to stay consistent with the new status field.

7. **Separate self-service (`/auth`) and admin (`/v1/customers/{id}`) paths** — self-service uses token identity (no `{id}` needed), admin specifies target customer explicitly.

8. **Call-manager handles SIP enforcement** — checks customer frozen status at call setup, covering both inbound and outbound SIP calls without Kamailio changes.

9. **Confirmation logic** — password-based accounts require password re-entry; SSO and API key users require confirmation phrase `"DELETE"` to prevent accidental script execution.

10. **Billing account gets its own `status` field** — `customer_frozen` sets billing account `status='frozen'` (not `tm_delete`). `customer_recovered` sets it back to `status='active'`. Only `customer_deleted` (Phase 3 expiry) sets `tm_delete` on billing accounts. This avoids conflating "frozen" with "deleted" — a billing account that was legitimately deleted before the freeze won't be incorrectly restored on recovery.

12. **`IsValidBalance` must check billing account `status`** — the existing balance validation only checks `tm_delete`. Since the freeze sets `status='frozen'` without setting `tm_delete`, `IsValidBalance` must be updated to also reject charges when `status` is `frozen` or `deleted`. This is the mechanism that actually enforces "reject any new billing records" in Phase 1.

13. **Existing billing `AccountDelete` must also set `status='deleted'`** — the existing `AccountDelete` DB method only sets `tm_delete`. After the migration adds the `status` column, this method must be updated to also set `status='deleted'`. This ensures consistency when `customer_deleted` fires in Phase 3 (the existing handler calls `AccountDelete` on each billing account).

11. **Migration backfills existing data** — existing customers and billing accounts with `tm_delete IS NOT NULL` must have `status='deleted'` set during migration to avoid inconsistency.

## 5. Services Impacted

| Service | Phase 1 (Freeze) | Phase 2 (Recovery) | Phase 3 (Expiry) |
|---------|-------------------|--------------------|-------------------|
| bin-customer-manager | State management, cron job, PII anonymization | Restore status | Publish customer_deleted |
| bin-api-manager | New /auth endpoints, 403 check, admin endpoints | No change | No change |
| bin-call-manager | Subscribe customer_frozen, hangup active calls, reject new calls | Subscribe customer_recovered, resume calls | Subscribe customer_deleted (existing) |
| bin-billing-manager | Subscribe customer_frozen, set account status='frozen', update IsValidBalance to check status | Subscribe customer_recovered, set account status='active' | Subscribe customer_deleted, set status='deleted' and tm_delete |
| bin-openapi-manager | New endpoint schemas, new fields | No change | No change |
| bin-dbscheme-manager | Migration: add status, tm_deletion_scheduled to customers; add status to billing_accounts; backfill existing deleted rows | No change | No change |
| All other services | No change | No change | Subscribe customer_deleted for cascade cleanup |

## 6. Database Migration

### Customer table

```sql
ALTER TABLE customer_customers
  ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT 'active',
  ADD COLUMN tm_deletion_scheduled DATETIME(6) DEFAULT NULL;

-- Backfill: existing soft-deleted customers must have status='deleted'
UPDATE customer_customers SET status = 'deleted' WHERE tm_delete IS NOT NULL;
```

### Billing account table

```sql
ALTER TABLE billing_accounts
  ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT 'active';

-- Backfill: existing soft-deleted billing accounts must have status='deleted'
UPDATE billing_accounts SET status = 'deleted' WHERE tm_delete IS NOT NULL;
```
