# Billing CostMode and Email Billing Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Introduce CostMode enum for explicit free/disabled/credit-only/token-first billing, update rates, and add email billing integration.

**Architecture:** Add CostMode enum + CostInfo struct to billing models, update the deduction algorithm and balance validation to use mode-based branching, wire email-manager events into billing-manager's subscription pipeline, and add pre-send balance checking to email-manager.

**Tech Stack:** Go, RabbitMQ events, gomock for testing

---

### Task 1: Add CostMode enum, CostInfo struct, CostTypeEmail, and update GetCostInfo

**Files:**
- Modify: `bin-billing-manager/models/billing/cost_type.go`

**Step 1: Update `cost_type.go` with CostMode, CostInfo, and new rates**

Replace the entire file content with:

```go
package billing

// CostMode declares how a cost type is charged.
type CostMode int

const (
	CostModeDisabled   CostMode = iota // Service not available — requests rejected
	CostModeFree                        // Allowed, no charge
	CostModeCreditOnly                  // Credit only, tokens not accepted
	CostModeTokenFirst                  // Token first, overflow to credits
)

// CostInfo holds the billing mode and rates for a cost type.
type CostInfo struct {
	Mode          CostMode
	TokenPerUnit  int64
	CreditPerUnit int64
}

// CostType classifies why a billing cost was applied.
type CostType string

const (
	CostTypeNone             CostType = ""
	CostTypeCallPSTNOutgoing CostType = "call_pstn_outgoing"
	CostTypeCallPSTNIncoming CostType = "call_pstn_incoming"
	CostTypeCallVN           CostType = "call_vn"
	CostTypeCallExtension    CostType = "call_extension"
	CostTypeCallDirectExt    CostType = "call_direct_ext"
	CostTypeSMS              CostType = "sms"
	CostTypeEmail            CostType = "email"
	CostTypeNumber           CostType = "number"
	CostTypeNumberRenew      CostType = "number_renew"
)

// Default credit rates per unit in micros (1 dollar = 1,000,000 micros).
const (
	DefaultCreditPerUnitCallPSTNOutgoing int64 = 10000   // $0.01/min
	DefaultCreditPerUnitCallPSTNIncoming int64 = 10000   // $0.01/min
	DefaultCreditPerUnitCallVN           int64 = 1000    // $0.001/min
	DefaultCreditPerUnitSMS              int64 = 10000   // $0.01/msg
	DefaultCreditPerUnitEmail            int64 = 10000   // $0.01/msg
	DefaultCreditPerUnitNumber           int64 = 5000000 // $5.00/number
)

// Default token rates per unit (plain integers).
const (
	DefaultTokenPerUnitCallVN int64 = 1
)

// GetCostInfo returns the billing mode and rates for a given cost type.
func GetCostInfo(ct CostType) CostInfo {
	switch ct {
	case CostTypeCallPSTNOutgoing:
		return CostInfo{CostModeCreditOnly, 0, DefaultCreditPerUnitCallPSTNOutgoing}
	case CostTypeCallPSTNIncoming:
		return CostInfo{CostModeCreditOnly, 0, DefaultCreditPerUnitCallPSTNIncoming}
	case CostTypeCallVN:
		return CostInfo{CostModeTokenFirst, DefaultTokenPerUnitCallVN, DefaultCreditPerUnitCallVN}
	case CostTypeCallExtension, CostTypeCallDirectExt:
		return CostInfo{CostModeFree, 0, 0}
	case CostTypeSMS:
		return CostInfo{CostModeCreditOnly, 0, DefaultCreditPerUnitSMS}
	case CostTypeEmail:
		return CostInfo{CostModeCreditOnly, 0, DefaultCreditPerUnitEmail}
	case CostTypeNumber, CostTypeNumberRenew:
		return CostInfo{CostModeCreditOnly, 0, DefaultCreditPerUnitNumber}
	default:
		return CostInfo{CostModeDisabled, 0, 0}
	}
}
```

**Step 2: Update `cost_type_test.go` to test new GetCostInfo return type**

Replace the entire file content with:

