# VOIP-1406: Migrate event subscribers from fanout to topic pattern bindings

- Date: 2026-08-29
- Ticket: VOIP-1406 (Follow-up B of VOIP-1404)
- Status: Approved (design review loop: R1 Request Changes, R2 Approve, R3 Approve -- 2
  consecutive; R3 included a live-broker topology audit confirming zero consumer
  bindings on bin-manager.event and 1:1 match of assumed fanout topology)
- Depends on: VOIP-1404/1405/1419 (all merged; every publisher dual-publishes to
  `bin-manager.event` with `<publisher>.<resource>.<subscription-id>.<action>` keys)
- Blocks: VOIP-1407 (Follow-up C: remove fanout publish + per-service fanout exchanges)

## 1. Goal and scope

Move the consumer side of the event bus onto the global topic exchange: each of the
**20 in-scope services** (21 consumers minus api-manager) stops consuming its per-service
fanout event exchanges and instead binds its EXISTING subscribe queue to
`bin-manager.event` with patterns matching exactly what its dispatch handles. Payloads
are byte-identical `sock.Event`s on both exchanges, so **dispatch logic, queue names,
queue types, worker counts, and metrics are all unchanged -- this is a bindings-only
migration per service**, independently deployable and rollback-able.

Out of scope: VOIP-1233 (callback error propagation -- orthogonal, same files, kept out
deliberately; see analysis §3), api-manager (topic-native on the client-facing VOIP-1258
exchange, which permanently coexists), fanout-publish removal (VOIP-1407).

## 2. The migration template (per service, inside Run())

Generalizes the proven VOIP-1258 idiom (bin-agent-manager subscribehandler) from
1-pattern/1-unbind to N-patterns/M-unbinds. Order inside `Run()`, strictly BEFORE the
`go ConsumeMessage(...)` (the 2026-07-14 AMQP 503 channel race rules this placement):

```go
// 1. queue + fanout subscriptions exactly as today (unchanged lines)
h.sockHandler.QueueCreate(queue, "normal")
for _, target := range h.subscribeTargets { h.sockHandler.QueueSubscribe(queue, target) }

// 2. idempotent exchange declare -- the reference doc's prescribed call, verbatim
//    (both-sides-declare rule; makes start order irrelevant; kind/durable are
//    hardcoded in rabbitmqhandler/topic.go so redeclare cannot 406)
if errDeclare := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic"); errDeclare != nil {
    log.Errorf(...)  // stay fully on fanout; skip 3 and 4
} else {

    // 3. bind ALL patterns -- all-or-nothing
    bound := []string{}
    ok := true
    for _, pattern := range topicPatterns {   // the service's ruled bind set, see §4
        if errBind := h.sockHandler.QueueBind(queue, pattern, string(commonoutline.QueueNameEvent), false, nil); errBind != nil {
            log.Errorf(...)
            ok = false
            break
        }
        bound = append(bound, pattern)
    }

    if !ok {
        // best-effort rollback of partial topic binds, then stay fully on fanout.
        // an unbind failure here leaves partial double delivery -- log CRITICAL.
        // (style note: topicPatterns / fanoutUnbindTargets are both package-level vars,
        // referenced without a receiver -- the h. prefix below is pseudo-code shorthand.)
        for _, pattern := range bound {
            if errUnbind := h.sockHandler.QueueUnbind(queue, pattern, string(commonoutline.QueueNameEvent), nil); errUnbind != nil {
                log.Errorf("CRITICAL: partial topic bind could not be rolled back ...")
            }
        }
    } else {
        // 4. unbind ALL old fanout exchanges -- only after every pattern bound.
        //    unbind failure: CRITICAL log, not fatal (double delivery beats loss).
        for _, target := range h.fanoutUnbindTargets {  // == subscribeTargets minus asterisk
            if errUnbind := h.sockHandler.QueueUnbind(queue, "", target, nil); errUnbind != nil {
                log.Errorf("CRITICAL: still bound to BOTH exchanges (double delivery). Manual intervention required ...")
            }
        }
    }
}

// 5. go ConsumeMessage(...)   (unchanged)
```

Load-bearing properties, inherited from 1258 plus the new rules:
- bind-new-before-unbind-old; no window bound to neither.
- **All-or-nothing (D-RULE)**: ANY pattern bind failure -> roll back the partial topic
  binds (best-effort) and unbind NO fanout exchange; the service keeps running fully on
  fanout. Never mix per-pair sources within one service.
