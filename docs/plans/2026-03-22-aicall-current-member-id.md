# Add CurrentMemberID to AIcall Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `current_member_id` field to the AIcall model so that team-based aicalls track which team member (AI agent) is currently active, enabling correct routing for async message handling (SMS/SNS/Email).

**Architecture:** Add a UUID field to the AIcall struct, DB table, OpenAPI schema, and webhook. On creation with `AssistanceTypeTeam`, set it to `team.StartMemberID`. Provide an update method for future transition notifications.

**Tech Stack:** Go, MySQL (Alembic migration), OpenAPI YAML

---

### Task 1: DB Migration — Add `current_member_id` column

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<rev>_ai_aicalls_add_column_current_member_id.py`

**Step 1: Create the migration file**

```bash
cd bin-dbscheme-manager/bin-manager
alembic -c alembic.ini revision -m "ai_aicalls add column current_member_id"
```

**Step 2: Edit the migration file**

```python
def upgrade():
    op.execute("ALTER TABLE ai_aicalls ADD COLUMN current_member_id binary(16) AFTER pipecatcall_id")


def downgrade():
    op.execute("ALTER TABLE ai_aicalls DROP COLUMN current_member_id")
```

**Step 3: Commit**

```bash
git add bin-dbscheme-manager/bin-manager/main/versions/
git commit -m "NOJIRA-Add-aicall-current-member-id

- bin-dbscheme-manager: Add current_member_id column to ai_aicalls table"
```

**IMPORTANT:** Do NOT run `alembic upgrade`. Only create the migration file.

---

### Task 2: Model layer — Add field to AIcall struct, Field constants, and WebhookMessage

**Files:**
- Modify: `bin-ai-manager/models/aicall/main.go:32-33` (add field after PipecatcallID)
- Modify: `bin-ai-manager/models/aicall/field.go:28-29` (add constant after FieldPipecatcallID)
- Modify: `bin-ai-manager/models/aicall/webhook.go:14-44` (add field to WebhookMessage)
- Modify: `bin-ai-manager/models/aicall/webhook.go:47-79` (add field to ConvertWebhookMessage)

**Step 1: Add `CurrentMemberID` to `AIcall` struct**

In `bin-ai-manager/models/aicall/main.go`, add after the `PipecatcallID` field (line 33):

```go
	PipecatcallID   uuid.UUID `json:"pipecatcall_id,omitempty" db:"pipecatcall_id,uuid"`
	CurrentMemberID uuid.UUID `json:"current_member_id,omitempty" db:"current_member_id,uuid"`
```

**Step 2: Add `FieldCurrentMemberID` constant**

In `bin-ai-manager/models/aicall/field.go`, add after `FieldPipecatcallID` (line 28):

```go
	FieldPipecatcallID   Field = "pipecatcall_id"
	FieldCurrentMemberID Field = "current_member_id"
```

**Step 3: Add field to `WebhookMessage`**

In `bin-ai-manager/models/aicall/webhook.go`, add after `ConfbridgeID` (line 33):

```go
	ConfbridgeID uuid.UUID `json:"confbridge_id,omitempty"`

	CurrentMemberID uuid.UUID `json:"current_member_id,omitempty"`
```

**Step 4: Add field to `ConvertWebhookMessage()`**

In `bin-ai-manager/models/aicall/webhook.go`, add in the return struct after `ConfbridgeID` (line 67):

```go
		ConfbridgeID: h.ConfbridgeID,

		CurrentMemberID: h.CurrentMemberID,
```

**Step 5: Run tests**

```bash
cd bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 6: Commit**

```bash
git add bin-ai-manager/models/aicall/
git commit -m "NOJIRA-Add-aicall-current-member-id

- bin-ai-manager: Add CurrentMemberID field to AIcall model, field constants, and WebhookMessage"
```

---

