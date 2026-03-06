# Configurable VAD Parameters Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make all 4 Pipecat VAD parameters (confidence, start_secs, stop_secs, min_volume) configurable per-AI agent, removing the hardcoded 0.5s stop_secs override.

**Architecture:** Add a `VADConfig` struct to the AI model, snapshot it into AIcall at creation, pass it through to pipecat-manager's Python runner where it builds `VADParams`. When no config is set, Pipecat's native defaults apply.

**Tech Stack:** Go (ai-manager, pipecat-manager), Python (pipecat scripts), MySQL (Alembic migration), OpenAPI

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-configurable-vad-config`

---

### Task 1: Add VADConfig struct and field to AI model

**Files:**
- Modify: `bin-ai-manager/models/ai/main.go:28` (add struct + field after STTType)
- Modify: `bin-ai-manager/models/ai/field.go:23` (add FieldVADConfig after FieldSTTType)
- Modify: `bin-ai-manager/models/ai/webhook.go:28-62` (add VADConfig to WebhookMessage + ConvertWebhookMessage)

**Step 1: Add VADConfig struct and field to AI model**

In `bin-ai-manager/models/ai/main.go`, add the struct before the `AI` struct and the field after `STTType`:

```go
// VADConfig holds Voice Activity Detection parameters.
// Nil pointer fields mean "use Pipecat default".
// Pipecat defaults: confidence=0.7, start_secs=0.2, stop_secs=0.2, min_volume=0.6.
type VADConfig struct {
	Confidence *float64 `json:"confidence,omitempty"`
	StartSecs  *float64 `json:"start_secs,omitempty"`
	StopSecs   *float64 `json:"stop_secs,omitempty"`
	MinVolume  *float64 `json:"min_volume,omitempty"`
}
```

In the `AI` struct, after `STTType`:
```go
VADConfig *VADConfig `json:"vad_config,omitempty" db:"vad_config,json"`
```

**Step 2: Add FieldVADConfig to field.go**

In `bin-ai-manager/models/ai/field.go`, after `FieldSTTType`:
```go
FieldVADConfig Field = "vad_config"
```

**Step 3: Add VADConfig to webhook.go**

In `WebhookMessage` struct, after `STTType`:
```go
VADConfig *VADConfig `json:"vad_config,omitempty"`
```

In `ConvertWebhookMessage()`, after `STTType: h.STTType,`:
```go
VADConfig: h.VADConfig,
```

**Step 4: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-configurable-vad-config/bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

---

### Task 2: Add AIVADConfig field to AIcall model and snapshot

**Files:**
- Modify: `bin-ai-manager/models/aicall/main.go:22` (add AIVADConfig after AISTTType)
- Modify: `bin-ai-manager/models/aicall/field.go:17` (add FieldAIVADConfig after FieldAISTTType)
- Modify: `bin-ai-manager/models/aicall/webhook.go:23-55` (add AIVADConfig to WebhookMessage + ConvertWebhookMessage)
- Modify: `bin-ai-manager/pkg/aicallhandler/db.go:50` (snapshot a.VADConfig → AIVADConfig)

**Step 1: Add AIVADConfig to AIcall model**

In `bin-ai-manager/models/aicall/main.go`, after `AISTTType`:
```go
AIVADConfig *ai.VADConfig `json:"ai_vad_config,omitempty" db:"ai_vad_config,json"`
```

**Step 2: Add FieldAIVADConfig to field.go**

In `bin-ai-manager/models/aicall/field.go`, after `FieldAISTTType`:
```go
FieldAIVADConfig Field = "ai_vad_config"
```

**Step 3: Add AIVADConfig to webhook.go**

In `WebhookMessage` struct, after `AISTTType`:
```go
AIVADConfig *ai.VADConfig `json:"ai_vad_config,omitempty"`
```

In `ConvertWebhookMessage()`, after `AISTTType: h.AISTTType,`:
```go
AIVADConfig: h.AIVADConfig,
```

**Step 4: Snapshot in db.go**

In `bin-ai-manager/pkg/aicallhandler/db.go:50`, after `AISTTType: c.STTType,`:
```go
AIVADConfig: c.VADConfig,
```

**Step 5: Update existing tests**

In `bin-ai-manager/pkg/aicallhandler/db_test.go`, add `VADConfig` to test AI structs where relevant, and `AIVADConfig` to expected AIcall structs.

In `bin-ai-manager/pkg/aicallhandler/start_test.go`, add `AIVADConfig` to expected AIcall structs.

**Step 6: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-configurable-vad-config/bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

---

### Task 3: Add VADConfig to AI CRUD call chain

**Files:**
- Modify: `bin-ai-manager/pkg/aihandler/main.go:21-51` (add vadConfig param to Create/Update interface)
- Modify: `bin-ai-manager/pkg/aihandler/chatbot.go:15-84` (add vadConfig param to Create/Update implementations)
- Modify: `bin-ai-manager/pkg/aihandler/db.go:15-140` (add vadConfig param to dbCreate/dbUpdate)
- Modify: `bin-ai-manager/pkg/listenhandler/models/request/ais.go:13-51` (add VADConfig to request structs)
- Modify: `bin-ai-manager/pkg/listenhandler/v1_ais.go:92-104,216-229` (pass req.VADConfig to handlers)
- Modify: `bin-ai-manager/cmd/ai-control/main.go:83-101,132-151,166-214,268-322` (add --vad-config flag and pass to handlers)
- Regenerate: `bin-ai-manager/pkg/aihandler/mock_main.go` (via go generate)

**Step 1: Update AIHandler interface**

In `bin-ai-manager/pkg/aihandler/main.go`, add `vadConfig *ai.VADConfig` parameter after `toolNames []tool.ToolName` in both `Create()` and `Update()`:

```go
Create(
    ctx context.Context,
    customerID uuid.UUID,
    name string,
    detail string,
    engineModel ai.EngineModel,
    parameter map[string]any,
    engineKey string,
    initPrompt string,
    ttsType ai.TTSType,
    ttsVoiceID string,
    sttType ai.STTType,
    toolNames []tool.ToolName,
    vadConfig *ai.VADConfig,
) (*ai.AI, error)
```

Same for `Update()`.

**Step 2: Update chatbot.go implementations**

In `bin-ai-manager/pkg/aihandler/chatbot.go`, add `vadConfig *ai.VADConfig` parameter after `toolNames []tool.ToolName` in both `Create()` and `Update()`. Pass it through to `h.dbCreate(...)` and `h.dbUpdate(...)`.

**Step 3: Update db.go (dbCreate and dbUpdate)**

In `dbCreate()`, add `vadConfig *ai.VADConfig` parameter after `toolNames`. In the `c := &ai.AI{...}` struct literal, add:
```go
VADConfig: vadConfig,
```

In `dbUpdate()`, add `vadConfig *ai.VADConfig` parameter after `toolNames`. In the `fields := map[ai.Field]any{...}` map, add:
```go
ai.FieldVADConfig: vadConfig,
```

**Step 4: Update request structs**

In `bin-ai-manager/pkg/listenhandler/models/request/ais.go`, add to both `V1DataAIsPost` and `V1DataAIsIDPut` after `ToolNames`:
```go
VADConfig *ai.VADConfig `json:"vad_config,omitempty"`
```

Add import `"monorepo/bin-ai-manager/models/ai"` if not already present.

**Step 5: Update listenhandler callers**

In `bin-ai-manager/pkg/listenhandler/v1_ais.go`:

In `processV1AIsPost()` (line 92-104), add `req.VADConfig,` after `req.ToolNames,` in the `h.aiHandler.Create(...)` call.

In `processV1AIsIDPut()` (line 216-229), add `req.VADConfig,` after `req.ToolNames,` in the `h.aiHandler.Update(...)` call.

**Step 6: Update CLI tool**

In `bin-ai-manager/cmd/ai-control/main.go`:

In `cmdCreate()`, add flag:
```go
flags.String("vad-config", "", "VAD configuration (JSON string, e.g., '{\"stop_secs\": 0.5}')")
```

In `cmdUpdate()`, add the same flag.

In `runCreate()`, after parsing `sttType`, add:
```go
var vadConfig *ai.VADConfig
if vadConfigStr := viper.GetString("vad-config"); vadConfigStr != "" {
    vadConfig = &ai.VADConfig{}
    if err := json.Unmarshal([]byte(vadConfigStr), vadConfig); err != nil {
        return fmt.Errorf("invalid vad-config JSON: %w", err)
    }
}
```

Then pass `vadConfig` as the new parameter in `handler.Create(...)` after `nil, // toolNames`.

