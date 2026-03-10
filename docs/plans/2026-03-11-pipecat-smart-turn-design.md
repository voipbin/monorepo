# Pipecat Smart Turn Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `smart_turn_enabled` boolean to AI model, flowing through AIcall → pipecat-manager Go → Python, enabling `LocalSmartTurnAnalyzerV3` on the input transport with forced `stop_secs=0.2`.

**Architecture:** Boolean field added to AI/AIcall models in bin-ai-manager, passed through existing config flow in bin-pipecat-manager Go (runner.go → pythonrunner.go → HTTP POST) to Python (main.py → run.py), where it creates a `LocalSmartTurnAnalyzerV3` on the input transport. Team pipelines carry per-member smart turn settings via `resolvedAIData`.

**Tech Stack:** Go (bin-ai-manager, bin-pipecat-manager), Python/Pipecat (scripts/pipecat), MySQL/Alembic, OpenAPI YAML

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-pipecat-smart-turn-support`

---

### Task 1: AI Model — Add SmartTurnEnabled field

**Files:**
- Modify: `bin-ai-manager/models/ai/main.go:60` (after VADConfig)
- Modify: `bin-ai-manager/models/ai/webhook.go:29` (after VADConfig in WebhookMessage) and line 57 (in ConvertWebhookMessage)
- Modify: `bin-ai-manager/models/ai/field.go:24` (after FieldVADConfig)

**Step 1: Add field to AI struct**

In `bin-ai-manager/models/ai/main.go`, after line 60 (`VADConfig`), add:

```go
	SmartTurnEnabled bool `json:"smart_turn_enabled,omitempty" db:"smart_turn_enabled"`
```

**Step 2: Add field to WebhookMessage struct**

In `bin-ai-manager/models/ai/webhook.go`, after line 29 (`VADConfig`), add:

```go
	SmartTurnEnabled bool `json:"smart_turn_enabled,omitempty"`
```

In `ConvertWebhookMessage()`, after line 57 (`VADConfig: h.VADConfig,`), add:

```go
		SmartTurnEnabled: h.SmartTurnEnabled,
```

**Step 3: Add Field constant**

In `bin-ai-manager/models/ai/field.go`, after line 24 (`FieldVADConfig`), add:

```go
	FieldSmartTurnEnabled Field = "smart_turn_enabled"
```

**Step 4: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-pipecat-smart-turn-support/bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All tests pass, lint clean.

---

### Task 2: AIcall Model — Add AISmartTurnEnabled field

**Files:**
- Modify: `bin-ai-manager/models/aicall/main.go:23` (after AIVADConfig)
- Modify: `bin-ai-manager/models/aicall/webhook.go:24` (after AIVADConfig in WebhookMessage) and line 57 (in ConvertWebhookMessage)
- Modify: `bin-ai-manager/models/aicall/field.go:18` (after FieldAIVADConfig)

**Step 1: Add field to AIcall struct**

In `bin-ai-manager/models/aicall/main.go`, after line 23 (`AIVADConfig`), add:

```go
	AISmartTurnEnabled bool `json:"ai_smart_turn_enabled,omitempty" db:"ai_smart_turn_enabled"`
```

**Step 2: Add field to WebhookMessage struct**

In `bin-ai-manager/models/aicall/webhook.go`, after line 24 (`AIVADConfig`), add:

```go
	AISmartTurnEnabled bool `json:"ai_smart_turn_enabled,omitempty"`
```

In `ConvertWebhookMessage()`, after line 57 (`AIVADConfig: h.AIVADConfig,`), add:

```go
		AISmartTurnEnabled: h.AISmartTurnEnabled,
```

**Step 3: Add Field constant**

In `bin-ai-manager/models/aicall/field.go`, after line 18 (`FieldAIVADConfig`), add:

```go
	FieldAISmartTurnEnabled Field = "ai_smart_turn_enabled"
