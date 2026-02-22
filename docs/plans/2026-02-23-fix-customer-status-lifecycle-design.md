# Fix Customer Status Lifecycle

**Date:** 2026-02-23

## Problem

Customer unregistration (`POST /auth/unregister`) fails with "Bad Request" because the `CustomerFreeze` DB query requires `WHERE status = 'active'`, but customers are created with `status = ''` (empty string).

### Root Cause

`customerhandler.Create()` and `customerhandler.Signup()` build a `Customer` struct without setting the `Status` field. The `PrepareFields` function includes ALL struct fields in the INSERT, so `status = ''` is explicitly inserted — overriding the MySQL column DEFAULT `'active'`. Every customer in the DB has `status = ''`.

### Impact

- `CustomerFreeze` always fails (requires `status = 'active'`, finds 0 rows)
- Customer unregistration is completely broken
- No status-based lifecycle tracking exists

## Design

### Status Lifecycle

```
Admin Create → "active" (immediately usable)

Signup → "initial" → (email verification) → "active" → (freeze/unregister) → "frozen" → (30-day expiry) → "deleted" (anonymized)
                  ↓ (1hr cleanup)
              "expired" (soft-deleted, tm_delete set)
```

### New Status Constants

Add to `bin-customer-manager/models/customer/customer.go`:
- `StatusInitial Status = "initial"` — just signed up, awaiting email verification
- `StatusExpired Status = "expired"` — verification window expired, soft-deleted

### Changes

#### 1. Set status during admin creation
**File:** `bin-customer-manager/pkg/customerhandler/db.go`
- In `Create()`: set `Status: customer.StatusActive` in the customer struct literal

#### 2. Set status during signup
**File:** `bin-customer-manager/pkg/customerhandler/signup.go`
- In `Signup()`: set `Status: customer.StatusInitial` in the customer struct literal

#### 3. Transition to "active" during email verification
**File:** `bin-customer-manager/pkg/customerhandler/signup.go`
- In `EmailVerify()`: add `customer.FieldStatus: string(customer.StatusActive)` to the update fields map
- In `CompleteSignup()`: add `customer.FieldStatus: string(customer.StatusActive)` to the update fields map

#### 4. Change cleanup to soft delete
**File:** `bin-customer-manager/pkg/customerhandler/cleanup.go`
- Replace `CustomerHardDelete` with `CustomerUpdate` setting:
  - `customer.FieldStatus: string(customer.StatusExpired)`
  - `customer.FieldTMDelete: now`
- Expired customers have `tm_delete` set, so `validateCreate` (which filters `tm_delete IS NULL`) allows re-signup with the same email

#### 5. DB migration for existing data
**File:** `bin-dbscheme-manager/bin-manager/main/versions/` (new migration)
- `UPDATE customer_customers SET status = 'active' WHERE status = '' AND email_verified = 1`
- `UPDATE customer_customers SET status = 'initial' WHERE status = '' AND email_verified = 0`

#### 6. Update tests
- Update freeze tests to use correct status setup
- Add tests for status transitions during verification
- Add tests for cleanup soft delete behavior

### What Doesn't Change

- **Freeze logic** (`freeze.go`): Already correctly requires `status = 'active'`. Once verification sets status to "active", freeze works.
- **Frozen expiry** (`expiry.go`): Queries for `status = 'frozen'`, unaffected by these changes.
- **Email uniqueness validation** (`validateCreate`): Uses `tm_delete IS NULL` filter, so expired/soft-deleted customers are excluded and re-signup works.
- **`CustomerHardDelete` DB method**: Kept for potential future use, but cleanup switches to soft delete.

### Edge Cases

- **Admin-created customers** (`POST /v1/customers`): Go directly to "active" with `EmailVerified: true`. No verification needed.
- **Unverified customers after 1hr**: Cleanup job soft-deletes with `status = "expired"` + `tm_delete = now`. Email becomes available for re-signup.
- **Concurrent freeze**: Existing race condition handling in `freeze.go` remains correct.
- **Already-verified check**: Both `EmailVerify()` and `CompleteSignup()` have idempotency guards that return early if already verified. These are unaffected.
