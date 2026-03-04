# AI Call Conversation Latency Optimization - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reduce AI call latency — initial greeting from ~5.8s to ~3.0-3.5s, turn-taking from ~1.5-1.7s to ~0.9-1.2s — through platform-side optimizations only.

**Architecture:** Five independent optimizations targeting VAD silence wait (A), platform system prompt size (B), sequential team member service creation (C), sequential Go RPCs (D1), and sequential Asterisk WS + Python init (D2). Changes span bin-pipecat-manager (Go + Python) and bin-ai-manager (Go).

**Tech Stack:** Go (errgroup, gorilla/websocket), Python (Pydantic, asyncio, pipecat-ai), RabbitMQ RPC

**Design doc:** `docs/plans/2026-03-04-optimize-aicall-conversation-latency-design.md`

---

### Task 1: VAD `stop_secs` Configuration (Optimization A)

Make the VAD silence detection threshold configurable instead of hardcoded at 0.8s. Default to 0.5s with a floor guard at 0.3s.

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/main.py:77-88` (PipelineRequest model)
- Modify: `bin-pipecat-manager/scripts/pipecat/run.py:128` (init_single_ai_pipeline VAD)
- Modify: `bin-pipecat-manager/scripts/pipecat/run.py:56-68` (init_pipeline signature)
- Modify: `bin-pipecat-manager/scripts/pipecat/run.py:527` (init_team_pipeline VAD)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/pythonrunner.go:38-53` (PythonRunner interface)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/pythonrunner.go:60-103` (Start method + request body)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:83-98` (pythonRunner.Start call)

**Step 1: Add `vad_stop_secs` to Python PipelineRequest model**

In `bin-pipecat-manager/scripts/pipecat/main.py`, add the field to `PipelineRequest`:

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
    vad_stop_secs: float = 0.5
```

**Step 2: Thread `vad_stop_secs` through the `/run` endpoint to `init_pipeline`**

In `main.py`, update the `init_pipeline` call (line 119) to pass the new field:

```python
        ctx = await init_pipeline(
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
            vad_stop_secs=req.vad_stop_secs,
        )
```

**Step 3: Update `init_pipeline` signature in `run.py`**

```python
async def init_pipeline(
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
    vad_stop_secs: float = 0.5,
) -> dict:
    """Initialize the pipeline. Returns context dict. Raises on failure."""
    if resolved_team:
        ctx = await init_team_pipeline(
            id, resolved_team,
            stt_language=stt_language,
            tts_language=tts_language,
            llm_messages=llm_messages,
            vad_stop_secs=vad_stop_secs,
        )
        ctx["type"] = "team"
        return ctx
    else:
        ctx = await init_single_ai_pipeline(
            id, llm_type, llm_key, llm_messages,
            stt_type, stt_language, tts_type, tts_language,
            tts_voice_id, tools_data,
            vad_stop_secs=vad_stop_secs,
        )
        ctx["type"] = "single"
        return ctx
```

**Step 4: Update `init_single_ai_pipeline` to use `vad_stop_secs`**

Add `vad_stop_secs: float = 0.5` parameter to `init_single_ai_pipeline` signature (line 97). Replace the hardcoded `stop_secs=0.8` at line 128 with:

```python
async def init_single_ai_pipeline(
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
    vad_stop_secs: float = 0.5,
) -> dict:
```

At line 128, change:
```python
            vad_analyzer = SileroVADAnalyzer(params=VADParams(stop_secs=max(vad_stop_secs, 0.3)))
```

**Step 5: Update `init_team_pipeline` to use `vad_stop_secs`**

Add `vad_stop_secs: float = 0.5` parameter to `init_team_pipeline` signature (line 453). Replace the hardcoded `stop_secs=0.8` at line 527 with:

```python
async def init_team_pipeline(
    id: str,
    resolved_team: dict,
    stt_language: str = None,
    tts_language: str = None,
    llm_messages: list = None,
    vad_stop_secs: float = 0.5,
) -> dict:
```

At line 527, change:
```python
        vad_analyzer = SileroVADAnalyzer(params=VADParams(stop_secs=max(vad_stop_secs, 0.3)))
```

**Step 6: Add `VADStopSecs` to Go PythonRunner interface and Start method**

In `bin-pipecat-manager/pkg/pipecatcallhandler/pythonrunner.go`:

Update the `PythonRunner` interface (add `vadStopSecs float64` param):
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
		vadStopSecs float64,
	) error
	Stop(ctx context.Context, pipecatcallID uuid.UUID) error
}
```

