# Billing System Refactoring: State + Ledger Pattern

**Date:** 2026-02-15
**Branch:** NOJIRA-Billing-ledger-refactoring
**Status:** Design

## 1. Objective

Unify the billing architecture by removing the standalone `billing_allowances` table. Adopt a State + Ledger pattern where:

- **`billing_accounts`** (State) holds the live balance and token count
- **`billing_billings`** (Ledger) acts as an immutable audit trail recording all transactions

Additionally fix float64 precision issues by migrating to int64.

## 2. Motivation

- **Audit/reconciliation gaps**: Cannot trace how balances reached their current values
- **3-table complexity**: Managing `billing_allowances` + `billing_accounts` + `billing_billings` separately adds unnecessary complexity
- **Float precision issues**: `float64` for monetary values causes rounding errors
- **Future features**: Refunds, adjustments, and top-ups are hard to model on the current architecture

## 3. Architecture

**State (Account):** Live `BalanceToken` (plain int64) + `BalanceCredit` (int64 micros) on `billing_accounts`.

**Ledger (Billing):** Every transaction recorded with signed deltas and post-transaction snapshots. Immutable audit trail.

**Eliminated:** `billing_allowances` table. Monthly resets handled by cron top-up transactions in the ledger.

## 4. Precision Rules

| Field Type | Representation | Example |
|-----------|---------------|---------|
| Credits (money) | int64 micros (1 dollar = 1,000,000) | $0.006/min = 6,000 micros |
| Tokens (allowance) | Plain int64 | 1 token/min for calls, 10 tokens/SMS |
| Duration | int (seconds) | 65 seconds |
| Billable units | int | Ceiling-rounded: (65+59)/60 = 2 minutes |

## 5. Data Models

### 5.1 Billing Struct (Ledger Entry)

```go
package billing

import (
    "time"
    commonidentity "monorepo/bin-common-handler/models/identity"
    "github.com/gofrs/uuid"
)

// TransactionType defines the nature of the ledger entry
type TransactionType string

const (
    TransactionTypeUsage      TransactionType = "usage"       // Consumption (Call, SMS, Number)
    TransactionTypeTopUp      TransactionType = "top_up"      // Monthly allowance reset
    TransactionTypeAdjustment TransactionType = "adjustment"  // Admin manual correction
    TransactionTypeRefund     TransactionType = "refund"      // Credit refund
)

// ReferenceType defines the source of the transaction
type ReferenceType string

const (
    ReferenceTypeCall             ReferenceType = "call"
    ReferenceTypeSMS              ReferenceType = "sms"
    ReferenceTypeNumber           ReferenceType = "number"
    ReferenceTypeMonthlyAllowance ReferenceType = "monthly_allowance"
)

type Billing struct {
    commonidentity.Identity

    AccountID uuid.UUID `json:"account_id" db:"account_id,uuid"`

    // Transaction classification
    TransactionType TransactionType `json:"transaction_type" db:"transaction_type"`
    Status          Status          `json:"status" db:"status"`

    // Source context
    ReferenceType ReferenceType `json:"reference_type" db:"reference_type"`
    ReferenceID   uuid.UUID     `json:"reference_id" db:"reference_id,uuid"`
    CostType      CostType      `json:"cost_type" db:"cost_type"` // Rate tier: call_pstn_outgoing, call_vn, sms, etc.

    // Usage measurement
    UsageDuration int `json:"usage_duration" db:"usage_duration"` // Actual duration in seconds (0 for non-duration items)
    BillableUnits int `json:"billable_units" db:"billable_units"` // Ceiling-rounded minutes for calls, 1 for SMS/numbers

    // Rates
    RateTokenPerUnit  int64 `json:"rate_token_per_unit" db:"rate_token_per_unit"`   // Plain int64 (tokens per billable unit)
    RateCreditPerUnit int64 `json:"rate_credit_per_unit" db:"rate_credit_per_unit"` // int64 micros (credit per billable unit)

    // Ledger delta (signed: usage = negative, top_up/refund = positive)
    AmountToken  int64 `json:"amount_token" db:"amount_token"`   // Plain int64
    AmountCredit int64 `json:"amount_credit" db:"amount_credit"` // int64 micros

    // Post-transaction balance snapshots
    BalanceTokenSnapshot  int64 `json:"balance_token_snapshot" db:"balance_token_snapshot"`
    BalanceCreditSnapshot int64 `json:"balance_credit_snapshot" db:"balance_credit_snapshot"`

    // Idempotency
    IdempotencyKey uuid.UUID `json:"idempotency_key" db:"idempotency_key,uuid"`

    // Billing timeframe
    TMBillingStart *time.Time `json:"tm_billing_start" db:"tm_billing_start"`
    TMBillingEnd   *time.Time `json:"tm_billing_end" db:"tm_billing_end"`

    TMCreate *time.Time `json:"tm_create" db:"tm_create"`
    TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
    TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}
```

