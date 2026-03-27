# Fix pipecat-manager Memory Leak — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix per-call memory leak in pipecat-manager caused by Python STT/TTS/LLM services never being cleaned up, plus minor Go-side session leak and dead code removal.

**Architecture:** Python `run.py` creates service instances per pipeline but only cleans transports in `finally` blocks. The fix passes service references through the `ctx` dict and adds `cleanup()` calls in both single and team pipeline `finally` blocks. Go-side: wrap a bare `RunnerStart` goroutine with `defer se.Cancel()` and remove dead `runWebSocketAsteriskRead`.

**Tech Stack:** Python 3 (asyncio, pipecat-ai), Go 1.22+ (gorilla/websocket, gomock)

---

### Task 1: Add service cleanup to single AI pipeline (Python)

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/run.py:287-293` (init return dict)
- Modify: `bin-pipecat-manager/scripts/pipecat/run.py:296-323` (execute finally block)

**Step 1: Add stt_service, tts_service to the ctx dict returned by `init_single_ai_pipeline`**

In `init_single_ai_pipeline`, the return dict at line 287 currently returns:
```python
return {
    "task": task,
    "transport_input": transport_input,
    "transport_output": transport_output,
    "llm_service": llm_service,
    "tool_names": tool_names,
}
```

Change it to:
```python
return {
    "task": task,
    "transport_input": transport_input,
    "transport_output": transport_output,
    "llm_service": llm_service,
    "tts_service": tts_service,
    "stt_service": stt_service,
    "tool_names": tool_names,
}
```

Note: `stt_service` may be `None` if no STT was configured. `tts_service` is always created. `llm_service` is already in the dict.

**Step 2: Add cleanup calls in `execute_single_ai_pipeline` finally block**

In `execute_single_ai_pipeline` (line 296), extract the new services from ctx and add cleanup in the finally block.

Change the function to:
```python
async def execute_single_ai_pipeline(id: str, ctx: dict):
    """Run the single AI pipeline loop and cleanup. Runs as background task."""
    task = ctx["task"]
    transport_input = ctx["transport_input"]
    transport_output = ctx["transport_output"]
    llm_service = ctx["llm_service"]
    tts_service = ctx["tts_service"]
    stt_service = ctx["stt_service"]
    tool_names = ctx["tool_names"]

    try:
        runner = PipelineRunner()
        logger.info(f"[RUN] Starting pipeline id={id}")
        await runner.run(task)
    except asyncio.CancelledError:
        logger.info(f"[RUN] Pipeline cancelled. pipeline id={id}")
    except Exception as e:
        logger.error(f"[RUN] Pipeline error: {e}. pipeline id={id}")
    finally:
        logger.info(f"[CLEANUP] Cleaning up pipeline. pipeline id={id}")
        if task:
            await task.cancel()
        if transport_input:
            await transport_input.cleanup()
        if transport_output:
            await transport_output.cleanup()
        if stt_service:
            await stt_service.cleanup()
        if tts_service:
            await tts_service.cleanup()
        if llm_service:
            tool_unregister(llm_service, tool_names)
            await llm_service.cleanup()
        await task_manager.remove(id)
        logger.info(f"[CLEANUP] Pipeline cleaned. pipeline id={id}")
```

Key changes:
- Extract `tts_service` and `stt_service` from ctx
- Add `await stt_service.cleanup()` before tts cleanup
- Add `await tts_service.cleanup()` after stt cleanup
- Add `await llm_service.cleanup()` after tool_unregister (moved inside the `if llm_service` block)

**Step 3: Also add cleanup in the init exception handler**

In `init_single_ai_pipeline`'s except block (lines 274-282), add service cleanup:
```python
    except Exception:
        # Cleanup resources created after task_manager.add on init failure
        await task.cancel()
        if transport_input:
            await transport_input.cleanup()
        await transport_output.cleanup()
        if stt_service:
            await stt_service.cleanup()
        if tts_service:
            await tts_service.cleanup()
        if llm_service:
            tool_unregister(llm_service, tool_names)
            await llm_service.cleanup()
        await task_manager.remove(id)
        raise
