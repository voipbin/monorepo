# VOIP-1510: Insight AI conversation listen stops after a few turns

Status: Draft for review (Round 1)
Author: Hermes (CPO), 2026-09-10
Jira: VOIP-1510
Branch: VOIP-1510-Fix-insight-listen-stop

## 1. Problem statement

Insight AI's real-time conversation listen (bin-ai-manager's `RunListenTurn` /
`listen_conversation.go`, design doc
`docs/plans/2026-09-03-insight-ai-realtime-listen-design.md`) stops producing
any activity partway through a live conversation, even though the customer
keeps messaging and the sibling response-generating AIcall on the SAME
conversation keeps answering normally. Confirmed live on
admin.voipbin.net, Case `a5c91eed-3cd7-43ac-91f6-5275701e9a74`, conversation
`0a593b87-9c00-4093-8e7b-bad3c2b12992`, AIcall (Insight)
`2d4d3412-f9e2-4c2c-a49d-d34053221a8c`, 2026-09-09 ~20:25-20:27 UTC.

Platform-wide Prometheus metric confirms this failure class is not a one-off:
```
ai_manager_aicall_listen_conversation_segment_total{result="dropped_unknown"} = 40  (both replicas)
ai_manager_aicall_listen_conversation_segment_total{result="buffered"}        = 13
```
`dropped_unknown` (conversation message arrived but the Redis resolver found
no listening AIcall) outnumbers successfully-buffered segments 3:1. **Caveat,
stated explicitly so expectations are not overset: `dropped_unknown` is
recorded at MESSAGE INTAKE (`EventCVMessageCreated`'s resolver lookup),
architecturally upstream of and separate from the turn-3 tool-dispatch
failure this PR investigates (§2-§3, which happens once a turn has already
started). This PR's fix does not, by itself, address the majority
(`dropped_unknown`, ~75% of the combined total) of this metric's unhealthy
signal — see §4.

## 2. Confirmed failure point (from live Loki/Prometheus evidence — NOT the root cause yet, see §3)

Reconstructed minute-by-minute from `bin-ai-manager`, `bin-pipecat-manager`
container logs (Loki, via Grafana proxy on bm-nyc-01) for AIcall
`2d4d3412-...` and its three listen-turn pipecatcalls:

| turn | pipecatcall_id | result |
|---|---|---|
| 1 (20:25:53) | 059a5185 | OK — `get_conversation_content` tool executed via `RunnerToolHandle`, ai-manager replied, notify_agent NOT called (correct per prompt) |
| 2 (20:25:59) | faa57911 | OK — same tool executed, no notify_agent (correct) |
| 3 (20:26:22) | d6c97d20 | **FAILED — tool call never reached `RunnerToolHandle`** |

Turn 3's pipecat-manager log shows the LLM (Gemini 2.5 Flash) invoking a
function:
```
20:26:22.754  llm-function-call-started    {}
20:26:22.754  metrics (tokens: completion_tokens=32)
20:26:22.756  llm-function-call-in-progress  tool_call_id=5337edd7-eb44-4e43-9319-c0517b5d9546
20:26:22.756  llm-function-call-stopped      tool_call_id=5337edd7-...  cancelled=false
```
The gap between `in-progress` and `stopped` is **239 microseconds**. On the
two successful turns, the equivalent gap (tool dispatch to `RunnerToolHandle`
+ ai-manager round trip + result callback) is **~240 milliseconds** — three
orders of magnitude longer. `RunnerToolHandle` (the Go HTTP handler the
Python runner's `tool_execute()` POSTs to) has ZERO log entries for
`d6c97d20` anywhere in the window, confirmed by direct Loki query. The sibling
AIcall's tool call in the exact same second (`36dcf504`, tool_call_id starting
with a different id) DID reach `RunnerToolHandle` in 1ms.

**Confirmed failure point: the LLM's function call for turn 3 was never
actually dispatched to the registered Python tool wrapper (`tools.py`'s
`tool_execute`) — pipecat's LLM service framework emitted the RTVI
start/in-progress/stopped bookkeeping frames (which the Go side logs as
"Unrecognized RTVI message type" and otherwise ignores by design) around a
function invocation that resolved to no callable action, and the turn
silently completed with no output.**

**What is NOT yet known** (important — do not read this as a closed
investigation): the RTVI frames captured in the log
(`llm-function-call-started`, `-in-progress`, `-stopped`) carry only
`tool_call_id`, never the function/tool NAME the LLM tried to call. Cross-
checked directly: `bin-ai-manager`'s logs for the same time window contain
ZERO occurrences of tool_call_id `5337edd7-...` anywhere (confirmed via a
targeted Loki grep), consistent with the call never reaching
`RunnerToolHandle` → ai-manager, but this does NOT tell us which of this
AIcall's two available tools (`get_contact_interactions`,
`get_conversation_content`) the LLM was attempting to invoke. §3's hypothesis
(a) below assumes it was the former (by elimination, since turns 1-2 both
successfully called the latter) — this is an unverified inference, not a
confirmed fact, and closing that specific gap is one of the reasons §5
proposes adding function-name logging before attempting a fix.

