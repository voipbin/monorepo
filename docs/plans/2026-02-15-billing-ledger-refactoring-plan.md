# Billing Ledger Refactoring Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor the billing system from a 3-table architecture (accounts + billings + allowances) to a State+Ledger pattern (accounts + billings) with int64 precision, transaction types, and balance snapshots.

**Architecture:** Account table holds live state (BalanceToken + BalanceCredit). Billing table becomes an immutable ledger recording every transaction with signed deltas and post-transaction snapshots. The allowances table and handler are eliminated entirely.

**Tech Stack:** Go, MySQL (Alembic migrations), squirrel query builder, gomock, Redis cache, OpenAPI/Sphinx docs

---

## Task 1: Update Billing Model (struct + types + rates)

**Files:**
- Modify: `bin-billing-manager/models/billing/billing.go`
- Modify: `bin-billing-manager/models/billing/cost_type.go`
- Modify: `bin-billing-manager/models/billing/field.go`
- Modify: `bin-billing-manager/models/billing/filters.go`
- Modify: `bin-billing-manager/models/billing/webhook.go`

**Step 1: Update billing.go — replace old struct with new ledger struct**

Replace the entire `Billing` struct (lines 12-36) with:

```go
type Billing struct {
	commonidentity.Identity

	AccountID uuid.UUID `json:"account_id" db:"account_id,uuid"`

	// Transaction classification
	TransactionType TransactionType `json:"transaction_type" db:"transaction_type"`
	Status          Status          `json:"status" db:"status"`

	// Source context
	ReferenceType ReferenceType `json:"reference_type" db:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id" db:"reference_id,uuid"`
	CostType      CostType      `json:"cost_type" db:"cost_type"`

	// Usage measurement
	UsageDuration int `json:"usage_duration" db:"usage_duration"`
	BillableUnits int `json:"billable_units" db:"billable_units"`

	// Rates
	RateTokenPerUnit  int64 `json:"rate_token_per_unit" db:"rate_token_per_unit"`
	RateCreditPerUnit int64 `json:"rate_credit_per_unit" db:"rate_credit_per_unit"`

	// Ledger delta (signed: usage = negative, top_up/refund = positive)
	AmountToken  int64 `json:"amount_token" db:"amount_token"`
	AmountCredit int64 `json:"amount_credit" db:"amount_credit"`

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

Add `TransactionType` type and constants above the struct:

```go
// TransactionType defines the nature of the ledger entry
type TransactionType string

const (
	TransactionTypeUsage      TransactionType = "usage"
	TransactionTypeTopUp      TransactionType = "top_up"
	TransactionTypeAdjustment TransactionType = "adjustment"
	TransactionTypeRefund     TransactionType = "refund"
)
```

Update `ReferenceType` constants (lines 42-50): remove `ReferenceTypeNone`, `ReferenceTypeCallExtension`, `ReferenceTypeCreditFreeTier`, `ReferenceTypeNumberRenew`. Add `ReferenceTypeNumber` and `ReferenceTypeMonthlyAllowance`. Keep `ReferenceTypeCall`, `ReferenceTypeSMS`.

Note: `ReferenceTypeCallExtension` and `ReferenceTypeNumberRenew` are still valid CostType values via `CostType`. They should remain as `ReferenceType` values since they're used in event handlers and the `BillingStart` switch. Keep them.

Final ReferenceType list:
```go
const (
	ReferenceTypeNone              ReferenceType = ""
	ReferenceTypeCall              ReferenceType = "call"
	ReferenceTypeCallExtension     ReferenceType = "call_extension"
	ReferenceTypeSMS               ReferenceType = "sms"
	ReferenceTypeNumber            ReferenceType = "number"
	ReferenceTypeNumberRenew       ReferenceType = "number_renew"
	ReferenceTypeCreditFreeTier    ReferenceType = "credit_free_tier"
	ReferenceTypeMonthlyAllowance  ReferenceType = "monthly_allowance"
)
```

Add `CalculateBillableUnits` function at end of billing.go:

```go
// CalculateBillableUnits returns billable minutes (ceiling-rounded from seconds).
func CalculateBillableUnits(durationSec int) int {
	if durationSec <= 0 {
		return 0
	}
	return (durationSec + 59) / 60
}
```

**Step 2: Update cost_type.go — change rates to int64 micros**

Replace the credit rate constants (lines 19-25) with int64 micros:

```go
// Default credit rates per unit in micros (1 dollar = 1,000,000 micros).
const (
	DefaultCreditPerUnitCallPSTNOutgoing int64 = 6000    // $0.006/min
	DefaultCreditPerUnitCallPSTNIncoming int64 = 4500    // $0.0045/min
	DefaultCreditPerUnitCallVN           int64 = 4500    // $0.0045/min
	DefaultCreditPerUnitSMS              int64 = 8000    // $0.008/msg
	DefaultCreditPerUnitNumber           int64 = 5000000 // $5.00/number
)
```