```go
package billing

import (
	"testing"
)

func Test_GetCostInfo(t *testing.T) {

	tests := []struct {
		name string

		costType CostType

		expectMode          CostMode
		expectTokenPerUnit  int64
		expectCreditPerUnit int64
	}{
		{
			name:                "call_pstn_outgoing - credit only",
			costType:            CostTypeCallPSTNOutgoing,
			expectMode:          CostModeCreditOnly,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: DefaultCreditPerUnitCallPSTNOutgoing,
		},
		{
			name:                "call_pstn_incoming - credit only",
			costType:            CostTypeCallPSTNIncoming,
			expectMode:          CostModeCreditOnly,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: DefaultCreditPerUnitCallPSTNIncoming,
		},
		{
			name:                "call_vn - token first",
			costType:            CostTypeCallVN,
			expectMode:          CostModeTokenFirst,
			expectTokenPerUnit:  DefaultTokenPerUnitCallVN,
			expectCreditPerUnit: DefaultCreditPerUnitCallVN,
		},
		{
			name:                "call_extension - free",
			costType:            CostTypeCallExtension,
			expectMode:          CostModeFree,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: 0,
		},
		{
			name:                "call_direct_ext - free",
			costType:            CostTypeCallDirectExt,
			expectMode:          CostModeFree,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: 0,
		},
		{
			name:                "sms - credit only",
			costType:            CostTypeSMS,
			expectMode:          CostModeCreditOnly,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: DefaultCreditPerUnitSMS,
		},
		{
			name:                "email - credit only",
			costType:            CostTypeEmail,
			expectMode:          CostModeCreditOnly,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: DefaultCreditPerUnitEmail,
		},
		{
			name:                "number - credit only",
			costType:            CostTypeNumber,
			expectMode:          CostModeCreditOnly,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: DefaultCreditPerUnitNumber,
		},
		{
			name:                "number_renew - credit only",
			costType:            CostTypeNumberRenew,
			expectMode:          CostModeCreditOnly,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: DefaultCreditPerUnitNumber,
		},
		{
			name:                "none - disabled",
			costType:            CostTypeNone,
			expectMode:          CostModeDisabled,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: 0,
		},
		{
			name:                "unknown - disabled",
			costType:            CostType("unknown"),
			expectMode:          CostModeDisabled,
			expectTokenPerUnit:  0,
			expectCreditPerUnit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCostInfo(tt.costType)
			if got.Mode != tt.expectMode {
				t.Errorf("Wrong Mode. expect: %d, got: %d", tt.expectMode, got.Mode)
			}
			if got.TokenPerUnit != tt.expectTokenPerUnit {
				t.Errorf("Wrong TokenPerUnit. expect: %d, got: %d", tt.expectTokenPerUnit, got.TokenPerUnit)
			}
			if got.CreditPerUnit != tt.expectCreditPerUnit {
				t.Errorf("Wrong CreditPerUnit. expect: %d, got: %d", tt.expectCreditPerUnit, got.CreditPerUnit)
			}
		})
	}
}
```

**Step 3: Update `billing_test.go` — remove the duplicate `TestGetCostInfo` function**

The `TestGetCostInfo` function in `bin-billing-manager/models/billing/billing_test.go` (lines 172-201) duplicates the test in `cost_type_test.go`. Delete lines 172-201 from `billing_test.go`.

**Step 4: Run tests to verify**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing/bin-billing-manager && go test ./models/billing/...`
Expected: PASS

**Step 5: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing
git add bin-billing-manager/models/billing/cost_type.go bin-billing-manager/models/billing/cost_type_test.go bin-billing-manager/models/billing/billing_test.go
git commit -m "NOJIRA-billing-cost-mode-email-billing

- bin-billing-manager: Add CostMode enum (Disabled/Free/CreditOnly/TokenFirst) and CostInfo struct
- bin-billing-manager: Add CostTypeEmail cost type
- bin-billing-manager: Update GetCostInfo to return CostInfo instead of bare ints
- bin-billing-manager: Update rates: SMS $0.01/msg, PSTN Out/In $0.01/min
- bin-billing-manager: Remove DefaultTokenPerUnitSMS (SMS is now credit-only)"
```

---

### Task 2: Add ReferenceTypeEmail and update BillingStart for email

**Files:**
- Modify: `bin-billing-manager/models/billing/billing.go`
- Modify: `bin-billing-manager/pkg/billinghandler/billing.go`

**Step 1: Add `ReferenceTypeEmail` in `billing.go`**

In `bin-billing-manager/models/billing/billing.go`, add `ReferenceTypeEmail` to the reference types (after `ReferenceTypeSMS`):

```go
ReferenceTypeEmail        ReferenceType = "email"
```

