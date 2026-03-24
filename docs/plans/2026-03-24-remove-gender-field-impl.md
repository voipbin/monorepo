# Remove Gender Field from AIcall — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove the redundant `gender` field from the AIcall creation/start chain, OpenAPI schema, database, and documentation.

**Architecture:** Pure removal — no new logic. Gender is threaded through the AIcall chain as a parameter but provides no value beyond what the TTS voice ID already encodes. The `aicall.Gender` type in bin-ai-manager is removed entirely. TTS streaming gender (bin-tts-manager) is out of scope.

**Tech Stack:** Go, OpenAPI 3.0 (oapi-codegen), Alembic (Python), RST/Sphinx

**Cross-Service Dependency Note:** Services use local `replace` directives in go.mod. Removing the `aicall.Gender` type from bin-ai-manager breaks bin-common-handler (which imports it), which breaks bin-api-manager and bin-flow-manager. **All Go code changes (Tasks 1–6) must be completed before any service will compile.** Run verification only after ALL code tasks are done.

---

### Task 1: Remove Gender from bin-ai-manager models

**Files:**
- Modify: `bin-ai-manager/models/aicall/main.go`
- Modify: `bin-ai-manager/models/aicall/field.go`
- Modify: `bin-ai-manager/models/aicall/filters.go`
- Modify: `bin-ai-manager/models/aicall/webhook.go`
- Modify: `bin-ai-manager/models/aicall/main_test.go`
- Modify: `bin-ai-manager/models/aicall/field_test.go`
- Modify: `bin-ai-manager/models/aicall/filters_test.go`
- Modify: `bin-ai-manager/models/aicall/webhook_test.go`

**Step 1: Remove Gender type, constants, and struct field from main.go**

Remove the `Gender` field from the AIcall struct (~line 38):
```go
// REMOVE this line from the AIcall struct:
Gender Gender `json:"gender,omitempty" db:"gender"`
```

Remove the Gender type definition and constants (~lines 81-89):
```go
// REMOVE this entire block:
type Gender string

const (
	GenderNone    Gender = ""
	GenderMale    Gender = "male"
	GenderFemale  Gender = "female"
	GenderNeutral Gender = "neutral"
)
```

**Step 2: Remove FieldGender from field.go**

Remove (~line 33):
```go
// REMOVE this line:
FieldGender Field = "gender"
```

**Step 3: Remove Gender from FieldStruct in filters.go**

Remove (~line 21):
```go
// REMOVE this line from FieldStruct:
Gender Gender `filter:"gender"`
```

**Step 4: Remove Gender from WebhookMessage in webhook.go**

Remove from WebhookMessage struct (~line 39):
```go
// REMOVE this line:
Gender Gender `json:"gender,omitempty"`
```

Remove from ConvertWebhookMessage() (~line 75):
```go
// REMOVE this line:
Gender: h.Gender,
```

**Step 5: Update test files**

- `main_test.go`: Remove `TestGenderConstants()` function entirely. Remove `Gender` field from all AIcall struct initializers in test cases.
- `field_test.go`: Remove `FieldGender` test case.
- `filters_test.go`: Remove `Gender` from FieldStruct test initializers.
- `webhook_test.go`: Remove `Gender` field from WebhookMessage test initializers and assertions.

---