Change token rate constants to int64 (lines 28-31):
```go
// Default token rates per unit (plain integers).
const (
	DefaultTokenPerUnitCallVN int64 = 1
	DefaultTokenPerUnitSMS    int64 = 10
)
```

Update `GetCostInfo` return types (line 34) from `(tokenPerUnit int, creditPerUnit float32)` to `(tokenPerUnit int64, creditPerUnit int64)`.

**Step 3: Update field.go — replace old cost fields with new ledger fields**

Replace lines 18-23 (the old `FieldCost*` constants) with:

```go
	FieldTransactionType TransactionType = "transaction_type"

	FieldUsageDuration int = "usage_duration"
	FieldBillableUnits int = "billable_units"

	FieldRateTokenPerUnit  Field = "rate_token_per_unit"
	FieldRateCreditPerUnit Field = "rate_credit_per_unit"

	FieldAmountToken  Field = "amount_token"
	FieldAmountCredit Field = "amount_credit"

	FieldBalanceTokenSnapshot  Field = "balance_token_snapshot"
	FieldBalanceCreditSnapshot Field = "balance_credit_snapshot"

	FieldIdempotencyKey Field = "idempotency_key"
```

**Step 4: Update filters.go — replace cost filter fields**

Replace `CostCreditTotal float32` (line 19) with:

```go
	TransactionType TransactionType `filter:"transaction_type"`
	AmountCredit    int64           `filter:"amount_credit"`
```

**Step 5: Update webhook.go — match new struct fields**

Replace the WebhookMessage struct (lines 13-37) and ConvertWebhookMessage (lines 40-65) to match the new Billing struct fields. Replace all `Cost*` fields with the new `TransactionType`, `UsageDuration`, `BillableUnits`, `RateTokenPerUnit`, `RateCreditPerUnit`, `AmountToken`, `AmountCredit`, `BalanceTokenSnapshot`, `BalanceCreditSnapshot`, `IdempotencyKey`.

**Step 6: Run tests to see what breaks**

```bash
cd bin-billing-manager && go build ./...
```

Expected: Compilation errors in handlers that reference old field names. This is expected — we'll fix them in subsequent tasks.

**Step 7: Commit**

```bash
git add bin-billing-manager/models/billing/
git commit -m "NOJIRA-Billing-ledger-refactoring

- bin-billing-manager: Update Billing struct to ledger model with TransactionType, deltas, snapshots
- bin-billing-manager: Change credit rates from float32 to int64 micros
- bin-billing-manager: Add CalculateBillableUnits function
- bin-billing-manager: Update billing field constants, filters, and webhook"
```

---

## Task 2: Update Account Model (struct + fields + webhook)

**Files:**
- Modify: `bin-billing-manager/models/account/account.go`
- Modify: `bin-billing-manager/models/account/field.go`
- Modify: `bin-billing-manager/models/account/filters.go`
- Modify: `bin-billing-manager/models/account/webhook.go`

**Step 1: Update account.go — replace Balance float32 with new fields**

Replace the `Balance float32` field (line 18) with:

```go
	BalanceCredit int64 `json:"balance_credit" db:"balance_credit"`
	BalanceToken  int64 `json:"balance_token" db:"balance_token"`

	TmLastTopUp *time.Time `json:"tm_last_topup" db:"tm_last_topup"`
	TmNextTopUp *time.Time `json:"tm_next_topup" db:"tm_next_topup"`
```

Note: Add `"time"` to the imports if not already present.

**Step 2: Update field.go — replace FieldBalance with new field constants**

Replace `FieldBalance Field = "balance"` (line 16) with:

```go
	FieldBalanceCredit Field = "balance_credit"
	FieldBalanceToken  Field = "balance_token"

	FieldTmLastTopUp Field = "tm_last_topup"
	FieldTmNextTopUp Field = "tm_next_topup"
```

**Step 3: Update filters.go — replace Balance filter**

Replace `Balance float64 \`filter:"balance"\`` (line 12) with:

```go
	BalanceCredit int64 `filter:"balance_credit"`
	BalanceToken  int64 `filter:"balance_token"`
```

**Step 4: Update webhook.go — replace Balance field**

Replace `Balance float32 \`json:"balance"\`` (line 19) with:

```go
	BalanceCredit int64 `json:"balance_credit"`
	BalanceToken  int64 `json:"balance_token"`

	TmLastTopUp *time.Time `json:"tm_last_topup"`
	TmNextTopUp *time.Time `json:"tm_next_topup"`
```

Update `ConvertWebhookMessage()` (line 31) to copy new fields instead of `Balance`.

**Step 5: Commit**

```bash
git add bin-billing-manager/models/account/
git commit -m "NOJIRA-Billing-ledger-refactoring

- bin-billing-manager: Replace Account.Balance (float32) with BalanceCredit (int64 micros) and BalanceToken (int64)
- bin-billing-manager: Add TmLastTopUp and TmNextTopUp fields for cron-based token top-up
- bin-billing-manager: Update account field constants, filters, and webhook"
```

---

## Task 3: Update Test SQL Schemas

