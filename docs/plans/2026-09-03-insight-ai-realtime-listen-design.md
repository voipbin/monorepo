# InsightAI Realtime Listen (Proactive Notification) — Design

Status: Draft (rev 2, addressing 2026-09-03 architect review round 1)
Branch: `NOJIRA-Insight-AI-realtime-listen`
Owner: CPO-directed backend feature

## 0. Revision history

| Rev | Date | Change |
|---|---|---|
| 1 | 2026-09-03 | Initial draft. Fed each transcript segment through `aicallHandler.Send` as a `role=user` message; proactive origin derived from tool-call history; `TranscribeV1TranscribeStop(transcribeID)`; hangup cleanup via the existing `EventCMCallHangup` lookup. |
| 2 | 2026-09-03 | **Rewritten after architect review (REQUEST_CHANGES, 7 BLOCKING + 4 HIGH).** Transcript segments no longer become `Message` rows and no longer go through `Send`. New two-layer architecture: a Redis-backed transcript buffer plus a debounced *listen evaluation turn* that runs on its own pipecatcall. `notify_agent` gets `RunLLM:false`. Proactive origin becomes a first-class `Message.Origin` field, stored as `role=assistant`. Hangup cleanup uses a new indexed `listen_call_id` column. Transcribe session runs under `IDAIManager`. STT stop on hangup delegated entirely to transcribe-manager's own handler. §10 maps every review item to its resolution. |

Every code reference below was re-verified against the worktree at rev 2 authoring time; file:line citations are load-bearing and were read, not assumed.

---

## 1. Problem statement

Today's Case Insight Assistant (`AI.Type == insight`) is purely reactive: the
agent asks a question in the Case Insight Assistant panel
(`square-admin/src/views/contacts/CaseInsightAssistantPanel.js`,
`square-talk/src/features/cases/CaseInsightAssistantPanel.jsx`), the LLM
calls read-only tools (`get_contact_interactions`, `get_call_transcript`, …
— `bin-ai-manager/pkg/aicallhandler/tool_insight.go`) to answer. The AI
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
2. The **same** Insight AIcall the agent already chats with is the one that
   watches the call and speaks up — no second AI config, no second AIcall,
   no separate "watcher" session record.
3. Proactive messages land in the existing panel thread over the existing
   delivery path (message row → webhook event → WebSocket / poll), and are
   structurally distinguishable in the UI from an answer to a question.
4. Listening stops automatically when the underlying call ends. No new
   billed STT session is left running.
5. What triggers a proactive message is entirely customer-configurable —
   ships as an extension of the AI's existing `init_prompt`, not a
   hardcoded rule set.
6. **(new in rev 2)** Watching a call must not: pollute the agent-visible
   message thread, emit a customer webhook per spoken sentence, evict the
   system prompt or the agent's Q&A history from the LLM context, kill an
   in-flight answer to the agent, or scale LLM cost with speech volume.

## 3. Non-goals (explicit scope cuts)

| Item | Why cut | Re-engagement signal |
|---|---|---|
| Detecting a call that starts *after* the Case is already open, when that call is not the Case's own `ReferenceID` call | `Case.ReferenceID` (VOIP-1253) is fixed at Case-creation time to the call that produced the Case. A later, different call touching the same contact is a distinct scenario (peer/contact matching) with real ambiguity (which of possibly several concurrent calls?) that the CPO has not asked for | A concrete request for "listen to whatever call this contact is on right now, even if it's not the one that opened the Case" |
| Rule-based / keyword-only condition detection | CPO explicitly directed LLM judgment over customer-defined prompt instead (2026-09-03 design discussion) | N/A — deliberately rejected for MVP |
| Separate watcher AI / second AIcall session | CPO explicitly directed single-AI consolidation (2026-09-03 design discussion) | A demonstrated need to decouple watch-cadence from the agent-facing chat session (e.g. cost isolation) |
| square-talk WebSocket push (replacing its 2s poll) | Existing poll cadence is adequate to surface a proactive message with acceptable latency; SQUARE-52 already tracks square-talk WebSocket parity as a separate concern | SQUARE-52 lands, or proactive-message latency is reported as a problem |
| Multi-party (3+) speaker attribution | `transcript.Direction` is binary (`in`/`out`); conferences already only distinguish two "sides" (`call_media.rst`: "direction indicates speaker relative to conference"). Out of reach without a data-model change upstream in `bin-transcribe-manager` | A concrete multi-party Insight request |
| Transcript content stored as `Message` rows (any role) | **Changed in rev 2.** Rev 1 proposed `role=user` with a speaker tag. That makes every spoken line a customer webhook delivery (`messagehandler/db.go:81` → `notifyhandler/publish.go:24-26`), a panel-visible bubble, and a consumer of the 100-row LLM replay window (`aicallhandler/start.go:620-661`) that would evict the system prompt itself. Transcript is now ephemeral Redis state (§5.3), never a row | A concrete need to persist/audit what the AI heard — which should then be a purpose-built table, not the Q&A message thread |
| Per-turn LLM tool restriction (listen turns limited to `notify_agent` only) | `PipecatV1PipecatcallStart` takes no tool list — pipecat-manager resolves tools from the AI record via `toolhandler.GetByNames` (`bin-pipecat-manager/pkg/toolhandler/main.go:91-108`). Restricting per session means a new field on the pipecatcall start contract, in both Go and the Python runner | Telemetry shows listen turns burning cost on unnecessary read-only tool calls |
| Billing Insight listening to the customer | The listen STT session runs under `cmcustomer.IDAIManager` (§5.2), so it is not attributed to the customer's transcription usage. Whether Insight listening becomes a billed line item is a pricing decision, not an architecture one | A pricing decision to monetise Insight listening |
| Dynamic per-transcribe RabbitMQ bindings | Available (`transcript.EventSubscriptionID()` returns `TranscribeID`, `models/transcript/transcript.go:33-35`) and would eliminate wasted deliveries, but introduces a bind/unbind lifecycle whose failure mode is a permanently leaked binding, and breaks the "all bindings are static wildcards" convention the golden test pins. The static-wildcard cost is bounded to one Redis `GET` per platform speech turn (§5.3) | Measured broker or ai-manager CPU cost from `transcript_created` fan-in; the escape hatch and its leak-sweeper requirement are pre-documented in §5.3 |

---

## 4. Architecture overview

The central rev-2 change: **listening and answering are two different
workloads on one AIcall, and they must not share a code path.**

```
                     transcribe-manager
                     transcript_created (per final STT result, platform-wide)
                              │
                              ▼
      ┌───────────────────────────────────────────────────────┐
      │ LAYER 1 — intake (no LLM, no DB write, no webhook)     │
      │  subscribehandler.processEventTMTranscriptCreated      │
      │   • drop if TMDelete != nil  (H3)                      │
      │   • Redis GET ai:listen:transcribe:<transcribe_id>     │
      │     → miss = not ours, drop (this is the whole filter) │
      │   • RPUSH pending + window lists, LTRIM, EXPIRE        │
      └───────────────────────────┬───────────────────────────┘
                                  │ try Redis SET NX EX (debounce lock)
                                  │ not acquired → return, stays buffered
                                  ▼
      ┌───────────────────────────────────────────────────────┐
      │ LAYER 2 — evaluation turn (≤ 1 per interval per AIcall)│
      │  aicallHandler.RunListenTurn                           │
      │   • LPOP-all pending; if empty → skip                  │
      │   • assemble a BOUNDED context explicitly:             │
      │       system prompt snapshot (from AIcall.Metadata)    │
      │     + listen-turn system prompt                        │
      │     + last 10 user/assistant Q&A rows                  │
      │     + rolling transcript window (last 40 lines)        │
      │   • fresh, throwaway pipecatcall id (NOT c.Pipecatcall)│
      │   • PipecatcallStart + TerminateWithDelay              │
      └───────────────────────────┬───────────────────────────┘
                                  │
              ┌───────────────────┴────────────────────┐
              │                                        │
   LLM calls notify_agent                   LLM emits text / nothing
   (RunLLM:false → no follow-up text)                  │
              │                                        ▼
              ▼                          every pipecat message event whose
   ToolHandle → tool rows                PipecatcallID != AIcall.PipecatcallID
   → then one assistant row              is DROPPED (§5.4.4). Nothing is
     with Origin=proactive               persisted, no webhook, no panel row.
   → webhook + panel (intended)
```

Four properties fall out of this shape, and they are the answers to the
review's blocking items:

- **The agent's Q&A path is untouched.** `Send` /
  `SendReferenceTypeOthers` (`pkg/aicallhandler/send.go:16-149`) is never
  invoked by this feature, so its 3-second cooldown
  (`send.go:27-32`, `internal/config/main.go:37`) never rejects a
  transcript, and its `interruptPreviousPipecatcall` +
  `UpdatePipecatcallID` sequence (`send.go:116-122`) never rotates the
  AIcall's pipecatcall out from under an in-flight answer.
- **LLM invocations are decoupled from speech volume.** One turn per
  interval per AIcall, regardless of how many sentences were spoken.
- **Context size is a constant**, assembled from known-bounded inputs,
  not from `getPipecatcallMessages`' newest-100 replay
  (`start.go:620-661`) — which, because the system prompt is itself
  message row #1 (`start.go:812-819`), would have evicted the AI's own
  instructions after ~100 spoken lines.
- **Nothing a listen turn does is visible to the customer** unless the
  LLM deliberately calls `notify_agent`.