### Task 2: Remove Gender from bin-ai-manager handlers

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/main.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/start.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/service.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/db.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/chat.go`
- Modify: `bin-ai-manager/pkg/listenhandler/models/request/aicalls.go`
- Modify: `bin-ai-manager/pkg/listenhandler/models/request/services.go`
- Modify: `bin-ai-manager/pkg/listenhandler/v1_aicalls.go`
- Modify: `bin-ai-manager/pkg/listenhandler/v1_services.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/start_test.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/service_test.go`
- Modify: `bin-ai-manager/pkg/aicallhandler/db_test.go`
- Modify: `bin-ai-manager/pkg/dbhandler/aicall_test.go`
- Modify: `bin-ai-manager/pkg/listenhandler/v1_aicalls_test.go`
- Modify: `bin-ai-manager/pkg/listenhandler/v1_services_test.go`
- Auto-regenerated: `bin-ai-manager/pkg/aicallhandler/mock_main.go`

**Step 1: Remove from aicallhandler/main.go**

Remove `gender aicall.Gender` parameter from `Start()` interface (~lines 41-49).

Remove `gender aicall.Gender` parameter from `ServiceStart()` interface (~lines 51-59).

Remove `variableGender` constant (~line 78):
```go
// REMOVE this line:
variableGender = "voipbin.aicall.gender"
```

**Step 2: Remove from aicallhandler/start.go**

Remove `gender aicall.Gender` parameter from all functions:
- `Start()` (~line 85) — remove param and stop passing to `startReferenceType*`
- `startReferenceTypeCall()` (~line 117) — remove param
- `startReferenceTypeConversation()` (~line 159) — remove param
- `startReferenceTypeNone()` (~line 226) — remove param
- `startAIcallByRealtime()` (~line 475) — remove param, stop passing to `h.Create()`
- `startAIcallByMessaging()` (~line 520) — remove param, stop passing to `h.CreateByMessaging()`

In `StartTask()` (~line 572), remove the `aicall.GenderNone` argument from the `h.Start()` call.

**Step 3: Remove from aicallhandler/service.go**

Remove `gender aicall.Gender` parameter from:
- `ServiceStart()` (~line 24) — remove param, stop passing to `serviceStartReferenceType*`
- `serviceStartReferenceTypeCall()` (~line 46) — remove param and `"gender": gender` from debug log fields (~line 54)
- `serviceStartReferenceTypeConversation()` (~line 90) — remove param and `"gender": gender` from debug log fields (~line 98)

**Step 4: Remove from aicallhandler/db.go**

Remove `gender aicall.Gender` parameter from:
- `Create()` (~line 29) — remove param and `Gender: gender` from struct init (~line 64)
- `CreateByMessaging()` (~line 102) — remove param and `Gender: gender` from struct init (~line 132)

**Step 5: Remove from aicallhandler/chat.go**

Remove from `setActiveflowVariables()` (~line 29):
```go
// REMOVE this line:
variableGender: string(cc.Gender),
```

**Step 6: Remove from listenhandler request models**

In `bin-ai-manager/pkg/listenhandler/models/request/aicalls.go`, remove (~line 22):
```go
// REMOVE this line from V1DataAIcallsPost:
Gender aicall.Gender `json:"gender,omitempty"`
```

In `bin-ai-manager/pkg/listenhandler/models/request/services.go`, remove (~line 21):
```go
// REMOVE this line from V1DataServicesPost:
Gender aicall.Gender `json:"gender"`
```

**Step 7: Remove from listenhandler source files**

In `bin-ai-manager/pkg/listenhandler/v1_aicalls.go`, remove `req.Gender` from the `h.Start()` call (~line 91).

In `bin-ai-manager/pkg/listenhandler/v1_services.go`, remove `req.Gender` from the `h.aicallHandler.ServiceStart()` call (~line 27).

**Step 8: Update test files**

- `start_test.go`: Remove `gender` field from all test case structs, remove `aicall.Gender*` values, remove `Gender` from struct initializers, remove `variableGender` from expected variable maps.
- `service_test.go`: Remove `gender` field from test case structs, remove `aicall.Gender*` values, remove `Gender` from struct initializers.
- `db_test.go`: Remove `gender` field from test case structs, remove `Gender` from struct initializers.
- `pkg/dbhandler/aicall_test.go`: Remove `Gender` from AIcall struct initializers.
- `v1_aicalls_test.go`: Remove `expectedGender` from test case structs, remove gender from JSON test data and mock expectations.
- `v1_services_test.go`: Remove `expectedGender` from test case structs, remove gender from JSON test data and mock expectations.

---

### Task 3: Remove Gender from bin-common-handler

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go`
- Modify: `bin-common-handler/pkg/requesthandler/ai_aicalls.go`
- Modify: `bin-common-handler/pkg/requesthandler/ai_services.go`
- Modify: `bin-common-handler/pkg/requesthandler/ai_aicalls_test.go`
- Modify: `bin-common-handler/pkg/requesthandler/ai_services_test.go`
- Auto-regenerated: `bin-common-handler/pkg/requesthandler/mock_main.go`

**Step 1: Remove from interface in main.go**

Remove `gender amaicall.Gender` param from `AIV1AIcallStart()` (~line 249).

