# Monthly Token Allowance System Design

## Problem Statement

The current billing system is purely credit-based (pay-as-you-go). Every billable event
(calls, SMS, number purchases) deducts from a prepaid USD balance. There is no concept
of included usage — customers pay per unit from the first use.

Additionally, the current billing stores call duration in seconds with a per-second rate,
which is incorrect. The correct unit should be per-minute with ceiling rounding.

We want to introduce a monthly token allowance system where each plan tier includes a
pool of tokens that cover certain resource types. This gives customers predictable monthly
usage at no extra credit cost, while PSTN calls and number purchases remain credit-based.

## Overview

Each billing account receives a monthly token pool based on its plan tier. Different
resource types consume tokens at different rates. When the token pool is exhausted,
usage overflows to credits at the normal per-unit rate.

### Token Allowances by Plan

| Plan          | Monthly Tokens |
|---------------|----------------|
| Free          | 1,000          |
| Basic         | 10,000         |
| Professional  | 100,000        |
| Unlimited     | No limit       |

### Token Consumption Rates

| Resource Type               | Token Cost       | Status          |
|-----------------------------|------------------|-----------------|
| Virtual Number call         | 1 token / minute | Implement now   |
| SMS                         | 10 tokens / msg  | Implement now   |
| Direct Extension call       | 1 token / minute | Future          |
| Email                       | 10 tokens / msg  | Future          |
| SNS                         | 10 tokens / msg  | Future          |

### Billing Unit Fix: Seconds to Minutes

The current system stores `billing_unit_count` in seconds and `cost_per_unit` as a
per-second rate ($0.020/sec). This is incorrect. As part of this change, all call
billing is updated to use **minutes with ceiling rounding**:

| Field | Before (incorrect) | After (correct) |
|---|---|---|
| `billing_unit_count` | Seconds (e.g., 150) | Minutes, ceiling (e.g., 3) |
| `cost_per_unit` | Per-second ($0.020) | Per-minute (varies by cost type) |
| `cost_total` formula | `cost_per_unit × seconds` | `cost_per_unit × minutes` |

A 2m30s call is stored as `cost_unit_count = 3` (ceiling of 150/60).

This change applies to **all** call billing (PSTN, VN, extension), not just the token system.

### Call Classification

Calls are classified per leg. Each call may have an incoming leg and an outgoing leg,
each billed independently.

**Call patterns:**

Incoming PSTN call (number provider to pstn.voipbin.net), 2 legs:

| Direction | Source | Destination | Cost Type | Billing |
|-----------|--------|-------------|-----------|---------|
| Incoming  | tel    | tel         | `call_pstn_incoming` | Credits ($0.0045/min) |
| Outgoing  | tel    | Extension   | `call_extension` | Free |

Incoming call to VN, routed to Extension (2 legs):

| Direction | Source | Destination | Cost Type | Billing |
|-----------|--------|-------------|-----------|---------|
| Incoming  | Any    | VN (+999xxx)| `call_vn` | Tokens  |
| Outgoing  | VN (+999xxx) | Extension | `call_extension` | Free |

Incoming SIP to Extension, routed to Extension (2 legs, future):

| Direction | Source | Destination | Cost Type | Billing |
|-----------|--------|-------------|-----------|---------|
| Incoming  | SIP (type=sip) | Extension | `call_direct_ext` | Tokens (future) |
| Outgoing  | Extension | Extension | `call_extension` | Free |

Outbound call to PSTN (1 leg):

| Direction | Source | Destination | Cost Type | Billing |
|-----------|--------|-------------|-----------|---------|
| Outgoing  | Any    | PSTN (type=tel) | `call_pstn_outgoing` | Credits ($0.006/min) |

**Classification rule (priority order):**
1. Incoming + src type is `tel` + dst type is `tel` → `call_pstn_incoming` (credits, $0.0045/min)
2. Outgoing + dst type is `tel` → `call_pstn_outgoing` (credits, $0.006/min)
3. Incoming + dst starts with `+999` → `call_vn` (tokens)
4. Incoming + src type is `sip` + dst type is extension → `call_direct_ext` (tokens, future)
5. Everything else → `call_extension` (free)

### Reference Types vs Cost Types

**Reference types are resource locators** — they tell you where to find the source resource:

