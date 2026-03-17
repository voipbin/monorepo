# Customer Identity Verification Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add identity verification gating to block unverified customers from buying PSTN numbers and making outbound PSTN calls, with a control CLI to manage verification status and a provider interface ready for future KYC integration.

**Architecture:** New `IdentityVerificationStatus` field on Customer model with four statuses (none/pending/verified/rejected). Gating enforced at two layers: API gateway (bin-api-manager) and call manager (bin-call-manager). Admin management via customer-control CLI. Provider-agnostic interface in bin-customer-manager for future Onfido/Sumsub integration.

**Tech Stack:** Go, MySQL (Alembic migrations), RabbitMQ events, Cobra CLI, OpenAPI/oapi-codegen

**Design doc:** `docs/plans/2026-03-18-customer-identity-verification-design.md`

---

### Task 1: Add IdentityVerificationStatus type and constants

**Files:**
- Create: `bin-customer-manager/models/customer/identity_verification_status.go`

**Step 1: Create the type file**

```go
package customer

// IdentityVerificationStatus represents the customer's identity verification state.
type IdentityVerificationStatus string

const (
	IdentityVerificationStatusNone     IdentityVerificationStatus = "none"
	IdentityVerificationStatusPending  IdentityVerificationStatus = "pending"
	IdentityVerificationStatusVerified IdentityVerificationStatus = "verified"
	IdentityVerificationStatusRejected IdentityVerificationStatus = "rejected"
)

// ValidIdentityVerificationStatuses contains all valid status values for input validation.
var ValidIdentityVerificationStatuses = []IdentityVerificationStatus{
	IdentityVerificationStatusNone,
	IdentityVerificationStatusPending,
	IdentityVerificationStatusVerified,
	IdentityVerificationStatusRejected,
}

// IsValid returns true if the status is one of the defined constants.
func (s IdentityVerificationStatus) IsValid() bool {
	for _, v := range ValidIdentityVerificationStatuses {
		if s == v {
			return true
		}
	}
	return false
}
```

**Step 2: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification/bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification
git add bin-customer-manager/models/customer/identity_verification_status.go
git commit -m "NOJIRA-Customer-identity-verification

- bin-customer-manager: Add IdentityVerificationStatus type with none/pending/verified/rejected constants"
```

---

### Task 2: Add field to Customer struct, Field type, and WebhookMessage

**Files:**
- Modify: `bin-customer-manager/models/customer/customer.go`
- Modify: `bin-customer-manager/models/customer/field.go`
- Modify: `bin-customer-manager/models/customer/webhook.go`

**Step 1: Add field to Customer struct**

In `bin-customer-manager/models/customer/customer.go`, add after the `Status` field (line 39), before `TermsAgreedVersion` (line 41):

```go
IdentityVerificationStatus IdentityVerificationStatus `json:"identity_verification_status" db:"identity_verification_status"`
```

**Step 2: Add Field constant**

In `bin-customer-manager/models/customer/field.go`, add after `FieldStatus`:

```go
FieldIdentityVerificationStatus Field = "identity_verification_status"
```

**Step 3: Add to WebhookMessage struct**

In `bin-customer-manager/models/customer/webhook.go`, add the field to `WebhookMessage` struct after `Status` (line 31), before `TMDeletionScheduled` (line 32):

```go
IdentityVerificationStatus IdentityVerificationStatus `json:"identity_verification_status"`
```

**Step 4: Add to ConvertWebhookMessage()**

In the `ConvertWebhookMessage()` method, add the mapping after `Status: h.Status,`:

```go
IdentityVerificationStatus: h.IdentityVerificationStatus,
```

**Step 5: Run verification for bin-customer-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification/bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 6: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification
git add bin-customer-manager/
git commit -m "NOJIRA-Customer-identity-verification

- bin-customer-manager: Add IdentityVerificationStatus field to Customer struct, Field type, and WebhookMessage"
```

---

### Task 3: Add event type

**Files:**
- Modify: `bin-customer-manager/models/customer/event.go`

**Step 1: Add event constant**

In `bin-customer-manager/models/customer/event.go`, add to the const block:

```go
EventTypeCustomerIdentityVerificationUpdated string = "customer_identity_verification_updated"
```

**Step 2: Run verification and commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification/bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification
git add bin-customer-manager/models/customer/event.go
git commit -m "NOJIRA-Customer-identity-verification

