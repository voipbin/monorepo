# Block Outbound Calls for Non-Active Customers — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Only allow outbound calls for `active` customers and incoming calls for `active` or `initial` customers.

**Architecture:** Replace `ValidateCustomerNotFrozen()` with two direction-specific functions (`ValidateCustomerStatusOutgoing`, `ValidateCustomerStatusIncoming`) in `bin-call-manager/pkg/callhandler/validate.go`. Update 2 call sites. Add unit tests.

**Tech Stack:** Go, gomock, bin-call-manager

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Block-outbound-call-non-active-customer`
**Service directory:** `bin-call-manager`

---

### Task 1: Write failing tests for `ValidateCustomerStatusOutgoing`

**Files:**
- Create: `bin-call-manager/pkg/callhandler/validate_test.go`

**Step 1: Write the test file**

```go
package callhandler

import (
	"context"
	"fmt"
	"testing"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_ValidateCustomerStatusOutgoing(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID

		responseCustomer *cucustomer.Customer
		responseErr      error

		expectCustomer *cucustomer.Customer
		expectValid    bool
	}{
		{
			name:       "active - allowed",
			customerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
				Status: cucustomer.StatusActive,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
				Status: cucustomer.StatusActive,
			},
			expectValid: true,
		},
		{
			name:       "initial - rejected",
			customerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000002"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000002"),
				Status: cucustomer.StatusInitial,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000002"),
				Status: cucustomer.StatusInitial,
			},
			expectValid: false,
		},
		{
			name:       "frozen - rejected",
			customerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000003"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000003"),
				Status: cucustomer.StatusFrozen,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000003"),
				Status: cucustomer.StatusFrozen,
			},
			expectValid: false,
		},
		{
			name:       "expired - rejected",
			customerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000004"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000004"),
				Status: cucustomer.StatusExpired,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000004"),
				Status: cucustomer.StatusExpired,
			},
			expectValid: false,
		},
		{
			name:       "deleted - rejected",
			customerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000005"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000005"),
				Status: cucustomer.StatusDeleted,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000005"),
				Status: cucustomer.StatusDeleted,
			},
			expectValid: false,
		},
		{
			name:             "customer-manager unavailable - fail open",
			customerID:       uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000006"),
			responseCustomer: nil,
			responseErr:      fmt.Errorf("connection refused"),
			expectCustomer:   nil,
			expectValid:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &callHandler{
				reqHandler:  mockReq,
				utilHandler: mockUtil,
			}

			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(tt.responseCustomer, tt.responseErr)

			cu, valid := h.ValidateCustomerStatusOutgoing(ctx, tt.customerID)
			if valid != tt.expectValid {
				t.Errorf("ValidateCustomerStatusOutgoing() valid = %v, want %v", valid, tt.expectValid)
			}

			if tt.expectCustomer == nil {
				if cu != nil {
					t.Errorf("ValidateCustomerStatusOutgoing() customer = %v, want nil", cu)
				}
			} else {
				if cu == nil {
					t.Errorf("ValidateCustomerStatusOutgoing() customer = nil, want %v", tt.expectCustomer)
				} else if cu.ID != tt.expectCustomer.ID {
					t.Errorf("ValidateCustomerStatusOutgoing() customer.ID = %v, want %v", cu.ID, tt.expectCustomer.ID)
				}
			}
		})
	}
}

