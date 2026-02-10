# Resource Limit Plan Tier Design

## Problem Statement

Currently, customers can create unlimited resources (extensions, agents, queues, flows, conferences, trunks) without any restrictions. There is no subscription/plan system to control resource usage per customer. This poses risks for platform capacity and prevents tiered service offerings.

## Approach

Introduce a plan-based resource limit system managed by billing-manager. Each customer is assigned a plan tier that defines maximum resource counts. Billing-manager acts as the central quota authority, checking limits by querying each resource manager for current counts.

## Plan Tiers

Four tiers with the following resource limits:

| Resource       | Free | Basic | Professional | Unlimited |
|----------------|:----:|:-----:|:------------:|:---------:|
| Extensions     |    5 |    50 |          500 |  0 (none) |
| Agents         |    5 |    50 |          500 |  0 (none) |
| Queues         |    2 |    10 |          100 |  0 (none) |
| Flows          |    5 |    50 |          500 |  0 (none) |
| Conferences    |    2 |    10 |          100 |  0 (none) |
| Trunks         |    1 |     5 |           50 |  0 (none) |

- A limit of `0` means unlimited (no restriction enforced).
- New customers default to `free`.
- Only Free tier is active initially. Basic, Professional, and Unlimited are defined but reserved for future use.
- Only platform admins can change a customer's plan.

## Resource Type Constants

Defined in `bin-common-handler/pkg/commonbilling/resource_type.go`:

```go
package commonbilling

type ResourceType string

const (
    ResourceTypeExtension  ResourceType = "extension"
    ResourceTypeAgent      ResourceType = "agent"
    ResourceTypeQueue      ResourceType = "queue"
    ResourceTypeFlow       ResourceType = "flow"
    ResourceTypeConference ResourceType = "conference"
    ResourceTypeTrunk      ResourceType = "trunk"
)
```

## Plan Model & Definitions

Plan definitions live as Go code in `bin-billing-manager/models/account/`:

```go
type PlanType string

const (
    PlanTypeFree         PlanType = "free"
    PlanTypeBasic        PlanType = "basic"
    PlanTypeProfessional PlanType = "professional"
    PlanTypeUnlimited    PlanType = "unlimited"
)
```

Plan limit lookup implemented as a map in billing-manager:

```go
type PlanLimits struct {
    Extensions  int
    Agents      int
    Queues      int
    Flows       int
    Conferences int
    Trunks      int
}

var PlanLimitMap = map[PlanType]PlanLimits{
    PlanTypeFree:         {Extensions: 5, Agents: 5, Queues: 2, Flows: 5, Conferences: 2, Trunks: 1},
    PlanTypeBasic:        {Extensions: 50, Agents: 50, Queues: 10, Flows: 50, Conferences: 10, Trunks: 5},
    PlanTypeProfessional: {Extensions: 500, Agents: 500, Queues: 100, Flows: 500, Conferences: 100, Trunks: 50},
    PlanTypeUnlimited:    {Extensions: 0, Agents: 0, Queues: 0, Flows: 0, Conferences: 0, Trunks: 0},
}
```

## Account Model Change

Add `PlanType` field to the existing billing Account model in `bin-billing-manager/models/account/account.go`:

```go
type Account struct {
    commonidentity.Identity

    Name   string `json:"name" db:"name"`
    Detail string `json:"detail" db:"detail"`

    Type     Type     `json:"type" db:"type"`
    PlanType PlanType `json:"plan_type" db:"plan_type"` // NEW

    Balance float32 `json:"balance" db:"balance"`

    PaymentType   PaymentType   `json:"payment_type" db:"payment_type"`
    PaymentMethod PaymentMethod `json:"payment_method" db:"payment_method"`

    TMCreate *time.Time `json:"tm_create" db:"tm_create"`
    TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
    TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}
```

## Database Migration

In `bin-dbscheme-manager`, add Alembic migration:

```sql
-- upgrade
ALTER TABLE billing_accounts
    ADD COLUMN plan_type VARCHAR(255) NOT NULL DEFAULT 'free' AFTER type;

-- downgrade
ALTER TABLE billing_accounts
    DROP COLUMN plan_type;
```

All existing customers automatically receive `"free"` plan via the column default.

## Quota Check Flow

### Full Call Chain

```
Service (e.g., registrar-manager)
  calls Create() for a resource
    -> CustomerV1CustomerIsValidResourceLimit() [RPC to customer-manager]
      -> customer-manager.IsValidResourceLimit()
        -> BillingV1AccountIsValidResourceLimit() [RPC to billing-manager]
          -> billing-manager.IsValidResourceLimit()
            -> look up customer's PlanType from account
            -> look up plan limits from PlanLimitMap
            -> if limit == 0: return true (unlimited)
            -> call back to resource manager for current count
               e.g., RegistrarV1ExtensionGetCountByCustomerID() [RPC]
            -> compare count vs limit
            -> return allowed: true/false
```

