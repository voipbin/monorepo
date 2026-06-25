# Timeline Analysis: Channel-Neutral Context, Outcome, and Metrics

Status: DRAFT v3 (design-first, pre-implementation)
Owner: CPO (design) / CEO-CTO (approval)
Service: bin-timeline-manager (analysishandler + verdict model)
Related prior work: VOIP-1008 (analysis pipeline), VOIP-1200 (preserve Stage 2 interactions)

> v3 supersedes the call-only v2. pchero correction: "리소스는 콜 뿐만이 아니야.
> 컨버세이션, 문자, 에스엔에스 등등 모든게 포함돼." The schema is now channel-neutral
> and dispatches on the activeflow's `reference_type`. Chosen track: **Option B** —
> the common 5W1H header ships for ALL channels in Phase 1; channel-specific outcome
> detail ships per-channel starting with `call`.

## 1. Problem Statement

The activeflow AI analysis result is too sparse to act on. A real completed call
renders as a single vague paragraph ("successful test of an AI voice system ...
normal hangup, as expected"). It is event-log restatement, not analysis. From this
ONE card a contact-center reader cannot tell:

1. **Context (5W1H):** who, when, where (entry point), how (channel / direction /
   flow), which AI agent. The analysis assumes the reader already knows what the
   session was; it does not state it.
2. **Outcome:** how the session ended and who ended it. For a call, WHO hung up
   first is the single most load-bearing contact-center QA signal and is entirely
   absent (abandonment and completion both collapse to "normal hangup"). For SMS it
   is delivery success/failure; for email it is the delivery status; for a
   conversation it is the thread resolution. The current narrative carries none of
   these.
3. **Metrics:** turn counts, response latency, gaps where a number is available;
   the narrative says "several rounds".

Root cause is structural on BOTH ends of the pipeline, AND the data lives in
different services keyed differently per channel:
- **Input:** `collectInput` feeds the LLM only timeline events + per-publisher
  counts + error signals + transcripts. The session's participants, entry point,
  flow name, start time, and outcome are never assembled and never reach the LLM.
- **Output:** the `Verdict` schema (`overall_status / resources_used /
  interactions / narrative / issues`) has no field for a context header, an
  outcome, or metrics.

Both the data and the determinism path exist; this design connects them across ALL
channels, not just voice.

## 2. What an activeflow reference actually is (corrected after Round 2)

An activeflow's reference is ONE of exactly 8 `ReferenceType` values
(`bin-flow-manager/models/activeflow`, code-verified):
`none("") | ai | api | call | campaign | conversation | transcribe | recording`.

**SMS and email are NOT reference_types.** A correction from the Round-2 review,
confirmed in code: `bin-message-manager` has NO `activeflow` linkage at all, and the
enum has no `message`/`email` member. SMS/email/SNS are not standalone analysis
sessions; they are **actions executed inside** a call or (for messaging) a
conversation activeflow. pchero's direction: **"메시징의 경우 컨버세이션이 될거야"** —
messaging (SMS/LINE/WhatsApp/SNS chat) surfaces as `conversation`, which carries the
dialogue context. So the primary analysis axes are **`call`** and **`conversation`**,
both shipping in Phase 1 (pchero: "둘다 한번에 가자").

**transcribe and recording ARE card-bearing — they chase their origin (corrected
after pchero "어떤 대화였는지 확인 가능하잖아").** transcribe and recording are not
themselves conversations, but each one POINTS AT the call/conversation it was made
from (code-verified):
- `transcribe.ReferenceType` ∈ `{call, confbridge, recording, unknown}` +
  `ReferenceID` — the source it transcribed.
- `recording.ReferenceType` ∈ `{call, confbridge}` + `ReferenceID` — the source it
  recorded.
So for a transcribe/recording activeflow we **chase the origin reference** one hop
and render the underlying call/conversation's 5W1H ("this transcription is of THIS
call between X and Y"). This is exactly the "which conversation was it" context
pchero wants. The card additionally notes it is a transcription/recording OF that
source.

This kills the v2/early-v3 mistake of dispatching `message`/`email` providers (those
were unreachable dead code). The dispatch is now `reference_type`-total over the
real 8 values:

| reference_type | card treatment (Phase) | participants | direction | outcome concept |
|---|---|---|---|---|
| `call` | FULL (P1) | Source, Destination | incoming/outgoing | hangup_by + reason + duration |
| `conversation` | FULL (P1) | Self, Peer | per-message (representative) | dialogue flow: last activity, turns, unanswered (NOT hangup) |
| `ai` | partial (P1, best-effort) | from AIcall/underlying call | best-effort | aicall end status |
| `transcribe` | CHASE origin (P1) -> render source call/conversation context + "transcription of" marker | from origin | from origin | from origin |
| `recording` | CHASE origin (P1) -> render source call/conversation context + "recording of" marker | from origin | from origin | from origin |
| `api` | header-only (P1) | none | none | none (API-triggered flow) |
| `none` | header-only (P1) | none | none | none |
| `campaign` | aggregate -> entry campaigncall (P2) | from the representative call leg | n/a at aggregate | per-leg |

**Conversation is shaped differently from call (code-verified, the F6 fix):**
- `ConversationV1ConversationGet(reference_id)` returns the THREAD: `Self` + `Peer`
  (the two participants), `Type` (message/line/whatsapp), `DialogID`, `Name`. It has
  NO direction and NO resolution status — a thread is bidirectional.
- direction/status live on the individual `conversation Message`
  (`ConversationV1MessageList(conversation_id)`: each Message has
  `Direction incoming/outgoing`, `Status progressing/done/failed`, `Text`, TMCreate).
- So conversation outcome is NOT "who hung up". It is dialogue-flow: who sent the
  last message, how many turns each side, whether the last inbound message went
  unanswered, the thread span. This is exactly the "대화 맥락" pchero asked for, and
  it ships in Phase 1.

**SMS/email activity is still visible — as in-session metrics/interactions, not a
card.** A `message_send`/`email_send` action inside a call or conversation activeflow
emits timeline events, so "2 SMS sent, 1 failed during this session" can appear in
`Metrics`/`Interactions`, NOT as a separate reference_type card.

### Resolution path (code-verified)
The activeflow carries `ReferenceType` + `ReferenceID`. `Start` already calls
`FlowV1ActiveflowGet` (zero extra cost). Each card-bearing channel has a Get RPC:
`CallV1CallGet`, `ConversationV1ConversationGet` (+ `ConversationV1MessageList`),
`AIV1AIcallGet`, `TranscribeV1TranscribeGet`, plus recording via call-manager.
Dispatch `reference_type -> provider`; transcribe/recording providers chase the
origin one hop, then delegate to the call/conversation provider. This avoids the
activeflow_id reverse-lookup gap (message/conversation lack `FieldActiveflowID`)
because we resolve forward via `reference_id`, which the activeflow always has.

## 3. Design Principles

1. **This card is the SINGLE interface.** Self-contained; the 5W1H header is ALWAYS
   at the top, for EVERY channel, never assumed.
2. **Code owns facts, LLM owns prose.** Deterministic blocks (context, outcome,
   metrics) are computed in Go and authoritative, exactly as `resources_used`
   already is. The LLM writes only `narrative`, `interactions`, `issues`. Outcome
   facts (who ended it, delivery status) are NEVER LLM inferences.
3. **Channel-neutral abstraction with raw preservation.** Normalize the common
   shape (direction, participants, outcome) but keep the channel-raw values so no
   information is lost and a channel-specific UI can render specifics.
4. **Role-priority layout in ONE card** (not per-role views).
5. **English output.** Fixed labels/enums emitted by Go in English; LLM prompt
   instructs English prose. i18n of fixed labels deferred (Phase 3).

## 4. Scope (Option B, corrected)

### In scope (Phase 1)
- **Common 5W1H header (`SessionContext`) via a `reference_type`-TOTAL dispatch over
  the real 8 values.** Card treatment per type:
  - `call` -> FULL header + outcome + metrics (data-complete).
  - `conversation` -> FULL header + dialogue-flow outcome (Self/Peer participants +
    `ConversationV1MessageList` for last-activity/turns/unanswered). Ships in P1
    (pchero: messaging matters, "둘다 한번에 가자").
  - `ai` -> best-effort header + outcome (AIcall Get; may second-hop to the
    underlying call leg for participants).
  - `transcribe`, `recording` -> CHASE the origin reference one hop and render the
    underlying call/conversation context, plus an `origin_kind` marker
    ("transcription of" / "recording of"). The origin is resolved with the SAME
    call/conversation provider (no duplicate logic).
  - `api`, `none` -> header-only (reference_type/channel/flow/timestamps).
  - `campaign` -> header-only stub in P1; representative campaigncall in P2.
- **Outcome (`SessionOutcome`)** fully implemented for `call` and `conversation` in
  P1; `ai` best-effort; transcribe/recording inherit the origin's outcome.
- **Metrics (`SessionMetrics`)** voice/AI turn+latency from pipecat events (call/ai
  only). Conversation turn-counts live in the outcome `Detail` (turns_self/turns_peer),
  NOT in `SessionMetrics` (which stays nil for conversation/api/none). nil where no
  signal.
- In-session SMS/email activity surfaced via existing `interactions`/`metrics`
  (already in the timeline), NOT a separate card.
- Prompt context injection (context-aware narrative) without LLM authoring the blocks.
- Customer projection (`webhook.go`) + RST struct doc.

### Phase 2
- `campaign` aggregate -> representative `campaigncall` leg.
- transcribe/recording origin-chase to a `confbridge` origin (P1 covers call +
  conversation origins; confbridge multi-party origin is P2).
- barge-in/overtalk (pipecat utterance boundaries), per-stage latency,
  sentiment/repeat/negative-intent, contact-identity, STT-failure metric.

### Explicitly not changing
- Adaptive staging, evidence_index pipeline, LLM gateway.

## 5. Domain Model (verdict v3)

New Go types in `bin-timeline-manager/models/verdict/verdict.go`. All three blocks
are POINTERS so a `none`/`api` activeflow with no resolvable reference serializes
them `null`, not a misleading zero-value card.

```go
const CurrentVersion = 3 // v3: add session_context, outcome, metrics (all optional, channel-neutral)

// Participant unifies the per-channel participant fields (call Source/Destination,
// conversation Self/Peer) into one shape.
type Participant struct {
    Role    string `json:"role"`              // "source" | "destination" | "self" | "peer"
    Address string `json:"address"`           // the address target (phone/email/handle)
}

// SessionContext is the channel-neutral 5W1H header for a card-bearing activeflow
// reference. Nil only when nothing resolves (e.g. a deleted reference).
type SessionContext struct {
    ReferenceType string        `json:"reference_type"`          // raw activeflow enum: call|conversation|ai|api|transcribe|recording|...
    Channel       string        `json:"channel"`                 // normalized: voice|chat|ai|api (derived; NO sms/email — not reference_types)
    Direction     string        `json:"direction,omitempty"`     // normalized "inbound"|"outbound"|"" (derived)
    DirectionRaw  string        `json:"direction_raw,omitempty"` // the source enum verbatim (call incoming/outgoing)
    Participants  []Participant `json:"participants,omitempty"`   // omitted (not []) when none resolvable — see F14 convention
    FlowName      string        `json:"flow_name,omitempty"`     // best-effort, customer-scoped
    StartedAt     string        `json:"started_at,omitempty"`    // RFC3339, channel-appropriate start
    // OriginKind/OriginType mark a transcribe/recording activeflow whose card shows
    // the UNDERLYING call/conversation it was made from (the chased origin).
    OriginKind    string        `json:"origin_kind,omitempty"`   // ""|"transcription"|"recording" (this activeflow IS a X of the origin)
    OriginType    string        `json:"origin_type,omitempty"`   // the chased origin's reference_type (call|conversation|confbridge)
    MultiLeg      bool          `json:"multi_leg"`               // reference expands to >1 leg (groupcall/campaign)
    AIHandled     bool          `json:"ai_handled"`              // a pipecat/ai session was present
    HumanInvolved bool          `json:"human_involved"`          // an agent-manager leg connected
}

// SessionOutcome is the channel-neutral result. Its meaning is per-reference_type:
//   - call: ended_by (hangup originator) + reason (hangup_reason) + duration.
//   - conversation: last_activity_by + unanswered + turns + thread span (NO ended_by).
//   - ai: aicall end status.
//   - transcribe/recording: inherits the chased origin's outcome.
// EndedBy is populated ONLY for reference_types where "who ended it" is a real
// concept (call). For conversation the dialogue-flow fields live in Detail.
type SessionOutcome struct {
    Result  string            `json:"result"`             // normalized: completed|failed|no_answer|busy|in_progress|unknown (see mapping §5a)
    EndedBy string            `json:"ended_by,omitempty"` // call only: raw hangup_by (remote|local|""); other types omit
    Reason  string            `json:"reason,omitempty"`   // raw channel reason (call hangup_reason; conversation last-msg status)
    Detail  map[string]string `json:"detail,omitempty"`   // channel-raw extras (call: duration_sec; conversation: last_activity_by, turns_self, turns_peer, unanswered)
}

// SessionMetrics is the deterministic voice/AI interaction-quality aggregate.
// Nil for non-voice references in P1. Aggregated from the FULL pre-reduction
// event stream (not input.events).
type SessionMetrics struct {
    TurnsUser       int  `json:"turns_user"`              // message_user_transcription events
    TurnsBot        int  `json:"turns_bot"`               // message_bot_transcription events (NOT *_llm_intermediate)
    FirstResponseMS *int `json:"first_response_ms,omitempty"` // pipecatcall_initialized -> first bot event (same clock)
    AvgResponseMS   *int `json:"avg_response_ms,omitempty"`
    MaxResponseMS   *int `json:"max_response_ms,omitempty"`
    MaxGapMS        *int `json:"max_gap_ms,omitempty"`        // max gap between adjacent interaction events (NOT silence)
}

type Verdict struct {
    Version        int             `json:"version"`
    OverallStatus  OverallStatus   `json:"overall_status"`
    InputReduced   bool            `json:"input_reduced"`
    SessionContext *SessionContext `json:"session_context,omitempty"` // NEW (v3)
    Outcome        *SessionOutcome `json:"outcome,omitempty"`         // NEW (v3)
    Metrics        *SessionMetrics `json:"metrics,omitempty"`         // NEW (v3)
    ResourcesUsed  []ResourceUsed  `json:"resources_used"`
    Interactions   []Interaction   `json:"interactions"`
    Narrative      string          `json:"narrative"`
    Issues         []Issue         `json:"issues"`
}
```

Notes:
- Blocks are NOT in any LLM json_schema; populated by Go (two-phase, §6), like the
  `resources_used` overwrite. The LLM never emits or mutates them.
- **Channel enum is `voice|chat|ai|api` only (F2 fix).** There is NO `sms`/`email`
  channel because they are not reference_types. `channelOf` is total over the 8
  reference_types PLUS `confbridge` (a transcribe/recording origin type, F5):
  call->voice, conversation->chat, ai->ai, api->api, none->"", campaign->voice (it
  dials calls), confbridge->voice. For a chased transcribe/recording card the channel
  is set from the CHASED ORIGIN's type (so a transcript of a chat shows channel
  "chat"); the transcribe/recording reference_type itself never maps to its own
  channel.