- bin-customer-manager: Add customer_identity_verification_updated event type"
```

---

### Task 4: Add UpdateIdentityVerificationStatus to customerhandler

**Files:**
- Modify: `bin-customer-manager/pkg/customerhandler/main.go` (interface)
- Create: `bin-customer-manager/pkg/customerhandler/identity_verification.go` (implementation)

**Step 1: Add method to CustomerHandler interface**

In `bin-customer-manager/pkg/customerhandler/main.go`, add to the `CustomerHandler` interface (after `UpdateMetadata`):

```go
UpdateIdentityVerificationStatus(ctx context.Context, id uuid.UUID, status customer.IdentityVerificationStatus) (*customer.Customer, error)
```

**Step 2: Create implementation file**

Create `bin-customer-manager/pkg/customerhandler/identity_verification.go`:

```go
package customerhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-customer-manager/models/customer"
)

// UpdateIdentityVerificationStatus updates the customer's identity verification status.
func (h *customerHandler) UpdateIdentityVerificationStatus(ctx context.Context, id uuid.UUID, status customer.IdentityVerificationStatus) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "UpdateIdentityVerificationStatus",
		"customer_id": id,
		"status":      status,
	})
	log.Debug("Updating customer identity verification status.")

	if !status.IsValid() {
		return nil, fmt.Errorf("invalid identity verification status: %s", status)
	}

	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return nil, err
	}

	if c.IdentityVerificationStatus == status {
		log.Infof("Customer already has status %s. customer_id: %s", status, id)
		return c, nil
	}

	fields := map[customer.Field]any{
		customer.FieldIdentityVerificationStatus: string(status),
	}
	if err := h.db.CustomerUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update identity verification status. err: %v", err)
		return nil, err
	}

	res, err := h.db.CustomerGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated customer. err: %v", err)
		return nil, fmt.Errorf("could not get updated customer")
	}

	h.notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerIdentityVerificationUpdated, res)

	return res, nil
}
```

**Step 3: Regenerate mocks and run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification/bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification
git add bin-customer-manager/
git commit -m "NOJIRA-Customer-identity-verification

- bin-customer-manager: Add UpdateIdentityVerificationStatus to CustomerHandler interface and implementation"
```

---

### Task 5: Add customer-control CLI command

**Files:**
- Modify: `bin-customer-manager/cmd/customer-control/main.go`

**Step 1: Register the new subcommand**

In `initCommand()`, after `cmdCustomer.AddCommand(cmdRecover())` (line 68), add:

```go
cmdCustomer.AddCommand(cmdSetIdentityVerification())
```

**Step 2: Add command and run functions**

Add before the `initHandler()` function (around line 380):

```go
func cmdSetIdentityVerification() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-identity-verification",
		Short: "Set customer identity verification status",
		RunE:  runSetIdentityVerification,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Customer ID (required)")
	flags.String("status", "", "Verification status: none, pending, verified, rejected (required)")

	return cmd
}

func runSetIdentityVerification(cmd *cobra.Command, args []string) error {
	targetID, err := resolveUUID("id", "Customer ID")
	if err != nil {
		return errors.Wrap(err, "invalid customer ID")
	}

	statusStr, err := resolveString("status", "Status")
	if err != nil {
		return errors.Wrap(err, "invalid status")
	}

	status := customer.IdentityVerificationStatus(statusStr)
	if !status.IsValid() {
		return fmt.Errorf("invalid status '%s': must be one of none, pending, verified, rejected", statusStr)
	}

	handler, err := initHandler()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	res, err := handler.UpdateIdentityVerificationStatus(context.Background(), targetID, status)
	if err != nil {
		return errors.Wrap(err, "failed to update identity verification status")
	}

	return printJSON(res)
}
```

**Step 3: Run verification and commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification/bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification
git add bin-customer-manager/
git commit -m "NOJIRA-Customer-identity-verification

- bin-customer-manager: Add set-identity-verification command to customer-control CLI"
```

---

### Task 6: Add identity verification provider interface

**Files:**
- Create: `bin-customer-manager/pkg/identityverificationhandler/main.go`
- Create: `bin-customer-manager/pkg/identityverificationhandler/noop.go`

**Step 1: Create the interface**

Create `bin-customer-manager/pkg/identityverificationhandler/main.go`:

```go
package identityverificationhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-customer-manager/models/customer"
)