**Step 2: Update `BillingStart` in `billinghandler/billing.go` to handle email**

In `bin-billing-manager/pkg/billinghandler/billing.go`, the `BillingStart` function's `flagEnd` switch (lines 69-79) needs to include email as an immediate-end type:

Change lines 72-73 from:
```go
	case billing.ReferenceTypeSMS:
		flagEnd = true
```
to:
```go
	case billing.ReferenceTypeSMS, billing.ReferenceTypeEmail:
		flagEnd = true
```

Also update the idempotency retry block (lines 51-56) to include email in the immediate-end set:

Change line 52 from:
```go
		case billing.ReferenceTypeSMS, billing.ReferenceTypeNumber, billing.ReferenceTypeNumberRenew:
```
to:
```go
		case billing.ReferenceTypeSMS, billing.ReferenceTypeEmail, billing.ReferenceTypeNumber, billing.ReferenceTypeNumberRenew:
```

**Step 3: Update `BillingEnd` in `billinghandler/billing.go` to handle email billable units**

In `bin-billing-manager/pkg/billinghandler/billing.go`, the `BillingEnd` function's billable units switch (lines 140-152) needs to include email:

Change line 146 from:
```go
	case billing.ReferenceTypeSMS, billing.ReferenceTypeNumber, billing.ReferenceTypeNumberRenew:
```
to:
```go
	case billing.ReferenceTypeSMS, billing.ReferenceTypeEmail, billing.ReferenceTypeNumber, billing.ReferenceTypeNumberRenew:
```

**Step 4: Run tests**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing/bin-billing-manager && go test ./models/billing/... ./pkg/billinghandler/...`
Expected: Compilation may fail due to callers of `GetCostInfo` expecting two return values. That's expected — Task 3 fixes it.

**Step 5: Commit** (defer to after Task 3 if compilation fails)

---

### Task 3: Update billinghandler Create to use CostInfo

**Files:**
- Modify: `bin-billing-manager/pkg/billinghandler/db.go`

**Step 1: Update `Create` function in `billinghandler/db.go`**

In `bin-billing-manager/pkg/billinghandler/db.go`, line 36:

Change:
```go
	tokenPerUnit, creditPerUnit := billing.GetCostInfo(costType)
```
to:
```go
	costInfo := billing.GetCostInfo(costType)
```

And lines 50-51:
```go
		RateTokenPerUnit:  tokenPerUnit,
		RateCreditPerUnit: creditPerUnit,
```
to:
```go
		RateTokenPerUnit:  costInfo.TokenPerUnit,
		RateCreditPerUnit: costInfo.CreditPerUnit,
```

**Step 2: Run tests**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing/bin-billing-manager && go build ./...`
Expected: PASS (all `GetCostInfo` callers now use new return type)

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing/bin-billing-manager && go test ./pkg/billinghandler/...`
Expected: Tests may fail due to changed rate constants (SMS was $0.008, now $0.01). Update test expectations as needed.

**Step 3: Commit** (combined with Task 2 if deferred)

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing
git add bin-billing-manager/models/billing/billing.go bin-billing-manager/pkg/billinghandler/billing.go bin-billing-manager/pkg/billinghandler/db.go
git commit -m "NOJIRA-billing-cost-mode-email-billing

- bin-billing-manager: Add ReferenceTypeEmail reference type
- bin-billing-manager: Update BillingStart/BillingEnd to handle email as immediate-end type
- bin-billing-manager: Update Create to use CostInfo struct from GetCostInfo"
```

---

### Task 4: Update deduction algorithm to use CostInfo

**Files:**
- Modify: `bin-billing-manager/pkg/dbhandler/billing.go`
- Modify: `bin-billing-manager/pkg/dbhandler/main.go` (interface)
- Modify: `bin-billing-manager/pkg/dbhandler/deduction_test.go`

**Step 1: Update `CalculateTokenCreditDeduction` signature**

In `bin-billing-manager/pkg/dbhandler/billing.go`, replace the function at lines 355-378 with:

