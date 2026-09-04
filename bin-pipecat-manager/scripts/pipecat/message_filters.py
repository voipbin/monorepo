"""Shared LLM message-list filtering for run.py and team_flow.py.

Kept deliberately dependency-free (no pipecat, no loguru) so it can be
imported and unit-tested in isolation. See VOIP-1460 design doc
(docs/plans/2026-09-04-fix-orphaned-tool-messages-design.md).
"""


def filter_valid_messages(messages):
    """Keep role+content-or-tool_calls messages, then drop any tool_calls /
    tool-result that isn't actually paired. A tool-call-request message
    survives only if EVERY one of its tool_calls has a matching role="tool"
    result somewhere in the candidates; a role="tool" message survives only
    if its tool_call_id belongs to a tool-call-request message that ALSO
    survives that same check. Both directions are decided against the same
    `kept_assistant_call_ids` set -- there is exactly one place (the loop
    below) where a message is appended to the output, so there is no way
    for the two halves of a pair to be judged by different criteria. This is
    what makes the fix safe regardless of WHY a pair might be incomplete (an
    unknown-tool-name error that never got recorded, an old-side window
    truncation cut, or anything else not yet discovered) -- see VOIP-1460
    design doc section 2-1.

    (This function went through two broken drafts in design review before
    reaching this shape -- both broke by emitting output from more than one
    loop/list, which let a message dropped in one place resurface via a
    stray `else: append` in another. The fix is structural: compute
    `kept_assistant_call_ids` first with NO side effect on the output, then
    emit the final list from a SINGLE loop that is the only place `append`
    is ever called.)
    """
    candidates = [m for m in messages if m.get("role") and (m.get("content") or m.get("tool_calls"))]
    result_ids = {m.get("tool_call_id") for m in candidates if m.get("role") == "tool"}

    # Pass 1: compute-only, no emission. Which assistant tool_calls entries
    # are answerable (every one of that message's ids has a matching
    # candidate result)? Conservative: if ANY tool_call in a message lacks a
    # matching result, none of that message's ids are added -- a provider
    # that requires every tool_calls entry to be answered would 400 on a
    # partial match anyway, and partial-answer messages are not something
    # this codebase currently produces (each ToolHandle call carries exactly
    # one ToolCall -- tool.go:39-47).
    kept_assistant_call_ids = set()
    for m in candidates:
        if m.get("role") == "assistant" and m.get("tool_calls"):
            ids = [tc.get("id") for tc in m["tool_calls"]]
            if all(cid in result_ids for cid in ids):
                kept_assistant_call_ids.update(ids)

    # Pass 2: the ONLY place output is emitted. Re-applies the same
    # completeness check to assistant tool_calls messages (against the now-
    # final `kept_assistant_call_ids`, not the provisional `result_ids`),
    # and checks role="tool" messages against that identical set -- so both
    # halves of a pair are always judged by the same criterion, in the same
    # loop, with no append site outside this loop.
    final_messages = []
    for m in candidates:
        role = m.get("role")
        if role == "assistant" and m.get("tool_calls"):
            if all(tc.get("id") in kept_assistant_call_ids for tc in m["tool_calls"]):
                final_messages.append(m)
            # else: drop -- unpaired tool-call request, would 400 downstream
        elif role == "tool":
            if m.get("tool_call_id") in kept_assistant_call_ids:
                final_messages.append(m)
            # else: drop -- orphaned tool result, its call was never kept
        else:
            final_messages.append(m)
    return final_messages
