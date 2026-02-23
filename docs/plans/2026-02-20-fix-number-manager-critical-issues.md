# Fix number-manager Critical/High Issues - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix 6 critical/high severity bugs found in bin-number-manager code review.

**Architecture:** Surgical fixes to existing code — no new packages, no refactoring. Each fix is independent and can be committed separately.

**Tech Stack:** Go, go.uber.org/mock, net/http

---

### Task 1: Fix empty Data slice panic in TelnyxPhoneNumbersGetByNumber

**Files:**
- Modify: `bin-number-manager/pkg/requestexternal/telnyx.go:193`

**Step 1: Add bounds check before accessing Data[0]**

In `TelnyxPhoneNumbersGetByNumber`, replace line 193:

```go
// BEFORE:
return &resParse.Data[0], nil

// AFTER:
if len(resParse.Data) == 0 {
    return nil, fmt.Errorf("no phone number found for number: %s", number)
}
return &resParse.Data[0], nil
```

**Step 2: Run tests**

Run: `cd bin-number-manager && go test ./pkg/requestexternal/...`
Expected: PASS (existing tests, if any, still pass)

**Step 3: Commit**

```
git add bin-number-manager/pkg/requestexternal/telnyx.go
git commit -m "NOJIRA-fix-number-manager-critical-issues

- bin-number-manager: Add bounds check in TelnyxPhoneNumbersGetByNumber to prevent panic on empty Data slice"
```

---

### Task 2: Add purchase rollback on Register failure

**Files:**
- Modify: `bin-number-manager/pkg/numberhandler/number.go` (Create function, lines 45-69)
- Modify: `bin-number-manager/pkg/numberhandler/number_test.go` (add rollback test)

**Step 1: Write the failing test for the rollback path**

Add this test to `number_test.go`:

```go
func Test_Create_rollback_on_register_failure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)

	h := numberHandler{
		reqHandler:          mockReq,
		db:                  mockDB,
		notifyHandler:       mockNotify,
		numberHandlerTelnyx: mockTelnyx,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f8509f38-7ff3-11ec-ac84-e3401d882a9f")
	num := "+821021656521"

	purchasedProvider := &providernumber.ProviderNumber{
		ID:     "7dfbe2b4-1f4e-11ee-8502-23ddd1432a09",
		Status: number.StatusActive,
	}

	// billing check passes
	mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, bmbilling.ReferenceTypeNumber, "", 1).Return(true, nil)

	// purchase succeeds
	mockTelnyx.EXPECT().NumberPurchase(num).Return(purchasedProvider, nil)

	// Register fails (number already exists)
	mockDB.EXPECT().NumberList(ctx, uint64(1), "", gomock.Any()).Return([]*number.Number{{Number: num}}, nil)

	// rollback: release the purchased number
	mockTelnyx.EXPECT().NumberRelease(ctx, gomock.Any()).Return(nil)

	_, err := h.Create(ctx, customerID, num, uuid.Nil, uuid.Nil, "", "")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd bin-number-manager && go test -v -run Test_Create_rollback_on_register_failure ./pkg/numberhandler/...`
Expected: FAIL — `NumberRelease` is never called (unexpected mock call or missing call)

**Step 3: Implement the rollback in Create**

In `pkg/numberhandler/number.go`, modify the `Create` function. Replace lines 45-70 with:

```go
	// use telnyx as a default
	tmp, err := h.numberHandlerTelnyx.NumberPurchase(num)
	if err != nil {
		log.Errorf("Could not create a number from the telnyx. err: %v", err)
		return nil, fmt.Errorf("could not create a number from the telnyx. err: %v", err)
	}

	res, err := h.Register(
		ctx,
		customerID,
		num,
		callFlowID,
		messageFlowID,
		name,
		detail,
		number.TypeNormal,
		number.ProviderNameTelnyx,
		tmp.ID,
		tmp.Status,
		tmp.T38Enabled,
		tmp.EmergencyEnabled,
	)
	if err != nil {
		log.Errorf("Could not create the number record. err: %v", err)

		// rollback: release the purchased number from the provider
		releaseNum := &number.Number{
			ProviderName:        number.ProviderNameTelnyx,
			ProviderReferenceID: tmp.ID,
		}
		if errRelease := h.numberHandlerTelnyx.NumberRelease(ctx, releaseNum); errRelease != nil {
			log.Errorf("Could not rollback purchased number from provider. provider_reference_id: %s, err: %v", tmp.ID, errRelease)
		}

		return nil, errors.Wrap(err, "could not create the number record")
	}
```

**Step 4: Run tests to verify they pass**

Run: `cd bin-number-manager && go test -v ./pkg/numberhandler/...`
Expected: ALL PASS (including existing Test_Create_OrderNumberTelnyx)

**Step 5: Commit**