```

**Step 4: Copy field in aicallhandler.Create()**

In `bin-ai-manager/pkg/aicallhandler/db.go`, after line 51 (`AIVADConfig: c.VADConfig,`), add:

```go
		AISmartTurnEnabled: c.SmartTurnEnabled,
```

**Step 5: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-pipecat-smart-turn-support/bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All tests pass, lint clean.

**Step 6: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-pipecat-smart-turn-support
git add bin-ai-manager/models/ai/main.go bin-ai-manager/models/ai/webhook.go bin-ai-manager/models/ai/field.go
git add bin-ai-manager/models/aicall/main.go bin-ai-manager/models/aicall/webhook.go bin-ai-manager/models/aicall/field.go
git add bin-ai-manager/pkg/aicallhandler/db.go
```

---

### Task 3: Database Migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/e2a3b4c5d6f7_ai_ais_aicalls_add_column_smart_turn_enabled.py`

**Step 1: Create migration file**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-pipecat-smart-turn-support/bin-dbscheme-manager/bin-manager
alembic -c alembic.ini revision -m "ai_ais_aicalls_add_column_smart_turn_enabled"
```

**Step 2: Edit migration file**

Replace the generated `upgrade()` and `downgrade()` with:

```python
def upgrade():
    op.execute("""ALTER TABLE ai_ais ADD smart_turn_enabled TINYINT(1) NOT NULL DEFAULT 0 AFTER vad_config;""")
    op.execute("""ALTER TABLE ai_aicalls ADD ai_smart_turn_enabled TINYINT(1) NOT NULL DEFAULT 0 AFTER ai_vad_config;""")


def downgrade():
    op.execute("""ALTER TABLE ai_ais DROP COLUMN smart_turn_enabled;""")
    op.execute("""ALTER TABLE ai_aicalls DROP COLUMN ai_smart_turn_enabled;""")