```go
// CalculateTokenCreditDeduction computes how many tokens and credits to deduct
// for a given billing operation using the cost type's mode.
func CalculateTokenCreditDeduction(balanceToken int64, billableUnits int, costInfo billing.CostInfo) DeductionResult {
	if billableUnits <= 0 {
		return DeductionResult{}
	}

	switch costInfo.Mode {
	case billing.CostModeFree, billing.CostModeDisabled:
		return DeductionResult{}

	case billing.CostModeCreditOnly:
		return DeductionResult{
			TokenDeducted:  0,
			CreditDeducted: int64(billableUnits) * costInfo.CreditPerUnit,
		}

	case billing.CostModeTokenFirst:
		totalTokenCost := int64(billableUnits) * costInfo.TokenPerUnit
		if totalTokenCost > 0 && balanceToken > 0 {
			if balanceToken >= totalTokenCost {
				return DeductionResult{TokenDeducted: totalTokenCost, CreditDeducted: 0}
			}
			fullUnitsInTokens := balanceToken / costInfo.TokenPerUnit
			tokenDeducted := fullUnitsInTokens * costInfo.TokenPerUnit
			remainingUnits := int64(billableUnits) - fullUnitsInTokens
			creditDeducted := remainingUnits * costInfo.CreditPerUnit
			return DeductionResult{TokenDeducted: tokenDeducted, CreditDeducted: creditDeducted}
		}
		return DeductionResult{TokenDeducted: 0, CreditDeducted: int64(billableUnits) * costInfo.CreditPerUnit}
	}

	return DeductionResult{}
}
```

**Step 2: Update `BillingConsumeAndRecord` signature**

In `bin-billing-manager/pkg/dbhandler/billing.go`, change the function signature at line 381 from:

```go
func (h *handler) BillingConsumeAndRecord(ctx context.Context, bill *billing.Billing, accountID uuid.UUID, billableUnits int, usageDuration int, rateTokenPerUnit int64, rateCreditPerUnit int64, tmBillingEnd *time.Time) (*billing.Billing, error) {
```
to:
```go
func (h *handler) BillingConsumeAndRecord(ctx context.Context, bill *billing.Billing, accountID uuid.UUID, billableUnits int, usageDuration int, costInfo billing.CostInfo, tmBillingEnd *time.Time) (*billing.Billing, error) {
```

Update the call to `CalculateTokenCreditDeduction` at line 401 from:
```go
	d := CalculateTokenCreditDeduction(balanceToken, billableUnits, rateTokenPerUnit, rateCreditPerUnit)
```
to:
```go
	d := CalculateTokenCreditDeduction(balanceToken, billableUnits, costInfo)
```

Update the billing record UPDATE SQL at lines 439-440 from:
```go
		rateTokenPerUnit,
		rateCreditPerUnit,
```
to:
```go
		costInfo.TokenPerUnit,
		costInfo.CreditPerUnit,
```

**Step 3: Update the interface in `dbhandler/main.go`**

In `bin-billing-manager/pkg/dbhandler/main.go`, line 45, change:
```go
	BillingConsumeAndRecord(ctx context.Context, bill *billing.Billing, accountID uuid.UUID, billableUnits int, usageDuration int, rateTokenPerUnit int64, rateCreditPerUnit int64, tmBillingEnd *time.Time) (*billing.Billing, error)
```
to:
```go
	BillingConsumeAndRecord(ctx context.Context, bill *billing.Billing, accountID uuid.UUID, billableUnits int, usageDuration int, costInfo billing.CostInfo, tmBillingEnd *time.Time) (*billing.Billing, error)
```

**Step 4: Update the caller in `billinghandler/billing.go`**

In `bin-billing-manager/pkg/billinghandler/billing.go`, line 155, change:
```go
	res, err := h.db.BillingConsumeAndRecord(ctx, bill, bill.AccountID, billableUnits, usageDuration, bill.RateTokenPerUnit, bill.RateCreditPerUnit, tmBillingEnd)
```
to:
```go
	costInfo := billing.GetCostInfo(bill.CostType)
	res, err := h.db.BillingConsumeAndRecord(ctx, bill, bill.AccountID, billableUnits, usageDuration, costInfo, tmBillingEnd)
```

**Step 5: Update `deduction_test.go` to use CostInfo**

Replace the test file with updated test struct and calls. The test struct changes from:
```go
	rateTokenPerUnit int64
	rateCreditPerUnit int64
```
to:
```go
	costInfo billing.CostInfo
```

And each test case needs its `rateTokenPerUnit`/`rateCreditPerUnit` fields changed to `costInfo`. The call site at line 262 changes from:
```go
	result := CalculateTokenCreditDeduction(tt.balanceToken, tt.billableUnits, tt.rateTokenPerUnit, tt.rateCreditPerUnit)
```
to:
```go
	result := CalculateTokenCreditDeduction(tt.balanceToken, tt.billableUnits, tt.costInfo)
```

