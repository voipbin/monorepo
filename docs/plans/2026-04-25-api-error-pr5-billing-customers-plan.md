# PR 5 — Billing & customer-data handler migration

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Migrate the billing, billing-account, storage-account, and conversation-account handlers to the canonical error envelope. Smaller than PR 4 — 22 handlers, 68 sites across 6 files. Customer surfaces (`customer.go`, `customers.go`) were already migrated in PR 1; this PR cleans up the remaining "customer-data" admin and self-service surfaces.

**Parent design:** `docs/plans/2026-04-24-api-error-response-codes-design.md`
**Predecessor:** PR 4 (`NOJIRA-api-error-pr4-numbers-providers`, merged `46f7cdc3f`).

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-pr5-billing-customers`
**Branch:** `NOJIRA-api-error-pr5-billing-customers` (branched from `origin/main` at `46f7cdc3f`)

## Scope

| File | LoC | Sites | Handlers |
|---|---|---|---|
| `server/billings.go` | 90 | 5 | 2 |
| `server/billing_accounts.go` | 295 | 23 | 6 |
| `server/billing_account.go` | 139 | 10 | 4 |
| `server/storage_account.go` | 33 | 2 | 1 |
| `server/storage_accounts.go` | 156 | 11 | 4 |
| `server/conversation_accounts.go` | 223 | 17 | 5 |

**Total: 68 sites across 22 handlers in 6 files.**

### Already migrated (out of scope for PR 5)

- `customer.go` (5 handlers) — done in PR 1 (auth-identity refactor)
- `customers.go` (10 handlers) — done in PR 1
- `service_agents_customer.go` (1 handler) — already uses canonical helpers

### Handler classification (per §6.1)

**Read (no path param) → 401, 500:**
- `GetBillings`, `GetBillingAccounts`, `GetBillingAccount`, `GetStorageAccount`, `GetStorageAccounts`, `GetConversationAccounts`

**Read (with resource ID) → 400, 401, 403, 404, 500:**
- `GetBillingsBillingId`, `GetBillingAccountsId`, `GetStorageAccountsId`, `GetConversationAccountsId`

**Write (no resource ID, self/customer) → 400, 401, 500:**
- `PutBillingAccount`, `PutBillingAccountPaymentInfo`, `PostBillingAccountPaddlePortalSession` — self-service customer endpoints under `/billing-account` (singular)

**Write (no resource ID, admin) → 400, 401, 403, 500:**
- `PostStorageAccounts`, `PostConversationAccounts` — admin-gated resource creation; declare 403 to match runtime emission via `"no permission"` translator pattern (§6.1 admin-baseline pattern established in PR 4)

**Write (with resource ID, admin) → 400, 401, 403, 404, 500:**
- `PutBillingAccountsId`, `PutBillingAccountsIdPaymentInfo` — admin overrides for any customer's billing
- `PostBillingAccountsIdBalanceAddForce`, `PostBillingAccountsIdBalanceSubtractForce` — admin balance manipulation (NOT billing-sensitive: these adjust a target customer's balance ledger directly, no charge against the caller)
- `DeleteStorageAccountsId`
- `PutConversationAccountsId`, `DeleteConversationAccountsId`

**State-transition (+409):** none.

**RPC-heavy (+503):** none — billing/storage/conversation operations route to single managers (billing-manager, storage-manager, conversation-manager respectively). Single-hop.

**Billing-sensitive (+402):** none. Despite the resource group name, none of these endpoints charge the customer balance during the call:
- Balance-add-force / balance-subtract-force are admin manual adjustments, not customer-initiated charges.
- Paddle portal session creation returns a URL; the actual payment happens on Paddle's side via webhook.
- Storage and conversation account creation are not chargeable operations.

## Forward-dependency notes

- The `billing-manager` catalog section was populated in PR 4 with `INSUFFICIENT_BALANCE`. PR 5 extends it with `BILLING_ACCOUNT_NOT_FOUND` (404) and adds two new sections (`storage-manager`, `conversation-manager`).
- **PR 6 (Messages & emails):** `POST /messages` and `POST /emails` are billing-sensitive — wire 402 same as PR 4 did for `/numbers`.

## Permission note (from `bin-api-manager/CLAUDE.md`)

**Billing and billing-account resources require `CustomerAdmin` permission ONLY (no Manager access).** This is enforced in `servicehandler` and is unchanged by PR 5. Permission failures continue to flow as `fmt.Errorf("user has no permission")` → translator → `403 PERMISSION_DENIED`.

## Tasks

### Task 1: Plan doc + commit (this file)

### Task 2: Migrate `server/billings.go`

2 handlers, 5 sites. Standard mappings.

`GetBillingsBillingId` already uses `openapi_types.UUID` for the path param — gin parses it; no UUID hardening needed in the handler body.

### Task 3: Migrate `server/billing_accounts.go`

6 handlers, 23 sites. Path-UUID hardening for `GetBillingAccountsId`, `PutBillingAccountsId`, `PutBillingAccountsIdPaymentInfo`, `PostBillingAccountsIdBalanceAddForce`, `PostBillingAccountsIdBalanceSubtractForce` (all use `id string`).

Sample tests:
- `Test_billingAccountsIDPut_InvalidID` (INVALID_ARGUMENT / INVALID_ID)
- `Test_billingAccountsIDBalanceAddForcePost_MissingAuthIdentity` (UNAUTHENTICATED / AUTHENTICATION_REQUIRED)

### Task 4: Migrate `server/billing_account.go`

4 handlers, 10 sites. Singular self-service endpoints (`/billing-account`, no path ID). Standard mappings.

Sample test:
- `Test_billingAccountPaddlePortalSessionPost_MissingAuthIdentity`

### Task 5: Migrate `server/storage_account.go` + `server/storage_accounts.go`

5 handlers total, 13 sites. Standard. Path-UUID hardening for `GetStorageAccountsId`, `DeleteStorageAccountsId`.

### Task 6: Migrate `server/conversation_accounts.go`

5 handlers, 17 sites. Standard. Path-UUID hardening for by-ID handlers.

### Task 7: OpenAPI path wiring

Wire all `/billings*`, `/billing-account`, `/billing-accounts*`, `/storage-account`, `/storage-accounts*`, `/conversation-accounts*` paths per §6.1 baseline:

- `GET /billings`, `GET /billing-accounts`, `GET /billing-account`, `GET /storage-account`, `GET /storage-accounts`, `GET /conversation-accounts` → 401, 500
- `GET /billings/{billing_id}`, `GET /billing-accounts/{id}`, `GET /storage-accounts/{id}`, `GET /conversation-accounts/{id}` → 400, 401, 403, 404, 500
- `PUT /billing-account`, `PUT /billing-account/payment-info`, `POST /billing-account/paddle-portal-session` → 400, 401, 500
- `POST /storage-accounts`, `POST /conversation-accounts` → 400, 401, 403, 500 (admin-gated; declare 403 per PR 4 pattern)
- `PUT /billing-accounts/{id}`, `PUT /billing-accounts/{id}/payment-info`, `POST /billing-accounts/{id}/balance-add-force`, `POST /billing-accounts/{id}/balance-subtract-force` → 400, 401, 403, 404, 500
- `DELETE /storage-accounts/{id}`, `PUT /conversation-accounts/{id}`, `DELETE /conversation-accounts/{id}` → 400, 401, 403, 404, 500

Regenerate `gens/openapi_server/gen.go`. Confirm loose `ServerInterface` signatures unchanged.

### Task 8: RST catalog updates

Edit `bin-api-manager/docsdev/source/restful_api_errors.rst`:

1. **Extend the `billing-manager` section** (populated in PR 4 with `INSUFFICIENT_BALANCE`):
   - Add `BILLING_ACCOUNT_NOT_FOUND` (404) — typed-error future; reachable today via translator's `"not found"` fallback.
   - Add `BILLING_NOT_FOUND` (404) — same, for individual billing records.

2. **Add `storage-manager` section** (new):
   - `STORAGE_ACCOUNT_NOT_FOUND` (404) — reachable via `"not found"` fallback.

3. **Add `conversation-manager` section** (new):
   - `CONVERSATION_ACCOUNT_NOT_FOUND` (404) — reachable via `"not found"` fallback.

Match the disclaimer style from PR 4's number-manager / provisioning-admin sections.

Rebuild Sphinx HTML.

### Task 9: Full verification

Standard 5-step workflow for both `bin-openapi-manager` and `bin-api-manager`.

### Task 10: Push + open PR

Conflict check, push, open PR.

## Conventions recap

- Commit title exactly `NOJIRA-api-error-pr5-billing-customers` on every commit.
- No AI attribution.
- `commonoutline.ServiceName*` constants.
- 3-group import layout in test files.
- Reuse `assertMissingAuthIdentity` and `assertErrorResponse` helpers.
- Path-UUID hardening pattern: `uuid.FromStringOrNil(id) == uuid.Nil` (matches PR 3 and PR 4).
- Preserve existing `log.Errorf/Infof` site-specific lines.

## Success criteria

- All 10 tasks committed
- `go test -race ./...` green for both `bin-api-manager` and `bin-openapi-manager`
- `golangci-lint` 0 issues
- Loose `ServerInterface` signatures unchanged
- Billing-manager catalog section extended with `BILLING_ACCOUNT_NOT_FOUND` and `BILLING_NOT_FOUND`
- New storage-manager and conversation-manager catalog sections
- All 68 sites converted
- 403 declared on admin-gated POST endpoints (`/storage-accounts`, `/conversation-accounts`) and admin balance manipulation endpoints