Same changes in `runUpdate()`.

**Step 7: Regenerate mocks and run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-configurable-vad-config/bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

---

### Task 4: Update pipecat-manager Go side

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/run.go:18-21` (remove defaultVADStopSecs, add VADConfig to resolvedAIData)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/pythonrunner.go:38-53,61-74,81-107` (change interface + impl to use *ai.VADConfig)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:83-96` (read aicall.AIVADConfig, pass to pythonRunner)
- Regenerate: `bin-pipecat-manager/pkg/pipecatcallhandler/mock_pythonrunner.go` (via go generate)

**Step 1: Update run.go**

Remove `defaultVADStopSecs = 0.5` from the const block (line 21). Add `VADConfig` field to `resolvedAIData` struct (after `STTType`):
```go
VADConfig *amai.VADConfig `json:"vad_config,omitempty"`
```

Add import for `amai "monorepo/bin-ai-manager/models/ai"` if not already present.

In `resolveTeamForPython()`, when building `resolvedAIData` (line 172), add after `STTType`:
```go
VADConfig: ai.VADConfig,
```

**Step 2: Update pythonrunner.go**

Change the `PythonRunner` interface `Start` signature — replace `vadStopSecs float64` with `vadConfig *amai.VADConfig`:
```go
Start(
    ctx context.Context,
    pipecatcallID uuid.UUID,
    llmType string,
    llmKey string,
    llmMessages []map[string]any,
    sttType string,
    sttLanguage string,
    ttsType string,
    ttsLanguage string,
    ttsVoiceID string,
    tools []aitool.Tool,
    resolvedTeam *resolvedTeamData,
    vadConfig *amai.VADConfig,
) error
```

Add import `amai "monorepo/bin-ai-manager/models/ai"`.

Update the `Start` implementation signature to match. In the request body struct, replace `VADStopSecs float64 \`json:"vad_stop_secs,omitempty"\`` with:
```go
VADConfig *amai.VADConfig `json:"vad_config,omitempty"`
```

