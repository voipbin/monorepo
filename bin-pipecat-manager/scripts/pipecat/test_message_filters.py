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


# ---------------------------------------------------------------------------
# VOIP-1733: a tool result must directly follow its tool call
#
# CHAR = characterisation (passes on the code before this change), RED = fails on it.
# ---------------------------------------------------------------------------
def _user(text="q"):
    return {"role": "user", "content": text}


def _say(text="I'll connect you now."):
    return {"role": "assistant", "content": text}


def _multi_call(*ids):
    return {
        "role": "assistant",
        "content": "",
        "tool_calls": [{"id": i, "type": "function", "function": {"name": f"tool_{i}", "arguments": "{}"}} for i in ids],
    }


def assert_valid_order(out, original):
    """The order rules every provider relies on, applied to the output of the filter.

    1. Each assistant row with tool_calls is directly followed by one tool row per call id, in tool_calls order.
    2. No tool row is anywhere else (no orphan, no duplicate).
    3. Every plain row of the input (not a call row, not a tool row, with content) is present exactly once.
       Only these moves are allowed: a hoisted sentence before its call, a row that sat between a call
       and its result after the result block, and results in tool_calls order. Rows dropped on purpose are
       unanswerable calls, orphan, misplaced and duplicate results, and ambiguous id-less rows.
    """
    i = 0
    while i < len(out):
        m = out[i]
        if m["role"] == "assistant" and m.get("tool_calls"):
            ids = [tc["id"] for tc in m["tool_calls"]]
            following = out[i + 1:i + 1 + len(ids)]
            assert [f.get("tool_call_id") for f in following] == ids and all(f["role"] == "tool" for f in following), (i, out)
            i += 1 + len(ids)
            continue
        assert m["role"] != "tool", ("orphan or misplaced tool row", i, out)
        i += 1
    plain_in = [id(m) for m in original if m.get("role") not in (None, "tool") and m.get("content") and not m.get("tool_calls")]
    plain_out = [id(m) for m in out if m["role"] != "tool" and not m.get("tool_calls")]
    assert sorted(plain_in) == sorted(plain_out)


def _check(messages, expected):
    before = copy.deepcopy(messages)
    out = filter_valid_messages(messages)
    assert messages == before, "the input must not be mutated"
    assert out == expected
    assert_valid_order(out, messages)
    assert filter_valid_messages(out) == out, "filtering is idempotent"
    return out


def test_plain_history_unchanged():  # S1, CHAR
    msgs = [_user("a"), _say("b"), _user("c")]
    _check(msgs, msgs)


def test_followup_turn_after_clean_tool_turn():  # S2, CHAR
    msgs = [_user(), _assistant_call("a"), _tool_result("a"), _say("done"), _user("next")]
    _check(msgs, msgs)


def test_sentence_between_call_and_result_moves_before_call():  # S3, RED
    u, c, s, r, a = _user(), _assistant_call("a"), _say(), _tool_result("a"), _say("connected")
    _check([u, c, s, r, a], [u, s, c, r, a])


def test_answer_and_trailing_sentence_are_not_moved():  # S3, CHAR
    u, c, r, a = _user(), _assistant_call("a"), _tool_result("a"), _say("answer")
    _check([u, c, r, a], [u, c, r, a])
    _check([c, r, a], [c, r, a])


def test_serial_calls_unchanged():  # S4, CHAR
    msgs = [_user(), _assistant_call("a"), _tool_result("a"), _assistant_call("b"), _tool_result("b"), _say("done")]
    _check(msgs, msgs)


def test_serial_calls_sentence_in_first_pair():  # S4, RED
    ca, s, ra, cb, rb, a = _assistant_call("a"), _say(), _tool_result("a"), _assistant_call("b"), _tool_result("b"), _say("done")
    _check([ca, s, ra, cb, rb, a], [s, ca, ra, cb, rb, a])


def test_parallel_separate_rows_become_serial_pairs():  # S5, RED
    ca, cb, ra, rb = _assistant_call("a"), _assistant_call("b"), _tool_result("a"), _tool_result("b")
    _check([ca, cb, ra, rb], [ca, ra, cb, rb])


def test_parallel_rows_with_sentence_mixed_tool_names():  # S5, S14, RED
    ca = _assistant_call("a", name="mcp_search_flights")
    cb = _assistant_call("b", name="set_variables")
    s, ra, rb = _say(), _tool_result("a"), _tool_result("b")
    out = _check([ca, cb, s, ra, rb], [ca, ra, s, cb, rb])
    assert out[0]["tool_calls"][0]["function"]["name"] == "mcp_search_flights"
    assert out[3]["tool_calls"][0]["function"]["name"] == "set_variables"
    # the sentence also stays valid when it sits between the two results
    ca, cb, ra, s, rb = _assistant_call("a"), _assistant_call("b"), _tool_result("a"), _say(), _tool_result("b")
    _check([ca, cb, ra, s, rb], [ca, ra, s, cb, rb])


def test_multi_id_row_results_follow_in_call_order():  # S5, RED
    c, ra, rb = _multi_call("a", "b"), _tool_result("a"), _tool_result("b")
    _check([c, rb, ra], [c, ra, rb])


