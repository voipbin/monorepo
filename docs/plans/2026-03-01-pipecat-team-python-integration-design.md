# Design: Pipecat Python Team Integration

**Date:** 2026-03-01
**Branch:** NOJIRA-add-pipecat-team-python-integration
**Status:** Draft
**Depends on:** NOJIRA-add-pipecat-flows-team-support (Go-side team infrastructure)

## Problem Statement

VoIPbin's team-based AI calls are fully configured on the Go side (ai-manager: Team model, validation, AIcall support; pipecat-manager: team resolution for LLM key extraction). However, the Python pipecat runner has no team awareness — it runs a single AI with fixed LLM/TTS/STT for the entire session.

We need the Python runner to support multi-member teams where each member can have different prompts, tools, LLM providers, TTS voices/providers, and STT providers. When the LLM calls a transition function (e.g., `transfer_to_billing`), the Python side should switch the active member's configuration — prompts, tools, and service providers — seamlessly within the same call session.

## Approach

Use pipecat-flows `FlowManager` for conversation flow management (prompt/tool switching, node transitions). Wrap LLM, TTS, and STT behind **routing services** that switch delegates per-member on transitions. Resolve team data in pipecat-manager (Go side) and pass to Python via the existing HTTP POST `/run` endpoint.

### Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Team resolution location | pipecat-manager (Go) | Avoids changing bin-common-handler RPC interface (30+ service blast radius) |
| Flow library | pipecat-flows | Purpose-built for pipecat conversation flows; handles prompt/tool/function management |
| Service switching | Routing proxy services | No audio gaps; single pipeline for entire session; extensible |
| Transition handling | Python-local only | No Go notification; simplest approach; Go tracking deferred |
| LLM switching | Per-member | Different providers/models per node (all currently OpenAI-compatible) |
| TTS switching | Per-member | Different providers and voices per node |
| STT switching | Per-member | Different providers per node; partial transcription loss acceptable |
| Context strategy | APPEND (default) | Full context preserved across transitions; safest default |

## Data Flow

### End-to-End Sequence

```
1. Flow-manager triggers ai_talk action with assistance_type="team"

2. ai-manager creates AIcall with:
   - AssistanceType = "team"
   - AssistanceID = team UUID
   - Snapshots start member's AI config (existing pattern)

3. ai-manager calls PipecatV1PipecatcallStart (existing RPC, no changes)
   - Sends start member's LLM/TTS/STT config as today

4. pipecat-manager receives start request:
   a. Creates pipecatcall DB record (existing)
   b. In runnerStartScript(), detects AssistanceTypeTeam
   c. Calls resolveTeamForPython() — NEW:
      - Fetches team via AIV1TeamGet
      - For each member: fetches AI via AIV1AIGet (includes EngineKey)
      - For each member: resolves tools via toolHandler.GetByNames(ai.ToolNames)
      - Builds resolvedTeamData struct with keys included
   d. Passes resolvedTeamData to pythonRunner.Start()

5. Python runner receives HTTP POST /run with resolved_team field:
   a. Detects resolved_team is present → team pipeline path
   b. Creates routing services (LLM, TTS, STT) wrapping per-member instances
   c. Builds pipeline with routing services instead of single services
   d. Creates FlowManager with routing LLM + context aggregator
   e. Initializes FlowManager with start member's NodeConfig
   f. Pipeline runs

6. During call, LLM calls "transfer_to_billing":
   a. FlowsFunctionSchema handler fires
   b. Handler switches routing services to billing member
   c. Handler returns billing member's NodeConfig
   d. FlowManager updates context (new prompt, new tools)
   e. LLM generates response with billing member's configuration
```

## Go Changes (bin-pipecat-manager only)

### resolveTeamForPython (run.go)

New function that builds the complete team data for Python, including engine keys:

```go
// resolvedTeamData is the Python-facing team struct that includes EngineKey per member.
// This is safe because Go↔Python is localhost communication within the same pod.
type resolvedTeamData struct {
    ID            uuid.UUID              `json:"id"`
    StartMemberID uuid.UUID              `json:"start_member_id"`
    Members       []resolvedMemberData   `json:"members"`
}

type resolvedMemberData struct {
    ID          uuid.UUID          `json:"id"`
    Name        string             `json:"name"`
    AI          resolvedAIData     `json:"ai"`
    Tools       []aitool.Tool      `json:"tools"`
    Transitions []amteam.Transition `json:"transitions"`
}

type resolvedAIData struct {
    EngineModel string         `json:"engine_model"`
    EngineKey   string         `json:"engine_key"`
    InitPrompt  string         `json:"init_prompt"`
    Parameter   map[string]any `json:"parameter,omitempty"`
    TTSType     string         `json:"tts_type"`
    TTSVoiceID  string         `json:"tts_voice_id"`
    STTType     string         `json:"stt_type"`
    STTLanguage string         `json:"stt_language,omitempty"`
    ToolNames   []string       `json:"tool_names,omitempty"`
}

func (h *pipecatcallHandler) resolveTeamForPython(
    ctx context.Context, c *amaicall.AIcall,
) (*resolvedTeamData, error) {
    if c.AssistanceType != amaicall.AssistanceTypeTeam {
        return nil, nil
    }

    team, err := h.requestHandler.AIV1TeamGet(ctx, c.AssistanceID)
    if err != nil {
        return nil, fmt.Errorf("could not get team: %w", err)
    }

    resolved := &resolvedTeamData{
        ID:            team.ID,
        StartMemberID: team.StartMemberID,
    }

    for _, m := range team.Members {
        ai, err := h.requestHandler.AIV1AIGet(ctx, m.AIID)
        if err != nil {
            return nil, fmt.Errorf("could not get AI for member %s: %w", m.ID, err)
        }

        tools := h.toolHandler.GetByNames(ai.ToolNames)

        resolved.Members = append(resolved.Members, resolvedMemberData{
            ID:   m.ID,
            Name: m.Name,
            AI: resolvedAIData{
                EngineModel: string(ai.EngineModel),
                EngineKey:   ai.EngineKey,
                InitPrompt:  ai.InitPrompt,
                Parameter:   ai.Parameter,
                TTSType:     string(ai.TTSType),
                TTSVoiceID:  ai.TTSVoiceID,
                STTType:     string(ai.STTType),
                ToolNames:   ai.ToolNameStrings(),
            },
            Tools:       tools,
            Transitions: m.Transitions,
        })
    }

    return resolved, nil
}
```

### PythonRunner.Start (pythonrunner.go)

Add `resolvedTeam` parameter:

```go
type PythonRunner interface {
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
        resolvedTeam *resolvedTeamData, // NEW — nil for single-AI sessions
    ) error
}
```

The HTTP POST body gains a `resolved_team` JSON field. When nil/absent, Python behaves as today.

### runnerStartScript (runner.go)

Updated to resolve team and pass to Python:

```go
func (h *pipecatcallHandler) runnerStartScript(pc *pipecatcall.Pipecatcall, se *pipecatcall.Session) error {
    tools := h.getToolsForPipecatcall(se.Ctx, pc)

    // Resolve team if this is a team-backed AIcall
    var resolvedTeam *resolvedTeamData
    if pc.ReferenceType == pipecatcall.ReferenceTypeAICall {
        aicall, err := h.requestHandler.AIV1AIcallGet(se.Ctx, pc.ReferenceID)
        if err == nil && aicall.AssistanceType == amaicall.AssistanceTypeTeam {
            resolvedTeam, err = h.resolveTeamForPython(se.Ctx, aicall)
            if err != nil {
                return fmt.Errorf("could not resolve team for python: %w", err)
            }
        }
    }

    return h.pythonRunner.Start(
        se.Ctx, pc.ID,
        string(pc.LLMType), string(se.LLMKey), pc.LLMMessages,
        string(pc.STTType), string(pc.STTLanguage),
        string(pc.TTSType), string(pc.TTSLanguage), pc.TTSVoiceID,
        tools,
        resolvedTeam,
    )
}
```

### Files Changed (Go)

| File | Change |
|------|--------|
| `pkg/pipecatcallhandler/run.go` | Add `resolvedTeamData` structs + `resolveTeamForPython()` function |
| `pkg/pipecatcallhandler/pythonrunner.go` | Add `resolvedTeam` parameter to `PythonRunner` interface + `Start()` impl |
| `pkg/pipecatcallhandler/runner.go` | Update `runnerStartScript()` to resolve team + pass to Python |
| `pkg/pipecatcallhandler/mock_pythonrunner.go` | Regenerated via `go generate` |
| `pkg/pipecatcallhandler/run_test.go` | Add tests for `resolveTeamForPython()` |
| `pkg/pipecatcallhandler/runner_test.go` | Update existing tests for new `Start()` signature |

## Python Changes

### Data Contract (Go → Python)

Existing `PipelineRequest` fields remain unchanged. New optional `resolved_team` field:

