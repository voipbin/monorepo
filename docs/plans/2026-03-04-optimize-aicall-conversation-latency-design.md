# Design: AI Call Conversation Latency Optimization

## Problem Statement

AI call conversations have excessive latency in two areas:
- **Initial greeting:** ~5.8 seconds from call creation to user hearing AI voice
- **Turn-taking:** ~1.5-1.7 seconds from user finishing speaking to hearing AI response

Measured from production call `b58fc6eb-d92f-4070-9d97-a52116a1c2ca` (Gemini 2.5 Flash + Google STT/TTS + team pipeline with 3 members).

## Target

- Initial greeting: ~3.0-3.5 seconds
- Turn-taking: ~0.9-1.2 seconds

## Constraints

- LLM model, customer prompt, TTS/STT choices are customer-controlled and must not be changed
- All optimizations must be platform-side
- No breaking changes to existing behavior

## Current Latency Breakdown

### Initial Greeting (~5.8s)

| Phase | Duration | Service | Function |
|-------|----------|---------|----------|
| AI-manager -> pipecat-manager RPC | ~0.5s | ai-manager | `startPipecatcall()` |
| External media creation | ~1.0s | call-manager | `CallV1ExternalMediaStart()` |
| Asterisk WS connect | ~2ms | pipecat-manager | `websocketAsteriskConnect()` |
| LLM key fetch + team resolve | ~0.15s | pipecat-manager | `runGetLLMKey()`, `resolveTeamForPython()` |
| **Python pipeline init (3x LLM/TTS/STT)** | **~2.1s** | pipecat-manager | `init_team_pipeline()` |
| LLM TTFB (greeting generation) | ~0.9s | Gemini API | `GoogleLLMService` |
| TTS TTFB (first sentence) | ~0.4s | Google TTS API | `GoogleTTSService` |
| Overhead (WS connects, events) | ~0.7s | various | various |

### Turn-Taking (~1.5-1.7s)

| Phase | Turn 1 | Turn 2 | Turn 3 |
|-------|--------|--------|--------|
| VAD silence wait (`stop_secs=0.8`) | 0.80s | 0.80s | 0.80s |
| LLM TTFB (Gemini 2.5 Flash) | 0.69s | 0.50s | 0.59s |
| TTS TTFB (Google TTS) | 0.19s | 0.18s | N/A |
| **Total perceived** | **~1.7s** | **~1.5s** | **~1.4s** |

## Optimizations

### A. VAD `stop_secs` Configuration (0.8s -> 0.5s default)

**Problem:** VAD silence detection threshold is hardcoded at 0.8s in two places, adding a mandatory 800ms wait after every user utterance.

**Current code locations:**
- `bin-pipecat-manager/scripts/pipecat/run.py` line ~128 (`init_single_ai_pipeline`)
- `bin-pipecat-manager/scripts/pipecat/run.py` line ~527 (`init_team_pipeline`)

**Changes:**

1. **Python** - Add `vad_stop_secs` field to `PipelineRequest` in `main.py`:
   ```python
   class PipelineRequest(BaseModel):
       # ... existing fields ...
       vad_stop_secs: float = 0.5
   ```
   Thread through `init_pipeline()` / `init_team_pipeline()` signatures:
   ```python
   vad_analyzer = SileroVADAnalyzer(params=VADParams(
       stop_secs=max(req.vad_stop_secs, 0.3)  # floor guard
   ))
   ```

2. **Go** - Add `VADStopSecs` to Python runner request in `pythonrunner.go`:
   ```go
   type startRequest struct {
       // ... existing fields ...
       VADStopSecs float64 `json:"vad_stop_secs,omitempty"`
   }
   ```
   Default to 0.5 if not set. Optionally expose in AI model config for customer tuning later.

**Files changed:**
- `bin-pipecat-manager/scripts/pipecat/run.py`
- `bin-pipecat-manager/scripts/pipecat/main.py`
- `bin-pipecat-manager/pkg/pipecatcallhandler/pythonrunner.go`

**Expected savings:** ~300ms per turn.

