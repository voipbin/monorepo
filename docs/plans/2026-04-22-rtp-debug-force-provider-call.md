# Force rtp_debug for Provider Calls Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Force `rtp_debug` recording on for all provider calls regardless of customer settings, and move all `rtp_debug` decisions to call creation time so `status.go` no longer needs to fetch customer info.

**Architecture:** Add `MetadataKeyRTPDebug: true` to the provider call metadata at creation time. In `outgoing_call.go`, embed the customer's `rtp_debug` preference into call metadata at creation (customer is already fetched there). In `status.go`, remove the customer fetch and DB update — just read `res.Metadata[MetadataKeyRTPDebug]` directly to decide whether to start recording.

**Tech Stack:** Go, RabbitMQ RPC, MySQL (call metadata stored as JSON column)

---

### Task 1: Force `rtp_debug` in `providercallhandler`

**Files:**
- Modify: `bin-route-manager/pkg/providercallhandler/providercall.go:73-76`

**Step 1: Add `MetadataKeyRTPDebug: true` to the metadata map**

```go
// bin-route-manager/pkg/providercallhandler/providercall.go
// lines 73-76 — change:
metadata := map[string]any{
    string(cmcall.MetadataKeyRouteProviderIDs):     []string{providerID.String()},
    string(cmcall.MetadataKeySkipSourceValidation): true,
    string(cmcall.MetadataKeyRTPDebug):             true, // force rtp_debug for all provider calls
}
```

**Step 2: Run tests**

```bash
cd bin-route-manager
go test ./pkg/providercallhandler/... -v
```

Expected: all existing tests pass (adding a metadata key does not break existing logic).

**Step 3: Commit**

```bash
git add bin-route-manager/pkg/providercallhandler/providercall.go
git commit -m "NOJIRA-rtp-debug-force-provider-call

- bin-route-manager: force rtp_debug metadata on all provider calls"
```

---

### Task 2: Embed customer `rtp_debug` preference at outgoing call creation time

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/outgoing_call.go:153` (after existing customer fetch)
- Test: `bin-call-manager/pkg/callhandler/outgoing_call_test.go`

**Context:** `CreateCallOutgoing` already fetches the customer at line 149. We add a block immediately after the debug log at line 153 to embed `rtp_debug: true` into the call metadata if the customer has it enabled. This is idempotent — if `providercallhandler` already set it to `true`, setting it again is a no-op.

**Step 1: Write the failing test**

Add a test case to `outgoing_call_test.go` (in the existing table-driven test for `CreateCallOutgoing`) that verifies that when the customer has `Metadata.RTPDebug = true`, the call is created with `rtp_debug: true` in its metadata.

Look for existing test patterns in `outgoing_call_test.go` — find how mock customer responses are set up (`mockReqHandler.EXPECT().CustomerV1CustomerGet(...)`) and follow the same pattern.

```go
{
    name: "customer has rtp_debug enabled — metadata embedded at creation",
    // Set up customer mock to return Metadata.RTPDebug = true
    // Verify the call passed to h.Create() has metadata[MetadataKeyRTPDebug] == true
},
```

**Step 2: Run the test to verify it fails**

```bash
cd bin-call-manager
go test ./pkg/callhandler/... -run TestCreateCallOutgoing -v
```

Expected: FAIL — the metadata block does not exist yet.

**Step 3: Add the implementation block**

In `bin-call-manager/pkg/callhandler/outgoing_call.go`, after line 153 (the `log.WithField("customer", cu).Debugf(...)` line):

```go
// embed rtp_debug in call metadata at creation time so status.go doesn't need to re-fetch the customer
if cu.Metadata.RTPDebug {
    if metadata == nil {
        metadata = map[string]interface{}{}
    }
    metadata[string(call.MetadataKeyRTPDebug)] = true
}
```

**Step 4: Run the test to verify it passes**

```bash
cd bin-call-manager
go test ./pkg/callhandler/... -run TestCreateCallOutgoing -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add bin-call-manager/pkg/callhandler/outgoing_call.go \
        bin-call-manager/pkg/callhandler/outgoing_call_test.go
git commit -m "NOJIRA-rtp-debug-force-provider-call