| Reference Type | Meaning |
|----------------|---------|
| `call`         | Look up this resource in call-manager |
| `call_extension` | Look up this resource in call-manager (extension call) |
| `sms`          | Look up this resource in message-manager |
| `number`       | Look up this resource in number-manager |
| `number_renew` | Look up this resource in number-manager |

Reference types are **unchanged**. No renames, no new types, no migration.

**Cost types are billing classifications** — they explain why this cost was applied:

| Cost Type | Description | Token Rate | Credit Rate |
|-----------|-------------|------------|-------------|
| `call_pstn_outgoing` | Outgoing PSTN call | N/A | $0.006/min |
| `call_pstn_incoming` | Incoming PSTN call | N/A | $0.0045/min |
| `call_vn` | Virtual number call | 1 token/min | $0.0045/min (overflow) |
| `call_extension` | Internal extension call | N/A | Free |
| `call_direct_ext` | Direct extension call (future) | 1 token/min | TBD |
| `sms` | SMS message | 10 tokens/msg | $0.008/msg (overflow) |
| `email` | Email message (future) | 10 tokens/msg | TBD |
| `sns` | SNS message (future) | 10 tokens/msg | TBD |
| `number` | Number purchase | N/A | $5.00 |
| `number_renew` | Number renewal | N/A | $5.00 |

## Data Model

### Updated Billing Struct

The billing struct is updated with new cost-related fields. All cost fields use a
consistent `Cost` prefix:

```go
type Billing struct {
    commonidentity.Identity

    AccountID uuid.UUID `json:"account_id" db:"account_id,uuid"`

    Status Status `json:"status" db:"status"`

    ReferenceType ReferenceType `json:"reference_type" db:"reference_type"`
    ReferenceID   uuid.UUID     `json:"reference_id" db:"reference_id,uuid"`

    CostType          CostType `json:"cost_type" db:"cost_type"`
    CostUnitCount     float32  `json:"cost_unit_count" db:"cost_unit_count"`
    CostTokenPerUnit  int      `json:"cost_token_per_unit" db:"cost_token_per_unit"`
    CostTokenTotal    int      `json:"cost_token_total" db:"cost_token_total"`
    CostCreditPerUnit float32  `json:"cost_credit_per_unit" db:"cost_credit_per_unit"`
    CostCreditTotal   float32  `json:"cost_credit_total" db:"cost_credit_total"`

    TMBillingStart *time.Time `json:"tm_billing_start" db:"tm_billing_start"`
    TMBillingEnd   *time.Time `json:"tm_billing_end" db:"tm_billing_end"`

    TMCreate *time.Time `json:"tm_create" db:"tm_create"`
    TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
    TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}
```

**Field descriptions:**
- `CostType` — Why this cost was applied (e.g., `call_vn`, `call_pstn_outgoing`)
- `CostUnitCount` — Total billable units (minutes for calls, count for SMS)
- `CostTokenPerUnit` — Token rate per unit (e.g., 1 token/min for VN calls)
- `CostTokenTotal` — Total tokens consumed from allowance
- `CostCreditPerUnit` — Credit rate per unit (e.g., $0.006/min for PSTN outgoing)
- `CostCreditTotal` — Total credits deducted from account balance

**Removed fields** (replaced by the above):
- `CostPerUnit` → split into `CostTokenPerUnit` + `CostCreditPerUnit`
- `CostTotal` → split into `CostTokenTotal` + `CostCreditTotal`
- `BillingUnitCount` → renamed to `CostUnitCount` (now stores minutes, not seconds)

### Example Billing Records

**VN call, 5 minutes, 2 tokens remaining:**
```
reference_type      = "call"
reference_id        = <call-uuid>
cost_type           = "call_vn"
cost_unit_count     = 5              (minutes, ceiling)
cost_token_per_unit = 1              (1 token/min)
cost_token_total    = 2              (2 tokens consumed)
cost_credit_per_unit = 0.0045        ($/min overflow rate)
cost_credit_total   = 0.0135         (3 overflow min × $0.0045)
```
Result: 2 tokens consumed + $0.0135 credits deducted.

