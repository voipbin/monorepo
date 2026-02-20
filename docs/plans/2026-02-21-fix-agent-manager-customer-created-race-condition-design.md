# Fix Agent Manager Customer Created Race Condition

## Problem

When a new customer is created, the `customer_created` event is published and consumed by two services asynchronously:

1. **billing-manager** creates a billing account and updates the customer's `BillingAccountID`
2. **agent-manager** creates a default admin agent, which requires validating resource limits against the billing account

If agent-manager processes the event before billing-manager has finished, the resource limit validation call (`BillingV1AccountIsValidResourceLimitByCustomerID`) fails because the customer's `BillingAccountID` is still `uuid.Nil`, and the billing account lookup returns "Not Found".

Error chain:
```
agent-manager: EventCustomerCreated
  -> agentHandler.Create
    -> reqHandler.BillingV1AccountIsValidResourceLimitByCustomerID
      -> billing-manager: GetByCustomerID
        -> CustomerV1CustomerGet (returns customer with BillingAccountID = uuid.Nil)
        -> db.AccountGet(uuid.Nil) -> Not Found
```

## Approach

Skip resource limit validation when creating the initial admin agent in `EventCustomerCreated`. The three pre-checks in `Create` are all logically unnecessary for this scenario:

1. **Resource limit validation** - The customer was just created and has zero agents. It cannot exceed any plan limit.
2. **Username format validation** - The customer's email was already validated at signup time.
3. **Username uniqueness check** - The customer was just created and no agents exist yet.

## Change

**File:** `bin-agent-manager/pkg/agenthandler/event.go`

In `EventCustomerCreated` (line 153), change `h.Create(...)` to `h.dbCreate(...)`.

`dbCreate` handles password hashing, DB insertion, and event publishing - everything needed for agent creation. It's already called by `Create` itself (`agent.go:124`), so it's a well-tested code path.

## Alternatives Considered

- **Retry with backoff** in agent-manager: Adds complexity, still fragile if billing-manager takes longer than retries.
- **Chain events** (agent-manager listens to `account_created` instead of `customer_created`): Clean but more complex refactor, changes the event dependency chain.
