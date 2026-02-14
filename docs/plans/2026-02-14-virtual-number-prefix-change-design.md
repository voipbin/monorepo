# Virtual Number Prefix Change: +999 to +899

**Date:** 2026-02-14
**Status:** Approved

## Problem

The current `+999` prefix for virtual numbers conflicts with the `999` emergency telephone number used in 30+ Commonwealth of Nations countries (UK, Malaysia, Singapore, Bangladesh, Hong Kong, etc.). This causes confusion in call detail records, number parsing, and telecom system logs.

Additionally, `+999` is explicitly reserved by ITU-T for future global service, meaning it could be assigned someday.

## Decision

Change the virtual number prefix from `+999` to `+899`.

**Why +899:**
- Unassigned and spare in the ITU-T E.164 numbering plan (Zone 8, never allocated)
- No emergency number conflicts -- no country uses `899` as an emergency code
- Same 3-digit country code length, so the 13-character format is unchanged
- Low ITU assignment risk (the 89x block has no stated future plans)

## Format

| Property | Before | After |
|----------|--------|-------|
| Prefix | `+999` | `+899` |
| Reserved prefix | `+999000` | `+899000` |
| Country code | `"999"` | `"899"` |
| Format | `+999XXXYYYYYY` | `+899XXXYYYYYY` |
| Length | 13 chars | 13 chars (unchanged) |

## Approach

All production code must reference constants rather than hardcoded prefix strings. This ensures a single point of change if the prefix ever needs to change again.

### Constants (source of truth)

In `bin-number-manager/models/number/validate.go`:

```go
const (
    VirtualNumberPrefix         = "+899"
    VirtualNumberLength         = 13
    VirtualNumberReservedPrefix = "+899000"
    VirtualNumberCountryCode    = "899"  // NEW constant
)
```

### Code changes using constants

| File | Change |
|------|--------|
| `bin-number-manager/models/number/validate.go` | Update `VirtualNumberPrefix` and `VirtualNumberReservedPrefix` constants. Add `VirtualNumberCountryCode` constant. |
| `bin-number-manager/pkg/numberhandler/available_number.go` | Replace hardcoded `"+999%03d%06d"` with `fmt.Sprintf("%s%%03d%%06d", number.VirtualNumberPrefix)` pattern. Replace `Country: "999"` with `Country: number.VirtualNumberCountryCode`. |
| `bin-number-manager/cmd/number-control/main.go` | Update comment referencing `+999000XXXXXX` to reference the constant name. |

### Test file changes (hardcoded strings)

Test files use hardcoded `"+899..."` strings since tests verify concrete expected values:

| File | Change |
|------|--------|
| `bin-number-manager/models/number/validate_test.go` | Update ~15 test cases from `+999...` to `+899...` |
| `bin-number-manager/pkg/numberhandler/number_test.go` | Update test data from `+999...` to `+899...` |
| `bin-api-manager/server/available_numbers_test.go` | Update test data from `+999123456789` to `+899123456789` |

### Database migration

New Alembic migration in `bin-dbscheme-manager/`:

```sql
-- upgrade
UPDATE number_numbers SET number = REPLACE(number, '+999', '+899') WHERE type = 'virtual';

-- downgrade
UPDATE number_numbers SET number = REPLACE(number, '+899', '+999') WHERE type = 'virtual';
```

AI creates the migration file only. Human runs `alembic upgrade`.

### Documentation

Update `docs/plans/2026-02-10-virtual-number-design.md` to reflect the new `+899` prefix.

## What does NOT change

- Number length (13 chars)
- Validation logic structure (prefix check, digit-only check, reserved range check)
- API interface (request/response schemas)
- OpenAPI spec (the `normal`/`virtual` type enum is unaffected)
- Billing integration (checks `type` field, not prefix)
- RPC methods (method names reference "VirtualNumber", not the prefix)
- Resource limits

## Files NOT affected

- `bin-number-manager/pkg/numberhandler/number.go` -- uses constant references, not hardcoded values
- `bin-common-handler/` -- only has method names, no prefix strings
- `bin-billing-manager/` -- checks `number.TypeVirtual`, not the prefix
- `bin-call-manager/` -- `"+99999888"` in tests is an unrelated phone number
- Mock files -- auto-generated via `go generate ./...`

## Verification scope

Only two services have code changes:
- `bin-number-manager`: Full verification workflow
- `bin-api-manager`: Full verification workflow

No `bin-common-handler` changes, so no need to verify all 30+ services.

## Migration plan

1. Deploy code changes (new prefix for new virtual numbers)
2. Run Alembic migration (update existing virtual numbers in database)
3. Both can happen in the same release since virtual numbers are internal-only
