# Fix Signup Email Verification Failure — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix verification email sending during signup by adding a system customer ID and bypassing balance validation for system-generated emails.

**Architecture:** Define `IDSystem` constant in customer-manager models. Add early return in billing-manager's `IsValidBalanceByCustomerID` for `IDSystem`. Replace `uuid.Nil` with `IDSystem` in customer-manager signup and agent-manager password reset email calls.

**Tech Stack:** Go, gomock, RabbitMQ RPC

---

### Task 1: Add IDSystem constant to customer-manager models

**Files:**
- Modify: `bin-customer-manager/models/customer/customer.go:61-70`

**Step 1: Add the IDSystem constant**

In `bin-customer-manager/models/customer/customer.go`, add `IDSystem` to the existing `var` block after `IDEmpty`:

```go
var (
	IDEmpty = uuid.FromStringOrNil("00000000-0000-0000-0000-00000000000") //

	// voipbin internal service's customer id
	IDCallManager = uuid.FromStringOrNil("00000000-0000-0000-0001-00000000001")
	IDAIManager   = uuid.FromStringOrNil("00000000-0000-0000-0001-00000000002")

	// IDBasicRoute is the customer ID for system-wide default routes.
	// Used by route-manager to look up fallback routes when no customer-specific route exists.
	IDBasicRoute = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001")

	// IDSystem is the customer ID used for system-generated operations
	// (e.g., signup verification emails, password reset emails).
	IDSystem = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000002")

	// GuestCustomerID is the guest/demo account customer id.
	GuestCustomerID = uuid.FromStringOrNil("a856c986-4b06-4496-9641-4d0ecbc67df5")
)
```

**Step 2: Verify it compiles**

Run: `cd bin-customer-manager && go build ./...`
Expected: success, no errors

---

### Task 2: Add billing-manager balance validation bypass for IDSystem

**Files:**
- Modify: `bin-billing-manager/pkg/accounthandler/balance.go:16-37`
- Modify: `bin-billing-manager/pkg/accounthandler/balance_test.go`

**Step 1: Write the failing test**

In `bin-billing-manager/pkg/accounthandler/balance_test.go`, add a new test case to the `tests` slice in `Test_IsValidBalanceByCustomerID` (after the "customer not found error" case at line 89):

```go
		{
			name: "system customer ID bypasses balance validation",

			customerID:  cmcustomer.IDSystem,
			billingType: billing.ReferenceTypeEmail,
			count:       1,

			responseCustomer: nil,
			responseAccount:  nil,
			expectRes:        true,
		},
```

Then add handling for this test case in the test loop. Replace the block starting at line 110 (`if tt.name == "customer not found error"`) with:

```go
			if tt.name == "system customer ID bypasses balance validation" {
				res, err := h.IsValidBalanceByCustomerID(ctx, tt.customerID, tt.billingType, tt.country, tt.count)
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				if res != tt.expectRes {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
				}
				return
			}

			if tt.name == "customer not found error" {
```

**Step 2: Run test to verify it fails**

Run: `cd bin-billing-manager && go test -v ./pkg/accounthandler/ -run Test_IsValidBalanceByCustomerID/system_customer`
Expected: FAIL — `IsValidBalanceByCustomerID` tries to look up customer and fails

**Step 3: Add early return for IDSystem**

In `bin-billing-manager/pkg/accounthandler/balance.go`, add the import and early return at the top of `IsValidBalanceByCustomerID` (after line 22, before line 24):

Add import:
```go
	cmcustomer "monorepo/bin-customer-manager/models/customer"
```

Add guard after the log setup (after line 22):
```go
	// System operations bypass balance validation.
	if customerID == cmcustomer.IDSystem {
		return true, nil
	}
```

**Step 4: Run test to verify it passes**

Run: `cd bin-billing-manager && go test -v ./pkg/accounthandler/ -run Test_IsValidBalanceByCustomerID`
Expected: ALL pass

**Step 5: Run full verification for billing-manager**

