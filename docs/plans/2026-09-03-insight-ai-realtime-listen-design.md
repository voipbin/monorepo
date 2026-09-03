# InsightAI Realtime Listen (Proactive Notification) — Design

Status: Draft (rev 5, addressing 2026-09-03 independent review round 4)
Branch: `NOJIRA-Insight-AI-realtime-listen`
Owner: CPO-directed backend feature

## 0. Revision history

| Rev | Date | Change |
|---|---|---|
| 1 | 2026-09-03 | Initial draft. Fed each transcript segment through `aicallHandler.Send` as a `role=user` message; proactive origin derived from tool-call history; `TranscribeV1TranscribeStop(transcribeID)`; hangup cleanup via the existing `EventCMCallHangup` lookup. |
| 2 | 2026-09-03 | **Rewritten after architect review round 1 (REQUEST_CHANGES, 7 BLOCKING + 4 HIGH).** Transcript segments no longer become `Message` rows and no longer go through `Send`. New two-layer architecture: a Redis-backed transcript buffer plus a debounced *listen evaluation turn* that runs on its own pipecatcall. `notify_agent` gets `RunLLM:false`. Proactive origin becomes a first-class `Message.Origin` field, stored as `role=assistant`. Hangup cleanup uses a new indexed `listen_call_id` column. Transcribe session runs under `IDAIManager`. STT stop on hangup delegated entirely to transcribe-manager's own handler. §10 maps every review item to its resolution. |
| 3 | 2026-09-03 | **Revised after independent review round 2 (REQUEST_CHANGES, 3 BLOCKING + 4 HIGH + 4 MEDIUM + 2 LOW).** Fixed a real correctness bug: the 1:1 Redis resolver key breaks the moment a second Case listens to the same call (§5.2.4/§5.3.3, now a set). Re-diagnosed §5.6.3 — the "tool_calls ordering" fix in rev 2 was a no-op; the actual mechanism (`ToolHandle` writes the tool-call row with empty `content`, which pipecat-manager's own context filter drops) surfaces a **pre-existing production defect predating this design**, now called out explicitly and routed to its own ticket rather than papered over here. Replaced the sole `RunLLM:false` defense for `notify_agent` with an explicit reject-if-called-from-a-real-Q&A-turn guard, closing the "the agent's question silently gets no answer" hole. Added the missing `InsightSystemPrompt` guardrails to the listen-turn context. Stated plainly (§4, §5.6.4) that a proactive notification surfaces multiple rows today and added a frontend render filter as the mitigation. Decided the listen-vs-AI-summary transcribe collision (§5.2.2). Corrected the hangup-cleanup justification (§5.7.1) and the STT-stream-count claim (§5.11) to what the cited code actually establishes, rather than overstating it. Fixed metric-name namespacing (§5.13). §10 gains a round-2 matrix. |
| 4 | 2026-09-03 | **Revised after independent review round 3 (REQUEST_CHANGES, 4 BLOCKING + 3 HIGH + 5 MEDIUM).** Two of rev 3's own new mechanisms turned out broken: §5.4.4(c)'s reject-guard assumed a pipecatcall id ai-manager doesn't actually receive (fixed by a real, scoped cross-service change, §5.4.3a) and §5.2.2a's summary-transcribe reuse-tolerance broke `summaryhandler`'s read/lifecycle assumptions (replaced by giving listen its own system customer id, §5.2.1, so it never shares a transcribe with `ai_summary` at all). Separately, rev 1's original defect — Q&A context eviction via `getPipecatcallMessages`'s 100-row window — resurfaced through listen-turn tool-call rows and is closed at the source (§5.4.5: a new `Origin=listen_internal` tag excluded from replay at the query level), which also narrows the orphaned-tool-message finding back to a non-blocking follow-up. Narrowed the pipecatcall-identity guard to the two handlers that can actually fire from a listen turn and scoped its cache-bypass re-read to the one handler that persists (§5.4.4(b)). Fixed a self-contradictory cleanup step order, a stale citation, and frontend field-name mismatches (camelCase vs. the actual snake_case wire fields) that would have made the render filter never fire. §10.2 gains a round-3 matrix. |
| 5 | 2026-09-03 | **Revised after independent review round 4 (REQUEST_CHANGES, 4 BLOCKING + 5 HIGH + 9 MEDIUM + 3 LOW).** Reviewer's own assessment: no remaining structural blockers, only implementation-level bugs in rev 4's three new mechanisms. Fixed §5.4.4(c)'s inverted branch logic (the cache-bypass re-read was in the wrong condition, both failing to catch the case it was meant for and creating a new false-allow) with a single always-fresh-read design. Added explicit `pipecatcallID == uuid.Nil` handling (safe default: treat as a real Q&A turn, never as listen-internal) to protect against permanent message-tagging corruption during a rolling deploy, and confirmed the wire field can be optional so no forced deployment order is needed. Corrected `ApplyFields`'s actual location (`bin-common-handler`, not `bin-ai-manager`) and replaced the unspecified `FieldOriginNot` with a concrete, generic `databasehandler.NotEq` wrapper type. Closed the remaining system-prompt-eviction gap (proactive/real-Q&A-tool rows still competing for the 100-row window) by fetching the leading system row(s) unconditionally, separate from the capped window. Scoped `Origin` tagging to `contact_case` only, so ordinary conversation-AIcall pipecatcall rotation is never mistagged. Fixed the cache-bypass re-read's actual code path (an RPC client argument, not a direct `dbhandler` call), a stop-time `Send`-cooldown collision, a missing OpenAPI tool-name enum update, and several citation errors. §10.3 gains a round-4 matrix. |

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
| Billing Insight listening to the customer | The listen STT session runs under `cmcustomer.IDAIManagerListen` (§5.2.1), so it is not attributed to the customer's transcription usage. Whether Insight listening becomes a billed line item is a pricing decision, not an architecture one | A pricing decision to monetise Insight listening |
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
      │   • Redis SMEMBERS ai:listen:transcribe:<transcribe_id>│
      │     → empty = not ours, drop; else fan out per AIcall  │
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
- **Nothing a listen turn produces reaches the customer's (tenant's) own
  webhook endpoint or the agent's panel** unless the LLM deliberately
  calls `notify_agent` — and **corrected in rev 3**: when it does, or when
  it uses any other Insight tool during a listen turn (§6, still allowed),
  more rows go out than the one intended notification, because
  `ToolHandle` writes a tool-call row and a tool-result row for every
  tool invocation, and both are webhook-published and panel-rendered
  today. So the guarantee is "silence unless the LLM acts," not "exactly
  one row when it does" — see §5.6.4 for what the tenant's webhook
  consumer and the agent's panel actually receive, and the mitigation.

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

#### 5.2.1 Owner — a listen-only system customer id, distinct from `IDAIManager` (resolves review-round-1 B6, revised in rev 4 to also resolve review-round-3 B3)

Rev 1 left this unresolved. Rev 2/3 decided `IDAIManager` (the same
system id `summaryhandler.startReferenceTypeCall` uses,
`bin-ai-manager/pkg/summaryhandler/start.go:84-99`) and then, when review
round 2 found that this makes listen collide with a concurrent AI
summary's transcribe session, tried to fix the collision by making the
two features share and hand off ownership of one session (§5.2.2a in rev
3). **Review round 3 (finding B3) showed that hand-off design is itself
broken two ways**: `summaryhandler.contentGetTranscripts`
(`pkg/summaryhandler/content.go:141-172`) reads with `size=1` and no
transcribe-id pin, so it can silently pick up listen's session instead of
its own; and §5.7.2's `owns=true` stop path can tear down a transcribe
that a *later-arriving* summary is now also depending on, cutting a paid
summary's STT off mid-call while the call is still live.

**Decision, revised in rev 4: give listen its own system customer id,
separate from `IDAIManager`.** `cmcustomer.IDAIManagerListen` (exact
value TBD at implementation time — a new constant alongside `IDAIManager`
in `bin-customer-manager/models/customer/customer.go`). This makes
listen's and summary's transcribe sessions **provably independent** at
the `startLive` dedup-guard layer (`transcribehandler/start.go:196-214`,
scoped by `customer_id`), because they are never the same owner — no
hand-off logic, no shared lifecycle, no read-path ambiguity, and §5.2.2a
is **deleted, not fixed**: `summaryhandler.startReferenceTypeCall` needs
no change at all.

| | `IDAIManagerListen` (chosen, rev 4) | shared `IDAIManager` (rev 2/3, reverted) | tenant `customer_id` (rejected) |
|---|---|---|---|
| Customer's transcribe list | clean | clean | shows a session they never started |
| Billing | platform-borne | platform-borne | silent surprise transcription charge |
| Collision with the customer's own live transcribe | none (dedup guard scoped by `customer_id`) | none | a same-language customer session 409s our `Start` |
| Collision with `ai_summary`'s transcribe | **none — different owner, guard never fires** | real, and its fix (§5.2.2a) introduced B3 | n/a |
| Cost when listen and a summary are both live on the same call | up to 2 separate transcribe sessions (up to 4 STT streams, §5.11) instead of 1 shared session — accepted, bounded, platform-borne | 1 shared session, but with B3's correctness risk | n/a |
| Precedent in-repo | new sentinel id, same *pattern* as `IDAIManager`'s existing precedent | yes (summary) | none |

**Implementation-time confirmation needed:** whether a new system
`customer_id` sentinel requires an actual row in `bin-customer-manager`'s
customer table (FK-backed) or is usable as a bare UUID constant the way
`IDAIManager` apparently already is (per round 1/2's verified read-only
usage) — flagged in §11 rather than assumed here.

**Deliberately not added, stated rather than silently decided (review
round 4, finding M1):** `bin-customer-manager`'s hardcoded
"known-system-id" whitelist (`models/customer/ids.go` and its inline
duplicate in `bin-call-manager/pkg/callhandler/validate.go`) is used to
gate certain call-origination paths — not anything on listen's transcribe
start/list/stop path, so `IDAIManagerListen` not being in that whitelist
does **not** block this feature (verified: listening never calls into
whatever checks that list). It is left out deliberately, not by
oversight: adding a new sentinel to a validation whitelist it was never
designed to need would be an unrelated, unrequested change to
`bin-customer-manager`'s and `bin-call-manager`'s call-path validation —
exactly the kind of scope creep root `CLAUDE.md`'s "smallest change that
works" principle argues against here. Revisit only if a future feature
needs `IDAIManagerListen` to pass that specific gate.

**Consequence, stated explicitly:** the event-time tenant check rev 1
proposed (`AIcall.CustomerID == transcript's CustomerID`) is impossible —
it would *always* fail, because the transcript's `CustomerID` is the
system id (`IDAIManagerListen`), never a tenant id. The tenant boundary is
therefore enforced **once, at listen-start time** (§5.1 steps 4 and 6:
customer-scoped `CaseGet` + `CustomerID` recheck on both the Case and the
call), and the event path instead verifies *provenance*: "is this
`transcribe_id` one we ourselves started and recorded?" (§5.3). That is a
stronger property than a field comparison — the id is one ai-manager
generated and persisted, not attacker-influenceable.

**Second consequence:** `get_call_transcript`'s own listing is filtered by
`tmtranscribe.FieldCustomerID: c.CustomerID` (`tool_insight.go:757-758`),
so it will not see the listen session. That is correct and intended: the
agent reads *finished* transcripts of *the customer's own* sessions
through that tool; the live listen session is an internal, platform-owned
stream. Nothing regresses.

#### 5.2.2 Reuse rule — listen-to-listen only, language-tolerant

