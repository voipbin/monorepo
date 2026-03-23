# Fix Non-Realtime Team AIcall Member Switch — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix non-realtime team AIcalls so subsequent messages after a member switch use the correct LLM, and non-realtime AIcalls don't populate meaningless TTS/STT/VAD fields.

**Architecture:** Split AIcall creation into realtime/messaging paths, extract shared team member resolution helper, sync `CurrentMemberID` on member switch events, and resolve team config fresh at send time for team AIcalls.

**Tech Stack:** Go, gomock, table-driven tests

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-nonrealtime-team-aicall-member-switch`

**Service:** `bin-ai-manager`

---

### Task 1: Extract `resolveTeamMemberAI` shared helper

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go:22-52`
- Test: `bin-ai-manager/pkg/aicallhandler/start_test.go`

**Step 1: Write the failing test**

Add test `Test_resolveTeamMemberAI` in `start_test.go`. Table-driven with cases:

1. `member_found` — `memberID` matches a team member → returns that member's AI config + member ID
2. `member_not_found_fallback_to_start_member` — `memberID` not in team → falls back to `team.StartMemberID` → returns start member's AI config + start member ID
3. `neither_found` — both `memberID` and `StartMemberID` missing from team → returns error

Mock expectations:
- `mockTeamHandler` is NOT called (team is passed as parameter)
- `mockAIHandler.Get(ctx, member.AIID)` returns the AI config

```go
func Test_resolveTeamMemberAI(t *testing.T) {
    tests := []struct {
        name     string
        team     *team.Team
        memberID uuid.UUID

        responseAI *ai.AI

        expectAIID    uuid.UUID // which member.AIID is fetched
        expectMemberID uuid.UUID
        expectErr     bool
    }{
        {
            name: "member_found",
            team: &team.Team{
                StartMemberID: uuid.FromStringOrNil("aaaa0000-..."),
                Members: []team.Member{
                    {ID: uuid.FromStringOrNil("aaaa0000-..."), AIID: uuid.FromStringOrNil("ai-start-...")},
                    {ID: uuid.FromStringOrNil("bbbb0000-..."), AIID: uuid.FromStringOrNil("ai-current-...")},
                },
            },
            memberID:   uuid.FromStringOrNil("bbbb0000-..."),
            responseAI: &ai.AI{...},
            expectAIID:    uuid.FromStringOrNil("ai-current-..."),
            expectMemberID: uuid.FromStringOrNil("bbbb0000-..."),
            expectErr:     false,
        },
        {
            name: "member_not_found_fallback_to_start_member",
            // memberID = "cccc0000-..." (not in team)
            // falls back to StartMemberID "aaaa0000-..."
        },
        {
            name: "neither_found",
            // memberID = "cccc0000-..." (not in team)
            // StartMemberID = "dddd0000-..." (also not in team)
            // expects error
        },
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-nonrealtime-team-aicall-member-switch/bin-ai-manager
go test -v -run Test_resolveTeamMemberAI ./pkg/aicallhandler/...
```

Expected: FAIL — `resolveTeamMemberAI` does not exist

**Step 3: Implement `resolveTeamMemberAI`**

Add to `start.go` before `resolveAI`:

```go
// resolveTeamMemberAI finds memberID in the team's members and fetches its AI config.
// If memberID is not found, falls back to team.StartMemberID.
// Returns the AI config and the actual resolved member ID.
func (h *aicallHandler) resolveTeamMemberAI(ctx context.Context, t *team.Team, memberID uuid.UUID) (*ai.AI, uuid.UUID, error) {
    // try preferred member
    for _, m := range t.Members {
        if m.ID == memberID {
            a, err := h.aiHandler.Get(ctx, m.AIID)
            if err != nil {
                return nil, uuid.Nil, errors.Wrapf(err, "could not get ai info for member. ai_id: %s", m.AIID)
            }
            return a, m.ID, nil
        }
    }

    // fallback to start member
    for _, m := range t.Members {
        if m.ID == t.StartMemberID {
            a, err := h.aiHandler.Get(ctx, m.AIID)
            if err != nil {
                return nil, uuid.Nil, errors.Wrapf(err, "could not get ai info for start member. ai_id: %s", m.AIID)
            }
            return a, m.ID, nil
        }
    }

    return nil, uuid.Nil, fmt.Errorf("could not find member or start member in team. team_id: %s, member_id: %s, start_member_id: %s", t.ID, memberID, t.StartMemberID)
}
```

**Step 4: Refactor `resolveAI` to use `resolveTeamMemberAI`**

