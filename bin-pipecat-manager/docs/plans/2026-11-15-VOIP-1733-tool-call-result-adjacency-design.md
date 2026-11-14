# VOIP-1733: restore tool call / tool result adjacency in the replayed history

## 1. Issue analysis

### 1.1 Claim and verdict

Claim: a text aicall fails on the turn after a tool call with an OpenAI 400
("an assistant message with tool_calls must be followed by tool messages"). The stored
history was `[user, assistant(tool_calls), assistant(sentence), tool(result), assistant(answer)]`.
Verdict: valid (checked against origin/main `2ec2fac22`). The stored order above comes from the
report; the reason why the sentence row lands between the call and the result is a hypothesis from
code reading (1.2 item 2), not a reproduction. The repair does not depend on which of the
candidate causes in 1.3 is the real one: it restores adjacency whatever put another row between the two.

### 1.2 Code facts (all re-read on origin/main)

1. `bin-ai-manager/pkg/aicallhandler/tool.go` stores the call row (`role=assistant`, empty content,
   one `tool_calls` entry) when tool execution starts, and the result row after the tool returned.
2. `bin-ai-manager/pkg/messagehandler/event.go` (`EventPMMessageBotLLM`) stores the bot sentence row when
   the pipecat event arrives. A sentence the model produced together with the tool call is therefore
   stored after the call row. If the tool is slow, it is also stored before the result row.
3. `dbhandler/message.go` `MessageCreate` sets `TMCreate = TimeNow()` at insert time; `MessageList`
   orders by `tm_create DESC`. The order is the insert order (equal timestamps are an unverified
   second candidate; not changed here).
4. `aicallhandler/start.go` `getPipecatcallMessages` (`start.go:776-903`) reads twice: up to 20 system rows
   (`:789`) and the newest 100 rows that are not system and not `listen_internal` (`:800`), then cuts rows
   before the Insight session start (`:818-827`), so the window is "100 rows or everything since the
   session start, whichever is smaller". It skips `notification`, strips `blocks` from `emit_info_card`
   results and neuters its call arguments. It does not reorder.
5. `bin-pipecat-manager/scripts/pipecat/message_filters.py` `filter_valid_messages` checks only that a
   call and its result both exist. It never checks adjacency. It is called from `run.py`
   (`create_llm_service`, team start messages) and `team_flow.py`, so one change covers every path.
6. `_repair_idless_pairs` pairs id-less rows only when `j == i + 1`; any other shape drops all id-less
   rows, which hides an executed call from the model (a side effect tool could be repeated).
7. Provider behaviour: the OpenAI chat API rule (every assistant `tool_calls` message must be followed by
   its tool messages) is taken from the error text of the report and the API documentation; the
   pipecat OpenAI path sends the list as is. pipecat's Gemini adapter
   (`GeminiLLMAdapter._from_standard_message`, `if tool_calls ... elif role == tool ... elif str`)
   converts an assistant message that has `tool_calls` to `function_call` parts only and **drops its
   `content`** (checked in pipecat 1.12.0, the version pinned in the lockfile, `adapters/services/gemini_adapter.py`
   lines 490-523: `if msg.get("tool_calls")` / `elif role == "tool"` / `elif isinstance(content, str)`; the
   same chain exists in 1.4.0). That a text turn between `function_call` and
   `function_response` is rejected by Gemini is NOT verified; the chosen order never produces it.
8. `ToolHandle` is one RPC per call and writes one row per call (`tool.go:115`, one `ToolCall` per row).
   Parallel calls of one model response are therefore stored as separate rows, `[C_a, C_b, R_a, R_b]`
   or interleaved, which OpenAI also rejects for `C_a` even with no sentence row. The same applies to a
   user row stored while a slow tool runs. The repair handles these shapes (2.3 rule 3).
