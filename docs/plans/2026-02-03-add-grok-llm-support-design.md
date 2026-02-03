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
| `bin-ai-manager/models/ai/main_test.go` | Add test cases for Grok models |
| `bin-pipecat-manager/scripts/pipecat/run.py` | Add `grok` case in `create_llm_service()` |
| `bin-pipecat-manager/scripts/pipecat/test_run.py` | New test file for `create_llm_service()` |
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

## Tests

### Go Tests (bin-ai-manager/models/ai/main_test.go)

Add test cases to existing test functions:

**TestEngineModelTargetConstants:**
```go
{
    name:     "engine_model_target_grok",
    constant: EngineModelTargetGrok,
    expected: "grok",
},
```

**TestEngineModelConstants:**
```go
{
    name:     "engine_model_grok3",
    constant: EngineModelGrok3,
    expected: "grok.grok-3",
},
{
    name:     "engine_model_grok3_mini",
    constant: EngineModelGrok3Mini,
    expected: "grok.grok-3-mini",
},
```

**TestGetEngineModelTarget:**
```go
{
    name:        "grok3_returns_grok",
    engineModel: EngineModelGrok3,
    expected:    EngineModelTargetGrok,
},
{
    name:        "grok3_mini_returns_grok",
    engineModel: EngineModelGrok3Mini,
    expected:    EngineModelTargetGrok,
},
```

**TestGetEngineModelName:**
```go
{
    name:        "grok3_returns_grok3",
    engineModel: EngineModelGrok3,
    expected:    "grok-3",
},
```

**TestIsValidEngineModel:**
```go
{
    name:        "grok_model_is_valid",
    engineModel: EngineModel("grok.grok-3"),
    expected:    true,
},
```

### Python Tests (bin-pipecat-manager/scripts/pipecat/test_run.py)

New test file for `create_llm_service()`:

```python
import pytest
from unittest.mock import patch, MagicMock
import os


class TestCreateLLMService:
    """Tests for create_llm_service function."""

    @patch("run.OpenAILLMService")
    @patch("run.OpenAILLMContext")
    def test_openai_service_creation(self, mock_context, mock_service):
        """Test OpenAI service is created with correct parameters."""
        from run import create_llm_service

        mock_llm = MagicMock()
        mock_service.return_value = mock_llm
        mock_llm.create_context_aggregator.return_value = MagicMock()

        llm, aggregator = create_llm_service(
            type="openai.gpt-4o",
            key="test-key",
            messages=[{"role": "user", "content": "hello"}],
            tools=[]
        )

        mock_service.assert_called_once_with(api_key="test-key", model="gpt-4o")

    @patch("run.OpenAILLMService")
    @patch("run.OpenAILLMContext")
    def test_grok_service_creation(self, mock_context, mock_service):
        """Test Grok service is created with xAI base URL."""
        from run import create_llm_service

        mock_llm = MagicMock()
        mock_service.return_value = mock_llm
        mock_llm.create_context_aggregator.return_value = MagicMock()

        llm, aggregator = create_llm_service(
            type="grok.grok-3",
            key="xai-test-key",
            messages=[{"role": "user", "content": "hello"}],
            tools=[]
        )

        mock_service.assert_called_once_with(
            api_key="xai-test-key",
            model="grok-3",
            base_url="https://api.x.ai/v1"
        )

    @patch("run.OpenAILLMService")
    @patch("run.OpenAILLMContext")
    def test_grok_uses_env_var_fallback(self, mock_context, mock_service):
        """Test Grok falls back to XAI_API_KEY env var."""
        from run import create_llm_service

        mock_llm = MagicMock()
        mock_service.return_value = mock_llm
        mock_llm.create_context_aggregator.return_value = MagicMock()

        with patch.dict(os.environ, {"XAI_API_KEY": "env-xai-key"}):
            llm, aggregator = create_llm_service(
                type="grok.grok-3-mini",
                key="",
                messages=[],
                tools=[]
            )

        mock_service.assert_called_once_with(
            api_key="env-xai-key",
            model="grok-3-mini",
            base_url="https://api.x.ai/v1"
        )

    def test_unsupported_service_raises_error(self):
        """Test unsupported service raises ValueError."""
        from run import create_llm_service

        with pytest.raises(ValueError, match="Unsupported LLM service"):
            create_llm_service(
                type="unsupported.model",
                key="key",
                messages=[],
                tools=[]
            )

    def test_invalid_format_raises_error(self):
        """Test invalid format without dot raises ValueError."""
        from run import create_llm_service

        with pytest.raises(ValueError, match="Wrong LLM format"):
            create_llm_service(
                type="invalidformat",
                key="key",
                messages=[],
                tools=[]
            )
```

## Validation

1. Run `go test ./...` in `bin-ai-manager` to ensure model constants and tests pass
2. Run `pytest test_run.py` in `bin-pipecat-manager/scripts/pipecat/` to ensure Python tests pass
3. Create an AI config with `engine_model: "grok.grok-3"` and test a conversation

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
