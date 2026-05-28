# Fix member_switched active_ai_id Attribution Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix `EventPMTeamMemberSwitched` so the `member_switched` notification message is attributed to the FROM AI (not the TO AI), giving each AI a complete and accurate audit transcript.

**Architecture:** Split the single `resolveTeamMemberAIID` call in `EventPMTeamMemberSwitched` into two — `fromAIID` for the notification message's `active_ai_id`, and `toAIID` for participant recording. No schema changes; no other services touched.

**Tech Stack:** Go 1.21+, gomock, `bin-ai-manager` service only.

---

## File Map

| File | Change |
|---|---|
| `bin-ai-manager/pkg/messagehandler/event.go` | Split `activeAIID` into `fromAIID` + `toAIID`; route each to its correct use |
| `bin-ai-manager/pkg/messagehandler/event_test.go` | Update all 4 `TestEventPMTeamMemberSwitched*` tests to reflect two-lookup behavior and corrected attribution |

No other files change.

---

## Task 1: Update Tests (TDD — write failing tests first)

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/event_test.go`

The four existing `TestEventPMTeamMemberSwitched*` tests all assume a single `resolveTeamMemberAIID` call (one `AIV1AIcallGet` + one `TeamGet`) and that `ActiveAIID` on the notification equals the TO member's AI ID. After the fix these tests must pass against the new behavior. Rewrite all four now so they fail red against the current code.

### 1a — `TestEventPMTeamMemberSwitched`: verify notification has `fromAIID`

- [ ] Replace the entire `TestEventPMTeamMemberSwitched` function with the version below.

```go
func TestEventPMTeamMemberSwitched(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	customerID   := uuid.Must(uuid.NewV4())
	aicallID     := uuid.Must(uuid.NewV4())
	teamID       := uuid.Must(uuid.NewV4())
	fromMemberID := uuid.Must(uuid.NewV4())
	toMemberID   := uuid.Must(uuid.NewV4())
	fromAIID     := uuid.Must(uuid.NewV4())
	toAIID       := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	createdMsgID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.MemberSwitchedEvent{
		CustomerID:             customerID,
		PipecatcallReferenceID: aicallID,
		ActiveflowID:           activeflowID,
		TransitionFunctionName: "escalate",
		FromMember: pmmessage.MemberInfo{
			ID:          fromMemberID,
			Name:        "Alice",
			EngineModel: "openai.gpt-4o",
		},
		ToMember: pmmessage.MemberInfo{
			ID:          toMemberID,
			Name:        "Bob",
			EngineModel: "openai.gpt-4o-mini",
		},
	}

	ac := &aicall.AIcall{
		AssistanceType: aicall.AssistanceTypeTeam,
		AssistanceID:   teamID,
	}
	mockReq := requesthandler.NewMockRequestHandler(ctrl)
	// resolveTeamMemberAIID is called twice: once for fromMember, once for toMember.
	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil).Times(2)

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	// TeamGet is called twice for the same team.
	mockDB.EXPECT().TeamGet(gomock.Any(), teamID).Return(&team.Team{
		Members: []team.Member{
			{ID: fromMemberID, AIID: fromAIID},
			{ID: toMemberID, AIID: toAIID},
		},
	}, nil).Times(2)

	// Notification message must be attributed to the FROM AI.
	mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, m *message.Message) error {
			if m.ActiveAIID != fromAIID {
				t.Errorf("expected ActiveAIID (fromAIID) %v, got %v", fromAIID, m.ActiveAIID)
			}
			if m.AIcallID != aicallID {
				t.Errorf("expected AIcallID %v, got %v", aicallID, m.AIcallID)
			}
			if m.Role != message.RoleNotification {
				t.Errorf("expected Role notification, got %s", m.Role)
			}
			m.ID = createdMsgID
			return nil
		},
	).Times(1)

	createdMsg := &message.Message{}
	createdMsg.ID = createdMsgID
	createdMsg.CustomerID = customerID
	mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(createdMsg, nil).Times(1)

	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	h := &messageHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	h.EventPMTeamMemberSwitched(context.Background(), evt)
}
```

### 1b — `TestEventPMTeamMemberSwitched_participant_written`: participant created with `toAIID`

- [ ] Replace the entire `TestEventPMTeamMemberSwitched_participant_written` function with the version below.

```go
func TestEventPMTeamMemberSwitched_participant_written(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	customerID   := uuid.Must(uuid.NewV4())
	aicallID     := uuid.Must(uuid.NewV4())
	teamID       := uuid.Must(uuid.NewV4())
	fromMemberID := uuid.Must(uuid.NewV4())
	toMemberID   := uuid.Must(uuid.NewV4())
	fromAIID     := uuid.Must(uuid.NewV4())
	toAIID       := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	createdMsgID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.MemberSwitchedEvent{
		CustomerID:             customerID,
		PipecatcallReferenceID: aicallID,
		ActiveflowID:           activeflowID,
		TransitionFunctionName: "escalate",
		FromMember: pmmessage.MemberInfo{ID: fromMemberID, Name: "Alice", EngineModel: "openai.gpt-4o"},
		ToMember:   pmmessage.MemberInfo{ID: toMemberID, Name: "Bob", EngineModel: "openai.gpt-4o-mini"},
	}

	ac := &aicall.AIcall{
		AssistanceType: aicall.AssistanceTypeTeam,
		AssistanceID:   teamID,
	}

	mockReq := requesthandler.NewMockRequestHandler(ctrl)
	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil).Times(2)

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockDB.EXPECT().TeamGet(gomock.Any(), teamID).Return(&team.Team{
		Members: []team.Member{
			{ID: fromMemberID, AIID: fromAIID},
			{ID: toMemberID, AIID: toAIID},
		},
	}, nil).Times(2)
	mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, m *message.Message) error {
			m.ID = createdMsgID
			return nil
		},
	).Times(1)
	createdMsg := &message.Message{}
	createdMsg.ID = createdMsgID
	createdMsg.CustomerID = customerID
	mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(createdMsg, nil).Times(1)

	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	mockParticipant := participanthandler.NewMockParticipantHandler(ctrl)
	// Participant must be recorded for the TO AI, not the FROM AI.
	mockParticipant.EXPECT().Create(gomock.Any(), aicallID, toAIID).Return(nil).Times(1)

	h := &messageHandler{
		db:                 mockDB,
		notifyHandler:      mockNotify,
		reqHandler:         mockReq,
		utilHandler:        utilhandler.NewUtilHandler(),
		participantHandler: mockParticipant,
	}

	h.EventPMTeamMemberSwitched(context.Background(), evt)
}
```

### 1c — `TestEventPMTeamMemberSwitched_participant_skipped_when_nil_ai`: AIcall fails twice

- [ ] Replace the entire `TestEventPMTeamMemberSwitched_participant_skipped_when_nil_ai` function with the version below.

```go
func TestEventPMTeamMemberSwitched_participant_skipped_when_nil_ai(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	customerID   := uuid.Must(uuid.NewV4())
	aicallID     := uuid.Must(uuid.NewV4())
	fromMemberID := uuid.Must(uuid.NewV4())
	toMemberID   := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	createdMsgID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.MemberSwitchedEvent{
		CustomerID:             customerID,
		PipecatcallReferenceID: aicallID,
		ActiveflowID:           activeflowID,
		TransitionFunctionName: "escalate",
		FromMember: pmmessage.MemberInfo{ID: fromMemberID, Name: "Alice"},
		ToMember:   pmmessage.MemberInfo{ID: toMemberID, Name: "Bob"},
	}

	// AIcall get fails for both resolveTeamMemberAIID calls → fromAIID=uuid.Nil, toAIID=uuid.Nil.
	mockReq := requesthandler.NewMockRequestHandler(ctrl)
	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(nil, errors.New("not found")).Times(2)

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, m *message.Message) error {
			m.ID = createdMsgID
			return nil
		},
	).Times(1)
	createdMsg := &message.Message{}
	createdMsg.ID = createdMsgID
	createdMsg.CustomerID = customerID
	mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(createdMsg, nil).Times(1)

	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	mockParticipant := participanthandler.NewMockParticipantHandler(ctrl)
	// toAIID is uuid.Nil → participant write must be skipped.
	mockParticipant.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	h := &messageHandler{
		db:                 mockDB,
		notifyHandler:      mockNotify,
		reqHandler:         mockReq,
		utilHandler:        utilhandler.NewUtilHandler(),
		participantHandler: mockParticipant,
	}

	h.EventPMTeamMemberSwitched(context.Background(), evt)
}
```

### 1d — `TestEventPMTeamMemberSwitched_participant_create_error`: participant error still uses `toAIID`

- [ ] Replace the entire `TestEventPMTeamMemberSwitched_participant_create_error` function with the version below.

```go
func TestEventPMTeamMemberSwitched_participant_create_error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	customerID   := uuid.Must(uuid.NewV4())
	aicallID     := uuid.Must(uuid.NewV4())
	teamID       := uuid.Must(uuid.NewV4())
	fromMemberID := uuid.Must(uuid.NewV4())
	toMemberID   := uuid.Must(uuid.NewV4())
	fromAIID     := uuid.Must(uuid.NewV4())
	toAIID       := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	createdMsgID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.MemberSwitchedEvent{
		CustomerID:             customerID,
		PipecatcallReferenceID: aicallID,
		ActiveflowID:           activeflowID,
		TransitionFunctionName: "escalate",
		FromMember: pmmessage.MemberInfo{ID: fromMemberID, Name: "Alice", EngineModel: "openai.gpt-4o"},
		ToMember:   pmmessage.MemberInfo{ID: toMemberID, Name: "Bob", EngineModel: "openai.gpt-4o-mini"},
	}

	ac := &aicall.AIcall{
		AssistanceType: aicall.AssistanceTypeTeam,
		AssistanceID:   teamID,
	}

	mockReq := requesthandler.NewMockRequestHandler(ctrl)
	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil).Times(2)

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockDB.EXPECT().TeamGet(gomock.Any(), teamID).Return(&team.Team{
		Members: []team.Member{
			{ID: fromMemberID, AIID: fromAIID},
			{ID: toMemberID, AIID: toAIID},
		},
	}, nil).Times(2)
	mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, m *message.Message) error {
			m.ID = createdMsgID
			return nil
		},
	).Times(1)
	createdMsg := &message.Message{}
	createdMsg.ID = createdMsgID
	createdMsg.CustomerID = customerID
	mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(createdMsg, nil).Times(1)

	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	mockParticipant := participanthandler.NewMockParticipantHandler(ctrl)
	// Participant create fails; handler must warn and continue (not panic or return early).
	mockParticipant.EXPECT().Create(gomock.Any(), aicallID, toAIID).Return(errors.New("db error")).Times(1)

	h := &messageHandler{
		db:                 mockDB,
		notifyHandler:      mockNotify,
		reqHandler:         mockReq,
		utilHandler:        utilhandler.NewUtilHandler(),
		participantHandler: mockParticipant,
	}

	h.EventPMTeamMemberSwitched(context.Background(), evt)
}
```

### 1e — `TestEventPMTeamMemberSwitched_from_ai_nil_participant_still_written`: only `fromAIID` nil, participant proceeds with `toAIID`

- [ ] Add the new test below after the four existing ones.

```go
func TestEventPMTeamMemberSwitched_from_ai_nil_participant_still_written(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	customerID   := uuid.Must(uuid.NewV4())
	aicallID     := uuid.Must(uuid.NewV4())
	teamID       := uuid.Must(uuid.NewV4())
	fromMemberID := uuid.Must(uuid.NewV4())
	toMemberID   := uuid.Must(uuid.NewV4())
	toAIID       := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	createdMsgID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.MemberSwitchedEvent{
		CustomerID:             customerID,
		PipecatcallReferenceID: aicallID,
		ActiveflowID:           activeflowID,
		TransitionFunctionName: "escalate",
		FromMember: pmmessage.MemberInfo{ID: fromMemberID, Name: "Alice"},
		ToMember:   pmmessage.MemberInfo{ID: toMemberID, Name: "Bob"},
	}

	ac := &aicall.AIcall{
		AssistanceType: aicall.AssistanceTypeTeam,
		AssistanceID:   teamID,
	}

	mockReq := requesthandler.NewMockRequestHandler(ctrl)
	// Both calls to resolveTeamMemberAIID go through AIV1AIcallGet.
	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil).Times(2)

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	// Team only contains the toMember; fromMember is not found → fromAIID = uuid.Nil.
	// TeamGet is still called twice (once per resolveTeamMemberAIID).
	mockDB.EXPECT().TeamGet(gomock.Any(), teamID).Return(&team.Team{
		Members: []team.Member{
			{ID: toMemberID, AIID: toAIID},
			// fromMemberID deliberately absent → resolveTeamMemberAIID returns uuid.Nil for it
		},
	}, nil).Times(2)

	// Notification is created with uuid.Nil for active_ai_id (from lookup failed).
	mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, m *message.Message) error {
			if m.ActiveAIID != uuid.Nil {
				t.Errorf("expected ActiveAIID uuid.Nil when from-member not found, got %v", m.ActiveAIID)
			}
			m.ID = createdMsgID
			return nil
		},
	).Times(1)
	createdMsg := &message.Message{}
	createdMsg.ID = createdMsgID
	createdMsg.CustomerID = customerID
	mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(createdMsg, nil).Times(1)

	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	mockParticipant := participanthandler.NewMockParticipantHandler(ctrl)
	// toAIID is valid → participant write must still proceed even though fromAIID is nil.
	mockParticipant.EXPECT().Create(gomock.Any(), aicallID, toAIID).Return(nil).Times(1)

	h := &messageHandler{
		db:                 mockDB,
		notifyHandler:      mockNotify,
		reqHandler:         mockReq,
		utilHandler:        utilhandler.NewUtilHandler(),
		participantHandler: mockParticipant,
	}

	h.EventPMTeamMemberSwitched(context.Background(), evt)
}
```

- [ ] Run the target tests to confirm they are RED:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-member-switched-active-ai-attribution/bin-ai-manager
go test ./pkg/messagehandler/... -run TestEventPMTeamMemberSwitched -v 2>&1 | tail -30
```