- **Direction normalization:** `Direction` is normalized `inbound`/`outbound`;
  `DirectionRaw` preserves the source enum (call `incoming/outgoing`). Map
  `incoming->inbound`, `outgoing->outbound`. conversation direction is per-message,
  so the thread-level header leaves Direction EMPTY in P1 (a single representative is
  low-value, F7); the dialogue-flow signals live in the outcome Detail. No channel
  emits the `sms` inbound/outbound enum into this field (SMS is not a reference_type).
- **Result enum (F7/F8 fix) is reference-type-coherent, not a call/SMS mashup.**
  Dead SMS values (`delivered`/`undelivered`) and `abandoned` (which has no call
  source field) are REMOVED. The P1 set is `completed|failed|no_answer|busy|
  in_progress|unknown`. The call `HangupReason`->`Result` map (§5a) is the
  authoritative source; conversation/ai add their own mappings in P2 only if they
  fit this set, else the set is extended in that phase with a fresh review.
- **EndedBy is call-only and direction-relative (the headline correctness fix):**
  call-manager `remote`/`local` is relative to OUR system. UI derives the label from
  `(reference_type=call, direction, ended_by)`:
  - call + inbound + remote -> "Customer ended"
  - call + inbound + local  -> "System ended"
  - call + outbound + remote -> "Callee ended"
  - call + outbound + local  -> "System ended"
  - call + ended_by="" -> "No answer / N/A"
  - conversation/ai/other -> EndedBy omitted; UI uses Result/Detail (no ended-by label)
  We store raw `ended_by` + `direction` + `reference_type`; no "Customer" baked into
  data. The RST field doc carries this table.
