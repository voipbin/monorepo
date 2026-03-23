# Add AI STT Language Field — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `stt_language` to the AI model, rename AICall's `Language` to `STTLanguage`, remove per-call language overrides, pass `""` for TTS language to Pipecat, and remove unused `resume` param from the aicall service start chain.

**Architecture:** `stt_language` on the AI config is the single source of truth. AICall copies it during creation. TTS language is implicit from voice ID. No backward compatibility shims. The `resume` field was dead code (never passed to `ServiceStart()`) and is removed entirely.

**Tech Stack:** Go monorepo, RabbitMQ RPC, MySQL/Alembic, OpenAPI codegen, Sphinx RST docs.

---

## Commit Strategy

Changes are grouped into compilable commits along service boundaries. Each commit compiles and passes tests independently.

| Commit | Services | Description |
|--------|----------|-------------|
| 1 | bin-ai-manager | All model, handler, CLI, and listen handler changes |
| 2 | bin-common-handler | Remove language and resume from RPC function signatures |
| 3 | bin-api-manager | Remove language from API layer |
| 4 | bin-flow-manager | Remove language and resume from flow action and caller |
| 5 | bin-dbscheme-manager | Alembic migration |
| 6 | bin-openapi-manager + bin-api-manager | OpenAPI spec + regenerated server code |
| 7 | bin-api-manager | RST documentation updates |

---

### Task 1: bin-ai-manager — All changes (single compilable commit)

This task covers all bin-ai-manager changes: AI model, AICall model, AIHandler, AICallHandler, ListenHandler, request models, CLI tool, and all tests.

**Files to modify:**
- `bin-ai-manager/models/ai/main.go`
- `bin-ai-manager/models/ai/field.go`
- `bin-ai-manager/models/ai/filters.go`
- `bin-ai-manager/models/ai/webhook.go`
- `bin-ai-manager/models/aicall/main.go`
- `bin-ai-manager/models/aicall/field.go`
- `bin-ai-manager/models/aicall/webhook.go`
- `bin-ai-manager/pkg/aihandler/main.go` (interface)
- `bin-ai-manager/pkg/aihandler/chatbot.go` (public wrappers)
- `bin-ai-manager/pkg/aihandler/db.go` (private DB methods)
- `bin-ai-manager/pkg/aicallhandler/main.go`
- `bin-ai-manager/pkg/aicallhandler/start.go`
- `bin-ai-manager/pkg/aicallhandler/service.go`
- `bin-ai-manager/pkg/aicallhandler/db.go`
- `bin-ai-manager/pkg/aicallhandler/chat.go`
- `bin-ai-manager/pkg/engine_dialogflow_handler/message.go`
- `bin-ai-manager/pkg/listenhandler/models/request/aicalls.go`
- `bin-ai-manager/pkg/listenhandler/models/request/services.go`
- `bin-ai-manager/pkg/listenhandler/models/request/ais.go`
- `bin-ai-manager/pkg/listenhandler/v1_aicalls.go`
- `bin-ai-manager/pkg/listenhandler/v1_services.go`
- `bin-ai-manager/pkg/listenhandler/v1_ais.go`
- `bin-ai-manager/cmd/ai-control/main.go`

**Test files to update:**
- `bin-ai-manager/models/aicall/main_test.go`
- `bin-ai-manager/models/aicall/webhook_test.go`
- `bin-ai-manager/pkg/aihandler/chatbot_test.go`
- `bin-ai-manager/pkg/aihandler/db_test.go`
- `bin-ai-manager/pkg/aicallhandler/start_test.go`
- `bin-ai-manager/pkg/aicallhandler/service_test.go`
- `bin-ai-manager/pkg/aicallhandler/send_test.go`

#### Step 1: AI Model — Add STTLanguage field

In `bin-ai-manager/models/ai/main.go`, add `STTLanguage` after `STTType`:

```go
// BEFORE (lines 62-64):
STTType          STTType    `json:"stt_type,omitempty" db:"stt_type"`
VADConfig        *VADConfig `json:"vad_config,omitempty" db:"vad_config,json"`

// AFTER:
STTType          STTType    `json:"stt_type,omitempty" db:"stt_type"`
STTLanguage      string     `json:"stt_language,omitempty" db:"stt_language"`
VADConfig        *VADConfig `json:"vad_config,omitempty" db:"vad_config,json"`
```

In `bin-ai-manager/models/ai/field.go`, add after `FieldSTTType`:

```go
FieldSTTLanguage Field = "stt_language"
```

In `bin-ai-manager/models/ai/filters.go`, add `STTLanguage` after `STTType` in `FieldStruct`:

```go
// BEFORE (lines 12-14):
STTType     STTType     `filter:"stt_type"`
Deleted     bool        `filter:"deleted"`

// AFTER:
STTType     STTType     `filter:"stt_type"`
STTLanguage string      `filter:"stt_language"`
Deleted     bool        `filter:"deleted"`
```

In `bin-ai-manager/models/ai/webhook.go`, add `STTLanguage` after `STTType` in the `WebhookMessage` struct:

```go
STTLanguage      string     `json:"stt_language,omitempty"`
```

And in `ConvertWebhookMessage()`, add:

```go
STTLanguage:      c.STTLanguage,
```

#### Step 2: AICall Model — Rename Language to STTLanguage

In `bin-ai-manager/models/aicall/main.go`, change line 39:

```go
// BEFORE:
Language string `json:"language,omitempty" db:"language"`

// AFTER:
STTLanguage string `json:"stt_language,omitempty" db:"stt_language"`
```

In `bin-ai-manager/models/aicall/field.go`, change:

```go
// BEFORE:
FieldLanguage Field = "language"

// AFTER:
FieldSTTLanguage Field = "stt_language"
```

In `bin-ai-manager/models/aicall/webhook.go`:

Struct field:
```go
// BEFORE:
Language string `json:"language,omitempty"`

// AFTER:
STTLanguage string `json:"stt_language,omitempty"`
```

In `ConvertWebhookMessage()`:
```go
// BEFORE:
Language: c.Language,

// AFTER:
STTLanguage: c.STTLanguage,
```

#### Step 3: AIHandler — Add sttLanguage param to Create/Update

In `bin-ai-manager/pkg/aihandler/main.go`, add `sttLanguage string` after `sttType ai.STTType` in both `Create()` and `Update()` interface methods:

```go
// Create — add after sttType:
sttType ai.STTType,
sttLanguage string,
toolNames []tool.ToolName,

// Update — add after sttType:
sttType ai.STTType,
sttLanguage string,
toolNames []tool.ToolName,
```

In `bin-ai-manager/pkg/aihandler/chatbot.go`:

`Create()` (line 15) — add `sttLanguage string` param after `sttType ai.STTType` (line 27). Update delegation call at line 49:
```go
// BEFORE:
res, err := h.dbCreate(ctx, customerID, name, detail, engineModel, parameter, engineKey, ragID, initPrompt, ttsType, ttsVoiceID, sttType, toolNames, vadConfig, smartTurnEnabled)

// AFTER:
res, err := h.dbCreate(ctx, customerID, name, detail, engineModel, parameter, engineKey, ragID, initPrompt, ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled)
```

`Update()` (line 58) — add `sttLanguage string` param after `sttType ai.STTType` (line 70). Update delegation call at line 92:
```go
// BEFORE:
res, err := h.dbUpdate(ctx, id, name, detail, engineModel, parameter, engineKey, ragID, initPrompt, ttsType, ttsVoiceID, sttType, toolNames, vadConfig, smartTurnEnabled)

// AFTER:
res, err := h.dbUpdate(ctx, id, name, detail, engineModel, parameter, engineKey, ragID, initPrompt, ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled)
```

In `bin-ai-manager/pkg/aihandler/db.go`:

`dbCreate` — add `sttLanguage string` param after `sttType` and set it in the struct:
```go
STTType:     sttType,
STTLanguage: sttLanguage,
```

`dbUpdate` — add `sttLanguage string` param after `sttType` and add to fields map:
```go
ai.FieldSTTLanguage: sttLanguage,
```

#### Step 4: AICall Handler — Remove language param, use AI config

In `bin-ai-manager/pkg/aicallhandler/main.go`:

Rename variable constant (line 81):
```go
// BEFORE:
variableLanguage      = "voipbin.aicall.language"

// AFTER:
variableSTTLanguage   = "voipbin.aicall.stt_language"
```

Remove `language string` from `Start()` interface (lines 41-50):
```go
Start(
    ctx context.Context,
    assistanceType aicall.AssistanceType,
    assistanceID uuid.UUID,
    activeflowID uuid.UUID,
    referenceType aicall.ReferenceType,
    referenceID uuid.UUID,
    gender aicall.Gender,
) (*aicall.AIcall, error)
```