9. Entry points audited: `run.py` (`create_llm_service`, team start messages), `team_flow.py`, and
   `main.py` (only hands `llm_messages` on). All text and team paths go through `filter_valid_messages`.
   The voice path with in-memory history is not audited.

### 1.3 Candidate causes and limits

- Sentence row stored after the call row (hypothesis above); parallel calls stored as separate rows
  (item 8); a user row stored during a slow tool; equal or skewed `tm_create` between rows written by
  different pods (`datetime(6)`, no secondary sort key; rare, not changed, and a later read-time reorder
  does not depend on it for rows that carry ids).
- `getPipecatcallMessages` has no `deleted` filter (`start.go` lists, and `listen.go` notes the same).
  A soft-deleted call or result row could split a pair; the pairing step drops the other half, so it
  fails safe. Not changed.
- The 100 row window can cut a pair at its old end. The reorder runs on the fetched window only, so a
  moved sentence cannot leave the window (it moves toward the call, within the window); an orphan half is dropped.

### 1.4 Not confirmed

Exact event timing of the sentence row, equal `tm_create` ordering, the voice call path (single
pipecatcall, in-memory history), providers other than OpenAI and Gemini, and how often a model
emits a sentence together with a call. Not claimed by this change.

## 2. Design

### 2.1 Decision: reorder at replay time in `filter_valid_messages`

Add one step to `filter_valid_messages`, before the pairing check, that rebuilds the order so each
tool result directly follows its call. This repairs histories that were already stored in the wrong
order (A2) and covers the three call sites. Storing in the right order is not done: the sentence
row is created by a different service event path, and fixing the stored `tm_create` would need
updates of rows that other readers (UI, summaries) also use. The replay side is the last line of
defence and is needed for existing rows anyway.

### 2.2 Where the sentence goes (A4)

The filter is provider independent (OpenAI, Gemini, Grok, OpenRouter, local, custom all use it), so the
chosen order must be valid for every provider.

| Option | OpenAI | Gemini (adapter) | Matches model output | Verdict |
|---|---|---|---|---|
| Move the sentence after the result | valid | valid | no: the reply text would follow the result and change the meaning order | rejected |
| Merge the sentence into the call row `content` | valid, closest to the model output | **text dropped** (the adapter ignores `content` when `tool_calls` exists) | yes for OpenAI only | rejected: loses what the user was told on Gemini |
| Drop the sentence on replay | valid | valid | loses text | rejected |
| Move the sentence before the call | valid | valid (text turn, then function_call turn, then function_response turn) | yes: text then call is the order a model emits | chosen |

Parallel calls (`[C_a, C_b, R_a, R_b]`, separate rows because each call is its own RPC and row) are
turned into `[C_a, R_a, C_b, R_b]`. The alternative of merging consecutive call rows into one assistant
row with several `tool_calls` is closer to the model output, but whether two rows came from one model
response is not stored, so it would be an inference. Both providers accept the serial form. The Gemini
parallel-merge helper only acts when thought signatures exist; the rows carry none, so it is not affected.
Merging is not done (no new inference, simplest valid form).

### 2.3 Rules (algorithm)

Input rows are the candidates after the existing role/content-or-tool_calls filter and the id-less repair.
At this point every remaining `tool_calls` entry and every `tool` row has an id.

Step A, claim. For each call row C at index i (assistant, non-empty `tool_calls`), in index order:
for each id of C in `tool_calls` order, claim the first `tool` row with that id at an index greater than i
that no earlier call claimed. If every id got a result, C forms a block (C, its results in `tool_calls`
order) and the claims are confirmed. Otherwise C is unanswerable and releases every result it tentatively
claimed, so a later call with the same id can still claim it.

Step B, hoist. For a block of C with last result index L, scan k from i+1: a row claimed as a result of
any block is transparent; a plain assistant row (`role=assistant`, no `tool_calls`) with k < L and not yet
hoisted by an earlier block is hoisted (kept in original order); any other row stops the scan. Only this
contiguous run of text rows between the call and its last result is hoisted, so a normal
`[C, R, answer]` is unchanged (the answer lies after L) and so is `[C, R, S]`.