- Momentary double delivery is accepted (1258 precedent; every handler already tolerates
  fanout's at-least-once). Precise window: on every post-migration boot it opens at
  step 1 (the fanout re-subscribe -- the topic binds already persist on the durable
  queue) and closes when step 4 completes; the whole sequence is a handful of
  synchronous broker RPCs, tens of milliseconds even for billing's 14 patterns.
- Fanout `QueueSubscribe` calls in step 1 stay in the code during 1406 (they are the
  rollback surface and the degrade path). VOIP-1407 deletes them.
- Reconnect: tracked binds make this self-healing -- `redeclareAll()` replays the topic
  binds; `QueueUnbind` removed the fanout entries from the tracked set so they are NOT
  resurrected by this image.

**No bin-common-handler changes.** The declare call above is what the reference doc
already prescribes as "the shared helper" and what webhook-manager uses for the 1258
exchange; QueueBind/QueueUnbind/TopicCreateWithKind all exist.

**Where the lists live**: `topicPatterns` and `fanoutUnbindTargets` are PACKAGE-LEVEL
vars in each service's `pkg/subscribehandler` (the timeline `subscribeTargets`
precedent), built from `eventtopic.PatternAction(...)` calls at init. cmd wiring stays
untouched (`subscribeTargets` keeps feeding the fanout QueueSubscribe loop as today),
and the binding golden test in the same package pins the REAL production list, not a
copy. One structural variance: webhook-manager's subscribehandler receives its targets
as a comma-joined string split inside Run() -- its fanoutUnbindTargets iterates the
split result; everything else follows the slice template.

## 3. Pattern strategy

- Default: **one `eventtopic.PatternAction(publisher, resource, action)` per dispatch
  pair** -- the ticket's explicit goal ("bind only the needed combinations; kill the
  receive-everything-discard-in-switch waste"). PatternAction and the publish-side key
  use the same `SplitN(type, "_", 2)` + normalization, so the mapping is 1:1 by
  construction (e.g. `confbridge_joined` -> `call-manager.confbridge.*.joined`).
- Publisher identity strings: patterns are built ONLY from `commonoutline.ServiceName*`
  constants (eventtopic normalizes them identically to the publish side); services
  currently using hardcoded per-service publisher consts keep those consts for their
  switch cases (dispatch is untouched) -- only the new pattern list uses the shared
  constants.
- call-manager's asterisk-proxy path (16 ARI types, its highest-volume source): the
  `asterisk.all.event` fanout subscription is PERMANENTLY retained (asterisk-proxy does
  not publish to the topic exchange). Only its customer/flow/sentinel subscriptions
  migrate. **The sentinel defensive `TopicCreate` STAYS during 1406** (call-manager and
  timeline both): §2 keeps every fanout `QueueSubscribe` as the rollback/degrade
  surface, so each boot still binds the sentinel fanout exchange -- without the
  defensive declare, a deployment where sentinel-manager is absent 404s that bind,
  closes the channel, and kills the service at boot (the exact failure the declare
  exists to prevent). The declare is deleted in VOIP-1407 together with the fanout
  QueueSubscribe lines. (`asterisk.all.event` needs no new defensive declare:
  asterisk-proxy's own notifyhandler declares it, and both retained subscribers bind it
  today without one -- a pre-existing, unchanged start-order gap.)
- timeline-manager (**D-RULE, superset accepted**): ONE `"#"` bind replaces 25 fanout
  subscriptions (asterisk stays). This deliberately widens intake to all 27 current
  topic publishers (adds direct/schedule/webchat) and auto-includes future publishers --
  the correct semantics for the archive-everything timeline service. Consequences
  accepted: modest ClickHouse row growth; the events section of timeline's docs states
  the new contract ("everything on bin-manager.event, plus asterisk fanout").
- api-manager: untouched.

## 4. Mismatch rulings (zero-behavior-change principle)

1406 changes WHERE events come from, never WHAT is processed. Each discovered
binding/dispatch mismatch gets an explicit ruling:

