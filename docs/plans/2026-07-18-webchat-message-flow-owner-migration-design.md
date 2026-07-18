# Webchat MessageFlowID: move Flow-trigger ownership from webchat-manager to conversation-manager

- Ticket: NOJIRA
- Author: Hermes (CPO), for pchero (CEO/CTO)
- Status: DRAFT — first pass, not yet reviewed
- Supersedes: the `MessageFlowID` half of
  `2026-07-17-webchat-widget-session-message-flow-split-design.md` (§2.3,
  §3.2 step 4, §5.5). That design's `SessionFlowID` half (§3, §3.3) is
  UNCHANGED by this doc — this is a narrower follow-up correcting only
  the remaining asymmetry it left in place.

## 1. Problem

The 2026-07-17 design split Widget's single `FlowID` into
`SessionFlowID` (session-create-time trigger, owned by
`bin-conversation-manager` via `ConversationV1ConversationCreateAndExecuteFlow`)
and `MessageFlowID` (per-message trigger, deliberately left owned by
`bin-webchat-manager` itself, for zero-added-latency reasons — see that
doc's §2.3, §3.1).

This left webchat as the *only* channel whose per-message Flow-trigger
does not go through `bin-conversation-manager`. SMS (`Number.MessageFlowID`),
LINE and WhatsApp (`Account.MessageFlowID`) all resolve their flow
source and call `executeActiveflow` from inside
`bin-conversation-manager`'s `execute_mode.go`
(`runExecuteModeFlowLine/Message/WhatsApp`). Every activeflow they
trigger carries `ReferenceType=ReferenceTypeConversation`,
`ReferenceID=Conversation.ID`.

Webchat's `MessageFlowID` trigger, by contrast, still runs inside
`bin-webchat-manager`'s `messagehandler.Create` → `triggerMessageFlow`,
producing `ReferenceType=ReferenceTypeWebchat`, `ReferenceID=Session.ID`.

This asymmetry has a real, confirmed cost: `bin-flow-manager` branches
on `activeflow.ReferenceType` in three places
(`variablehandler/substitute.go`'s `{{voipbin.reference_data}}`
resolution, `activeflowhandler/actionhandle.go`'s `actionHandleAITalk`
AI-reference-type mapping, and `actionHandleCaseCreate`'s Case-peer
derivation) and all three only recognize `ReferenceTypeCall` and
`ReferenceTypeConversation`. A Flow triggered by webchat's
`MessageFlowID` silently gets none of: `reference_data` variable
substitution, correct AI reference-type propagation (it currently
falls through to `ReferenceTypeCall`, which is wrong), or Case
auto-creation (falls into `actionHandleCaseCreate`'s explicit
"not supported" warning branch, no Case is ever created).

pchero's directive (this session): there is no reason for webchat to be
the one exception. Move `MessageFlowID`'s trigger to
`bin-conversation-manager`, matching every other channel. Channel
parity and code consistency are the priority; §2 shows this also
happens to improve, not worsen, the visitor-facing latency profile, so
there is no tradeoff to accept here after all — see §2 for the
corrected analysis (the 2026-07-17 design's original "zero added
latency" framing for keeping this trigger in webchat-manager assumed a
tradeoff that turns out not to exist once the synchronous-vs-async
distinction is worked through properly).

## 2. Decision

`bin-webchat-manager` no longer triggers any activeflow of its own.
`bin-conversation-manager`'s existing `messageEventReceivedWebchat`
(the async subscriber on `webchat_message_created`, inbound direction)
gains a real `runExecuteModeFlowWebchat`, following the exact same
shape as `runExecuteModeFlowMessage`/`Line`/`WhatsApp`: resolve the
channel-specific flow source, then call the existing `executeActiveflow`
(unchanged — always uses `ReferenceType=ReferenceTypeConversation`,
`ReferenceID=cv.ID` internally).

This means, after this change:

- `ReferenceTypeWebchat` (in `bin-flow-manager/models/activeflow`) is no
  longer produced by any code path. It is NOT removed from the enum —
  historical activeflow rows already created with this value must
  remain readable/queryable — but no new activeflow will ever carry it.
  A comment is added at the constant noting this.
- The flow-manager reference_type gaps described in §1 close themselves
  automatically; no changes needed inside `bin-flow-manager` at all.
- Latency shape changes, in one direction only (no tradeoff, despite
  how the 2026-07-17 design's original wording might suggest): today,
  `bin-webchat-manager`'s synchronous message-create path blocks on
  `FlowV1ActiveflowCreate`+`Execute` before returning to the visitor
  (`triggerMessageFlow` is called inline inside `Create`, best-effort
  but still on the request path). After this change, webchat-manager's
  message-create response no longer waits on any Flow RPC at all — Flow
  triggering moves fully off the synchronous path and becomes
  event-driven (via the existing `webchat_message_created` event → the
  async `messageEventReceivedWebchat` subscriber), exactly matching
  SMS/LINE/WhatsApp's existing profile. The visitor-facing send
  response gets faster; the Flow's own effects (e.g. an immediate bot
  reply) start slightly later, bounded by event-publish +
  subscriber-pickup latency — the same latency profile SMS/LINE/WhatsApp
  already live with today, not a new one introduced by this design.

## 3. Flow source resolution: how conversation-manager gets `MessageFlowID`

Unlike SMS (`Number.MessageFlowID` via `cv.Self.Target`) and LINE/WhatsApp
(`Account.MessageFlowID` via `cv.AccountID`), webchat conversations have
no `AccountID` and `cv.Self.Target` is the Widget's UUID string (not a
number). The flow source is `Widget.MessageFlowID`, fetched via the
already-existing `WebchatV1WidgetGet(ctx, widgetID)` RPC
(`bin-common-handler/pkg/requesthandler/webchat_widget.go:56`) — no new
RPC needed.

