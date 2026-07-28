# Insight AI: enforce one per customer — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enforce "at most one non-deleted `type=insight` AI per customer" at the DB level in `bin-ai-manager`, with a clear 409 API error on violation, per [SQUARE-23](https://voipbin.atlassian.net/browse/SQUARE-23).

**Architecture:** A `STORED` generated column (`active_insight_key`) plus a `UNIQUE INDEX` on `ai_ais`, mirroring the existing `ai_aicalls.active_reference_key` precedent (VOIP-1234). `aihandler.Create`/`Update` translate the resulting duplicate-key DB error into `cerrors.AlreadyExists` (HTTP 409) via the existing `dbhandler.IsErrDuplicate` helper.

**Tech Stack:** Go, MySQL (production) / SQLite (`go-sqlite3`, dbhandler unit tests), Alembic (`bin-dbscheme-manager`), gomock.

**Design doc:** [docs/plans/2026-07-29-insight-ai-one-per-customer-design.md](2026-07-29-insight-ai-one-per-customer-design.md)

---

## Task 1: SQLite test schema + dbhandler regression tests

**Files:**
- Modify: `bin-ai-manager/scripts/database_scripts_test/table_ai_ais.sql`
- Modify: `bin-ai-manager/pkg/dbhandler/ai_test.go`

- [ ] **Step 1: Write the failing test**

