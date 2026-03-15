# Add activeflow_id to AI Message — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `activeflow_id` field to the AI message model so AI messages can be included in timeline-manager's aggregated events endpoint.

**Architecture:** Denormalize `activeflow_id` from the AIcall onto the AI message. For Pipecat-originated messages, propagate through `Session` → event structs → message creation. For RPC-originated messages, read from the AIcall record directly.

**Tech Stack:** Go, MySQL (Alembic migrations), OpenAPI 3.0, oapi-codegen

---

### Task 1: Alembic Migration — add activeflow_id column to ai_messages

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<hash>_ai_messages_add_column_activeflow_id.py`

**Step 1: Create the migration file**

```bash
cd bin-dbscheme-manager/bin-manager
alembic -c alembic.ini revision -m "ai_messages add column activeflow_id"
```

**Step 2: Edit the generated migration**

Replace the `upgrade()` and `downgrade()` functions:

```python
def upgrade():
    op.execute("""
        ALTER TABLE ai_messages
        ADD COLUMN activeflow_id binary(16) AFTER aicall_id
    """)
    op.execute("""
        CREATE INDEX idx_ai_messages_activeflow_id ON ai_messages(activeflow_id)
    """)


def downgrade():
    op.execute("""
        DROP INDEX idx_ai_messages_activeflow_id ON ai_messages
    """)
    op.execute("""
        ALTER TABLE ai_messages
        DROP COLUMN activeflow_id
    """)
```

Set `down_revision = 'fd2a3b4c5d6e'` (current head).

**Step 3: Commit**

```bash
git add bin-dbscheme-manager/
git commit -m "NOJIRA-Add-activeflow-id-to-ai-message

- bin-dbscheme-manager: Add Alembic migration for activeflow_id column on ai_messages table"
```

---

### Task 2: bin-pipecat-manager — add ActiveflowID to models

**Files:**
- Modify: `bin-pipecat-manager/models/pipecatcall/session.go`
- Modify: `bin-pipecat-manager/models/message/main.go`
- Modify: `bin-pipecat-manager/models/message/member_switched.go`

**Step 1: Add ActiveflowID to Session struct**

In `bin-pipecat-manager/models/pipecatcall/session.go`, add after line 17 (`PipecatcallReferenceID`):

```go
	ActiveflowID             uuid.UUID     `json:"activeflow_id,omitempty"`         // copied from pipecatcall
```

**Step 2: Add ActiveflowID to pmmessage.Message struct**

In `bin-pipecat-manager/models/message/main.go`, add after `PipecatcallReferenceID` field (line 17):

```go
	ActiveflowID             uuid.UUID                 `json:"activeflow_id,omitempty"`
```

**Step 3: Add ActiveflowID to MemberSwitchedEvent struct**

In `bin-pipecat-manager/models/message/member_switched.go`, add after `PipecatcallReferenceID` field (line 15):

```go
	ActiveflowID             uuid.UUID                 `json:"activeflow_id,omitempty"`
```

**Step 4: Verify it compiles**

```bash
cd bin-pipecat-manager && go build ./cmd/...
```

---

### Task 3: bin-pipecat-manager — populate ActiveflowID in handlers

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/session.go`
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go`

**Step 1: Populate ActiveflowID in SessionCreate()**

In `bin-pipecat-manager/pkg/pipecatcallhandler/session.go`, add `ActiveflowID` to the Session struct literal (after `PipecatcallReferenceID: pc.ReferenceID,` on line 35):

```go
		ActiveflowID:             pc.ActiveflowID,
```

**Step 2: Populate ActiveflowID in newMessageEvent()**

In `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go`, add to the `message.Message` literal in `newMessageEvent()` (after `PipecatcallReferenceID` on line 443):

```go
		ActiveflowID:             se.ActiveflowID,
```

**Step 3: Populate ActiveflowID in RunnerHTTPTeamMemberSwitched()**

In `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go`, add to the `message.MemberSwitchedEvent` literal (after `PipecatcallReferenceID` on line 420):

```go
		ActiveflowID:             pc.ActiveflowID,