Update the `Start` method signature to match, and add `VADStopSecs` to the request body struct:

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
	vadStopSecs float64,
) error {
	// ...

	reqBody := struct {
		ID           uuid.UUID         `json:"id,omitempty"`
		LLMType      string            `json:"llm_type,omitempty"`
		LLMKey       string            `json:"llm_key,omitempty"`
		LLMMessages  []map[string]any  `json:"llm_messages,omitempty"`
		STTType      string            `json:"stt_type,omitempty"`
		STTLanguage  string            `json:"stt_language,omitempty"`
		TTSType      string            `json:"tts_type,omitempty"`
		TTSLanguage  string            `json:"tts_language,omitempty"`
		TTSVoiceID   string            `json:"tts_voice_id,omitempty"`
		Tools        []aitool.Tool     `json:"tools,omitempty"`
		ResolvedTeam *resolvedTeamData `json:"resolved_team,omitempty"`
		VADStopSecs  float64           `json:"vad_stop_secs,omitempty"`
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
		VADStopSecs:  vadStopSecs,
	}
	// ... rest unchanged
```

**Step 7: Update `runnerStartScript` call site in `runner.go`**

In `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` line 83-98, add the `vadStopSecs` argument (default 0.5):

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
		0.5, // VAD stop_secs default
	); errStart != nil {
```

**Step 8: Regenerate mocks and run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-optimize-aicall-conversation-latency/bin-pipecat-manager
go generate ./...
go test ./...
golangci-lint run -v --timeout 5m
```

Expected: All tests pass after updating mock call sites to include the new `vadStopSecs` parameter.

**Step 9: Commit**

```bash
git add bin-pipecat-manager/scripts/pipecat/main.py bin-pipecat-manager/scripts/pipecat/run.py bin-pipecat-manager/pkg/pipecatcallhandler/pythonrunner.go bin-pipecat-manager/pkg/pipecatcallhandler/runner.go bin-pipecat-manager/pkg/pipecatcallhandler/mock_pythonrunner.go
git commit -m "NOJIRA-optimize-aicall-conversation-latency

Make VAD stop_secs configurable with 0.5s default (was hardcoded 0.8s).

- bin-pipecat-manager: Add vad_stop_secs field to PipelineRequest model
- bin-pipecat-manager: Thread vad_stop_secs through init_pipeline to both single and team pipelines
- bin-pipecat-manager: Add VADStopSecs to PythonRunner.Start interface and request body
- bin-pipecat-manager: Apply floor guard of 0.3s to prevent misconfiguration"
```

---

### Task 2: Compress Platform Base System Prompt (Optimization B)

Rewrite `defaultCommonAIcallSystemPrompt` from ~1,000 words to ~300 words while preserving all behavioral requirements.

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/main.go:200-271`

**Step 1: Replace the system prompt constant**

In `bin-ai-manager/pkg/aicallhandler/main.go`, replace `defaultCommonAIcallSystemPrompt` (lines 200-271) with:

```go
	defaultCommonAIcallSystemPrompt = `You are an AI assistant for VoIPBin. Follow the user's system/custom prompt strictly. Adapt to their persona, style, and tone.

Tool Usage:
- When a user requests an action you can perform with a tool, ACT IMMEDIATELY — invoke the tool right away.
- Do NOT describe what you could do or ask "would you like me to?". Just do it.
- Only ask for clarification if required parameters are genuinely missing.
- Use tool parameters exactly as specified.
- Never mention tool names, JSON, or backend logic to the user. Respond naturally: "I'll connect you now."

DTMF Events:
- Messages like "DTMF_EVENT: N" are telephone keypad presses. Treat as events, not text. Respond naturally per context.

Additional Data:
- You may receive JSON context data (call details, user profile, metadata). Use it to improve responses. Never expose, quote, or describe raw data to the user.

System/Tool Messages:
- If you receive messages with role "system" or "tool", or tool function responses, do not respond or react. Reference them silently unless explicitly instructed otherwise.

Response Rules:
- Ask clarifying questions one at a time, not all at once.
- Use tools or provided data for facts — avoid hallucinations.
- Maintain conversation continuity and prior context.
- Never expose raw JSON or tool responses to the user.`
```

**Step 2: Run ai-manager verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-optimize-aicall-conversation-latency/bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All tests pass. The constant is not tested directly — behavior verified via integration testing.

**Step 3: Commit**

```bash
git add bin-ai-manager/pkg/aicallhandler/main.go
git commit -m "NOJIRA-optimize-aicall-conversation-latency

Compress platform base system prompt from ~1000 to ~300 words.

- bin-ai-manager: Rewrite defaultCommonAIcallSystemPrompt preserving all behavioral requirements
- bin-ai-manager: Remove redundant instructions (no-JSON stated 3x, etc.)
- bin-ai-manager: Expect ~50-100ms savings per LLM call from fewer input tokens"
```

---

### Task 3: Parallelize Team Member Service Creation (Optimization C)

Replace sequential for-loop creating per-member LLM/TTS/STT services with parallel `asyncio.gather`.

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/run.py:470-489` (init_team_pipeline Step 1)

**Step 1: Replace sequential service creation with parallel asyncio.gather**

In `bin-pipecat-manager/scripts/pipecat/run.py`, replace the sequential for-loop in `init_team_pipeline` (lines 475-489):

```python
    # --- Step 1: Create per-member service instances (parallel) ---
    llm_services = {}
    tts_services = {}
    stt_services = {}

    async def _init_member_services(member):
        mid = member["id"]
        ai = member["ai"]
        start = time.monotonic()

        llm_svc, _ = await asyncio.to_thread(
            create_llm_service, ai["engine_model"], ai["engine_key"], [], []
        )

        tts_svc = None
        if ai.get("tts_type"):
            tts_svc = await asyncio.to_thread(
                create_tts_service, ai["tts_type"],
                voice_id=ai.get("tts_voice_id"), language=tts_language,
            )

        stt_svc = None
        if ai.get("stt_type"):
            stt_svc = await asyncio.to_thread(
                create_stt_service, ai["stt_type"], language=stt_language,
            )

        logger.info(f"[TEAM][INIT] Member {mid} services created in {time.monotonic() - start:.3f}s")
        return mid, llm_svc, tts_svc, stt_svc

    member_results = await asyncio.gather(*[
        _init_member_services(m) for m in members
    ])

    for mid, llm_svc, tts_svc, stt_svc in member_results:
        llm_services[mid] = llm_svc
        if tts_svc:
            tts_services[mid] = tts_svc
        if stt_svc:
            stt_services[mid] = stt_svc
```

The rest of init_team_pipeline (Step 2 onwards: routing services, context aggregator, transports, pipeline, FlowManager) remains unchanged. All services are fully populated before any of that code runs, so lifecycle is preserved.

**Step 2: Run pipecat-manager verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-optimize-aicall-conversation-latency/bin-pipecat-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All Go tests pass (Python changes don't affect Go tests).

**Step 3: Commit**

```bash
git add bin-pipecat-manager/scripts/pipecat/run.py
git commit -m "NOJIRA-optimize-aicall-conversation-latency

Parallelize team member service creation with asyncio.gather.

- bin-pipecat-manager: Replace sequential for-loop with parallel asyncio.gather in init_team_pipeline
- bin-pipecat-manager: Each member's LLM/TTS/STT created concurrently via asyncio.to_thread
- bin-pipecat-manager: All services fully populated before FlowManager.initialize() — no lifecycle change
- bin-pipecat-manager: Expect ~0.8-0.9s savings on initial greeting for team pipelines"
```

---

### Task 4: Parallelize Independent Go RPCs (Optimization D1)

Run `AIV1AIcallGet` and `runGetLLMKey` concurrently in `startReferenceTypeAIcall` using `errgroup`.

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/start.go:156-207` (startReferenceTypeAIcall)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/run.go:103-123` (runGetLLMKey — change to return error)
- Test: `bin-pipecat-manager/pkg/pipecatcallhandler/start_test.go` (existing tests)

**Step 1: Write the failing test for parallel RPC execution**

Add a test to `start_test.go` that verifies both RPCs are called when `startReferenceTypeAIcall` is invoked for a `ReferenceTypeCall` aicall:

```go
func Test_startReferenceTypeAIcall_parallelRPCs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &pipecatcallHandler{
		requestHandler:        mockReq,
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
	}

	pcID := uuid.FromStringOrNil("a1b2c3d4-1111-2222-3333-444455556666")
	referenceID := uuid.FromStringOrNil("b2c3d4e5-1111-2222-3333-444455556666")
	aicallID := uuid.FromStringOrNil("c3d4e5f6-1111-2222-3333-444455556666")
	aiID := uuid.FromStringOrNil("d4e5f6a7-1111-2222-3333-444455556666")

	pc := &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{
			ID:         pcID,
			CustomerID: uuid.FromStringOrNil("e5f6a7b8-1111-2222-3333-444455556666"),
		},
		ReferenceType: pipecatcall.ReferenceTypeAICall,
		ReferenceID:   referenceID,
	}

	// AIV1AIcallGet — called by both the parallel RPC and runGetLLMKey
	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).
		Return(&amaicall.AIcall{
			Identity: commonidentity.Identity{ID: aicallID},
			ReferenceType: amaicall.ReferenceTypeCall,
			ReferenceID:   uuid.FromStringOrNil("f6a7b8c9-1111-2222-3333-444455556666"),
			AssistanceType: amaicall.AssistanceTypeAI,
			AssistanceID:   aiID,
		}, nil).Times(2)

	// AIV1AIGet — called by runGetLLMKey -> resolveAIFromAIcall
	mockReq.EXPECT().AIV1AIGet(gomock.Any(), aiID).
		Return(&amai.AI{
			Identity:  commonidentity.Identity{ID: aiID},
			EngineKey: "test-key-123",
		}, nil)

	// ExternalMediaStart fails so we don't need to set up WS mocks
	mockReq.EXPECT().CallV1ExternalMediaStart(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return(nil, fmt.Errorf("external media error"))

	err := h.startReferenceTypeAIcall(context.Background(), pc)
	if err == nil {
		t.Fatal("expected error from ExternalMediaStart but got nil")
	}
}
```

**Step 2: Run the test to verify it fails**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-optimize-aicall-conversation-latency/bin-pipecat-manager
go test -v ./pkg/pipecatcallhandler/ -run Test_startReferenceTypeAIcall_parallelRPCs
```

