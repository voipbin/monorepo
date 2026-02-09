# Design: Add update-plan-type Command to billing-control

## Problem

The billing-control CLI tool has no command to change an account's plan tier (PlanType).
Currently, plan types can only be changed by directly updating the database. Admins need
a safe, consistent way to change account tiers via the existing CLI tool.

## Approach

Add an `update-plan-type` subcommand to the `billing-control account` command group,
following the same pattern as the existing `update` and `update-payment-info` commands.

### Usage

```bash
billing-control account update-plan-type --id <account-uuid> --plan-type <free|basic|professional|unlimited>
```

Returns the updated account as JSON.

### Changes

**1. AccountHandler interface** (`pkg/accounthandler/main.go`)
- Add `UpdatePlanType(ctx context.Context, id uuid.UUID, planType account.PlanType) (*account.Account, error)`

**2. AccountHandler implementation** (`pkg/accounthandler/account.go`)
- Add `UpdatePlanType` method that delegates to `dbUpdatePlanType`

**3. DB helper** (`pkg/accounthandler/db.go`)
- Add `dbUpdatePlanType` that builds `{FieldPlanType: planType}` field map and calls `h.db.AccountUpdate`

**4. CLI command** (`cmd/billing-control/main.go`)
- Add `cmdAccountUpdatePlanType()` with `--id` and `--plan-type` flags
- Add `runAccountUpdatePlanType()` that validates plan-type against known constants before calling handler
- Register under `cmdAccount`

**5. Mock regeneration**
- Run `go generate ./...` to update mock for the new interface method

### Validation

The CLI validates `--plan-type` against the four known values: `free`, `basic`, `professional`, `unlimited`.
Invalid values are rejected before calling the handler.

### No validation on downgrade

Per user requirement, this is a direct admin override with no resource-count validation.
The admin is responsible for ensuring the tier change is appropriate.
