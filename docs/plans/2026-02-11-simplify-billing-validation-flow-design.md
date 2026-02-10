# Simplify Billing Validation Flow

## Problem

Two issues with the current billing validation architecture:

1. **Misplaced `ResourceType`**: `bin-common-handler/models/billing/resource_type.go` defines `ResourceType` (extension, agent, queue, flow, conference, trunk, virtual_number) which is used for plan-based resource quota enforcement, not billing. It lives in a `billing` package despite having nothing to do with billing calculations.

2. **Unnecessary indirection through customer-manager**: Both balance checks and resource limit checks follow an indirect path:
   ```
   Service → customer-manager (customerID → billingAccountID) → billing-manager
   ```
   Customer-manager's only role is translating `customerID` to `billingAccountID` and forwarding. This adds an extra RPC hop for every validation call. Billing-manager already has `GetByCustomerID` and `IsValidBalanceByCustomerID` internally, proving it can handle customer-based lookups directly.

## Approach

- Move `ResourceType` to `bin-billing-manager/models/account/` alongside `PlanLimits` (which defines limits per resource type)
- Add `IsValidResourceLimitByCustomerID` to billing-manager (following the existing `IsValidBalanceByCustomerID` pattern)
- Expose both `ByCustomerID` variants via new billing-manager RPC endpoints
- Update all calling services to call billing-manager directly
- Remove the customer-manager proxy methods and routes

## Changes by Service

### 1. bin-billing-manager/models/account/

**New file: `resource_type.go`**

Move `ResourceType` type and 7 constants from `bin-common-handler/models/billing/resource_type.go`:
- `ResourceTypeExtension`
- `ResourceTypeAgent`
- `ResourceTypeQueue`
- `ResourceTypeFlow`
- `ResourceTypeConference`
- `ResourceTypeTrunk`
- `ResourceTypeVirtualNumber`

**Modify: `plan.go`**

Update `PlanLimits.GetLimit()` to use the local `ResourceType` (same package) instead of importing `commonbilling`.

### 2. bin-billing-manager/pkg/accounthandler/

**Modify: `main.go`**

- Add `IsValidResourceLimitByCustomerID(ctx, customerID, resourceType)` to `AccountHandler` interface
- Update `IsValidResourceLimit` signature to use `account.ResourceType` instead of `commonbilling.ResourceType`
- Remove `commonbilling` import

**New method: `IsValidResourceLimitByCustomerID`** (in `resource_limit.go`)

Follow the exact pattern of `IsValidBalanceByCustomerID`:
```go
func (h *accountHandler) IsValidResourceLimitByCustomerID(ctx context.Context, customerID uuid.UUID, resourceType account.ResourceType) (bool, error) {
    a, err := h.GetByCustomerID(ctx, customerID)
    if err != nil {
        return false, ...
    }
    return h.IsValidResourceLimit(ctx, a.ID, resourceType)
}
```

**Modify: `resource_limit.go`**

- Update `IsValidResourceLimit` and `getResourceCount` to use `account.ResourceType` instead of `commonbilling.ResourceType`

**Add tests** for `IsValidResourceLimitByCustomerID`.

### 3. bin-billing-manager/pkg/listenhandler/

**Add two new RPC routes:**
- `POST /v1/accounts/is_valid_balance_by_customer_id` — accepts `customerID` in request body
- `POST /v1/accounts/is_valid_resource_limit_by_customer_id` — accepts `customerID` in request body

### 4. bin-common-handler/models/billing/

**Delete: `resource_type.go`**

Package becomes empty — delete the entire `models/billing/` directory.

### 5. bin-common-handler/pkg/requesthandler/

**Add new methods:**
- `BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, billingType, country, count) (bool, error)`
- `BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, resourceType) (bool, error)`

**Remove old methods:**
- `CustomerV1CustomerIsValidBalance`
- `CustomerV1CustomerIsValidResourceLimit`

Update the `RequestHandler` interface accordingly.

### 6. bin-customer-manager/pkg/customerhandler/

**Modify: `etc.go`**
- Remove `IsValidBalance` method
- Remove `IsValidResourceLimit` method

**Modify: `main.go`**
- Remove both methods from `CustomerHandler` interface

### 7. bin-customer-manager/pkg/listenhandler/

- Remove routes for `is_valid_balance` and `is_valid_resource_limit`

### 8. Calling services — update validation calls

**Resource limit checks** (switch from `CustomerV1CustomerIsValidResourceLimit` to `BillingV1AccountIsValidResourceLimitByCustomerID`):

| Service | File |
|---|---|
| bin-agent-manager | `pkg/agenthandler/agent.go` |
| bin-queue-manager | `pkg/queuehandler/create.go` |
| bin-flow-manager | `pkg/flowhandler/db.go` |
| bin-conference-manager | `pkg/conferencehandler/conference.go` |
| bin-registrar-manager | `pkg/extensionhandler/extension.go` |
| bin-registrar-manager | `pkg/trunkhandler/trunk.go` |
| bin-number-manager | `pkg/numberhandler/number.go` |

All also update their import from `commonbilling "monorepo/bin-common-handler/models/billing"` to `bmaccount "monorepo/bin-billing-manager/models/account"`.

**Balance checks** (switch from `CustomerV1CustomerIsValidBalance` to `BillingV1AccountIsValidBalanceByCustomerID`):

| Service | File |
|---|---|
| bin-call-manager | `pkg/callhandler/validate.go` |
| bin-message-manager | `pkg/messagehandler/send.go` |
| bin-number-manager | `pkg/numberhandler/number.go` |

### 9. Test updates

- **billing-manager**: Add tests for `IsValidResourceLimitByCustomerID`, update existing tests for import changes
- **customer-manager**: Remove tests for deleted methods in `etc_test.go`
- **All calling services**: Update mock expectations in test files (change from `CustomerV1Customer*` to `BillingV1Account*ByCustomerID`)

## Verification

Since this touches `bin-common-handler`, run full verification across all 30+ services:
```bash
for dir in bin-*/; do
  if [ -f "$dir/go.mod" ]; then
    echo "=== $dir ===" && \
    (cd "$dir" && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m) || echo "FAILED: $dir"
  fi
done
```

## What does NOT change

- Actual billing logic (costs, balances, charges) — untouched
- `ReferenceType` in `bin-billing-manager/models/billing/` — stays where it is
- Database schema — no migrations needed
- API-facing endpoints in bin-api-manager — untouched (api-manager calls billing/customer managers internally)
