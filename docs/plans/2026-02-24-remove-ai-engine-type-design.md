# Remove EngineType from AI Model - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove the dead `EngineType` field from the AI model across all layers (database, Go structs, API, OpenAPI, CLI, metrics, docs).

**Architecture:** The `EngineType` field exists in the AI struct and flows through: database → dbhandler → aihandler → listenhandler → requesthandler (RPC) → servicehandler → server (HTTP). We remove it from every layer bottom-up, starting with the database migration, then models, then handlers, then API/OpenAPI, then docs. The `ai_aicalls.ai_engine_type` column was already dropped in migration `07e99bfda2ef`.

**Tech Stack:** Go, Alembic (Python), OpenAPI YAML, RST/Sphinx docs

---

### Task 1: Alembic migration to drop `engine_type` column from `ai_ais`

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<hash>_ai_ais_drop_column_engine_type.py`

**Step 1: Create migration file**

```bash
cd bin-dbscheme-manager/bin-manager
alembic -c alembic.ini revision -m "ai_ais drop column engine_type"
```

If alembic is not configured locally, create the file manually with a new revision hash. The `down_revision` must be `07e99bfda2ef` (current head).

**Step 2: Edit migration**

```python
"""ai_ais drop column engine_type

Revision ID: <generated>
Revises: 07e99bfda2ef
Create Date: 2026-02-24

"""
from alembic import op
import sqlalchemy as sa