**Files:**
- Modify: `bin-billing-manager/scripts/database_scripts_test/table_billing_billings.sql`
- Modify: `bin-billing-manager/scripts/database_scripts_test/table_billing_accounts.sql`
- Delete: `bin-billing-manager/scripts/database_scripts_test/table_billing_allowances.sql` (if it exists)

**Step 1: Rewrite table_billing_billings.sql**

```sql
create table billing_billings(
  id            binary(16),
  customer_id   binary(16),
  account_id    binary(16),

  transaction_type varchar(32),
  status           varchar(32),

  reference_type  varchar(32),
  reference_id    binary(16),

  cost_type             varchar(64),
  usage_duration        integer default 0,
  billable_units        integer default 0,

  rate_token_per_unit   bigint default 0,
  rate_credit_per_unit  bigint default 0,

  amount_token          bigint default 0,
  amount_credit         bigint default 0,

  balance_token_snapshot  bigint default 0,
  balance_credit_snapshot bigint default 0,

  idempotency_key binary(16),

  tm_billing_start  datetime(6),
  tm_billing_end    datetime(6),

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_billing_billings_customer_id on billing_billings(customer_id);
create index idx_billing_billings_account_id on billing_billings(account_id);
create index idx_billing_billings_reference_id on billing_billings(reference_id);
create unique index idx_billings_ref_type_id_active on billing_billings(reference_type, reference_id, tm_delete);
create index idx_billing_billings_create on billing_billings(tm_create);
```

**Step 2: Rewrite table_billing_accounts.sql**

```sql
create table billing_accounts(
  id            binary(16),
  customer_id   binary(16),

  plan_type varchar(255),

  name    varchar(255),
  detail  text,

  balance_credit bigint default 0,
  balance_token  bigint default 0,

  payment_type      varchar(255),
  payment_method    varchar(255),

  tm_last_topup datetime(6),
  tm_next_topup datetime(6),

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_billing_accounts_customer_id on billing_accounts(customer_id);
create index idx_billing_accounts_create on billing_accounts(tm_create);
```

**Step 3: Remove allowances test schema if it exists**

Check if `table_billing_allowances.sql` exists and delete it.

**Step 4: Commit**

```bash
git add bin-billing-manager/scripts/database_scripts_test/
git commit -m "NOJIRA-Billing-ledger-refactoring

- bin-billing-manager: Update test SQL schemas for ledger model (int64 columns, new billing fields)
- bin-billing-manager: Remove allowances test schema"
```

---

## Task 4: Update DBHandler Interface and Billing DB Operations

**Files:**
- Modify: `bin-billing-manager/pkg/dbhandler/main.go`
- Modify: `bin-billing-manager/pkg/dbhandler/billing.go`

**Step 1: Update DBHandler interface in main.go**

Remove ALL `Allowance*` methods (lines 34-39):
```go
// DELETE these lines:
AllowanceCreate(...)
AllowanceGet(...)
AllowanceGetCurrentByAccountID(...)
AllowanceList(...)
AllowanceUpdate(...)
AllowanceConsumeTokens(...)
```

Remove `"monorepo/bin-billing-manager/models/allowance"` import.

Update `BillingSetStatusEndWithCosts` signature. Replace the old signature (line 47):
```go
BillingSetStatusEndWithCosts(ctx context.Context, id uuid.UUID, costUnitCount float32, costTokenTotal int, costCreditTotal float32, tmBillingEnd *time.Time) error
```
with:
```go
BillingSetStatusEnd(ctx context.Context, id uuid.UUID, billableUnits int, usageDuration int, amountToken int64, amountCredit int64, balanceTokenSnapshot int64, balanceCreditSnapshot int64, tmBillingEnd *time.Time) error
```

Update account balance methods — replace float32 with int64:
```go
AccountAddBalance(ctx context.Context, accountID uuid.UUID, amount int64) error
AccountSubtractBalance(ctx context.Context, accountID uuid.UUID, amount int64) error
AccountSubtractBalanceWithCheck(ctx context.Context, accountID uuid.UUID, amount int64) error
```

Add new method for atomic ledger transaction:
```go
BillingConsumeAndRecord(ctx context.Context, bill *billing.Billing, accountID uuid.UUID, billableUnits int, usageDuration int, rateTokenPerUnit int64, rateCreditPerUnit int64, tmBillingEnd *time.Time) (*billing.Billing, error)
```

Add new method for monthly top-up:
```go
AccountTopUpTokens(ctx context.Context, accountID uuid.UUID, customerID uuid.UUID, tokenAmount int64, planType string) error
```

**Step 2: Update billing.go DB operations**

Update `BillingSetStatusEndWithCosts` → `BillingSetStatusEnd` implementation to use the new column names.

Add `BillingConsumeAndRecord` implementation — this is the core atomic transaction:
1. Begin transaction
2. SELECT FOR UPDATE on billing_accounts row
3. Calculate token and credit deductions (token-first, credit-overflow)
4. Update billing_accounts balances
5. Update billing_billings with final amounts and snapshots
6. Commit
7. Invalidate cache

