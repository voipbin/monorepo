# billing-control CLI Design

## Overview

A read-only CLI tool for inspecting billing accounts and billing records in bin-billing-manager, following the same pattern as customer-control in bin-customer-manager.

## Commands

```bash
# Account operations (read-only)
billing-control account get --id <uuid>
billing-control account list [--limit N] [--token T] [--customer-id <uuid>]

# Billing record operations (read-only)
billing-control billing get --id <uuid>
billing-control billing list [--limit N] [--token T] [--customer-id <uuid>] [--account-id <uuid>]
```

## File Structure

```
bin-billing-manager/
└── cmd/
    └── billing-control/
        └── main.go          # Single file (~250 lines)
```

Following customer-control's pattern: one file with Cobra commands, Viper config binding, and handler initialization.

## Dependencies

Same as customer-control:
- `cobra` - CLI framework with subcommands
- `viper` - Config/flag binding
- `survey` - Interactive UUID prompts when flags missing
- Existing handlers: `accounthandler`, `billinghandler`, `dbhandler`, `cachehandler`

## Output Format

### account get

```
--- Account Information ---
ID:           abc123...
Customer ID:  def456...
Name:         Main Account
Balance:      $150.00
Type:         normal
Payment Type: prepaid
----------------------------

--- Raw Data (JSON) ---
{ ... }
-----------------------
```

### account list

```
Retrieving Accounts (limit: 100, token: )...
Success! accounts count: 2
 - [uuid1] Main Account | $150.00 | normal
 - [uuid2] Test Account | $0.00 | normal
```

### billing get

```
--- Billing Information ---
ID:            abc123...
Customer ID:   def456...
Account ID:    ghi789...
Reference:     call (uuid)
Status:        end
Cost:          $0.12
Duration:      6.5 units
----------------------------

--- Raw Data (JSON) ---
{ ... }
-----------------------
```

### billing list

```
Retrieving Billings (limit: 100, token: )...
Success! billings count: 3
 - [uuid1] call   | $0.12 | end | 2024-01-15
 - [uuid2] sms    | $0.01 | end | 2024-01-15
 - [uuid3] number | $5.00 | end | 2024-01-14
```

## Handler Initialization

The CLI reuses existing handlers without modification:

```
initHandler()
├── Database connection (config.DatabaseDSN)
├── Redis cache (cachehandler)
├── RabbitMQ socket (for requesthandler/notifyhandler)
├── dbhandler.NewHandler(db, cache)
├── accounthandler.NewAccountHandler(reqHandler, db, notifyHandler)
└── billinghandler.NewBillingHandler(reqHandler, db, notifyHandler, accountHandler)
```

No changes needed to existing handlers - the CLI calls their `Get` and `List` methods directly.

## Flags Summary

| Command | Flags |
|---------|-------|
| account get | --id (required) |
| account list | --limit (default 100), --token, --customer-id |
| billing get | --id (required) |
| billing list | --limit (default 100), --token, --customer-id, --account-id |

## Implementation Notes

1. Follow customer-control's code structure exactly
2. Use `resolveUUID()` helper for interactive UUID prompts
3. Use `survey.AskOne` when required flags are missing
4. Config loaded via `config.LoadGlobalConfig()` and `config.Get()`
5. Same environment variables as billing-manager (DATABASE_DSN, RABBITMQ_ADDRESS, REDIS_ADDRESS, etc.)

## Changes Required

- **New file:** `bin-billing-manager/cmd/billing-control/main.go`
- **No changes** to existing handlers, models, or other code
