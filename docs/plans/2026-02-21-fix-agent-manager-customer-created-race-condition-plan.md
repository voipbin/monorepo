# Fix Agent Manager Customer Created Race Condition - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix the race condition where agent-manager's `EventCustomerCreated` fails because billing-manager hasn't yet created the billing account.

**Architecture:** Change one function call in `EventCustomerCreated` from `h.Create()` to `h.dbCreate()`, bypassing resource limit validation that depends on the billing account existing. Update three existing tests to remove mock expectations for the skipped checks.

**Tech Stack:** Go, gomock

**Design doc:** `docs/plans/2026-02-21-fix-agent-manager-customer-created-race-condition-design.md`

---

### Task 1: Update EventCustomerCreated to use dbCreate

**Files:**
- Modify: `bin-agent-manager/pkg/agenthandler/event.go:153`

**Step 1: Change `h.Create` to `h.dbCreate`**

In `EventCustomerCreated`, change line 153 from:

```go
	a, err := h.Create(
```

to:

```go
	a, err := h.dbCreate(
```

No other changes needed — `dbCreate` has the same signature as `Create`.

**Step 2: Verify the code compiles**

Run: `cd bin-agent-manager && go build ./...`
Expected: No errors (both functions have identical signatures).

---

### Task 2: Update Test_EventCustomerCreated

**Files:**
- Modify: `bin-agent-manager/pkg/agenthandler/event_test.go:276-278`

**Step 1: Remove mock expectations for skipped checks**

In `Test_EventCustomerCreated`, the test currently sets up mock expectations for `Create`'s pre-checks (lines 277-279). Remove these three lines:

```go
// REMOVE these three lines:
mockReq.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, tt.customer.ID, bmaccount.ResourceTypeAgent).Return(true, nil)
mockUtil.EXPECT().EmailIsValid(tt.customer.Email).Return(true)
mockDB.EXPECT().AgentGetByUsername(ctx, tt.customer.Email).Return(nil, fmt.Errorf(""))
```

The remaining expectations (HashGenerate, UUIDCreate, AgentCreate, AgentGet, PublishWebhookEvent, and the PasswordForgot chain) stay as-is — those are `dbCreate` expectations.

**Step 2: Run the test**

Run: `cd bin-agent-manager && go test -v ./pkg/agenthandler/ -run Test_EventCustomerCreated -count=1`
Expected: PASS

---

### Task 3: Update Test_EventCustomerCreated_Headless

**Files:**
- Modify: `bin-agent-manager/pkg/agenthandler/event_test.go:334-336`

**Step 1: Remove mock expectations for skipped checks**

In `Test_EventCustomerCreated_Headless`, remove these three lines:

```go
// REMOVE these three lines:
mockReq.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, cu.ID, bmaccount.ResourceTypeAgent).Return(true, nil)
mockUtil.EXPECT().EmailIsValid(cu.Email).Return(true)
mockDB.EXPECT().AgentGetByUsername(ctx, cu.Email).Return(nil, fmt.Errorf(""))
```

**Step 2: Run the test**

Run: `cd bin-agent-manager && go test -v ./pkg/agenthandler/ -run Test_EventCustomerCreated_Headless -count=1`
Expected: PASS

---

### Task 4: Update Test_EventCustomerCreated_EmailFails

**Files:**
- Modify: `bin-agent-manager/pkg/agenthandler/event_test.go:385-387`

**Step 1: Remove mock expectations for skipped checks**

In `Test_EventCustomerCreated_EmailFails`, remove these three lines:

```go
// REMOVE these three lines:
mockReq.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customer.ID, bmaccount.ResourceTypeAgent).Return(true, nil)
mockUtil.EXPECT().EmailIsValid(customer.Email).Return(true)
mockDB.EXPECT().AgentGetByUsername(ctx, customer.Email).Return(nil, fmt.Errorf(""))
```

**Step 2: Run the test**

Run: `cd bin-agent-manager && go test -v ./pkg/agenthandler/ -run Test_EventCustomerCreated_EmailFails -count=1`
Expected: PASS

---

### Task 5: Run full verification and commit

**Step 1: Run full verification workflow**

```bash
cd bin-agent-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All steps pass, no lint errors.

**Step 2: Check for unused imports**

After removing the `BillingV1AccountIsValidResourceLimitByCustomerID` mock expectations from all three tests, verify whether the `bmaccount` import in `event_test.go` is still used by any remaining test. If not, remove it.

Also verify whether the `bmaccount` import in `event.go` is still used. Since we removed the call to `Create` (which was the only place in event.go referencing billing models), check if the import needs removal.

**Step 3: Commit**

```bash
git add bin-agent-manager/pkg/agenthandler/event.go bin-agent-manager/pkg/agenthandler/event_test.go
git commit -m "NOJIRA-fix-agent-manager-customer-created-race-condition

- bin-agent-manager: Use dbCreate instead of Create in EventCustomerCreated to skip
  resource limit validation that races with billing-manager account creation
- bin-agent-manager: Update event tests to remove billing validation mock expectations"
```
