# Webchat Widget: split single FlowID into SessionFlowID + MessageFlowID

- Ticket: NOJIRA
- Author: Hermes (CPO), for pchero (CEO/CTO)
- Status: DRAFT — locked after a 16-round adversarial review loop (analysis
  stage) against a background subagent; ready for implementation
- Supersedes: `2026-06-29-add-webchat-manager-design.md` (v8, merged to
  main as part of PR #1104/#1108/#1109/#1110/#1111) — this is v9 of that
  service's design history, in a new file because the original worktree
  is gone (its branch already merged).

## 1. Problem

`bin-webchat-manager`'s `Widget` currently exposes exactly one `FlowID`,
triggered once per Session at the Session's first inbound message
(`pkg/messagehandler/create.go`'s `triggerFirstMessageFlow`, gated by
`Session.ActiveflowID != uuid.Nil` under a per-session in-process lock,
`lockSession`/`unlockSession` in `pkg/messagehandler/main.go`). There is
no way to run a Flow at session start (before the visitor has typed
anything — e.g. to pre-route, greet, or set up state) independently from
a Flow that reacts to every subsequent message.

pchero directed: split this into two independently configurable Flows:
one that fires when a webchat session is created/started, and one that
fires whenever a message is received.

## 2. Decisions locked (CEO-directed, confirmed against real code across
   16 rounds of adversarial review)

