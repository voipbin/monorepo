# Paddle Customer Portal — Cancel & Upgrade Support

**Date:** 2026-03-29
**Status:** Approved

## Problem Statement

The backend currently has no way to generate Paddle Customer Portal session URLs. Users cannot cancel or upgrade their subscriptions through our platform. Additionally, the existing `subscription.updated` webhook handler relies on `custom_data.plan_type` to determine plan changes, which may be stale when users change plans through the Paddle Customer Portal (the portal updates `items[].price`, not `custom_data`). There is also no "cancellation pending" state — when a user schedules a cancellation via the portal, the backend has no way to surface this to the frontend.

## Approach

Use Paddle's Customer Portal (hosted by Paddle) for the cancel/upgrade UI. The backend generates a time-limited portal session URL via the Paddle API. The frontend redirects the user to this URL. Paddle handles the UX, and our backend processes the resulting webhook events.

## Components Changed

| Component | Change |
|-----------|--------|
| `bin-billing-manager/models/account/` | Add `PlanStatus` field (`active`, `canceling`) |
| `bin-billing-manager/pkg/paddlehandler/` | **New** — Paddle API client (portal sessions, price-to-plan mapping) |
| `bin-billing-manager/pkg/listenhandler/` | New route + update webhook handlers |
| `bin-billing-manager/pkg/accounthandler/` | New portal session method + scheduled cancel logic |
| `bin-billing-manager/pkg/dbhandler/` | DB operations for `plan_status` |
| `bin-common-handler/pkg/requesthandler/` | RPC client for portal session |
| `bin-api-manager/pkg/servicehandler/` | External endpoint handler |
| `bin-openapi-manager/openapi/` | OpenAPI schema + path |
| `bin-dbscheme-manager/` | Alembic migration for `plan_status` column |

## New Environment Variables

```
PADDLE_API_KEY=pdl_live_...
PADDLE_PRODUCT_ID=12345
PADDLE_PRICE_ID_BASIC=11234...
PADDLE_PRICE_ID_PROFESSIONAL=1123...
```

## New Model Field

```go
// In models/account/account.go
PlanStatus PlanStatus `json:"plan_status" db:"plan_status"`

type PlanStatus string
const (
    PlanStatusActive    PlanStatus = "active"
    PlanStatusCanceling PlanStatus = "canceling"
)
```

## New Endpoints

- **External HTTP**: `POST /billing_account/paddle_portal_session` (CustomerAdmin permission)
- **Internal RPC**: `POST /v1/accounts/<id>/paddle_portal_session`
- **Response**: `{ "url": "https://customer-portal.paddle.com/..." }`

## New Package: paddlehandler

Location: `bin-billing-manager/pkg/paddlehandler/`

```go
type PaddleHandler interface {
    CreatePortalSession(ctx context.Context, paddleCustomerID string) (string, error)
    GetPlanTypeByPriceID(priceID string) (account.PlanType, error)
}
```

- Plain `net/http` client, no Paddle SDK dependency
- Bearer token auth via `PADDLE_API_KEY`
- Base URL: `https://api.paddle.com`

## Sequence Diagrams

### Flow 1: User Opens Customer Portal

```
User (Frontend)
  |
  |-> POST /billing_account/paddle_portal_session
  |     |
  |     |-> bin-api-manager
  |     |     |-- Auth: verify JWT, check CustomerAdmin permission
  |     |     |-- Get billing account (has paddle_customer_id)
  |     |     \-> RabbitMQ RPC: POST /v1/accounts/<id>/paddle_portal_session
  |     |           |
  |     |           |-> bin-billing-manager (listenhandler)
  |     |           |     \-> paddlehandler.CreatePortalSession(paddle_customer_id)
  |     |           |           |
  |     |           |           \-> POST https://api.paddle.com/customer-portal-sessions
  |     |           |                 Body: { "customer_id": "ctm_abc123" }
  |     |           |                 Auth: Bearer pdl_live_...
  |     |           |                 |
  |     |           |           <-----/ Response: { "urls": { "general": { "overview": "https://..." } } }
  |     |           |
  |     |           <-- Return portal URL
  |     |
  |     <-- Return portal URL to frontend
  |
  |-> Redirect user to Paddle portal URL
  |     (User cancels or upgrades in Paddle's UI)
```