Expected: FAIL — the current code calls `AIV1AIcallGet` only once, and `runGetLLMKey` calls it again separately later. The `.Times(2)` expectation won't match the current sequential code where `runGetLLMKey` is called after ExternalMediaStart.

**Step 3: Refactor `startReferenceTypeAIcall` to parallelize RPCs**

In `bin-pipecat-manager/pkg/pipecatcallhandler/start.go`, add `"golang.org/x/sync/errgroup"` to imports, then replace the `ReferenceTypeCall` case in `startReferenceTypeAIcall` (lines 162-233):

```go
	case amaicall.ReferenceTypeCall:
		// Parallel Phase: fetch AIcall info and LLM key concurrently
		g, gctx := errgroup.WithContext(ctx)
		var c *amaicall.AIcall
		var llmKey string

		g.Go(func() error {
			var errGet error
			c, errGet = h.requestHandler.AIV1AIcallGet(gctx, pc.ReferenceID)
			if errGet != nil {
				return fmt.Errorf("could not get ai call info: %w", errGet)
			}
			log.WithField("ai_call", c).Info("Retrieved ai call info. ai_call_id: ", c.ID)
			return nil
		})
		g.Go(func() error {
			llmKey = h.runGetLLMKey(gctx, pc)
			return nil // runGetLLMKey logs errors internally and returns ""
		})
		if err := g.Wait(); err != nil {
			return errors.Wrap(err, "parallel RPC phase failed")
		}

		// start the external media
		em, err := h.requestHandler.CallV1ExternalMediaStart(
			ctx,
			pc.ID,
			cmexternalmedia.ReferenceTypeCall,
			c.ReferenceID,
			"INCOMING",
			defaultEncapsulation,
			defaultTransport,
			"", // transportData
			defaultConnectionType,
			defaultFormat,
			cmexternalmedia.DirectionIn,
			cmexternalmedia.DirectionOut,
		)
		if err != nil {
			return errors.Wrapf(err, "could not create external media")
		}
		log.WithField("external_media", em).Info("Created external media. external_media_id: ", em.ID)

		// Connect to Asterisk via WebSocket
		conn, err := h.websocketAsteriskConnect(ctx, em.MediaURI)
		if err != nil {
			log.Errorf("Could not connect WebSocket to Asterisk. err: %v", err)
			if _, errStop := h.requestHandler.CallV1ExternalMediaStop(ctx, em.ID); errStop != nil {
				log.Errorf("Could not stop orphaned external media. err: %v", errStop)
			}
			return errors.Wrapf(err, "could not connect to asterisk websocket")
		}
		log.Debugf("WebSocket connected to Asterisk. media_uri: %s", em.MediaURI)

		connAstDone := make(chan struct{})

		se, err := h.SessionCreate(pc, pc.ID, conn, connAstDone, llmKey)
		if err != nil {
			_ = conn.Close()
			return errors.Wrapf(err, "could not create pipecatcall session")
		}

		// Start pipecat runner
		go func() {
			defer se.Cancel()
			h.RunnerStart(pc, se)
		}()

		// Start media handler
		go func() {
			defer se.Cancel()
			h.runAsteriskReceivedMediaHandle(se)
		}()

		// Monitor lifecycle
		go func() {
			select {
			case <-se.Ctx.Done():
			case <-connAstDone:
			}
			log.Debugf("Asterisk connection or context done, terminating. pipecatcall_id: %s", pc.ID)
			h.terminate(context.Background(), pc)
		}()

		return nil
```