Update the assignment from `VADStopSecs: vadStopSecs,` to:
```go
VADConfig: vadConfig,
```

**Step 3: Update runner.go**

In `runnerStartScript()`, the AIcall is already fetched at line 54. After resolving tools/team, read the VAD config from the AIcall and pass it. Replace line 96 (`defaultVADStopSecs`) with:

```go
aicall.AIVADConfig,
```

For the non-AICall branch (line 78-80), pass `nil` for vadConfig.

The full `pythonRunner.Start` call becomes:
```go
if errStart := h.pythonRunner.Start(
    se.Ctx,
    pc.ID,
    string(pc.LLMType),
    string(se.LLMKey),
    pc.LLMMessages,
    string(pc.STTType),
    string(pc.STTLanguage),
    string(pc.TTSType),
    string(pc.TTSLanguage),
    pc.TTSVoiceID,
    tools,
    resolvedTeam,
    aicall.AIVADConfig,
); errStart != nil {
```

Note: The `aicall` variable is already fetched at line 54. For the non-AICall path, we need a `var vadConfig *amai.VADConfig` that stays nil.

**Step 4: Regenerate mocks and run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-configurable-vad-config/bin-pipecat-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

---

### Task 5: Update Python pipecat scripts

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/main.py:77-89` (replace vad_stop_secs with vad_config model)
- Modify: `bin-pipecat-manager/scripts/pipecat/run.py:35-36,57-69,101-112,133,473-505,584` (build VADParams from config dict)

**Step 1: Update main.py**

Replace `vad_stop_secs: float = 0.5` on `PipelineRequest` (line 89) with:
```python
vad_config: Optional[dict] = None
```

Update the `init_pipeline` call (lines 120-132) — replace `vad_stop_secs=req.vad_stop_secs` with:
```python
vad_config=req.vad_config,
```

**Step 2: Update run.py**

Add a helper function near the top (after imports):
```python
def build_vad_params(vad_config: dict | None) -> VADParams:
    """Build VADParams from config dict. None/empty = Pipecat defaults."""
    if not vad_config:
        return VADParams()

    kwargs = {}
    if vad_config.get("confidence") is not None:
        kwargs["confidence"] = vad_config["confidence"]
    if vad_config.get("start_secs") is not None:
        kwargs["start_secs"] = vad_config["start_secs"]
    if vad_config.get("stop_secs") is not None:
        kwargs["stop_secs"] = vad_config["stop_secs"]
    if vad_config.get("min_volume") is not None:
        kwargs["min_volume"] = vad_config["min_volume"]

    return VADParams(**kwargs)
