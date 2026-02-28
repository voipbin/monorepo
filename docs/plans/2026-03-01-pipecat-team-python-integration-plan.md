# Pipecat Team Python Integration - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add multi-member team support to the Python pipecat runner so each team member can have different prompts, tools, LLM providers, TTS voices/providers, and STT providers, with seamless transitions during a call.

**Architecture:** Go side (bin-pipecat-manager) resolves team data from ai-manager RPCs and passes it to the Python runner via the existing HTTP POST `/run` endpoint. Python side uses pipecat-flows `FlowManager` for conversation flow management and routing services (LLM, TTS, STT) that switch delegates per-member on transitions. Single-AI sessions are completely unchanged.

**Tech Stack:** Go (bin-pipecat-manager), Python (pipecat-ai, pipecat-flows), FastAPI, gomock

**Design Doc:** `docs/plans/2026-03-01-pipecat-team-python-integration-design.md`

---

## Phase 1: Go Changes

### Task 1: Add resolvedTeamData Structs

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/run.go`

**Step 1: Add the data structs after the existing imports and constants**

Add these structs at the top of `run.go` (after the constants block at line 18), before any functions:

```go
// resolvedTeamData is the Python-facing team struct that includes EngineKey per member.
// This is safe because Go<>Python is localhost communication within the same pod.
type resolvedTeamData struct {
	ID            uuid.UUID            `json:"id"`
	StartMemberID uuid.UUID            `json:"start_member_id"`
	Members       []resolvedMemberData `json:"members"`
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
}
```

Note: `resolvedAIData` intentionally omits `STTLanguage` because the `ai.AI` model does not have this field. STT language is set at the pipecatcall level and is passed to Python at the top level, shared across all members.

Add these imports to the import block (they may already exist partially):

```go
import (
	"context"
	"fmt"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	aitool "monorepo/bin-ai-manager/models/tool"
	amteam "monorepo/bin-ai-manager/models/team"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)
```

**Step 2: Verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration/bin-pipecat-manager && go build ./...`
Expected: BUILD SUCCESS (structs are unused yet but go allows unused types)

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration
git add bin-pipecat-manager/pkg/pipecatcallhandler/run.go
git commit -m "NOJIRA-add-pipecat-team-python-integration

- bin-pipecat-manager: Add resolvedTeamData, resolvedMemberData, resolvedAIData structs for Python-facing team data"
```

---

### Task 2: Add resolveTeamForPython Function

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/run.go`

**Step 1: Write the failing test**

Create file `bin-pipecat-manager/pkg/pipecatcallhandler/run_test.go`:

```go
package pipecatcallhandler

import (
	"context"
	"fmt"
	"testing"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	amteam "monorepo/bin-ai-manager/models/team"
	aitool "monorepo/bin-ai-manager/models/tool"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_resolveTeamForPython(t *testing.T) {
	tests := []struct {
		name string

		aicall *amaicall.AIcall

		// mock returns
		team    *amteam.Team
		teamErr error

		ais    map[uuid.UUID]*amai.AI
		aiErrs map[uuid.UUID]error

		tools []aitool.Tool

		expectNil bool
		expectErr bool
		expectLen int // expected number of members
	}{
		{
			name: "non-team aicall returns nil",
			aicall: &amaicall.AIcall{
				AssistanceType: amaicall.AssistanceTypeAI,
				AssistanceID:   uuid.Must(uuid.NewV4()),
			},
			expectNil: true,
		},
		{
			name: "team with two members resolves correctly",
			aicall: &amaicall.AIcall{
				AssistanceType: amaicall.AssistanceTypeTeam,
				AssistanceID:   uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000001")),
			},
			team: &amteam.Team{
				Identity: commonidentity.Identity{
					ID: uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000001")),
				},
				StartMemberID: uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000010")),
				Members: []amteam.Member{
					{
						ID:   uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000010")),
						Name: "Greeter",
						AIID: uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000100")),
						Transitions: []amteam.Transition{
							{
								FunctionName: "transfer_to_billing",
								Description:  "Transfer to billing",
								NextMemberID: uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000020")),
							},
						},
					},
					{
						ID:   uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000020")),
						Name: "Billing",
						AIID: uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000200")),
						Transitions: []amteam.Transition{
							{
								FunctionName: "transfer_to_greeter",
								Description:  "Transfer to greeter",
								NextMemberID: uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000010")),
							},
						},
					},
				},
			},
			ais: map[uuid.UUID]*amai.AI{
				uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000100")): {
					Identity: commonidentity.Identity{
						ID: uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000100")),
					},
					EngineModel: "openai.gpt-4o",
					EngineKey:   "sk-test-greeter",
					InitPrompt:  "You are a greeter",
					TTSType:     "cartesia",
					TTSVoiceID:  "voice-greeter",
					STTType:     "deepgram",
					ToolNames:   []aitool.ToolName{"connect_call"},
				},
				uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000200")): {
					Identity: commonidentity.Identity{
						ID: uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000200")),
					},
					EngineModel: "grok.grok-3",
					EngineKey:   "xai-test-billing",
					InitPrompt:  "You are billing specialist",
					TTSType:     "elevenlabs",
					TTSVoiceID:  "voice-billing",
					STTType:     "google",
					ToolNames:   []aitool.ToolName{"send_email"},
				},
			},
			tools: []aitool.Tool{
				{Name: "connect_call", Description: "Connect a call"},
				{Name: "send_email", Description: "Send an email"},
			},
			expectLen: 2,
		},
		{
			name: "team fetch error returns error",
			aicall: &amaicall.AIcall{
				AssistanceType: amaicall.AssistanceTypeTeam,
				AssistanceID:   uuid.Must(uuid.NewV4()),
			},
			teamErr:   fmt.Errorf("team not found"),
			expectErr: true,
		},
		{
			name: "member AI fetch error returns error",
			aicall: &amaicall.AIcall{
				AssistanceType: amaicall.AssistanceTypeTeam,
				AssistanceID:   uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000001")),
			},
			team: &amteam.Team{
				Identity: commonidentity.Identity{
					ID: uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000001")),
				},
				StartMemberID: uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000010")),
				Members: []amteam.Member{
					{
						ID:   uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000010")),
						Name: "Greeter",
						AIID: uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000100")),
					},
				},
			},
			aiErrs: map[uuid.UUID]error{
				uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000100")): fmt.Errorf("AI not found"),
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReqHandler := NewMockRequestHandler(mc)
			mockToolHandler := NewMockToolHandler(mc)

			h := &pipecatcallHandler{
				requestHandler: mockReqHandler,
				toolHandler:    mockToolHandler,
			}

			ctx := context.Background()

			// Set up mock expectations
			if tt.aicall.AssistanceType == amaicall.AssistanceTypeTeam {
				mockReqHandler.EXPECT().AIV1TeamGet(ctx, tt.aicall.AssistanceID).Return(tt.team, tt.teamErr)

				if tt.teamErr == nil && tt.team != nil {
					for _, m := range tt.team.Members {
						ai := tt.ais[m.AIID]
						aiErr := tt.aiErrs[m.AIID]
						mockReqHandler.EXPECT().AIV1AIGet(ctx, m.AIID).Return(ai, aiErr)
						if aiErr != nil {
							break // stop after first error
						}
						mockToolHandler.EXPECT().GetByNames(ai.ToolNames).Return(tt.tools)
					}
				}
			}

			result, err := h.resolveTeamForPython(ctx, tt.aicall)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.expectNil {
				if result != nil {
					t.Errorf("expected nil result but got: %+v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("expected non-nil result")
				return
			}

			if len(result.Members) != tt.expectLen {
				t.Errorf("expected %d members, got %d", tt.expectLen, len(result.Members))
			}

			// Verify start member ID is set
			if result.StartMemberID != tt.team.StartMemberID {
				t.Errorf("expected start_member_id %s, got %s", tt.team.StartMemberID, result.StartMemberID)
			}

			// Verify engine keys are included (the whole point of this struct)
			for _, m := range result.Members {
				if m.AI.EngineKey == "" {
					t.Errorf("member %s has empty engine key", m.Name)
				}
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration/bin-pipecat-manager && go test ./pkg/pipecatcallhandler/... -run Test_resolveTeamForPython -v`
Expected: FAIL — `resolveTeamForPython` undefined

