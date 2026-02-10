# Enforce 1:1 Billing Account per Customer

## Problem Statement

Currently, customers can have multiple billing accounts (1:many relationship). Since credits (`balance`) and resource limits (`plan_type`) are stored on the billing account, creating additional accounts results in:

- New accounts starting with 0 balance
- Potential confusion about which account's plan governs resource limits
- Unnecessary complexity in a feature no customer has ever used

## Decision

Enforce a 1:1 relationship between customer and billing account by removing the API endpoints that allow manual creation, listing, and deletion of billing accounts. Internal auto-creation (on customer signup) and auto-deletion (on customer deletion) remain unchanged.

## Scope

### Remove (API-facing only)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v1/billing_accounts` | `POST` | Create billing account |
| `/v1/billing_accounts` | `GET` | List billing accounts |
| `/v1/billing_accounts/{id}` | `DELETE` | Delete billing account |

### Keep unchanged

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v1/billing_accounts/{id}` | `GET` | Get single billing account |
| `/v1/billing_accounts/{id}` | `PUT` | Update billing account info |
| `/v1/billing_accounts/{id}/payment_info` | `PUT` | Update payment info |
| `/v1/billing_accounts/{id}/balance_add_force` | `POST` | Add balance (admin) |
| `/v1/billing_accounts/{id}/balance_subtract_force` | `POST` | Subtract balance (admin) |

### Internal logic (unchanged)

- Auto-creation on customer signup event (`bin-billing-manager/pkg/subscribehandler/customer.go`)
- Auto-deletion on customer deletion event (same file)
- Internal `accountHandler.Create()`, `Delete()`, and `List()` methods remain for event-driven use

## Files to Change

### bin-openapi-manager

- `openapi/paths/billing_accounts/main.yaml` - Remove `POST` and `GET` operations from `/v1/billing_accounts`
- `openapi/paths/billing_accounts/{billing_account_id}/main.yaml` - Remove `DELETE` operation
- `openapi/openapi.yaml` - Remove create request schema if defined inline
- Regenerate: `go generate ./...`

### bin-api-manager

- `pkg/servicehandler/billingaccount.go` - Remove `BillingAccountCreate`, `BillingAccountGets`/`BillingAccountList`, `BillingAccountDelete` methods
- Route registration files - Remove route entries for the 3 endpoints
- Regenerate server code: `go generate ./...`

### bin-billing-manager

- Remove RPC handler methods that serve create/list/delete requests from bin-api-manager
- Keep internal `accountHandler.Create()`, `Delete()`, and `List()` (used by event handlers)

## Migration

**No data migration needed.** No customer has ever created a second billing account. Every customer already has exactly 1:1.

**No database schema change needed.** The 1:1 enforcement is at the API layer. A database unique constraint on `customer_id` could be added in a future iteration for defense-in-depth.

## Backward Compatibility

Clients calling the removed endpoints will receive 404/405 responses. Since these endpoints were never used by customers, this is a non-breaking change in practice.

## What This Does NOT Change

- The `billing_account_id` field on the customer model
- The billing account's `customer_id` field
- Balance operations, plan types, resource limits
- Internal RPC between services that read billing accounts