Remove `language string` from `ServiceStart()` interface (lines 52-61):
```go
ServiceStart(
    ctx context.Context,
    assistanceType aicall.AssistanceType,
    assistanceID uuid.UUID,
    activeflowID uuid.UUID,
    referenceType aicall.ReferenceType,
    referenceID uuid.UUID,
    gender aicall.Gender,
) (*commonservice.Service, error)
```

In `bin-ai-manager/pkg/aicallhandler/chat.go`, line 30:
```go
// BEFORE:
variableLanguage:      cc.Language,

// AFTER:
variableSTTLanguage:   cc.STTLanguage,
```

In `bin-ai-manager/pkg/aicallhandler/db.go`:

`Create()` (line 18) — remove `language string` param. Change:
```go
// BEFORE:
Gender:   gender,
Language: language,

// AFTER:
Gender:      gender,
STTLanguage: c.STTLanguage,
```

`CreateByMessaging()` (line 93) — remove `language string` param. Change:
```go
// BEFORE:
Gender:   gender,
Language: language,

// AFTER:
Gender:      gender,
STTLanguage: c.STTLanguage,
```

In `bin-ai-manager/pkg/aicallhandler/start.go`:

Remove `language string` param from ALL functions in the call chain:
- `Start()` (line 78)
- `startReferenceTypeCall()` (line 119)
- `startReferenceTypeConversation()` (line 162)
- `startReferenceTypeNone()` (line 230)
- `startAIcallByRealtime()` (line 480) — also remove `language` from `h.Create()` call
- `startAIcallByMessaging()` (line 526) — also remove `language` from `h.CreateByMessaging()` call

In `startPipecatcall()` (lines 365-369), change STT/TTS language:
```go
// BEFORE:
sttType,
c.Language,    // STT language
ttsType,
c.Language,    // TTS language
ttsVoiceID,

// AFTER:
sttType,
c.STTLanguage, // STT language from AI config
ttsType,
"",            // TTS language empty — voice ID handles it
ttsVoiceID,
```

In `bin-ai-manager/pkg/aicallhandler/service.go`:

Remove `language string` param from:
- `ServiceStart()` (line 17)
- `serviceStartReferenceTypeCall()` (line 41) — also remove from log fields
- `serviceStartReferenceTypeConversation()` (line 87) — also remove from log fields

Update calls from `ServiceStart` to `Start` to drop `language`.

#### Step 4b: Engine Dialogflow Handler — Rename cc.Language to cc.STTLanguage

In `bin-ai-manager/pkg/engine_dialogflow_handler/message.go`, line 83:
```go
// BEFORE:
lang := GetLanguage(cc.Language)

// AFTER:
lang := GetLanguage(cc.STTLanguage)
```

This is the Dialogflow integration that extracts a 2-letter language code from the AIcall's language field for the `DetectIntentRequest.LanguageCode`. It uses `cc *aicall.AIcall` so it breaks when `Language` is renamed to `STTLanguage`.

#### Step 5: Listen Handler — Update callers and request models

In `bin-ai-manager/pkg/listenhandler/models/request/aicalls.go`, remove:
```go
// REMOVE:
Language string `json:"language,omitempty"`
```

In `bin-ai-manager/pkg/listenhandler/models/request/services.go`, remove both `Resume` and `Language` from `V1DataServicesTypeAIcallPost`:
```go
// REMOVE both lines:
Resume bool   `json:"resume"`
Language string `json:"language"`
```

The `Resume` field was dead code — it was never passed from `v1_services.go` to `ServiceStart()`. Resume/pause is not supported for aicall.

In `bin-ai-manager/pkg/listenhandler/models/request/ais.go`:

Add to `V1DataAIsPost` after `STTType`:
```go
STTType     ai.STTType `json:"stt_type,omitempty"`
STTLanguage string     `json:"stt_language,omitempty"`
```

Add to `V1DataAIsIDPut` after `STTType`:
```go
STTType     ai.STTType `json:"stt_type,omitempty"`
STTLanguage string     `json:"stt_language,omitempty"`
```

In `bin-ai-manager/pkg/listenhandler/v1_aicalls.go`, line 91 — remove `req.Language`:
```go
// BEFORE:
tmp, err := h.aicallHandler.Start(ctx, req.AssistanceType, req.AssistanceID, req.ActiveflowID, req.ReferenceType, req.ReferenceID, req.Gender, req.Language)

// AFTER:
tmp, err := h.aicallHandler.Start(ctx, req.AssistanceType, req.AssistanceID, req.ActiveflowID, req.ReferenceType, req.ReferenceID, req.Gender)
```