Step C, emit. Walk the rows once, with one `emitted` set so a row is output at most once:
- a block call: emit its hoisted rows, then C, then its results;
- a claimed result or hoisted row: skipped (already emitted with its block);
- an unanswerable call, and every `tool` row no block claimed (orphan, result placed before its call,
  duplicate result of an id): dropped;
- every other row (user, system, other assistant text, a row between call and result that was not
  hoisted): emitted at its own position. A user row stored during a slow tool therefore ends up after
  the result block.

Results:
- `[C, S, R, A]` gives `[S, C, R, A]` (the defect). `[C, R, A]`, `[C, R, S]` unchanged.
- `[C_a, C_b, S, R_a, R_b]` gives `[C_a, R_a, S, C_b, R_b]`; which call the sentence belongs to is not
  stored and is not resolved, the only goal is a valid order.
- `[C, U, R]` (user row) gives `[C, R, U]`.
- A multi-id call row followed by its results in any order gives results in `tool_calls` order.
- Result before call `[R, C]`: R dropped, C dropped (no result after it).
- Duplicate result for one id: the first after the call is kept, others dropped. A call row with no
  result left to claim is dropped, so with one result for a duplicated call id the second call row is
  dropped; with two results (`[C1(a), R1(a), C2(a), R2(a)]`) each call claims one and both stay.
- Termination and idempotence: one pass, linear; the output has no row between a call and its last result,
  so a second pass finds nothing to hoist or drop and returns an equal list.

The existing two-pass pairing check stays as it is and runs after this step; it only sees well formed
pairs. The function does not mutate its input and returns the same row objects (rows repaired by the id-less step are new dicts).

Id-less rows: `_repair_idless_pairs` runs first. Its adjacency test `j == i + 1` becomes "every row between
the call and the result is a plain row (not a tool row, not an assistant row with `tool_calls`)", so a
sentence or user row stored between an id-less call and its result no longer hides the executed call.
The walk and the legacy ids are unchanged. Remaining limit, unchanged: an id-less pair with an id-bearing
or another id-less tool row in between, an id-less call row with several calls, or two id-less calls in a
row cannot be paired without guessing, and all id-less rows are dropped as before (the call history is
then not shown to the model). Tests cover id-less variants of S3, S5 and S8.

Window cut and `deleted` rows are as in 1.3: only the fetched rows are reordered and an orphan half is dropped.

### 2.4 Not changed

- Go `getPipecatcallMessages` (no reorder, no new reads). `blocks` removal and argument neuter stay there.
- The example sentence in the common system prompt (`aicallhandler/main.go`): behaviour change of
  the prompt is out of scope; proposed as a one-line follow-up.
- No flag, no schema change, no new dependency. No ARI, SIP, RTP or Kamailio code is touched.

## 3. Detailed design and tests

### 3.1 Implementation

`message_filters.py`: new `_restore_tool_adjacency(candidates)` (steps A to C of 2.3); `_repair_idless_pairs`
adjacency relaxed; `filter_valid_messages` becomes repair id-less, restore adjacency, then the existing
two-pass pairing. Linear with an index of result rows by id. The red input of the defect test is the
hypothesised stored order of 1.1; the same tests also cover the other shapes of 1.3, which do not depend on that hypothesis.

An existing test changes: `test_ambiguous_idless_rows_are_all_dropped`, case `"text between call and
result"` (`[idless C, assistant "wait", idless R]`, now expected to be all dropped) becomes a kept pair
with the sentence hoisted (`[wait, C(legacy id), R(legacy id)]`). It moves out of the ambiguous list into
its own test. This is the intended behaviour change of the id-less relaxation (2.3). The existing `_repair_idless_pairs` warning (`kept_pairs`, `dropped_rows`) is unchanged and its counts stay correct because a kept pair is still two id-less rows.