Replace `resolveAI` Team case (start.go:31-47) with:

```go
case aicall.AssistanceTypeTeam:
    t, err := h.teamHandler.Get(ctx, assistanceID)
    if err != nil {
        return nil, nil, uuid.Nil, errors.Wrapf(err, "could not get team info. team_id: %s", assistanceID)
    }

    a, memberID, err := h.resolveTeamMemberAI(ctx, t, t.StartMemberID)
    if err != nil {
        return nil, nil, uuid.Nil, err
    }
    return a, t.Parameter, memberID, nil
```

**Step 5: Run all tests to verify nothing broke**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-nonrealtime-team-aicall-member-switch/bin-ai-manager
go test -v ./pkg/aicallhandler/...
```

Expected: ALL PASS (including new `Test_resolveTeamMemberAI`)

**Step 6: Commit**

```bash
git add bin-ai-manager/pkg/aicallhandler/start.go bin-ai-manager/pkg/aicallhandler/start_test.go
git commit -m "NOJIRA-Fix-nonrealtime-team-aicall-member-switch

- bin-ai-manager: Extract resolveTeamMemberAI shared helper with fallback to StartMemberID
- bin-ai-manager: Refactor resolveAI to use resolveTeamMemberAI"
```

---

### Task 2: Split AIcall creation into realtime and messaging paths

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/db.go:18-89` (add `CreateByMessaging`)
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go:430-486` (split `startAIcall`)
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go:87-127` (`startReferenceTypeCall` → use `startAIcallByRealtime`)
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go:129-228` (`startReferenceTypeConversation`, `startReferenceTypeNone` → use `startAIcallByMessaging`)
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go:488-521` (`StartTask` → use `startAIcallByMessaging`)
- Test: `bin-ai-manager/pkg/aicallhandler/start_test.go`
- Test: `bin-ai-manager/pkg/aicallhandler/db_test.go`

**Step 1: Write the failing test for `CreateByMessaging`**

Add `Test_CreateByMessaging` in `db_test.go`. Pattern matches existing `Test_Create` test but verifies:
- `AIEngineModel` is set from the AI config
- `AITTSType`, `AITTSVoiceID`, `AISTTType`, `AIVADConfig`, `AISmartTurnEnabled` are zero values
- All common fields are set correctly

Mock expectations: same as `Test_Create` but the `expectAIcall` struct has empty TTS/STT/VAD.

**Step 2: Run test to verify it fails**

```bash
go test -v -run Test_CreateByMessaging ./pkg/aicallhandler/...
```

Expected: FAIL — `CreateByMessaging` does not exist

**Step 3: Implement `CreateByMessaging` in db.go**

Add after the existing `Create` method. Same as `Create` but the AIcall struct only sets `AIEngineModel`:

```go
func (h *aicallHandler) CreateByMessaging(
    ctx context.Context,
    c *ai.AI,
    assistanceType aicall.AssistanceType,
    assistanceID uuid.UUID,
    activeflowID uuid.UUID,
    referenceType aicall.ReferenceType,
    referenceID uuid.UUID,
    pipecatcallID uuid.UUID,
    currentMemberID uuid.UUID,
    gender aicall.Gender,
    language string,
    parameter map[string]any,
) (*aicall.AIcall, error) {
    // Same as Create but:
    // - No confbridgeID parameter
    // - Only sets AIEngineModel (no TTS/STT/VAD/SmartTurn)
    // - ConfbridgeID is uuid.Nil
}
```

Key differences from `Create`:
- Line 48: `AIEngineModel: c.EngineModel,` — kept
- Lines 49-53: TTS/STT/VAD fields — REMOVED (zero values)
- Line 61: `ConfbridgeID: uuid.Nil,` — hardcoded nil (no confbridge param)

**Step 4: Implement `startAIcallByRealtime` and `startAIcallByMessaging` in start.go**

Replace `startAIcall` (lines 430-486) with two functions:

`startAIcallByRealtime` — identical to current `startAIcall` (calls `Create` with confbridgeID param, `isTask` is always false for realtime).

`startAIcallByMessaging` — same structure but:
- Calls `CreateByMessaging` instead of `Create`
- No `confbridgeID` parameter
- `isTask` parameter still needed (for `startInitMessages`)

**Step 5: Update callers**

- `startReferenceTypeCall` (line 113): change `h.startAIcall(...)` → `h.startAIcallByRealtime(...)`
- `startReferenceTypeConversation` (line 164): change `h.startAIcall(...)` → `h.startAIcallByMessaging(...)`
- `startReferenceTypeNone` (line 216): change `h.startAIcall(...)` → `h.startAIcallByMessaging(...)`
- `StartTask` (line 503): change `h.startAIcall(...)` → `h.startAIcallByMessaging(...)`