Expected: tests fail (mock call count mismatch — `AIV1AIcallGet` and `TeamGet` called once but expected twice, and `ActiveAIID` mismatch).

---

## Task 2: Implement the Fix

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/event.go`

- [ ] In `EventPMTeamMemberSwitched`, replace the single `activeAIID` resolution with two separate lookups. Find this block (near the end of the function, after `contentBytes` is marshalled):

```go
	activeAIID := h.resolveTeamMemberAIID(ctx, evt.PipecatcallReferenceID, evt.ToMember.ID)
	tmp, err := h.Create(ctx, uuid.Nil, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionOutgoing, message.RoleNotification, string(contentBytes), nil, "",
		WithActiveAIID(activeAIID))
	if err != nil {
		log.Errorf("Could not create the notification message. err: %v", err)
		return
	}
	log.WithField("message", tmp).Debugf("Created member-switched notification message.")

	if h.participantHandler != nil {
		if activeAIID == uuid.Nil {
			log.Warnf("Could not resolve AI ID for new member — skipping participant write. aicall_id: %s, member_id: %s",
				evt.PipecatcallReferenceID, evt.ToMember.ID)
		} else if err := h.participantHandler.Create(ctx, evt.PipecatcallReferenceID, activeAIID); err != nil {
			log.Warnf("Could not record aicall participant. aicall_id: %s, ai_id: %s, err: %v",
				evt.PipecatcallReferenceID, activeAIID, err)
		}
	}