### 3.2 Shared invariant checker

`test_message_filters.py` gets one helper `assert_valid_order(out, original)` applied to the output of every
new test: every assistant `tool_calls` row is directly followed by its tool rows in `tool_calls` order, no
tool row is an orphan, every non-tool row of the input is still present exactly once, and the plain rows keep
their relative order except for the allowed moves (results re-ordered to `tool_calls` order, a hoisted sentence before its call, and a row that sat
between a call and its result now after the result block). Rows dropped on purpose are the allowed
exceptions: an unanswerable call, an orphan result, a result placed before its call, a duplicate result, and
id-less rows dropped as ambiguous. `pipecat_realtest` carries
its own copy (the directory is isolated from `scripts/pipecat`, see S16).

### 3.3 RED plan (test first)

1. Write all new Python tests first, run `pytest scripts/pipecat/test_message_filters.py` against the
   unchanged `message_filters.py` (origin/main), and save the failing output as the RED evidence on the card.
2. Concrete Python tests, input, expected output and class against origin/main (U user, A answer, S
   sentence, `C(x)` call with id x, `R(x)` its result, `C0`/`R0` id-less; classes were checked by running
   the current filter on a scratch copy):

   | Test | S | Input | Expected after | Before | Function |
   |---|---|---|---|---|---|
   | plain history | S1 | `[U, A, U2]` | unchanged | CHAR | `test_plain_history_unchanged` |
   | follow-up after clean turn | S2 | `[U, C(a), R(a), A, U2]` | unchanged | CHAR | `test_followup_turn_after_clean_tool_turn` |
   | sentence between call and result | S3 | `[U, C(a), S, R(a), A]` | `[U, S, C(a), R(a), A]` | RED | `test_sentence_between_call_and_result_moves_before_call` |
   | answer / trailing sentence not moved | S3 | `[U, C(a), R(a), A]`, `[C(a), R(a), S]` | unchanged | CHAR | `test_answer_and_trailing_sentence_are_not_moved` |
   | serial calls | S4 | `[C(a), R(a), C(b), R(b)]` | unchanged | CHAR | `test_serial_calls_unchanged` |
   | serial with sentence in first pair | S4 | `[C(a), S, R(a), C(b), R(b), A]` | `[S, C(a), R(a), C(b), R(b), A]` | RED | `test_serial_calls_sentence_in_first_pair` |
   | parallel separate rows | S5 | `[C(a), C(b), R(a), R(b)]` | `[C(a), R(a), C(b), R(b)]` | RED | `test_parallel_separate_rows_become_serial_pairs` |
   | parallel with sentence | S5, S14 | `[C(a), C(b), S, R(a), R(b)]` (names `mcp_x`, built-in) | `[C(a), R(a), S, C(b), R(b)]` | RED | `test_parallel_rows_with_sentence_mixed_tool_names` |
   | multi-id row, results reversed | S5 | `[C(a,b), R(b), R(a)]` | `[C(a,b), R(a), R(b)]` | RED | `test_multi_id_row_results_follow_in_call_order` |
   | multi-id row, one result missing | S5 | `[C(a,b), R(a)]` | `[]` | CHAR | `test_multi_id_row_with_missing_result_is_dropped` |
   | duplicate result | S5 | `[C(a), R(a), R(a)]` | one `R(a)` | RED | `test_duplicate_result_keeps_first` |
   | duplicate call id, one result | S5 | `[C(a), C(a), R(a)]` | `[C(a), R(a)]` | RED | `test_duplicate_call_id_one_result_drops_second_call` |
   | duplicate call id, two results | S5 | `[C(a), R(a), C(a), R(a)]` | unchanged | CHAR | `test_duplicate_call_id_two_results_both_kept` |
   | claim release (mutation target) | S5 | `[C(a,b), C(b), R(b)]` | `[C(b), R(b)]` | CHAR | `test_unanswerable_call_releases_its_claims` |
   | failed result | S6 | `[C(a), R(a, result failed), A]` | unchanged | CHAR | `test_failed_result_kept_and_adjacent` |
   | call without result then follow-up (unknown tool name is the same input) | S7, S8 | `[U, C(a), S, U2]` | `[U, S, U2]` | CHAR | `test_unpaired_call_with_sentence_and_followup` |
   | result before its call | S9 | `[R(a), C(a)]` | `[]` | RED | `test_result_before_its_call_is_dropped` |
   | id-bearing window cut | S10 | `[R(a), C(b), R(b)]` | `[C(b), R(b)]` | CHAR | `test_id_bearing_window_cut_drops_only_the_orphan_result` |
   | user row between | S11 | `[U, C(a), U2, R(a), A]` | `[U, C(a), R(a), U2, A]` | RED | `test_user_row_between_call_and_result_moves_after_result` |
   | emit_info_card pass-through | S13 | `[C(card, args "{}"), S, R(no blocks)]` | `[S, C, R]`, C and R equal to the input rows | RED (order), CHAR (content) | `test_card_call_and_result_pass_through_unchanged` |
   | idempotent, input unchanged | S15 | every defect input above | `f(f(x)) == f(x)`, input deep-equal, checker passes | idempotence CHAR, checker RED | `test_filter_is_idempotent_and_does_not_mutate_input` |
   | id-less sentence between | S3 | `[C0, S, R0]` | `[S, C(legacy), R(legacy)]` | RED | `test_idless_pair_with_sentence_between_is_kept` |
   | id-less user between | S11 | `[U, C0, U2, R0]` | `[U, C(legacy), R(legacy), U2]` | RED | `test_idless_pair_with_user_row_between_is_kept` |
   | id-less call without result | S8 | `[U, C0, S, U2]` | `[U, S, U2]` | CHAR | `test_idless_call_without_result_is_dropped` |
   | id-less parallel (remaining limit) | S5 | `[C0, C0, R0, R0]` | `[]` | CHAR | `test_idless_parallel_calls_are_dropped` |

   The RED evidence file on the card lists the actual pass or fail per test of step 1.
