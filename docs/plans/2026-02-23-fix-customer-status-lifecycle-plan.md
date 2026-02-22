# Fix Customer Status Lifecycle — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix customer unregistration by implementing proper status lifecycle (initial → active → frozen → deleted).

**Architecture:** Add `StatusInitial` and `StatusExpired` to the customer model. Set status during creation, transition to active during email verification, and switch cleanup from hard delete to soft delete.

**Tech Stack:** Go, MySQL (Alembic migrations), gomock tests

**Design doc:** `docs/plans/2026-02-23-fix-customer-status-lifecycle-design.md`

**Worktree:** `~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-customer-status-lifecycle/`

---

### Task 1: Add new status constants

**Files:**
- Modify: `bin-customer-manager/models/customer/customer.go:12-16`

**Step 1: Add StatusInitial and StatusExpired constants**

In `bin-customer-manager/models/customer/customer.go`, change the status constants block from:

```go
const (
	StatusActive  Status = "active"
	StatusFrozen  Status = "frozen"
	StatusDeleted Status = "deleted"
)
```

to:

```go
const (
	StatusInitial Status = "initial"
	StatusActive  Status = "active"
	StatusFrozen  Status = "frozen"
	StatusDeleted Status = "deleted"
	StatusExpired Status = "expired"
)
```

**Step 2: Run tests to confirm no breakage**

