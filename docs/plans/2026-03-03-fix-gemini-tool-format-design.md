# Fix Gemini Tool Format in Pipecat Manager

**Date:** 2026-03-03
**Status:** Implemented

## Problem

When starting an AI call with `llm_type=gemini.*` and tools configured, the pipeline fails with:

```
27 validation errors for GenerateContentConfig
tools.0.Tool.type - Extra inputs are not permitted [type=extra_forbidden, input_value='function']
tools.0.Tool.function - Extra inputs are not permitted
tools.0.callable - Input should be callable
```

This error repeats for all 9 tools (connect_call, send_email, send_message, stop_media, stop_service, stop_flow, set_variables, get_variables, get_aicall_messages).

### Root Cause

In `scripts/pipecat/run.py`, `create_llm_service()` handles all LLM types the same way for context creation:

```python
# Line 375-382 (gemini case)
llm = GoogleLLMService(api_key=api_key, model=model_name)
ctx = OpenAILLMContext(messages=valid_messages, tools=tools)  # tools in OpenAI format
aggregator = llm.create_context_aggregator(ctx)
```

The `tools` parameter contains OpenAI-formatted tools:
```json
[{"type": "function", "function": {"name": "connect_call", "description": "...", "parameters": {...}}}]
```

But Google's Gemini API expects:
```json
[{"function_declarations": [{"name": "connect_call", "description": "...", "parameters": {...}}]}]
```

**Code path analysis:**

1. `create_context_aggregator()` upgrades `OpenAILLMContext` to `GoogleLLMContext` (a subclass of `OpenAILLMContext`), but does NOT convert tool format - only messages.
2. The aggregator emits `OpenAILLMContextFrame`.
3. `GoogleLLMService.process_frame()` receives `OpenAILLMContextFrame` and routes to `_stream_content_specific_context()`.
4. `_stream_content_specific_context()` passes `context.tools` directly to `GenerateContentConfig` without conversion.
5. `GenerateContentConfig` (pydantic model from `google.genai.types`) rejects the OpenAI format.

The conversion exists but only activates on the **universal `LLMContext`** path:
- `LLMContextFrame` → `_stream_content_universal_context()` → `GeminiLLMAdapter.get_llm_invocation_params()` → `from_standard_tools()` → `to_provider_tools_format()` → proper Google format.

## Approach

Use pipecat-ai's universal `LLMContext` + `LLMContextAggregatorPair` for the Gemini case, replacing the deprecated `OpenAILLMContext` + `create_context_aggregator()`.

### Why This Approach

- Aligns with pipecat-ai's migration direction (the `OpenAILLMContext`/`create_context_aggregator` API is deprecated since v0.0.99)
- Tool format conversion is handled internally by `GeminiLLMAdapter`, so we don't maintain custom conversion logic
- OpenAI/Grok cases remain unchanged since `OpenAILLMContext` works correctly with those services

### Changes

**Single file:** `bin-pipecat-manager/scripts/pipecat/run.py`

1. **Add imports:**
   ```python
   from pipecat.processors.aggregators.llm_context import LLMContext
   from pipecat.processors.aggregators.llm_response_universal import LLMContextAggregatorPair
   from pipecat.adapters.schemas.function_schema import FunctionSchema
   from pipecat.adapters.schemas.tools_schema import ToolsSchema
   ```

2. **Add helper function** `_openai_tools_to_standard(openai_tools)`:
   - Converts `[{"type": "function", "function": {"name": ..., "description": ..., "parameters": {...}}}]`
   - To `[FunctionSchema(name=..., description=..., properties=..., required=...)]`
   - Returns empty list for empty/None input

3. **Modify `create_llm_service()` gemini case:**
   ```python
   elif service_name == "gemini":
       api_key = key or os.getenv("GOOGLE_API_KEY")
       llm = GoogleLLMService(api_key=api_key, model=model_name)

       # Use universal LLMContext so GeminiLLMAdapter properly converts
       # OpenAI-format tools to Google's function_declarations format.
       standard_tools = _openai_tools_to_standard(tools)
       if standard_tools:
           tools_schema = ToolsSchema(standard_tools=standard_tools)
           logger.debug(f"Converted {len(standard_tools)} tools to FunctionSchema for Gemini")
       else:
           tools_schema = NOT_GIVEN
       ctx = LLMContext(messages=valid_messages, tools=tools_schema)
       aggregator = LLMContextAggregatorPair(ctx)

       return llm, aggregator
   ```

### Data Flow After Fix

```
OpenAI tools → _openai_tools_to_standard() → FunctionSchema objects
  → ToolsSchema(standard_tools=[...])
  → LLMContext(tools=tools_schema)
  → LLMContextAggregatorPair emits LLMContextFrame
  → GoogleLLMService.process_frame detects LLMContextFrame
  → _stream_content_universal_context()
  → GeminiLLMAdapter.get_llm_invocation_params()
  → from_standard_tools() → to_provider_tools_format()
  → {"function_declarations": [...]}  ← Google format
  → GenerateContentConfig accepts ← No validation errors
```

### Scope

- Team pipeline is unaffected (passes `tools=[]` so no format issue)
- Tool registration (`llm_service.register_function`) is on the LLM service, independent of context type
- OpenAI and Grok cases unchanged

### Risks

- `LLMContextAggregatorPair` uses `LLMUserAggregator`/`LLMAssistantAggregator` instead of Google-specific ones. These are the new universal aggregators and should handle Google's message format correctly since they delegate to the LLM service's adapter.
- If pipecat-ai changes the `LLMContext`/`ToolsSchema` API, we'd need to update. But this is the officially supported path going forward.

## Verification

1. Deploy to the running pipecat-manager pod (or test locally)
2. Start an AI call with `llm_type=gemini.gemini-1.5-pro`, `stt_type=google`, `tts_type=google`, and tools configured
3. Verify no `GenerateContentConfig` validation errors in logs
4. Verify tool calls (connect_call, etc.) work end-to-end
5. Verify OpenAI-based calls still work (regression check)
