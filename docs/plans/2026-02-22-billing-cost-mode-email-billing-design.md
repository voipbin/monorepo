# Billing CostMode and Email Billing Design

## Problem Statement

1. SMS and call rates need updating to new pricing.
2. Email has no billing integration — no pre-send balance check, no billing records, no cost deduction.
3. No explicit distinction between "free" (allowed, no charge) and "not available" (tokens cannot be used) — both currently use rate = 0.

## Rate Changes

| Type | Current Rate | New Rate |
|------|-------------|----------|
| SMS | 10 tokens OR $0.008/msg | $0.01/msg (credit only) |
| Email | Not billed | $0.01/msg (credit only) |
| Call (PSTN Out) | $0.006/min | $0.01/min (credit only) |
| Call (PSTN In) | $0.0045/min | $0.01/min (credit only) |
| Call (Virtual) | 1 token, $0.001/min | 1 token, $0.001/min (unchanged) |
| Extension | Free | Free (unchanged) |
| Number | $5.00 | $5.00 (unchanged) |

## Approach: CostMode Enum

Introduce a `CostMode` enum that explicitly declares how each cost type is charged:

```go
type CostMode int

const (
    CostModeDisabled   CostMode = iota // Service not available — requests rejected
    CostModeFree                        // Allowed, no charge
    CostModeCreditOnly                  // Credit only, tokens not accepted
    CostModeTokenFirst                  // Token first, overflow to credits
)

type CostInfo struct {
    Mode          CostMode
    TokenPerUnit  int64
    CreditPerUnit int64
}
```

This replaces the current `GetCostInfo()` which returns two bare ints `(tokenPerUnit, creditPerUnit)`, making it impossible to distinguish free from disabled.

## Changes

### Part 1: CostMode Enum and CostInfo Struct

**File:** `bin-billing-manager/models/billing/cost_type.go`

- Add `CostMode` enum with four modes: `Disabled`, `Free`, `CreditOnly`, `TokenFirst`
- Add `CostInfo` struct with `Mode`, `TokenPerUnit`, `CreditPerUnit`
- Add `CostTypeEmail CostType = "email"`
- Update `GetCostInfo()` to return `CostInfo`
- Update rate constants to new values
- Remove `DefaultTokenPerUnitSMS` (tokens no longer used for SMS)

### Part 2: Add ReferenceTypeEmail

**File:** `bin-billing-manager/models/billing/billing.go`

- Add `ReferenceTypeEmail ReferenceType = "email"`

### Part 3: Update Deduction Algorithm

**File:** `bin-billing-manager/pkg/dbhandler/billing.go`

- Update `CalculateTokenCreditDeduction` to accept `billing.CostInfo` instead of separate rate args
- Add mode-based switch:
  - `CostModeFree` -> zero deduction
  - `CostModeCreditOnly` -> credit only deduction
  - `CostModeTokenFirst` -> existing token-overflow logic
  - `CostModeDisabled` -> zero deduction
- Update `BillingConsumeAndRecord` signature to accept `CostInfo`
- Update all callers

### Part 4: Update Balance Validation

**File:** `bin-billing-manager/pkg/accounthandler/balance.go`

- Refactor `IsValidBalance` to use `GetCostInfo()` and branch on `CostInfo.Mode`:
  - `CostModeDisabled` -> return false
  - `CostModeFree` -> return true
  - `CostModeCreditOnly` -> check `BalanceCredit >= CreditPerUnit * count`
  - `CostModeTokenFirst` -> check tokens first, then credits
- Add `ReferenceTypeEmail` case
- Remove SMS-specific token check (SMS is now credit-only)

### Part 5: Email Billing Event Handling in billing-manager

**File:** `bin-billing-manager/pkg/subscribehandler/main.go`
- Import `bin-email-manager/models/email`
- Add event subscription case for `email-manager` / `email_created`

**New file:** `bin-billing-manager/pkg/subscribehandler/email.go`
- Add `processEventEMEmailCreated()` — same pattern as SMS event handler
- Unmarshal email event data, call `billingHandler.EventEMEmailCreated()`

**File:** `bin-billing-manager/pkg/billinghandler/event.go`
- Add `EventEMEmailCreated()` — creates billing records per email destination
- Uses `CostTypeEmail`, `ReferenceTypeEmail`
- Immediate-end billing (charge at creation time, same as SMS)

### Part 6: Pre-send Balance Check in email-manager

**File:** `bin-email-manager/pkg/emailhandler/email.go`
- In `Create()`, before calling `h.create()`, call `reqHandler.BillingV1AccountIsValidBalanceByCustomerID()` with `ReferenceTypeEmail` and `count = len(destinations)`
- If balance insufficient, return error (email not sent)

**File:** `bin-email-manager/pkg/emailhandler/main.go`
- Add `reqHandler` dependency to emailHandler struct (if not already present)

### Part 7: Update billing-manager subscribeTargets

**File:** `bin-billing-manager/cmd/billing-manager/main.go`
- Add `email-manager` event queue to subscribe targets so billing-manager receives email events

### Part 8: Tests

- `bin-billing-manager/pkg/dbhandler/deduction_test.go` — Update for new `CostInfo` parameter
- `bin-billing-manager/pkg/accounthandler/balance_test.go` — Update for refactored balance validation
- `bin-billing-manager/pkg/billinghandler/event_test.go` — Add tests for `EventEMEmailCreated`
- `bin-billing-manager/pkg/subscribehandler/email_test.go` — Add tests for email event processing
- `bin-email-manager/pkg/emailhandler/email_test.go` — Add tests for pre-send balance check

## Services Affected

| Service | Changes |
|---------|---------|
| `bin-billing-manager` | CostMode enum, rate updates, email event handling, updated deduction/balance logic |
| `bin-email-manager` | Pre-send balance check in Create() |

## What Does NOT Change

- `BillingConsumeAndRecord` database transaction logic (lock -> deduct -> record -> commit)
- Call billing flow (progressing -> hangup events)
- Number billing flow
- Virtual call token pricing
- Extension call free pricing
