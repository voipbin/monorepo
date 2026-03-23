# Fix: Team Pipeline Conversation History Loss

**Date:** 2026-03-23
**Status:** Complete
**Component:** bin-pipecat-manager (Python)

## Problem

Non-realtime team AIcalls repeat the same greeting message regardless of user input. The LLM always responds with the initial greeting (e.g., "Hi, thank you for calling Voipbin!") because it never sees the conversation history.

**Observed on:** AIcall `f4e204fd-084c-4e57-9908-aa2dadc45801`

## Root Cause

In `scripts/pipecat/run.py` `init_team_pipeline()`:

1. **Line 608**: A shared `LLMContext` is created with `start_messages` (system prompt + conversation history from `llm_messages`).
2. **Line 685**: `flow_manager.initialize(start_node)` calls `_set_node()` → `_update_llm_context()`.
3. Since this is the first node, FlowManager uses `LLMMessagesUpdateFrame` which **replaces** the shared context messages with the node's `role_messages` + `task_messages`.
4. The start node has `role_messages = [system_prompt]` and `task_messages = []`.
5. **The conversation history is wiped.** The LLM only sees the system prompt + current user message → generates the greeting every time.

The per-member LLM services being created with empty `[]` messages at line 561 is not the issue — LLM services use context from the shared `LLMContextFrame`, not internal state.

## Fix: Pass conversation history via `task_messages`

The `NodeConfig.task_messages` field is the intended mechanism in pipecat-flows for conversation context. Include `llm_messages` in the start node's `task_messages` so FlowManager preserves them during initialization.

### Files Changed

**`scripts/pipecat/team_flow.py`**
- Add `llm_messages` parameter to `build_team_flow()`
- After building the start node, set its `task_messages` to the filtered conversation history

**`scripts/pipecat/run.py`**
- Pass `llm_messages` to `build_team_flow()` at the call site (line 650)

### Why This Works

- `FlowManager.initialize()` builds `LLMMessagesUpdateFrame(messages=role_messages + task_messages)`
- With the fix, `task_messages` contains the conversation history
- The resulting context is: `[system_prompt, user_msg_1, assistant_msg_1, ...]`
- On member transitions, FlowManager uses `LLMMessagesAppendFrame` (append, not replace), so history is preserved naturally

### Alternatives Considered

| Approach | Verdict |
|----------|---------|
| Post-initialize context injection | Fragile, depends on internal FlowManager ordering |
| Per-member service contexts at line 561 | Won't work — FlowManager replaces shared context anyway |