**Step 6: Update existing `Test_startAIcall` test**

The existing test (start_test.go:892) references `startAIcall`. Split into:
- `Test_startAIcallByRealtime` — same as existing, expects TTS/STT/VAD populated
- `Test_startAIcallByMessaging` — expects TTS/STT/VAD empty, no confbridgeID

**Step 7: Run all tests**

```bash
go test -v ./pkg/aicallhandler/...
```

Expected: ALL PASS

**Step 8: Commit**

```bash
git add bin-ai-manager/pkg/aicallhandler/db.go bin-ai-manager/pkg/aicallhandler/db_test.go \
        bin-ai-manager/pkg/aicallhandler/start.go bin-ai-manager/pkg/aicallhandler/start_test.go
git commit -m "NOJIRA-Fix-nonrealtime-team-aicall-member-switch

- bin-ai-manager: Split startAIcall into startAIcallByRealtime and startAIcallByMessaging
- bin-ai-manager: Add CreateByMessaging that only sets AIEngineModel (no TTS/STT/VAD)
- bin-ai-manager: Non-realtime AIcalls no longer populate meaningless audio fields"
```

---

### Task 3: Update `processEventPMTeamMemberSwitched` to call `UpdateCurrentMemberID`

**Files:**
- Modify: `bin-ai-manager/pkg/subscribehandler/pipecat_message.go:48-64`
- Test: `bin-ai-manager/pkg/subscribehandler/pipecat_message_test.go`

**Step 1: Write the failing test**

Add `TestProcessEventPMTeamMemberSwitched` in `pipecat_message_test.go`. Follow the existing test pattern from `TestProcessEventPMMessageUserTranscription` (line 17-78).

Cases:
1. `processes_team_member_switched_successfully` — verifies both `messageHandler.EventPMTeamMemberSwitched` AND `aicallHandler.UpdateCurrentMemberID` are called
2. `handles_invalid_json_data` — verifies error returned, no handlers called
3. `continues_when_update_current_member_fails` — verifies notification still created even if UpdateCurrentMemberID returns error

Mock setup:
```go
mockMessageHandler := messagehandler.NewMockMessageHandler(ctrl)
mockAIcallHandler := aicallhandler.NewMockAIcallHandler(ctrl)

h := &subscribeHandler{
    messageHandler: mockMessageHandler,
    aicallHandler:  mockAIcallHandler,
}
```

For case 1:
```go
mockMessageHandler.EXPECT().EventPMTeamMemberSwitched(gomock.Any(), gomock.Any()).Times(1)
mockAIcallHandler.EXPECT().UpdateCurrentMemberID(
    gomock.Any(),
    evt.PipecatcallReferenceID,  // aicall ID
    evt.ToMember.ID,              // new member ID
).Return(&aicall.AIcall{}, nil).Times(1)
```

**Step 2: Run test to verify it fails**

```bash
go test -v -run TestProcessEventPMTeamMemberSwitched ./pkg/subscribehandler/...
```

Expected: FAIL — `UpdateCurrentMemberID` mock expectation not satisfied

**Step 3: Implement the change**

Update `processEventPMTeamMemberSwitched` in `pipecat_message.go:48-64`:

```go
func (h *subscribeHandler) processEventPMTeamMemberSwitched(ctx context.Context, m *sock.Event) error {
    log := logrus.WithFields(logrus.Fields{
        "func":  "processEventPMTeamMemberSwitched",
        "event": m,
    })
    log.Debugf("Received the pipecat-manager's team_member_switched event.")

    var evt pmmessage.MemberSwitchedEvent
    if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
        log.Errorf("Could not unmarshal the data. err: %v", err)
        return err
    }

    // create notification message (existing)
    h.messageHandler.EventPMTeamMemberSwitched(ctx, &evt)

    // update current member ID on the aicall record (NEW)
    if _, err := h.aicallHandler.UpdateCurrentMemberID(ctx, evt.PipecatcallReferenceID, evt.ToMember.ID); err != nil {
        log.Errorf("Could not update current member id. aicall_id: %s, to_member_id: %s, err: %v", evt.PipecatcallReferenceID, evt.ToMember.ID, err)
        // continue — notification message was already created, send path has fallback
    }

    return nil
}
```

**Step 4: Run all tests**

```bash
go test -v ./pkg/subscribehandler/...
```