`startLive`'s duplicate guard is scoped `(customer_id, reference_id,
language, status=progressing, deleted=false)`
(`transcribehandler/start.go:196-214`). Under §5.2.1's rev-4 decision
(listen's own `IDAIManagerListen`, never shared with `ai_summary`'s
`IDAIManager`), the **only** sessions listening can ever collide with are
**other listen sessions on the same call** — the two-Cases-one-call case
(§5.11) — never a concurrent `ai_summary`.

```
existing := TranscribeV1TranscribeList(ctx, "", 10, {
    customer_id:   cmcustomer.IDAIManagerListen,
    reference_id:  callID,
    status:        progressing,
    deleted:       false,
})
```

- **Any** progressing `IDAIManagerListen` session on this call is reused,
  *regardless of its language*, and `Metadata[listen_owns_transcribe] =
  false`.
  Rationale: starting a second session only because the language string
  differs would double the STT cost on one call to gain nothing — the LLM
  reads whatever language comes out. Maximising reuse is the cheaper and
  simpler rule. This is the explicit answer to review round 2's "the
  reuse rule must account for language/owner."
- Otherwise start one, with `Metadata[listen_owns_transcribe] = true`:

  ```go
  tr, err := h.reqHandler.TranscribeV1TranscribeStart(
      ctx,
      cmcustomer.IDAIManagerListen, // customerID  (§5.2.1)
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
  under their own `customer_id`, or one started by `ai_summary` under
  `IDAIManager` — is never reused and never touched. We cannot see it
  (different owner in our filter) and must not affect its lifecycle. This
  is now structurally guaranteed for `ai_summary` specifically, not just
  a filtering convention — see §5.2.1's rev-4 revision. (§5.2.2a, which
  in rev 3 tried to fix the `ai_summary` collision by making
  `summaryhandler` reuse-tolerant, is **deleted in rev 4**: review round
  3 (finding B3) showed that fix broke `summaryhandler`'s own read path
  and lifecycle assumptions. §5.2.1's separate-owner decision removes the
  collision at its source instead, so `summaryhandler` needs no change at
  all.)

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

**New in rev 4 (review round 3, finding M2):** if `c.Metadata` already
carries a *different* `listen_transcribe_id` — the §5.1.1 step-3
idempotency check found the old session no longer valid and started a
fresh one — `UpdateListenState` first `SREM`s this AIcall's own id from
the **old** transcribe's resolver set before `SADD`-ing it to the new
one. Without this, the stale membership survives until its 12h TTL,
which does no *functional* harm (the old transcribe's own events have
stopped, so nothing is buffered against it — §5.4.1's precondition would
also refuse to act on a mismatched `listen_transcribe_id`) but leaves an
unnecessary, undocumented dangling set entry.

`UpdateListenState` performs **one** `AIcallUpdate` writing:
- column `listen_call_id = callID` (§5.8),
- `metadata[listen_transcribe_id] = tr.ID`,
- `metadata[listen_owns_transcribe] = owns`,

and then adds this AIcall to the Redis resolver **set**:
`SADD ai:listen:transcribe:<tr.ID> <c.ID>`, `EXPIRE ai:listen:transcribe:<tr.ID> 12h`.

**Set, not a single value — fixed in rev 3.** Rev 2 wrote a single key
(`ai:listen:transcribe:<tr.ID> = <c.ID>`), which directly contradicts
§5.2.2's own reuse rule and §5.11's own edge case: **N AIcalls can share
one listen transcribe** (two Cases on one call, §5.11). With a
single-valued key, the second AIcall's `UpdateListenState` would silently
overwrite the first's mapping — the first AIcall stops receiving segments
for the rest of the call, with no error and no metric — and either
AIcall's `clearListenState` would delete the shared key out from under the
other. A set fixes both: every listening AIcall adds itself
(`SADD`), every listening AIcall removes only itself on cleanup (`SREM`,
§5.7.3), and Redis deletes the key automatically once the set is empty.
§5.3.2's intake step becomes `SMEMBERS`, fanning the same segment out to
every AIcall in the set (still one Redis round trip per platform speech
turn — `SMEMBERS` costs the same order as `GET` for the tiny (≤2-3 member,
in every observed case) sets this key ever holds).

**This is the only `ai_aicalls` write the feature makes during a listening
session** (one at start, one at stop). It is *not* per turn. That bounds
the known `tm_update` ↔ `Send`-cooldown coupling
(`dbhandler/aicall.go:240` bumps `tm_update`; `send.go:27-32` reads it) to
two ~3s windows per listening session, not an unbounded number: one right
after listening starts (inside a detached goroutine during panel open,
before the agent could plausibly have typed a question — negligible), and
**one right after it stops (§5.7.3) — flagged as a real cost in rev 5,
review round 4 finding H1, not negligible.** Listening stops on call
hangup, which is exactly when an agent is likely to ask the Insight AI a
follow-up ("what was that about?"); a `Send()` landing inside that ~3s
window gets rejected by the cooldown it did nothing to deserve. Rather
than accept this silently, `clearListenState` (§5.7.3) skips its
`AIcallUpdate`'s `tm_update` bump specifically: the write uses
`dbhandler.AIcallUpdateWithoutTouchingTMUpdate` (or an equivalent
targeted-column update that bypasses the standard `tm_update`-on-any-write
convention) for this one write path, so listen's own bookkeeping never
contributes to the cooldown at all — start or stop. This is narrower and
safer than the `tm_last_send` decoupling recorded as a follow-up in §11:
it fixes listen's own two writes specifically, without touching `Send`'s
cooldown semantics for every other AIcall write path.

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