```json
{
  "id": "pipecatcall-uuid",
  "llm_type": "openai.gpt-4o",
  "llm_key": "sk-...",
  "llm_messages": [...],
  "stt_type": "deepgram",
  "stt_language": "en",
  "tts_type": "cartesia",
  "tts_language": "en",
  "tts_voice_id": "voice-123",
  "tools": [...],
  "resolved_team": {
    "id": "team-uuid",
    "start_member_id": "greeter-member-uuid",
    "members": [
      {
        "id": "greeter-member-uuid",
        "name": "Greeter",
        "ai": {
          "engine_model": "openai.gpt-4o",
          "engine_key": "sk-...",
          "init_prompt": "You are a friendly greeter...",
          "parameter": {"company": "Acme Corp"},
          "tts_type": "cartesia",
          "tts_voice_id": "voice-123",
          "stt_type": "deepgram",
          "stt_language": "en",
          "tool_names": ["connect_call", "set_variables"]
        },
        "tools": [
          {"name": "connect_call", "description": "Transfer the call", "parameters": {...}},
          {"name": "set_variables", "description": "Save data", "parameters": {...}}
        ],
        "transitions": [
          {
            "function_name": "transfer_to_billing",
            "description": "Customer has a billing question",
            "next_member_id": "billing-member-uuid"
          }
        ]
      },
      {
        "id": "billing-member-uuid",
        "name": "Billing Specialist",
        "ai": {
          "engine_model": "grok.grok-3",
          "engine_key": "xai-...",
          "init_prompt": "You are a billing specialist...",
          "parameter": {},
          "tts_type": "elevenlabs",
          "tts_voice_id": "voice-456",
          "stt_type": "google",
          "stt_language": "en",
          "tool_names": ["send_email"]
        },
        "tools": [
          {"name": "send_email", "description": "Send an email", "parameters": {...}}
        ],
        "transitions": [
          {
            "function_name": "transfer_to_greeter",
            "description": "Customer needs something else",
            "next_member_id": "greeter-member-uuid"
          }
        ]
      }
    ]
  }
}
```

### Pydantic Models (main.py)

```python
class ResolvedAI(BaseModel):
    engine_model: str
    engine_key: str
    init_prompt: Optional[str] = None
    parameter: Optional[dict] = None
    tts_type: Optional[str] = None
    tts_voice_id: Optional[str] = None
    stt_type: Optional[str] = None
    stt_language: Optional[str] = None
    tool_names: Optional[List[str]] = Field(default_factory=list)

class TeamTransition(BaseModel):
    function_name: str
    description: str
    next_member_id: str

class ResolvedMember(BaseModel):
    id: str
    name: str
    ai: ResolvedAI
    tools: Optional[List[Tool]] = Field(default_factory=list)
    transitions: Optional[List[TeamTransition]] = Field(default_factory=list)

class ResolvedTeam(BaseModel):
    id: str
    start_member_id: str
    members: List[ResolvedMember]

class PipelineRequest(BaseModel):
    # ... existing fields ...
    resolved_team: Optional[ResolvedTeam] = None  # NEW
```

### Pipeline Architecture

**Single-AI session (unchanged):**
```
Input → STT → UserAggregator → LLM → TTS → AssistantAggregator → Output
```

**Team session:**
```
Input → RoutingSTT → UserAggregator → RoutingLLM → RoutingTTS → AssistantAggregator → Output
         ↑                              ↑               ↑
    delegates to:                  delegates to:    delegates to:
    Deepgram/Google                OpenAI/Grok      Cartesia/ElevenLabs/Google
    (per active member)            (per active member)  (per active member)
```

### Routing Service Pattern

Each routing service follows the same pattern:

1. **Pre-create** service instances for each unique config across all members at startup
2. **Deduplicate** — if two members share identical config, reuse the same instance
3. **Extend `FrameProcessor`** (pipecat's pipeline base class)
4. **Delegate `process_frame()`** to the active member's service instance
5. **Intercept output** by overriding each wrapped service's `push_frame` to route through the routing service
6. **`set_active_member(member_id)`** switches the active delegate

```python
class RoutingLLMService(FrameProcessor):
    """Routes LLM processing to the appropriate provider based on active member."""

    def __init__(self, member_services: dict[str, OpenAILLMService]):
        super().__init__()
        self._services = member_services
        self._active_id = None

        # Override each service's push_frame to route through us
        for svc in self._services.values():
            original_push = svc.push_frame
            svc.push_frame = self._create_routing_push(svc)

    def _create_routing_push(self, svc):
        async def routing_push(frame, direction=FrameDirection.DOWNSTREAM):
            await self.push_frame(frame, direction)
        return routing_push

    def set_active_member(self, member_id: str):
        self._active_id = member_id

    async def process_frame(self, frame, direction):
        active = self._services.get(self._active_id)
        if active:
            await active.process_frame(frame, direction)
        else:
            await self.push_frame(frame, direction)

    # Delegate FlowManager-facing methods to active service
    def register_function(self, name, handler):
        self._services[self._active_id].register_function(name, handler)

    def unregister_function(self, name):
        self._services[self._active_id].unregister_function(name)
```

The same pattern applies to `RoutingTTSService` and `RoutingSTTService`, substituting the appropriate service types and delegate methods.

### Team Flow Translation (team_flow.py)

Translates `resolved_team` into pipecat-flows `NodeConfig` objects:

```python
def build_team_flow(
    resolved_team: ResolvedTeam,
    pipecatcall_id: str,
    routing_llm: RoutingLLMService,
    routing_tts: RoutingTTSService,
    routing_stt: RoutingSTTService,
) -> tuple[dict[str, NodeConfig], NodeConfig]:
    """Build NodeConfig objects for all members. Returns (member_nodes, start_node)."""

    member_nodes = {}

    for member in resolved_team.members:
        # Build regular tool functions (call Go's HTTP endpoint)
        tool_functions = []
        for tool in member.tools:
            tool_functions.append(FlowsFunctionSchema(
                name=tool.name,
                description=tool.description,
                properties=tool.parameters.get("properties", {}),
                required=tool.parameters.get("required", []),
                handler=create_tool_handler(tool.name, pipecatcall_id),
            ))

        # Build transition functions (switch routing services + return next node)
        for transition in member.transitions:
            tool_functions.append(FlowsFunctionSchema(
                name=transition.function_name,
                description=transition.description,
                properties={},
                required=[],
                handler=create_transition_handler(
                    transition.next_member_id,
                    member_nodes,  # reference to dict, populated below
                    routing_llm, routing_tts, routing_stt,
                ),
            ))

        # Build NodeConfig
        role_messages = []
        if member.ai.init_prompt:
            role_messages.append({
                "role": "system",
                "content": member.ai.init_prompt,
            })

        node = NodeConfig(
            name=member.name,
            role_messages=role_messages,
            task_messages=[],
            functions=tool_functions,
        )
        member_nodes[member.id] = node

    start_node = member_nodes[resolved_team.start_member_id]
    return member_nodes, start_node


def create_tool_handler(tool_name: str, pipecatcall_id: str):
    """Create a FlowsFunctionSchema handler that calls Go's tool endpoint."""
    async def handler(args: FlowArgs, flow_manager: FlowManager):
        result = await call_go_tool_endpoint(tool_name, args, pipecatcall_id)
        return result, None  # None = stay on current node
    return handler


def create_transition_handler(
    next_member_id: str,
    member_nodes: dict,
    routing_llm, routing_tts, routing_stt,
):
    """Create a FlowsFunctionSchema handler for member transitions."""
    async def handler(args: FlowArgs, flow_manager: FlowManager):
        try:
            routing_llm.set_active_member(next_member_id)
            routing_tts.set_active_member(next_member_id)
            routing_stt.set_active_member(next_member_id)
        except Exception as e:
            logger.error(f"Transition failed: {e}")
            return {"error": str(e)}, None  # stay on current node
        return {"status": "transferred"}, member_nodes[next_member_id]
    return handler
```

### Modified run.py

Two-path pipeline creation:

```python
async def run_pipeline(
    id: str,
    llm_type: str,
    llm_key: str,
    llm_messages: list = None,
    stt_type: str = None,
    stt_language: str = None,
    tts_type: str = None,
    tts_language: str = None,
    tts_voice_id: str = None,
    tools_data: list = None,
    resolved_team: dict = None,  # NEW
):
    if resolved_team:
        await run_team_pipeline(id, resolved_team, stt_language, tts_language, llm_messages, tools_data)
    else:
        await run_single_ai_pipeline(id, llm_type, llm_key, llm_messages,
                                      stt_type, stt_language, tts_type, tts_language,
                                      tts_voice_id, tools_data)
```

The existing `run_pipeline` logic moves into `run_single_ai_pipeline()` (renamed, no behavior change). The new `run_team_pipeline()` implements:

1. Create routing services from resolved_team member configs
2. Build pipeline with routing services
3. Create FlowManager with routing LLM + context aggregator
4. Build team flow (NodeConfigs for all members)
5. Initialize FlowManager with start member's NodeConfig
6. Run pipeline

### Files Changed (Python)

| File | Change |
|------|--------|
| `main.py` | Add Pydantic models (ResolvedTeam, ResolvedMember, ResolvedAI, TeamTransition). Add `resolved_team` to PipelineRequest. Pass to `run_pipeline()`. |
| `run.py` | Split into `run_single_ai_pipeline()` (existing) + `run_team_pipeline()` (new). Add team detection in `run_pipeline()`. |
| `routing_llm.py` | **New.** RoutingLLMService implementation. |
| `routing_tts.py` | **New.** RoutingTTSService implementation. |
| `routing_stt.py` | **New.** RoutingSTTService implementation. |
| `team_flow.py` | **New.** Team-to-NodeConfig translation, transition handlers, tool handlers. |
| `pyproject.toml` | Add `pipecat-flows` dependency. |

## Transition Flow Detail

```
1. User: "I have a billing question"

2. LLM (Greeter member, openai.gpt-4o) processes user input
   - Available functions: connect_call, set_variables, transfer_to_billing, transfer_to_support
   - LLM decides to call transfer_to_billing()

3. FlowsFunctionSchema handler fires:
   a. routing_llm.set_active_member("billing-uuid")
      → Next LLM call uses grok.grok-3 with xai key
   b. routing_tts.set_active_member("billing-uuid")
      → Next TTS uses elevenlabs with voice-456
   c. routing_stt.set_active_member("billing-uuid")
      → Next STT uses google
   d. Returns ({"status": "transferred"}, billing_node_config)

4. FlowManager applies billing_node_config:
   a. Updates context: adds billing role_messages (InitPrompt)
   b. Unregisters greeter's functions
   c. Registers billing's functions: send_email, transfer_to_greeter
   d. Triggers LLM response generation

5. LLM (now Billing member, grok.grok-3) generates response:
   "I'd be happy to help with your billing question. What would you like to know?"

6. TTS (now elevenlabs, voice-456) synthesizes response
   → User hears a different voice (billing specialist)
```

## Backward Compatibility

- `resolved_team` is optional in the HTTP POST body (None for single-AI sessions)
- All existing single-AI behavior is completely unchanged (same code path)
- Go callers that don't set `AssistanceTypeTeam` are unaffected
- No bin-common-handler changes needed
- No RPC interface changes

## Risks and Mitigations

| Risk | Level | Impact | Mitigation |
|------|-------|--------|------------|
| Routing service FrameProcessor integration | HIGH | Core feature broken | Monkey-patch `push_frame` on wrapped services. Build RoutingLLMService first as PoC since all LLM providers are OpenAI-compatible. |
| Context aggregator compatibility | MEDIUM | Context corruption across LLM switches | Use `LLMContextAggregatorPair(LLMContext())` directly (pipecat-flows pattern) rather than `llm.create_context_aggregator()`. |
| Startup time increase | MEDIUM | Slower call pickup | Deduplicate service instances with identical configs. Parallel initialization (existing pattern). |
| pipecat-flows version incompatibility | LOW | Build failure | Check pipecat-flows requirements against pipecat>=0.0.103. Pin both versions. |
| Partial STT transcription loss during switch | LOW | Brief transcription gap | Acceptable — transitions happen during function calls, not mid-speech. |
| Transition error recovery | MEDIUM | Broken conversation | Wrap service switches in rollback pattern: on failure, revert all services to previous member. |

## Not in Scope

- OpenAPI schema updates for teams
- API manager servicehandler for teams
- Database migration execution
- bin-common-handler RPC signature changes
- Go-side tracking of active member transitions
- pre_actions / post_actions on nodes
- Context strategies (RESET, RESET_WITH_SUMMARY)
- STT language auto-detection
- Visual flow editor integration

## Test Coverage

### Go Tests

**resolveTeamForPython (run_test.go):**
- Team with multiple members → all AIs fetched, tools resolved, keys included
- Single-AI AIcall → returns nil
- Member's AI not found → error
- Tool resolution failure → graceful degradation (empty tools)

**runnerStartScript (runner_test.go):**
- Team AIcall → resolvedTeam passed to pythonRunner.Start()
- Non-team AIcall → nil passed (existing behavior)
- Team resolution error → error propagated

### Python Tests

**RoutingLLMService (test_routing_llm.py):**
- set_active_member switches delegate
- process_frame delegates to active service
- register_function/unregister_function delegate correctly
- Invalid member_id → error handling

**RoutingTTSService (test_routing_tts.py):**
- Same pattern as LLM routing tests
- Different TTS provider types work correctly

**RoutingSTTService (test_routing_stt.py):**
- Same pattern as LLM routing tests

**team_flow (test_team_flow.py):**
- build_team_flow creates correct NodeConfigs for all members
- Transition handler switches all routing services
- Tool handler calls Go endpoint correctly
- Start node matches start_member_id

**run (test_run.py):**
- resolved_team present → team pipeline path taken
- resolved_team absent → single-AI pipeline path (existing)