---

## 5. Design

### 5.1 Trigger: where the listen-ensure hook goes

Rev 1 placed the hook at "the `initiating → progressing` transition."
That is wrong: `startReferenceTypeContactCase` has **three** success
returns and only two of them transition status.

Verified in `pkg/aicallhandler/start.go`:

| Return | Line | Goes through `startContactCaseTurn` (which owns the `UpdateStatus(...Progressing)` at `start.go:542`)? |
|---|---|---|
| Fresh create | `start.go:439` | yes |
| Existing row stuck at `Initiating` | `start.go:509` | yes |
| **Reuse an already-active AIcall** | `start.go:512-513` | **no — returns `existing` with no status write at all** |

The reuse path is the *common* path: it is what every panel re-open hits.
A hook on the status transition would therefore essentially never fire in
production.

**Decision: hook the caller, not the callee.** `Start`
(`start.go:168-199`) is the sole caller of
`startReferenceTypeContactCase` (`start.go:190-191`). The
`case aicall.ReferenceTypeContactCase:` branch becomes:

```go
case aicall.ReferenceTypeContactCase:
    res, err := h.startReferenceTypeContactCase(ctx, c, assistanceType, assistanceID, activeflowID, referenceID, teamParameter, currentMemberID)
    if err != nil {
        return nil, err
    }
    h.ensureListenAsync(c, res) // best-effort, non-blocking; §5.1.1
    return res, nil
```

One hook, all three success returns covered, no duplication, no coupling
to the status machine.

#### 5.1.1 `ensureListenAsync` / `ensureListen`

`ensureListenAsync` spawns a detached goroutine with its own
`context.Background()` + timeout (the pattern already used at
`tool.go:191-199` and `start.go:97-100`) so panel-open latency is
unchanged and no listen failure can ever fail an AIcall start. It is
fire-and-forget by design; §6 covers the failure modes.

`ensureListen(ctx, a *ai.AI, c *aicall.AIcall)` runs:

1. **Feature gate.** `config.Get().AIcallListenEnabled` false → return.
2. **Type gate.** `a.Type != ai.TypeInsight` → return. (`contact_case`
   AIcalls are Insight in practice, but this is deny-by-default.)
3. **Idempotency.** If `c.Metadata[listen_transcribe_id]` is set and
   `TranscribeV1TranscribeGet(thatID)` reports
   `Status == progressing && TMDelete == nil && ReferenceID == <the call
   we are about to resolve>` → already listening, return. This is what
   makes repeated panel opens free.
4. **Case lookup.** `ContactV1CaseGet(c.CustomerID, c.ReferenceID)` — the
   same customer-scoped RPC `toolHandleGetContactInteractions` already
   uses (`tool_insight.go:125`), followed by the same
   `kase.CustomerID != c.CustomerID || kase.CustomerID == uuid.Nil`
   recheck (`tool_insight.go:135-138`). **This is the tenant boundary for
   the whole feature** — see §5.3 for why it cannot be re-checked at
   event time.
5. **Reference typing.** `Case.ReferenceType` and `Case.ReferenceID` are
   plain `string` (`bin-contact-manager/models/kase/kase.go:65-66`), and
   there is no `ReferenceTypeCall` constant anywhere in the repo — call
   sites use the bare literal `"call"`. This design adds one:

   ```go
   // bin-contact-manager/models/kase/kase.go
   // ReferenceTypeCall is the stored ReferenceType value for a Case
   // created from a call. The field is a plain string (it mirrors
   // contact_interactions.reference_type's existing vocabulary), so this
   // is an untyped string constant rather than a typed enum member.
   const ReferenceTypeCall = "call"
   ```

   The checks then typecheck:
   ```go
   if kase.ReferenceType != kmkase.ReferenceTypeCall { return }
   callID, errParse := uuid.FromString(kase.ReferenceID)
   if errParse != nil || callID == uuid.Nil { return }
   ```
   (Rev 1's `kase.ReferenceID == uuid.Nil` did not compile against a
   `string` field. The design doc it cited also lives at
   `bin-contact-manager/docs/plans/2026-07-24-case-reference-id-design.md`,
   not under `monorepo/docs/plans/`.)
6. **Call liveness + ownership.** `CallV1CallGet(callID)`; require
   `call.CustomerID == c.CustomerID` (defence in depth, same shape as
   `tool_insight.go:738`), `call.TMDelete == nil`, and `call.Status ∈
   {dialing, ringing, progressing}` — the exact set
   `transcribehandler.isValidReference` treats as transcribable
   (`bin-transcribe-manager/pkg/transcribehandler/start.go:107-115`).
   Anything else → the call is over; return (the agent can still use
   `get_call_transcript` on the finished call, unchanged).
7. Proceed to §5.2.

No new event subscription is needed for the trigger — it is a one-shot
check at AIcall-start time, not a standing watch for "some future call on
this contact."

### 5.2 The transcribe session: owner, start, and reuse

#### 5.2.1 Owner — `cmcustomer.IDAIManager` (resolves review B6)

Rev 1 left this unresolved. It is load-bearing, so it is decided here.

**Decision: the listen transcribe runs under `cmcustomer.IDAIManager`**,
exactly mirroring `summaryhandler.startReferenceTypeCall`
(`bin-ai-manager/pkg/summaryhandler/start.go:84-99`), whose own comment
states the reason: *"here, we set the customer id as the ai manager id …
because if we use the customer id, the created transcribe will be shown
to the customer's transcribe list."*

| | `IDAIManager` (chosen) | tenant `customer_id` (rejected) |
|---|---|---|
| Customer's transcribe list | clean | shows a session they never started |
| Billing | platform-borne, an explicit Insight cost | silent surprise transcription charge |
| Collision with the customer's own live transcribe | none — `startLive`'s dedup guard is scoped by `customer_id` and its comment names this exact AI-manager case as the reason (`transcribehandler/start.go:181-188`) | a same-language customer session makes our `Start` return `TRANSCRIBE_ALREADY_PROGRESSING` (409) |
| Precedent in-repo | yes (summary) | none |

**Consequence, stated explicitly:** the event-time tenant check rev 1
proposed (`AIcall.CustomerID == transcript's CustomerID`) is impossible —
it would *always* fail, because the transcript's `CustomerID` is
`IDAIManager`. The tenant boundary is therefore enforced **once, at
listen-start time** (§5.1 steps 4 and 6: customer-scoped `CaseGet` +
`CustomerID` recheck on both the Case and the call), and the event path
instead verifies *provenance*: "is this `transcribe_id` one we ourselves
started and recorded?" (§5.3). That is a stronger property than a field
comparison — the id is one ai-manager generated and persisted, not
attacker-influenceable.

**Second consequence:** `get_call_transcript`'s own listing is filtered by
`tmtranscribe.FieldCustomerID: c.CustomerID` (`tool_insight.go:757-758`),
so it will not see the listen session. That is correct and intended: the
agent reads *finished* transcripts of *the customer's own* sessions
through that tool; the live listen session is an internal, platform-owned
stream. Nothing regresses.

#### 5.2.2 Reuse rule — owner-aware and language-tolerant