**Step 3: Write the implementation**

Add to `run.go` after the `resolveAIFromAIcall` function:

```go
// resolveTeamForPython builds the full team data for the Python runner, including engine keys.
// Returns nil if the AIcall is not team-backed.
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
	logrus.WithField("team", team).Debugf("Retrieved team info. team_id: %s", team.ID)

	resolved := &resolvedTeamData{
		ID:            team.ID,
		StartMemberID: team.StartMemberID,
	}

	for _, m := range team.Members {
		ai, errAI := h.requestHandler.AIV1AIGet(ctx, m.AIID)
		if errAI != nil {
			return nil, fmt.Errorf("could not get AI for member %s: %w", m.ID, errAI)
		}
		logrus.WithField("ai", ai).Debugf("Retrieved AI info for member. member_id: %s, ai_id: %s", m.ID, m.AIID)

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
			},
			Tools:       tools,
			Transitions: m.Transitions,
		})
	}

	return resolved, nil
}
```

Also add the needed imports (add `aitool`, `amteam`, and `uuid` to the import block):

```go
import (
	"context"
	"fmt"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	amteam "monorepo/bin-ai-manager/models/team"
	aitool "monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)
```

**Step 4: Run test to verify it passes**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration/bin-pipecat-manager && go test ./pkg/pipecatcallhandler/... -run Test_resolveTeamForPython -v`
Expected: PASS

**Step 5: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration
git add bin-pipecat-manager/pkg/pipecatcallhandler/run.go bin-pipecat-manager/pkg/pipecatcallhandler/run_test.go
git commit -m "NOJIRA-add-pipecat-team-python-integration

- bin-pipecat-manager: Add resolveTeamForPython function with unit tests"
```

---

### Task 3: Update PythonRunner Interface

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/pythonrunner.go`

**Step 1: Update the PythonRunner interface**

In `pythonrunner.go`, add `resolvedTeam *resolvedTeamData` as the last parameter to `Start()` in both the interface and the implementation.

Interface (line 38-53):
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
		resolvedTeam *resolvedTeamData,
	) error
	Stop(ctx context.Context, pipecatcallID uuid.UUID) error
}
```

Implementation (line 59-134) — update function signature and add `ResolvedTeam` to the request body struct:

```go
func (h *pythonRunner) Start(
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
) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Start",
	})

	// Request body structure for Python runner
	reqBody := struct {
		ID           uuid.UUID          `json:"id,omitempty"`
		LLMType      string             `json:"llm_type,omitempty"`
		LLMKey       string             `json:"llm_key,omitempty"`
		LLMMessages  []map[string]any   `json:"llm_messages,omitempty"`
		STTType      string             `json:"stt_type,omitempty"`
		STTLanguage  string             `json:"stt_language,omitempty"`
		TTSType      string             `json:"tts_type,omitempty"`
		TTSLanguage  string             `json:"tts_language,omitempty"`
		TTSVoiceID   string             `json:"tts_voice_id,omitempty"`
		Tools        []aitool.Tool      `json:"tools,omitempty"`
		ResolvedTeam *resolvedTeamData  `json:"resolved_team,omitempty"`
	}{
		ID:           pipecatcallID,
		LLMType:      llmType,
		LLMKey:       llmKey,
		LLMMessages:  llmMessages,
		STTType:      sttType,
		STTLanguage:  sttLanguage,
		TTSType:      ttsType,
		TTSLanguage:  ttsLanguage,
		TTSVoiceID:   ttsVoiceID,
		Tools:        tools,
		ResolvedTeam: resolvedTeam,
	}

	// ... rest of the function is unchanged
```