and `EventTMTranscriptCreated` opens with a single Redis `SMEMBERS`
(§5.2.4's fix — a set, not a single value):

```
aicallIDs, ok := cache.ListenAIcallIDsGet(ctx, evt.TranscribeID)  // SMEMBERS ai:listen:transcribe:<id>
if !ok || len(aicallIDs) == 0 { return nil }   // not a session we started — 99.9% of platform events end here
for _, aicallID := range aicallIDs {
    // buffer + debounce (§5.3.3/§5.3.4) independently per listening AIcall
}
```

**Sized cost of keeping the wildcard:** per final STT result anywhere on
the platform — one AMQP delivery, one goroutine, one JSON unmarshal, one
Redis `SMEMBERS`. No DB query, no RPC. At VoIPBin's current single-node
scale that is a rounding error; the escape hatch (dynamic per-transcribe
binding) and its leak-sweeper requirement are pre-documented in §3 so the
switch is a decision, not a redesign.

**On the Redis resolver being the sole filter:** the key is written
explicitly at listen start and deleted (well, `SREM`'d) explicitly at
listen stop (§5.7). It is *not* part of `cachehandler.AIcallSet`'s
snapshot-index scheme (`pkg/cachehandler/handler.go:79-97`), which writes
secondary keys (`ai:aicall:reference_id:<id>`, `ai:aicall:pipecatcall_id:<id>`)
and never invalidates the old key when the indexed field changes. Reusing
that scheme for listen state would leave stale keys pointing at stale
snapshots and would collide every non-listening AIcall on a shared
nil-UUID key (review round-1 M1). This key is a purpose-built,
explicitly-managed pointer, not a snapshot index — that distinction is
the fix.

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
| 1 | `system` | `InsightSystemPrompt` (`pkg/aicallhandler/main.go:264-282`) — **added in rev 3**. `startInitMessages` normally puts this first for every `ai.TypeInsight` AIcall (`start.go:790-797`), ahead of the customer's own `init_prompt`. Rev 2's context assembly read only `Metadata[prompt_snapshots]`, which holds **just** the substituted `init_prompt` (`buildPromptSnapshots`, `start.go:128-166`) — it never captured `InsightSystemPrompt`. Without it, a listen turn ran with none of the platform's own Insight guardrails ("base every answer strictly on retrieved data", "never expose raw JSON or tool responses", "never mention tool names/JSON/backend logic") — exactly the rules that keep *unsolicited* output sane. `InsightSystemPrompt` is a fixed platform constant (not per-customer), so it needs no DB read either | 1 message |
| 2 | `system` | The frozen prompt snapshot from `c.Metadata[prompt_snapshots]` (`models/aicall/main.go:12-22`) — for `AssistanceTypeAI` there is exactly one; for `AssistanceTypeTeam`, the one whose `MemberID == c.CurrentMemberID`, else the first. Already substituted at AIcall start (`start.go:128-166`), so **no DB read and no re-substitution** | 1 message |
| 3 | `system` | `ListenTurnSystemPrompt` — a new constant beside `InsightSystemPrompt`, describing the watch task and the `notify_agent` contract (§5.5.3) | 1 message |
| 4 | `user`/`assistant` | The last `AIcallListenQAContextSize` (default 10) rows of this AIcall with `Role ∈ {user, assistant}`, oldest-first. Fetched as `messageHandler.List(ctx, 30, "", {FieldAIcallID: c.ID, FieldDeleted: false})` then filtered in-process (`ApplyFields` has no `IN` support) and truncated. Gives the AI continuity with what the agent asked and with its own earlier notifications | ≤10 messages |
| 5 | `user` | The transcript block: `cache.ListenWindowGet` (≤40 lines) rendered with a marker separating already-seen lines from the newly popped ones | 1 message, ≤40 lines |

Total: a constant-shaped, small prompt, independent of call length. Both
system prompts can never be evicted, because they are not competing with
transcript rows for a 100-row window — transcript lines are not rows at
all. (The flow-parameter JSON block `startInitMessages` also appends for
some AIcalls is deliberately *not* replayed here — listen turns are never
flow-parameterized, since a panel-started `contact_case` AIcall has
`ActiveflowID == uuid.Nil`, so there is nothing for it to carry.)

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

#### 5.4.3a Threading pipecatcall identity into `ToolHandle` (new in rev 4 — infrastructure prerequisite for §5.4.4(c), §5.6.5, and B1's fix below)

**Review round 3 (finding B4) showed §5.4.4(c) as written in rev 3 cannot
be implemented**: it assumed `ToolHandle` already knows which pipecatcall
a tool call arrived on, but it does not. Traced end to end:
`bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:457` calls
`AIV1AIcallToolExecute(ctx, pc.ReferenceID, request.ID, ...)` — `pc.ID`
(the pipecatcall id) is in hand but never passed. The RPC signature
(`bin-common-handler/pkg/requesthandler`), the wire DTO
(`bin-ai-manager/pkg/listenhandler/models/request/aicalls.go`), and
`ToolHandle(ctx, id, toolID, toolType, function)`
(`aicallhandler/tool.go:24`) all lack the field.

**Decision: thread it through, as a small, explicit cross-service
addition — not a workaround.** This single addition is what makes
§5.4.4(b)'s drop-if-foreign guard, §5.4.4(c)'s reject-guard, and B1's fix
(below) all implementable from one shared signal, rather than three
separate mechanisms:

1. `runner.go:457` passes `pc.ID` as an added `pipecatcallID uuid.UUID`
   argument to `AIV1AIcallToolExecute`.
2. `bin-common-handler/pkg/requesthandler`'s `AIV1AIcallToolExecute`
   signature and its wire marshalling gain the field; regenerate its
   mock.
3. `V1DataAIcallsIDToolExecutePost` (`listenhandler/models/request/aicalls.go`)
   gains `pipecatcall_id`, **`json:"pipecatcall_id,omitempty"` — omittable,
   not required** (see the rollout note below); the `listenhandler`
   handler for `POST /v1/aicalls/<uuid>/tool_execute`
   (`pkg/listenhandler/v1_aicalls.go`) passes it through as an unwrapped
   argument (root `CLAUDE.md`'s transport-DTO-ownership rule — this stays
   a domain argument from here on, never a `request.*` value past
   `listenhandler`).
4. `ToolHandle(ctx, id, toolID, toolType, function, pipecatcallID
   uuid.UUID)` — one new parameter on the interface method itself
   (`pkg/aicallhandler/main.go`'s `AIcallHandler` interface, not just its
   implementation). **Scope corrected in rev 5, review round 4 finding
   M2**: this is not "one new parameter" in isolation — `mapFunctions`'
   value type (`tool.go:54-76`) is
   `func(ctx, *aicall.AIcall, *message.ToolCall) *messageContent`, shared
   by all 21 `toolHandleXxx` functions. `pipecatcallID` is consumed by
   `ToolHandle` itself (for §5.4.5's `Origin` tagging, decided before
   dispatch) and passed explicitly only to `toolHandleNotifyAgent`
   (§5.4.4(c)) — the other 20 handlers' signatures are **unchanged**,
   avoiding a 21-function signature churn for a value only one handler
   needs. Regenerate the `AIcallHandler` mock (any other file that
   implements or mocks this interface — checked at implementation time —
   picks up the new method signature too).
5. Inside `ToolHandle`, the same comparison used for §5.4.5's `Origin`
   tagging and §5.4.4(c)'s reject-guard — see both for the exact,
   corrected logic (rev 5 fixes both; rev 4's version of each had a
   distinct bug).

This is a real, scoped cross-service change (pipecat-manager +
common-handler + ai-manager's listen surface + `ToolHandle`'s callers),
correctly reflected in §8's rollout and §9's impacted-files list — rev 3
under-scoped this as a single `mapFunctions` line; that was wrong.

**Rollout ordering, new in rev 5 (review round 4, finding B2).** Making
the wire field optional (step 3) means a rolling deploy where
`bin-pipecat-manager` still runs the old binary sends no
`pipecatcall_id` at all — unmarshalled as `uuid.Nil`. §5.4.4(c) and
§5.4.5 both already treat `uuid.Nil` as "assume this is the agent's real
turn" (fail toward doing nothing new, never toward mistagging real
content — see §5.4.4(c)'s worked rationale), so an old `pipecat-manager`
talking to a new `ai-manager` **degrades safely**: `notify_agent` calls
simply get rejected (harmless — nothing calls it before listening ships
anyway) and no row is ever mistagged. The unsafe direction — new
`pipecat-manager` behavior reaching an old `ai-manager` that doesn't
expect the field — cannot happen (an old `ai-manager` ignores an unknown
JSON field). Net: **no forced deployment order is required between the
three touched services**, but `bin-pipecat-manager` should still land
first as ordinary good practice (its change is additive and inert until
`ai-manager`'s side consumes it).

#### 5.4.4 Suppressing all output except `notify_agent` (resolves review-round-1 B1)

Three independent mechanisms, not two — rev 2's "belt and braces" was
missing a buckle. Review round 2 (finding F3) showed `RunLLM: false` is
not the reliable primitive rev 2 treated it as, so it is now the weakest
of three layers rather than the primary one.

**(a) `notify_agent` is defined with `RunLLM: false` — a best-effort
hint, not a guarantee.** Verified end to end: `tool.Tool.RunLLM`
(`models/tool/main.go:74-83`) is serialised to pipecat,
`_build_run_llm_defaults` reads it (`scripts/pipecat/tools.py:58-85`), and
the happy path passes `FunctionCallResultProperties(run_llm=should_run_llm)`
(`tools.py:105,142-152`), suppressing the follow-up `bot_llm` text frame.
**Three caveats review round 2 found, all confirmed by re-reading
`tools.py`:**
1. **Every error path drops `properties` entirely** (`tools.py:135-138`
   HTTP≥400, `:156-159` timeout, `:163-166` `ClientError`, `:170-173`
   generic) — a failed or slow `notify_agent` call re-runs the LLM and can
   still emit follow-up text.
2. **The model can override the default.** `args.pop("run_llm", …)`
   (`tools.py:105` — **line corrected in rev 5, review round 4 finding
   M5**: rev 4 cited `tools.py:~60`) takes the LLM's own `run_llm`
   argument first if it supplies one — `models/tool/main.go:75-77`'s own
   comment says as much: *"The LLM can still override this per-call via a
   `run_llm` argument."*
3. **Correcting a false claim from rev 2**: it is not true that "every
   other tool in `definitions.go` uses `RunLLM: true`" — 9 of the 21 tools
   do not (`connect_call`, `send_email`, `send_message`, `stop_media`,
   `stop_service`, `stop_flow`, `set_variables`, `get_variables`,
   `get_aicall_messages`; `definitions.go` lines 10, 83, 163, 240, 276,
   306, 335, 376, 409). The accurate, and stronger, statement is that
   **all six existing Insight tools** use `RunLLM: true` — their tool
   *definitions* (as opposed to their handler implementations, which do
   live in `tool_insight.go`) are registered in
   `pkg/toolhandler/definitions.go` at **lines 754-755, 785-786, 824-825,
   849-850, 879-880, 906-907** (**line numbers corrected in rev 5, review
   round 4 finding M4**: rev 4's fix for review round 3's M3 corrected the
   *file* but reused the *wrong* line numbers — the ones for the
   `RunLLM: false` tools listed just above, not the six Insight tools'
   own lines) — which is exactly why `notify_agent` being the one Insight
   tool with `RunLLM: false` is a deliberate outlier, not a "usual
   pattern," and needs (b)/(c) below to actually hold.

**(b) A pipecatcall-identity guard on every inbound pipecat message event
that can actually fire from a listen turn.** `messagehandler` gains one
shared helper:

```go
// isForeignPipecatcall reports whether evt.PipecatcallID differs from the
// AIcall's currently-bound PipecatcallID. True means the event came from a
// session the AIcall no longer (or never did) consider its conversational
// turn — a listen evaluation turn, or a genuinely stale reply — and MUST
// NOT be persisted or delivered.
func (h *messageHandler) isForeignPipecatcall(ac *aicall.AIcall, evtPipecatcallID uuid.UUID) bool
```

applied for `ac.ReferenceType == aicall.ReferenceTypeContactCase` in the
**two** handlers that can actually fire from a listen turn — **narrowed
from four in rev 3, review round 3 finding H2**: `EventPMMessageUserLLM`
(`event.go:293-307`) and `EventPMMessageUserTranscription`
(`event.go:115-133`) are both driven by an STT leg, and a listen turn is
started with `STTTypeNone` (§5.4.3's `startListenPipecatcall` call), so
neither event can ever originate from a listen turn's pipecatcall. **Cost
argument corrected in rev 5, review round 4 finding M6**: both handlers
already call `resolveActiveAIID` → `AIV1AIcallGet` per event today
(`event.go:73-83`, called from `:125` and `:299`), so adding the guard
would not be a *new* AIcall lookup — the real reason to leave them
unguarded is purely structural (the condition the guard checks for cannot
occur on these two paths, per `STTTypeNone` above), not a cost argument
this design invented. Left unchanged, exactly as rev 1 had them.

| Handler | File:line | Today | Rev 4 |
|---|---|---|---|
| `EventPMMessageBotLLM` | `messagehandler/event.go:167-180` | persists **any** non-empty text unconditionally on the non-conversation branch | drop if foreign; also pass `WithPipecatcallID(evt.PipecatcallID)` on the row it does persist |
| `EventPMMessageBotLLMIntermediate` | `event.go:260-291` | publishes an `EventTypeMessageIntermediate` **webhook per token chunk**, no aicall check | drop if foreign |

This is a strict improvement beyond this feature: it extends to
`contact_case` the same stale-response guard the `conversation` branch
already has (`event.go:182-189`), so today's silently-persisted stale
contact_case replies stop appearing too. Metric:
`ai_manager_aicall_foreign_pipecatcall_dropped_total{handler}`.

**Correctness caveat found in review round 2 (F4): a stale cache read can
turn (b) into a false positive against a genuine answer.**
`AIcallGet` is cache-first (`pkg/dbhandler/aicall.go:112-115`), and
`AIcallUpdate`'s cache refresh discards its own error
(`_ = h.aicallUpdateToCache(ctx, id)`). If the Redis write right after a
real `Send()`'s `UpdatePipecatcallID` (`aicallhandler/db.go:244-248`)
transiently fails, the cached AIcall keeps the *old* `PipecatcallID` for
up to its TTL — and (b) would then drop the agent's genuine answer as
"foreign."

**Fix, path corrected in rev 5 (review round 4, finding H4).**
`EventPMMessageBotLLM` gets the AIcall via
`h.reqHandler.AIV1AIcallGet(ctx, ...)` (`messagehandler/event.go:160`) —
a RabbitMQ RPC to ai-manager's own `listenhandler`, not a direct
`dbhandler` call, and rev 4's `dbhandler.AIcallGet(skipCache: true)`
citation was never on this code path. `AIV1AIcallGet`'s underlying
`listenhandler` route already resolves through the same cache-first
`dbhandler.AIcallGet` rev 4 assumed `messagehandler` called directly, so
the fix is: `AIV1AIcallGet` gains an optional cache-bypass argument (or a
sibling method), threaded down to the `dbhandler.AIcallGet(skipCache:
true)` call rev 4 correctly identified as the eventual target — it is
just one RPC hop further away than rev 4's snippet showed. On a
mismatch, `messagehandler` issues this cache-bypassing variant of the
same RPC it already makes, following the same shape as the `conversation`
branch's own stale-reply guard at `event.go:209-219`, which this design's
guard is explicitly modelled on. Only drop if the DB-authoritative read
still disagrees. (`bin-ai-manager`'s `listenhandler` route and RPC client
both need this optional parameter — a small, contained addition confined
to `bin-ai-manager`, unlike §5.4.3a's cross-service thread.)

**Scoped to the one handler that actually persists — review round 3
finding H1.** `EventPMMessageBotLLMIntermediate` fires once per streamed
token chunk, not once per message; re-reading the AIcall bypassing cache
on every mismatched intermediate chunk would put an uncached DB read on
that hot path for the entire duration of every listen-turn (or genuinely
stale) reply. It only ever *publishes a webhook*, never persists a row —
so a false-positive drop there costs nothing more than one skipped
intermediate-token webhook, which is not user-visible (only the final
`EventPMMessageBotLLM` message matters to the agent). The cache-bypass
re-read therefore applies **only** to `EventPMMessageBotLLM`;
`EventPMMessageBotLLMIntermediate` drops on a plain in-memory mismatch,
no re-read.

Two existing handlers were checked and need **no** change:
- `EventPMPipecatcallTerminated` returns early unless
  `ac.ReferenceType == ReferenceTypeConversation` (`event.go:405-408`) —
  **noted as a real gap, not dismissed**: this means `contact_case` has no
  termination-triggered "Sorry, I'm having trouble responding right now"
  backstop the way `conversation` does, so (b)'s cache-bypass re-read in
  the paragraph above is the only safety net for a stale-cache false
  drop on this reference type. Acceptable because it is a re-read, not a
  guess.
- `EventPMPipecatcallInitialized` returns early unless
  `cc.ReferenceType == ReferenceTypeCall` (`aicallhandler/event.go:110-112`).

**(c) `toolHandleNotifyAgent` itself rejects the call outright when it did
not arrive on a listen turn — closing the review-round-2 F3 hole, now
actually implementable via §5.4.3a.** (a) alone has a failure mode rev 2
did not analyze: if `notify_agent` is invoked during the agent's *own*
Q&A turn (on `c.PipecatcallID` itself, not a throwaway listen-turn id) and
`run_llm=False` actually takes effect, the agent's real question gets
**no answer at all** — just an unrelated notification. §6 ("LLM calls
`notify_agent` during a normal Q&A turn → Allowed... Harmless") was wrong
about this in rev 2; it is not harmless. **Rev 3's claim that pipecat
"already sends the pipecatcall id on every `tool_execute` POST" to
`ToolHandle` was wrong** (review round 3, finding B4) — `tools.py:107`
puts it in the URL path from pipecat to pipecat-manager only;
pipecat-manager's own `runner.go:457` never forwards it to ai-manager. §5.4.3a
fixes exactly this, and `toolHandleNotifyAgent` now receives the real
`pipecatcallID` as a parameter (threaded from `ToolHandle`, itself
threaded per §5.4.3a).

**Rewritten in rev 5 (review round 4, finding B1): rev 4's two-branch
version put the cache-bypass re-read inside the wrong branch.** Walking
both stale-cache directions shows why a single-branch check is wrong no
matter which branch carries the re-read: if the cached `c.PipecatcallID`
is stale, an `==` comparison against it can be a false match *or* a false
mismatch depending on which way the staleness runs, so *either* outcome
of the cheap comparison can be wrong. The fix is not to move the re-read
to the other branch — it is to not trust the cheap comparison's *outcome*
at all, only use it as a hint, and always resolve the actual decision
from one authoritative read:

```go
// notify_agent is not on a hot path (at most one call per listen turn,
// itself debounced to one per AICallListenEvaluateIntervalSeconds — §5.3.4),
// so unlike (b) there is no cost reason to trust the cache here. Always
// resolve against a fresh, cache-bypassing read.
fresh, err := h.db.AIcallGet(ctx, c.ID, dbhandler.SkipCache(true))
if err != nil {
    // Can't determine the real turn; fail closed on the side that costs
    // nothing (a rejected tool call), not the side that could silently
    // eat a real answer.
    fillFailed(res, errors.Wrap(err, "could not verify calling turn"))
    return res
}

// New in rev 5 (review round 4, finding B2): a pipecatcallID of uuid.Nil
// means the caller didn't send one at all — either a pipecat-manager
// build that predates §5.4.3a's field (a rolling-deploy window), or a
// genuine bug upstream. Treat unknown as "assume this is the agent's
// real turn," never as "assume it's a listen turn": the failure mode of
// guessing wrong the safe way is one rejected notify_agent call (costs
// nothing); the failure mode of guessing wrong the unsafe way is exactly
// §5.4.5's data-corruption risk (a real Q&A tool row permanently
// mistagged as listen_internal and dropped from every future context).
if pipecatcallID == uuid.Nil || pipecatcallID == fresh.PipecatcallID {
    // This tool fired on the agent's own conversational turn (or we
    // can't prove otherwise) — reject rather than let RunLLM's
    // best-effort suppression silently eat the agent's real question.
    fillFailed(res, fmt.Errorf("notify_agent is only usable while proactively monitoring a call; you were asked a question — answer it directly instead"))
    return res
}
```

This makes `notify_agent` fail closed exactly when (a)'s guarantee is
weakest, independent of whether (a) actually suppressed the follow-up,
and fails closed (reject, not silently corrupt) on the unknown-id case
too.

Net effect: **a listen turn produces exactly zero persisted rows and zero
webhooks unless the LLM calls `notify_agent` from that turn — and a call
to `notify_agent` from any other context is rejected rather than silently
eating a real answer.**

#### 5.4.5 Keeping listen turns from evicting the agent's own Q&A context (new in rev 4, resolves review-round-3 B1)

**Review round 3 (finding B1) showed rev 1's original defect —
`getPipecatcallMessages`'s newest-100-row replay evicting the AIcall's
own system prompt — returns through a different door.** §5.4.2 stops the
*listen turn's own* context from using that replay path, but says nothing
about the rows a listen turn *writes*. Per §6, an Insight tool (including
`notify_agent`) can fire during a listen turn, and every tool call writes
two `Message` rows via `ToolHandle` (tool-call, tool-result —
`tool.go:47,88`). At the `AIcallListenMaxTurnsPerAIcall=60` cap, even one
tool call per turn is 120 rows — enough on its own to push the real
Q&A history (and the leading system-prompt rows) out of `getPipecatcallMessages`'s
top-100 window (`start.go:620-661`) the **next** time the agent asks the
Insight AI a real question. This is exactly rev 1's B2 defect, recurring
through a mechanism rev 2 did not create and rev 3 did not check for.

**Fix: tag every row a listen turn writes, exclude tagged rows from the
agent's own Q&A replay at the query level, and — new in rev 5 — guarantee
the leading system-prompt row(s) survive independent of that window
entirely.**

1. `message.Origin` (§5.6.2) gains a second value:
   ```go
   OriginListenInternal Origin = "listen_internal"  // tool-call/tool-result
   // rows written during a listen evaluation turn — never replayed into
   // any future context, listen or Q&A. The listen turn's own context
   // (§5.4.2) is assembled explicitly and never reads message rows for
   // the transcript/tool-call portion either, so this exclusion only
   // matters for getPipecatcallMessages (a real Q&A turn's context).
   ```
2. `ToolHandle` (extended per §5.4.3a to know `pipecatcallID`) tags the
   tool-call and tool-result rows it writes. **Scoping corrected in rev 5,
   review round 4 finding H2**: rev 4's rule (`pipecatcallID !=
   c.PipecatcallID` → tag) fires for *any* reference type, but
   `PipecatcallID` also rotates on every ordinary `Send()` for
   `conversation`/`task`/`none` AIcalls (`send.go:116-122`) — none of
   which have anything to do with listening. Tagging those would mislabel
   ordinary conversation history. The rule is now:
   ```go
   listenTurn := ac.ReferenceType == aicall.ReferenceTypeContactCase &&
       pipecatcallID != uuid.Nil &&        // §5.4.3a's B2 fix: unknown id
       pipecatcallID != c.PipecatcallID    // is never treated as a listen turn
   ```
   `Origin = OriginListenInternal` when `listenTurn`, `Origin = OriginNone`
   otherwise — unchanged from today for every AIcall this feature doesn't
   touch. The proactive `notify_agent` output row itself keeps `Origin =
   OriginProactive` (§5.6.2) — it is real conversational content the
   agent should see and the AI should remember, unlike the mechanical
   tool-call/result exchange that produced it.
3. **Mechanism and location corrected in rev 5, review round 4 finding
   B3.** `ApplyFields` does **not** live in `bin-ai-manager` — it is
   `bin-common-handler/pkg/databasehandler/main.go:61-110`, shared by
   every service in the monorepo (`bin-ai-manager`, `bin-call-manager`,
   `bin-message-manager`, …), and today it only ever builds
   `squirrel.Eq{...}` per field, with one hardcoded special case for the
   `"deleted"` key (`main.go:76-85`). Passing a `FieldOriginNot` key
   straight through would build `squirrel.Eq{"origin_not": ...}` — a SQL
   error on a nonexistent column, on every single `getPipecatcallMessages`
   call.

   **Decision: add a typed exclusion wrapper to `databasehandler`, not
   another hardcoded field-name special case** (the `"deleted"` special
   case is not a pattern to extend — see `bin-common-handler/CLAUDE.md`'s
   3-service admission rule, which already governs what this shared
   package may grow):
   ```go
   // bin-common-handler/pkg/databasehandler/main.go
   // NotEq wraps a filter value to signal "!=" instead of ApplyFields'
   // default "=". Any field, any service — generic, not string-keyed.
   type NotEq struct{ Value any }

   // inside ApplyFields' per-field switch:
   if ne, ok := value.(NotEq); ok {
       sb = sb.Where(squirrel.NotEq{string(field): ne.Value})
       continue
   }
   ```
   `getPipecatcallMessages` then passes
   `{FieldAIcallID: c.ID, FieldOrigin: databasehandler.NotEq{Value:
   message.OriginListenInternal}}` — no new `Field` constant needed
   (`FieldOrigin` already exists for `Origin`'s equality filter, §5.8),
   and no `FieldStruct`/`ConvertFilters` allowlist entry either (review
   round 4 finding M3: `FieldStruct` only gates what an *external RPC
   caller* may filter by, `pkg/listenhandler/v1_messages.go:45` —
   `getPipecatcallMessages` builds its filter map directly in Go code, so
   exposing an `origin_not` RPC filter would be pure unused surface area,
   not a requirement). Being a `bin-common-handler` change, it needs the
   full monorepo-wide verification workflow for every consumer, not just
   the services this design otherwise touches (§8).
4. **New in rev 5, review round 4 finding B4: excluding listen-internal
   rows narrows the eviction risk but does not eliminate it** — rev 4's
   own worked example (60 turns × 2 tagged rows = 120 rows evicting the
   system prompt) applies just as well to 60 *proactive* rows (never
   tagged, since they are real content) plus the agent's own Q&A tool
   rows (also never tagged). The fix has to guarantee the system prompt
   specifically, not merely reduce the row count competing for the
   window. `getPipecatcallMessages` (`start.go:620-661`) is restructured
   into two fetches instead of one:
   ```go
   // (1) The leading system row(s) — always small (1-3 rows: InsightSystemPrompt
   //     is not applicable here, this is the Q&A path; a normal AIcall has
   //     exactly the init_prompt + optional parameter-JSON system rows from
   //     startInitMessages, start.go:790-819) — fetched by role, oldest
   //     first, independent of the window below. These can never be evicted
   //     no matter how much conversation follows.
   systemRows, _ := h.messageHandler.List(ctx, 5, "", map[message.Field]any{
       message.FieldAIcallID: c.ID,
       message.FieldRole:     message.RoleSystem,
   })

   // (2) The newest 100 non-system, non-listen-internal rows — Q&A history
   //     and proactive notifications, exactly as before, minus §5.4.5's
   //     listen-internal exclusion.
   rest, _ := h.messageHandler.List(ctx, 100, "", map[message.Field]any{
       message.FieldAIcallID: c.ID,
       message.FieldRole:     databasehandler.NotEq{Value: message.RoleSystem},
       message.FieldOrigin:   databasehandler.NotEq{Value: message.OriginListenInternal},
   })
   // merge: systemRows (oldest-first) ++ rest (oldest-first, after the
   // existing reversal step, start.go:633-635)
   ```
   This guarantees the AI's own instructions are never lost regardless of
   conversation or notification volume — the thing rev 1's B2, and its
   rev-4 recurrence, were both actually about. The 100-row cap on *the
   rest* still means very old Q&A history can eventually be pushed out by
   a long call full of proactive notifications, same as any capped window
   already accepts for an unusually long conversation; that is a bounded,
   pre-existing trade-off, not the defect being fixed here.

**This also resolves review round 3's B2** (the "orphaned `tool`-role
message" finding, §5.6.3, §11 item 3), for the instances this feature
itself creates: a listen turn's tool-call/tool-result rows are now
permanently excluded from ever being replayed into any future context —
`getPipecatcallMessages` (step 4 above) and the listen turn's own
explicit assembly (§5.4.2, which never reads message rows for this
portion in the first place). They cannot become "orphaned" in a context
they are never placed into. **§11 item 3 is narrowed accordingly**: the
general defect (an agent-initiated tool call's own tool-call row already
gets filtered by `run.py:450`'s empty-content check today, independent of
listening) is still real and still worth confirming, but it is no longer
gated on or made worse by this feature shipping — restored to a
follow-up-ticket item rather than a rollout blocker.

**Known remaining gap, stated rather than silently left (review round 4
finding M7): `get_aicall_messages`.** This existing tool
(`toolHandleGetAIcallMessages`, `tool.go:761-763`) reads up to 1000
messages by `FieldAIcallID` alone and hands them to the LLM verbatim — it
does not go through `getPipecatcallMessages` and is therefore unaffected
by this section's `NotEq` filtering. A `contact_case` AIcall's `Origin =
OriginListenInternal` rows can reach the LLM through this tool. This is
lower severity than the context-eviction defect (it is a content-leak of
mechanical tool-call JSON into a Q&A answer, not a lost system prompt or
lost history), and is left as a follow-up rather than fixed in this
design — recorded in §11.

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

**OpenAPI surface — missing from rev 4, added in rev 5 (review round 4
finding H3).** The tool-name vocabulary is also public API surface, not
just an internal Go constant: `bin-openapi-manager/openapi/openapi.yaml`
carries a tool-name enum, and `paths/ais/main.yaml` /
`paths/ais/id.yaml` document the Insight-allowed tool list in prose —
both were updated for every prior Insight tool (`get_call_transcript`,
etc.) and must be updated here too, followed by the standard
`go generate ./...` regen in both `bin-openapi-manager` and
`bin-api-manager`. Listed explicitly in §9, not left implicit under
"`Origin` in the spec."

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
step. **Corrected in rev 3**: the worst case if a model calls it during an
ordinary Q&A turn is *not* "one extra harmless message" — §5.4.4(c) now
rejects that call outright, precisely because rev 2's "harmless" framing
was wrong (§5.4.4(c)'s own rationale). So the actual worst case is a
failed tool call the agent never sees (it just gets its real answer,
unaffected) — no external action, no spend, no data change, and no
silently-eaten answer either. The tool description still explicitly
tells the model not to do this; (c) is the backstop for when the
description is not followed.

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

#### 5.6.3 Message ordering — re-diagnosed in rev 3; surfaces a pre-existing production defect

Rev 2 claimed creating the proactive row *inside* `ToolHandle`'s step 2
would produce `assistant(tool_calls) → assistant(text) → tool(result)`,
which OpenAI rejects unless the `tool_calls` message is immediately
followed by its results — and "fixed" this by moving the proactive row to
*after* the tool-result row. **Review round 2 (finding F2) showed this
diagnosis was wrong, and the fix was consequently a no-op.** Re-verified
directly against the code for rev 3:

- `ToolHandle` creates the tool-call row with **empty content**:
  `h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, c.ID, c.ActiveflowID,
  message.DirectionIncoming, message.RoleAssistant, "", []message.ToolCall{*tool}, ...)`
  (`tool.go:47` — the `""` is the `content` argument).
- `bin-pipecat-manager/scripts/pipecat/run.py:450` builds the replayed
  context as `valid_messages = [m for m in messages if m.get("role") and
  m.get("content")]` — an empty-string `content` is falsy in Python, so
  **the `assistant(tool_calls)` row is filtered out of the context before
  it ever reaches `LLMContext`**, regardless of which row was created
  first or second on the DB side. The same filter runs on the
  team-conversation path (`run.py:637`). Rev 2's reordering therefore
  changes nothing about what pipecat-manager actually sends to the LLM —
  **both orderings produce byte-identical payloads.**
- What *does* reach the LLM, because it has real content, is the
  `role=tool` result row (`toolCreateResultMessage`, `tool.go:88`) — with
  **no preceding `tool_calls` entry in the same context**, because that
  entry was just filtered out. This is the actual OpenAI-API-incompatible
  shape (a `tool`-role message must reference a `tool_calls` entry present
  in the same request), just the reverse of what rev 2 diagnosed.

**This is not a defect this feature introduces.** Every existing Insight
tool (`get_call_transcript`, `get_contact_profile`, …) follows the exact
same `ToolHandle` path today, so **any multi-turn `contact_case` Q&A
session where the agent's first question triggers a tool call, and the
agent then asks a follow-up question, already replays this orphaned
`tool`-role row into the follow-up's context** — independent of listen,
independent of `notify_agent`. This design does not change that shape for
`notify_agent`'s own tool-call/result pair (they follow the identical
`ToolHandle` sequence as every other tool), and does not attempt to fix
it here: fixing `run.py:450`'s content-truthiness filter (or giving the
tool-call row non-empty placeholder content) is a change to shared
`ToolHandle`/pipecat-manager behaviour affecting every AI tool call
platform-wide, not something scoped to Insight listening.

**Action item, separate from this design (§11 item 3, escalated in rev
3 from "follow-up" to "recommend filing and investigating immediately"):**
confirm empirically whether this is actually causing production failures
today (it may be masked if OpenAI's Chat Completions API is in practice
lenient about an orphaned `tool` message, or if few real sessions exercise
a tool call followed by a genuine follow-up question on the same AIcall —
that needs checking, not assuming). If confirmed, it is a platform-wide
correctness bug predating and independent of this feature and should be
triaged on its own ticket, at whatever urgency the confirmation warrants.

The proactive `Origin=proactive` row itself needs no special ordering
relative to the tool-call/result pair — it is a wholly separate `Message`
row, written by `toolHandleNotifyAgent` (§5.5) once it validates the
argument, with no OpenAI-payload constraint linking it to the tool-call
sequence above.

#### 5.6.4 What actually surfaces per `notify_agent` call (new in rev 3, resolves review-round-2 F7)

Rev 2 asserted a single new row per notification. **That is wrong.**
`ToolHandle` (`aicallhandler/tool.go:24-100`) writes, for **every** tool
invocation including `notify_agent`:

1. `role=assistant`, `content=""`, carrying `ToolCalls` (`tool.go:47`),
2. `role=tool`, the raw JSON result (`tool.go:88` →
   `toolCreateResultMessage`),
3. — and only for `notify_agent` — the new `role=assistant`,
   `Origin=proactive` row (§5.6.2).

`messageHandler.Create` publishes `aimessage_created` to the customer's
(tenant's) configured webhook **unconditionally** for every row
(`pkg/messagehandler/db.go:81` → `notifyhandler/publish.go:24-26`), and
neither panel special-cases roles when rendering — square-admin styles on
`msg.role === 'user'` and treats everything else uniformly
(`CaseInsightAssistantPanel.js:44`); square-talk does the same
(`.jsx:62`). So **one proactive notification is, today, three
webhook-published, panel-rendered rows**: an empty bubble, a raw-JSON
blob, and the intended note.

This is **not new to this feature** — every existing Insight tool call
already produces rows 1 and 2 and already surfaces them the same way; it
predates this design (same root cause as §5.6.3's finding: `ToolHandle`'s
two-row-per-tool-call shape). This design makes it materially more
visible because listening is the first thing that can trigger a tool call
*without the agent having asked for it*, so the noise now appears
unprompted mid-call, not as a byproduct of a question the agent typed.

**Mitigation shipped with this design (frontend, §5.10.1): filter, don't
suppress.** Both `CaseInsightAssistantPanel` components stop rendering a
message if `msg.role === 'tool'` or (`msg.role === 'assistant' &&
!msg.content && msg.tool_calls?.length`) — **field names corrected in
rev 4, review round 3 finding H3**: `WebhookMessage`'s wire field is
`tool_calls` (`models/message/webhook.go:23`,
`json:"tool_calls,omitempty"`), snake_case like every other field the
panels already read (`msg.role`, `msg.content`, `msg.tm_create` —
`CaseInsightAssistantPanel.js:44,46,47`); rev 3's `toolCalls` (camelCase)
would never have matched and the filter would silently never have fired.
This is a generically useful filter, not scoped to listening, since it
also cleans up every existing Insight Q&A tool call's panel noise today
(scope note: it does **not** hide the `role='system'` rows
`startInitMessages` writes at AIcall creation, `start.go:812-819` — those
predate this feature and are a separate, out-of-scope cleanup). It is a
client-side render filter only; it does **not** touch the webhook
delivery, `Create`, or `PublishWebhookEvent` — those keep firing for rows
1 and 2 exactly as before, so a tenant's own webhook-consuming automation
still sees every tool-call row it does today. Suppressing *those*
webhooks is a separate, larger decision (whether tenants rely on
tool-call webhooks for their own automation is unknown) and is recorded
as a follow-up (§11 item 6), not attempted here.

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

**STT stop: not ai-manager's job on this path — reasoning corrected in
rev 3.** `bin-transcribe-manager/pkg/transcribehandler/event.go:51-81`
(`EventCMCallHangup`) does list every non-deleted transcribe with
`reference_id == call.ID`, owner-agnostic, and call `Stop` on each — but
**review round 2 (finding F8) showed the DB-status-flip half of that path
is pod-local, not the platform-wide guarantee rev 2 stated.** `Stop` →
`stopLive` → `streamingHandler.Stop` reads the **per-pod in-memory**
`mapStreaming`; on whichever pod happens to consume the shared hangup
subscribe queue (`cmd/transcribe-manager/main.go:201`) that is *not* the
session's owning pod, the in-memory lookup misses,
`isSafeToConsiderStopped` treats that as "already gone," the physical
streaming stop is skipped, and `UpdateStatus(StatusDone)` is written
anyway — regardless of whether the physical STT stream actually stopped.
The safety comment at `stop.go:155-166` that justifies this branch
explicitly assumes routed-to-owning-pod delivery (*"the RPC can only ever
reach the pod identified by the transcribe's `HostID`"*), which is true of
the direct `TranscribeV1TranscribeStop`/health-check RPCs but **not** of
this in-process hangup-event path.

The *actual* backstop this design relies on is simpler and does not
depend on that DB write: hanging up the call closes Asterisk's external-
media WebSocket connection that was feeding the streaming session's audio
(`bin-transcribe-manager/CLAUDE.md`'s own description of the transport —
"Go dials out to Asterisk's `chan_websocket` endpoint... raw 8kHz slin
binary frames"). Once that socket closes, the STT read loop ends and
billing for that stream stops, independent of whether `ai_transcribes`'s
row shows `progressing` or `done` on a non-owning pod. **This is a
pre-existing property of every transcribe session on the platform today,
not something this design introduces or changes** — listening rides the
same guarantee every flow-driven and summary-driven transcribe already
relies on. ai-manager's hangup path therefore still issues no stop RPC (a
persisted `HostID` could address a queue that no longer exists after a
transcribe-manager restart, per `bin-transcribe-manager/CLAUDE.md`), but
the justification is "the audio transport itself terminates," not "the DB
row is guaranteed accurate."

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
session's audio transport is guaranteed to end when the call itself ends
(§5.7.1's corrected reasoning), which is at most one call-duration away.
The failure mode is a slightly-longer-than-necessary STT session, never a
permanently orphaned one.

**Second-order consequence of §5.7.1's correction, noted here rather than
hidden:** the `tr.Status == tmtranscribe.StatusProgressing` gate above can
itself already read `done` on a call that hung up moments earlier but
whose STT-stop RPC never reached the owning pod (§5.7.1) — in which case
this branch is simply skipped, which is the correct outcome (nothing left
to stop).

#### 5.7.3 Clearing state (all paths)

`clearListenState(ctx, aicallID)` — **step order corrected in rev 4,
review round 3 finding M1**: rev 3's numbered steps cleared the AIcall's
`listen_transcribe_id` metadata *before* using that same value to `SREM`
the resolver set, which cannot work (the value needed for step 2 no
longer exists once step 1 runs). Actual order:
1. Read `transcribeID := c.Metadata[listen_transcribe_id]` from the
   AIcall already in hand (no extra fetch — `clearListenState`'s callers
   already hold `c`).
2. Redis: `SREM ai:listen:transcribe:<transcribeID> <aicallID>` (§5.2.4's
   set fix — removes only this AIcall's membership; Redis deletes the key
   itself once the set empties, so a shared transcribe stays resolvable
   for whichever AIcall(s) are still listening to it), plus
   `DEL ai:listen:pending:<aicallID>`, `ai:listen:window:<aicallID>`,
   `ai:listen:lock:<aicallID>`, `ai:listen:turns:<aicallID>` (these four
   are per-AIcall, never shared, so a plain `DEL` is correct for them).
3. `AIcallUpdate` → `listen_call_id = uuid.Nil`, remove both metadata keys
   (one write) — last, since nothing downstream needs the old value once
   step 2 has consumed it.

Removing this AIcall's set membership before clearing the DB metadata is
what guarantees a stale `(transcribe_id, aicall_id)` pairing can never be
matched again by §5.3.

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
  (line 44); it gains:
  - a render filter (resolves review-round-2 F7, field names corrected in
    rev 4 per review-round-3 H3, detailed in §5.6.4): skip rendering any
    message with `msg.role === 'tool'`, or `msg.role === 'assistant' &&
    !msg.content && msg.tool_calls?.length`. This is a client-side filter
    only — it hides the two noise rows every tool call (not just
    `notify_agent`) already produces, without touching what the tenant's
    own webhook consumer receives (§5.6.4, §11 item 6);
  - a third branch for `msg.origin === 'proactive'` (distinct surface + a
    `Sparkles`/bell affordance + an accessible label such as "Proactive
    insight"), so a notification is never mistaken for an answer.
- **square-talk** (`src/features/cases/CaseInsightAssistantPanel.jsx`):
  unchanged transport (2s poll); identical render-filter and
  `origin`-driven treatment at line 62.
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
| **Transcribe sessions** per call — for listen alone | 1 — any progressing `IDAIManagerListen` session on that call is reused (§5.2.2) |
| **Transcribe sessions** per call — listen + a concurrent `ai_summary` (rare) | up to 2, since rev 4 (§5.2.1) deliberately no longer shares ownership with `ai_summary`'s `IDAIManager` session |
| **STT streams** per call (corrected in rev 3, review-round-2 F11) | 2, not 1 — `DirectionBoth` expands to two independent streamings, one per direction (`transcribehandler/start.go:~216-219`: `directions := []transcript.Direction{DirectionIn, DirectionOut}`), each its own external-media leg and provider stream. One shared *transcribe session* still means one shared *billing/lifecycle record* (§5.2.2's reuse rule dedupes at that level), but the underlying STT cost is two streams, not one |
| LLM turns per AIcall per minute | `60 / AIcallListenEvaluateIntervalSeconds` = 3 at the default |
| LLM turns per AIcall, total | `AIcallListenMaxTurnsPerAIcall` = 60 (hard stop, then listening ends) |
| Tokens per turn | constant-shaped: 3 system messages (`InsightSystemPrompt` + prompt snapshot + `ListenTurnSystemPrompt`, §5.4.2) + ≤10 Q&A messages + ≤40 transcript lines |
| Concurrent listen sessions | number of open Case panels whose Case call is live |

**Worst case per listened call at defaults:** 3 small LLM turns/min,
capped at 60 turns (~20 min of continuous speech), one shared transcribe
session (two STT streams). Contrast with rev 1, which was one
*unbounded-context* LLM call per spoken sentence.

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

**Naming corrected in rev 3 (review-round-2 F12).** Existing ai-manager
metrics are declared with `Namespace: metricsNamespace` (`"ai_manager"`)
plus a bare `Name:` (e.g. `aicall_create_total`,
`aicall_tool_execute_total`, `message_create_total`) — the namespace is
prepended by the Prometheus client library, not typed into the name
string. Rev 2's names already included `ai_manager_` as a literal prefix,
which would render as `ai_manager_ai_manager_aicall_listen_start_total`.
The table below gives the `Name:` value only; the namespace is implicit,
exactly like every existing ai-manager metric.

| Metric (full name = `ai_manager_` + this) | Labels | Meaning |
|---|---|---|
| `aicall_listen_start_total` | `result` = started / reused / skipped_not_listenable / failed | §5.1–5.2 outcomes |
| `aicall_listen_segment_total` | `result` = buffered / dropped_deleted / dropped_unknown | §5.3 intake |
| `aicall_listen_turn_total` | `result` = ran / skipped_locked / skipped_empty / skipped_cap / failed | §5.4 turns |
| `aicall_listen_notify_total` | — | proactive messages actually delivered |
| `aicall_foreign_pipecatcall_dropped_total` | `handler` | §5.4.4(b) guard firings (also covers pre-existing stale contact_case replies) |
| `aicall_listen_stop_failed_total` | — | §5.7.2 stop RPC failures falling back to the call-hangup-ends-the-transport backstop (§5.7.1) |

`aicall_listen_turn_total{result="skipped_locked"}` is the
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
| `RunListenTurn` fails after popping `pending` (pipecatcall start error, pod loss) (**new in rev 3**) | The popped lines are gone — `LPOP` already removed them, and only the ≤40-line `window` retains a copy for the *next* turn's context. Accepted, bounded data loss: at most one debounce interval's worth of transcript is skipped from evaluation, never from the call itself (nothing about the actual call or its recording is affected) |
| LLM emits text instead of calling `notify_agent` | Dropped by the pipecatcall-identity guard on all four pipecat message handlers (§5.4.4(b)); metered. Nothing persisted, no webhook |
| LLM calls `notify_agent` with empty/whitespace/oversized `message` | Rejected in `parseNotifyAgentMessage`; `fillFailed` (same style as the other tools in `tool_insight.go`); tool-result row records the failure; **no** proactive message row |
| LLM calls `notify_agent` during a normal Q&A turn (**corrected in rev 3, was "harmless" in rev 2**) | **Rejected outright** by §5.4.4(c)'s pipecatcall-identity check in `toolHandleNotifyAgent` — the call fails, the agent's real answer proceeds unaffected. Rev 2 called this harmless; it was not (§5.4.4(c)) |
| LLM calls other Insight tools during a listen turn | Allowed (no per-turn tool restriction — §3). Adds tool-call/result rows, webhook-published as always; hidden from the panel by the render filter (§5.6.4, §5.10.1) but still delivered to the tenant's webhook consumer. Discouraged by `ListenTurnSystemPrompt` |
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
   drop; empty-set drop (asserting **no** DB call); buffered-but-locked
   (no turn); buffered-and-unlocked (turn runs); **new in rev 3, pins F1's
   fix**: two AIcalls in the same resolver set both get the segment
   buffered independently, and clearing one (`SREM`) leaves the other's
   membership and buffering intact.
4. `RunListenTurn` — empty pending → skip; turn cap → stop listening;
   context assembly golden test asserting exact message count and order —
   `InsightSystemPrompt` first, the prompt snapshot second, then
   `ListenTurnSystemPrompt` (this is the direct regression test for both
   the rev-1 context-eviction defect and rev-2's missing-guardrails defect,
   F6); asserts `getPipecatcallMessages` is **not** called and
   `c.PipecatcallID` is **not** written.
5. `messagehandler.isForeignPipecatcall` — for each of the four handlers:
   matching id persists/publishes, mismatched id drops, and (for
   `EventPMMessageBotLLMIntermediate`) **no webhook is published** on
   mismatch.
6. `toolHandleNotifyAgent` — success writes exactly one proactive row with
   `Role=assistant` and `Origin=proactive`; called on `c.PipecatcallID`
   itself (§5.4.4(c)) is rejected with no row written; empty/whitespace/
   oversized argument writes no proactive row.
7. Cleanup — `stopListenByCallID` clears **all** matching AIcalls (two-row
   case); `ProcessTerminate` stops only when `owns=true`; stop-RPC failure
   is non-fatal and metered; `clearListenState` `SREM`s only its own
   membership, not the whole resolver set.
8. **New in rev 3, pins F4:** the pipecatcall-identity guard's
   cache-bypass re-read — a mismatch against the cached `PipecatcallID`
   that resolves to a match on a DB-authoritative re-read persists the
   message; a mismatch that still disagrees on re-read drops it.
9. **New in rev 4, pins B3:** listen and a concurrent `ai_summary` on the
   same call each get their own transcribe session (`IDAIManagerListen`
   vs. `IDAIManager`); `TranscribeV1TranscribeList` scoped to
   `IDAIManagerListen` never returns a summary's session and vice versa;
   `summaryhandler.startReferenceTypeCall`'s own tests are unaffected by
   listen having run on the same call (a direct regression test for the
   rev-3 defect this replaces).
10. **New in rev 4, pins B4:** `ToolHandle` receives and uses
    `pipecatcallID` — a call arriving with `pipecatcallID == c.PipecatcallID`
    is treated as the agent's real Q&A turn; a call with any other id is
    treated as a listen turn (tags rows `Origin=listen_internal` per
    §5.4.5, and is what §5.4.4(c)'s `notify_agent` guard checks).
11. `getPipecatcallMessages`'s two-fetch context assembly (§5.4.5,
    revised in rev 5): a golden test seeding 150+ listen-internal rows
    interleaved with 10 real Q&A rows and the leading system-prompt rows
    asserts (a) the system-prompt row(s) are always present regardless of
    how many listen-internal or proactive rows follow, and (b) the
    "rest" fetch excludes every listen-internal row via the `NotEq`
    filter. Additional cases, new in rev 5:
    - `Origin` tagging only applies for `ReferenceTypeContactCase`; a
      `conversation`-type AIcall's ordinary `Send()`-driven pipecatcall
      rotation never tags anything `listen_internal` (pins review round 4
      H2).
    - a `pipecatcallID == uuid.Nil` tool call is tagged `OriginNone` (a
      real turn), never `listen_internal` (pins review round 4 B2).
    - `toolHandleNotifyAgent`'s reject logic: called with the AIcall's
      true current `PipecatcallID` → rejected; called with any other id
      (including after a simulated stale-cache read that used to
      disagree) → allowed; `AIcallGet` returning an error → rejected,
      never allowed (pins review round 4 B1 — the corrected single-path
      logic, replacing rev 4's inverted two-branch version).

**`bin-ai-manager` model/golden:**

12. `models/ai/allowed_tools_test.go` — `notify_agent` passes via
    `knownSanctionedWrite`; a hypothetical unlisted write tool still fails;
    `TestValidateToolNames_WriteToolNeverAllowedForInsight` still passes
    unchanged.
13. `pkg/subscribehandler/binding_golden_test.go` — updated to 12 patterns
    with the new one appended last.
14. `models/aicall/field_test.go`, `filters_test.go`,
    `models/message/field_test.go`, `webhook_test.go` — new fields,
    including `Origin`'s two values.

**Boundary:**

15. `requesthandler` mock expectations pinning the exact
    `TranscribeV1TranscribeStart` argument list including `provider` and
    `onEndFlowID` (§5.2.2), and `TranscribeV1TranscribeStop(ctx, hostID,
    transcribeID)` argument order (§5.7.2) — both were wrong in rev 1, so
    both get an argument-shape test. **New in rev 4:**
    `AIV1AIcallToolExecute`'s new `pipecatcallID` argument (§5.4.3a), both
    at the `bin-common-handler` client and the `bin-pipecat-manager`
    call-site.

**Deferred until §5.9's empirical check lands:**

16. A pinned golden-transcript test for the `in`/`out` → `[CUSTOMER]`/
    `[AGENT]` mapping. This is exactly the silent-wrong-attribution class
    that deserves a pinned test rather than a happy-path assertion.

**Frontend (`monorepo-javascript`):**

17. Both `CaseInsightAssistantPanel` suites: renders an
    `origin: 'proactive'` message with its distinct treatment and
    accessible label; renders a normal assistant message unchanged;
    renders a message with no `origin` field (backward compatibility with
    every existing row) unchanged; the tool-call/tool-result render
    filter (§5.6.4) hides `role='tool'` and empty-content
    `role='assistant'` rows (**field names corrected in rev 4** — a
    regression test asserting the filter actually matches the real
    `tool_calls`/`content` wire field names, not the rev-3 typo).

---

## 8. Rollout

1. Backend PR in `monorepo` — code + migrations (drafted, **not**
   applied), service docs, RST + rebuilt HTML. **Corrected in rev 4**:
   `bin-common-handler` is touched unconditionally now (§5.4.3a's
   `pipecatcallID` parameter, and §5.4.5's `databasehandler.NotEq`), not
   "if touched." Full verification workflow in `bin-ai-manager`,
   `bin-common-handler` (monorepo-wide, since `databasehandler` is shared
   by every service — §5.4.5 step 3), `bin-pipecat-manager` (§5.4.3a),
   `bin-contact-manager`, `bin-customer-manager` (§5.2.1, pending §11 item
   5), `bin-api-manager`, `bin-openapi-manager`.
2. **Migration-before-deploy ordering — new in rev 5, review round 4
   finding H5.** `messageHandler.List`/`MessageList` builds its `SELECT`
   column list by reflecting the `Message` struct
   (`commondatabasehandler.GetDBFields`, `dbhandler/message.go:210`);
   `AIcallList` does the same for `AIcall`. The instant the `Origin` field
   (or `ListenCallID`) exists in the Go struct, every message/AIcall
   query — including ones this feature never touches — selects that
   column. **A code deploy landing before its migration is a hard outage
   for every AIcall/message read in `bin-ai-manager`** (`Unknown column`),
   not a soft degradation. Human applies the Alembic migrations *before*
   the code deploy that references the new columns reaches any pod — not
   merely "before implementation sign-off" as §9 lists them, but as an
   explicit, ordered deploy-gate step here.
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
  (both values: `proactive` and `listen_internal`, §5.4.5)
- `models/aicall/{main,field,filters}.go` + tests — `ListenCallID`, metadata keys
- `pkg/aicallhandler/main.go` — **the `AIcallHandler` interface itself**
  (`ToolHandle`'s new `pipecatcallID` parameter, §5.4.3a — corrected in
  rev 5: this is an interface-level change, not just the implementation
  file below), plus `ListenTurnSystemPrompt`, config-derived constants
- `pkg/aicallhandler/start.go` — `Start` hook, `ensureListen`,
  `startListenPipecatcall`, `getPipecatcallMessages`'s two-fetch rewrite
  (leading system rows + `databasehandler.NotEq`-filtered rest, §5.4.5)
- `pkg/aicallhandler/listen.go` *(new)* — `EventTMTranscriptCreated`, `RunListenTurn`, context assembly, `stopListenByCallID`, `clearListenState`
- `pkg/aicallhandler/tool.go` — `ToolHandle`'s implementation of the new
  `pipecatcallID` parameter, `mapFunctions` entry for `notify_agent`
  (unchanged signature, §5.4.3a step 4), `Origin` tagging on tool-call/
  tool-result rows (§5.4.5)
- `pkg/aicallhandler/tool_insight.go` — `toolHandleNotifyAgent` (fresh
  cache-bypass read + reject logic, §5.4.4(c)), `parseNotifyAgentMessage`
- `pkg/aicallhandler/event.go` — `EventCMCallHangup` second lookup
- `pkg/aicallhandler/process.go` — terminate-path stop
- `pkg/listenhandler/v1_aicalls.go`, `pkg/listenhandler/models/request/aicalls.go`
  — **new in rev 5**: `pipecatcall_id` on `V1DataAIcallsIDToolExecutePost`
  and its handler (§5.4.3a step 3; missing from rev 3/4's file list)
- `pkg/messagehandler/main.go`, `event.go` — `isForeignPipecatcall`,
  applied to the two handlers that can actually fire from a listen turn
  (§5.4.4(b)); the cache-bypass re-read now goes through `AIV1AIcallGet`'s
  RPC client with a new optional argument, not a direct `dbhandler` call
  (§5.4.4(b), path corrected in rev 5)
- `pkg/toolhandler/definitions.go` — `notify_agent` (`RunLLM:false`)
- `pkg/cachehandler/{main,handler}.go` **or a new `pkg/listencachehandler`
  package** — see the scope note below
- `pkg/subscribehandler/{main,transcribemanager,binding_golden_test}.go`
- `internal/config/main.go` — seven flags
- `docs/{domain,architecture,operations}.md`

`bin-contact-manager` — `models/kase/kase.go` (`ReferenceTypeCall`)

`bin-customer-manager` — **new in rev 4**: `models/customer/customer.go`,
new `IDAIManagerListen` system customer constant (§5.2.1), pending §11
item 5's confirmation of whether it needs a backing row

`bin-pipecat-manager` — **new in rev 4 (§5.4.3a)**:
`pkg/pipecatcallhandler/runner.go` forwards `pc.ID` on `tool_execute`

`bin-common-handler` — `pkg/requesthandler`'s `AIV1AIcallToolExecute`
signature gains a `pipecatcallID` parameter (§5.4.3a) and its
`AIV1AIcallGet` client gains an optional cache-bypass argument (§5.4.4(b),
corrected in rev 5); mock regen for both. **New in rev 5 (§5.4.5, review
round 4 finding B3)**: `pkg/databasehandler/main.go` gains the `NotEq`
wrapper type and its handling in `ApplyFields` — this is the
monorepo-wide-consumed change, requiring the full verification workflow
across every service that calls `ApplyFields`, not just the ones this
design otherwise touches (§8).

`bin-dbscheme-manager` — three generated migrations (two from rev 1–3,
plus `bin-customer-manager`'s system-customer seed if §11 item 5
concludes one is needed)

`bin-openapi-manager`, `bin-api-manager` — `Origin` in the spec, both
values (**corrected in rev 5, review round 4 finding M8**: rev 4 said to
document only `proactive` and leave `listen_internal` undocumented, but
`listen_internal` rows are created through the same `ToolHandle` →
`messageHandler.Create` → `ConvertWebhookMessage` path as any other
message, so the value genuinely reaches a tenant's webhook payload —
leaving it undocumented while it's on the wire is worse than documenting
it plainly as "internal bookkeeping; do not depend on this value's
presence or meaning" in the RST `origin` field description, §5.10.2),
regen, RST + build. `ToolHandle`'s new parameter and
`AIV1AIcallToolExecute`'s new argument are internal RPC surface, not
public API — no OpenAPI change from those.

`monorepo-javascript` — both `CaseInsightAssistantPanel` files + tests

**Scope note on `cachehandler` (review-round-2 F10), acknowledged rather
than resolved here:** today the package is a pure JSON entity-snapshot
cache — two primitives, `getSerialize`/`setSerialize`
(`handler.go:17-41`), fixed 24h TTL, no raw Redis data-structure or lock
primitive of any kind. §5.3.3/§5.3.4/§5.4.1 add `SADD`/`SREM`/`SMEMBERS`,
`RPUSH`/`LTRIM`/`EXPIRE`, `LPOP count`, `SET NX EX`, and `INCR` — a
second, structurally different responsibility (ephemeral buffers +
distributed rate limiting) that does not fit the existing package's
shape. This design does not resolve which is right (extend
`cachehandler` with a new file, or give listen state its own small
package sharing the same Redis client) — it belongs with whoever
implements this, informed by how the team wants shared-infra Redis usage
organised. Flagged, not decided, in §11 item 7.

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

### 10.1 Review-response matrix (round 2 → rev 3)

Round 2 (an independent review, deliberately skeptical of rev 2's own
claims, run against the actual code rather than the review-round-1
matrix) confirmed most of rev 2's fixes hold, and found 3 BLOCKING + 4
HIGH + 4 MEDIUM + 2 LOW new issues, all introduced by rev 2's own new
mechanisms.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| F1 | Single-valued Redis resolver key contradicts the N-AIcalls-share-one-transcribe design; second listener silently steals the first's mapping, either's cleanup deletes the shared key | Resolver key changed to a Redis set (`SADD`/`SREM`/`SMEMBERS`); every listening AIcall keeps its own membership | §5.2.4, §5.3.2, §5.7.3 |
| F2 | The "tool_calls ordering" defect and fix in rev 2's §5.6.3 were mis-diagnosed; the reordering is a no-op because the tool-call row is filtered out of context by empty `content`, not by ordering | Re-diagnosed against `run.py:450` directly. The real (and pre-existing, feature-independent) defect is an orphaned `tool`-role row with no preceding `tool_calls` entry. Documented plainly, not silently fixed here, escalated to an immediate-verification item rather than a deferred ticket | §5.6.3, §11 item 2 |
| F3 | `RunLLM:false` has undocumented error-path, override, and Q&A-turn-collision holes; the "every other tool uses RunLLM:true" supporting claim is false | Rewrote as a three-layer defense: (a) `RunLLM:false` as a best-effort hint with its real caveats stated, (b) the pipecatcall-identity guard, (c) a new explicit reject-if-invoked-from-the-agent's-own-turn check in `toolHandleNotifyAgent`. Corrected the false claim (all six *Insight* tools use `RunLLM:true`, not "every other tool") | §5.4.4(a)(c) |
| F4 | The pipecatcall-identity guard can false-positive against a genuine reply if a post-`Send` cache write transiently fails, and `contact_case` has no termination-triggered backstop the way `conversation` does | Guard now re-reads the AIcall bypassing cache before dropping on a mismatch; the missing backstop is stated explicitly rather than silently assumed covered | §5.4.4(b) |
| F5 | Listen-vs-`ai_summary` transcribe collision only analysed in one direction; the reverse (listen starts first) makes a later summary attempt fail with `TRANSCRIBE_ALREADY_PROGRESSING` | `summaryhandler.startReferenceTypeCall` made reuse-tolerant, symmetric with listen's own rule, via a shared `ensureIDAIManagerTranscribe` helper | §5.2.2a |
| F6 | Listen-turn context assembly omits `InsightSystemPrompt` (the platform's own hallucination/tool-leakage guardrails for Insight AIs) | Added as message #1, ahead of the customer's own prompt snapshot | §5.4.2 |
| F7 | A proactive notification is claimed to be "one new row"; it is actually three (tool-call row, tool-result row, proactive row), all webhook-published and panel-rendered, contradicting §4's "invisible unless notified" claim | Stated plainly as a pre-existing, feature-independent `ToolHandle` shape made more visible by listening; a frontend render filter (not a webhook suppression) is the shipped mitigation, with the larger webhook-level fix recorded as a separate follow-up | §5.6.4, §5.10.1, §11 item 6 |
| F8 | "transcribe-manager's `EventCMCallHangup` already stops every session, owner-agnostic" overstates a pod-local mechanism as a platform-wide guarantee | Reasoning corrected: the real backstop is that hanging up the call closes the Asterisk WebSocket feeding the STT stream, independent of whether the DB status write on a non-owning pod is accurate. Conclusion (no stop RPC needed here) unchanged; justification rewritten to match what the cited code actually establishes | §5.7.1, §5.7.2 |
| F9 | Redis `SET NX EX` debounce is a rate limiter, not a lock; a turn that fails after popping `pending` loses those lines beyond the 40-line window; `LPOP count` needs Redis ≥6.2, unverified against the deployed server | Both risks stated explicitly as accepted/bounded rather than left implicit; Redis-version confirmation added as an implementation-time open item | §6 (new row), §11 item 7 |
| F10 | The `cachehandler` change (six Redis primitives beyond its current two) is a second, structurally different responsibility, understated as "+ mock regen" | Acknowledged explicitly as an open scope question (extend `cachehandler` vs. a new package) rather than a decided detail | §9 scope note, §11 item 7 |
| F11 | "STT sessions per call: 1" understates cost — `DirectionBoth` is two independent STT streams, not one | Split into "transcribe sessions" (1, still deduped) vs. "STT streams" (2) in the bounds table | §5.11 |
| F12 | Metric names double the `ai_manager` namespace prefix | Corrected to bare `Name:` values, namespace implicit per existing convention | §5.13 |
| F13 | §5.3.1's "three coupled places" was read as undercounting the golden-test edit count | Table already enumerated all four edits within `binding_golden_test.go` (expected slice, length check, message string, doc comment) grouped under one of the three *places*; left as-is, no defect found on re-check |  — |

### 10.2 Review-response matrix (round 3 → rev 4)

Round 3 (another independent, skeptical-by-instruction review, run against
the code again rather than trusting rev 3's own §10/§10.1) confirmed most
of rev 3's fixes hold (F1, F6, F8, F11, F12, and the §5.6.3
re-diagnosis all checked out against the code directly), and found 4
BLOCKING + 3 HIGH + 5 MEDIUM new issues — every one of them introduced by
rev 3's own new mechanisms, not by anything earlier.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| B1 | Rev 1's context-eviction defect (system prompt/Q&A history pushed out of `getPipecatcallMessages`'s 100-row window) resurfaces through the tool-call/tool-result rows a listen turn's own tool use writes | New `Origin=listen_internal` tag on listen-turn tool rows, excluded from `getPipecatcallMessages`'s query via a new `FieldOriginNot` filter — excluded at the SQL layer, not by fetching more and filtering in Go | §5.4.5 |
| B2 | §11 item 2 (orphaned `tool`-role message) deferred to a follow-up ticket when this feature's own happy path (an unprompted `notify_agent` call) actively creates the condition | §5.4.5's fix also removes listen's own tool rows from ever being replayed anywhere, so this feature can no longer produce an instance of the defect; the general (agent-initiated) case is restored to a follow-up item since it is no longer made worse by this feature | §5.4.5, §11 item 3 |
| B3 | §5.2.2a's "make `summaryhandler` reuse-tolerant" fix for the listen/`ai_summary` transcribe collision breaks `summaryhandler`'s own read path (`contentGetTranscripts`'s unpinned `size=1` list) and lifecycle assumptions (listen's `owns=true` stop can cut off a summary that later attached to the same session) | §5.2.2a deleted. Listen gets its own system customer id (`IDAIManagerListen`, distinct from `ai_summary`'s `IDAIManager`), so the two features' transcribe sessions are provably independent — no reuse, no hand-off, no shared lifecycle, `summaryhandler` needs no change at all | §5.2.1, §5.2.2 |
| B4 | §5.4.4(c) (reject `notify_agent` when called from the agent's real Q&A turn) assumed `ToolHandle` already receives the invoking pipecatcall id; it does not — `runner.go:457` never forwards `pc.ID` to ai-manager | New §5.4.3a: pipecatcall id threaded through `runner.go` → `bin-common-handler`'s RPC → the wire DTO → `ToolHandle`'s signature, as a real, scoped cross-service change reflected in §8/§9. §5.4.4(c) and §5.4.5's `Origin` tagging both consume this one signal | §5.4.3a |
| H1 | The pipecatcall-identity guard's cache-bypass re-read, applied to all four original handlers, would put an uncached DB read on the highest-volume one (`EventPMMessageBotLLMIntermediate`, fired per token chunk) | Cache-bypass re-read scoped to `EventPMMessageBotLLM` only (the one that persists); `EventPMMessageBotLLMIntermediate` drops on a plain cached mismatch — its only cost on a false positive is one skipped, non-user-visible intermediate webhook | §5.4.4(b) |
| H2 | Two of the four guarded handlers (`EventPMMessageUserLLM`, `EventPMMessageUserTranscription`) can never fire from a listen turn (`STTTypeNone`), so guarding them only adds a per-utterance AIcall lookup to a platform-wide hot path for a condition that cannot occur | Guard narrowed to the two handlers that can actually fire from a listen turn; the other two left unchanged, exactly as before this feature | §5.4.4(b) |
| H3 | Frontend render-filter condition used `toolCalls` (camelCase); the real wire field is `tool_calls` (snake_case, matching every other field the panels already read) — the filter as written would never have fired | Corrected to `msg.tool_calls`, `msg.content`, `msg.role` throughout §5.6.4/§5.10.1 | §5.6.4, §5.10.1 |
| M1 | §5.7.3's clear-state steps used a value (the transcribe id) in step 2 that step 1 had already deleted in step 3's original ordering — self-contradictory as written | Reordered: read the transcribe id from the AIcall already in hand, `SREM` using it, *then* clear the DB metadata | §5.7.3 |
| M2 | `UpdateListenState` `SADD`s the new resolver-set membership but never `SREM`s a stale old one when the §5.1.1 idempotency check restarts listening with a fresh transcribe | Old membership is now explicitly `SREM`'d before the new `SADD`, when a prior `listen_transcribe_id` existed | §5.2.4 |
| M3 | The six Insight tools' `RunLLM: true` citation was attributed to `tool_insight.go` (which contains no `RunLLM` occurrences); the real location is `pkg/toolhandler/definitions.go` at the same line numbers | Citation corrected | §5.4.4(a) |
| M4 | §5.4.2 item 4's `Role ∈ {user, assistant}` filter still admits empty-content tool-call rows that `run.py:450` discards anyway, so the effective Q&A context is smaller than the configured 10-row budget | Noted as a known, minor inefficiency; not blocking (§5.4.5's `Origin` exclusion is the mechanism that matters for correctness, this is a small further headroom improvement left for implementation) |  — |
| M5 | §5.6.4's render-filter coverage claim ("every existing... panel noise") overstated: it does not hide `role=system` rows (`startInitMessages`, `start.go:812-819`), which both panels already render today | Scope note added: the filter covers tool-call/tool-result noise specifically; `role=system` rendering is a separate, pre-existing, out-of-scope concern | §5.6.4 |

### 10.3 Review-response matrix (round 4 → rev 5)

Round 4 (again an independent, skeptical-by-instruction review) found
that of rev 4's three new mechanisms, only §5.2.1 (the separate system
customer id) held up cleanly; §5.4.4(c) and §5.4.5 each had a real bug,
and the reviewer's own closing assessment was that the design's skeleton
is sound and consistent with the code — the remaining issues were all in
implementation-level details of the newest additions, not architecture.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| B1 | §5.4.4(c)'s cache-bypass re-read sat in the wrong branch: it neither caught the false-allow case it was written for, nor avoided creating a new one | Rewritten as a single always-fresh-read (not on a hot path, so no reason to trust the cache at all here), removing the two-branch logic entirely | §5.4.4(c) |
| B2 | A `pipecatcallID` of `uuid.Nil` (old `pipecat-manager` build, rolling-deploy window) was unhandled; depending on how §5.4.4(c)/§5.4.5 read it, could permanently mistag real Q&A content as `listen_internal` | `uuid.Nil` explicitly treated as "assume a real Q&A turn" in both places — the fail-safe direction, since guessing wrong that way costs one rejected tool call, never data corruption. Wire field made optional so no deploy ordering is forced | §5.4.3a, §5.4.4(c), §5.4.5 |
| B3 | `ApplyFields` was mis-located in `bin-ai-manager` (actually `bin-common-handler/pkg/databasehandler`, shared by every service); the proposed `FieldOriginNot` mechanism was unspecified and would have produced a SQL error on every Q&A turn | Correct location cited; mechanism decided as a generic `databasehandler.NotEq` wrapper type (not another hardcoded field-name special case), verification scope widened to the whole monorepo for this one change | §5.4.5 step 3, §8, §9 |
| B4 | Excluding only listen-internal rows narrows but does not close the context-eviction risk — proactive rows and the agent's own Q&A tool rows still compete for the 100-row window and can still evict the system prompt | `getPipecatcallMessages` restructured into two fetches: the leading system row(s), always included regardless of window pressure, plus the capped, `NotEq`-filtered rest | §5.4.5 step 4 |
| H1 | Guarding all four original pipecatcall-message handlers would put an uncached DB read on the highest-volume one; narrowed to two, but the *justification* given (the other two "can never fire from a listen turn") was correct while the guard itself was still overbroad on the two it did cover | (carried forward from round 3, no new action — round 4 confirmed round 3's H1/H2 fix as correct) |  — |
| H2 | `Origin` tagging in `ToolHandle` was not scoped by reference type, so an ordinary `conversation`/`task`/`none` AIcall's routine pipecatcall rotation (via `Send()`) would be mistagged `listen_internal` and silently lose history | Tagging rule now requires `ac.ReferenceType == ReferenceTypeContactCase` in addition to the pipecatcall-id mismatch | §5.4.5 step 2 |
| H3 | `notify_agent` was missing from the public OpenAPI tool-name enum and the Insight-tool-list prose docs, unlike every prior Insight tool | Added explicitly to §5.5.1 and §9 | §5.5.1 |
| H4 | The cache-bypass re-read's cited code path (`dbhandler.AIcallGet(skipCache)`) doesn't match how `EventPMMessageBotLLM` actually fetches the AIcall (an RPC, `AIV1AIcallGet`, not a direct `dbhandler` call) | Corrected: the RPC client gains the optional cache-bypass argument instead, one hop further down than rev 4's snippet showed | §5.4.4(b) |
| H5 | No explicit migration-before-deploy ordering; `Origin`/`ListenCallID` existing in the Go struct changes every message/AIcall query's `SELECT`, so a code deploy landing before its migration is a hard outage, not a soft degradation | Made an explicit, ordered rollout step rather than an implicit assumption | §8 |
| M1 | `IDAIManagerListen` not registered in `bin-customer-manager`'s known-system-id whitelist | Confirmed the listen path never traverses that gate (so this doesn't block), and stated the omission is deliberate scope discipline, not an oversight | §5.2.1 |
| M2 | §5.4.3a's "one new parameter" understated the change — `mapFunctions`' shared signature is used by 21 handlers | Clarified: only `ToolHandle` and `toolHandleNotifyAgent` need the new parameter; the other 20 handlers are unaffected | §5.4.3a step 4 |
| M3 | Adding `FieldOriginNot` to `FieldStruct` would expose an unused, unnecessary RPC filter surface (`FieldStruct` only gates external-RPC-visible filters; `getPipecatcallMessages` builds its filter map directly) | Not added — the existing `FieldOrigin` constant plus the new `NotEq` wrapper is sufficient, no `FieldStruct`/`ConvertFilters` change needed | §5.4.5 step 3 |
| M4 | The `RunLLM: true` line-number citation for the six Insight tools was still wrong after round 3's file-path fix — it reused the `RunLLM: false` tools' line numbers | Corrected to the six tools' own lines (754-755, 785-786, 824-825, 849-850, 879-880, 906-907) | §5.4.4(a) |
| M5 | `args.pop("run_llm", …)` cited at `tools.py:~60`; actual location `tools.py:105` | Corrected | §5.4.4(a) |
| M6 | The stated cost reason for not guarding two handlers ("adds a per-utterance AIcall lookup") was factually wrong — both already do that lookup today; the real reason is structural (the condition cannot occur) | Cost claim removed, structural reasoning kept (conclusion was already correct) | §5.4.4(b) |
| M7 | `get_aicall_messages` (an existing tool) bypasses `getPipecatcallMessages` entirely and can leak `listen_internal` rows' raw JSON into an LLM answer | Acknowledged as a real, lower-severity gap and left as a follow-up rather than fixed here | §5.4.5, §11 |
| M8 | `listen_internal` was to be left undocumented in the public RST while still appearing on actual webhook payloads | Documented plainly instead (as an internal-bookkeeping value, not relied upon), since hiding a value that's genuinely on the wire is worse than documenting it | §9 |
| M9 | Two items in §11 (Redis version, speaker mapping) are still open and self-marked blocking | Correct as stated — both remain open pending empirical/operational confirmation, not resolved by this revision |  — |
| L1–L3 | Minor citation/line-number errors (`PromptSnapshot` location, missing files in §9, raw-vs-`.String()` UUID convention) | Corrected where cited above; `.String()` convention left to implementation (both forms work through `ConvertFilters`) | various |

---

## 11. Open items before implementation sign-off

1. **`in`/`out` speaker mapping empirical verification (§5.9) — blocking.**
   Capture one real or staged agent-bridged call and confirm the mapping
   against known speaker identity before merge. A reversed mapping is a
   silent correctness failure, not a cosmetic one.
2. **Confirm the deployed Redis version supports `LPOP key count` (Redis
   ≥ 6.2) — blocking, elevated in rev 4 (review round 3: this is not a
   deferrable implementation nicety).** §5.4.1's atomic pop-all of the
   `pending` buffer is load-bearing for the "no line lost to a
   concurrent appender" property; if the deployed Redis predates 6.2,
   §5.4.1 needs a different primitive (e.g. `MULTI`/`EXEC` wrapping
   `LRANGE`+`LTRIM`) chosen *before* implementation starts, not
   discovered during it. A five-minute check (`redis-cli INFO server`
   against the production instance) resolves this; do it before rev 4 is
   considered final.
3. **Pre-existing orphaned `tool`-role message defect (§5.6.3) —
   narrowed in rev 4.** §5.4.5's fix (tagging listen-turn tool rows
   `Origin=listen_internal` and excluding them from every future replay)
   means this feature can no longer *create* an instance of the defect.
   The defect itself — an agent-initiated tool call's own tool-call row
   already gets filtered by `run.py:450`'s empty-content check today,
   independent of listening — is real, predates this design, and is
   worth confirming against production traffic/logs. It is a genuine
   platform bug, but no longer gated on or made worse by this feature
   shipping, so it returns to a follow-up ticket (recommend filing
   promptly regardless, given its severity if confirmed) rather than a
   rollout blocker.
4. **No Jira ticket filed.** Recommend filing a `VOIP-*` ticket for this
   feature before implementation starts, per project convention — and a
   **separate** ticket for item 3 above.
5. **Confirm whether a new system `customer_id` sentinel (§5.2.1,
   `IDAIManagerListen`) needs a real `bin-customer-manager` row or is
   usable as a bare constant.** Small, but needs an answer before
   `pkg/aicallhandler` can reference it.
6. **Follow-up ticket (separate):** `transcripthandler.dbDelete` publishes
   `EventTypeTranscriptCreated` on delete
   (`bin-transcribe-manager/pkg/transcripthandler/db.go:33`). This design
   defends against it rather than fixing it, because changing the emitted
   event type is a routing-key-visible change affecting every current
   subscriber. It should be fixed on its own ticket.
7. **Follow-up ticket (separate):** decouple `Send`'s cooldown from
   `tm_update` onto a dedicated `tm_last_send`. Pre-existing fragility
   (`send.go:27-32` + `dbhandler/aicall.go:240`) that this design bounds
   but does not remove.
8. **Follow-up ticket (separate):** webhook noise from tool-call rows.
   §5.6.4/§5.10.1 hide the two per-tool-call noise rows from the *panel*
   (a client-side render filter), but the tenant's own webhook consumer
   still receives every `aimessage_created` delivery for them, exactly as
   today. Whether to suppress those webhooks too (and whether any tenant
   automation actually depends on receiving them) is a genuinely separate
   decision from the frontend fix shipped with this design.
9. **Implementation-time decision, not blocking:** where listen's new
   Redis operations live — extend `pkg/cachehandler` or give them their
   own small package (§9's scope note).
10. **Product decision (not blocking implementation):** whether Insight
    listening becomes a billed line item, and if so under which meter
    (§3, §5.2.1). The architecture keeps the STT cost off the customer's
    transcription bill, which makes this a clean, deliberate pricing
    choice rather than an accident.
