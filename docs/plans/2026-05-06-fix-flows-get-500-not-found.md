# Fix GET /v1.0/flows/{id} Returns 500 Instead of 404 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix `GET /v1.0/flows/{id}` to return 404 NOT_FOUND (instead of 500 INTERNAL) when the flow UUID is valid but does not exist.

**Architecture:** The flow-manager backend already returns a typed `*cerrors.VoipbinError(NOT_FOUND)` when a flow UUID is not found. This typed error is reconstructed by the request handler's `parseResponse()` and flows up through `servicehandler.flowGet()` wrapped with `errors.Wrapf`. However, `servicehandler.FlowGet()` at line 153 replaces it with a fresh `errors.Errorf(...)` call — discarding the original error and its chain — so `translateToVoipbinError()` falls through to the INTERNAL default. Fix: swap `errors.Errorf` for `errors.Wrapf(err, ...)`.

**Tech Stack:** Go, `github.com/pkg/errors`, `monorepo/bin-common-handler/models/errors` (cerrors), gomock

---

### Task 1: Add a failing test for the NOT_FOUND propagation

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/flow_test.go`

**Context:**
The existing `Test_FlowGet` only covers success paths. We need an error-path test case that verifies: when `FlowV1FlowGet` returns a `*cerrors.VoipbinError(NOT_FOUND)`, `FlowGet` propagates that exact error (chain intact) so callers can recover it via `errors.As`.

**Step 1: Understand the test structure**

Open `bin-api-manager/pkg/servicehandler/flow_test.go` and locate `Test_FlowGet` (around line 185). Note the pattern: table-driven, `mockReq.EXPECT().FlowV1FlowGet(...).Return(response, nil)`.

**Step 2: Add the error test case**

Add a new standalone test function after `Test_FlowGet`:

```go
func Test_FlowGet_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})
	flowID := uuid.FromStringOrNil("9877fa1f-3bd6-4e90-af61-f713d5cac4d3")

	notFoundErr := cerrors.NotFound(
		commonoutline.ServiceNameFlowManager,
		"FLOW_NOT_FOUND",
		"The flow was not found.",
	)
	mockReq.EXPECT().FlowV1FlowGet(ctx, flowID).Return(nil, notFoundErr)

	_, err := h.FlowGet(ctx, agent, flowID)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("Expected *cerrors.VoipbinError in chain, got: %T: %v", err, err)
	}
	if ve.Status != cerrors.StatusNotFound {
		t.Errorf("Expected StatusNotFound, got: %v", ve.Status)
	}
}
```

**Required imports to add** (if not already present):
```go
stderrors "errors"
cerrors "monorepo/bin-common-handler/models/errors"
commonoutline "monorepo/bin-common-handler/models/outline"
```

**Step 3: Run the test to confirm it fails**

```bash
cd bin-api-manager
go test ./pkg/servicehandler/ -run Test_FlowGet_NotFound -v
```

Expected: **FAIL** — `Expected *cerrors.VoipbinError in chain` (because `errors.Errorf` currently drops the chain).

---

### Task 2: Apply the one-line fix

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/flow.go:152-154`

**Step 1: Make the change**

In `FlowGet()` (line ~152), change:

```go
// BEFORE
if err != nil {
    return nil, errors.Errorf("could not get the flow")
}
```

to:

```go
// AFTER
if err != nil {
    return nil, errors.Wrapf(err, "could not get the flow")
}
```

No other changes needed — `errors.Wrapf` from `github.com/pkg/errors` preserves the error chain so `errors.As` can traverse it.

**Step 2: Run the new test to confirm it passes**

```bash
cd bin-api-manager
go test ./pkg/servicehandler/ -run Test_FlowGet_NotFound -v
```

Expected: **PASS**

**Step 3: Run the full servicehandler test suite**

```bash
cd bin-api-manager
go test ./pkg/servicehandler/... -v
```

Expected: all tests **PASS** (no regressions).

---

### Task 3: Run full verification workflow

**Files:** none (verification only)

```bash
cd bin-api-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

All five steps must pass before committing.

---

### Task 4: Commit

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-flows-500-not-found

git add bin-api-manager/pkg/servicehandler/flow.go \
        bin-api-manager/pkg/servicehandler/flow_test.go

git commit -m "NOJIRA-Fix-flows-500-not-found

- bin-api-manager: Fix FlowGet propagating VoipbinError chain via errors.Wrapf
- bin-api-manager: Add Test_FlowGet_NotFound to cover NOT_FOUND error path"
```

---

### Task 5: Create PR

```bash
git push -u origin NOJIRA-Fix-flows-500-not-found

gh pr create \
  --title "NOJIRA-Fix-flows-500-not-found" \
  --body "Fix GET /v1.0/flows/{id} returning 500 INTERNAL instead of 404 NOT_FOUND when the flow UUID does not exist.

- bin-api-manager: Replace errors.Errorf (discards chain) with errors.Wrapf(err, ...) in FlowGet so the *cerrors.VoipbinError(NOT_FOUND) from flow-manager flows through translateToVoipbinError correctly
- bin-api-manager: Add Test_FlowGet_NotFound regression test covering the NOT_FOUND propagation path

Fixes: voipbin/monorepo-monitoring#114"
```