Also add `"fmt"` to imports if not already present, and add `"golang.org/x/sync/errgroup"` to imports.

**Step 4: Run the test to verify it passes**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-optimize-aicall-conversation-latency/bin-pipecat-manager
go test -v ./pkg/pipecatcallhandler/ -run Test_startReferenceTypeAIcall_parallelRPCs
```

Expected: PASS

**Step 5: Run full verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-optimize-aicall-conversation-latency/bin-pipecat-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All tests pass.

**Step 6: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/start.go bin-pipecat-manager/pkg/pipecatcallhandler/start_test.go
git commit -m "NOJIRA-optimize-aicall-conversation-latency

Parallelize AIcallGet and LLM key fetch with errgroup.

- bin-pipecat-manager: Run AIV1AIcallGet and runGetLLMKey concurrently in startReferenceTypeAIcall
- bin-pipecat-manager: Add test for parallel RPC execution
- bin-pipecat-manager: Expect ~50-100ms savings on initial greeting"
```

---

### Task 5: Split Session + Parallel Asterisk/Python Init (Optimization D2)

Allow Python pipeline init to start before Asterisk WebSocket is connected by splitting session creation from the Asterisk connection. Python init (~2.1s) runs in parallel with external media creation + Asterisk WS connect (~1.0s).