3. Mutation checks on a scratch copy, each must turn the named function red:
   - remove the hoist (step B): `test_sentence_between_call_and_result_moves_before_call`, `test_serial_calls_sentence_in_first_pair`;
   - drop of a tool row that no block claimed (step C, tool side) removed: `test_result_before_its_call_is_dropped`, `test_duplicate_result_keeps_first`;
   - drop of an unanswerable call (step C, call side) removed: `test_multi_id_row_with_missing_result_is_dropped`, `test_duplicate_call_id_one_result_drops_second_call`;
   - claim release removed: `test_unanswerable_call_releases_its_claims`;
   - the `k < L` bound of step B removed (a trailing sentence gets hoisted): `test_answer_and_trailing_sentence_are_not_moved`;
   - hoist removed also turns `test_card_call_and_result_pass_through_unchanged` red;
   - rows outside a block emitted at their own position changed to dropped or moved before the call: `test_user_row_between_call_and_result_moves_after_result`;
   - id-less relaxation reverted to `j == i + 1`: `test_idless_pair_with_sentence_between_is_kept`, `test_idless_pair_with_user_row_between_is_kept`.
4. After the fix every test passes; the mutation checks of item 3 are run and their output kept.
5. Go: the new `getPipecatcallMessages` cases are characterisation of pass-through (the Go side is not
   changed): a notification between call and result, a sentence stored between an `emit_info_card` call and
   its result, and `mcp_` and built-in call rows. They pass before and after, and pin that Go does not drop
   or reorder them.
6. S16 RED: the test runs the validator on the stored-order input through the real adapters (must report
   a violation) and on the filtered input (must report none).

