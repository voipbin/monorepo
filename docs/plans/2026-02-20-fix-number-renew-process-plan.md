# Fix Number Renew Process Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix critical bugs, add pagination, and improve test coverage in the number-renew process.

**Architecture:** All changes are within `bin-number-manager`. The core renewal logic lives in `pkg/numberhandler/renew.go`. The CronJob schedule lives in `k8s/cronjob.yml`. Tests are in `pkg/numberhandler/renew_test.go`.

**Tech Stack:** Go, gomock, Kubernetes CronJob

**Design doc:** `docs/plans/2026-02-20-fix-number-renew-process-design.md`

---

### Task 1: Fix the delete-then-renew fallthrough bug

**Files:**
- Modify: `bin-number-manager/pkg/numberhandler/renew.go:71-79`

**Context:** When a customer has insufficient balance, the code deletes the number but then falls through to the renewal code (no `continue` or `else`). The deleted number gets its `tm_renew` updated and a `number_renewed` event is published.

**Step 1: Edit the insufficient balance block**

In `renew.go`, replace lines 71-79:

```go
		if !valid {
			log.WithField("number", n).Errorf("The customer has not enough balance for number renew.")
			tmp, err := h.Delete(ctx, n.ID)
			if err != nil {
				log.Errorf("Could not release the number. err: %v", err)
				continue
			}
			log.WithField("number", tmp).Debugf("Deleted number.")
		}
```

With:

```go
		if !valid {
			log.WithField("number", n).Errorf("The customer has not enough balance for number renew.")
			tmp, err := h.Delete(ctx, n.ID)
			if err != nil {
				log.Errorf("Could not release the number. err: %v", err)
			} else {
				log.WithField("number", tmp).Debugf("Deleted number.")
			}
			continue
		}
```

Key change: The `continue` is now outside the `if err` block — so whether the delete succeeds or fails, we always skip renewal for this number.

**Step 2: Run tests to verify nothing breaks**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-fix-number-renew-process/bin-number-manager && go test ./pkg/numberhandler/... -run Test_RenewNumbers -v`

Expected: All existing tests pass (they only test the valid-balance path, so they're unaffected).

---

### Task 2: Fix nil slice initialization in RenewNumbers

**Files:**
- Modify: `bin-number-manager/pkg/numberhandler/renew.go:25`

**Context:** `var res []*number.Number` initializes to nil, which serializes to JSON `null` instead of `[]`. Per project conventions, list functions must use empty slice initialization.

**Step 1: Change nil slice to empty slice**

In `renew.go`, replace line 25:

```go
	var res []*number.Number
```

With:

```go
	res := []*number.Number{}
```

Also remove line 26 (`var err error`) since `res` is now assigned with `:=`, and change the switch to use assignment. Replace lines 25-37:

```go
	var res []*number.Number
	var err error
	switch {
	case days != 0:
		res, err = h.renewNumbersByDays(ctx, days)
	case hours != 0:
		res, err = h.renewNumbersByHours(ctx, hours)
	case tmRenew != "":
		res, err = h.renewNumbersByTMRenew(ctx, tmRenew)
	default:
		log.Errorf("Could not find correct renew time")
		return nil, fmt.Errorf("could not find correct renew time")
	}
```

With:

```go
	var (
		res []*number.Number
		err error
	)
	switch {
	case days != 0:
		res, err = h.renewNumbersByDays(ctx, days)
	case hours != 0:
		res, err = h.renewNumbersByHours(ctx, hours)
	case tmRenew != "":
		res, err = h.renewNumbersByTMRenew(ctx, tmRenew)
	default:
		log.Errorf("Could not find correct renew time")
		return nil, fmt.Errorf("could not find correct renew time")
	}
```

Note: The nil-to-empty-slice fix is actually handled by `renewNumbersByTMRenew` already (line 62 uses `res := []*number.Number{}`). The `RenewNumbers` wrapper just passes through its return, so the slice coming from the sub-functions is already empty-initialized. However, since `RenewNumbers` can also return `nil` in the error/default paths, this is fine — error paths return nil anyway. The key fix is ensuring `renewNumbersByTMRenew` uses `[]*number.Number{}` (it already does on line 62).

Actually, on re-examination, line 25 is fine as-is because the value is always overwritten by the switch arms, and the sub-functions already return `[]*number.Number{}`. No change needed to line 25. Skip this step — the existing code at line 62 already initializes correctly.

**Step 2: Run tests**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-fix-number-renew-process/bin-number-manager && go test ./pkg/numberhandler/... -run Test_RenewNumbers -v`

