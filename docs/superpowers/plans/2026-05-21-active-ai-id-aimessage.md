# Add `active_ai_id` to AI Messages — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `active_ai_id` to `ai_messages` DB table, `message.Message` model, and both webhook structs so consumers of `aimessage_created` and `aimessage_intermediate` events can identify which AI config was active when the message was created.

**Architecture:** Add a single `uuid.UUID` field to the data model (DB + Go structs), add a `WithActiveAIID` create option plus three private resolution helpers in `messagehandler`, and wire all 13 existing call sites across `messagehandler/event.go` and `aicallhandler/start.go|send.go|tool.go`. All changes are confined to `bin-ai-manager` and the DB migration in `bin-dbscheme-manager`.

**Tech Stack:** Go 1.22, `github.com/gofrs/uuid`, gomock table-driven tests, Alembic (Python) for MySQL migrations, Sphinx RST for user-facing docs.

---

## File Map

| File | Action | Purpose |
|---|---|---|
| `bin-ai-manager/models/message/main.go` | Modify | Add `ActiveAIID uuid.UUID` field to `Message` |
| `bin-ai-manager/models/message/field.go` | Modify | Add `FieldActiveAIID` constant |
| `bin-ai-manager/models/message/webhook.go` | Modify | Add `ActiveAIID` to both webhook structs + propagate in `ConvertWebhookMessage` |
| `bin-ai-manager/pkg/messagehandler/main.go` | Modify | Add `activeAIID` to `createParams`; add `WithActiveAIID` option |
| `bin-ai-manager/pkg/messagehandler/db.go` | Modify | Apply `p.activeAIID` when constructing `message.Message` in `Create` |
| `bin-ai-manager/pkg/messagehandler/event.go` | Modify | Add `resolveActiveAIIDFromAIcall`, `resolveActiveAIID`, `resolveTeamMemberAIID` helpers; wire all 7 event handlers |
| `bin-ai-manager/pkg/aicallhandler/start.go` | Modify | Wire 2 `messageHandler.Create` call sites |
| `bin-ai-manager/pkg/aicallhandler/send.go` | Modify | Wire 2 `messageHandler.Create` call sites |
| `bin-ai-manager/pkg/aicallhandler/tool.go` | Modify | Wire 2 `messageHandler.Create` call sites |
| `bin-ai-manager/pkg/messagehandler/main_test.go` | Modify | Add `WithActiveAIID` option test |
| `bin-ai-manager/pkg/messagehandler/db_test.go` | Modify | Add `active_ai_id` field assertions to existing `Test_Create` cases |
| `bin-ai-manager/pkg/messagehandler/event_test.go` | Modify | Add tests for the three resolution helpers + updated event handlers |
| `bin-ai-manager/docs/domain.md` | Modify | Add `active_ai_id` to Message section |
| `bin-dbscheme-manager/bin-manager/main/versions/<rev>.py` | Create | Alembic migration for `active_ai_id BINARY(16) NOT NULL DEFAULT 0x000...` |
| `bin-api-manager/docsdev/source/ai_struct_message.rst` | Modify | Document `active_ai_id` field |
| `bin-api-manager/docsdev/build/` | Rebuild | Force-add built HTML alongside RST edit |

---

## Task 1: Data model — `Message` struct + `field.go`

**Files:**
- Modify: `bin-ai-manager/models/message/main.go`
- Modify: `bin-ai-manager/models/message/field.go`
- Modify: `bin-ai-manager/models/message/main_test.go` (existing)

- [ ] **Step 1: Add `ActiveAIID` to the `Message` struct**

In `bin-ai-manager/models/message/main.go`, add the new field right after `AIcallID` and `ActiveflowID`:

```go
type Message struct {
    identity.Identity

    AIcallID     uuid.UUID `json:"aicall_id,omitempty" db:"aicall_id,uuid"`
    ActiveflowID uuid.UUID `json:"activeflow_id,omitempty" db:"activeflow_id,uuid"`
    ActiveAIID   uuid.UUID `json:"active_ai_id,omitempty" db:"active_ai_id,uuid"`

    // ... rest unchanged
```

- [ ] **Step 2: Add `FieldActiveAIID` constant**

In `bin-ai-manager/models/message/field.go`, add after `FieldActiveflowID`:

```go
FieldAIcallID     Field = "aicall_id"
FieldActiveflowID Field = "activeflow_id"
FieldActiveAIID   Field = "active_ai_id"
```

- [ ] **Step 3: Verify compilation**

```bash
cd bin-ai-manager && go build ./models/message/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add bin-ai-manager/models/message/main.go bin-ai-manager/models/message/field.go
git commit -m "NOJIRA-Add-active-ai-id-to-aimessage

- bin-ai-manager: Add ActiveAIID field to message.Message and FieldActiveAIID constant"
```

---

## Task 2: Webhook structs — `webhook.go`

**Files:**
- Modify: `bin-ai-manager/models/message/webhook.go`
- Modify: `bin-ai-manager/models/message/webhook_test.go`

- [ ] **Step 1: Add `ActiveAIID` to `WebhookMessage`**

In `bin-ai-manager/models/message/webhook.go`:

```go
type WebhookMessage struct {
    identity.Identity

    AIcallID     uuid.UUID `json:"aicall_id,omitempty"`
    ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`
    ActiveAIID   uuid.UUID `json:"active_ai_id,omitempty"`

    Role      Role      `json:"role"`
    Content   string    `json:"content"`
    Direction Direction `json:"direction"`

    ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
    ToolCallID string     `json:"tool_call_id,omitempty"`

    TMCreate *time.Time `json:"tm_create"`
}
```

- [ ] **Step 2: Propagate in `ConvertWebhookMessage`**

```go
func (h *Message) ConvertWebhookMessage() *WebhookMessage {
    return &WebhookMessage{
        Identity: h.Identity,

        AIcallID:     h.AIcallID,
        ActiveflowID: h.ActiveflowID,
        ActiveAIID:   h.ActiveAIID,

        Role:      h.Role,
        Content:   h.Content,
        Direction: h.Direction,

        ToolCalls:  h.ToolCalls,
        ToolCallID: h.ToolCallID,

        TMCreate: h.TMCreate,
    }
}
```

- [ ] **Step 3: Add `ActiveAIID` to `IntermediateWebhookMessage`**

```go
type IntermediateWebhookMessage struct {
    identity.Identity

    AIcallID     uuid.UUID `json:"aicall_id,omitempty"`
    ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`
    ActiveAIID   uuid.UUID `json:"active_ai_id,omitempty"`

    Role      Role      `json:"role"`
    Content   string    `json:"content"`
    Direction Direction `json:"direction"`

    Sequence int `json:"sequence"`
}
```

- [ ] **Step 4: Write a failing test verifying `ConvertWebhookMessage` propagates `ActiveAIID`**

Open `bin-ai-manager/models/message/webhook_test.go` and read its existing test structure. Add a new test case to the existing `ConvertWebhookMessage` test (or create a new one if none exists):

```go
func TestConvertWebhookMessage_propagatesActiveAIID(t *testing.T) {
    aiID := uuid.Must(uuid.NewV4())
    m := &Message{
        ActiveAIID: aiID,
    }
    wm := m.ConvertWebhookMessage()
    if wm.ActiveAIID != aiID {
        t.Errorf("expected ActiveAIID %s, got %s", aiID, wm.ActiveAIID)
    }
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd bin-ai-manager && go test ./models/message/... -v -run TestConvertWebhookMessage
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add bin-ai-manager/models/message/webhook.go bin-ai-manager/models/message/webhook_test.go
git commit -m "NOJIRA-Add-active-ai-id-to-aimessage

- bin-ai-manager: Add ActiveAIID to WebhookMessage, IntermediateWebhookMessage, and ConvertWebhookMessage"
```

---

## Task 3: `CreateOption` + `createParams` in `messagehandler`

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/main.go`
- Modify: `bin-ai-manager/pkg/messagehandler/db.go`
- Modify: `bin-ai-manager/pkg/messagehandler/main_test.go`

- [ ] **Step 1: Write a failing test for the new `WithActiveAIID` option**

In `bin-ai-manager/pkg/messagehandler/main_test.go`, add to the `TestCreateOptions_apply` test (after the existing `WithPipecatcallID` and `WithDeliveryStatus` assertions):

```go
func TestCreateOptions_WithActiveAIID(t *testing.T) {
    aiID := uuid.Must(uuid.NewV4())
    var p createParams
    WithActiveAIID(aiID)(&p)
    if p.activeAIID != aiID {
        t.Fatalf("WithActiveAIID not applied: got %s", p.activeAIID)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd bin-ai-manager && go test ./pkg/messagehandler/... -v -run TestCreateOptions_WithActiveAIID
```

Expected: FAIL — `createParams` has no `activeAIID` field.

- [ ] **Step 3: Add `activeAIID` to `createParams` and add `WithActiveAIID`**

In `bin-ai-manager/pkg/messagehandler/main.go`:

```go
// createParams holds optional parameters for Create.
type createParams struct {
    pipecatcallID  uuid.UUID
    deliveryStatus message.DeliveryStatus
    activeAIID     uuid.UUID
}

// WithActiveAIID sets the active AI ID on createParams.
func WithActiveAIID(id uuid.UUID) CreateOption {
    return func(p *createParams) { p.activeAIID = id }
}
```

- [ ] **Step 4: Apply `p.activeAIID` in `Create`**

In `bin-ai-manager/pkg/messagehandler/db.go`, update the `message.Message` construction to include `ActiveAIID`:

```go
m := &message.Message{
    Identity: identity.Identity{
        ID:         id,
        CustomerID: customerID,
    },
    AIcallID:     aicallID,
    ActiveflowID: activeflowID,
    ActiveAIID:   p.activeAIID,

    Direction:  direction,
    Role:       role,
    Content:    content,
    ToolCalls:  tmpToolCalls,
    ToolCallID: toolCallID,

    PipecatcallID:  p.pipecatcallID,
    DeliveryStatus: p.deliveryStatus,
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd bin-ai-manager && go test ./pkg/messagehandler/... -v -run TestCreateOptions_WithActiveAIID
```

Expected: PASS.

- [ ] **Step 6: Update `Test_Create` in `db_test.go` to assert `ActiveAIID`**

In `bin-ai-manager/pkg/messagehandler/db_test.go`, add `activeAIID` fields to the test table. For the "have all" case, assign a non-nil value and verify it appears in `expectMessage`. For "empty", the zero value is fine. Example diff for "have all" case:

```go
// in test struct add:
activeAIID uuid.UUID

// in "have all" case:
activeAIID: uuid.FromStringOrNil("aabbccdd-1234-5678-abcd-ef1234567890"),

// in expectMessage "have all":
ActiveAIID: uuid.FromStringOrNil("aabbccdd-1234-5678-abcd-ef1234567890"),
```

Then update the `h.Create(...)` call at the bottom of the test to pass `WithActiveAIID(tt.activeAIID)` and update `mockDB.EXPECT().MessageCreate(ctx, tt.expectMessage)` (the existing matcher uses `reflect.DeepEqual` so `expectMessage` must include `ActiveAIID`).

- [ ] **Step 7: Run all messagehandler tests**

```bash
cd bin-ai-manager && go test ./pkg/messagehandler/... -v
```

Expected: all PASS.

- [ ] **Step 8: Commit**

```bash
git add bin-ai-manager/pkg/messagehandler/main.go \
        bin-ai-manager/pkg/messagehandler/db.go \
        bin-ai-manager/pkg/messagehandler/main_test.go \
        bin-ai-manager/pkg/messagehandler/db_test.go
git commit -m "NOJIRA-Add-active-ai-id-to-aimessage

- bin-ai-manager: Add WithActiveAIID CreateOption and apply in messagehandler.Create"
```

---

## Task 4: Resolution helpers in `messagehandler/event.go`

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/event.go`
- Modify: `bin-ai-manager/pkg/messagehandler/event_test.go`

### Background

Three private helpers are added to `messageHandler`. All are non-blocking: they log `Warnf` and return `uuid.Nil` on any error so message creation is never interrupted.

- `resolveActiveAIIDFromAIcall(ctx, ac *aicall.AIcall) uuid.UUID` — core logic, takes an already-fetched AIcall to avoid extra RPCs.
- `resolveActiveAIID(ctx, aicallID uuid.UUID) uuid.UUID` — thin wrapper for call sites that only have an aicall ID.
- `resolveTeamMemberAIID(ctx, aicallID, memberID uuid.UUID) uuid.UUID` — used for `EventPMTeamMemberSwitched` where the notification message is created *before* `UpdateCurrentMemberID` commits, so `CurrentMemberID` still points to the outgoing member.

Note: `h.db.TeamGet(ctx, id)` is available on `dbhandler.DBHandler` (backed by Redis cache) — no new dependency.

- [ ] **Step 1: Add the three helpers to `event.go`**

Add the following functions after the `deliveryStatusUpdateSleep` variable declaration in `bin-ai-manager/pkg/messagehandler/event.go`:

```go
// resolveActiveAIIDFromAIcall returns the active AI UUID from an already-fetched AIcall.
// For AssistanceTypeAI it is ac.AssistanceID directly.
// For AssistanceTypeTeam it looks up the team and walks Members to find CurrentMemberID.
// Returns uuid.Nil on any error (non-blocking: logs Warnf).
func (h *messageHandler) resolveActiveAIIDFromAIcall(ctx context.Context, ac *aicall.AIcall) uuid.UUID {
    switch ac.AssistanceType {
    case aicall.AssistanceTypeAI:
        return ac.AssistanceID

    case aicall.AssistanceTypeTeam:
        t, err := h.db.TeamGet(ctx, ac.AssistanceID)
        if err != nil {
            logrus.Warnf("resolveActiveAIIDFromAIcall: could not get team. team_id: %s, err: %v", ac.AssistanceID, err)
            return uuid.Nil
        }
        for _, m := range t.Members {
            if m.ID == ac.CurrentMemberID {
                return m.AIID
            }
        }
        logrus.Warnf("resolveActiveAIIDFromAIcall: CurrentMemberID not found in team. team_id: %s, member_id: %s", ac.AssistanceID, ac.CurrentMemberID)
        return uuid.Nil

    default:
        return uuid.Nil
    }
}

// resolveActiveAIID fetches the AIcall by ID, then delegates to resolveActiveAIIDFromAIcall.
// Use this at call sites that only have the aicall UUID.
// Returns uuid.Nil on any error (non-blocking: logs Warnf).
func (h *messageHandler) resolveActiveAIID(ctx context.Context, aicallID uuid.UUID) uuid.UUID {
    ac, err := h.reqHandler.AIV1AIcallGet(ctx, aicallID)
    if err != nil {
        logrus.Warnf("resolveActiveAIID: could not get aicall. aicall_id: %s, err: %v", aicallID, err)
        return uuid.Nil
    }
    return h.resolveActiveAIIDFromAIcall(ctx, ac)
}

// resolveTeamMemberAIID resolves the active AI UUID for a specific team member,
// independent of ac.CurrentMemberID. Used by EventPMTeamMemberSwitched where
// the notification message is created before UpdateCurrentMemberID commits.
// Returns uuid.Nil on any error (non-blocking: logs Warnf).
func (h *messageHandler) resolveTeamMemberAIID(ctx context.Context, aicallID, memberID uuid.UUID) uuid.UUID {
    ac, err := h.reqHandler.AIV1AIcallGet(ctx, aicallID)
    if err != nil {
        logrus.Warnf("resolveTeamMemberAIID: could not get aicall. aicall_id: %s, err: %v", aicallID, err)
        return uuid.Nil
    }
    if ac.AssistanceType != aicall.AssistanceTypeTeam {
        return uuid.Nil
    }
    t, err := h.db.TeamGet(ctx, ac.AssistanceID)
    if err != nil {
        logrus.Warnf("resolveTeamMemberAIID: could not get team. team_id: %s, err: %v", ac.AssistanceID, err)
        return uuid.Nil
    }
    for _, m := range t.Members {
        if m.ID == memberID {
            return m.AIID
        }
    }
    logrus.Warnf("resolveTeamMemberAIID: memberID not found in team. team_id: %s, member_id: %s", ac.AssistanceID, memberID)
    return uuid.Nil
}
```

- [ ] **Step 2: Write failing tests for the three helpers**

In `bin-ai-manager/pkg/messagehandler/event_test.go`, add the following test functions. These tests rely on `mockReq` for `AIV1AIcallGet` and `mockDB` for `TeamGet`.

```go
func TestResolveActiveAIIDFromAIcall_AI(t *testing.T) {
    aiID := uuid.Must(uuid.NewV4())
    ac := &aicall.AIcall{
        AssistanceType: aicall.AssistanceTypeAI,
        AssistanceID:   aiID,
    }
    h := &messageHandler{}
    got := h.resolveActiveAIIDFromAIcall(context.Background(), ac)
    if got != aiID {
        t.Fatalf("expected %s, got %s", aiID, got)
    }
}

func TestResolveActiveAIIDFromAIcall_Team(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    teamID  := uuid.Must(uuid.NewV4())
    memberID := uuid.Must(uuid.NewV4())
    aiID    := uuid.Must(uuid.NewV4())

    ac := &aicall.AIcall{
        AssistanceType:  aicall.AssistanceTypeTeam,
        AssistanceID:    teamID,
        CurrentMemberID: memberID,
    }

    mockDB := dbhandler.NewMockDBHandler(ctrl)
    mockDB.EXPECT().TeamGet(gomock.Any(), teamID).Return(&team.Team{
        Members: []team.Member{{ID: memberID, AIID: aiID}},
    }, nil)

    h := &messageHandler{db: mockDB}
    got := h.resolveActiveAIIDFromAIcall(context.Background(), ac)
    if got != aiID {
        t.Fatalf("expected %s, got %s", aiID, got)
    }
}

func TestResolveActiveAIID_delegatesToFromAIcall(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    aicallID := uuid.Must(uuid.NewV4())
    aiID     := uuid.Must(uuid.NewV4())
    ac := &aicall.AIcall{
        AssistanceType: aicall.AssistanceTypeAI,
        AssistanceID:   aiID,
    }

    mockReq := requesthandler.NewMockRequestHandler(ctrl)
    mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil)

    h := &messageHandler{reqHandler: mockReq}
    got := h.resolveActiveAIID(context.Background(), aicallID)
    if got != aiID {
        t.Fatalf("expected %s, got %s", aiID, got)
    }
}

func TestResolveTeamMemberAIID(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    aicallID := uuid.Must(uuid.NewV4())
    teamID   := uuid.Must(uuid.NewV4())
    memberID := uuid.Must(uuid.NewV4())
    aiID     := uuid.Must(uuid.NewV4())

    ac := &aicall.AIcall{
        AssistanceType: aicall.AssistanceTypeTeam,
        AssistanceID:   teamID,
    }

    mockReq := requesthandler.NewMockRequestHandler(ctrl)
    mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil)

    mockDB := dbhandler.NewMockDBHandler(ctrl)
    mockDB.EXPECT().TeamGet(gomock.Any(), teamID).Return(&team.Team{
        Members: []team.Member{{ID: memberID, AIID: aiID}},
    }, nil)

    h := &messageHandler{db: mockDB, reqHandler: mockReq}
    got := h.resolveTeamMemberAIID(context.Background(), aicallID, memberID)
    if got != aiID {
        t.Fatalf("expected %s, got %s", aiID, got)
    }
}
```

Note: the import block in `event_test.go` must include `"monorepo/bin-ai-manager/models/team"`.

- [ ] **Step 3: Run failing tests**

```bash
cd bin-ai-manager && go test ./pkg/messagehandler/... -v -run "TestResolveActiveAIID|TestResolveTeamMemberAIID"
```

Expected: FAIL — helpers not yet defined.

- [ ] **Step 4: Run tests after adding helpers**

```bash
cd bin-ai-manager && go test ./pkg/messagehandler/... -v -run "TestResolveActiveAIID|TestResolveTeamMemberAIID"
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add bin-ai-manager/pkg/messagehandler/event.go bin-ai-manager/pkg/messagehandler/event_test.go
git commit -m "NOJIRA-Add-active-ai-id-to-aimessage

- bin-ai-manager: Add resolveActiveAIIDFromAIcall, resolveActiveAIID, resolveTeamMemberAIID helpers"
```

---

## Task 5: Wire `active_ai_id` in `messagehandler/event.go` handlers

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/event.go`
- Modify: `bin-ai-manager/pkg/messagehandler/event_test.go`

Wire each handler one at a time. The table below shows the exact change for each.

| Handler | Resolution call | Notes |
|---|---|---|
| `EventPMMessageUserTranscription` | `resolveActiveAIID(ctx, evt.PipecatcallReferenceID)` | |
| `EventPMMessageBotLLM` — non-AICall guard (line ~73) | `uuid.Nil` (pass no option) | Reference ID is not an aicall |
| `EventPMMessageBotLLM` — voice/task branch (after ac fetched) | `resolveActiveAIIDFromAIcall(ctx, ac)` | `ac` already in scope |
| `EventPMMessageBotLLM` — conversation path (pending create, line ~114) | `resolveActiveAIIDFromAIcall(ctx, ac)` | `ac` already in scope |
| `EventPMMessageBotLLMIntermediate` | `resolveActiveAIID(ctx, evt.PipecatcallReferenceID)` | Set on `webhookMsg.ActiveAIID` directly — no DB write |
| `EventPMMessageUserLLM` | `resolveActiveAIID(ctx, evt.PipecatcallReferenceID)` | |
| `EventPMTeamMemberSwitched` | `resolveTeamMemberAIID(ctx, evt.PipecatcallReferenceID, evt.ToMember.ID)` | Cannot use CurrentMemberID — committed after this handler runs |
| `EventPMPipecatcallTerminated` | `resolveActiveAIIDFromAIcall(ctx, ac)` | `ac` already fetched |

- [ ] **Step 1: Wire `EventPMMessageUserTranscription`**

Change the `h.Create` call from:

```go
tmp, err := h.Create(ctx, uuid.Nil, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionOutgoing, message.RoleUser, evt.Text, nil, "")
```

To:

```go
activeAIID := h.resolveActiveAIID(ctx, evt.PipecatcallReferenceID)
tmp, err := h.Create(ctx, uuid.Nil, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionOutgoing, message.RoleUser, evt.Text, nil, "",
    WithActiveAIID(activeAIID))
```

- [ ] **Step 2: Wire `EventPMMessageBotLLM` — non-AICall guard path (line ~73)**

The non-AICall early-return path (reference type is not AICall) cannot resolve the AI ID from the pipecat reference. Pass no `WithActiveAIID` option (defaults to `uuid.Nil`). No change needed for this path — leave it as-is.

- [ ] **Step 3: Wire `EventPMMessageBotLLM` — voice/task branch (after `ac` fetch)**

Change the create call in the `ac.ReferenceType != aicall.ReferenceTypeConversation` block:

```go
// add before the Create call:
voiceActiveAIID := h.resolveActiveAIIDFromAIcall(ctx, ac)
tmp, errCreate := h.Create(ctx, evt.ID, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID,
    message.DirectionIncoming, message.RoleAssistant, evt.Text, nil, "",
    WithActiveAIID(voiceActiveAIID))
```

- [ ] **Step 4: Wire `EventPMMessageBotLLM` — conversation path (pending create, line ~114)**

```go
// add before the Create call (ac is still in scope from the fetch at line 83):
convActiveAIID := h.resolveActiveAIIDFromAIcall(ctx, ac)
tmp, err := h.Create(ctx, evt.ID, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID,
    message.DirectionIncoming, message.RoleAssistant, evt.Text, nil, "",
    WithPipecatcallID(evt.PipecatcallID),
    WithDeliveryStatus(message.DeliveryStatusPending),
    WithActiveAIID(convActiveAIID))
```

- [ ] **Step 5: Wire `EventPMMessageBotLLMIntermediate`**

The intermediate event does not write to DB — it publishes directly via `webhookMsg`. Set `ActiveAIID` on the struct:

```go
activeAIID := h.resolveActiveAIID(ctx, evt.PipecatcallReferenceID)
webhookMsg := &message.IntermediateWebhookMessage{
    Identity: identity.Identity{
        ID:         evt.ID,
        CustomerID: evt.CustomerID,
    },
    AIcallID:     evt.PipecatcallReferenceID,
    ActiveflowID: evt.ActiveflowID,
    ActiveAIID:   activeAIID,
    Role:         message.RoleAssistant,
    Content:      evt.Text,
    Direction:    message.DirectionIncoming,
    Sequence:     evt.Sequence,
}
```

- [ ] **Step 6: Wire `EventPMMessageUserLLM`**

```go
activeAIID := h.resolveActiveAIID(ctx, evt.PipecatcallReferenceID)
tmp, err := h.Create(ctx, uuid.Nil, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionOutgoing, message.RoleUser, evt.Text, nil, "",
    WithActiveAIID(activeAIID))
```

- [ ] **Step 7: Wire `EventPMTeamMemberSwitched`**

```go
// Use evt.ToMember.ID explicitly — CurrentMemberID is still the outgoing member when this fires
activeAIID := h.resolveTeamMemberAIID(ctx, evt.PipecatcallReferenceID, evt.ToMember.ID)
tmp, err := h.Create(ctx, uuid.Nil, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionOutgoing, message.RoleNotification, string(contentBytes), nil, "",
    WithActiveAIID(activeAIID))
```

- [ ] **Step 8: Wire `EventPMPipecatcallTerminated`**

The backstop handler already has `ac` in scope from its `AIV1AIcallGet` call. Use `resolveActiveAIIDFromAIcall` to avoid a redundant RPC:

```go
backstopActiveAIID := h.resolveActiveAIIDFromAIcall(ctx, ac)
msg, err := h.Create(ctx, uuid.Nil, ac.CustomerID, ac.ID, ac.ActiveflowID,
    message.DirectionIncoming, message.RoleAssistant, backstopReplyText, nil, "",
    WithPipecatcallID(evt.ID),
    WithDeliveryStatus(message.DeliveryStatusDelivered),
    WithActiveAIID(backstopActiveAIID))
```

- [ ] **Step 9: Run all messagehandler tests**

```bash
cd bin-ai-manager && go test ./pkg/messagehandler/... -v
```

Expected: all PASS.

- [ ] **Step 10: Commit**

```bash
git add bin-ai-manager/pkg/messagehandler/event.go bin-ai-manager/pkg/messagehandler/event_test.go
git commit -m "NOJIRA-Add-active-ai-id-to-aimessage

- bin-ai-manager: Wire WithActiveAIID in all 7 messagehandler event handlers"
```

---

## Task 6: Wire `active_ai_id` in `aicallhandler/`

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/send.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/tool.go`

Note: `aicallhandler` does not have access to the `resolveActiveAIID*` helpers — those are private to `messagehandler`. Instead, use `WithActiveAIID(...)` directly with values already in scope, or call `h.reqHandler.AIV1AIcallGet` for call sites where only the aicall ID is available. Since resolving from aicall is a local concern, the pattern here is: when `*ai.AI` is in scope, use `a.ID`; when `*aicall.AIcall` is in scope, inline the same logic.

However, per the approved design, the cleaner approach for `aicallhandler` sites is:

- `start.go:475` — `a *ai.AI` is in scope → `WithActiveAIID(a.ID)`
- `start.go:230` — `res *aicall.AIcall` is in scope → inline `resolveActiveAIIDForAIcall(res)`
- `send.go:47` (SendReferenceTypeCall) — `c *aicall.AIcall` in scope → inline resolve
- `send.go:70` (SendReferenceTypeOthers) — `c *aicall.AIcall` in scope → inline resolve
- `tool.go:40` — `c *aicall.AIcall` in scope → inline resolve
- `tool.go:103` — `c *aicall.AIcall` in scope → inline resolve

Because these are in `aicallhandler` (not `messagehandler`), add a package-local helper `resolveActiveAIIDFromAIcall` in a new file `helpers.go` (or add to the existing `helpers.go` if it exists). Check the file first.

- [ ] **Step 1: Check `bin-ai-manager/pkg/aicallhandler/helpers.go`**

```bash
head -50 bin-ai-manager/pkg/aicallhandler/helpers.go
```

Read the file. It contains existing helpers. Add the new helper to it rather than creating a new file.

- [ ] **Step 2: Add `resolveActiveAIIDFromAIcall` helper to `aicallhandler/helpers.go`**

```go
// resolveActiveAIIDFromAIcall returns the active AI UUID for the given AIcall.
// For AssistanceTypeAI it returns ac.AssistanceID.
// For AssistanceTypeTeam it walks the team members to find CurrentMemberID's AIID.
// Returns uuid.Nil on any error (non-blocking: logs Warnf).
func (h *aicallHandler) resolveActiveAIIDFromAIcall(ctx context.Context, ac *aicall.AIcall) uuid.UUID {
    switch ac.AssistanceType {
    case aicall.AssistanceTypeAI:
        return ac.AssistanceID
    case aicall.AssistanceTypeTeam:
        t, err := h.teamHandler.Get(ctx, ac.AssistanceID)
        if err != nil {
            logrus.Warnf("resolveActiveAIIDFromAIcall: could not get team. team_id: %s, err: %v", ac.AssistanceID, err)
            return uuid.Nil
        }
        for _, m := range t.Members {
            if m.ID == ac.CurrentMemberID {
                return m.AIID
            }
        }
        logrus.Warnf("resolveActiveAIIDFromAIcall: CurrentMemberID not found in team. team_id: %s, member_id: %s", ac.AssistanceID, ac.CurrentMemberID)
        return uuid.Nil
    default:
        return uuid.Nil
    }
}
```

Note: `h.teamHandler.Get` is available on `aicallHandler` (it fetches from Redis cache via the team handler). Verify by checking `bin-ai-manager/pkg/aicallhandler/main.go` for the `teamHandler` field.

- [ ] **Step 3: Wire `start.go:475` — init messages (system role)**

`startInitMessages(ctx, a *ai.AI, c *aicall.AIcall, isTask bool)` has `a *ai.AI` in scope. Change the create call:

```go
for _, msg := range messages {
    tmp, err := h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, c.ID, c.ActiveflowID, message.DirectionOutgoing, message.RoleSystem, msg, nil, "",
        messagehandler.WithActiveAIID(a.ID))
    if err != nil {
        return errors.Wrapf(err, "could not create the init message to the ai. aicall_id: %s", c.ID)
    }
    log.WithField("message", tmp).Debugf("Created the init message to the ai. aicall_id: %s", c.ID)
}
```

- [ ] **Step 4: Wire `start.go:230` — conversation start user message**

In `startReferenceTypeConversation`, `res *aicall.AIcall` is in scope at line 230. Use `resolveActiveAIIDFromAIcall`:

```go
convUserActiveAIID := h.resolveActiveAIIDFromAIcall(ctx, res)
tmp, err := h.messageHandler.Create(ctx, uuid.Nil, res.CustomerID, res.ID, res.ActiveflowID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "",
    messagehandler.WithActiveAIID(convUserActiveAIID))
```

- [ ] **Step 5: Wire `send.go:47` — `SendReferenceTypeCall` user message**

`c *aicall.AIcall` is in scope:

```go
sendCallActiveAIID := h.resolveActiveAIIDFromAIcall(ctx, c)
res, err := h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, c.ID, c.ActiveflowID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "",
    messagehandler.WithActiveAIID(sendCallActiveAIID))
```

- [ ] **Step 6: Wire `send.go:70` — `SendReferenceTypeOthers` user message**

`c *aicall.AIcall` is in scope at line 70:

```go
sendOtherActiveAIID := h.resolveActiveAIIDFromAIcall(ctx, c)
res, errTerminate := h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, aicallID, c.ActiveflowID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "",
    messagehandler.WithActiveAIID(sendOtherActiveAIID))
```

- [ ] **Step 7: Wire `tool.go:40` — assistant message wrapping tool call**

`c *aicall.AIcall` is in scope:

```go
toolCallActiveAIID := h.resolveActiveAIIDFromAIcall(ctx, c)
tmp, errCreate := h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, c.ID, c.ActiveflowID, message.DirectionIncoming, message.RoleAssistant, "", []message.ToolCall{*tool}, "",
    messagehandler.WithActiveAIID(toolCallActiveAIID))
```

- [ ] **Step 8: Wire `tool.go:103` — tool result message in `toolCreateResultMessage`**

`c *aicall.AIcall` is in scope:

```go
toolResultActiveAIID := h.resolveActiveAIIDFromAIcall(ctx, c)
tmp, err := h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, c.ID, c.ActiveflowID, message.DirectionOutgoing, message.RoleTool, string(content), nil, tool.ID,
    messagehandler.WithActiveAIID(toolResultActiveAIID))
```

- [ ] **Step 9: Run all aicallhandler tests**

```bash
cd bin-ai-manager && go test ./pkg/aicallhandler/... -v
```

Expected: all PASS.

- [ ] **Step 10: Commit**

```bash
git add bin-ai-manager/pkg/aicallhandler/helpers.go \
        bin-ai-manager/pkg/aicallhandler/start.go \
        bin-ai-manager/pkg/aicallhandler/send.go \
        bin-ai-manager/pkg/aicallhandler/tool.go
git commit -m "NOJIRA-Add-active-ai-id-to-aimessage

- bin-ai-manager: Wire WithActiveAIID at all 6 aicallhandler Create call sites"
```

---

## Task 7: Alembic migration — `bin-dbscheme-manager`

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<generated-rev>.py`

- [ ] **Step 1: Generate the migration file**

```bash
cd bin-dbscheme-manager/bin-manager
alembic -c alembic.ini revision -m "ai_messages_add_column_active_ai_id"
```

This creates a new file in `main/versions/` with a unique revision ID and the correct `down_revision` chain. Note the filename — it will be something like `<hex>_ai_messages_add_column_active_ai_id.py`.

- [ ] **Step 2: Edit the generated file**

Open the newly generated file and fill in the `upgrade()` and `downgrade()` functions:

```python
from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import mysql


def upgrade():
    op.add_column('ai_messages',
        sa.Column('active_ai_id', mysql.BINARY(16), nullable=False,
                  server_default=sa.text("0x00000000000000000000000000000000")))


def downgrade():
    op.drop_column('ai_messages', 'active_ai_id')
```

- [ ] **Step 3: Verify migration chain**

```bash
cd bin-dbscheme-manager/bin-manager
alembic -c alembic.ini heads
```

Expected: exactly one head (the newly generated revision).

- [ ] **Step 4: Commit**

```bash
git add bin-dbscheme-manager/bin-manager/main/versions/
git commit -m "NOJIRA-Add-active-ai-id-to-aimessage

- bin-dbscheme-manager: Add Alembic migration for ai_messages.active_ai_id BINARY(16) column"
```

---

## Task 8: Service domain docs — `bin-ai-manager/docs/domain.md`

**Files:**
- Modify: `bin-ai-manager/docs/domain.md`

This suppresses the monorepo `check-service-docs.sh` hook warning that fires when `models/.../*.go` changes without a matching `docs/domain.md` update.

- [ ] **Step 1: Update the Message section in `domain.md`**

Find the `### Message` section and add the `active_ai_id` field:

```markdown
### Message
Individual message within an AIcall conversation. Persisted in MySQL for context replay and summaries.

- `role`: `system` | `user` | `assistant` | `tool` | `notification`
- `direction`: `inbound` | `outbound`
- `active_ai_id`: UUID of the AI config active when the message was created. For team AIcalls this is the current member's backing AI UUID; for direct AI calls it is `assistance_id`. Zero UUID (`00000000-...`) for historical rows created before this field was added.
- Supports tool call payloads for function-calling workflows
```

- [ ] **Step 2: Commit**

```bash
git add bin-ai-manager/docs/domain.md
git commit -m "NOJIRA-Add-active-ai-id-to-aimessage

- bin-ai-manager: Update docs/domain.md to document active_ai_id on Message"
```

---

## Task 9: RST user-facing docs + rebuild HTML

**Files:**
- Modify: `bin-api-manager/docsdev/source/ai_struct_message.rst`
- Rebuild: `bin-api-manager/docsdev/build/`

- [ ] **Step 1: Add `active_ai_id` to the RST struct docs**

In `bin-api-manager/docsdev/source/ai_struct_message.rst`, add `active_ai_id` to the JSON example block and the field list.

JSON example block — add after `"activeflow_id"`:

```
    {
        "id": "<string>",
        "customer_id": "<string>",
        "aicall_id": "<string>",
        "activeflow_id": "<string>",
        "active_ai_id": "<string>",
        "role": "<string>",
        "content": "<string>",
        "direction": "<string>",
        "tool_calls": [],
        "tool_call_id": "<string>",
        "tm_create": "<string>"
    }
```

Field description — add after the `activeflow_id` bullet:

```
* ``active_ai_id`` (UUID): The UUID of the AI configuration that was active when this message was created. For team AIcalls this is the backing AI of the active team member; for direct AI calls it is the AI UUID directly. Set to ``00000000-0000-0000-0000-000000000000`` for messages created before this field was introduced.
```

Also update the Example section to include a realistic value:

```
        "active_ai_id": "d4e5f6a7-b8c9-0123-deff-234567890123",
```

- [ ] **Step 2: Clean rebuild HTML**

```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
```

Expected: build completes with no errors.

- [ ] **Step 3: Force-add built HTML**

```bash
cd bin-api-manager/docsdev && git add -f build/
```

- [ ] **Step 4: Commit RST source and built HTML together**

```bash
git add bin-api-manager/docsdev/source/ai_struct_message.rst
git commit -m "NOJIRA-Add-active-ai-id-to-aimessage

- bin-api-manager: Document active_ai_id in ai_struct_message.rst and rebuild HTML"
```

---

## Task 10: Full verification workflow

**Files:** none (verification only)

- [ ] **Step 1: Run full verification in `bin-ai-manager`**

```bash
cd bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: all steps exit 0.

Note: `go generate ./...` regenerates mocks. The new helpers (`resolveActiveAIIDFromAIcall`, `resolveActiveAIID`, `resolveTeamMemberAIID`) are unexported methods and `WithActiveAIID` does not change the `MessageHandler` interface, so no mock files should change. If the mock for `MessageHandler` is regenerated, commit it.

- [ ] **Step 2: Fix any lint issues**

Common issues to watch for:
- `golangci-lint` may flag `uuid.Nil` comparison warnings. If so, use `== uuid.Nil` (the idiomatic form already used elsewhere in this codebase).
- Unused import in `event_test.go` if the `team` package import was added but not used.

- [ ] **Step 3: Commit any verification-generated diffs**

If `go generate ./...` changed mock files:

```bash
git add bin-ai-manager/pkg/messagehandler/mock_main.go
git commit -m "NOJIRA-Add-active-ai-id-to-aimessage

- bin-ai-manager: Regenerate mocks after go generate"
```

---

## Task 11: Final check — conflict-free PR creation

- [ ] **Step 1: Fetch latest main and check for conflicts**

```bash
# Run from inside the worktree
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

Expected: no conflicts, no new commits on main that touch the same files.

- [ ] **Step 2: Create the PR**

```bash
gh pr create \
  --title "NOJIRA-Add-active-ai-id-to-aimessage" \
  --body "$(cat <<'EOF'
Add active_ai_id to ai_messages to identify which AI config was active at message creation time.

- bin-ai-manager: Add ActiveAIID field to message.Message (db tag: active_ai_id,uuid)
- bin-ai-manager: Add FieldActiveAIID constant to models/message/field.go
- bin-ai-manager: Add ActiveAIID to WebhookMessage, IntermediateWebhookMessage, ConvertWebhookMessage
- bin-ai-manager: Add WithActiveAIID CreateOption and wire in messagehandler.Create
- bin-ai-manager: Add resolveActiveAIIDFromAIcall, resolveActiveAIID, resolveTeamMemberAIID helpers
- bin-ai-manager: Wire active_ai_id at all 7 event handlers in messagehandler/event.go
- bin-ai-manager: Wire active_ai_id at all 6 Create call sites in aicallhandler/start.go|send.go|tool.go
- bin-ai-manager: Update docs/domain.md to document active_ai_id
- bin-dbscheme-manager: Add Alembic migration for ai_messages.active_ai_id BINARY(16) column
- bin-api-manager: Document active_ai_id in ai_struct_message.rst, rebuild HTML
EOF
)"
```

---

## Invariants to verify before marking done

- [ ] `active_ai_id` is always the UUID of an `ai` resource, never a team UUID.
- [ ] `uuid.Nil` (`00000000-0000-0000-0000-000000000000`) is the zero value — it serializes in JSON (Go's `omitempty` does NOT suppress `[16]byte`). Consumers should treat the zero UUID as "AI not identifiable."
- [ ] The `EventPMTeamMemberSwitched` handler uses `evt.ToMember.ID` (not `CurrentMemberID`) to get the incoming member's AI UUID.
- [ ] All three resolution helpers return `uuid.Nil` (never block/error) — they log at `Warnf`.
- [ ] `FieldActiveAIID` constant exists for completeness; no changes to `MessageList` filters.
- [ ] No changes to `bin-pipecat-manager`, `bin-conversation-manager`, or any other service.