This still explains the observed user-visible symptom: no further evaluation
was ever scheduled for this AIcall because `notify_agent` (the ONLY tool that
would correspond to observable output for a no-alert turn) was never called
AND no error surfaced anywhere to trigger a retry — `promListenTurnTotal`
recorded `ran` for this turn regardless (bin-ai-manager only meters whether
the turn executed, not whether the LLM's requested tool call succeeded), so
nothing downstream knew turn 3 was degraded. Turns 4+ additionally never got
scheduled, which is either a consequence of the same underlying bug or a
distinct, architecturally separate gap — see §4, which this PR does NOT
resolve.

## 3. Leading hypotheses for the SPECIFIC dispatch failure (needs code-level fix, unconfirmed which applies)

This AI's tool set (2 tools: `get_contact_interactions`,
`get_conversation_content` — from `AllInsightToolNames`, its stored
`tool_names`) is resolved and handed to the Python runner NOT by
`bin-ai-manager`'s `buildListenTurnMessages` (that function only assembles
the LLM message array — system/user prompts, Q&A history, transcript; it
carries no tool information at all), but by `bin-pipecat-manager`'s own
`resolveAIFromAIcall` + `toolHandler.GetByNames(ai.Type, ai.ToolNames)`
(`pkg/pipecatcallhandler/runner.go`), which is then passed to
`h.pythonRunner.Start(...)` and on to the Python runner's `tool_register()`
(`scripts/pipecat/tools.py`). Turns 1-2 both successfully called
`get_conversation_content`. §2 established that turn 3's tool_call_id never
reached ai-manager but NOT which of the 2 tools it was for — so hypothesis
(a) below (by elimination: "probably `get_contact_interactions`, since that's
the other one") is an inference from absence, not a confirmed fact.

The three hypotheses below sit at DIFFERENT points in the dispatch pipeline
and are not claimed to be mutually exclusive in the sense of "exactly one is
true" — they are listed as the candidate STAGES at which the failure could
occur, ordered earliest-to-latest in the call path, and distinguishing them
is exactly what §5's added logging is for (each stage failing produces a
different log signature once instrumented, per the mapping below):

- **(a) `get_contact_interactions` specifically fails to register/resolve**
  in the Python pipecat runner's `tool_register()` — e.g. a schema issue
  Gemini's function-calling rejects only for this tool's parameter shape, or
  a name mismatch between what `convert_to_openai_format` produces and what
  the LLM echoes back in its function-call frame. Signature once
  instrumented: `tool_register`'s per-tool log line would show the name was
  registered, but `tool_execute` would never log an entry for that
  `tool_call_id` — i.e. the failure is between "LLM decided to call X" and
  "our wrapper's `wrapper()` closure was ever invoked", entirely inside
  pipecat's own LLM-service/function-calling internals, outside this
  codebase's control flow.
- **(b) A transient registration race**: `tool_register` runs inside
  `init_single_ai_pipeline`'s `try` block AFTER `task_manager.add`, but BEFORE
  `task.queue_frames([LLMRunFrame()])`. If a function-call frame for one of
  the 2 tools arrives before registration completes (unlikely given the
  logged sequencing, but not ruled out under concurrent goroutines), the
  dispatch would resolve against an incomplete function table. Signature:
  `tool_register`'s "Registered N tools" log line would show a timestamp
  AFTER the LLM's function-call frame, which is directly checkable once both
  are logged with consistent timestamps.
