# InsightAI Realtime Listen for Conversation (message) Cases (VOIP-1470) - Design

Status: **Approved (rev 7, implementation notes added) - 2 consecutive review approvals (rounds 4 and 5) on rev 5; design review loop CLOSED.** Round history: round 1 REQUEST_CHANGES (14), round 2 REQUEST_CHANGES (9, 2 HIGH), round 3 REQUEST_CHANGES (7, none blocking), round 4 APPROVE (4 LOW, folded into rev 5), round 5 APPROVE (5 LOW, explicitly judged not to warrant another round, folded into rev 6). Matrices in §11 to §11.4. Next step: implementation PR under VOIP-1470, followed by the code review loop.

Ticket: [VOIP-1470](https://voipbin.atlassian.net/browse/VOIP-1470).
Parent design (call variant, merged in PR #1266, `c243f926b`):
[`2026-09-03-insight-ai-realtime-listen-design.md`](2026-09-03-insight-ai-realtime-listen-design.md),
referred to below as **CL** (call listen). Section numbers written `CL §x.y` point into that document.
Everything CL already specifies and that this document does not override is inherited unchanged.

All `file:line` references were read against `main @ c243f926b` on 2026-09-05.

---

## 0. Revision history

| Rev | Date | Change |
|---|---|---|
| 1 | 2026-09-05 | Initial draft after the issue-analysis review loop (2 consecutive approvals). |
| 2 | 2026-09-05 | Review round 1 (REQUEST_CHANGES): idle-expiry path corrected (HIGH-1), replica model corrected to competing consumers on the shared `QueueNameAISubscribe` queue and bounds re-derived from the Redis lock (HIGH-2), `ContactV1CaseGet` failure semantics fixed (M-3), `kase.ReferenceTypeConversationMessage` constant added (M-4), flag-off result unified to `skipped_disabled` (M-5), turn-time sub-flag check scoped (M-6), ordering stated (M-7), `Delete`-before-`TryLock` invariant stated (M-8), overlapping-turn exposure stated (M-9), false import claim removed (L-10), terminate gate keeps the `ReferenceType` guard (L-11), six citations corrected (L-12), metrics strategy unified to a `kind` label (L-13), tenant assertion at intake (L-14). Matrix in §11. |
| 3 | 2026-09-05 | Review round 2 (REQUEST_CHANGES): idle-expiry fix scoped to the only reachable site `start.go:492` (HIGH-1), conversation idempotency moved into `startListenConversation` with `SISMEMBER` and the `proceed=false` control flow specified (HIGH-2), `ListenPendingLen` empty short-circuit so flush turns never burn the turn cap or a Case RPC (M-3), §9 reconciled with §5.13 (M-4), metrics blast radius enumerated incl. `notify` Counter->CounterVec and test pins (M-5), `tool_insight.go:21` (L-6), residual citations (L-7), §4 diagram tenant line (L-8), cache-first/DB-on-miss wording and replica-count decoupling (L-9). Matrix in §11.1. |
| 4 | 2026-09-05 | Review round 3 (REQUEST_CHANGES, none blocking): `kind="unknown"` value for the pre-branch metric sites (M-1), §9 cells reconciled for the resolver quartet, `ListenPendingLen` and step 5b (M-2), §4 diagram and §5.3.2 step 5 folded into the per-AIcall loop (L-3), flush `ran` read against turn `skipped_empty` (L-4), §11 row 1 marked superseded (L-5), raw `media.Type` with all nine values (L-6), `publisherConversationManager` constant (L-7). Matrix in §11.2. |
| 5 | 2026-09-05 | Review round 4 (APPROVE, 4 LOW): three more `kind` sites classified (`listen_trigger.go:254`, `listen.go:272` -> `unknown`; `listen_trigger.go:217` -> `call`) (L-1); §4 diagram corrected (pending is not trimmed) and a §6 row for the closed-Case, outgoing-only accumulation path with its bounds (L-2); §5.3.2 wrapper returns `error`, handler has no return (L-3); §11 row 10 annotated (L-4). Matrix in §11.3. |
| 6 | 2026-09-05 | Review round 5 (APPROVE, 5 LOW, no re-review required): §5.1 call signature matched to §5.1.1 (L-1); intake-path `kind` attribution incl. `listen.go:720` and the `none -> unknown` rule (L-2, L-3); `failed` bucket for the step-4 `h.Get` error (L-4); `processEvent*` signature citation (L-5). Design APPROVED. Matrix in §11.4. |
| 7 | 2026-09-05 | Implementation notes (VOIP-1470 PR): (a) the trigger's conversation branch additionally fetches the conversation via ConversationV1ConversationGet and refuses a cross-customer or nil-customer conversation with skipped_not_listenable (RPC error -> failed), a defence-in-depth check the intake tenant assertion already implied (code review finding); (b) listenKind helpers live in pkg/aicallhandler/listen_kind.go to keep listen.go under the 800-line guideline; (c) the terminate gate is the pure helper listenTerminateNeedsStop (ListenCallID OR listen pointer), unit-tested directly; (d) RunListenTurn's predicate-failed skipped_invalid emits the AIcall's actual kind (unknown only when no pointer exists); (e) intake meters stale resolver entries as dropped_stale (a resolved AIcall already over, or a pointer naming another conversation), keeping dropped_unknown for an unresolved or errored lookup; (f) the intake re-arms the conversation resolver TTL (EXPIRE only, never SADD, so a concurrently stopped membership is never resurrected) once per message that buffered at least one line, because start is the only SADD and a session can outlive listenResolverTTL (code review round 3, round 4). |

---

## 1. Problem statement

The Case Insight Assistant can follow a live **call** transcript and proactively push a
`notify_agent` note to the agent panel (CL). It cannot do the same for a Case that originated from a
messaging conversation (SMS/MMS, LINE, WhatsApp, webchat, email).

This is a hard gap, not a flag:

- `bin-ai-manager/pkg/aicallhandler/listen_trigger.go:253` (step 5 of `checkListenEligible`) rejects any
  Case whose `ReferenceType != "call"` with `skipped_not_listenable`.
- `bin-ai-manager/pkg/subscribehandler/main.go:56-74` binds no `conversation-manager` topic
  (`binding_golden_test.go:31` pins the count at 12).
- `listen.go:667 EventTMTranscriptCreated` is the only intake and it resolves by `transcribe_id`.
- `listen.go:268-273 RunListenTurn` requires `listen_transcribe_id` in `AIcall.Metadata`.
- `aicallhandler/main.go:315 ListenTurnSystemPrompt` says "live phone call".

Today the only conversation-aware path is pull: the `get_conversation_content` tool
(`tool_insight.go:201`) when the agent asks. Nothing pushes.

## 2. Goals

1. A Case with `reference_type = "conversation_message"` whose Insight panel is open receives
   proactive `notify_agent` notes when new customer messages match the customer-configured
   `init_prompt` conditions, exactly as the call variant does.
2. Reuse the CL pipeline wholesale: the `POST /service_agents/aicalls/{id}/listen` trigger, the Redis
   pending/window/lock/turn-count keys, `RunListenTurn`, the listen-turn pipecatcall id set, the
   `notify_agent` tool and its listen-turn gating, `Origin=proactive` rows, webhook delivery and
   panel rendering. **Only the input source and the lifecycle differ.**
3. No new billed resource. There is no STT session; the only paid unit per turn is the LLM call,
   bounded by the existing debounce and per-AIcall turn cap.
4. Ship dark behind the existing `aicall_listen_enabled` master switch plus a variant-specific
   sub-switch, with a rollback that stops in-flight sessions at their next evaluated turn.
5. No DB migration.

## 3. Non-goals

- Changing the customer-facing conversation chatbot (`ReferenceTypeConversation` AIcalls,
  `start.go:245`). That product replies to the customer; this one assists the agent.
- Any frontend change. Both panels already fire the listen trigger on every open regardless of the
  Case's reference type (`square-talk/src/features/cases/CaseInsightAssistantPanel.jsx:241,520`,
  `square-admin/src/views/contacts/CaseInsightAssistantPanel.js:174`) and already render
  `origin === 'proactive'` rows (`InsightAssistantTimelineParts.jsx:58-59`).
- Adding `case_closed` / `case_continued` events to contact-manager. Listed in §10 as an optional
  follow-up; this design does not depend on them.
- Inbound email. `bin-conversation-manager/pkg/conversationhandler/email.go` creates only
  `DirectionOutgoing` rows today (`:93-100`); there is no `DirectionIncoming` email writer. Email-origin
  Cases therefore never produce a trigger line (§5.3.3) until inbound email lands in
  conversation-manager under a separate ticket. They are deliberately not excluded from the resolver,
  so they start working automatically when that ticket ships.
- Evaluating outgoing (agent or bot authored) messages as a trigger (§5.3.3).
- Cross-Case fan-out. One conversation may, over time, back more than one Case (`previous_case_id`
  chains). The resolver is a set (§5.2.2) so this is handled, but no ordering or dedupe across Cases
  is attempted.

## 4. Architecture overview

```
              conversation-manager
              conversation_message_created  (every message row, both directions, all channels)
              routing key: conversation-manager.conversation.<conversation_id>.message_created
                              |
                              v
   +-----------------------------------------------------------------------+
   | LAYER 1 - intake (no LLM, no DB write, no RPC)                          |
   |  subscribehandler.processEventCVMessageCreated                          |
   |   . drop if TMDelete != nil or (Text empty and no Medias)              |
   |   . Redis SMEMBERS ai:listen:conversation:<conversation_id>            |
   |       -> empty = not ours, drop; else fan out per AIcall               |
   |   . per AIcall: Get (cache-first) + assert customer_id matches         |
   |   . line = "[CUSTOMER] ..." / "[AGENT] ..." from Message.Direction      |
   |   . per AIcall: RPUSH pending (EXPIRE only) + window (LTRIM, EXPIRE)   |
   |   . per AIcall, direction == incoming -> try SET NX lock for THAT id   |
   |        acquired  -> go RunListenTurn(aicallID)                         |
   |        not acquired -> schedule ONE deferred flush for it (new, §5.4)  |
   |     direction == outgoing  -> context only, never a lock attempt       |
   +------------------------------------+----------------------------------+
                                        |
                                        v
   +-----------------------------------------------------------------------+
   | LAYER 2 - evaluation turn (unchanged shape, CL §5.4)                    |
   |  aicallHandler.RunListenTurn                                            |
   |   . predicate generalised: transcribe id OR conversation id (§5.5.1)  |
   |   . NEW: ContactV1CaseGet -> if Case closed, stopListening (§5.7.2)    |
   |   . bounded context: Insight prompt + prompt snapshot                  |
   |       + ListenTurnConversationSystemPrompt (variant, §5.5.3)           |
   |       + last N Q&A rows + rolling message window                       |
   |   . throwaway pipecatcall, registered in ai:listen:turnpcid            |
   +------------------------------------+----------------------------------+
                                        |
                       notify_agent -> Origin=proactive row -> webhook + panel
                       anything else -> dropped (CL §5.4.4)
```

Lifecycle differences from CL, summarised (details in §5.7):

| | Call (CL) | Conversation (this doc) |
|---|---|---|
| Start | trigger API -> confbridge wait -> transcribe start | trigger API -> resolver SADD only |
| Owned resource | transcribe session (billed) | none |
| Resolver key | `ai:listen:transcribe:<transcribe_id>` | `ai:listen:conversation:<conversation_id>` |
| Persisted pointer | `ai_aicalls.listen_call_id` column + metadata | `AIcall.Metadata["listen_conversation_id"]` only |
| Tail flush | call_hangup event | deferred flush timer (§5.4) |
| Stop | call_hangup sweep, terminate, cap, flag | terminate, closed Case at turn time, cap, flag, idle |

## 5. Design

### 5.1 Trigger: reuse `POST /service_agents/aicalls/{id}/listen`

The endpoint, api-manager handler (`serviceagent_aicall.go:140-174`, a pure forwarder to
`AIV1AIcallListen` with agent/tenant checks only, so it needs no Case-type gate), and
`ProcessListen` (`listen_trigger.go:59`) are unchanged. The panel already calls it on every open and on
Start (CL §5.10.1a).

`checkListenEligible` (`listen_trigger.go:125`) keeps steps 1-4 verbatim (flag, Insight-type and
liveness gate, idempotency, Case lookup with tenant check) and **branches at step 5** on
`kase.ReferenceType`:

| `kase.ReferenceType` | Behaviour |
|---|---|
| `kmkase.ReferenceTypeCall` (`"call"`) | existing path, unchanged (steps 6-8: call resolve, confbridge wait, transcribe start) |
| `kmkase.ReferenceTypeConversationMessage` (`"conversation_message"`, **new constant**, §5.8) | **new** `checkListenEligibleConversation` (§5.1.1) |
| anything else, or `ReferenceID` that does not parse to a non-nil UUID (`actionhandle.go:1383-1386` allows `""`) | `skipped_not_listenable`, as today (`listen_trigger.go:257-260` guard reused) |

Today `"conversation_message"` exists only as a bare literal in flow-manager
(`actionhandle.go:1341`); `models/kase/kase.go:109` defines only `ReferenceTypeCall`. The constant is added
to `bin-contact-manager/models/kase` and the flow-manager literal is converted in the same PR, so the three
producers/consumers are linked at compile time.

**Control flow (rev 3, review round 2 HIGH-2).** Step 3's existing idempotency check
(`listen_trigger.go:203-215`) compares a transcribe's `ReferenceID` against `c.ListenCallID`; it is a
no-op for a conversation listen (no `listen_transcribe_id` in metadata, so `existingID == uuid.Nil` and the
block is skipped) and is left untouched. The conversation branch's own idempotency lives inside
`checkListenEligibleConversation`, because the conversation id it needs is only known after step 5
resolves `kase.ReferenceID`.

`checkListenEligible` keeps its return shape `(a, kase, callID, call, proceed, err)`. On the
conversation branch it parses `kase.ReferenceID` and calls `h.startListenConversation(ctx, c, conversationID)`
**inline** (§5.1.1, two idempotent operations, no goroutine) and then returns `proceed = false`, so `ProcessListen`
(`listen_trigger.go:69-82`) never spawns `runListenStart`, which is call-only. `ProcessListen` itself
is unchanged. `startListenConversation` meters `aicall_listen_start_total{kind="conversation"}` with
`started | reused | failed`; the eligibility gates before it meter `skipped_not_listenable | skipped_disabled`.

Variant gate (step 5b, evaluated only on the conversation branch, after the reference-type switch):
`aicall_listen_conversation_enabled` (§5.12). Off -> `skipped_disabled` on
`aicall_listen_start_total{kind="conversation"}` (§5.13). Placed at step 5b rather than next to the master
flag so a disabled conversation variant can never affect the call branch. This lets the call variant be
enabled in production while this one stays dark, and vice versa.

#### 5.1.1 `checkListenEligibleConversation` -> `startListenConversation`

Inline, synchronous, no goroutine and no lock. CL's `runListenStart` goroutine, confbridge poll and
per-AIcall start lock (CL §5.1.1 step 7, §5.2.2) exist because a call listen has a slow, billed,
externally owned resource to create or reuse. A conversation listen has none, so the whole start is an
idempotency check followed by two idempotent operations:

```
// checkListenEligibleConversation (inside checkListenEligible's step-5 switch)
if !config.Get().AIcallListenConversationEnabled { skipped_disabled; return proceed=false }        // step 5b
conversationID := uuid.FromStringOrNil(kase.ReferenceID)
if conversationID == uuid.Nil { skipped_not_listenable; return proceed=false }
h.startListenConversation(ctx, c, conversationID)
return proceed=false                                                     // never spawn runListenStart

// startListenConversation
// 0. idempotency: pointer already set AND resolver membership present -> reused (the common panel-reopen path)
if listenConversationIDFromMetadata(c) == conversationID {
    if member, err := cache.ListenConversationAIcallIDIsMember(ctx, conversationID, c.ID); err == nil && member {
        promListenStartTotal(kind=conversation, "reused"); return
    }
    // pointer set but membership missing (Redis flush): fall through and re-SADD; the DB write below is a no-op rewrite
}
// 1. resolver membership (idempotent SADD, TTL listenResolverTTL = 12h, same constant as CL)
if err := cache.ListenConversationAIcallIDAdd(ctx, conversationID, c.ID, listenResolverTTL); err != nil {
    promListenStartTotal(kind=conversation, "failed"); return
}
// 2. persist the pointer for stop paths (no TMUpdate touch, same as CL: AIcallUpdateNoTouchTMUpdate)
metadata[MetaKeyListenConversationID] = conversationID.String()
if err := db.AIcallUpdateNoTouchTMUpdate(ctx, c.ID, {FieldMetadata: metadata}); err != nil {
    _ = cache.ListenConversationAIcallIDRemove(ctx, conversationID, c.ID)   // best-effort rollback
    promListenStartTotal(kind=conversation, "failed"); return
}
promListenStartTotal(kind=conversation, "started")
```

`ListenConversationAIcallIDIsMember` (`SISMEMBER`) is the fourth cache primitive, alongside the
`Get|Add|Remove` trio (§5.8). A "stale metadata but missing resolver entry" state (Redis flush)
therefore self-heals on the next panel open at the cost of one `SADD` and one no-op metadata rewrite;
no rollback logic is needed, and `RunListenTurn`'s predicate reads the metadata pointer, not the
resolver, so the AIcall is never stopped in between (§5.5.1).

Ordering: resolver first, then DB, mirroring CL rev 16's "write the resolver before anything can
publish" reasoning (CL §5.1.1). If the DB write fails the resolver entry is removed; if the removal
also fails the entry expires in 12h and every intake hit on it lands in `RunListenTurn`'s predicate
(§5.5.1), which sees no metadata pointer and calls `stopListening`, which `SREM`s it. Bounded either way.

No "already progressing" check against conversation-manager is needed: the conversation is not a
session, it is a stream that exists whether or not anyone listens.

Concurrency: two panels opening the same AIcall at once both run the sequence above. Both SADDs are
idempotent and both DB writes write the same value. No lock.

The response is unchanged and still carries no listening-status field (CL §5.1, §11 item 14).

### 5.2 Listen state

#### 5.2.1 Persisted pointer: metadata, not a column

CL added `ai_aicalls.listen_call_id` because `call_hangup` needs a DB sweep by call id
(`stopListenByCallID`, `listen.go:543`, filter `FieldListenCallID`). No conversation-side event drives a
sweep here (§5.7), so the pointer lives in `AIcall.Metadata` next to the existing listen keys:

| Key | Type | Meaning |
|---|---|---|
| `listen_conversation_id` (`aicall.MetaKeyListenConversationID`) | string UUID | the conversation this AIcall is listening to |

`listen_transcribe_id` / `listen_owns_transcribe` are never set on a conversation listen. An AIcall
row therefore carries at most one of {`listen_transcribe_id`, `listen_conversation_id`}; the helper
`listenKind(c)` returns `call | conversation | none` from that and is the single discriminator used by
`RunListenTurn`, `stopListening`, `clearListenState` and `ProcessTerminate`.

Because `commondatabasehandler.GetDBFields` reflects the struct for every SELECT (CL's deploy-order
warning), **not adding a column is what makes this change deployable in one step.** No Alembic
migration, no deploy ordering constraint.

#### 5.2.2 Redis resolver

| Key | Type | Written | Read | Removed |
|---|---|---|---|---|
| `ai:listen:conversation:<conversation_id>` | SET of AIcall ids, `EXPIRE` 12h refreshed on every `SADD` | §5.1.1 | §5.3.2 (`SMEMBERS`) | `clearListenState` (`SREM`) |

A SET, for the same reason as CL §5.2.4: two Insight AIcalls (two agents, or an admin and an agent)
may listen to one conversation, and a conversation may back a later Case after `previous_case_id`
chaining. The key is purpose-built and explicitly managed; it is **not** part of `cachehandler.AIcallSet`'s
snapshot index scheme (CL §5.3.2 explains why that scheme must not be reused).

Cache-loss behaviour, stated: a Redis flush drops resolver membership; the AIcall stops receiving
lines until the panel is reopened, which re-runs §5.1 and repopulates in one `SADD`. There is
deliberately no DB fallback on the intake hot path.

Reuse of every other key (`ai:listen:pending|window|lock|turns|turnpcid:<aicall_id>`) is unchanged,
including `ListenStateClear`'s exclusion of `turnpcid` (rationale `cachehandler/listen.go:347-355`, func `:356`).

### 5.3 Layer 1 - event intake

#### 5.3.1 Binding

`topicPatterns` (`subscribehandler/main.go:56-74`) gains one pattern **appended at the end**:

```
eventtopic.PatternForEventType(string(commonoutline.ServiceNameConversationManager), cvmessage.EventTypeMessageCreated)
// = "conversation-manager.conversation.*.message_created"
```

Derivation, verified: `splitEventType` splits on the first underscore
(`bin-common-handler/models/eventtopic/routingkey.go:61-69`, `SplitN(_, "_", 2)`), so
`conversation_message_created` -> resource `conversation`, action `message_created`; the subscription
segment is `Message.ConversationID` (`models/message/message.go EventSubscriptionID`); the produced key
is pinned by `bin-conversation-manager/models/conversation/routingkey_golden_test.go:155-159`. The
pattern does **not** overlap `conversation_created` (action `created`).

Four coupled edits, as in CL §5.3.1: `topicPatterns` (append); a `publisherConversationManager`
constant added to the existing publisher const block (`subscribehandler/main.go:33-37`, the file's
established style, used at `:229`); `processEvent`'s switch (new case
`m.Publisher == publisherConversationManager && m.Type == cvmessage.EventTypeMessageCreated`); and
`binding_golden_test.go` (`expected` slice append, `12 -> 13` in both places, doc comment).