`startLive`'s duplicate guard is scoped `(customer_id, reference_id,
language, status=progressing, deleted=false)`
(`transcribehandler/start.go:196-214`). So under `IDAIManager` the only
sessions we can ever collide with are ai-manager's own — a concurrent
`ai_summary` on the same call, or a listen session started for a *second*
Case on the same call.

```
existing := TranscribeV1TranscribeList(ctx, "", 10, {
    customer_id:   cmcustomer.IDAIManager,
    reference_id:  callID,
    status:        progressing,
    deleted:       false,
})
```

- **Any** progressing `IDAIManager` session on this call is reused,
  *regardless of its language*, and `Metadata[listen_owns_transcribe] =
  false`.
  Rationale: starting a second session only because the language string
  differs would double the STT cost on one call to gain nothing — the LLM
  reads whatever language comes out. Maximising reuse is the cheaper and
  simpler rule. This is the explicit answer to the review's "the reuse
  rule must account for language/owner."
- Otherwise start one, with `Metadata[listen_owns_transcribe] = true`:

  ```go
  tr, err := h.reqHandler.TranscribeV1TranscribeStart(
      ctx,
      cmcustomer.IDAIManager,      // customerID  (§5.2.1)
      call.ActiveflowID,           // activeflowID — the call's, not the AIcall's:
                                   //   a panel-started contact_case AIcall has
                                   //   ActiveflowID == uuid.Nil. Mirrors
                                   //   summaryhandler/start.go:79-82.
      uuid.Nil,                    // onEndFlowID — no on-end flow for listening
      tmtranscribe.ReferenceTypeCall,
      callID,
      language,                    // §5.2.3
      tmtranscribe.DirectionBoth,  // both legs; §5.9
      tmtranscribe.ProviderEmpty,  // provider: default order gcp → aws
      5000,                        // timeout ms, same as summaryhandler
  )
  ```
  Signature verified against
  `bin-common-handler/pkg/requesthandler/transcribe_transcribes.go:64-105`.
  (Rev 1 omitted `provider` and `onEndFlowID` entirely.)

- A session ai-manager does **not** own — i.e. one started by the customer
  under their own `customer_id` — is never reused and never touched. We
  cannot see it (different owner in our filter) and must not affect its
  lifecycle.

#### 5.2.3 Language selection

`language` for a session we start: `c.STTLanguage` if non-empty, else
`config.Get().AIcallListenDefaultLanguage` (default `"en-US"`).
`transcribe-manager` normalises to BCP47 itself
(`transcribehandler/start.go:64-66`), so no client-side validation.

#### 5.2.4 Persisting listen state

On success:

```go
h.UpdateListenState(ctx, c.ID, callID, tr.ID, owns)
```

which performs **one** `AIcallUpdate` writing:
- column `listen_call_id = callID` (§5.8),
- `metadata[listen_transcribe_id] = tr.ID`,
- `metadata[listen_owns_transcribe] = owns`,

and then writes the Redis resolver key
`ai:listen:transcribe:<tr.ID> = <c.ID>` (TTL 12h).

**This is the only `ai_aicalls` write the feature makes during a listening
session** (one at start, one at stop). It is *not* per turn. That bounds
the known `tm_update` ↔ `Send`-cooldown coupling
(`dbhandler/aicall.go:240` bumps `tm_update`; `send.go:27-32` reads it) to
a single ~3s window immediately after listening starts — which happens
inside a detached goroutine during panel open, before the agent could
plausibly have typed a question. Accepted as a bounded cost, with a
recorded follow-up (§11) to decouple the cooldown from `tm_update` onto a
dedicated `tm_last_send`, which is a pre-existing fragility this design
deliberately does not widen.

### 5.3 Layer 1 — event intake

#### 5.3.1 Binding

`bin-ai-manager/pkg/subscribehandler/main.go` gains one static pattern
appended to `topicPatterns` (`main.go:52-64`):

```
transcribe-manager.transcript.*.created
```

Adding it requires editing three coupled places, all confirmed:
1. `topicPatterns` — **append at the end**; the golden test is
   position-sensitive.
2. `processEvent`'s switch (`main.go:179-220`) — a new case using the
   already-declared-but-currently-unused `publisherTranscribeManager`
   constant (`main.go:34`).
3. `binding_golden_test.go` — the `expected` slice (append
   `"transcribe-manager.transcript.*.created"`), the hardcoded
   `len(topicPatterns) != 11` → `12` (two occurrences: the check and the
   message), and the doc comment above `topicPatterns`.

#### 5.3.2 Volume, and why the wildcard is affordable (resolves review H4)

`transcript_created` fires per final STT result for **every** transcription
session platform-wide — flow-driven, summary-driven, customer-started —
not just calls we listen to. `processEventRun` spawns an unbounded
goroutine per event (`main.go:161-165`, prefetch 10). Rev 1 did not
account for this.

The per-event work is therefore made unconditionally cheap:

```go
func (h *subscribeHandler) processEventTMTranscriptCreated(ctx context.Context, m *sock.Event) error {
    var evt tmtranscript.Transcript
    if err := json.Unmarshal([]byte(m.Data), &evt); err != nil { ... }

    // H3: transcripthandler.dbDelete publishes EventTypeTranscriptCreated on
    // DELETE too (bin-transcribe-manager/pkg/transcripthandler/db.go:33 — a
    // known bug, documented in models/transcribe/routingkey_golden_test.go:
    // 182-184). Without this guard a deleted line replays into the LLM as
    // freshly-spoken content.
    if evt.TMDelete != nil || strings.TrimSpace(evt.Message) == "" {
        return nil
    }

    return h.aicallHandler.EventTMTranscriptCreated(ctx, &evt)
}
```

and `EventTMTranscriptCreated` opens with a single Redis `GET`:

```
aicallID, ok := cache.ListenAIcallIDGet(ctx, evt.TranscribeID)  // GET ai:listen:transcribe:<id>
if !ok { return nil }   // not a session we started — 99.9% of platform events end here
```

**Sized cost of keeping the wildcard:** per final STT result anywhere on
the platform — one AMQP delivery, one goroutine, one JSON unmarshal, one
Redis `GET`. No DB query, no RPC. At VoIPBin's current single-node scale
that is a rounding error; the escape hatch (dynamic per-transcribe
binding) and its leak-sweeper requirement are pre-documented in §3 so the
switch is a decision, not a redesign.

**On the Redis resolver being the sole filter:** the key is written
explicitly at listen start and deleted explicitly at listen stop (§5.7).
It is *not* part of `cachehandler.AIcallSet`'s snapshot-index scheme
(`pkg/cachehandler/handler.go:79-97`), which writes secondary keys
(`ai:aicall:reference_id:<id>`, `ai:aicall:pipecatcall_id:<id>`) and never
invalidates the old key when the indexed field changes. Reusing that
scheme for listen state would leave stale keys pointing at stale snapshots
and would collide every non-listening AIcall on a shared nil-UUID key
(review M1). This key is a purpose-built, explicitly-managed pointer, not
a snapshot index — that distinction is the fix.

**Cache-loss behaviour (stated, not hidden):** a Redis flush drops the
resolver keys, so in-flight calls stop being listened to until the panel
is reopened (which re-runs §5.1 and repopulates). There is deliberately no
DB fallback on miss, because a DB fallback would put a query on the
platform-wide hot path — exactly the cost this design removes. Losing
best-effort proactive notifications for the remainder of a call is an
acceptable, self-healing degradation; the DB column remains the source of
truth for cleanup.

#### 5.3.3 Buffering

Two Redis lists per AIcall, both `EXPIRE`d to
`AIcallListenBufferTTLHours` (default 6h) on every push:

| Key | Op | Purpose |
|---|---|---|
| `ai:listen:pending:<aicall_id>` | `RPUSH` | lines not yet evaluated; drained atomically by the turn |
| `ai:listen:window:<aicall_id>` | `RPUSH` + `LTRIM -W -1` | rolling last `W` lines (default 40) for continuity across turns |

The line format is the structural speaker tag (see §5.9):
`"[CUSTOMER] …"` / `"[AGENT] …"`.

Two lists rather than one list plus a counter: both operations are single
atomic Redis commands, so no cross-command consistency reasoning is
needed. A line briefly present in `window` but not yet popped from
`pending` is harmless — it is context either way.

#### 5.3.4 Debounce

After buffering:

```
if !cache.ListenTurnTryLock(ctx, aicallID, interval) {  // SET ai:listen:lock:<id> NX EX <interval>
    return nil   // a turn ran recently; this line waits in the buffer
}
go h.aicallHandler.RunListenTurn(detachedCtx, aicallID)
```

`interval = config.Get().AIcallListenEvaluateIntervalSeconds` (default
20). This is a leaky-bucket debounce that:
- works across replicas (both ai-manager pods share Redis),
- needs no timers, no goroutine-per-AIcall, no in-process state,
- self-heals on pod loss (the lock TTL expires).

**Known behaviour, stated:** the last few lines before a silence are not
evaluated until the *next* line arrives. In practice a call ends shortly
after and §5.7's hangup path performs one final flush turn, so the tail is
not lost. A wall-clock flush timer is deliberately not introduced.

### 5.4 Layer 2 — the listen evaluation turn

`aicallHandler.RunListenTurn(ctx, aicallID)`:

#### 5.4.1 Preconditions

1. `c := h.Get(aicallID)`; require `c.Status == progressing`,
   `c.ReferenceType == contact_case`, `c.Metadata[listen_transcribe_id]`
   set. Otherwise clear listen state (§5.7) and return.
2. `lines := cache.ListenPendingPopAll(ctx, aicallID)` — a single `LPOP
   key count` (Redis ≥6.2), atomic, so no concurrent appender can lose a
   line between a read and a trim. Empty → return (`skipped_empty`).
3. Cost cap: a per-AIcall turn counter (`INCR ai:listen:turns:<id>`, same
   TTL as the buffer) bounded by
   `AIcallListenMaxTurnsPerAIcall` (default 60 ≈ 20 minutes of continuous
   speech at a 20s interval). Exceeded → stop listening entirely (§5.7)
   and return (`skipped_cap`). This is the hard backstop against a
   pathological long call.

#### 5.4.2 Context assembly (resolves review B2)

Built explicitly — **`getPipecatcallMessages` is not called**:

| # | Role | Content | Bound |
|---|---|---|---|
| 1 | `system` | The frozen prompt snapshot from `c.Metadata[prompt_snapshots]` (`models/aicall/main.go:12-22`) — for `AssistanceTypeAI` there is exactly one; for `AssistanceTypeTeam`, the one whose `MemberID == c.CurrentMemberID`, else the first. Already substituted at AIcall start (`start.go:128-166`), so **no DB read and no re-substitution** | 1 message |
| 2 | `system` | `ListenTurnSystemPrompt` — a new constant beside `InsightSystemPrompt` (`pkg/aicallhandler/main.go:264-282`), describing the watch task and the `notify_agent` contract (§5.5.3) | 1 message |
| 3 | `user`/`assistant` | The last `AIcallListenQAContextSize` (default 10) rows of this AIcall with `Role ∈ {user, assistant}`, oldest-first. Fetched as `messageHandler.List(ctx, 30, "", {FieldAIcallID: c.ID, FieldDeleted: false})` then filtered in-process (`ApplyFields` has no `IN` support) and truncated. Gives the AI continuity with what the agent asked and with its own earlier notifications | ≤10 messages |
| 4 | `user` | The transcript block: `cache.ListenWindowGet` (≤40 lines) rendered with a marker separating already-seen lines from the newly popped ones | 1 message, ≤40 lines |

Total: a constant-shaped, small prompt, independent of call length. The
system prompt can never be evicted, because it is not competing with
transcript rows for a 100-row window — transcript lines are not rows at
all.

#### 5.4.3 Session start

```go
turnPipecatcallID := h.utilHandler.UUIDCreate()   // NOT written to c.PipecatcallID
pc, err := h.startListenPipecatcall(ctx, c, turnPipecatcallID, llmMessages)
// → PipecatV1PipecatcallStart(ctx, turnPipecatcallID, c.CustomerID, c.ActiveflowID,
//      pmpipecatcall.ReferenceTypeAICall, c.ID, llmType, llmMessages,
//      STTTypeNone, "", TTSTypeNone, "", "")
_ = h.reqHandler.PipecatV1PipecatcallTerminateWithDelay(ctx, pc.HostID, pc.ID, defaultListenTurnTimeout) // 60s
```

`startListenPipecatcall` is a sibling of `startPipecatcall`
(`start.go:697-744`) that takes the pipecatcall id and the message list as
parameters instead of reading `c.PipecatcallID` and calling
`getPipecatcallMessages`.

**Not writing `turnPipecatcallID` to the AIcall row is the load-bearing
decision.** It means:
- no `AIcallUpdate` per turn → no `tm_update` bump → no `Send` cooldown
  interference (review L3),
- `interruptPreviousPipecatcall` is never called → an in-flight answer to
  the agent is never killed (review B1),
- and the mismatch itself becomes the drop signal (§5.4.4).

Tool calls still route correctly: pipecat POSTs to
`…/<pipecatcall_id>/tools` (`scripts/pipecat/tools.py:107`), pipecat-manager
resolves the aicall from the *pipecatcall's* `ReferenceID` (= `c.ID`), not
from `AIcall.PipecatcallID`. `ToolHandle` therefore operates on the right
AIcall.

#### 5.4.4 Suppressing all output except `notify_agent` (resolves review B1)

Two independent mechanisms, belt and braces:

**(a) `notify_agent` is defined with `RunLLM: false`.** Verified end to
end: `tool.Tool.RunLLM` (`models/tool/main.go:74-83`) is serialised to
pipecat, `_build_run_llm_defaults` reads it
(`scripts/pipecat/tools.py:58-85`), and `tool_execute` passes it as
`FunctionCallResultProperties(run_llm=should_run_llm)`
(`tools.py:105,142-152`). With `run_llm=False` the pipeline does **not**
run the LLM again after the tool result, so no `bot_llm` text frame is
produced. This is the in-convention way to express "tool fired, no
follow-up text" — every other tool in `definitions.go` uses
`RunLLM: true` precisely because it *wants* the follow-up.

**(b) A pipecatcall-identity guard on every inbound pipecat message
event.** A misbehaving model can still emit text instead of (or before)
calling the tool. `messagehandler` gains one shared helper:

```go
// isForeignPipecatcall reports whether evt.PipecatcallID differs from the
// AIcall's currently-bound PipecatcallID. True means the event came from a
// session the AIcall no longer (or never did) consider its conversational
// turn — a listen evaluation turn, or a genuinely stale reply — and MUST
// NOT be persisted or delivered.
func (h *messageHandler) isForeignPipecatcall(ac *aicall.AIcall, evtPipecatcallID uuid.UUID) bool
```

applied for `ac.ReferenceType == aicall.ReferenceTypeContactCase` in all
four handlers that today would persist or publish:

| Handler | File:line | Today | Rev 2 |
|---|---|---|---|
| `EventPMMessageBotLLM` | `messagehandler/event.go:167-180` | persists **any** non-empty text unconditionally on the non-conversation branch | drop if foreign; also pass `WithPipecatcallID(evt.PipecatcallID)` on the row it does persist |
| `EventPMMessageBotLLMIntermediate` | `event.go:260-291` | publishes an `EventTypeMessageIntermediate` **webhook per token chunk**, no aicall check | drop if foreign |
| `EventPMMessageUserLLM` | `event.go:293-307` | persists `role=user` | drop if foreign |
| `EventPMMessageUserTranscription` | `event.go:115-133` | persists `role=user` | drop if foreign |

This is a strict improvement beyond this feature: it extends to
`contact_case` the same stale-response guard the `conversation` branch
already has (`event.go:182-189`), so today's silently-persisted stale
contact_case replies stop appearing too. Metric:
`ai_manager_aicall_foreign_pipecatcall_dropped_total{handler}`.

Two existing handlers were checked and need **no** change:
- `EventPMPipecatcallTerminated` returns early unless
  `ac.ReferenceType == ReferenceTypeConversation` (`event.go:405-408`), so
  a listen turn's termination never triggers the "Sorry, I'm having
  trouble responding right now" backstop.
- `EventPMPipecatcallInitialized` returns early unless
  `cc.ReferenceType == ReferenceTypeCall` (`aicallhandler/event.go:110-112`).

Net effect: **a listen turn produces exactly zero persisted rows and zero
webhooks unless the LLM calls `notify_agent`.**

### 5.5 The `notify_agent` tool

#### 5.5.1 Definition

```go
{
    Name:   tool.ToolNameNotifyAgent,
    RunLLM: false,   // deliberate: the notification IS the output; no follow-up text
    Description: `Pushes a short, actionable note to the human agent's Insight
Assistant panel, without the agent having asked anything.

WHEN TO USE:
- You are watching a live call transcript and something just happened that the
  agent needs to know right now, per your configured instructions.

WHEN NOT TO USE:
- The agent asked you a question — answer normally instead; do not call this.
- You have nothing new or actionable to say. Saying nothing is the correct and
  expected outcome for most checks.
- You want to repeat something you already notified about on this call.

ARGUMENTS:
- message (required): one or two sentences, written for a busy human mid-call.

This is the only way to reach the agent proactively. It writes into the same
panel thread the agent is already reading; it cannot place calls, send email or
SMS, change CRM records, or spend money.`,
    Parameters: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "message": map[string]any{
                "type":        "string",
                "description": "The note to show the agent. One or two sentences.",
            },
        },
        "required": []string{"message"},
    },
}
```

Registration: `tool.ToolNameNotifyAgent` is added to
`tool.AllInsightToolNames` (`models/tool/main.go:65-72`) and **not** to
`tool.AllToolNames`. Gating then works through the existing, verified
mechanism: `amai.AllowedToolNames(TypeInsight)`
(`models/ai/tool_validation.go:36-47`) re-applied at expansion time in
`bin-pipecat-manager/pkg/toolhandler/GetByNames`
(`toolhandler/main.go:91-108`). Also needs
`message.FunctionCallNameNotifyAgent` (`models/message/tool.go`) and an
entry in `ToolHandle`'s `mapFunctions` (`aicallhandler/tool.go:54-76`).

#### 5.5.2 Relaxing the Insight read-only invariant (resolves review H1)

`models/tool/main.go:62-64` currently says *"Every entry MUST be read-only
(no side effects)"*, and
`models/ai/allowed_tools_test.go:72-88`
(`TestAllInsightToolNamesAreReadOnly`) hard-fails on any name missing from
its hardcoded `knownReadOnly` map. `notify_agent` persists and delivers a
message. **This design deliberately relaxes that invariant, narrowly:**

- The comment on `AllInsightToolNames` is rewritten to: *"Every entry must
  be read-only with respect to customer data and external systems. The
  single sanctioned exception is `notify_agent`, whose only effect is to
  write a message into the AIcall's own conversation thread — the same
  thread the agent is already reading. It cannot place calls, send
  email/SMS, mutate CRM records, or spend money. See
  docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.5.2."*
- The test keeps its name and its fail-loudly property but gains a second,
  separate map so that any *other* write tool still fails:

  ```go
  knownReadOnly := map[tool.ToolName]bool{ /* the existing six */ }
  // Sanctioned write exception -- see the design doc cited above. Adding a
  // name here requires the same explicit design-level justification.
  knownSanctionedWrite := map[tool.ToolName]bool{
      tool.ToolNameNotifyAgent: true,
  }
  for _, n := range tool.AllInsightToolNames {
      if !knownReadOnly[n] && !knownSanctionedWrite[n] { t.Errorf(...) }
  }
  ```
- `docs/plans/2026-07-30-case-insight-assistant-tool-expansion-design.md`
  §2.6 gains a one-line pointer noting the exception, so the two documents
  do not silently disagree.
- `TestValidateToolNames_WriteToolNeverAllowedForInsight`
  (`allowed_tools_test.go:94-108`) needs **no** change: it iterates
  `tool.AllToolNames`, and `notify_agent` is not a member.

**Auto-grant blast radius, acknowledged:** Insight AIs typically store
`tool_names=["all"]`, which `AllowedToolNames` expands at runtime, so every
existing Insight AI gains `notify_agent` on deploy with no re-consent
step. The accepted worst case is that a model calls it during an ordinary
Q&A turn and the agent sees one extra message in the thread they are
already looking at — no external action, no spend, no data change. The
tool description explicitly tells the model not to do this.

#### 5.5.3 Customer-configurable triggering

The Insight AI's `init_prompt` (existing field, existing editing UI) is
where the customer defines *when* to notify, e.g. *"if the customer
mentions cancellation, a compliance keyword, or requests something
requiring approval, call notify_agent with a short actionable note;
otherwise say nothing."* The frozen snapshot of that prompt is message #1
of every listen turn (§5.4.2), so no schema change and no new field is
needed to make triggering customer-defined — directly per 대표님's
direction.

`ListenTurnSystemPrompt` supplies only the mechanics (you are watching a
live call; most checks should produce no output; use `notify_agent` and
nothing else to reach the agent; do not repeat a notification you already
sent), never the business conditions.

### 5.6 How a proactive message is stored (resolves review H2)

#### 5.6.1 Rejecting `role=notification`

`message.RoleNotification` exists (`models/message/main.go:68`), is
produced today only by `EventPMTeamMemberSwitched` (`event.go:348`), and
**is skipped when assembling LLM context**: `getPipecatcallMessages` does
`if m.Role == message.RoleNotification { continue }` with the comment
*"skip non-LLM roles (e.g. notification) that would cause API errors"*
(`start.go:637-641`).

That skip is exactly why it is the **wrong** home for a proactive message.
A proactive notification is a genuine assistant utterance in the
conversation. If it were stored as `notification`, then when the agent
replies *"what did you mean by that?"*, the Q&A turn's context would not
contain what the AI had just told them — the AI would have no memory of
its own notification. That is a functional defect, not a desirable
property. `RoleNotification`'s existing use (a machine-readable
member-switch JSON blob that is genuinely not conversational) does not
generalise here.

Checked on the UI side too: neither panel special-cases roles —
square-admin styles `msg.role === 'user'` one way and everything else the
other (`CaseInsightAssistantPanel.js:44`); square-talk does the same
(`CaseInsightAssistantPanel.jsx:62`). So `role=notification` would give no
UI distinction for free anyway.

#### 5.6.2 Decision: `role=assistant` + a new `Message.Origin` field

```go
// models/message/main.go
Origin Origin `json:"origin,omitempty" db:"origin"`