1. **`Widget.FlowID` is renamed to `Widget.SessionFlowID`** — not
   remapped to `MessageFlowID`. The existing field's cardinality (fires
   once per Session) matches "SessionFlowID" conceptually; only the
   *trigger event* changes (see #2), not the field's identity.
2. **`SessionFlowID`'s trigger point moves to session creation/start**
   (`POST /webchat_sessions`), not the first inbound message. This
   deliberately reverses the v7 decision ("an empty Session with no
   message should not itself start an activeflow") — pchero directed
   this explicitly. Cost: a visitor who loads the widget and never types
   anything still causes one activeflow execution (and any billed
   actions inside it) per session-create call. Accepted, cost is on
   pchero's call — client-side responsibility to avoid spamming session
   creation (see §7).
3. **`Widget.MessageFlowID` is a brand-new, opt-in (nullable, default
   nil) field.** When set, `bin-webchat-manager`'s own
   `messagehandler.Create` triggers a new, independent, stateless
   activeflow on **every** inbound message — mirroring
   `bin-conversation-manager`'s existing `Account.MessageFlowID`/
   `Number.MessageFlowID` pattern for LINE/WhatsApp/SMS
   (`execute_mode.go`'s `runExecuteModeFlowLine/Message/WhatsApp` →
   `message.go`'s `executeActiveflow`) exactly. This is the *only*
   technically viable stateless-repeat-trigger shape: `bin-flow-manager`
   has no message-wait/gather-style action that could keep one
   activeflow alive across multiple inbound messages (confirmed by full
   enumeration of `bin-flow-manager/models/action/action.go`'s ~41
   action types — none exist).
4. **Ownership/ownership-gating of the *decision to run a bot vs. an
   agent* is NOT a webchat-manager concern.** Earlier drafts of this
   design added a webchat-local `Session.OwnerType`/`OwnerID` pair,
   mirroring `bin-conversation-manager`'s existing
   `Conversation.OwnerType`/`OwnerID` (assignable-conversation feature,
   merged 2026-04-30). Round 6 review caught that this would create TWO
   independent, unsynchronized owner stores for the same real-world
   concept (an agent taking over a webchat session) — an agent using
   square-talk's existing "assign this conversation to me" action would
   have no effect on webchat-manager's separate flag, and vice versa.
   **Rejected.** `Conversation.OwnerType` remains the single source of
   truth. See §5.
5. **Session creation now always creates a `Conversation`, eagerly, and
   `bin-conversation-manager` — not `bin-webchat-manager` — owns
   `SessionFlowID`'s Create+Execute.** This is the single biggest
   architecture change from the original v7/v8 design (where
   webchat-manager triggered its own Flow directly, "B안"). See §5 for
   the full mechanics and why.
6. **Cross-tenant flow-ownership validation (the fix from Round 4 of the
   original v6/v7 review, `bin-api-manager`'s `webchat_widget.go`) must
   be duplicated onto both `SessionFlowID` and `MessageFlowID`
   independently** in `WebchatWidgetCreate`/`WebchatWidgetUpdate`.
   Missing this on the new `MessageFlowID` field would silently
   reintroduce the exact cross-tenant flow-execution vector Round 4
   closed for the single `FlowID` field.
7. **Session-create RPC retries are a client concern, not a backend
   concern.** `POST /webchat_sessions` has no idempotency key; a
   client/gateway-level retry after a timeout will always create a
   distinct new `Session` + trigger a distinct new `SessionFlowID`
   execution. pchero directed this be explicitly accepted, not solved
   backend-side, in this PR.
8. **Widget soft-delete checks are NOT duplicated into
   webchat-manager/conversation-manager.** `bin-api-manager`'s
   `webchat_widget.go`'s `widgetGet` helper (mirrors `flowGet`'s
   established pattern) already rejects a soft-deleted Widget before
   `WebchatSessionCreate` ever issues the downstream RPC — consistent
   with the platform-wide "Auth/ownership checks belong ONLY in
   bin-api-manager" rule (`bin-api-manager/CLAUDE.md`). No new check
   needed in either backend service.

## 3. Architecture: who creates the Conversation, who triggers the Flow

### 3.1 Current (pre-this-design) state, for contrast

`bin-webchat-manager`'s `messagehandler.Create` triggers `Widget.FlowID`
directly (Create+Execute against `bin-flow-manager`, `ReferenceType=
webchat`, reference_id=`Session.ID`) on the session's first inbound
message only. `bin-conversation-manager` separately subscribes to
webchat-manager's `webchat_message_created` event and lazily
get-or-creates a `Conversation` (self=Widget.ID, peer=Session.ID) purely
for the unified cross-channel Conversation/Message timeline — and
**never** triggers a Flow for `TypeWebchat` conversations
(`event_webchat.go`'s explicit "B안" comment + a passing negative test
asserting no `FlowV1ActiveflowCreate` call). SMS/LINE/WhatsApp, by
contrast, have conversation-manager itself own both Create+Execute of
the Flow (`message.go`'s `executeActiveflow`), directly from
`runExecuteModeFlowLine/Message/WhatsApp`.

### 3.2 New flow (this design)

```
1. Visitor's browser: POST /auth/boot { direct_hash }              (unchanged)
2. Visitor's browser: POST /webchat_sessions?token=<direct-jwt>
   -> bin-api-manager's WebchatSessionCreate (unchanged): resolves
      Widget via direct-scope check, confirms not soft-deleted
      (widgetGet), derives ownerCustomerID = w.CustomerID, calls
      h.reqHandler.WebchatV1SessionCreate(ctx, ownerCustomerID, widgetID)
   -> bin-webchat-manager's sessionhandler.Create:
      a. Creates a brand-new Session row (fresh UUID every call — no
         get-or-create semantics, matches existing behavior).
      b. Fetches Widget once (h.db.WidgetGet) — this single fetch
         serves BOTH purposes: (i) read Widget.WelcomeMessage to return
         to the visitor, (ii) read Widget.SessionFlowID to decide
         whether to trigger anything.
      c. If Widget.SessionFlowID != uuid.Nil: calls a NEW
         bin-conversation-manager RPC (see §3.3) passing
         { customer_id: ownerCustomerID (the value bin-api-manager
           already verified — NOT re-fetched from Widget by
           sessionhandler, which never queries Widget for identity
           purposes, only for WelcomeMessage/SessionFlowID content),
           flow_id: w.SessionFlowID,
           self: {type: webchat, target: widget_id},
           peer: {type: webchat, target: session_id} }
      d. Records the returned activeflow_id onto Session.ActiveflowID
         (write-only marker, best-effort — Session creation itself has
         already succeeded and must not fail if this RPC fails).
   -> Response: { session_id, welcome_message } (welcome_message always
      populated regardless of SessionFlowID trigger success/failure —
      it's an in-memory copy of Widget.WelcomeMessage attached to the
      Session/WebhookMessage response, not a DB-persisted column; see §6).
3. bin-conversation-manager's new RPC handler:
   a. Creates a Conversation (self=Widget.ID, peer=Session.ID,
      customer_id=the trusted value passed in step 2c — no additional
      cross-check needed, see §2.8's precedent) — since peer always
      carries a fresh, unique Session.ID, this always creates a new
      Conversation row (not a get-or-create; the dedup key is
      unreachable by construction because Session.ID is always new).
   b. FlowV1ActiveflowCreate(ReferenceType=ReferenceTypeConversation,
      reference_id=Conversation.ID, ...) — NOT ReferenceTypeWebchat.
      This is the key shift: SessionFlowID's activeflow is now scoped
      to the Conversation, exactly like every other channel's Flow.
   c. FlowV1ActiveflowExecute(...).
   d. Publishes conversation_created (existing Create behavior,
      unchanged).
4. Visitor's first (and every subsequent) message:
   POST /webchat_sessions/{id}/messages?token=<direct-jwt> { text }
   -> bin-webchat-manager's messagehandler.Create:
      - Saves the Message (unchanged).
      - If Widget.MessageFlowID != uuid.Nil: triggers a brand-new,
        independent activeflow for THIS message
        (ReferenceType=ReferenceTypeWebchat, reference_id=Session.ID —
        unchanged shape from the old FlowID trigger, just gated on the
        new field and firing on every message instead of only the
        first). No lock, no "already triggered" check — this is
        deliberately stateless, matching SMS/LINE/WhatsApp's
        MessageFlowID exactly (see §2.3, §4 for the accepted overlap
        risk this implies).
   -> bin-conversation-manager's async subscriber
      (messageEventReceivedWebchat, triggered by the
      webchat_message_created event) calls GetOrCreateBySelfAndPeer
      with the SAME self/peer pair used in step 3a — this always
      resolves to a GET (never a CREATE) because step 3a already
      created that exact Conversation. Guaranteed by NormalizeTarget's
      handling of TypeWebchat addresses as opaque, unchanged
      identifiers (`commonaddress` normalize logic) — both call sites
      produce byte-identical self/peer values.
   -> This subscriber's getExecuteMode/runExecuteModeFlow path remains a
      structural no-op for TypeWebchat conversations (no AccountID ->
      no MessageFlowID-style trigger at the conversation-manager layer)
      — this is NOT related to bin-webchat-manager's own MessageFlowID
      trigger in the same step; they are two independent triggers in
      two different services, and this no-op exists solely to prevent
      THIS async subscriber from adding a second, redundant
      SessionFlowID-equivalent trigger on top of the one already fired
      at session-create time (step 3).
```

### 3.3 New conversation-manager RPC

A new, dedicated RPC — not an optional-parameter extension of the
existing `ConversationV1ConversationCreate` — per this repo's own
established pattern of separating "pure create" from "create with side
effects" (`ConversationV1ConversationCreate` vs.
`ConversationV1ConversationGetOrCreateBySelfAndPeer` are already two
distinct RPCs in `conversation_conversations.go`, with the latter's own
doc comment explaining exactly this kind of "distinct from X above"
reasoning). Extending the existing `Create` RPC's signature would force
recompilation/re-mocking of its three unrelated existing call sites
(`linehandler/hook.go`, `whatsapphandler/hook.go`, and their tests) for
zero benefit to them.

Working name: `ConversationV1ConversationCreateAndExecuteFlow`. Request
carries `customer_id`, `flow_id`, `self`, `peer` (and whatever
`FlowV1ActiveflowCreate` needs passed through, e.g. no extra Flow
variables for this trigger point — session_id/widget_id are
recoverable from the Conversation's own self/peer once the activeflow's
Flow references it, so nothing extra needs injecting here, unlike the
old first-message trigger which injected `voipbin.webchat.message.text`
because a message existed at that point and now doesn't).

## 4. Accepted risk: overlapping/concurrent MessageFlowID executions

With `lockSession`/`unlockSession` removed (see §5), nothing prevents two
inbound messages arriving close together from each triggering their own
independent activeflow concurrently. This is the same shape of risk
`bin-conversation-manager`'s existing SMS/LINE/WhatsApp
`executeActiveflow` already lives with today (no per-conversation lock
there either) — this design does not introduce a new *class* of risk,
only applies an existing, already-accepted platform risk to a channel
(webchat) with a higher realistic message frequency (rapid visitor
typing vs. SMS/LINE's naturally network-latency-spaced messages). This
difference in degree — not kind — must be called out explicitly when
requesting pchero's final sign-off, not left as an unstated assumption.
The real fix (flow-manager publishing a completion event that
webchat-manager could queue against) is out of scope — tracked as a
follow-up issue, not blocking this PR.

## 5. Code changes — bin-webchat-manager

1. `models/widget/{widget.go,field.go,webhook.go}`: rename `FlowID` →
   `SessionFlowID`; add new `MessageFlowID uuid.UUID`
   (`json:"message_flow_id,omitempty" db:"message_flow_id,uuid"`).
2. `pkg/widgethandler/{create.go,db.go}`: `Create`/`Update`/
   `UpdateBasicInfo` gain both fields (mirroring how `FlowID` was
   threaded through today); `FieldFlowID` becomes `FieldSessionFlowID` +
   new `FieldMessageFlowID`.
3. `pkg/listenhandler/v1_widgets.go` + `models/request/v1_widgets.go`:
   `req.FlowID` → `req.SessionFlowID` + `req.MessageFlowID`.
4. `pkg/sessionhandler/create.go`: add the Widget fetch (§3.2 step 2b/c)
   and the new conversation-manager RPC call (§3.2 step 2c/d). Add the
   in-memory `WelcomeMessage` attachment (§6).
5. `pkg/messagehandler/create.go`: **remove** `triggerFirstMessageFlow`
   entirely, remove the `sess.ActiveflowID != uuid.Nil` first-message
   gate, remove the `lockSession(sessionID)`/`defer unlockSession(...)`
   call. **Add** new MessageFlowID-based trigger: on every inbound
   message, if `Widget.MessageFlowID != uuid.Nil`, unconditionally
   `FlowV1ActiveflowCreate`+`Execute` (`ReferenceType=
   ReferenceTypeWebchat`, reference_id=`Session.ID`) — no gate, no
   "already triggered" bookkeeping.
6. `pkg/messagehandler/main.go`: remove `sessionLocksMu`, `sessionLocks`
   map field, and the `lockSession`/`unlockSession` functions entirely —
   nothing references them once step 5 lands.
7. `pkg/messagehandler/concurrency_test.go`: delete
   `Test_Create_ConcurrentFirstMessages_TriggersFlowExactlyOnce` — the
   mechanism it tests (lockSession) no longer exists, and the property
   it verified (first-message-only trigger) is no longer this field's
   behavior.
8. `pkg/messagehandler/create_test.go`: rewrite the "first message
   triggers Flow" / "no FlowID means no trigger" tests to instead cover
   "every message triggers MessageFlowID when set" / "no MessageFlowID
   means no trigger" / "MessageFlowID never gates on prior messages".
9. `models/session/session.go`/`webhook.go`: add a transient (DB-unbound)
   `WelcomeMessage string` field, tagged `db:"-"` — an established
   pattern in this codebase (`bin-agent-manager/models/agent/agent.go`'s
   `Addresses ... db:"-"` is the precedent), verified against the actual
   DB mapper (`bin-common-handler/pkg/databasehandler/mapping.go`'s
   `getDBFieldsRecursive`/`prepareFieldsFromStruct`/`buildScanTargets`
   all skip `tag == "-"`) — squirrel+custom-reflection based, not
   GORM/sqlx, but the skip convention holds regardless.

## 6. welcome_message wiring

`sessionhandler.Create`'s single `WidgetGet` call (§3.2 step 2b) reads
`Widget.WelcomeMessage` and attaches it, in-memory only, to the
`Session`/`WebhookMessage` struct instance being returned — no new
wrapper DTO, no change to `WebchatV1SessionCreate`'s existing
`var res wcsession.Session` parse path
(`bin-common-handler/pkg/requesthandler/webchat_session.go:17-41`,
confirmed: this function calls `parseResponse(tmp, &res)` directly into
`wcsession.Session`, unmodified by this design — adding a transient
field to that struct requires zero changes here), no change to
`ConvertWebhookMessage()`'s call-site signature (it gains one new field
copy internally). If the Widget fetch fails for any reason, Session
creation still succeeds (already-created row is not rolled back);
`welcome_message` is simply left empty in that response.

OpenAPI: the shared `WebchatManagerSession` schema (used by
Create/List/Get/End responses alike) gains an optional `welcome_message`
field, annotated in the spec that only the Create response ever
populates it — List/Get/End always return it empty, which is
consistent with the actual v7 widget-load flow (the client always calls
`POST /sessions` fresh on every page load; there is no documented
reload-via-GET path that would need this value repopulated).

## 7. Non-goals for this PR

- Idempotency key on `POST /webchat_sessions` (accepted client-side
  responsibility, §2.7).
- Any fix to the accepted MessageFlowID concurrent-overlap risk (§4) —
  tracked as a follow-up.
- Any change to `bin-flow-manager` itself (no new action types).
- `Session.ConversationID` storage — no confirmed need; the Conversation
  is always re-derivable from `(self=WidgetID, peer=SessionID)`.

## 8. Migration scope

- **DB**: Alembic migration, `bin-webchat-manager`'s `webchat_widgets`
  table — `ALTER TABLE ... RENAME COLUMN flow_id TO session_flow_id`,
  `ALTER TABLE ... ADD COLUMN message_flow_id BINARY(16) NULL`. Also
  update `bin-webchat-manager/scripts/database_scripts_test/widgets.sql`
  test schema to match.
- **OpenAPI**: `bin-openapi-manager/openapi/paths/webchat_widgets/
  {main,id}.yaml` — replace `flow_id` with `session_flow_id` +
  `message_flow_id`; `webchat_sessions/*.yaml`'s shared
  `WebchatManagerSession` schema gains `welcome_message` (§6). Regenerate
  `gens/models`/`gens/openapi_server` per the standard OpenAPI-first
  workflow (`bin-openapi-manager` first, then `bin-api-manager
  go generate`).
- **RST docs**: `bin-api-manager/docsdev/source/webchat_struct_widget.rst`,
  `webchat_overview.rst` — update field documentation to match
  `WebhookMessage`, not the internal model (per this repo's own RST
  sync rule).
- **square-admin** (separate repository, `monorepo-javascript`): 
  `square-admin/src/views/webchat_widgets/create.js` (and any
  update/detail views sharing the same field mapping) — replace the
  single `flowId`/`body.flow_id` field with two: `sessionFlowId` +
  `messageFlowId`. Tracked as a companion PR in the `monorepo-javascript`
  repository, not part of this PR's diff.
- **Breaking rename, no deprecation period**: `bin-webchat-manager` as a
  whole was merged into `main` on 2026-07-16/17 (this same work session)
  — confirmed via `git log --oneline -- bin-webchat-manager` showing the
  service's very first commit dated 2026-07-16 22:56. No real external
  API consumer of `flow_id` exists yet. (Caveat, explicitly flagged for
  pchero: k8s deployment manifests for the service are already merged,
  meaning *some* environment could theoretically have live widgets —
  worth a final human confirmation before merging this PR, not a
  blocking technical concern.)

## 9. Cross-tenant flow-ownership validation (§2.6 detail)

`bin-api-manager/pkg/servicehandler/webchat_widget.go`'s
`WebchatWidgetCreate` (existing flow-ownership check around line
142-161: `f.CustomerID != a.CustomerID` after `h.flowGet`) and
`WebchatWidgetUpdate` (existing check around line 206-233:
`f.CustomerID != w.CustomerID`, using the *widget's* owner rather than
the caller's, per that function's own comment explaining why —
preventing a ProjectSuperAdmin from repointing another customer's
widget at their own tenant's flow) must each run this same check
**twice** — once for `session_flow_id`, once for `message_flow_id`,
independently, whenever either is non-nil in the request. Skipping this
on the new `message_flow_id` field would silently reopen the exact
cross-tenant flow-execution vector the original Round 4 review closed
for the single `flow_id` field.

## 10. Test plan

- `bin-webchat-manager`: unit tests for the new `sessionhandler.Create`
  trigger path (mocked `ConversationV1ConversationCreateAndExecuteFlow`
  RPC — success, Widget-fetch-failure best-effort behavior, no-op when
  `SessionFlowID == uuid.Nil`); rewritten `messagehandler.Create` tests
  for `MessageFlowID` (every-message trigger, no-op when nil, no
  cross-message state); deleted concurrency test (§5.7).
- `bin-conversation-manager`: unit tests for the new RPC handler
  (Conversation always created fresh, `ReferenceType=Conversation`
  passed to `FlowV1ActiveflowCreate`, customer_id passthrough); existing
  `Test_EventWebchat_Inbound` negative assertion (no
  `FlowV1ActiveflowCreate` call from the async subscriber) must continue
  to pass unmodified — re-verify, don't rewrite.
- `bin-api-manager`: unit tests for `WebchatWidgetCreate`/`Update`'s
  duplicated cross-tenant check on both new fields (§9).
- Full verification workflow (`go mod tidy && go mod vendor &&
  go generate ./... && go test ./... && golangci-lint run -v --timeout
  5m`) in `bin-webchat-manager`, `bin-conversation-manager`,
  `bin-api-manager`, `bin-openapi-manager`.

## 11. Review history

16 rounds of adversarial analysis-stage review (background subagent,
fresh independent instance each round, full repo file access) were run
against this design before implementation. Key findings, in order:

- R1-R4: caught that plain "matches SMS/LINE's MessageFlowID pattern"
  framing missed a critical gap — no agent-takeover suppression existed
  for webchat, concurrency-lock removal needed re-justifying, and the
  rename's DB/OpenAPI/RST/frontend migration scope was originally
  unstated.
- R6: caught the dual-owner-store design flaw (§2.4) — the single
  biggest correction, driven by pchero's own "flow-level condition
  branch, not a platform owner field" redirect after this was explained.
- R8-R9: caught that `FlowID` conceptually maps to `SessionFlowID` (not
  `MessageFlowID`) — pchero's own correction — and that CPO's own
  first attempt at implementing the fix over-corrected by also silently
  reverting the trigger's event (session-create vs. first-message),
  which was pchero's actual, deliberate ask.
- R12-R14: pchero's own further correction — session creation should
  eagerly create a Conversation and hand Flow-triggering ownership to
  conversation-manager entirely (the current §3 design) — caught a
  missing customer_id-trust justification and a MessageFlowID scope
  under-statement (lock removal, cross-tenant check duplication) along
  the way.
- R15-R16: final confirmation rounds, SOUND twice consecutively (the
  loop's termination condition) after all of the above were folded in.
