# Fix Python /run Endpoint Error Handling

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the Python `/run` endpoint return HTTP errors when pipeline initialization fails, instead of always returning 200 OK.

**Architecture:** Split both `run_team_pipeline` and `run_single_ai_pipeline` into init + execute phases. The `/run` endpoint awaits the init phase (returning 4xx/5xx on failure) and spawns the execute phase as a background task (returning 200 only on successful init). Go side needs no changes — `pythonrunner.go:131-132` already handles non-200 correctly.

**Tech Stack:** Python (FastAPI, asyncio, pipecat-ai), Go (existing — no changes)

---

## Problem

`main.py:130` does `asyncio.create_task(run_pipeline_wrapper(req))` and immediately returns `{"status": "ok"}`. If the pipeline fails during initialization (unsupported service type, invalid team config, build_team_flow ValueError), Go never knows. The call appears to start but audio goes into a black hole.

## Solution

Split each pipeline function into two phases:

1. **Init phase** (synchronous — awaited by `/run`): Create services, build pipeline, set up FlowManager. Raises on failure.
2. **Execute phase** (async — background task): `runner.run(task)` + cleanup. Only started after init succeeds.

## What Init Catches vs. What It Doesn't

| Caught during init | NOT caught (runtime) |
|---|---|
| Unsupported LLM format (no `.` or `:`) | Invalid API key (fails on first LLM call) |
| Unsupported LLM/TTS/STT provider | LLM rate limiting |
| `start_member_id` not in members | WebSocket connection failure |
| `build_team_flow()` ValueError | TTS/STT service errors |
| `flow_manager.initialize()` errors | Audio processing errors |

---

### Task 1: Split `run_team_pipeline` into init + execute

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/run.py:379-532`

**Step 1: Create `init_team_pipeline` function**

Extract lines 379-515 (everything up to `runner.run(task)`) into a new function `init_team_pipeline` that returns a context dict. Add cleanup-on-failure for resources created after `task_manager.add`.

```python
async def init_team_pipeline(
    id: str,
    resolved_team: dict,
    stt_language: str = None,
    tts_language: str = None,
    llm_messages: list = None,
) -> dict:
    """Initialize team pipeline. Returns context dict. Raises on failure."""
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

        llm_svc, _ = create_llm_service(ai["engine_model"], ai["engine_key"], [], [])
        llm_services[mid] = llm_svc

        if ai.get("tts_type"):
            tts_svc = create_tts_service(ai["tts_type"], voice_id=ai.get("tts_voice_id"), language=tts_language)
            tts_services[mid] = tts_svc

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
    start_member = next((m for m in members if m["id"] == start_member_id), None)
    if start_member is None:
        raise ValueError(f"start_member_id {start_member_id} not found in members list")
    start_messages = []
    if start_member["ai"].get("init_prompt"):
        start_messages.append({"role": "system", "content": start_member["ai"]["init_prompt"]})
    start_messages.extend([m for m in llm_messages if m.get("role") and m.get("content")])

    context = OpenAILLMContext(messages=start_messages, tools=[])
    context_aggregator = llm_services[start_member_id].create_context_aggregator(context)

    # --- Step 4: Create transports ---
    transport_input = None
    if routing_stt:
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

    try:
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

        if transport_input:
            transport_input.event_handler("on_disconnected")(partial(handle_disconnect_or_error, "Input"))
            transport_input.event_handler("on_error")(partial(handle_disconnect_or_error, "Input"))
        transport_output.event_handler("on_disconnected")(partial(handle_disconnect_or_error, "Output"))
        transport_output.event_handler("on_error")(partial(handle_disconnect_or_error, "Output"))

        # --- Step 9: Initialize FlowManager ---
        await flow_manager.initialize(start_node)

    except Exception:
        # Cleanup resources created after task_manager.add on init failure
        await task.cancel()
        if transport_input:
            await transport_input.cleanup()
        await transport_output.cleanup()
        await task_manager.remove(id)
        raise

    init_total = time.monotonic() - total_start
    logger.info(f"[TEAM][INIT][total] All initialization completed in {init_total:.3f} sec. pipeline id={id}")

    return {
        "task": task,
        "transport_input": transport_input,
        "transport_output": transport_output,
    }