**Step 2: Regenerate mocks**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration/bin-pipecat-manager && go generate ./pkg/pipecatcallhandler/...`
Expected: `mock_pythonrunner.go` regenerated with new `Start()` signature

**Step 3: Verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration/bin-pipecat-manager && go build ./...`
Expected: BUILD FAILURE — `runner.go` still calls old `Start()` signature (expected, fixed in next task)

**Step 4: Commit interface + mock changes**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration
git add bin-pipecat-manager/pkg/pipecatcallhandler/pythonrunner.go bin-pipecat-manager/pkg/pipecatcallhandler/mock_pythonrunner.go
git commit -m "NOJIRA-add-pipecat-team-python-integration

- bin-pipecat-manager: Add resolvedTeam parameter to PythonRunner.Start interface"
```

---

### Task 4: Update runnerStartScript and Run Verification

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:40-69`

**Step 1: Update runnerStartScript**

Replace the `runnerStartScript` function (lines 40-69) in `runner.go`:

```go
func (h *pipecatcallHandler) runnerStartScript(pc *pipecatcall.Pipecatcall, se *pipecatcall.Session) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Start",
		"pipecatcall_id": pc.ID,
	})
	log.Debugf("Starting pipecat runner. pipecatcall_id: %s", pc.ID)

	// Get tools for this pipecat call based on reference type
	tools := h.getToolsForPipecatcall(se.Ctx, pc)
	log.WithField("tool_count", len(tools)).Debugf("Retrieved tools for pipecat call")

	// Resolve team if this is a team-backed AIcall
	var resolvedTeam *resolvedTeamData
	if pc.ReferenceType == pipecatcall.ReferenceTypeAICall {
		aicall, err := h.requestHandler.AIV1AIcallGet(se.Ctx, pc.ReferenceID)
		if err == nil {
			resolvedTeam, err = h.resolveTeamForPython(se.Ctx, aicall)
			if err != nil {
				return fmt.Errorf("could not resolve team for python: %w", err)
			}
			if resolvedTeam != nil {
				log.WithField("team_id", resolvedTeam.ID).Debugf("Resolved team for python runner")
			}
		} else {
			log.WithError(err).Warnf("Could not get AIcall for team resolution, proceeding without team data")
		}
	}

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
	); errStart != nil {
		return errors.Wrapf(errStart, "could not start python client")
	}
	log.Debugf("Pipecat runner started successfully.")

	return nil
}
```

Add `fmt` to the import block in `runner.go` if not already present. Also add `amaicall` import:

```go
import (
	// ... existing imports ...
	amaicall "monorepo/bin-ai-manager/models/aicall"
	"fmt"
)
```

**Step 2: Run full verification workflow**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration/bin-pipecat-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: ALL PASS

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration
git add bin-pipecat-manager/
git commit -m "NOJIRA-add-pipecat-team-python-integration

- bin-pipecat-manager: Update runnerStartScript to resolve team data and pass to Python runner
- bin-pipecat-manager: Add team resolution with AIcall detection for team-backed sessions"
```

---

## Phase 2: Python Foundation

### Task 5: Add pipecat-flows Dependency

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/pyproject.toml`

**Step 1: Add pipecat-flows to dependencies**

Add `"pipecat-ai-flows>=0.0.10"` to the dependencies list in `pyproject.toml`. The updated dependencies section:

```toml
dependencies = [
    "python-dotenv",
    "uvicorn",
    "pipecat-ai[silero,deepgram,openai,cartesia,websocket,google]>=0.0.103",
    "pipecat-ai-flows>=0.0.10",
    "onnxruntime>=1.20.1",
    "pillow>=12.1.1",
    "protobuf>=5.29.6",
    "loguru",
]
```

Note: The pipecat-flows PyPI package name is `pipecat-ai-flows`. Check the actual package name before installing. If the package is named differently, adjust accordingly.

**Step 2: Verify dependency installs**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration/bin-pipecat-manager/scripts/pipecat && pip install -e .`
Expected: pipecat-ai-flows installed successfully

If the package name is wrong, check `pip search pipecat-flows` or `pip install pipecat-flows` to find the correct name and update `pyproject.toml`.

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration
git add bin-pipecat-manager/scripts/pipecat/pyproject.toml
git commit -m "NOJIRA-add-pipecat-team-python-integration

- bin-pipecat-manager: Add pipecat-flows dependency for team conversation flow support"
```

---

### Task 6: Add Pydantic Models for Team Data

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/main.py`

**Step 1: Add Pydantic models**

Add these models in `main.py` after the existing `Tool` model (after line 37) and before `PipelineRequest`:

```python
class ResolvedAI(BaseModel):
    engine_model: str
    engine_key: str
    init_prompt: Optional[str] = None
    parameter: Optional[dict] = None
    tts_type: Optional[str] = None
    tts_voice_id: Optional[str] = None
    stt_type: Optional[str] = None

    class Config:
        extra = "ignore"

class TeamTransition(BaseModel):
    function_name: str
    description: str
    next_member_id: str

    class Config:
        extra = "ignore"

class ResolvedMember(BaseModel):
    id: str
    name: str
    ai: ResolvedAI
    tools: Optional[List[Tool]] = Field(default_factory=list)
    transitions: Optional[List[TeamTransition]] = Field(default_factory=list)

    class Config:
        extra = "ignore"

class ResolvedTeam(BaseModel):
    id: str
    start_member_id: str
    members: List[ResolvedMember]

    class Config:
        extra = "ignore"