**Files:**
- Modify: `bin-pipecat-manager/models/pipecatcall/session.go:12-35` (add ConnAstReady channel)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/session.go:14-54` (SessionCreate accepts nil ConnAst)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/session.go:75-106` (SessionStop handles ConnAstReady)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:501-523` (runnerWebsocketHandleAudio waits on ConnAstReady)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/run.go:52-101` (runAsteriskReceivedMediaHandle waits on ConnAstReady)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/start.go:156-233` (restructure startReferenceTypeAIcall)
- Test: `bin-pipecat-manager/pkg/pipecatcallhandler/session_test.go`
- Test: `bin-pipecat-manager/pkg/pipecatcallhandler/start_test.go`

**Step 1: Add `ConnAstReady` channel and `SetConnAst` method to Session model**

In `bin-pipecat-manager/models/pipecatcall/session.go`, add the `ConnAstReady` field and helper methods:

```go
type Session struct {
	identity.Identity // copied from pipecatcall

	PipecatcallReferenceType ReferenceType `json:"reference_type,omitempty"` // copied from pipecatcall
	PipecatcallReferenceID   uuid.UUID     `json:"reference_id,omitempty"`   // copied from pipecatcall

	Ctx    context.Context    `json:"-"`
	Cancel context.CancelFunc `json:"-"`

	// Runner info
	RunnerWebsocketChan chan *SessionFrame `json:"-"`

	// asterisk info
	AsteriskStreamingID uuid.UUID       `json:"-"`
	ConnAst             *websocket.Conn `json:"-"`
	ConnAstDone         chan struct{}    `json:"-"`
	ConnAstReady        chan struct{}    `json:"-"` // closed when ConnAst is set

	// llm
	LLMKey     string `json:"-"`
	LLMBotText string `json:"-"`

	// audio quality monitoring
	DroppedFrames atomic.Int64 `json:"-"`
}

// SetConnAst sets the Asterisk WebSocket connection and signals readiness.
// Must be called at most once. Panics on double-close of ConnAstReady.
func (s *Session) SetConnAst(conn *websocket.Conn, done chan struct{}) {
	s.ConnAst = conn
	s.ConnAstDone = done
	close(s.ConnAstReady)
}
```

Add `"github.com/gorilla/websocket"` to the import block in session.go if not already present.

**Step 2: Update SessionCreate to initialize ConnAstReady**

In `bin-pipecat-manager/pkg/pipecatcallhandler/session.go`, update `SessionCreate` to always create `ConnAstReady`, and close it immediately if `connAst` is already provided:

```go
func (h *pipecatcallHandler) SessionCreate(
	pc *pipecatcall.Pipecatcall,
	asteriskStreamingID uuid.UUID,
	connAst *websocket.Conn,
	connAstDone chan struct{},
	llmKey string,
) (*pipecatcall.Session, error) {

	ctx, cancel := context.WithCancel(context.Background())
	connAstReady := make(chan struct{})

	// If connection already provided, mark as ready immediately
	if connAst != nil {
		close(connAstReady)
	}

	res := &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID:         pc.ID,
			CustomerID: pc.CustomerID,
		},

		PipecatcallReferenceType: pc.ReferenceType,
		PipecatcallReferenceID:   pc.ReferenceID,

		Ctx:    ctx,
		Cancel: cancel,

		RunnerWebsocketChan: make(chan *pipecatcall.SessionFrame, defaultRunnerWebsocketChanBufferSize),

		AsteriskStreamingID: asteriskStreamingID,
		ConnAst:             connAst,
		ConnAstDone:         connAstDone,
		ConnAstReady:        connAstReady,

		LLMKey: llmKey,
	}

	h.muPipecatcallSession.Lock()
	defer h.muPipecatcallSession.Unlock()

	_, ok := h.mapPipecatcallSession[res.ID]
	if ok {
		return nil, fmt.Errorf("session already exists. session_id: %s", res.ID)
	}

	h.mapPipecatcallSession[res.ID] = res
	return res, nil
}
```

**Step 3: Guard audio output — wait for ConnAstReady before writing**

In `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go`, update `runnerWebsocketHandleAudio` (line 501) to wait for the Asterisk connection:

```go
func (h *pipecatcallHandler) runnerWebsocketHandleAudio(se *pipecatcall.Session, sampleRate int, numChannels int, data []byte) error {
	if numChannels != 1 {
		return errors.Errorf("only mono audio is supported. num_channels: %d", numChannels)
	}

	audioData := data
	if sampleRate != defaultMediaSampleRate {
		var err error
		audioData, err = h.audiosocketHandler.GetDataSamples(sampleRate, data)
		if err != nil {
			return errors.Wrapf(err, "could not resample audio data")
		}
	}

	if len(audioData) == 0 {
		return nil
	}

	// Wait for Asterisk connection to be ready before writing audio
	select {
	case <-se.ConnAstReady:
	case <-se.Ctx.Done():
		return nil
	}

	if se.ConnAst == nil {
		return nil
	}

	if err := h.websocketHandler.WriteMessage(se.ConnAst, websocket.BinaryMessage, audioData); err != nil {
		return errors.Wrapf(err, "could not write audio data to asterisk websocket")
	}

	return nil
}
```

**Step 4: Guard Asterisk media reader — wait for ConnAstReady**

In `bin-pipecat-manager/pkg/pipecatcallhandler/run.go`, update `runAsteriskReceivedMediaHandle` to wait for ConnAstReady at the top:

```go
func (h *pipecatcallHandler) runAsteriskReceivedMediaHandle(se *pipecatcall.Session) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "runAsteriskReceivedMediaHandle",
		"pipecatcall_id": se.ID,
	})

	// Wait for Asterisk connection to be established
	select {
	case <-se.ConnAstReady:
	case <-se.Ctx.Done():
		log.Debugf("Context cancelled while waiting for Asterisk connection.")
		return
	}

	if se.ConnAst == nil {
		log.Debugf("No Asterisk WebSocket connection, skipping media handle.")
		return
	}

	// ... rest unchanged from line 63 onwards