```

**IMPORTANT: Do NOT run `alembic upgrade`. Just create and commit the migration file.**

**Step 3: Stage migration file**

```bash
git add bin-dbscheme-manager/bin-manager/main/versions/*smart_turn_enabled*.py
```

---

### Task 4: OpenAPI Schema Update

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (AIManagerAI schema at ~line 1814, AIManagerAIcall schema at ~line 2130)

**Step 1: Add to AIManagerAI schema**

After the `vad_config` property (~line 1815), add:

```yaml
        smart_turn_enabled:
          type: boolean
          description: Enable smart turn detection using Pipecat's LocalSmartTurnAnalyzerV3. When enabled, forces VAD stop_secs to 0.2 for optimal turn-taking.
          example: false
```

**Step 2: Add to AIManagerAIcall schema**

After the `ai_vad_config` property (~line 2131), add:

```yaml
        ai_smart_turn_enabled:
          type: boolean
          description: Smart turn detection setting frozen from the AI configuration at call start.
          example: false
```

**Step 3: Run verification for bin-openapi-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-pipecat-smart-turn-support/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Regenerate bin-api-manager from updated OpenAPI**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-pipecat-smart-turn-support/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
git add bin-openapi-manager/openapi/openapi.yaml
# Also add any regenerated files in bin-openapi-manager and bin-api-manager
```

---

### Task 5: Pipecat Manager Go — Pass smart_turn_enabled to Python

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/run.go:42-51` (resolvedAIData struct)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/run.go:172-180` (resolveTeamForPython — resolvedAIData construction)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/pythonrunner.go:39-54` (PythonRunner interface)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/pythonrunner.go:62-108` (Start implementation)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:41-106` (runnerStartScript)

**Step 1: Add SmartTurnEnabled to resolvedAIData**

In `run.go`, after line 50 (`VADConfig`), add:

```go
	SmartTurnEnabled bool            `json:"smart_turn_enabled,omitempty"`
```

**Step 2: Set SmartTurnEnabled in resolveTeamForPython**

In `run.go`, inside the `resolvedAIData{...}` literal (~line 172-180), after line 180 (`VADConfig: ai.VADConfig,`), add:

```go
				SmartTurnEnabled: ai.SmartTurnEnabled,
```

**Step 3: Add smartTurnEnabled parameter to PythonRunner interface**

In `pythonrunner.go`, after line 53 (`vadConfig *amai.VADConfig,`), add:

```go
		smartTurnEnabled bool,
```

**Step 4: Add smartTurnEnabled parameter to Start implementation**

In `pythonrunner.go`, after line 75 (`vadConfig *amai.VADConfig,`), add:

```go
	smartTurnEnabled bool,
```

**Step 5: Add SmartTurnEnabled to request body struct and assignment**

In `pythonrunner.go`, after line 94 (`VADConfig *amai.VADConfig \`json:"vad_config,omitempty"\``), add:

```go
		SmartTurnEnabled bool              `json:"smart_turn_enabled,omitempty"`
```

After line 107 (`VADConfig: vadConfig,`), add:

```go
		SmartTurnEnabled: smartTurnEnabled,
```

**Step 6: Extract and pass smartTurnEnabled in runnerStartScript**

In `runner.go`, after line 53 (`var vadConfig *amai.VADConfig`), add:

```go
	var smartTurnEnabled bool
```

After line 61 (`vadConfig = aicall.AIVADConfig`), add:

```go
		smartTurnEnabled = aicall.AISmartTurnEnabled
```

After line 100 (`vadConfig,`), add:

```go
		smartTurnEnabled,
```

**Step 7: Regenerate mocks and run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-pipecat-smart-turn-support/bin-pipecat-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: Mock regenerated for PythonRunner interface. Tests may need updating for new parameter.

**Step 8: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/
```

---

### Task 6: Pipecat Manager Python — Enable LocalSmartTurnAnalyzerV3

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/main.py` (PipelineRequest, ResolvedAI, init_pipeline call)
- Modify: `bin-pipecat-manager/scripts/pipecat/run.py` (build_vad_params, init_pipeline, init_single_ai_pipeline, init_team_pipeline, create_websocket_transport)

**Step 1: Add import for LocalSmartTurnAnalyzerV3**

In `run.py`, after line 36 (`from pipecat.audio.vad.vad_analyzer import VADParams`), add:

```python
from pipecat.audio.turn.smart_turn.local_smart_turn_v3 import LocalSmartTurnAnalyzerV3
```

**Step 2: Update build_vad_params to accept smart_turn_enabled**

Replace `build_vad_params` function (lines 57-72) with:

```python
def build_vad_params(vad_config: dict | None, smart_turn_enabled: bool = False) -> VADParams:
    """Build VADParams from config dict. None/empty = Pipecat defaults.
    When smart_turn_enabled is True, forces stop_secs=0.2 for optimal turn detection.
    """
    if not vad_config:
        vad_config = {}

    kwargs = {}
    if vad_config.get("confidence") is not None:
        kwargs["confidence"] = vad_config["confidence"]
    if vad_config.get("start_secs") is not None:
        kwargs["start_secs"] = vad_config["start_secs"]

    # Smart turn requires stop_secs=0.2 (matches training data)
    if smart_turn_enabled:
        kwargs["stop_secs"] = 0.2
    elif vad_config.get("stop_secs") is not None:
        kwargs["stop_secs"] = vad_config["stop_secs"]

    if vad_config.get("min_volume") is not None:
        kwargs["min_volume"] = vad_config["min_volume"]

    return VADParams(**kwargs)
```

**Step 3: Add turn_analyzer parameter to create_websocket_transport**

Replace `create_websocket_transport` function (lines 489-507) with:

```python
def create_websocket_transport(direction: str, id: str, vad_analyzer=None, turn_analyzer=None):
    uri = f"{common.PIPECATCALL_WS_URL}/{id}/ws?direction={direction}"
    logger.info(f"Establishing WebSocket connection to URI: {uri}")

    params = WebsocketClientParams(
        serializer=ProtobufFrameSerializer(),
        audio_in_enabled=True,
        audio_out_enabled=True,
        add_wav_header=False,
        vad_analyzer=vad_analyzer,
        turn_analyzer=turn_analyzer,
        session_timeout=common.PIPELINE_SESSION_TIMEOUT,
    )

    if direction == "output":
        transport = UnpacedWebsocketClientTransport(uri=uri, params=params)
    else:
        transport = WebsocketClientTransport(uri=uri, params=params)

    return transport
```

**Step 4: Add smart_turn_enabled to init_pipeline signature**

In `run.py`, update `init_pipeline` (line 75-88). After `vad_config: dict = None,` (line 87), add:

```python
    smart_turn_enabled: bool = False,
```

Pass it to both sub-functions:

For team pipeline call (~line 91-97), add `smart_turn_enabled=smart_turn_enabled`:

```python
        ctx = await init_team_pipeline(
            id, resolved_team,
            stt_language=stt_language,
            tts_language=tts_language,
            llm_messages=llm_messages,
            vad_config=vad_config,
            smart_turn_enabled=smart_turn_enabled,
        )
```

For single AI pipeline call (~line 101-106), add `smart_turn_enabled=smart_turn_enabled`:

```python
        ctx = await init_single_ai_pipeline(
            id, llm_type, llm_key, llm_messages,
            stt_type, stt_language, tts_type, tts_language,
            tts_voice_id, tools_data,
            vad_config=vad_config,
            smart_turn_enabled=smart_turn_enabled,
        )
```

**Step 5: Add smart_turn_enabled to init_single_ai_pipeline**

After `vad_config: dict = None,` (line 130), add:

```python
    smart_turn_enabled: bool = False,
```

Update the `init_stt_and_input_ws` inner function (~line 148-158). Replace the VAD/transport creation:

```python
            stt_service = create_stt_service(stt_type, language=stt_language)
            vad_analyzer = SileroVADAnalyzer(params=build_vad_params(vad_config, smart_turn_enabled=smart_turn_enabled))
            turn_analyzer = LocalSmartTurnAnalyzerV3() if smart_turn_enabled else None
            transport = create_websocket_transport("input", id, vad_analyzer=vad_analyzer, turn_analyzer=turn_analyzer)
```

**Step 6: Add smart_turn_enabled to init_team_pipeline**

After `vad_config: dict = None,` (line 516), add:

```python
    smart_turn_enabled: bool = False,
```

Update the transport creation (~line 594-596). Replace:

```python
        vad_analyzer = SileroVADAnalyzer(params=build_vad_params(vad_config, smart_turn_enabled=smart_turn_enabled))
        turn_analyzer = LocalSmartTurnAnalyzerV3() if smart_turn_enabled else None
        transport_input = create_websocket_transport("input", id, vad_analyzer=vad_analyzer, turn_analyzer=turn_analyzer)
```

**Step 7: Update main.py — PipelineRequest and ResolvedAI**

In `main.py`, after line 47 (`vad_config: Optional[dict] = None`) in `ResolvedAI`, add:

```python
    smart_turn_enabled: Optional[bool] = False
```

After line 90 (`vad_config: Optional[dict] = None`) in `PipelineRequest`, add:

```python
    smart_turn_enabled: Optional[bool] = False
```

In the `init_pipeline` call (~line 121-133), after `vad_config=req.vad_config,` (line 133), add:

```python
            smart_turn_enabled=req.smart_turn_enabled,
```

**Step 8: Run Python tests**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-pipecat-smart-turn-support/bin-pipecat-manager/scripts/pipecat
python -m pytest -v
```

Expected: Existing tests pass. Some may need updating for new parameters.

**Step 9: Commit**

```bash
git add bin-pipecat-manager/scripts/pipecat/main.py bin-pipecat-manager/scripts/pipecat/run.py
```

---

### Task 7: RST Documentation Update

**Files:**
- Modify: `bin-api-manager/docsdev/source/ai_struct_ai.rst`

**Step 1: Add smart_turn_enabled to JSON struct**

After line 30 (`"min_volume": <number>`) and before line 31 (`},`), the vad_config block ends. After the vad_config closing brace (line 31: `        },`), add:

```rst
        "smart_turn_enabled": <boolean>,
```

**Step 2: Add field description**

After line 48 (vad_config description), add:

```rst
* ``smart_turn_enabled`` (Boolean, Optional): Enable smart turn detection using Pipecat's LocalSmartTurnAnalyzerV3 for more natural turn-taking. When ``true``, the VAD ``stop_secs`` parameter is automatically forced to ``0.2`` regardless of ``vad_config`` settings. Defaults to ``false``.
```

**Step 3: Add to example**

After line 81 (`"stop_secs": 0.5`), add:

```rst
        "smart_turn_enabled": true,
```

**Step 4: Add AI Implementation Hint**

After the VAD Config section (~line 202), add a new section:

```rst
.. _ai-struct-ai-smart_turn:

Smart Turn
----------
When ``smart_turn_enabled`` is ``true``, the Pipecat pipeline uses ``LocalSmartTurnAnalyzerV3`` — a local ONNX model that analyzes speech and transcription context to detect when the user has truly finished their turn, rather than pausing mid-sentence. This results in more natural conversations with fewer premature interruptions.

.. note:: **AI Implementation Hint**

   Smart Turn detection requires VAD ``stop_secs=0.2``. When ``smart_turn_enabled`` is ``true``, any ``stop_secs`` value in ``vad_config`` is silently overridden to ``0.2``. This value matches the model's training data and allows Smart Turn to dynamically adjust timing.
```

**Step 5: Rebuild HTML**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-pipecat-smart-turn-support/bin-api-manager/docsdev
rm -rf build && python3 -m sphinx -M html source build
```

**Step 6: Commit**

```bash
git add bin-api-manager/docsdev/source/ai_struct_ai.rst
git add -f bin-api-manager/docsdev/build/
```

---

### Task 8: Final Verification and Commit

**Step 1: Run full verification for all changed services**

```bash
# bin-ai-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-pipecat-smart-turn-support/bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-pipecat-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-pipecat-smart-turn-support/bin-pipecat-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-openapi-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-pipecat-smart-turn-support/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-api-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-pipecat-smart-turn-support/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# Python tests
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-pipecat-smart-turn-support/bin-pipecat-manager/scripts/pipecat
python -m pytest -v
```

**Step 2: Create commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-pipecat-smart-turn-support
git add -A
# Verify no vendor/ files are staged
git diff --cached --name-only | grep vendor/ && echo "WARNING: vendor files staged!" || echo "OK: no vendor files"
git commit
```

Commit message:
```
NOJIRA-Add-pipecat-smart-turn-support

Add smart_turn_enabled boolean to AI model for enabling Pipecat's
LocalSmartTurnAnalyzerV3 turn detection.

- bin-ai-manager: Add SmartTurnEnabled to AI model, webhook, and field
- bin-ai-manager: Add AISmartTurnEnabled to AIcall model, webhook, and field
- bin-ai-manager: Copy SmartTurnEnabled from AI to AIcall on creation
- bin-dbscheme-manager: Add migration for smart_turn_enabled columns
- bin-openapi-manager: Add smart_turn_enabled to AI and AIcall schemas
- bin-pipecat-manager: Pass smart_turn_enabled through Go to Python runner
- bin-pipecat-manager: Add SmartTurnEnabled to resolvedAIData for team members
- bin-pipecat-manager: Enable LocalSmartTurnAnalyzerV3 when smart_turn_enabled
- bin-pipecat-manager: Force stop_secs=0.2 when smart turn is active
- bin-api-manager: Update RST docs with smart_turn_enabled field
```

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-Add-pipecat-smart-turn-support
```