### 5.2 Account Struct Updates (State)

New and modified fields on the Account struct:

```go
type Account struct {
    // ... existing fields (ID, CustomerID, Name, Detail, PlanType, PaymentType, PaymentMethod) ...

    // [Live Balance]
    BalanceCredit int64 `json:"balance_credit" db:"balance_credit"` // int64 micros (replaces float64 Balance)
    BalanceToken  int64 `json:"balance_token" db:"balance_token"`   // Plain int64 (replaces allowance cycle)

    // [Top-Up Schedule]
    TmLastTopUp *time.Time `json:"tm_last_topup" db:"tm_last_topup"`
    TmNextTopUp *time.Time `json:"tm_next_topup" db:"tm_next_topup"`

    // PlanType (already exists) determines monthly top-up token amount
}
```

**Deferred to future work:** `LockedCredit` (reservations for active calls), `DailySpendLimit` (fraud circuit breaker).

## 6. Per-Minute Billing Calculation

Billing always rounds up to the next minute (ceiling):

```go
func CalculateBillableUnits(durationSec int) int {
    if durationSec <= 0 {
        return 0
    }
    return (durationSec + 59) / 60
}
```

## 7. Rate Defaults (Unchanged)

| CostType | Token Rate | Credit Rate (micros) |
|----------|-----------|---------------------|
| call_pstn_outgoing | 0 (credit only) | 6,000 ($0.006/min) |
| call_pstn_incoming | 0 (credit only) | 4,500 ($0.0045/min) |
| call_vn | 1/min | 4,500 ($0.0045/min) |
| call_extension | 0 | 0 (free) |
| sms | 10/msg | 8,000 ($0.008/msg) |
| number | 0 (credit only) | 5,000,000 ($5.00/number) |
| number_renew | 0 (credit only) | 5,000,000 ($5.00/number) |

## 8. Transaction Flows

### 8.1 Usage Transaction (Call End)

1. Calculate `UsageDuration` (seconds) and `BillableUnits` (ceiling minutes)
2. `SELECT FOR UPDATE` on account row (row lock)
3. Determine `CostType` from call metadata (PSTN outgoing/incoming, VN, extension)
4. Calculate costs: `tokenCost = BillableUnits * RateTokenPerUnit`, `creditCost = BillableUnits * RateCreditPerUnit`
5. Deduct from `BalanceToken` first; if insufficient, overflow remainder to `BalanceCredit`
6. Insert ledger entry with negative `AmountToken`/`AmountCredit` and post-transaction snapshots
7. Update account `BalanceToken`/`BalanceCredit`
8. Commit transaction

### 8.2 Monthly Top-Up (Cron)

1. Select accounts where `TmNextTopUp <= Now`
2. Per account (in transaction):
   - Set `BalanceToken = PlanTokenLimit` (reset, no rollover)
   - Update `TmLastTopUp = Now`, `TmNextTopUp = first of next month`
   - Insert ledger entry: `TransactionType=top_up`, `ReferenceType=monthly_allowance`, `AmountToken=+plan_limit`, snapshots
3. Commit

### 8.3 IsValidBalance (Pre-flight Check)

Simplified from current 3-table lookup to single account row:

1. Load account (cached in Redis)
2. Unlimited plan = always valid
3. Extension calls = always valid
4. For token-eligible types: check `BalanceToken > 0` OR `BalanceCredit >= estimated cost`
5. For credit-only types (PSTN, number): check `BalanceCredit >= estimated cost`