type Origin string
const (
    OriginNone      Origin = ""          // default; every message that answers or asks
    OriginProactive Origin = "proactive"  // AI-initiated, not a reply to anything
)
```

- `role=assistant` → the AI remembers what it told the agent; ordinary LLM
  context replay is correct with no special-casing.
- `Origin` is an explicit, first-class marker written by ai-manager itself,
  which is possible here precisely because the row is created by
  `ToolHandle` — not, as rev 1 assumed, reconstructed from tool-call
  archaeology. Rev 1's premise was wrong: the eventual assistant *text*
  reply is created independently in `EventPMMessageBotLLM` with
  `toolCalls=nil` and no correlation column back to the tool call, so
  "did this message's generation include a `notify_agent` call" is not
  answerable from the schema.
- A string enum rather than a bool, matching `Role` / `Direction` /
  `DeliveryStatus`, so a future third origin does not need another column.
- Exposed in `message.WebhookMessage` + `ConvertWebhookMessage` — the
  frontends key their badge off it (§5.10).

#### 5.6.3 Message ordering — a correctness constraint, not a detail

`ToolHandle` (`aicallhandler/tool.go:24-100`) writes, in order:
1. `role=assistant` with `ToolCalls` (line 47),
2. runs the handler,
3. `role=tool` with the result (line 88 → `toolCreateResultMessage`).

If `toolHandleNotifyAgent` created the proactive row *inside* step 2, the
persisted sequence would be `assistant(tool_calls)` →
`assistant(text)` → `tool(result)`. `getPipecatcallMessages` replays
`tool_calls` and `tool_call_id` verbatim (`start.go:643-655`), and the
OpenAI chat API rejects a `tool_calls` assistant message that is not
immediately followed by its `tool` results — so every *subsequent* Q&A
turn on that AIcall would 400. That would be a latent, hard-to-diagnose
break.

**Therefore the proactive row is created after the tool-result row**, as
an explicit post-step in `ToolHandle`:

```go
msg, err := h.toolCreateResultMessage(ctx, c, tool, tmpMessageContent, toolCallActiveAIID)
...
// Post-step: notify_agent's visible output is emitted only AFTER the tool
// result row, so the persisted sequence stays a valid
// assistant(tool_calls) -> tool(result) -> assistant(text) chain for any
// later LLM replay (see design §5.6.3).
if tool.Function.Name == message.FunctionCallNameNotifyAgent && tmpMessageContent.Result == "success" {
    if text, errParse := parseNotifyAgentMessage(tool.Function.Arguments); errParse == nil {
        h.createProactiveMessage(ctx, c, text, toolCallActiveAIID)  // role=assistant, Origin=proactive
    }
}
```

`parseNotifyAgentMessage` (unmarshal + trim + non-empty + length cap) is
shared with `toolHandleNotifyAgent`, so validation cannot drift between the
two call sites. `toolHandleNotifyAgent` itself only validates and returns
success/failure — it writes nothing.

### 5.7 Lifecycle and cleanup

#### 5.7.1 Call hangup (resolves review B4 and simplifies B5)

**Lookup.** `EventCMCallHangup` currently resolves via
`GetByReferenceID(evt.ID)` (`aicallhandler/event.go:53-58`). For an
Insight AIcall `ReferenceID` is the **Case** id, so that lookup can never
find the listening AIcall from a call id. Rev 1's "add one more check
here" does not work.

A genuinely new lookup is required:

```go
func (h *aicallHandler) EventCMCallHangup(ctx context.Context, evt *cmcall.Call) {
    // existing path, unchanged: the AIcall whose reference IS this call
    if cc, err := h.GetByReferenceID(ctx, evt.ID); err == nil {
        _, _ = h.ProcessTerminate(ctx, cc.ID)
    }

    // new path: every contact_case AIcall listening to this call
    h.stopListenByCallID(ctx, evt.ID)
}
```

`stopListenByCallID` runs `AIcallList` with
`{FieldReferenceType: contact_case, FieldListenCallID: evt.ID,
FieldDeleted: false}` — hence the indexed **column** in §5.8 — and clears
each match. Plural on purpose: two Cases on one call each get their own
AIcall (see §5.11), and both must be cleared.

**STT stop: not ai-manager's job on this path.**
`bin-transcribe-manager/pkg/transcribehandler/event.go:51-81`
(`EventCMCallHangup`) already lists **every** non-deleted transcribe with
`reference_id == call.ID`, owner-agnostic, and stops each one. The listen
session is therefore already stopped by transcribe-manager itself. Rev 1's
`TranscribeV1TranscribeStop` call on hangup was both wrongly-signed and
redundant.

This removes the per-pod `HostID` reachability problem from the hangup
path entirely — which matters, because `bin-transcribe-manager/CLAUDE.md`
documents that `HostID` is a fresh random UUID generated on every process
start, so a persisted `HostID` can address a queue that no longer exists.

Before returning, `stopListenByCallID` runs one final flush turn
(`RunListenTurn`, bypassing the debounce lock) if `pending` is non-empty,
so the last words of the call are still evaluated. Then it clears state.

#### 5.7.2 AIcall terminates while the call is still live

Agent closes the panel, session idles out, or the turn cap (§5.4.1) trips.
Hooked into `ProcessTerminate` (`pkg/aicallhandler/process.go:38`) for
`contact_case` AIcalls. Here ai-manager *does* own the stop:

```go
if !owns { /* someone else's session (e.g. a concurrent ai_summary) — never touch it */ }
else {
    tr, err := h.reqHandler.TranscribeV1TranscribeGet(ctx, listenTranscribeID)  // shared queue, always reachable
    if err == nil && tr.Status == tmtranscribe.StatusProgressing {
        // signature verified: (ctx, hostID, transcribeID)
        //   bin-common-handler/pkg/requesthandler/transcribe_transcribes.go:113-117
        if _, errStop := h.reqHandler.TranscribeV1TranscribeStop(ctx, tr.HostID, tr.ID); errStop != nil {
            log.Warnf(...)   // NOT fatal — see below
            promListenStopFailed.Inc()
        }
    }
}
```

`HostID` is fetched **fresh** via `TranscribeV1TranscribeGet` rather than
read from a persisted column, precisely because it is regenerated on every
transcribe-manager restart.

**Stated fallback, rather than asserting cleanup always succeeds:** if the
owning pod restarted, the per-pod queue
`bin-manager.transcribe-manager-<host_id>.request` no longer exists and
the stop RPC times out. That is logged, metered, and tolerated — the
session is guaranteed to be cleaned up by transcribe-manager's own
`EventCMCallHangup` when the call ends, which is at most one call-duration
away. The failure mode is a slightly-longer-than-necessary STT session,
never a permanently orphaned one.

#### 5.7.3 Clearing state (all paths)

`clearListenState(ctx, aicallID)`:
1. `AIcallUpdate` → `listen_call_id = uuid.Nil`, remove both metadata keys
   (one write).
2. Redis: `DEL ai:listen:transcribe:<transcribe_id>`,
   `ai:listen:pending:<aicall_id>`, `ai:listen:window:<aicall_id>`,
   `ai:listen:lock:<aicall_id>`, `ai:listen:turns:<aicall_id>`.

Deleting the resolver key first is what guarantees a stale
`transcribe_id` can never be matched again by §5.3.

### 5.8 Data model and plumbing scope (resolves review M2)

The rev-1 proposal of three columns is reduced to **one column + two
metadata keys**, on the principle that only a field we must *query by*
needs to be a column.

| Field | Where | Why |
|---|---|---|
| `listen_call_id` | **new column** `ai_aicalls.listen_call_id binary(16)`, **indexed** | `EventCMCallHangup` must run `WHERE listen_call_id = ?` (§5.7.1). JSON metadata is not usefully indexable |
| `listen_transcribe_id` | `AIcall.Metadata["listen_transcribe_id"]` | only ever read with the row already in hand (idempotency check, stop path). Resolution in the hot path goes through Redis, not a DB query, so no index is needed |
| `listen_owns_transcribe` | `AIcall.Metadata["listen_owns_transcribe"]` | same |

`Metadata` is the established home for per-AIcall flags — see
`MetaKeyPromptSnapshots` / `MetaKeyAutoAuditEnabled`
(`models/aicall/main.go:21-27`). Two new constants
`MetaKeyListenTranscribeID`, `MetaKeyListenOwnsTranscribe` follow that
pattern.

**On `ai_aicalls.transcribe_id` having existed before:** migration
`bad27b40fe8e` deliberately dropped a `transcribe_id` column from
`ai_aicalls` (per `docs/plans/2026-02-24-aicall-schema-cleanup-design.md`
§3). This design does **not** re-add it. The old column was the
chatbot-era per-AIcall transcribe binding for a feature that no longer
exists; the transcribe id now lives in `Metadata`. What *is* added is a
different thing: a **call** reference, needed for an event lookup that has
no other key.

Full plumbing checklist for the one column:

- `models/aicall/main.go` — field, `json:"listen_call_id,omitempty" db:"listen_call_id,uuid"`
- `models/aicall/field.go` — `FieldListenCallID Field = "listen_call_id"`
- `models/aicall/field_test.go` — golden constant list
- `models/aicall/filters.go` — `FieldStruct` entry (`filter:"listen_call_id"`),
  required for `AIcallList` filtering via `ApplyFields`
  (`dbhandler/aicall.go:206`)
- `models/aicall/filters_test.go`
- **not** added to `models/aicall/webhook.go` / `ConvertWebhookMessage` —
  internal plumbing, same treatment as `Message.PipecatcallID`
  (`models/message/main.go:26`, `json:"-"`). No OpenAPI change, no
  `aicall_struct_aicall.rst` change follows from it.
- `bin-dbscheme-manager` migration, **generated** with
  `alembic -c alembic.ini revision -m "..."` (never a hand-picked revision
  id), adding the column and `create index idx_ai_aicalls_listen_call_id
  on ai_aicalls(listen_call_id)`. AI drafts the file; a human applies it.
- `bin-ai-manager/docs/domain.md` — AIcall entity + Metadata keys.
- `bin-ai-manager/docs/architecture.md` — the new subscription
  (`subscribeTargets` / `topicPatterns` change triggers the
  `scripts/check-service-docs.sh` warning).
- `bin-ai-manager/docs/operations.md` — new config flags and metrics.

For `Message.Origin` (§5.6.2), which **is** user-visible:

- `models/message/main.go` (field + `Origin` type), `field.go`,
  `field_test.go`, `filters.go`
- `models/message/webhook.go` + `ConvertWebhookMessage`
- `bin-dbscheme-manager` migration:
  `ai_messages.origin varchar(16) not null default ''`
- `bin-openapi-manager/openapi/openapi.yaml` → `go generate ./...`, then
  `bin-api-manager` → `go generate ./...`
- RST (§5.10.2)

`bin-contact-manager` is touched only by the one-line
`kmkase.ReferenceTypeCall` constant (§5.1 step 5) and must be run through
the full verification workflow as well.

### 5.9 Speaker mapping — unchanged open item

**`in`/`out` → customer/agent mapping still needs empirical confirmation
before merge.** Traced through
`bin-transcribe-manager/pkg/transcribehandler/start.go` and
`docsdev/source/transcribe_tutorial.rst`: `direction` is relative to the
transcribed *channel*, not the call as a whole — `in` is audio arriving
into that channel from the far end, `out` is audio sent out through it.
`Case.ReferenceID` is the customer-facing (inbound) call leg. Once that leg
is bridged to an agent, audio "sent out" through it is the bridged agent's
voice, so `in=customer`/`out=agent` is the structurally correct reading.
The only documented example (`quickstart_transcribe.rst`) is a flow/TTS
scenario, not an agent-bridged one, so this has **not been empirically
verified against a real agent-bridged call's transcript**. Reversed, this
silently mislabels who said what and could produce a materially wrong
proactive message (e.g. attributing a customer's complaint to the agent).

**Action item for implementation, before this ships:** capture one real
(or staged) agent-bridged call's transcript segments and confirm `in`/`out`
against known speaker identity. Blocks §11's sign-off, not the rest of
this design.

Tags are structural (`[CUSTOMER]`/`[AGENT]`), not localized, so prompt
behaviour does not fork by call language. `transcript.Direction` also
carries a `both` value (`models/transcript/transcript.go:41-45`); a
segment with `direction=both` is tagged `[SPEAKER]` rather than guessed.

### 5.10 Frontend (`monorepo-javascript`)

Correction to the rev-1 review's M6: `square-admin` and `square-talk` are
**not** separate repositories. Both live in the single
`monorepo-javascript` repo
(`/home/pchero/gitvoipbin/monorepo-javascript/square-admin`,
`…/square-talk`). So this feature is **two PRs total**: one in `monorepo`
(backend, lands first), one in `monorepo-javascript` (both apps).

#### 5.10.1 Both panels

- **square-admin** (`src/views/contacts/CaseInsightAssistantPanel.js`):
  already subscribes to `customer_id:{id}:aicall:{aicallId}` and receives
  every new message over the existing WebSocket path — no transport
  change. `MessageThread` currently styles on `msg.role === 'user'`
  (line 44); it gains a third branch for `msg.origin === 'proactive'`
  (distinct surface + a `Sparkles`/bell affordance + an accessible label
  such as "Proactive insight"), so a notification is never mistaken for an
  answer.
- **square-talk** (`src/features/cases/CaseInsightAssistantPanel.jsx`):
  unchanged transport (2s poll); identical `origin`-driven treatment at
  line 62.
- No backend read-surface work is needed:
  `ServiceAgentAImessageList` returns `ConvertWebhookMessage()` output
  (`bin-api-manager/pkg/servicehandler/serviceagent_aimessage.go:68-72`),
  so adding `Origin` to `WebhookMessage` is sufficient. Rev 1's open item
  #2 ("does the read surface expose what the badge needs?") is hereby
  **closed**: it does, once the field exists.

#### 5.10.2 RST docs (resolves review M7)

Mandatory per root `CLAUDE.md` for user-visible changes:

- `bin-api-manager/docsdev/source/ai_struct_message.rst` — new `origin`
  field (compared against `WebhookMessage`, not the internal struct).
- `bin-api-manager/docsdev/source/ai_struct_tool.rst` and/or
  `ai_overview.rst` — the `notify_agent` tool and the Insight-only tool
  set.
- `ai_overview.rst` — a short "Insight Assistant: live call listening"
  subsection: what triggers it, that it is `init_prompt`-driven, and that
  proactive messages are marked `origin=proactive`.
- Build procedure: `cd bin-api-manager/docsdev && rm -rf build &&
  python3 -m sphinx -M html source build`, then
  `git add -f bin-api-manager/docsdev/build/`, RST + HTML in the same
  commit.

### 5.11 Cost and concurrency bounds

**Concurrency.** Correcting rev 1: the bound is *not* "one Insight AI per
customer." The DB constraint is one active AIcall per
`(customer_id, reference_type, reference_id)` — i.e. per **Case** — via
the `active_reference_key` unique index (`start.go:359-368`). A customer
with N open Cases on N live calls therefore gets N concurrent listen
sessions. Bounds that actually apply:

| Bound | Value |
|---|---|
| STT sessions per call | 1 — any progressing `IDAIManager` session on that call is reused (§5.2.2) |
| LLM turns per AIcall per minute | `60 / AIcallListenEvaluateIntervalSeconds` = 3 at the default |
| LLM turns per AIcall, total | `AIcallListenMaxTurnsPerAIcall` = 60 (hard stop, then listening ends) |
| Tokens per turn | constant-shaped: 2 system messages + ≤10 Q&A messages + ≤40 transcript lines (§5.4.2) |
| Concurrent listen sessions | number of open Case panels whose Case call is live |

**Worst case per listened call at defaults:** 3 small LLM turns/min,
capped at 60 turns (~20 min of continuous speech), one shared STT session.
Contrast with rev 1, which was one *unbounded-context* LLM call per spoken
sentence.

**Kill switch.** `AIcallListenEnabled` defaults to **false**. The feature
ships dark and is enabled deliberately.

### 5.12 New configuration

| Flag / env | Default | Purpose |
|---|---|---|
| `aicall_listen_enabled` / `AICALL_LISTEN_ENABLED` | `false` | master kill switch |
| `aicall_listen_evaluate_interval_seconds` | `20` | debounce window (§5.3.4) |
| `aicall_listen_window_size` | `40` | rolling transcript lines in context |
| `aicall_listen_qa_context_size` | `10` | Q&A rows in context |
| `aicall_listen_max_turns_per_aicall` | `60` | hard per-AIcall turn cap |
| `aicall_listen_buffer_ttl_hours` | `6` | Redis TTL on all listen keys |
| `aicall_listen_default_language` | `en-US` | fallback when `STTLanguage` is empty |

All in `internal/config/main.go` with `SetXxxForTest` helpers, following
the existing pattern (`config/main.go:159-177`), and documented in
`bin-ai-manager/docs/operations.md`.

### 5.13 New metrics

| Metric | Labels | Meaning |
|---|---|---|
| `ai_manager_aicall_listen_start_total` | `result` = started / reused / skipped_not_listenable / failed | §5.1–5.2 outcomes |
| `ai_manager_aicall_listen_segment_total` | `result` = buffered / dropped_deleted / dropped_unknown | §5.3 intake |
| `ai_manager_aicall_listen_turn_total` | `result` = ran / skipped_locked / skipped_empty / skipped_cap / failed | §5.4 turns |
| `ai_manager_aicall_listen_notify_total` | — | proactive messages actually delivered |
| `ai_manager_aicall_foreign_pipecatcall_dropped_total` | `handler` | §5.4.4 guard firings (also covers pre-existing stale contact_case replies) |
| `ai_manager_aicall_listen_stop_failed_total` | — | §5.7.2 stop RPC failures falling back to transcribe-manager cleanup |

`ai_manager_aicall_listen_turn_total{result="skipped_locked"}` is the
direct measure of how much LLM spend the debounce is saving; if it is near
zero, the interval is too short for the traffic.

### 5.14 Code hygiene: the commented-out ghost (resolves review M3)

`bin-ai-manager/pkg/subscribehandler/transcribemanager.go` currently
contains nothing but a fully commented-out `processEventTMTranscriptCreated`
that does `aicallHandler.GetByTranscribeID(evt.TranscribeID)` →
`aicallHandler.ChatMessage(...)` — i.e. precisely the naive
one-LLM-call-per-segment design this revision rejects. It is **not
revived**. The file is rewritten to hold the real §5.3.1 implementation,
and the commented block is deleted in the same change (it would otherwise
read as an endorsed alternative).

---

## 6. Error handling and edge cases

| Case | Behaviour |
|---|---|
| Case lookup, call lookup, transcribe list/start fails | Logged, metered `skipped_*`/`failed`, listening simply does not start. Never fails AIcall start — the whole ensure path runs detached (§5.1.1) |
| Call ends between the §5.1 liveness check and `TranscribeV1TranscribeStart` | `isValidReference` rejects a non-active call (`transcribehandler/start.go:107-115`); treated as a no-op, logged |
| `TranscribeV1TranscribeStart` returns `TRANSCRIBE_ALREADY_PROGRESSING` despite the list showing none (read-then-create race, acknowledged at `transcribehandler/start.go:190-195`) | Re-run the list once and reuse the winner; if still nothing, give up and log |
| Two Cases on one live call | Two AIcalls (the unique key is per-Case), one shared STT session. The first to arrive owns it (`owns=true`); the second reuses (`owns=false`) and never stops it. Both are cleared on hangup by `stopListenByCallID`'s plural lookup |
| `transcript_created` arrives for a deleted transcript | Dropped on `TMDelete != nil` (§5.3.2, review H3) |
| `transcript_created` arrives after listening stopped | Redis resolver key already deleted → dropped |
| Redis unavailable | Buffering and the debounce lock fail → no listen turns run. Q&A is completely unaffected (it never touches these keys). Degrades to today's reactive-only behaviour |
| Redis flushed mid-call | Listening silently stops for in-flight calls until the panel is reopened (§5.3.2). Stated, accepted, self-healing |
| LLM emits text instead of calling `notify_agent` | Dropped by the pipecatcall-identity guard on all four pipecat message handlers (§5.4.4); metered. Nothing persisted, no webhook |
| LLM calls `notify_agent` with empty/whitespace/oversized `message` | Rejected in `parseNotifyAgentMessage`; `fillFailed` (same style as the other tools in `tool_insight.go`); tool-result row records the failure; **no** proactive message row |
| LLM calls `notify_agent` during a normal Q&A turn | Allowed. Produces one extra `origin=proactive` message alongside the normal answer. Harmless (§5.5.2) |
| LLM calls other Insight tools during a listen turn | Allowed (no per-turn tool restriction — §3). Adds tool rows and their webhooks. Discouraged by `ListenTurnSystemPrompt` |
| Turn cap reached on a very long call | Listening stops cleanly with `skipped_cap`; the Q&A panel keeps working normally |
| Agent asks a question while a listen turn is mid-flight | Both proceed independently on separate pipecatcalls. `Send` rotates `c.PipecatcallID`, which makes the still-running listen turn's id "foreign" — so if that turn later emits text it is dropped, and if it calls `notify_agent` the notification still lands correctly (tool routing goes through the pipecatcall's `ReferenceID`, not `AIcall.PipecatcallID`) |
| transcribe-manager pod restarted; stop RPC unreachable | Logged + metered; transcribe-manager's own hangup handler is the guaranteed backstop (§5.7.2) |

---

## 7. Testing strategy

**`bin-ai-manager` unit (gomock, table-driven, following
`pkg/aicallhandler/start_test.go`):**

1. `ensureListen` — every branch of §5.1: disabled flag; non-Insight AI;
   already-listening idempotency (must make **zero** transcribe-start
   calls); `ReferenceType != "call"`; unparseable `ReferenceID`;
   cross-customer Case; cross-customer call; each non-live call status;
   happy path start; happy path reuse.
2. **`Start` hook coverage — one test per success return of
   `startReferenceTypeContactCase`** (fresh create, stuck-Initiating,
   **reuse**), asserting the ensure step is invoked in all three. The
   reuse case is the specific regression rev 1 shipped; it gets a named
   test.
3. `EventTMTranscriptCreated` — `TMDelete != nil` drop; empty-message
   drop; Redis-miss drop (asserting **no** DB call); buffered-but-locked
   (no turn); buffered-and-unlocked (turn runs).
4. `RunListenTurn` — empty pending → skip; turn cap → stop listening;
   context assembly golden test asserting exact message count, order, and
   that the snapshot system prompt is message #1 (this is the direct
   regression test for the rev-1 context-eviction defect); asserts
   `getPipecatcallMessages` is **not** called and `c.PipecatcallID` is
   **not** written.
5. `messagehandler.isForeignPipecatcall` — for each of the four handlers:
   matching id persists/publishes, mismatched id drops, and (for
   `EventPMMessageBotLLMIntermediate`) **no webhook is published** on
   mismatch.
6. `toolHandleNotifyAgent` + the `ToolHandle` post-step — success writes
   the proactive row **after** the tool-result row (assert relative
   ordering explicitly; this pins §5.6.3), with `Role=assistant` and
   `Origin=proactive`; empty/whitespace/oversized argument writes no
   proactive row; a failed tool writes no proactive row.
7. Cleanup — `stopListenByCallID` clears **all** matching AIcalls (two-row
   case); `ProcessTerminate` stops only when `owns=true`; stop-RPC failure
   is non-fatal and metered; `clearListenState` deletes every Redis key.

**`bin-ai-manager` model/golden:**

8. `models/ai/allowed_tools_test.go` — `notify_agent` passes via
   `knownSanctionedWrite`; a hypothetical unlisted write tool still fails;
   `TestValidateToolNames_WriteToolNeverAllowedForInsight` still passes
   unchanged.
9. `pkg/subscribehandler/binding_golden_test.go` — updated to 12 patterns
   with the new one appended last.
10. `models/aicall/field_test.go`, `filters_test.go`,
    `models/message/field_test.go`, `webhook_test.go` — new fields.

**Boundary:**

11. `requesthandler` mock expectations pinning the exact
    `TranscribeV1TranscribeStart` argument list including `provider` and
    `onEndFlowID` (§5.2.2), and `TranscribeV1TranscribeStop(ctx, hostID,
    transcribeID)` argument order (§5.7.2) — both were wrong in rev 1, so
    both get an argument-shape test.

**Deferred until §5.9's empirical check lands:**

12. A pinned golden-transcript test for the `in`/`out` → `[CUSTOMER]`/
    `[AGENT]` mapping. This is exactly the silent-wrong-attribution class
    that deserves a pinned test rather than a happy-path assertion.

**Frontend (`monorepo-javascript`):**

13. Both `CaseInsightAssistantPanel` suites: renders an
    `origin: 'proactive'` message with its distinct treatment and
    accessible label; renders a normal assistant message unchanged;
    renders a message with no `origin` field (backward compatibility with
    every existing row) unchanged.

---

## 8. Rollout

1. Backend PR in `monorepo` — code + migrations (drafted, **not**
   applied), service docs, RST + rebuilt HTML. Full verification workflow
   in `bin-ai-manager`, `bin-common-handler` (if the requesthandler is
   touched), `bin-contact-manager`, `bin-api-manager`,
   `bin-openapi-manager`.
2. Human applies the two Alembic migrations.
3. Deploy with `aicall_listen_enabled=false`. Confirm zero behaviour
   change: `ai_manager_aicall_listen_*` all flat, existing Insight Q&A
   metrics unchanged.
4. Frontend PR in `monorepo-javascript` (both apps). Safe to deploy while
   the backend flag is off — no message will ever carry
   `origin=proactive`.
5. Enable the flag for one pilot customer. Watch
   `listen_turn_total{result}` (especially `skipped_locked` vs `ran`),
   `listen_notify_total`, `foreign_pipecatcall_dropped_total`, and LLM
   spend.
6. Tune `aicall_listen_evaluate_interval_seconds` from observed data
   before wider enablement.

**Rollback:** set `aicall_listen_enabled=false`. In-flight listen sessions
stop being evaluated immediately; their STT sessions are reaped by
transcribe-manager on call hangup; the `listen_call_id` column and
`origin` field are inert. No migration rollback is required.

---

## 9. Impacted files (indicative)

`bin-ai-manager`
- `models/tool/main.go` — `ToolNameNotifyAgent`, `AllInsightToolNames`, invariant comment
- `models/ai/allowed_tools_test.go` — sanctioned-write map
- `models/message/{main,field,filters,webhook}.go` + tests — `Origin`
- `models/aicall/{main,field,filters}.go` + tests — `ListenCallID`, metadata keys
- `pkg/toolhandler/definitions.go` — `notify_agent` (`RunLLM:false`)
- `pkg/aicallhandler/main.go` — `ListenTurnSystemPrompt`, config-derived constants
- `pkg/aicallhandler/start.go` — `Start` hook, `ensureListen`, `startListenPipecatcall`
- `pkg/aicallhandler/listen.go` *(new)* — `EventTMTranscriptCreated`, `RunListenTurn`, context assembly, `stopListenByCallID`, `clearListenState`
- `pkg/aicallhandler/tool.go` — `mapFunctions` entry + proactive post-step
- `pkg/aicallhandler/tool_insight.go` — `toolHandleNotifyAgent`, `parseNotifyAgentMessage`
- `pkg/aicallhandler/event.go` — `EventCMCallHangup` second lookup
- `pkg/aicallhandler/process.go` — terminate-path stop
- `pkg/messagehandler/event.go` — `isForeignPipecatcall` + four call sites
- `pkg/cachehandler/{main,handler}.go` — six listen key operations + mock regen
- `pkg/subscribehandler/{main,transcribemanager,binding_golden_test}.go`
- `internal/config/main.go` — seven flags
- `docs/{domain,architecture,operations}.md`

`bin-contact-manager` — `models/kase/kase.go` (`ReferenceTypeCall`)

`bin-dbscheme-manager` — two generated migrations

`bin-openapi-manager`, `bin-api-manager` — `Origin` in the spec, regen, RST + build

`monorepo-javascript` — both `CaseInsightAssistantPanel` files + tests

---

## 10. Review-response matrix (round 1 → rev 2)

| # | Review finding | Resolution | Where |
|---|---|---|---|
| B1 | `messagehandler` is not the dispatch path; `SendReferenceTypeOthers` rotates the pipecatcall and kills in-flight answers; 3s cooldown drops most segments; assistant text always persisted | `Send` is not used at all. Listen turns run on their own throwaway pipecatcall id, never written to the AIcall row. Output suppressed by `RunLLM:false` **and** a pipecatcall-identity guard on all four pipecat message handlers | §4, §5.4.3, §5.4.4 |
| B2 | Per-segment 100-message replay blows up cost and evicts the agent's Q&A history (and, in fact, the system prompt) | Transcript is never a message row; context is assembled explicitly from the frozen prompt snapshot + ≤10 Q&A rows + ≤40 transcript lines. Batching/debounce is core, not an optimisation | §5.3.3, §5.3.4, §5.4.2 |
| B3 | Every segment becomes a customer webhook **and** a panel-visible `role=user` bubble | Segments live only in Redis. The sole new row per notification is the intended proactive message | §5.3.3, §3 (non-goal row) |
| B4 | Hangup cleanup cannot work — `GetByReferenceID(evt.ID)` never matches an Insight AIcall (its `ReferenceID` is the Case id) | New indexed `listen_call_id` column + a second, plural lookup in `EventCMCallHangup` | §5.7.1, §5.8 |
| B5 | `TranscribeV1TranscribeStop` signature wrong; per-pod `HostID` regenerated on restart | Hangup path needs **no** stop RPC (transcribe-manager's own `EventCMCallHangup` already stops every transcribe on the call, owner-agnostic). Terminate path uses the correct `(ctx, hostID, transcribeID)` with `HostID` fetched fresh, and a stated fallback on failure | §5.7.1, §5.7.2 |
| B6 | Which `customer_id` the listen transcribe runs under is unresolved and load-bearing | Decided: `IDAIManager`, with the tenant check relocated to listen-start time and provenance-checked at event time; reuse rule made owner-aware and language-tolerant | §5.2.1, §5.2.2 |
| B7 | Trigger never fires on the Case-resume path (`start.go:512-513` returns with no status transition) | Hook moved to `Start`'s dispatch branch, covering all three success returns; a named regression test per return | §5.1 |
| H1 | `notify_agent` breaks the documented read-only invariant and its guarding test; `tool_names=["all"]` auto-grants | Invariant explicitly relaxed with a named exception; comment, test (`knownSanctionedWrite`), and the 2026-07-30 design doc all updated; auto-grant blast radius stated and bounded | §5.5.2 |
| H2 | "Origin comes free from the tool-call record" is false; `RoleNotification` is the existing precedent | Rev-1 premise confirmed wrong and dropped. `role=notification` **evaluated and rejected** (it is skipped in LLM context, so the AI would forget its own notification). Decision: `role=assistant` + first-class `Message.Origin`. Ordering constraint discovered and handled | §5.6.1, §5.6.2, §5.6.3 |
| H3 | `transcript.*.created` also carries DELETE events | `TMDelete != nil` guard at intake, with the upstream bug cited and left for its own ticket | §5.3.2, §11 |
| H4 | Event-intake volume unbounded and unaccounted for | Per-event work reduced to one Redis `GET` (no DB, no RPC); volume explicitly sized; dynamic per-transcribe binding documented as a pre-analysed escape hatch with its leak caveat | §5.3.2, §3 |
| M1 | `AIcallSet` secondary keys are never invalidated; a listen cache index would go stale and collide on nil-UUID | No cache index. The resolver key is explicitly written/deleted by the listen lifecycle and is deliberately outside `AIcallSet`'s snapshot scheme; hangup lookup uses a filtered `AIcallList` on an indexed column | §5.3.2, §5.7.1 |
| M2 | Schema/plumbing scope understated; `transcribe_id` was deliberately dropped; `Metadata` already exists for flags | Reduced to one column + two `Metadata` keys, with a full plumbing checklist and an explicit note that `transcribe_id` is **not** being re-added | §5.8 |
| M3 | Commented-out `processEventTMTranscriptCreated` ghost | Deleted; the file holds the real implementation | §5.14 |
| M4 | `kase.ReferenceType`/`ReferenceID` are `string`; no `ReferenceTypeCall` constant; wrong doc path | Constant added to the owning model; checks rewritten to `!= ReferenceTypeCall` + `uuid.FromString`; doc path corrected | §5.1 step 5 |
| M5 | `TranscribeV1TranscribeStart` real signature | Full argument list written out with `provider` and `onEndFlowID` justified, plus an argument-shape test | §5.2.2, §7.11 |
| M6 | "square-admin/square-talk are separate repos" | **Corrected**: both live in the single `monorepo-javascript` repo. Two PRs total, backend first | §5.10 |
| M7 | RST sync missing from the plan | Added, with named files and the clean-rebuild procedure | §5.10.2 |
| L1 | `transcript.Transcript` embeds `CustomerID` | Noted, but under B6's resolution it holds `IDAIManager`, so it is not usable as a tenant check; used only as a provenance sanity assertion | §5.2.1 |
| L2 | Concurrency bound is per-Case, not per-customer | Corrected, with the N-Cases-N-sessions consequence and the shared-STT mitigation spelled out | §5.11 |
| L3 | `AIcallUpdate` bumps `tm_update`, feeding the `Send` cooldown | Reduced to exactly two writes per listening session (start, stop) — never per turn — so the window is bounded and pre-agent-input; decoupling the cooldown recorded as a follow-up | §5.2.4, §11 |
| — | §3 non-goals table, §4.4 self-flagged speaker mapping (confirmed correct; do not re-litigate) | Preserved verbatim / carried forward unchanged | §3, §5.9 |

---

## 11. Open items before implementation sign-off

1. **`in`/`out` speaker mapping empirical verification (§5.9) — blocking.**
   Capture one real or staged agent-bridged call and confirm the mapping
   against known speaker identity before merge. A reversed mapping is a
   silent correctness failure, not a cosmetic one.
2. **No Jira ticket filed.** Recommend filing a `VOIP-*` ticket for this
   feature before implementation starts, per project convention.
3. **Follow-up ticket (separate):** `transcripthandler.dbDelete` publishes
   `EventTypeTranscriptCreated` on delete
   (`bin-transcribe-manager/pkg/transcripthandler/db.go:33`). This design
   defends against it rather than fixing it, because changing the emitted
   event type is a routing-key-visible change affecting every current
   subscriber. It should be fixed on its own ticket.
4. **Follow-up ticket (separate):** decouple `Send`'s cooldown from
   `tm_update` onto a dedicated `tm_last_send`. Pre-existing fragility
   (`send.go:27-32` + `dbhandler/aicall.go:240`) that this design bounds
   but does not remove.
5. **Follow-up ticket (separate):** panel noise from tool-call rows.
   `ToolHandle` writes two rows per tool invocation, both webhook-published
   and both rendered by the panels today. Pre-existing for every Insight
   tool; listening makes it more visible. Options are a role/origin-based
   render filter or suppressing webhooks for `role=tool` rows.
6. **Product decision (not blocking implementation):** whether Insight
   listening becomes a billed line item, and if so under which meter
   (§3, §5.2.1). The architecture keeps the STT cost off the customer's
   transcription bill, which makes this a clean, deliberate pricing choice
   rather than an accident.