| Case | Ruling |
|---|---|
| ai-manager dead binds (transcribe, tts fanout exchanges; zero dispatch cases) | Drop -- no patterns created, fanout binds removed with the others. Pure waste removal, zero behavior change. |
| ai-manager unreachable `conference-manager/conference_updated` case | NO pattern (keeping it unreachable = today's behavior). The dead case stays in code with a one-line comment pointing at the follow-up ticket. Register a follow-up Jira ticket: product decision to activate (bind) or delete the case. |
| queue-manager dead binds (agent, conference exchanges) | Drop, as above. |
| queue-manager unreachable `customer-manager/customer_deleted` case | NO pattern; same follow-up ticket, flagged prominently -- this smells like a LATENT BUG (queue records likely should be cleaned on customer deletion) but activating it inside a bindings migration would be an unreviewed behavior change. |
| flow-manager unreachable `call-manager/call_hangup` case (subscribeTargets is customer-only; `EventCallHangup` has NEVER run in production -- found in design review R1, missed by the analysis inventory) | NO pattern; same follow-up ticket, flagged with the queue case -- activeflow cleanup on hangup looks intended, so this is the second latent-bug candidate. |
| timeline dead bind (transfer fanout; transfer publishes nothing) | Subsumed by the `#` ruling -- the fanout bind is removed; if transfer ever publishes AND opts into topic publish (it has no `WithGlobalTopicPublish` today), `#` picks it up. |
| number-manager wrong-valued dead const | Delete the dead const while the file is touched (inline cleanup, no behavior). |
| campaign-manager wrong consumer tag (`ServiceNameQueueManager`) | Fix to its own service name while the file is touched (cosmetic; changes only the AMQP consumer tag string). |
| contact-manager stale VOIP-1233 comment | Refresh while the file is touched (comment-only). |

## 5. Per-service bind sets (NORMATIVE -- source for the binding golden tests)

Derived 1:1 from the verified dispatch inventory (analysis §2 of tasks/todo.md). Format:
publisher.resource.*.action. 19 pattern-bound services + timeline `#`:

- agent-manager (4): call-manager.groupcall.*.created / call-manager.groupcall.*.progressing / customer-manager.customer.*.deleted / customer-manager.customer.*.created
- ai-manager (11): call-manager.confbridge.*.joined / call-manager.confbridge.*.leaved / call-manager.call.*.hangup / call-manager.dtmf.*.received / conference-manager.conference.*.updated -> **excluded per §4** / pipecat-manager.message.*.user_transcription / pipecat-manager.message.*.bot_llm / pipecat-manager.message.*.bot_llm_intermediate / pipecat-manager.pipecatcall.*.initialized / pipecat-manager.pipecatcall.*.terminated / pipecat-manager.team.*.member_switched  => 10 patterns
- billing-manager (14): call-manager.call.*.progressing / call-manager.call.*.hangup / call-manager.recording.*.started / call-manager.recording.*.finished / message-manager.message.*.created / email-manager.email.*.created / customer-manager.customer.*.deleted / .created / .frozen / .recovered / number-manager.number.*.created / number-manager.number.*.renewed / tts-manager.speaking.*.started / tts-manager.speaking.*.stopped
- call-manager (4): customer-manager.customer.*.deleted / customer-manager.customer.*.frozen / flow-manager.activeflow.*.updated / sentinel-manager.pod.*.deleted  (+ asterisk fanout retained)
- campaign-manager (2): call-manager.call.*.hangup / flow-manager.activeflow.*.deleted
- conference-manager (2): call-manager.confbridge.*.joined / call-manager.confbridge.*.leaved
- contact-manager (1): customer-manager.customer.*.deleted
- conversation-manager (4): message-manager.message.*.created / email-manager.email.*.created / email-manager.email.*.updated / webchat-manager.webchat.*.message_created
- direct-manager (1): customer-manager.customer.*.deleted
- flow-manager (1): customer-manager.customer.*.deleted  (the call_hangup dispatch case
  is UNREACHABLE today -- see §4 -- and gets NO pattern)
- number-manager (2): customer-manager.customer.*.deleted / flow-manager.flow.*.deleted
- queue-manager (4): call-manager.call.*.hangup / call-manager.confbridge.*.joined / call-manager.confbridge.*.leaved / customer-manager.customer.*.deleted -> the customer pattern is **excluded per §4** => 3 patterns
- registrar-manager (2): customer-manager.customer.*.created / customer-manager.customer.*.deleted
- schedule-manager (1): customer-manager.customer.*.deleted
- storage-manager (2): customer-manager.customer.*.created / customer-manager.customer.*.deleted
- tag-manager (1): customer-manager.customer.*.deleted
- timeline-manager: `#` (+ asterisk fanout retained)
- transcribe-manager (3): call-manager.call.*.hangup / call-manager.confbridge.*.terminated / customer-manager.customer.*.deleted
- transfer-manager (3): call-manager.groupcall.*.progressing / call-manager.groupcall.*.hangup / call-manager.call.*.hangup
- webhook-manager (5): customer-manager.customer.*.created / customer-manager.customer.*.updated / flow-manager.activeflow.*.created / flow-manager.activeflow.*.updated / flow-manager.activeflow.*.deleted

Implementation detail: patterns are generated in code via
`eventtopic.PatternAction(...)` calls, never as string literals -- so the same
normalization that builds publish keys builds the bindings.

## 6. Tests

- **Binding golden test per service** (the consumer-side sibling of the routing-key
  goldens): a table test in each subscribehandler package pinning the EXACT pattern
  strings the service binds (and, for call/timeline, the retained fanout targets).
  Mirrors §5 byte-for-byte; catches drift in either direction (a dispatch case added
  without a pattern, or a pattern added without a case). Mutation-checked where cheap
  (assert the excluded §4 patterns -- ai conference_updated, queue customer_deleted,
  flow call_hangup -- are NOT in their sets).
- **Run() sequencing tests**: add per service (extending where the VOIP-1258 tests
  already exist -- only agent and timeline have Run() sequencing tests today) with
  gomock `InOrder`: QueueCreate -> fanout QueueSubscribes -> TopicCreateWithKind ->
  QueueBinds (all patterns) -> QueueUnbinds (all fanout targets) -> ConsumeMessage.
  Failure-path cases (at minimum in 2 representative services + call-manager +
  timeline): declare fails -> no binds/unbinds; bind i fails -> best-effort unbind of
  0..i-1 topic binds, zero fanout unbinds; fanout unbind fails -> CRITICAL logged,
  Run() still succeeds.
- No bch tests to change (no bch code changes).

## 7. Verification and rollout

- Per-service verification workflow (tidy/vendor/generate/test/lint); no cross-service
  compile risk (no shared-code change). Per-service docs sync: each touched
  `pkg/subscribehandler/main.go` updates its service's `docs/architecture.md` events
  section in the same commit (root CLAUDE.md rule; the docs hook warns otherwise).
- **Rollout order** (independent per service): wave 1 = low-volume services with 1-2
  simple patterns (contact, direct, schedule, storage, tag, flow, registrar, number);
  wave 2 =
  multi-pattern (agent, campaign, conference, conversation, transcribe, transfer,
  webhook, queue, ai, billing); wave 3 = timeline (`#`), then call-manager last
  (highest volume; asterisk leg keeps it partially on fanout regardless).
- **Live verification per wave**: broker binding inspection
  (`rabbitmqadmin list bindings` filtered on bin-manager.event and the service queue)
  shows exactly the §5 set; the old fanout binding is gone from the queue;
  `promEventProcessTime{Publisher,Type}` series continue advancing for the service's
  pairs; `bin-manager.event` `publish_out` starts growing (first real consumers).
- **Stale-binding policy (D-RULE)**: two triggers, one policy -- (a) image rollback
  (old Run() re-subscribes fanout; topic binds persist on the durable queue), (b)
  rolling-deploy window on 2-replica services (old-image pod reconnect replays its
  tracked fanout bind after the new pod unbound it). Policy: TOLERATE the resulting
  double delivery (at-least-once was always the contract), detect via broker-binding
  inspection, remediate by manual `rabbitmqadmin` unbind or roll-forward. No automated
  cleanup in 1406; the runbook section in the PR records the exact inspection and
  unbind commands.
- Fanout publish stays on everywhere until VOIP-1407.

## 8. Non-goals

- No dispatch/handler logic changes; no VOIP-1233 coupling (follow-up stays separate).
- No talk-control opt-in (black-holed CLI publishes would surface for the first time --
  needs its own ruling; trap documented in the analysis).
- No PatternInstance consumers (bind-after-start race options unused); no api-manager
  changes; no fanout exchange deletion.
- New follow-up ticket to register at PR time: "activate or delete the three unreachable
  dispatch cases (ai conference_updated, queue customer_deleted, flow call_hangup)" --
  the latter two are latent-bug candidates (missing cleanup on customer delete / call
  hangup) and should lead the ticket.