```

**Step 5: Restructure `startReferenceTypeAIcall` for parallel Asterisk + Python init**

In `bin-pipecat-manager/pkg/pipecatcallhandler/start.go`, the `ReferenceTypeCall` case becomes:

```go
	case amaicall.ReferenceTypeCall:
		// Parallel Phase 1: fetch AIcall info and LLM key concurrently
		g, gctx := errgroup.WithContext(ctx)
		var c *amaicall.AIcall
		var llmKey string

		g.Go(func() error {
			var errGet error
			c, errGet = h.requestHandler.AIV1AIcallGet(gctx, pc.ReferenceID)
			if errGet != nil {
				return fmt.Errorf("could not get ai call info: %w", errGet)
			}
			log.WithField("ai_call", c).Info("Retrieved ai call info. ai_call_id: ", c.ID)
			return nil
		})
		g.Go(func() error {
			llmKey = h.runGetLLMKey(gctx, pc)
			return nil
		})
		if err := g.Wait(); err != nil {
			return errors.Wrap(err, "parallel RPC phase failed")
		}

		// Create session with nil Asterisk connection — Python runner can start immediately.
		// ConnAstReady channel will be closed when Asterisk WS connects.
		se, err := h.SessionCreate(pc, pc.ID, nil, nil, llmKey)
		if err != nil {
			return errors.Wrapf(err, "could not create pipecatcall session")
		}

		// Parallel Phase 2: Asterisk WS connect + Python runner init
		astErrCh := make(chan error, 1)
		go func() {
			// External media creation + Asterisk WS connect
			em, errEM := h.requestHandler.CallV1ExternalMediaStart(
				ctx,
				pc.ID,
				cmexternalmedia.ReferenceTypeCall,
				c.ReferenceID,
				"INCOMING",
				defaultEncapsulation,
				defaultTransport,
				"",
				defaultConnectionType,
				defaultFormat,
				cmexternalmedia.DirectionIn,
				cmexternalmedia.DirectionOut,
			)
			if errEM != nil {
				astErrCh <- fmt.Errorf("could not create external media: %w", errEM)
				return
			}
			log.WithField("external_media", em).Info("Created external media. external_media_id: ", em.ID)

			conn, errConn := h.websocketAsteriskConnect(ctx, em.MediaURI)
			if errConn != nil {
				log.Errorf("Could not connect WebSocket to Asterisk. err: %v", errConn)
				if _, errStop := h.requestHandler.CallV1ExternalMediaStop(ctx, em.ID); errStop != nil {
					log.Errorf("Could not stop orphaned external media. err: %v", errStop)
				}
				astErrCh <- fmt.Errorf("could not connect to asterisk websocket: %w", errConn)
				return
			}
			log.Debugf("WebSocket connected to Asterisk. media_uri: %s", em.MediaURI)

			connAstDone := make(chan struct{})
			se.SetConnAst(conn, connAstDone)
			astErrCh <- nil
		}()

		// Start pipecat runner (runs in parallel with Asterisk setup above)
		go func() {
			defer se.Cancel()
			h.RunnerStart(pc, se)
		}()

		// Wait for Asterisk connection phase to complete
		if astErr := <-astErrCh; astErr != nil {
			se.Cancel()
			h.sessionDelete(pc.ID)
			return errors.Wrap(astErr, "asterisk setup failed")
		}

		// Start media handler (now that ConnAst is set)
		go func() {
			defer se.Cancel()
			h.runAsteriskReceivedMediaHandle(se)
		}()

		// Monitor lifecycle
		go func() {
			select {
			case <-se.Ctx.Done():
			case <-se.ConnAstDone:
			}
			log.Debugf("Asterisk connection or context done, terminating. pipecatcall_id: %s", pc.ID)
			h.terminate(context.Background(), pc)
		}()

		return nil
