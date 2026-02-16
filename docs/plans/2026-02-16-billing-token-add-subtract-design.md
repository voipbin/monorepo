# Billing Token Add/Subtract Design

## Problem

The billing-control CLI and dbhandler support adding/subtracting credit balance with ledger entries (`accountAdjustCreditWithLedger`), but there is no equivalent for token balance. Tokens are only modified by the monthly top-up (`AccountTopUpTokens`), which resets to the plan limit. There is no general-purpose token adjustment with ledger tracking.

## Approach

Mirror the existing credit adjustment pattern exactly:

1. Add `accountAdjustTokenWithLedger` — atomic transaction that adjusts `balance_token` and inserts a billing ledger entry with `reference_type = "token_adjustment"`.
2. Add public wrappers: `AccountAddTokens`, `AccountSubtractTokens`, `AccountSubtractTokensWithCheck`.
3. Add accounthandler methods: `AddTokens`, `SubtractTokens`, `SubtractTokensWithCheck`.
4. Add billing-control CLI commands: `account add-tokens`, `account subtract-tokens`.
5. Add sqlmock-based tests for the new dbhandler function.

## Design Decisions

- **Balance check on subtract**: `SubtractTokensWithCheck` returns `ErrInsufficientBalance` if `balance_token < amount`. The accounthandler checks `PlanTypeUnlimited` and bypasses the check for unlimited accounts (same as credit).
- **No event publishing**: Consistent with credit add/subtract — the accounthandler methods do DB operations only, no `notifyHandler.PublishEvent`.
- **New reference type**: `ReferenceTypeTokenAdjustment = "token_adjustment"` in `models/billing/billing.go`.
- **Unique ReferenceID**: `ledgerEntry.ReferenceID = ledgerEntry.ID` (learned from the credit adjustment duplicate key bug fix).
- **Ledger entry fields**: `AmountToken = signedAmount`, `AmountCredit = 0` (inverse of credit adjustment which sets `AmountToken = 0`).

## Files to Change

### models/billing/billing.go
Add constant:
```go
ReferenceTypeTokenAdjustment ReferenceType = "token_adjustment"
```

### pkg/dbhandler/account.go
Add `accountAdjustTokenWithLedger` (mirrors `accountAdjustCreditWithLedger`):
- BEGIN transaction
- SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE id = ? FOR UPDATE
- Balance check: if `checkBalance && signedAmount < 0 && currentToken < -signedAmount` → return ErrInsufficientBalance
- UPDATE billing_accounts SET balance_token = ?, tm_update = ? WHERE id = ?
- INSERT billing ledger entry with:
  - TransactionType: adjustment
  - Status: end
  - ReferenceType: token_adjustment
  - ReferenceID: ledgerEntry.ID (unique per entry)
  - AmountToken: signedAmount
  - AmountCredit: 0
  - BalanceTokenSnapshot: newBalance (post-adjustment)
  - BalanceCreditSnapshot: currentCredit (unchanged)
- COMMIT
- Update cache

Add public wrappers:
```go
func (h *handler) AccountAddTokens(ctx, accountID, amount int64) error
func (h *handler) AccountSubtractTokens(ctx, accountID, amount int64) error
func (h *handler) AccountSubtractTokensWithCheck(ctx, accountID, amount int64) error
```

### pkg/dbhandler/main.go
Add to DBHandler interface:
```go
AccountAddTokens(ctx context.Context, accountID uuid.UUID, amount int64) error
AccountSubtractTokens(ctx context.Context, accountID uuid.UUID, amount int64) error
AccountSubtractTokensWithCheck(ctx context.Context, accountID uuid.UUID, amount int64) error
```

### pkg/accounthandler/main.go
Add to AccountHandler interface:
```go
AddTokens(ctx context.Context, accountID uuid.UUID, amount int64) (*account.Account, error)
SubtractTokens(ctx context.Context, accountID uuid.UUID, amount int64) (*account.Account, error)
SubtractTokensWithCheck(ctx context.Context, accountID uuid.UUID, amount int64) (*account.Account, error)
```

### pkg/accounthandler/db.go
Add implementations mirroring AddBalance/SubtractBalance/SubtractBalanceWithCheck:
- `AddTokens` → calls `h.db.AccountAddTokens`, then `h.db.AccountGet`
- `SubtractTokens` → calls `h.db.AccountSubtractTokens`, then `h.db.AccountGet`
- `SubtractTokensWithCheck` → checks PlanTypeUnlimited, then calls `h.db.AccountSubtractTokensWithCheck`, then `h.db.AccountGet`

### cmd/billing-control/main.go
Add two new account subcommands:
- `account add-tokens --id <uuid> --amount <int>` → calls `accountHandler.AddTokens`
- `account subtract-tokens --id <uuid> --amount <int>` → calls `accountHandler.SubtractTokens`

### pkg/dbhandler/account_test.go
Add sqlmock tests for `accountAdjustTokenWithLedger`:
- Happy path: add tokens
- Happy path: subtract tokens
- Subtract with check: sufficient balance
- Subtract with check: insufficient balance → ErrInsufficientBalance
- Account not found → ErrNotFound

### pkg/accounthandler/db_test.go
Add gomock tests for accounthandler AddTokens/SubtractTokens/SubtractTokensWithCheck.

## What Does NOT Change

- No API endpoint changes (token adjustments are admin-only via billing-control CLI)
- No OpenAPI spec changes
- No RabbitMQ event publishing
- No database schema changes (balance_token column already exists, billing_billings table already supports the new reference_type as a string)
- Existing `AccountTopUpTokens` remains unchanged (monthly top-up is a separate operation)