For test cases that used `rateTokenPerUnit: 0, rateCreditPerUnit: 6000`:
- If they test "credit only" behavior → use `CostInfo{Mode: billing.CostModeCreditOnly, CreditPerUnit: 6000}`
- If they test "both rates" behavior → use `CostInfo{Mode: billing.CostModeTokenFirst, TokenPerUnit: 10, CreditPerUnit: 4500}`
- If they test "zero rates" → use `CostInfo{Mode: billing.CostModeFree}`

Add new test cases for:
```go
{
	name:        "disabled mode - zero deduction",
	balanceToken: 100,
	billableUnits: 5,
	costInfo:     billing.CostInfo{Mode: billing.CostModeDisabled},
	expectTokenDeducted: 0,
	expectCreditDeducted: 0,
},
{
	name:        "free mode - zero deduction",
	balanceToken: 100,
	billableUnits: 5,
	costInfo:     billing.CostInfo{Mode: billing.CostModeFree},
	expectTokenDeducted: 0,
	expectCreditDeducted: 0,
},
{
	name:        "email credit only at $0.01",
	balanceToken: 50,
	billableUnits: 1,
	costInfo:     billing.CostInfo{Mode: billing.CostModeCreditOnly, CreditPerUnit: 10000},
	expectTokenDeducted: 0,
	expectCreditDeducted: 10000,
},
```

**Step 6: Regenerate mocks**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing/bin-billing-manager && go generate ./pkg/dbhandler/...`

**Step 7: Update all test files that call `BillingConsumeAndRecord` with old signature**

Test files to update (replace separate `rateTokenPerUnit, rateCreditPerUnit` args with `billing.CostInfo{...}`):
- `bin-billing-manager/pkg/billinghandler/event_test.go` — all `mockDB.EXPECT().BillingConsumeAndRecord(...)` calls
- `bin-billing-manager/pkg/billinghandler/billing_test.go` — all `mockDB.EXPECT().BillingConsumeAndRecord(...)` calls

In each case, replace the two rate args (e.g., `billing.DefaultTokenPerUnitSMS, billing.DefaultCreditPerUnitSMS`) with a single `gomock.Any()` or the matching `billing.CostInfo{...}`.

For SMS tests, the old pattern:
```go
mockDB.EXPECT().BillingConsumeAndRecord(ctx, tt.responseBilling, tt.responseBilling.AccountID, 1, 0, billing.DefaultTokenPerUnitSMS, billing.DefaultCreditPerUnitSMS, tt.tmBillingStart)
```
becomes:
```go
mockDB.EXPECT().BillingConsumeAndRecord(ctx, tt.responseBilling, tt.responseBilling.AccountID, 1, 0, billing.GetCostInfo(billing.CostTypeSMS), tt.tmBillingStart)
```

For call tests using `tt.billing.RateTokenPerUnit, tt.billing.RateCreditPerUnit`:
```go
mockDB.EXPECT().BillingConsumeAndRecord(ctx, tt.billing, tt.billing.AccountID, expectBillableUnits, expectUsageDuration, tt.billing.RateTokenPerUnit, tt.billing.RateCreditPerUnit, tt.tmBillingEnd)
```
becomes:
```go
mockDB.EXPECT().BillingConsumeAndRecord(ctx, tt.billing, tt.billing.AccountID, expectBillableUnits, expectUsageDuration, billing.GetCostInfo(tt.billing.CostType), tt.tmBillingEnd)
```

**Step 8: Run all billing-manager tests**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing/bin-billing-manager && go test ./...`
Expected: PASS

**Step 9: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing
git add bin-billing-manager/pkg/dbhandler/ bin-billing-manager/pkg/billinghandler/
git commit -m "NOJIRA-billing-cost-mode-email-billing