**Outgoing PSTN call, 3 minutes:**
```
reference_type      = "call"
reference_id        = <call-uuid>
cost_type           = "call_pstn_outgoing"
cost_unit_count     = 3              (minutes, ceiling)
cost_token_per_unit = 0              (no tokens for PSTN)
cost_token_total    = 0
cost_credit_per_unit = 0.006         ($/min)
cost_credit_total   = 0.018          (3 min × $0.006)
```
Result: $0.018 credits deducted.

**Incoming PSTN call, 10 minutes:**
```
reference_type      = "call"
reference_id        = <call-uuid>
cost_type           = "call_pstn_incoming"
cost_unit_count     = 10             (minutes, ceiling)
cost_token_per_unit = 0              (no tokens for PSTN)
cost_token_total    = 0
cost_credit_per_unit = 0.0045        ($/min)
cost_credit_total   = 0.045          (10 min × $0.0045)
```
Result: $0.045 credits deducted.

**SMS with full token coverage:**
```
reference_type      = "sms"
reference_id        = <msg-uuid>
cost_type           = "sms"
cost_unit_count     = 1              (1 message)
cost_token_per_unit = 10             (10 tokens/msg)
cost_token_total    = 10             (fully covered by tokens)
cost_credit_per_unit = 0.008         ($/msg overflow rate)
cost_credit_total   = 0              (no overflow)
```
Result: 10 tokens consumed, no credits deducted.

**Extension call (free):**
```
reference_type      = "call"
reference_id        = <call-uuid>
cost_type           = "call_extension"
cost_unit_count     = 5              (minutes, ceiling)
cost_token_per_unit = 0
cost_token_total    = 0
cost_credit_per_unit = 0
cost_credit_total   = 0
```
Result: No charge.

### Database Schema Changes

**Modified table: `billing_billings`**

Remove old cost columns and add new ones:

```sql
-- Remove old columns
ALTER TABLE billing_billings DROP COLUMN cost_per_unit;
ALTER TABLE billing_billings DROP COLUMN cost_total;
ALTER TABLE billing_billings DROP COLUMN billing_unit_count;

-- Add new columns
ALTER TABLE billing_billings ADD COLUMN cost_type VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE billing_billings ADD COLUMN cost_unit_count FLOAT NOT NULL DEFAULT 0;
ALTER TABLE billing_billings ADD COLUMN cost_token_per_unit INT NOT NULL DEFAULT 0;
ALTER TABLE billing_billings ADD COLUMN cost_token_total INT NOT NULL DEFAULT 0;
ALTER TABLE billing_billings ADD COLUMN cost_credit_per_unit FLOAT NOT NULL DEFAULT 0;
ALTER TABLE billing_billings ADD COLUMN cost_credit_total FLOAT NOT NULL DEFAULT 0;
```

**New table: `billing_allowances`**

One row per account per billing cycle. Preserves full usage history.

```sql
CREATE TABLE billing_allowances (
    id              BINARY(16) PRIMARY KEY,
    customer_id     BINARY(16) NOT NULL,
    account_id      BINARY(16) NOT NULL,
    cycle_start     DATETIME(6) NOT NULL,
    cycle_end       DATETIME(6) NOT NULL,
    tokens_total    INT NOT NULL,
    tokens_used     INT NOT NULL DEFAULT 0,
    tm_create       DATETIME(6) NOT NULL,
    tm_update       DATETIME(6) NOT NULL,
    tm_delete       DATETIME(6) NOT NULL DEFAULT '9999-01-01 00:00:00.000000',

    INDEX idx_billing_allowances_customer_id (customer_id),
    INDEX idx_billing_allowances_account_id (account_id),
    UNIQUE INDEX idx_billing_allowances_account_cycle (account_id, cycle_start)
);
```

Key design points:
- `UNIQUE INDEX (account_id, cycle_start)` ensures idempotent cycle creation.
- `tokens_total` is set from the plan tier but can be updated mid-cycle on plan upgrade/downgrade.
- `tokens_used` is incremented atomically using `SELECT ... FOR UPDATE` transactions.
- Remaining tokens calculation: `max(0, tokens_total - tokens_used)`. Negative values
  (possible after plan downgrade) are treated as zero.

## Billing Flow

### Two-Phase Call Billing

Calls use a two-phase billing process:

**Phase 1 — Call progressing (`call_progressing` event):**
Create a billing record with status `progressing`. At this point, duration is unknown,
so cost fields are zero. The `cost_type` is determined by the classification rules.