Remove `gender amaicall.Gender` param from `AIV1ServiceTypeAIcallStart()` (~lines 278-287).

**Step 2: Remove from ai_aicalls.go implementation**

Remove `gender` param from function signature and remove `Gender: gender` from the request data struct.

**Step 3: Remove from ai_services.go implementation**

Remove `gender` param from function signature and remove `Gender: gender` from the request data struct.

**Step 4: Update test files**

- `ai_aicalls_test.go`: Remove `gender` field from test case structs and `amaicall.Gender*` values.
- `ai_services_test.go`: Remove `gender` field from test case structs and `amaicall.Gender*` values.

**Note:** `go generate` will regenerate `mock_main.go`. No manual mock updates needed.

---

### Task 4: Remove Gender from bin-api-manager

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/main.go`
- Modify: `bin-api-manager/pkg/servicehandler/aicall.go`
- Modify: `bin-api-manager/server/aicalls.go`
- Modify: `bin-api-manager/server/aicalls_test.go`
- Modify: `bin-api-manager/pkg/servicehandler/aicall_test.go`
- Auto-regenerated: `bin-api-manager/pkg/servicehandler/mock_main.go`

**Step 1: Remove from servicehandler interface in main.go**

Remove `gender amaicall.Gender` param from `AIcallCreate()` (~lines 281-289).

**Step 2: Remove from servicehandler/aicall.go implementation**

Remove `gender` param from the function signature. Remove `gender` from the `h.reqHandler.AIV1AIcallStart()` call.

**Step 3: Remove from server/aicalls.go**

Remove the `amaicall.Gender(req.Gender)` conversion and stop passing gender to `h.serviceHandler.AIcallCreate()`.

**Step 4: Update test files**

- `server/aicalls_test.go`: Remove `expectedGender` field from test case structs, remove `amaicall.Gender*` values, remove gender from mock expectations.
- `pkg/servicehandler/aicall_test.go`: Remove gender fields and params from test cases and mock expectations.

**Note:** `go generate` will regenerate `mock_main.go`. No manual mock updates needed.

---

### Task 5: Remove Gender from bin-flow-manager

**Files:**
- Modify: `bin-flow-manager/models/action/option.go`
- Modify: `bin-flow-manager/pkg/activeflowhandler/actionhandle.go`
- Modify: `bin-flow-manager/models/action/option_test.go`
- Modify: `bin-flow-manager/pkg/activeflowhandler/actionhandle_test.go`

**Step 1: Remove from OptionAITalk in option.go**

Remove (~line 58):
```go
// REMOVE this line from OptionAITalk:
Gender amaicall.Gender `json:"gender,omitempty"`
```

**Step 2: Remove from actionhandle.go**

Remove `opt.Gender` from the `h.reqHandler.AIV1ServiceTypeAIcallStart()` call (~line 1086).

**Step 3: Update test files**

- `option_test.go`: Remove `Gender` from OptionAITalk struct initializers.
- `actionhandle_test.go`: Remove `expectedGender` field from test case structs, remove `amaicall.Gender*` values, remove gender from mock expectations.

---

### Task 6: Update OpenAPI spec and regenerate

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml`
- Modify: `bin-openapi-manager/openapi/paths/aicalls/main.yaml`
- Regenerate: `bin-openapi-manager/gens/models/gen.go`
- Regenerate: `bin-api-manager/gens/openapi_server/gen.go`

**Step 1: Remove AIManagerAIcallGender enum schema from openapi.yaml**

Remove the entire schema definition (~lines 2100-2113):
```yaml
AIManagerAIcallGender:
  type: string
  description: Gender associated with the AI call.
  example: "female"
  enum:
    - ""
    - male
    - female
    - neutral
  x-enum-varnames:
    - AIManagerAIcallGenderNone
    - AIManagerAIcallGenderMale
    - AIManagerAIcallGenderFemale
    - AIManagerAIcallGenderNeutral
```

**Step 2: Remove gender property from AIManagerAIcall response schema**

Remove (~lines 2330-2333):
```yaml
gender:
  $ref: '#/components/schemas/AIManagerAIcallGender'
  description: Gender associated with the AI call.
  example: "female"
```

**Step 3: Remove gender from POST /aicalls request body**