Add `AccountTopUpTokens` implementation for cron:
1. Begin transaction
2. SELECT FOR UPDATE on billing_accounts row
3. Set balance_token = tokenAmount (reset)
4. Update tm_last_topup and tm_next_topup
5. Insert billing_billings ledger entry (transaction_type=top_up)
6. Commit

**Step 3: Update billing DB field references**

Update all squirrel queries in billing.go that reference old column names (`cost_unit_count`, `cost_token_per_unit`, `cost_token_total`, `cost_credit_per_unit`, `cost_credit_total`) to use new column names.

**Step 4: Commit**

```bash
git add bin-billing-manager/pkg/dbhandler/
git commit -m "NOJIRA-Billing-ledger-refactoring

- bin-billing-manager: Remove Allowance methods from DBHandler interface
- bin-billing-manager: Update billing DB ops with new ledger column names
- bin-billing-manager: Add BillingConsumeAndRecord for atomic ledger+state transaction
- bin-billing-manager: Add AccountTopUpTokens for cron-based monthly token reset
- bin-billing-manager: Change balance methods from float32 to int64"
```

---

## Task 5: Update Account DB Operations

**Files:**
- Modify: `bin-billing-manager/pkg/dbhandler/account.go`

**Step 1: Update AccountAddBalance and AccountSubtractBalance**

Change parameter types from `float32` to `int64`. Update the SQL column name from `balance` to `balance_credit`.

**Step 2: Update AccountSubtractBalanceWithCheck**

Change parameter type and column name. The check condition changes from `balance >= ?` to `balance_credit >= ?`.

**Step 3: Update accountGetFromRow**

No change needed — `commondatabasehandler.ScanRow` handles field mapping via db tags.

**Step 4: Commit**

```bash
git add bin-billing-manager/pkg/dbhandler/account.go
git commit -m "NOJIRA-Billing-ledger-refactoring

- bin-billing-manager: Update account DB ops for int64 balance_credit column"
```

---

## Task 6: Remove AllowanceHandler and Allowance DB Operations

**Files:**
- Delete: `bin-billing-manager/pkg/allowancehandler/` (entire directory)
- Delete: `bin-billing-manager/pkg/dbhandler/allowance.go`
- Delete: `bin-billing-manager/models/allowance/` (entire directory)

**Step 1: Delete allowancehandler package**

Remove the entire `bin-billing-manager/pkg/allowancehandler/` directory (main.go, mock_main.go, allowance.go, cycle.go, and all test files).

**Step 2: Delete allowance DB operations**

Remove `bin-billing-manager/pkg/dbhandler/allowance.go`.

**Step 3: Delete allowance model**

Remove the entire `bin-billing-manager/models/allowance/` directory.

**Step 4: Commit**

```bash
git add -A bin-billing-manager/pkg/allowancehandler/ bin-billing-manager/pkg/dbhandler/allowance.go bin-billing-manager/models/allowance/
git commit -m "NOJIRA-Billing-ledger-refactoring

- bin-billing-manager: Remove allowancehandler package (logic absorbed into billinghandler)
- bin-billing-manager: Remove allowance DB operations
- bin-billing-manager: Remove allowance model"
```

---

## Task 7: Update BillingHandler (Core Logic Refactoring)

**Files:**
- Modify: `bin-billing-manager/pkg/billinghandler/main.go`
- Modify: `bin-billing-manager/pkg/billinghandler/billing.go`
- Modify: `bin-billing-manager/pkg/billinghandler/db.go`
- Modify: `bin-billing-manager/pkg/billinghandler/event.go`

**Step 1: Update main.go — remove allowanceHandler dependency**

Remove `allowancehandler` import (line 23). Remove `allowanceHandler` field from struct (line 57). Update `NewBillingHandler` constructor to not accept `allowanceHandler` parameter.

Update `BillingHandler` interface — change `UpdateStatusEnd` signature:
```go
// Old:
UpdateStatusEnd(ctx context.Context, id uuid.UUID, costUnitCount float32, costTokenTotal int, costCreditTotal float32, tmBillingEnd *time.Time) (*billing.Billing, error)

// New:
UpdateStatusEnd(ctx context.Context, id uuid.UUID, billableUnits int, usageDuration int, amountToken int64, amountCredit int64, balanceTokenSnapshot int64, balanceCreditSnapshot int64, tmBillingEnd *time.Time) (*billing.Billing, error)
```

**Step 2: Rewrite billing.go — BillingEnd to use atomic DB transaction**

The core change: `BillingEnd` no longer calls `allowanceHandler.ConsumeTokens()` and `accountHandler.SubtractBalanceWithCheck()` separately. Instead it calls `h.db.BillingConsumeAndRecord()` which does everything atomically.

Rewrite `BillingEnd`:
1. Calculate `usageDuration` (seconds) — for calls: `tmBillingEnd.Sub(*bill.TMBillingStart).Seconds()`; for non-calls: 0
2. Calculate `billableUnits` using `billing.CalculateBillableUnits(usageDuration)` for calls; 1 for SMS/number
3. Call `h.db.BillingConsumeAndRecord(ctx, bill, bill.AccountID, billableUnits, usageDuration, bill.RateTokenPerUnit, bill.RateCreditPerUnit, tmBillingEnd)`
4. Publish update event