// Session represents a verification session initiated with a provider.
type Session struct {
	ID          string    // Provider-assigned session ID
	CustomerID  uuid.UUID // Customer being verified
	ProviderURL string    // URL to redirect user for verification
}

// Result represents the outcome of a verification session.
type Result struct {
	SessionID  string                              // Provider-assigned session ID
	CustomerID uuid.UUID                           // Customer being verified
	Status     customer.IdentityVerificationStatus // Resulting verification status
	Reason     string                              // Rejection reason, empty if verified
}

// IdentityVerificationProvider defines the interface for identity verification providers.
type IdentityVerificationProvider interface {
	// CreateSession initiates a verification session for a customer.
	CreateSession(ctx context.Context, customerID uuid.UUID) (*Session, error)

	// GetResult retrieves the verification result for a session.
	GetResult(ctx context.Context, sessionID string) (*Result, error)

	// HandleWebhook processes a callback from the verification provider.
	HandleWebhook(ctx context.Context, payload []byte) (*Result, error)
}
```

**Step 2: Create the noop implementation**

Create `bin-customer-manager/pkg/identityverificationhandler/noop.go`:

```go
package identityverificationhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-customer-manager/models/customer"
)

// noopProvider is a no-op implementation that immediately returns verified status.
// Useful for development and testing environments.
type noopProvider struct{}

// NewNoopProvider returns an IdentityVerificationProvider that always succeeds.
func NewNoopProvider() IdentityVerificationProvider {
	return &noopProvider{}
}

func (p *noopProvider) CreateSession(_ context.Context, customerID uuid.UUID) (*Session, error) {
	return &Session{
		ID:          "noop-" + customerID.String(),
		CustomerID:  customerID,
		ProviderURL: "",
	}, nil
}

func (p *noopProvider) GetResult(_ context.Context, sessionID string) (*Result, error) {
	return &Result{
		SessionID:  sessionID,
		CustomerID: uuid.Nil,
		Status:     customer.IdentityVerificationStatusVerified,
		Reason:     "",
	}, nil
}

func (p *noopProvider) HandleWebhook(_ context.Context, _ []byte) (*Result, error) {
	return &Result{
		Status: customer.IdentityVerificationStatusVerified,
		Reason: "",
	}, nil
}
```

**Step 3: Run verification and commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification/bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification
git add bin-customer-manager/pkg/identityverificationhandler/
git commit -m "NOJIRA-Customer-identity-verification

- bin-customer-manager: Add IdentityVerificationProvider interface with noop implementation"
```

---

### Task 7: Add gating in bin-api-manager for NumberCreate and CallCreate

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/numbers.go`
- Modify: `bin-api-manager/pkg/servicehandler/call.go`

**HTTP response:** Verification failures should ideally return **HTTP 403 Forbidden** to distinguish from authentication failures (401) and bad requests (400). However, the current server layer in `bin-api-manager/server/numbers.go` and `server/calls.go` maps ALL servicehandler errors to HTTP 400. To return 403, you must also update the server layer (`PostNumbers` at line 127-130 and `PostCalls` at line 66-70) to check the error message and return 403 for verification failures. Example pattern: `if strings.Contains(err.Error(), "identity verification required") { c.AbortWithStatus(403); return }`. Alternatively, keep 400 for now and add 403 support in a follow-up if needed.

**Step 1: Add verification gate to NumberCreate**

In `bin-api-manager/pkg/servicehandler/numbers.go`, in the `NumberCreate` function, after the permission check (line 107) and before the `h.reqHandler.NumberV1NumberCreate` call (line 110), add:

```go
	// check identity verification for non-virtual number purchases
	if numType != nmnumber.TypeVirtual {
		cu, err := h.customerGet(ctx, a.CustomerID)
		if err != nil {
			log.Errorf("Could not get customer info for verification check. err: %v", err)
			return nil, fmt.Errorf("could not verify customer identity status")
		}
		log.WithField("customer", cu).Debugf("Retrieved customer info for verification check. customer_id: %s", cu.ID)

		if cu.IdentityVerificationStatus != cscustomer.IdentityVerificationStatusVerified {
			log.Infof("Customer identity verification required for number purchase. customer_id: %s, status: %s", a.CustomerID, cu.IdentityVerificationStatus)
			return nil, fmt.Errorf("customer identity verification required for number purchase")
		}
	}
