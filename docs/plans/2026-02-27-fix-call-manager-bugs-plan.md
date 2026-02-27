# Fix call-manager Bugs Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix 5 confirmed bugs in bin-call-manager: wrong error variable in 6 methods, wrong log variable in goroutine, copy-paste field bug in webhook, JSON tag typo, and swallowed error in mute.

**Architecture:** All fixes are localized to bin-call-manager. No cross-service changes needed. Each fix is a 1-3 line code change plus test updates.

**Tech Stack:** Go, gomock, pkg/errors

**Worktree:** `~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-call-manager-bugs`

---

### Task 1: Fix errors.Wrap wrapping nil in hold.go, moh.go, silence.go

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/hold.go:27,49`
- Modify: `bin-call-manager/pkg/callhandler/moh.go:27,49`
- Modify: `bin-call-manager/pkg/callhandler/silence.go:27,49`
- Modify: `bin-call-manager/pkg/callhandler/hold_test.go`
- Modify: `bin-call-manager/pkg/callhandler/moh_test.go`
- Modify: `bin-call-manager/pkg/callhandler/silence_test.go`

**Context:** In each of these 6 methods (HoldOn, HoldOff, MOHOn, MOHOff, SilenceOn, SilenceOff), when the channelHandler operation fails, the code returns `errors.Wrap(err, ...)` where `err` is from the prior successful `h.Get()` call (so it's nil). `errors.Wrap(nil, ...)` returns nil, making the function silently succeed. Also `mute.go:35,67` has the same bug — fix those too.

**Step 1: Add failing test cases for channelHandler errors**

Add an error test case to each existing test. Follow the same table-driven pattern. In each test, mock `h.Get` to succeed and the channelHandler operation to return an error. Assert that the method returns a non-nil error.

For `hold_test.go`, add to `Test_HoldOn`:
```go
{
    name: "channel hold error",
    id:   uuid.FromStringOrNil("97864554-cef3-11ed-9ba5-a7e641ec5c06"),
    responseCall: &call.Call{
        Identity: commonidentity.Identity{
            ID: uuid.FromStringOrNil("97864554-cef3-11ed-9ba5-a7e641ec5c06"),
        },
        ChannelID: "9a4086ec-cef3-11ed-b377-ef35b455442f",
    },
},
```

And update the test loop to conditionally mock the channel error:
```go
// For the error test case, mock channelHandler to return error
mockChannel.EXPECT().HoldOn(ctx, tt.responseCall.ChannelID).Return(fmt.Errorf("channel error"))
if err := h.HoldOn(ctx, tt.id); err == nil {
    t.Errorf("Expected error, got nil")
}
```

Apply the same pattern for `Test_HoldOff`, `Test_MOHOn`, `Test_MOHOff`, `Test_SilenceOn`, `Test_SilenceOff`.

**Step 2: Run tests to verify they fail**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-call-manager-bugs/bin-call-manager && go test ./pkg/callhandler/ -run "Test_HoldOn|Test_HoldOff|Test_MOHOn|Test_MOHOff|Test_SilenceOn|Test_SilenceOff" -v`
Expected: FAIL — the error test cases return nil instead of an error.

**Step 3: Fix the source code**

In `hold.go:27`: change `errors.Wrap(err,` to `errors.Wrap(errHold,`
In `hold.go:49`: change `errors.Wrap(err,` to `errors.Wrap(errHold,`
In `moh.go:27`: change `errors.Wrap(err,` to `errors.Wrap(errHold,`
In `moh.go:49`: change `errors.Wrap(err,` to `errors.Wrap(errHold,`
In `silence.go:27`: change `errors.Wrap(err,` to `errors.Wrap(errHold,`
In `silence.go:49`: change `errors.Wrap(err,` to `errors.Wrap(errHold,`
In `mute.go:35`: change `errors.Wrap(err,` to `errors.Wrap(errHold,`
In `mute.go:67`: change `errors.Wrap(err,` to `errors.Wrap(errHold,`

**Step 4: Run tests to verify they pass**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-call-manager-bugs/bin-call-manager && go test ./pkg/callhandler/ -run "Test_HoldOn|Test_HoldOff|Test_MOHOn|Test_MOHOff|Test_SilenceOn|Test_SilenceOff|Test_MuteOn|Test_MuteOff" -v`
Expected: PASS

---

### Task 2: Fix goroutine logging wrong error variable in hangup.go

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/hangup.go:71`

**Context:** At line 71 in the goroutine, `log.Errorf("... err: %v", err)` captures the outer scope's `err` variable instead of the goroutine's own `errReq`. This logs the wrong error.

**Step 1: Fix the source code**

In `hangup.go:71`: change `err)` to `errReq)`.

The line should read:
```go
log.Errorf("Could not hangup the groupcall. err: %v", errReq)
```

