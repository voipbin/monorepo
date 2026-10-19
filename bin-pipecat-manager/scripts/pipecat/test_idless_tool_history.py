"""/run accepts history whose tool calls carry no id (VOIP-1633).

ai-manager omits an empty tool call id (`omitempty`), so history written without ids reaches the runner
without the field. It used to be rejected with HTTP 422 (`body.llm_messages.N.tool_calls.0.id:missing`).
"""

import pytest

import main


def _history(call_id_fields):
    call = {"type": "function", "function": {"name": "get_x", "arguments": "{}"}}
    call.update(call_id_fields)
    return [
        {"role": "user", "content": "hi"},
        {"role": "assistant", "tool_calls": [call]},
        {"role": "tool", "content": "{\"ok\": 1}"},
        {"role": "user", "content": "again"},
    ]


@pytest.mark.parametrize("fields", [{}, {"id": None}, {"id": ""}, {"id": "call_1"}])
def test_run_request_accepts_missing_null_and_empty_tool_call_id(fields):
    req = main.PipelineRequest(id="p1", llm_type="openai.gpt-4o", llm_messages=_history(fields))
    dumped = [m.model_dump() for m in req.llm_messages]
    assert dumped[1]["tool_calls"][0]["id"] == fields.get("id")
