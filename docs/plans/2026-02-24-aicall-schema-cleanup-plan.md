# AIcall Schema Cleanup Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove deprecated `AIEngineType` field, add STT/TTS/EngineData fields to WebhookMessage and OpenAPI, remove stale `transcribe_id` from OpenAPI, and fix field name mismatches.

**Architecture:** The AIcall model (Go struct, DB, OpenAPI, WebhookMessage) has drifted. This plan syncs all four representations by removing deprecated fields and exposing missing ones externally.

**Tech Stack:** Go, Alembic (Python), OpenAPI 3.0, oapi-codegen

---

### Task 1: Remove AIEngineType from AIcall struct and update field constants

**Files:**
- Modify: `bin-ai-manager/models/aicall/main.go:17` (remove AIEngineType field)
- Modify: `bin-ai-manager/models/aicall/field.go:12` (remove FieldAIEngineType constant)
- Modify: `bin-ai-manager/models/aicall/filters.go:13` (remove AIEngineType from FieldStruct)

**Step 1: Remove `AIEngineType` field from AIcall struct**

In `bin-ai-manager/models/aicall/main.go`, remove line 17:
```go
AIEngineType  ai.EngineType  `json:"ai_engine_type,omitempty" db:"ai_engine_type"`
```

**Step 2: Remove `FieldAIEngineType` constant from field.go**

In `bin-ai-manager/models/aicall/field.go`, remove line 12:
```go
FieldAIEngineType  Field = "ai_engine_type"
```

**Step 3: Remove `AIEngineType` from FieldStruct in filters.go**

In `bin-ai-manager/models/aicall/filters.go`, remove line 13:
```go
AIEngineType  ai.EngineType  `filter:"ai_engine_type"`
```

Also remove the `"monorepo/bin-ai-manager/models/ai"` import if no other fields reference it. Check: `AIEngineModel` still uses `ai.EngineModel`, so the import stays.

---

### Task 2: Update WebhookMessage to add missing fields

**Files:**
- Modify: `bin-ai-manager/models/aicall/webhook.go`

**Step 1: Update WebhookMessage struct**

Replace the current struct fields (lines 14-36) with:

```go
type WebhookMessage struct {
	identity.Identity

	AIID          uuid.UUID      `json:"ai_id,omitempty"`
	AIEngineModel ai.EngineModel `json:"ai_engine_model,omitempty"`
	AIEngineData  map[string]any `json:"ai_engine_data,omitempty"`
	AITTSType     ai.TTSType     `json:"ai_tts_type,omitempty"`
	AITTSVoiceID  string         `json:"ai_tts_voice_id,omitempty"`
	AISTTType     ai.STTType     `json:"ai_stt_type,omitempty"`

	ActiveflowID  uuid.UUID     `json:"activeflow_id,omitempty"`
	ReferenceType ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty"`

	ConfbridgeID uuid.UUID `json:"confbridge_id,omitempty"`

	Status Status `json:"status,omitempty"`

	Gender   Gender `json:"gender,omitempty"`
	Language string `json:"language,omitempty"`

	TMEnd    *time.Time `json:"tm_end"`
	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}
```

Changes: Removed `AIEngineType`. Added `AIEngineData`, `AITTSType`, `AITTSVoiceID`, `AISTTType`.

**Step 2: Update ConvertWebhookMessage()**

Replace the return struct (lines 40-62) with:

```go
func (h *AIcall) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		AIID:          h.AIID,
		AIEngineModel: h.AIEngineModel,
		AIEngineData:  h.AIEngineData,
		AITTSType:     h.AITTSType,
		AITTSVoiceID:  h.AITTSVoiceID,
		AISTTType:     h.AISTTType,

		ActiveflowID:  h.ActiveflowID,
		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		ConfbridgeID: h.ConfbridgeID,

		Status: h.Status,

		Gender:   h.Gender,
		Language: h.Language,

		TMEnd:    h.TMEnd,
		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}
```

---

### Task 3: Remove AIEngineType from handler and creation code

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/db.go:42` (remove AIEngineType assignment)

**Step 1: Remove AIEngineType from Create method**

In `bin-ai-manager/pkg/aicallhandler/db.go`, remove line 42:
```go
AIEngineType:  c.EngineType,
```

---

### Task 4: Update all test files

**Files:**
- Modify: `bin-ai-manager/models/aicall/main_test.go`
- Modify: `bin-ai-manager/models/aicall/webhook_test.go`
- Modify: `bin-ai-manager/models/aicall/field_test.go`
- Modify: `bin-ai-manager/models/aicall/filters_test.go`
- Modify: `bin-ai-manager/pkg/dbhandler/aicall_test.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/db_test.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/start_test.go`

**Step 1: Update main_test.go**

