# Customer Identity Verification

## Problem Statement

VoIPbin currently has no identity verification gate for customers. Any active customer can buy PSTN numbers and make outbound PSTN calls, which poses both regulatory (KYC/KYB for telecom) and fraud risks. We need a verification mechanism that gates cost-incurring PSTN operations behind identity verification, with a provider-agnostic interface ready for future Onfido/Sumsub integration.

## Approach

Add a top-level `IdentityVerificationStatus` field to the Customer model with four statuses. Gate number buying (non-virtual) and PSTN outbound calling behind `verified` status at both the API layer and call-manager layer. Provide a control CLI command for manual status management. Define a provider interface for future integration.

## Design

### 1. Customer Model Changes

**New type and constants** in `bin-customer-manager/models/customer/`:

```go
type IdentityVerificationStatus string

const (
    IdentityVerificationStatusNone     IdentityVerificationStatus = "none"
    IdentityVerificationStatusPending  IdentityVerificationStatus = "pending"
    IdentityVerificationStatusVerified IdentityVerificationStatus = "verified"
    IdentityVerificationStatusRejected IdentityVerificationStatus = "rejected"
)
```

**Customer struct** — new top-level field:
```go
IdentityVerificationStatus IdentityVerificationStatus `db:"identity_verification_status" json:"identity_verification_status"`
```

**Field type** — new constant:
```go
FieldIdentityVerificationStatus Field = "identity_verification_status"
```

**WebhookMessage** — add `IdentityVerificationStatus` field and include in `ConvertWebhookMessage()`. This is intentionally exposed to external clients so customers can see their own verification status and know they need to verify.

**Customer creation paths** — both `Create()` (admin) and `Signup()` (self-service) must explicitly set `IdentityVerificationStatus: IdentityVerificationStatusNone` in the struct literal. `commondatabasehandler.PrepareFields` uses Go's zero value (`""`) for unset string fields, which bypasses the DB `DEFAULT 'none'`.

### 2. Database Migration

In `bin-dbscheme-manager`, new Alembic migration:

```sql
-- upgrade
ALTER TABLE customer_customers
    ADD COLUMN identity_verification_status VARCHAR(32) NOT NULL DEFAULT 'none';

-- Set all existing active customers to 'verified' (grandfathered in).
-- New customers created after this migration will default to 'none'.
UPDATE customer_customers
    SET identity_verification_status = 'verified'
    WHERE status = 'active';

-- downgrade
ALTER TABLE customer_customers
    DROP COLUMN identity_verification_status;
```

**Migration strategy for existing customers:** All existing active customers are set to `verified` during migration so they are not disrupted. New customers created after deployment will default to `none` and must complete verification before buying numbers or making PSTN calls.

### 3. Gating Logic

Two enforcement points to cover both API-initiated and flow/action-initiated PSTN operations.

#### bin-api-manager (`pkg/servicehandler/`)

- **NumberCreate**: After the permission check and before the `NumberV1NumberCreate` RPC call, fetch the customer and check `IdentityVerificationStatus == verified`. Skip check if number type is `Virtual`.
- **CallCreate**: After the permission check and before flow creation, fetch the customer and check `IdentityVerificationStatus == verified` when any destination is `TypeTel`. No check for SIP/agent/extension/conference destinations.

Error responses should return HTTP 403 Forbidden status to distinguish from authentication failures (401) and bad requests (400).

Error messages:
- `"customer identity verification required for number purchase"`
- `"customer identity verification required for PSTN calls"`

#### bin-call-manager (`pkg/callhandler/validate.go`)

Refactor `ValidateCustomerNotFrozen` to return the fetched `*customer.Customer` object. Then pass it to a new `ValidateCustomerIdentityVerified` function that checks the verification status without making a second RPC call.

- If `direction == DirectionOutgoing` and `destination.Type == TypeTel` and customer's `IdentityVerificationStatus != verified`, reject the call.
- Known internal customer IDs (`IDCallManager`, `IDAIManager`, `IDSystem`, `IDBasicRoute`) bypass the verification check — these are used by internal services for system operations.
- Inbound calls are not checked.
- Groupcall PSTN destinations are covered because groupcalls ultimately call `CreateCallOutgoing` for each resolved address, which includes the verification check.

### 4. Control CLI