In `bin-ai-manager/pkg/listenhandler/v1_services.go`, line 27 — remove `req.Language`:
```go
// BEFORE:
tmp, err := h.aicallHandler.ServiceStart(ctx, req.AssistanceType, req.AssistanceID, req.ActiveflowID, req.ReferenceType, req.ReferenceID, req.Gender, req.Language)

// AFTER:
tmp, err := h.aicallHandler.ServiceStart(ctx, req.AssistanceType, req.AssistanceID, req.ActiveflowID, req.ReferenceType, req.ReferenceID, req.Gender)
```

In `bin-ai-manager/pkg/listenhandler/v1_ais.go`:

`processV1AIsPost()` (line 92) — add `req.STTLanguage` after `req.STTType`:
```go
tmp, err := h.aiHandler.Create(
    ctx,
    req.CustomerID,
    req.Name,
    req.Detail,
    req.EngineModel,
    req.Parameter,
    req.EngineKey,
    req.RagID,
    req.InitPrompt,
    req.TTSType,
    req.TTSVoiceID,
    req.STTType,
    req.STTLanguage,
    req.ToolNames,
    req.VADConfig,
    req.SmartTurnEnabled,
)
```

`processV1AIsIDPut()` (line 219) — add `req.STTLanguage` after `req.STTType`:
```go
tmp, err := h.aiHandler.Update(
    ctx,
    id,
    req.Name,
    req.Detail,
    req.EngineModel,
    req.Parameter,
    req.EngineKey,
    req.RagID,
    req.InitPrompt,
    req.TTSType,
    req.TTSVoiceID,
    req.STTType,
    req.STTLanguage,
    req.ToolNames,
    req.VADConfig,
    req.SmartTurnEnabled,
)
```

#### Step 6: ai-control CLI — Add --stt-language flag

In `bin-ai-manager/cmd/ai-control/main.go`:

`cmdCreate()` (line 83) — add flag after `stt-type` (line 99):
```go
flags.String("stt-type", "", "STT type (e.g., deepgram, elevenlabs)")
flags.String("stt-language", "", "STT language in BCP-47 format (e.g., en-US, ko-KR)")
```

`cmdUpdate()` (line 134) — add flag after `stt-type` (line 150):
```go
flags.String("stt-type", "", "STT type (e.g., deepgram, elevenlabs)")
flags.String("stt-language", "", "STT language in BCP-47 format (e.g., en-US, ko-KR)")
```

`runCreate()` (line 170) — add after `sttType` (line 188):
```go
sttType := ai.STTType(viper.GetString("stt-type"))
sttLanguage := viper.GetString("stt-language")
```

Update `handler.Create()` call (line 220) — add `sttLanguage` between `sttType` and `nil` (toolNames):
```go
res, err := handler.Create(
    context.Background(),
    customerID,
    name,
    detail,
    engineModel,
    map[string]any{}, // engineData
    engineKey,
    uuid.Nil,         // ragID
    initPrompt,
    ttsType,
    ttsVoiceID,
    sttType,
    sttLanguage,      // NEW
    nil,              // toolNames
    vadConfig,
    smartTurnEnabled,
)
```

`runUpdate()` (line 290) — add after `sttType` (line 308):
```go
sttType := ai.STTType(viper.GetString("stt-type"))
sttLanguage := viper.GetString("stt-language")
```

Update `handler.Update()` call (line 340) — add `sttLanguage` between `sttType` and `nil`:
```go
res, err := handler.Update(
    context.Background(),
    targetID,
    name,
    detail,
    engineModel,
    map[string]any{}, // engineData
    engineKey,
    uuid.Nil,         // ragID
    initPrompt,
    ttsType,
    ttsVoiceID,
    sttType,
    sttLanguage,      // NEW
    nil,              // toolNames
    vadConfig,
    smartTurnEnabled,
)
```

#### Step 7: Update all test files

Update test files to match new signatures and renamed fields. Key changes:

**Model tests (struct field rename):**
- `models/aicall/main_test.go` — rename `Language:` to `STTLanguage:` in struct literals (e.g., `Language: "en-US"` → `STTLanguage: "en-US"` at line 40, `Language: "ko-KR"` → `STTLanguage: "ko-KR"` at line 153) and assertions (`ac.Language` → `ac.STTLanguage` at lines 162-163)
- `models/aicall/webhook_test.go` — rename `Language:` to `STTLanguage:` in struct literals and `wh.Language`/`ac.Language` to `wh.STTLanguage`/`ac.STTLanguage` in assertions (lines 40, 76-77, 153)