Remove all `aiEngineType` test struct fields, test data assignments, and assertions:
- Remove `aiEngineType ai.EngineType` from test struct definition (line 16)
- Remove `aiEngineType: ai.EngineTypeNone` from all test cases (lines 34, 52, 70, 88)
- Remove `AIEngineType: tt.aiEngineType,` from AIcall construction (line 108)
- Remove `if ac.AIEngineType != tt.aiEngineType` assertion block (lines 126-128)

**Step 2: Update webhook_test.go**

Remove `AIEngineType: ai.EngineTypeNone,` from test case AIcall structs (line 27).

**Step 3: Update field_test.go**

Remove the `field_ai_engine_type` test case (lines 28-32):
```go
{
    name:     "field_ai_engine_type",
    constant: FieldAIEngineType,
    expected: "ai_engine_type",
},
```

**Step 4: Update filters_test.go**

Remove `AIEngineType: ai.EngineTypeNone,` (line 20) and `AIEngineType: "",` (line 38) from test cases.

If `ai.EngineTypeNone` was the only usage of the `ai` import, remove the import too.

**Step 5: Update dbhandler/aicall_test.go**

Remove `AIEngineType: ai.EngineTypeNone,` from both test case structs (lines 42, 68).

**Step 6: Update aicallhandler/db_test.go**

Remove `AIEngineType: ai.EngineTypeNone,` from the expectAIcall struct (line 79).

**Step 7: Update aicallhandler/start_test.go**

Remove `AIEngineType: ai.EngineTypeNone,` from `responseAIcall` (line 92) and `expectAIcall` (line 121) structs.

**Step 8: Run tests to verify**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-aicall-schema-cleanup/bin-ai-manager && go test ./...`
Expected: All tests pass.

---

### Task 5: Update test SQL script

**Files:**
- Modify: `bin-ai-manager/scripts/database_scripts_test/table_ai_aicalls.sql`

**Step 1: Remove ai_engine_type column and stale transcribe_id index**

Remove line 7: `ai_engine_type   varchar(255), -- ai engine type`
Remove line 39: `create index idx_ai_aicalls_transcribe_id on ai_aicalls(transcribe_id);`

---

### Task 6: Run verification for bin-ai-manager

**Step 1: Run full verification workflow**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-aicall-schema-cleanup/bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: All steps pass.

**Step 2: Commit Go changes**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-aicall-schema-cleanup
git add bin-ai-manager/
git commit -m "NOJIRA-aicall-schema-cleanup

- bin-ai-manager: Remove deprecated AIEngineType field from AIcall struct
- bin-ai-manager: Add AIEngineData, AITTSType, AITTSVoiceID, AISTTType to WebhookMessage
- bin-ai-manager: Remove stale transcribe_id index from test SQL
- bin-ai-manager: Update all tests for field removal"
```

---

### Task 7: Create Alembic migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<hash>_ai_aicalls_drop_column_ai_engine_type.py`

**Step 1: Generate migration file**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-aicall-schema-cleanup/bin-dbscheme-manager/bin-manager
cp alembic.ini.sample alembic.ini 2>/dev/null; true
```

Create the migration file manually with revision ID and down_revision matching the latest:

```python
"""ai_aicalls drop column ai_engine_type

Revision ID: a1b2c3d4e5f6
Revises: fe1a2b3c4d5e
Create Date: 2026-02-24 15:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'a1b2c3d4e5f6'
down_revision = 'fe1a2b3c4d5e'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE ai_aicalls DROP COLUMN ai_engine_type;""")


def downgrade():
    op.execute("""ALTER TABLE ai_aicalls ADD ai_engine_type varchar(255) DEFAULT '' AFTER ai_id;""")
```

**Step 2: Commit migration**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-aicall-schema-cleanup
git add bin-dbscheme-manager/
git commit -m "NOJIRA-aicall-schema-cleanup

- bin-dbscheme-manager: Add migration to drop ai_engine_type column from ai_aicalls"
```

---

### Task 8: Update OpenAPI schema

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (AIManagerAIcall schema, ~lines 1808-1895)

**Step 1: Update AIManagerAIcall schema**

Replace the properties section of `AIManagerAIcall` (lines ~1810-1895). The changes:

1. Remove `engine_type` property (lines 1829-1832)
2. Rename `engine_model` to `ai_engine_model` (lines 1833-1836)
3. Remove `transcribe_id` property (lines 1859-1864)
4. Add `ai_engine_data` property (after ai_engine_model)
5. Add `ai_tts_type` property (after ai_engine_data)
6. Add `ai_tts_voice_id` property (after ai_tts_type)
7. Add `ai_stt_type` property (after ai_tts_voice_id)

The updated properties section for AIManagerAIcall should be:

```yaml
    AIManagerAIcall:
      type: object
      properties:
        id:
          type: string
          format: uuid
          x-go-type: string
          description: The unique identifier of the AI call.
          example: "550e8400-e29b-41d4-a716-446655440000"
        customer_id:
          type: string
          format: uuid
          x-go-type: string
          description: "The unique identifier of the associated customer. Returned from the `GET /customers` response."
          example: "7c4d2f3a-1b8e-4f5c-9a6d-3e2f1a0b4c5d"
        ai_id:
          type: string
          format: uuid
          x-go-type: string
          description: "The unique identifier of the associated AI. Returned from the `POST /ais` or `GET /ais` response."
          example: "550e8400-e29b-41d4-a716-446655440000"
        ai_engine_model:
          $ref: '#/components/schemas/AIManagerAIEngineModel'
          description: Model of the AI engine used for this call.
          example: "openai.gpt-4o"
        ai_engine_data:
          type: object
          additionalProperties: true
          description: Custom key-value configuration data specific to the AI engine type.
        ai_tts_type:
          type: string
          description: Text-to-speech provider type used for this call.
          example: "elevenlabs"
        ai_tts_voice_id:
          type: string
          description: Text-to-speech voice identifier used for this call.
          example: "21m00Tcm4TlvDq8ikWAM"
        ai_stt_type:
          type: string
          description: Speech-to-text provider type used for this call.
          example: "deepgram"
        activeflow_id:
          type: string
          format: uuid
          x-go-type: string
          description: "The unique identifier of the activeflow. Returned from the `GET /activeflows` response."
          example: "550e8400-e29b-41d4-a716-446655440000"
        reference_type:
          $ref: '#/components/schemas/AIManagerAIcallReferenceType'
          description: Type of reference associated with the AI call.
          example: "call"
        reference_id:
          type: string
          format: uuid
          x-go-type: string
          description: "The unique identifier of the referenced resource. The actual resource type is determined by reference_type. Returned from the corresponding resource endpoint."
          example: "550e8400-e29b-41d4-a716-446655440000"
        confbridge_id:
          type: string
          format: uuid
          x-go-type: string
          description: "The unique identifier of the conference bridge. Returned from the `GET /conferences` response."
          example: "550e8400-e29b-41d4-a716-446655440000"
        status:
          $ref: '#/components/schemas/AIManagerAIcallStatus'
          description: Status of the AI call.
          example: "progressing"
        gender:
          $ref: '#/components/schemas/AIManagerAIcallGender'
          description: Gender associated with the AI call.
          example: "female"
        language:
          type: string
          description: Language used during the AI call.
          example: "en-US"
        tm_end:
          type: string
          format: date-time
          x-go-type: string
          description: Timestamp when the AI call ended.
          example: "2026-01-15T09:30:00.000000Z"
        tm_create:
          type: string
          format: date-time
          x-go-type: string
          description: Timestamp when the AI call was created.
          example: "2026-01-15T09:30:00.000000Z"
        tm_update:
          type: string
          format: date-time
          x-go-type: string
          description: Timestamp when the AI call was last updated.
          example: "2026-01-15T09:30:00.000000Z"
        tm_delete:
          type: string
          format: date-time
          x-go-type: string
          description: Timestamp when the AI call was deleted.
          example: "2026-01-15T09:30:00.000000Z"
```

---

### Task 9: Regenerate OpenAPI and API manager code

**Step 1: Regenerate bin-openapi-manager**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-aicall-schema-cleanup/bin-openapi-manager
go generate ./...
```

**Step 2: Verify bin-openapi-manager**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-aicall-schema-cleanup/bin-openapi-manager
go mod tidy && go mod vendor && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Regenerate bin-api-manager**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-aicall-schema-cleanup/bin-api-manager
go generate ./...
```

**Step 4: Verify bin-api-manager**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-aicall-schema-cleanup/bin-api-manager
go mod tidy && go mod vendor && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit OpenAPI and generated code**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-aicall-schema-cleanup
git add bin-openapi-manager/ bin-api-manager/
git commit -m "NOJIRA-aicall-schema-cleanup

- bin-openapi-manager: Remove transcribe_id and engine_type from AIManagerAIcall schema
- bin-openapi-manager: Add ai_engine_data, ai_tts_type, ai_tts_voice_id, ai_stt_type
- bin-openapi-manager: Fix engine_model field name to ai_engine_model
- bin-api-manager: Regenerate server code from updated OpenAPI spec"
```

---

### Task 10: Final verification and push

**Step 1: Re-run bin-ai-manager verification**

The test SQL was updated and regeneration may have affected things:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-aicall-schema-cleanup/bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Check for conflicts with main**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-aicall-schema-cleanup
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-aicall-schema-cleanup
```

Create PR with title `NOJIRA-aicall-schema-cleanup` and body summarizing all changes.