```

Note: Uses `is not None` instead of truthiness checks to correctly handle explicit `0.0` values.

In `init_pipeline()` (line 57-70): replace `vad_stop_secs: float = 0.5` param with `vad_config: dict = None`. Pass `vad_config=vad_config` to both `init_team_pipeline` and `init_single_ai_pipeline`.

In `init_single_ai_pipeline()` (line 101-112): replace `vad_stop_secs: float = 0.5` param with `vad_config: dict = None`. Replace line 133:
```python
# Old: vad_analyzer = SileroVADAnalyzer(params=VADParams(stop_secs=max(vad_stop_secs, 0.3)))
vad_analyzer = SileroVADAnalyzer(params=build_vad_params(vad_config))
```

In `init_team_pipeline()` (line 499-506): replace `vad_stop_secs: float = 0.5` param with `vad_config: dict = None`. Replace line 584:
```python
# Old: vad_analyzer = SileroVADAnalyzer(params=VADParams(stop_secs=max(vad_stop_secs, 0.3)))
vad_analyzer = SileroVADAnalyzer(params=build_vad_params(vad_config))
```

**Step 3: Update conftest.py mocks if needed**

The mock for `pipecat.audio.vad.vad_analyzer` already covers `VADParams` — no changes needed.

---

### Task 6: Create database migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<hash>_ai_ais_aicalls_add_column_vad_config.py`

**Step 1: Create migration file**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-configurable-vad-config/bin-dbscheme-manager/bin-manager
alembic -c alembic.ini revision -m "ai_ais_aicalls_add_column_vad_config"
```

**Step 2: Edit the migration file**

```python
def upgrade():
    op.execute("""ALTER TABLE ai_ais ADD vad_config JSON DEFAULT NULL AFTER stt_type;""")
    op.execute("""ALTER TABLE ai_aicalls ADD ai_vad_config JSON DEFAULT NULL AFTER ai_stt_type;""")


def downgrade():
    op.execute("""ALTER TABLE ai_ais DROP COLUMN vad_config;""")
    op.execute("""ALTER TABLE ai_aicalls DROP COLUMN ai_vad_config;""")
```

---

### Task 7: Update OpenAPI schema

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (add AIManagerVADConfig schema, add fields to AI and AIcall schemas)
- Modify: `bin-openapi-manager/openapi/paths/ais/main.yaml` (add vad_config to POST request body)
- Modify: `bin-openapi-manager/openapi/paths/ais/id.yaml` (add vad_config to PUT request body)

**Step 1: Read and follow bin-openapi-manager/CLAUDE.md rules before editing**

**Step 2: Add AIManagerVADConfig schema**

Add a new schema definition (near other AIManager schemas) in `openapi.yaml`:
```yaml
    AIManagerVADConfig:
      type: object
      description: Voice Activity Detection configuration. Omitted fields use Pipecat defaults (confidence=0.7, start_secs=0.2, stop_secs=0.2, min_volume=0.6).
      properties:
        confidence:
          type: number
          format: double
          description: Minimum confidence threshold to detect voice (0.0-1.0).
          example: 0.7
        start_secs:
          type: number
          format: double
          description: Duration of continuous speech needed to confirm speaking started.
          example: 0.2
        stop_secs:
          type: number
          format: double
          description: Duration of silence needed to confirm speaking stopped.
          example: 0.5
        min_volume:
          type: number
          format: double
          description: Minimum audio volume for voice detection (0.0-1.0).
          example: 0.6
```

**Step 3: Add vad_config to AIManagerAI schema**

After `stt_type` in the AIManagerAI properties:
```yaml
        vad_config:
          $ref: '#/components/schemas/AIManagerVADConfig'
          description: Voice Activity Detection configuration for this AI agent.
```

**Step 4: Add ai_vad_config to AIManagerAIcall schema**

After `ai_stt_type` in the AIManagerAIcall properties:
```yaml
        ai_vad_config:
          $ref: '#/components/schemas/AIManagerVADConfig'
          description: Snapshotted VAD configuration from the AI agent at call creation time.
```

**Step 5: Add vad_config to POST request body (paths/ais/main.yaml)**

In the POST `requestBody` properties, after `stt_type`:
```yaml
            vad_config:
              $ref: '#/components/schemas/AIManagerVADConfig'
              description: Voice Activity Detection configuration. All fields are optional; omitted fields use Pipecat defaults.
