# Paddle Billing Integration Design

**Date:** 2026-03-19
**Status:** Draft
**Branch:** NOJIRA-Paddle-billing-integration

## Problem Statement

VoIPbin currently has a billing system that tracks account balances (credits and tokens) and charges for service usage (calls, SMS, numbers, etc.), but has no payment gateway integration. Balance top-ups and plan changes are manual (CLI tool or admin API). Customers cannot self-service purchase credits or subscribe to plans.

## Goal

Integrate Paddle Billing v2 to enable:
1. **Credit top-ups** — Customers purchase fixed credit packages via Paddle checkout
2. **Subscription plans** — Customers subscribe to monthly plans (Basic/Professional/Unlimited) that allocate tokens and resource limits
3. **Refund handling** — Subtract credits when Paddle issues refunds

## Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Payment provider | Paddle Billing v2 | Modern API, handles tax/compliance, Go SDK available |
| Checkout UX | Paddle.js overlay | Minimal frontend work, PCI compliant, hosted by Paddle |
| Webhook receiver | bin-hook-manager | Already serves as public HTTPS webhook gateway at hook.voipbin.net |
| Communication pattern | RPC (request/response) | Follows existing hook-manager → service pattern via requesthandler |
| Customer linking | custom_data metadata in checkout | Stateless — customer_id passed through Paddle, returned in webhooks |
| Subscription tracking | Store paddle_subscription_id on billing account | Needed for renewal/cancellation events that don't carry custom_data |
| Credit amounts | Fixed tiers | Predefined packages in Paddle (e.g., $10, $25, $50, $100) |
| Currency | USD only | All Paddle products priced in USD; reject non-USD transactions |
| Refunds | Handle in v1 | Subtract credits on transaction.refunded |
| Payment failure notifications | Log only | Paddle handles customer communication |
| Token replenishment | Webhook-driven for Paddle subscribers | Replace cron-based replenishment; keep cron for non-Paddle accounts |
| Paddle SDK | github.com/PaddleHQ/paddle-go-sdk/v4 | Provides WebhookVerifier + typed event structs |

## Architecture

### End-to-End Flow

```
                     Paddle.js overlay
Customer ────────────────────────────── Paddle Cloud
(browser)         checkout / subscribe          │
                                                │ HTTPS POST
                                                │ Paddle-Signature header
                                                ▼
                                     ┌─────────────────────┐
                                     │   bin-hook-manager   │
                                     │   hook.voipbin.net   │
                                     │                     │
                                     │ POST /v1.0/billing/ │
                                     │   paddle            │
                                     │                     │
                                     │ 1. Verify signature │
                                     │ 2. Wrap in Hook     │
                                     │ 3. RPC to billing   │
                                     └─────────┬───────────┘
                                               │ RabbitMQ RPC
                                               │ POST /v1/hooks/paddle
                                               ▼
                                     ┌─────────────────────┐
                                     │ bin-billing-manager  │
                                     │                     │
                                     │ 1. Parse Paddle evt │
                                     │ 2. Check idempotency│
                                     │ 3. Extract customer │
                                     │ 4. Update balance   │
                                     │    or plan          │
                                     │ 5. Create billing   │
                                     │    record           │
                                     └─────────────────────┘
```

### Customer Identification

1. Frontend opens Paddle.js with `customData: { customer_id: "<uuid>" }`
2. Paddle includes `custom_data` in every webhook event payload
3. billing-manager extracts `customer_id` from `custom_data`
4. Looks up billing account via existing `GetByCustomerID()` (which calls customer-manager RPC to map customer_id → billing_account_id, then fetches the account by ID)
5. On first subscription, stores `paddle_subscription_id` and `paddle_customer_id` on the billing account for future event correlation
6. For subscription events without `custom_data` (renewals, cancellations), looks up by `paddle_subscription_id`
7. For subscriptions created without `custom_data` (e.g., admin-created in Paddle dashboard): log warning and return 200 (prevents retry storm)

### Idempotency

Paddle may retry webhooks. Use Paddle's `event_id` as the idempotency key:
- Derive a deterministic UUID from `event_id` via `uuid.NewV5(uuid.NamespaceDNS, eventID)`
- Before processing, query `billing_billings` by `idempotency_key` column using a new `BillingGetByIdempotencyKey(ctx, key uuid.UUID)` method (single DB query)
- If found, skip processing and return 200
- If not found, process the event and create a billing record with `idempotency_key = derived_uuid`
- All Paddle billing records MUST set `IdempotencyKey` for dedup and `ReferenceID` for the same derived UUID (used as a stable reference)