- bin-billing-manager: Update CalculateTokenCreditDeduction to accept CostInfo with mode-based switch
- bin-billing-manager: Update BillingConsumeAndRecord signature to accept CostInfo
- bin-billing-manager: Update all callers and tests for new signatures"
```

---

### Task 5: Refactor balance validation to use CostInfo

**Files:**
- Modify: `bin-billing-manager/pkg/accounthandler/balance.go`
- Modify: `bin-billing-manager/pkg/accounthandler/balance_test.go`

**Step 1: Refactor `IsValidBalance` in `balance.go`**

Replace the `switch billingType` block (lines 71-106) with CostInfo-based logic. Note: `ReferenceTypeCall` needs special handling because one ReferenceType maps to multiple CostTypes (PSTN out/in, VN, extension). The pre-flight check is optimistic.

```go
	switch billingType {
	case billing.ReferenceTypeCall:
		// Calls can be VN (TokenFirst) or PSTN (CreditOnly).
		// Optimistic: accept if tokens exist (might be VN) OR enough credit for worst-case PSTN rate.
		if a.BalanceToken > 0 {
			promAccountBalanceCheckTotal.WithLabelValues("valid").Inc()
			return true, nil
		}
		costInfo := billing.GetCostInfo(billing.CostTypeCallPSTNOutgoing)
		expectCost := costInfo.CreditPerUnit * int64(count)
		if a.BalanceCredit >= expectCost {
			promAccountBalanceCheckTotal.WithLabelValues("valid").Inc()
			return true, nil
		}

	case billing.ReferenceTypeSMS:
		costInfo := billing.GetCostInfo(billing.CostTypeSMS)
		expectCost := costInfo.CreditPerUnit * int64(count)
		if a.BalanceCredit >= expectCost {
			promAccountBalanceCheckTotal.WithLabelValues("valid").Inc()
			return true, nil
		}

	case billing.ReferenceTypeEmail:
		costInfo := billing.GetCostInfo(billing.CostTypeEmail)
		expectCost := costInfo.CreditPerUnit * int64(count)
		if a.BalanceCredit >= expectCost {
			promAccountBalanceCheckTotal.WithLabelValues("valid").Inc()
			return true, nil
		}

	case billing.ReferenceTypeNumber, billing.ReferenceTypeNumberRenew:
		costInfo := billing.GetCostInfo(billing.CostTypeNumber)
		expectCost := costInfo.CreditPerUnit * int64(count)
		if a.BalanceCredit >= expectCost {
			promAccountBalanceCheckTotal.WithLabelValues("valid").Inc()
			return true, nil
		}

	default:
		log.Errorf("Unsupported billing type. billing_type: %s", billingType)
		return false, fmt.Errorf("unsupported billing type")
	}
```

**Step 2: Update `balance_test.go`**

The test "sms with tokens but no credit returns true" should now return **false** because SMS is credit-only (tokens no longer accepted for SMS). Update:

```go
	{
		name: "sms with tokens but no credit returns false (SMS is credit-only)",
		// ... same setup ...
		expectRes: false,  // Changed from true
	},
```

Also update any test expectations that used the old SMS rate ($0.008 = 8000 micros). The new rate is $0.01 = 10000 micros. Review test accounts' `BalanceCredit` values to ensure they still pass/fail as expected with the new rate.

Add a new test case for email:

```go
	{
		name: "email with enough credit balance",

		accountID:   uuid.FromStringOrNil("b1b1b1b1-1111-11ee-86c6-111111111111"),
		billingType: billing.ReferenceTypeEmail,
		count:       1,

		responseAccount: &account.Account{
			Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("b1b1b1b1-1111-11ee-86c6-111111111111"),
			},
			BalanceCredit: 1000000,
			TMDelete:      nil,
		},
		expectRes: true,
	},
	{
		name: "email with insufficient credit balance",

		accountID:   uuid.FromStringOrNil("b2b2b2b2-2222-11ee-86c6-222222222222"),
		billingType: billing.ReferenceTypeEmail,
		count:       1,

		responseAccount: &account.Account{
			Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("b2b2b2b2-2222-11ee-86c6-222222222222"),
			},
			BalanceCredit: 1,
			TMDelete:      nil,
		},
		expectRes: false,
	},
```

**Step 3: Run tests**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing/bin-billing-manager && go test ./pkg/accounthandler/...`
Expected: PASS

**Step 4: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing
git add bin-billing-manager/pkg/accounthandler/
git commit -m "NOJIRA-billing-cost-mode-email-billing