```

**Step 4: Verify Python syntax**

Run: `cd bin-pipecat-manager/scripts/pipecat && python -c "import ast; ast.parse(open('run.py').read()); print('OK')"`
Expected: `OK`

**Step 5: Commit**

```bash
git add bin-pipecat-manager/scripts/pipecat/run.py
git commit -m "NOJIRA-Fix-pipecat-manager-memory-leak

- bin-pipecat-manager: Add STT/TTS/LLM service cleanup in single AI pipeline finally block
- bin-pipecat-manager: Pass stt_service and tts_service through ctx dict for cleanup access"
```

---

### Task 2: Add service cleanup to team pipeline (Python)

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/run.py:700-704` (init return dict)
- Modify: `bin-pipecat-manager/scripts/pipecat/run.py:707-730` (execute finally block)

**Step 1: Add routing services to the ctx dict returned by `init_team_pipeline`**

In `init_team_pipeline`, the return dict at line 700 currently returns:
```python
return {
    "task": task,
    "transport_input": transport_input,
    "transport_output": transport_output,
}
```

Change it to:
```python
return {
    "task": task,
    "transport_input": transport_input,
    "transport_output": transport_output,
    "routing_llm": routing_llm,
    "routing_tts": routing_tts,
    "routing_stt": routing_stt,
}
```

Note: `routing_tts` and `routing_stt` may be `None` if no TTS/STT services were configured. `routing_llm` is always created. Their `cleanup()` methods propagate to all member services.

**Step 2: Add cleanup calls in `execute_team_pipeline` finally block**

Change the function to:
```python
async def execute_team_pipeline(id: str, ctx: dict):
    """Run the team pipeline loop and cleanup. Runs as background task."""
    task = ctx["task"]
    transport_input = ctx["transport_input"]
    transport_output = ctx["transport_output"]
    routing_llm = ctx["routing_llm"]
    routing_tts = ctx["routing_tts"]
    routing_stt = ctx["routing_stt"]

    try:
        runner = PipelineRunner()
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
        if routing_stt:
            await routing_stt.cleanup()
        if routing_tts:
            await routing_tts.cleanup()
        if routing_llm:
            await routing_llm.cleanup()
        await task_manager.remove(id)
        logger.info(f"[TEAM][CLEANUP] Team pipeline cleaned. pipeline id={id}")
```

Key changes:
- Extract `routing_llm`, `routing_tts`, `routing_stt` from ctx
- Add `await routing_stt.cleanup()`, `await routing_tts.cleanup()`, `await routing_llm.cleanup()` in finally
- Each routing service's `cleanup()` propagates to ALL member services it wraps

**Step 3: Also add cleanup in the init exception handler**

In `init_team_pipeline`'s except block (lines 688-695), add routing service cleanup:
```python
    except Exception:
        # Cleanup resources created after task_manager.add on init failure
        await task.cancel()
        if transport_input:
            await transport_input.cleanup()
        await transport_output.cleanup()
        if routing_stt:
            await routing_stt.cleanup()
        if routing_tts:
            await routing_tts.cleanup()
        if routing_llm:
            await routing_llm.cleanup()
        await task_manager.remove(id)
        raise
```

**Step 4: Verify Python syntax**

Run: `cd bin-pipecat-manager/scripts/pipecat && python -c "import ast; ast.parse(open('run.py').read()); print('OK')"`
Expected: `OK`

**Step 5: Commit**

```bash
git add bin-pipecat-manager/scripts/pipecat/run.py
git commit -m "NOJIRA-Fix-pipecat-manager-memory-leak

- bin-pipecat-manager: Add routing service cleanup in team pipeline finally block
- bin-pipecat-manager: Pass routing_llm, routing_tts, routing_stt through ctx dict for cleanup access"
```

---

### Task 3: Fix missing defer se.Cancel() in Go start.go

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/start.go:276-283`

**Step 1: Wrap RunnerStart goroutine with defer se.Cancel()**

The default case at line 276 currently has:
```go
default:
    se, err := h.SessionCreate(pc, uuid.Nil, nil, nil, llmKey)
    if err != nil {
        return errors.Wrapf(err, "could not create pipecatcall session")
    }

    go h.RunnerStart(pc, se)
    return nil
```

Change to:
```go
default:
    se, err := h.SessionCreate(pc, uuid.Nil, nil, nil, llmKey)
    if err != nil {
        return errors.Wrapf(err, "could not create pipecatcall session")
    }

    go func() {
        defer se.Cancel()
        h.RunnerStart(pc, se)
    }()
    return nil