```

Then add `resolved_team` to `PipelineRequest`:

```python
class PipelineRequest(BaseModel):
    id: Optional[str] = None
    llm_type: Optional[str] = None
    llm_key: Optional[str] = None
    llm_messages: Optional[List[Message]] = Field(default_factory=list)
    stt_type: Optional[str] = None
    stt_language: Optional[str] = None
    tts_type: Optional[str] = None
    tts_language: Optional[str] = None
    tts_voice_id: Optional[str] = None
    tools: Optional[List[Tool]] = Field(default_factory=list)
    resolved_team: Optional[ResolvedTeam] = None
```

**Step 2: Update run_pipeline_wrapper to pass resolved_team**

Update the `run_pipeline_wrapper` function to pass `resolved_team`:

```python
async def run_pipeline_wrapper(req: PipelineRequest):
    try:
        logger.info(f"Pipeline started: id={req.id}")
        # Convert Tool objects to dict format for LLM
        tools_data = [t.model_dump() for t in req.tools] if req.tools else []
        resolved_team_data = req.resolved_team.model_dump() if req.resolved_team else None
        await run_pipeline(
            req.id,
            req.llm_type,
            req.llm_key,
            [m.model_dump() for m in req.llm_messages],
            req.stt_type,
            req.stt_language,
            req.tts_type,
            req.tts_language,
            req.tts_voice_id,
            tools_data,
            resolved_team=resolved_team_data,
        )
        logger.info(f"Pipeline finished successfully: id={req.id}")
    except Exception as e:
        logger.exception(f"Pipeline failed (id={req.id}): {e}")
```

Also update the `/run` endpoint log to include resolved_team presence:

```python
logger.info(json.dumps({
    "event": "run_request",
    "id": req.id,
    "llm_type": req.llm_type,
    "llm_message_count": msg_count,
    "stt_type": req.stt_type,
    "stt_language": req.stt_language,
    "tts_type": req.tts_type,
    "tts_language": req.tts_language,
    "tts_voice_id": req.tts_voice_id,
    "has_resolved_team": req.resolved_team is not None,
}))
```

**Step 3: Verify FastAPI starts**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration/bin-pipecat-manager/scripts/pipecat && python -c "from main import app; print('OK')"`
Expected: `OK`

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration
git add bin-pipecat-manager/scripts/pipecat/main.py
git commit -m "NOJIRA-add-pipecat-team-python-integration

- bin-pipecat-manager: Add Pydantic models for team data (ResolvedTeam, ResolvedMember, ResolvedAI, TeamTransition)
- bin-pipecat-manager: Add resolved_team field to PipelineRequest"
```

---

### Task 7: Update run.py with Two-Path Dispatch

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/run.py`

**Step 1: Add resolved_team parameter and dispatch**

Update the `run_pipeline` function signature (line 46) to accept `resolved_team` and dispatch:

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
    resolved_team: dict = None,
):
    if resolved_team:
        await run_team_pipeline(
            id, resolved_team,
            stt_language=stt_language,
            tts_language=tts_language,
            llm_messages=llm_messages,
            tools_data=tools_data,
        )
    else:
        await run_single_ai_pipeline(
            id, llm_type, llm_key, llm_messages,
            stt_type, stt_language, tts_type, tts_language,
            tts_voice_id, tools_data,
        )
```

Rename the existing `run_pipeline` body to `run_single_ai_pipeline`. The entire body of the current function (from `total_start = time.monotonic()` through the end) becomes `run_single_ai_pipeline` with the same parameters (minus `resolved_team`):

```python
async def run_single_ai_pipeline(
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
):
    # <entire existing run_pipeline body unchanged>
    total_start = time.monotonic()
    # ... everything else stays identical ...
```

Add a placeholder `run_team_pipeline` at the end of the file:

```python
async def run_team_pipeline(
    id: str,
    resolved_team: dict,
    stt_language: str = None,
    tts_language: str = None,
    llm_messages: list = None,
    tools_data: list = None,
):
    """Team pipeline — implemented in a later task."""
    logger.error(f"[TEAM] Team pipeline not yet implemented. pipeline id={id}")
    raise NotImplementedError("Team pipeline not yet implemented")
```

**Step 2: Verify the single-AI path still works**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration/bin-pipecat-manager/scripts/pipecat && python -c "from run import run_pipeline, run_single_ai_pipeline, run_team_pipeline; print('OK')"`
Expected: `OK`

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration
git add bin-pipecat-manager/scripts/pipecat/run.py
git commit -m "NOJIRA-add-pipecat-team-python-integration

- bin-pipecat-manager: Split run_pipeline into two-path dispatch (single AI vs team)
- bin-pipecat-manager: Rename existing pipeline logic to run_single_ai_pipeline
- bin-pipecat-manager: Add run_team_pipeline placeholder"
```

---

## Phase 3: Routing Services

### Task 8: Create RoutingLLMService

**Files:**
- Create: `bin-pipecat-manager/scripts/pipecat/routing_llm.py`

**Step 1: Create the routing LLM service**

Create `bin-pipecat-manager/scripts/pipecat/routing_llm.py`:

```python
from loguru import logger

from pipecat.frames.frames import Frame
from pipecat.processors.frame_processor import FrameDirection, FrameProcessor