```

This requires adding the import:
```go
cscustomer "monorepo/bin-customer-manager/models/customer"
```

**Step 2: Add verification gate to CallCreate**

In `bin-api-manager/pkg/servicehandler/call.go`, in the `CallCreate` function, after the permission check (line 56) and before the flow creation (line 58), add:

```go
	// check identity verification for PSTN outbound calls
	hasTelDestination := false
	for _, d := range destinations {
		if d.Type == commonaddress.TypeTel {
			hasTelDestination = true
			break
		}
	}
	if hasTelDestination {
		cu, err := h.customerGet(ctx, a.CustomerID)
		if err != nil {
			log.Errorf("Could not get customer info for verification check. err: %v", err)
			return nil, nil, fmt.Errorf("could not verify customer identity status")
		}
		log.WithField("customer", cu).Debugf("Retrieved customer info for verification check. customer_id: %s", cu.ID)

		if cu.IdentityVerificationStatus != cscustomer.IdentityVerificationStatusVerified {
			log.Infof("Customer identity verification required for PSTN calls. customer_id: %s, status: %s", a.CustomerID, cu.IdentityVerificationStatus)
			return nil, nil, fmt.Errorf("customer identity verification required for PSTN calls")
		}
	}
```

This requires adding the import:
```go
cscustomer "monorepo/bin-customer-manager/models/customer"
```

**Step 3: Run verification for bin-api-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification
git add bin-api-manager/
git commit -m "NOJIRA-Customer-identity-verification

- bin-api-manager: Add identity verification gate for non-virtual number purchases
- bin-api-manager: Add identity verification gate for PSTN outbound calls"
```

---

### Task 8: Refactor validate.go and add identity verification gating in bin-call-manager

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/validate.go`
- Modify: `bin-call-manager/pkg/callhandler/outgoing_call.go`

**Key design decision:** `ValidateCustomerNotFrozen` already fetches the customer via RPC. Instead of making a second RPC call for verification, refactor `ValidateCustomerNotFrozen` to return the customer object, then check the verification status on the same object.

**Step 1: Refactor ValidateCustomerNotFrozen to return the customer**

In `bin-call-manager/pkg/callhandler/validate.go`, change `ValidateCustomerNotFrozen` to return `(*cucustomer.Customer, bool)`:

```go
// ValidateCustomerNotFrozen returns the customer and true if the given customer is not frozen.
// Returns nil and false if the customer is frozen. Returns nil and true (fail-open) if
// customer-manager is unavailable.
func (h *callHandler) ValidateCustomerNotFrozen(ctx context.Context, customerID uuid.UUID) (*cucustomer.Customer, bool) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ValidateCustomerNotFrozen",
		"customer_id": customerID,
	})

	cu, err := h.reqHandler.CustomerV1CustomerGet(ctx, customerID)
	if err != nil {
		// Fail open: if customer-manager is unavailable, allow the call rather than
		// rejecting ALL calls. Billing-manager provides a second enforcement layer.
		log.Errorf("Could not get customer info, failing open. err: %v", err)
		return nil, true
	}
	log.WithField("customer", cu).Debugf("Retrieved customer info. customer_id: %s", cu.ID)

	if cu.Status == cucustomer.StatusFrozen {
		log.Infof("Customer account is frozen. Rejecting call.")
		return cu, false
	}

	return cu, true
}
```

**Step 2: Add ValidateCustomerIdentityVerified function**

Add after `ValidateCustomerNotFrozen`:

```go
// ValidateCustomerIdentityVerified returns true if the given customer has verified identity.
// Only checks for outgoing PSTN (TypeTel) calls. Inbound and non-PSTN calls skip this check.
// Known internal customer IDs bypass the check.
// Accepts a pre-fetched customer to avoid redundant RPC calls. If cu is nil (fail-open
// from frozen check), returns true.
func (h *callHandler) ValidateCustomerIdentityVerified(ctx context.Context, cu *cucustomer.Customer, customerID uuid.UUID, direction call.Direction, destination commonaddress.Address) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ValidateCustomerIdentityVerified",
		"customer_id": customerID,
		"direction":   direction,
		"destination": destination,
	})

	// only check outgoing PSTN calls
	if direction != call.DirectionOutgoing || destination.Type != commonaddress.TypeTel {
		return true
	}

	// bypass for known internal/system customer IDs
	if customerID == cucustomer.IDCallManager ||
		customerID == cucustomer.IDAIManager ||
		customerID == cucustomer.IDSystem ||
		customerID == cucustomer.IDBasicRoute {
		log.Debugf("Internal customer ID, bypassing identity verification. customer_id: %s", customerID)
		return true
	}

	// if customer was not fetched (fail-open from frozen check), allow
	if cu == nil {
		log.Debugf("Customer not available (fail-open), bypassing identity verification.")
		return true
	}

	if cu.IdentityVerificationStatus != cucustomer.IdentityVerificationStatusVerified {
		log.Infof("Customer identity not verified. Rejecting outgoing PSTN call. customer_id: %s, status: %s", customerID, cu.IdentityVerificationStatus)
		return false
	}

	return true
}
```

**Step 3: Update CreateCallOutgoing to use refactored validators**

In `bin-call-manager/pkg/callhandler/outgoing_call.go`, replace the existing frozen check (lines 132-136):

```go
	// validate customer is not frozen
	if !h.ValidateCustomerNotFrozen(ctx, customerID) {
		log.Infof("Customer account is frozen. Rejecting outgoing call. customer_id: %s", customerID)
		return nil, fmt.Errorf("customer account is frozen")
	}
