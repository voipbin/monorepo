# Virtual Number Feature Design

## Problem Statement

Currently, all numbers in VoIPbin must be purchased from a provider (Telnyx/Twilio). There is no way to create a number that exists purely as a logical identifier within the system. Customers need the ability to create arbitrary numbers for testing, internal routing, or as caller ID placeholders without incurring provider costs or triggering external API calls.

## Approach

Add a `type` field to the Number model that distinguishes between `normal` (provider-backed) and `virtual` (no provider) numbers. Virtual numbers use the `+899` country code prefix, are free (no billing), and skip all provider interactions. Their creation is gated by plan-tier resource limits.

## Virtual Number Format

All virtual numbers must follow strict format rules:

- **Prefix**: Must start with `+899`
- **Length**: Exactly 13 characters (`+` followed by 12 digits)
- **Characters**: Only digits after the `+` sign
- **Format**: `+899 XXX YYYYYY` (conceptually three groups: country `899`, area `XXX`, subscriber `YYYYYY`)

### Reserved Range

The range `+899000000000` through `+899000999999` (where the area code is `000`) is reserved for test and internal use:

- **API** (`POST /v1/numbers`): Always rejects numbers in the reserved range. Returns 400 error.
- **CLI** (`number-control number create --virtual-number`): Allows numbers in the reserved range. This is the only way to create reserved virtual numbers.

### Normal Number Validation

Normal numbers (type `normal`) must NOT start with `+899`. The `+899` prefix is exclusively for virtual numbers.

## Data Model Changes

### Number Model

Add `Type` field to `bin-number-manager/models/number/number.go`:

```go
type Type string

const (
    TypeNormal  Type = "normal"
    TypeVirtual Type = "virtual"
)
```

```go
type Number struct {
    commonidentity.Identity

    Number string `json:"number" db:"number"`
    Type   Type   `json:"type" db:"type"`       // NEW

    // ... rest unchanged
}
```

### Field Constants

Add to `bin-number-manager/models/number/field.go`:

```go
FieldType Field = "type"
```

### Database Migration

Alembic migration in `bin-dbscheme-manager`:

```sql
-- upgrade
ALTER TABLE number_numbers
    ADD COLUMN type VARCHAR(255) NOT NULL DEFAULT 'normal' AFTER number;
CREATE INDEX idx_number_numbers_type ON number_numbers (type);

-- downgrade
DROP INDEX idx_number_numbers_type ON number_numbers;
ALTER TABLE number_numbers DROP COLUMN type;
```

All existing numbers automatically become `normal` via the column default.

## Creation Flow

### API: `POST /v1/numbers`

The request body gains an optional `type` field (defaults to `"normal"`).

**When `type` is `"virtual"`:**

1. Validate virtual number format (`+899`, 13 chars, digits only)
2. Reject if number is in reserved range `+899000XXXXXX`
3. Check uniqueness (same as normal numbers)
4. Check plan-tier resource limit via `CustomerV1CustomerIsValidResourceLimit(ctx, customerID, ResourceTypeVirtualNumber)`
5. Skip balance check (no `CustomerV1CustomerIsValidBalance` call)
6. Skip provider purchase (no Telnyx/Twilio API call)
7. Register in database with `type: "virtual"`, `provider_name: ""`, `provider_reference_id: ""`
8. Publish `number_created` event (billing-manager ignores virtual numbers)
9. Skip tag update (no provider)

**When `type` is `"normal"` (default):**

1. Reject if number starts with `+899` (reserved for virtual numbers)
2. Existing flow unchanged

### CLI: `number-control number create`

Add `--virtual-number` boolean flag:

**When `--virtual-number` is set:**

1. Validate virtual number format (`+899`, 13 chars, digits only)
2. Allow reserved range `+899000XXXXXX` (CLI-only privilege)
3. Skip provider purchase and billing
4. Call `numberHandler.CreateVirtual()` which calls `Register()` with `type: "virtual"`