```

**Step 2: Create `execute_team_pipeline` function**

```python
async def execute_team_pipeline(id: str, ctx: dict):
    """Run the team pipeline loop and cleanup. Runs as background task."""
    task = ctx["task"]
    transport_input = ctx["transport_input"]
    transport_output = ctx["transport_output"]

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
        await task_manager.remove(id)
        logger.info(f"[TEAM][CLEANUP] Team pipeline cleaned. pipeline id={id}")
```

**Step 3: Remove old `run_team_pipeline` function**

Delete the entire `run_team_pipeline` function (lines 379-532).

**Step 4: Verify syntax**

Run: `cd bin-pipecat-manager/scripts/pipecat && python -c "import run"`
Expected: No import errors

---

### Task 2: Split `run_single_ai_pipeline` into init + execute

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/run.py:79-262`

**Step 1: Create `init_single_ai_pipeline` function**

Extract everything up to `runner.run(task)` (lines 79-237) into `init_single_ai_pipeline`. Returns context dict with all objects needed for execute phase.

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
) -> dict:
    """Initialize single AI pipeline. Returns context dict. Raises on failure."""
    total_start = time.monotonic()
    logger.info(f"[INIT] Starting Pipecat client pipeline id={id}")

    if llm_messages is None:
        llm_messages = []

    if tools_data is None:
        tools_data = []
    openai_tools = convert_to_openai_format(tools_data)
    tool_names = get_tool_names(tools_data)
    logger.info(f"[INIT] Received {len(tool_names)} tools: {tool_names}")

    init_tasks = {}

    if stt_type:
        async def init_stt_and_input_ws():
            start = time.monotonic()
            stt_service = create_stt_service(stt_type, language=stt_language)
            vad_analyzer = SileroVADAnalyzer(params=VADParams(stop_secs=0.8))
            transport = create_websocket_transport("input", id, vad_analyzer=vad_analyzer)
            logger.info(f"[INIT][stt+ws_input] done in {time.monotonic() - start:.3f} sec. pipeline id={id}")
            return {
                "stt_service": stt_service,
                "transport_input": transport,
                "vad_analyzer": vad_analyzer,
            }
        init_tasks["stt_input"] = asyncio.create_task(init_stt_and_input_ws())

    if tts_type:
        async def init_tts():
            start = time.monotonic()
            tts_service = create_tts_service(tts_type, voice_id=tts_voice_id, language=tts_language)
            logger.info(f"[INIT][tts] done in {time.monotonic() - start:.3f} sec. pipeline id={id}")
            return {
                "tts_service": tts_service,
            }
        init_tasks["tts"] = asyncio.create_task(init_tts())

    async def init_llm():
        start = time.monotonic()
        llm_service, aggregator = create_llm_service(llm_type, llm_key, llm_messages, openai_tools)
        logger.info(f"[INIT][llm] done in {time.monotonic() - start:.3f} sec. pipeline id={id}")
        return {
            "llm_service": llm_service,
            "llm_context_aggregator": aggregator,
        }
    init_tasks["llm"] = asyncio.create_task(init_llm())

    async def init_output_ws():
        start = time.monotonic()
        transport = create_websocket_transport("output", id, vad_analyzer=None)
        logger.info(f"[INIT][ws_output] done in {time.monotonic() - start:.3f} sec. pipeline id={id}")
        return {
            "transport_output": transport,
        }
    init_tasks["ws_output"] = asyncio.create_task(init_output_ws())

    # Await all init tasks
    try:
        results_list = await asyncio.gather(*init_tasks.values())
    except Exception as e:
        logger.error(f"[INIT] Pipeline initialization failed: {e}")
        for t in init_tasks.values():
            if not t.done():
                t.cancel()
        raise
    logger.info(f"[INIT] All components initialized in {time.monotonic() - total_start:.3f} sec. pipeline id={id}")

    results = {}
    for part in results_list:
        results.update(part)

    stt_service = results.get("stt_service")
    transport_input = results.get("transport_input")
    tts_service = results.get("tts_service")
    llm_service = results["llm_service"]
    llm_context_aggregator = results["llm_context_aggregator"]
    transport_output = results["transport_output"]

    # Assemble pipeline stages
    pipeline_stages = []
    if transport_input:
        pipeline_stages.append(transport_input.input())
        pipeline_stages.append(stt_service)
    pipeline_stages.append(llm_context_aggregator.user())
    pipeline_stages.append(llm_service)
    if tts_service:
        pipeline_stages.append(tts_service)
    pipeline_stages.append(llm_context_aggregator.assistant())
    pipeline_stages.append(transport_output.output())

    pipeline = Pipeline(pipeline_stages)

    # Create Pipeline Task
    task_start = time.monotonic()
    task = PipelineTask(
        pipeline,
        params=PipelineParams(
            audio_out_sample_rate=16000,
            enable_metrics=True,
            enable_usage_metrics=True,
        ),
    )

    await task_manager.add(id, task)
    logger.info(f"[INIT][task_create] done in {time.monotonic() - task_start:.3f} sec. pipeline id={id}")

    try:
        # Register tools (after task_manager.add so cleanup-on-failure can unregister)
        tool_register(llm_service, id, tool_names)

        async def handle_disconnect_or_error(name, transport, error=None):
            logger.error(f"{name} WebSocket disconnected or errored: {error}. pipeline id={id}")
            await task.cancel()

        if transport_input:
            transport_input.event_handler("on_disconnected")(partial(handle_disconnect_or_error, "Input"))
            transport_input.event_handler("on_error")(partial(handle_disconnect_or_error, "Input"))
        transport_output.event_handler("on_disconnected")(partial(handle_disconnect_or_error, "Output"))
        transport_output.event_handler("on_error")(partial(handle_disconnect_or_error, "Output"))

        # Warmup frame
        await task.queue_frames([LLMRunFrame()])

    except Exception:
        # Cleanup resources created after task_manager.add on init failure
        await task.cancel()
        if transport_input:
            await transport_input.cleanup()
        await transport_output.cleanup()
        tool_unregister(llm_service, tool_names)
        await task_manager.remove(id)
        raise

    init_total = time.monotonic() - total_start
    logger.info(f"[INIT][total] All initialization completed in {init_total:.3f} sec. pipeline id={id}")

    return {
        "task": task,
        "transport_input": transport_input,
        "transport_output": transport_output,
        "llm_service": llm_service,
        "tool_names": tool_names,
    }