```

**Step 6: Update SessionStop to handle ConnAstReady**

In `bin-pipecat-manager/pkg/pipecatcallhandler/session.go`, update `SessionStop` to wait briefly for ConnAstReady before closing:

```go
func (h *pipecatcallHandler) SessionStop(id uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "SessionStop",
		"pipecatcall_id": id,
	})
	log.Debugf("Stopping pipecatcall session. pipecatcall_id: %s", id)

	pc, err := h.SessionGet(id)
	if err != nil {
		log.Errorf("Could not get pipecatcall session: %v", err)
		return
	}

	if dropped := pc.DroppedFrames.Load(); dropped > 0 {
		log.Warnf("Session had %d dropped audio frames. pipecatcall_id: %s", dropped, id)
	}

	// Wait for Asterisk connection to be ready before closing (or context done)
	select {
	case <-pc.ConnAstReady:
	case <-pc.Ctx.Done():
	}

	if pc.ConnAst != nil {
		if errClose := pc.ConnAst.Close(); errClose != nil {
			log.Errorf("Could not close the asterisk connection. err: %v", errClose)
		} else {
			log.Infof("Closed the asterisk connection.")
		}
	}

	h.sessionDelete(pc.ID)
	if errStop := h.pythonRunner.Stop(context.Background(), id); errStop != nil {
		log.Errorf("Could not stop the pipecatcall in python runner. err: %v", errStop)
	}

	log.Debugf("Stopped pipecatcall session. pipecatcall_id: %s", id)
}
```

**Step 7: Update `startReferenceTypeCall` to use same pattern**

The `startReferenceTypeCall` function (lines 78-154) should use the same parallel pattern for consistency. Update it similarly: parallel LLM key fetch, create session with nil conn, parallel Asterisk + Python init. (Same pattern as the AIcall case above, but simpler because it already has the call info.)

```go
func (h *pipecatcallHandler) startReferenceTypeCall(ctx context.Context, pc *pipecatcall.Pipecatcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "startReferenceTypeCall",
		"pipecatcall_id": pc.ID,
	})

	c, err := h.requestHandler.CallV1CallGet(ctx, pc.ReferenceID)
	if err != nil {
		return errors.Wrapf(err, "could not get call info")
	}
	log.WithField("call", c).Info("Retrieved call info. call_id: ", c.ID)

	llmKey := h.runGetLLMKey(ctx, pc)

	// Create session with nil Asterisk connection
	se, err := h.SessionCreate(pc, pc.ID, nil, nil, llmKey)
	if err != nil {
		return errors.Wrapf(err, "could not create pipecatcall session")
	}

	// Parallel: Asterisk WS connect + Python runner init
	astErrCh := make(chan error, 1)
	go func() {
		em, errEM := h.requestHandler.CallV1ExternalMediaStart(
			ctx,
			pc.ID,
			cmexternalmedia.ReferenceTypeCall,
			c.ID,
			"INCOMING",
			defaultEncapsulation,
			defaultTransport,
			"",
			defaultConnectionType,
			defaultFormat,
			cmexternalmedia.DirectionIn,
			cmexternalmedia.DirectionOut,
		)
		if errEM != nil {
			astErrCh <- fmt.Errorf("could not create external media: %w", errEM)
			return
		}
		log.WithField("external_media", em).Info("Created external media. external_media_id: ", em.ID)

		conn, errConn := h.websocketAsteriskConnect(ctx, em.MediaURI)
		if errConn != nil {
			log.Errorf("Could not connect WebSocket to Asterisk. err: %v", errConn)
			if _, errStop := h.requestHandler.CallV1ExternalMediaStop(ctx, em.ID); errStop != nil {
				log.Errorf("Could not stop orphaned external media. err: %v", errStop)
			}
			astErrCh <- fmt.Errorf("could not connect to asterisk websocket: %w", errConn)
			return
		}
		log.Debugf("WebSocket connected to Asterisk. media_uri: %s", em.MediaURI)

		connAstDone := make(chan struct{})
		se.SetConnAst(conn, connAstDone)
		astErrCh <- nil
	}()

	// Start pipecat runner (parallel with Asterisk setup)
	go func() {
		defer se.Cancel()
		h.RunnerStart(pc, se)
	}()

	// Wait for Asterisk connection
	if astErr := <-astErrCh; astErr != nil {
		se.Cancel()
		h.sessionDelete(pc.ID)
		return errors.Wrap(astErr, "asterisk setup failed")
	}

	// Start media handler
	go func() {
		defer se.Cancel()
		h.runAsteriskReceivedMediaHandle(se)
	}()

	// Monitor lifecycle
	go func() {
		select {
		case <-se.Ctx.Done():
		case <-se.ConnAstDone:
		}
		log.Debugf("Asterisk connection or context done, terminating. pipecatcall_id: %s", pc.ID)
		h.terminate(context.Background(), pc)
	}()

	return nil
}
```

**Step 8: Write test for SetConnAst**

Add to `session_test.go`:

```go
func TestSession_SetConnAst(t *testing.T) {
	se := &pipecatcall.Session{
		ConnAstReady: make(chan struct{}),
	}

	// ConnAstReady should not be closed yet
	select {
	case <-se.ConnAstReady:
		t.Fatal("ConnAstReady should not be closed before SetConnAst")
	default:
		// expected
	}

	done := make(chan struct{})
	se.SetConnAst(nil, done)

	// ConnAstReady should now be closed
	select {
	case <-se.ConnAstReady:
		// expected
	default:
		t.Fatal("ConnAstReady should be closed after SetConnAst")
	}
}
```

**Step 9: Run full verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-optimize-aicall-conversation-latency/bin-pipecat-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All tests pass. May need to update existing test expectations for `SessionCreate` calls and adjust mock expectations.

**Step 10: Commit**

```bash
git add bin-pipecat-manager/models/pipecatcall/session.go bin-pipecat-manager/pkg/pipecatcallhandler/session.go bin-pipecat-manager/pkg/pipecatcallhandler/start.go bin-pipecat-manager/pkg/pipecatcallhandler/start_test.go bin-pipecat-manager/pkg/pipecatcallhandler/session_test.go bin-pipecat-manager/pkg/pipecatcallhandler/runner.go bin-pipecat-manager/pkg/pipecatcallhandler/run.go
git commit -m "NOJIRA-optimize-aicall-conversation-latency

