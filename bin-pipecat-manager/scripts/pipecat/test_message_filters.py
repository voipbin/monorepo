"""Pure unit tests for filter_valid_messages() (VOIP-1460).

message_filters.py has no dependencies (no pipecat, no loguru), so these
tests import it directly with no mocking.
"""

from message_filters import filter_valid_messages


def _assistant_call(call_id, name="get_resource", content=""):
    """Production shape: content="" + tool_calls (tool.go creates it this way)."""
    return {
        "role": "assistant",
        "content": content,
        "tool_calls": [
            {
                "id": call_id,
                "type": "function",
                "function": {"name": name, "arguments": "{}"},
            }
        ],
    }


def _tool_result(call_id, content='{"result": "success"}'):
    return {"role": "tool", "content": content, "tool_call_id": call_id}


def test_preserves_paired_tool_call():
    """content=""+tool_calls assistant + its role=tool pair both survive, in order."""
    messages = [
        {"role": "system", "content": "You are helpful."},
        {"role": "user", "content": "What is the status?"},
        _assistant_call("call_1"),
        _tool_result("call_1"),
        {"role": "assistant", "content": "The status is fine."},
    ]

    result = filter_valid_messages(messages)

    assert result == messages, "a complete pair plus plain messages must pass through untouched"
    # order is load-bearing: the tool result must follow its call request
    assert result[2]["tool_calls"][0]["id"] == "call_1"
    assert result[3]["tool_call_id"] == "call_1"


def test_drops_unpaired_tool_call():
    """Design doc 2-1 case 1: unknown-tool-name early return left no result message."""
    orphan_call = _assistant_call("call_missing")
    messages = [
        {"role": "user", "content": "Do the thing"},
        orphan_call,
        {"role": "user", "content": "Hello?"},
    ]

    result = filter_valid_messages(messages)

    assert orphan_call not in result
    assert result == [
        {"role": "user", "content": "Do the thing"},
        {"role": "user", "content": "Hello?"},
    ]


def test_drops_orphaned_tool_result():
    """Design doc 2-1 case 2: window truncation cut the call request away."""
    orphan_result = _tool_result("call_truncated")
    messages = [
        orphan_result,
        {"role": "assistant", "content": "Here is what I found."},
        {"role": "user", "content": "Thanks"},
    ]

    result = filter_valid_messages(messages)

    assert orphan_result not in result
    assert result == [
        {"role": "assistant", "content": "Here is what I found."},
        {"role": "user", "content": "Thanks"},
    ]


def test_existing_fixtures_unaffected():
    """Existing fixtures with NO role="tool" message keep their old behavior.

    These are the three fixtures from the design doc section 3 table that
    contain no role="tool" message, so pass 2 cannot change their outcome.
    (test_tool_role_messages_preserved is deliberately excluded -- its
    fixture carries an orphaned role="tool" message and correctly loses it,
    see design doc section 3 correction / section 5-9.)
    """
    # test_openai_filters_invalid_messages
    fixture_1 = [
        {"role": "system", "content": "You are helpful."},
        {"role": "", "content": "no role"},
        {"content": "missing role key"},
        {"role": "user"},
        {"role": "user", "content": ""},
        {"role": "user", "content": "valid message"},
    ]
    assert filter_valid_messages(fixture_1) == [
        {"role": "system", "content": "You are helpful."},
        {"role": "user", "content": "valid message"},
    ]

    # test_malformed_messages_filtered_out
    fixture_2 = [
        {"role": "user", "content": "valid message"},
        {"role": "assistant"},
        {"content": "orphaned content"},
        {},
        {"role": "", "content": "empty role"},
        {"role": "user", "content": ""},
        {"role": "assistant", "content": "also valid"},
    ]
    assert filter_valid_messages(fixture_2) == [
        {"role": "user", "content": "valid message"},
        {"role": "assistant", "content": "also valid"},
    ]

    # test_none_values_in_messages_filtered_out
    fixture_3 = [
        {"role": None, "content": "text"},
        {"role": "user", "content": None},
        {"role": "user", "content": "valid"},
    ]
    assert filter_valid_messages(fixture_3) == [
        {"role": "user", "content": "valid"},
    ]

    # each of the three matches the old predicate-only result exactly
    for fixture in (fixture_1, fixture_2, fixture_3):
        predicate_only = [m for m in fixture if m.get("role") and m.get("content")]
        assert filter_valid_messages(fixture) == predicate_only


def test_partial_pairing_drops_both():
    """Both halves are judged against the SAME surviving set, never candidates.

    An assistant message with two tool_calls (c1, c2) where only c1 has a
    result: nothing survives. If pass 2 re-checked role="tool" messages
    against the provisional `result_ids` instead of the final
    `kept_assistant_call_ids`, the c1 result would survive alone as an
    orphan -- which is exactly the bug found in design review round 2.
    """
    partial_call = {
        "role": "assistant",
        "content": "",
        "tool_calls": [
            {"id": "c1", "type": "function", "function": {"name": "a", "arguments": "{}"}},
            {"id": "c2", "type": "function", "function": {"name": "b", "arguments": "{}"}},
        ],
    }
    messages = [
        {"role": "user", "content": "Do two things"},
        partial_call,
        _tool_result("c1"),
    ]

    result = filter_valid_messages(messages)

    assert result == [{"role": "user", "content": "Do two things"}], (
        "an assistant message with a partially answered tool_calls list, and "
        "the lone result that answered it, must BOTH be dropped"
    )