- **(c) An unhandled exception inside `tool_execute()` before its `aiohttp`
  POST is issued** (e.g. `params.arguments` shape assumption failing for this
  specific tool's parameters), silently swallowed by pipecat's function-call
  frame lifecycle rather than surfaced as an error frame. Signature: entry
  log line for `tool_execute` WOULD appear (unlike (a)), but the log line
  immediately before the `aiohttp.ClientSession().post(...)` call would not,
  or a traceback would appear instead.

Confirming which applies requires either a live reproduction with `pipecat`
DEBUG-level Python stdout captured (not currently shipped to Loki — verified:
`stream=stdout` for pipecat-manager containers returns 0 rows for this window,
meaning Python's own logger output is not currently centrally logged), or
instrumenting `tool_execute()`/`tool_register()` with additional log lines
before this ships. **This is the first implementation task.**

## 4. Secondary gap (documented, likely separate ticket — not blocking this fix)

Once a listen turn silently no-ops (§2/§3), no further turn is ever
re-scheduled for that AIcall unless ANOTHER conversation message arrives
inside the same debounce window and re-triggers `EventCVMessageCreated` ->
`ListenTurnTryLock`. In the reproduction, three more customer messages DID
arrive after turn 3 (20:26:34, etc.) but produced ZERO further listen turns
for this AIcall — worth checking separately whether `dropped_unknown`
(§1 Prometheus evidence) is the SAME root cause manifesting a second way (the
Redis resolver membership silently not covering later messages) or a truly
distinct bug. Flagging here for visibility; will re-investigate once §3 is
fixed and re-tested, rather than conflating two unconfirmed hypotheses in one
PR.

## 5. Proposed fix (Phase 1 — observability first, mandatory before a blind fix)

1. Add Python-side structured logging (loguru, matching the existing style) in
   `tools.py`'s `tool_register` (log EVERY registered tool name explicitly,
   not just the count) and in `tool_execute` (log entry BEFORE the aiohttp
   POST, including the tool_call_id and function name so a failure can be
   correlated with the RTVI frame's tool_call_id, and any exception with full
   traceback) so the next real occurrence is directly diagnosable from Loki
   instead of requiring log-timing forensics.
2. Confirm `voipbin-pipecat-manager-*`'s Python subprocess stdout IS actually
   captured by the Alloy/Loki pipeline (§3 noted a `stream=stdout` query
   returned 0 rows in the reproduction window — need to confirm whether that
   is because nothing logged, or because stdout genuinely isn't collected;
   fix the pipeline if the latter, since debugging pipecat-manager's Python
   side is currently blind). **Timebox: this step must be confirmed within
   one business day of implementation start — if the collection gap itself
   turns out to be non-trivial to fix (e.g. requires an infra-loki change),
   escalate/split rather than let this PR stall waiting on it.**
3. Reproduce live again with logging from (1)+(2) in place, capture the exact
   failure signature, and confirm which of §3(a)/(b)/(c) is the real cause. If
   reproduction does not occur naturally within a reasonable window, drive it
   directly via a live admin.voipbin.net webchat test session with multiple
   turns (same technique used to find this bug originally).
4. Implement the targeted fix for the confirmed cause.
5. Add a regression test in `bin-pipecat-manager` exercising the failure mode
   once confirmed (e.g. a fake LLM service that requests a tool call for
   every registered tool name, asserting `RunnerToolHandle`-equivalent code
   path is reached for each).

## 6. Acceptance criteria

- Root cause of the turn-3 dispatch failure is confirmed by reproduction with
  logging in place (not just inferred from timing, as this doc necessarily
  does today).
- Fix ships with a regression test that fails without the fix and passes with
  it.
- Live re-verification: a multi-turn webchat conversation (5+ customer
  messages spanning multiple tool-triggering listen turns) is run against
  admin.voipbin.net, and Loki confirms every turn's tool call actually
  reaches `RunnerToolHandle` (or, for turns that correctly decide to stay
  silent, that the LLM's response contained no unresolved function call).
- `ai_manager_aicall_listen_conversation_segment_total{result="dropped_unknown"}`
  vs `buffered` ratio is re-measured after the fix is live for a representative
  period and reported (even if §4 turns out to be a separate root cause,
  confirm this fix didn't regress that ratio further).
- If §5 step 3's investigation concludes §4 (the resolver/turn-rescheduling
  gap) is a SEPARATE bug from the one this PR fixes, a follow-up VOIP ticket
  is filed BEFORE this PR is merged (not left as a dangling TODO in this
  doc), so the majority contributor to `dropped_unknown` (§1) has an owned
  tracking record.

## 7. Non-goals for this PR

- §4 (resolver/debounce turn-rescheduling gap) is explicitly NOT fixed here
  unless investigation in §5 step 3 reveals it is the SAME bug. If it is
  separate, file a follow-up VOIP ticket rather than scope-creeping this PR.
- The "prompt configuration gap" (Insight AI assistants shipping without a
  case-appropriate system prompt) reported earlier in this same investigation
  thread is a product/config issue, already being addressed operationally
  (prompt was hand-corrected on the live `Test Insight Assistant` AI during
  this investigation) — not a code change and not in scope here.
