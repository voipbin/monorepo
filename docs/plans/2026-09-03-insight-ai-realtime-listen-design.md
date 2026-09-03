# InsightAI Realtime Listen (Proactive Notification) — Design

Status: Draft
Branch: `NOJIRA-Insight-AI-realtime-listen`
Owner: CPO-directed backend feature

## 1. Problem statement

Today's Case Insight Assistant (`AI.Type == insight`) is purely reactive: the
agent asks a question in the Case Insight Assistant panel
(`square-admin/src/views/contacts/CaseInsightAssistantPanel.js`,
`square-talk/src/features/cases/CaseInsightAssistantPanel.jsx`), the LLM
calls read-only tools (`get_contact_interactions`, `get_call_transcript`,
etc. — `bin-ai-manager/pkg/aicallhandler/tool_insight.go`) to answer. The AI
never initiates. It has no visibility into a call that is happening right
now.

대표님's request: while an agent is on a call tied to an open Case, InsightAI
should follow the live conversation as it happens and, if it judges the
situation warrants it, speak up first — surfacing a warning or suggestion to
the agent before being asked.

## 2. Goals

1. When an agent opens/resumes a Case whose linked call is still in
   progress, InsightAI automatically starts listening to that call's live
   transcript.
2. The existing Insight AIcall (the one the agent already chats with) is fed
   the live transcript as it arrives and, per its own judgment (governed by
   its `init_prompt`), may proactively push a message to the agent — no
   separate "watcher" AI or session.
3. Proactive messages reach the agent through the exact delivery path
   already used for Q&A answers (no new push channel), and are visually
   distinguishable in the UI from a direct answer to a question.
4. Listening stops automatically when the underlying call ends. No new
   billed STT session is left running.
5. What triggers a proactive message is entirely customer-configurable —
   ships as an extension of the AI's existing `init_prompt`, not a
   hardcoded rule set.

## 3. Non-goals (explicit scope cuts)

| Item | Why cut | Re-engagement signal |
|---|---|---|
| Detecting a call that starts *after* the Case is already open, when that call is not the Case's own `ReferenceID` call | `Case.ReferenceID` (VOIP-1253) is fixed at Case-creation time to the call that produced the Case. A later, different call touching the same contact is a distinct scenario (peer/contact matching) with real ambiguity (which of possibly several concurrent calls?) that the CPO has not asked for | A concrete request for "listen to whatever call this contact is on right now, even if it's not the one that opened the Case" |
| Rule-based / keyword-only condition detection | CPO explicitly directed LLM judgment over customer-defined prompt instead (2026-09-03 design discussion) | N/A — deliberately rejected for MVP |
| Separate watcher AI / second AIcall session | CPO explicitly directed single-AI consolidation (2026-09-03 design discussion) | A demonstrated need to decouple watch-cadence from the agent-facing chat session (e.g. cost isolation) |
| square-talk WebSocket push (replacing its 2s poll) | Existing poll cadence is adequate to surface a proactive message with acceptable latency; SQUARE-52 already tracks square-talk WebSocket parity as a separate concern | SQUARE-52 lands, or proactive-message latency is reported as a problem |
| Multi-party (3+) speaker attribution | `transcript.Direction` is binary (`in`/`out`); conferences already only distinguish two "sides" (`call_media.rst`: "direction indicates speaker relative to conference"). Out of reach without a data-model change upstream in `bin-transcribe-manager` | A concrete multi-party Insight request |
| New `Message.Role` value for transcript content | Rejected in favor of reusing `role=user` with a structural speaker tag — avoids touching role-mapping logic in every engine handler for a distinction with no consumer yet (see §4.3) | A concrete need to filter/compact transcript-origin messages independently of Q&A messages |

## 4. Design

### 4.1 Trigger: Case-linked call auto-detection

When an Insight AIcall (`reference_type=contact_case`) transitions to
`progressing` — whether freshly created or resumed on panel open — the
AIcall lifecycle handler (`pkg/aicallhandler`, alongside the existing
`initiating → progressing` transition logic) performs one synchronous check:

1. `ContactV1CaseGet(customer_id, case_id)` — same call already used by
   `toolHandleGetContactInteractions`.