```

With:

```go
	// validate customer is not frozen (also fetches customer for subsequent checks)
	cu, notFrozen := h.ValidateCustomerNotFrozen(ctx, customerID)
	if !notFrozen {
		log.Infof("Customer account is frozen. Rejecting outgoing call. customer_id: %s", customerID)
		return nil, fmt.Errorf("customer account is frozen")
	}

	// validate customer identity verification for outgoing PSTN calls
	if !h.ValidateCustomerIdentityVerified(ctx, cu, customerID, call.DirectionOutgoing, destination) {
		log.Infof("Customer identity not verified. Rejecting outgoing PSTN call. customer_id: %s", customerID)
		return nil, fmt.Errorf("customer identity verification required for PSTN calls")
	}
```

**Step 4: Update the other caller of ValidateCustomerNotFrozen in start.go**

The second caller is in `start.go:572` (`startCallTypeFlow`, for incoming calls). Since incoming calls do NOT need identity verification, discard the customer return value:

Replace (line 572):
```go
	if !h.ValidateCustomerNotFrozen(ctx, customerID) {
```

With:
```go
	_, notFrozen := h.ValidateCustomerNotFrozen(ctx, customerID)
	if !notFrozen {
```

Verify no other callers exist:
```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification
grep -rn "ValidateCustomerNotFrozen" bin-call-manager/ --include="*.go" | grep -v "_test.go" | grep -v "mock_"
```

**Step 4b: Update test mocked Customer objects**

Tests in `outgoing_call_test.go` (lines 176, 375) mock `CustomerV1CustomerGet` returning `&cucustomer.Customer{Status: cucustomer.StatusActive}`. After this change, the mocked customer must also include `IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified`, otherwise `ValidateCustomerIdentityVerified` will reject outgoing PSTN calls in tests.

Note: `start_test.go` (lines 256, 452, 646) also mocks `CustomerV1CustomerGet`, but those tests cover incoming call flows that do NOT pass through `ValidateCustomerIdentityVerified`. Adding the field there is harmless but not required — the `ValidateCustomerNotFrozen` return signature change is handled by the compiler (Step 4 already updates the caller).

Example fix for each mocked customer:
```go
// Before
mockReq.EXPECT().CustomerV1CustomerGet(ctx, ...).Return(&cucustomer.Customer{Status: cucustomer.StatusActive}, nil)

// After
mockReq.EXPECT().CustomerV1CustomerGet(ctx, ...).Return(&cucustomer.Customer{
    Status:                     cucustomer.StatusActive,
    IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
}, nil)
```

**Step 5: Run verification for bin-call-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification/bin-call-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 6: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification
git add bin-call-manager/
git commit -m "NOJIRA-Customer-identity-verification

- bin-call-manager: Refactor ValidateCustomerNotFrozen to return customer object (avoids redundant RPC)
- bin-call-manager: Add ValidateCustomerIdentityVerified with internal customer ID bypass
- bin-call-manager: Integrate identity verification check in CreateCallOutgoing flow"
```

**Note:** Groupcall PSTN destinations are automatically covered because groupcalls ultimately call `CreateCallOutgoing` for each resolved address, which includes this verification check.

---

### Task 9: Create Alembic database migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<revision>_add_customer_identity_verification_status.py`

**Step 1: Generate migration file**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification/bin-dbscheme-manager/bin-manager/main
alembic -c alembic.ini revision -m "add customer identity verification status"
```

**Step 2: Edit the generated migration file**

Fill in the `upgrade()` and `downgrade()` functions:

```python
def upgrade():
    op.execute("""ALTER TABLE customer_customers ADD COLUMN identity_verification_status VARCHAR(32) NOT NULL DEFAULT 'none';""")
    # Grandfather all existing active customers to 'verified' so they are not disrupted.
    # New customers created after this migration will default to 'none'.
    op.execute("""UPDATE customer_customers SET identity_verification_status = 'verified' WHERE status = 'active';""")


def downgrade():
    op.execute("""ALTER TABLE customer_customers DROP COLUMN identity_verification_status;""")
```

**IMPORTANT: Do NOT run `alembic upgrade`. Only create and commit the migration file.**

**Deployment ordering:** The migration MUST run BEFORE the new code deploys. If code deploys first, the column won't exist and the service will fail. If the migration runs first without the code, new customers default to `none` but no gating logic exists yet — which is safe.

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification
git add bin-dbscheme-manager/
git commit -m "NOJIRA-Customer-identity-verification

- bin-dbscheme-manager: Add migration for identity_verification_status column on customer_customers
- bin-dbscheme-manager: Grandfather existing active customers to verified status"
```

---

### Task 10: Update OpenAPI schema and regenerate

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml`
- Regenerate: `bin-openapi-manager` and `bin-api-manager`

**IMPORTANT:** Before modifying OpenAPI, read `bin-openapi-manager/CLAUDE.md` for AI-Native Specification Rules.

**Step 1: Add a separate enum schema and reference it in CustomerManagerCustomer**

In `bin-openapi-manager/openapi/openapi.yaml`, first add a new schema definition immediately before `CustomerManagerCustomer` (around line 3301), following the same pattern as `CustomerManagerCustomerStatus` (line 3289):

```yaml
    CustomerManagerCustomerIdentityVerificationStatus:
      type: string
      description: Customer's identity verification status. Determines access to PSTN number purchases and outbound PSTN calls.
      example: "none"
      enum:
        - none
        - pending
        - verified
        - rejected
      x-enum-varnames:
        - CustomerManagerCustomerIdentityVerificationStatusNone
        - CustomerManagerCustomerIdentityVerificationStatusPending
        - CustomerManagerCustomerIdentityVerificationStatusVerified
        - CustomerManagerCustomerIdentityVerificationStatusRejected
```

Then, in the `CustomerManagerCustomer` properties, add after the `status` field (line 3354), before `tm_deletion_scheduled` (line 3357):

```yaml
        identity_verification_status:
          $ref: '#/components/schemas/CustomerManagerCustomerIdentityVerificationStatus'
          description: >-
            Customer's identity verification status. Only 'verified' customers
            can purchase PSTN numbers and make outbound PSTN calls.
          example: "none"
```

**Step 2: Regenerate OpenAPI types**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Regenerate API server code**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification
git add bin-openapi-manager/ bin-api-manager/
git commit -m "NOJIRA-Customer-identity-verification

- bin-openapi-manager: Add identity_verification_status to CustomerManagerCustomer schema
- bin-api-manager: Regenerate server code with new OpenAPI schema"
```

---

### Task 11: Update RST documentation

**Files:**
- Modify: RST docs in `bin-api-manager/docsdev/source/` (customer struct docs)

**Step 1: Find and update customer struct RST**

Find the customer struct RST file and add the `identity_verification_status` field documentation. Match the `WebhookMessage` fields (not internal struct).

**Step 2: Rebuild HTML**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification/bin-api-manager/docsdev
rm -rf build && python3 -m sphinx -M html source build
```

**Step 3: Commit both RST source and built HTML**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification
git add bin-api-manager/docsdev/source/
git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-Customer-identity-verification

- bin-api-manager: Update RST docs with identity_verification_status field
- bin-api-manager: Rebuild HTML documentation"
```

---

### Task 12: Final verification and PR

**Step 1: Run verification for all changed services**

```bash
# bin-customer-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification/bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-api-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-call-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification/bin-call-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-openapi-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Check for conflicts with main**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Customer-identity-verification
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-Customer-identity-verification
```

Create PR with title `NOJIRA-Customer-identity-verification` and body listing all changes.
