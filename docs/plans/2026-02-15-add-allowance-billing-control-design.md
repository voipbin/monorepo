# Add Allowance Commands to billing-control

**Date:** 2026-02-15
**Status:** Approved

## Problem

The `billing-control` CLI has `account` and `billing` subcommand groups but no way to inspect or manage token allowances. Operators need to view allowance cycles, trigger cycle creation, and adjust token allocations directly.

## Solution

Add a new `allowance` subcommand group to `billing-control` with six commands:

```
billing-control allowance get              --account-id <uuid>
billing-control allowance list             --account-id <uuid> [--limit 100] [--token <string>]
billing-control allowance process-all
billing-control allowance ensure           --account-id <uuid>
billing-control allowance add-tokens       --account-id <uuid> --amount <int>
billing-control allowance subtract-tokens  --account-id <uuid> --amount <int>
```

### Commands

1. **get** — Returns the current active cycle for an account (`GetCurrentCycle`)
2. **list** — Returns paginated history of all cycles for an account (`ListByAccountID`)
3. **process-all** — Triggers `ProcessAllCycles` to create missing cycles for all accounts
4. **ensure** — Force-creates the current cycle for a specific account if none exists. Looks up `customerID` and `planType` from the account automatically (`EnsureCurrentCycle`)
5. **add-tokens** — Force-add to `tokens_total` on the current cycle (new `AllowanceHandler` method)
6. **subtract-tokens** — Force-subtract from `tokens_total` on the current cycle (new `AllowanceHandler` method)

### Changes Required

#### AllowanceHandler (new methods)
- `AddTokens(ctx, accountID, amount)` — Get current cycle, increase `tokens_total` by `amount`
- `SubtractTokens(ctx, accountID, amount)` — Get current cycle, decrease `tokens_total` by `amount`

#### DBHandler (new method)
- `AllowanceUpdateTokensTotal(ctx, allowanceID, newTotal)` — Update `tokens_total` for a given allowance

#### billing-control main.go
- Update `initHandlers()` to return `allowanceHandler` as third value
- Add `allowance` subcommand group with all six commands
- `ensure` command fetches the account first to get `customerID` and `planType`

### Output

All commands output JSON to stdout, matching existing billing-control patterns.