New subcommand in `bin-customer-manager/cmd/customer-control/`:

```bash
customer-control customer set-identity-verification --id <uuid> --status <none|pending|verified|rejected>
```

Behavior:
- Validates `--status` is one of the four valid values
- Fetches customer to confirm existence
- Updates `IdentityVerificationStatus` via `customerhandler.UpdateIdentityVerificationStatus()`, which validates the status, checks idempotency, updates the DB, and publishes a `customer_identity_verification_updated` event
- Prints updated customer as JSON to stdout
- Follows existing patterns (same as `freeze`/`recover` commands)

**No listenhandler RPC route is required.** The CLI tool accesses `customerHandler` directly (same process, direct DB/cache access). Future provider webhook handlers will also live within `bin-customer-manager` and can call `customerHandler` directly.

### 5. Identity Verification Provider Interface

New package: `bin-customer-manager/pkg/identityverificationhandler/`

```go
type IdentityVerificationProvider interface {
    CreateSession(ctx context.Context, customerID uuid.UUID) (*Session, error)
    GetResult(ctx context.Context, sessionID string) (*Result, error)
    HandleWebhook(ctx context.Context, payload []byte) (*Result, error)
}

type Session struct {
    ID          string
    CustomerID  uuid.UUID
    ProviderURL string
}

type Result struct {
    SessionID  string
    CustomerID uuid.UUID
    Status     customer.IdentityVerificationStatus
    Reason     string
}
```

**Noop implementation** (`noop.go`): Returns `verified` immediately for dev/test environments.

No actual provider implementations (Onfido/Sumsub) in this phase.

### 6. Events

New event type in `bin-customer-manager/models/customer/`:

```go
EventTypeCustomerIdentityVerificationUpdated = "customer_identity_verification_updated"
```

Fired when `IdentityVerificationStatus` changes via any path (control CLI, future provider webhook). Payload is the updated Customer object, following the existing event pattern.

### 7. OpenAPI & Documentation

**OpenAPI** (`bin-openapi-manager/openapi/openapi.yaml`):
- Add `identity_verification_status` to `CustomerManagerCustomer` schema with enum values and description.
- Regenerate types in `bin-openapi-manager` (`go generate ./...`).
- Regenerate server code in `bin-api-manager` (`go generate ./...`).

**RST docs** (`bin-api-manager/docsdev/source/`):
- Update customer struct docs to include the new field.
- Rebuild HTML.

## Files Changed

| Service | Files | Change |
|---------|-------|--------|
| bin-customer-manager | models/customer/customer.go | Add IdentityVerificationStatus field |
| bin-customer-manager | models/customer/identity_verification_status.go | New type + constants |
| bin-customer-manager | models/customer/field.go | Add FieldIdentityVerificationStatus |
| bin-customer-manager | models/customer/webhook.go | Add field to WebhookMessage + ConvertWebhookMessage |
| bin-customer-manager | models/customer/event.go | Add new event type |
| bin-customer-manager | pkg/identityverificationhandler/ | New package: interface + noop implementation |
| bin-customer-manager | cmd/customer-control/ | Add set-identity-verification subcommand |
| bin-dbscheme-manager | alembic/versions/ | New migration for identity_verification_status column |
| bin-api-manager | pkg/servicehandler/numbers.go | Add verification gate for non-virtual numbers |
| bin-api-manager | pkg/servicehandler/call.go | Add verification gate for PSTN destinations |
| bin-call-manager | pkg/callhandler/validate.go | Add verification check for outbound PSTN calls |
| bin-openapi-manager | openapi/openapi.yaml | Add identity_verification_status to CustomerManagerCustomer |
| bin-api-manager | docsdev/source/ | Update customer struct RST docs |

## Trade-offs

- **Dual enforcement (API + call-manager)**: Slightly redundant for API-initiated calls, but necessary because flows/actions bypass the API layer. In call-manager, `ValidateCustomerNotFrozen` is refactored to return the customer object, so the verification check reuses it — no extra RPC call.
- **No dedicated verification service**: Keeping verification logic in customer-manager for now. Can extract to a dedicated service later if the verification workflow grows complex.
- **No audit trail**: Verification status changes are tracked via events but not stored in a dedicated history table. Sufficient for now; can add history table when provider integration comes.