Expected: All tests pass.

---

### Task 3: Add pagination loop in renewNumbersByTMRenew

**Files:**
- Modify: `bin-number-manager/pkg/numberhandler/renew.go:48-96`

**Context:** `dbListByTMRenew` fetches at most 100 numbers. If >100 need renewal, the rest are silently skipped. Since each renewed number's `tm_renew` gets set to now (moving past the threshold) and deleted numbers are removed, re-querying naturally returns the next batch.

**Step 1: Wrap the query and processing in a loop**

In `renew.go`, replace the entire `renewNumbersByTMRenew` function (lines 48-96):

```go
// renewNumbersByTMRenew renew the numbers by tm_renew
func (h *numberHandler) renewNumbersByTMRenew(ctx context.Context, tmRenew string) ([]*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "renewNumbersByTMRenew",
		"tm_renew": tmRenew,
	})

	res := []*number.Number{}
	for {
		// get list of numbers
		numbers, err := h.dbListByTMRenew(ctx, tmRenew)
		if err != nil {
			log.Errorf("Could not get list of numbers. err: %v", err)
			return nil, errors.Wrap(err, "could not get list of numbers")
		}

		if len(numbers) == 0 {
			break
		}

		// renew the numbers
		for _, n := range numbers {

			valid, err := h.reqHandler.BillingV1AccountIsValidBalanceByCustomerID(ctx, n.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1)
			if err != nil {
				log.Errorf("Could not validate the customer balance. err: %v", err)
				continue
			}

			if !valid {
				log.WithField("number", n).Errorf("The customer has not enough balance for number renew.")
				tmp, err := h.Delete(ctx, n.ID)
				if err != nil {
					log.Errorf("Could not release the number. err: %v", err)
				} else {
					log.WithField("number", tmp).Debugf("Deleted number.")
				}
				continue
			}

			log.WithField("number", n).Debugf("Renewing the number. number_id: %s, number: %s", n.ID, n.Number)

			fields := map[number.Field]any{
				number.FieldTMRenew: h.utilHandler.TimeNow(),
			}
			tmp, err := h.dbUpdate(ctx, n.ID, fields, number.EventTypeNumberRenewed)
			if err != nil {
				log.Errorf("Could not update the number's renew info. err: %v", err)
				continue
			}
			log.WithField("number", n).Debugf("Renewed the number info. number_id: %s, number: %s", n.ID, n.Number)
			res = append(res, tmp)
		}
	}

	return res, nil
}
```

This incorporates both the fallthrough fix (Task 1) and the pagination loop.

**Step 2: Update existing tests for pagination**

With the pagination loop, after processing the batch, the function queries again and gets an empty result to exit the loop. Update `Test_RenewNumbers_renewNumbersByTMRenew` — add a second `NumberGetsByTMRenew` mock call returning empty slice after the existing expectations.

In `renew_test.go`, after line 88 (the end of the `for _, n := range` loop), add:

```go
			// pagination loop: second query returns empty to terminate the loop
			mockDB.EXPECT().NumberGetsByTMRenew(ctx, tt.tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{}, nil)
```

Similarly update `Test_RenewNumbers_renewNumbersByDays` — after line 171, add:

```go
			// pagination loop: second query returns empty to terminate the loop
			mockDB.EXPECT().NumberGetsByTMRenew(ctx, tt.responseCurTime, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{}, nil)
```

And update `Test_RenewNumbers_renewNumbersByHours` — after line 253, add:

```go
			// pagination loop: second query returns empty to terminate the loop
			mockDB.EXPECT().NumberGetsByTMRenew(ctx, tt.responseCurTime, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{}, nil)
```

