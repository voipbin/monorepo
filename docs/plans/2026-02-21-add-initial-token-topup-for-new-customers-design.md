# Design: Initial Token Topup for New Customers

## Problem

When a new customer signs up, billing-manager creates a billing account with `BalanceToken: 0` and no `PlanType` set. The customer cannot use any token-based services. The monthly topup cron only processes accounts that have `tm_next_topup` set, which never happens for new accounts since they never receive an initial topup. This creates a dead state where new customers have zero tokens and no way to receive them automatically.

## Approach

After creating the billing account in `EventCUCustomerCreated()`, set the plan type to `PlanTypeFree` and call `AccountTopUpTokens()` to give the account its first month of tokens (1,000 for the Free tier).

This reuses the existing topup infrastructure which:
- Sets `balance_token` to the plan's token amount
- Creates a billing ledger entry (`TransactionTypeTopUp` / `ReferenceTypeMonthlyAllowance`)
- Sets `tm_last_topup` and `tm_next_topup` so the monthly cron picks up future topups

## Changes

### `bin-billing-manager/pkg/accounthandler/event.go`

In `EventCUCustomerCreated()`, after the account is created and linked to the customer, add:

1. `dbUpdatePlanType(ctx, b.ID, account.PlanTypeFree)` - Set the default plan type
2. `db.AccountTopUpTokens(ctx, b.ID, cu.ID, tokenAmount, string(account.PlanTypeFree))` - Perform initial token topup using the plan's token amount from `PlanTokenMap`

The topup is non-fatal: if it fails, the account is still created and linked. The customer just starts with 0 tokens (same as current behavior) and can be topped up manually or by the next cron run once `tm_next_topup` is set.

## Post-Signup State

After this change, a new customer's billing account will have:
- `PlanType: "free"`
- `BalanceToken: 1000`
- `tm_last_topup`: set to creation time
- `tm_next_topup`: set to first of next month
- A billing record showing the initial topup