**When `--virtual-number` is not set:**

- Existing behavior unchanged (calls `numberHandler.Create()`)

### CLI: Remove `register` Command

Remove `cmdRegister()` from the CLI. The `register` command is replaced by `create --virtual-number` for the no-provider use case. The internal `Register()` method on `numberHandler` remains — it is still used internally by `Create()` after provider purchase.

## Deletion Flow

### `DELETE /v1/numbers/{id}`

The existing `Delete()` method checks `provider_name` to determine which provider to call for release. For virtual numbers:

- `provider_name` is `""` (ProviderNameNone)
- The existing `default:` case in the switch statement returns an error for unsupported providers
- **Change needed**: Add a case for `ProviderNameNone` that skips provider release and proceeds directly to database deletion

```go
switch num.ProviderName {
case number.ProviderNameTelnyx:
    err = h.numberHandlerTelnyx.NumberRelease(ctx, num)
case number.ProviderNameTwilio:
    err = h.numberHandlerTwilio.ReleaseNumber(ctx, num)
case number.ProviderNameNone:
    // Virtual number or no provider — skip provider release
default:
    err = fmt.Errorf("unsupported number provider. provider_name: %s", num.ProviderName)
}
```

## Renewal Flow

Virtual numbers do not need renewal since they are not backed by a provider. The existing renewal logic queries numbers by `tm_renew`. Virtual numbers will have `tm_renew` set to `nil`, so they will naturally be excluded from renewal queries.

## Available Virtual Numbers

### Current Behavior

`GET /v1/available_numbers` accepts `country_code` and `page_size` in the request body/query, queries Telnyx for purchasable numbers, and returns a list of `AvailableNumber` objects.

### Change: Add `type` Filter

Add an optional `type` field to the available numbers request body. When `type` is `"virtual"`:

1. Ignore `country_code` (not needed for virtual numbers)
2. Generate random virtual numbers in the valid range `+899001000000` through `+899999999999` (excluding reserved `+899000XXXXXX`)
3. Check each generated number against the database to ensure it's not already registered
4. Return up to `page_size` available numbers
5. Return `AvailableNumber` objects with:
   - `number`: The generated virtual number
   - `provider_name`: `""` (no provider)
   - `country`: `"899"`
   - `region`: `""`
   - `postal_code`: `""`
   - `features`: `["voice"]` (virtual numbers support voice routing)

When `type` is `"normal"` or empty, existing Telnyx behavior is unchanged.

### NumberHandler Interface

Update `GetAvailableNumbers` signature to accept the type parameter:

```go
GetAvailableNumbers(countryCode string, limit uint, numType number.Type) ([]*availablenumber.AvailableNumber, error)
```

### Implementation: GetAvailableVirtualNumbers

New method in `bin-number-manager/pkg/numberhandler/available_number.go`:

```go
func (h *numberHandler) GetAvailableVirtualNumbers(limit uint) ([]*availablenumber.AvailableNumber, error) {
    // 1. Generate random numbers in valid range
    // 2. Query DB to filter out taken numbers
    // 3. Return up to `limit` results as AvailableNumber objects
}
```

The generation loop:
- Generate a batch of random candidate numbers (e.g., 3x the limit to account for collisions)
- Each candidate: random integer in range [899001000000, 899999999999], formatted as `+%012d`
- Query DB with `WHERE number IN (candidates...) AND tm_delete = '9999-01-01'` to find taken ones
- Remove taken numbers from candidates
- Return first `limit` results

### listenhandler Changes

Update `processV1AvailableNumbersGet` in `v1_available_numbers.go`:
- Extract `type` from filters (same pattern as `country_code`)
- If `type` is `"virtual"`, call `numberHandler.GetAvailableVirtualNumbers(pageSize)`
- Otherwise, call existing `numberHandler.GetAvailableNumbers(countryCode, pageSize)`

### OpenAPI Changes

