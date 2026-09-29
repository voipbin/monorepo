"""Real-library check of gemini_tool_filter against the pinned pipecat/google-genai.

Lives outside scripts/pipecat/ on purpose: that directory's conftest.py
replaces the pipecat tree with MagicMocks, and pytest also loads parent
directory conftests, so a real-library test anywhere under scripts/pipecat/
would never see real pipecat (design 15.7).

Not run by CI. Run manually in a venv with the pinned versions:

    pytest --noconftest bin-pipecat-manager/scripts/pipecat_realtest
"""

import json
import os
import sys
from pathlib import Path

import pytest

pytest.importorskip("pipecat.adapters.services.gemini_adapter")

from google.genai.types import GenerateContentConfig  # noqa: E402
from pipecat.adapters.schemas.function_schema import FunctionSchema  # noqa: E402
from pipecat.adapters.schemas.tools_schema import ToolsSchema  # noqa: E402
from pipecat.adapters.services.gemini_adapter import GeminiLLMAdapter  # noqa: E402
from pydantic import ValidationError  # noqa: E402

_HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(_HERE.parent / "pipecat"))

from gemini_tool_filter import drop_gemini_invalid_tools  # noqa: E402

# Go goldens from bin-ai-manager/pkg/mcpschema (design 15.7 item 4): the
# normalized output the Go side advertises must be accepted by the client.
_GOLDEN_DIR = _HERE.parents[2] / "bin-ai-manager" / "pkg" / "mcpschema" / "testdata" / "github"
_GOLDENS = sorted(_GOLDEN_DIR.glob("*.golden.json")) if _GOLDEN_DIR.is_dir() else []


def _validates(schemas) -> bool:
    tools = GeminiLLMAdapter().to_provider_tools_format(ToolsSchema(standard_tools=schemas))
    try:
        GenerateContentConfig(tools=tools)
        return True
    except ValidationError:
        return False


def _good():
    return FunctionSchema(
        name="connect_call",
        description="d",
        properties={"a": {"type": "string"}},
        required=["a"],
    )


def _x_mcp_header():
    return FunctionSchema(
        name="mcp_x_bad",
        description="d",
        properties={"repo": {"type": "string", "x-mcp-header": "repo"}},
        required=[],
    )


def test_x_mcp_header_tool_fails_the_whole_set():
    assert _validates([_good()])
    assert not _validates([_good(), _x_mcp_header()])


def test_filter_keeps_only_the_good_tool_and_the_result_validates():
    good, bad = _good(), _x_mcp_header()

    kept = drop_gemini_invalid_tools([good, bad], "realtest")

    assert [fs.name for fs in kept] == ["connect_call"]
    assert _validates(kept)


def test_filter_returns_valid_set_unchanged():
    schemas = [_good()]

    assert drop_gemini_invalid_tools(schemas, "realtest") is schemas


@pytest.mark.skipif(not _GOLDENS, reason=f"no Go goldens at {_GOLDEN_DIR}")
@pytest.mark.parametrize("golden", _GOLDENS, ids=lambda p: p.name)
def test_go_golden_is_accepted(golden):
    params = json.loads(golden.read_text())
    fs = FunctionSchema(
        name=os.path.basename(golden).split(".")[0],
        description="d",
        properties=params.get("properties", {}),
        required=params.get("required", []),
    )

    assert _validates([fs])
    assert drop_gemini_invalid_tools([fs], "realtest") == [fs]