class RoutingLLMService(FrameProcessor):
    """Routes LLM processing to the appropriate provider based on active member.

    Wraps multiple LLM service instances and delegates process_frame to the
    active member's service. Output from wrapped services is routed through
    this processor via push_frame interception.
    """

    def __init__(self, member_services: dict[str, any]):
        """Initialize with a dict mapping member_id -> LLM service instance."""
        super().__init__()
        self._services = member_services
        self._active_id = None

        # Override each service's push_frame to route output through us
        for member_id, svc in self._services.items():
            svc.push_frame = self._create_routing_push(svc)

    def _create_routing_push(self, svc):
        async def routing_push(frame: Frame, direction: FrameDirection = FrameDirection.DOWNSTREAM):
            await self.push_frame(frame, direction)
        return routing_push

    def set_active_member(self, member_id: str):
        if member_id not in self._services:
            logger.error(f"Unknown member_id for LLM routing: {member_id}")
            return
        self._active_id = member_id
        logger.info(f"LLM routing switched to member: {member_id}")

    async def process_frame(self, frame: Frame, direction: FrameDirection):
        if self._active_id and self._active_id in self._services:
            await self._services[self._active_id].process_frame(frame, direction)
        else:
            await self.push_frame(frame, direction)

    # Delegate FlowManager-facing methods to active service
    def register_function(self, name, handler):
        if self._active_id and self._active_id in self._services:
            self._services[self._active_id].register_function(name, handler)
        else:
            logger.warning(f"Cannot register function '{name}': no active LLM service")

    def unregister_function(self, name):
        # Unregister from ALL services since function may have been registered
        # on a previously active service
        for svc in self._services.values():
            try:
                svc.unregister_function(name)
            except (KeyError, Exception):
                pass

    @property
    def active_service(self):
        """Return the currently active LLM service instance."""
        if self._active_id:
            return self._services.get(self._active_id)
        return None
```

**Step 2: Verify import**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration/bin-pipecat-manager/scripts/pipecat && python -c "from routing_llm import RoutingLLMService; print('OK')"`
Expected: `OK`

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration
git add bin-pipecat-manager/scripts/pipecat/routing_llm.py
git commit -m "NOJIRA-add-pipecat-team-python-integration

- bin-pipecat-manager: Add RoutingLLMService that delegates to per-member LLM instances"
```

---

### Task 9: Create RoutingTTSService

**Files:**
- Create: `bin-pipecat-manager/scripts/pipecat/routing_tts.py`

**Step 1: Create the routing TTS service**

Create `bin-pipecat-manager/scripts/pipecat/routing_tts.py`:

```python
from loguru import logger

from pipecat.frames.frames import Frame
from pipecat.processors.frame_processor import FrameDirection, FrameProcessor


class RoutingTTSService(FrameProcessor):
    """Routes TTS processing to the appropriate provider based on active member.

    Same pattern as RoutingLLMService but for TTS services.
    """

    def __init__(self, member_services: dict[str, any]):
        """Initialize with a dict mapping member_id -> TTS service instance."""
        super().__init__()
        self._services = member_services
        self._active_id = None

        # Override each service's push_frame to route output through us
        for member_id, svc in self._services.items():
            svc.push_frame = self._create_routing_push(svc)

    def _create_routing_push(self, svc):
        async def routing_push(frame: Frame, direction: FrameDirection = FrameDirection.DOWNSTREAM):
            await self.push_frame(frame, direction)
        return routing_push

    def set_active_member(self, member_id: str):
        if member_id not in self._services:
            logger.error(f"Unknown member_id for TTS routing: {member_id}")
            return
        self._active_id = member_id
        logger.info(f"TTS routing switched to member: {member_id}")

    async def process_frame(self, frame: Frame, direction: FrameDirection):
        if self._active_id and self._active_id in self._services:
            await self._services[self._active_id].process_frame(frame, direction)
        else:
            await self.push_frame(frame, direction)
```

**Step 2: Verify import**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration/bin-pipecat-manager/scripts/pipecat && python -c "from routing_tts import RoutingTTSService; print('OK')"`
Expected: `OK`

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration
git add bin-pipecat-manager/scripts/pipecat/routing_tts.py
git commit -m "NOJIRA-add-pipecat-team-python-integration

- bin-pipecat-manager: Add RoutingTTSService that delegates to per-member TTS instances"
```

---

### Task 10: Create RoutingSTTService

**Files:**
- Create: `bin-pipecat-manager/scripts/pipecat/routing_stt.py`

**Step 1: Create the routing STT service**

Create `bin-pipecat-manager/scripts/pipecat/routing_stt.py`:

```python
from loguru import logger

from pipecat.frames.frames import Frame
from pipecat.processors.frame_processor import FrameDirection, FrameProcessor


class RoutingSTTService(FrameProcessor):
    """Routes STT processing to the appropriate provider based on active member.

    Same pattern as RoutingLLMService but for STT services.
    """

    def __init__(self, member_services: dict[str, any]):
        """Initialize with a dict mapping member_id -> STT service instance."""
        super().__init__()
        self._services = member_services
        self._active_id = None

        # Override each service's push_frame to route output through us
        for member_id, svc in self._services.items():
            svc.push_frame = self._create_routing_push(svc)

    def _create_routing_push(self, svc):
        async def routing_push(frame: Frame, direction: FrameDirection = FrameDirection.DOWNSTREAM):
            await self.push_frame(frame, direction)
        return routing_push

    def set_active_member(self, member_id: str):
        if member_id not in self._services:
            logger.error(f"Unknown member_id for STT routing: {member_id}")
            return
        self._active_id = member_id
        logger.info(f"STT routing switched to member: {member_id}")

    async def process_frame(self, frame: Frame, direction: FrameDirection):
        if self._active_id and self._active_id in self._services:
            await self._services[self._active_id].process_frame(frame, direction)
        else:
            await self.push_frame(frame, direction)
```

**Step 2: Verify import**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration/bin-pipecat-manager/scripts/pipecat && python -c "from routing_stt import RoutingSTTService; print('OK')"`
Expected: `OK`

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration
git add bin-pipecat-manager/scripts/pipecat/routing_stt.py
git commit -m "NOJIRA-add-pipecat-team-python-integration

