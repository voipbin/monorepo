# Store Evaluated Message IDs on AI Audits — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `message_ids` JSON column to `ai_ai_audits` and populate it with the IDs of messages sent to Gemini during a completed audit evaluation.

**Architecture:** A new `message_ids JSON NULL` column is added to `ai_ai_audits`. The `AIAuditUpdateFinal` DB function gains a `messageIDs []uuid.UUID` parameter which is marshalled and stored only on successful completion. The audit handler collects IDs after `MessageList` and passes them only in the Step-7 success block. The field is exposed via `WebhookMessage` and OpenAPI schema.

**Tech Stack:** Go, MySQL (Alembic migrations), `oapi-codegen` v2, Gomock, RST/Sphinx

---

## CRITICAL RULES (non-negotiable throughout all tasks)

- **ALWAYS work in the worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids/` — NEVER edit files in `~/gitvoipbin/monorepo` directly
- **NEVER commit to `main`** — always on branch `NOJIRA-aiaudit-store-message-ids`
- **NEVER merge without explicit user instruction saying "merge"**
- **ALWAYS squash merge:** `gh pr merge <pr-number> --squash --delete-branch`
- **NEVER run `alembic upgrade` or `alembic downgrade`** — create migration files only
- **NEVER manually create Alembic migration files with hand-picked revision IDs** — always use `alembic revision`
- **NEVER save account keys, API keys, or credentials to any file**
- **vendor/ is NOT committed to git** — do not use `git add -f` for vendor files
- **Verification before every commit:** `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`

---

## File Map

| File | Action |
|---|---|
| `bin-dbscheme-manager/bin-manager/main/versions/<rev>_add_message_ids_to_ai_ai_audits.py` | Create (Alembic) |
| `bin-openapi-manager/openapi/openapi.yaml` | Modify — add `message_ids` field to `AIManagerAIAudit` |
| `bin-openapi-manager/gens/models/gen.go` | Regenerated (do not edit manually) |
| `bin-ai-manager/models/aiaudit/main.go` | Modify — add `MessageIDs` field |
| `bin-ai-manager/models/aiaudit/webhook.go` | Modify — add to `WebhookMessage` + `ConvertWebhookMessage` |
| `bin-ai-manager/pkg/dbhandler/main.go` | Modify — update interface signature |
| `bin-ai-manager/pkg/dbhandler/aiaudit.go` | Modify — implement new `AIAuditUpdateFinal`, add reset to `AIAuditUpsert` |
| `bin-ai-manager/pkg/dbhandler/mock_main.go` | Regenerated via `go generate ./...` |
| `bin-ai-manager/pkg/dbhandler/aiaudit_test.go` | Modify — update existing test, add round-trip test |
| `bin-ai-manager/pkg/aiaudithandler/mock_main.go` | Regenerated via `go generate ./...` |
| `bin-ai-manager/pkg/aiaudithandler/main.go` | Modify — `runAuditJob` + `SweepStaleAudits` |
| `bin-ai-manager/pkg/aiaudithandler/main_test.go` | Modify — update happy-path mock arg count, add completed-path test |
| `bin-api-manager/docsdev/source/ai_struct_aiaudit.rst` | Modify — add `message_ids` to schema, bullets, example |
| `bin-api-manager/docsdev/build/` | Regenerated (force-add to git) |
| `bin-ai-manager/docs/domain.md` | Modify — add `MessageIDs` to AIAudit entity table |

---

## Task 1: Alembic Migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<rev>_add_message_ids_to_ai_ai_audits.py`

- [ ] **Step 1: Generate the migration file**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids/bin-dbscheme-manager/bin-manager/main
alembic -c ../alembic.ini revision -m "add_message_ids_to_ai_ai_audits"
```

Expected: A new file `versions/<hex>_add_message_ids_to_ai_ai_audits.py` is created. The revision ID is auto-generated (e.g. `a1b2c3d4e5f6`).

- [ ] **Step 2: Fill in the migration body**

Open the generated file and replace the placeholder `upgrade()` and `downgrade()` functions:

```python
def upgrade():
    op.execute("""
        ALTER TABLE ai_ai_audits
          ADD COLUMN message_ids JSON NULL
          AFTER evaluation
    """)