Run: `cd bin-billing-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: all pass

---

### Task 3: Use IDSystem in customer-manager sendVerificationEmail

**Files:**
- Modify: `bin-customer-manager/pkg/customerhandler/signup.go:429-432`
- Modify: `bin-customer-manager/pkg/customerhandler/signup_test.go`

**Step 1: Update sendVerificationEmail to use IDSystem**

In `bin-customer-manager/pkg/customerhandler/signup.go`, replace `uuid.Nil` with `customer.IDSystem` at line 431:

```go
	if _, err := h.reqHandler.EmailV1EmailSend(
		ctx,
		customer.IDSystem,
		uuid.Nil,
		destinations,
		subject,
		content,
		nil,
	); err != nil {
```

**Step 2: Update test expectations**

In `bin-customer-manager/pkg/customerhandler/signup_test.go`:

Line 110 — change `uuid.Nil, uuid.Nil` to `customer.IDSystem, uuid.Nil`:
```go
			mockReq.EXPECT().EmailV1EmailSend(ctx, customer.IDSystem, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
```

Line 498 — change `uuid.Nil, uuid.Nil` to `customer.IDSystem, uuid.Nil`:
```go
	mockReq.EXPECT().EmailV1EmailSend(ctx, customer.IDSystem, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("email service down"))
```

Lines 627-628 — change `uuid.Nil` to `customer.IDSystem` (only the first one):
```go
	mockReq.EXPECT().EmailV1EmailSend(
		ctx,
		customer.IDSystem,
		uuid.Nil,
```

Note: The test at `complete_signup_test.go:686` (`Test_sendVerificationEmail_error`) uses `gomock.Any()` for all params and does NOT need updating.

**Step 3: Run tests**

Run: `cd bin-customer-manager && go test -v ./pkg/customerhandler/ -run "Test_Signup|Test_sendVerificationEmail"`
Expected: ALL pass

**Step 4: Run full verification for customer-manager**

Run: `cd bin-customer-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: all pass

---

### Task 4: Use IDSystem in agent-manager PasswordForgot

**Files:**
- Modify: `bin-agent-manager/pkg/agenthandler/agent.go:493`
- Modify: `bin-agent-manager/pkg/agenthandler/agent_password_test.go`
- Modify: `bin-agent-manager/pkg/agenthandler/event_test.go:288`

**Step 1: Update PasswordForgot to use IDSystem**

In `bin-agent-manager/pkg/agenthandler/agent.go`, add import:
```go
	cmcustomer "monorepo/bin-customer-manager/models/customer"
```

Line 493 — change `uuid.Nil, uuid.Nil` to `cmcustomer.IDSystem, uuid.Nil`:
```go
	_, err = h.reqHandler.EmailV1EmailSend(ctx, cmcustomer.IDSystem, uuid.Nil, destinations, subject, content, nil)
```

**Step 2: Update test expectations**

In `bin-agent-manager/pkg/agenthandler/agent_password_test.go`, add import:
```go
	cmcustomer "monorepo/bin-customer-manager/models/customer"
```

Line 88 — change `uuid.Nil, uuid.Nil` to `cmcustomer.IDSystem, uuid.Nil`:
```go
			mockReq.EXPECT().EmailV1EmailSend(ctx, cmcustomer.IDSystem, uuid.Nil, []commonaddress.Address{
```

Line 184 — change `uuid.Nil, uuid.Nil` to `cmcustomer.IDSystem, uuid.Nil`:
```go
	mockReq.EXPECT().EmailV1EmailSend(ctx, cmcustomer.IDSystem, uuid.Nil, []commonaddress.Address{
```

In `bin-agent-manager/pkg/agenthandler/event_test.go`, line 288 — change `uuid.Nil, uuid.Nil` to `cmcustomer.IDSystem, uuid.Nil`:
```go
			mockReq.EXPECT().EmailV1EmailSend(ctx, cmcustomer.IDSystem, uuid.Nil, []commonaddress.Address{
```

**Step 3: Run tests**

Run: `cd bin-agent-manager && go test -v ./pkg/agenthandler/ -run "Test_PasswordForgot|Test_EventCustomerCreated"`
Expected: ALL pass

**Step 4: Run full verification for agent-manager**

Run: `cd bin-agent-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: all pass

---

### Task 5: Commit and push

**Step 1: Verify all three services**

Run each in sequence:
```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-fix-signup-email-verification

cd bin-customer-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m && cd ..

cd bin-billing-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m && cd ..

cd bin-agent-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m && cd ..
```

**Step 2: Commit**

```bash
git add bin-customer-manager/models/customer/customer.go \
        bin-billing-manager/pkg/accounthandler/balance.go \
        bin-billing-manager/pkg/accounthandler/balance_test.go \
        bin-customer-manager/pkg/customerhandler/signup.go \
        bin-customer-manager/pkg/customerhandler/signup_test.go \
        bin-agent-manager/pkg/agenthandler/agent.go \
        bin-agent-manager/pkg/agenthandler/agent_password_test.go \
        bin-agent-manager/pkg/agenthandler/event_test.go \
        docs/plans/2026-02-22-fix-signup-email-verification-design.md

git commit -m "NOJIRA-fix-signup-email-verification

Add IDSystem customer ID for system-generated emails and bypass balance
validation in billing-manager for system operations.

- bin-customer-manager: Add IDSystem constant for system customer ID
- bin-billing-manager: Skip balance validation for IDSystem in IsValidBalanceByCustomerID
- bin-customer-manager: Use IDSystem in sendVerificationEmail instead of uuid.Nil
- bin-agent-manager: Use IDSystem in PasswordForgot email send instead of uuid.Nil"
```

Also add any vendor changes if go mod vendor produced diffs.

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-fix-signup-email-verification
```