2. If `kase.ReferenceType != kmkase.ReferenceTypeCall` or
   `kase.ReferenceID == uuid.Nil` → not listenable, stop here (e.g. the case
   opened from a conversation message, not a call).
3. `CallV1CallGet(kase.ReferenceID)`. If the call's `Status` is one of the
   still-active statuses (`dialing`/`ringing`/`progressing` — same set
   `transcribehandler.isValidReference` already treats as valid) and
   `TMDelete == nil` → proceed to §4.2. Otherwise the call has already
   ended; do nothing (the agent can still use `get_call_transcript` for the
   finished call, unchanged).

No new event subscription is needed for this step — it is a one-shot check
at AIcall-progressing time, not a standing watch for "some future call on
this contact."

### 4.2 Starting or reusing the transcribe session

1. `TranscribeV1TranscribeList(reference_type=call, reference_id=<call_id>,
   status=progressing)` (same RPC `get_call_transcript` already uses to
   enumerate sessions). If a `progressing` session already exists (started
   by a flow action, another feature, or manually), reuse its `transcribe_id`
   and set `AIcall.ListenOwnsTranscribe = false`.
2. Otherwise call `TranscribeV1TranscribeStart(reference_type=call,
   reference_id=call_id, direction=both, language=<AI's configured
   language, fallback to call's or a default>)` and set
   `AIcall.ListenOwnsTranscribe = true`.
3. Persist `AIcall.ListenTranscribeID = <transcribe_id>` (new nullable
   column, see §4.6) and `AIcall.ListenCallID = kase.ReferenceID` (needed at
   hangup time — see §4.5).

### 4.3 Feeding the live transcript into the LLM

`bin-ai-manager`'s subscribehandler gains one additional static binding
pattern on the global topic exchange `bin-manager.event`:

```
transcribe-manager.transcript.*.created
```

This follows the existing convention exactly — every current binding
(`call-manager.call.*.hangup`, etc.) is already a cross-tenant wildcard,
filtered in-process. On receipt of a `transcript.EventTypeTranscriptCreated`
event:

1. Look up whether any `progressing`, `reference_type=contact_case` AIcall
   has `ListenTranscribeID == payload.TranscribeID` (indexed DB lookup;
   also read the cache used elsewhere in this package — see
   `bin-ai-manager/CLAUDE.md` cache-invariants note, which does not cover
   this new column but the same read-after-write discipline applies).
2. If none, drop the event (not a call anyone is listening to).
3. If found, re-verify `AIcall.CustomerID == payload's owning transcribe's
   CustomerID` before doing anything else — defense in depth, same pattern
   `tool_insight.go` already applies at every tenant boundary.
4. Format the segment as a `Message{Role: message.RoleUser, Direction:
   Inbound}` with content:
   - `direction=in` → `"[CUSTOMER] <text>"`
   - `direction=out` → `"[AGENT] <text>"`
   (See §4.4 for why `in`/`out` map this way, and the verification this
   still needs before merge.)
5. Persist the message and drive it through the **existing**
   `messagehandler` dispatch path exactly as an agent-typed question would
   be — same engine selection, same context assembly, same tool-execution
   loop. This is the one behavioral change needed in `messagehandler`: today
   every dispatch is assumed to produce a reply worth persisting; a
   transcript-fed dispatch must be allowed to produce **no** assistant
   message at all when the LLM chooses not to call `notify_agent` (see
   §4.4). No `Message{Role: assistant}` row is written unless the tool
   fires.

Structural tags (`[CUSTOMER]`/`[AGENT]`), not localized ones, so prompt
behavior doesn't fork by call language — see §4.4 discussion.

Real agent-typed questions to the panel are unprefixed, exactly as today.
This is what lets the LLM (and, later, any log/debug reading of the raw
message history) tell "the agent is asking me something" apart from "this
is what was just said on the call" without a schema change.

### 4.4 Proactive notification: `notify_agent` tool + prompt extension

New LLM tool, restricted to Insight AIs (same gating pattern as
`get_call_transcript`/`get_contact_profile` — see
`bin-ai-manager/docs/domain.md` §LLM Tools):

