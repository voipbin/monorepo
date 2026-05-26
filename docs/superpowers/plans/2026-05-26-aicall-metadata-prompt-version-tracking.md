# AIcall Metadata + Prompt Version Tracking Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `current_prompt_history_id` to the AI model and a generic `metadata` field (containing `prompt_snapshots`) to AIcall, capturing the exact prompt version in effect for every participant at call start time.

**Architecture:** Two Alembic migrations add columns to `ai_ais` and `ai_aicalls`. `commondatabasehandler.PrepareFields` reads struct tags automatically, so no dbhandler interface changes are needed. The AI handler `Create`/`Update` are restructured to write the version pointer atomically with each prompt write. A new `resolveAIForTeam` helper fetches all team members' AI configs in parallel; `buildPromptSnapshots` builds the metadata slice; `Create`/`CreateByMessaging` gain a `metadata map[string]any` parameter.

**Tech Stack:** Go, MySQL 8.0, Squirrel query builder, gomock (go.uber.org/mock), Alembic (Python), table-driven tests.

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aicall-metadata-prompt-version-tracking`

---

## File Map

| Action | Path |
|--------|------|
| Create | `bin-dbscheme-manager/bin-manager/main/versions/<rev1>_ai_ais_add_current_prompt_history_id.py` |
| Create | `bin-dbscheme-manager/bin-manager/main/versions/<rev2>_ai_aicalls_add_metadata.py` |
| Modify | `bin-ai-manager/models/ai/main.go` |
| Modify | `bin-ai-manager/models/ai/field.go` |
| Modify | `bin-ai-manager/models/aicall/main.go` |
| Modify | `bin-ai-manager/models/aicall/field.go` |
| Modify | `bin-ai-manager/models/aicall/webhook.go` |
| Modify | `bin-ai-manager/models/aicall/webhook_test.go` |
| Modify | `bin-ai-manager/pkg/aihandler/db.go` |
| Modify | `bin-ai-manager/pkg/aihandler/chatbot.go` |
| Modify | `bin-ai-manager/pkg/aihandler/chatbot_test.go` |
| Modify | `bin-ai-manager/pkg/aicallhandler/start.go` |
| Modify | `bin-ai-manager/pkg/aicallhandler/db.go` |
| Modify | `bin-ai-manager/pkg/aicallhandler/db_test.go` |
| Modify | `bin-ai-manager/pkg/aicallhandler/start_test.go` |
| Modify | `bin-api-manager/docsdev/source/*.rst` |
| Modify | `bin-openapi-manager/` (YAML schema) |

---

### Task 1: Alembic Migrations

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<rev1>_ai_ais_add_current_prompt_history_id.py`
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<rev2>_ai_aicalls_add_metadata.py`

- [ ] **Step 1: Generate migration 1 for ai_ais**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aicall-metadata-prompt-version-tracking/bin-dbscheme-manager/bin-manager
alembic -c alembic.ini revision -m "ai_ais_add_current_prompt_history_id"
```

Expected: A new file appears in `main/versions/` named `<hash>_ai_ais_add_current_prompt_history_id.py`. Note the revision ID printed in the terminal — you will verify Migration 2 chains to it.

- [ ] **Step 2: Edit migration 1 — fill upgrade/downgrade bodies**

Open the newly generated file. Replace the empty `upgrade()` and `downgrade()` stubs:

```python
def upgrade():
    op.execute(
        "ALTER TABLE ai_ais "
        "ADD COLUMN current_prompt_history_id BINARY(16) NOT NULL "
        "DEFAULT (X'00000000000000000000000000000000')"
    )


def downgrade():
    op.execute("ALTER TABLE ai_ais DROP COLUMN current_prompt_history_id")
```

- [ ] **Step 3: Generate migration 2 for ai_aicalls**

```bash
alembic -c alembic.ini revision -m "ai_aicalls_add_metadata"
```

Expected: A new file whose `down_revision` field matches the revision ID from Step 1. Open the file and verify `down_revision == "<migration1_revision_id>"`.

- [ ] **Step 4: Edit migration 2 — fill upgrade/downgrade bodies**

```python
def upgrade():
    op.execute(
        "ALTER TABLE ai_aicalls "
        "ADD COLUMN metadata JSON NOT NULL DEFAULT (JSON_OBJECT())"
    )


def downgrade():
    op.execute("ALTER TABLE ai_aicalls DROP COLUMN metadata")
```

- [ ] **Step 5: Commit migrations**

```bash
git add bin-dbscheme-manager/bin-manager/main/versions/
git commit -m "NOJIRA-Add-aicall-metadata-prompt-version-tracking

- bin-dbscheme-manager: Add current_prompt_history_id column to ai_ais table
- bin-dbscheme-manager: Add metadata JSON column to ai_aicalls table"
```

---

### Task 2: AI Model — Add CurrentPromptHistoryID

**Files:**
- Modify: `bin-ai-manager/models/ai/main.go`
- Modify: `bin-ai-manager/models/ai/field.go`

- [ ] **Step 1: Add field to AI struct**

In `bin-ai-manager/models/ai/main.go`, add after the `InitPrompt string` line:

```go
CurrentPromptHistoryID uuid.UUID `json:"current_prompt_history_id" db:"current_prompt_history_id,uuid"`
```

No `omitempty` — `uuid.UUID` is `[16]byte`; Go's `encoding/json` does not omit zero-value fixed-size arrays regardless of the tag.

- [ ] **Step 2: Add field constant**

In `bin-ai-manager/models/ai/field.go`, after `FieldInitPrompt`, add:

```go
FieldCurrentPromptHistoryID Field = "current_prompt_history_id"
```

- [ ] **Step 3: Run build check and model tests**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aicall-metadata-prompt-version-tracking/bin-ai-manager
go build ./... && go test ./models/ai/...
```

Expected: PASS — the new field is additive; all existing tests continue to pass.

- [ ] **Step 4: Commit**

```bash
git add bin-ai-manager/models/ai/main.go bin-ai-manager/models/ai/field.go
git commit -m "NOJIRA-Add-aicall-metadata-prompt-version-tracking

- bin-ai-manager: Add CurrentPromptHistoryID field to AI struct
- bin-ai-manager: Add FieldCurrentPromptHistoryID constant to ai/field.go"
```

---

### Task 3: AIcall Model — Add Metadata, PromptSnapshot, MetaKeyPromptSnapshots, FieldMetadata

**Files:**
- Modify: `bin-ai-manager/models/aicall/main.go`
- Modify: `bin-ai-manager/models/aicall/field.go`

- [ ] **Step 1: Add PromptSnapshot, MetaKeyPromptSnapshots, and Metadata to main.go**

At the top of `bin-ai-manager/models/aicall/main.go`, before the `AIcall` struct, add:

```go
// PromptSnapshot records the prompt version and final substituted text for one
// AI participant at AIcall start time.
type PromptSnapshot struct {
	AIID            uuid.UUID `json:"ai_id"`
	PromptHistoryID uuid.UUID `json:"prompt_history_id"` // zero UUID = no history recorded yet
	Prompt          string    `json:"prompt"`
	MemberID        uuid.UUID `json:"member_id"` // zero UUID for single-AI calls
}

// MetaKeyPromptSnapshots is the Metadata map key for the prompt snapshot slice.
const MetaKeyPromptSnapshots = "prompt_snapshots"
```

In the `AIcall` struct, add the `Metadata` field (after `STTLanguage`, before the `TM*` fields):

```go
Metadata map[string]any `json:"metadata,omitempty" db:"metadata,json"`
```

- [ ] **Step 2: Add FieldMetadata to field.go**

In `bin-ai-manager/models/aicall/field.go`, after `FieldSTTLanguage`, add:

```go
FieldMetadata Field = "metadata"
```

- [ ] **Step 3: Build check and model tests**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aicall-metadata-prompt-version-tracking/bin-ai-manager
go build ./... && go test ./models/aicall/...
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add bin-ai-manager/models/aicall/main.go bin-ai-manager/models/aicall/field.go
git commit -m "NOJIRA-Add-aicall-metadata-prompt-version-tracking

- bin-ai-manager: Add PromptSnapshot struct, MetaKeyPromptSnapshots constant to aicall model
- bin-ai-manager: Add Metadata field to AIcall struct
- bin-ai-manager: Add FieldMetadata constant to aicall/field.go"
```

---

### Task 4: AIcall Webhook — Expose Metadata in WebhookMessage

**Files:**
- Modify: `bin-ai-manager/models/aicall/webhook.go`
- Modify: `bin-ai-manager/models/aicall/webhook_test.go`

- [ ] **Step 1: Write the failing test**

Add to `bin-ai-manager/models/aicall/webhook_test.go`:

```go
func TestConvertWebhookMessage_includesMetadata(t *testing.T) {
	h := &AIcall{
		Metadata: map[string]any{
			MetaKeyPromptSnapshots: []PromptSnapshot{
				{Prompt: "hello world"},
			},
		},
	}
	msg := h.ConvertWebhookMessage()
	if msg.Metadata == nil {
		t.Fatal("expected Metadata to be non-nil in WebhookMessage")
	}
	if _, ok := msg.Metadata[MetaKeyPromptSnapshots]; !ok {
		t.Errorf("expected %q key in Metadata", MetaKeyPromptSnapshots)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aicall-metadata-prompt-version-tracking/bin-ai-manager
go test ./models/aicall/... -run TestConvertWebhookMessage_includesMetadata -v
```

Expected: FAIL — `WebhookMessage` has no `Metadata` field yet (compile error or missing field).

- [ ] **Step 3: Add Metadata to WebhookMessage and ConvertWebhookMessage**

In `bin-ai-manager/models/aicall/webhook.go`, add to `WebhookMessage` after `STTLanguage`:

```go
Metadata map[string]any `json:"metadata,omitempty"`
```

In `ConvertWebhookMessage()`, add after `STTLanguage: h.STTLanguage,`:

```go
Metadata: h.Metadata,
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./models/aicall/... -run TestConvertWebhookMessage_includesMetadata -v
```

Expected: PASS.

- [ ] **Step 5: Run all aicall model tests**

```bash
go test ./models/aicall/...
```

Expected: PASS — no regressions.

- [ ] **Step 6: Commit**

```bash
git add bin-ai-manager/models/aicall/webhook.go bin-ai-manager/models/aicall/webhook_test.go
git commit -m "NOJIRA-Add-aicall-metadata-prompt-version-tracking

- bin-ai-manager: Expose Metadata field in AIcall WebhookMessage
- bin-ai-manager: Copy Metadata in ConvertWebhookMessage"
```

---

### Task 5: AI Handler — Restructure Create and Update for Prompt Version Tracking

**Files:**
- Modify: `bin-ai-manager/pkg/aihandler/db.go`
- Modify: `bin-ai-manager/pkg/aihandler/chatbot.go`
- Modify: `bin-ai-manager/pkg/aihandler/chatbot_test.go`

- [ ] **Step 1: Write failing tests for the new Create behavior**

Add to `bin-ai-manager/pkg/aihandler/chatbot_test.go` (within `TestCreate` test table or as a new case):

```go
{
    name:             "create_with_prompt_sets_current_history_id_atomically",
    customerID:       uuid.Must(uuid.NewV4()),
    aiName:           "Prompt AI",
    engineModel:      ai.EngineModelOpenaiGPT5,
    ttsType:          ai.TTSTypeNone,
    sttType:          ai.STTTypeNone,
    initPrompt:       "initial system prompt",
    setupMock: func(m *dbhandler.MockDBHandler, r *requesthandler.MockRequestHandler) {
        var capturedAI *ai.AI
        preGeneratedHistoryID := uuid.Must(uuid.NewV4())
        // utilHandler.UUIDCreate called once for historyID, once for aiID inside dbCreate
        // (adjust if the mock util also handles the directCreate ID — check constructor)
        r.EXPECT().DirectV1DirectCreate(gomock.Any(), gomock.Any(), dmdirect.ResourceTypeAI, gomock.Any()).
            Return(&dmdirect.Direct{Hash: "abc"}, nil).Times(1)
        m.EXPECT().AICreate(gomock.Any(), gomock.Any()).DoAndReturn(
            func(_ context.Context, a *ai.AI) error {
                capturedAI = a
                // The AI struct must carry the pre-generated historyID
                if a.CurrentPromptHistoryID == uuid.Nil {
                    return fmt.Errorf("expected non-nil CurrentPromptHistoryID on AI struct")
                }
                return nil
            },
        ).Times(1)
        m.EXPECT().AIGet(gomock.Any(), gomock.Any()).DoAndReturn(
            func(_ context.Context, id uuid.UUID) (*ai.AI, error) {
                return capturedAI, nil
            },
        ).Times(1)
        m.EXPECT().AIPromptHistoryCreate(gomock.Any(), gomock.Any()).DoAndReturn(
            func(_ context.Context, h *aiprompthistory.AIPromptHistory) error {
                if capturedAI != nil && h.ID != capturedAI.CurrentPromptHistoryID {
                    return fmt.Errorf("history ID %v does not match AI.CurrentPromptHistoryID %v",
                        h.ID, capturedAI.CurrentPromptHistoryID)
                }
                return nil
            },
        ).Times(1)
        _ = preGeneratedHistoryID
    },
    wantError: false,
},
{
    name:        "create_without_prompt_does_not_create_history",
    customerID:  uuid.Must(uuid.NewV4()),
    aiName:      "No Prompt AI",
    engineModel: ai.EngineModelOpenaiGPT5,
    ttsType:     ai.TTSTypeNone,
    sttType:     ai.STTTypeNone,
    initPrompt:  "",
    setupMock: func(m *dbhandler.MockDBHandler, r *requesthandler.MockRequestHandler) {
        r.EXPECT().DirectV1DirectCreate(gomock.Any(), gomock.Any(), dmdirect.ResourceTypeAI, gomock.Any()).
            Return(&dmdirect.Direct{Hash: "xyz"}, nil).Times(1)
        m.EXPECT().AICreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
        m.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(&ai.AI{}, nil).Times(1)
        // AIPromptHistoryCreate MUST NOT be called
    },
    wantError: false,
},
```

- [ ] **Step 2: Run Create tests to verify they fail**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aicall-metadata-prompt-version-tracking/bin-ai-manager
go test ./pkg/aihandler/... -run TestCreate -v 2>&1 | grep -E "PASS|FAIL|--- "
```

Expected: FAIL on the new cases (because `dbCreate` does not yet accept `currentPromptHistoryID`).

- [ ] **Step 3: Add currentPromptHistoryID param to dbCreate in db.go**

In `bin-ai-manager/pkg/aihandler/db.go`, change the `dbCreate` signature to add `currentPromptHistoryID uuid.UUID` as the **last** parameter:

```go
func (h *aiHandler) dbCreate(
    ctx context.Context,
    customerID uuid.UUID,
    name string,
    detail string,
    engineModel ai.EngineModel,
    parameter map[string]any,
    engineKey string,
    ragID uuid.UUID,
    initPrompt string,
    ttsType ai.TTSType,
    ttsVoiceID string,
    sttType ai.STTType,
    sttLanguage string,
    toolNames []tool.ToolName,
    vadConfig *ai.VADConfig,
    smartTurnEnabled bool,
    currentPromptHistoryID uuid.UUID,
) (*ai.AI, error) {
```

In the `ai.AI` struct literal inside `dbCreate`, add:

```go
CurrentPromptHistoryID: currentPromptHistoryID,
```

Also add a private helper at the end of `db.go` (shared by the two Update branches):

```go
func (h *aiHandler) buildUpdateFields(
    name, detail string,
    engineModel ai.EngineModel,
    parameter map[string]any,
    engineKey string,
    ragID uuid.UUID,
    initPrompt string,
    ttsType ai.TTSType,
    ttsVoiceID string,
    sttType ai.STTType,
    sttLanguage string,
    toolNames []tool.ToolName,
    vadConfig *ai.VADConfig,
    smartTurnEnabled bool,
) map[ai.Field]any {
    return map[ai.Field]any{
        ai.FieldName:             name,
        ai.FieldDetail:           detail,
        ai.FieldEngineModel:      engineModel,
        ai.FieldParameter:        parameter,
        ai.FieldEngineKey:        engineKey,
        ai.FieldRagID:            ragID,
        ai.FieldInitPrompt:       initPrompt,
        ai.FieldTTSType:          ttsType,
        ai.FieldTTSVoiceID:       ttsVoiceID,
        ai.FieldSTTType:          sttType,
        ai.FieldSTTLanguage:      sttLanguage,
        ai.FieldToolNames:        toolNames,
        ai.FieldVADConfig:        vadConfig,
        ai.FieldSmartTurnEnabled: smartTurnEnabled,
    }
}
```

- [ ] **Step 4: Restructure Create in chatbot.go**

Replace the body of `Create` in `bin-ai-manager/pkg/aihandler/chatbot.go`:

```go
func (h *aiHandler) Create(
    ctx context.Context,
    customerID uuid.UUID,
    name string,
    detail string,
    engineModel ai.EngineModel,
    parameter map[string]any,
    engineKey string,
    ragID uuid.UUID,
    initPrompt string,
    ttsType ai.TTSType,
    ttsVoiceID string,
    sttType ai.STTType,
    sttLanguage string,
    toolNames []tool.ToolName,
    vadConfig *ai.VADConfig,
    smartTurnEnabled bool,
) (*ai.AI, error) {

    if !ai.IsValidEngineModel(engineModel) {
        return nil, fmt.Errorf("invalid engine model: %s", engineModel)
    }
    if !ttsType.IsValid() {
        return nil, fmt.Errorf("invalid tts_type: %s. valid values: %s", ttsType, strings.Join(ttsType.ValidValues(), ", "))
    }
    if !sttType.IsValid() {
        return nil, fmt.Errorf("invalid stt_type: %s. valid values: %s", sttType, strings.Join(sttType.ValidValues(), ", "))
    }
    if err := vadConfig.Validate(); err != nil {
        return nil, fmt.Errorf("invalid vad_config: %w", err)
    }

    // Pre-generate the history ID so we can write it into the AI row at creation time,
    // ensuring the AI always points to its own first prompt history entry.
    var currentPromptHistoryID uuid.UUID
    if initPrompt != "" {
        currentPromptHistoryID = h.utilHandler.UUIDCreate()
    }

    res, err := h.dbCreate(ctx, customerID, name, detail, engineModel, parameter, engineKey, ragID,
        initPrompt, ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled,
        currentPromptHistoryID)
    if err != nil {
        return nil, errors.Wrapf(err, "could not create ai")
    }

    if initPrompt != "" {
        if errHistory := h.db.AIPromptHistoryCreate(ctx, &aiprompthistory.AIPromptHistory{
            Identity: identity.Identity{
                ID:         currentPromptHistoryID,
                CustomerID: res.CustomerID,
            },
            AIID:   res.ID,
            Prompt: initPrompt,
        }); errHistory != nil {
            logrus.WithField("func", "Create").Errorf("Could not create prompt history. err: %v", errHistory)
        }
    }

    return res, nil
}
```

- [ ] **Step 5: Run Create tests to verify they pass**

```bash
go test ./pkg/aihandler/... -run TestCreate -v 2>&1 | grep -E "PASS|FAIL|--- "
```

Expected: PASS on all Create test cases.

- [ ] **Step 6: Write failing tests for the Update three-branch restructuring**

Add to `bin-ai-manager/pkg/aihandler/chatbot_test.go`:

```go
func TestUpdate_promptChangedCreatesHistoryAndSetsID(t *testing.T) {
    mc := gomock.NewController(t)
    defer mc.Finish()

    mockDB := dbhandler.NewMockDBHandler(mc)
    mockReq := requesthandler.NewMockRequestHandler(mc)
    mockNotify := notifyhandler.NewMockNotifyHandler(mc)
    mockUtil := utilhandler.NewMockUtilHandler(mc)

    aiID := uuid.Must(uuid.NewV4())
    newHistoryID := uuid.Must(uuid.NewV4())
    existing := &ai.AI{InitPrompt: "old prompt"}
    existing.ID = aiID
    existing.CustomerID = uuid.Must(uuid.NewV4())

    mockUtil.EXPECT().UUIDCreate().Return(newHistoryID).Times(1)
    mockDB.EXPECT().AIGet(gomock.Any(), aiID).Return(existing, nil).Times(1) // pre-fetch
    mockDB.EXPECT().AIUpdate(gomock.Any(), aiID, gomock.Any()).DoAndReturn(
        func(_ context.Context, id uuid.UUID, fields map[ai.Field]any) error {
            v, ok := fields[ai.FieldCurrentPromptHistoryID]
            if !ok || v != newHistoryID {
                return fmt.Errorf("expected FieldCurrentPromptHistoryID=%v, got %v", newHistoryID, v)
            }
            return nil
        },
    ).Times(1)
    mockDB.EXPECT().AIGet(gomock.Any(), aiID).Return(existing, nil).Times(1) // post-update
    mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), ai.EventTypeUpdated, gomock.Any()).Times(1)
    mockDB.EXPECT().AIPromptHistoryCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)

    h := New(mockDB, mockReq, mockNotify, mockUtil)
    _, err := h.Update(context.Background(), aiID, "name", "", ai.EngineModelOpenaiGPT5, nil, "", uuid.Nil,
        "new prompt", ai.TTSTypeNone, "", ai.STTTypeNone, "", nil, nil, false)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

func TestUpdate_promptClearedResetsID(t *testing.T) {
    mc := gomock.NewController(t)
    defer mc.Finish()

    mockDB := dbhandler.NewMockDBHandler(mc)
    mockReq := requesthandler.NewMockRequestHandler(mc)
    mockNotify := notifyhandler.NewMockNotifyHandler(mc)
    mockUtil := utilhandler.NewMockUtilHandler(mc)

    aiID := uuid.Must(uuid.NewV4())
    existing := &ai.AI{InitPrompt: "old prompt"}
    existing.ID = aiID
    existing.CustomerID = uuid.Must(uuid.NewV4())

    // pre-fetch needed for cleared branch detection
    mockDB.EXPECT().AIGet(gomock.Any(), aiID).Return(existing, nil).Times(1)
    mockDB.EXPECT().AIUpdate(gomock.Any(), aiID, gomock.Any()).DoAndReturn(
        func(_ context.Context, id uuid.UUID, fields map[ai.Field]any) error {
            v, ok := fields[ai.FieldCurrentPromptHistoryID]
            if !ok || v != uuid.Nil {
                return fmt.Errorf("expected FieldCurrentPromptHistoryID=uuid.Nil for cleared prompt, got %v", v)
            }
            return nil
        },
    ).Times(1)
    mockDB.EXPECT().AIGet(gomock.Any(), aiID).Return(existing, nil).Times(1) // post-update
    mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), ai.EventTypeUpdated, gomock.Any()).Times(1)
    // AIPromptHistoryCreate MUST NOT be called for cleared prompt

    h := New(mockDB, mockReq, mockNotify, mockUtil)
    _, err := h.Update(context.Background(), aiID, "name", "", ai.EngineModelOpenaiGPT5, nil, "", uuid.Nil,
        "", ai.TTSTypeNone, "", ai.STTTypeNone, "", nil, nil, false)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

func TestUpdate_promptUnchangedDoesNotCreateHistory(t *testing.T) {
    mc := gomock.NewController(t)
    defer mc.Finish()

    mockDB := dbhandler.NewMockDBHandler(mc)
    mockReq := requesthandler.NewMockRequestHandler(mc)
    mockNotify := notifyhandler.NewMockNotifyHandler(mc)
    mockUtil := utilhandler.NewMockUtilHandler(mc)

    aiID := uuid.Must(uuid.NewV4())
    same := "same prompt"
    existing := &ai.AI{InitPrompt: same}
    existing.ID = aiID
    existing.CustomerID = uuid.Must(uuid.NewV4())

    // pre-fetch (non-empty prompt always triggers pre-fetch)
    mockDB.EXPECT().AIGet(gomock.Any(), aiID).Return(existing, nil).Times(1)
    // unchanged branch uses dbUpdate, which internally calls h.db.AIUpdate
    mockDB.EXPECT().AIUpdate(gomock.Any(), aiID, gomock.Any()).Return(nil).Times(1)
    mockDB.EXPECT().AIGet(gomock.Any(), aiID).Return(existing, nil).Times(1)
    mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), ai.EventTypeUpdated, gomock.Any()).Times(1)
    // AIPromptHistoryCreate MUST NOT be called

    h := New(mockDB, mockReq, mockNotify, mockUtil)
    _, err := h.Update(context.Background(), aiID, "name", "", ai.EngineModelOpenaiGPT5, nil, "", uuid.Nil,
        same, ai.TTSTypeNone, "", ai.STTTypeNone, "", nil, nil, false)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}
```

- [ ] **Step 7: Run Update tests to verify they fail**

```bash
go test ./pkg/aihandler/... -run "TestUpdate_prompt" -v 2>&1 | grep -E "PASS|FAIL|--- "
```

Expected: FAIL (old Update() structure does not match the new logic).

- [ ] **Step 8: Restructure Update in chatbot.go**

Replace the entire `Update` function in `bin-ai-manager/pkg/aihandler/chatbot.go`:

```go
func (h *aiHandler) Update(
    ctx context.Context,
    id uuid.UUID,
    name string,
    detail string,
    engineModel ai.EngineModel,
    parameter map[string]any,
    engineKey string,
    ragID uuid.UUID,
    initPrompt string,
    ttsType ai.TTSType,
    ttsVoiceID string,
    sttType ai.STTType,
    sttLanguage string,
    toolNames []tool.ToolName,
    vadConfig *ai.VADConfig,
    smartTurnEnabled bool,
) (*ai.AI, error) {

    if !ai.IsValidEngineModel(engineModel) {
        return nil, fmt.Errorf("invalid engine model: %s", engineModel)
    }
    if !ttsType.IsValid() {
        return nil, fmt.Errorf("invalid tts_type: %s. valid values: %s", ttsType, strings.Join(ttsType.ValidValues(), ", "))
    }
    if !sttType.IsValid() {
        return nil, fmt.Errorf("invalid stt_type: %s. valid values: %s", sttType, strings.Join(sttType.ValidValues(), ", "))
    }
    if err := vadConfig.Validate(); err != nil {
        return nil, fmt.Errorf("invalid vad_config: %w", err)
    }

    // Pre-fetch unconditionally so all three branches can detect changes.
    preUpdateAI, errGet := h.db.AIGet(ctx, id)
    if errGet != nil {
        return nil, errors.Wrapf(errGet, "could not get current ai for update")
    }

    promptChanged := initPrompt != "" && initPrompt != preUpdateAI.InitPrompt
    promptCleared := initPrompt == "" && preUpdateAI.InitPrompt != ""

    switch {
    case promptChanged:
        historyID := h.utilHandler.UUIDCreate()
        fields := h.buildUpdateFields(name, detail, engineModel, parameter, engineKey, ragID, initPrompt,
            ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled)
        fields[ai.FieldCurrentPromptHistoryID] = historyID
        if err := h.db.AIUpdate(ctx, id, fields); err != nil {
            return nil, errors.Wrapf(err, "could not update ai")
        }
        res, err := h.db.AIGet(ctx, id)
        if err != nil {
            return nil, errors.Wrapf(err, "could not get updated ai")
        }
        h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, ai.EventTypeUpdated, res)
        if errHistory := h.db.AIPromptHistoryCreate(ctx, &aiprompthistory.AIPromptHistory{
            Identity: identity.Identity{
                ID:         historyID,
                CustomerID: res.CustomerID,
            },
            AIID:   id,
            Prompt: initPrompt,
        }); errHistory != nil {
            logrus.WithField("func", "Update").Errorf("Could not create prompt history. err: %v", errHistory)
        }
        return res, nil

    case promptCleared:
        fields := h.buildUpdateFields(name, detail, engineModel, parameter, engineKey, ragID, "",
            ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled)
        fields[ai.FieldCurrentPromptHistoryID] = uuid.Nil
        if err := h.db.AIUpdate(ctx, id, fields); err != nil {
            return nil, errors.Wrapf(err, "could not update ai (clear prompt)")
        }
        res, err := h.db.AIGet(ctx, id)
        if err != nil {
            return nil, errors.Wrapf(err, "could not get updated ai")
        }
        h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, ai.EventTypeUpdated, res)
        return res, nil

    default: // prompt unchanged
        return h.dbUpdate(ctx, id, name, detail, engineModel, parameter, engineKey, ragID, initPrompt,
            ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled)
    }
}
```

- [ ] **Step 9: Run all aihandler tests**

```bash
go test ./pkg/aihandler/... -v 2>&1 | grep -E "PASS|FAIL|--- "
```

Expected: All PASS.

- [ ] **Step 10: Commit**

```bash
git add bin-ai-manager/pkg/aihandler/
git commit -m "NOJIRA-Add-aicall-metadata-prompt-version-tracking

- bin-ai-manager: Add currentPromptHistoryID param to dbCreate; set on AI struct at creation
- bin-ai-manager: Restructure Update() to maintain current_prompt_history_id on prompt change/clear
- bin-ai-manager: Add buildUpdateFields helper to consolidate AI field-map construction
- bin-ai-manager: Add tests for Create history ID atomicity and Update three-branch logic"
```

---

### Task 6: AIcall Handler — resolveAIForTeam, Metadata, Create/CreateByMessaging

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/db.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/db_test.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/start_test.go`

- [ ] **Step 1: Write failing test for resolveAIForTeam**

Add to `bin-ai-manager/pkg/aicallhandler/start_test.go`:

```go
func Test_resolveAIForTeam(t *testing.T) {
    tests := []struct {
        name        string
        teamID      uuid.UUID
        setupMock   func(*dbhandler.MockDBHandler, *teamhandler.MockTeamHandler, *aihandler.MockAIHandler)
        expectCount int
        expectErr   bool
    }{
        {
            name:   "both_members_succeed_returns_full_map",
            teamID: uuid.FromStringOrNil("aaaa0000-0000-0000-0000-000000000001"),
            setupMock: func(db *dbhandler.MockDBHandler, th *teamhandler.MockTeamHandler, ah *aihandler.MockAIHandler) {
                m1 := uuid.FromStringOrNil("bbbb0000-0000-0000-0000-000000000001")
                m2 := uuid.FromStringOrNil("bbbb0000-0000-0000-0000-000000000002")
                a1 := uuid.FromStringOrNil("cccc0000-0000-0000-0000-000000000001")
                a2 := uuid.FromStringOrNil("cccc0000-0000-0000-0000-000000000002")
                th.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil("aaaa0000-0000-0000-0000-000000000001")).
                    Return(&team.Team{
                        Members: []team.Member{{ID: m1, AIID: a1}, {ID: m2, AIID: a2}},
                    }, nil).Times(1)
                ah.EXPECT().Get(gomock.Any(), a1).Return(&ai.AI{}, nil).Times(1)
                ah.EXPECT().Get(gomock.Any(), a2).Return(&ai.AI{}, nil).Times(1)
            },
            expectCount: 2,
        },
        {
            name:   "one_member_ai_fetch_fails_returns_partial_map",
            teamID: uuid.FromStringOrNil("aaaa0000-0000-0000-0000-000000000002"),
            setupMock: func(db *dbhandler.MockDBHandler, th *teamhandler.MockTeamHandler, ah *aihandler.MockAIHandler) {
                m1 := uuid.FromStringOrNil("dddd0000-0000-0000-0000-000000000001")
                m2 := uuid.FromStringOrNil("dddd0000-0000-0000-0000-000000000002")
                a1 := uuid.FromStringOrNil("eeee0000-0000-0000-0000-000000000001")
                a2 := uuid.FromStringOrNil("eeee0000-0000-0000-0000-000000000002")
                th.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil("aaaa0000-0000-0000-0000-000000000002")).
                    Return(&team.Team{
                        Members: []team.Member{{ID: m1, AIID: a1}, {ID: m2, AIID: a2}},
                    }, nil).Times(1)
                ah.EXPECT().Get(gomock.Any(), a1).Return(&ai.AI{}, nil).Times(1)
                ah.EXPECT().Get(gomock.Any(), a2).Return(nil, errors.New("not found")).Times(1)
            },
            expectCount: 1,
        },
        {
            name:   "teamhandler_get_fails_returns_error",
            teamID: uuid.FromStringOrNil("aaaa0000-0000-0000-0000-000000000003"),
            setupMock: func(db *dbhandler.MockDBHandler, th *teamhandler.MockTeamHandler, ah *aihandler.MockAIHandler) {
                th.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error")).Times(1)
            },
            expectErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mc := gomock.NewController(t)
            defer mc.Finish()

            mockDB := dbhandler.NewMockDBHandler(mc)
            mockTeam := teamhandler.NewMockTeamHandler(mc)
            mockAI := aihandler.NewMockAIHandler(mc)
            // construct handler using the same New() function used in other tests
            h := &aicallHandler{
                db:          mockDB,
                teamHandler: mockTeam,
                aiHandler:   mockAI,
            }

            tt.setupMock(mockDB, mockTeam, mockAI)

            res, err := h.resolveAIForTeam(context.Background(), tt.teamID)
            if tt.expectErr {
                if err == nil {
                    t.Fatal("expected error, got nil")
                }
                return
            }
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if len(res) != tt.expectCount {
                t.Errorf("expected %d results, got %d", tt.expectCount, len(res))
            }
        })
    }
}
```

- [ ] **Step 2: Run resolveAIForTeam test to verify it fails**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aicall-metadata-prompt-version-tracking/bin-ai-manager
go test ./pkg/aicallhandler/... -run Test_resolveAIForTeam -v 2>&1 | grep -E "PASS|FAIL|--- "
```

Expected: FAIL — `resolveAIForTeam` does not exist yet.

- [ ] **Step 3: Implement resolveAIForTeam in start.go**

Add `"sync"` to the import block of `start.go`, then add this function after `resolveTeamMemberAI`:

```go
// resolveAIForTeam fetches all team members' AI configs, keyed by member UUID.
// Partial-failure: if individual member AI fetches fail, logs a warning for each
// and returns the partial map. Only a teamHandler.Get failure is fatal.
func (h *aicallHandler) resolveAIForTeam(ctx context.Context, teamID uuid.UUID) (map[uuid.UUID]*ai.AI, error) {
	t, err := h.teamHandler.Get(ctx, teamID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get team for resolveAIForTeam. team_id: %s", teamID)
	}

	type memberResult struct {
		memberID uuid.UUID
		ai       *ai.AI
		err      error
	}

	ch := make(chan memberResult, len(t.Members))
	for _, m := range t.Members {
		m := m
		go func() {
			a, errGet := h.aiHandler.Get(ctx, m.AIID)
			ch <- memberResult{memberID: m.ID, ai: a, err: errGet}
		}()
	}

	var mu sync.Mutex
	res := make(map[uuid.UUID]*ai.AI, len(t.Members))
	for range t.Members {
		r := <-ch
		if r.err != nil {
			logrus.WithField("func", "resolveAIForTeam").
				Warnf("Could not get AI for team member — skipping. member_id: %s, err: %v", r.memberID, r.err)
			continue
		}
		mu.Lock()
		res[r.memberID] = r.ai
		mu.Unlock()
	}

	return res, nil
}
```

- [ ] **Step 4: Run resolveAIForTeam test to verify it passes**

```bash
go test ./pkg/aicallhandler/... -run Test_resolveAIForTeam -v 2>&1 | grep -E "PASS|FAIL|--- "
```

Expected: PASS.

- [ ] **Step 5: Add metadata param to Create and CreateByMessaging in db.go**

Change `Create` signature — add `metadata map[string]any` as the **last** parameter:

```go
func (h *aicallHandler) Create(
    ctx context.Context,
    c *ai.AI,
    assistanceType aicall.AssistanceType,
    assistanceID uuid.UUID,
    activeflowID uuid.UUID,
    referenceType aicall.ReferenceType,
    referenceID uuid.UUID,
    confbridgeID uuid.UUID,
    pipecatcallID uuid.UUID,
    currentMemberID uuid.UUID,
    parameter map[string]any,
    metadata map[string]any,
) (*aicall.AIcall, error) {
```

In the `aicall.AIcall` struct literal inside `Create`, add:

```go
Metadata: metadata,
```

Change `CreateByMessaging` the same way — add `metadata map[string]any` as the last parameter, and `Metadata: metadata,` in the struct literal.

- [ ] **Step 6: Fix db_test.go call site — add nil metadata**

In `bin-ai-manager/pkg/aicallhandler/db_test.go` at line 237, the direct call to `h.Create(...)` must pass the new parameter. Add `nil` as the last argument:

```go
res, err := h.Create(ctx, tt.ai, tt.assistanceType, tt.assistanceID, tt.activeflowID,
    tt.referenceType, tt.referenceID, tt.confbridgeID, tt.pipecatcallID, tt.currentMemberID,
    tt.parameter, nil)
```

- [ ] **Step 7: Build to catch remaining call sites**

```bash
go build ./...
```

Expected: Build errors only at `startAIcallByRealtime` (calls `h.Create`) and `startAIcallByMessaging` (calls `h.CreateByMessaging`). Fix both temporarily with `nil`:

In `start.go:531`:
```go
res, err := h.Create(ctx, a, assistanceType, assistanceID, activeflowID, referenceType, referenceID,
    confbridgeID, pipecatcallID, currentMemberID, parameter, nil)
```

In `start.go:581`:
```go
res, err := h.CreateByMessaging(ctx, a, assistanceType, assistanceID, activeflowID, referenceType, referenceID,
    pipecatcallID, currentMemberID, parameter, nil)
```

```bash
go build ./...
```

Expected: Clean.

- [ ] **Step 8: Add buildPromptSnapshots helper to start.go**

Add this helper after `resolveAIForTeam` in `start.go`:

```go
// buildPromptSnapshots constructs the []PromptSnapshot to store in AIcall.Metadata at call start.
// For AssistanceTypeAI: one snapshot for the single AI config.
// For AssistanceTypeTeam: one snapshot per team member (partial-failure-tolerant via resolveAIForTeam).
// activeflowID == uuid.Nil: getInitPrompt returns the raw init_prompt without variable substitution.
func (h *aicallHandler) buildPromptSnapshots(ctx context.Context, a *ai.AI, assistanceType aicall.AssistanceType, assistanceID uuid.UUID, activeflowID uuid.UUID) []aicall.PromptSnapshot {
	switch assistanceType {
	case aicall.AssistanceTypeAI:
		substituted := h.getInitPrompt(ctx, a, activeflowID)
		return []aicall.PromptSnapshot{
			{
				AIID:            a.ID,
				PromptHistoryID: a.CurrentPromptHistoryID,
				Prompt:          substituted,
				// MemberID is zero UUID for single-AI calls
			},
		}

	case aicall.AssistanceTypeTeam:
		memberAIs, err := h.resolveAIForTeam(ctx, assistanceID)
		if err != nil {
			logrus.WithField("func", "buildPromptSnapshots").
				Errorf("Could not resolve team AIs — storing empty snapshots. err: %v", err)
			return []aicall.PromptSnapshot{}
		}
		snapshots := make([]aicall.PromptSnapshot, 0, len(memberAIs))
		for memberID, memberAI := range memberAIs {
			substituted := h.getInitPrompt(ctx, memberAI, activeflowID)
			snapshots = append(snapshots, aicall.PromptSnapshot{
				AIID:            memberAI.ID,
				PromptHistoryID: memberAI.CurrentPromptHistoryID,
				Prompt:          substituted,
				MemberID:        memberID,
			})
		}
		return snapshots

	default:
		return []aicall.PromptSnapshot{}
	}
}
```

- [ ] **Step 9: Wire buildPromptSnapshots into startAIcallByRealtime and startAIcallByMessaging**

In `startAIcallByRealtime`, replace the `nil` metadata:

```go
snapshots := h.buildPromptSnapshots(ctx, a, assistanceType, assistanceID, activeflowID)
metadata := map[string]any{
    aicall.MetaKeyPromptSnapshots: snapshots,
}
res, err := h.Create(ctx, a, assistanceType, assistanceID, activeflowID, referenceType, referenceID,
    confbridgeID, pipecatcallID, currentMemberID, parameter, metadata)
```

In `startAIcallByMessaging`, replace the `nil` metadata:

```go
snapshots := h.buildPromptSnapshots(ctx, a, assistanceType, assistanceID, activeflowID)
metadata := map[string]any{
    aicall.MetaKeyPromptSnapshots: snapshots,
}
res, err := h.CreateByMessaging(ctx, a, assistanceType, assistanceID, activeflowID, referenceType, referenceID,
    pipecatcallID, currentMemberID, parameter, metadata)
```

- [ ] **Step 10: Run all aicallhandler tests**

```bash
go test ./pkg/aicallhandler/... 2>&1 | tail -5
```

Expected: PASS.

- [ ] **Step 11: Commit**

```bash
git add bin-ai-manager/pkg/aicallhandler/
git commit -m "NOJIRA-Add-aicall-metadata-prompt-version-tracking

- bin-ai-manager: Add resolveAIForTeam to fetch all team members' AIs concurrently
- bin-ai-manager: Add buildPromptSnapshots to construct PromptSnapshot slice at call start
- bin-ai-manager: Add metadata param to Create/CreateByMessaging; populate with prompt snapshots
- bin-ai-manager: Add tests for resolveAIForTeam partial-failure and metadata propagation"
```

---

### Task 7: RST Documentation Update

**Files:**
- Modify: relevant `.rst` files in `bin-api-manager/docsdev/source/`

- [ ] **Step 1: Locate AI and AIcall struct RST files**

```bash
ls ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aicall-metadata-prompt-version-tracking/bin-api-manager/docsdev/source/ | grep -E "ai_"
```

Identify the files for the AI struct fields and AIcall struct fields (e.g., `ai_struct_overview.rst`, `aicall_struct_overview.rst`).

- [ ] **Step 2: Add current_prompt_history_id to the AI struct doc**

In the AI struct RST file, add a row documenting the new field:

```rst
current_prompt_history_id
   string (UUID). UUID of the most-recent ``ai_ai_prompt_histories`` entry. Zero UUID
   (``00000000-0000-0000-0000-000000000000``) means no versioned history has been
   recorded for this AI yet.
```

- [ ] **Step 3: Add metadata and PromptSnapshot to the AIcall struct doc**

In the AIcall struct RST file, add:

```rst
metadata
   object. Generic key-value store attached to this AIcall. At call start time the
   key ``prompt_snapshots`` holds an array of ``PromptSnapshot`` objects (one per AI
   participant). Additional audit or operational data may appear under other keys in
   future releases.

PromptSnapshot object fields:

- ``ai_id`` (string/UUID): ID of the AI configuration.
- ``prompt_history_id`` (string/UUID): ID of the ``ai_ai_prompt_histories`` entry in
  effect at call start. Zero UUID if no history entry exists yet.
- ``prompt`` (string): Final variable-substituted ``init_prompt`` as sent to the LLM.
- ``member_id`` (string/UUID): Team member UUID for team calls; zero UUID for
  single-AI calls.
```

- [ ] **Step 4: Clean rebuild and force-add build output**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aicall-metadata-prompt-version-tracking/bin-api-manager/docsdev
rm -rf build && python3 -m sphinx -M html source build
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aicall-metadata-prompt-version-tracking
git add -f bin-api-manager/docsdev/build/
```

Expected: Sphinx builds without errors.

- [ ] **Step 5: Commit**

```bash
git add bin-api-manager/docsdev/source/
git commit -m "NOJIRA-Add-aicall-metadata-prompt-version-tracking

- bin-api-manager: Document current_prompt_history_id in AI struct RST
- bin-api-manager: Document metadata and PromptSnapshot in AIcall struct RST
- bin-api-manager: Rebuild Sphinx HTML docs"
```

---

### Task 8: OpenAPI Spec Update

**Files:**
- Modify: schema YAML in `bin-openapi-manager/`
- Modify: `bin-api-manager/` (reference regenerated types if needed)

- [ ] **Step 1: Locate AI and AIcall schemas in the OpenAPI YAML**

```bash
grep -rn "\"AI\"\|\"AIcall\"\|current_prompt_history_id\|metadata" \
    ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aicall-metadata-prompt-version-tracking/bin-openapi-manager/ | grep -v vendor | head -20
```

Identify the YAML files and the property blocks for `AI` and `AIcall`.

- [ ] **Step 2: Add current_prompt_history_id to AI schema**

In the `AI` schema properties block, add:

```yaml
current_prompt_history_id:
  type: string
  format: uuid
  example: "00000000-0000-0000-0000-000000000000"
  description: >
    UUID of the most-recent prompt history entry.
    Zero UUID means no versioned history has been recorded yet.
```

- [ ] **Step 3: Add metadata to AIcall schema and add PromptSnapshot component**

In the `AIcall` schema properties block, add:

```yaml
metadata:
  type: object
  additionalProperties: true
  description: >
    Generic key-value store. Contains prompt_snapshots (array of PromptSnapshot)
    at call start time.
```

In the `components/schemas` section, add a new `PromptSnapshot` component:

```yaml
PromptSnapshot:
  type: object
  properties:
    ai_id:
      type: string
      format: uuid
    prompt_history_id:
      type: string
      format: uuid
      description: Zero UUID means no history entry exists yet.
    prompt:
      type: string
      description: Variable-substituted init_prompt as sent to the LLM.
    member_id:
      type: string
      format: uuid
      description: Zero UUID for single-AI calls; team member UUID for team calls.
```

- [ ] **Step 4: Regenerate Go types**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aicall-metadata-prompt-version-tracking/bin-openapi-manager
go generate ./...
```

Expected: Updated Go type files.

- [ ] **Step 5: Run verification for bin-openapi-manager**

```bash
go mod tidy && go mod vendor && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: PASS.

- [ ] **Step 6: Run verification for bin-api-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aicall-metadata-prompt-version-tracking/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add bin-openapi-manager/ bin-api-manager/
git commit -m "NOJIRA-Add-aicall-metadata-prompt-version-tracking

- bin-openapi-manager: Add current_prompt_history_id to AI schema
- bin-openapi-manager: Add metadata field to AIcall schema
- bin-openapi-manager: Add PromptSnapshot component schema
- bin-api-manager: Update to reference regenerated OpenAPI types"
```

---

### Task 9: Full Verification for bin-ai-manager

- [ ] **Step 1: Run full verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aicall-metadata-prompt-version-tracking/bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All five steps pass cleanly. Pay special attention to:
- `go generate ./...` may regenerate `mock_main.go` files — if so, the mock for `AIcallHandler` interface may change. Review the diff; if the mock changed due to our additions, commit it.
- `golangci-lint` must pass with zero errors.

- [ ] **Step 2: If mocks changed, commit them**

```bash
git add bin-ai-manager/pkg/aicallhandler/mock_main.go bin-ai-manager/pkg/aihandler/mock_main.go
git diff --cached --stat
git commit -m "NOJIRA-Add-aicall-metadata-prompt-version-tracking

- bin-ai-manager: Regenerate mocks after interface and struct changes"
```

---

### Task 10: Pre-PR Check and PR Creation

- [ ] **Step 1: Fetch latest main and check for conflicts**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-aicall-metadata-prompt-version-tracking
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

Expected: No conflicts printed. If conflicts exist, rebase onto `origin/main`, resolve them, and re-run Task 9.

- [ ] **Step 2: Create the PR**

```bash
gh pr create \
  --title "NOJIRA-Add-aicall-metadata-prompt-version-tracking" \
  --body "$(cat <<'EOF'
Add metadata field to AIcall and current_prompt_history_id to AI for prompt version tracking. Prerequisite for the VoIPBin Assistants Audit feature.

- bin-dbscheme-manager: Add current_prompt_history_id column to ai_ais table
- bin-dbscheme-manager: Add metadata JSON column to ai_aicalls table
- bin-ai-manager: Add CurrentPromptHistoryID field to AI model and FieldCurrentPromptHistoryID constant
- bin-ai-manager: Add PromptSnapshot struct, MetaKeyPromptSnapshots constant, Metadata field to AIcall model
- bin-ai-manager: Add FieldMetadata constant; expose Metadata in WebhookMessage and ConvertWebhookMessage
- bin-ai-manager: Restructure AI Create to write AI row with pre-generated history ID before writing history row
- bin-ai-manager: Restructure AI Update into three branches (changed/cleared/unchanged) each maintaining current_prompt_history_id atomically
- bin-ai-manager: Add resolveAIForTeam helper to fetch all team members' AIs concurrently with partial-failure tolerance
- bin-ai-manager: Add buildPromptSnapshots to build PromptSnapshot slice at call start for both AI and Team assistance types
- bin-ai-manager: Add metadata param to Create/CreateByMessaging; pass prompt snapshots at call creation
- bin-api-manager: Update RST docs with current_prompt_history_id and metadata fields; rebuild Sphinx HTML
- bin-openapi-manager: Add new fields to OpenAPI schema; add PromptSnapshot component; regenerate Go types
EOF
)"
```

Expected: PR URL printed. Do NOT merge without explicit user instruction.