```

**Step 2: Create `execute_single_ai_pipeline` function**

```python
async def execute_single_ai_pipeline(id: str, ctx: dict):
    """Run the single AI pipeline loop and cleanup. Runs as background task."""
    task = ctx["task"]
    transport_input = ctx["transport_input"]
    transport_output = ctx["transport_output"]
    llm_service = ctx["llm_service"]
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
        if llm_service:
            tool_unregister(llm_service, tool_names)
        await task_manager.remove(id)
        logger.info(f"[CLEANUP] Pipeline cleaned. pipeline id={id}")
```

**Step 3: Remove old `run_single_ai_pipeline` function**

Delete the entire `run_single_ai_pipeline` function (lines 79-262).

**Step 4: Update `run_pipeline` to become `init_pipeline` + `execute_pipeline`**

Replace the old `run_pipeline` function with:

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
) -> dict:
    """Initialize the pipeline. Returns context dict. Raises on failure."""
    if resolved_team:
        ctx = await init_team_pipeline(
            id, resolved_team,
            stt_language=stt_language,
            tts_language=tts_language,
            llm_messages=llm_messages,
        )
        ctx["type"] = "team"
        return ctx
    else:
        ctx = await init_single_ai_pipeline(
            id, llm_type, llm_key, llm_messages,
            stt_type, stt_language, tts_type, tts_language,
            tts_voice_id, tools_data,
        )
        ctx["type"] = "single"
        return ctx


async def execute_pipeline(id: str, ctx: dict):
    """Execute the pipeline loop. Runs as background task."""
    if ctx["type"] == "team":
        await execute_team_pipeline(id, ctx)
    else:
        await execute_single_ai_pipeline(id, ctx)
```