**Step 3: Update db.go — Create function with new fields**

Update `Create` to populate `TransactionType: billing.TransactionTypeUsage` and use `int64` types for rates. The `billing.GetCostInfo(costType)` now returns `int64`.

Update `UpdateStatusEnd` to match new signature.

**Step 4: Update event.go — no changes needed to event routing**

The event handlers (`EventCMCallProgressing`, `EventCMCallHangup`, `EventMMMessageCreated`, `EventNMNumberCreated`, `EventNMNumberRenewed`) should not need changes — they call `BillingStart` and `BillingEnd` which handle the new logic internally.

**Step 5: Commit**

```bash
git add bin-billing-manager/pkg/billinghandler/
git commit -m "NOJIRA-Billing-ledger-refactoring

- bin-billing-manager: Remove allowanceHandler dependency from billinghandler
- bin-billing-manager: Rewrite BillingEnd to use atomic BillingConsumeAndRecord
- bin-billing-manager: Update Create to set TransactionType and int64 rates
- bin-billing-manager: Update UpdateStatusEnd for new ledger fields"
```

---

## Task 8: Update AccountHandler

**Files:**
- Modify: `bin-billing-manager/pkg/accounthandler/main.go`
- Modify: `bin-billing-manager/pkg/accounthandler/balance.go`
- Modify: `bin-billing-manager/pkg/accounthandler/account.go`
- Modify: `bin-billing-manager/pkg/accounthandler/db.go`
- Modify: `bin-billing-manager/pkg/accounthandler/event.go`

**Step 1: Update main.go — remove allowanceHandler dependency**

Remove `allowancehandler` import. Remove `allowanceHandler` field from struct. Update `NewAccountHandler` to not accept `allowanceHandler` parameter.

Change balance method signatures from `float32` to `int64`:
```go
SubtractBalance(ctx context.Context, accountID uuid.UUID, amount int64) (*account.Account, error)
SubtractBalanceWithCheck(ctx context.Context, accountID uuid.UUID, amount int64) (*account.Account, error)
AddBalance(ctx context.Context, accountID uuid.UUID, amount int64) (*account.Account, error)
```

**Step 2: Rewrite balance.go — simplify IsValidBalance**

The pre-flight check no longer queries the allowance table. It reads `BalanceToken` and `BalanceCredit` directly from the account struct.

Rewrite `IsValidBalance`:
```go
func (h *accountHandler) IsValidBalance(ctx context.Context, accountID uuid.UUID, billingType billing.ReferenceType, country string, count int) (bool, error) {
	a, err := h.Get(ctx, accountID)
	// ... error handling ...
	// unlimited plan = always valid
	// extension calls = always valid

	switch billingType {
	case billing.ReferenceTypeCall:
		// Optimistic: tokens available OR sufficient credit
		if a.BalanceToken > 0 {
			return true, nil
		}
		expectCost := billing.DefaultCreditPerUnitCallPSTNOutgoing * int64(count)
		if a.BalanceCredit > expectCost {
			return true, nil
		}
	case billing.ReferenceTypeSMS:
		tokensNeeded := billing.DefaultTokenPerUnitSMS * int64(count)
		if a.BalanceToken >= tokensNeeded {
			return true, nil
		}
		expectCost := billing.DefaultCreditPerUnitSMS * int64(count)
		if a.BalanceCredit > expectCost {
			return true, nil
		}
	case billing.ReferenceTypeNumber, billing.ReferenceTypeNumberRenew:
		expectCost := billing.DefaultCreditPerUnitNumber * int64(count)
		if a.BalanceCredit > expectCost {
			return true, nil
		}
	}
	return false, nil
}
```

**Step 3: Update db.go — change float32 → int64 for balance methods**

**Step 4: Commit**

```bash
git add bin-billing-manager/pkg/accounthandler/
git commit -m "NOJIRA-Billing-ledger-refactoring

- bin-billing-manager: Remove allowanceHandler dependency from accounthandler
- bin-billing-manager: Simplify IsValidBalance to read account.BalanceToken/BalanceCredit directly
- bin-billing-manager: Change balance methods from float32 to int64"
```

---

## Task 9: Update ListenHandler (Remove Allowance Endpoints)

**Files:**
- Modify: `bin-billing-manager/pkg/listenhandler/main.go`
- Delete: `bin-billing-manager/pkg/listenhandler/v1_allowance.go`
- Delete: `bin-billing-manager/pkg/listenhandler/v1_allowances.go`
- Modify: test files in `bin-billing-manager/pkg/listenhandler/`

**Step 1: Update main.go — remove allowanceHandler dependency**

Remove `allowancehandler` import and `allowanceHandler` field from struct. Remove regex patterns for allowance endpoints (lines 62-63). Remove allowance request routing from `processRequest`.

Update `NewListenHandler` to not accept `allowanceHandler`.

**Step 2: Delete allowance endpoint files**