**Step 2: Verify existing tests still pass**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-call-manager-bugs/bin-call-manager && go test ./pkg/callhandler/ -run "Test_Hangup" -v`
Expected: PASS (this is a log-only change, existing tests verify behavior not log content).

---

### Task 3: Fix copy-paste bug in groupcall ConvertWebhookMessage

**Files:**
- Modify: `bin-call-manager/models/groupcall/webhook.go:67`
- Modify: `bin-call-manager/models/groupcall/groupcall_test.go`

**Context:** Line 67 assigns `AnswerGroupcallID: h.AnswerCallID` instead of `h.AnswerGroupcallID`. The existing test sets up `AnswerCallID` but never sets `AnswerGroupcallID`, so it doesn't catch this bug.

**Step 1: Add test assertion for AnswerGroupcallID**

In `groupcall_test.go`, in the `TestConvertWebhookMessage` function:

Add `answerGroupcallID` variable alongside the existing ones:
```go
answerGroupcallID := uuid.Must(uuid.NewV4())
```

Set it on the Groupcall struct:
```go
AnswerGroupcallID: answerGroupcallID,
```

Add assertion after the `AnswerCallID` check (after line 189):
```go
if webhook.AnswerGroupcallID != answerGroupcallID {
    t.Errorf("ConvertWebhookMessage AnswerGroupcallID = %v, expected %v", webhook.AnswerGroupcallID, answerGroupcallID)
}
```

**Step 2: Run test to verify it fails**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-call-manager-bugs/bin-call-manager && go test ./models/groupcall/ -run "TestConvertWebhookMessage" -v`
Expected: FAIL — AnswerGroupcallID will contain AnswerCallID's value.

**Step 3: Fix the source code**

In `webhook.go:67`: change `h.AnswerCallID` to `h.AnswerGroupcallID`:
```go
AnswerGroupcallID: h.AnswerGroupcallID,
```

**Step 4: Run test to verify it passes**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-call-manager-bugs/bin-call-manager && go test ./models/groupcall/ -run "TestConvertWebhookMessage" -v`
Expected: PASS

---

### Task 4: Fix JSON tag typo in externalmedia model

**Files:**
- Modify: `bin-call-manager/models/externalmedia/main.go:16`
- Modify: `bin-call-manager/pkg/listenhandler/v1_external_medias_test.go` (5 occurrences)

**Context:** The JSON tag `json:"reference_typee"` has a double-e. This is a breaking API change — any existing client using the misspelled key will need to update. The typo also exists in 5 test fixture JSON blobs.

**Step 1: Fix the model**

In `models/externalmedia/main.go:16`: change `json:"reference_typee"` to `json:"reference_type"`.

**Step 2: Fix all test fixtures**

In `pkg/listenhandler/v1_external_medias_test.go`: replace all occurrences of `reference_typee` with `reference_type` (5 occurrences on lines 67, 143, 168, 230, 290).

**Step 3: Run tests to verify they pass**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-call-manager-bugs/bin-call-manager && go test ./pkg/listenhandler/ -run "External" -v && go test ./models/externalmedia/ -v`
Expected: PASS

---

### Task 5: Fix swallowed UpdateMuteDirection error in mute.go

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/mute.go:38-41,85-88`

**Context:** In `MuteOn` (line 38-41) and `MuteOff` (line 85-88), when `UpdateMuteDirection` fails, the error is logged but `nil` is returned. The DB state diverges from Asterisk state.

**Step 1: Fix MuteOn**

Change lines 38-41 from:
```go
_, err = h.UpdateMuteDirection(ctx, id, direction)
if err != nil {
    log.Errorf("Could not update the call mute direction. err: %v", err)
}

return nil
```
To:
```go
_, err = h.UpdateMuteDirection(ctx, id, direction)
if err != nil {
    log.Errorf("Could not update the call mute direction. err: %v", err)
    return errors.Wrap(err, "could not update the call mute direction")
}

return nil
```

**Step 2: Fix MuteOff**

Change lines 85-88 from:
```go
_, err = h.UpdateMuteDirection(ctx, id, newDirection)
if err != nil {
    log.Errorf("Could not update the call mute direction. err: %v", err)
}

return nil
```
To:
```go
_, err = h.UpdateMuteDirection(ctx, id, newDirection)
if err != nil {
    log.Errorf("Could not update the call mute direction. err: %v", err)
    return errors.Wrap(err, "could not update the call mute direction")
}

return nil
```

**Step 3: Run tests to verify they pass**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-call-manager-bugs/bin-call-manager && go test ./pkg/callhandler/ -run "Test_MuteOn|Test_MuteOff" -v`
Expected: PASS (existing tests mock UpdateMuteDirection to return nil, so they are unaffected).

---

### Task 6: Run full verification workflow

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-call-manager-bugs/bin-call-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All steps pass with no errors.

---

### Task 7: Commit and push

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-call-manager-bugs
git add bin-call-manager/
git commit -m "NOJIRA-fix-call-manager-bugs

Fix 5 bugs found during architecture review of call-manager.

- bin-call-manager: Fix errors.Wrap using wrong variable in HoldOn, HoldOff, MOHOn, MOHOff, SilenceOn, SilenceOff, MuteOn, MuteOff (was wrapping nil instead of actual error)
- bin-call-manager: Fix goroutine in Hangup logging outer err instead of errReq
- bin-call-manager: Fix ConvertWebhookMessage copy-paste bug setting AnswerGroupcallID from AnswerCallID
- bin-call-manager: Fix JSON tag typo reference_typee to reference_type in externalmedia model
- bin-call-manager: Return error from UpdateMuteDirection instead of swallowing it in MuteOn/MuteOff
- bin-call-manager: Add test coverage for error paths in Hold, MOH, Silence, and ConvertWebhookMessage"
```