Update `bin-openapi-manager/openapi/paths/available_numbers/main.yaml` to document the optional `type` query parameter:

```yaml
- name: type
  in: query
  description: >
    The type of available numbers to retrieve.
    Use "virtual" to get available virtual numbers.
    Defaults to "normal" (queries provider).
  required: false
  schema:
    $ref: '#/components/schemas/NumberManagerNumberType'
```

### CLI Changes

Update `cmdGetAvailable` in `cmd/number-control/main.go`:
- Add `--virtual` boolean flag
- When set, call `GetAvailableVirtualNumbers(limit)` instead of `GetAvailableNumbers(countryCode, limit)`

### Additional Files to Modify

- `bin-number-manager/pkg/numberhandler/main.go` — Update `GetAvailableNumbers` signature or add `GetAvailableVirtualNumbers` method
- `bin-number-manager/pkg/numberhandler/available_number.go` — Add `GetAvailableVirtualNumbers` implementation
- `bin-number-manager/pkg/listenhandler/v1_available_numbers.go` — Parse `type` filter, route accordingly
- `bin-number-manager/pkg/dbhandler/number.go` — Add query to check which numbers from a list are already taken
- `bin-number-manager/cmd/number-control/main.go` — Add `--virtual` flag to `get-available` command
- `bin-openapi-manager/openapi/paths/available_numbers/main.yaml` — Add `type` parameter

## Resource Limits Integration

Extends the existing plan-tier design from `docs/plans/2026-02-09-resource-limit-plan-tier-design.md`.

### New Resource Type

Add to `bin-common-handler/pkg/commonbilling/resource_type.go`:

```go
ResourceTypeVirtualNumber ResourceType = "virtual_number"
```

### Plan Limits

Add `VirtualNumbers` to `PlanLimits` struct in `bin-billing-manager/models/account/plan.go`:

| Resource        | Free | Basic | Professional | Unlimited |
|-----------------|:----:|:-----:|:------------:|:---------:|
| VirtualNumbers  |    5 |    50 |          500 |  0 (none) |

### Count Endpoint

`number-manager` exposes a new internal endpoint:

```
GET /v1/numbers/count_by_customer?customer_id=<uuid>&type=virtual
```

Returns `{"count": 3}`. Billing-manager calls this during the limit check.

### Quota Check Chain

```
numberHandler.CreateVirtual()
  -> CustomerV1CustomerIsValidResourceLimit(ctx, customerID, ResourceTypeVirtualNumber)
    -> customer-manager -> billing-manager -> number-manager (count)
    -> compare count vs plan limit
    -> return allowed: true/false
```

## Billing Integration

### billing-manager Changes

In `bin-billing-manager/pkg/subscribehandler/number.go`, the `processEventNMNumberCreated` handler receives the number event. Since the Number struct now includes a `type` field, billing-manager can check:

```go
if c.Type == nmnumber.TypeVirtual {
    log.Debugf("Skipping billing for virtual number. number_id: %s", c.ID)
    return nil
}
```

This ensures no billing record is created for virtual numbers.

## OpenAPI Changes

### NumberManagerNumberType Enum

Add new schema in `bin-openapi-manager/openapi/openapi.yaml`:

```yaml
NumberManagerNumberType:
  type: string
  description: The type of the number.
  enum:
    - normal
    - virtual
  x-enum-varnames:
    - NumberManagerNumberTypeNormal
    - NumberManagerNumberTypeVirtual
```

### NumberManagerNumber Schema

Add `type` property after `number`:

```yaml
NumberManagerNumber:
  type: object
  properties:
    # ... existing fields ...
    type:
      $ref: '#/components/schemas/NumberManagerNumberType'
      description: The type of the number. Normal numbers are purchased from a provider. Virtual numbers are logical identifiers with no provider backing.
```

### POST /v1/numbers Request Body

Add optional `type` field in `bin-openapi-manager/openapi/paths/numbers/main.yaml`:

```yaml
type:
  $ref: '#/components/schemas/NumberManagerNumberType'
  description: The type of number to create. Defaults to "normal".
```

## Validation Logic

Shared validation function in `bin-number-manager/models/number/`:

```go
// number/validate.go

const (
    VirtualNumberPrefix        = "+899"
    VirtualNumberLength        = 13 // "+" plus 12 digits
    VirtualNumberReservedPrefix = "+899000"
)

// ValidateVirtualNumber validates a virtual number string.
// If allowReserved is false, numbers in the reserved range +899000XXXXXX are rejected.
func ValidateVirtualNumber(num string, allowReserved bool) error {
    if !strings.HasPrefix(num, VirtualNumberPrefix) {
        return fmt.Errorf("virtual number must start with %s", VirtualNumberPrefix)
    }

    if len(num) != VirtualNumberLength {
        return fmt.Errorf("virtual number must be exactly %d characters", VirtualNumberLength)
    }

    // Check all characters after "+" are digits
    for _, c := range num[1:] {
        if c < '0' || c > '9' {
            return fmt.Errorf("virtual number must contain only digits after +")
        }
    }

    if !allowReserved && strings.HasPrefix(num, VirtualNumberReservedPrefix) {
        return fmt.Errorf("virtual number range %sXXXXXX is reserved", VirtualNumberReservedPrefix)
    }

    return nil
}
```

## NumberHandler Interface Changes

Update `bin-number-manager/pkg/numberhandler/main.go`:

```go
type NumberHandler interface {
    // ... existing methods ...

    // CreateVirtual creates a virtual number without provider purchase or billing.
    CreateVirtual(ctx context.Context, customerID uuid.UUID, num string, callFlowID, messageFlowID uuid.UUID, name, detail string, allowReserved bool) (*number.Number, error)
}
```

### CreateVirtual Implementation

In `bin-number-manager/pkg/numberhandler/number.go`:

```go
func (h *numberHandler) CreateVirtual(
    ctx context.Context,
    customerID uuid.UUID,
    num string,
    callFlowID, messageFlowID uuid.UUID,
    name, detail string,
    allowReserved bool,
) (*number.Number, error) {
    // 1. Validate virtual number format
    if err := number.ValidateVirtualNumber(num, allowReserved); err != nil {
        return nil, err
    }

    // 2. Check resource limit (skip for CLI/allowReserved path if desired)
    valid, err := h.reqHandler.CustomerV1CustomerIsValidResourceLimit(
        ctx, customerID, commonbilling.ResourceTypeVirtualNumber)
    if err != nil {
        return nil, fmt.Errorf("could not validate resource limit: %w", err)
    }
    if !valid {
        return nil, fmt.Errorf("virtual number resource limit exceeded")
    }

    // 3. Register without provider
    return h.Register(
        ctx, customerID, num,
        callFlowID, messageFlowID,
        name, detail,
        number.ProviderNameNone, "",
        number.StatusActive, false, false,
    )
}
```

**Note:** The `Register()` method and `dbCreate()` need to be updated to accept and persist the `type` field.

## API Request Changes

### listenhandler

Update `processV1NumbersPost` in `bin-number-manager/pkg/listenhandler/v1_numbers.go` to:
1. Parse the `type` field from the request
2. If `type` is `"virtual"`, call `numberHandler.CreateVirtual()` with `allowReserved: false`
3. If `type` is `"normal"` or empty, validate number does NOT start with `+899`, then call existing `numberHandler.Create()`

### Request Model

Add `Type` field to `V1DataNumbersPost` in `bin-number-manager/pkg/listenhandler/models/request/v1_numbers.go`:

```go
type V1DataNumbersPost struct {
    CustomerID    uuid.UUID   `json:"customer_id"`
    Number        string      `json:"number"`
    Type          number.Type `json:"type"`
    // ... rest unchanged
}
```