**Phase 2 — Call hangup (`call_hangup` event):**
Calculate duration, consume tokens and/or credits, and UPDATE the existing billing
record in a single transaction.

SMS and number billing are single-phase (immediate insert + charge).

### Token-Eligible Event (VN Call, SMS)

```
1. Billable event arrives (e.g., call_hangup for a VN call)

2. Classify the event using the call classification rules.
   Determine the cost_type (call_vn, call_pstn_outgoing, etc.)

3. Calculate billable units:
   - Calls: ceiling(duration_seconds / 60) → minutes
   - SMS: 1 per message

4. Calculate token cost:
   - VN call: cost_unit_count × 1 token/min
   - SMS: cost_unit_count × 10 tokens/msg

5. Find current cycle row for this account:
   WHERE account_id = ? AND cycle_start <= NOW() AND cycle_end > NOW()
   - If no row exists → lazy-create one (see Cycle Creation)

6. Execute in a SINGLE DATABASE TRANSACTION
   (all writes succeed or all roll back):

   Case A: Enough tokens (remaining >= needed)
   → Increment tokens_used by needed amount
   → Update billing record:
     cost_token_total = needed
     cost_credit_total = 0

   Case B: Partial tokens (remaining > 0 but < needed)
   → Increment tokens_used to tokens_total (consume all remaining)
   → Calculate overflow in token units:
     overflow_tokens = tokens_needed - remaining
     overflow_credit = (overflow_tokens / cost_token_per_unit) × cost_credit_per_unit
   → Deduct overflow_credit from account balance
   → Update billing record:
     cost_token_total = remaining
     cost_credit_total = overflow_credit
   Note: For SMS (10 tokens/msg), partial coverage produces fractional credit
   charges. E.g., 3 tokens remaining for 1 SMS (10 tokens): overflow = 7 tokens,
   credit = (7/10) × $0.008 = $0.0056. This is acceptable.

   Case C: No tokens (remaining = 0)
   → Deduct full credits from account balance
   → Update billing record:
     cost_token_total = 0
     cost_credit_total = cost_unit_count × cost_credit_per_unit

   For calls, this UPDATEs the existing progressing-status billing record.
   For SMS, this INSERTs a new billing record.

   All operations (allowance update, balance deduction, billing record write)
   are wrapped in a single transaction with SELECT ... FOR UPDATE on both the
   allowance row and the account row to prevent race conditions.
```

### Credit-Only Event (PSTN Call)

```
1. Call hangup event arrives for a PSTN call.

2. Calculate billable units: ceiling(duration_seconds / 60)

3. Determine rate by cost_type:
   - call_pstn_outgoing: $0.006/min
   - call_pstn_incoming: $0.0045/min

4. Deduct from account balance (atomic, same as current flow).

5. Update billing record:
   cost_token_total = 0
   cost_credit_total = cost_unit_count × cost_credit_per_unit
```

### Balance Validation (IsValidBalance)

The RPC signature remains unchanged — callers pass `ReferenceType` (not `CostType`)
because the caller (e.g., call-manager) does not know the cost type at validation time.
Call-manager only knows it's a `call` — it cannot determine VN vs PSTN vs extension
until billing time when source/destination addresses are available.

Billing-manager maps `ReferenceType` internally:

```
IsValidBalance(ctx, accountID, referenceType, country, count):

  If plan is Unlimited → return true

  If referenceType is "call" or "call_extension":
    1. Check remaining tokens in current cycle: max(0, tokens_total - tokens_used)
    2. If any tokens remain → return true (call might be VN, which uses tokens)
    3. If no tokens → check credit balance using the most expensive
       applicable call rate ($0.006/min outgoing) as a conservative estimate

  If referenceType is "sms":
    1. Check remaining tokens in current cycle
    2. If tokens >= 10 (cost for 1 SMS) → return true
    3. If insufficient tokens → check credit balance at $0.008/msg overflow rate

  If referenceType is "number" or "number_renew":
    → Check credit balance at $5.00/unit

  Return: tokens available OR credits sufficient
```

The pre-flight check is a guard, not the actual billing. It answers "can this customer
use this service at all?" The precise classification and charging happens at billing time.

