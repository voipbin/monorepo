# Initial Token Topup for New Customers - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Give new customers their Free tier monthly token allowance (1,000 tokens) at signup so they can immediately use token-based services.

**Architecture:** After billing account creation in `EventCUCustomerCreated()`, set plan type to Free and call the existing `AccountTopUpTokens()` DB method. Both steps are non-fatal — if they fail, the account is still created (same as current behavior).

**Tech Stack:** Go, gomock for testing, existing billing-manager infrastructure.

---

### Task 1: Update the existing test to expect plan type + topup calls

**Files:**
- Modify: `bin-billing-manager/pkg/accounthandler/event_test.go:109-167` (the `Test_EventCUCustomerCreated` function)

**Step 1: Update the test to expect the new mock calls**

In `Test_EventCUCustomerCreated`, after the existing mock expectations (line 160), add expectations for the two new calls that `EventCUCustomerCreated` will make after linking the account to the customer:

```go
// existing expectations stay the same, then add:
mockDB.EXPECT().AccountUpdate(ctx, tt.responseAccount.ID, map[account.Field]any{
    account.FieldPlanType: account.PlanTypeFree,
}).Return(nil)
mockDB.EXPECT().AccountGet(ctx, tt.responseAccount.ID).Return(tt.responseAccount, nil)
mockDB.EXPECT().AccountTopUpTokens(ctx, tt.responseAccount.ID, tt.customer.ID, int64(1000), string(account.PlanTypeFree)).Return(nil)
```

Note: `dbUpdatePlanType` internally calls `AccountUpdate` + `AccountGet`, so we need both mocks.

**Step 2: Run test to verify it fails**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-initial-token-topup-for-new-customers/bin-billing-manager && go test -v ./pkg/accounthandler/ -run Test_EventCUCustomerCreated -count=1`

Expected: FAIL — the new mock expectations are not satisfied because `EventCUCustomerCreated` doesn't make those calls yet.

---

### Task 2: Implement the plan type + topup in EventCUCustomerCreated

**Files:**
- Modify: `bin-billing-manager/pkg/accounthandler/event.go:46-75` (the `EventCUCustomerCreated` function)

**Step 1: Add plan type + topup code after the customer link**

In `EventCUCustomerCreated()`, after the `CustomerV1CustomerUpdateBillingAccountID` call succeeds (after line 72), add:

```go
	// set default plan type for new account
	if _, errPlan := h.dbUpdatePlanType(ctx, b.ID, account.PlanTypeFree); errPlan != nil {
		log.Errorf("Could not set default plan type. err: %v", errPlan)
		// non-fatal: account is created, customer can still use the platform
	}

	// initial token topup for new customer
	tokenAmount, ok := account.PlanTokenMap[account.PlanTypeFree]
	if ok && tokenAmount > 0 {
		if errTopup := h.db.AccountTopUpTokens(ctx, b.ID, cu.ID, tokenAmount, string(account.PlanTypeFree)); errTopup != nil {
			log.Errorf("Could not perform initial token topup. err: %v", errTopup)
			// non-fatal: account is created, tokens can be topped up later
		}
	}
```

**Step 2: Run test to verify it passes**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-initial-token-topup-for-new-customers/bin-billing-manager && go test -v ./pkg/accounthandler/ -run Test_EventCUCustomerCreated -count=1`

Expected: PASS

---

### Task 3: Update error tests for the new code paths

**Files:**
- Modify: `bin-billing-manager/pkg/accounthandler/event_test.go:169-252` (the `Test_EventCUCustomerCreated_error` function)

**Step 1: Update the "update customer error" test case**

The "update customer error" test currently expects `EventCUCustomerCreated` to return an error when `CustomerV1CustomerUpdateBillingAccountID` fails. Since the new plan type + topup code runs *after* the customer link, this test should still return an error without hitting the new code. No changes needed for this case.

**Step 2: Update mock expectations for both error test cases if needed**

Verify that the "create account error" and "update customer error" test cases still pass. The new code only runs after the customer link succeeds, so these error paths should short-circuit before reaching the new code.

**Step 3: Add a test for topup failure (non-fatal)**

Add a new test case to verify that if `AccountTopUpTokens` fails, `EventCUCustomerCreated` still returns `nil` (non-fatal):

```go
func Test_EventCUCustomerCreated_topup_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := accountHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("a1b2c3d4-e5f6-11ef-a1b2-c3d4e5f6a7b8")
	accountID := uuid.FromStringOrNil("b2c3d4e5-f6a7-11ef-b2c3-d4e5f6a7b8c9")

	customer := &cucustomer.Customer{
		ID: customerID,
	}
	responseAccount := &account.Account{
		Identity: commonidentity.Identity{
			ID: accountID,
		},
	}

	// account creation + link succeed
	mockUtil.EXPECT().UUIDCreate().Return(accountID)
	mockDB.EXPECT().AccountCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().AccountGet(ctx, accountID).Return(responseAccount, nil)
	mockNotify.EXPECT().PublishEvent(ctx, account.EventTypeAccountCreated, responseAccount)
	mockReq.EXPECT().CustomerV1CustomerUpdateBillingAccountID(ctx, customerID, accountID).Return(customer, nil)

	// plan type update succeeds
	mockDB.EXPECT().AccountUpdate(ctx, accountID, map[account.Field]any{
		account.FieldPlanType: account.PlanTypeFree,
	}).Return(nil)
	mockDB.EXPECT().AccountGet(ctx, accountID).Return(responseAccount, nil)

	// topup fails — should NOT cause EventCUCustomerCreated to return error
	mockDB.EXPECT().AccountTopUpTokens(ctx, accountID, customerID, int64(1000), string(account.PlanTypeFree)).Return(fmt.Errorf("topup failed"))

	err := h.EventCUCustomerCreated(ctx, customer)
	if err != nil {
		t.Errorf("Expected nil error (topup failure is non-fatal), got: %v", err)
	}
}
```

**Step 4: Run all event tests to verify they pass**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-initial-token-topup-for-new-customers/bin-billing-manager && go test -v ./pkg/accounthandler/ -run Test_EventCUCustomerCreated -count=1`

Expected: PASS (all 3 test functions)

---

### Task 4: Run full verification and commit

**Step 1: Run the full verification workflow**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-initial-token-topup-for-new-customers/bin-billing-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All steps pass.

**Step 2: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-initial-token-topup-for-new-customers
git add bin-billing-manager/pkg/accounthandler/event.go bin-billing-manager/pkg/accounthandler/event_test.go docs/plans/
git commit -m "NOJIRA-add-initial-token-topup-for-new-customers

- bin-billing-manager: Set PlanTypeFree on new billing accounts at creation
- bin-billing-manager: Call AccountTopUpTokens to give initial 1000 tokens to new customers
- bin-billing-manager: Add test for non-fatal topup failure path
- docs: Add design and implementation plan"
```

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-add-initial-token-topup-for-new-customers
```
