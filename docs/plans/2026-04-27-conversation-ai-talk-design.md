# Conversation ↔ AI Talk Bridge — Design

**Date:** 2026-04-27
**Branch:** `NOJIRA-conversation-ai-talk-bridge`
**Status:** Design (pre-implementation)
**Scope:** `bin-ai-manager` only (with RST docs update in `bin-api-manager/docsdev/`)

---

## 1. Problem

VoIPbin's `ai_talk` feature is voice-only today. When SMS/LINE messages arrive at `bin-conversation-manager`, a flow can technically include `ai_talk`, but the LLM's reply never reaches the end user — the loop is broken between `bin-ai-manager` and the conversation channel:

- `bin-ai-manager.startReferenceTypeConversation` already exists, creates an AIcall, and starts a pipecat session.
- `bin-pipecat-manager` runs the LLM and emits a `BotLLM` event.
- `bin-ai-manager.EventPMMessageBotLLM` stores the assistant message in `ai_messages`.
- **Nothing delivers that text back to the conversation channel.** No call to `ConversationV1MessageSend` exists.

Additionally:
- The flow advances past `ai_talk` immediately, so a follow-up `conversation_send` action would have no AI response to read.
- Concurrent inbound messages on the same conversation would launch parallel pipecat sessions for the same AIcall with no interrupt or response-ordering protection.
- Pipecat (audio-oriented) is engaged for what is effectively a one-shot text LLM call, but this is acceptable because it preserves a single execution path for both `AssistanceTypeAI` and `AssistanceTypeTeam` and reuses tool calling, member switching, and engine integrations already built into pipecat.

## 2. Goals

1. AI participates as a first-class chatbot in text conversations (SMS, LINE) using the existing per-message activeflow trigger model.
2. The LLM's reply is delivered to the user via the same channel the inbound message arrived on.
3. `AssistanceTypeTeam` works without a separate code path.
4. Concurrency, pod restarts, and pipecat-pod death are handled correctly across multi-pod deployments.
5. Voice paths are byte-for-byte unchanged. No regression risk to existing AI-call behavior.
6. No changes to `bin-pipecat-manager`, `bin-common-handler`, OpenAPI, DB schema, or proto definitions. All wiring fits inside `bin-ai-manager`.

## 3. Non-goals