Expected: ALL PASS

**Step 5: Commit**

```bash
git add bin-ai-manager/pkg/subscribehandler/pipecat_message.go bin-ai-manager/pkg/subscribehandler/pipecat_message_test.go
git commit -m "NOJIRA-Fix-nonrealtime-team-aicall-member-switch

- bin-ai-manager: Call UpdateCurrentMemberID when MemberSwitchedEvent received
- bin-ai-manager: Log and continue on error (send path has StartMemberID fallback)"
```

---

### Task 4: Add team resolution in `SendReferenceTypeOthers`

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/send.go:55-86`
- Test: `bin-ai-manager/pkg/aicallhandler/send_test.go` (new file)

**Step 1: Write the failing tests**

Create `send_test.go` with `Test_SendReferenceTypeOthers`. Table-driven cases:

1. `non_team_aicall_uses_existing_engine_model` — `AssistanceType=ai`, no team resolution, uses `c.AIEngineModel` as-is
2. `team_aicall_resolves_current_member` — `AssistanceType=team`, `CurrentMemberID` found in team, overrides `AIEngineModel` with current member's AI EngineModel
3. `team_aicall_fallback_to_start_member` — `CurrentMemberID` not in team, falls back to `StartMemberID`, updates `CurrentMemberID` on DB
4. `team_fetch_fails_uses_fallback` — `teamHandler.Get` returns error, uses existing `c.AIEngineModel` and logs warning

Mock expectations for case 2:
```go
// message create
mockMessageHandler.EXPECT().Create(...).Return(&message.Message{}, nil)
// pipecatcall ID update
mockDB.EXPECT().AIcallUpdate(...).Return(nil)
mockDB.EXPECT().AIcallGet(...).Return(updatedAIcall, nil)
// team resolution
mockTeamHandler.EXPECT().Get(ctx, assistanceID).Return(team, nil)
mockAIHandler.EXPECT().Get(ctx, currentMemberAIID).Return(currentMemberAI, nil)
// pipecatcall start
mockReqHandler.EXPECT().PipecatV1PipecatcallStart(
    ctx,
    gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
    pmpipecatcall.LLMType(currentMemberAI.EngineModel),  // ← KEY: correct engine model
    gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
).Return(&pmpipecatcall.Pipecatcall{HostID: "host1"}, nil)
// terminate with delay
mockReqHandler.EXPECT().PipecatV1PipecatcallTerminateWithDelay(...).Return(nil)
```

**Step 2: Run test to verify it fails**

```bash
go test -v -run Test_SendReferenceTypeOthers ./pkg/aicallhandler/...
```

Expected: FAIL — team resolution not implemented yet

**Step 3: Implement team resolution in `SendReferenceTypeOthers`**

Update `send.go:55-86`. Add team resolution after the message create and pipecatcall ID update, before calling `startPipecatcall`:

```go
func (h *aicallHandler) SendReferenceTypeOthers(ctx context.Context, c *aicall.AIcall, role message.Role, messageText string) (*message.Message, error) {
    log := logrus.WithFields(logrus.Fields{
        "func":      "SendReferenceTypeOthers",
        "aicall_id": c.ID,
    })

    // create user message
    aicallID := c.ID
    res, err := h.messageHandler.Create(ctx, c.CustomerID, aicallID, c.ActiveflowID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "")
    if err != nil {
        return nil, errors.Wrapf(err, "could not create the message. aicall_id: %s", aicallID)
    }

    newPipecatcallID := h.utilHandler.UUIDCreate()
    c, err = h.UpdatePipecatcallID(ctx, aicallID, newPipecatcallID)
    if err != nil {
        return nil, errors.Wrapf(err, "could not update the pipecatcall id for existing aicall. aicall_id: %s", aicallID)
    }
    log.WithField("message", res).Debugf("Created the message to the ai. aicall_id: %s, message_id: %s", aicallID, res.ID)

    // resolve current team member's AI config for team-based aicalls (NEW)
    if c.AssistanceType == aicall.AssistanceTypeTeam {
        if err := h.resolveTeamMemberForSend(ctx, c); err != nil {
            log.Warnf("Could not resolve team member AI config, using existing. err: %v", err)
        }
    }

    pc, err := h.startPipecatcall(ctx, c)
    if err != nil {
        return nil, errors.Wrapf(err, "could not start pipecatcall for aicall. aicall_id: %s", res.ID)
    }
    log.WithField("pipecatcall", pc).Debugf("Started pipecatcall for aicall. aicall_id: %s", res.ID)

    if err = h.reqHandler.PipecatV1PipecatcallTerminateWithDelay(ctx, pc.HostID, pc.ID, defaultAITaskTimeout); err != nil {
        return nil, errors.Wrapf(err, "could not send the pipecatcall terminate request correctly")
    }

    return res, nil
}