**Handler tests (signature changes):**
- `pkg/aihandler/chatbot_test.go` and `db_test.go` — add `sttLanguage` param to Create/Update calls and mock expectations
- `pkg/aicallhandler/start_test.go` — remove `language` param from Start calls, rename `.Language` to `.STTLanguage` in aicall struct literals, update mock expectations
- `pkg/aicallhandler/service_test.go` — remove `language` param from ServiceStart calls, rename `.Language` to `.STTLanguage` in aicall struct literals
- `pkg/aicallhandler/send_test.go` — rename `.Language` to `.STTLanguage` in aicall struct literals, update `variableLanguage` references to `variableSTTLanguage`

The compiler will guide you to all locations that need updating.

#### Step 8: Regenerate mocks and run full verification

```bash
cd bin-ai-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

#### Step 9: Commit

```bash
git add bin-ai-manager/
git commit -m "NOJIRA-Add-AI-STT-language-field

Add stt_language field to AI model and rename AICall Language to STTLanguage.

- bin-ai-manager: Add STTLanguage to AI model, field constant, filters, and WebhookMessage
- bin-ai-manager: Rename AIcall Language to STTLanguage in model, field, and WebhookMessage
- bin-ai-manager: Add sttLanguage param to AIHandler Create and Update (interface, chatbot.go, db.go)
- bin-ai-manager: Remove language param from AIcallHandler Start/ServiceStart
- bin-ai-manager: Use AI config STTLanguage for STT, pass empty string for TTS language
- bin-ai-manager: Update Dialogflow engine handler to use cc.STTLanguage
- bin-ai-manager: Remove unused Resume field from services request model
- bin-ai-manager: Update listen handler request models and callers
- bin-ai-manager: Add --stt-language flag to ai-control CLI
- bin-ai-manager: Update all tests"
```

---

### Task 2: bin-common-handler — Remove language and resume from RPC functions

**Prerequisite:** Before making changes, verify all callers with grep:

```bash
# Verify callers of AIV1AIcallStart (expect: bin-api-manager/pkg/servicehandler/aicall.go + test files)
grep -rn "AIV1AIcallStart" --include="*.go" | grep -v "_test.go" | grep -v "mock_"

# Verify callers of AIV1ServiceTypeAIcallStart (expect: bin-flow-manager/pkg/activeflowhandler/actionhandle.go + test files)
grep -rn "AIV1ServiceTypeAIcallStart" --include="*.go" | grep -v "_test.go" | grep -v "mock_"
```

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go`
- Modify: `bin-common-handler/pkg/requesthandler/ai_aicalls.go`
- Modify: `bin-common-handler/pkg/requesthandler/ai_services.go`

**Test files to update:**
- `bin-common-handler/pkg/requesthandler/ai_aicalls_test.go`
- `bin-common-handler/pkg/requesthandler/ai_services_test.go`

**Step 1: Update AIV1AIcallStart — Remove language param**

In `bin-common-handler/pkg/requesthandler/ai_aicalls.go`, line 18:
```go
// BEFORE:
func (r *requestHandler) AIV1AIcallStart(ctx context.Context, assistanceType amaicall.AssistanceType, assistanceID uuid.UUID, activeflowID uuid.UUID, referenceType amaicall.ReferenceType, referenceID uuid.UUID, gender amaicall.Gender, language string) (*amaicall.AIcall, error) {

// AFTER:
func (r *requestHandler) AIV1AIcallStart(ctx context.Context, assistanceType amaicall.AssistanceType, assistanceID uuid.UUID, activeflowID uuid.UUID, referenceType amaicall.ReferenceType, referenceID uuid.UUID, gender amaicall.Gender) (*amaicall.AIcall, error) {
```

Remove `Language: language` from the request data (line 31):
```go
// BEFORE:
Gender:   gender,
Language: language,

// AFTER:
Gender: gender,
```

**Step 2: Update AIV1ServiceTypeAIcallStart — Remove resume and language params**