**Step 3: Run tests**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-fix-number-renew-process/bin-number-manager && go test ./pkg/numberhandler/... -run Test_RenewNumbers -v`

Expected: All 3 existing tests pass.

---

### Task 4: Fix typos and comment mismatch

**Files:**
- Modify: `bin-number-manager/pkg/numberhandler/renew.go:106,117,125`

**Step 1: Fix the three issues**

1. Line 106: Change `"Renwing numbers. tm_renew: %s"` to `"Renewing numbers. tm_renew: %s"`
2. Line 117: Change `// renewNumbersByDays renew the numbers by tm_renew` to `// renewNumbersByHours renew the numbers by hours`
3. Line 125: Change `"Renwing numbers. tm_renew: %s"` to `"Renewing numbers. tm_renew: %s"`

**Step 2: Run tests**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-fix-number-renew-process/bin-number-manager && go test ./pkg/numberhandler/... -run Test_RenewNumbers -v`

Expected: All tests pass.

---

### Task 5: Fix CronJob schedule

**Files:**
- Modify: `bin-number-manager/k8s/cronjob.yml:6`

**Step 1: Correct the cron expression**

Replace:

```yaml
  schedule: "* 1 * * *"
```

With:

```yaml
  schedule: "0 1 * * *"
```

This changes from "every minute during the 1 AM hour" (60 executions) to "once at 1:00 AM daily".

No tests needed for this change.

---

### Task 6: Add test cases for error/edge paths

**Files:**
- Modify: `bin-number-manager/pkg/numberhandler/renew_test.go`

**Context:** The `Delete` method on `numberHandler` calls:
1. `h.Get(ctx, id)` → `h.db.NumberGet(ctx, id)`
2. Provider release based on `ProviderName` (for `ProviderNameNone`, this is a no-op)
3. `h.dbDelete(ctx, id)` → `h.db.NumberDelete(ctx, id)` → `h.db.NumberGet(ctx, id)` → `h.notifyHandler.PublishWebhookEvent(...)`

So for "insufficient balance" tests, the mock expectations for a number with `ProviderName: ""` are:
- `db.NumberGet` (for Get inside Delete)
- `db.NumberDelete` (inside dbDelete)
- `db.NumberGet` (for Get inside dbDelete, after delete)
- `notifyHandler.PublishWebhookEvent` (inside dbDelete)

**Step 1: Add "empty result" test case**

Add to `Test_RenewNumbers_renewNumbersByTMRenew` tests slice:

```go
		{
			name:    "empty result",
			tmRenew: "2021-02-26T18:26:49.000Z",
			responseNumbers: []*number.Number{},
		},