- bin-pipecat-manager: Add RoutingSTTService that delegates to per-member STT instances"
```

---

## Phase 4: Team Flow and Pipeline

### Task 11: Create team_flow.py

**Files:**
- Create: `bin-pipecat-manager/scripts/pipecat/team_flow.py`

**Step 1: Create the team flow module**

This module translates the resolved_team data into pipecat-flows NodeConfig objects and creates transition/tool handlers.

Create `bin-pipecat-manager/scripts/pipecat/team_flow.py`:

```python
import json
import aiohttp
import asyncio
import common

from loguru import logger
from pipecat_flows import FlowManager, FlowArgs, FlowResult, NodeConfig, FlowsFunctionSchema


def build_team_flow(
    resolved_team: dict,
    pipecatcall_id: str,
    routing_llm,
    routing_tts,
    routing_stt,
) -> tuple[dict, NodeConfig]:
    """Build NodeConfig objects for all team members.

    Args:
        resolved_team: The resolved team dict from Go.
        pipecatcall_id: The pipecatcall ID for tool endpoint calls.
        routing_llm: RoutingLLMService instance.
        routing_tts: RoutingTTSService instance.
        routing_stt: RoutingSTTService instance.

    Returns:
        Tuple of (member_nodes dict, start_node NodeConfig).
    """
    member_nodes = {}
    members_by_id = {m["id"]: m for m in resolved_team["members"]}

    for member in resolved_team["members"]:
        member_id = member["id"]
        member_name = member["name"]
        ai = member["ai"]

        # Build regular tool functions (call Go's HTTP endpoint)
        tool_functions = []
        for tool in member.get("tools", []):
            tool_functions.append(FlowsFunctionSchema(
                name=tool["name"],
                description=tool.get("description", ""),
                properties=tool.get("parameters", {}).get("properties", {}),
                required=tool.get("parameters", {}).get("required", []),
                handler=_create_tool_handler(tool["name"], pipecatcall_id),
            ))

        # Build transition functions
        for transition in member.get("transitions", []):
            tool_functions.append(FlowsFunctionSchema(
                name=transition["function_name"],
                description=transition["description"],
                properties={},
                required=[],
                handler=_create_transition_handler(
                    transition["next_member_id"],
                    member_nodes,
                    routing_llm,
                    routing_tts,
                    routing_stt,
                ),
            ))

        # Build NodeConfig
        role_messages = []
        if ai.get("init_prompt"):
            role_messages.append({
                "role": "system",
                "content": ai["init_prompt"],
            })

        node = NodeConfig(
            name=member_name,
            role_messages=role_messages,
            task_messages=[],
            functions=tool_functions,
        )
        member_nodes[member_id] = node

    start_node = member_nodes[resolved_team["start_member_id"]]
    return member_nodes, start_node


def _create_tool_handler(tool_name: str, pipecatcall_id: str):
    """Create a FlowsFunctionSchema handler that calls Go's tool endpoint."""
    async def handler(args: FlowArgs) -> FlowResult:
        result = await _call_go_tool_endpoint(tool_name, args, pipecatcall_id)
        return result
    return handler


def _create_transition_handler(
    next_member_id: str,
    member_nodes: dict,
    routing_llm,
    routing_tts,
    routing_stt,
):
    """Create a FlowsFunctionSchema handler for member transitions."""
    async def handler(args: FlowArgs) -> FlowResult:
        # Save current active member for rollback
        prev_llm = routing_llm._active_id
        prev_tts = routing_tts._active_id
        prev_stt = routing_stt._active_id

        try:
            routing_llm.set_active_member(next_member_id)
            routing_tts.set_active_member(next_member_id)
            routing_stt.set_active_member(next_member_id)
        except Exception as e:
            logger.error(f"Transition to {next_member_id} failed, rolling back: {e}")
            # Rollback
            if prev_llm:
                routing_llm.set_active_member(prev_llm)
            if prev_tts:
                routing_tts.set_active_member(prev_tts)
            if prev_stt:
                routing_stt.set_active_member(prev_stt)
            return {"error": str(e)}, None

        next_node = member_nodes.get(next_member_id)
        if next_node is None:
            logger.error(f"No NodeConfig for member {next_member_id}")
            return {"error": f"unknown member {next_member_id}"}, None

        logger.info(f"Transition to member {next_member_id} successful")
        return {"status": "transferred"}, next_node
    return handler


async def _call_go_tool_endpoint(tool_name: str, args: dict, pipecatcall_id: str) -> dict:
    """Call Go's tool execution HTTP endpoint."""
    http_url = f"{common.PIPECATCALL_HTTP_URL}/{pipecatcall_id}/tools"
    http_body = {
        "id": f"team-tool-{tool_name}",
        "type": "function",
        "function": {
            "name": tool_name,
            "arguments": json.dumps(args if isinstance(args, dict) else {}, ensure_ascii=False),
        },
    }
    logger.debug(f"[team_flow][{tool_name}] POST {http_url}")

    try:
        async with aiohttp.ClientSession() as session:
            async with session.post(http_url, json=http_body, timeout=aiohttp.ClientTimeout(total=10)) as response:
                text = await response.text()
                if response.status >= 400:
                    logger.warning(f"[team_flow][{tool_name}] HTTP {response.status}: {text[:500]}")
                    return {"status": "error", "error": f"HTTP {response.status}: {text}"}

                content_type = response.headers.get("Content-Type", "")
                if content_type.startswith("application/json"):
                    try:
                        return {"status": "ok", "data": json.loads(text)}
                    except json.JSONDecodeError:
                        return {"status": "ok", "data": {"raw": text}}
                return {"status": "ok", "data": {"raw": text}}

    except asyncio.TimeoutError:
        logger.error(f"[team_flow][{tool_name}] Request timed out")
        return {"status": "error", "error": "Request timed out"}
    except Exception as e:
        logger.exception(f"[team_flow][{tool_name}] Unexpected error: {e}")
        return {"status": "error", "error": str(e)}