- bin-billing-manager: Refactor IsValidBalance to use GetCostInfo for rate lookups
- bin-billing-manager: Add ReferenceTypeEmail balance validation
- bin-billing-manager: SMS is now credit-only (tokens no longer accepted for SMS)"
```

---

### Task 6: Add email event handler in billing-manager

**Files:**
- Modify: `bin-billing-manager/pkg/billinghandler/main.go` (interface)
- Modify: `bin-billing-manager/pkg/billinghandler/event.go`

**Step 1: Add `EventEMEmailCreated` to the BillingHandler interface**

In `bin-billing-manager/pkg/billinghandler/main.go`, add to the interface (after `EventMMMessageCreated`):

```go
	EventEMEmailCreated(ctx context.Context, e *ememail.Email) error
```

Add the import at the top:
```go
	ememail "monorepo/bin-email-manager/models/email"
```

**Step 2: Implement `EventEMEmailCreated` in `event.go`**

Add the import for email models in `event.go`:
```go
	ememail "monorepo/bin-email-manager/models/email"
```

Add the function (same pattern as `EventMMMessageCreated`):

```go
// EventEMEmailCreated handles the email-manager's email_created event
func (h *billingHandler) EventEMEmailCreated(ctx context.Context, e *ememail.Email) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "EventEMEmailCreated",
		"email": e,
	})
	log.Debugf("Received email_created event. email_id: %s", e.ID)

	for i, dest := range e.Destinations {
		targetRefID := uuid.NewV5(e.ID, fmt.Sprintf("target-%d", i))
		log.WithField("destination", dest).Debugf("Creating billing for email. destination: %v, target_ref_id: %s", dest, targetRefID)
		if errBilling := h.BillingStart(ctx, e.CustomerID, billing.ReferenceTypeEmail, targetRefID, billing.CostTypeEmail, e.TMCreate, e.Source, &dest); errBilling != nil {
			return errors.Wrapf(errBilling, "could not create a billing. destination: %v", dest)
		}
	}

	return nil
}
```

**Step 3: Regenerate mocks**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing/bin-billing-manager && go generate ./pkg/billinghandler/...`

**Step 4: Run tests**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing/bin-billing-manager && go test ./pkg/billinghandler/...`
Expected: PASS (existing tests should still pass, new function has no tests yet — added in Task 8)

**Step 5: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing
git add bin-billing-manager/pkg/billinghandler/
git commit -m "NOJIRA-billing-cost-mode-email-billing

- bin-billing-manager: Add EventEMEmailCreated handler for email billing
- bin-billing-manager: Create billing records per email destination with immediate-end charging"
```

---

### Task 7: Wire email events into subscribe handler

**Files:**
- Modify: `bin-billing-manager/pkg/subscribehandler/main.go`
- Create: `bin-billing-manager/pkg/subscribehandler/email.go`
- Modify: `bin-billing-manager/cmd/billing-manager/main.go`

**Step 1: Create `email.go` in subscribehandler**

Create `bin-billing-manager/pkg/subscribehandler/email.go`:

```go
package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	ememail "monorepo/bin-email-manager/models/email"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// processEventEMEmailCreated handles the email-manager's email_created event
func (h *subscribeHandler) processEventEMEmailCreated(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventEMEmailCreated",
		"event": m,
	})
	log.Debugf("Received email event. event: %s", m.Type)

	var e ememail.Email
	if err := json.Unmarshal([]byte(m.Data), &e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return errors.Wrap(err, "could not unmarshal the data")
	}

	if errEvent := h.billingHandler.EventEMEmailCreated(ctx, &e); errEvent != nil {
		log.Errorf("Could not handle the event. err: %v", errEvent)
		return errEvent
	}

	return nil
}
```

**Step 2: Add email event routing in `main.go`**

In `bin-billing-manager/pkg/subscribehandler/main.go`, add the import:
```go
	ememail "monorepo/bin-email-manager/models/email"
```

In the `processEvent` function's switch block (around line 157), add the email case after the message-manager cases:

```go
	//// email-manager
	// email
	case m.Publisher == string(commonoutline.ServiceNameEmailManager) && m.Type == ememail.EventTypeCreated:
		err = h.processEventEMEmailCreated(ctx, m)
```

**Step 3: Add email event queue to subscribe targets**

In `bin-billing-manager/cmd/billing-manager/main.go`, add `QueueNameEmailEvent` to the subscribe targets list (around line 162):

```go
	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
		string(commonoutline.QueueNameMessageEvent),
		string(commonoutline.QueueNameCustomerEvent),
		string(commonoutline.QueueNameNumberEvent),
		string(commonoutline.QueueNameEmailEvent),
	}
```

