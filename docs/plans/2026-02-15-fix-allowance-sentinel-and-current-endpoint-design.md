# Fix Allowance Sentinel Inconsistency and Add Current Allowance Endpoint

**Date:** 2026-02-15
**Status:** Approved

## Problem

The `billing_allowances` table was created AFTER the sentinel-to-null migration (`071504ef41d0`) but inconsistently uses the OLD sentinel pattern (`tm_delete NOT NULL DEFAULT '9999-01-01 00:00:00.000000'`). This causes two bugs:

1. **AllowanceCreate fails**: Sets `TMDelete = nil`, which `PrepareFields` converts to SQL NULL. MySQL rejects NULL into a NOT NULL column. The `ProcessAllCycles` cron logs errors per-account but continues, resulting in zero allowance cycles created.

2. **AllowanceList returns nothing**: Uses `FieldDeleted: false` which `ApplyFields` converts to `WHERE tm_delete IS NULL`. But the table uses sentinel values, so no records match even if they existed.

Note: `AllowanceGetCurrentByAccountID` hardcodes the sentinel comparison and would work correctly if records existed.

Additionally, there is no endpoint to retrieve just the current active allowance cycle. The existing `GET /v1/accounts/{id}/allowances` returns a paginated history list.

## Solution

### Part 1: Fix Sentinel Inconsistency

1. New Alembic migration to make `tm_delete` nullable and convert sentinels to NULL
2. Update `AllowanceGetCurrentByAccountID` to use `tm_delete IS NULL` instead of sentinel
3. After deploy + migration, `ProcessAllCycles` will create cycles on startup automatically

### Part 2: Add Current Allowance Endpoint

New endpoint: `GET /v1/accounts/{account-id}/allowance` (singular)
- Returns the single active cycle for the current month
- Returns 404 if no cycle exists (does NOT auto-create)
- Uses existing `GetCurrentCycle` method

## Files to Modify

### Migration
- `bin-dbscheme-manager/bin-manager/main/versions/` — New migration file

### bin-billing-manager
- `pkg/dbhandler/allowance.go` — Fix sentinel query in `AllowanceGetCurrentByAccountID`
- `pkg/listenhandler/` — New route + handler for current allowance
- `pkg/listenhandler/main.go` — Register the new route

### bin-common-handler
- `pkg/requesthandler/` — New RPC method `BillingV1AccountAllowanceGet`

### bin-openapi-manager
- `openapi/openapi.yaml` — New endpoint definition

### bin-api-manager
- Regenerate server code via `go generate ./...`