`bin-ai-manager` already imports `bin-conversation-manager/models/message` as `cvmessage`
(`pkg/aicallhandler/tool_insight.go:21`); the subscribehandler import is the same model-only dependency, no
new RPC client. `commonoutline.ServiceNameConversationManager` exists (`models/outline/servicename.go:21`)
and conversation-manager publishes on the global topic exchange
(`cmd/conversation-manager/main.go:123`, `WithGlobalTopicPublish`).

#### 5.3.2 Handler and the per-event floor

```go
func (h *subscribeHandler) processEventCVMessageCreated(ctx context.Context, m *sock.Event) error {
    var evt cvmessage.Message
    if err := json.Unmarshal([]byte(m.Data), &evt); err != nil { return err }
    h.aicallHandler.EventCVMessageCreated(ctx, &evt)   // no return value; every drop is metered, never an error
    return nil
}
```

`EventCVMessageCreated(ctx, *cvmessage.Message)` (aicallhandler, new file `listen_conversation.go`, same
no-return shape as `EventTMTranscriptCreated` at `listen.go:667`; the subscribehandler wrapper keeps the
`error` return every sibling `processEvent*` has, e.g. `subscribehandler/transcribemanager.go:24`, dispatched
from the switch at `subscribehandler/main.go:222-231`):