```

**Step 6: Add vad_config to PUT request body (paths/ais/id.yaml)**

In the PUT `requestBody` properties, after `stt_type`:
```yaml
            vad_config:
              $ref: '#/components/schemas/AIManagerVADConfig'
              description: Voice Activity Detection configuration. All fields are optional; omitted fields use Pipecat defaults.
```

**Step 7: Regenerate OpenAPI types and API server**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-configurable-vad-config/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-configurable-vad-config/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

---

### Task 8: Update RST documentation

**Files:**
- Modify: `bin-api-manager/docsdev/source/ai_struct_ai.rst` (add vad_config field)

**Step 1: Add vad_config to AI struct documentation**

In `bin-api-manager/docsdev/source/ai_struct_ai.rst`, after the `stt_type` field line in the code block, add:
```
        "vad_config": {
            "confidence": <number>,
            "start_secs": <number>,
            "stop_secs": <number>,
            "min_volume": <number>
        },
```

After the `stt_type` field description, add:
```
* ``vad_config`` (Object, Optional): Voice Activity Detection configuration. All fields are optional — omitted fields use Pipecat defaults. See :ref:`VAD Config <ai-struct-ai-vad_config>`.
```

Add the example value in the example block after `stt_type`:
```
        "vad_config": {
            "stop_secs": 0.5
        },
```

Add a new section after the STT Type section:

```rst
.. _ai-struct-ai-vad_config:

VAD Config
----------
Voice Activity Detection configuration for tuning speech detection sensitivity and timing.

All fields are optional. Omitted fields use Pipecat's native defaults.

================ ======== ====================================
Field            Default  Description
================ ======== ====================================
confidence       0.7      Minimum confidence threshold (0.0-1.0) to detect voice.
start_secs       0.2      Duration (seconds) of continuous speech needed to confirm speaking started.
stop_secs        0.2      Duration (seconds) of silence needed to confirm speaking stopped.
min_volume       0.6      Minimum audio volume (0.0-1.0) for voice detection.
================ ======== ====================================

.. note:: **AI Implementation Hint**

   When ``vad_config`` is ``null`` or omitted, Pipecat's native defaults apply (confidence=0.7, start_secs=0.2, stop_secs=0.2, min_volume=0.6). To keep the AI responsive but avoid cutting off speech mid-sentence, increase ``stop_secs`` (e.g., 0.5). To make the AI more patient before responding, increase both ``stop_secs`` and ``start_secs``.
```

**Step 2: Rebuild HTML docs**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-configurable-vad-config/bin-api-manager/docsdev
rm -rf build && python3 -m sphinx -M html source build
```

**Step 3: Stage built docs**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-configurable-vad-config
git add -f bin-api-manager/docsdev/build/
```

---

### Task 9: Final cross-service verification and commit

**Step 1: Run verification for all changed services**

```bash
# ai-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-configurable-vad-config/bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# pipecat-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-configurable-vad-config/bin-pipecat-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# openapi-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-configurable-vad-config/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# api-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-configurable-vad-config/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-configurable-vad-config
git add -A
git status  # verify no vendor/ or unwanted files
```

Commit message:
```
NOJIRA-Add-configurable-vad-config

Add configurable VAD parameters per-AI agent, replacing the hardcoded
stop_secs=0.5 override with Pipecat's native defaults when no config is set.

- bin-ai-manager: Add VADConfig struct with pointer fields to AI model
- bin-ai-manager: Add AIVADConfig snapshot field to AIcall model
- bin-ai-manager: Snapshot VADConfig at AIcall creation in db.go
- bin-ai-manager: Add vadConfig parameter to AI Create/Update CRUD chain
- bin-ai-manager: Add vad_config to request structs and CLI tool
- bin-pipecat-manager: Remove defaultVADStopSecs constant
- bin-pipecat-manager: Pass VADConfig from AIcall to Python runner
- bin-pipecat-manager: Build VADParams from config dict in Python
- bin-dbscheme-manager: Add vad_config columns to ai_ais and ai_aicalls
- bin-openapi-manager: Add AIManagerVADConfig schema and fields
- bin-openapi-manager: Add vad_config to POST/PUT request bodies
- bin-api-manager: Update RST documentation for vad_config field
- docs: Add design document for configurable VAD parameters
```