### New RPC Endpoint in billing-manager

```
POST /v1/accounts/{account-id}/is_valid_resource_limit

Request:
{
    "resource_type": "extension"
}

Response:
{
    "valid": true
}
```

### New Count Endpoints in Each Resource Manager

Each resource manager exposes a count endpoint that billing-manager queries:

| Resource Manager     | Endpoint                                          |
|----------------------|---------------------------------------------------|
| registrar-manager    | `GET /v1/extensions/count_by_customer?customer_id=uuid` |
| agent-manager        | `GET /v1/agents/count_by_customer?customer_id=uuid`     |
| queue-manager        | `GET /v1/queues/count_by_customer?customer_id=uuid`     |
| flow-manager         | `GET /v1/flows/count_by_customer?customer_id=uuid`      |
| conference-manager   | `GET /v1/conferences/count_by_customer?customer_id=uuid` |
| registrar-manager    | `GET /v1/trunks/count_by_customer?customer_id=uuid`     |

Each returns:

```json
{
    "count": 4
}
```

### Limit Check Logic in billing-manager

```go
func (h *accountHandler) IsValidResourceLimit(
    ctx context.Context,
    accountID uuid.UUID,
    resourceType commonbilling.ResourceType,
) (bool, error) {
    // 1. Get account
    a, err := h.Get(ctx, accountID)
    if err != nil {
        return false, err
    }

    // 2. Get plan limits
    limits, ok := PlanLimitMap[a.PlanType]
    if !ok {
        return false, fmt.Errorf("unknown plan type: %s", a.PlanType)
    }

    // 3. Get limit for this resource type
    limit := limits.GetLimit(resourceType)
    if limit == 0 {
        return true, nil // unlimited
    }

    // 4. Query resource manager for current count
    count, err := h.getResourceCount(ctx, a.CustomerID, resourceType)
    if err != nil {
        return false, err
    }

    // 5. Compare
    if count >= limit {
        return false, nil // limit reached
    }

    return true, nil
}
```

## Integration into Service Creation Flows

Each service's `Create()` method adds a validation step before any resource creation:

```go
func (h *extensionHandler) Create(ctx context.Context, customerID uuid.UUID, ...) (*extension.Extension, error) {
    // Check resource limit
    valid, err := h.reqHandler.CustomerV1CustomerIsValidResourceLimit(
        ctx, customerID, commonbilling.ResourceTypeExtension)
    if err != nil {
        return nil, fmt.Errorf("could not validate resource limit: %w", err)
    }
    if !valid {
        return nil, fmt.Errorf("resource limit exceeded")
    }

    // ... existing creation logic unchanged
}
```

### Services Requiring Changes

| Service                | Create Method              | ResourceType Constant      |
|------------------------|----------------------------|----------------------------|
| bin-registrar-manager  | `extensionHandler.Create()` | `ResourceTypeExtension`   |
| bin-agent-manager      | `agentHandler.Create()`     | `ResourceTypeAgent`       |
| bin-queue-manager      | `queueHandler.Create()`     | `ResourceTypeQueue`       |
| bin-flow-manager       | `flowHandler.Create()`      | `ResourceTypeFlow`        |
| bin-conference-manager | `conferenceHandler.Create()`| `ResourceTypeConference`  |
| bin-registrar-manager  | `trunkHandler.Create()`     | `ResourceTypeTrunk`       |

## New Request Handler Methods in bin-common-handler

```go
// bin-common-handler/pkg/requesthandler/billing_accounts.go
func (h *requestHandler) BillingV1AccountIsValidResourceLimit(
    ctx context.Context,
    accountID uuid.UUID,
    resourceType commonbilling.ResourceType,
) (bool, error)

// bin-common-handler/pkg/requesthandler/customer_*.go
func (h *requestHandler) CustomerV1CustomerIsValidResourceLimit(
    ctx context.Context,
    customerID uuid.UUID,
    resourceType commonbilling.ResourceType,
) (bool, error)

// Count endpoints (one per resource manager)
func (h *requestHandler) RegistrarV1ExtensionGetCountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error)
func (h *requestHandler) AgentV1AgentGetCountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error)
func (h *requestHandler) QueueV1QueueGetCountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error)
func (h *requestHandler) FlowV1FlowGetCountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error)
func (h *requestHandler) ConferenceV1ConferenceGetCountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error)
func (h *requestHandler) RegistrarV1TrunkGetCountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error)
```

## New Wrapper Method in bin-customer-manager

Following the existing `IsValidBalance` pattern in `bin-customer-manager/pkg/customerhandler/etc.go`:

```go
func (h *customerHandler) IsValidResourceLimit(
    ctx context.Context,
    customerID uuid.UUID,
    resourceType commonbilling.ResourceType,
) (bool, error) {
    c, err := h.Get(ctx, customerID)
    if err != nil {
        return false, err
    }

    valid, err := h.reqHandler.BillingV1AccountIsValidResourceLimit(
        ctx, c.BillingAccountID, resourceType)
    if err != nil {
        return false, err
    }

    return valid, nil
}
```