Remove `v1_allowance.go` and `v1_allowances.go`.

**Step 3: Commit**

```bash
git add bin-billing-manager/pkg/listenhandler/
git commit -m "NOJIRA-Billing-ledger-refactoring

- bin-billing-manager: Remove allowance endpoints from listen handler
- bin-billing-manager: Remove allowanceHandler dependency from listen handler"
```

---

## Task 10: Update SubscribeHandler and Main Entry Point

**Files:**
- Modify: `bin-billing-manager/cmd/billing-manager/main.go`
- Modify: `bin-billing-manager/pkg/subscribehandler/main.go` (if needed)

**Step 1: Update main.go entry point**

Remove `allowancehandler` import. Remove `allowanceHandler` creation (line 124). Update `NewAccountHandler` call to remove `allowanceHandler` param. Update `NewBillingHandler` call to remove `allowanceHandler` param. Update `runListen` call to remove `allowanceHandler` param.

Replace the allowance cycle processing goroutine (lines 139-157) with a monthly top-up cron that calls `db.AccountTopUpTokens`:

```go
// run monthly token top-up cron
go func() {
	ticker := time.NewTicker(1 * time.Hour) // check hourly
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := runMonthlyTopUp(db); err != nil {
				log.Errorf("Monthly top-up failed. err: %v", err)
			}
		case <-chDone:
			return
		}
	}
}()
```

Add `runMonthlyTopUp` function that queries accounts needing top-up and calls `db.AccountTopUpTokens` for each.

**Step 2: Update runListen signature**

Remove `allowanceHandler` from `runListen` function parameter.

**Step 3: Commit**

```bash
git add bin-billing-manager/cmd/billing-manager/main.go bin-billing-manager/pkg/subscribehandler/
git commit -m "NOJIRA-Billing-ledger-refactoring

- bin-billing-manager: Remove allowanceHandler from service wiring
- bin-billing-manager: Replace allowance cycle processing with hourly top-up cron"
```

---

## Task 11: Update billing-control CLI

**Files:**
- Modify: `bin-billing-manager/cmd/billing-control/main.go`

**Step 1: Remove allowance commands**

Remove the entire `allowance` command group (get, list, process-all, ensure, add-tokens, subtract-tokens).

**Step 2: Update balance commands**

Change `add-balance` and `subtract-balance` to accept int64 (micros) instead of float32.

**Step 3: Add top-up command**

Add a `topup run` command that triggers immediate monthly top-up processing.

**Step 4: Commit**

```bash
git add bin-billing-manager/cmd/billing-control/
git commit -m "NOJIRA-Billing-ledger-refactoring

- bin-billing-manager: Remove allowance CLI commands
- bin-billing-manager: Update balance CLI commands for int64 micros
- bin-billing-manager: Add top-up CLI command"
```

---

## Task 12: Update CacheHandler

**Files:**
- Modify: `bin-billing-manager/pkg/cachehandler/handler.go`
- Modify: `bin-billing-manager/pkg/cachehandler/main.go`

**Step 1: Verify cache serialization**

The cache stores JSON-serialized Account and Billing structs. Since we changed the struct fields, the cache will automatically serialize/deserialize the new fields. Verify there are no hardcoded field references.

**Step 2: Commit (if changes needed)**

```bash
git add bin-billing-manager/pkg/cachehandler/
git commit -m "NOJIRA-Billing-ledger-refactoring

- bin-billing-manager: Verify cache handler works with new struct fields"
```

---

## Task 13: Regenerate Mocks and Fix All Tests

**Step 1: Regenerate all mocks**

```bash
cd bin-billing-manager && go generate ./...
```

**Step 2: Fix model tests**

Update tests in `models/billing/billing_test.go`, `models/billing/cost_type_test.go`, `models/account/account_test.go`, etc. to use new field names and types.

Add tests for `CalculateBillableUnits`:
```go
func TestCalculateBillableUnits(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		expected int
	}{
		{"zero", 0, 0},
		{"negative", -5, 0},
		{"one_second", 1, 1},
		{"59_seconds", 59, 1},
		{"60_seconds", 60, 1},
		{"61_seconds", 61, 2},
		{"120_seconds", 120, 2},
		{"121_seconds", 121, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := billing.CalculateBillableUnits(tt.seconds)
			if result != tt.expected {
				t.Errorf("CalculateBillableUnits(%d) = %d, want %d", tt.seconds, result, tt.expected)
			}
		})
	}
}
```

**Step 3: Fix handler tests**

Update tests in `billinghandler/`, `accounthandler/`, `listenhandler/`, `subscribehandler/` to:
- Remove all references to `AllowanceHandler` mock
- Update billing struct field references
- Update float32 amounts to int64

**Step 4: Fix DB handler tests**

Update tests in `dbhandler/billing_test.go` and `dbhandler/account_test.go` to use new column names and types. Remove `dbhandler/allowance_test.go` (already deleted with the model).

**Step 5: Run full test suite**

```bash
cd bin-billing-manager && go test ./...
```

Expected: All tests pass.