## Paddle Event Handling

### Events Processed

| Paddle Event | Condition | billing-manager Action |
|---|---|---|
| `transaction.completed` | No `subscription_id` | Credit top-up: `AddBalance()` + billing record (type=`top_up`, ref=`paddle_credit_purchase`) |
| `transaction.completed` | Has `subscription_id` | Subscription renewal: replenish tokens per plan via `AddBalance()` for tokens |
| `subscription.created` | — | Set `PlanType`, store `paddle_subscription_id`/`paddle_customer_id`, allocate initial tokens |
| `subscription.updated` | — | Update `PlanType`, adjust token allocation and resource limits |
| `subscription.canceled` | — | Downgrade to `PlanTypeFree` immediately (Paddle sends `subscription.canceled` at end of billing period when `effective_from=next_billing_period`) |
| `transaction.refunded` | — | Subtract refunded amount from `BalanceCredit` + billing record (type=`refund`, ref=`paddle_refund`) |
| `transaction.payment_failed` | — | Log warning at Error level; no account changes (Paddle handles customer emails) |
| Unknown event type | — | Log at Debug level, return 200 (prevent infinite retries) |

### Plan Mapping

Paddle products are configured in the Paddle dashboard with metadata:
- `plan_type`: `"basic"`, `"professional"`, or `"unlimited"`
- Token allocation is determined by existing `PlanTokenMap` in `models/account/plan.go`
- Resource limits are determined by existing `PlanLimitMap` in `models/account/plan.go`

No hardcoded plan-to-Paddle-product mapping in Go code — driven entirely by product metadata in Paddle.

### Subscription Lifecycle

| Lifecycle Event | What Happens |
|---|---|
| Customer subscribes | `subscription.created` → set plan, allocate tokens, store paddle IDs |
| Monthly renewal | `transaction.completed` (with subscription_id) → replenish tokens |
| Upgrade (Basic→Pro) | `subscription.updated` → update plan type, adjust tokens/limits |
| Downgrade (Pro→Basic) | `subscription.updated` → update plan type, adjust tokens/limits |
| Cancellation | `subscription.canceled` → downgrade to Free immediately (Paddle fires this event at end of billing period when customer chose end-of-period cancellation) |

## Changes by Service

### bin-hook-manager

**ServiceHandler interface refactor:**

Change all methods from `(ctx, uri string, m []byte) error` to `(ctx context.Context, r *http.Request) error`. The `*http.Request` carries both headers (needed for Paddle signature verification) and body. Each implementation reads the body internally.

```go
type ServiceHandler interface {
    Email(ctx context.Context, r *http.Request) error
    Message(ctx context.Context, r *http.Request) error
    Conversation(ctx context.Context, r *http.Request) error
    Billing(ctx context.Context, r *http.Request) error  // NEW
}
```

**Gin handlers simplify** — no more `io.ReadAll()` at HTTP layer:

```go
func billingPOST(c *gin.Context) {
    ctx := context.Background()
    serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
    if err := serviceHandler.Billing(ctx, c.Request); err != nil {
        c.AbortWithStatus(http.StatusInternalServerError)
        return
    }
    c.AbortWithStatus(200)
}
```

**serviceHandler.Billing() implementation:**

```go
func (h *serviceHandler) Billing(ctx context.Context, r *http.Request) error {
    // 1. Buffer body first (Paddle SDK's Verify() consumes r.Body)
    body, err := io.ReadAll(r.Body)
    if err != nil {
        return fmt.Errorf("could not read body: %w", err)
    }
    // Restore body for Paddle SDK signature verification
    r.Body = io.NopCloser(bytes.NewReader(body))

    // 2. Verify Paddle webhook signature
    if h.paddleVerifier != nil {
        ok, err := h.paddleVerifier.Verify(r)
        if err != nil || !ok {
            return fmt.Errorf("paddle signature verification failed: %w", err)
        }
    }

    // 3. Wrap in Hook and send RPC to billing-manager
    req := &hmhook.Hook{
        ReceviedURI:  r.Host + r.URL.Path,
        ReceivedData: body,
    }
    return h.reqHandler.BillingV1PaddleHook(ctx, req)
}
```

**New files:**
- `api/v1.0/billing/main.go` — route: `POST /billing/paddle`
- `api/v1.0/billing/billing.go` — handler
- `api/v1.0/billing/billing_test.go`
- `pkg/servicehandler/billing.go` — Paddle verification + RPC
- `pkg/servicehandler/billing_test.go`