```

This matches the pattern used in other cases (e.g., the `ai_call` case at line 258-262).

**Step 2: Run Go tests**

Run: `cd bin-pipecat-manager && go test ./...`
Expected: All tests pass

**Step 3: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/start.go
git commit -m "NOJIRA-Fix-pipecat-manager-memory-leak

- bin-pipecat-manager: Add defer se.Cancel() wrapper around RunnerStart goroutine in default case"
```

---

### Task 4: Remove dead runWebSocketAsteriskRead function and test

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/websocket.go:91-110` (remove function)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/websocket_test.go:34-83` (remove test)

**Step 1: Remove runWebSocketAsteriskRead from websocket.go**

Delete lines 91-110 in `websocket.go`:
```go
// runWebSocketAsteriskRead reads from the WebSocket connection to handle
// ping/pong and close frames. Without a read loop, gorilla/websocket won't
// acknowledge pings. Closes doneCh when the connection is closed or encounters
// an error, signalling handlers to tear down their sessions.
func runWebSocketAsteriskRead(conn *websocket.Conn, doneCh chan struct{}) {
	log := logrus.WithField("func", "runWebSocketAsteriskRead")
	defer close(doneCh)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Debugf("Asterisk WebSocket closed normally: %v", err)
			} else {
				log.Errorf("Asterisk WebSocket read error: %v", err)
			}
			return
		}
	}
}
```

**Step 2: Remove Test_runWebSocketAsteriskRead from websocket_test.go**

Delete lines 34-83 in `websocket_test.go` (the entire `Test_runWebSocketAsteriskRead` function).

**Step 3: Check if any unused imports need cleaning up in websocket_test.go**

After removing the test, check if `strings` and `time` imports are still used by the remaining tests. If not, remove them.

**Step 4: Run Go tests**

Run: `cd bin-pipecat-manager && go test ./...`
Expected: All tests pass

**Step 5: Run lint**

Run: `cd bin-pipecat-manager && golangci-lint run -v --timeout 5m`
Expected: No errors

**Step 6: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/websocket.go bin-pipecat-manager/pkg/pipecatcallhandler/websocket_test.go
git commit -m "NOJIRA-Fix-pipecat-manager-memory-leak

- bin-pipecat-manager: Remove dead runWebSocketAsteriskRead function (replaced by runAsteriskReceivedMediaHandle)
- bin-pipecat-manager: Remove Test_runWebSocketAsteriskRead test"
```

---

### Task 5: Run full Go verification workflow

**Files:** None (verification only)

**Step 1: Run full verification**

```bash
cd bin-pipecat-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All steps pass with zero errors.

**Step 2: Check for any uncommitted changes from go mod tidy/generate**

Run: `git status`
Expected: No unexpected changes. If go.mod/go.sum changed, stage and amend the last commit.

---

### Task 6: Verify Python changes

**Files:** None (verification only)

**Step 1: Verify syntax**

```bash
cd bin-pipecat-manager/scripts/pipecat && python -c "import ast; ast.parse(open('run.py').read()); print('OK')"
```
Expected: `OK`

**Step 2: Run Python tests (if any exist)**

```bash
cd bin-pipecat-manager/scripts/pipecat && python -m pytest 2>/dev/null || echo "No pytest tests found"
```

---

## Files Changed Summary

| File | Change |
|------|--------|
| `scripts/pipecat/run.py` | Add services to ctx dict; add cleanup in both single and team pipeline finally blocks and init exception handlers |
| `pkg/pipecatcallhandler/start.go` | Wrap RunnerStart with `defer se.Cancel()` in default case |
| `pkg/pipecatcallhandler/websocket.go` | Remove dead `runWebSocketAsteriskRead` function |
| `pkg/pipecatcallhandler/websocket_test.go` | Remove `Test_runWebSocketAsteriskRead` test |

## Verification

1. Go: `cd bin-pipecat-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
2. Python: `cd bin-pipecat-manager/scripts/pipecat && python -c "import ast; ast.parse(open('run.py').read())"`
3. Deploy to staging and run multiple pipecat calls, verify memory returns to baseline after calls end