Add this test to the end of `bin-ai-manager/pkg/dbhandler/ai_test.go` (after the existing `Test_AIDelete` function — check the file's last line number first with `tail -5 bin-ai-manager/pkg/dbhandler/ai_test.go` and append after it):

```go
func Test_AICreate_InsightUniquePerCustomer(t *testing.T) {
	ctx := context.Background()

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockCache.EXPECT().AISet(ctx, gomock.Any()).AnyTimes()

	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}

	customerID := uuid.FromStringOrNil("a1a1a1a1-0000-0000-0000-000000000001")

	// first insight AI for the customer succeeds
	firstID := uuid.FromStringOrNil("a1a1a1a1-0000-0000-0000-000000000002")
	t1 := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&t1)
	first := &ai.AI{
		Identity: identity.Identity{ID: firstID, CustomerID: customerID},
		Type:     ai.TypeInsight,
	}
	if err := h.AICreate(ctx, first); err != nil {
		t.Fatalf("first insight AICreate: expected ok, got: %v", err)
	}

	// second insight AI for the SAME customer must be rejected
	secondID := uuid.FromStringOrNil("a1a1a1a1-0000-0000-0000-000000000003")
	t2 := time.Date(2026, 7, 29, 0, 0, 1, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&t2)
	second := &ai.AI{
		Identity: identity.Identity{ID: secondID, CustomerID: customerID},
		Type:     ai.TypeInsight,
	}
	err := h.AICreate(ctx, second)
	if err == nil {
		t.Fatal("second insight AICreate: expected an error, got nil")
	}
	if !IsErrDuplicate(err) {
		t.Errorf("second insight AICreate: expected IsErrDuplicate(err)=true, got err: %v", err)
	}

	// a normal-type AI for the same customer is NOT affected by the constraint
	normalID := uuid.FromStringOrNil("a1a1a1a1-0000-0000-0000-000000000004")
	t3 := time.Date(2026, 7, 29, 0, 0, 2, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&t3)
	normal := &ai.AI{
		Identity: identity.Identity{ID: normalID, CustomerID: customerID},
		Type:     ai.TypeNormal,
	}
	if err := h.AICreate(ctx, normal); err != nil {
		t.Errorf("normal-type AICreate for same customer: expected ok, got: %v", err)
	}

	// soft-delete the first insight AI, freeing the slot
	t4 := time.Date(2026, 7, 29, 0, 0, 3, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&t4)
	mockCache.EXPECT().AIGet(ctx, firstID).Return(nil, fmt.Errorf(""))
	mockCache.EXPECT().AISet(ctx, gomock.Any())
	if err := h.AIDelete(ctx, firstID); err != nil {
		t.Fatalf("AIDelete(first): expected ok, got: %v", err)
	}

	// a new insight AI for the same customer now succeeds
	thirdID := uuid.FromStringOrNil("a1a1a1a1-0000-0000-0000-000000000005")
	t5 := time.Date(2026, 7, 29, 0, 0, 4, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&t5)
	third := &ai.AI{
		Identity: identity.Identity{ID: thirdID, CustomerID: customerID},
		Type:     ai.TypeInsight,
	}
	if err := h.AICreate(ctx, third); err != nil {
		t.Errorf("insight AICreate after soft-delete: expected ok, got: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd bin-ai-manager && go test ./pkg/dbhandler/... -run Test_AICreate_InsightUniquePerCustomer -v`
Expected: FAIL at the `if !IsErrDuplicate(err)` check — `ai_ais` has no uniqueness constraint yet, so the second insight `AICreate` currently succeeds and `err` is `nil` on the "expected an error, got nil" line.

- [ ] **Step 3: Add the SQLite stand-in generated column + unique index**

Edit `bin-ai-manager/scripts/database_scripts_test/table_ai_ais.sql`. Insert the new generated column right before `primary key(id)`, and add the unique index alongside the other `create index` statements at the bottom:

```sql
create table ai_ais(
  -- identity
  id            binary(16),   -- id
  customer_id   binary(16),   -- customer id

  -- info
  name        varchar(255),   -- name
  detail      text,           -- detail description

  engine_type   varchar(255),
  engine_model  varchar(255),
  parameter     json,
  engine_key    varchar(255),
  rag_id        binary(16),

  init_prompt   text,           -- initial prompt

  tts_type      varchar(255),
  tts_voice_id  varchar(255),

  stt_type      varchar(255),
  stt_language  varchar(255),

  vad_config  json,           -- VAD configuration

  smart_turn_enabled  boolean not null default 0,      -- smart turn detection enabled

  auto_aicall_audit_enabled  boolean not null default 0,   -- auto aicall audit enabled

  type  varchar(255) not null default 'normal',   -- ai type: normal, insight

  tool_names  json,           -- enabled tools for this AI

  current_prompt_history_id  binary(16),   -- current prompt history id

  direct_id     binary(16),     -- direct id
  direct_hash   varchar(255),   -- direct hash

  -- timestamps
  tm_create datetime(6),  --
  tm_update datetime(6),  --
  tm_delete datetime(6),  --

  -- SQLite-compatible stand-in for the MySQL STORED generated column
  -- active_insight_key (BINARY(16), see bin-dbscheme-manager migration
  -- for SQUARE-23, mirroring the active_reference_key pattern in
  -- table_ai_aicalls.sql / a5a40c93d3e6). Computes customer_id only when
  -- type='insight' and the row is not soft-deleted; NULL otherwise (any
  -- number of NULLs may coexist under UNIQUE), enforcing "at most one
  -- active Insight AI per customer" without affecting type='normal' rows.
  active_insight_key varchar(255) GENERATED ALWAYS AS (
    CASE WHEN type = 'insight' AND tm_delete IS NULL
      THEN customer_id
      ELSE NULL
    END
  ) STORED,

  primary key(id)
);

create index idx_ai_ais_create on ai_ais(tm_create);
create index idx_ai_ais_customer_id on ai_ais(customer_id);
create unique index uq_ai_active_insight_key on ai_ais(active_insight_key);
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd bin-ai-manager && go test ./pkg/dbhandler/... -run Test_AICreate_InsightUniquePerCustomer -v`
Expected: PASS

- [ ] **Step 5: Run the full dbhandler package tests to check for regressions**

Run: `cd bin-ai-manager && go test ./pkg/dbhandler/... -v`
Expected: PASS (all existing `ai_ais`-related tests, e.g. `Test_AICreate`, `Test_AIDelete`, still pass — they use `type=""`/default, which computes `NULL` and is unaffected)

- [ ] **Step 6: Commit**

```bash
cd bin-ai-manager
git add scripts/database_scripts_test/table_ai_ais.sql pkg/dbhandler/ai_test.go
git commit -m "$(cat <<'EOF'
SQUARE-23-Insight-AI-one-per-customer

- bin-ai-manager: Add active_insight_key SQLite test-schema stand-in and regression test for one-insight-AI-per-customer
EOF
)"
```

---

## Task 2: Production Alembic migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<generated>_ai_ais_add_active_insight_key_.py`

- [ ] **Step 1: Check the current migration head**

Run: `cd bin-dbscheme-manager/bin-manager && alembic -c alembic.ini heads`
Expected: exactly one head revision printed (note its ID — `alembic revision` will chain `down_revision` to it automatically).

If `alembic.ini` does not exist yet in this worktree, copy it first: `cp alembic.ini.sample alembic.ini`, then edit `sqlalchemy.url` to point at a local database before running any Alembic command (per `bin-dbscheme-manager/CLAUDE.md`). Do not point it at staging or production.

- [ ] **Step 2: Generate the migration file**

Run: `cd bin-dbscheme-manager/bin-manager && alembic -c alembic.ini revision -m "ai_ais_add_active_insight_key_unique"`
Expected: a new file appears in `bin-dbscheme-manager/bin-manager/main/versions/`, e.g. `<hash>_ai_ais_add_active_insight_key_unique.py`, with `down_revision` already set to the head from Step 1.

Never hand-author this file's `revision`/`down_revision` IDs — always generate via this command (`bin-dbscheme-manager/CLAUDE.md`).

- [ ] **Step 3: Fill in `upgrade()` / `downgrade()`**

Edit the generated file. Replace its body with (keep the auto-generated `revision`/`down_revision`/`branch_labels`/`depends_on` values at the top exactly as generated — do not edit those):

```python
"""ai_ais_add_active_insight_key_unique

SQUARE-23: adds active_insight_key to ai_ais as a STORED generated column
to enforce "at most one active (non-deleted) type=insight AI per
customer" at the DB level, mirroring the ai_aicalls.active_reference_key
pattern (a5a40c93d3e6_ai_aicalls_add_active_reference_key_.py, VOIP-1234).

active_insight_key carries customer_id only when type='insight' AND
tm_delete IS NULL; all other rows (type='normal', or soft-deleted
type='insight' rows) compute NULL, which MySQL treats as distinct under
UNIQUE, so any number of such rows may coexist while at most one
genuinely-active Insight AI may exist per customer.

Before applying this migration to any non-local environment: confirm no
customer currently has 2+ non-deleted type='insight' AIs (see SQUARE-23
design doc §3 for the audit query and cleanup approach — extras must be
removed via the existing DELETE /ais/{id} API, not raw SQL, before this
migration's CREATE UNIQUE INDEX can succeed), and confirm ai_ais's row
count / the expected ALTER TABLE ... ADD COLUMN ... STORED rewrite
duration are acceptable for an in-place lock, per the precedent's
documented incident (see design doc §1 "Operational cost").
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '<KEEP THE AUTO-GENERATED VALUE>'
down_revision = '<KEEP THE AUTO-GENERATED VALUE>'
branch_labels = None
depends_on = None


def _column_exists(conn, table, column):
    result = conn.execute(sa.text(
        "SELECT COUNT(*) FROM information_schema.columns "
        "WHERE table_schema = DATABASE() AND table_name = :table AND column_name = :col"
    ), {'table': table, 'col': column})
    return result.scalar() > 0


def _index_exists(conn, table, index):
    result = conn.execute(sa.text(
        "SELECT COUNT(*) FROM information_schema.statistics "
        "WHERE table_schema = DATABASE() AND table_name = :table AND index_name = :idx"
    ), {'table': table, 'idx': index})
    return result.scalar() > 0


def upgrade():
    conn = op.get_bind()

    if not _column_exists(conn, 'ai_ais', 'active_insight_key'):
        op.execute("""
            ALTER TABLE ai_ais
            ADD COLUMN active_insight_key BINARY(16) GENERATED ALWAYS AS (
                IF(type = 'insight' AND tm_delete IS NULL, customer_id, NULL)
            ) STORED
        """)

    if not _index_exists(conn, 'ai_ais', 'uq_ai_active_insight_key'):
        op.execute("""
            CREATE UNIQUE INDEX uq_ai_active_insight_key
            ON ai_ais(active_insight_key)
        """)


def downgrade():
    conn = op.get_bind()

    if _index_exists(conn, 'ai_ais', 'uq_ai_active_insight_key'):
        op.execute("""
            DROP INDEX uq_ai_active_insight_key ON ai_ais
        """)

    if _column_exists(conn, 'ai_ais', 'active_insight_key'):
        op.execute("""
            ALTER TABLE ai_ais
            DROP COLUMN active_insight_key
        """)
```

Keep the `revision`/`down_revision` values exactly as `alembic revision` generated them in Step 2 — do not paste over them with the placeholder text shown above.

- [ ] **Step 4: Verify exactly one head remains**

Run: `cd bin-dbscheme-manager/bin-manager && alembic -c alembic.ini heads`
Expected: exactly one head — the newly created revision.

Do **not** run `alembic upgrade` or `alembic downgrade` against anything but a local throwaway database (per `bin-dbscheme-manager/CLAUDE.md`). Applying this migration to staging/production is a separate, human-authorized step outside this plan.

- [ ] **Step 5: Commit**

```bash
cd bin-dbscheme-manager
git add bin-manager/main/versions/*ai_ais_add_active_insight_key_unique.py
git commit -m "$(cat <<'EOF'
SQUARE-23-Insight-AI-one-per-customer

- bin-dbscheme-manager: Add ai_ais.active_insight_key migration enforcing one active Insight AI per customer
EOF
)"
```

---

## Task 3: `aihandler.Create` — translate duplicate-key error

**Files:**
- Modify: `bin-ai-manager/pkg/aihandler/chatbot.go:1-18` (imports), `:74-79` (error check)
- Test: `bin-ai-manager/pkg/aihandler/chatbot_test.go`

- [ ] **Step 1: Write the failing test**

Add to `bin-ai-manager/pkg/aihandler/chatbot_test.go`, near `TestCreate_AllowsValidInsightAI` (around line 1088 — append after that function, before `TestUpdate_RejectsInsightAIWithNormalTools`):

```go
// assertAlreadyExists fails the test unless err is a *cerrors.VoipbinError
// with Status == cerrors.StatusAlreadyExists and the given Reason, per
// SQUARE-23's design (a duplicate-key failure on ai_ais.active_insight_key
// must be translated to a 409 ALREADY_EXISTS, not a raw SQL error).
func assertAlreadyExists(t *testing.T, err error, wantReason string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("expected a *cerrors.VoipbinError, got: %v (%T)", err, err)
	}
	if ve.Status != cerrors.StatusAlreadyExists {
		t.Errorf("expected Status=%v, got %v (err: %v)", cerrors.StatusAlreadyExists, ve.Status, err)
	}
	if ve.Reason != wantReason {
		t.Errorf("expected Reason=%v, got %v (err: %v)", wantReason, ve.Reason, err)
	}
}

// SQUARE-23: Create must translate a duplicate-key failure on
// ai_ais.active_insight_key into cerrors.AlreadyExists (HTTP 409), not a
// raw wrapped SQL error.
func TestCreate_RejectsDuplicateInsightAI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockReq := requesthandler.NewMockRequestHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)

	mockReq.EXPECT().DirectV1DirectCreate(gomock.Any(), gomock.Any(), dmdirect.ResourceTypeAI, gomock.Any()).
		Return(&dmdirect.Direct{Hash: "a1b2c3d4e5f6"}, nil).Times(1)
	mockDB.EXPECT().AICreate(gomock.Any(), gomock.Any()).
		Return(fmt.Errorf("Error 1062: Duplicate entry 'x' for key 'uq_ai_active_insight_key'")).Times(1)
	mockReq.EXPECT().DirectV1DirectDelete(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

	h := &aiHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	_, err := h.Create(
		context.Background(),
		uuid.Must(uuid.NewV4()),
		"Insight AI",
		"",
		ai.TypeInsight,
		ai.EngineModelOpenaiGPT5,
		nil,
		"test-key",
		uuid.Nil,
		"",
		ai.TTSTypeNone,
		"",
		ai.STTTypeNone,
		"",
		[]tool.ToolName{},
		nil,
		false,
		false,
	)

	assertAlreadyExists(t, err, "AI_INSIGHT_ALREADY_EXISTS")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd bin-ai-manager && go test ./pkg/aihandler/... -run TestCreate_RejectsDuplicateInsightAI -v`
Expected: FAIL — `Create` currently returns a plain wrapped error ("could not create ai: ..."), not a `*cerrors.VoipbinError`, so `errors.As(err, &ve)` fails the test at "expected a `*cerrors.VoipbinError`".

- [ ] **Step 3: Add the `dbhandler` import**

In `bin-ai-manager/pkg/aihandler/chatbot.go`, update the import block (currently lines 1-18):

```go
package aihandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aiprompthistory"
	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
)
```

- [ ] **Step 4: Translate the duplicate-key error in `Create`**

In the same file, replace the `h.dbCreate` error check (currently lines 74-79):

```go
	res, err := h.dbCreate(ctx, customerID, name, detail, aiType, engineModel, parameter, engineKey, ragID,
		initPrompt, ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled,
		autoAICallAuditEnabled, currentPromptHistoryID)
	if err != nil {
		if dbhandler.IsErrDuplicate(err) {
			return nil, cerrors.AlreadyExists(
				commonoutline.ServiceNameAIManager,
				"AI_INSIGHT_ALREADY_EXISTS",
				"This customer already has an active Insight AI. Delete it before creating another.",
			).Wrap(err)
		}
		return nil, errors.Wrapf(err, "could not create ai")
	}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd bin-ai-manager && go test ./pkg/aihandler/... -run TestCreate_RejectsDuplicateInsightAI -v`
Expected: PASS

- [ ] **Step 6: Run the full aihandler package tests to check for regressions**

Run: `cd bin-ai-manager && go test ./pkg/aihandler/... -v`
Expected: PASS (in particular `TestCreate`'s `"handles_database_error"` case, which returns a generic non-duplicate error and must still hit the `errors.Wrapf(err, "could not create ai")` fallback, not the new branch)

- [ ] **Step 7: Commit**

```bash
cd bin-ai-manager
git add pkg/aihandler/chatbot.go pkg/aihandler/chatbot_test.go
git commit -m "$(cat <<'EOF'
SQUARE-23-Insight-AI-one-per-customer

- bin-ai-manager: Translate ai_ais duplicate-key error into AI_INSIGHT_ALREADY_EXISTS (409) in Create
EOF
)"
```

---

## Task 4: `aihandler.Update` — translate duplicate-key error in all three branches

**Files:**
- Modify: `bin-ai-manager/pkg/aihandler/chatbot.go:162-201` (`promptChanged`, `promptCleared` branches)
- Modify: `bin-ai-manager/pkg/aihandler/db.go:1-21` (imports — none needed, `dbhandler`/`cerrors`/`commonoutline` already imported), `:193-198` (`dbUpdate`)
- Test: `bin-ai-manager/pkg/aihandler/chatbot_test.go`

This task has three sub-parts, one per branch identified in the design doc §2. Do them in order; each is independently testable.

### 4a. `promptChanged` branch (`chatbot.go`)

- [ ] **Step 1: Write the failing test**

Add to `bin-ai-manager/pkg/aihandler/chatbot_test.go`, after `TestCreate_RejectsDuplicateInsightAI` (added in Task 3):

```go
// SQUARE-23: Update's promptChanged branch (chatbot.go) must translate a
// duplicate-key failure the same way Create does.
func TestUpdate_promptChanged_RejectsDuplicateInsightAI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	aiID := uuid.Must(uuid.NewV4())
	existing := &ai.AI{Type: ai.TypeNormal, InitPrompt: "old prompt"}
	existing.ID = aiID
	existing.CustomerID = uuid.Must(uuid.NewV4())

	mockUtil.EXPECT().UUIDCreate().Return(uuid.Must(uuid.NewV4())).Times(1)
	mockDB.EXPECT().AIGet(gomock.Any(), aiID).Return(existing, nil).Times(1) // pre-fetch
	mockDB.EXPECT().AIUpdate(gomock.Any(), aiID, gomock.Any()).
		Return(fmt.Errorf("Error 1062: Duplicate entry 'x' for key 'uq_ai_active_insight_key'")).Times(1)

	h := &aiHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		utilHandler:   mockUtil,
	}
	_, err := h.Update(context.Background(), aiID, "name", "", ai.TypeInsight, ai.EngineModelOpenaiGPT5, nil, "", uuid.Nil,
		"new prompt", ai.TTSTypeNone, "", ai.STTTypeNone, "", nil, nil, false, false)

	assertAlreadyExists(t, err, "AI_INSIGHT_ALREADY_EXISTS")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd bin-ai-manager && go test ./pkg/aihandler/... -run TestUpdate_promptChanged_RejectsDuplicateInsightAI -v`
Expected: FAIL — same reason as Task 3 Step 2 (plain wrapped error, not `*cerrors.VoipbinError`).

- [ ] **Step 3: Translate the duplicate-key error in the `promptChanged` branch**

In `bin-ai-manager/pkg/aihandler/chatbot.go`, inside `Update`'s `switch`, replace the `promptChanged` case's `h.db.AIUpdate` check (currently around lines 163-170):

```go
	case promptChanged:
		historyID := h.utilHandler.UUIDCreate()
		fields := h.buildUpdateFields(name, detail, aiType, engineModel, parameter, engineKey, ragID, initPrompt,
			ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled, autoAICallAuditEnabled)
		fields[ai.FieldCurrentPromptHistoryID] = historyID
		if err := h.db.AIUpdate(ctx, id, fields); err != nil {
			if dbhandler.IsErrDuplicate(err) {
				return nil, cerrors.AlreadyExists(
					commonoutline.ServiceNameAIManager,
					"AI_INSIGHT_ALREADY_EXISTS",
					"This customer already has an active Insight AI. Delete it before creating another.",
				).Wrap(err)
			}
			return nil, errors.Wrapf(err, "could not update ai")
		}
```

(The rest of the `promptChanged` case — fetching the updated AI, publishing the webhook event, recording prompt history — is unchanged.)

- [ ] **Step 4: Run test to verify it passes**

Run: `cd bin-ai-manager && go test ./pkg/aihandler/... -run TestUpdate_promptChanged_RejectsDuplicateInsightAI -v`
Expected: PASS

### 4b. `promptCleared` branch (`chatbot.go`)

- [ ] **Step 5: Write the failing test**

Add to `chatbot_test.go`, after the test from 4a:

```go
// SQUARE-23: Update's promptCleared branch must also translate a
// duplicate-key failure — clearing the prompt and changing type to
// insight in the same request is a valid combination (see design doc §2).
func TestUpdate_promptCleared_RejectsDuplicateInsightAI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	aiID := uuid.Must(uuid.NewV4())
	existing := &ai.AI{Type: ai.TypeNormal, InitPrompt: "old prompt"}
	existing.ID = aiID
	existing.CustomerID = uuid.Must(uuid.NewV4())

	mockDB.EXPECT().AIGet(gomock.Any(), aiID).Return(existing, nil).Times(1) // pre-fetch
	mockDB.EXPECT().AIUpdate(gomock.Any(), aiID, gomock.Any()).
		Return(fmt.Errorf("Error 1062: Duplicate entry 'x' for key 'uq_ai_active_insight_key'")).Times(1)

	h := &aiHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		utilHandler:   mockUtil,
	}
	// initPrompt == "" clears the prompt; type changes to insight in the same call
	_, err := h.Update(context.Background(), aiID, "name", "", ai.TypeInsight, ai.EngineModelOpenaiGPT5, nil, "", uuid.Nil,
		"", ai.TTSTypeNone, "", ai.STTTypeNone, "", nil, nil, false, false)

	assertAlreadyExists(t, err, "AI_INSIGHT_ALREADY_EXISTS")
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `cd bin-ai-manager && go test ./pkg/aihandler/... -run TestUpdate_promptCleared_RejectsDuplicateInsightAI -v`
Expected: FAIL, same reason as Step 2.

- [ ] **Step 7: Translate the duplicate-key error in the `promptCleared` branch**

In `chatbot.go`, replace the `promptCleared` case's `h.db.AIUpdate` check (currently around lines 189-194):

```go
	case promptCleared:
		fields := h.buildUpdateFields(name, detail, aiType, engineModel, parameter, engineKey, ragID, "",
			ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled, autoAICallAuditEnabled)
		fields[ai.FieldCurrentPromptHistoryID] = uuid.Nil
		if err := h.db.AIUpdate(ctx, id, fields); err != nil {
			if dbhandler.IsErrDuplicate(err) {
				return nil, cerrors.AlreadyExists(
					commonoutline.ServiceNameAIManager,
					"AI_INSIGHT_ALREADY_EXISTS",
					"This customer already has an active Insight AI. Delete it before creating another.",
				).Wrap(err)
			}
			return nil, errors.Wrapf(err, "could not update ai (clear prompt)")
		}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `cd bin-ai-manager && go test ./pkg/aihandler/... -run TestUpdate_promptCleared_RejectsDuplicateInsightAI -v`
Expected: PASS

### 4c. `default` branch (`dbUpdate` in `db.go`)

- [ ] **Step 9: Write the failing test**

Add to `chatbot_test.go`, after the test from 4b:

```go
// SQUARE-23: Update's default ("prompt unchanged") branch routes through
// dbUpdate() in db.go, a different file than the other two branches — the
// translation must live there too (design doc §2, point 3).
func TestUpdate_promptUnchanged_RejectsDuplicateInsightAI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	aiID := uuid.Must(uuid.NewV4())
	same := "same prompt"
	existing := &ai.AI{Type: ai.TypeNormal, InitPrompt: same}
	existing.ID = aiID
	existing.CustomerID = uuid.Must(uuid.NewV4())

	mockDB.EXPECT().AIGet(gomock.Any(), aiID).Return(existing, nil).Times(1) // pre-fetch
	mockDB.EXPECT().AIUpdate(gomock.Any(), aiID, gomock.Any()).
		Return(fmt.Errorf("Error 1062: Duplicate entry 'x' for key 'uq_ai_active_insight_key'")).Times(1)

	h := &aiHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		utilHandler:   mockUtil,
	}
	_, err := h.Update(context.Background(), aiID, "name", "", ai.TypeInsight, ai.EngineModelOpenaiGPT5, nil, "", uuid.Nil,
		same, ai.TTSTypeNone, "", ai.STTTypeNone, "", nil, nil, false, false)

	assertAlreadyExists(t, err, "AI_INSIGHT_ALREADY_EXISTS")
}
```

- [ ] **Step 10: Run test to verify it fails**

Run: `cd bin-ai-manager && go test ./pkg/aihandler/... -run TestUpdate_promptUnchanged_RejectsDuplicateInsightAI -v`
Expected: FAIL, same reason as Step 2.

- [ ] **Step 11: Translate the duplicate-key error inside `dbUpdate` (`db.go`)**

In `bin-ai-manager/pkg/aihandler/db.go`, replace the `h.db.AIUpdate` check inside `dbUpdate` (currently lines 196-198; `cerrors`, `commonoutline`, and `dbhandler` are already imported in this file — no import changes needed):

```go
	if err := h.db.AIUpdate(ctx, id, fields); err != nil {
		if dbhandler.IsErrDuplicate(err) {
			return nil, cerrors.AlreadyExists(
				commonoutline.ServiceNameAIManager,
				"AI_INSIGHT_ALREADY_EXISTS",
				"This customer already has an active Insight AI. Delete it before creating another.",
			).Wrap(err)
		}
		return nil, errors.Wrapf(err, "could not update ai")
	}
```

- [ ] **Step 12: Run test to verify it passes**

Run: `cd bin-ai-manager && go test ./pkg/aihandler/... -run TestUpdate_promptUnchanged_RejectsDuplicateInsightAI -v`
Expected: PASS

- [ ] **Step 13: Run the full aihandler package tests to check for regressions**

Run: `cd bin-ai-manager && go test ./pkg/aihandler/... -v`
Expected: PASS — in particular the pre-existing `TestUpdate_promptChangedCreatesHistoryAndSetsID`, `TestUpdate_promptClearedResetsID`, and `TestUpdate_promptUnchangedDoesNotCreateHistory` (Task 4's new duplicate-key branches must not affect the non-error path those tests cover).

- [ ] **Step 14: Commit**

```bash
cd bin-ai-manager
git add pkg/aihandler/chatbot.go pkg/aihandler/db.go pkg/aihandler/chatbot_test.go
git commit -m "$(cat <<'EOF'
SQUARE-23-Insight-AI-one-per-customer

- bin-ai-manager: Translate ai_ais duplicate-key error into AI_INSIGHT_ALREADY_EXISTS (409) in all three Update branches
EOF
)"
```

---

## Task 5: Full verification workflow

**Files:** none (verification only)

- [ ] **Step 1: Run the mandatory `bin-ai-manager` verification workflow**

Run:
```bash
cd bin-ai-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: all five steps succeed with no errors. `go mod tidy`/`go mod vendor` should produce no diff (no new dependencies were added — this change only uses existing stdlib/`pkg/errors`/`cerrors`/`dbhandler` symbols).

If `go mod tidy` or `go mod vendor` change `go.mod`/`go.sum`, commit those alongside (per root `CLAUDE.md`'s verification rule) — do not skip.

- [ ] **Step 2: Confirm no unintended changes outside `bin-ai-manager` / `bin-dbscheme-manager`**

Run: `git status --short` from the worktree root (`~/gitvoipbin/monorepo/.worktrees/SQUARE-23-Insight-AI-one-per-customer`)
Expected: clean (everything already committed in Tasks 1-4), or only `vendor/`-adjacent files that are gitignored.

- [ ] **Step 3: Re-confirm the ticket's AC against the finished state**

- [ ] A customer with an existing `type=insight` AI gets a clear 409 `AI_INSIGHT_ALREADY_EXISTS` error creating (Task 3) or updating-into (Task 4) a second one — verified by `TestCreate_RejectsDuplicateInsightAI`, `TestUpdate_promptChanged_RejectsDuplicateInsightAI`, `TestUpdate_promptCleared_RejectsDuplicateInsightAI`, `TestUpdate_promptUnchanged_RejectsDuplicateInsightAI`, and the DB-level `Test_AICreate_InsightUniquePerCustomer`.
- [ ] Migration/cleanup for pre-existing duplicate customers is reviewed — Task 2's migration docstring plus design doc §3 (audit query + human-approved cleanup via the existing `DELETE /ais/{id}` API, executed before this migration is applied to any shared environment — tracked as a follow-up outside this plan, per `bin-dbscheme-manager/CLAUDE.md`'s prohibition on AI-run schema/data changes against non-local databases).
- [ ] `type=normal` AIs are unaffected — verified by `Test_AICreate_InsightUniquePerCustomer`'s normal-type sub-case and the untouched pre-existing `Test_AICreate`/aihandler test suite.

No further code changes needed for this step — it's a checklist confirmation, not new work.