- A direct text-only LLM path in `bin-ai-manager` that bypasses pipecat. (Considered; rejected to keep one execution path.)
- Mid-stream LLM cancellation. (Pipecat doesn't surface it.)
- Background reapers for orphan `pipecatcall` rows or abandoned AIcalls. (Deferred to v2.)
- Heartbeat/lease records on `pipecatcall`. (`PipecatV1Ping` already on `main` solves the immediate problem.)
- Customer-cross-check RPC at delivery time. (Asserted via tests; cost not justified.)

## 4. End-to-end data flow

```
User (SMS/LINE)
   │ ① inbound
   ▼
bin-message-manager  ─②─►  bin-conversation-manager (subscribehandler)
                            │ ③ MessageEventReceived: persist conversation + message
                            │ ④ NumberV1NumberGet → MessageFlowID
                            │ ⑤ FlowV1ActiveflowCreate(ReferenceTypeConversation)
                            │     + variables: voipbin.conversation_message.text, ...
                            │ ⑥ FlowV1ActiveflowExecute
                            ▼
                        bin-flow-manager
                            │ ⑦ ai_talk action handler
                            │ ⑧ AIV1AIcallStart(ReferenceTypeConversation,
                            │                    ref_id = conversation_id)
                            ▼
                  bin-ai-manager (startReferenceTypeConversation)
                            │
                            │ ⑨ GetByReferenceID(conversation_id)
                            │     reusable = exists
                            │              ∧ Status ∉ {Terminated, Terminating}
                            │              ∧ not idle-expired (24h default)
                            │     ├─ not reusable → startAIcallByMessaging (new AIcall)
                            │     │                  + UpdateStatus(Progressing)
                            │     └─ reusable     → interruptPreviousPipecatcall(old_pcc)
                            │                       + UpdatePipecatcallID(new_pcc)
                            │                       + UpdateActiveflowID(new_activeflow)
                            │
                            │ ⑩ persist user message (DirectionOutgoing, RoleUser)
                            │
                            │ ⑪ FilterToolsForConversation(ai.tool_names)
                            │     → drops connect_call, stop_media, stop_flow
                            │
                            │ ⑫ startPipecatcall(c)         (shared queue, any live pod)
                            │ ⑬ TerminateWithDelay(...)      (existing safety net)
                            │ ⑭ return immediately — flow advances to next action
                            ▼
                  bin-pipecat-manager (default branch — no audio I/O)
                            │ ⑮ Python pipecat session: load AIcall.messages → LLM
                            │ ⑯ BotLLM event published (RabbitMQ durable)
                            ▼
                  bin-ai-manager (any pod, EventPMMessageBotLLM)
                            │ ⑰ AIV1AIcallGet(evt.PipecatcallReferenceID)
                            │ ⑱ if ReferenceType ≠ Conversation:
                            │      persist + return  (existing voice/task behavior)
                            │ ⑲ guard #1: ac.PipecatcallID == evt.PipecatcallID?
                            │      no  → drop, increment stale-dropped metric, return
                            │      yes → persist assistant message
                            │ ⑳ guard #2: re-fetch AIcall, re-check PipecatcallID
                            │      (narrows the dual-delivery race window)
                            │ ㉑ ConversationV1MessageSend(ac.ReferenceID, evt.Text, nil)
                            │      success → metric++, log debug
                            │      failure → metric++, log error, **silent to user**
                            ▼
              bin-conversation-manager
                            │ ㉒ outgoing message persisted in the conversation
                            │ ㉓ smshandler / linehandler delivers to the channel
                            ▼
                          User (SMS/LINE reply)
```

### 4.1 Step ⑨ — interrupt path (synchronous, ping-gated)

When the existing AIcall is reusable, before allocating a new pipecatcall_id:

```
interruptPreviousPipecatcall(old_pcc_id):
  with timeout 1.5s:
    pc = PipecatV1PipecatcallGet(old_pcc_id)        // CB-protected
  if pc not found → skip (already gone)
  if pingPipecatHost(pc.HostID) is false:
    skip terminate  (dead pod or CB open — fast-fail)
  else:
    PipecatV1PipecatcallTerminate(pc.HostID, pc.ID) // best effort
```

The interrupt is best-effort. The decisive correctness mechanism is the response guard at ⑲/⑳: any LLM response whose `PipecatcallID` doesn't match the AIcall's current `PipecatcallID` is dropped. Because `UpdatePipecatcallID` is last-writer-wins, the guard automatically delivers only the response from the most recent user message and silently discards superseded sessions.

### 4.2 Why pipecat is reused for text

- Same code path for `AssistanceTypeAI` and `AssistanceTypeTeam` (member switching, team logic already in pipecat).
- Tool-calling already wired through pipecat's `tools.py`.
- Engine integrations (OpenAI, Grok, Gemini, Anthropic, etc.) already wrapped.
- Cost: per-turn pipecat session-startup overhead. Acceptable for v1.

### 4.3 Why ai-manager delivers (not flow)

- `ai_talk` is non-blocking by existing convention; flow advances immediately. There is no synchronous AI-response variable for a follow-up `conversation_send` action to read.
- ai-manager already owns AIcall lifecycle and message persistence — it is the natural owner of "got an LLM response, deliver it."
- A flow-driven alternative would require either a new blocking action variant or a new "wait for AI response" primitive. Both are larger changes with no clear product benefit.

## 5. Component-level changes

### 5.1 Files modified / added in `bin-ai-manager`

| File | Change |
|---|---|
| `pkg/aicallhandler/start.go` | Replace reuse branch (lines ~187–194) with status+idle reusability check; call `interruptPreviousPipecatcall`; `UpdatePipecatcallID` + `UpdateActiveflowID`; on create-new path, `UpdateStatus(Progressing)`; add deferred-whitelist NOTE comment before pipecat payload assembly (Path A — see plan Slice 0 decision) |
| `pkg/aicallhandler/send.go` | Same interrupt + `UpdateActiveflowID` pattern in `SendReferenceTypeOthers` |
| `pkg/aicallhandler/helpers.go` (new) | `interruptPreviousPipecatcall(ctx, pcID)` (1.5s `Get` timeout, then ping, then terminate-if-alive); `isAIcallIdleExpired(ac)`; `isAIcallReusable(ac)` |
| `pkg/aicallhandler/db.go` | New `UpdateActiveflowID(ctx, id, activeflowID)` |
| `pkg/messagehandler/event.go` | `EventPMMessageBotLLM` reordered: voice path unchanged; conversation path runs guard #1 → persist → guard #2 → `ConversationV1MessageSend` (silent failure with metric) |
| `pkg/toolhandler/whitelist.go` (new) | `ConversationSafeTools = {SendEmail, SendMessage, SetVariables, GetVariables, StopService, GetAIcallMessages, SearchKnowledge}`; `FilterToolsForConversation([]ToolName)` strips voice-only entries and expands `ToolNameAll` to the whitelist |
| `pkg/metricshandler/` | New counters (see §8) |
| `cmd/ai-manager/init.go` + `pkg/config/` | `aicall_conversation_idle_timeout_hours` (default 24) |

### 5.2 Files modified outside `bin-ai-manager`

| Service | Change |
|---|---|
| `bin-api-manager/docsdev/source/` | RST: conversation+AI flow example; tool-availability matrix update; rebuild HTML and force-add `bin-api-manager/docsdev/build/` per project rule |
| `monitoring/grafana/dashboards/ai-manager.json` | Add panels for the four new counters (additive only) |

### 5.3 Files NOT modified

| Service | Why unchanged |
|---|---|
| `bin-pipecat-manager` | `pmmessage.Message.PipecatcallID` already populated; `/v1/ping` already routed; per-pod queue routing already correct |
| `bin-common-handler` | `PipecatV1Ping`, circuit breaker, per-pod RPC plumbing all already present |
| `bin-conversation-manager` | Existing `ConversationV1MessageSend` is sufficient |
| `bin-flow-manager` | Existing `ai_talk` action calls `serviceStartReferenceTypeConversation` correctly |
| OpenAPI / DB / proto | None of these surfaces change |

## 6. Concurrency model

- **AIcall reuse key** = `(reference_type=conversation, reference_id=conversation_id)`. One AIcall per conversation, reused across turns.
- **`AIcall.PipecatcallID` is the per-turn handle.** Last-writer-wins under concurrent updates is the design intent — newer messages override older sessions.
- **Interrupt is best-effort. Response guard is decisive.** The guard at delivery time is the source of correctness. It works identically whether the interrupt succeeded, raced, or hit a dead pod.
- **Per-pod liveness preflight** (`PipecatV1Ping` with 1.1s outer / 1s inner) wrapped in CB short-circuits dead-pod RPCs and avoids 30s timeout cascades.
- **Multi-pod safe.** The guard is DB-derived, so any `bin-ai-manager` pod processing a `BotLLM` event reaches the same decision.
- **Per-target circuit breaker** (in `requesthandler`) protects the shared pipecat queue, the per-pod queues, and the conversation-manager queue. Repeated failures fast-fail after 5 consecutive errors / 30 s open.

## 7. Lifecycle

| Event | AIcall transition |
|---|---|
| First inbound message for a conversation | Create with `Status=Progressing`, `PipecatcallID=X1`, `ActiveflowID=A1` |
| Subsequent message, AIcall fresh | Reuse: `PipecatcallID=Xn`, `ActiveflowID=An`; previous session interrupted (best-effort) |
| LLM calls `stop_service` | `Status=Terminated` (existing path) |
| Inbound message after termination | Treated as not reusable → fresh AIcall (`Status=Progressing`, ...) |
| 24 h elapsed since `TMUpdate` | Marked `Terminated` and a fresh AIcall is started for the next message |
| Manual `POST /v1/aicalls/<id>/terminate` | `Status=Terminated`; next message starts fresh |
| `bin-pipecat-manager` pod crashes | AIcall unchanged. Next message: ping detects dead pod → skip terminate; new pipecat session lands on a live pod via shared queue; `AIcall.messages` carries full prior context. |
| `bin-ai-manager` pod crashes | AIcall lives in DB; any pod can pick up subsequent events |

The terminated AIcall row stays in DB (`tm_delete = default`); it is functionally retired by the status check. Cleanup is a v2 enhancement.

## 8. Observability

### 8.1 New Prometheus counters (under `ai_manager` namespace)

- `ai_manager_conversation_reply_send_total{result="success|failure"}` — outcome of `ConversationV1MessageSend` calls from AI delivery.
- `ai_manager_aicall_stale_response_dropped_total{guard="primary|secondary"}` — drops attributable to PipecatcallID mismatch (guard #1 vs guard #2).
- `ai_manager_aicall_idle_expired_total` — count of AIcalls terminated by idle-expiry on the reuse path.
- `ai_manager_aicall_interrupt_attempted_total{result="alive|dead|gone|error"}` — interrupt outcomes.

### 8.2 Free metrics already exported by the shared circuit breaker

- `ai_manager_circuitbreaker_state{target}`
- `ai_manager_circuitbreaker_state_transitions_total{target,from,to}`
- `ai_manager_circuitbreaker_rejected_total{target}`

### 8.3 Logging additions

- `INFO` when an AIcall is idle-expired and recreated.
- `INFO` when guard rejects a stale response (with both `PipecatcallID`s logged).
- `DEBUG` on interrupt outcomes.
- `ERROR` on `ConversationV1MessageSend` failure (silent to user, but loud in logs).

### 8.4 Grafana

`monitoring/grafana/dashboards/ai-manager.json` gains a "Conversation AI" row with the four new counters and a CB-state panel filtered to `target = "bin-manager.conversation-manager.request"`.

## 9. Tool whitelist

For `ReferenceType == Conversation`, ai-manager filters the AI's `tool_names` through `FilterToolsForConversation` before assembling the pipecat session payload.

| Tool | Conversation? | Rationale |
|---|---|---|
| `send_email` | ✅ | Channel-agnostic |
| `send_message` | ✅ | Channel-agnostic |
| `set_variables` | ✅ | Flow context |
| `get_variables` | ✅ | Flow context |
| `stop_service` | ✅ | "AI stops replying" semantics translate cleanly |
| `get_aicall_messages` | ✅ | Context-neutral |
| `search_knowledge` | ✅ | Context-neutral |
| `connect_call` | ❌ | Requires a live phone call |
| `stop_media` | ❌ | No media playback in chat |
| `stop_flow` | ❌ | Per-message flow already short-lived; flow-control via stop_service |

`ToolNameAll` expands to the whitelist (not the full registry) when `ReferenceType == Conversation`.

## 10. Error handling

- **LLM/pipecat failure or `startPipecatcall` failure** → `ai_talk` action returns error to flow. Flow handles per its own rules. User receives no reply (silent failure by product decision).
- **`ConversationV1MessageSend` failure** → log `ERROR`, increment `result="failure"` metric, do not retry. AIcall stays as-is; next user message reuses it.
- **Dead pipecat pod** → ping detects it; terminate is skipped; new pipecat session lands on a live pod via shared queue. AIcall self-heals on next user message.
- **All pipecat pods unreachable** → CB on shared queue opens; subsequent attempts fast-fail; surfaces as flow-action error rather than 30s hangs.

## 11. Accepted v1 limits

| Limit | Mitigation path (post-v1) |
|---|---|
| First-turn race may produce duplicate AIcalls for the same conversation | Add unique constraint on `(reference_type, reference_id)` for non-terminated rows |
| Dual-delivery window between guard #2 and `ConversationV1MessageSend` is non-zero | Tighter coupling or a delivered-message pointer |
| LLM tokens spent on interrupted sessions are not refundable | Pipecat-level mid-stream cancellation |
| Abandoned conversations sit `Progressing` until idle-expiry triggers on the next message | Background reaper for AIcalls beyond idle threshold |
| Calico POD_IP recycle: ping may return success against a different (live) pod that doesn't own the session | Replace `HostID = POD_IP` with `POD_UID` |
| Customer cross-check between AIcall and conversation skipped at the RPC level | Already enforced upstream; one-time integration test asserts isolation |
| Orphan `pipecatcall` rows from crashed pods accumulate | Pod-startup reaper for self-owned `host_id` rows |
| Silent failure if `ConversationV1MessageSend` fails | Operator visibility via metric + log |

## 12. Testing strategy

### 12.1 Unit tests required (gomock against `MockRequestHandler`)

- `startReferenceTypeConversation` — fresh AIcall path
- `startReferenceTypeConversation` — reuse + alive previous pipecat (interrupt invoked)
- `startReferenceTypeConversation` — reuse + dead previous pipecat (interrupt skipped after ping)
- `startReferenceTypeConversation` — terminated AIcall → fresh
- `startReferenceTypeConversation` — idle-expired AIcall → fresh
- `startReferenceTypeConversation` — `AssistanceTypeTeam` smoke
- `EventPMMessageBotLLM` — voice path unchanged (regression test)
- `EventPMMessageBotLLM` — conversation guard #1 miss → drop, no persistence
- `EventPMMessageBotLLM` — conversation guard #1 pass, guard #2 miss → drop after persistence
- `EventPMMessageBotLLM` — guards pass → `ConversationV1MessageSend` invoked
- `EventPMMessageBotLLM` — `ConversationV1MessageSend` fails → silent, metric incremented
- `FilterToolsForConversation` — whitelist enforcement; `ToolNameAll` expansion
- `interruptPreviousPipecatcall` — alive / dead / gone / Get-error branches
- `isAIcallIdleExpired`, `isAIcallReusable` — boundary cases

### 12.2 Integration coverage

In `~/gitvoipbin/monorepo-monitoring/api-validator/`:

- End-to-end: SMS in → AI reply out (mocked LLM)
- Customer-isolation invariant under conversation+AI flow

### 12.3 Coverage target

≥80% per project standard. Test files co-located: `pkg/<pkg>/<feature>_test.go`.

## 13. Verification items (must check during implementation)

1. **`FilterToolsForConversation` wiring site (DEFERRED — Path A).** Slice 0 chose Path A: ship the utility unwired and document the deferred enforcement. See `docs/plans/2026-04-27-conversation-ai-talk-plan.md` Slice 0. The utility lives at `pkg/toolhandler/whitelist.go`; the deferred-wiring comment in `pkg/aicallhandler/start.go` documents the future hook site. Verification of the actual wiring is a v2 follow-up if Path B is later adopted.
2. **Voice paths are byte-for-byte unchanged.** Confirm by running existing test suite before adding new tests; all pass.
3. **`evt.PipecatcallID` populated** — confirmed already in `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:454,670,691`. Belt-and-suspenders: assert non-nil in unit tests of `EventPMMessageBotLLM`.
4. **`UpdateActiveflowID` is needed in dbhandler** — verify whether an `Update` helper accepts a single field generically; if not, add the explicit method.

## 14. Rollout

- Behavior fully gated by `ReferenceType == Conversation`. Voice paths byte-for-byte unchanged.
- Activation is opt-in by flow design — a customer must include `ai_talk` in a conversation flow.
- Deployment: `bin-ai-manager` only. No coordinated deploy with other services. No DB migration. No proto regeneration.
- Default idle-timeout 24 h via env var (`AICALL_CONVERSATION_IDLE_TIMEOUT_HOURS`); tunable per environment.
- RST docs land with the implementation PR; HTML rebuilt and force-added per project rule.
- No feature flag required (opt-in via flow authoring).

## 15. Deferred / out-of-scope

- Direct text-only LLM path in `bin-ai-manager` bypassing pipecat.
- Pipecat mid-stream LLM cancellation.
- Background reapers (orphan pipecatcalls, abandoned AIcalls).
- `POD_UID`-based identity to defeat Calico IP recycle.
- Customer-cross-check RPC at delivery time.
- Heartbeat/lease records on `pipecatcall` (superseded by `PipecatV1Ping`).

---

## Appendix A — Code change footprint summary

```
bin-ai-manager/
  pkg/aicallhandler/
    start.go            (modified: ~30 lines)
    send.go             (modified: ~10 lines)
    db.go               (modified: ~15 lines, new UpdateActiveflowID)
    helpers.go          (new: ~60 lines)
  pkg/messagehandler/
    event.go            (modified: ~50 lines)
  pkg/toolhandler/
    whitelist.go        (new: ~40 lines)
  pkg/metricshandler/
    *.go                (modified: ~20 lines, new counters)
  cmd/ai-manager/
    init.go             (modified: 1 line, viper default)
  pkg/config/
    config.go           (modified: ~5 lines, getter)

bin-api-manager/
  docsdev/source/       (RST updates: conversation+AI flow, tool matrix)
  docsdev/build/        (rebuilt HTML, force-added)

monitoring/grafana/dashboards/
  ai-manager.json       (additive panels)
```

Net: ~200 lines of Go in `bin-ai-manager`, plus RST docs and a dashboard panel.

## Appendix B — Open questions to resolve at implementation time

None for design. All open items are tracked under §13 (Verification) and surface during code review.