In `bin-common-handler/pkg/requesthandler/ai_services.go`, line 19:
```go
// BEFORE:
func (r *requestHandler) AIV1ServiceTypeAIcallStart(
    ctx context.Context,
    assistanceType amaicall.AssistanceType,
    assistanceID uuid.UUID,
    activeflowID uuid.UUID,
    referenceType amaicall.ReferenceType,
    referenceID uuid.UUID,
    resume bool,
    gender amaicall.Gender,
    language string,
    requestTimeout int,
) (*service.Service, error) {

// AFTER:
func (r *requestHandler) AIV1ServiceTypeAIcallStart(
    ctx context.Context,
    assistanceType amaicall.AssistanceType,
    assistanceID uuid.UUID,
    activeflowID uuid.UUID,
    referenceType amaicall.ReferenceType,
    referenceID uuid.UUID,
    gender amaicall.Gender,
    requestTimeout int,
) (*service.Service, error) {
```

Remove `Resume: resume` and `Language: language` from request data (lines 39-41):
```go
// BEFORE:
Resume:         resume,
Gender:         gender,
Language:       language,

// AFTER:
Gender: gender,
```

**NOTE:** `AIV1ServiceTypeSummaryStart()` in the same file is UNCHANGED — it has its own `language` param for summary language, which is unrelated.

**Step 3: Update interface in main.go**

In `bin-common-handler/pkg/requesthandler/main.go`:

Line 247 — remove `language string`:
```go
// BEFORE:
AIV1AIcallStart(ctx context.Context, assistanceType amaicall.AssistanceType, assistanceID uuid.UUID, activeflowID uuid.UUID, referenceType amaicall.ReferenceType, referenceID uuid.UUID, gender amaicall.Gender, language string) (*amaicall.AIcall, error)

// AFTER:
AIV1AIcallStart(ctx context.Context, assistanceType amaicall.AssistanceType, assistanceID uuid.UUID, activeflowID uuid.UUID, referenceType amaicall.ReferenceType, referenceID uuid.UUID, gender amaicall.Gender) (*amaicall.AIcall, error)
```

Lines 276-287 — remove `resume bool` and `language string`:
```go
// BEFORE:
AIV1ServiceTypeAIcallStart(
    ctx context.Context,
    assistanceType amaicall.AssistanceType,
    assistanceID uuid.UUID,
    activeflowID uuid.UUID,
    referenceType amaicall.ReferenceType,
    referenceID uuid.UUID,
    resume bool,
    gender amaicall.Gender,
    language string,
    requestTimeout int,
) (*service.Service, error)

// AFTER:
AIV1ServiceTypeAIcallStart(
    ctx context.Context,
    assistanceType amaicall.AssistanceType,
    assistanceID uuid.UUID,
    activeflowID uuid.UUID,
    referenceType amaicall.ReferenceType,
    referenceID uuid.UUID,
    gender amaicall.Gender,
    requestTimeout int,
) (*service.Service, error)
```

**Step 4: Update tests and regenerate mocks**

Update `ai_aicalls_test.go` and `ai_services_test.go` — remove `language` from call args and mock expectations.

```bash
cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
git add bin-common-handler/
git commit -m "NOJIRA-Add-AI-STT-language-field

- bin-common-handler: Remove language param from AIV1AIcallStart
- bin-common-handler: Remove resume and language params from AIV1ServiceTypeAIcallStart"
```

---

### Task 3: bin-api-manager — Remove language from API layer

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/main.go` (interface)
- Modify: `bin-api-manager/pkg/servicehandler/aicall.go` (implementation)
- Modify: `bin-api-manager/server/aicalls.go` (HTTP handler)

**Test files to update:**
- `bin-api-manager/pkg/servicehandler/aicall_test.go`
- `bin-api-manager/server/aicalls_test.go`

**Step 1: Update ServiceHandler interface**

In `bin-api-manager/pkg/servicehandler/main.go`, lines 279-288 — remove `language string`:

```go
// BEFORE:
AIcallCreate(
    ctx context.Context,
    a *amagent.Agent,
    assistanceType amaicall.AssistanceType,
    assistanceID uuid.UUID,
    referenceType amaicall.ReferenceType,
    referenceID uuid.UUID,
    gender amaicall.Gender,
    language string,
) (*amaicall.WebhookMessage, error)