def test_multi_id_row_with_missing_result_is_dropped():  # S5, CHAR
    _check([_multi_call("a", "b"), _tool_result("a")], [])


def test_duplicate_result_keeps_first():  # S5, RED
    c, r1 = _assistant_call("a"), _tool_result("a", "first")
    r2 = _tool_result("a", "second")
    _check([c, r1, r2], [c, r1])


def test_duplicate_call_id_one_result_drops_second_call():  # S5, RED
    c1, c2, r = _assistant_call("a"), _assistant_call("a"), _tool_result("a")
    _check([c1, c2, r], [c1, r])


def test_duplicate_call_id_two_results_both_kept():  # S5, CHAR
    msgs = [_assistant_call("a"), _tool_result("a", "1"), _assistant_call("a"), _tool_result("a", "2")]
    _check(msgs, msgs)


def test_unanswerable_call_releases_its_claims():  # S5, CHAR (mutation target)
    # c1 claims the result of b first and then finds no result for a: it is dropped and must give b back to c2
    c1, c2, r = _multi_call("b", "a"), _assistant_call("b"), _tool_result("b")
    _check([c1, c2, r], [c2, r])


def test_failed_result_kept_and_adjacent():  # S6, CHAR
    msgs = [_user(), _assistant_call("a"), _tool_result("a", '{"result": "failed", "message": "boom"}'), _say("sorry")]
    _check(msgs, msgs)


def test_unpaired_call_with_sentence_and_followup():  # S7, S8, CHAR
    u, c, s, u2 = _user(), _assistant_call("a", name="no_such_tool"), _say(), _user("again")
    _check([u, c, s, u2], [u, s, u2])


def test_result_before_its_call_is_dropped():  # S9, RED
    _check([_tool_result("a"), _assistant_call("a")], [])


def test_id_bearing_window_cut_drops_only_the_orphan_result():  # S10, CHAR
    ra, cb, rb = _tool_result("a"), _assistant_call("b"), _tool_result("b")
    _check([ra, cb, rb], [cb, rb])


def test_user_row_between_call_and_result_moves_after_result():  # S11, RED
    u, c, u2, r, a = _user(), _assistant_call("a"), _user("hurry"), _tool_result("a"), _say("done")
    _check([u, c, u2, r, a], [u, c, r, u2, a])


def test_card_call_and_result_pass_through_unchanged():  # S13, RED (order), CHAR (content)
    call = _assistant_call("a", name="emit_info_card")
    call["tool_calls"][0]["function"]["arguments"] = "{}"
    result = _tool_result("a", '{"result": "success"}')
    s = _say()
    out = _check([call, s, result], [s, call, result])
    assert out[1] is call and out[2] is result


def test_filter_is_idempotent_and_does_not_mutate_input():  # S15, idempotence CHAR, checker RED
    c, s, r = _assistant_call("a"), _say(), _tool_result("a")
    inputs = [
        [_user(), c, s, r, _say("x")],
        [_assistant_call("a"), _assistant_call("b"), _tool_result("a"), _tool_result("b")],
        [_user(), _assistant_call("a"), _user("hurry"), _tool_result("a")],
        [_multi_call("a", "b"), _tool_result("b"), _tool_result("a")],
        [_idless_call("a"), _say(), _idless_result("ra")],
    ]
    for msgs in inputs:
        before = copy.deepcopy(msgs)
        out = filter_valid_messages(msgs)
        assert msgs == before
        assert filter_valid_messages(out) == out
        assert_valid_order(out, msgs)


def test_idless_pair_with_sentence_between_is_kept():  # S3 id-less, RED
    c, s, r = _idless_call("a"), _say("wait"), _idless_result("ra")
    out = filter_valid_messages([c, s, r])
    assert out[0] is s
    assert _ids(out) == [("call", "a", "legacy_call_1"), ("result", "ra", "legacy_call_1")]
    assert [m["role"] for m in out] == ["assistant", "assistant", "tool"] and out[1].get("tool_calls")
    assert_valid_order(out, [c, s, r])


def test_idless_pair_with_user_row_between_is_kept():  # S11 id-less, RED
    u, c, u2, r = _user(), _idless_call("a"), _user("hurry"), _idless_result("ra")
    out = filter_valid_messages([u, c, u2, r])
    assert [m["role"] for m in out] == ["user", "assistant", "tool", "user"]
    assert _ids(out) == [("call", "a", "legacy_call_1"), ("result", "ra", "legacy_call_1")]
    assert_valid_order(out, [u, c, u2, r])


def test_idless_call_without_result_is_dropped():  # S8 id-less, CHAR
    u, c, s, u2 = _user(), _idless_call("a"), _say(), _user("again")
    _check([u, c, s, u2], [u, s, u2])


def test_idless_parallel_calls_are_dropped():  # S5 id-less, CHAR (remaining limit)
    _check([_idless_call("a"), _idless_call("b"), _idless_result("ra"), _idless_result("rb")], [])


def test_unanswerable_call_between_stops_the_hoist_scan():  # S5, characterises the scan rule
    ca, cx, s, ra = _assistant_call("a"), _assistant_call("x"), _say(), _tool_result("a")
    _check([ca, cx, s, ra], [ca, ra, s])