def downgrade():
    op.execute("""
        ALTER TABLE ai_ai_audits
          DROP COLUMN message_ids
    """)
```

Do NOT run `alembic upgrade` — leave that to the human operator.

- [ ] **Step 3: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids
git add bin-dbscheme-manager/bin-manager/main/versions/
git commit -m "NOJIRA-aiaudit-store-message-ids

- bin-dbscheme-manager: Add migration to add message_ids JSON NULL column to ai_ai_audits"
```

---

## Task 2: OpenAPI Schema Update

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (after line 2837, before `status:`)
- Regenerate: `bin-openapi-manager/gens/models/gen.go`

- [ ] **Step 1: Add `message_ids` field to `AIManagerAIAudit` schema**

In `bin-openapi-manager/openapi/openapi.yaml`, locate the `prompt_history_id:` block (~line 2832). Add the following block **after** `prompt_history_id:` and **before** `status:`:

```yaml
        message_ids:
          type: array
          nullable: true
          items:
            type: string
            format: uuid
            x-go-type: string
          description: >
            Ordered list of message IDs (newest-first) that were evaluated by Gemini.
            Null while progressing, on failure, or for audits completed before this feature.
            Present and non-empty on successful completion for calls with messages.
          example:
            - "550e8400-e29b-41d4-a716-446655440001"
            - "550e8400-e29b-41d4-a716-446655440002"
```

- [ ] **Step 2: Validate the spec**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids/bin-openapi-manager
oapi-codegen -config configs/config_model/config.generate.yaml openapi/openapi.yaml > /dev/null
```

Expected: no output (exit 0). Fix any YAML indentation errors if the command fails.

- [ ] **Step 3: Regenerate `gen.go`**

```bash
go generate ./...
```

Expected: `gens/models/gen.go` is updated with a `MessageIds []string` field (or `*[]string` depending on nullability) on the `AIManagerAIAudit` struct.

- [ ] **Step 4: Verify consumer build**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids/bin-api-manager
go build ./...
```

Expected: build succeeds.

- [ ] **Step 5: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids
git add bin-openapi-manager/openapi/openapi.yaml bin-openapi-manager/gens/models/gen.go
git commit -m "NOJIRA-aiaudit-store-message-ids

- bin-openapi-manager: Add message_ids field to AIManagerAIAudit schema and regenerate gen.go"
```

---

## Task 3: Model and Webhook Fields

**Files:**
- Modify: `bin-ai-manager/models/aiaudit/main.go` (after line 42)
- Modify: `bin-ai-manager/models/aiaudit/webhook.go`

- [ ] **Step 1: Add `MessageIDs` to `AIAudit` struct**

In `bin-ai-manager/models/aiaudit/main.go`, after the `Evaluation` field (line 42), insert:

```go
	MessageIDs []uuid.UUID `json:"message_ids,omitempty" db:"message_ids,json"`
```

The struct should look like:

```go
	OverallScore *int            `json:"overall_score"     db:"overall_score"`
	Evaluation   json.RawMessage `json:"evaluation"        db:"evaluation,json"`
	MessageIDs   []uuid.UUID     `json:"message_ids,omitempty" db:"message_ids,json"`
	Language     string          `json:"language,omitempty" db:"language"`
```

- [ ] **Step 2: Add `MessageIDs` to `WebhookMessage`**

In `bin-ai-manager/models/aiaudit/webhook.go`, after the `Evaluation` field (line 20), add:

```go
	MessageIDs   []uuid.UUID     `json:"message_ids,omitempty"`
```

- [ ] **Step 3: Copy `MessageIDs` in `ConvertWebhookMessage`**

In `bin-ai-manager/models/aiaudit/webhook.go`, in the `ConvertWebhookMessage` function, add after `Evaluation: a.Evaluation,`:

```go
		MessageIDs:      a.MessageIDs,
```

The full return block should be:

```go
	return &WebhookMessage{
		Identity:        a.Identity,
		AIcallID:        a.AIcallID,
		AIID:            a.AIID,
		PromptHistoryID: a.PromptHistoryID,
		Status:          a.Status,
		OverallScore:    a.OverallScore,
		Evaluation:      a.Evaluation,
		MessageIDs:      a.MessageIDs,
		Language:        a.Language,
		Error:           a.Error,
		TMCreate:        a.TMCreate,
		TMUpdate:        a.TMUpdate,
		TMDelete:        a.TMDelete,
	}