```
git add bin-number-manager/pkg/numberhandler/number.go bin-number-manager/pkg/numberhandler/number_test.go
git commit -m "NOJIRA-fix-number-manager-critical-issues

- bin-number-manager: Add rollback to release purchased number from provider when Register fails after NumberPurchase"
```

---

### Task 3: Add HTTP client timeout and reuse

**Files:**
- Modify: `bin-number-manager/pkg/requestexternal/telnyx.go` (6 occurrences of `&http.Client{}`)

**Step 1: Add package-level HTTP client with timeout**

Add after the imports in `telnyx.go`:

```go
var telnyxHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}
```

Add `"time"` to the imports.

**Step 2: Replace all 6 occurrences of `&http.Client{}`**

Replace every `client := &http.Client{}` with `client := telnyxHTTPClient` at lines 45, 117, 164, 207, 301, 362.

**Step 3: Run tests**

Run: `cd bin-number-manager && go test ./pkg/requestexternal/...`
Expected: PASS

**Step 4: Commit**

```
git add bin-number-manager/pkg/requestexternal/telnyx.go
git commit -m "NOJIRA-fix-number-manager-critical-issues

- bin-number-manager: Add 30s timeout to Telnyx HTTP client and reuse single client instance"
```

---

### Task 4: Fix listenHandler.Run() error swallowed at startup

**Files:**
- Modify: `bin-number-manager/cmd/number-manager/main.go` (runServiceListen function, lines 150-154)

**Step 1: Return the error from runServiceListen**

Replace lines 149-155:

```go
// BEFORE:
	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameNumberRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil

// AFTER:
	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameNumberRequest), string(commonoutline.QueueNameDelay)); err != nil {
		return fmt.Errorf("could not run the listenhandler: %w", err)
	}

	return nil
```

Add `"fmt"` to the imports.

**Step 2: Build to verify compilation**

Run: `cd bin-number-manager && go build ./cmd/...`
Expected: Build succeeds

**Step 3: Commit**

```
git add bin-number-manager/cmd/number-manager/main.go
git commit -m "NOJIRA-fix-number-manager-critical-issues

- bin-number-manager: Return listenHandler.Run() error to halt startup on failure instead of silently continuing"
```

---

### Task 5: Fix flow-deleted cascade pagination break condition

**Files:**
- Modify: `bin-number-manager/pkg/numberhandler/event.go` (lines 81, 109)

**Step 1: Fix both break conditions**

Replace `if len(numbs) < 100` with `if len(numbs) < 1000` at both locations (lines 81 and 109).

Line 81:
```go
// BEFORE:
		if len(numbs) < 100 {
// AFTER:
		if len(numbs) < 1000 {
```

Line 109:
```go
// BEFORE:
		if len(numbs) < 100 {
// AFTER:
		if len(numbs) < 1000 {
```

**Step 2: Run existing tests**

Run: `cd bin-number-manager && go test -v -run Test_EventFlowDeleted ./pkg/numberhandler/...`
Expected: PASS (existing tests use small sets that hit `len(numbs) <= 0` break first)

**Step 3: Commit**

```
git add bin-number-manager/pkg/numberhandler/event.go
git commit -m "NOJIRA-fix-number-manager-critical-issues

- bin-number-manager: Fix EventFlowDeleted pagination break condition to match page size of 1000"
```

---

### Task 6: Fix nil slice in renew results

**Files:**
- Modify: `bin-number-manager/pkg/numberhandler/renew.go` (lines 25, 62)

**Step 1: Fix both nil slice initializations**

Line 25 in `RenewNumbers`:
```go
// BEFORE:
	var res []*number.Number
// AFTER:
	res := []*number.Number{}
```

Line 62 in `renewNumbersByTMRenew`:
```go
// BEFORE:
	var res []*number.Number
// AFTER:
	res := []*number.Number{}
```

**Step 2: Run existing tests**

Run: `cd bin-number-manager && go test -v -run Test_RenewNumbers ./pkg/numberhandler/...`
Expected: PASS

**Step 3: Commit**

```
git add bin-number-manager/pkg/numberhandler/renew.go
git commit -m "NOJIRA-fix-number-manager-critical-issues

- bin-number-manager: Initialize renew result slices as empty instead of nil to prevent JSON null responses"
```

---

### Task 7: Run full verification workflow

**Step 1: Run complete verification**

```bash
cd bin-number-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All steps pass.

**Step 2: If any issues, fix and re-run**

Fix lint or test issues that arise, then re-run the full verification.

---

## Notes

- **Dropped finding:** `sockHandler.Connect()` has no return value (returns void), so there is no error to check. This finding from the review was incorrect.
- **Commented-out Telnyx tests:** Not included in this plan because writing proper mock-based tests for the Telnyx provider integration is a separate, larger effort that requires designing the test fixtures and mock setup. It can be tracked as a follow-up task.
- Each task is independent and can be committed separately, but they should all be on the same feature branch and combined into one PR.