In `bin-openapi-manager/openapi/paths/aicalls/main.yaml`:
- Remove `gender` property (~lines 44-45)
- Remove `gender` from the `required` list (~line 51)

**Step 4: Remove gender from FlowManagerActionOptionAITalk**

Remove (~lines 3853-3856):
```yaml
gender:
  $ref: '#/components/schemas/AIManagerAIcallGender'
  description: Voice gender for the AI.
  example: "female"
```

**Step 5: Regenerate models**

```bash
cd bin-openapi-manager && go generate ./...
cd ../bin-api-manager && go generate ./...
```

---

### Task 7: Verify all services

Run the full verification workflow for every changed service. All code changes (Tasks 1–6) must be complete before this step.

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

**Step 3: Verify bin-api-manager**
```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Verify bin-flow-manager**
```bash
cd bin-flow-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Verify bin-openapi-manager**
```bash
cd bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

---

### Task 8: Create Alembic migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<generated>_ai_aicalls_drop_column_gender.py`

**Step 1: Generate the migration file**

```bash
cd bin-dbscheme-manager/bin-manager
alembic -c alembic.ini revision -m "ai_aicalls drop column gender"
```

**Step 2: Edit the generated migration file**

```python
def upgrade():
    op.execute("""ALTER TABLE ai_aicalls DROP COLUMN gender""")

def downgrade():
    op.execute("""ALTER TABLE ai_aicalls ADD COLUMN gender VARCHAR(255) NOT NULL DEFAULT ''""")
```

---

### Task 9: Update RST documentation

**Files:**
- Modify: `bin-api-manager/docsdev/source/ai_overview.rst` (~line 378)
- Modify: `bin-api-manager/docsdev/source/call_overview.rst` (~line 20)
- Modify: `bin-api-manager/docsdev/source/variable_variable.rst` (~line 89)
- Modify: `bin-api-manager/docsdev/source/flow_advanced_patterns.rst` (~line 448)
- Modify: `bin-api-manager/docsdev/source/flow_struct_action.rst` (~lines 332, 339)
- Modify: `bin-api-manager/docsdev/source/quickstart_realtime.rst` (~line 55)

**Step 1: Remove gender references from each RST file**

- `ai_overview.rst`: Remove/reword the sentence about "various languages and genders" — the voice ID table already shows the voice names, no need to reference gender.
- `call_overview.rst` (~line 20): Reword "different gender and accents" to "different voices and accents".
- `variable_variable.rst`: Remove the `voipbin.aicall.gender` variable entry entirely.
- `flow_advanced_patterns.rst`: Remove `"gender": "female"` from the AI talk action example JSON.
- `flow_struct_action.rst`: Remove `"gender": "<string>"` from JSON structure and remove the gender field description line.
- `quickstart_realtime.rst`: Remove `"gender": "female"` from the talk action example JSON.

**Step 2: Rebuild HTML docs**

```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
```

**Step 3: Force-add built HTML**

```bash
git add -f bin-api-manager/docsdev/build/
```

---

### Task 10: Commit and push

**Step 1: Stage all changes**

```bash
git add -A
# Remove any vendor files that got staged
git reset HEAD */vendor/
```

**Step 2: Commit**

```bash
git commit -m "NOJIRA-Remove-gender-field

Remove redundant gender field from the AIcall creation chain. The TTS voice ID
already determines voice characteristics, making gender unnecessary.

- bin-ai-manager: Remove Gender type, constants, model field, webhook field, handler params
- bin-ai-manager: Remove gender from aicallhandler Start/ServiceStart/Create chain
- bin-ai-manager: Remove gender from listenhandler request models and handlers
- bin-ai-manager: Remove voipbin.aicall.gender activeflow variable
- bin-common-handler: Remove gender param from AIV1AIcallStart and AIV1ServiceTypeAIcallStart
- bin-api-manager: Remove gender from AIcallCreate interface and server handler
- bin-flow-manager: Remove gender from OptionAITalk and actionhandle
- bin-openapi-manager: Remove AIManagerAIcallGender enum and gender properties from AIcall and FlowManagerActionOptionAITalk
- bin-dbscheme-manager: Add migration to drop gender column from ai_aicalls
- docs: Remove gender references from RST documentation"
```

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-Remove-gender-field
```