## Files to Create or Modify

### New Files
- `bin-common-handler/pkg/commonbilling/resource_type.go` - ResourceType constants
- `bin-dbscheme-manager/alembic/versions/xxx_add_plan_type_to_billing_accounts.py` - DB migration

### Modified Files

**bin-billing-manager:**
- `models/account/account.go` - Add PlanType field and constants
- `models/account/plan.go` (new) - PlanLimits struct and PlanLimitMap
- `pkg/accounthandler/main.go` - Add IsValidResourceLimit to interface
- `pkg/accounthandler/resource_limit.go` (new) - IsValidResourceLimit implementation
- `pkg/listenhandler/v1_accounts.go` - Add route for is_valid_resource_limit
- `pkg/listenhandler/models/request/v1_accounts.go` - Add request struct
- `pkg/listenhandler/models/response/account.go` - Add response struct (reuse V1ResponseAccountsIDIsValidBalance pattern)
- `pkg/dbhandler/` - Include plan_type in account queries

**bin-common-handler:**
- `pkg/requesthandler/billing_accounts.go` - Add BillingV1AccountIsValidResourceLimit
- `pkg/requesthandler/customer_*.go` - Add CustomerV1CustomerIsValidResourceLimit
- `pkg/requesthandler/registrar_extensions.go` - Add RegistrarV1ExtensionGetCountByCustomerID
- `pkg/requesthandler/registrar_trunks.go` - Add RegistrarV1TrunkGetCountByCustomerID
- `pkg/requesthandler/agent_agents.go` - Add AgentV1AgentGetCountByCustomerID
- `pkg/requesthandler/queue_queues.go` - Add QueueV1QueueGetCountByCustomerID
- `pkg/requesthandler/flow_flows.go` - Add FlowV1FlowGetCountByCustomerID
- `pkg/requesthandler/conference_conferences.go` - Add ConferenceV1ConferenceGetCountByCustomerID

**bin-customer-manager:**
- `pkg/customerhandler/etc.go` - Add IsValidResourceLimit wrapper
- `pkg/listenhandler/` - Add route for customer-level resource limit check

**Resource managers (each needs count endpoint):**
- `bin-registrar-manager/pkg/listenhandler/` - Add extensions and trunks count routes
- `bin-registrar-manager/pkg/extensionhandler/` - Add GetCountByCustomerID
- `bin-registrar-manager/pkg/trunkhandler/` - Add GetCountByCustomerID
- `bin-registrar-manager/pkg/dbhandler/` - Add count queries
- `bin-agent-manager/pkg/listenhandler/` - Add agents count route
- `bin-agent-manager/pkg/agenthandler/` - Add GetCountByCustomerID
- `bin-agent-manager/pkg/dbhandler/` - Add count query
- `bin-queue-manager/pkg/listenhandler/` - Add queues count route
- `bin-queue-manager/pkg/queuehandler/` - Add GetCountByCustomerID
- `bin-queue-manager/pkg/dbhandler/` - Add count query
- `bin-flow-manager/pkg/listenhandler/` - Add flows count route
- `bin-flow-manager/pkg/flowhandler/` - Add GetCountByCustomerID
- `bin-flow-manager/pkg/dbhandler/` - Add count query
- `bin-conference-manager/pkg/listenhandler/` - Add conferences count route
- `bin-conference-manager/pkg/conferencehandler/` - Add GetCountByCustomerID
- `bin-conference-manager/pkg/dbhandler/` - Add count query

**Integration into creation flows:**
- `bin-registrar-manager/pkg/extensionhandler/extension.go` - Add limit check in Create()
- `bin-registrar-manager/pkg/trunkhandler/trunk.go` - Add limit check in Create()
- `bin-agent-manager/pkg/agenthandler/db.go` - Add limit check in Create()
- `bin-queue-manager/pkg/queuehandler/create.go` - Add limit check in Create()
- `bin-flow-manager/pkg/listenhandler/v1_flows.go` - Add limit check in Create()
- `bin-conference-manager/pkg/conferencehandler/conference.go` - Add limit check in Create()

## Trade-offs

- **Latency**: Each resource creation now involves 3 RPC hops (service -> customer-manager -> billing-manager -> resource-manager). This adds ~100-200ms. Acceptable since resource creation is infrequent.
- **No caching of counts**: Counts are queried live each time. This ensures accuracy but adds load. Caching could be added later if needed.
- **Plan definitions in code**: Changing limits requires a deploy. This is acceptable since plan changes are rare. Can be moved to database later if self-service plan management is needed.
- **No partial limit overrides**: A customer is on one plan with fixed limits. Per-customer overrides would require a separate mechanism if needed in the future.
