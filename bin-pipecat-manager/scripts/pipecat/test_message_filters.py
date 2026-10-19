"""Pure unit tests for filter_valid_messages() (VOIP-1460).

message_filters.py has no pipecat dependency, so these tests import it
directly with no mocking.
"""

import copy
import itertools

import pytest

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


# ---------------------------------------------------------------------------
# VOIP-1633: tool calls and results without an id (design 2.3)
# ---------------------------------------------------------------------------
def _idless_call(name="get_x", **extra):
    msg = {"role": "assistant", "content": "", "tool_calls": [{"type": "function", "function": {"name": name, "arguments": "{}"}}]}
    msg.update(extra)
    return msg


def _idless_result(content="ok"):
    return {"role": "tool", "content": content}


def _ids(messages):
    """(call id or result id) per tool related row, to compare pairing."""
    out = []
    for m in messages:
        if m.get("role") == "assistant" and m.get("tool_calls"):
            out.append(("call", m["tool_calls"][0]["function"]["name"], m["tool_calls"][0].get("id")))
        elif m.get("role") == "tool":
            out.append(("result", m.get("content"), m.get("tool_call_id")))
    return out


def test_idless_adjacent_pair_is_kept_with_one_generated_id():
    messages = [{"role": "user", "content": "hi"}, _idless_call("a"), _idless_result("ra")]
    result = filter_valid_messages(messages)
    assert [m["role"] for m in result] == ["user", "assistant", "tool"]
    assert result[1]["tool_calls"][0]["id"] == "legacy_call_1" == result[2]["tool_call_id"]


def test_two_serial_idless_pairs_get_distinct_ids_in_order():
    messages = [_idless_call("a"), _idless_result("ra"), _idless_call("b"), _idless_result("rb")]
    result = filter_valid_messages(messages)
    assert _ids(result) == [("call", "a", "legacy_call_1"), ("result", "ra", "legacy_call_1"),
                            ("call", "b", "legacy_call_2"), ("result", "rb", "legacy_call_2")]


def test_idless_pair_between_pairs_with_ids_is_kept():
    messages = [
        _assistant_call("x"), _tool_result("x"),
        _idless_call("a"), _idless_result("ra"),
        _assistant_call("y"), _tool_result("y"),
    ]
    result = filter_valid_messages(messages)
    assert len(result) == 6
    assert result[2]["tool_calls"][0]["id"] == result[3]["tool_call_id"] == "legacy_call_1"


def test_idless_pair_split_by_an_id_row_is_dropped():
    messages = [_idless_call("a"), _assistant_call("x"), _tool_result("x"), _idless_result("ra")]
    result = filter_valid_messages(messages)
    # only the rows that have ids survive; nothing is paired by guessing
    assert _ids(result) == [("call", "get_resource", "x"), ("result", '{"result": "success"}', "x")]


@pytest.mark.parametrize("name, messages", [
    ("parallel A1 A2 R1 R2", [_idless_call("a"), _idless_call("b"), _idless_result("ra"), _idless_result("rb")]),
    ("parallel A1 A2 R2 R1", [_idless_call("a"), _idless_call("b"), _idless_result("rb"), _idless_result("ra")]),
    ("text between call and result", [_idless_call("a"), {"role": "assistant", "content": "wait"}, _idless_result("ra")]),
    ("window cut leaves A2 R1 R2", [_idless_call("b"), _idless_result("ra"), _idless_result("rb")]),
    ("lost result then a later pair", [_idless_call("a"), _idless_call("b"), _idless_result("rb")]),
    ("leading orphan result", [_idless_result("ra"), _idless_call("b"), _idless_result("rb")]),
    ("clean pair then orphan result", [_idless_call("a"), _idless_result("ra"), _idless_result("rb")]),
])
def test_ambiguous_idless_rows_are_all_dropped(name, messages):
    result = filter_valid_messages(messages)
    assert _ids(result) == [], name


def test_trailing_unpaired_call_is_dropped_but_earlier_pairs_stay():
    result = filter_valid_messages([_idless_call("a"), _idless_result("ra"), _idless_call("b")])
    assert _ids(result) == [("call", "a", "legacy_call_1"), ("result", "ra", "legacy_call_1")]


def test_trailing_unpaired_call_may_be_followed_by_an_id_row():
    result = filter_valid_messages([_idless_call("a"), _idless_result("ra"), _idless_call("b"), {"role": "user", "content": "q"}])
    assert _ids(result) == [("call", "a", "legacy_call_1"), ("result", "ra", "legacy_call_1")]
    assert result[-1] == {"role": "user", "content": "q"}


