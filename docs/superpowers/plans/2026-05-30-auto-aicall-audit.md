# Auto AICall Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Automatically trigger an AICall audit when an AI call finishes, gated by a new per-AI option (`auto_aicall_audit_enabled`).

**Architecture:** A new boolean AI option is captured into the AICall `Metadata` (logical OR across all participating AIs) when the call is created. When the call terminates, `ProcessTerminate` reads that frozen flag and, if set, fire-and-forgets a delayed audit request through the existing `requesthandler` queue; any `ai-manager` pod runs the unchanged audit pipeline.

**Tech Stack:** Go, RabbitMQ RPC (`bin-common-handler/requesthandler`), MySQL via Alembic (`bin-dbscheme-manager`), gomock, OpenAPI (`bin-openapi-manager` → `bin-api-manager` codegen), Sphinx RST docs.

**Spec:** `docs/superpowers/specs/2026-05-30-auto-aicall-audit-design.md`

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Auto-aicall-audit` (branch `NOJIRA-Auto-aicall-audit`). All commands below assume you are in this worktree.

---

## Conventions used in every Go task

- After editing a Go service, run its full verification workflow from the service directory:
  ```bash
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
  ```
  For fast inner-loop iteration you may run a single package's tests first (shown per task), but the **full workflow must pass before the task's commit**.
- `go generate ./...` regenerates gomock mocks (`mock_main.go`). Never hand-edit mock files.
- Commit messages: title `NOJIRA-Auto-aicall-audit`, body bullets prefixed with `- <service>:`. No AI attribution.

---

## File Structure (what changes and why)

**`bin-dbscheme-manager`** — Alembic migration adding `ai_ais.auto_aicall_audit_enabled`.

**`bin-ai-manager`**
- `models/ai/main.go`, `field.go`, `webhook.go` — new option field + DB-field constant + webhook exposure.
- `scripts/database_scripts_test/table_ai_ais.sql` — test-schema column (paired with the `db:` tag).
- `pkg/aihandler/chatbot.go`, `db.go`, `main.go` (+ regenerated `mock_main.go`) — thread the new param through Create/Update.
- `pkg/listenhandler/models/request/ais.go`, `pkg/listenhandler/v1_ais.go` — accept + pass the field on the AI create/update routes.
- `models/aicall/main.go` — new `MetaKeyAutoAuditEnabled` constant.
- `pkg/aicallhandler/start.go` — compute the OR flag and freeze it into AICall metadata.
- `pkg/aicallhandler/process.go` — trigger the audit on termination.
- `docs/domain.md` — domain doc sync.

**`bin-common-handler`**
- `pkg/requesthandler/ai_aiaudits.go` — new `AIV1AIAuditCreateWithDelay` (the async trigger).
- `pkg/requesthandler/ai_ais.go` — add the new param to `AIV1AICreate`/`AIV1AIUpdate`.
- `pkg/requesthandler/main.go` (+ regenerated `mock_main.go`) — interface updates.

**`bin-api-manager`**
- `pkg/servicehandler/ai.go`, `main.go` (+ regenerated `mock_main.go`) — thread param into `AICreate`/`AIUpdate`.
- `server/ais.go` — map the generated request field.
- `gens/openapi_server/gen.go` — regenerated.
- `docsdev/source/*` — RST docs.

**`bin-openapi-manager`**
- `openapi/paths/ais/main.yaml`, `id.yaml`, `openapi/openapi.yaml` (+ regenerated `gens/models/gen.go`) — API contract.

**`monorepo-monitoring/api-validator`** — round-trip test for the new field.

---

## Task 1: Database migration — add `ai_ais.auto_aicall_audit_enabled`

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<generated>.py` (revision ID auto-generated — never hand-pick)

- [ ] **Step 1: Generate the migration file**

Run (from the worktree root):
```bash
cd bin-dbscheme-manager/bin-manager/main && alembic -c alembic.ini revision -m "ai_ais_add_column_auto_aicall_audit_enabled"
```
Expected: prints `Generating .../versions/<hash>_ai_ais_add_column_auto_aicall_audit_enabled.py ... done`. Note the generated path.

- [ ] **Step 2: Fill in upgrade()/downgrade()**

Edit the generated file's `upgrade()` and `downgrade()` bodies to exactly:

```python
def upgrade():
    op.execute("""ALTER TABLE ai_ais ADD auto_aicall_audit_enabled TINYINT(1) NOT NULL DEFAULT 0 AFTER smart_turn_enabled;""")


def downgrade():
    op.execute("""ALTER TABLE ai_ais DROP COLUMN auto_aicall_audit_enabled;""")
```

Leave the auto-generated `revision`/`down_revision` header untouched. (Note: unlike the `smart_turn_enabled` migration, we do **not** touch `ai_aicalls` — the auto-audit flag lives in the AICall `Metadata` JSON, not a column.)

- [ ] **Step 3: Verify the migration is syntactically valid and correctly chained**

Run:
```bash
cd bin-dbscheme-manager/bin-manager/main && alembic -c alembic.ini history | head -5
```
Expected: the new revision appears at the head with a valid `down_revision` pointing at the previous head. **Do NOT run `alembic upgrade`** (applying migrations requires human authorization).

- [ ] **Step 4: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Auto-aicall-audit
git add bin-dbscheme-manager/bin-manager/main/versions/
git commit -m "NOJIRA-Auto-aicall-audit

- bin-dbscheme-manager: Add ai_ais.auto_aicall_audit_enabled column migration"
```

---

## Task 2: AI model — option field, DB-field constant, webhook exposure, test schema

**Files:**
- Modify: `bin-ai-manager/models/ai/main.go`
- Modify: `bin-ai-manager/models/ai/field.go`
- Modify: `bin-ai-manager/models/ai/webhook.go`
- Modify: `bin-ai-manager/scripts/database_scripts_test/table_ai_ais.sql`

- [ ] **Step 1: Add the struct field**

In `models/ai/main.go`, immediately after the `SmartTurnEnabled` field (currently line 67), add:

```go
	SmartTurnEnabled bool `json:"smart_turn_enabled,omitempty" db:"smart_turn_enabled"`

	// AutoAICallAuditEnabled, when true, makes any finished AICall involving this AI
	// trigger an AICall audit automatically.
	AutoAICallAuditEnabled bool `json:"auto_aicall_audit_enabled,omitempty" db:"auto_aicall_audit_enabled"`
```

- [ ] **Step 2: Add the DB-field constant**

In `models/ai/field.go`, after `FieldSmartTurnEnabled`:

```go
	FieldSmartTurnEnabled Field = "smart_turn_enabled"

	FieldAutoAICallAuditEnabled Field = "auto_aicall_audit_enabled"
```

- [ ] **Step 3: Expose in WebhookMessage + ConvertWebhookMessage**

In `models/ai/webhook.go`, add the field to the `WebhookMessage` struct after `SmartTurnEnabled`:

```go
	SmartTurnEnabled bool `json:"smart_turn_enabled,omitempty"`

	AutoAICallAuditEnabled bool `json:"auto_aicall_audit_enabled,omitempty"`
```

And in `ConvertWebhookMessage()` after `SmartTurnEnabled: h.SmartTurnEnabled,`:

```go
		SmartTurnEnabled: h.SmartTurnEnabled,

		AutoAICallAuditEnabled: h.AutoAICallAuditEnabled,
```

- [ ] **Step 4: Add the column to the test schema**

In `scripts/database_scripts_test/table_ai_ais.sql`, after the `smart_turn_enabled` column definition (`smart_turn_enabled boolean not null default 0`), add:

```sql
  smart_turn_enabled boolean not null default 0,
  auto_aicall_audit_enabled boolean not null default 0,
```

(Match the surrounding comma/formatting exactly — it is a column in the `create table ai_ais (...)` list.)

- [ ] **Step 5: Run the model package tests**

Run:
```bash
cd bin-ai-manager && go test ./models/ai/... -v
```
Expected: PASS (webhook conversion tests still pass; new field defaults to false).

- [ ] **Step 6: Full verification + commit**

Run the full workflow in `bin-ai-manager` (see Conventions). Then:
```bash
git add bin-ai-manager/models/ai/ bin-ai-manager/scripts/database_scripts_test/table_ai_ais.sql
git commit -m "NOJIRA-Auto-aicall-audit

- bin-ai-manager: Add AutoAICallAuditEnabled field to AI model, webhook, and test schema"
```

---

## Task 3: Thread the option through the AI create/update handler chain

**Files:**
- Modify: `bin-ai-manager/pkg/aihandler/main.go` (interface)
- Modify: `bin-ai-manager/pkg/aihandler/chatbot.go` (Create/Update wrappers)
- Modify: `bin-ai-manager/pkg/aihandler/db.go` (dbCreate/dbUpdate/buildUpdateFields)
- Modify: `bin-ai-manager/pkg/listenhandler/models/request/ais.go`
- Modify: `bin-ai-manager/pkg/listenhandler/v1_ais.go`
- Regenerate: `bin-ai-manager/pkg/aihandler/mock_main.go`

The convention: add `autoAICallAuditEnabled bool` as the **last parameter** of `Create`/`Update`, mirroring exactly how `smartTurnEnabled bool` is threaded.

- [ ] **Step 1: Update the AIHandler interface**

In `pkg/aihandler/main.go`, add `autoAICallAuditEnabled bool,` after `smartTurnEnabled bool,` in **both** `Create(...)` and `Update(...)` interface methods:

```go
		vadConfig *ai.VADConfig,
		smartTurnEnabled bool,
		autoAICallAuditEnabled bool,
	) (*ai.AI, error)
```

- [ ] **Step 2: Update the Create wrapper**

In `pkg/aihandler/chatbot.go`, add the param to the `Create(...)` signature (after `smartTurnEnabled bool,`):

```go
	vadConfig *ai.VADConfig,
	smartTurnEnabled bool,
	autoAICallAuditEnabled bool,
) (*ai.AI, error) {
```

and pass it into the `dbCreate(...)` call (it currently ends `..., vadConfig, smartTurnEnabled, currentPromptHistoryID)`):

```go
	res, err := h.dbCreate(ctx, customerID, name, detail, engineModel, parameter, engineKey, ragID,
		initPrompt, ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled,
		autoAICallAuditEnabled, currentPromptHistoryID)
```

- [ ] **Step 3: Update the Update wrapper**

In `pkg/aihandler/chatbot.go`, add `autoAICallAuditEnabled bool,` to the `Update(...)` signature after `smartTurnEnabled bool,`. Then in **every** `buildUpdateFields(...)` call inside `Update` (there are multiple branches: `promptChanged`, `promptCleared`, and the default), append `autoAICallAuditEnabled` as the last argument. Example for the `promptChanged` branch:

```go
		fields := h.buildUpdateFields(name, detail, engineModel, parameter, engineKey, ragID, initPrompt,
			ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled, autoAICallAuditEnabled)
```

Apply the same change to the other `buildUpdateFields(...)` call sites in `Update`. (Search the file for `buildUpdateFields(` to find them all.)

- [ ] **Step 4: Update dbCreate**

In `pkg/aihandler/db.go`, add `autoAICallAuditEnabled bool,` to the `dbCreate(...)` signature after `smartTurnEnabled bool,` (before `currentPromptHistoryID uuid.UUID,`), and set it on the `&ai.AI{...}` literal after `SmartTurnEnabled: smartTurnEnabled,`:

```go
		VADConfig:        vadConfig,
		SmartTurnEnabled: smartTurnEnabled,

		AutoAICallAuditEnabled: autoAICallAuditEnabled,
```

- [ ] **Step 5: Update dbUpdate + buildUpdateFields**

In `pkg/aihandler/db.go`:

`dbUpdate(...)` — add `autoAICallAuditEnabled bool,` after `smartTurnEnabled bool,`, and pass it to its `buildUpdateFields(...)` call as the last arg.

`buildUpdateFields(...)` — add `autoAICallAuditEnabled bool,` as the last parameter and add the map entry:

```go
		ai.FieldVADConfig:        vadConfig,
		ai.FieldSmartTurnEnabled: smartTurnEnabled,
		ai.FieldAutoAICallAuditEnabled: autoAICallAuditEnabled,
	}
```

- [ ] **Step 6: Update the listenhandler request models**

In `pkg/listenhandler/models/request/ais.go`, add to **both** `V1DataAIsPost` and `V1DataAIsIDPut`, after `SmartTurnEnabled`:

```go
	VADConfig        *ai.VADConfig `json:"vad_config,omitempty"`
	SmartTurnEnabled bool          `json:"smart_turn_enabled,omitempty"`

	AutoAICallAuditEnabled bool `json:"auto_aicall_audit_enabled,omitempty"`
```

- [ ] **Step 7: Pass the field in the listen routes**

In `pkg/listenhandler/v1_ais.go`:
- In `processV1AIsPost`, the `h.aiHandler.Create(...)` call currently ends with `req.VADConfig, req.SmartTurnEnabled,`. Add `req.AutoAICallAuditEnabled,` as the last argument.
- In `processV1AIsIDPut`, the `h.aiHandler.Update(...)` call similarly — add `req.AutoAICallAuditEnabled,` as the last argument.

- [ ] **Step 8: Regenerate mocks and run handler tests**

Run:
```bash
cd bin-ai-manager && go generate ./pkg/aihandler/... && go test ./pkg/aihandler/... ./pkg/listenhandler/... -v
```
Expected: PASS. If existing tests call `Create`/`Update` with positional args, they will fail to compile — update those test call sites to pass a trailing `false` (or a meaningful value) for the new param. Fix until green.

- [ ] **Step 9: Full verification + commit**

Run the full workflow in `bin-ai-manager`. Then:
```bash
git add bin-ai-manager/pkg/aihandler/ bin-ai-manager/pkg/listenhandler/
git commit -m "NOJIRA-Auto-aicall-audit

- bin-ai-manager: Thread auto_aicall_audit_enabled through AI create/update handler chain"
```

---

## Task 4: Freeze the auto-audit flag into AICall metadata at creation

**Files:**
- Modify: `bin-ai-manager/models/aicall/main.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go`
- Test: `bin-ai-manager/pkg/aicallhandler/start_test.go`

- [ ] **Step 1: Add the metadata key constant**

In `models/aicall/main.go`, after the existing `MetaKeyPromptSnapshots` constant (line 22), add:

```go
// MetaKeyPromptSnapshots is the Metadata map key for the prompt snapshot slice.
const MetaKeyPromptSnapshots = "prompt_snapshots"

// MetaKeyAutoAuditEnabled is the Metadata map key (bool) recording whether this AICall
// should be auto-audited when it terminates. Frozen from the participating AI option(s)
// at call-creation time.
const MetaKeyAutoAuditEnabled = "auto_audit_enabled"
```

- [ ] **Step 2: Write the failing test for the flag computation**

In `pkg/aicallhandler/start_test.go`, add a table-driven test for a new helper `buildPromptSnapshots` returning the flag. (The team path needs `teamHandler`/`aiHandler` mocks; model it on the existing team-path tests in this file — search for `AssistanceTypeTeam`.) Minimum single-AI cases:

```go
func Test_buildPromptSnapshots_autoAudit(t *testing.T) {
	tests := []struct {
		name           string
		a              *ai.AI
		assistanceType aicall.AssistanceType
		expectAudit    bool
	}{
		{
			name:           "single AI with auto audit enabled",
			a:              &ai.AI{Identity: identity.Identity{ID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")}, AutoAICallAuditEnabled: true},
			assistanceType: aicall.AssistanceTypeAI,
			expectAudit:    true,
		},
		{
			name:           "single AI with auto audit disabled",
			a:              &ai.AI{Identity: identity.Identity{ID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000002")}, AutoAICallAuditEnabled: false},
			assistanceType: aicall.AssistanceTypeAI,
			expectAudit:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &aicallHandler{
				reqHandler: requesthandler.NewMockRequestHandler(mc),
			}
			// getInitPrompt path: single-AI does not call team resolution.
			_, gotAudit := h.buildPromptSnapshots(context.Background(), tt.a, tt.assistanceType, tt.a.ID, uuid.Nil)
			if gotAudit != tt.expectAudit {
				t.Errorf("wrong auto audit flag. expect: %v, got: %v", tt.expectAudit, gotAudit)
			}
		})
	}
}
```

(Add a team case `team any-enabled → true` and `team all-disabled → false` following the existing team-path test setup with `teamHandler`/`aiHandler` mocks. If `getInitPrompt` requires `reqHandler`/`db` calls, wire the same mock expectations the existing snapshot tests use.)

- [ ] **Step 3: Run the test to verify it fails**

Run:
```bash
cd bin-ai-manager && go test ./pkg/aicallhandler/ -run Test_buildPromptSnapshots_autoAudit -v
```
Expected: compile error — `buildPromptSnapshots` returns a single value, not two.

- [ ] **Step 4: Change buildPromptSnapshots to return the flag**

In `pkg/aicallhandler/start.go`, change the signature and bodies:

```go
func (h *aicallHandler) buildPromptSnapshots(ctx context.Context, a *ai.AI, assistanceType aicall.AssistanceType, assistanceID uuid.UUID, activeflowID uuid.UUID) ([]aicall.PromptSnapshot, bool) {
	switch assistanceType {
	case aicall.AssistanceTypeAI:
		substituted := h.getInitPrompt(ctx, a, activeflowID)
		return []aicall.PromptSnapshot{
			{
				AIID:            a.ID,
				PromptHistoryID: a.CurrentPromptHistoryID,
				Prompt:          substituted,
			},
		}, a.AutoAICallAuditEnabled

	case aicall.AssistanceTypeTeam:
		memberAIs, err := h.resolveAIForTeam(ctx, assistanceID)
		if err != nil {
			logrus.WithField("func", "buildPromptSnapshots").
				Errorf("Could not resolve team AIs — storing empty snapshots. err: %v", err)
			return []aicall.PromptSnapshot{}, false
		}
		snapshots := make([]aicall.PromptSnapshot, 0, len(memberAIs))
		autoAudit := false
		for memberID, memberAI := range memberAIs {
			if memberAI.AutoAICallAuditEnabled {
				autoAudit = true
			}
			substituted := h.getInitPrompt(ctx, memberAI, activeflowID)
			snapshots = append(snapshots, aicall.PromptSnapshot{
				AIID:            memberAI.ID,
				PromptHistoryID: memberAI.CurrentPromptHistoryID,
				Prompt:          substituted,
				MemberID:        memberID,
			})
		}
		return snapshots, autoAudit

	default:
		return []aicall.PromptSnapshot{}, false
	}
}
```

- [ ] **Step 5: Update both call sites to store the flag**

In `pkg/aicallhandler/start.go`, in `startAIcallByRealtime` (~line 613) and `startAIcallByMessaging` (~line 668), change:

```go
	snapshots := h.buildPromptSnapshots(ctx, a, assistanceType, assistanceID, activeflowID)
	metadata := map[string]any{
		aicall.MetaKeyPromptSnapshots: snapshots,
	}
```
to:
```go
	snapshots, autoAudit := h.buildPromptSnapshots(ctx, a, assistanceType, assistanceID, activeflowID)
	metadata := map[string]any{
		aicall.MetaKeyPromptSnapshots:  snapshots,
		aicall.MetaKeyAutoAuditEnabled: autoAudit,
	}
```

- [ ] **Step 6: Run the test to verify it passes**

Run:
```bash
cd bin-ai-manager && go test ./pkg/aicallhandler/ -run Test_buildPromptSnapshots_autoAudit -v
```
Expected: PASS. Existing snapshot tests that assert on `Metadata` may now need the extra `MetaKeyAutoAuditEnabled` key — update them to expect it (`false` unless the test AI enabled it).

- [ ] **Step 7: Full verification + commit**

Run the full workflow in `bin-ai-manager`. Then:
```bash
git add bin-ai-manager/models/aicall/main.go bin-ai-manager/pkg/aicallhandler/start.go bin-ai-manager/pkg/aicallhandler/start_test.go
git commit -m "NOJIRA-Auto-aicall-audit

- bin-ai-manager: Freeze auto-audit flag into AICall metadata at creation"
```

---

## Task 5: Add the async audit trigger to requesthandler

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/ai_aiaudits.go`
- Modify: `bin-common-handler/pkg/requesthandler/main.go` (interface)
- Test: `bin-common-handler/pkg/requesthandler/ai_aiaudits_test.go`
- Regenerate: `bin-common-handler/pkg/requesthandler/mock_main.go`

- [ ] **Step 1: Write the failing test**

In `ai_aiaudits_test.go`, model on the existing `Test_AIV1AIAuditCreate` and on the delayed pattern. Add:

```go
func Test_AIV1AIAuditCreateWithDelay(t *testing.T) {
	tests := []struct {
		name       string
		customerID uuid.UUID
		aicallID   uuid.UUID
		language   string
		delay      int

		expectTarget  string
		expectRequest *sock.Request
	}{
		{
			name:       "normal",
			customerID: uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001"),
			aicallID:   uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000001"),
			language:   "",
			delay:      1000,

			expectTarget: string(commonoutline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      "/v1/aiaudits",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"aaaaaaaa-0000-0000-0000-000000000001","aicall_id":"bbbbbbbb-0000-0000-0000-000000000001"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{sock: mockSock}

			mockSock.EXPECT().RequestPublishWithDelay(gomock.Any(), tt.expectTarget, tt.expectRequest, tt.delay).Return(nil)

			if err := reqHandler.AIV1AIAuditCreateWithDelay(context.Background(), tt.customerID, tt.aicallID, tt.language, tt.delay); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
```

> Before finalizing, open `ai_aicalls_test.go` and copy the **exact** mock expectation the existing `AIV1AIcallTerminateWithDelay` test uses for the delayed path (method name and argument order on `MockSockHandler` — e.g. `RequestPublishWithDelay`). Match it precisely; the snippet above assumes that method name.

- [ ] **Step 2: Run the test to verify it fails**

Run:
```bash
cd bin-common-handler && go test ./pkg/requesthandler/ -run Test_AIV1AIAuditCreateWithDelay -v
```
Expected: compile error — `AIV1AIAuditCreateWithDelay` undefined.

- [ ] **Step 3: Implement the method**

In `ai_aiaudits.go`, after `AIV1AIAuditCreate`, add:

```go
// AIV1AIAuditCreateWithDelay asks ai-manager to create audit job(s) for an aicall after a
// delay. It is fire-and-forget: with delay > 0 the request is published to the queue and no
// response is awaited. It returns nil on a successful publish.
func (r *requestHandler) AIV1AIAuditCreateWithDelay(ctx context.Context, customerID uuid.UUID, aicallID uuid.UUID, language string, delay int) error {
	uri := "/v1/aiaudits"

	data := &amrequest.V1DataAIAuditsPost{
		CustomerID: customerID,
		AIcallID:   aicallID,
		Language:   language,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/aiaudits", requestTimeoutDefault, delay, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	return parseResponse(tmp, nil)
}
```

- [ ] **Step 4: Add to the RequestHandler interface**

In `pkg/requesthandler/main.go`, after the `AIV1AIAuditCreate(...)` line (line 333):

```go
	AIV1AIAuditCreate(ctx context.Context, customerID uuid.UUID, aicallID uuid.UUID, language string) ([]*amaiaudit.AIAudit, error)
	AIV1AIAuditCreateWithDelay(ctx context.Context, customerID uuid.UUID, aicallID uuid.UUID, language string, delay int) error
```

- [ ] **Step 5: Regenerate mocks, run the test**

Run:
```bash
cd bin-common-handler && go generate ./pkg/requesthandler/... && go test ./pkg/requesthandler/ -run Test_AIV1AIAuditCreateWithDelay -v
```
Expected: PASS.

- [ ] **Step 6: Full verification + commit**

Run the full workflow in `bin-common-handler` (this changes the shared interface — also confirm consumers still build; the change is additive). Then:
```bash
git add bin-common-handler/pkg/requesthandler/
git commit -m "NOJIRA-Auto-aicall-audit

- bin-common-handler: Add AIV1AIAuditCreateWithDelay async audit trigger"
```

---

## Task 6: Trigger the audit on AICall termination

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/process.go`
- Test: `bin-ai-manager/pkg/aicallhandler/process_test.go`

- [ ] **Step 1: Add the trigger-delay constant**

At the top of `pkg/aicallhandler/process.go` (after the imports), add:

```go
// autoAuditTriggerDelay is the publish delay (ms) for the auto-audit request. A non-zero
// delay selects the fire-and-forget async publish path and lets trailing state settle.
const autoAuditTriggerDelay = 1000
```

- [ ] **Step 2: Write the failing test**

In `process_test.go`, add cases asserting the trigger fires only when the metadata flag is true, and that a publish error does not fail termination. Model the handler/mocks on the existing `ProcessTerminate` tests in this file (reuse their `UpdateStatus`/`Get`/`reqHandler` setup). Core assertions:

```go
func Test_ProcessTerminate_autoAuditTrigger(t *testing.T) {
	tests := []struct {
		name        string
		aicall      *aicall.AIcall
		expectAudit bool
		publishErr  error
	}{
		{
			name: "auto audit enabled triggers create",
			aicall: &aicall.AIcall{
				Identity:      identity.Identity{ID: uuid.FromStringOrNil("cccccccc-0000-0000-0000-000000000001"), CustomerID: uuid.FromStringOrNil("dddddddd-0000-0000-0000-000000000001")},
				Status:        aicall.StatusProgressing,
				ReferenceType: aicall.ReferenceTypeCall,
				Metadata:      map[string]any{aicall.MetaKeyAutoAuditEnabled: true},
			},
			expectAudit: true,
		},
		{
			name: "auto audit disabled does not trigger",
			aicall: &aicall.AIcall{
				Identity:      identity.Identity{ID: uuid.FromStringOrNil("cccccccc-0000-0000-0000-000000000002"), CustomerID: uuid.FromStringOrNil("dddddddd-0000-0000-0000-000000000001")},
				Status:        aicall.StatusProgressing,
				ReferenceType: aicall.ReferenceTypeCall,
				Metadata:      map[string]any{aicall.MetaKeyAutoAuditEnabled: false},
			},
			expectAudit: false,
		},
		{
			name: "publish error does not fail termination",
			aicall: &aicall.AIcall{
				Identity:      identity.Identity{ID: uuid.FromStringOrNil("cccccccc-0000-0000-0000-000000000003"), CustomerID: uuid.FromStringOrNil("dddddddd-0000-0000-0000-000000000001")},
				Status:        aicall.StatusProgressing,
				ReferenceType: aicall.ReferenceTypeCall,
				Metadata:      map[string]any{aicall.MetaKeyAutoAuditEnabled: true},
			},
			expectAudit: true,
			publishErr:  fmt.Errorf("publish failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &aicallHandler{reqHandler: mockReq, db: mockDB, notifyHandler: mockNotify}

			// Get returns the progressing aicall; UpdateStatus returns the terminated one.
			// Wire these exactly as the existing ProcessTerminate tests do (Get -> cache/db,
			// FlowV1ActiveflowServiceStop, UpdateStatus). Copy that scaffolding.
			terminated := *tt.aicall
			terminated.Status = aicall.StatusTerminated
			// ... existing-test scaffolding for Get/UpdateStatus/FlowV1ActiveflowServiceStop ...

			if tt.expectAudit {
				mockReq.EXPECT().
					AIV1AIAuditCreateWithDelay(gomock.Any(), tt.aicall.CustomerID, tt.aicall.ID, "", autoAuditTriggerDelay).
					Return(tt.publishErr)
			}

			_, err := h.ProcessTerminate(context.Background(), tt.aicall.ID)
			if err != nil {
				t.Errorf("ProcessTerminate must succeed even on publish error. got: %v", err)
			}
		})
	}
}
```

> Reuse the exact mock scaffolding from the existing `ProcessTerminate` test(s) in this file for `Get`, `UpdateStatus`, and `FlowV1ActiveflowServiceStop`. The new assertion is only the `AIV1AIAuditCreateWithDelay` expectation. For the disabled case, assert it is **not** called (default gomock behavior — no `EXPECT` means a call fails the test).

- [ ] **Step 3: Run the test to verify it fails**

Run:
```bash
cd bin-ai-manager && go test ./pkg/aicallhandler/ -run Test_ProcessTerminate_autoAuditTrigger -v
```
Expected: FAIL — the trigger call is never made.

- [ ] **Step 4: Add the trigger hook**

In `pkg/aicallhandler/process.go`, replace the tail of `ProcessTerminate`:

```go
	res, err := h.UpdateStatus(ctx, id, aicall.StatusTerminated)
	if err != nil {
		return nil, errors.Wrap(err, "could not terminate the aicall")
	}

	return res, nil
```
with:
```go
	res, err := h.UpdateStatus(ctx, id, aicall.StatusTerminated)
	if err != nil {
		return nil, errors.Wrap(err, "could not terminate the aicall")
	}

	// Best-effort: if auto-audit was enabled at call creation, fire-and-forget an audit
	// request through the queue. Any failure here must never affect call termination.
	if enabled, _ := res.Metadata[aicall.MetaKeyAutoAuditEnabled].(bool); enabled {
		if errAudit := h.reqHandler.AIV1AIAuditCreateWithDelay(ctx, res.CustomerID, res.ID, "", autoAuditTriggerDelay); errAudit != nil {
			log.Errorf("Could not enqueue the auto aicall audit. Continuing anyway. aicall_id: %s, err: %v", res.ID, errAudit)
		}
	}

	return res, nil
```

- [ ] **Step 5: Run the test to verify it passes**

Run:
```bash
cd bin-ai-manager && go test ./pkg/aicallhandler/ -run Test_ProcessTerminate_autoAuditTrigger -v
```
Expected: PASS.

- [ ] **Step 6: Full verification + commit**

Run the full workflow in `bin-ai-manager`. Then:
```bash
git add bin-ai-manager/pkg/aicallhandler/process.go bin-ai-manager/pkg/aicallhandler/process_test.go
git commit -m "NOJIRA-Auto-aicall-audit

- bin-ai-manager: Trigger auto aicall audit on termination when enabled"
```

---

## Task 7: Make the option settable via the public API (requesthandler + api-manager)

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/ai_ais.go` (`AIV1AICreate`, `AIV1AIUpdate`)
- Modify: `bin-common-handler/pkg/requesthandler/main.go` (interface)
- Regenerate: `bin-common-handler/pkg/requesthandler/mock_main.go`
- Modify: `bin-api-manager/pkg/servicehandler/main.go` (interface), `pkg/servicehandler/ai.go`
- Regenerate: `bin-api-manager/pkg/servicehandler/mock_main.go`
- Modify: `bin-api-manager/server/ais.go`

> The new param goes at the **end** of each signature. `bin-common-handler` is consumed by all services, but this is an additive trailing param — consumers still compile.

- [ ] **Step 1: requesthandler — add param to AIV1AICreate**

In `pkg/requesthandler/ai_ais.go`, add `autoAICallAuditEnabled bool,` after `toolNames []amtool.ToolName,` in `AIV1AICreate(...)`, and set it on the `V1DataAIsPost{...}` literal:

```go
		ToolNames: toolNames,

		AutoAICallAuditEnabled: autoAICallAuditEnabled,
	}
```

- [ ] **Step 2: requesthandler — add param to AIV1AIUpdate**

Same change in `AIV1AIUpdate(...)`: add `autoAICallAuditEnabled bool,` after `toolNames []amtool.ToolName,`, and set `AutoAICallAuditEnabled: autoAICallAuditEnabled,` on the `V1DataAIsIDPut{...}` literal.

- [ ] **Step 3: requesthandler — update the interface**

In `pkg/requesthandler/main.go`, add `autoAICallAuditEnabled bool,` as the last param (after `toolNames []amtool.ToolName,`) in both the `AIV1AICreate(...)` and `AIV1AIUpdate(...)` interface declarations.

- [ ] **Step 4: api-manager servicehandler — interface + impl**

In `bin-api-manager/pkg/servicehandler/main.go`, add `autoAICallAuditEnabled bool,` as the last param to both `AICreate(...)` and `AIUpdate(...)` interface methods.

In `bin-api-manager/pkg/servicehandler/ai.go`:
- Add `autoAICallAuditEnabled bool,` to the `AICreate(...)` and `AIUpdate(...)` function signatures.
- Pass it as the last argument to the `h.reqHandler.AIV1AICreate(...)` and `h.reqHandler.AIV1AIUpdate(...)` calls.

- [ ] **Step 5: api-manager server — map the generated request field**

In `bin-api-manager/server/ais.go`, in `PostAis` (before the `AICreate` call) and `PutAisId` (before the `AIUpdate` call), extract the optional generated field following the existing `req.SttLanguage` nil-guard pattern:

```go
	autoAICallAuditEnabled := false
	if req.AutoAicallAuditEnabled != nil {
		autoAICallAuditEnabled = *req.AutoAicallAuditEnabled
	}
```

Then add `autoAICallAuditEnabled` as the last argument to the `h.serviceHandler.AICreate(...)` / `h.serviceHandler.AIUpdate(...)` calls.

> The exact generated field name (`req.AutoAicallAuditEnabled`) comes from oapi-codegen and only exists after Task 8 regenerates `gen.go`. **Do Task 8 before compiling Task 7's server change**, or temporarily stub the value to `false` and wire it after regeneration. Recommended: implement Steps 1–4 here, run Task 8, then return for Step 5.

- [ ] **Step 6: Regenerate mocks**

Run:
```bash
cd bin-common-handler && go generate ./pkg/requesthandler/...
cd ../bin-api-manager && go generate ./pkg/servicehandler/...
```

- [ ] **Step 7: Fix call sites + run tests**

Compilation will flag every existing caller of `AIV1AICreate`/`AIV1AIUpdate`/servicehandler `AICreate`/`AIUpdate` (tests and the ai-manager listenhandler is unaffected — it calls the ai-manager `aiHandler`, not requesthandler). Update test call sites to pass a trailing `false`. Run:
```bash
cd bin-common-handler && go test ./pkg/requesthandler/... 
cd ../bin-api-manager && go test ./pkg/servicehandler/... ./server/...
```
Expected: PASS after fixing call sites.

- [ ] **Step 8: Full verification + commit**

Run the full workflow in `bin-common-handler` and `bin-api-manager`. Then:
```bash
git add bin-common-handler/pkg/requesthandler/ bin-api-manager/pkg/servicehandler/ bin-api-manager/server/ais.go
git commit -m "NOJIRA-Auto-aicall-audit

- bin-common-handler: Add auto_aicall_audit_enabled to AIV1AICreate/AIV1AIUpdate
- bin-api-manager: Thread auto_aicall_audit_enabled through AI create/update service"
```

---

## Task 8: OpenAPI contract — request bodies + response schema

**Files:**
- Modify: `bin-openapi-manager/openapi/paths/ais/main.yaml` (POST body)
- Modify: `bin-openapi-manager/openapi/paths/ais/id.yaml` (PUT body)
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (`AIManagerAI` response schema)
- Regenerate: `bin-openapi-manager/gens/models/gen.go`, `bin-api-manager/gens/openapi_server/gen.go`

- [ ] **Step 1: Add to the POST request body**

In `bin-openapi-manager/openapi/paths/ais/main.yaml`, in the request body `properties` (after `tool_names`), add:

```yaml
            tool_names:
              type: array
              items:
                $ref: '#/components/schemas/AIManagerToolName'
              description: List of tool names to enable for this AI. Use ["all"] to enable all available tools.
            auto_aicall_audit_enabled:
              type: boolean
              description: When true, any finished AICall involving this AI is audited automatically.
              example: false
```

- [ ] **Step 2: Add to the PUT request body**

In `bin-openapi-manager/openapi/paths/ais/id.yaml`, add the same `auto_aicall_audit_enabled` property block to the request body `properties` (after `tool_names`).

- [ ] **Step 3: Add to the AIManagerAI response schema**

In `bin-openapi-manager/openapi/openapi.yaml`, in the `AIManagerAI` schema after `smart_turn_enabled` (~line 2137), add:

```yaml
        smart_turn_enabled:
          type: boolean
          description: Enable smart turn detection using Pipecat's LocalSmartTurnAnalyzerV3. When enabled, forces VAD stop_secs to 0.2 for optimal turn-taking.
          example: false
        auto_aicall_audit_enabled:
          type: boolean
          description: When true, any finished AICall involving this AI is audited automatically.
          example: false
```

- [ ] **Step 4: Regenerate generated models**

Run:
```bash
cd bin-openapi-manager && go generate ./...
cd ../bin-api-manager && go generate ./...
```
Expected: `bin-openapi-manager/gens/models/gen.go` and `bin-api-manager/gens/openapi_server/gen.go` now contain `AutoAicallAuditEnabled *bool` on the AI request/response structs.

- [ ] **Step 5: Verify the generated field name**

Run:
```bash
grep -rn "AutoAicallAuditEnabled" bin-api-manager/gens/openapi_server/gen.go | head
```
Expected: appears on `PostAisJSONBody`, `PutAisIdJSONBody`, and the AI response type. Use this exact name in Task 7 Step 5.

- [ ] **Step 6: Complete Task 7 Step 5 (server mapping)** if deferred, then build.

Run:
```bash
cd bin-api-manager && go build ./...
```
Expected: builds clean.

- [ ] **Step 7: Full verification + commit**

Run the full workflow in `bin-openapi-manager` and `bin-api-manager`. Then:
```bash
git add bin-openapi-manager/ bin-api-manager/gens/ bin-api-manager/server/ais.go
git commit -m "NOJIRA-Auto-aicall-audit

- bin-openapi-manager: Add auto_aicall_audit_enabled to AI create/update/response schema
- bin-api-manager: Regenerate openapi server and map auto_aicall_audit_enabled"
```

---

## Task 9: api-validator round-trip test

**Files:**
- Modify/Create: `monorepo-monitoring/api-validator/tests/scenarios/test_ais_*.py` (follow the existing AI test layout)

> Read-only/CRUD on a test AI only — no calls, no audits, no cost.

- [ ] **Step 1: Locate the AI test scenarios**

Run:
```bash
ls ~/gitvoipbin/monorepo-monitoring/api-validator/tests/scenarios/ | grep -i ai
```
Identify the existing AI create/update scenario file and its fixtures/auth pattern.

- [ ] **Step 2: Add the round-trip assertion**

Add a test that creates an AI with `auto_aicall_audit_enabled: true`, asserts the response echoes `true`, updates it to `false`, and asserts the GET reflects `false`. Mirror the existing AI scenario's client/auth/cleanup fixtures exactly. (No real call is made.)

- [ ] **Step 3: Run the scenario**

Run the api-validator suite for the AI scenarios per that repo's README (e.g. `pytest tests/scenarios/test_ais_*.py -v`). Expected: PASS.

- [ ] **Step 4: Commit (in the api-validator repo/worktree)**

```bash
git add tests/scenarios/
git commit -m "NOJIRA-Add-ais-auto-audit-test

- api-validator: Round-trip test for ai auto_aicall_audit_enabled field"
```

---

## Task 10: RST documentation

**Files:**
- Modify: `bin-api-manager/docsdev/source/*` (AI struct + overview/tutorial)
- Rebuild + commit: `bin-api-manager/docsdev/build/`

- [ ] **Step 1: Find the AI RST docs**

Run:
```bash
grep -rln "smart_turn_enabled\|tool_names" bin-api-manager/docsdev/source/ | head
```
Identify the AI struct doc (`*_struct_*.rst`) and the AI overview/tutorial files.

- [ ] **Step 2: Document the field**

Add `auto_aicall_audit_enabled` (boolean, default false) to the AI struct field table and a short note in the AI overview explaining auto-audit behavior. Only document fields present in `WebhookMessage` (it is — added in Task 2).

- [ ] **Step 3: Clean rebuild the HTML**

Run:
```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
```
Expected: build succeeds with no new warnings about the AI pages.

- [ ] **Step 4: Force-add the build output + commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Auto-aicall-audit
git add bin-api-manager/docsdev/source/
git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-Auto-aicall-audit

- bin-api-manager: Document auto_aicall_audit_enabled in AI RST docs"
```

---

## Task 11: Domain doc sync + whole-feature verification

**Files:**
- Modify: `bin-ai-manager/docs/domain.md`

- [ ] **Step 1: Sync the AI domain entity**

Add `auto_aicall_audit_enabled` to the AI entity description in `bin-ai-manager/docs/domain.md` (the PostToolUse hook warns when `models/**` changes without a domain.md update). Use `bash docs/reference/extractor.sh bin-ai-manager` if it regenerates the section; otherwise edit by hand to match the existing entity format.

- [ ] **Step 2: Final full verification across all touched Go services**

Run the full workflow in each (independently):
```bash
for svc in bin-common-handler bin-ai-manager bin-openapi-manager bin-api-manager; do
  echo "=== $svc ===" && (cd $svc && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m) || break
done
```
Expected: all four PASS.

- [ ] **Step 3: Commit**

```bash
git add bin-ai-manager/docs/domain.md
git commit -m "NOJIRA-Auto-aicall-audit

- bin-ai-manager: Sync AI domain doc with auto_aicall_audit_enabled"
```

- [ ] **Step 4: Confirm clean tree + branch readiness**

Run:
```bash
git status && git log --oneline origin/main..HEAD
```
Expected: clean working tree; commit list covers migration, model, handler chain, metadata, trigger, requesthandler, api wiring, openapi, docs.

---

## Notes for the implementer

- **Do NOT run `alembic upgrade`/`downgrade`** — migrations are created and committed only.
- **Do NOT merge** — open a PR only when explicitly asked; stop at "ready for review".
- The migration must be deployed before the new column is selected in production; coordinate ordering at deploy time (out of scope for this plan).
- `vendor/` is never committed (`.gitignore`); `go mod vendor` is for local build/test only.
- If `bin-common-handler` interface changes break a consumer's compile, fix that consumer's call sites (trailing `false`) — the param is additive and defaults to disabled.