```
notify_agent(message: string)
```

Calling it is the *only* way a transcript-driven dispatch produces a
persisted, delivered message. The Insight AI's `init_prompt` is extended
(by the customer, in the existing prompt-editing UI — no new field) with
guidance on when to call it, e.g. "if the customer mentions cancellation,
a compliance keyword, or requests something requiring approval, call
notify_agent with a short actionable note; otherwise say nothing." Because
this reuses `init_prompt`, no schema or UI change is needed to let each
customer define their own trigger conditions — directly per 대표님's
direction.

**`in`/`out` → customer/agent mapping — needs empirical confirmation before
merge.** Traced through the code (`bin-transcribe-manager/pkg/
transcribehandler/start.go`, `docsdev/source/transcribe_tutorial.rst`):
`direction` is relative to the transcribed *channel*, not the call as a
whole — `in` is audio arriving into that channel from the far end, `out` is
audio sent out through it. `Case.ReferenceID` is the customer-facing
(inbound) call leg. Once that leg is bridged to an agent, audio "sent out"
through it is the bridged agent's voice, so `in=customer`/`out=agent` is the
structurally correct reading. The only documented example
(`quickstart_transcribe.rst`) is a flow/TTS scenario, not an agent-bridged
one, so this has **not been empirically verified against a real
agent-bridged call's transcript**. Reversed, this silently mislabels who
said what and could produce a materially wrong proactive message (e.g.
attributing a customer's complaint to the agent). **Action item for
implementation, before this ships**: capture one real (or staged)
agent-bridged call's transcript segments and confirm `in`/`out` against
known speaker identity, or find an existing telemetry sample with square-talk
call handling. Blocks §7's sign-off, not the rest of this design.

### 4.5 Lifecycle / cleanup

- On `call-manager.call.*.hangup` for `AIcall.ListenCallID` (existing
  subscription, already dispatched to `aicallhandler` for other reasons —
  add one more check here): if `AIcall.ListenOwnsTranscribe`, call
  `TranscribeV1TranscribeStop(ListenTranscribeID)`. Clear
  `ListenTranscribeID`/`ListenCallID`/`ListenOwnsTranscribe` regardless of
  ownership, so a stale id is never matched by §4.3 step 1 again.
- If the Insight AIcall itself terminates first (agent closes the Case
  panel, session times out) while the call is still active: same cleanup,
  triggered from the AIcall termination path instead of call hangup.
- A transcribe session InsightAI does **not** own (reused, §4.2 step 1) is
  never stopped by this feature — whoever started it owns its lifecycle.

### 4.6 Data model changes

`bin-ai-manager` `AIcall` (table `ai_aicalls`), three new nullable columns:

| Field | Type | Notes |
|---|---|---|
| `listen_transcribe_id` | UUID, nullable | The `transcribe.Transcribe` session this AIcall is currently following. `NULL` when not listening. |
| `listen_call_id` | UUID, nullable | The call being listened to (`Case.ReferenceID` at the time listening started). Needed at hangup time since the hangup event carries a call id, not a transcribe id. |
| `listen_owns_transcribe` | bool, default false | Whether this AIcall started the transcribe session (and is therefore responsible for stopping it) vs. reused one already running. |

Indexed on `listen_transcribe_id` (lookup path in §4.3 step 1).

Alembic migration in `bin-dbscheme-manager` per the root CLAUDE.md rule
(AI drafts the migration file; a human applies it).

`message.Message` — **no new field.** Origin (`agent_request` vs.
`proactive`) is derived structurally: a `Message{Role: assistant}` row
produced by a `notify_agent` tool call is proactive; every other assistant
row is a direct answer. The tool-call record already persisted for every
tool invocation (existing `ToolCall` support on `Message`, per
`bin-ai-manager/docs/domain.md` §Message) carries this distinction for free
— `listenhandler`/frontend can check "did this message's generation include
a `notify_agent` tool call" rather than a new column. (If review surfaces a
case this doesn't cleanly cover — e.g. a UI that needs the origin without
walking tool-call history — add a derived `is_proactive` bool at that
point; not added now to avoid a column with no confirmed reader.)