```

Replace with:

```go
	fromAIID := h.resolveTeamMemberAIID(ctx, evt.PipecatcallReferenceID, evt.FromMember.ID)
	toAIID := h.resolveTeamMemberAIID(ctx, evt.PipecatcallReferenceID, evt.ToMember.ID)
	tmp, err := h.Create(ctx, uuid.Nil, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionOutgoing, message.RoleNotification, string(contentBytes), nil, "",
		WithActiveAIID(fromAIID))
	if err != nil {
		log.Errorf("Could not create the notification message. err: %v", err)
		return
	}
	log.WithField("message", tmp).Debugf("Created member-switched notification message.")

	if h.participantHandler != nil {
		if toAIID == uuid.Nil {
			log.Warnf("Could not resolve AI ID for new member — skipping participant write. aicall_id: %s, member_id: %s",
				evt.PipecatcallReferenceID, evt.ToMember.ID)
		} else if err := h.participantHandler.Create(ctx, evt.PipecatcallReferenceID, toAIID); err != nil {
			log.Warnf("Could not record aicall participant. aicall_id: %s, ai_id: %s, err: %v",
				evt.PipecatcallReferenceID, toAIID, err)
		}
	}
```

- [ ] Run the target tests to confirm they are GREEN:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-member-switched-active-ai-attribution/bin-ai-manager
go test ./pkg/messagehandler/... -run TestEventPMTeamMemberSwitched -v 2>&1 | tail -30
```

Expected: all four `TestEventPMTeamMemberSwitched*` tests pass.

---

## Task 3: Full Verification and Commit

**Files:** No new changes — verification only.

- [ ] Run the full verification workflow:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-member-switched-active-ai-attribution/bin-ai-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: all steps exit 0.

- [ ] Commit:

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-member-switched-active-ai-attribution
git add bin-ai-manager/pkg/messagehandler/event.go \
        bin-ai-manager/pkg/messagehandler/event_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-Fix-member-switched-active-ai-attribution

- bin-ai-manager: Attribute member_switched notification to FROM AI for correct audit transcripts
- bin-ai-manager: Update EventPMTeamMemberSwitched tests to reflect two-lookup behavior
EOF
)"
```