**Edited files:**
- `api/v1.0/v1.0.go` — add `billing.ApplyRoutes(v1)`
- `pkg/servicehandler/main.go` — interface change + add `paddleVerifier` field
- `pkg/servicehandler/email.go`, `message.go`, `conversation.go` — refactor to `*http.Request`
- `api/v1.0/emails/emails.go`, `messages/messages.go`, `conversation/conversation.go` — simplify handlers
- `cmd/hook-manager/main.go` — add `PADDLE_WEBHOOK_SECRET_KEY` config, pass to serviceHandler
- `go.mod` — add `github.com/PaddleHQ/paddle-go-sdk/v4`
- `k8s/deployment.yml` — add env var from K8s secret (`bin-manager-secrets`)
- All `*_test.go` — update for new interface signatures

### bin-common-handler

**New requesthandler method:**

```go
// BillingV1PaddleHook sends a Paddle webhook hook to billing-manager
func (r *requestHandler) BillingV1PaddleHook(ctx context.Context, hm *hmhook.Hook) error {
    uri := "/v1/hooks/paddle"
    m, err := json.Marshal(hm)
    if err != nil {
        return err
    }
    tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodPost, "billing/hooks/paddle", requestTimeoutDefault, 0, ContentTypeJSON, m)
    if err != nil {
        return err
    }
    return parseResponse(tmp, nil)
}
```

**Files:**
- `pkg/requesthandler/billing_hooks.go` — new
- `pkg/requesthandler/billing_hooks_test.go` — new
- `pkg/requesthandler/main.go` — add `BillingV1PaddleHook` to `RequestHandler` interface

### bin-billing-manager

**New listen handler route:**

```go
// In main.go regex declarations:
regV1HooksPaddle = regexp.MustCompile("/v1/hooks/paddle$")

// In processRequest switch:
case regV1HooksPaddle.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
    response, err = h.processV1HooksPaddlePost(ctx, m)
    requestType = "/v1/hooks/paddle"
```

**processV1HooksPaddlePost flow:**

1. Unmarshal `hook.Hook` from request data
2. Parse `ReceivedData` as Paddle notification JSON
3. Extract `event_type` field
4. Check idempotency (event_id → existing billing record with matching idempotency_key)
5. Extract `custom_data.customer_id` (for transaction events) or look up by `paddle_subscription_id` (for subscription events)
6. Route to appropriate accountHandler method:
   - Credit purchase → `accountHandler.PaddleCreditTopUp(ctx, customerID, amountMicros, eventID)`
   - Subscription created → `accountHandler.PaddleSubscriptionCreate(ctx, customerID, planType, paddleSubID, paddleCustID, eventID)`
   - Subscription updated → `accountHandler.PaddleSubscriptionUpdate(ctx, paddleSubID, newPlanType, eventID)`
   - Subscription canceled → `accountHandler.PaddleSubscriptionCancel(ctx, paddleSubID, effectiveFrom, eventID)`
   - Subscription renewal → `accountHandler.PaddleSubscriptionRenew(ctx, paddleSubID, eventID)`
   - Refund → `accountHandler.PaddleRefund(ctx, customerID, amountMicros, eventID)`
7. Return 200 on success

**New accountHandler methods:**

Each method:
- Validates the billing account exists
- Performs the account operation (balance add/subtract, plan change)
- Creates an immutable billing record for audit trail
- Uses the Paddle `event_id` as `IdempotencyKey`

**New files:**
- `pkg/listenhandler/v1_hooks_paddle.go`
- `pkg/listenhandler/v1_hooks_paddle_test.go`
- `pkg/listenhandler/models/request/v1_hooks.go`
- `pkg/accounthandler/paddle.go`
- `pkg/accounthandler/paddle_test.go`

**Edited files:**
- `pkg/listenhandler/main.go` — add regex + switch case
- `pkg/accounthandler/main.go` — add Paddle methods to interface
- `pkg/dbhandler/main.go` — add `AccountGetByPaddleSubscriptionID` and `BillingGetByIdempotencyKey` to interface
- `pkg/dbhandler/account_paddle.go` — new: `AccountGetByPaddleSubscriptionID` implementation
- `pkg/dbhandler/billing_paddle.go` — new: `BillingGetByIdempotencyKey` implementation
- `models/billing/billing.go` — add new ReferenceType constants
- `models/account/account.go` — add `PaddleSubscriptionID`, `PaddleCustomerID` fields
- `models/account/field.go` — add new Field constants

**Paddle amount format:**
Paddle Billing v2 sends transaction totals as decimal strings in the base currency unit (e.g., `"10.00"` for ten US dollars). Convert to micros by parsing as float64 and multiplying by 1,000,000. Always validate currency is USD before processing.