revision = '<generated>'
down_revision = '07e99bfda2ef'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE ai_ais DROP COLUMN engine_type;""")


def downgrade():
    op.execute("""ALTER TABLE ai_ais ADD engine_type varchar(255) DEFAULT '' AFTER detail;""")
```

**Step 3: Commit**

```bash
git add bin-dbscheme-manager/bin-manager/main/versions/*_ai_ais_drop_column_engine_type.py
git commit -m "NOJIRA-remove-ai-engine-type

- bin-dbscheme-manager: Add migration to drop engine_type from ai_ais"
```

---

### Task 2: Remove EngineType from AI models

**Files:**
- Modify: `bin-ai-manager/models/ai/main.go`
- Modify: `bin-ai-manager/models/ai/webhook.go`
- Modify: `bin-ai-manager/models/ai/field.go`
- Modify: `bin-ai-manager/models/ai/filters.go`
- Modify: `bin-ai-manager/models/ai/main_test.go`
- Modify: `bin-ai-manager/models/ai/field_test.go`

**Step 1: Edit `main.go`**

Remove line 18 (`EngineType` field from AI struct) and lines 40-46 (EngineType type definition and constant):

```go
// Remove from struct:
	EngineType  EngineType     `json:"engine_type,omitempty" db:"engine_type"`   // currently not in used.

// Remove type definition:
// EngineType define
type EngineType string

// list of engine types
const (
	EngineTypeNone EngineType = ""
)
```

**Step 2: Edit `webhook.go`**

Remove `EngineType` from `WebhookMessage` struct (line 19) and from `ConvertWebhookMessage` (line 47):

```go
// Remove from struct:
	EngineType  EngineType     `json:"engine_type,omitempty"`

// Remove from ConvertWebhookMessage:
		EngineType:  h.EngineType,
```

**Step 3: Edit `field.go`**

Remove line 14:
```go
	FieldEngineType  Field = "engine_type"
```

**Step 4: Edit `filters.go`**

Remove line 11:
```go
	EngineType  EngineType  `filter:"engine_type"`
```

The `EngineType` type is removed, so the import is no longer needed. The struct now looks like:
```go
type FieldStruct struct {
	CustomerID  uuid.UUID   `filter:"customer_id"`
	Name        string      `filter:"name"`
	Detail      string      `filter:"detail"`
	EngineModel EngineModel `filter:"engine_model"`
	TTSType     TTSType     `filter:"tts_type"`
	STTType     STTType     `filter:"stt_type"`
	Deleted     bool        `filter:"deleted"`
}
```

**Step 5: Edit `main_test.go`**

- In `TestAI`: Remove `engineType` from test struct fields, test case values, AI construction, and assertion.
- Remove entire `TestEngineTypeConstants` function (lines 107-127).

**Step 6: Edit `field_test.go`**

Remove the `field_engine_type` test case (lines 33-37):
```go
		{
			name:     "field_engine_type",
			constant: FieldEngineType,
			expected: "engine_type",
		},
```

**Step 7: Run tests**

```bash
cd bin-ai-manager && go test ./models/ai/...
```
Expected: PASS

**Step 8: Commit**

```bash
git add bin-ai-manager/models/ai/
git commit -m "NOJIRA-remove-ai-engine-type

- bin-ai-manager: Remove EngineType from AI model, webhook, field, and filter structs"
```

---

### Task 3: Remove EngineType from ai-manager handlers

**Files:**
- Modify: `bin-ai-manager/pkg/aihandler/main.go` (interface + metric)
- Modify: `bin-ai-manager/pkg/aihandler/chatbot.go`
- Modify: `bin-ai-manager/pkg/aihandler/db.go`
- Modify: `bin-ai-manager/pkg/aihandler/chatbot_test.go`
- Modify: `bin-ai-manager/pkg/aihandler/db_test.go`
- Modify: `bin-ai-manager/pkg/listenhandler/models/request/ais.go`
- Modify: `bin-ai-manager/pkg/listenhandler/v1_ais.go`
- Modify: `bin-ai-manager/pkg/listenhandler/v1_ais_test.go`
- Modify: `bin-ai-manager/pkg/dbhandler/ai_test.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/start_test.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/db_test.go`
- Modify: `bin-ai-manager/cmd/ai-control/main.go`

**Step 1: Edit `pkg/aihandler/main.go`**

Remove `engineType ai.EngineType` parameter from both `Create` and `Update` in the interface (lines 27, 45).

Remove the `promAICreateTotal` metric definition (lines 68-76) and the `init()` function that registers it (lines 78-82). The metric is never incremented and only exists for EngineType tracking.

**Step 2: Edit `pkg/aihandler/chatbot.go`**

Remove `engineType ai.EngineType` parameter from `Create` (line 18) and `Update` (line 47).
Remove `engineType` from the `h.dbCreate(...)` call (line 33) and `h.dbUpdate(...)` call (line 62).

**Step 3: Edit `pkg/aihandler/db.go`**

Remove `engineType ai.EngineType` parameter from `dbCreate` (line 20) and `dbUpdate` (line 109).
Remove `EngineType: engineType,` from the AI struct in `dbCreate` (line 40).
Remove `ai.FieldEngineType: engineType,` from the fields map in `dbUpdate` (line 122).

**Step 4: Edit `pkg/aihandler/chatbot_test.go`**

In `TestCreate`: Remove `ai.EngineTypeNone` argument from `h.Create(...)` call (line 86).
In `TestUpdate`: Remove `ai.EngineTypeNone` argument from `h.Update(...)` call (line 182).

**Step 5: Edit `pkg/aihandler/db_test.go`**

In `Test_Create`:
- Remove `engineType ai.EngineType` from test struct (line 28)
- Remove `engineType: ai.EngineTypeNone,` from test case (line 48)
- Remove `EngineType: ai.EngineTypeNone,` from expectAI (line 73)
- Remove `tt.engineType,` from `h.Create(...)` call (line 111)

In `Test_Update`:
- Remove `engineType ai.EngineType` from test struct (line 307)
- Remove `engineType: ai.EngineTypeNone,` from test case (line 324)
- Remove `tt.engineType,` from `h.Update(...)` call (line 371)

**Step 6: Edit `pkg/listenhandler/models/request/ais.go`**

Remove `EngineType ai.EngineType` from both `V1DataAIsPost` (line 18) and `V1DataAIsIDPut` (line 40).

**Step 7: Edit `pkg/listenhandler/v1_ais.go`**

In `processV1AIsPost`: Remove `req.EngineType,` from `h.aiHandler.Create(...)` call (line 97).
In `processV1AIsIDPut`: Remove `req.EngineType,` from `h.aiHandler.Update(...)` call (line 222).

**Step 8: Edit `pkg/listenhandler/v1_ais_test.go`**

In `Test_processV1AIsPost`:
- Remove `expectEngineType` from struct (line 103) and test case (line 131)
- Remove `"engine_type":""` from JSON request body (line 119)
- Remove `tt.expectEngineType,` from mock expectation (line 167)

In `Test_processV1AIsIDPut`:
- Remove `expectEngineType` from struct (line 320) and test case (line 348)
- Remove `"engine_type":"",` from JSON request body (line 336)
- Remove `tt.expectEngineType,` from mock expectation (line 385)

**Step 9: Edit `pkg/dbhandler/ai_test.go`**

In `Test_AICreate`:
- Remove `EngineType: ai.EngineTypeNone,` from AI struct (line 42) and expectRes (line 62)

In `Test_AIUpdate`:
- Remove `ai.FieldEngineType: ai.EngineTypeNone,` from fields map (line 351)
- Remove `EngineType: ai.EngineTypeNone,` from expectRes (line 371)

**Step 10: Edit `pkg/aicallhandler/start_test.go`**

Remove `EngineType: ai.EngineTypeNone,` (lines 62, 260) and `EngineType: ai.EngineType("openai.gpt-3.5-turbo"),` (line 362) and `EngineType: ai.EngineType("test"),` (line 1217) from AI struct literals used in test fixtures.

**Step 11: Edit `pkg/aicallhandler/db_test.go`**

Remove `EngineType: ai.EngineTypeNone,` (line 49) from AI struct literal.

**Step 12: Edit `cmd/ai-control/main.go`**

In `cmdCreate()`: Remove `flags.String("engine-type", ...)` (line 93).
In `cmdUpdate()`: Remove `flags.String("engine-type", ...)` (line 143).
In `runCreate()`: Remove `engineType := ai.EngineType(viper.GetString("engine-type"))` (line 180) and `engineType,` from `handler.Create(...)` call (line 198).
In `runUpdate()`: Remove `engineType := ai.EngineType(viper.GetString("engine-type"))` (line 274) and `engineType,` from `handler.Update(...)` call (line 292).

**Step 13: Regenerate mocks and run tests**

```bash
cd bin-ai-manager
go generate ./...
go test ./...
```
Expected: PASS

**Step 14: Commit**

```bash
git add bin-ai-manager/
git commit -m "NOJIRA-remove-ai-engine-type

- bin-ai-manager: Remove EngineType from all handlers, request models, CLI, and tests
- bin-ai-manager: Remove unused promAICreateTotal metric"
```

---

### Task 4: Remove EngineType from bin-common-handler

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go`
- Modify: `bin-common-handler/pkg/requesthandler/ai_ais.go`
- Modify: `bin-common-handler/pkg/requesthandler/ai_ais_test.go`

**Step 1: Edit `main.go`**

Remove `engineType amai.EngineType,` from both `AIV1AICreate` (line 188) and `AIV1AIUpdate` (line 205) in the interface.

**Step 2: Edit `ai_ais.go`**

Remove `engineType amai.EngineType,` from `AIV1AICreate` (line 68) and `AIV1AIUpdate` (line 145) function signatures.
Remove `EngineType: engineType,` from the data struct in both functions (line 85, line 161).

**Step 3: Edit `ai_ais_test.go`**

In `Test_AIV1AICreate`:
- Remove `engineType amai.EngineType` from test struct (line 168)
- Remove `engineType: amai.EngineTypeNone,` from test case (line 188)
- Remove `tt.engineType,` from `reqHandler.AIV1AICreate(...)` call (line 234)

In `Test_AIV1AIUpdate`:
- Remove `engineType amai.EngineType` from test struct (line 316)
- Remove `engineType: amai.EngineTypeNone,` from test case (line 337)
- Remove `tt.engineType,` from `reqHandler.AIV1AIUpdate(...)` call (line 383)

**Step 4: Regenerate mocks and run tests**

```bash
cd bin-common-handler
go generate ./...
go test ./...
```
Expected: PASS

**Step 5: Commit**

```bash
git add bin-common-handler/
git commit -m "NOJIRA-remove-ai-engine-type

- bin-common-handler: Remove engineType from AIV1AICreate and AIV1AIUpdate RPC signatures"
```

---

### Task 5: Remove EngineType from OpenAPI spec and regenerate

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml`
- Modify: `bin-openapi-manager/openapi/paths/ais/main.yaml`
- Modify: `bin-openapi-manager/openapi/paths/ais/id.yaml`
- Regenerate: `bin-openapi-manager/gens/models/gen.go`

**Step 1: Edit `openapi.yaml`**

Remove the `AIManagerAIEngineType` schema definition (lines ~1677-1684):
```yaml
    AIManagerAIEngineType:
      type: string
      description: Type of engine used by the AI. Currently only empty string is defined in the backend.
      example: ""
      enum:
        - ""
      x-enum-varnames:
        - AIManagerAIEngineTypeNone
```

Remove `engine_type` from the `AIManagerAI` properties (lines ~1744-1745):
```yaml
        engine_type:
          $ref: '#/components/schemas/AIManagerAIEngineType'
          description: Type of AI engine.
          example: "chatGPT"
```

**Step 2: Edit `paths/ais/main.yaml`**

Remove `engine_type` property (lines 40-41):
```yaml
            engine_type:
              $ref: '#/components/schemas/AIManagerAIEngineType'
```

Remove `- engine_type` from required list (line 70).

**Step 3: Edit `paths/ais/id.yaml`**

Remove `engine_type` property (lines 64-65):
```yaml
            engine_type:
              $ref: '#/components/schemas/AIManagerAIEngineType'
```

Remove `- engine_type` from required list (line 94).

**Step 4: Regenerate and verify**

```bash
cd bin-openapi-manager
go generate ./...
go test ./...
golangci-lint run -v --timeout 5m
```
Expected: PASS

**Step 5: Commit**

```bash
git add bin-openapi-manager/
git commit -m "NOJIRA-remove-ai-engine-type

- bin-openapi-manager: Remove AIManagerAIEngineType schema and engine_type from AI request/response"
```

---

### Task 6: Remove EngineType from bin-api-manager

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/main.go`
- Modify: `bin-api-manager/pkg/servicehandler/ai.go`
- Modify: `bin-api-manager/pkg/servicehandler/ai_test.go`
- Modify: `bin-api-manager/server/ais.go`
- Modify: `bin-api-manager/server/ais_test.go`

**Step 1: Edit `pkg/servicehandler/main.go`**

Remove `engineType amai.EngineType,` from `AICreate` (line 236) and `AIUpdate` (line 255) in the interface.

**Step 2: Edit `pkg/servicehandler/ai.go`**

In `AICreate`: Remove `engineType amai.EngineType,` from signature (line 40), remove `"engine_type": engineType,` from log fields (line 55), remove `engineType,` from `h.reqHandler.AIV1AICreate(...)` call (line 76).

In `AIUpdate`: Remove `engineType amai.EngineType,` from signature (line 233), remove `"engine_type": engineType,` from log fields (line 248), remove `engineType,` from `h.reqHandler.AIV1AIUpdate(...)` call (line 276).

**Step 3: Edit `pkg/servicehandler/ai_test.go`**

In `Test_AICreate`:
- Remove `engineType amai.EngineType` from test struct (line 29)
- Remove `engineType: amai.EngineTypeNone,` from test case (line 53)
- Remove `tt.engineType,` from both `mockReq.EXPECT().AIV1AICreate(...)` (line 96) and `h.AICreate(...)` (line 112)

**Step 4: Edit `server/ais.go`**

In `PostAis`: Remove `amai.EngineType(req.EngineType),` from `h.serviceHandler.AICreate(...)` call (line 52).
In `PutAisId`: Remove `amai.EngineType(req.EngineType),` from `h.serviceHandler.AIUpdate(...)` call (line 235).

**Step 5: Edit `server/ais_test.go`**

In `Test_PostAis`:
- Remove `expectedEngineType amai.EngineType` from test struct (line 34)
- Remove `expectedEngineType: amai.EngineTypeNone,` from all test cases (lines 64, 96, 130)
- Remove `"engine_type":"",` from reqBody JSON strings (lines 54, 86, 118)
- Remove `tt.expectedEngineType,` from `mockSvc.EXPECT().AICreate(...)` call (line 169)

In `Test_PutAisId`:
- Remove `expectedEngineType amai.EngineType` from test struct (line 451)
- Remove `expectedEngineType: amai.EngineTypeNone,` from all test cases (lines 482, 515)
- Remove `"engine_type":"",` from reqBody JSON strings (lines 471, 504)
- Remove `tt.expectedEngineType,` from `mockSvc.EXPECT().AIUpdate(...)` call (line 556)

**Step 6: Vendor, regenerate, and verify**

```bash
cd bin-api-manager
go mod tidy && go mod vendor
go generate ./...
go test ./...
golangci-lint run -v --timeout 5m
```
Expected: PASS

**Step 7: Commit**

```bash
git add bin-api-manager/pkg/ bin-api-manager/server/ bin-api-manager/vendor/ bin-api-manager/go.mod bin-api-manager/go.sum bin-api-manager/gens/
git commit -m "NOJIRA-remove-ai-engine-type

- bin-api-manager: Remove engineType from service handler, server handlers, and all tests"
```

---

### Task 7: Update RST documentation

**Files:**
- Modify: `bin-api-manager/docsdev/source/ai_struct_ai.rst`

**Step 1: Edit RST**

Remove `"engine_type": "<string>",` from the JSON example (line 18).
Remove the `engine_type` field description (line 36):
```
* ``engine_type`` (String): Reserved for future use. Leave empty (``""``).
```
Remove `"engine_type": "",` from the full example (line 67).

**Step 2: Clean rebuild HTML**

```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
```

**Step 3: Commit**

```bash
git add bin-api-manager/docsdev/source/ai_struct_ai.rst
git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-remove-ai-engine-type

- bin-api-manager: Remove engine_type from AI struct RST documentation"
```

---

### Task 8: Run full verification on all affected services

**Step 1: Verify bin-ai-manager**

```bash
cd bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Verify bin-common-handler**

```bash
cd bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Verify bin-openapi-manager**

```bash
cd bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Verify bin-api-manager**

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

All must PASS before proceeding.

---

### Task 9: Final commit and PR

**Step 1: Check for conflicts with main**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

**Step 2: Push and create PR**

```bash
git push -u origin NOJIRA-remove-ai-engine-type
gh pr create --title "NOJIRA-remove-ai-engine-type" --body "Remove the dead EngineType field from the AI model across all layers.

- bin-dbscheme-manager: Add migration to drop engine_type column from ai_ais
- bin-ai-manager: Remove EngineType type, field, constants, and CLI flag
- bin-ai-manager: Remove unused promAICreateTotal metric
- bin-common-handler: Remove engineType from AIV1AICreate and AIV1AIUpdate RPC signatures
- bin-openapi-manager: Remove AIManagerAIEngineType schema and engine_type from request/response
- bin-api-manager: Remove engineType from service handler, server handlers, and tests
- bin-api-manager: Remove engine_type from RST documentation"
```