**Step 6: Commit**

```bash
git add bin-billing-manager/
git commit -m "NOJIRA-Billing-ledger-refactoring

- bin-billing-manager: Regenerate all mocks for updated interfaces
- bin-billing-manager: Add CalculateBillableUnits tests
- bin-billing-manager: Fix all handler and DB tests for ledger model
- bin-billing-manager: Remove allowance tests"
```

---

## Task 14: Run Full Verification

**Step 1: Full verification workflow**

```bash
cd bin-billing-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All steps pass cleanly.

**Step 2: Fix any issues found**

**Step 3: Commit any fixes**

```bash
git add bin-billing-manager/
git commit -m "NOJIRA-Billing-ledger-refactoring

- bin-billing-manager: Fix issues found during full verification"
```

---

## Task 15: Create Alembic Migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<hash>_billing_ledger_refactoring.py`

**Step 1: Create migration file**

```bash
cd bin-dbscheme-manager/bin-manager/main && alembic -c alembic.ini revision -m "billing_ledger_refactoring"
```

**Step 2: Write the migration**

The `upgrade()` function:

```python
def upgrade():
    # 1. billing_billings: Add new columns
    op.add_column('billing_billings', sa.Column('transaction_type', sa.String(32), server_default='usage'))
    op.add_column('billing_billings', sa.Column('usage_duration', sa.Integer(), server_default='0'))
    op.add_column('billing_billings', sa.Column('billable_units', sa.Integer(), server_default='0'))
    op.add_column('billing_billings', sa.Column('rate_token_per_unit', sa.BigInteger(), server_default='0'))
    op.add_column('billing_billings', sa.Column('rate_credit_per_unit', sa.BigInteger(), server_default='0'))
    op.add_column('billing_billings', sa.Column('amount_token', sa.BigInteger(), server_default='0'))
    op.add_column('billing_billings', sa.Column('amount_credit', sa.BigInteger(), server_default='0'))
    op.add_column('billing_billings', sa.Column('balance_token_snapshot', sa.BigInteger(), server_default='0'))
    op.add_column('billing_billings', sa.Column('balance_credit_snapshot', sa.BigInteger(), server_default='0'))
    op.add_column('billing_billings', sa.Column('idempotency_key', sa.LargeBinary(16)))

    # 2. billing_billings: Convert existing data
    op.execute("""
        UPDATE billing_billings SET
            billable_units = CEIL(cost_unit_count),
            rate_token_per_unit = cost_token_per_unit,
            rate_credit_per_unit = CAST(cost_credit_per_unit * 1000000 AS SIGNED),
            amount_token = -cost_token_total,
            amount_credit = -CAST(cost_credit_total * 1000000 AS SIGNED)
    """)

    # 3. billing_billings: Drop old columns
    op.drop_column('billing_billings', 'cost_unit_count')
    op.drop_column('billing_billings', 'cost_token_per_unit')
    op.drop_column('billing_billings', 'cost_token_total')
    op.drop_column('billing_billings', 'cost_credit_per_unit')
    op.drop_column('billing_billings', 'cost_credit_total')

    # 4. billing_accounts: Add new columns
    op.add_column('billing_accounts', sa.Column('balance_credit', sa.BigInteger(), server_default='0'))
    op.add_column('billing_accounts', sa.Column('balance_token', sa.BigInteger(), server_default='0'))
    op.add_column('billing_accounts', sa.Column('tm_last_topup', sa.DateTime(timezone=False)))
    op.add_column('billing_accounts', sa.Column('tm_next_topup', sa.DateTime(timezone=False)))

    # 5. billing_accounts: Convert balance to balance_credit (micros)
    op.execute("""
        UPDATE billing_accounts SET
            balance_credit = CAST(balance * 1000000 AS SIGNED)
    """)

    # 6. billing_accounts: Snapshot allowance tokens into balance_token
    op.execute("""
        UPDATE billing_accounts a
        LEFT JOIN billing_allowances al ON a.id = al.account_id
            AND al.cycle_start <= NOW()
            AND al.cycle_end > NOW()
            AND al.tm_delete IS NULL
        SET a.balance_token = COALESCE(al.tokens_total - al.tokens_used, 0)
    """)

    # 7. billing_accounts: Drop old balance column
    op.drop_column('billing_accounts', 'balance')

    # 8. Drop billing_allowances table
    op.drop_table('billing_allowances')
```

The `downgrade()` function should reverse these operations.

**Step 3: Commit**

```bash
git add bin-dbscheme-manager/
git commit -m "NOJIRA-Billing-ledger-refactoring

- bin-dbscheme-manager: Add Alembic migration for billing ledger refactoring
- bin-dbscheme-manager: Convert billing_billings to ledger model with int64 columns
- bin-dbscheme-manager: Convert billing_accounts balance to int64 micros
- bin-dbscheme-manager: Migrate allowance tokens to account balance_token
- bin-dbscheme-manager: Drop billing_allowances table"
```

---