### 4.7 Frontend

- **square-admin** (`CaseInsightAssistantPanel.js`): already subscribes to
  `customer_id:{id}:aicall:{aicallId}` and receives every new message over
  the existing WebSocket path — no transport change. Add a badge/icon on
  messages whose origin is proactive (derived per §4.6, or exposed as a
  boolean the panel component computes from the message's tool-call data if
  `bin-api-manager`'s service-agent message read surface already includes
  it — needs a routing/API-manager-layer check during implementation,
  flagged for the execution plan rather than resolved here).
- **square-talk** (`CaseInsightAssistantPanel.jsx`): unchanged transport
  (2s poll, per its existing "no WebSocket push" note); same badge
  treatment on render.

### 4.8 Cost / concurrency bound

Insight AI is capped at one active instance per customer
(`ai_ais.active_insight_key`). Concurrent Listen sessions are therefore
naturally bounded by "how many Cases with an in-progress originating call
are open in an agent panel right now" — no separate rate limit is
introduced by this design. LLM call frequency is bounded by transcript
segment frequency (`transcript.EventTypeTranscriptCreated` fires per final
STT result, i.e. per speech turn, not per audio frame) — no additional
debounce is added in this design; if telemetry post-launch shows call
volume is a problem, a segment-count or time-based batching layer can be
added in `messagehandler` without changing the tool/prompt contract.

## 5. Error handling / edge cases

- Case lookup fails, or call lookup fails (`4xx`/`5xx` from the RPC):
  treated as "not listenable right now" — logged, does not fail AIcall
  progression. The agent still gets the Q&A panel; only the proactive
  listening step is skipped.
- Race: call ends between the §4.1 status check and the §4.2 transcribe
  start call. `TranscribeV1TranscribeStart` will itself fail
  (`isValidReference` rejects a non-active call) — treated as a no-op,
  logged, same as above.
- Two agents open the same Case concurrently (unusual but not impossible):
  each gets its own Insight AIcall/session per existing AIcall semantics: only one AIcall
  per (customer, active Insight AI, reference) is expected today per the
  panel's auto-resume behavior; if this invariant does not hold this design
  does not change it, and both would independently attempt §4.2 — the
  second one reuses the transcribe session the first already started
  (`listen_owns_transcribe=false` for the second), avoiding a duplicate
  billed session.
- `notify_agent` called with empty/whitespace message: rejected at the tool
  layer (same validation style as other tools' argument checks in
  `tool_insight.go`), logged as a failed tool call, no message persisted.

## 6. Testing strategy

- `bin-ai-manager`: unit tests for the §4.1 trigger check (case
  lookup → call lookup → active/inactive branching), §4.2 start-vs-reuse
  branching, §4.3 subscribe-handler dispatch (including the
  cross-tenant-mismatch defensive path), §4.4 `notify_agent` tool
  (including the "LLM chooses not to call it → no message persisted" path),
  §4.5 cleanup on hangup and on AIcall termination. Table-driven, gomock,
  following `pkg/aicallhandler/start_test.go` conventions.
- `bin-ai-manager` ↔ `bin-transcribe-manager` boundary: integration-style
  test double for `TranscribeV1TranscribeList`/`Start`/`Stop` request/
  response shapes (matching existing `requesthandler` mock patterns).
- Regression test specifically for the `in`/`out` → customer/agent mapping
  once §4.4's empirical verification lands — this is exactly the kind of
  silent-wrong-attribution bug that deserves a golden-transcript-style
  pinned test, not just a happy-path check.
- Frontend: existing `CaseInsightAssistantPanel` test suites (both apps)
  extended with a case for rendering a proactive-origin message with its
  badge.

## 7. Open items before implementation sign-off

1. **`in`/`out` speaker mapping empirical verification** (§4.4) — blocking.
2. Confirm whether `bin-api-manager`'s service-agent message read surface
   already exposes tool-call data needed for the frontend badge, or needs
   its own small addition (§4.7) — affects execution plan scope, not this
   design's direction.
3. No Jira ticket filed yet for this feature — recommend filing one
   (VOIP-*) before implementation starts, per project convention.