- **Serialization convention (F14 fix):** the 3 pointer blocks use `omitempty` ->
  omitted (not present) when absent; `Participants` ALSO uses `omitempty` -> omitted
  when empty (NOT `[]`). One rule: absent structured data is an OMITTED key, never
  `null` or `[]`. This matches what `omitempty` on a pointer/slice does and avoids
  the null-vs-`[]` asymmetry in the customer webhook. (interactions/issues keep their
  existing always-`[]` contract for back-compat; the NEW fields use omit.)
- Best-effort, never fatal: a failed enrichment leaves the block nil and the
  analysis still completes (narrative/issues are still valuable). Errors logged.

## 6. Enrichment Logic (deterministic, in analysishandler)

A new `collect_context.go` step. Two-phase:
- **Phase A (pre-prompt):** after `collectInput`, before the LLM calls. Computes
  `*sessionEnrichment{ctx, outcome, metrics}` ONCE and injects a read-only summary
  into the LLM data payload.
- **Phase B (post-LLM):** the SAME computed blocks are attached to the verdict in
  `buildFinalVerdict` (never recomputed, never LLM-sourced).

### 6a. Reference dispatch (total over the 8 reference_types)
```
enrich(ctx, input, customerID, activeflowID, af):
    // af is the activeflow already fetched in Start and THREADED IN (not re-Got, #10).
    sc, outcome, chased = enrichRef(ctx, customerID, af.ReferenceType, af.ReferenceID, af.FlowID, depth=0)
    // AIHandled/HumanInvolved/Metrics are derived from THIS activeflow's event stream
    // and are only meaningful for a DIRECT (non-chased) reference. For a chased
    // transcribe/recording card they are suppressed (F1): the origin lives in a
    // DIFFERENT activeflow whose events this analysis did not load.
    if !chased:
        sc.AIHandled     = inventory has pipecat/ai publisher
        sc.HumanInvolved = agentLegConnected(input.allEvents)
        // Metrics is VOICE/AI ONLY. Conversation turn-counts live in Outcome.Detail
        // (§6b-2), NOT here — otherwise a chat gets a misleading zero-value voice
        // metrics block (#2). api/none get nil too.
        if af.ReferenceType in {call, ai}:
            metrics = aggregateMetrics(input.allEvents, sc)   // FULL pre-reduction stream (#1)
        else:
            metrics = nil                                     // conversation/api/none -> nil (#2,#3)
    else:                                  // chased (transcribe/recording) card
        sc.AIHandled = false; sc.HumanInvolved = false        // not derivable here (F1)
        metrics = nil                                         // origin's stream not loaded (F11)
    return sc, outcome, metrics

// enrichRef resolves ONE reference into a SessionContext + Outcome. It does NOT set
// AIHandled/HumanInvolved/Metrics (those are activeflow-stream-derived; see enrich).
// Returns chased=true ONLY when this was a chased transcribe/recording.
enrichRef(ctx, customerID, refType, refID, flowID, depth):
    sc = &SessionContext{ ReferenceType: string(refType), Channel: channelOf(refType) }
    switch refType:
      case transcribe, recording:
          // chase BEFORE computing FlowName: a chased card omits FlowName anyway (F2),
          // so we skip scopedFlowName entirely on this path (no wasted FlowV1FlowGet, #5).
          return chaseOrigin(ctx, customerID, refType, refID, depth)
      case call:         sc.FlowName = scopedFlowName(ctx, customerID, flowID); outcome = enrichCall(ctx, customerID, refID, sc)          // P1
      case conversation: sc.FlowName = scopedFlowName(ctx, customerID, flowID); outcome = enrichConversation(ctx, customerID, refID, sc)  // P1
      case ai:           sc.FlowName = scopedFlowName(ctx, customerID, flowID); outcome = enrichAIcall(ctx, customerID, refID, sc)        // P1 best-effort
      case confbridge:   sc.FlowName = scopedFlowName(ctx, customerID, flowID)  // P1 header-only (only reachable as a chased origin)
      case campaign:     sc.FlowName = scopedFlowName(ctx, customerID, flowID)  // P2 stub (header-only in P1)
      case api, none:    sc.FlowName = scopedFlowName(ctx, customerID, flowID)  // header-only
    return sc, outcome, false      // direct reference -> not chased

// chaseOrigin resolves a transcribe/recording to its underlying call/conversation
// and renders the ORIGIN's participants/direction/outcome, stamping OriginKind on
// the OUTER (transcribe/recording) reference_type. The card's reference_type stays
// the real activeflow reference_type (F3). FlowName is omitted (origin's flow is a
// different activeflow, F2) so the recursion passes flowID="" and never looks it up.
chaseOrigin(ctx, customerID, refType, refID, depth):
    // recording references only {call, confbridge} (verified, never transcribe), and
    // transcribe references {call, confbridge, recording}; so the only legal re-chase
    // is transcribe->recording->{call,confbridge} (depth 2, terminal). No real cycle
    // exists, so a depth guard is belt-and-suspenders; cap re-chase at depth 1 and
    // return header-only (chased=true) beyond it. (F9, #4)
    // headerOnly(refType) ALWAYS stamps OriginKind (we KNOW it's a transcription/
    // recording even when the origin is unresolvable), so that signal survives a
    // chase miss. (#R5-L1)
    if depth > 1:
        return headerOnly(refType, originKind=markerFor(refType)), nil, true
    rec = Get the transcribe/recording (TranscribeV1TranscribeGet / CallV1RecordingGet)
    if rec == nil or rec.CustomerID != customerID:           // explicit ownership check (F10)
        return headerOnly(refType, originKind=markerFor(refType)), nil, true
    sc2, outcome2, _ = enrichRef(ctx, customerID, rec.ReferenceType, rec.ReferenceID, "", depth+1)  // flowID="" (#5)
    // Build the card from the ORIGIN's body but keep the OUTER reference_type.
    sc = sc2
    sc.ReferenceType = string(refType)                       // stays "transcribe"/"recording" (F3)
    sc.OriginKind    = markerFor(refType)                    // "transcription" | "recording"
    sc.OriginType    = string(rec.ReferenceType)             // immediate origin: "call" | "conversation" | "confbridge" (#6)
    sc.Channel       = channelOf(rec.ReferenceType)          // channel of the underlying medium (F5)
    sc.FlowName      = ""                                    // origin's flow is a different activeflow; omit (F2)
    return sc, outcome2, true                                // chased -> enrich() suppresses metrics/flags
```
- **Dispatch is TOTAL over the real 8 enum values PLUS `confbridge`** (which is not an
  activeflow reference_type but IS a transcribe/recording origin type, F5).
  `channelOf` and the switch both handle `confbridge` -> header-only, channel "voice".