### Task 3: Handler layer — Thread CurrentMemberID through resolveAI, startAIcall, and Create

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go:22-52` (resolveAI return value)
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go:54-84` (Start)
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go:87-126` (startReferenceTypeCall)
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go:129-195` (startReferenceTypeConversation)
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go:198-225` (startReferenceTypeNone)
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go:427-482` (startAIcall)
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go:484-517` (StartTask)
- Modify: `bin-ai-manager/pkg/aicallhandler/db.go:18-87` (Create)

**Step 1: Modify `resolveAI` to return start member ID**

Change the return signature from `(*ai.AI, map[string]any, error)` to `(*ai.AI, map[string]any, uuid.UUID, error)`.

For `AssistanceTypeAI`: return `uuid.Nil` as the member ID.
For `AssistanceTypeTeam`: return `m.ID` (the start member's ID).

```go
func (h *aicallHandler) resolveAI(ctx context.Context, assistanceType aicall.AssistanceType, assistanceID uuid.UUID) (*ai.AI, map[string]any, uuid.UUID, error) {
	switch assistanceType {
	case aicall.AssistanceTypeAI:
		c, err := h.aiHandler.Get(ctx, assistanceID)
		if err != nil {
			return nil, nil, uuid.Nil, errors.Wrapf(err, "could not get ai info. ai_id: %s", assistanceID)
		}
		return c, nil, uuid.Nil, nil

	case aicall.AssistanceTypeTeam:
		t, err := h.teamHandler.Get(ctx, assistanceID)
		if err != nil {
			return nil, nil, uuid.Nil, errors.Wrapf(err, "could not get team info. team_id: %s", assistanceID)
		}

		for _, m := range t.Members {
			if m.ID == t.StartMemberID {
				c, errAI := h.aiHandler.Get(ctx, m.AIID)
				if errAI != nil {
					return nil, nil, uuid.Nil, errors.Wrapf(errAI, "could not get ai info for team start member. ai_id: %s", m.AIID)
				}
				return c, t.Parameter, m.ID, nil
			}
		}
		return nil, nil, uuid.Nil, fmt.Errorf("could not find start member in team. team_id: %s, start_member_id: %s", assistanceID, t.StartMemberID)

	default:
		return nil, nil, uuid.Nil, fmt.Errorf("unsupported assistance type: %s", assistanceType)
	}
}
```

**Step 2: Update all callers of `resolveAI`**

In `Start()` (line 66):
```go
c, teamParameter, currentMemberID, err := h.resolveAI(ctx, assistanceType, assistanceID)
```

Then thread `currentMemberID` to `startReferenceTypeCall`, `startReferenceTypeConversation`, `startReferenceTypeNone`.

In `StartTask()` (line 494):
```go
c, teamParameter, currentMemberID, err := h.resolveAI(ctx, assistanceType, assistanceID)
```

Then thread `currentMemberID` to `startAIcall`.

**Step 3: Add `currentMemberID` parameter to `startReferenceTypeCall`, `startReferenceTypeConversation`, `startReferenceTypeNone`, and `startAIcall`**

Each function gains a `currentMemberID uuid.UUID` parameter and passes it through to `startAIcall`.

`startAIcall` gains `currentMemberID uuid.UUID` and passes it to `Create`.

For `startReferenceTypeConversation`, the existing aicall path (line 166-173) does NOT reset `currentMemberID` — it keeps whatever was already stored (preserving the last active member).

**Step 4: Add `currentMemberID` parameter to `Create`**

In `bin-ai-manager/pkg/aicallhandler/db.go`, add `currentMemberID uuid.UUID` parameter to `Create()` function signature. Set it in the struct:

```go
	tmp := &aicall.AIcall{
		// ... existing fields ...
		PipecatcallID:   pipecatcallID,
		CurrentMemberID: currentMemberID,
		// ... rest ...
	}
```

**Step 5: Run tests (expect failures from mock interface mismatch)**

```bash
cd bin-ai-manager
go generate ./... && go test ./...
```

Fix any test files that call `Create` with the old signature — add `uuid.Nil` as the new parameter.

**Step 6: Run full verification**

```bash
cd bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 7: Commit**

```bash
git add bin-ai-manager/pkg/aicallhandler/
git commit -m "NOJIRA-Add-aicall-current-member-id

- bin-ai-manager: Thread CurrentMemberID through resolveAI, startAIcall, and Create"
```

---

### Task 4: Handler layer — Add UpdateCurrentMemberID method

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/main.go:30-71` (AIcallHandler interface)
- Modify: `bin-ai-manager/pkg/aicallhandler/db.go` (add method)

**Step 1: Add `UpdateCurrentMemberID` to `AIcallHandler` interface**

In `bin-ai-manager/pkg/aicallhandler/main.go`, add to the interface:

```go
	UpdateCurrentMemberID(ctx context.Context, id uuid.UUID, currentMemberID uuid.UUID) (*aicall.AIcall, error)
```

**Step 2: Implement `UpdateCurrentMemberID`**

In `bin-ai-manager/pkg/aicallhandler/db.go`, add after `UpdatePipecatcallID`:

```go
func (h *aicallHandler) UpdateCurrentMemberID(ctx context.Context, id uuid.UUID, currentMemberID uuid.UUID) (*aicall.AIcall, error) {
	fields := map[aicall.Field]any{
		aicall.FieldCurrentMemberID: currentMemberID,
	}
	if errUpdate := h.db.AIcallUpdate(ctx, id, fields); errUpdate != nil {
		return nil, errors.Wrapf(errUpdate, "could not update the current member id for aicall. aicall_id: %s", id)
	}

	res, err := h.db.AIcallGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated aicall info. aicall_id: %s", id)
	}

	return res, nil
}
```

**Step 3: Regenerate mocks and run tests**

```bash
cd bin-ai-manager
go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
git add bin-ai-manager/pkg/aicallhandler/
git commit -m "NOJIRA-Add-aicall-current-member-id

- bin-ai-manager: Add UpdateCurrentMemberID method to AIcallHandler"
```

---

### Task 5: OpenAPI — Add `current_member_id` to AIManagerAIcall schema

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml:2208-2213` (after confbridge_id)

**Step 1: Read and follow `bin-openapi-manager/CLAUDE.md` rules**

**Step 2: Add `current_member_id` to `AIManagerAIcall` schema**

After the `confbridge_id` property (around line 2213), add:

```yaml
        current_member_id:
          type: string
          format: uuid
          x-go-type: string
          description: "The unique identifier of the currently active team member. Only set when assistance_type is 'team'. Updated when the AI agent transitions to a different team member."
          example: "550e8400-e29b-41d4-a716-446655440000"
```

**Step 3: Regenerate OpenAPI types**

```bash
cd bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Regenerate API server code**

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
git add bin-openapi-manager/ bin-api-manager/
git commit -m "NOJIRA-Add-aicall-current-member-id

- bin-openapi-manager: Add current_member_id to AIManagerAIcall schema
- bin-api-manager: Regenerate server code with new field"
```

---

### Task 6: Final verification

**Step 1: Run full verification for all affected services**

```bash
cd bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

cd bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Verify no other services import `aicallhandler` and would break**

```bash
grep -r "aicallhandler" --include="*.go" . | grep -v vendor | grep -v bin-ai-manager
```

If any external callers exist, update them to pass the new `currentMemberID` parameter.
