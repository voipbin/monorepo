# Add Grok LLM Support Design

**Date:** 2026-02-03

## Overview

Add Grok (`grok-3`, `grok-3-mini`) as an LLM provider for real-time voice calls, conversations, and tasks.

Grok's API is OpenAI-compatible, so we reuse `OpenAILLMService` with a custom `base_url` (`https://api.x.ai/v1`).

## Scope

**In scope:**
- Grok support for voice calls, conversations, and tasks (via Pipecat)
- Models: `grok.grok-3`, `grok.grok-3-mini`
- Per-AI API key via `engine_key` field
- Default/global API key via `XAI_API_KEY` environment variable

**Out of scope:**
- Summary generation with Grok (uses separate Go handler)
- Go engine handler changes (not needed - Pipecat handles all LLM calls)

## Files to Modify

| File | Changes |
|------|---------|
| `bin-ai-manager/models/ai/main.go` | Add model constants + update `GetEngineModelTarget()` |
| `bin-pipecat-manager/scripts/pipecat/run.py` | Add `grok` case in `create_llm_service()` |
| `bin-pipecat-manager/k8s/deployment.yml` | Add `XAI_API_KEY` env var (2 places) |
| `.circleci/config_work.yml` | Add sed substitution for `XAI_API_KEY` |

## Implementation Details

### 1. Model Definitions (bin-ai-manager/models/ai/main.go)

Add Grok model constants after line 111:

```go
EngineModelGrok3     EngineModel = "grok.grok-3"
EngineModelGrok3Mini EngineModel = "grok.grok-3-mini"
```

Update `GetEngineModelTarget()` map:

```go
EngineModelGrok3:     EngineModelTargetGrok,
EngineModelGrok3Mini: EngineModelTargetGrok,
```

Note: `EngineModelTargetGrok` constant already exists (line 59).

### 2. Pipecat Python (bin-pipecat-manager/scripts/pipecat/run.py)

Update `create_llm_service()` function to handle Grok:

```python
def create_llm_service(type: str, key: str, messages, tools):
    # ... existing parsing logic ...

    service_name = service_name.lower()
    if service_name == "openai":
        api_key = key or os.getenv("OPENAI_API_KEY")
        llm = OpenAILLMService(api_key=api_key, model=model_name)

        ctx = OpenAILLMContext(messages=valid_messages, tools=tools)
        aggregator = llm.create_context_aggregator(ctx)
        return llm, aggregator

    elif service_name == "grok":
        api_key = key or os.getenv("XAI_API_KEY")
        llm = OpenAILLMService(
            api_key=api_key,
            model=model_name,
            base_url="https://api.x.ai/v1"
        )

        ctx = OpenAILLMContext(messages=valid_messages, tools=tools)
        aggregator = llm.create_context_aggregator(ctx)
        return llm, aggregator

    else:
        raise ValueError(f"Unsupported LLM service: {service_name}")
```

### 3. Kubernetes Deployment (bin-pipecat-manager/k8s/deployment.yml)

Add `XAI_API_KEY` environment variable in both container sections (around lines 52-53 and 78-79):

```yaml
- name: XAI_API_KEY
  value: ${XAI_API_KEY}
```

### 4. CircleCI Config (.circleci/config_work.yml)

Add sed substitution around line 1702:

```bash
find . -type f -exec sed -i -e "s|\${XAI_API_KEY}|$CC_XAI_API_KEY|g" {} +
```

Note: `CC_XAI_API_KEY` needs to be added to CircleCI project environment variables.

## API Key Flow

1. AI config can specify `engine_key` for per-AI Grok API key
2. If no `engine_key`, falls back to `XAI_API_KEY` environment variable
3. Key is passed from Go (bin-ai-manager) → Python (bin-pipecat-manager) via `/run` endpoint

## Validation

1. Run `go test ./...` in `bin-ai-manager` to ensure model constants work
2. Create an AI config with `engine_model: "grok.grok-3"` and test a conversation

## Usage

After implementation, create an AI configuration like:

```json
{
  "name": "Grok Assistant",
  "engine_model": "grok.grok-3",
  "engine_key": "xai-xxx...",
  "init_prompt": "You are a helpful assistant.",
  "tts_type": "elevenlabs",
  "stt_type": "deepgram"
}
```

Or use `grok.grok-3-mini` for faster/cheaper responses.