**Risk:** Lower stop_secs may occasionally cut off slow speakers. 0.5s is industry standard (Vapi, Retell). Floor guard at 0.3s prevents misconfiguration.

---

### B. Compress Platform Base System Prompt

**Problem:** The platform-generated base system prompt (`defaultCommonAIcallSystemPrompt`) in `bin-ai-manager/pkg/aicallhandler/main.go` (line 200) is ~1,000 words / ~1,500 tokens. This prompt is prepended to EVERY AI call regardless of customer configuration, adding unnecessary LLM processing time.

**Analysis:** The prompt contains redundant instructions (e.g., "never expose JSON" stated 3+ times) and verbose explanations that can be compressed without losing behavioral requirements.

**Changes:** Rewrite the constant from ~1,000 words to ~300 words, preserving:
- Role definition (AI assistant for VoIPBin)
- Tool usage rules (detect tool, generate function call)
- Response guidelines (no JSON, respond naturally)
- DTMF event handling instructions
- Additional data interpretation rules (JSON context data)
- "Action Over Talk" directive
- System/tool message suppression rule

**Files changed:**
- `bin-ai-manager/pkg/aicallhandler/main.go` (line 200 `defaultCommonAIcallSystemPrompt`)

**Expected savings:** ~50-100ms per LLM call (fewer input tokens to process).

**Risk:** Must regression-test with actual calls to verify AI behavior is preserved. Key behaviors to test: DTMF handling, tool execution, response naturalness, data privacy.

---

### C. Parallelize Team Member Service Creation

**Problem:** In `init_team_pipeline()` (run.py lines 475-488), all team member services are created in a **sequential for-loop**:
```python
for member in members:        # 3 members = 3 iterations
    llm_svc = create_llm_service(...)   # sync, blocking
    tts_svc = create_tts_service(...)   # sync, blocking
    stt_svc = create_stt_service(...)   # sync, blocking
```
This takes ~1.3s for 3 members. Each member's service creation is independent.

**Why not lazy-init:** Review identified critical issues:
- `FlowManager.initialize()` calls `register_function` on ALL member LLM services at init time
- `RoutingLLMService.set_active_member()` validates member_id exists in `_services` dict
- Pipeline `StartFrame` is sent to ALL services at startup
- `build_team_flow()` builds ALL transition handlers upfront

**Changes:** Replace sequential loop with parallel `asyncio.gather`:

```python
async def init_member_services(member, ai_data, ...):
    mid = member["id"]
    ai = ai_data[mid]
    llm = await asyncio.to_thread(create_llm_service, ai, ...)
    tts = await asyncio.to_thread(create_tts_service, ai, ...) if ai.get("tts_type") else None
    stt = await asyncio.to_thread(create_stt_service, ai, ...) if ai.get("stt_type") else None
    return mid, llm, tts, stt

results = await asyncio.gather(*[
    init_member_services(m, ai_data, ...) for m in members
])
for mid, llm, tts, stt in results:
    llm_services[mid] = llm
    if tts: tts_services[mid] = tts
    if stt: stt_services[mid] = stt
```

All services still exist before `FlowManager.initialize()`, avoiding all lifecycle issues.

**Files changed:**
- `bin-pipecat-manager/scripts/pipecat/run.py` (`init_team_pipeline` function)

**Expected savings:** ~0.8-0.9s on initial greeting (parallel init of 3 members instead of sequential).

**Risk:** Low. All services are fully initialized before any pipeline logic runs, just as before. Only the creation order changes from sequential to parallel.

---

### D. Parallelize Go Setup Steps

#### D1: Parallelize Independent RPCs (Easy)

**Problem:** `startReferenceTypeAIcall()` in `start.go` runs `AIV1AIcallGet()` and `runGetLLMKey()` sequentially, but they are independent RPCs (~50ms each).

**Changes:** Use `errgroup` to run them concurrently:
```go
g, gctx := errgroup.WithContext(ctx)
var c *aicall.AIcall
var llmKey string

g.Go(func() error {
    var err error
    c, err = h.reqHandler.AIV1AIcallGet(gctx, pc.ReferenceID)
    return err
})
g.Go(func() error {
    var err error
    llmKey, err = runGetLLMKey(gctx, pc)
    return err
})
if err := g.Wait(); err != nil {
    return fmt.Errorf("failed to fetch aicall/llm key: %w", err)
}
```