### 3.4 Scenario coverage (T)

Python function names are the ones of the 3.3 table (file `scripts/pipecat/test_message_filters.py`).

| S | Scenario | Test |
|---|---|---|
| S1 | plain multi-turn | `test_plain_history_unchanged` |
| S2 | one call, no sentence, answer, follow-up | existing `test_preserves_paired_tool_call` + `test_followup_turn_after_clean_tool_turn` |
| S3 | one call with sentence (the defect) | `test_sentence_between_call_and_result_moves_before_call`, `test_answer_and_trailing_sentence_are_not_moved`, id-less `test_idless_pair_with_sentence_between_is_kept`; real adapters: S16 |
| S4 | serial calls | `test_serial_calls_unchanged`, `test_serial_calls_sentence_in_first_pair` |
| S5 | parallel calls, partial results | `test_parallel_separate_rows_become_serial_pairs`, `test_multi_id_row_results_follow_in_call_order`, `test_multi_id_row_with_missing_result_is_dropped`, `test_duplicate_result_keeps_first`, `test_duplicate_call_id_one_result_drops_second_call`, `test_duplicate_call_id_two_results_both_kept`, `test_unanswerable_call_releases_its_claims`, `test_idless_parallel_calls_are_dropped` |
| S6 | failed tool result | `test_failed_result_kept_and_adjacent` |
| S7 | unknown tool name (never recorded, same input as S8) | existing `test_drops_unpaired_tool_call`, `test_unpaired_call_with_sentence_and_followup` |
| S8 | call without result, then follow-up | `test_unpaired_call_with_sentence_and_followup`, `test_idless_call_without_result_is_dropped` |
| S9 | orphan result | existing `test_drops_orphaned_tool_result`, `test_result_before_its_call_is_dropped` |
| S10 | window cut | existing `test_partial_pairing_drops_both`, `test_no_wrong_pairing_for_window_cut_alone_or_lost_results_alone` (id-less) + `test_id_bearing_window_cut_drops_only_the_orphan_result` |
| S11 | non-LLM role between call and result | existing Go case `filters out notification role messages` (skip) + new Go case named `notification row between a tool call and its result is skipped` + Python `test_user_row_between_call_and_result_moves_after_result`, `test_idless_pair_with_user_row_between_is_kept` |
| S12 | listen-internal rows | Go `Test_getPipecatcallMessages_TwoFetch` (start_test.go:6143), which pins the `origin != listen_internal` filter of the second fetch (helper `pipecatRestMessageFilters`). The exclusion is an SQL filter, so no such row reaches Python; no new test |
| S13 | `emit_info_card` blocks and args | Go existing `strips blocks from role:tool message content on history rebuild`, `neuters emit_info_card tool_calls Arguments while keeping the entry`, `full card turn: strips blocks and neuters tool_calls together` + new Go case `sentence stored between a card call and its result keeps stored order and both transformations`; Python `test_card_call_and_result_pass_through_unchanged` |
| S14 | MCP and built-in mixed | Python `test_parallel_rows_with_sentence_mixed_tool_names` + new Go case `mcp_ and built-in tool call rows pass through with ids and arguments unchanged` |
| S15 | idempotent, input unchanged | `test_filter_is_idempotent_and_does_not_mutate_input` |
| S16 | provider conversion | new `scripts/pipecat_realtest/test_message_order_real.py` (real pipecat 1.12.0, no network, `pytest --noconftest`, not run by CI like the other files in that directory, run by hand and the result is on the card): `test_stored_order_is_invalid_for_openai_and_gemini` (negative control through the real adapters), `test_filtered_order_is_valid_for_openai_and_gemini`, `test_gemini_keeps_the_sentence_as_a_model_text_turn`. The CI-run guarantee of the same invariant is the Python checker of 3.2 |

A3 (real provider calls) is separate and needs API keys.