**Step 4: Update go.mod for email-manager dependency**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing/bin-billing-manager && go mod tidy && go mod vendor`

**Step 5: Regenerate mocks for subscribehandler**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing/bin-billing-manager && go generate ./pkg/subscribehandler/...`

**Step 6: Run tests**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing/bin-billing-manager && go test ./pkg/subscribehandler/...`
Expected: PASS

**Step 7: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing
git add bin-billing-manager/pkg/subscribehandler/ bin-billing-manager/cmd/billing-manager/main.go bin-billing-manager/go.mod bin-billing-manager/go.sum bin-billing-manager/vendor/
git commit -m "NOJIRA-billing-cost-mode-email-billing

- bin-billing-manager: Add email event subscription handler
- bin-billing-manager: Subscribe to email-manager event queue for email_created events
- bin-billing-manager: Route email events to billing handler"
```

---

### Task 8: Add pre-send balance check in email-manager

**Files:**
- Modify: `bin-email-manager/pkg/emailhandler/email.go`
- Modify: `bin-email-manager/pkg/emailhandler/email_test.go`

**Step 1: Add balance check in `Create` function**

In `bin-email-manager/pkg/emailhandler/email.go`, add the import:
```go
	bmbilling "monorepo/bin-billing-manager/models/billing"
```

In the `Create` function (after destination validation, before calling `h.create()`), add:

```go
	// validate balance before sending
	valid, err := h.reqHandler.BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, bmbilling.ReferenceTypeEmail, "", len(destinations))
	if err != nil {
		return nil, errors.Wrap(err, "could not validate the customer's balance")
	}
	if !valid {
		return nil, errors.New("insufficient balance for email")
	}
```

**Step 2: Update tests in `email_test.go`**

For existing `Create` tests that expect success, add the mock expectation before `mockDB.EXPECT().EmailCreate`:

```go
mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.customerID, bmbilling.ReferenceTypeEmail, "", len(tt.destinations)).Return(true, nil)
```

Add a new test case for insufficient balance:

```go
{
	name: "insufficient balance returns error",
	// ... setup with destinations ...
	// mockReq returns (false, nil) for balance check
	// expect: error, no email created
}
```

**Step 3: Update go.mod for billing-manager dependency**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing/bin-email-manager && go mod tidy && go mod vendor`

**Step 4: Run tests**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing/bin-email-manager && go test ./pkg/emailhandler/...`
Expected: PASS

**Step 5: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing
git add bin-email-manager/
git commit -m "NOJIRA-billing-cost-mode-email-billing

- bin-email-manager: Add pre-send balance check before creating emails
- bin-email-manager: Reject email creation when customer has insufficient balance"
```

---

### Task 9: Full verification workflow

**Files:** All changed services

**Step 1: Run full verification for bin-billing-manager**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing/bin-billing-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: All PASS

**Step 2: Run full verification for bin-email-manager**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing/bin-email-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: All PASS

**Step 3: Commit any changes from verification**

If `go generate` produced changes to mock files, commit them.

---

### Task 10: Check for conflicts with main and create PR

**Step 1: Fetch latest main and check for conflicts**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-billing-cost-mode-email-billing
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

**Step 2: If conflicts exist, rebase and re-run verification**

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-billing-cost-mode-email-billing
gh pr create --title "NOJIRA-billing-cost-mode-email-billing" --body "$(cat <<'EOF'
Add CostMode enum for explicit billing mode classification and integrate email billing.

- bin-billing-manager: Add CostMode enum (Disabled/Free/CreditOnly/TokenFirst) and CostInfo struct
- bin-billing-manager: Add CostTypeEmail and ReferenceTypeEmail
- bin-billing-manager: Update GetCostInfo to return CostInfo with explicit mode
- bin-billing-manager: Update rates: SMS $0.01/msg, PSTN Out/In $0.01/min
- bin-billing-manager: SMS is now credit-only (tokens no longer accepted)
- bin-billing-manager: Update CalculateTokenCreditDeduction with mode-based switch
- bin-billing-manager: Update BillingConsumeAndRecord to use CostInfo
- bin-billing-manager: Refactor IsValidBalance to use GetCostInfo for rate lookups
- bin-billing-manager: Add email event subscription and billing handler
- bin-email-manager: Add pre-send balance check before email creation
EOF
)"
```