Token consumption is atomic (SELECT FOR UPDATE on allowance row). If tokens are
exhausted between the `IsValidBalance` check and actual consumption, the call
falls back to credit billing. If credits are also insufficient, the entire
transaction rolls back (same pattern as current credit billing).

## Cycle Management

### Cycle Reset

- Cycles are based on **account anniversary** (the billing account's `tm_create` date).
- No rollover — unused tokens are lost when the cycle ends.
- Each cycle is a new row in `billing_allowances`.

### Cycle Creation

Three mechanisms ensure cycles are always available:

**1. Immediate creation on account creation:**
When the `customer_created` event is handled and a billing account is created, the
first allowance cycle row is also created immediately.

**2. Lazy creation on billing events:**
If a token-eligible billing event arrives and no current cycle row exists (e.g., the
cron missed it or timing gap after anniversary), create the cycle row on-the-fly.
Lazy creation uses the unique index for idempotency — concurrent requests that
discover a missing cycle will attempt INSERT, one succeeds, others get a duplicate
key error and retry the read. This is safe under concurrent load.

**3. Cron job as safety net:**
A periodic background job (every 24 hours, replacing the free tier credit handler)
scans all active accounts (Free, Basic, Professional — excluding Unlimited) and
creates missing cycle rows. This ensures cycles exist even for accounts with no
billing activity.

**Idempotency:** The unique index on `(account_id, cycle_start)` prevents duplicate
cycle rows regardless of which mechanism creates them.

### Plan Changes Mid-Cycle

- **Upgrade** (e.g., Free → Basic): Update the current cycle's `tokens_total`
  immediately. Customer gets the higher allowance for the rest of the cycle.
- **Downgrade** (e.g., Basic → Free): Update `tokens_total` to the lower value.
  If `tokens_used > tokens_total`, remaining is calculated as `max(0, tokens_total - tokens_used)`,
  which yields zero — all further usage overflows to credits. This is acceptable behavior.

## Free Tier Credit Removal

The existing free tier credit system ($1.00/month for free plan accounts) is removed
entirely. The monthly token allowance replaces it. The `credithandler` package and its
cron job are replaced by the allowance cycle handler.

The `credit_free_tier` reference type is deprecated — no new billing records will use it.

## API Endpoints

### Account Response Enhancement

Include token information in the existing account response:

```json
{
  "id": "...",
  "customer_id": "...",
  "plan_type": "basic",
  "balance": 25.50,
  "tokens_total": 10000,
  "tokens_used": 3500,
  "tokens_remaining": 6500,
  "cycle_start": "2026-02-15T00:00:00Z",
  "cycle_end": "2026-03-15T00:00:00Z"
}
```

The token fields are populated from the current active allowance cycle.

### New Endpoint: Allowance History

```
GET /v1/accounts/{id}/allowances
```

Returns a paginated list of allowance cycle records for the account, ordered by
`cycle_start` descending. Useful for usage history dashboards.

```json
{
  "items": [
    {
      "id": "...",
      "account_id": "...",
      "cycle_start": "2026-02-15T00:00:00Z",
      "cycle_end": "2026-03-15T00:00:00Z",
      "tokens_total": 10000,
      "tokens_used": 3500,
      "tm_create": "..."
    },
    {
      "id": "...",
      "account_id": "...",
      "cycle_start": "2026-01-15T00:00:00Z",
      "cycle_end": "2026-02-15T00:00:00Z",
      "tokens_total": 10000,
      "tokens_used": 9200,
      "tm_create": "..."
    }
  ]
}
```

## Events

The following events are published by billing-manager:

| Event                 | Trigger                                      |
|-----------------------|----------------------------------------------|
| `allowance_created`   | New allowance cycle row is created           |
| `allowance_exhausted` | `tokens_used` reaches `tokens_total`         |
| `allowance_low`       | `tokens_used` exceeds 80% of `tokens_total`  |

These events enable downstream services (notification-manager, etc.) to alert customers
about their token usage.

## Files to Change

### New Files

| File | Purpose |
|------|---------|
| `bin-billing-manager/models/allowance/allowance.go` | Allowance model struct |
| `bin-billing-manager/models/allowance/field.go` | Field type for updates |
| `bin-billing-manager/models/billing/cost_type.go` | CostType definition and constants |
| `bin-billing-manager/pkg/allowancehandler/main.go` | Allowance handler interface |
| `bin-billing-manager/pkg/allowancehandler/allowance.go` | Token consumption, cycle creation |
| `bin-billing-manager/pkg/allowancehandler/cycle.go` | Cycle management, cron job |
| `bin-billing-manager/pkg/dbhandler/allowance.go` | Allowance DB operations |
| `bin-dbscheme-manager/bin-manager/main/versions/xxxx_add_billing_allowances.py` | Alembic migration for new table |
| `bin-dbscheme-manager/bin-manager/main/versions/xxxx_update_billing_billings_cost_fields.py` | Alembic migration for billing schema changes |

### Modified Files

| File | Change |
|------|--------|
| `bin-billing-manager/models/billing/billing.go` | Replace old cost fields with new `Cost`-prefixed fields, add `CostType` |
| `bin-billing-manager/models/billing/webhook.go` | Update `WebhookMessage` and `ConvertWebhookMessage` for new cost fields |
| `bin-billing-manager/models/billing/filters.go` | Update `FieldStruct` filter fields for new cost columns |
| `bin-billing-manager/models/billing/field.go` | Update field constants: remove `FieldCostPerUnit`, `FieldCostTotal`, `FieldBillingUnitCount`; add new cost field constants |
| `bin-billing-manager/models/account/plan.go` | Add token limits per plan tier |
| `bin-billing-manager/pkg/billinghandler/billing.go` | Two-phase billing with token-first consumption, single transaction, per-minute units |
| `bin-billing-manager/pkg/billinghandler/event.go` | Call classification logic (direction + cost type determination), update `Create` call to pass `CostType` |
| `bin-billing-manager/pkg/billinghandler/db.go` | Update `Create` function signature to accept `CostType` parameter |
| `bin-billing-manager/pkg/billinghandler/main.go` | Update `BillingHandler` interface for new `Create` signature |
| `bin-billing-manager/pkg/accounthandler/balance.go` | `IsValidBalance` checks tokens + credits, maps `ReferenceType` to token/credit check internally |
| `bin-billing-manager/pkg/dbhandler/billing.go` | Rewrite `BillingSetStatusEnd` (raw SQL references dropped columns), remove `BillingCreditTopUp`, update all billing queries for new cost fields, add new transaction method for token+credit |
| `bin-billing-manager/pkg/dbhandler/main.go` | Add allowance DB methods, remove `BillingCreditTopUp` from interface |
| `bin-billing-manager/cmd/billing-manager/main.go` | Wire up allowance handler, replace credit handler |
| `bin-billing-manager/pkg/listenhandler/main.go` | Add allowance history endpoint |
| `bin-openapi-manager/openapi/openapi.yaml` | Add allowance schema, replace old billing cost fields with new ones (breaking API change) |
| `bin-api-manager/` | Regenerate server code |

### Removed/Replaced

| File | Action |
|------|--------|
| `bin-billing-manager/pkg/credithandler/` | Replace with allowance cycle handler (cron logic reused) |

## Trade-offs

- **Single token pool vs per-type allowances:** Simpler to track and explain to customers.
  Trade-off is that a customer could burn all tokens on one resource type (e.g., all on
  SMS), but this is acceptable.
- **No rollover:** Simpler cycle management. Customers lose unused tokens, which
  incentivizes right-sizing their plan.
- **Lazy cycle creation:** Slightly more complex than cron-only, but prevents unfair
  credit charges during timing gaps. Concurrent lazy creation is safe via unique index
  idempotency.
- **Separate token and credit cost fields:** Adds more columns to billing_billings but
  provides full transparency into what was paid with tokens vs credits in every record.
- **Billing unit fix (seconds → minutes):** Breaking change for existing billing records,
  but corrects an existing bug. Old records remain in seconds; new records use minutes.
  API clients should use `cost_credit_total` and `cost_token_total` for accurate totals
  rather than recalculating from per-unit rates.
- **Cost type as separate field from reference type:** Maintains clean separation between
  resource identification (reference_type) and billing classification (cost_type).
  No changes to existing reference types or the unique index.
- **Single transaction for partial consumption:** Token deduction, credit deduction, and
  billing record update are atomic. If any step fails, all roll back. This prevents
  consuming tokens without completing the billing record.
