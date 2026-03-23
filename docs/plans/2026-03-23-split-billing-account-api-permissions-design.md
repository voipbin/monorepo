# Split Billing Account API Permissions

## Problem

Currently, customers access their billing account via `/billing_accounts/{id}` with `PermissionCustomerAdmin`. This requires customers to know their billing account ID and uses the same endpoint prefix as admin operations. We want to split the API into two tiers following the existing `/customer` vs `/customers` pattern.

## Approach

Introduce a singular `/billing_account` endpoint for customer admin users that auto-resolves the billing account from the authenticated agent's customer record. Restrict the plural `/billing_accounts` endpoints to project super admin only. Add a new `GET /billing_accounts` list endpoint for project admins.

## API Surface

### Singular endpoints (`/billing_account`) — PermissionCustomerAdmin

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/billing_account` | Get caller's own billing account |
| PUT | `/billing_account` | Update name/detail |
| PUT | `/billing_account/payment_info` | Update payment type/method |

### Plural endpoints (`/billing_accounts`) — PermissionProjectSuperAdmin

| Method | Path | Change |
|--------|------|--------|
| GET | `/billing_accounts` | NEW — list all billing accounts |
| GET | `/billing_accounts/{id}` | Permission tightened to project admin only |
| PUT | `/billing_accounts/{id}` | Permission tightened to project admin only |
| PUT | `/billing_accounts/{id}/payment_info` | Permission tightened to project admin only |
| POST | `/billing_accounts/{id}/balance_add_force` | Unchanged (already project admin only) |
| POST | `/billing_accounts/{id}/balance_subtract_force` | Unchanged (already project admin only) |

## Resolution for Singular Endpoint

1. Get customer: `reqHandler.CustomerV1CustomerGet(ctx, agent.CustomerID)`
2. Extract: `customer.BillingAccountID`
3. Get billing account: `reqHandler.BillingV1AccountGet(ctx, customer.BillingAccountID)`
4. Return: `billingAccount.ConvertWebhookMessage()`

Error cases:
- Customer has no billing account (nil/zero BillingAccountID) → 404
- Customer not found → 404

## Files to Change

### bin-api-manager (HTTP layer)
- `server/billing_account.go` — NEW — singular endpoint handlers (GET, PUT, PUT payment_info)
- `server/billing_accounts.go` — MODIFY — add `GetBillingAccounts()` list handler
- `pkg/servicehandler/billingaccount.go` — MODIFY — add `BillingAccountSelfGet()`, `BillingAccountSelfUpdateBasicInfo()`, `BillingAccountSelfUpdatePaymentInfo()`, `BillingAccountList()`. Update existing methods to require project admin only.

### bin-openapi-manager (API spec)
- `openapi/paths/billing_account/main.yaml` — NEW — singular endpoint definitions (uses `BillingManagerAccount` schema)
- `openapi/paths/billing_account/payment_info.yaml` — NEW — singular payment_info endpoint
- `openapi/paths/billing_accounts/main.yaml` — NEW — list endpoint definition (uses `BillingManagerAccountAdmin` schema)
- `openapi/openapi.yaml` — MODIFY — add new path references, add `BillingManagerAccountAdmin` schema and `BillingManagerAccountStatus` enum

### bin-common-handler (shared RPC)
- `pkg/requesthandler/billing_accounts.go` — MODIFY — add `BillingV1AccountGets()` RPC method

### bin-billing-manager (billing service)
- `pkg/listenhandler/v1_accounts.go` — MODIFY — add `processV1AccountsGet()` list handler

### bin-api-manager (RST documentation)
- `docsdev/source/billing_account_overview.rst` — UPDATE — document new `/billing_account` endpoints for customer admins
- Rebuild HTML after RST changes: `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build`
- Force-add build output: `git add -f bin-api-manager/docsdev/build/`

### No changes needed
- `bin-billing-manager/pkg/dbhandler/account.go` — `AccountList()` already exists
- Database schema — no migrations
- Force balance endpoints — already project-admin-only
- Other services — none call these endpoints

## Response Model Changes

The singular `/billing_account` endpoints (customer-facing) return `WebhookMessage` — which includes Paddle IDs (intentionally exposed for customer reference). These use the existing `BillingManagerAccount` OpenAPI schema.

The plural `/billing_accounts` endpoints (project-admin-only) return the raw internal `Account` model — exposing all fields including `status` and Paddle IDs, since admins need full visibility. This means the existing `BillingAccountGet`, `BillingAccountUpdateBasicInfo`, and `BillingAccountUpdatePaymentInfo` methods must be changed to return `*bmaccount.Account` instead of `*bmaccount.WebhookMessage` (remove `.ConvertWebhookMessage()` calls).

### New OpenAPI Schema: `BillingManagerAccountAdmin`

A new `BillingManagerAccountAdmin` schema is needed for the plural (admin) endpoints. It contains all `BillingManagerAccount` fields plus `status`:

| Field | Type | Description |
|-------|------|-------------|
| `status` | `$ref: BillingManagerAccountStatus` | Account status (active, frozen, deleted) |
| _(all other fields)_ | _(same as BillingManagerAccount)_ | Inherited from existing schema |

New enum schema: `BillingManagerAccountStatus` — `enum: [active, frozen, deleted]`

The plural endpoint path YAML files reference `BillingManagerAccountAdmin` instead of `BillingManagerAccount`.

## Breaking Changes

This is a hard cutover with no transition period:
- Customer admin users currently calling `/billing_accounts/{id}` will get permission denied. They must switch to `/billing_account`.
- The `/billing_accounts/{id}` response format changes from WebhookMessage to raw Account model. Admin clients may see additional fields.
- Accepted risk: frontend clients need to be updated to use the new singular endpoints.

## Permission Changes

| Endpoint | Before | After |
|----------|--------|-------|
| `GET /billing_account` | N/A | PermissionCustomerAdmin (NEW) |
| `PUT /billing_account` | N/A | PermissionCustomerAdmin (NEW) |
| `PUT /billing_account/payment_info` | N/A | PermissionCustomerAdmin (NEW) |
| `GET /billing_accounts` | N/A | PermissionProjectSuperAdmin (NEW) |
| `GET /billing_accounts/{id}` | PermissionCustomerAdmin | PermissionProjectSuperAdmin |
| `PUT /billing_accounts/{id}` | PermissionCustomerAdmin | PermissionProjectSuperAdmin |
| `PUT /billing_accounts/{id}/payment_info` | PermissionCustomerAdmin | PermissionProjectSuperAdmin |
| `POST /billing_accounts/{id}/balance_add_force` | PermissionProjectSuperAdmin | PermissionProjectSuperAdmin (unchanged) |
| `POST /billing_accounts/{id}/balance_subtract_force` | PermissionProjectSuperAdmin | PermissionProjectSuperAdmin (unchanged) |