```

**Step 4: Update tests**

In `bin-pipecat-manager/pkg/pipecatcallhandler/session_test.go`, add `ActiveflowID` to the test `Pipecatcall` input and verify it appears on the returned session.

In any test constructing `Session{}` or `Message{}` structs, add `ActiveflowID` field if the test validates field propagation.

**Step 5: Run verification**

```bash
cd bin-pipecat-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 6: Commit**

```bash
git add bin-pipecat-manager/
git commit -m "NOJIRA-Add-activeflow-id-to-ai-message

- bin-pipecat-manager: Add ActiveflowID field to Session, Message, and MemberSwitchedEvent models
- bin-pipecat-manager: Populate ActiveflowID in SessionCreate, newMessageEvent, and RunnerHTTPTeamMemberSwitched"
```

---

### Task 4: bin-ai-manager — add ActiveflowID to message models

**Files:**
- Modify: `bin-ai-manager/models/message/main.go`
- Modify: `bin-ai-manager/models/message/webhook.go`
- Modify: `bin-ai-manager/models/message/field.go`
- Modify: `bin-ai-manager/models/message/filters.go`

**Step 1: Add ActiveflowID to Message struct**

In `bin-ai-manager/models/message/main.go`, add after `AIcallID` field (line 15):

```go
	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty" db:"activeflow_id,uuid"`
```

**Step 2: Add ActiveflowID to WebhookMessage struct**

In `bin-ai-manager/models/message/webhook.go`, add after `AIcallID` field (line 15):

```go
	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`
```

Update `ConvertWebhookMessage()` to include ActiveflowID (add after `AIcallID: h.AIcallID,` on line 31):

```go
		ActiveflowID: h.ActiveflowID,
```

**Step 3: Add FieldActiveflowID to field.go**

In `bin-ai-manager/models/message/field.go`, add after `FieldAIcallID` (line 11):

```go
	FieldActiveflowID Field = "activeflow_id"
```

**Step 4: Add ActiveflowID to FieldStruct in filters.go**

In `bin-ai-manager/models/message/filters.go`, add after `AIcallID` (line 9):

```go
	ActiveflowID uuid.UUID `filter:"activeflow_id"`
```

**Step 5: Verify it compiles**

```bash
cd bin-ai-manager && go build ./cmd/...
```

---

### Task 5: bin-ai-manager — update Create() interface and implementation

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/main.go`
- Modify: `bin-ai-manager/pkg/messagehandler/db.go`

**Step 1: Update the MessageHandler interface**

In `bin-ai-manager/pkg/messagehandler/main.go`, update the `Create` method signature (lines 21-30). Add `activeflowID uuid.UUID` parameter after `aicallID uuid.UUID`:

```go
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		aicallID uuid.UUID,
		activeflowID uuid.UUID,
		direction message.Direction,
		role message.Role,
		content string,
		toolCalls []message.ToolCall,
		toolCallID string,
	) (*message.Message, error)