### bin-dbscheme-manager

**New Alembic migration** — add columns to `billing_accounts`:

```python
def upgrade():
    op.add_column('billing_accounts', sa.Column('paddle_subscription_id', sa.String(255), nullable=True))
    op.add_column('billing_accounts', sa.Column('paddle_customer_id', sa.String(255), nullable=True))
    op.create_index('ix_billing_accounts_paddle_subscription_id', 'billing_accounts', ['paddle_subscription_id'])
    op.create_index('ix_billing_accounts_paddle_customer_id', 'billing_accounts', ['paddle_customer_id'])

def downgrade():
    op.drop_index('ix_billing_accounts_paddle_customer_id', 'billing_accounts')
    op.drop_index('ix_billing_accounts_paddle_subscription_id', 'billing_accounts')
    op.drop_column('billing_accounts', 'paddle_customer_id')
    op.drop_column('billing_accounts', 'paddle_subscription_id')
```

### New ReferenceTypes (billing model)

```go
ReferenceTypePaddleCreditPurchase ReferenceType = "paddle_credit_purchase"
ReferenceTypePaddleSubscription   ReferenceType = "paddle_subscription"
ReferenceTypePaddleRefund         ReferenceType = "paddle_refund"
```

## Security

- **Signature verification**: Paddle Go SDK `WebhookVerifier` validates HMAC signature + timestamp (5-min replay protection)
- **Public endpoint**: `hook.voipbin.net/v1.0/billing/paddle` — protected by signature verification; invalid signatures return 400 (stops Paddle retries for bad signatures, prevents retry storm from misconfigured secrets)
- **Secret management**: `PADDLE_WEBHOOK_SECRET_KEY` stored as K8s secret, injected via env var
- **No sensitive data logging**: Raw Paddle payload logged at Debug level only (may contain payment details)
- **custom_data integrity**: Paddle signs the entire payload including custom_data — tampering impossible without the signing key

## Failure Modes

| Failure | Behavior | Recovery |
|---|---|---|
| hook-manager → billing-manager RPC timeout | hook-manager returns 500 to Paddle | Paddle retries (up to 30 attempts over days) |
| billing-manager DB error | RPC returns error → 500 | Paddle retries |
| Invalid customer_id | RPC returns 404 | Paddle retries; exhausts after max retries |
| Duplicate event (Paddle retry) | Idempotency check → skip + return 200 | No duplicate processing |
| Unknown event type | Log + return 200 | Prevents infinite retries for new event types |
| Paddle signature invalid | hook-manager returns 400 | Stops Paddle retries; attacker gets no useful response |
| Missing custom_data | Log warning + return 200 | Prevents retry storm for admin-created subscriptions |
| Balance goes negative after refund | Allow negative balance, freeze account | Manual intervention by admin |

## Deployment Order

1. **Alembic migration** — add `paddle_subscription_id`, `paddle_customer_id` columns (safe, additive, nullable)
2. **bin-common-handler** — new `BillingV1PaddleHook` method + mock regeneration
3. **bin-billing-manager** — new listen handler + account handler methods
4. **bin-hook-manager** — new endpoint, refactored ServiceHandler, Paddle SDK
5. **K8s secret** — create `PADDLE_WEBHOOK_SECRET_KEY` secret
6. **Paddle dashboard** — configure webhook URL: `https://hook.voipbin.net/v1.0/billing/paddle`
7. **Paddle dashboard** — configure products with metadata (`plan_type`, pricing)

If hook-manager deploys before billing-manager: Paddle webhooks fail → 500 → Paddle retries. Safe.

## What This Design Does NOT Cover

- **Frontend (Paddle.js)**: Integration on admin.voipbin.net / talk.voipbin.net is a separate concern. Frontend needs: Paddle client token, product/price IDs, customer_id for customData.
- **OpenAPI / public API changes**: Paddle webhooks are internal; no customer-facing API changes.
- **RST documentation**: No user-facing feature documentation changes (unless we want to document billing plans for customers).
- **Paddle product/price creation**: Products are configured in the Paddle dashboard, not in code.
- **Payment failure customer notifications**: Paddle handles this via their own email system.

## Testing Strategy

- **Unit tests**: Mock Paddle SDK, mock requesthandler, mock accountHandler. Test each handler method with various Paddle event payloads.
- **Integration tests**: Use Paddle sandbox to send simulated webhooks. Verify billing records and balance changes.
- **Paddle webhook simulator**: Paddle dashboard has a webhook simulator for testing individual event types.
- **Idempotency tests**: Send the same event_id twice, verify only one billing record is created.