def test_row_with_two_calls_one_without_id_drops_all_idless_rows():
    mixed = {"role": "assistant", "content": "", "tool_calls": [
        {"id": "c1", "type": "function", "function": {"name": "a", "arguments": "{}"}},
        {"type": "function", "function": {"name": "b", "arguments": "{}"}},
    ]}
    result = filter_valid_messages([mixed, _tool_result("c1"), _idless_result("rb")])
    assert _ids(result) == []


def test_row_with_two_idless_calls_and_one_result_is_dropped_not_half_paired():
    """A second call must not vanish silently behind a pair that only labels the first one."""
    two = _idless_call("a")
    two["tool_calls"].append({"type": "function", "function": {"name": "b", "arguments": "{}"}})
    result = filter_valid_messages([{"role": "user", "content": "q"}, two, _idless_result("ra")])
    assert result == [{"role": "user", "content": "q"}]


def test_empty_string_ids_are_idless_like_none():
    call = _idless_call("a")
    call["tool_calls"][0]["id"] = ""
    call2 = _idless_call("b")
    call2["tool_calls"][0]["id"] = None
    res = _idless_result("ra")
    res["tool_call_id"] = ""
    res2 = _idless_result("rb")
    res2["tool_call_id"] = None
    result = filter_valid_messages([call, res, call2, res2])
    assert _ids(result) == [("call", "a", "legacy_call_1"), ("result", "ra", "legacy_call_1"),
                            ("call", "b", "legacy_call_2"), ("result", "rb", "legacy_call_2")]


def test_idless_result_with_empty_content_drops_its_call_too():
    """The existing candidate filter removes an empty tool row first, so its call is a trailing unpaired call."""
    result = filter_valid_messages([_idless_call("a"), _idless_result("")])
    assert _ids(result) == []


def test_valid_history_is_returned_unchanged_and_not_mutated():
    messages = [
        {"role": "system", "content": "s"}, {"role": "user", "content": "q"},
        _assistant_call("c1"), _tool_result("c1"),
        {"role": "assistant", "content": "done"},
    ]
    before = copy.deepcopy(messages)
    assert filter_valid_messages(messages) == before
    assert messages == before


def test_input_with_idless_rows_is_not_mutated_and_filtering_is_idempotent():
    messages = [{"role": "user", "content": "q"}, _idless_call("a"), _idless_result("ra")]
    before = copy.deepcopy(messages)
    once = filter_valid_messages(messages)
    assert messages == before
    assert filter_valid_messages(once) == once


def _build(order, cut, lost):
    """order: sequence of ('A', k) / ('R', k). Returns messages after a window cut and lost results."""
    rows = list(order)[cut:]
    rows = [r for r in rows if not (r[0] == "R" and r[1] in lost)]
    return [_idless_call(f"t{k}") if kind == "A" else _idless_result(f"r{k}") for kind, k in rows]


def _wrong_pairs(order, cut, lost):
    result = filter_valid_messages(_build(order, cut, lost))
    by_id = {}
    for kind, label, row_id in _ids(result):
        by_id.setdefault(row_id, {})[kind] = label
    return [i for i, v in by_id.items() if v.get("call", "t")[1:] != v.get("result", "r")[1:]]


def test_no_wrong_pairing_for_window_cut_alone_or_lost_results_alone():
    """Exhaustive: every order of up to 4 (parallel) calls whose result follows its call.

    A window cut together with a lost result row can fake a clean pair (A2,R1 after A1 was cut); that
    case is a documented residual risk and is not asserted here.
    """
    checked = 0
    for n in (1, 2, 3, 4):
        items = [("A", k) for k in range(1, n + 1)] + [("R", k) for k in range(1, n + 1)]
        for order in itertools.permutations(items):
            pos = {x: i for i, x in enumerate(order)}
            if any(pos[("A", k)] > pos[("R", k)] for k in range(1, n + 1)):
                continue
            for cut in range(len(order)):
                assert _wrong_pairs(order, cut, lost=()) == [], (order, cut)
                checked += 1
            for lost_mask in range(1 << n):
                lost = {k for k in range(1, n + 1) if lost_mask >> (k - 1) & 1}
                assert _wrong_pairs(order, 0, lost) == [], (order, lost)
                checked += 1
    assert checked > 10000