### Flow 2: Paddle Sends Webhook After User Action

```
Paddle Cloud
  |
  |-> POST https://hook.voipbin.net/v1.0/billing/paddle
  |     |
  |     |-> bin-hook-manager
  |     |     |-- Verify HMAC-SHA256 signature
  |     |     \-> RabbitMQ RPC: POST /v1/hooks/paddle
  |     |           |
  |     |           |-> bin-billing-manager (listenhandler)
  |     |           |
  |     |           |-- Case: subscription.updated (plan change)
  |     |           |     |-- Parse items[].price.id
  |     |           |     |-- paddlehandler.GetPlanTypeByPriceID(priceID)
  |     |           |     |     \-- Match against PADDLE_PRICE_ID_BASIC / PROFESSIONAL
  |     |           |     |-- accounthandler.PaddleSubscriptionUpdate(subID, newPlanType)
  |     |           |     \-- Update plan_type + reset tokens
  |     |           |
  |     |           |-- Case: subscription.updated (scheduled cancel)
  |     |           |     |-- Detect scheduled_change.action == "cancel"
  |     |           |     |-- accounthandler.PaddleSubscriptionScheduleCancel(subID)
  |     |           |     \-- Set plan_status = "canceling"
  |     |           |
  |     |           |-- Case: subscription.canceled (cancel takes effect)
  |     |           |     |-- accounthandler.PaddleSubscriptionCancel(subID)
  |     |           |     \-- Downgrade plan_type = "free", plan_status = "active"
```

## Webhook Handler Changes

### subscription.updated — Now Handles Two Scenarios

| Scenario | Detection | Action |
|----------|-----------|--------|
| Plan change (upgrade/downgrade) | `items[].price.id` changed, no `scheduled_change` | `GetPlanTypeByPriceID(priceID)` -> update `plan_type`, reset tokens, set `plan_status = "active"` |
| Scheduled cancellation | `scheduled_change.action == "cancel"` | Set `plan_status = "canceling"` (plan stays active until period end) |

### subscription.canceled — Enhanced

- Downgrade `plan_type` to `free`
- Reset `plan_status` to `active`
- Reset tokens to free allowance (existing behavior)

### Plan Type Resolution Priority (subscription.updated)

1. Match `items[].price.id` against `PADDLE_PRICE_ID_BASIC` / `PADDLE_PRICE_ID_PROFESSIONAL`
2. Fall back to `custom_data.plan_type` (backward compatibility)
3. If neither resolves -> skip with log (return 200)

## DB Migration

```sql
ALTER TABLE billing_accounts
ADD COLUMN plan_status VARCHAR(32) NOT NULL DEFAULT 'active';
```

## Test Coverage

### paddlehandler
- Successful portal session creation
- Paddle API returns error (4xx, 5xx)
- Paddle API timeout
- Missing/invalid paddle_customer_id
- Price ID mapping: known IDs, unknown ID, empty ID

### listenhandler webhook updates
- subscription.updated with plan change (price ID match)
- subscription.updated with plan change (fallback to custom_data.plan_type)
- subscription.updated with plan change (neither resolves — skip)
- subscription.updated with scheduled cancellation (scheduled_change.action == "cancel")
- subscription.updated that is neither plan change nor scheduled cancel (e.g. payment method update — skip)
- subscription.canceled — sets plan to free, plan_status to active
- Idempotency — duplicate events are skipped

### accounthandler
- PaddleSubscriptionScheduleCancel — sets plan_status to canceling
- PaddleSubscriptionUpdate — resets plan_status to active on plan change
- PaddleSubscriptionCancel — resets plan_status to active on cancel
- Portal session — account has no paddle_customer_id (error)

### API layer (servicehandler)
- Successful portal session request
- User has no paddle subscription (no paddle_customer_id)
- Permission denied (not CustomerAdmin)

## Out of Scope

- No admin endpoint for portal sessions (admin manages via Paddle dashboard)
- No `past_due` / `paused` handling (can add later)
- No frontend changes (separate work)