Parallelize Asterisk WS connect with Python pipeline init.

- bin-pipecat-manager: Add ConnAstReady channel and SetConnAst method to Session
- bin-pipecat-manager: SessionCreate accepts nil ConnAst, closes ConnAstReady if conn provided
- bin-pipecat-manager: runnerWebsocketHandleAudio waits on ConnAstReady before writing
- bin-pipecat-manager: runAsteriskReceivedMediaHandle waits on ConnAstReady before reading
- bin-pipecat-manager: Restructure startReferenceTypeAIcall and startReferenceTypeCall for parallel init
- bin-pipecat-manager: Python init (~2.1s) now runs alongside ExternalMedia+WS connect (~1.0s)
- bin-pipecat-manager: Expect ~1.0s savings on initial greeting"
```

---

### Task 6: Final Verification and Integration

Run all verification across both services and review for regressions.

**Step 1: Run bin-pipecat-manager full verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-optimize-aicall-conversation-latency/bin-pipecat-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Run bin-ai-manager full verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-optimize-aicall-conversation-latency/bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Review all changes**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-optimize-aicall-conversation-latency
git log --oneline
git diff main..HEAD --stat
```

Verify:
- No breaking changes to existing behavior
- All optimizations are platform-side only
- No customer-facing API changes
- errgroup import added to go.mod if needed

**Step 4: Push and create PR**

```bash
git push -u origin NOJIRA-optimize-aicall-conversation-latency
```

Then create PR with summary of all 5 optimizations and expected latency improvements.

---

## Expected Total Savings

| Optimization | Initial Greeting | Turn-Taking |
|-------------|-----------------|-------------|
| A: VAD 0.8→0.5s | — | -300ms |
| B: Prompt compression | -50-100ms | -50-100ms |
| C: Parallel member init | -800-900ms | — |
| D1: Parallel RPCs | -50-100ms | — |
| D2: Parallel Asterisk/Python | -1000ms | — |
| **Total** | **~1.9-2.1s** | **~350-400ms** |

**Projected results:**
- Initial greeting: ~5.8s → ~3.7-3.9s (target: 3.0-3.5s)
- Turn-taking: ~1.5-1.7s → ~1.1-1.3s (target: 0.9-1.2s)
