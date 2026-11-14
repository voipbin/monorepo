"""Shared LLM message-list filtering for run.py and team_flow.py.

Kept deliberately free of pipecat (loguru only) so it can be imported and
unit-tested in isolation. See VOIP-1460 design doc
(docs/plans/2026-09-04-fix-orphaned-tool-messages-design.md) and the VOIP-1633
design doc (docs/plans/2026-10-20-runner-silent-failures-design.md) for the
id-less pair handling.
"""

from loguru import logger


def _is_idless_call(m: dict) -> bool:
    return m.get("role") == "assistant" and bool(m.get("tool_calls")) and any(not tc.get("id") for tc in m["tool_calls"])


def _is_idless_result(m: dict) -> bool:
    return m.get("role") == "tool" and not m.get("tool_call_id")


def _is_plain(m: dict) -> bool:
    """A row that is neither a tool-call request nor a tool result."""
    return m.get("role") != "tool" and not m.get("tool_calls")


def _only_plain_rows_between(candidates: list[dict], i: int, j: int) -> bool:
    return all(_is_plain(candidates[k]) for k in range(i + 1, j))


def _repair_idless_pairs(candidates: list[dict]) -> list[dict]:
    """Handle tool calls and results that carry no id (VOIP-1633, design 2.3).

    ai-manager writes the call row first and the result row after the tool ran, so one executed call
    is a single-call assistant row directly followed by its tool row. Such a pair gets one generated
    id (legacy_call_<n>) and is kept: dropping it would hide an executed call from the model, which
    could then repeat a side effect tool. Walk the id-less rows in order, expecting [call, result]
    pairs; one unpaired call at the end is dropped. Any other shape (orphan result, two calls in a row,
    several calls in one row, rows that are not adjacent) cannot be paired without guessing, so every
    id-less row is dropped. Rows with ids are never touched; the input is not mutated.

    The call and its result need not be adjacent (VOIP-1733): plain rows (a sentence, a user row) stored
    between them are allowed, because they cannot be tool rows of another call, so the pairing is as
    certain as for adjacent rows. _restore_tool_adjacency then moves those rows out from between the two.
    """
    idless = [i for i, m in enumerate(candidates) if _is_idless_call(m) or _is_idless_result(m)]
    if not idless:
        return candidates

    pairs = []
    p = 0
    while p < len(idless):
        i = idless[p]
        call = candidates[i]
        if not _is_idless_call(call):
            pairs = []
            break
        if p + 1 >= len(idless):
            break
        j = idless[p + 1]
        paired = _only_plain_rows_between(candidates, i, j) and len(call["tool_calls"]) == 1 and _is_idless_result(candidates[j])
        if not paired:
            pairs = []
            break
        pairs.append((i, j))
        p += 2

    repaired = {}
    for n, (i, j) in enumerate(pairs, start=1):
        legacy_id = f"legacy_call_{n}"
        repaired[i] = {**candidates[i], "tool_calls": [{**candidates[i]["tool_calls"][0], "id": legacy_id}]}
        repaired[j] = {**candidates[j], "tool_call_id": legacy_id}

    result = []
    for i, m in enumerate(candidates):
        if i in repaired:
            result.append(repaired[i])
        elif i not in idless:
            result.append(m)
    logger.warning(f"[tool_history] id-less tool rows. kept_pairs={len(pairs)} dropped_rows={len(idless) - 2 * len(pairs)}")
    return result


def _restore_tool_adjacency(candidates: list[dict]) -> list[dict]:
    """Make every tool result directly follow its tool call (VOIP-1733).

    ai-manager stores the call row when the tool starts, the result row when it returns, and the bot
    sentence row when the sentence event arrives, so a sentence the model produced together with the call
    can be stored between the two. OpenAI rejects such a history (a tool_calls message must be followed by
    its tool messages), and pipecat's Gemini adapter pairs function_call and function_response turns the same way.

    Input: candidates after _repair_idless_pairs, so every tool_calls entry and tool row has an id.
    A. Claim: for each call row in order, claim for each of its ids the first unclaimed tool row with that
       id after the call. All ids answered: the call forms a block and the claims are kept. Otherwise the
       call is unanswerable and releases its claims.
    B. Hoist: the plain assistant rows (sentences) that follow a block's call directly (rows claimed by a
       block are transparent), before its last result, are emitted before the call. This is the order a model
       emits them in, and it is kept as a separate row because the Gemini adapter drops the content of a
       tool_calls row. Anything after the last result (the answer) is never moved.
    C. Emit once: a block call is emitted as hoisted rows, call, results (in tool_calls order). Rows that sat
       between a call and its results stay in their order after the block. Unanswerable calls and tool rows
       no block claimed (orphan, placed before their call, duplicate) are dropped.
    The result is idempotent and the input is not mutated; the same row objects are returned. A result
    placed before its call is dropped with the call (it used to pass the pairing check below).
    """
    n = len(candidates)

    def is_call(m: dict) -> bool:
        return m.get("role") == "assistant" and bool(m.get("tool_calls"))

    claimed = {}  # tool row index -> call row index
    blocks = {}  # call row index -> list of result row indexes in tool_calls order
    for i, m in enumerate(candidates):
        if not is_call(m):
            continue
        taken = []
        for tc in m["tool_calls"]:
            found = next(
                (
                    k
                    for k in range(i + 1, n)
                    if k not in claimed
                    and k not in taken
                    and candidates[k].get("role") == "tool"
                    and candidates[k].get("tool_call_id") == tc.get("id")
                ),
                None,
            )
            if found is None:
                taken = None
                break
            taken.append(found)
        if taken is None:
            continue
        for k in taken:
            claimed[k] = i
        blocks[i] = taken

    hoisted = {}  # call row index -> sentence row indexes
    hoisted_rows = set()
    for i, results in blocks.items():
        last = max(results)
        rows = []
        k = i + 1
        while k < last:
            if k in claimed:
                k += 1
            elif candidates[k].get("role") == "assistant" and not candidates[k].get("tool_calls") and k not in hoisted_rows:
                rows.append(k)
                hoisted_rows.add(k)
                k += 1
            else:
                break
        hoisted[i] = rows

    output = []
    for i, m in enumerate(candidates):
        if i in blocks:
            output.extend(candidates[k] for k in hoisted[i])
            output.append(m)
            output.extend(candidates[k] for k in blocks[i])
        elif i in claimed or i in hoisted_rows:
            continue
        elif is_call(m) or m.get("role") == "tool":
            continue  # unanswerable call, or a tool row that no call claimed
        else:
            output.append(m)
    return output


def filter_valid_messages(messages: list[dict]) -> list[dict]:
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

    Structural invariant: `kept_assistant_call_ids` is computed with NO side
    effect on the output (pass 1), then the final list is emitted from a
    SINGLE loop that is the only place `append` is ever called (pass 2) --
    this is what keeps the two halves of a pair from being judged by
    different criteria (see design doc section 2-1 for the two earlier
    drafts that broke by emitting output from more than one place).
    """
    candidates = [m for m in messages if m.get("role") and (m.get("content") or m.get("tool_calls"))]
    candidates = _repair_idless_pairs(candidates)
    candidates = _restore_tool_adjacency(candidates)
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