```go
// execute_mode.go, new function, mirrors runExecuteModeFlowLine's shape
func (h *conversationHandler) runExecuteModeFlowWebchat(ctx context.Context, cv *conversation.Conversation, m *message.Message) error {
	widgetID, errParse := uuid.FromString(cv.Self.Target)
	if errParse != nil {
		return errors.Wrapf(errParse, "invalid widget id in conversation self target: %s", cv.Self.Target)
	}
	w, errGet := h.reqHandler.WebchatV1WidgetGet(ctx, widgetID)
	if errGet != nil {
		return errors.Wrapf(errGet, "could not get widget. widget_id: %s", widgetID)
	}
	if errExecute := h.executeActiveflow(ctx, cv, m, w.MessageFlowID); errExecute != nil {
		return errors.Wrapf(errExecute, "could not execute activeflow. widget_id: %s", w.ID)
	}
	return nil
}
```

`runExecuteModeFlow`'s switch gains `case conversation.TypeWebchat:
return h.runExecuteModeFlowWebchat(ctx, cv, m)` — no longer falls
through to the `default` no-op branch.

`executeActiveflow` itself is unchanged: it already no-ops cleanly when
`flowID == uuid.Nil` (`Widget.MessageFlowID` unset), exactly matching
SMS/LINE/WhatsApp's existing "no flow source configured" behavior.

## 4. Code changes

### bin-webchat-manager

1. `pkg/messagehandler/create.go`: **remove** `triggerMessageFlow`
   entirely. **Remove** the `w, err := h.db.WidgetGet(...)` call and the
   `w.MessageFlowID == uuid.Nil` gate inside `Create` — nothing in this
   function needs Widget any more once the trigger call is gone.
   `Create` becomes: persist the message (`h.create(...)`), publish the
   event (already inside `h.create`), return. No Flow-related code
   remains in this file.
2. `pkg/messagehandler/create_test.go`: remove/rewrite the
   `triggerMessageFlow`-related test cases (the ones asserting
   `FlowV1ActiveflowCreate`/`FlowV1ActiveflowExecute` calls from this
   package) — webchat-manager itself never calls these RPCs again for
   the message-trigger path. Replace with an assertion that `Create`
   does NOT call `WidgetGet` or any `FlowV1*` RPC on the inbound path
   any more (a negative test, mirroring `Test_EventWebchat_Inbound`'s own
   negative-assertion style on the conversation-manager side, updated to
   its NEW meaning — see §5 below).
3. Confirm no other caller of `triggerMessageFlow`/the removed
   `WidgetGet` call exists before deleting (grep first).

### bin-conversation-manager

1. `pkg/conversationhandler/execute_mode.go`: add
   `runExecuteModeFlowWebchat` (§3). Add `case conversation.TypeWebchat`
   to `runExecuteModeFlow`'s switch.
2. `pkg/conversationhandler/event_webchat.go`: update the stale comment
   block in `messageEventReceivedWebchat` (currently lines 124-130,
   "B안 (confirmed, design doc §16.5): webchat-manager alone owns the
   real-time Flow trigger... This subscriber path must NEVER trigger a
   Flow of its own") — this is now FALSE and must be corrected to
   describe the new reality: this subscriber path DOES trigger
   `MessageFlowID`'s Flow now, via `ExecuteModeFlow` →
   `runExecuteModeFlowWebchat`, exactly like every other channel. Do
   not leave a comment describing the old, now-incorrect behavior next
   to code that behaves differently (the exact drift trap called out in
   the `new-channel-conv-mgr-integration.md` skill reference for this
   same file).
3. `pkg/conversationhandler/event_webchat_test.go`:
   `Test_EventWebchat_Inbound` currently asserts the ABSENCE of any
   `FlowV1ActiveflowCreate` call — this assertion is now WRONG and must
   be rewritten to assert the OPPOSITE: given a `Widget.MessageFlowID !=
   uuid.Nil` fixture, `FlowV1ActiveflowCreate` IS called with
   `ReferenceType=ReferenceTypeConversation`, `ReferenceID=cv.ID`
   (mirroring `Test_runExecuteModeFlowLine`'s assertion shape in
   `execute_mode_test.go`). Add a second case: `Widget.MessageFlowID ==
   uuid.Nil` → no RPC call, no error (the existing no-flow-configured
   no-op path). Add a `WebchatV1WidgetGet` mock expectation to both.
4. `pkg/conversationhandler/execute_mode_test.go`: add
   `Test_runExecuteModeFlowWebchat` mirroring
   `Test_runExecuteModeFlowLine`/`Test_runExecuteModeFlowMessage`'s
   existing table-driven shape exactly (happy path with flow id set;
   `WebchatV1WidgetGet` fetch failure → error wrapped and returned;
   Widget has no flow id → no activeflow created, no error; malformed
   `cv.Self.Target` → error, no RPC calls).

### bin-flow-manager

No code changes. `ReferenceTypeWebchat` constant stays (historical
data), gains a one-line comment: `// no longer produced by any current
code path (webchat's MessageFlowID trigger moved to
bin-conversation-manager, 2026-07-18) — retained for historical
activeflow rows only`.

## 5. Test plan

- `bin-webchat-manager`: rewritten `create_test.go` cases per §4.
  Full verification workflow.
- `bin-conversation-manager`: rewritten `Test_EventWebchat_Inbound`,
  new `Test_runExecuteModeFlowWebchat` per §4. `Test_EventWebchat_Outbound`
  and `Test_EventWebchat_UnknownDirection` are unaffected (outbound
  never triggers a Flow regardless of channel; this doc does not touch
  that). Full verification workflow.
- Manual/integration smoke: create a widget with `MessageFlowID` set,
  send an inbound webchat message via the existing dogfood/sandbox path,
  confirm exactly ONE activeflow is created with
  `ReferenceType=conversation` (query `bin-flow-manager`'s activeflow
  list, filter by the conversation's ID) — not two, not
  `ReferenceType=webchat`.

## 6. Non-goals

- `SessionFlowID`'s trigger path (`ConversationV1ConversationCreateAndExecuteFlow`,
  owned by conversation-manager since the 2026-07-17 design) is
  unchanged — already on the conversation-manager side, already uses
  `ReferenceType=conversation`.
- No change to `bin-api-manager`, OpenAPI schemas, or any DB migration —
  `Widget.MessageFlowID` itself is unchanged; only which service reads
  it and triggers against it moves.
- No change to the accepted-risk note in the 2026-07-17 design (§4,
  concurrent overlapping MessageFlowID executions with no lock) — this
  doc does not add or remove that risk, it only relocates which service
  the (already-accepted) risk lives in.

## 7. Rollout sequencing (deploy-order double-trigger / no-trigger window)

`bin-webchat-manager` and `bin-conversation-manager` are two
independently deployed services (separate k8s rolling deploys). This
design is NOT a single atomic commit across both in effect — deploying
one service's change does not deploy the other's. Because of this, the
deploy ORDER between the two creates two distinct transitional-window
risks that must be handled explicitly, not left implicit:

1. **conversation-manager's new trigger deploys BEFORE
   webchat-manager's old trigger is removed** → during the window where
   both are live, every inbound webchat message with `MessageFlowID`
   set triggers **two independent activeflows** — one from each
   service (double bot reply, double AI session start, duplicate
   `Case` creation attempts).
2. **webchat-manager's old trigger is removed BEFORE
   conversation-manager's new trigger deploys** → during that window,
   inbound webchat messages trigger **no activeflow at all** (silent
   gap — no error, just a missing Flow execution for any customer with
   `MessageFlowID` configured).

**Resolution: land BOTH service-level diffs (§4) in a single PR/commit
(simpler review, simpler squash-merge history), but deploy them
sequentially using this repo's existing CI/CD sequencing mechanism —
`bin-conversation-manager`'s change deploys FIRST, is verified in
production, THEN `bin-webchat-manager`'s trigger removal deploys
second.** This is enforced via `.circleci/config.yml`'s
`path-filtering` orb (triggers one independent pipeline per changed
`bin-*-manager/` directory) combined with each service pipeline's own
manual `build-approval` gate (`type: approval`, gating
`<service>-test`/`-build`/`-release` in `config_work.yml`) — merging
this single PR triggers the `bin-conversation-manager` and
`bin-webchat-manager` pipelines (and, incidentally, `bin-flow-manager`'s
pipeline too, for the comment-only change in §4 — that pipeline's
approval can be clicked at any time, since it carries no behavioral
change and has no sequencing dependency on the other two), but neither
of the two SEQUENCING-RELEVANT pipelines proceeds past its
`build-approval` step without an explicit manual click. The rollout
operator approves `bin-conversation-manager`'s gate first, verifies in
production (the REDUCED criterion below), THEN approves
`bin-webchat-manager`'s gate. No code-level PR split is needed — the
existing per-service approval gate already provides the sequencing
control this design requires. (An earlier draft of this design
required splitting this migration into two separate PRs; that
requirement is superseded by this section once the CI/CD mechanism
above was verified to already provide equivalent sequencing control
without a PR split.)

This accepts window (1) — a bounded period of double-triggering — over
window (2) — a silent gap — because a double bot-reply/duplicate Case
is visibly wrong and immediately noticeable (customer complaint, log
volume spike), whereas a silent no-trigger gap can go unnoticed for an
arbitrary period. Concretely:

- Approval 1 (`bin-conversation-manager`'s `build-approval` gate):
  approve first. Deploy. **Verification at this stage is NOT the full
  §5 smoke test** — §5's "exactly ONE activeflow, not two" success
  criterion only holds once BOTH deploys have landed. At this
  intermediate stage, the correct (and only) check is a REDUCED
  criterion: confirm that a fresh inbound webchat message with
  `MessageFlowID` set now ALSO produces a `ReferenceType=conversation`
  activeflow (existence check only) — seeing a `ReferenceType=webchat`
  activeflow ALONGSIDE it at this point is the expected, accepted
  transitional state, not a failure. Do not run §5's full pass/fail
  smoke test until after deploy 2; running it here would misclassify
  the correct transitional double-trigger state as a regression and
  risk triggering an unnecessary rollback. Because
  `MessageFlowID` has no existing production adopters at design time
  (see below), this existence check requires standing up ONE dedicated
  test widget with `MessageFlowID` set for the duration of the
  verification step — this is the one deliberate, intentional
  exception to the "avoid configuring new widgets with `MessageFlowID`"
  guidance below; that guidance is aimed at customer-facing widget
  configuration during the window, not at this design's own required
  verification step.
- Approval 2 (`bin-webchat-manager`'s `build-approval` gate): approve
  only after Approval 1's verification has passed. Deploy.
  Double-triggering ends; only the `conversation`-typed activeflow is
  produced from this point forward. **§5's full smoke test ("exactly
  ONE activeflow, not two") is the correct and only point to run it —
  it is Approval-2's acceptance criterion, not Approval-1's.**
- Between the two deploys, avoid configuring NEW customer-facing
  widgets with `MessageFlowID` if practical (existing widgets with it
  already set will double-trigger for the duration of the window
  regardless — this is accepted, not preventable without a feature
  flag, which is explicitly out of scope below). This does not apply to
  the single dedicated verification widget above.
- Feature-flag-gated sequential cutover (mentioned as an alternative
  during design review) is explicitly REJECTED as unnecessary
  complexity for this migration, on the following honestly-stated
  basis (correcting an earlier draft of this section that understated
  the window): the double-trigger window is NOT bounded merely by
  ordinary single-PR merge-to-deploy latency — it spans Approval 1's
  rollout, an UNBOUNDED production-verification step this design itself
  requires before clicking Approval 2's gate, and Approval 2's own
  rollout. This window could plausibly span hours to a few days if
  verification is not prioritized promptly, not merely minutes. The
  rejection of a feature flag stands anyway, because blast radius (not
  window duration) is the controlling factor here: query
  `bin-webchat-manager`'s `webchat_widgets` table for `COUNT(*) WHERE
  message_flow_id IS NOT NULL` before starting the rollout to confirm
  the actual number of exposed customers (expected to be zero or
  near-zero, since `MessageFlowID` is a brand-new opt-in field with no
  existing adopters at the time of this design — verify this count
  explicitly rather than assuming it). If that count is non-trivial by
  the time this rollout actually happens, re-open the feature-flag
  question rather than proceeding on this design's original assumption.