// resolveTeamMemberForSend fetches the team config, resolves the current member's AI,
// and overrides c.AIEngineModel in-memory. If CurrentMemberID was not found in the team,
// falls back to StartMemberID and updates CurrentMemberID on the DB record.
func (h *aicallHandler) resolveTeamMemberForSend(ctx context.Context, c *aicall.AIcall) error {
    log := logrus.WithFields(logrus.Fields{
        "func":      "resolveTeamMemberForSend",
        "aicall_id": c.ID,
    })

    t, err := h.teamHandler.Get(ctx, c.AssistanceID)
    if err != nil {
        return errors.Wrapf(err, "could not get team info. team_id: %s", c.AssistanceID)
    }
    log.WithField("team", t).Debugf("Retrieved team info. team_id: %s", t.ID)

    a, resolvedMemberID, err := h.resolveTeamMemberAI(ctx, t, c.CurrentMemberID)
    if err != nil {
        return errors.Wrapf(err, "could not resolve team member AI")
    }
    log.WithField("ai", a).Debugf("Resolved team member AI. member_id: %s, ai_engine_model: %s", resolvedMemberID, a.EngineModel)

    // override engine model in-memory for this pipecat session
    c.AIEngineModel = a.EngineModel

    // if fallback occurred, update CurrentMemberID on the DB record
    if resolvedMemberID != c.CurrentMemberID {
        log.Infof("CurrentMemberID not found in team, fell back to StartMemberID. updating. aicall_id: %s, old: %s, new: %s", c.ID, c.CurrentMemberID, resolvedMemberID)
        if _, errUpdate := h.UpdateCurrentMemberID(ctx, c.ID, resolvedMemberID); errUpdate != nil {
            log.Errorf("Could not update CurrentMemberID after fallback. err: %v", errUpdate)
        }
        c.CurrentMemberID = resolvedMemberID
    }

    return nil
}
```

**Step 4: Run all tests**

```bash
go test -v ./pkg/aicallhandler/...
```

Expected: ALL PASS

**Step 5: Commit**

```bash
git add bin-ai-manager/pkg/aicallhandler/send.go bin-ai-manager/pkg/aicallhandler/send_test.go
git commit -m "NOJIRA-Fix-nonrealtime-team-aicall-member-switch

- bin-ai-manager: Resolve team member AI config fresh at send time for team-based AIcalls
- bin-ai-manager: Fall back to StartMemberID if CurrentMemberID no longer in team"
```

---

### Task 5: Full verification workflow

**Step 1: Run the full verification workflow**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-nonrealtime-team-aicall-member-switch/bin-ai-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: ALL PASS, no lint errors

**Step 2: Fix any issues found by lint or tests**

If `go generate` regenerates mocks, commit them. If lint finds issues, fix and recommit.

**Step 3: Final commit (if needed for generated files)**

```bash
git add bin-ai-manager/pkg/aicallhandler/mock_main.go
git commit -m "NOJIRA-Fix-nonrealtime-team-aicall-member-switch

- bin-ai-manager: Regenerate mocks after interface changes"
```

---

### Task 6: Push branch and create PR

**Step 1: Check for conflicts with main**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-nonrealtime-team-aicall-member-switch
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
```

**Step 2: Push and create PR**

```bash
git push -u origin NOJIRA-Fix-nonrealtime-team-aicall-member-switch
```

PR title: `NOJIRA-Fix-nonrealtime-team-aicall-member-switch`

PR body:
```
Fix non-realtime team AIcalls so subsequent messages after a member switch use the
correct LLM engine model. Also stop populating meaningless TTS/STT/VAD fields on
non-realtime AIcalls.

- bin-ai-manager: Extract resolveTeamMemberAI shared helper with fallback to StartMemberID
- bin-ai-manager: Split startAIcall into startAIcallByRealtime and startAIcallByMessaging
- bin-ai-manager: Add CreateByMessaging that only sets AIEngineModel (no TTS/STT/VAD)
- bin-ai-manager: Call UpdateCurrentMemberID when MemberSwitchedEvent received
- bin-ai-manager: Resolve team member AI config fresh at send time for team-based AIcalls
- bin-ai-manager: Fall back to StartMemberID if CurrentMemberID no longer in team

Known limitation: system prompt continuity after member switch tracked in #735.
```