### 8.4 SMS Billing (Message Created)

1. One billing per target (deterministic UUID v5 reference ID per target)
2. `BillableUnits = 1` per message
3. Token cost = 10 per message, credit cost = 8,000 micros ($0.008)
4. Same atomic deduction flow as call billing

### 8.5 Number Billing (Number Created / Renewed)

1. `BillableUnits = 1` per number
2. Credit-only: 5,000,000 micros ($5.00)
3. For renewals: deterministic reference ID per (number, year-month)

## 9. Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Token precision | Plain int64 (not micros) | Tokens are discrete units (1/min, 10/SMS). Micros adds unnecessary scale. |
| Credit precision | int64 micros | Eliminates float64 rounding for monetary values |
| Rollover policy | Reset (no rollover) | Matches current behavior. `BalanceToken = plan_limit` each month. |
| Top-up mechanism | Cron job | Predictable timing, clean ledger entries |
| Migration strategy | Single cut-over | No customers yet, no need for dual-write complexity |
| LockedCredit / DailySpendLimit | Deferred | Reduces scope; can be added as separate feature |
| CostType field | Kept | Required for rate tier differentiation (PSTN vs VN vs SMS) |
| Snapshots | Included | Essential for audit trail and reconciliation |

## 10. Migration Plan (Single Cut-over)

### 10.1 Schema Migration (Alembic)

**`billing_billings` table changes:**
- Add columns: `transaction_type`, `usage_duration`, `billable_units`, `rate_token_per_unit`, `rate_credit_per_unit`, `amount_token`, `amount_credit`, `balance_token_snapshot`, `balance_credit_snapshot`, `idempotency_key`
- Rename/transform existing cost fields: `cost_unit_count` -> `billable_units`, `cost_token_per_unit` -> `rate_token_per_unit`, etc.
- Remove old float-based cost columns after data conversion

**`billing_accounts` table changes:**
- Add columns: `balance_token` (int64), `tm_last_topup`, `tm_next_topup`
- Convert `balance` (float64) to `balance_credit` (int64 micros): `balance_credit = CAST(balance * 1000000 AS SIGNED)`
- Remove old `balance` column

**Drop `billing_allowances` table:**
- Before dropping: snapshot `tokens_total - tokens_used` into `accounts.balance_token`
- Then drop the table

### 10.2 Code Changes

- **bin-billing-manager/models**: Update Billing and Account structs
- **bin-billing-manager/pkg/billinghandler**: Integrate token deduction into billing flow (remove separate allowancehandler dependency)
- **bin-billing-manager/pkg/accounthandler**: Update balance methods for int64, update IsValidBalance
- **bin-billing-manager/pkg/dbhandler**: New queries for atomic ledger+state updates
- **bin-billing-manager/pkg/allowancehandler**: Remove entirely (logic absorbed into billinghandler)
- **bin-billing-manager/billing-control**: Remove allowance commands, update token management

### 10.3 API / Documentation Updates

- **bin-openapi-manager**: Update billing and account schemas in openapi.yaml
- **bin-api-manager**: Regenerate server code (`go generate ./...`)
- **bin-api-manager/docsdev**: Update RST documentation, rebuild HTML with Sphinx

## 11. Affected Services

| Service | Change Type |
|---------|------------|
| bin-billing-manager | Core refactoring (models, handlers, dbhandler, allowancehandler removal) |
| bin-billing-manager/billing-control | CLI updates (remove allowance commands, update token management) |
| bin-dbscheme-manager | Alembic migration files |
| bin-openapi-manager | Updated billing/account API schemas |
| bin-api-manager | Regenerated server code |
| bin-api-manager/docsdev | Updated RST API docs + rebuilt HTML |
| call-manager, message-manager, number-manager | **No changes** (event contracts unchanged) |

## 12. Testing Strategy

- Unit tests for `CalculateBillableUnits` with edge cases (0, 1, 59, 60, 61 seconds)
- Unit tests for token-first-then-credit overflow logic
- Unit tests for snapshot calculation
- Unit tests for idempotency (duplicate billing prevention)
- Integration tests for atomic ledger+state transaction
- Existing event-processing tests updated for new struct fields
