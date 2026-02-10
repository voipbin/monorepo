# Design: Remove Billing Account Type Field

## Problem

The billing account `Type` field (`admin`/`normal`) adds complexity without providing value.
Its only behavioral effect is bypassing balance checks for admin accounts. This behavior
maps directly onto the existing `PlanType` tier system, where `PlanTypeUnlimited` serves the
same purpose. Removing `Type` simplifies the code and reduces confusion.

## Approach

Replace `TypeAdmin` balance bypass with `PlanTypeUnlimited` check, then remove the `Type`
field entirely from the codebase.

### Changes

**1. Balance validation** (`pkg/accounthandler/balance.go`)
- Replace `a.Type == account.TypeAdmin` with `a.PlanType == account.PlanTypeUnlimited`

**2. Balance subtraction** (`pkg/accounthandler/db.go`)
- Replace `a.Type == account.TypeAdmin` with `a.PlanType == account.PlanTypeUnlimited` in
  `SubtractBalanceWithCheck`
- Remove `Type: account.TypeNormal` from account creation in `dbCreate`

**3. Account model** (`models/account/account.go`)
- Remove `Type` field from `Account` struct
- Remove `Type` type definition and `TypeAdmin`/`TypeNormal` constants

**4. Field constants** (`models/account/field.go`)
- Remove `FieldType`

**5. Filters** (`models/account/filters.go`)
- Remove `Type` from filter struct

**6. Tests**
- Update tests referencing `TypeAdmin`/`TypeNormal` to use `PlanTypeUnlimited`/`PlanTypeFree`

**7. Database migration** (`bin-dbscheme-manager`)
- Create Alembic migration to drop the `type` column from `billing_accounts`
- Column is safely ignored in the meantime since GetDBFields/ScanRow only operate on
  struct-tagged fields

### No OpenAPI changes needed

The `type` field was never documented in the OpenAPI spec.