Run: `cd bin-customer-manager && go test ./models/customer/...`
Expected: PASS (adding constants doesn't break anything)

**Step 3: Commit**

```bash
git add bin-customer-manager/models/customer/customer.go
git commit -m "NOJIRA-fix-customer-status-lifecycle

- bin-customer-manager: Add StatusInitial and StatusExpired status constants"
```

---

### Task 2: Set status during admin creation

**Files:**
- Modify: `bin-customer-manager/pkg/customerhandler/db.go:71-85`
- Modify: `bin-customer-manager/pkg/customerhandler/db_test.go` (expectedCustomer)

**Step 1: Update the Create test to expect StatusActive**

In `bin-customer-manager/pkg/customerhandler/db_test.go`, find the `expectedCustomer` struct in the Create test (around line 156-168) and add `Status: customer.StatusActive`:

```go
expectedCustomer: &customer.Customer{
	ID:               uuid.FromStringOrNil("4b9ff112-02ec-11ee-b037-5b5c308ec044"),
	Name:             "test1",
	Detail:           "detail1",
	Email:            "test@voipbin.net",
	PhoneNumber:      "+821100000001",
	Address:          "somewhere",
	WebhookMethod:    customer.WebhookMethodPost,
	WebhookURI:       "test.com",
	BillingAccountID: uuid.Nil,
	EmailVerified:    true,
	Status:           customer.StatusActive,
},
```

**Step 2: Run test to verify it fails**

Run: `cd bin-customer-manager && go test -v ./pkg/customerhandler/ -run Test_Create`
Expected: FAIL (Create doesn't set Status yet)

**Step 3: Add Status to the Create function**

In `bin-customer-manager/pkg/customerhandler/db.go`, in the `Create()` function, add `Status: customer.StatusActive` to the customer struct literal (around line 71-85). Change:

```go
	u := &customer.Customer{
		ID: id,

		Name:   name,
		Detail: detail,

		Email:       email,
		PhoneNumber: phoneNumber,
		Address:     address,

		WebhookMethod: webhookMethod,
		WebhookURI:    webhookURI,

		EmailVerified: true,
	}
```

to:

```go
	u := &customer.Customer{
		ID: id,

		Name:   name,
		Detail: detail,

		Email:       email,
		PhoneNumber: phoneNumber,
		Address:     address,

		WebhookMethod: webhookMethod,
		WebhookURI:    webhookURI,

		EmailVerified: true,
		Status:        customer.StatusActive,
	}
```

**Step 4: Run test to verify it passes**

Run: `cd bin-customer-manager && go test -v ./pkg/customerhandler/ -run Test_Create`
Expected: PASS

**Step 5: Commit**

```bash
git add bin-customer-manager/pkg/customerhandler/db.go bin-customer-manager/pkg/customerhandler/db_test.go
git commit -m "NOJIRA-fix-customer-status-lifecycle

- bin-customer-manager: Set status=active during admin customer creation"
```

---

### Task 3: Set status during signup

**Files:**
- Modify: `bin-customer-manager/pkg/customerhandler/signup.go:74-91`
- Modify: `bin-customer-manager/pkg/customerhandler/signup_test.go` (expectedCustomer + DoAndReturn)

**Step 1: Update the Signup test to verify StatusInitial**

In `bin-customer-manager/pkg/customerhandler/signup_test.go`, in the `DoAndReturn` callback (around line 96-103), add a check for Status:

```go
mockDB.EXPECT().CustomerCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, c *customer.Customer) error {
	if c.TermsAgreedIP != "192.168.1.1" {
		t.Errorf("Expected TermsAgreedIP=192.168.1.1, got: %s", c.TermsAgreedIP)
	}
	if c.TermsAgreedVersion == "" {
		t.Errorf("Expected TermsAgreedVersion to be set, got empty")
	}
	if c.Status != customer.StatusInitial {
		t.Errorf("Expected Status=initial, got: %s", c.Status)
	}
	return nil
})
```

**Step 2: Run test to verify it fails**

Run: `cd bin-customer-manager && go test -v ./pkg/customerhandler/ -run Test_Signup`
Expected: FAIL (Signup doesn't set Status yet)

**Step 3: Add Status to the Signup function**

In `bin-customer-manager/pkg/customerhandler/signup.go`, in the `Signup()` function, add `Status: customer.StatusInitial` to the customer struct literal (around line 74-91). Change:

```go
	u := &customer.Customer{
		ID: id,

		Name:   name,
		Detail: detail,

		Email:       email,
		PhoneNumber: phoneNumber,
		Address:     address,

		WebhookMethod: webhookMethod,
		WebhookURI:    webhookURI,

		EmailVerified: false,

		TermsAgreedVersion: time.Now().UTC().Format(time.RFC3339),
		TermsAgreedIP:      clientIP,
	}
```

to:

```go
	u := &customer.Customer{
		ID: id,

		Name:   name,
		Detail: detail,

		Email:       email,
		PhoneNumber: phoneNumber,
		Address:     address,

		WebhookMethod: webhookMethod,
		WebhookURI:    webhookURI,

		EmailVerified: false,
		Status:        customer.StatusInitial,

		TermsAgreedVersion: time.Now().UTC().Format(time.RFC3339),
		TermsAgreedIP:      clientIP,
	}
```

**Step 4: Run test to verify it passes**

Run: `cd bin-customer-manager && go test -v ./pkg/customerhandler/ -run Test_Signup`
Expected: PASS

**Step 5: Commit**

```bash
git add bin-customer-manager/pkg/customerhandler/signup.go bin-customer-manager/pkg/customerhandler/signup_test.go
git commit -m "NOJIRA-fix-customer-status-lifecycle

- bin-customer-manager: Set status=initial during signup customer creation"
```

---

### Task 4: Transition to active during email verification

**Files:**
- Modify: `bin-customer-manager/pkg/customerhandler/signup.go:215-217` (EmailVerify)
- Modify: `bin-customer-manager/pkg/customerhandler/signup.go:342-344` (CompleteSignup)

**Step 1: Update EmailVerify to set status=active**

In `bin-customer-manager/pkg/customerhandler/signup.go`, in the `EmailVerify()` function, change the update fields map (around line 215-217) from:

```go
	// mark as verified
	fields := map[customer.Field]any{
		customer.FieldEmailVerified: true,
	}
```

to:

```go
	// mark as verified and activate
	fields := map[customer.Field]any{
		customer.FieldEmailVerified: true,
		customer.FieldStatus:        string(customer.StatusActive),
	}
```

**Step 2: Update CompleteSignup to set status=active**

In the same file, in the `CompleteSignup()` function, change the update fields map (around line 342-344) from:

```go
	// Mark customer as verified
	fields := map[customer.Field]any{
		customer.FieldEmailVerified: true,
	}
```

to:

```go
	// Mark customer as verified and activate
	fields := map[customer.Field]any{
		customer.FieldEmailVerified: true,
		customer.FieldStatus:        string(customer.StatusActive),
	}
```

**Step 3: Run all signup tests**

Run: `cd bin-customer-manager && go test -v ./pkg/customerhandler/ -run "Test_Signup|Test_EmailVerify|Test_CompleteSignup"`
Expected: PASS (mock expectations use `gomock.Any()` for update fields)

**Step 4: Commit**

```bash
git add bin-customer-manager/pkg/customerhandler/signup.go
git commit -m "NOJIRA-fix-customer-status-lifecycle

- bin-customer-manager: Set status=active during email verification (both EmailVerify and CompleteSignup)"
```

---

### Task 5: Change cleanup from hard delete to soft delete

**Files:**
- Modify: `bin-customer-manager/pkg/customerhandler/cleanup.go:36-64`
- Modify: `bin-customer-manager/pkg/customerhandler/cleanup_test.go`

**Step 1: Write test for cleanup soft delete**

Replace `bin-customer-manager/pkg/customerhandler/cleanup_test.go` with:

```go
package customerhandler

import (
	"context"
	"testing"
	"time"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/dbhandler"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func TestCleanupConstants(t *testing.T) {
	if cleanupInterval != 15*time.Minute {
		t.Errorf("cleanupInterval = %v, expected %v", cleanupInterval, 15*time.Minute)
	}
	if unverifiedMaxAge != time.Hour {
		t.Errorf("unverifiedMaxAge = %v, expected %v", unverifiedMaxAge, time.Hour)
	}
}

func Test_cleanupUnverified(t *testing.T) {
	tests := []struct {
		name string

		responseCustomers []*customer.Customer
		expectUpdate      bool
	}{
		{
			name: "no unverified customers",

			responseCustomers: []*customer.Customer{},
			expectUpdate:      false,
		},
		{
			name: "one unverified customer - soft deleted",

			responseCustomers: []*customer.Customer{
				{
					ID:            uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
					Email:         "expired@test.com",
					EmailVerified: false,
					Status:        customer.StatusInitial,
				},
			},
			expectUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &customerHandler{
				db:          mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockDB.EXPECT().CustomerList(ctx, uint64(100), gomock.Any(), gomock.Any()).Return(tt.responseCustomers, nil)

			if tt.expectUpdate {
				now := time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)
				mockUtil.EXPECT().TimeNow().Return(&now)

				mockDB.EXPECT().CustomerUpdate(ctx, tt.responseCustomers[0].ID, gomock.Any()).DoAndReturn(
					func(_ context.Context, _ uuid.UUID, fields map[customer.Field]any) error {
						status, ok := fields[customer.FieldStatus]
						if !ok || status != string(customer.StatusExpired) {
							t.Errorf("Expected status=expired, got: %v", status)
						}
						tmDelete, ok := fields[customer.FieldTMDelete]
						if !ok || tmDelete == nil {
							t.Errorf("Expected tm_delete to be set")
						}
						return nil
					},
				)
			}

			h.cleanupUnverified(ctx)
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd bin-customer-manager && go test -v ./pkg/customerhandler/ -run Test_cleanupUnverified`
Expected: FAIL (cleanup still uses CustomerHardDelete)

**Step 3: Update cleanup.go to use soft delete**

Replace the `cleanupUnverified` function in `bin-customer-manager/pkg/customerhandler/cleanup.go` with:

```go
func (h *customerHandler) cleanupUnverified(ctx context.Context) {
	log := logrus.WithField("func", "cleanupUnverified")
	log.Debug("Running unverified customer cleanup.")

	cutoff := time.Now().Add(-unverifiedMaxAge)
	cutoffStr := cutoff.Format("2006-01-02 15:04:05.000000")

	filters := map[customer.Field]any{
		customer.FieldEmailVerified: false,
		customer.FieldDeleted:       false,
	}

	customers, err := h.db.CustomerList(ctx, 100, cutoffStr, filters)
	if err != nil {
		log.Errorf("Could not list unverified customers. err: %v", err)
		return
	}

	for _, c := range customers {
		log.Infof("Expiring unverified customer. customer_id: %s, email: %s", c.ID, c.Email)

		now := h.utilHandler.TimeNow()
		fields := map[customer.Field]any{
			customer.FieldStatus:   string(customer.StatusExpired),
			customer.FieldTMDelete: now,
		}
		if err := h.db.CustomerUpdate(ctx, c.ID, fields); err != nil {
			log.Errorf("Could not expire customer. customer_id: %s, err: %v", c.ID, err)
		}
	}

	if len(customers) > 0 {
		log.Infof("Cleanup completed. expired: %d", len(customers))
	}
}
```

**Step 4: Run test to verify it passes**

Run: `cd bin-customer-manager && go test -v ./pkg/customerhandler/ -run Test_cleanupUnverified`
Expected: PASS

**Step 5: Commit**

```bash
git add bin-customer-manager/pkg/customerhandler/cleanup.go bin-customer-manager/pkg/customerhandler/cleanup_test.go
git commit -m "NOJIRA-fix-customer-status-lifecycle

- bin-customer-manager: Change cleanup from hard delete to soft delete with status=expired"
```

---

### Task 6: DB migration for existing data

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/a1b2c3d4e5f7_customer_fix_empty_status.py`

**Step 1: Create the Alembic migration**

Create `bin-dbscheme-manager/bin-manager/main/versions/a1b2c3d4e5f7_customer_fix_empty_status.py`:

```python
"""customer_customers fix empty status values

Revision ID: a1b2c3d4e5f7
Revises: 455debd049b2
Create Date: 2026-02-23 00:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'a1b2c3d4e5f7'
down_revision = '455debd049b2'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""update customer_customers set status = 'active' where status = '' and email_verified = 1;""")
    op.execute("""update customer_customers set status = 'initial' where status = '' and email_verified = 0;""")


def downgrade():
    op.execute("""update customer_customers set status = '' where status = 'initial';""")
    op.execute("""update customer_customers set status = '' where status = 'active' and tm_delete is null;""")
```

**Step 2: Commit**

```bash
git add bin-dbscheme-manager/bin-manager/main/versions/a1b2c3d4e5f7_customer_fix_empty_status.py
git commit -m "NOJIRA-fix-customer-status-lifecycle

- bin-dbscheme-manager: Add migration to fix empty customer status values"
```

---

### Task 7: Run full verification and final commit

**Step 1: Run full verification for bin-customer-manager**

```bash
cd bin-customer-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All pass

**Step 2: Fix any issues found by lint or tests**

If any tests fail or lint issues arise, fix them before proceeding.

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-fix-customer-status-lifecycle
```

Then create PR with:
- Title: `NOJIRA-fix-customer-status-lifecycle`
- Body: Summary of changes from the design doc