func Test_ValidateCustomerStatusIncoming(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID

		responseCustomer *cucustomer.Customer
		responseErr      error

		expectCustomer *cucustomer.Customer
		expectValid    bool
	}{
		{
			name:       "active - allowed",
			customerID: uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001"),
				Status: cucustomer.StatusActive,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001"),
				Status: cucustomer.StatusActive,
			},
			expectValid: true,
		},
		{
			name:       "initial - allowed",
			customerID: uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000002"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000002"),
				Status: cucustomer.StatusInitial,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000002"),
				Status: cucustomer.StatusInitial,
			},
			expectValid: true,
		},
		{
			name:       "frozen - rejected",
			customerID: uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000003"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000003"),
				Status: cucustomer.StatusFrozen,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000003"),
				Status: cucustomer.StatusFrozen,
			},
			expectValid: false,
		},
		{
			name:       "expired - rejected",
			customerID: uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000004"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000004"),
				Status: cucustomer.StatusExpired,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000004"),
				Status: cucustomer.StatusExpired,
			},
			expectValid: false,
		},
		{
			name:       "deleted - rejected",
			customerID: uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000005"),
			responseCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000005"),
				Status: cucustomer.StatusDeleted,
			},
			responseErr: nil,
			expectCustomer: &cucustomer.Customer{
				ID:     uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000005"),
				Status: cucustomer.StatusDeleted,
			},
			expectValid: false,
		},
		{
			name:             "customer-manager unavailable - fail open",
			customerID:       uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000006"),
			responseCustomer: nil,
			responseErr:      fmt.Errorf("connection refused"),
			expectCustomer:   nil,
			expectValid:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &callHandler{
				reqHandler:  mockReq,
				utilHandler: mockUtil,
			}

			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(tt.responseCustomer, tt.responseErr)

			cu, valid := h.ValidateCustomerStatusIncoming(ctx, tt.customerID)
			if valid != tt.expectValid {
				t.Errorf("ValidateCustomerStatusIncoming() valid = %v, want %v", valid, tt.expectValid)
			}

			if tt.expectCustomer == nil {
				if cu != nil {
					t.Errorf("ValidateCustomerStatusIncoming() customer = %v, want nil", cu)
				}
			} else {
				if cu == nil {
					t.Errorf("ValidateCustomerStatusIncoming() customer = nil, want %v", tt.expectCustomer)
				} else if cu.ID != tt.expectCustomer.ID {
					t.Errorf("ValidateCustomerStatusIncoming() customer.ID = %v, want %v", cu.ID, tt.expectCustomer.ID)
				}
			}
		})
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Block-outbound-call-non-active-customer/bin-call-manager && go test -v ./pkg/callhandler/ -run "Test_ValidateCustomerStatus"`
Expected: FAIL — `ValidateCustomerStatusOutgoing` and `ValidateCustomerStatusIncoming` are not defined

---

### Task 2: Implement `ValidateCustomerStatusOutgoing` and `ValidateCustomerStatusIncoming`

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/validate.go:18-42`

**Step 1: Replace `ValidateCustomerNotFrozen` with the two new functions**

Remove lines 18-42 (`ValidateCustomerNotFrozen`) and replace with:

```go
// ValidateCustomerStatusOutgoing returns the customer and true if the customer status is active.
// Only active customers are allowed to make outgoing calls.
// Returns (customer, false) if the status is not active.
// Returns (nil, true) if customer-manager is unavailable (fail-open).
func (h *callHandler) ValidateCustomerStatusOutgoing(ctx context.Context, customerID uuid.UUID) (*cucustomer.Customer, bool) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ValidateCustomerStatusOutgoing",
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

	if cu.Status != cucustomer.StatusActive {
		log.Infof("Customer account is not active. Rejecting outgoing call. status: %s", cu.Status)
		return cu, false
	}

	return cu, true
}

// ValidateCustomerStatusIncoming returns the customer and true if the customer status is active or initial.
// Active and initial customers are allowed to receive incoming calls.
// Returns (customer, false) if the status is not active or initial.
// Returns (nil, true) if customer-manager is unavailable (fail-open).
func (h *callHandler) ValidateCustomerStatusIncoming(ctx context.Context, customerID uuid.UUID) (*cucustomer.Customer, bool) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ValidateCustomerStatusIncoming",
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

	if cu.Status != cucustomer.StatusActive && cu.Status != cucustomer.StatusInitial {
		log.Infof("Customer account is not active or initial. Rejecting incoming call. status: %s", cu.Status)
		return cu, false
	}

	return cu, true
}
```

**Step 2: Run the new tests to verify they pass**