1. Drop if `evt.TMDelete != nil` -> `dropped_deleted`. (Defensive parity with CL H3; `messagehandler/db.go`
   does not publish `message_created` on delete today, but the guard is one line.)
2. Drop if `strings.TrimSpace(evt.Text) == ""` **and** `len(evt.Medias) == 0` -> `dropped_empty`.
3. `aicallIDs := cache.ListenConversationAIcallIDsGet(ctx, evt.ConversationID)` (`SMEMBERS`). Empty ->
   `dropped_unknown`, return. **This is where >99% of platform events end.**
4. Build one line (§5.3.3), then per AIcall: `cache`-first `h.Get(aicallID)`; if that errors,
   `failed` + warn and skip that AIcall (the tenant assertion cannot be made, so nothing is buffered);
   otherwise **assert `c.CustomerID == evt.CustomerID`** (both non-nil), else `dropped_tenant_mismatch`
   + warn and skip that AIcall. The resolver key is a UUID so a collision is impractical, but every sibling path in this
   package asserts the tenant. `h.Get` is cache-first with a DB fallback on miss
   (`dbhandler/aicall.go:111-125`); acceptable because it runs only for the resolved fraction, never for
   the platform-wide majority that ended at step 3. Then `ListenPendingPush`,
   `ListenWindowPush` (same TTL and window size flags as CL), `buffered`.
5. Still inside step 4's per-AIcall loop: if `evt.Direction == DirectionIncoming`, run the debounce
   (§5.4) **for that `aicallID`**, so two AIcalls listening to one conversation each take their own
   `ai:listen:lock:<aicall_id>` and each may arm their own deferred flush. Outgoing: nothing further.