```

- [ ] **Step 4: Quick compile check**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids/bin-ai-manager
go build ./models/...
```

Expected: exit 0.

---

## Task 4: DB Handler — Failing Test First (TDD)

**Files:**
- Modify: `bin-ai-manager/pkg/dbhandler/aiaudit_test.go`

The existing `Test_AIAuditUpdateFinal_OnlyUpdatesWhenNotDeleted` test (line 117) calls `AIAuditUpdateFinal` with 6 args. After the interface change in Task 5, it will fail to compile. We write the **new round-trip test first** so both tests compile and pass together.

- [ ] **Step 1: Add the new round-trip test**

Append the following test to `bin-ai-manager/pkg/dbhandler/aiaudit_test.go`:

```go
func Test_AIAuditUpdateFinal_StoresMessageIDs(t *testing.T) {
	curTime := func() *time.Time {
		t := time.Date(2024, 6, 2, 12, 0, 0, 0, time.UTC)
		return &t
	}()

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}

	ctx := context.Background()
	tm := time.Date(2024, 6, 2, 10, 0, 0, 0, time.UTC)

	auditID := uuid.FromStringOrNil("aaaa0010-0010-0010-0010-000000000001")
	customerID := uuid.FromStringOrNil("bbbb0010-0010-0010-0010-000000000001")
	msgID1 := uuid.FromStringOrNil("cccc0010-0010-0010-0010-000000000001")
	msgID2 := uuid.FromStringOrNil("cccc0010-0010-0010-0010-000000000002")

	a := &aiaudit.AIAudit{
		Identity:        identity.Identity{ID: auditID, CustomerID: customerID},
		AIcallID:        uuid.FromStringOrNil("dddd0010-0010-0010-0010-000000000001"),
		AIID:            uuid.FromStringOrNil("eeee0010-0010-0010-0010-000000000001"),
		PromptHistoryID: uuid.FromStringOrNil("ffff0010-0010-0010-0010-000000000001"),
		Language:        "en-US",
	}
	insertTestAudit(t, a, aiaudit.StatusProgressing, nil, &tm)

	score := 90
	msgIDs := []uuid.UUID{msgID1, msgID2}

	mockUtil.EXPECT().TimeNow().Return(curTime)
	n, err := h.AIAuditUpdateFinal(ctx, auditID, aiaudit.StatusCompleted, &score, nil, "", msgIDs)
	if err != nil {
		t.Fatalf("AIAuditUpdateFinal error = %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 row updated, got %d", n)
	}

	got, getErr := h.AIAuditGet(ctx, auditID)
	if getErr != nil {
		t.Fatalf("AIAuditGet after update error = %v", getErr)
	}
	if got.Status != aiaudit.StatusCompleted {
		t.Errorf("expected status completed, got %s", got.Status)
	}
	if len(got.MessageIDs) != 2 {
		t.Fatalf("expected 2 message IDs, got %d", len(got.MessageIDs))
	}
	if got.MessageIDs[0] != msgID1 {
		t.Errorf("expected MessageIDs[0] = %s, got %s", msgID1, got.MessageIDs[0])
	}
	if got.MessageIDs[1] != msgID2 {
		t.Errorf("expected MessageIDs[1] = %s, got %s", msgID2, got.MessageIDs[1])
	}
}
```