Run: `cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Block-outbound-call-non-active-customer/bin-call-manager && go test -v ./pkg/callhandler/ -run "Test_ValidateCustomerStatus"`
Expected: PASS — all 12 test cases pass

---

### Task 3: Update call sites

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/outgoing_call.go:132-137`
- Modify: `bin-call-manager/pkg/callhandler/start.go:571-577`

**Step 1: Update outgoing_call.go**

Replace lines 132-137:
```go
	// validate customer is not frozen (also fetches customer for subsequent checks)
	cu, notFrozen := h.ValidateCustomerNotFrozen(ctx, customerID)
	if !notFrozen {
		log.Infof("Customer account is frozen. Rejecting outgoing call. customer_id: %s", customerID)
		return nil, fmt.Errorf("customer account is frozen")
	}
```

With:
```go
	// validate customer status (also fetches customer for subsequent checks)
	cu, validStatus := h.ValidateCustomerStatusOutgoing(ctx, customerID)
	if !validStatus {
		log.Infof("Customer account is not active. Rejecting outgoing call. customer_id: %s", customerID)
		return nil, fmt.Errorf("customer account is not active")
	}
```

**Step 2: Update start.go**

Replace lines 571-577:
```go
	// validate customer is not frozen
	_, notFrozen := h.ValidateCustomerNotFrozen(ctx, customerID)
	if !notFrozen {
		log.Errorf("Customer account is frozen. Rejecting incoming call. customer_id: %s", customerID)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder)
		return
	}
```

With:
```go
	// validate customer status
	_, validStatus := h.ValidateCustomerStatusIncoming(ctx, customerID)
	if !validStatus {
		log.Errorf("Customer account is not active. Rejecting incoming call. customer_id: %s", customerID)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder)
		return
	}
```

**Step 3: Run all callhandler tests to verify nothing broke**

Run: `cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Block-outbound-call-non-active-customer/bin-call-manager && go test -v ./pkg/callhandler/...`
Expected: PASS — all existing tests still pass (they mock `CustomerV1CustomerGet` returning `StatusActive`)

---

### Task 4: Update test comments

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/start_test.go` (lines 255, 452, 646)

**Step 1: Update comments**

Replace all 3 occurrences of:
```go
			// Times(2): first call for ValidateCustomerNotFrozen, second for RTP debug check
```

With:
```go
			// Times(2): first call for ValidateCustomerStatusIncoming, second for RTP debug check
```

**Step 2: Run tests to confirm**

Run: `cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Block-outbound-call-non-active-customer/bin-call-manager && go test -v ./pkg/callhandler/...`
Expected: PASS

---

### Task 5: Run full verification workflow and commit

**Step 1: Run verification workflow**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Block-outbound-call-non-active-customer/bin-call-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All steps pass with no errors.

**Step 2: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Block-outbound-call-non-active-customer
git add docs/plans/2026-04-02-block-outbound-call-non-active-customer-design.md
git add docs/plans/2026-04-02-block-outbound-call-non-active-customer.md
git add bin-call-manager/pkg/callhandler/validate.go
git add bin-call-manager/pkg/callhandler/validate_test.go
git add bin-call-manager/pkg/callhandler/outgoing_call.go
git add bin-call-manager/pkg/callhandler/start.go
git add bin-call-manager/pkg/callhandler/start_test.go
git commit -m "NOJIRA-Block-outbound-call-non-active-customer

Block outbound calls for non-active customers and restrict incoming calls
to active or initial status customers only.

- bin-call-manager: Replace ValidateCustomerNotFrozen with ValidateCustomerStatusOutgoing and ValidateCustomerStatusIncoming
- bin-call-manager: Outgoing calls require customer status active
- bin-call-manager: Incoming calls require customer status active or initial
- bin-call-manager: Add unit tests for both new validation functions
- docs: Add design document for customer status call validation"
```

**Step 3: Push and create PR**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Block-outbound-call-non-active-customer
git push -u origin NOJIRA-Block-outbound-call-non-active-customer
```