```

**Important note on FlowsFunctionSchema handler signature:**
The pipecat-flows handler signature may be `async def handler(args: FlowArgs) -> FlowResult` or `async def handler(args: FlowArgs, flow_manager: FlowManager) -> FlowResult`. Check the actual pipecat-flows API docs during implementation. The transition handler must return `(result, NodeConfig)` for transitions or `(result, None)` to stay on current node.

**Step 2: Verify import**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration/bin-pipecat-manager/scripts/pipecat && python -c "from team_flow import build_team_flow; print('OK')"`
Expected: `OK` (may need pipecat-flows installed from Task 5)

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration
git add bin-pipecat-manager/scripts/pipecat/team_flow.py
git commit -m "NOJIRA-add-pipecat-team-python-integration

- bin-pipecat-manager: Add team_flow module for building NodeConfigs and transition/tool handlers"
```

---

### Task 12: Implement run_team_pipeline

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/run.py`

**Step 1: Add imports at the top of run.py**

Add these imports near the existing imports:

```python
from routing_llm import RoutingLLMService
from routing_tts import RoutingTTSService
from routing_stt import RoutingSTTService
from team_flow import build_team_flow
```

Also add the pipecat-flows imports:

```python
from pipecat_flows import FlowManager
from pipecat.processors.aggregators.openai_llm_context import OpenAILLMContext
```

Note: `OpenAILLMContext` may already be imported. The `FlowManager` import is new.

**Step 2: Replace the run_team_pipeline placeholder**

Replace the placeholder `run_team_pipeline` with the full implementation:

```python
async def run_team_pipeline(
    id: str,
    resolved_team: dict,
    stt_language: str = None,
    tts_language: str = None,
    llm_messages: list = None,
    tools_data: list = None,
):
    """Run a team-based pipeline with routing services and FlowManager."""
    total_start = time.monotonic()
    logger.info(f"[TEAM][INIT] Starting team pipeline. pipeline id={id}")

    if llm_messages is None:
        llm_messages = []

    members = resolved_team.get("members", [])
    start_member_id = resolved_team["start_member_id"]

    # --- Step 1: Create per-member service instances ---
    llm_services = {}
    tts_services = {}
    stt_services = {}

    for member in members:
        mid = member["id"]
        ai = member["ai"]

        # LLM service
        llm_svc, _ = create_llm_service(ai["engine_model"], ai["engine_key"], [], [])
        llm_services[mid] = llm_svc

        # TTS service
        if ai.get("tts_type"):
            tts_svc = create_tts_service(ai["tts_type"], voice_id=ai.get("tts_voice_id"), language=tts_language)
            tts_services[mid] = tts_svc

        # STT service
        if ai.get("stt_type"):
            stt_svc = create_stt_service(ai["stt_type"], language=stt_language)
            stt_services[mid] = stt_svc

    logger.info(f"[TEAM][INIT] Created {len(llm_services)} LLM, {len(tts_services)} TTS, {len(stt_services)} STT services. pipeline id={id}")

    # --- Step 2: Create routing services ---
    routing_llm = RoutingLLMService(llm_services)
    routing_llm.set_active_member(start_member_id)

    routing_tts = None
    if tts_services:
        routing_tts = RoutingTTSService(tts_services)
        routing_tts.set_active_member(start_member_id)

    routing_stt = None
    if stt_services:
        routing_stt = RoutingSTTService(stt_services)
        routing_stt.set_active_member(start_member_id)

    # --- Step 3: Create context aggregator ---
    # Use standalone LLMContext (not tied to a specific LLM service)
    # to avoid coupling context aggregation with the routing service.
    start_member = next(m for m in members if m["id"] == start_member_id)
    start_messages = []
    if start_member["ai"].get("init_prompt"):
        start_messages.append({"role": "system", "content": start_member["ai"]["init_prompt"]})
    start_messages.extend([m for m in llm_messages if m.get("role") and m.get("content")])

    context = OpenAILLMContext(messages=start_messages, tools=[])
    # Use the start member's LLM service to create the aggregator
    context_aggregator = llm_services[start_member_id].create_context_aggregator(context)

    # --- Step 4: Create transports ---
    vad_analyzer = SileroVADAnalyzer(params=VADParams(stop_secs=0.8))
    transport_input = create_websocket_transport("input", id, vad_analyzer=vad_analyzer)
    transport_output = create_websocket_transport("output", id, vad_analyzer=None)

    # --- Step 5: Build pipeline ---
    pipeline_stages = []

    if routing_stt:
        pipeline_stages.append(transport_input.input())
        pipeline_stages.append(routing_stt)

    pipeline_stages.append(context_aggregator.user())
    pipeline_stages.append(routing_llm)

    if routing_tts:
        pipeline_stages.append(routing_tts)

    pipeline_stages.append(context_aggregator.assistant())
    pipeline_stages.append(transport_output.output())

    pipeline = Pipeline(pipeline_stages)

    # --- Step 6: Create pipeline task ---
    task = PipelineTask(
        pipeline,
        params=PipelineParams(
            audio_out_sample_rate=16000,
            enable_metrics=True,
            enable_usage_metrics=True,
        ),
    )

    await task_manager.add(id, task)

    # --- Step 7: Set up FlowManager ---
    member_nodes, start_node = build_team_flow(
        resolved_team, id,
        routing_llm, routing_tts, routing_stt,
    )

    flow_manager = FlowManager(
        task=task,
        llm=routing_llm,
        context_aggregator=context_aggregator,
    )

    # --- Step 8: Event handlers ---
    async def handle_disconnect_or_error(name, transport, error=None):
        logger.error(f"[TEAM] {name} WebSocket disconnected or errored: {error}. pipeline id={id}")
        await task.cancel()

    transport_input.event_handler("on_disconnected")(partial(handle_disconnect_or_error, "Input"))
    transport_input.event_handler("on_error")(partial(handle_disconnect_or_error, "Input"))
    transport_output.event_handler("on_disconnected")(partial(handle_disconnect_or_error, "Output"))
    transport_output.event_handler("on_error")(partial(handle_disconnect_or_error, "Output"))

    # --- Step 9: Initialize and run ---
    # Initialize flow with start node
    await flow_manager.initialize(start_node)

    runner = PipelineRunner()

    init_total = time.monotonic() - total_start
    logger.info(f"[TEAM][INIT][total] All initialization completed in {init_total:.3f} sec. pipeline id={id}")

    try:
        logger.info(f"[TEAM][RUN] Starting team pipeline. pipeline id={id}")
        await runner.run(task)
    except asyncio.CancelledError:
        logger.info(f"[TEAM][RUN] Pipeline cancelled. pipeline id={id}")
    except Exception as e:
        logger.error(f"[TEAM][RUN] Pipeline error: {e}. pipeline id={id}")
    finally:
        logger.info(f"[TEAM][CLEANUP] Cleaning up team pipeline. pipeline id={id}")
        if task:
            await task.cancel()
        if transport_input:
            await transport_input.cleanup()
        if transport_output:
            await transport_output.cleanup()
        await task_manager.remove(id)
        logger.info(f"[TEAM][CLEANUP] Team pipeline cleaned. pipeline id={id}")
```