**Delivery model, corrected in rev 2 (review round 1 HIGH-2).** ai-manager consumes one **shared** queue,
`commonoutline.QueueNameAISubscribe` (`cmd/ai-manager/main.go:202`, `subscribehandler/main.go:163`). The two
replicas are competing consumers: each event is delivered to exactly one replica, round-robin, not copied
to both. Consequences used below: (a) intake cost is paid once per event platform-wide, not per replica;
(b) consecutive messages for one conversation can land on different replicas, so any per-replica
in-process state (§5.4's timer marker) is advisory only and every correctness bound must come from Redis.

**Ordering is best-effort.** `processEventRun` spawns a goroutine per event
(`subscribehandler/main.go:172`, prefetch 10), so two messages seconds apart can be pushed to the window
out of order, and cross-replica delivery adds a second source of reordering. CL accepts the same for
transcript segments; for a chat the effect is more visible but still harmless (the LLM sees both lines
in one window). No `TMCreate`-based reordering is attempted; stated so no one assumes a transcript-exact
window.

**Fan-out cost, sized.** The wildcard delivers every conversation message platform-wide, all channels,
both directions, once. Per event: one AMQP delivery, one goroutine, one JSON unmarshal, one Redis
`SMEMBERS`. No DB, no RPC (the `h.Get` in step 4 runs only for the tiny fraction that resolves). This is
the same justification CL §5.3.2 and `subscribehandler/main.go:68-73` already make for the transcript
wildcard, and message volume is lower than final-STT-segment volume by construction. The dynamic
per-conversation binding escape hatch (`conversation-manager.conversation.<id>.#`,
`eventtopic.PatternInstance`) exists and is pre-documented in CL §3 with its leak-sweeper requirement; not
adopted here.

#### 5.3.3 Line format, direction rule, media, truncation

Speaker tag is structural, from `Message.Direction`:

| `Direction` | Tag | Trigger? |
|---|---|---|
| `incoming` | `[CUSTOMER]` | yes (§5.4) |
| `outgoing` | `[AGENT]` | **no**: buffered as context only, never a lock attempt |
| `""` (`DirectionNond`) | `[SPEAKER]` | no |

Why outgoing never triggers: outgoing rows publish too (`messagehandler/db.go:75` is unconditional and
`send.go:66,106,191,294` all route through it), and an outgoing row may be authored by the agent, by a
flow, or by a customer-facing chatbot AIcall on the same conversation. Evaluating those as a fresh
signal would (a) spend a turn on text the agent already knows, and (b) let the AI react to its own
sibling's output. Confirmed per channel that customer-authored rows are `incoming` for webchat
(`event_webchat.go:104-109`), SMS/MMS (`conversationhandler/message.go:123-128`), LINE
(`linehandler/hook.go:177`) and WhatsApp (`whatsapphandler/hook.go:163`); email has no inbound writer (§3).

Outgoing rows with `Status = progressing` at create time (email, `email.go:99`) are buffered as-is even
if delivery later fails; they are context, never a trigger, so a failed send costs nothing but one
stale context line. Accepted.

Text assembly (`conversationMessageLine(evt)`):

```
[TAG] <Subject: ...\n if Subject non-empty><Text, whitespace-trimmed, truncated to
      aicall_listen_conversation_max_message_chars with a trailing " [truncated]">
      <one " [media: <type>]" token per Medias entry, e.g. " [media: image]">
```

Truncation exists because an email or pasted webchat body can be tens of kilobytes and the window is
line-counted, not byte-counted (CL §5.3.3). Default 2000 chars (§5.12). Media placeholders carry the
raw `media.Type` string, whatever value the row holds (the nine declared today are
`image|video|audio|file|location|sticker|template|imagemap|flex`, `models/media/media.go:26-34`), and
nothing else; no URL, no payload. No allowlist, so a future media type needs no change here.

### 5.4 Debounce and the deferred flush

CL §5.3.4's leaky-bucket debounce is reused verbatim: `ListenTurnTryLock` = `SET ai:listen:lock:<id> NX EX interval`.
CL accepts that "the last few lines before a silence wait for the next line" because a call always
ends with a `call_hangup` that performs a final flush turn. **A conversation has no hangup.** Two customer
messages 5s apart would leave the second unevaluated until the customer writes again, possibly never.
That is unacceptable for a chat assistant, so this variant adds one mechanism:

```
acquired := ListenTurnTryLock(aicallID, interval)
if acquired { go RunListenTurn(aicallID); return }
promListenTurnTotal(kind=conversation, "skipped_locked")
// NEW: a deferred flush. The marker is per-process and ADVISORY (see "Bounds" below).
if flushScheduled.SetIfAbsent(aicallID) {           // in-process sync.Map
    h.afterFunc(interval + jitter(0..flush_jitter_ms), func() {   // injectable seam, defaults to time.AfterFunc
        flushScheduled.Delete(aicallID)             // INVARIANT: Delete BEFORE TryLock (see below)
        if ok, _ := ListenTurnTryLock(aicallID, interval); ok {
            RunListenTurn(detachedCtx, aicallID)     // pops whatever is pending; skipped_empty if nothing
            promListenFlushTotal("ran")
        } else {
            promListenFlushTotal("skipped_locked")
        }
    })
} else {
    promListenFlushTotal("skipped_scheduled")
}
```

**Bounds, derived from Redis, not from the process (rev 2, review round 1 HIGH-2).** Because the two
replicas are competing consumers on one queue (§5.3.2), a burst of N incoming messages for one AIcall is
split between them, and each replica may hold its own timer for the same AIcall. The in-process marker
therefore only de-duplicates timers *within* a replica; it is not the correctness bound. The bounds that
hold are:

- **At most one LLM turn per `interval` per AIcall, platform-wide.** Every turn, whether started from
  intake or from a timer on either replica, must first win `SET ai:listen:lock:<id> NX EX interval`. Two
  timers firing together on two replicas: one wins, the other meters `skipped_locked`. This is the same
  Redis-only argument CL §5.3.4 makes ("works across replicas").
- **No line is lost.** Lines live in `ai:listen:pending:<id>` (Redis, 6h TTL) until a turn pops them.
  A timer that fires and loses the lock leaves them in place; the winning turn pops them. A timer
  lost to a pod restart leaves them in place for the next incoming message, whichever replica gets
  it. Worst case is delay, never loss, until the 6h buffer TTL (identical to CL).
- **Timer count per AIcall is at most one per replica**, i.e. two, and each fires exactly once. The
  marker exists only to stop a burst of N messages on one replica from arming N timers.

**Timing.** The lock is set at some T0 <= T (T = the time this event was handled) and expires at
T0 + interval. The timer fires at T + interval + jitter >= T0 + interval, so it can never fire while
the lock that caused the `skipped_locked` is still valid. Jitter (`aicall_listen_conversation_flush_jitter_ms`,
§5.12) spreads the two replicas' timers so they do not race the lock at the same instant.

**Invariant: `flushScheduled.Delete` runs before `ListenTurnTryLock`, never after the turn.** If the
marker were cleared after the turn, an incoming message that arrives *during* the flush turn would find
the marker set, meter `skipped_scheduled`, arm nothing, and then wait for the next inbound line, which is
exactly the gap this mechanism exists to close. With `Delete` first, that message either wins the lock
itself (the timer's `TryLock` then loses, `skipped_locked`) or arms a fresh timer. Pinned by a test (§7 item 3).

**Overlapping turns, stated.** The flush is engineered to fire at lock expiry, and a listen turn may run
for up to `defaultListenTurnTimeout` = 60s (`listen.go:235`) against a default `interval` of 20s
(`config/main.go:100`). A flush turn can therefore start while the previous turn is still running, and
both may call `notify_agent` on overlapping context. CL already has this exposure on the regular path
(any new segment after lock expiry starts a turn regardless of a still-running one); the flush does not
create it, it only makes it more likely for chat bursts. The prompt's "never repeat a notification" rule
is the mitigation, as in CL; a per-AIcall "turn in flight" guard is deliberately not added here because it
would reintroduce the tail-loss the flush fixes. Recorded in §6 and §10.

**Not a scheduler.** The timer holds nothing but an AIcall id. No Redis keyspace notifications, no cron.

**Idempotent with the ordinary path, and free when empty (rev 3, review round 2 M-3).** A timer often
fires after a regular turn has already consumed the lines. Today `RunListenTurn` increments the
per-AIcall turn counter (`ListenTurnCountIncr`, `listen.go:278`) **before** it pops
(`ListenPendingPopAll`, `listen.go:287`), so an empty turn would still burn one of
`aicall_listen_max_turns_per_aicall` (60) and, at the cap, end the session. For the conversation kind
`RunListenTurn` therefore checks `cache.ListenPendingLen(aicallID)` (`LLEN`, new primitive, §5.8)
**first**, right after the flag/predicate gates and before the Case RPC and the counter: `0 ->
skipped_empty, return`. The check is non-atomic with the later pop; a line arriving in between only
means that one turn is counted normally, which is the existing behaviour. The call kind keeps CL's
order untouched (its turns are only ever started by a real segment, never by a timer).

`afterFunc` is a struct-field seam (like `runListenStartHook`, `aicallhandler/main.go:146-153`) so tests
can fire it deterministically without sleeping.

The flush is conversation-only. The call path keeps its hangup flush and is not touched.

### 5.5 Layer 2 - the evaluation turn

#### 5.5.1 Predicate generalisation

`RunListenTurn` (`listen.go:268-273`) currently requires `listenTranscribeIDFromMetadata(c) != uuid.Nil`.
It becomes `listenKind(c) != none` (§5.2.1).

The flag check at `listen.go:262-266` (`!AIcallListenEnabled -> stopListening, skipped_disabled`) stays as
the first gate for both kinds. Immediately after it, and **scoped to the conversation kind only**:

```
if listenKind(c) == listenKindConversation && !config.Get().AIcallListenConversationEnabled {
    h.stopListening(ctx, c)
    promListenTurnTotal(kind=conversation, "skipped_disabled")
    return
}
```

An unscoped check would stop call listens when only the conversation switch is off (review round 1
M-6). Everything else in `RunListenTurn` and `runListenTurnWithLines` (turn cap, drain, window read,
turn-id registration, throwaway pipecatcall, `TerminateWithDelay`) is unchanged.

#### 5.5.2 Turn-time Case status check (conversation only)

For `listenKind == conversation`, the turn order is: flag checks (§5.5.1) -> predicate -> **`ListenPendingLen`
empty short-circuit (§5.4)** -> **this Case check** -> turn counter (`listen.go:278`) -> `ListenPendingPopAll`
(`listen.go:287`) -> context build -> pipecatcall. The Case check therefore runs only when there is
something to evaluate, and before anything is counted or popped:

```
kase, err := reqHandler.ContactV1CaseGet(ctx, c.CustomerID, c.ReferenceID)   // requesthandler/contact_cases.go:173
if err != nil            { log; promListenTurnTotal(kind=conversation, "failed"); return }
if kase.Status == closed { stopListening(c); promListenTurnTotal(kind=conversation, "skipped_case_closed"); return }
```

Placement matters (review round 1 M-3): because the RPC runs before anything is popped, a transient
`ContactV1CaseGet` failure loses nothing. The pending list is untouched, the turn counter is not
incremented, and the next incoming message (or a deferred flush) retries the whole turn. The failure is
metered so a sustained rate is visible.

One RPC per turn, against an LLM call it precedes; negligible. This is the primary stop signal for a
conversation listen because contact-manager publishes nothing on `Close` (`casehandler/lifecycle.go:67`,
`models/kase/event.go` has only tag/contact events). Reopen (`lifecycle.go:126 Continue`) is likewise
silent; it is covered by the panel re-issuing the trigger on its next open (§5.1), which is exactly the
existing "repeated panel opens are free" path.

The call variant does not get this check: its stop signal is `call_hangup`, and adding an RPC there is
out of scope.

#### 5.5.3 Context assembly

`buildListenTurnMessages` (`listen.go:80-166`) is parameterised on `listenKind` in two places only:

1. The mechanics prompt: `ListenTurnSystemPrompt` for calls, **`ListenTurnConversationSystemPrompt`** for
   conversations. Same structure and the same CRITICAL RULES block as `main.go:315-329`; the differences
   are the framing sentences:

   ```
   You are silently monitoring a live messaging conversation between a human agent and a customer
   (SMS, chat, or similar). You are NOT talking to anyone right now.

   Below you will see a rolling window of the messages exchanged so far, tagged by sender. Lines after
   the "--- NEW SINCE YOUR LAST CHECK ---" marker are what you have not evaluated yet; everything
   before it you have already considered on a previous check. Lines tagged [AGENT] were written by the
   agent you are assisting (or an automated reply on their behalf); never alert the agent about their
   own messages.
   ```
   followed by the unchanged task list and rules (notify_agent-only, silence is the expected outcome,
   never repeat, do not summarise, do not use other tools unless required).

2. The transcript block header (`buildListenTranscriptBlock`, `listen.go:169`): `"Live call transcript so far:"`
   becomes `"Conversation so far:"` for the conversation kind. The `--- NEW SINCE YOUR LAST CHECK ---`
   marker is shared.

The Insight base prompt, the prompt snapshot (customer's `init_prompt`, message #2), and the Q&A replay
budget are untouched. The customer's conditions therefore apply to both channels without any
per-channel configuration, which is the same product decision CL §5.4 made.

#### 5.5.4 Tool handling

Unchanged. Listen-turn membership is resolved from `ai:listen:turnpcid:<aicall_id>` (`tool.go:52-68`),
`notify_agent` refuses non-listen turns (`tool_insight.go:1141-1160`), mechanical rows are tagged
`Origin=listen_internal`, the notification row is `Origin=proactive`. `get_conversation_content` remains
callable from a listen turn (CL §6 allows other tools "if answering genuinely requires it"); with the
window already in context the prompt discourages it.

### 5.6 Proactive message storage, webhook, panel

Unchanged (CL §5.6). Nothing here is channel-specific.

### 5.7 Lifecycle and cleanup

| Path | Call (CL) | Conversation |
|---|---|---|
| `ProcessTerminate` (`process.go:63-65`) | `ReferenceType == ContactCase && ListenCallID != uuid.Nil` | `ReferenceType == ContactCase && listenKind(tmp) != none` (the `ReferenceType` guard is kept; only the second clause widens) |
| Flag off at next turn (`listen.go:262-266`, plus the scoped sub-flag check §5.5.1) | `stopListening` | same |
| Turn cap (`listen.go:278-286`) | `stopListening` | same; **flush turns that find an empty buffer do not reach the counter** (§5.4, §5.5.2) |
| External end | `call_hangup` -> `stopListenByCallID` sweep (`listen.go:543`; guards `callID == uuid.Nil` at `:556`, so it can never touch a conversation listen) | **turn-time closed-Case check (§5.5.2)** |
| Idle expiry (**corrected in rev 2 and again in rev 3, review rounds 1 HIGH-1 / 2 HIGH-1**) | n/a (call ends) | `isAIcallIdleExpired` (`helpers.go:44-49`) is consulted only on the next Start/reuse. There are two `UpdateStatus(StatusTerminated)` sites: `start.go:282` inside `startReferenceTypeConversation` (`:245`), which only ever resolves `ReferenceTypeConversation` chatbot rows (`GetByReferenceID(conversationID)`, `:286`) and therefore can never hold a listen AIcall; and **`start.go:492` inside `startReferenceTypeContactCase`, the only reachable site**. `UpdateStatus` (`db.go:359-390`) never clears listen state. This design adds, at `:492` only, `if listenKind(existing) != none { h.stopListening(ctx, existing) }` immediately after the `UpdateStatus` call (`existing` is a full `*aicall.AIcall` from `GetByReferenceID`, metadata included; benefits the call kind too). Until that runs, an idle-expired listen is bounded by the 12h resolver TTL and by `RunListenTurn`'s `Status != Progressing` branch on the next inbound line, which calls `stopListening` |
| Redis loss | resolver gone until panel reopen | same |
| Key TTLs | buffer/lock/turns 6h, resolver 12h | same constants |

`stopListening` (`listen.go:421`): for `listenKind == conversation` there is no external session to
stop, so it goes straight to `clearListenState`.

`clearListenState` (`listen.go:462`): additionally `SREM ai:listen:conversation:<id> c.ID` when
`listen_conversation_id` is present, and strips `listen_conversation_id` from metadata alongside the
two existing keys. The `FieldListenCallID: uuid.Nil` write is harmless for a conversation listen and
stays.

A stray resolver entry whose AIcall has already been cleared (crash between `SREM` and DB write, or a
lost `SREM`) is bounded by: the 12h TTL, and `RunListenTurn`'s predicate + `stopListening` on the first
line that reaches it. Same guarantee CL §5.7 gives for the transcribe resolver.

### 5.8 Data model and plumbing scope

- `bin-contact-manager/models/kase/kase.go`: `ReferenceTypeConversationMessage = "conversation_message"`
  next to `ReferenceTypeCall` (`:109`), with the same value-pinning test pattern (`kase_test.go:171-177`).
- `bin-flow-manager/pkg/activeflowhandler/actionhandle.go:1341`: replace the bare literal with the constant
  (flow-manager already imports `bin-contact-manager` models for `ContactV1CaseCreate`).
- `bin-ai-manager/pkg/aicallhandler/start.go:492`: `stopListening` after the idle-expiry
  `UpdateStatus` in `startReferenceTypeContactCase` (§5.7; `:282` is the chatbot path and unreachable).
- `models/aicall/main.go`: one constant `MetaKeyListenConversationID = "listen_conversation_id"`.
- `pkg/cachehandler/listen.go`: `listenConversationKey`, `ListenConversationAIcallIDsGet|Add|Remove|IsMember`
  (copies of the transcribe trio at `:140-186` with the key swapped, plus `SISMEMBER`), and
  `ListenPendingLen` (`LLEN ai:listen:pending:<id>`, §5.4); interface + mock regeneration.
- `pkg/aicallhandler`: `listen_conversation.go` (intake, line builder, deferred flush,
  `checkListenEligibleConversation`), `listenKind` helper in `listen.go`, prompt constant in `main.go`,
  `afterFunc` seam + `flushScheduled` map on `aicallHandler`.
- `pkg/subscribehandler`: `conversationmanager.go` (new), `main.go` binding + switch, golden test.
- `internal/config/main.go`: §5.12 flags, `SetXxxForTest` helpers, defaults test.
- `bin-ai-manager/docs/operations.md`: flag table.
- `bin-conversation-manager/models/conversation/metadata.go:11-12`: correct the stale comment that says
  contact-manager writes `ContactCaseID` (the only writer is `bin-api-manager/pkg/servicehandler/case_message.go:191`).
  Comment-only; included because this design's §5.2 reasoning would otherwise appear to contradict it.
- **No** DB migration, **no** api-manager change, **no** common-handler RPC client change, **no** frontend change.

### 5.9 Speaker mapping

Structural. `Message.Direction` is authored by the channel handler that created the row, not inferred
from media legs, so the empirical verification the call variant needs (VOIP-1461, CL §5.9) does not
apply. The only ambiguity is `DirectionNond` (`""`), tagged `[SPEAKER]` and never a trigger.

### 5.10 Frontend

No change (§3). One observable difference is worth a release note: on a conversation Case the panel's
first proactive note can arrive while the agent is reading the same message in `CaseLinkedContent`
(`square-talk/.../CaseLinkedContent.jsx:33`). That is the intended experience for a multi-chat agent.

### 5.11 Cost and concurrency bounds

| Bound | Value | Source |
|---|---|---|
| LLM turns per AIcall | <= 1 per `aicall_listen_evaluate_interval_seconds`, <= `aicall_listen_max_turns_per_aicall` total | lock + turn counter (unchanged) |
| Deferred flush timers | <= 1 per AIcall per replica; advisory, not a correctness bound | `flushScheduled` marker (§5.4) |
| Counted turns per AIcall | <= `aicall_listen_max_turns_per_aicall`; empty flush turns are not counted | `ListenPendingLen` short-circuit (§5.4) |
| Concurrent turns per AIcall | up to 2 during the overlap window (turn timeout 60s > interval 20s) | stated exposure, §5.4 |
| Context size | constant: window lines x (max_message_chars + tag) + Q&A rows | §5.3.3 truncation |
| Intake per platform message | 1 delivery + 1 unmarshal + 1 `SMEMBERS`, once platform-wide (competing consumers) | §5.3.2 |
| RPC per turn | 1 (`ContactV1CaseGet`) | §5.5.2 |
| Billed media | none | no STT |

### 5.12 New configuration

All in `internal/config/main.go`, env-mapped, with `SetXxxForTest` helpers, and documented in
`docs/operations.md`. Existing `aicall_listen_*` flags apply to both kinds unchanged.

| Flag / env | Default | Purpose |
|---|---|---|
| `aicall_listen_conversation_enabled` / `AICALL_LISTEN_CONVERSATION_ENABLED` | `false` | variant switch, evaluated after the master `aicall_listen_enabled` and only on the conversation branch. Off: trigger step 5b returns `skipped_disabled`; a running conversation listen stops at its next turn via the scoped check in §5.5.1 (`skipped_disabled`). Call listens are unaffected by this flag in both places |
| `aicall_listen_conversation_max_message_chars` | `2000` | per-line truncation before buffering (§5.3.3) |
| `aicall_listen_conversation_flush_jitter_ms` | `1000` | upper bound of the random jitter added to the deferred flush delay (§5.4); reduces replica timer collisions |

The deferred flush delay itself is `aicall_listen_evaluate_interval_seconds` and is deliberately not a
separate flag: a flush that fires before the lock can expire is pure waste.

### 5.13 New metrics

`Name:` values only; `ai_manager_` namespace is implicit (CL §5.13).

**Strategy (unified in rev 2, review round 1 L-13; value set completed in rev 4, review round 3 M-1):
one `kind` label, values `call | conversation | unknown`, on every per-AIcall listen counter; separate
`_conversation_` vecs only for sources that have no call counterpart.** `unknown` is emitted by every
site that fires before the Case's reference type is known or before a listen pointer exists: the
trigger's steps 1-4 gates (`listen_trigger.go:172,176,234,241,248`, i.e. `failed` on the AI lookup,
`skipped_not_listenable` on type/liveness/AIcall-reference-type, `failed`/`skipped_not_listenable` on
the Case lookup and tenant check), the step-5 "neither `call` nor `conversation_message`" arm
(`listen_trigger.go:254`, which belongs to no branch), and `RunListenTurn`'s two `skipped_invalid`
sites (`listen.go:253`, `h.Get` failed; `listen.go:272`, predicate failed, which by definition means
`listenKind(c) == none`). One pre-step-5 site is **not** `unknown`: step 3's `reused`
(`listen_trigger.go:217`) can only fire when `listen_transcribe_id` is set, so it emits `call`.
Everywhere else the kind is known from the branch taken (trigger), from `listenKind(c)` (turn,
notify, terminate), or from the intake path itself (transcript intake `listen.go:720` `skipped_locked`
is `call`; §5.3.2/§5.4 intake and flush sites are `conversation`). Rule: `listenKind == none` maps to the
label value `unknown`. A dashboard splitting by kind therefore sees pre-branch rejections under
`unknown` rather than misattributed to a kind. The feature is dark and no Grafana dashboard references `aicall_listen_*` yet
(grep of `monorepo-monitoring` and `monorepo-etc` on 2026-09-05 returned nothing), so relabelling costs
only mechanical edits, enumerated (review round 2 finding 5): every `promListenStartTotal` /
`promListenTurnTotal` `.WithLabelValues(...)` site in `pkg/aicallhandler/listen.go` and
`listen_trigger.go`; `promListenNotifyTotal`, today a plain `prometheus.NewCounter`
(`metrics_listen.go:55`) whose single `Inc()` is at `tool_insight.go:1179`, converted to a
`CounterVec{kind}`; the `aicall_listen_start_total` Help string at `metrics_listen.go:24`, which
currently says "no new CounterVec" and enumerates the results; and the label-arity pins in
`metrics_listen_test.go:22-25`, `listen_test.go:318-320,803-812` and
`listen_trigger_test.go:1597-1610,1734-1738`. In return a dashboard can split every stage by kind with
one selector.

| Metric | Labels | Change |
|---|---|---|
| `aicall_listen_start_total` (existing) | `kind`, `result` | `kind` label added; `result` gains `skipped_disabled` (conversation sub-flag off, §5.1). Existing call sites pass `kind="call"` |
| `aicall_listen_turn_total` (existing) | `kind`, `result` | `kind` label added; `result` gains `skipped_case_closed` (§5.5.2) |
| `aicall_listen_notify_total` (existing) | `kind` | `kind` label added |
| `aicall_listen_segment_total` (existing, transcript) | `result` | unchanged; transcript-only by nature |
| `aicall_listen_conversation_segment_total` (**new**) | `result` = buffered / dropped_deleted / dropped_empty / dropped_unknown / dropped_tenant_mismatch / failed | §5.3.2 intake. `dropped_unknown` dominates by design; `dropped_tenant_mismatch` must stay at zero; `failed` is the `h.Get` error at step 4 |
| `aicall_listen_conversation_flush_total` (**new**) | `result` = ran / skipped_locked / skipped_scheduled | §5.4 deferred flush. `ran` means "the timer won the lock and invoked `RunListenTurn`"; whether that turn evaluated anything is in `aicall_listen_turn_total{kind="conversation"}` (`ran` vs `skipped_empty`), so the useful flush signal is `flush{ran}` minus `turn{skipped_empty}` over the same window = tails actually rescued by the timer. A high `skipped_scheduled` rate means bursts are frequent relative to `interval` |
| `aicall_listen_stop_failed_total`, `aicall_foreign_pipecatcall_dropped_total`, `aicall_listen_membership_check_failed_total` | unchanged | not kind-specific (stop RPC failures are call-only; the other two are per-turn mechanics) |

`ai_manager_aicall_listen_turn_total{kind="conversation",result="skipped_locked"}` against
`{result="ran"}` remains the direct read on debounce savings, per kind.

## 6. Error handling and edge cases

| Case | Handling |
|---|---|
| Case `ReferenceID` empty or non-UUID | `skipped_not_listenable` at step 5 (guard reused from `listen_trigger.go:257-260`) |
| Conversation deleted while listening | messages stop arriving; idle expiry or panel close ends the AIcall; resolver entry expires in 12h |
| Message with only media | one `[CUSTOMER] [media: image]` line; a customer sending a photo can be a legitimate trigger ("customer sent a document") |
| Very long email body | truncated to `max_message_chars` with `[truncated]` suffix; the Insight tool `get_conversation_content` remains available for the full text on demand |
| Chatbot AIcall and Insight AIcall on the same conversation | chatbot output is `outgoing` -> `[AGENT]` context only; never triggers (§5.3.3) |
| Two agents' AIcalls on one conversation | resolver SET fans out; each AIcall has its own buffer, lock and turn budget; each agent gets their own notes (same as CL §5.2.4) |
| Case closed between trigger and first message | first turn's `ContactV1CaseGet` sees `closed` -> `stopListening` |
| `ContactV1CaseGet` transient failure at turn time | `failed`; the RPC runs before `ListenPendingPopAll` (§5.5.2), so nothing is popped and nothing is lost; the next incoming message or deferred flush retries the whole turn |
| Out-of-order delivery (per-event goroutines, competing consumers) | window may hold two nearby lines swapped; best-effort ordering, stated in §5.3.2; no reordering attempted |
| Overlapping turns (flush at lock expiry while a 60s turn is still running) | both may `notify_agent`; prompt's "never repeat" rule is the mitigation; exposure stated in §5.4, same as CL's regular path |
| Tenant mismatch between event and resolved AIcall | `dropped_tenant_mismatch` + warn; never buffered |
| Redis down at intake | `SMEMBERS` error -> `dropped_unknown` + warn; no crash, no retry (hot path) |
| Idle-expired AIcall reclaimed on next Start/reuse | `UpdateStatus(Terminated)` followed by the new `stopListening` call (§5.7); resolver `SREM`, keys cleared |
| Closed Case, only outgoing lines ever arrive afterwards | Outgoing lines never start a turn (§5.3.3), so no turn-time Case check runs and the pending list grows (`ListenPendingPush` is `RPUSH`+`EXPIRE` with no trim, `cachehandler/listen.go:187-192`; only the window is `LTRIM`med, `:222-227`). Bounded by: 6h buffer TTL; 12h resolver TTL (refreshed only by a panel open, never by traffic); idle expiry at `start.go:492`; `ProcessTerminate`; and the next incoming line, whose turn runs the Case check and stops the session. A later drain pops at most `listenPendingPopMax` = 500 lines per turn (`:38`). Zero LLM or billing exposure while it accumulates |
| `aicall_listen_enabled` or `_conversation_enabled` flipped off | next turn stops the session (`skipped_disabled`) and clears state |
| Pod restart with a pending deferred flush | timer lost; lines wait for the next incoming message; stated degradation |
| Duplicate `message_created` delivery (AMQP redelivery) | duplicate line in the window; the LLM sees it twice; no dedupe, matching CL's handling of duplicate transcript segments |
| Message `TMDelete != nil` on the created event | dropped (defensive; not produced today) |

## 7. Testing strategy

Go unit tests with gomock, following the `listen_test.go` / `listen_trigger_test.go` patterns
(`cachehandler` mock, `reqHandler` mock, `config.SetXxxForTest`). Coverage target: every new function
and every row of §6.

1. **Trigger**: table test over `kase.ReferenceType` in {`call`, `conversation_message`, `""`, other} x
   `ReferenceID` in {valid, `""`, garbage}; asserts which branch runs and which `promListen*StartTotal`
   result is incremented. Idempotency: metadata present + `SISMEMBER` true -> `reused` with no writes;
   metadata present + `SISMEMBER` false -> re-`SADD`. Rollback: DB write error -> `SREM` called.
   Both flags off/on matrix.
2. **Intake**: `EventCVMessageCreated` with incoming/outgoing/nond, empty text with and without media,
   `TMDelete` set, subject present, text over the char cap, two AIcalls in the resolver; asserts exact
   line strings and that outgoing never calls `ListenTurnTryLock`.
3. **Deferred flush**: `afterFunc` seam captures the callback; test asserts one scheduling per AIcall
   while a marker is held, that firing it retries the lock, runs the turn when acquired, and clears the
   marker; that a second `skipped_locked` while scheduled meters `skipped_scheduled`; **and the
   `Delete`-before-`TryLock` invariant**: an intake call made from inside the captured callback (simulating
   a message arriving mid-flush) must be able to arm a new timer. Also: timer fires, lock held by the
   other replica (mock `SetNX` false) -> `skipped_locked`, pending list untouched.
4. **Turn**: `RunListenTurn` with `listen_conversation_id` metadata: closed Case -> `stopListening` +
   `skipped_case_closed`; `ContactV1CaseGet` error -> `failed` **and `ListenPendingPopAll` / turn counter
   never called**; open Case -> proceeds and the built messages contain
   `ListenTurnConversationSystemPrompt` and `"Conversation so far:"`; call-kind AIcall never calls
   `ContactV1CaseGet`; conversation sub-flag off stops a conversation listen but a call-kind AIcall in the
   same test table proceeds.
5. **Stop paths**: `ProcessTerminate` with only `listen_conversation_id` set calls `stopListening`, and
   with `ReferenceType != contact_case` does not; `clearListenState` `SREM`s the conversation key and
   strips the metadata key; **idle-expiry at `start.go:492` (`startReferenceTypeContactCase`) calls
   `stopListening` for a listening row and not for a non-listening one**; call-kind behaviour unchanged
   (existing tests must stay green).
5c. **Idempotency inside `startListenConversation`**: pointer set + `SISMEMBER` true -> `reused`, no
   `SADD`, no DB write; pointer set + `SISMEMBER` false -> `SADD` + metadata rewrite, `started`; pointer
   absent -> `SADD` + write, `started`; `SADD` error -> `failed`, no DB write; DB error -> `SREM`, `failed`.
   And `checkListenEligible` returns `proceed=false` on the conversation branch so `ProcessListen` does
   not spawn `runListenStart` (assert the `runListenStartHook` seam is never invoked).
5d. **Flush empty short-circuit**: `RunListenTurn` on a conversation-kind AIcall with `ListenPendingLen == 0`
   returns `skipped_empty` without calling `ContactV1CaseGet` or `ListenTurnCountIncr`.
5b. **Tenant**: intake with `evt.CustomerID != c.CustomerID` meters `dropped_tenant_mismatch` and pushes
   nothing.
6. **Binding golden**: `binding_golden_test.go` expects 13 patterns with the new one last.
7. **Config**: `Test_ListenConfigDefaults` extended with the three new defaults.
8. **Metrics**: `metrics_listen_test.go` registration test extended.
9. **Golden routing-key cross-check**: a test in `bin-ai-manager` asserting the bound pattern equals
   `eventtopic.PatternForEventType("conversation-manager", "conversation_message_created")` and that
   a sample `message.Message` routing key matches it, so a future rename of the event type fails
   here rather than in production.

Manual verification (sandbox or staging, no cost): open a webchat conversation, create a Case via the
`case_create` flow action, open the Insight panel in square-talk, send two customer messages 5s apart,
observe one note after the interval (deferred flush) and none for an agent reply. Then close the Case,
send another message, observe no turn (`skipped_case_closed`).

## 8. Rollout

1. Merge with both flags default `false`. Deployable in one step; no migration.
2. Staging: `AICALL_LISTEN_ENABLED=true`, `AICALL_LISTEN_CONVERSATION_ENABLED=true`; run §7 manual
   verification; watch `aicall_listen_conversation_segment_total{result="dropped_unknown"}` rate as the
   fan-out cost readout and `aicall_listen_conversation_flush_total`.
3. Production: enable the conversation switch independently of the call switch. The call variant's
   VOIP-1461 gate does not apply here.
4. Rollback: flip `AICALL_LISTEN_CONVERSATION_ENABLED=false`; in-flight sessions stop at their next
   evaluated turn and clear their own state. No data to clean up.

## 9. Impacted files (indicative)

| File | Change |
|---|---|
| `bin-ai-manager/models/aicall/main.go` | `MetaKeyListenConversationID` |
| `bin-ai-manager/pkg/cachehandler/listen.go`, `main.go`, `mock_main.go` | conversation resolver quartet (`Get|Add|Remove|IsMember`) + `ListenPendingLen` + interface |
| `bin-ai-manager/pkg/aicallhandler/listen_conversation.go` (new) | intake, line builder, deferred flush, conversation eligibility/start |
| `bin-ai-manager/pkg/aicallhandler/listen.go` | `listenKind`, predicate, scoped sub-flag check, `ListenPendingLen` empty short-circuit, Case check, `stopListening`/`clearListenState` branches, header string, `kind` on every metric site |
| `bin-ai-manager/pkg/aicallhandler/listen_trigger.go` | step 5 branch into `checkListenEligibleConversation` (step 5b sub-flag, `startListenConversation`, `proceed=false`), `kind="unknown"` on the steps 1-4 metric sites |
| `bin-ai-manager/pkg/aicallhandler/process.go` | terminate gate (second clause only) |
| `bin-ai-manager/pkg/aicallhandler/start.go` | `stopListening` after idle-expiry `UpdateStatus` at `:492` only |
| `bin-ai-manager/pkg/aicallhandler/tool_insight.go:1179` | `promListenNotifyTotal.Inc()` -> `.WithLabelValues(kind).Inc()` |
| `bin-ai-manager/pkg/aicallhandler/metrics_listen_test.go:22-25`, `listen_test.go:318-320,803-812`, `listen_trigger_test.go:1597-1610,1734-1738` | label-arity pins updated for the `kind` label |
| `bin-contact-manager/models/kase/kase.go`, `kase_test.go` | `ReferenceTypeConversationMessage` constant + value pin |
| `bin-flow-manager/pkg/activeflowhandler/actionhandle.go` | literal -> constant at `:1341` |
| `bin-ai-manager/pkg/aicallhandler/main.go` | prompt constant, `afterFunc` seam, `flushScheduled`, interface method `EventCVMessageCreated` |
| `bin-ai-manager/pkg/aicallhandler/metrics_listen.go` | `kind` label on `aicall_listen_start_total` and `aicall_listen_turn_total`; `aicall_listen_notify_total` converted from `Counter` (`:55`) to `CounterVec{kind}`; two new vecs (`_conversation_segment_`, `_conversation_flush_`); two new `result` values (`skipped_disabled`, `skipped_case_closed`); the `start_total` Help string at `:24` ("no new CounterVec", result list) rewritten |
| `bin-ai-manager/pkg/subscribehandler/main.go`, `conversationmanager.go` (new), `binding_golden_test.go` | binding + dispatch |
| `bin-ai-manager/internal/config/main.go`, `main_test.go` | three flags |
| `bin-ai-manager/docs/operations.md` | flag table |
| `bin-conversation-manager/models/conversation/metadata.go` | comment fix only |
| tests for each of the above | §7 |

## 10. Open items and optional follow-ups

1. **`case_closed` / `case_continued` events** from contact-manager (`PublishEvent`, the precedent at
   `casehandler/casenote.go:40`). Would let ai-manager stop instantly instead of at the next turn and
   would remove the per-turn RPC. Not required; separate ticket if wanted.
2. **Inbound email into conversation-manager.** Email-origin Cases are inert until then (§3).
3. **Dynamic per-conversation binding** if platform message volume ever makes the wildcard floor
   measurable (CL §3 escape hatch).
4. **Product question for the CEO/CTO:** should outgoing lines be shown to the LLM at all? This draft
   says yes (context, never trigger) so the AI can judge whether the agent already handled the
   customer's point. The alternative (customer lines only) halves the window's token cost but loses
   that judgement.
5. **Per-AIcall "turn in flight" guard** to remove the overlapping-turn duplicate-notification exposure
   (§5.4). Not adopted in this design because a naive guard reintroduces tail loss; a correct version
   needs a "turn finished, re-check pending" hook and belongs in a CL follow-up since the call path has
   the same exposure.

## 11. Review-response matrix (round 1 -> rev 2)

| # | Severity | Finding (abridged) | Resolution in rev 2 |
|---|---|---|---|
| 1 | HIGH | §5.7 idle row claimed `ProcessTerminate` clears state; idle expiry actually uses `UpdateStatus` (`start.go:282,492`) which never clears listen state | Row rewritten with the real path; rev 2 added `stopListening` after both `UpdateStatus` sites. **Superseded by §11.1 row 1 (rev 3): only `start.go:492` is reachable for a listen AIcall; `:282` is the chatbot path and is NOT edited.** Honest TTL + next-line bound stated for the interim |
| 2 | HIGH | Replica model wrong: shared `QueueNameAISubscribe`, competing consumers, not per-replica copies | §5.3.2 delivery model corrected; §5.4 bounds re-derived from the Redis lock and pending list, marker declared advisory; §5.11 rows corrected |
| 3 | MEDIUM | §6 contradicted §5.5.2 on `ContactV1CaseGet` failure (nothing is popped before the RPC) | §5.5.2 states placement before `ListenPendingPopAll` and the no-loss consequence; §6 row replaced; test §7 item 4 pins it |
| 4 | MEDIUM | No `conversation_message` constant; third bare literal would be added | `kmkase.ReferenceTypeConversationMessage` added, flow-manager literal converted in the same PR (§5.1, §5.8, §9) |
| 5 | MEDIUM | §5.12 (`skipped_not_listenable`) vs §5.13 (`skipped_disabled`) on sub-flag off | Unified to `skipped_disabled` everywhere (§5.1, §5.12, §5.13) |
| 6 | MEDIUM | Turn-time sub-flag check unscoped would stop call listens | §5.5.1 specifies `listenKind == conversation && !ConversationEnabled`; test §7 item 4 covers a call-kind row in the same table |
| 7 | MEDIUM | Event processing unordered, unstated | §5.3.2 "Ordering is best-effort" paragraph; §6 row |
| 8 | MEDIUM | `Delete`-before-`TryLock` ordering load-bearing but unstated | §5.4 invariant paragraph with the failure it prevents; test §7 item 3 |
| 9 | MEDIUM | Flush at lock expiry can overlap a still-running 60s turn; duplicate `notify_agent` exposure | §5.4 "Overlapping turns" paragraph; §5.11 row; §6 row; §10 item 5 records the guard as a follow-up with the reason it is not adopted |
| 10 | LOW | False claim that ai-manager does not import `cvmessage` | Removed; §5.3.1 cites the existing import (**line corrected to `tool_insight.go:21` in rev 3, §11.1 row 6**) and adds the `ServiceNameConversationManager` / global-topic-publish confirmations |
| 11 | LOW | Terminate gate rewrite dropped the `ReferenceType` guard | §5.7 row keeps `ReferenceType == ContactCase &&` and widens only the second clause; test §7 item 5 |
| 12 | LOW | Six citations off by a few lines | `process.go:63-65`, `email.go:99`, `cachehandler/listen.go:351-357`, `serviceagent_aicall.go:140-174`, `listen.go:276-285`, `tool.go:52-68` corrected |
| 13 | LOW | Metric split inconsistent; malformed table row | §5.13 rewritten: `kind` label on `start`/`turn`/`notify`, new `_conversation_segment_` and `_conversation_flush_` vecs, table fixed |
| 14 | LOW | No tenant assertion at intake | §5.3.2 step 4 asserts `evt.CustomerID == c.CustomerID`, `dropped_tenant_mismatch` result; §6 row; test §7 item 5b |

### 11.1 Review-response matrix (round 2 -> rev 3)

| # | Severity | Finding (abridged) | Resolution in rev 3 |
|---|---|---|---|
| 1 | HIGH | `start.go:282` is inside `startReferenceTypeConversation` (chatbot rows only); a listen AIcall can never be its `res`, so half the HIGH-1 fix was dead | §5.7 idle row, §5.8, §9 and §7 item 5 scoped to `start.go:492` (`startReferenceTypeContactCase`), with the reason `:282` is unreachable recorded |
| 2 | HIGH | Conversation idempotency referenced a `conversationID` not yet resolved at step 3; §5.1.1 had no `reused` path; no way to stop `ProcessListen` spawning `runListenStart` | §5.1 control-flow paragraph: step 3 left untouched (no-op for conversations), idempotency moved into `startListenConversation` with a new `ListenConversationAIcallIDIsMember` primitive, branch returns `proceed=false`; §5.1.1 pseudocode rewritten; tests §7 item 5c |
| 3 | MEDIUM | Empty flush turns burn the 60-turn cap (counter increments before pop) and cost a Case RPC | `ListenPendingLen` short-circuit before the Case RPC and the counter, conversation kind only (§5.4, §5.5.2 order, §5.7, §5.8, §5.11, test §7 item 5d) |
| 4 | MEDIUM | §9 "three new vecs, one new result" contradicted §5.13 | §9 row rewritten to match §5.13 exactly |
| 5 | MEDIUM | Metrics blast radius understated: `notify` is a plain Counter used in `tool_insight.go:1179`, Help string, test pins | §5.13 enumerates all sites; §9 rows added |
| 6 | LOW | `tool_insight.go:7` is `"fmt"` | `:21` |
| 7 | LOW | Residual off-by-one citations | `start.go:282`, `listen.go:278-286`, `listen.go:287`, `cachehandler/listen.go:347-355/356`, `listen.go:556` corrected |
| 8 | LOW | §4 diagram missing the tenant assertion | line added |
| 9 | LOW | `h.Get` at intake is cache-first with DB fallback; "so <= 2" hardcodes replica count | §5.3.2 wording corrected with `dbhandler/aicall.go:111-125`; §5.11 row reworded per replica |

### 11.2 Review-response matrix (round 3 -> rev 4)

| # | Severity | Finding (abridged) | Resolution in rev 4 |
|---|---|---|---|
| 1 | MEDIUM | `kind` label undefined for the steps 1-4 trigger sites and `RunListenTurn`'s `skipped_invalid` on `Get` failure | §5.13: `kind` value set is `call | conversation | unknown`; the pre-branch sites are enumerated and emit `unknown`; §9 trigger row |
| 2 | MEDIUM | §9 still said "trio", omitted `ListenPendingLen`, said "step 1b" | §9 cachehandler, `listen.go` and `listen_trigger.go` rows rewritten against §5.1/§5.4/§5.8 |
| 3 | LOW | §4 put the debounce outside the per-AIcall fan-out; §5.3.2 step 5 was top-level | §4 diagram and §5.3.2 step 5 both inside the per-AIcall loop, lock keyed by that AIcall id |
| 4 | LOW | Flush `ran` ambiguous after the empty short-circuit | §5.13 flush row explains `ran` vs turn `skipped_empty` and the derived "tails rescued" signal |
| 5 | LOW | §11 row 1 still said "both sites" | Row annotated as superseded by §11.1 row 1 |
| 6 | LOW | Media type list incomplete (`imagemap`, `flex`), citation short | §5.3.3: raw `media.Type` passed through, all nine listed, `:26-34` |
| 7 | LOW | Switch condition inlined the service name instead of the file's publisher const style | §5.3.1: `publisherConversationManager` constant added to the block at `main.go:33-37` |

### 11.3 Review-response matrix (round 4 -> rev 5)

| # | Severity | Finding (abridged) | Resolution in rev 5 |
|---|---|---|---|
| 1 | LOW | Three metric sites unclassified: `listen_trigger.go:254` (no-branch arm), `listen.go:272` (predicate `skipped_invalid`), `listen_trigger.go:217` (`reused` at step 3) | §5.13: first two -> `unknown`, third -> `call` with the reason |
| 2 | LOW | Pending list is `RPUSH`+`EXPIRE` only (no trim); §4 implied both lists are trimmed; closed-Case + outgoing-only path accumulates | §4 diagram corrected; §6 row added with all five bounds and the 500-line pop cap |
| 3 | LOW | §5.3.2 snippet returned a value from a no-return handler | Wrapper returns `nil` (`error` shape kept), handler declared without a return, mirroring `EventTMTranscriptCreated` |
| 4 | LOW | §11 row 10 still cited `tool_insight.go:7` | Row annotated with the rev 3 correction |

### 11.4 Review-response matrix (round 5 -> rev 6, folded without re-review per the round 5 verdict)

| # | Severity | Finding (abridged) | Resolution in rev 6 |
|---|---|---|---|
| 1 | LOW | §5.1 said `startListenConversation(ctx, c, kase)`; §5.1.1 passes `conversationID` | §5.1 now parses `kase.ReferenceID` and passes `conversationID` |
| 2 | LOW | `listen.go:720` (`skipped_locked` in transcript intake) not in the `kind` map; intake paths not named as a kind source | §5.13 names intake as a source: transcript intake = `call`, §5.3.2/§5.4 = `conversation` |
| 3 | LOW | `listenKind` returns `none`, label value is `unknown`, mapping unstated | §5.13 rule: `none -> unknown` |
| 4 | LOW | No result bucket for an `h.Get` error at intake step 4 | §5.3.2 step 4 `failed` exit; §5.13 segment result list gains `failed` |
| 5 | LOW | `processEvent*` `error`-return citation pointed at the dispatch switch | §5.3.2 cites `transcribemanager.go:24` as the exemplar, switch kept as the dispatch reference |