## Files to Create or Modify

### New Files
- `bin-number-manager/models/number/type.go` — Type constants (`TypeNormal`, `TypeVirtual`)
- `bin-number-manager/models/number/validate.go` — `ValidateVirtualNumber()` function
- `bin-number-manager/models/number/validate_test.go` — Validation tests
- `bin-dbscheme-manager/alembic/versions/xxx_add_type_to_number_numbers.py` — DB migration

### Modified Files

**bin-number-manager:**
- `models/number/number.go` — Add `Type` field to struct
- `models/number/field.go` — Add `FieldType` constant
- `pkg/numberhandler/main.go` — Add `CreateVirtual()` and `GetAvailableVirtualNumbers()` to interface
- `pkg/numberhandler/number.go` — Add `CreateVirtual()` implementation, update `Create()` to reject `+899`, update `Delete()` to handle `ProviderNameNone`
- `pkg/numberhandler/available_number.go` — Add `GetAvailableVirtualNumbers()`, update `GetAvailableNumbers()` to route by type
- `pkg/numberhandler/db.go` — Update `dbCreate()` to accept and persist `type` field
- `pkg/listenhandler/v1_numbers.go` — Parse `type` from request, route to `Create()` or `CreateVirtual()`
- `pkg/listenhandler/v1_available_numbers.go` — Parse `type` filter, route to virtual or provider path
- `pkg/listenhandler/models/request/v1_numbers.go` — Add `Type` field to `V1DataNumbersPost`
- `pkg/dbhandler/number.go` — Include `type` in queries; add count-by-customer-and-type query; add bulk existence check for available number generation
- `cmd/number-control/main.go` — Remove `cmdRegister()`, add `--virtual-number` flag to `cmdCreate()`, add `--virtual` flag to `cmdGetAvailable()`

**bin-openapi-manager:**
- `openapi/openapi.yaml` — Add `NumberManagerNumberType` enum schema, add `type` to `NumberManagerNumber`
- `openapi/paths/numbers/main.yaml` — Add `type` field to POST request body
- `openapi/paths/available_numbers/main.yaml` — Add `type` query parameter

**bin-api-manager:**
- Regenerate server code via `go generate ./...`

**bin-common-handler:**
- `pkg/commonbilling/resource_type.go` — Add `ResourceTypeVirtualNumber`
- `pkg/requesthandler/number_numbers.go` — Add `NumberV1NumberGetCountByCustomerIDAndType()` RPC method

**bin-billing-manager:**
- `models/account/plan.go` — Add `VirtualNumbers` to `PlanLimits` struct and `PlanLimitMap`
- `pkg/subscribehandler/number.go` — Skip billing for virtual numbers in `processEventNMNumberCreated`
- `pkg/billinghandler/event.go` — Skip billing for virtual numbers in `EventNMNumberCreated`

**bin-customer-manager:**
- No changes needed (existing `IsValidResourceLimit` wrapper is generic)

## Trade-offs

- **Single endpoint vs separate endpoints**: Using a single `POST /v1/numbers` with a `type` field keeps the API surface small and avoids duplicating CRUD logic. The trade-off is slightly more complex validation in the create handler.
- **+899 prefix**: Spare/unassigned in the ITU-T E.164 numbering plan (Zone 8). No emergency number conflicts. Risk of future assignment is minimal since virtual numbers never touch the telecom network. Originally used +999, changed to +899 to avoid confusion with the 999 emergency number in Commonwealth countries.
- **No billing for virtual numbers**: Simplifies implementation. If monetization is needed later, a new billing reference type can be added.
- **Resource limits via plan tier**: Relies on the plan-tier system from `2026-02-09-resource-limit-plan-tier-design.md`. If that system is not yet implemented, resource limits would need to be deferred or implemented as part of this work.
- **Reserved range only via CLI**: Keeps the API simple while providing admin flexibility. No API-level admin override is needed for now.