- bin-call-manager: embed customer rtp_debug preference in call metadata at creation time"
```

---

### Task 3: Simplify `status.go` — remove customer fetch, read call metadata directly

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/status.go:69-88`
- Test: `bin-call-manager/pkg/callhandler/status_test.go`

**Context:** The current block (lines 69–88) fetches the customer, checks `cs.Metadata.RTPDebug`, sets `res.Metadata[MetadataKeyRTPDebug] = true`, calls `h.db.CallUpdate()` to persist it, then calls `rtpDebugStartRecording`. With Task 2 done, the flag is already in `res.Metadata` from creation time — so none of that is needed. The replacement is a single `if` check on the call metadata.

**Step 1: Write the failing test**

In `status_test.go`, add test cases for `updateStatusProgressing` that verify:
1. When `res.Metadata[MetadataKeyRTPDebug] = true` — `rtpDebugStartRecording` is called (no customer fetch mock needed).
2. When `res.Metadata` does not have the key — `rtpDebugStartRecording` is NOT called.

Follow the existing table-driven test patterns in `status_test.go`.

**Step 2: Run the test to verify it fails**

```bash
cd bin-call-manager
go test ./pkg/callhandler/... -run TestUpdateStatusProgressing -v
```

Expected: FAIL — the current code fetches customer and checks customer metadata instead of call metadata.

**Step 3: Replace the block in `status.go`**

Remove lines 69–88:

```go
// DELETE this entire block:
cs, errCS := h.reqHandler.CustomerV1CustomerGet(ctx, res.CustomerID)
if errCS != nil {
    log.Errorf("Could not get customer for RTP debug check. customer_id: %s, err: %v", res.CustomerID, errCS)
} else {
    log.WithField("customer", cs).Debugf("Retrieved customer for RTP debug check. customer_id: %s", cs.ID)
    if cs.Metadata.RTPDebug {
        if res.Metadata == nil {
            res.Metadata = map[string]interface{}{}
        }
        res.Metadata[call.MetadataKeyRTPDebug] = true
        if errMeta := h.db.CallUpdate(ctx, res.ID, map[call.Field]any{
            call.FieldMetadata: res.Metadata,
        }); errMeta != nil {
            log.Errorf("Could not update call metadata for RTP debug. err: %v", errMeta)
        } else {
            h.rtpDebugStartRecording(ctx, res, cn)
        }
    }
}
```

Replace with:

```go
// rtp_debug is set in call metadata at creation time (outgoing_call.go or providercallhandler)
if rtpDebug, _ := res.Metadata[call.MetadataKeyRTPDebug].(bool); rtpDebug {
    h.rtpDebugStartRecording(ctx, res, cn)
}
```

**Step 4: Run the test to verify it passes**

```bash
cd bin-call-manager
go test ./pkg/callhandler/... -run TestUpdateStatusProgressing -v
```

Expected: PASS.

**Step 5: Run full test suite for bin-call-manager**

```bash
cd bin-call-manager
go test ./... -v
```

Expected: all tests pass.

**Step 6: Commit**

```bash
git add bin-call-manager/pkg/callhandler/status.go \
        bin-call-manager/pkg/callhandler/status_test.go
git commit -m "NOJIRA-rtp-debug-force-provider-call

- bin-call-manager: read rtp_debug from call metadata in status.go instead of fetching customer"
```

---

### Task 4: Run full verification for both services

**Step 1: Verify bin-call-manager**

```bash
cd bin-call-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: all steps pass with no errors.

**Step 2: Verify bin-route-manager**

```bash
cd bin-route-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: all steps pass with no errors.

**Step 3: Push branch and create PR**

```bash
# From the worktree:
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
# Expected: no output (no conflicts)

git push -u origin NOJIRA-rtp-debug-force-provider-call

gh pr create \
  --title "NOJIRA-rtp-debug-force-provider-call" \
  --body "Force rtp_debug on for all provider calls and move rtp_debug decisions to call creation time.

- bin-route-manager: force rtp_debug metadata on all provider calls in providercallhandler
- bin-call-manager: embed customer rtp_debug preference in call metadata at outgoing_call.go creation time
- bin-call-manager: simplify status.go — remove customer fetch, read rtp_debug from call metadata directly"
```