- [ ] **Step 2: Verify the test fails to compile (expected)**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids/bin-ai-manager
go test ./pkg/dbhandler/... 2>&1 | head -20
```

Expected: compilation error like `too many arguments in call to h.AIAuditUpdateFinal`. This confirms the test is driving the implementation.

---

## Task 5: DB Handler — Interface + Implementation

**Files:**
- Modify: `bin-ai-manager/pkg/dbhandler/main.go`
- Modify: `bin-ai-manager/pkg/dbhandler/aiaudit.go`

- [ ] **Step 1: Update the interface in `main.go`**

In `bin-ai-manager/pkg/dbhandler/main.go`, find the `AIAuditUpdateFinal` line (~line 62) and change it to:

```go
AIAuditUpdateFinal(ctx context.Context, id uuid.UUID, status aiaudit.Status, overallScore *int, evaluation json.RawMessage, errStr string, messageIDs []uuid.UUID) (int64, error)
```

- [ ] **Step 2: Replace `AIAuditUpdateFinal` implementation in `aiaudit.go`**

Replace the function body at lines 185–217 with:

```go
func (h *handler) AIAuditUpdateFinal(ctx context.Context, id uuid.UUID, status aiaudit.Status, overallScore *int, evaluation json.RawMessage, errStr string, messageIDs []uuid.UUID) (int64, error) {
	ts := h.utilHandler.TimeNow()

	var evalJSON sql.NullString
	if evaluation != nil {
		evalJSON = sql.NullString{String: string(evaluation), Valid: true}
	}

	var msgIDsJSON sql.NullString
	if len(messageIDs) > 0 {
		b, err := json.Marshal(messageIDs)
		if err != nil {
			return 0, fmt.Errorf("AIAuditUpdateFinal: could not marshal message_ids: %w", err)
		}
		msgIDsJSON = sql.NullString{String: string(b), Valid: true}
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET status = ?, overall_score = ?, evaluation = ?, message_ids = ?, error = ?, tm_update = ?
		WHERE id = ? AND tm_delete IS NULL AND status = 'progressing'
	`, aiauditTable)

	result, err := h.db.ExecContext(ctx, query,
		string(status),  // 1
		overallScore,    // 2
		evalJSON,        // 3
		msgIDsJSON,      // 4
		errStr,          // 5
		ts,              // 6
		id.Bytes(),      // 7 (WHERE)
	)
	if err != nil {
		return 0, fmt.Errorf("AIAuditUpdateFinal: could not execute. err: %v", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("AIAuditUpdateFinal: could not get rows affected. err: %v", err)
	}

	return n, nil
}
```

- [ ] **Step 3: Add `message_ids = NULL` to `AIAuditUpsert` ON DUPLICATE KEY UPDATE**

In `aiaudit.go`, find the `ON DUPLICATE KEY UPDATE` block (lines 35–43). Add `message_ids = NULL,` after `evaluation = NULL,`:

```sql
		ON DUPLICATE KEY UPDATE
			status            = IF(status = 'progressing', status, 'progressing'),
			tm_delete         = NULL,
			overall_score     = NULL,
			evaluation        = NULL,
			message_ids       = NULL,
			error             = NULL,
			language          = VALUES(language),
			prompt_history_id = VALUES(prompt_history_id),
			tm_update         = NULL
```

- [ ] **Step 4: Update the existing DB test to pass 7 args**

In `bin-ai-manager/pkg/dbhandler/aiaudit_test.go`, find `Test_AIAuditUpdateFinal_OnlyUpdatesWhenNotDeleted` (line 117). The two calls to `AIAuditUpdateFinal` currently pass 6 args. Add `nil` as the 7th arg to both:

```go
// Live record call (was: h.AIAuditUpdateFinal(ctx, liveID, aiaudit.StatusCompleted, &score, nil, ""))
n, err := h.AIAuditUpdateFinal(ctx, liveID, aiaudit.StatusCompleted, &score, nil, "", nil)

// Soft-deleted record call (was: h.AIAuditUpdateFinal(ctx, deletedID, aiaudit.StatusCompleted, &score, nil, ""))
n, err = h.AIAuditUpdateFinal(ctx, deletedID, aiaudit.StatusCompleted, &score, nil, "", nil)
```

- [ ] **Step 5: Run DB tests — expect pass**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids/bin-ai-manager
go test ./pkg/dbhandler/... -v -run "Test_AIAuditUpdateFinal" 2>&1
```

Expected: both `Test_AIAuditUpdateFinal_OnlyUpdatesWhenNotDeleted` and `Test_AIAuditUpdateFinal_StoresMessageIDs` pass.

---

## Task 6: Regenerate Mocks + Fix Existing Handler Test Signature

**Files:**
- Regenerate: `bin-ai-manager/pkg/dbhandler/mock_main.go`
- Regenerate: `bin-ai-manager/pkg/aiaudithandler/mock_main.go`
- Modify: `bin-ai-manager/pkg/aiaudithandler/main_test.go`

- [ ] **Step 1: Regenerate mocks**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids/bin-ai-manager
go generate ./...
```

Expected: `pkg/dbhandler/mock_main.go` and `pkg/aiaudithandler/mock_main.go` are updated. The `AIAuditUpdateFinal` mock now accepts 7 args.

- [ ] **Step 2: Verify the existing `TestCreate_HappyPath` mock call fails to compile**

```bash
go test ./pkg/aiaudithandler/... 2>&1 | head -20
```

Expected: compilation error — the `AIAuditUpdateFinal` mock call in `TestCreate_HappyPath` still has 6 `gomock.Any()` matchers.

- [ ] **Step 3: Update `TestCreate_HappyPath` mock expectation**

In `bin-ai-manager/pkg/aiaudithandler/main_test.go`, find the line (~line 68):

```go
mockDB.EXPECT().AIAuditUpdateFinal(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(1), nil).AnyTimes()
```

Change it to add a 7th `gomock.Any()`:

```go
mockDB.EXPECT().AIAuditUpdateFinal(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(1), nil).AnyTimes()
```

- [ ] **Step 4: Run existing handler tests — expect pass**

```bash
go test ./pkg/aiaudithandler/... -v -run "TestCreate_HappyPath|TestGet_HappyPath" 2>&1
```

Expected: all listed tests pass.

---

## Task 7: New Completed-Path Test (TDD)

**Files:**
- Modify: `bin-ai-manager/pkg/aiaudithandler/main_test.go`

`TestCreate_HappyPath` cannot be reused: it stubs `MessageList → nil, nil` and `Evaluate → nil, nil, nil`, which causes the goroutine job to fail (the success block — Step 7 — is never reached). We need a dedicated test that exercises the happy-path goroutine.

- [ ] **Step 1: Add the completed-path test**

Append to `bin-ai-manager/pkg/aiaudithandler/main_test.go`:

```go
func TestCreate_CompletedPath_StoresMessageIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockGemini := geminiaudithandler.NewMockGeminiAuditHandler(ctrl)

	auditID := uuid.FromStringOrNil("11110001-0001-0001-0001-000000000001")
	customerID := uuid.FromStringOrNil("22220001-0001-0001-0001-000000000001")
	aicallID := uuid.FromStringOrNil("33330001-0001-0001-0001-000000000001")
	aiID := uuid.FromStringOrNil("44440001-0001-0001-0001-000000000001")
	msgID1 := uuid.FromStringOrNil("55550001-0001-0001-0001-000000000001")
	msgID2 := uuid.FromStringOrNil("55550001-0001-0001-0001-000000000002")

	// A minimal AIcall with an embedded PromptSnapshot for the AI.
	promptSnapshots, _ := json.Marshal([]aicall.PromptSnapshot{
		{
			AIID:            aiID,
			PromptHistoryID: uuid.FromStringOrNil("66660001-0001-0001-0001-000000000001"),
			Prompt:          "You are a helpful assistant.",
		},
	})
	ac := &aicall.AIcall{
		Identity: commonidentity.Identity{
			ID:         aicallID,
			CustomerID: customerID,
		},
		AIID:           aiID,
		AssistanceType: aicall.AssistanceTypeSolo,
		Metadata: map[string]any{
			"prompt_snapshots": string(promptSnapshots),
		},
	}

	expectedAudit := &aiaudit.AIAudit{
		Identity: commonidentity.Identity{
			ID:         auditID,
			CustomerID: customerID,
		},
		AIcallID: aicallID,
		AIID:     aiID,
		Status:   aiaudit.StatusProgressing,
	}

	msgs := []*message.Message{
		{Identity: commonidentity.Identity{ID: msgID1}},
		{Identity: commonidentity.Identity{ID: msgID2}},
	}

	score := 88
	evalResult := &geminiaudithandler.EvaluationResponse{OverallScore: score}
	rawJSON := json.RawMessage(`{"overall_score":88}`)

	// done channel used to synchronize with the background goroutine.
	done := make(chan struct{})

	mockDB.EXPECT().AIcallGet(gomock.Any(), aicallID).Return(ac, nil)
	mockDB.EXPECT().AIAuditCountProgressing(gomock.Any(), customerID).Return(int64(0), nil)
	mockDB.EXPECT().AIAuditUpsert(gomock.Any(), gomock.Any()).Return(int64(1), nil)
	mockDB.EXPECT().AIAuditList(gomock.Any(), uint64(1), "", gomock.Any()).Return([]*aiaudit.AIAudit{expectedAudit}, nil)
	mockDB.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(msgs, nil)
	mockGemini.EXPECT().Evaluate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(evalResult, rawJSON, nil)
	mockDB.EXPECT().AIAuditUpdateFinal(
		gomock.Any(),
		auditID,
		aiaudit.StatusCompleted,
		gomock.Any(),
		gomock.Any(),
		"",
		gomock.Any(),
	).DoAndReturn(func(_ context.Context, _ uuid.UUID, _ aiaudit.Status, _ *int, _ json.RawMessage, _ string, messageIDs []uuid.UUID) (int64, error) {
		if len(messageIDs) != 2 {
			t.Errorf("expected 2 message IDs, got %d", len(messageIDs))
		} else {
			if messageIDs[0] != msgID1 {
				t.Errorf("expected messageIDs[0] = %s, got %s", msgID1, messageIDs[0])
			}
			if messageIDs[1] != msgID2 {
				t.Errorf("expected messageIDs[1] = %s, got %s", msgID2, messageIDs[1])
			}
		}
		close(done)
		return int64(1), nil
	})

	h := aiaudithandler.NewAIAuditHandler(mockDB, mockGemini)
	_, err := h.Create(context.Background(), customerID, aicallID, "en-US")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for the goroutine to complete (timeout = 5s).
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for background audit goroutine to complete")
	}
}
```

The test imports needed (add to the import block if not already present):
- `"encoding/json"`
- `"time"`
- `message "monorepo/bin-ai-manager/models/message"`
- `"monorepo/bin-ai-manager/models/aicall"`
- `geminiaudithandler "monorepo/bin-ai-manager/pkg/geminiaudithandler"`

- [ ] **Step 2: Run the new test — expect fail**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids/bin-ai-manager
go test ./pkg/aiaudithandler/... -v -run "TestCreate_CompletedPath_StoresMessageIDs" 2>&1
```

Expected: test fails because `runAuditJob` does not yet collect or pass `finalMsgIDs`. Compilation errors mean Task 6 isn't complete yet.

---

## Task 8: Audit Handler — Collect and Pass `finalMsgIDs`

**Files:**
- Modify: `bin-ai-manager/pkg/aiaudithandler/main.go`

- [ ] **Step 1: Declare `finalMsgIDs` alongside other `final*` vars**

In `runAuditJob`, find the block where `finalStatus`, `finalScore`, etc. are declared (~line 198):

```go
finalStatus := aiaudit.StatusFailed
var finalScore *int
var finalEvalJSON json.RawMessage
finalErr := ""
```

Add after `finalErr := ""`:

```go
var finalMsgIDs []uuid.UUID  // nil unless audit completes successfully
```

- [ ] **Step 2: Collect message IDs after `MessageList` (Step 4)**

Find the line after the successful `MessageList` call (around line 269):

```go
log.Debugf("step4: loaded %d message(s)", len(msgs))
```

Add immediately after this log line:

```go
msgIDs := make([]uuid.UUID, len(msgs))
for i, m := range msgs {
    msgIDs[i] = m.ID
}
```

- [ ] **Step 3: Assign `finalMsgIDs` only in the Step-7 success block**

Find the Step-7 success block (around line 310):

```go
// Step 7: Success.
log.Debugf("step7: gemini evaluation complete score=%d raw_len=%d", result.OverallScore, len(rawJSON))
score := result.OverallScore
finalStatus = aiaudit.StatusCompleted
finalScore = &score
finalEvalJSON = rawJSON
```

Add after `finalEvalJSON = rawJSON`:

```go
finalMsgIDs = msgIDs  // only assigned on success; nil on every failure path
```

- [ ] **Step 4: Pass `finalMsgIDs` in the deferred `AIAuditUpdateFinal` call**

Find the deferred call (around line 213):

```go
n, dbErr := h.db.AIAuditUpdateFinal(writeCtx, recordID, finalStatus, finalScore, finalEvalJSON, finalErr)
```

Change it to:

```go
n, dbErr := h.db.AIAuditUpdateFinal(writeCtx, recordID, finalStatus, finalScore, finalEvalJSON, finalErr, finalMsgIDs)
```

- [ ] **Step 5: Pass `nil` in `SweepStaleAudits`**

Find the `SweepStaleAudits` call (around line 368):

```go
if _, dbErr := h.db.AIAuditUpdateFinal(ctx, a.ID, aiaudit.StatusFailed, nil, nil, string(aiaudit.ErrorEvaluatorUnavailable)); dbErr != nil {
```

Change it to:

```go
if _, dbErr := h.db.AIAuditUpdateFinal(ctx, a.ID, aiaudit.StatusFailed, nil, nil, string(aiaudit.ErrorEvaluatorUnavailable), nil); dbErr != nil {
```

- [ ] **Step 6: Run the completed-path test — expect pass**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids/bin-ai-manager
go test ./pkg/aiaudithandler/... -v -run "TestCreate_CompletedPath_StoresMessageIDs" 2>&1
```

Expected: PASS.

- [ ] **Step 7: Run all handler tests**

```bash
go test ./pkg/aiaudithandler/... -v 2>&1
```

Expected: all tests pass.

---

## Task 9: Full Verification + Commit (bin-ai-manager)

**Files:** All `bin-ai-manager` changes so far

- [ ] **Step 1: Run the full verification workflow**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids/bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: all steps exit 0 with no errors.

- [ ] **Step 2: Commit all bin-ai-manager changes**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids
git add bin-ai-manager/models/aiaudit/main.go \
        bin-ai-manager/models/aiaudit/webhook.go \
        bin-ai-manager/pkg/dbhandler/main.go \
        bin-ai-manager/pkg/dbhandler/aiaudit.go \
        bin-ai-manager/pkg/dbhandler/mock_main.go \
        bin-ai-manager/pkg/dbhandler/aiaudit_test.go \
        bin-ai-manager/pkg/aiaudithandler/mock_main.go \
        bin-ai-manager/pkg/aiaudithandler/main.go \
        bin-ai-manager/pkg/aiaudithandler/main_test.go \
        bin-ai-manager/go.mod bin-ai-manager/go.sum
git commit -m "NOJIRA-aiaudit-store-message-ids

- bin-ai-manager: Add MessageIDs field to AIAudit model and WebhookMessage
- bin-ai-manager: Extend AIAuditUpdateFinal to accept and store message_ids
- bin-ai-manager: Reset message_ids in AIAuditUpsert ON DUPLICATE KEY UPDATE
- bin-ai-manager: Collect and pass finalMsgIDs in runAuditJob success path
- bin-ai-manager: Pass nil message_ids in SweepStaleAudits
- bin-ai-manager: Regenerate mocks; update and add tests"
```

---

## Task 10: RST Docs and Service Domain Doc

**Files:**
- Modify: `bin-api-manager/docsdev/source/ai_struct_aiaudit.rst`
- Modify: `bin-ai-manager/docs/domain.md`

- [ ] **Step 1: Update `ai_struct_aiaudit.rst` — schema code block**

In the schema code block (the JSON object showing field names and types), add after the `evaluation` line:

```
"message_ids":      [array of UUID strings, or null],
```

- [ ] **Step 2: Update `ai_struct_aiaudit.rst` — field description list**

In the field description bullet list, add after the `evaluation` bullet:

```rst
* **message_ids** (*array of strings, nullable*) —
  Ordered list (newest-first) of the message IDs evaluated by Gemini.
  ``null`` while progressing, on failure, or for audits created before this feature.
  Present and non-empty on successful completion for calls with messages.
  Use ``status = completed`` as the authoritative completion signal — ``message_ids`` absent
  on a completed audit means the call had zero messages.
```

- [ ] **Step 3: Update `ai_struct_aiaudit.rst` — example JSON block**

In the example JSON block showing a completed audit, add `message_ids` after `evaluation`:

```json
"message_ids": [
  "550e8400-e29b-41d4-a716-446655440001",
  "550e8400-e29b-41d4-a716-446655440002"
],
```

- [ ] **Step 4: Update `bin-ai-manager/docs/domain.md` — AIAudit entity table**

In the AIAudit entity table (starting at line 153), add a row after `evaluation`:

```
| message_ids | JSON array / null | IDs (newest-first) of messages sent to Gemini; `null` on failure, historical records, or zero-message calls |
```

- [ ] **Step 5: Clean rebuild of RST HTML**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids/bin-api-manager/docsdev
rm -rf build
python3 -m sphinx -M html source build
```

Expected: build completes with no errors (warnings about external links are OK).

- [ ] **Step 6: Commit RST source, built HTML, and domain.md together**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids
git add -f bin-api-manager/docsdev/build/
git add bin-api-manager/docsdev/source/ai_struct_aiaudit.rst
git add bin-ai-manager/docs/domain.md
git commit -m "NOJIRA-aiaudit-store-message-ids

- bin-api-manager: Update ai_struct_aiaudit.rst with message_ids field (schema, bullets, example)
- bin-ai-manager: Update domain.md AIAudit entity table with MessageIDs field"
```

---

## Task 11: Final Cross-Service Verification + PR

**Files:** All services

- [ ] **Step 1: Verify bin-openapi-manager**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids/bin-openapi-manager
oapi-codegen -config configs/config_model/config.generate.yaml openapi/openapi.yaml > /dev/null
go generate ./...
go build ./...
```

Expected: all exit 0.

- [ ] **Step 2: Verify bin-ai-manager**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids/bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: all exit 0.

- [ ] **Step 3: Verify bin-api-manager**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: all exit 0.

- [ ] **Step 4: Fetch latest main and check for conflicts**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-aiaudit-store-message-ids
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

Expected: no conflicts. If conflicts exist, rebase, resolve, and re-run verification for affected services before continuing.

- [ ] **Step 5: Create the PR**

```bash
gh pr create \
  --title "NOJIRA-aiaudit-store-message-ids" \
  --body "$(cat <<'EOF'
Store the list of message IDs evaluated by Gemini as part of each completed AI audit record.

- bin-dbscheme-manager: Add Alembic migration to add message_ids JSON NULL column to ai_ai_audits
- bin-openapi-manager: Add message_ids field to AIManagerAIAudit schema; regenerate gen.go
- bin-ai-manager: Add MessageIDs []uuid.UUID field to AIAudit model and WebhookMessage
- bin-ai-manager: Extend AIAuditUpdateFinal interface and implementation to accept and store message_ids
- bin-ai-manager: Reset message_ids = NULL in AIAuditUpsert ON DUPLICATE KEY UPDATE (re-audit safety)
- bin-ai-manager: Collect msgIDs in runAuditJob after MessageList; assign finalMsgIDs only in Step-7 success block
- bin-ai-manager: Pass nil message_ids in SweepStaleAudits (failed path)
- bin-ai-manager: Regenerate mocks; update TestCreate_HappyPath signature; add round-trip DB test and completed-path goroutine test
- bin-api-manager: Update ai_struct_aiaudit.rst with message_ids field (schema, description, example); rebuild HTML
- bin-ai-manager: Update docs/domain.md AIAudit entity table with MessageIDs field
EOF
)"
```

**Do NOT merge.** Wait for the user to explicitly say "merge" before running `gh pr merge`.

---

## Self-Review Checklist

- [x] **Spec coverage:** All 6 spec sections are covered: migration, OpenAPI, model/webhook, DB handler (write + upsert reset), audit handler (`runAuditJob` + `SweepStaleAudits`), and docs (RST + domain.md)
- [x] **Placeholder scan:** No TBD, TODO, or "similar to task N" references. Every code block is complete.
- [x] **Type consistency:** `[]uuid.UUID` used throughout — model field, webhook field, interface param, implementation param, test assertions. `msgIDs` (local) and `finalMsgIDs` (package-local final var) naming is consistent between Task 8 steps.
- [x] **Second caller:** `SweepStaleAudits` passing `nil` as 7th arg is explicitly covered in Task 8 Step 5.
- [x] **Re-audit safety:** `message_ids = NULL` reset in `AIAuditUpsert` is covered in Task 5 Step 3.
- [x] **TDD ordering:** Failing test written in Task 4 before implementation in Task 5; goroutine test written in Task 7 before implementation in Task 8.
- [x] **CRITICAL rules preserved:** Alembic, vendor, worktrees, no-merge, squash-merge rules all explicitly stated.