**Files changed:**
- `bin-pipecat-manager/pkg/pipecatcallhandler/start.go`

**Expected savings:** ~50-100ms on initial greeting.

**Risk:** Minimal. Both RPCs are read-only and independent.

#### D2: Split Session Creation + Parallel Asterisk/Python Init (Medium)

**Problem:** The current flow is strictly sequential:
```
ExternalMediaStart (~1.0s) → AsteriskWSConnect (~2ms) → SessionCreate → RunnerStart (~2.1s Python init)
```
Python init doesn't need the Asterisk WS connection. It only needs the session to exist (for Python to connect back via WebSocket).

**Changes:**

1. **Modify `SessionCreate`** to accept nil `ConnAst`:
   - Add `ConnAstReady chan struct{}` field to Session
   - Add `SetConnAst(conn *websocket.Conn, done chan struct{})` method

2. **Restructure `startReferenceTypeAIcall`:**
   ```
   goroutine A: ExternalMediaStart → AsteriskWSConnect → session.SetConnAst(conn, done) → close(ConnAstReady)
   goroutine B: SessionCreate(nil conn) → RunnerStart(HTTP POST to Python, ~2.1s)
   After both: start runAsteriskReceivedMediaHandle (waits on ConnAstReady)
   ```

3. **Guard audio output:** In `RunnerWebsocketHandleOutput`, the audio writer (`runnerWebsocketHandleAudio`) must wait on `session.ConnAstReady` before writing TTS audio to the Asterisk connection:
   ```go
   func (h *handler) runnerWebsocketHandleAudio(se *Session, data []byte) {
       <-se.ConnAstReady  // wait for Asterisk connection
       se.ConnAst.WriteMessage(websocket.BinaryMessage, data)
   }
   ```

**Timing safety:** External media takes ~1.0s. Python init takes ~2.1s. Python starts producing audio only after init completes. So the Asterisk connection has ~1.1s margin before any audio output is needed.

**Files changed:**
- `bin-pipecat-manager/pkg/pipecatcallhandler/start.go`
- `bin-pipecat-manager/pkg/pipecatcallhandler/session.go` (or equivalent session management)
- `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` (audio output guard)

**Expected savings:** ~1.0s on initial greeting.

**Risk:** Medium. Requires careful synchronization. If external media creation fails while Python is already initializing, context cancellation propagates to both goroutines. The `ConnAstReady` channel prevents nil pointer writes.

---

## Implementation Order

1. **A** (VAD tuning) - Simplest, immediate turn-taking improvement
2. **B** (Prompt compression) - Single constant change, needs A/B testing
3. **C** (Parallel member init) - Biggest greeting improvement with lowest risk
4. **D1** (Parallel RPCs) - Small but easy win
5. **D2** (Split session) - Largest single improvement, most complex

## Files Changed Summary

| File | Optimizations |
|------|---------------|
| `bin-pipecat-manager/scripts/pipecat/run.py` | A, C |
| `bin-pipecat-manager/scripts/pipecat/main.py` | A |
| `bin-pipecat-manager/pkg/pipecatcallhandler/pythonrunner.go` | A |
| `bin-pipecat-manager/pkg/pipecatcallhandler/start.go` | D1, D2 |
| `bin-pipecat-manager/pkg/pipecatcallhandler/session.go` | D2 |
| `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` | D2 |
| `bin-ai-manager/pkg/aicallhandler/main.go` | B |

## Verification Plan

- **A:** Make test call, measure turn-taking latency from RTVI metrics. Verify no speech cut-offs.
- **B:** Make test call with same prompt, verify AI behavior (tool usage, DTMF, response naturalness). Compare TTFB metrics.
- **C:** Make test call with team pipeline, compare `init_team_pipeline` duration from Python logs.
- **D1/D2:** Make test call, measure time from pipecatcall creation to `pipecatcall_initialized` event.
- **Overall:** Compare initial greeting time and turn-taking latency against the baseline measurements documented above.
