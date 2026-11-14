"""Real-adapter tests for the order of tool call and tool result rows (VOIP-1733).

The history that bin-ai-manager replays can hold a bot sentence between a tool call row and its result row
(the sentence event is stored after the call row). The OpenAI chat API rejects that order, and the Gemini
turns must keep every function_call turn directly followed by its function_response turn. These tests convert
the history with the real pipecat OpenAI and Gemini adapters (no network, no key, no cost) and check the order.

Lives outside scripts/pipecat/ because that conftest replaces pipecat with MagicMocks. Not run by CI. Run
manually in a venv with the pinned versions (scripts/pipecat/requirements.txt or `uv sync --frozen` in
scripts/pipecat) plus `pytest`:

    pytest --noconftest bin-pipecat-manager/scripts/pipecat_realtest/test_message_order_real.py

The same order invariant is guarded in CI by assert_valid_order in scripts/pipecat/test_message_filters.py.
"""
import os
import sys
import warnings

import pytest

HERE = os.path.dirname(__file__)
sys.path.insert(0, os.path.abspath(os.path.join(HERE, "..", "pipecat")))
warnings.simplefilter("ignore")

pytest.importorskip("pipecat.adapters.services.gemini_adapter")

from message_filters import filter_valid_messages  # noqa: E402
from pipecat.adapters.services.gemini_adapter import GeminiLLMAdapter  # noqa: E402
from pipecat.adapters.services.open_ai_adapter import OpenAILLMAdapter  # noqa: E402
from pipecat.processors.aggregators.llm_context import LLMContext  # noqa: E402


def _call(call_id, name="mcp_search_flights"):
    return {"role": "assistant", "content": "", "tool_calls": [{"id": call_id, "type": "function", "function": {"name": name, "arguments": "{}"}}]}


def _result(call_id):
    return {"role": "tool", "content": '{"result": "success"}', "tool_call_id": call_id}


SENTENCE = "I'll connect you now."

# The order ai-manager can store: the sentence event arrives after the call row and before the result row.
STORED = [
    {"role": "user", "content": "Find me a flight."},
    _call("a"),
    {"role": "assistant", "content": SENTENCE},
    _result("a"),
    {"role": "assistant", "content": "Here is a flight."},
    {"role": "user", "content": "Thanks, and one more thing."},
]


def _openai_messages(messages):
    params = OpenAILLMAdapter().get_llm_invocation_params(LLMContext(messages=messages), convert_developer_to_user=False)
    return params["messages"]


def _gemini_contents(messages):
    return GeminiLLMAdapter().get_llm_invocation_params(LLMContext(messages=messages))["messages"]


def _openai_violations(messages):
    """OpenAI rule: every assistant message with tool_calls is directly followed by its tool messages."""
    bad = []
    i = 0
    while i < len(messages):
        m = messages[i]
        if m["role"] == "assistant" and m.get("tool_calls"):
            ids = [tc["id"] for tc in m["tool_calls"]]
            following = messages[i + 1:i + 1 + len(ids)]
            if [f.get("tool_call_id") for f in following] != ids or any(f["role"] != "tool" for f in following):
                bad.append(i)
            i += 1 + len(ids)
            continue
        if m["role"] == "tool":
            bad.append(i)
        i += 1
    return bad


def _kind(part):
    if part.function_call:
        return "function_call"
    if part.function_response:
        return "function_response"
    return "text"


def _gemini_violations(contents):
    """Gemini rule: a turn with function_call parts is directly followed by the turn with the function_response parts."""
    bad = []
    for i, c in enumerate(contents):
        kinds = [_kind(p) for p in c.parts]
        if "function_call" in kinds:
            nxt = contents[i + 1] if i + 1 < len(contents) else None
            if nxt is None or [_kind(p) for p in nxt.parts] != ["function_response"] * len(kinds):
                bad.append(i)
        if "function_response" in kinds:
            prev = contents[i - 1] if i > 0 else None
            if prev is None or not any(p.function_call for p in prev.parts):
                bad.append(i)
    return bad


def test_stored_order_is_invalid_for_openai_and_gemini():
    """Negative control: the validators do report the stored order, so the passing test below proves something."""
    assert 1 in _openai_violations(_openai_messages(STORED)), "the call row is not followed by its tool message"
    assert _gemini_violations(_gemini_contents(STORED)) != []


def test_filtered_order_is_valid_for_openai_and_gemini():
    filtered = filter_valid_messages(STORED)
    assert _openai_violations(_openai_messages(filtered)) == []
    assert _gemini_violations(_gemini_contents(filtered)) == []


def test_gemini_keeps_the_sentence_as_a_model_text_turn():
    contents = _gemini_contents(filter_valid_messages(STORED))
    texts = [(c.role, p.text) for c in contents for p in c.parts if p.text]
    assert ("model", SENTENCE) in texts, "the sentence must not be lost on the Gemini path"
    kinds = [(c.role, [_kind(p) for p in c.parts]) for c in contents]
    sentence_turn = kinds.index(("model", ["text"]))
    assert kinds[sentence_turn + 1] == ("model", ["function_call"])
    assert kinds[sentence_turn + 2] == ("user", ["function_response"])