// AFTER:
AIcallCreate(
    ctx context.Context,
    a *amagent.Agent,
    assistanceType amaicall.AssistanceType,
    assistanceID uuid.UUID,
    referenceType amaicall.ReferenceType,
    referenceID uuid.UUID,
    gender amaicall.Gender,
) (*amaicall.WebhookMessage, error)
```

**Step 2: Update AIcallCreate implementation**

In `bin-api-manager/pkg/servicehandler/aicall.go`, line 25 — remove `language string` from func signature:

```go
// BEFORE:
func (h *serviceHandler) AIcallCreate(ctx context.Context, a *amagent.Agent, assistanceType amaicall.AssistanceType, assistanceID uuid.UUID, referenceType amaicall.ReferenceType, referenceID uuid.UUID, gender amaicall.Gender, language string) (*amaicall.WebhookMessage, error) {

// AFTER:
func (h *serviceHandler) AIcallCreate(ctx context.Context, a *amagent.Agent, assistanceType amaicall.AssistanceType, assistanceID uuid.UUID, referenceType amaicall.ReferenceType, referenceID uuid.UUID, gender amaicall.Gender) (*amaicall.WebhookMessage, error) {
```

And remove `language` from `AIV1AIcallStart()` call (line 51):
```go
// BEFORE:
tmp, err := h.reqHandler.AIV1AIcallStart(
    ctx,
    assistanceType,
    assistanceID,
    uuid.Nil,
    referenceType,
    referenceID,
    gender,
    language,
)

// AFTER:
tmp, err := h.reqHandler.AIV1AIcallStart(
    ctx,
    assistanceType,
    assistanceID,
    uuid.Nil,
    referenceType,
    referenceID,
    gender,
)
```

**Step 3: Update server/aicalls.go HTTP handler**

In `bin-api-manager/server/aicalls.go`, line 40 — remove `req.Language`:

```go
// BEFORE:
res, err := h.serviceHandler.AIcallCreate(c.Request.Context(), &a, amaicall.AssistanceType(req.AssistanceType), assistanceID, amaicall.ReferenceType(req.ReferenceType), referenceID, amaicall.Gender(req.Gender), req.Language)

// AFTER:
res, err := h.serviceHandler.AIcallCreate(c.Request.Context(), &a, amaicall.AssistanceType(req.AssistanceType), assistanceID, amaicall.ReferenceType(req.ReferenceType), referenceID, amaicall.Gender(req.Gender))
```

**Step 4: Update tests, regenerate mocks, verify**

Update `aicall_test.go` and `server/aicalls_test.go` — remove `language` from call args and mock expectations.

```bash
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
git add bin-api-manager/
git commit -m "NOJIRA-Add-AI-STT-language-field

- bin-api-manager: Remove language param from AIcallCreate in ServiceHandler interface, implementation, and HTTP handler
- bin-api-manager: Update tests"
```

---

### Task 4: bin-flow-manager — Remove Resume and Language from OptionAITalk and caller

**Files:**
- Modify: `bin-flow-manager/models/action/option.go`
- Modify: `bin-flow-manager/pkg/activeflowhandler/actionhandle.go`

**Test files to update:**
- `bin-flow-manager/pkg/activeflowhandler/actionhandle_test.go`

**Step 1: Remove Resume and Language from OptionAITalk**

In `bin-flow-manager/models/action/option.go`, remove lines 58 and 60:
```go
// REMOVE both lines:
Resume         bool                   `json:"resume,omitempty"` // resume the previous ai talk.
Language       string                 `json:"language,omitempty"` // BCP47 format. en-US
```

Resume/pause is not supported for aicall. The field was dead code (never passed to `ServiceStart()`).

**Step 2: Remove opt.Resume and opt.Language from AIV1ServiceTypeAIcallStart call**

In `bin-flow-manager/pkg/activeflowhandler/actionhandle.go`, line 1086:
```go
// BEFORE:
sv, err := h.reqHandler.AIV1ServiceTypeAIcallStart(ctx, assistanceType, assistanceID, af.ID, referenceType, af.ReferenceID, opt.Resume, opt.Gender, opt.Language, 30000)

// AFTER:
sv, err := h.reqHandler.AIV1ServiceTypeAIcallStart(ctx, assistanceType, assistanceID, af.ID, referenceType, af.ReferenceID, opt.Gender, 30000)
```

**Step 3: Update tests, regenerate mocks, verify**

Update `actionhandle_test.go` — remove `Resume` and `Language` from mock expectations and test data.

```bash
cd bin-flow-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
git add bin-flow-manager/
git commit -m "NOJIRA-Add-AI-STT-language-field

- bin-flow-manager: Remove Resume and Language from OptionAITalk
- bin-flow-manager: Remove resume and language params from AIV1ServiceTypeAIcallStart call"
```

---

### Task 5: Database Migration

**Files:**
- Create: `bin-dbscheme-manager/alembic/versions/<auto>_add_ai_stt_language.py`

**Step 1: Create Alembic migration**

```bash
cd bin-dbscheme-manager
alembic -c alembic.ini revision -m "add ai stt language"
```

**Step 2: Edit the migration file**

```python
def upgrade() -> None:
    # Add stt_language to ais table
    op.add_column('ais', sa.Column('stt_language', sa.String(16), nullable=False, server_default=''))

    # Rename language → stt_language in aicalls table
    op.alter_column('aicalls', 'language', new_column_name='stt_language')


def downgrade() -> None:
    # Rename stt_language → language in aicalls table
    op.alter_column('aicalls', 'stt_language', new_column_name='language')

    # Remove stt_language from ais table
    op.drop_column('ais', 'stt_language')
```

**Step 3: Commit**

```bash
git add bin-dbscheme-manager/alembic/versions/
git commit -m "NOJIRA-Add-AI-STT-language-field

- bin-dbscheme-manager: Add migration for stt_language column on ais table and rename language to stt_language on aicalls table"
```

---

### Task 6: OpenAPI Spec Updates

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml`

**Step 1: Update AI schema — Add stt_language**

In the `AIManagerAI` schema, add `stt_language` after `stt_type`:

```yaml
stt_language:
  type: string
  description: "STT language in BCP-47 format (e.g., ko-KR, en-US). Empty for auto-detect."
```

**Step 2: Update AIcall schema — Rename language to stt_language**

In the `AIManagerAIcall` schema, rename `language` → `stt_language`:

```yaml
stt_language:
  type: string
  description: "STT language copied from AI config at creation time."
```

**Step 3: Update AI POST/PUT request body — Add stt_language**

Add `stt_language` to the POST request body for `/ais` and PUT request body for `/ais/{id}`.

**Step 4: Update AIcall POST request body — Remove language**

Remove `language` from the POST request body for `/aicalls`. Remove it from `required` if listed.

**Step 5: Update FlowManagerActionOptionAITalk — Remove resume and language**

Remove `resume` and `language` from the `FlowManagerActionOptionAITalk` schema.

**Step 6: Regenerate and verify**

```bash
cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 7: Commit**

```bash
git add bin-openapi-manager/ bin-api-manager/gens/
git commit -m "NOJIRA-Add-AI-STT-language-field

- bin-openapi-manager: Add stt_language to AI schema, rename in AIcall schema, remove resume and language from flow action
- bin-api-manager: Regenerate OpenAPI server code"
```

---

### Task 7: RST Documentation Updates

**Files:**
- Modify: `bin-api-manager/docsdev/source/ai_struct_ai.rst`
- Modify: `bin-api-manager/docsdev/source/ai_tutorial.rst`
- Modify: `bin-api-manager/docsdev/source/ai_overview.rst`

**Step 1: Update ai_struct_ai.rst — Add stt_language field**

Add `stt_language` field documentation after `stt_type`:

```rst
* ``stt_language`` (String, Optional): STT language in BCP-47 format (e.g., ``ko-KR``, ``en-US``).
  Controls which language the Speech-to-Text engine listens for.
  Empty string means auto-detect (Deepgram) or default to ``en-US`` (Google).
```

**Step 2: Update ai_tutorial.rst — Add stt_language to examples**

Update AI creation examples to include `stt_language` and remove `language` from flow action examples.

**Step 3: Update ai_overview.rst — Update multilingual section**

Update the multilingual support section to explain that STT language is configured on the AI, and TTS language is determined by voice ID.

**Step 4: Rebuild HTML**

```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
```

**Step 5: Commit**

```bash
git add bin-api-manager/docsdev/source/ && git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-Add-AI-STT-language-field

- bin-api-manager: Update RST docs for stt_language field, rebuild HTML"
```

---

### Task 8: Final Verification

**Step 1: Run full verification for all 5 affected services**

```bash
cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

cd bin-ai-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

cd bin-flow-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Verify all tests pass with zero failures**

**Step 3: Create PR**

```bash
git push -u origin NOJIRA-Add-AI-STT-language-field
gh pr create --title "NOJIRA-Add-AI-STT-language-field" --body "..."
```

---

## Deploy Order

1. **Alembic migration** — Add `stt_language` to `ais` table, rename `language` → `stt_language` in `aicalls` table. Must run first.
2. **Code deploy** — All services together (standard k8s rolling update).