```

The mock setup for this case: `NumberGetsByTMRenew` returns empty slice. No per-number mocks needed. No second pagination query needed (loop exits immediately on empty).

Update the test's mock setup to handle this: the existing mock setup iterates over `responseNumbers` and sets up per-number mocks. For empty slice, this loop does nothing. The only issue is the pagination termination query — for the empty case, the first query already returns empty so no second query happens. For the normal case, we need the second empty query.

To handle this cleanly, restructure the mock setup section of the test. After the main `NumberGetsByTMRenew` mock and the per-number loop, add the second pagination query only when `len(responseNumbers) > 0`:

```go
			mockDB.EXPECT().NumberGetsByTMRenew(ctx, tt.tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return(tt.responseNumbers, nil)
			for _, n := range tt.responseNumbers {
				// ... existing per-number mocks ...
			}
			if len(tt.responseNumbers) > 0 {
				// pagination loop: second query returns empty to terminate the loop
				mockDB.EXPECT().NumberGetsByTMRenew(ctx, tt.tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{}, nil)
			}
```

And update the assertion for the empty case — `res` should be `[]*number.Number{}` (empty, not nil). The existing assertion uses `reflect.DeepEqual(tt.responseNumbers, res)` which will correctly compare `[]*number.Number{}` with `[]*number.Number{}`.

**Step 2: Add "insufficient balance - deletes number" test**

This test needs a separate test function because the mock setup is fundamentally different from the happy path (needs Delete chain mocks instead of renew mocks).

Add a new test function `Test_RenewNumbers_renewNumbersByTMRenew_insufficientBalance`:

```go
func Test_RenewNumbers_renewNumbersByTMRenew_insufficientBalance(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := numberHandler{
		utilHandler:         mockUtil,
		reqHandler:          mockReq,
		db:                  mockDB,
		notifyHandler:       mockNotify,
		numberHandlerTelnyx: mockTelnyx,
	}

	ctx := context.Background()
	tmRenew := "2021-02-26T18:26:49.000Z"

	n := &number.Number{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("aaa00000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("ccc00000-0000-0000-0000-000000000001"),
		},
	}
	deletedN := &number.Number{
		Identity: commonidentity.Identity{
			ID:         n.ID,
			CustomerID: n.CustomerID,
		},
		Status: number.StatusDeleted,
	}

	// First query returns one number
	mockDB.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{n}, nil)

	// Balance check returns invalid
	mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, n.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(false, nil)

	// Delete chain: Get -> provider release (ProviderNameNone = no-op) -> dbDelete -> Get -> PublishWebhookEvent
	mockDB.EXPECT().NumberGet(ctx, n.ID).Return(n, nil)        // Get inside Delete
	mockDB.EXPECT().NumberDelete(ctx, n.ID).Return(nil)         // inside dbDelete
	mockDB.EXPECT().NumberGet(ctx, n.ID).Return(deletedN, nil)  // Get inside dbDelete
	mockNotify.EXPECT().PublishWebhookEvent(ctx, n.CustomerID, number.EventTypeNumberDeleted, deletedN)

	// Pagination: second query returns empty
	mockDB.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{}, nil)

	res, err := h.RenewNumbers(ctx, 0, 0, tmRenew)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// No numbers should be renewed (the number was deleted, not renewed)
	expected := []*number.Number{}
	if !reflect.DeepEqual(expected, res) {
		t.Errorf("Expected empty result, got: %v", res)
	}
}
```

**Step 3: Add "balance check error - skips number" test**

```go
func Test_RenewNumbers_renewNumbersByTMRenew_balanceCheckError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := numberHandler{
		utilHandler:         mockUtil,
		reqHandler:          mockReq,
		db:                  mockDB,
		notifyHandler:       mockNotify,
		numberHandlerTelnyx: mockTelnyx,
	}

	ctx := context.Background()
	tmRenew := "2021-02-26T18:26:49.000Z"

	n := &number.Number{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("bbb00000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("ccc00000-0000-0000-0000-000000000001"),
		},
	}

	mockDB.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{n}, nil)

	// Balance check returns error — number should be skipped (not deleted, not renewed)
	mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, n.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(false, fmt.Errorf("billing service unavailable"))

	// Pagination: second query returns empty
	mockDB.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{}, nil)

	res, err := h.RenewNumbers(ctx, 0, 0, tmRenew)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expected := []*number.Number{}
	if !reflect.DeepEqual(expected, res) {
		t.Errorf("Expected empty result, got: %v", res)
	}
}
```

**Step 4: Add "db update error - skips number" test**

```go
func Test_RenewNumbers_renewNumbersByTMRenew_dbUpdateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := numberHandler{
		utilHandler:         mockUtil,
		reqHandler:          mockReq,
		db:                  mockDB,
		notifyHandler:       mockNotify,
		numberHandlerTelnyx: mockTelnyx,
	}

	ctx := context.Background()
	tmRenew := "2021-02-26T18:26:49.000Z"

	n := &number.Number{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("ddd00000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("ccc00000-0000-0000-0000-000000000001"),
		},
	}

	mockDB.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{n}, nil)

	// Balance is valid
	mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, n.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(true, nil)

	// TimeNow for the update
	mockUtil.EXPECT().TimeNow().Return(&testCurTime)

	// DB update fails
	mockDB.EXPECT().NumberUpdate(ctx, n.ID, gomock.Any()).Return(fmt.Errorf("database error"))

	// Pagination: second query returns empty
	mockDB.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{}, nil)

	res, err := h.RenewNumbers(ctx, 0, 0, tmRenew)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expected := []*number.Number{}
	if !reflect.DeepEqual(expected, res) {
		t.Errorf("Expected empty result, got: %v", res)
	}
}
```

**Step 5: Add "mixed valid and invalid numbers" test**

```go
func Test_RenewNumbers_renewNumbersByTMRenew_mixed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := numberHandler{
		utilHandler:         mockUtil,
		reqHandler:          mockReq,
		db:                  mockDB,
		notifyHandler:       mockNotify,
		numberHandlerTelnyx: mockTelnyx,
	}

	ctx := context.Background()
	tmRenew := "2021-02-26T18:26:49.000Z"

	// 3 numbers: first valid, second insufficient balance, third valid
	n1 := &number.Number{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("eee00000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("ccc00000-0000-0000-0000-000000000001"),
		},
	}
	n2 := &number.Number{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("eee00000-0000-0000-0000-000000000002"),
			CustomerID: uuid.FromStringOrNil("ccc00000-0000-0000-0000-000000000002"),
		},
	}
	n2Deleted := &number.Number{
		Identity: commonidentity.Identity{
			ID:         n2.ID,
			CustomerID: n2.CustomerID,
		},
		Status: number.StatusDeleted,
	}
	n3 := &number.Number{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("eee00000-0000-0000-0000-000000000003"),
			CustomerID: uuid.FromStringOrNil("ccc00000-0000-0000-0000-000000000003"),
		},
	}

	// First query returns 3 numbers
	mockDB.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{n1, n2, n3}, nil)

	// n1: valid balance -> renewed
	mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, n1.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(true, nil)
	mockUtil.EXPECT().TimeNow().Return(&testCurTime)
	mockDB.EXPECT().NumberUpdate(ctx, n1.ID, gomock.Any()).Return(nil)
	mockDB.EXPECT().NumberGet(ctx, n1.ID).Return(n1, nil)
	mockNotify.EXPECT().PublishEvent(ctx, number.EventTypeNumberRenewed, n1)

	// n2: insufficient balance -> deleted
	mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, n2.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(false, nil)
	mockDB.EXPECT().NumberGet(ctx, n2.ID).Return(n2, nil)          // Get inside Delete
	mockDB.EXPECT().NumberDelete(ctx, n2.ID).Return(nil)            // inside dbDelete
	mockDB.EXPECT().NumberGet(ctx, n2.ID).Return(n2Deleted, nil)    // Get inside dbDelete
	mockNotify.EXPECT().PublishWebhookEvent(ctx, n2.CustomerID, number.EventTypeNumberDeleted, n2Deleted)

	// n3: valid balance -> renewed
	mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, n3.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(true, nil)
	mockUtil.EXPECT().TimeNow().Return(&testCurTime)
	mockDB.EXPECT().NumberUpdate(ctx, n3.ID, gomock.Any()).Return(nil)
	mockDB.EXPECT().NumberGet(ctx, n3.ID).Return(n3, nil)
	mockNotify.EXPECT().PublishEvent(ctx, number.EventTypeNumberRenewed, n3)

	// Pagination: second query returns empty
	mockDB.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{}, nil)

	res, err := h.RenewNumbers(ctx, 0, 0, tmRenew)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Only n1 and n3 should be renewed (n2 was deleted)
	expected := []*number.Number{n1, n3}
	if !reflect.DeepEqual(expected, res) {
		t.Errorf("Wrong result.\nexpect: %v\ngot: %v", expected, res)
	}
}
```

**Step 6: Add `fmt` import**

The new test functions use `fmt.Errorf`. Add `"fmt"` to the imports in `renew_test.go` if not already present.

**Step 7: Run all tests**

Run: `cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-fix-number-renew-process/bin-number-manager && go test ./pkg/numberhandler/... -v`

Expected: All tests pass (old and new).

---

### Task 7: Run full verification workflow and commit

**Step 1: Run full verification**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-fix-number-renew-process/bin-number-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All steps pass. If linting flags issues, fix them before committing.

**Step 2: Commit all changes**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-fix-number-renew-process
git add bin-number-manager/pkg/numberhandler/renew.go \
        bin-number-manager/pkg/numberhandler/renew_test.go \
        bin-number-manager/k8s/cronjob.yml
git commit -m "NOJIRA-fix-number-renew-process

- bin-number-manager: Fix delete-then-renew fallthrough bug where deleted numbers were still renewed
- bin-number-manager: Add pagination loop in renewNumbersByTMRenew to handle >100 numbers
- bin-number-manager: Fix CronJob schedule from every-minute-at-1AM to once-at-1AM
- bin-number-manager: Fix typos and comment mismatch in renew functions
- bin-number-manager: Add test cases for insufficient balance, balance error, DB error, empty result, and mixed scenarios"
```

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-fix-number-renew-process
```