**Step 3: Verify imports**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration/bin-pipecat-manager/scripts/pipecat && python -c "from run import run_pipeline, run_single_ai_pipeline, run_team_pipeline; print('OK')"`
Expected: `OK`

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration
git add bin-pipecat-manager/scripts/pipecat/run.py
git commit -m "NOJIRA-add-pipecat-team-python-integration

- bin-pipecat-manager: Implement run_team_pipeline with routing services and FlowManager
- bin-pipecat-manager: Create per-member LLM/TTS/STT instances with routing delegation
- bin-pipecat-manager: Integrate pipecat-flows for team conversation management"
```

---

## Phase 5: Final Verification

### Task 13: Run Full Go Verification

**Files:** None (verification only)

**Step 1: Run full Go verification**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration/bin-pipecat-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: ALL PASS

**Step 2: Fix any issues**

If tests fail or lint errors appear, fix them before proceeding.

---

### Task 14: Verify Python Imports and Structure

**Files:** None (verification only)

**Step 1: Verify all Python modules import correctly**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration/bin-pipecat-manager/scripts/pipecat && \
python -c "
from main import app, PipelineRequest, ResolvedTeam, ResolvedMember, ResolvedAI, TeamTransition
from run import run_pipeline, run_single_ai_pipeline, run_team_pipeline
from routing_llm import RoutingLLMService
from routing_tts import RoutingTTSService
from routing_stt import RoutingSTTService
from team_flow import build_team_flow
print('All imports OK')
"
```

Expected: `All imports OK`

**Step 2: Verify Pydantic model serialization**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration/bin-pipecat-manager/scripts/pipecat && \
python -c "
from main import PipelineRequest, ResolvedTeam, ResolvedMember, ResolvedAI, TeamTransition, Tool
import json

# Build a sample team request
team = ResolvedTeam(
    id='team-1',
    start_member_id='member-1',
    members=[
        ResolvedMember(
            id='member-1',
            name='Greeter',
            ai=ResolvedAI(engine_model='openai.gpt-4o', engine_key='sk-test'),
            tools=[Tool(name='connect_call', description='Connect a call')],
            transitions=[TeamTransition(function_name='transfer_to_billing', description='Billing', next_member_id='member-2')],
        ),
        ResolvedMember(
            id='member-2',
            name='Billing',
            ai=ResolvedAI(engine_model='grok.grok-3', engine_key='xai-test', tts_type='elevenlabs', tts_voice_id='v2'),
            tools=[],
            transitions=[],
        ),
    ],
)

req = PipelineRequest(
    id='test-1',
    llm_type='openai.gpt-4o',
    llm_key='sk-test',
    resolved_team=team,
)

data = req.model_dump()
assert data['resolved_team'] is not None
assert len(data['resolved_team']['members']) == 2
assert data['resolved_team']['members'][0]['ai']['engine_key'] == 'sk-test'
print('Pydantic serialization OK')
"
```

Expected: `Pydantic serialization OK`

---

### Task 15: Final Commit and Summary

**Step 1: Verify no uncommitted changes**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-pipecat-team-python-integration && git status
```

**Step 2: Review all commits**

```bash
git log --oneline main..HEAD
```

Expected: Multiple commits covering Go structs, function, interface, runner update, Python models, routing services, team flow, and pipeline.

---

## Implementation Notes

### FlowManager API Verification

The exact pipecat-flows API must be verified during implementation. Key things to check:
1. `FlowManager` constructor parameters — may need `task`, `llm`, and `context_aggregator` or different parameter names
2. `FlowsFunctionSchema` handler signature — may be `(args)`, `(args, flow_manager)`, or something else
3. `FlowManager.initialize(node)` — verify this is the correct method to set the initial node
4. `NodeConfig` constructor — verify field names match `name`, `role_messages`, `task_messages`, `functions`

Use the Context7 MCP tool or `pip show pipecat-ai-flows` to check the installed version's API.

### STT Language Per-Member

The `ai.AI` Go model does NOT have an `STTLanguage` field. The STT language is set at the pipecatcall level (`pc.STTLanguage`) and shared across all members. Only the STT provider type (`stt_type`) varies per member. The `resolvedAIData` struct intentionally omits `STTLanguage`.

### Backward Compatibility

- `resolved_team` is `omitempty` in Go and `Optional[...] = None` in Python
- When absent, the Python runner takes the existing `run_single_ai_pipeline` path
- No changes to any existing API contracts