```

**Step 2: Update the Create() implementation**

In `bin-ai-manager/pkg/messagehandler/db.go`, update function signature (lines 12-21) to add `activeflowID uuid.UUID` parameter after `aicallID uuid.UUID`:

```go
func (h *messageHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	aicallID uuid.UUID,
	activeflowID uuid.UUID,
	direction message.Direction,
	role message.Role,
	content string,
	toolCalls []message.ToolCall,
	toolCallID string,
) (*message.Message, error) {
```

Add `ActiveflowID: activeflowID,` to the message struct literal, after `AIcallID: aicallID,` (line 34):

```go
		ActiveflowID: activeflowID,
```

**Step 3: Verify it compiles (will fail until callers are updated — that's expected)**

```bash
cd bin-ai-manager && go build ./cmd/... 2>&1 | head -20
# Expect: compilation errors from callers not yet updated
```

---

### Task 6: bin-ai-manager — update all Create() callers

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/event.go` (4 call sites)
- Modify: `bin-ai-manager/pkg/aicallhandler/send.go` (2 call sites)
- Modify: `bin-ai-manager/pkg/aicallhandler/tool.go` (2 call sites)
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go` (2 call sites)

**Step 1: Update event.go — 4 callers**

Line 23 (`EventPMMessageUserTranscription`): Change:
```go
	tmp, err := h.Create(ctx, evt.CustomerID, evt.PipecatcallReferenceID, message.DirectionOutgoing, message.RoleUser, evt.Text, nil, "")
```
To:
```go
	tmp, err := h.Create(ctx, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionOutgoing, message.RoleUser, evt.Text, nil, "")
```

Line 42 (`EventPMMessageBotLLM`): Change:
```go
	tmp, err := h.Create(ctx, evt.CustomerID, evt.PipecatcallReferenceID, message.DirectionIncoming, message.RoleAssistant, evt.Text, nil, "")
```
To:
```go
	tmp, err := h.Create(ctx, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionIncoming, message.RoleAssistant, evt.Text, nil, "")
```

Line 56 (`EventPMMessageUserLLM`): Change:
```go
	tmp, err := h.Create(ctx, evt.CustomerID, evt.PipecatcallReferenceID, message.DirectionOutgoing, message.RoleUser, evt.Text, nil, "")
```
To:
```go
	tmp, err := h.Create(ctx, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionOutgoing, message.RoleUser, evt.Text, nil, "")
```

Line 101 (`EventPMTeamMemberSwitched`): Change:
```go
	tmp, err := h.Create(ctx, evt.CustomerID, evt.PipecatcallReferenceID, message.DirectionOutgoing, message.RoleNotification, string(contentBytes), nil, "")
```
To:
```go
	tmp, err := h.Create(ctx, evt.CustomerID, evt.PipecatcallReferenceID, evt.ActiveflowID, message.DirectionOutgoing, message.RoleNotification, string(contentBytes), nil, "")
```

**Step 2: Update send.go — 2 callers**

Line 40 (`SendReferenceTypeCall`): Change:
```go
	res, err := h.messageHandler.Create(ctx, c.CustomerID, c.ID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "")
```
To:
```go
	res, err := h.messageHandler.Create(ctx, c.CustomerID, c.ID, c.ActiveflowID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "")
```

Line 63 (`SendReferenceTypeOthers`): Change:
```go
	res, errTerminate := h.messageHandler.Create(ctx, c.CustomerID, aicallID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "")
```
To:
```go
	res, errTerminate := h.messageHandler.Create(ctx, c.CustomerID, aicallID, c.ActiveflowID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "")
```

**Step 3: Update tool.go — 2 callers**

Line 38 (`ToolHandle`): Change:
```go
	tmp, errCreate := h.messageHandler.Create(ctx, c.CustomerID, c.ID, message.DirectionIncoming, message.RoleAssistant, "", []message.ToolCall{*tool}, "")
```
To:
```go
	tmp, errCreate := h.messageHandler.Create(ctx, c.CustomerID, c.ID, c.ActiveflowID, message.DirectionIncoming, message.RoleAssistant, "", []message.ToolCall{*tool}, "")
```

Line 100 (`toolCreateResultMessage`): Change:
```go
	tmp, err := h.messageHandler.Create(ctx, c.CustomerID, c.ID, message.DirectionOutgoing, message.RoleTool, string(content), nil, tool.ID)
```
To:
```go
	tmp, err := h.messageHandler.Create(ctx, c.CustomerID, c.ID, c.ActiveflowID, message.DirectionOutgoing, message.RoleTool, string(content), nil, tool.ID)
```

**Step 4: Update start.go — 2 callers**

Line 178: Change:
```go
	tmp, err := h.messageHandler.Create(ctx, res.CustomerID, res.ID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "")
```
To:
```go
	tmp, err := h.messageHandler.Create(ctx, res.CustomerID, res.ID, res.ActiveflowID, message.DirectionOutgoing, message.RoleUser, messageText, nil, "")
```

Line 417: Change:
```go
		tmp, err := h.messageHandler.Create(ctx, c.CustomerID, c.ID, message.DirectionOutgoing, message.RoleSystem, msg, nil, "")
```
To:
```go
		tmp, err := h.messageHandler.Create(ctx, c.CustomerID, c.ID, c.ActiveflowID, message.DirectionOutgoing, message.RoleSystem, msg, nil, "")
```

**Step 5: Verify it compiles**

```bash
cd bin-ai-manager && go build ./cmd/...
```

---

### Task 7: bin-ai-manager — update tests and regenerate mocks

**Files:**
- Modify: `bin-ai-manager/pkg/messagehandler/db_test.go`
- Modify: `bin-ai-manager/pkg/messagehandler/event_test.go`
- Regenerate: `bin-ai-manager/pkg/messagehandler/mock_main.go`

**Step 1: Regenerate mocks**

```bash
cd bin-ai-manager && go generate ./...
```

This regenerates `mock_main.go` from the updated `MessageHandler` interface.

**Step 2: Update db_test.go**

Add `activeflowID uuid.UUID` field to test struct and all `h.Create()` calls.

In the test struct definition (around line 23), add:
```go
		activeflowID uuid.UUID
```

In the "have all" test case (around line 37), add:
```go
			activeflowID: uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
```

In the `expectMessage` struct (around line 61), add after `AIcallID`:
```go
				ActiveflowID: uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
```

Update the `h.Create()` call on line 114 to include `tt.activeflowID`:
```go
			res, err := h.Create(ctx, tt.customerID, tt.aicallID, tt.activeflowID, tt.direction, tt.role, tt.content, tt.toolCalls, tt.toolCallID)
```

**Step 3: Update event_test.go**

Add `ActiveflowID` to the test event structs. For each test that creates a `pmmessage.Message`, add:
```go
				ActiveflowID:             uuid.Must(uuid.NewV4()),
```

The mock expectations use `gomock.Any()` so no other changes needed.

**Step 4: Run tests**

```bash
cd bin-ai-manager && go test ./...
```

**Step 5: Run full verification**

```bash
cd bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 6: Commit**

```bash
git add bin-ai-manager/
git commit -m "NOJIRA-Add-activeflow-id-to-ai-message

- bin-ai-manager: Add ActiveflowID field to Message model, WebhookMessage, Field type, and FieldStruct
- bin-ai-manager: Add activeflowID parameter to MessageHandler.Create() interface and implementation
- bin-ai-manager: Update all 10 Create() call sites in event, send, tool, and start handlers
- bin-ai-manager: Update tests and regenerate mocks"
```

---

### Task 8: bin-openapi-manager — update AIManagerMessage schema

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml`

**Step 1: Add activeflow_id property to AIManagerMessage**

In `bin-openapi-manager/openapi/openapi.yaml`, add `activeflow_id` property to the `AIManagerMessage` schema (after `aicall_id` property, around line 2260):

```yaml
        activeflow_id:
          type: string
          format: uuid
          x-go-type: string
          description: "The unique identifier of the active flow execution. Returned from the `POST /activeflows` response."
          example: "550e8400-e29b-41d4-a716-446655440000"
```

**Step 2: Run verification for bin-openapi-manager**

```bash
cd bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Run verification for bin-api-manager**

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
git add bin-openapi-manager/ bin-api-manager/
git commit -m "NOJIRA-Add-activeflow-id-to-ai-message

- bin-openapi-manager: Add activeflow_id field to AIManagerMessage OpenAPI schema
- bin-api-manager: Regenerate server code from updated OpenAPI spec"
```

---

### Task 9: Final verification and RST docs check

**Step 1: Check RST docs for ai message struct**

Check if there is an RST doc for AI messages that needs updating:

```bash
find bin-api-manager/docsdev/source/ -name "*message*" -o -name "*ai*" | grep -i struct
```

If a struct RST file exists for AI messages, add the `activeflow_id` field to match the updated `WebhookMessage`. Then rebuild:

```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
git add -f bin-api-manager/docsdev/build/
```

**Step 2: Run final verification on all changed services**

```bash
cd bin-pipecat-manager && go test ./... && golangci-lint run -v --timeout 5m
cd ../bin-ai-manager && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Commit any RST updates if needed**

```bash
git add bin-api-manager/docsdev/
git commit -m "NOJIRA-Add-activeflow-id-to-ai-message

- bin-api-manager: Update RST struct docs for AI message activeflow_id field"
```