**Step 5: Verify syntax**

Run: `cd bin-pipecat-manager/scripts/pipecat && python -c "import run"`
Expected: No import errors

---

### Task 3: Update `/run` endpoint in `main.py`

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/main.py:10,90-137`

**Step 1: Update imports**

Change line 10 from:
```python
from run import run_pipeline
```
to:
```python
from run import init_pipeline, execute_pipeline
```

**Step 2: Replace `run_pipeline_wrapper` and `/run` endpoint**

Replace lines 90-137 with:

```python
async def execute_pipeline_wrapper(id: str, ctx: dict):
    """Background task wrapper for execute_pipeline."""
    try:
        await execute_pipeline(id, ctx)
        logger.info(f"Pipeline finished successfully: id={id}")
    except Exception as e:
        logger.exception(f"Pipeline execution failed (id={id}): {e}")


@app.post("/run")
async def run_pipeline_endpoint(req: PipelineRequest):
    msg_count = len(req.llm_messages or [])
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

    try:
        tools_data = [t.model_dump() for t in req.tools] if req.tools else []
        resolved_team_data = req.resolved_team.model_dump() if req.resolved_team else None

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
        )
    except ValueError as e:
        logger.error(f"Pipeline validation failed (id={req.id}): {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.exception(f"Pipeline initialization failed (id={req.id}): {e}")
        raise HTTPException(status_code=500, detail=str(e))

    asyncio.create_task(execute_pipeline_wrapper(req.id, ctx))

    return {"status": "ok", "message": "Pipeline initialized and started"}
```

**Step 3: Verify syntax**

Run: `cd bin-pipecat-manager/scripts/pipecat && python -c "import main"`
Expected: No import errors

---

### Task 4: Add Python tests for init error handling

**Files:**
- Create: `bin-pipecat-manager/scripts/pipecat/test_init_pipeline.py`

**Step 1: Write tests**

```python
"""Tests for pipeline init error handling.

Verifies that init_pipeline raises exceptions for invalid configurations
instead of silently failing in a background task.
"""
import pytest
from unittest.mock import patch, AsyncMock

from run import init_pipeline, init_team_pipeline, init_single_ai_pipeline


@pytest.mark.asyncio
async def test_init_single_ai_pipeline_unsupported_llm_type():
    """Unsupported LLM service name raises ValueError."""
    with pytest.raises(ValueError, match="Unsupported LLM service"):
        await init_single_ai_pipeline(
            id="test-1",
            llm_type="unsupported.model",
            llm_key="fake-key",
        )


@pytest.mark.asyncio
async def test_init_single_ai_pipeline_bad_llm_format():
    """LLM type without separator raises ValueError."""
    with pytest.raises(ValueError, match="Wrong LLM format"):
        await init_single_ai_pipeline(
            id="test-2",
            llm_type="no-separator",
            llm_key="fake-key",
        )


@pytest.mark.asyncio
async def test_init_single_ai_pipeline_unsupported_tts_type():
    """Unsupported TTS type raises ValueError."""
    with pytest.raises(ValueError, match="Unsupported TTS service"):
        await init_single_ai_pipeline(
            id="test-3",
            llm_type="openai.gpt-4o",
            llm_key="fake-key",
            tts_type="unsupported",
        )


@pytest.mark.asyncio
async def test_init_single_ai_pipeline_unsupported_stt_type():
    """Unsupported STT type raises ValueError."""
    with pytest.raises(ValueError, match="Unsupported STT service"):
        await init_single_ai_pipeline(
            id="test-4",
            llm_type="openai.gpt-4o",
            llm_key="fake-key",
            stt_type="unsupported",
        )


@pytest.mark.asyncio
async def test_init_team_pipeline_start_member_not_found():
    """start_member_id not in members raises ValueError."""
    resolved_team = {
        "id": "team-1",
        "start_member_id": "nonexistent-member",
        "members": [
            {
                "id": "member-1",
                "name": "Agent A",
                "ai": {
                    "engine_model": "openai.gpt-4o",
                    "engine_key": "fake-key",
                },
                "tools": [],
                "transitions": [],
            }
        ],
    }
    with pytest.raises(ValueError, match="start_member_id .* not found"):
        await init_team_pipeline(
            id="test-5",
            resolved_team=resolved_team,
        )


@pytest.mark.asyncio
async def test_init_team_pipeline_unsupported_member_llm():
    """Unsupported LLM type in a team member raises ValueError."""
    resolved_team = {
        "id": "team-2",
        "start_member_id": "member-1",
        "members": [
            {
                "id": "member-1",
                "name": "Agent A",
                "ai": {
                    "engine_model": "unsupported.model",
                    "engine_key": "fake-key",
                },
                "tools": [],
                "transitions": [],
            }
        ],
    }
    with pytest.raises(ValueError, match="Unsupported LLM service"):
        await init_team_pipeline(
            id="test-6",
            resolved_team=resolved_team,
        )


@pytest.mark.asyncio
async def test_init_team_pipeline_empty_members():
    """Empty members list still fails at start_member_id lookup."""
    resolved_team = {
        "id": "team-3",
        "start_member_id": "member-1",
        "members": [],
    }
    with pytest.raises(ValueError, match="start_member_id .* not found"):
        await init_team_pipeline(
            id="test-7",
            resolved_team=resolved_team,
        )
```

**Step 2: Run tests**

Run: `cd bin-pipecat-manager/scripts/pipecat && python -m pytest test_init_pipeline.py -v`
Expected: All tests pass

---

### Task 5: Commit and verify

**Step 1: Verify Python syntax**

Run:
```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-fix-python-run-endpoint-error-handling/bin-pipecat-manager/scripts/pipecat
python -c "import main; import run"
```
Expected: No errors

**Step 2: Run Python tests**

Run:
```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-fix-python-run-endpoint-error-handling/bin-pipecat-manager/scripts/pipecat
python -m pytest test_init_pipeline.py -v
```
Expected: All tests pass

**Step 3: Run Go tests (no Go changes, but verify nothing broke)**

Run:
```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-fix-python-run-endpoint-error-handling/bin-pipecat-manager
go test ./...
```
Expected: All tests pass

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-fix-python-run-endpoint-error-handling
git add bin-pipecat-manager/scripts/pipecat/main.py \
        bin-pipecat-manager/scripts/pipecat/run.py \
        bin-pipecat-manager/scripts/pipecat/test_init_pipeline.py \
        docs/plans/2026-03-02-fix-python-run-endpoint-error-handling-design.md
git commit -m "NOJIRA-fix-python-run-endpoint-error-handling

Split Python pipeline init from execute so /run endpoint returns HTTP errors
on initialization failures instead of always returning 200 OK.

- bin-pipecat-manager: Split run_team_pipeline into init_team_pipeline + execute_team_pipeline
- bin-pipecat-manager: Split run_single_ai_pipeline into init_single_ai_pipeline + execute_single_ai_pipeline
- bin-pipecat-manager: Update /run endpoint to await init phase, spawn execute as background task
- bin-pipecat-manager: Add Python tests for init error handling
- docs: Add design document for the fix"
```

**Step 5: Push and create PR**

```bash
git push -u origin NOJIRA-fix-python-run-endpoint-error-handling
```