## Task 16: Update OpenAPI Schemas

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml`

**Step 1: Update BillingManagerBilling schema (lines 444-498)**

Replace old cost fields with new ledger fields:
- Remove: `cost_unit_count`, `cost_token_per_unit`, `cost_token_total`, `cost_credit_per_unit`, `cost_credit_total`
- Add: `transaction_type`, `usage_duration`, `billable_units`, `rate_token_per_unit`, `rate_credit_per_unit`, `amount_token`, `amount_credit`, `balance_token_snapshot`, `balance_credit_snapshot`, `idempotency_key`

Add `BillingManagerBillingTransactionType` enum (usage, top_up, adjustment, refund).

Update `BillingManagerBillingreferenceType` enum — add `monthly_allowance`.

**Step 2: Update BillingManagerAccount schema (lines 378-412)**

Replace `balance` (number/float) with:
- `balance_credit` (integer/int64)
- `balance_token` (integer/int64)
- `tm_last_topup` (string/date-time)
- `tm_next_topup` (string/date-time)

**Step 3: Remove BillingManagerAllowance schema (lines 501-533)**

Delete the entire `BillingManagerAllowance` schema definition.

Remove allowance endpoint paths:
- Delete `billing_accounts/id_allowance.yaml`
- Delete `billing_accounts/id_allowances.yaml`

**Step 4: Regenerate OpenAPI types**

```bash
cd bin-openapi-manager && go generate ./...
```

**Step 5: Regenerate api-manager server code**

```bash
cd bin-api-manager && go generate ./...
```

**Step 6: Run verification for both**

```bash
cd bin-openapi-manager && go mod tidy && go mod vendor && go test ./... && golangci-lint run -v --timeout 5m
cd bin-api-manager && go mod tidy && go mod vendor && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 7: Commit**

```bash
git add bin-openapi-manager/ bin-api-manager/
git commit -m "NOJIRA-Billing-ledger-refactoring

- bin-openapi-manager: Update billing and account schemas for ledger model
- bin-openapi-manager: Remove allowance schema and endpoints
- bin-api-manager: Regenerate server code from updated OpenAPI spec"
```

---

## Task 17: Update RST Documentation

**Files:**
- Modify: `bin-api-manager/docsdev/source/billing_account_overview.rst`
- Modify: `bin-api-manager/docsdev/source/billing_account_struct.rst`

**Step 1: Update billing_account_overview.rst**

Rewrite to describe the State+Ledger architecture:
- Account holds live `balance_credit` (int64 micros) and `balance_token` (int64)
- Billings act as immutable ledger entries
- Monthly token top-up via cron (not cycle-based)
- Remove all references to allowance cycles

**Step 2: Update billing_account_struct.rst**

Update struct documentation:
- Remove Allowance struct section
- Update Account struct: `balance` → `balance_credit` + `balance_token` + `tm_last_topup` + `tm_next_topup`
- Update Billing struct: document new fields (transaction_type, deltas, snapshots)

**Step 3: Rebuild HTML docs**

```bash
cd bin-api-manager/docsdev && python3 -m sphinx -M html source build
```

**Step 4: Commit (force-add build directory)**

```bash
git add bin-api-manager/docsdev/source/
git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-Billing-ledger-refactoring

- bin-api-manager: Update RST docs for State+Ledger billing architecture
- bin-api-manager: Rebuild HTML documentation"
```

---

## Task 18: Final Verification and Cleanup

**Step 1: Run full verification for all affected services**

```bash
cd bin-billing-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Verify no dangling references to allowance**

```bash
grep -r "allowance" bin-billing-manager/ --include="*.go" | grep -v "_test.go" | grep -v "vendor/"
```

Expected: No results (all allowance references removed).

**Step 3: Check for stale imports**

```bash
grep -r "models/allowance" bin-billing-manager/ --include="*.go" | grep -v "vendor/"
grep -r "allowancehandler" bin-billing-manager/ --include="*.go" | grep -v "vendor/"
```

Expected: No results.

**Step 4: Final commit if any cleanup needed**

```bash
git add -A
git commit -m "NOJIRA-Billing-ledger-refactoring

- bin-billing-manager: Final cleanup and verification"
```

---

## Dependency Order

```
Task 1 (billing model) ──┐
Task 2 (account model) ──┼── Task 3 (test schemas) ── Task 4 (DBHandler) ── Task 5 (account DB)
                          │                                    │
                          │                           Task 6 (remove allowance)
                          │                                    │
                          ├── Task 7 (billinghandler) ── Task 8 (accounthandler)
                          │                                    │
                          ├── Task 9 (listenhandler) ── Task 10 (main + subscribe)
                          │                                    │
                          ├── Task 11 (CLI) ── Task 12 (cache)
                          │                          │
                          └── Task 13 (mocks + tests) ── Task 14 (full verification)
                                                               │
                                                    Task 15 (migration) ── Task 16 (OpenAPI)
                                                                                │
                                                                    Task 17 (RST docs) ── Task 18 (final)
```

**Tasks 1-2 can run in parallel.** Tasks 4-6 can be partially parallelized. Task 13 (tests) depends on all handler changes being complete.