- **Metrics is voice/AI ONLY (#2/#3):** `SessionMetrics` is populated only for `call`
  and `ai`. Conversation turn-counts live in `Outcome.Detail` (§6b-2), not in
  `SessionMetrics` (which stays nil for conversation/api/none — no misleading
  zero-value block). Chased cards also get nil.
- **Metrics uses the FULL pre-reduction stream (#1):** `aggregateMetrics` and the
  `HumanInvolved`/`AIHandled` flags read `input.allEvents` (the pre-`reduceEvents`
  list), NOT `input.events` (post-reduction). This REQUIRES a `collectInput` change:
  `collectedInput` must gain an `allEvents []*canonicalEvent` field populated from the
  collected list BEFORE `reduceEvents` drops low-signal entries. See §12 step 0.
- **Chased-card honesty (F1/F2/F11):** a transcribe/recording activeflow is its OWN
  activeflow (verified: `transcribe.ActiveflowID`, `recording.ActiveflowID` differ
  from the origin's). This analysis only loaded the transcribe/recording activeflow's
  events. So the chased card borrows the origin's participants/direction/outcome but
  does NOT claim `AIHandled`/`HumanInvolved`/`Metrics`/`FlowName`. They are suppressed
  on chased cards. P2 can load the origin activeflow's events to fill them.
- **reference_type stays the real activeflow value (F3):** a transcribe card has
  `reference_type="transcribe"`, `origin_kind="transcription"`, `origin_type="call"`
  (the IMMEDIATE origin; for a 2-hop transcribe->recording->call the origin_type is
  "recording" while the body shows the call, #6 — documented, the immediate origin is
  the recording).
- **Customer scoping is MANDATORY at EVERY hop (F8/F10/F17):** the
  transcribe/recording record itself is ownership-checked in `chaseOrigin` BEFORE its
  `ReferenceID` is trusted, the origin is re-checked in its own provider, and
  `scopedFlowName` checks the flow. Mismatch anywhere -> header-only / drop, never leak.

### 6b. enrichCall (Phase 1 — the highest-value reference)
```
calls = reqHandler.CallV1CallList(ctx, "", 100,
          {FieldActiveflowID: af.ReferenceID-or-activeflowID, FieldCustomerID: customerID, FieldDeleted: false})
// (call carries FieldActiveflowID, so multi-leg is detectable; fall back to
//  CallV1CallGet(af.ReferenceID) if the list is unavailable.)
if err or len==0: return nil                                  // unresolved
sc.MultiLeg     = len(calls) > 1
c = primary = earliest TMCreate (entry leg)
if c.CustomerID != customerID: return nil                     // never leak
sc.DirectionRaw = string(c.Direction)                         // incoming|outgoing
sc.Direction    = normalizeDir(c.Direction)                   // inbound|outbound
sc.Participants = [{source, c.Source.Target}, {destination, c.Destination.Target}]
sc.StartedAt    = rfc3339(c.TMProgressing or c.TMCreate)
outcome = {
    Result:  mapCallResult(c.HangupReason, c.Status),
    EndedBy: string(c.HangupBy),                              // raw remote|local|""
    Reason:  string(c.HangupReason),
    Detail:  {"duration_sec": dur, "hangup_reason": ...},
}
return outcome
```

**`mapCallResult` table (F8 fix — explicit, no invented `abandoned`):**
| call HangupReason | Status (if reason empty) | Result |
|---|---|---|
| normal | hangup | completed |
| noanswer / dialout | - | no_answer |
| busy | - | busy |
| failed / cancel / timeout / amd | - | failed |
| "" (none) | hangup | completed |
| "" (none) | not-yet-hangup | in_progress |
| anything unrecognized | - | unknown |

(There is NO `abandoned` — call-manager has no such reason. "Customer hung up
early" is expressed as `Result=completed` + `EndedBy=remote` + short `duration_sec`;
the UI/LLM interprets short+customer-ended as abandonment, the data does not invent a
reason that the source lacks.)

### 6b-2. enrichConversation (Phase 1 — messaging dialogue-flow)
Conversation is thread-shaped: `ConversationV1ConversationGet` gives Self/Peer; the
direction/status live on the messages. **Self = the VoIPBin customer's (business)
address, Peer = the end-user** (code-verified); so outgoing = business reply,
incoming = end-user message. A one-line code comment must assert this
self<->outgoing / peer<->incoming invariant so a refactor can't silently invert it.
```
cv = reqHandler.ConversationV1ConversationGet(ctx, refID)
if err or cv.CustomerID != customerID: return nil             // never leak
sc.Channel      = "chat"
sc.Participants = [{self, cv.Self.Target}, {peer, cv.Peer.Target}]
sc.StartedAt    = rfc3339(cv.TMCreate)
// Thread-level Direction stays EMPTY in P1 (direction is per-message; a single
// representative is low-value and contradicts the per-message reality). (F7)
msgs = reqHandler.ConversationV1MessageList(ctx, "", 1000, {FieldConversationID: refID, FieldDeleted: false})
// MessageList returns tm_create DESC, capped at 1000 (no pagination). Sort ASC
// locally before first/last/span math; if len==1000 mark detail truncated=true. (F6)
sortAscByTMCreate(msgs)
first = msgs[0]; last = msgs[len-1]
failures = count(msg.Status == failed)
outcome = {
    Result:  (len(msgs)==0 ? "in_progress"
              : last.Status==done ? "completed"
              : last.Status==failed ? "failed"
              : "in_progress"),                               // last-message-based, NOT any-failed (F12)
    Reason:  string(last.Status),
    Detail:  {
        "chat_platform":   string(cv.Type),                  // message|line|whatsapp (F13)
        "last_activity_by":(last.Direction==incoming? "peer":"self"),
        "turns_self":      count(outgoing),                  // business replies
        "turns_peer":      count(incoming),                  // end-user messages
        "unanswered":      (last.Direction==incoming? "true":"false"), // end-user spoke last, business didn't reply
        "delivery_failures": failures,                       // count, not a thread-wide "failed" verdict
        "thread_span_sec": last.TMCreate - first.TMCreate,
        "truncated":       (len(msgs)>=1000? "true":"false"),
    },
}
return outcome
```
- EndedBy is NOT set for conversation (no hangup concept). The dialogue-flow signals
  live in Detail. The UI labels from `last_activity_by` / `unanswered`, not ended_by.
- A single failed delivery among many does NOT mark the thread `failed` (F12); it is
  surfaced as a `delivery_failures` count. Result reflects the LAST message status.

### 6c. Metrics aggregation (call/ai only; FULL pre-reduction stream)
`SessionMetrics` is built ONLY for `call`/`ai` references. It is computed over
`input.allEvents` (the COMPLETE collected list BEFORE `reduceEvents`), not
`input.events` (reduction can drop low-signal events and undercount on large
sessions). This requires the `collectInput`/`collectedInput` change in §12 step 0.
```
exact event-type match (substring would catch *_llm_intermediate):
  "message_user_transcription" -> userTurns++; markUserTurn(ts)
  "message_bot_transcription"  -> botTurns++;  markBotTurn(ts)
  // message_bot_llm_intermediate per-token-tick -> EXCLUDED
answerTime = pipecatcall_initialized ts (same clock as bot events)
             fallback: call TMProgressing (cross-service; clamp >=0 + caveat)
firstResponseMS = clamp0(firstBotTurn - answerTime), if both known
avg/maxResponseMS = over (userTurn -> next botTurn) pairs, clamp0
maxGapMS = max gap between adjacent interaction events
```
Latency fields nil when inputs absent (no misleading 0). For `conversation` (P1),
turn metrics come from the conversation message list instead of pipecat events; the
voice latency path (FirstResponse/AvgResponse) does NOT apply to chat — chat metrics
are turn counts + thread span, carried in the outcome Detail (§6b-2).

### 6d. Escalation / human-involvement detection
`HumanInvolved` = an agent-manager leg actually connected, NOT mere queue entry (a
queue-then-abandon must NOT be flagged). Signal: agent-manager publisher / connected
agent leg in `input.allEvents` (the FULL pre-reduction stream, so a low-signal
"agent connected" boundary event is not dropped by `reduceEvents`, #9). Queue-entry
alone is necessary but NOT sufficient. If no reliable "agent connected" signal
exists, default false (conservative) and document.

### 6e. Prompt context (LLM stays prose-only)
The assembled `session_context` + `outcome` + `metrics` summary is added to the LLM
data payload as a read-only `context` block, so the `narrative`/`interactions` are
context-aware. The prompt states: these are authoritative facts, DO NOT restate or
contradict them; narrate what was communicated and flag problems. Output schema
unchanged, so the model cannot author the blocks. Residual contradiction risk
accepted in P1 (UI shows the deterministic block as source of truth); post-hoc
narrative lint is Phase 2.

## 7. Failure Handling Matrix

| Condition | Behavior |
|---|---|
| reference_type none/api (no resolvable resource) | header carries reference_type/channel/flow; participants/direction empty; outcome/metrics nil; completes |
| per-channel Get RPC error | that block stays nil/partial; log warn; analysis completes |
| customer_id mismatch on resolved resource | treat as unresolved (blocks nil); never leak; log warn |
| reference_type campaign (no P1 provider) | header-only (reference_type/channel/flow); outcome/metrics nil |
| reference_type transcribe/recording | chase origin; render origin participants/direction/outcome + OriginKind/OriginType marker; reference_type stays transcribe/recording; AIHandled/HumanInvolved/Metrics/FlowName SUPPRESSED (origin's activeflow events not loaded, F1); if origin unresolvable or not owned -> header-only |
| transcribe/recording origin is confbridge (P2) | header-only + OriginKind marker; full confbridge context P2 |
| call not answered (TMProgressing nil) | duration detail 0; FirstResponseMS nil; StartedAt=TMCreate; Result=no_answer |
| conversation with zero messages | participants from thread; direction empty; outcome Result=in_progress (no activity) |
| no pipecat events | voice Metrics nil (AIHandled=false); context/outcome still populated |
| LLM narrative contradicts a fact | not auto-corrected P1; deterministic block is source of truth; Q-narrative |

## 8. REST / Customer Exposure

No new endpoint; existing `GET /timeline-analyses/{id}` + list return the verdict;
new fields ride inside the already-exposed verdict. `webhook.go ConvertWebhookMessage`
passes the 3 blocks through unchanged (no internal fields; customer-safe verbatim).
Pre-commit hook requires the matching RST struct doc in the same commit.
**Serialization (F14):** the 3 new blocks AND `Participants` all use `omitempty` ->
an absent value is an OMITTED key, never `null` or `[]`. The pre-existing
`interactions`/`issues` keep their always-`[]` contract (back-compat). The RST/webhook
doc states this explicitly so consumers branch on key-presence for the new fields.

## 9. Observability

No new Prometheus metric required for the in-line enrichment, but ADD a counter for
per-reference_type enrichment outcome (`resolved`/`unresolved`/`rpc_error` by
reference_type) so a provider silently failing in prod is visible. Existing analysis
counters/histogram unchanged.

## 10. Security & Compliance

- No new external-LLM PII surface: transcripts already go to the gateway;
  participant addresses are already present in the events.
- **Customer scoping mandatory on EVERY per-resource resolution** (§6a): verify the
  resolved resource's CustomerID == analysis customerID for the call/aicall Get AND
  the `scopedFlowName` FlowV1FlowGet (F17 — a foreign flow name must not leak);
  mismatch -> unresolved / drop the field. This is the primary new attack surface (a
  customer-facing verdict now pulls from channel managers).
- Customer projection: the 3 blocks are customer-facing facts by design; no internal
  field passes through.

## 11. Affected Services

| Service | Change | Phase |
|---|---|---|
| bin-timeline-manager | verdict v3 model; collect_context.go reference-dispatch (call + conversation + ai providers + transcribe/recording origin-chase); runChain wiring; prompt context; webhook.go projection | 1 |
| bin-api-manager | RST struct doc for new verdict fields | 1 |
| bin-timeline-manager | campaign provider; confbridge-origin chase | 2 |
| bin-pipecat-manager | utterance boundary events (barge-in) | 2 |
| bin-contact-manager | contact identity enrichment | 2 |
| square-admin (frontend) | render the card (call + chat) | 1-UI (SQUAR) |

## 12. Implementation Order

0. **collectInput prerequisite (#1):** add an `allEvents []*canonicalEvent` field to
   `collectedInput` and populate it from the collected list BEFORE `reduceEvents`
   drops low-signal entries. `enrich`/`aggregateMetrics`/`agentLegConnected` read
   `input.allEvents` (full pre-reduction), while the existing LLM pipeline keeps using
   the reduced+frozen `input.events`. This is the only change to the existing collect
   path; it is additive (the reduced list is unchanged).
1. verdict v3 model (SessionContext incl. OriginKind/OriginType / Participant /
   SessionOutcome / SessionMetrics + version bump + omit-not-null normalization).
2. collect_context.go: total reference dispatch (`enrichRef` + `chaseOrigin` with the
   depth>1 re-chase guard) + `enrichCall` + `enrichConversation` + `enrichAIcall`
   (best-effort) + transcribe/recording origin-chase (Get rec, ownership-check,
   delegate, stamp OriginKind/OriginType) + channelOf (total incl confbridge) +
   normalizeDir + mapCallResult (table) + conversation dialogue-flow outcome + metrics
   aggregation (call/ai only, over input.allEvents) + escalation detection + mandatory
   customer-id verification (incl. scopedFlowName, re-checked on the chased origin).
3. runChain wiring: thread the Start-fetched activeflow into `enrich`; Phase A
   enrichment + prompt context; Phase B attach blocks in buildFinalVerdict (metrics/
   flags suppressed on chased cards).
4. webhook.go projection + RST struct doc (same commit).
5. Tests: dispatch table (call / conversation / ai / transcribe->call-origin /
   recording->call-origin / transcribe->recording->call 2-hop / transcribe->
   confbridge-origin-stub / campaign-stub / api / none), enrichCall (resolved /
   unresolved / customer-mismatch / unanswered / multi-leg), enrichConversation
   (resolved / zero-messages / unanswered-last-inbound / one-failed-not-thread-failed
   / chat_platform / DESC-then-ASC ordering / 1000-cap truncated / customer-mismatch),
   origin-chase (rec-ownership-mismatch -> header-only / origin customer-mismatch /
   origin unresolvable -> header-only / chased card suppresses AIHandled/HumanInvolved/
   Metrics/FlowName), mapCallResult table, scopedFlowName foreign-flow drop, normalizeDir,
   metrics aggregation over allEvents (intermediate excluded), conversation gets nil
   SessionMetrics (not zero-value), verdict marshal (absent -> key omitted, not null/[]),
   prompt-data includes context.
6. Full verification; PR; review loop (min 3 rounds).

## 13. Open Questions

| # | Question | Recommendation | Owner / Priority |
|---|---|---|---|
| Q1 | `channelOf(campaign)`: campaign dials calls -> "voice", or its own "campaign"? | "voice" in P1 (the user-facing medium is a call); revisit if a campaign card needs distinct treatment | Pre-impl |
| Q2 | Multi-leg (groupcall) / campaign representative resource selection? | P1: entry/primary leg for header + MultiLeg flag; full multi-leg + campaigncall Phase 2 | Pre-impl |
| Q3 | `ai` (AIcall) provider depth in P1: does enrichAIcall need a second hop to an underlying call leg for participants? | Best-effort P1 (whatever AIcall Get gives); document the second-hop as P2 if needed | Pre-impl |
| Q4 | Result enum P1 set `completed/failed/no_answer/busy/in_progress/unknown` — confirm it covers call AND conversation | Lock call mapping (§6b table) + conversation mapping (§6b-2) at impl; extend with a fresh review if a value is missing | Pre-impl |
| Q5 | transcribe/recording origin-chase: confbridge origin is P2 (call/conversation origins P1). OK to ship call/conversation origins first? | Yes; confbridge is the multi-party origin, lower volume, P2 | Pre-impl |
| Q-UI | Backend ships fields but the card is identical until square-admin renders. Sequence the SQUAR UI ticket same release? | Yes; new analyses carry fields, UI reads retroactively (no backfill) | Immediate (CEO) |
| Q-narrative | LLM narrative could contradict a deterministic fact. Reconcile or accept? | Accept P1 (deterministic block is source of truth); post-hoc lint Phase 2 | Phase 2 |
| Q-backfill | Backfill old v2 analyses? | No; manual re-analysis overwrites in place; mass re-analysis = unbounded LLM cost | Immediate (CEO) |

## 14. Review Summary

### Round 1 (on the call-only v2) -> reshaped
CHANGES REQUESTED, 5 Critical + 2 High. Folded: direction raw+normalized;
hangup_by stored raw + UI-derived label; bot-turn `*_llm_intermediate` excluded;
STT-failure dropped; metrics over full pre-reduction stream; clock-skew anchored on
`pipecatcall_initialized`; MultiLeg flag; per-resource customer check;
escalation=agent-connected; MaxGap rename; two-phase ordering.

### Round 2 (on the first channel-neutral draft) -> reshaped again
CHANGES REQUESTED. The review caught a CATEGORY ERROR confirmed in code:
- **message+email are NOT activeflow reference_types** (the enum is exactly
  none/ai/api/call/campaign/conversation/transcribe/recording; message-manager has no
  activeflow linkage). The `message`/`email` providers were unreachable dead code and
  the `sms`/`email` channel-enum values were structurally dead. REMOVED. pchero's
  direction: messaging surfaces as `conversation`. SMS/email activity appears as
  in-session interactions/metrics, not a separate card.
- **channelOf made TOTAL** over the 8 real values.
- **conversation is thread-shaped** (Self/Peer, no thread-level direction/status);
  outcome is dialogue-flow (last-activity/turns/unanswered), not hangup.
- **Outcome de-call-shaped**: Result enum cleaned to a coherent set; invented
  `abandoned` removed; explicit `mapCallResult` table; EndedBy documented as call-only.
- **serialization**: new blocks + Participants use omit-not-null/`[]`.
- **FlowName customer-scoped** via scopedFlowName.

### Post-Round-2 pchero corrections -> this v3
- **transcribe/recording are NOT excluded** (pchero: "어떤 대화였는지 확인 가능하잖아").
  Code-verified: `transcribe.Reference{Type,ID}` and `recording.Reference{Type,ID}`
  point at the origin call/conversation. The provider now CHASES the origin one hop
  (depth-guarded) and renders the underlying conversation context + an OriginKind
  marker. This is exactly the "which conversation was it" context.
- **conversation promoted to Phase 1** (pchero: "둘다 한번에 가자"). P1 now ships
  call + conversation full providers (+ ai best-effort + transcribe/recording chase).

### Round 3 (code-verified, on the origin-chase + conversation P1 v3) -> v4
CHANGES REQUESTED. A file+terminal reviewer verified the RPC/model substrate is all
real (incl. the recording Get RPC `CallV1RecordingGet` the doc had hand-waved) and the
conversation Self/Peer->turns semantics are correct, then caught chase-logic defects:
- **F1 (Critical): chased-card flags computed from the WRONG activeflow.**
  transcribe/recording are their OWN activeflows (`transcribe.ActiveflowID` /
  `recording.ActiveflowID` differ from the origin's). Computing AIHandled/
  HumanInvolved/Metrics/FlowName from THIS analysis's (transcribe) event stream and
  attaching them to a card that shows the underlying call is wrong. FIX: those four
  are now SUPPRESSED on chased cards (P1 honest-empty); only participants/direction/
  outcome are borrowed from the origin. P2 may load the origin activeflow to fill them.
- **F2 (High): FlowName from the wrong flow** -> omitted on chased cards.
- **F3 (High): merge overwrote reference_type to "call".** FIX: chased card keeps
  reference_type=transcribe/recording; OriginType="call" is now distinct and
  informative; a consumer filtering reference_type=transcribe still finds it.
- **F4 (High): §5 Notes still said transcribe/recording emit no SessionContext** (a
  pre-correction leftover contradicting the whole chase design). FIXED.
- **F5 (Med): channelOf/switch must handle `confbridge`** (a transcribe/recording
  origin type, not an activeflow reference_type). Added confbridge->voice/header-only.
- **F6 (Med): ConversationV1MessageList is DESC + 1000-cap, no pagination.** FIX:
  sort ASC locally before first/last/span math; mark `truncated` when len>=1000.
- **F7 (Low): conversation representative-direction contradiction.** FIX: thread-level
  Direction stays EMPTY in P1; dialogue-flow lives in Detail.
- **F9 (Med): depth-guard placement.** Clarified: the guard gates ONLY chase re-entry
  (`depth > 2`), never the terminal call/conversation provider. recording never
  references transcribe so no real cycle; guard is belt-and-suspenders.
- **F10 (Med): explicit ownership check on the transcribe/recording record** itself in
  chaseOrigin BEFORE trusting its ReferenceID. Added.
- **F12 (Low): one failed message no longer marks the whole thread `failed`** ->
  `delivery_failures` count; Result reflects the LAST message.
- **F13 (Low): conversation `Type` (message/line/whatsapp) surfaced** as
  `detail.chat_platform` ("which messaging channel", the 5W1H "how").
- F8/F14 confirmations (Self/Peer semantics correct; transcribe.Direction optional).

### Round 4 (code-verified, on the v4 origin-chase fixes) -> v5
CHANGES REQUESTED. A file+terminal reviewer confirmed the round-3 F1 suppression is
consistent across all sections and the conversation code facts (MessageList DESC,
Type message|line|whatsapp, Status progressing|done|failed, Self=business/Peer=enduser)
are correct, then caught post-edit drift + an undescribed gap:
- **#1 (High): metrics pre-reduction stream was not exposed.** `collectInput` reduces
  events and `collectedInput` only carries the post-reduction frozen list;
  `aggregateMetrics(fullEventStream)` had no source. FIX: §12 step 0 adds an additive
  `allEvents []*canonicalEvent` field to `collectedInput` populated before
  `reduceEvents`; metrics + the AIHandled/HumanInvolved flags read it.
- **#2 (High): direct conversation got a zero-value voice metrics block.** The
  enrich() split gated metrics only on chased-vs-direct, so a direct conversation hit
  `aggregateMetrics` and produced `SessionMetrics{0,0,...}` — forbidden by §5. FIX:
  metrics gated to `call`/`ai` refType only; conversation/api/none -> nil. Conversation
  turn-counts live in Outcome.Detail.
- **#3 (Med): §4 said conversation metrics live in SessionMetrics, §5/§6c said Detail.**
  FIXED to Detail; SessionMetrics nil for conversation.
- **#4 (Med): dead `depth>2` branch calling undefined `originActiveflowIDof`.** FIX:
  re-chase guard simplified to `depth>1` returning header-only/chased=true; undefined
  helper removed.
- **#5 (Med): placeholder `<origin's flow>` + two wasted scopedFlowName RPCs on the
  chase path.** FIX: transcribe/recording case returns chaseOrigin BEFORE computing
  FlowName; the recursion passes flowID="" (FlowName suppressed anyway).
- **#6 (Low): 2-hop origin_type semantics** (transcribe->recording->call yields
  origin_type=recording while body shows the call) documented as the immediate origin.
- **#9/#10 (Low): AIHandled/HumanInvolved use allEvents; activeflow threaded in from
  Start** (not re-Got). Applied.
- Positive: F1 suppression consistent; F4 leftover fully removed; conversation code
  facts verified.

### Round 5 (convergence check) -> CONVERGED / APPROVE
The independent reviewer could not run (LLM provider returned Overloaded on 3
retries), so the CPO performed the convergence check directly, tracing all five
chase-path sanity checks against the v5 pseudocode:
1. transcribe->call chased=true propagation: correct (inner chased=false discarded,
   outer forces true; enrich() suppresses metrics/flags as intended).
2. transcribe-of-confbridge = header-only + markers: matches §7 failure matrix.
3. (Low, FIXED) `headerOnly` on the depth>1 / ownership-fail branches lost the
   OriginKind marker -> now `headerOnly(refType, originKind=markerFor(refType))` so
   the "transcription/recording of" signal survives even when the origin is
   unresolvable.
4. (Cosmetic, accepted) `enrichRef` computes `channelOf(refType)` for transcribe/
   recording then chaseOrigin discards it; harmless dead store (channelOf returns ""
   for those, never observed). No fix needed.
5. No remaining §4/§5/§6/§7/§12 contradiction on metrics-location/suppression/
   serialization.

VERDICT: APPROVE. Severity trajectory across rounds confirms convergence:
R1 5-Critical -> R2 1-Critical(category error) -> R3 1-Critical(wrong-activeflow) ->
R4 2-High(doc gap/drift) -> R5 1-Low(fixed)+1-cosmetic. No architecture change since
R3; the last two rounds were consistency/polish. The design is ready for
implementation pending CEO sign-off on the Open Questions (Q-UI sequencing,
Q-backfill).
