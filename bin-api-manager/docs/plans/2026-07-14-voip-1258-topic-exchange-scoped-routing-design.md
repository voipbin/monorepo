# VOIP-1258: bin-webhook-manager / bin-api-manager WebSocket Event Subscription — Broker-Level Scoped Routing (Reduced Scope)

## 1. Origin and decisions already made

Tracking ticket: VOIP-1258. Discovered and verified (2 independent adversarial review rounds,
both APPROVED with zero changes requested) in a CPO/CEO discussion on 2026-07-14: per-pod
processing cost for events published to `QueueNameWebhookEvent` is paid unconditionally for
every published event, multiplied by api-manager pod count, with zero dependency on connected
websocket client count. Chat-type events additionally pay a live `TalkV1ParticipantList` RPC
call per event, also unconditional.

**pchero has decided**: go broker-level (RabbitMQ `fanout` -> `topic` exchange), with
**scope-first** routing key ordering (`customer_id.<uuid>.#` / `agent_id.<uuid>.#`), explicitly
rejecting resource-first ordering and deferring functional sharding as out of scope.

**This is REVISION 2 of the design, reflecting a scope-reduction decision made 2026-07-14 in
follow-up discussion, superseding the original 8-open-question, ~25-service-blast-radius draft.**
The reduction was triggered by a decisive finding (verified below, §2): `bin-api-manager`'s
event-processing switch statement handles exactly ONE event type
(`webhook.EventTypeWebhookPublished` from `bin-webhook-manager`) and unconditionally discards
everything else. This means the original design's ~116-call-site blast radius across ~25+
services (bare `notifyHandler.PublishEvent` calls feeding `QueueNameAgentEvent`/
`QueueNameTalkEvent`) was solving a problem that does not exist for websocket delivery purposes
-- those events were never reaching a client anyway. **pchero's explicit direction: touch only
`bin-webhook-manager` (publisher) and `bin-api-manager` (consumer).**

## 2. Current implementation (verified from source, revised)

**Decisive finding: bin-api-manager only ever acts on webhook-manager's events.**

```go
// bin-api-manager/pkg/subscribehandler/main.go:132-144, processEvent()
switch {
case m.Publisher == string(commonoutline.ServiceNameWebhookManager) && (m.Type == string(wmwebhook.EventTypeWebhookPublished)):
    err = h.processEventWebhookManagerWebhookPublished(ctx, m)
default:
    // ignore the event
    return
}
```

There is exactly one `case`. Events arriving via the `QueueNameAgentEvent` / `QueueNameTalkEvent`
subscriptions (`cmd/api-manager/main.go:159-162`) hit `default: return` unconditionally --
they are consumed off RabbitMQ (so the RabbitMQ-side cost the original design worried about does
apply to them too) but never reach `createTopics()`/ZMQ/any websocket client. **This subscription
is dead weight for websocket delivery purposes today.** Real client-visible agent/chat events
reach clients ONLY by first going through `bin-webhook-manager`:

```
bin-agent-manager / bin-talk-manager (any handler)
  -> notifyHandler.PublishWebhookEvent(ctx, customerID, eventType, data)
      -> go PublishEvent(...)          [-> QueueNameAgentEvent / QueueNameTalkEvent -- DEAD END for websocket, per above]
      -> go PublishWebhook(...)        [-> reqHandler.WebhookV1WebhookSend(...) RPC to webhook-manager]
          -> bin-webhook-manager.SendWebhookToCustomer(ctx, customerID, dataType, data)
              -> h.notifyHandler.PublishEvent(ctx, webhook.EventTypeWebhookPublished, wh)
                  -> QueueNameWebhookEvent (fanout exchange)   <- THIS is the only path that
                                                                    reaches a websocket client
```

Full existing pipeline for the path that matters:

```
bin-webhook-manager (webhookhandler/webhook.go):
  SendWebhookToCustomer(ctx, customerID, dataType, data)     [line 26, customerID EXPLICIT param]
  SendWebhookToURI(ctx, customerID, uri, method, dataType, data)  [line 120, customerID EXPLICIT param]
    both build wh := &webhook.Webhook{CustomerID: customerID, DataType: dataType, Data: data}
    both call h.notifyHandler.PublishEvent(ctx, webhook.EventTypeWebhookPublished, wh)
      -> sockHandler.EventPublish(exchange="bin-manager.webhook-manager.event", key="", evt)
          -> channel.PublishWithContext(...)  [amqp]

Exchange declared once at NewNotifyHandler() time via sockHandler.TopicCreate(name)
    -> ExchangeDeclare(name, "fanout", durable=true, ...)
       [bin-common-handler/pkg/rabbitmqhandler/topic.go:5-9]

bin-api-manager, at pod boot (cmd/api-manager/main.go:160, runSubscribe, unconditional):
    per-pod queue (QueueNameAPISubscribe-<uuid>) bound to QueueNameWebhookEvent via
    QueueSubscribe -> QueueBind(name, "", exchange, ...)
    [bin-common-handler/pkg/rabbitmqhandler/queue.go:158-160]
    -- empty routing key, irrelevant for fanout, binding lives for the pod's whole lifetime

subscribehandler.processEventRun -> processEvent -> (webhook_published, only case) ->
  processEventWebhookManagerWebhookPublished (bin-api-manager/pkg/subscribehandler/webhookmanager.go:48)
    -- 3x json.Unmarshal (Webhook envelope, Data, commonWebhookData), createTopics() [may call
       TalkV1ParticipantList RPC for chat types], zmqpubHandler.Publish(topic, data) per
       generated topic string -- all unconditional, regardless of local subscriber count

Local per-connection ZMQ SUB filtering (bin-api-manager/pkg/zmqsubhandler) is the ONLY
connection-count-dependent step, and it sits after all the above.
```

`createTopics()` (`bin-api-manager/pkg/subscribehandler/webhookmanager.go:107-175`) unmarshals
the doubly-nested webhook envelope (`Webhook.Data` -> `Data.Data` -> `commonWebhookData{Identity,
Owner, AIcallID, ChatID}`) to extract `CustomerID`/`OwnerID`, and for
`chat`/`chatmessage`/`chatparticipant` resource types calls
`h.reqHandler.TalkV1ParticipantList(ctx, chatID)` to fan out a topic string per chat participant.

Existing client-facing topic string format (already public API contract, documented in
`bin-api-manager/docsdev/source/websocket_overview.rst` etc., UNCHANGED by this design):
```
<scope>:<scope_id>:<resource>:<resource_id>
e.g. customer_id:abc123:call:xyz789
     agent_id:agent123:queue:*
```

## 3. Design goal (unchanged)

Move scope-based filtering from "after full local processing, at the final ZMQ hop" to "at the
RabbitMQ broker, before the event ever reaches a pod without a matching subscriber." A pod with
zero clients subscribed to a given `customer_id`/`agent_id` scope should never receive, consume,
unmarshal, or run `createTopics()`/RPC for that scope's events at all.

## 4. Reduced scope: two services only

**In scope**: `bin-webhook-manager` (publisher-side routing key computation) and
`bin-api-manager` (topic-exchange consumer + dynamic bind/unbind). No other service's publish
call sites, signatures, or event-handling logic changes.

**Consequence for `bin-common-handler`**: `notifyHandler.PublishEvent`/`PublishWebhookEvent`'s
public signatures are UNCHANGED. The original design's Open Questions 1 and 5 (bare
`PublishEvent`'s customerID-less signature, ~116-call-site blast radius, disqualified
Owner/Identity interface option) are now MOOT -- those call sites feed `QueueNameAgentEvent`/
`QueueNameTalkEvent`, which are out of scope entirely (see §7).

**Consequence for routing-key computation**: both webhook-manager entry points that reach
`QueueNameWebhookEvent` already receive `customerID` as an explicit parameter:
- `SendWebhookToCustomer(ctx, customerID uuid.UUID, dataType, data)` (`webhook.go:26`)
- `SendWebhookToURI(ctx, customerID uuid.UUID, uri, method, dataType, data)` (`webhook.go:120`)

Both already construct `wh := &webhook.Webhook{CustomerID: customerID, ...}` before calling
`notifyHandler.PublishEvent`. No signature change is needed on the webhook-manager side to
obtain `customer_id` for the routing key -- it is already in scope at the call site. This is
a materially simpler starting point than the original design's ~116-site migration.

**What is NOT simplified**: the `agent_id`/owner scope and the chat-participant fan-out (see §6)
still require moving logic that currently lives in `createTopics()` (consumer side) to the
publisher side, because the routing key must exist at `channel.PublishWithContext` time.

## 5. Proposed routing key format (scope-first, unchanged from revision 1)

```
<scope>.<scope_id>.#
e.g. customer_id.a1b2c3d4-....call.xyz789   (as the PUBLISHED routing key)
     customer_id.a1b2c3d4-....#             (as a pod's BOUND pattern, wildcard tail)
     agent_id.98765432-....#
```

Rationale (recapped): resource_id is unknown at bind time, so any pattern putting it before the
scope segment forces the scope filter to collapse into an unfiltered wildcard, defeating the
purpose. Scope-first keeps the wildcard tail (`#`) confined to the low-value, high-cardinality
segments (resource type + resource id), while the scope segment -- known at
websocket-connect/subscribe time, low cardinality per pod -- does the actual filtering work at
the trie's first level.

Two scope namespaces (`customer_id`, `agent_id`) must coexist; a single event may need to be
published under BOTH routing keys (mirrors the existing dual topic-string generation already
done in `createTopics()` today for `customer_id:...` and `agent_id:...`).

## 6. Who computes the routing key: moves into bin-webhook-manager

`createTopics()`'s logic (currently in `bin-api-manager/pkg/subscribehandler/webhookmanager.go`)
must move into `bin-webhook-manager`, executed BEFORE `notifyHandler.PublishEvent` is called, so
the routing key exists at publish time.

**Straightforward part**: `customer_id` is already an explicit parameter at both call sites
(§4) -- computing `customer_id.<customerID>.#`-shaped keys requires no new plumbing for that
scope alone.

**Harder part 1 -- `agent_id`/owner scope extraction**: today's `createTopics()` unmarshals the
event `data` payload (via the nested `Webhook.Data` -> `Data.Data` -> `commonWebhookData{Owner}`
envelope) to find `OwnerID` for the `agent_id:...` routing key. `SendWebhookToCustomer`/
`SendWebhookToURI` receive `data json.RawMessage` as an opaque blob (`webhook.go:26,120`) --
they do not know the owner/agent ID structurally, only that it may be present somewhere inside
`data`'s nested JSON. **This requires porting the SAME nested-unmarshal logic
(`commonWebhookData` struct, currently in `bin-api-manager/pkg/subscribehandler/webhookmanager.go`)
into `bin-webhook-manager`**, run against `data` before publish, to extract `OwnerID` the same
way `createTopics()` does today. This is a genuine, non-trivial logic move (not just a signature
change), but it is confined to `bin-webhook-manager`, not spread across ~25 services.

**Harder part 2 -- chat participant fan-out RPC moves to the publish path**: for
`chat`/`chatmessage`/`chatparticipant` resource types, `createTopics()` today calls
`h.reqHandler.TalkV1ParticipantList(ctx, chatID)` (`webhookmanager.go:143`) AFTER receiving the
event, to generate one `agent_id:<participant>:chat:...` routing key per chat participant. Moving
this to webhook-manager means:
- `webhook-manager`'s `webhookHandler` struct (`pkg/webhookhandler/main.go:32-40`) does NOT
  currently hold a `reqHandler requesthandler.RequestHandler` dependency -- verified by reading
  the struct definition. **A new dependency injection is required** (constructor signature
  change to `NewWebhookHandler`, plus wiring in `cmd/webhook-manager/main.go` and
  `cmd/webhook-control/main.go`) to call `TalkV1ParticipantList` from webhook-manager.
- This means every chat-type webhook publish now makes a synchronous RPC call to
  `bin-talk-manager` BEFORE the event reaches RabbitMQ at all, rather than after api-manager
  receives it. The RPC cost does not disappear -- it relocates from "per-pod, per-event, only if
  a subscriber happens to be listening after the fact" to "once per event, at publish time,
  regardless of subscriber count." This is a NET IMPROVEMENT versus today (today it is
  unconditional AND multiplied by pod count; after this change it is unconditional but paid
  exactly once, not once-per-pod) but it is NOT eliminated, and should not be described as
  "solved" -- flagged as Open Question 4.

## 7. Explicit scope exclusion: QueueNameAgentEvent / QueueNameTalkEvent

**Decision, not deferred**: `bin-api-manager`'s subscriptions to `QueueNameAgentEvent`
(`bin-agent-manager`'s direct publishes) and `QueueNameTalkEvent` (`bin-talk-manager`'s direct
publishes) are OUT OF SCOPE for the topic-exchange conversion, because they are proven dead
weight for websocket delivery (§2) -- `bin-api-manager`'s `processEvent` switch never acts on
them.

**Bonus cleanup, in scope for this ticket** (small, low-risk, and a direct consequence of the
finding in §2, not a new feature): remove `bin-api-manager`'s subscription to
`QueueNameAgentEvent`/`QueueNameTalkEvent` entirely --
`cmd/api-manager/main.go:159-162`'s `subscribeTargets` list shrinks to just
`QueueNameWebhookEvent`. This eliminates RabbitMQ consumption cost (per pod) for two exchanges'
worth of events that were always discarded, with zero behavior change (nothing was ever acted on
from those two subscriptions). Low risk: `git log`/blast-radius check should confirm no other
code path in `bin-api-manager` depends on having consumed (vs. ignored) these two subscriptions
before removing them -- flagged as Open Question 6, a verification step, not a design decision.

This means `QueueNameAgentEvent` and `QueueNameTalkEvent` exchanges themselves are UNCHANGED --
still `fanout`, still consumed by `bin-queue-manager` (`QueueNameAgentEvent`, per
`bin-queue-manager/cmd/queue-manager/main.go:149`) and `bin-timeline-manager`
(`QueueNameTalkEvent`, per `bin-timeline-manager/pkg/subscribehandler/main.go:50`) for their own
unrelated purposes (queue routing, audit-log ingestion). `bin-agent-manager` and
`bin-talk-manager` are the PUBLISHERS to `QueueNameAgentEvent`/`QueueNameTalkEvent` respectively
(`bin-agent-manager/cmd/agent-manager/main.go:95`, `bin-talk-manager/cmd/talk-manager/main.go:82-87`)
-- they do not subscribe to their own exchange, and are unaffected either way. Only
`bin-api-manager`'s now-pointless subscription to both exchanges is removed.

## 8. Exchange migration strategy (reduced: only QueueNameWebhookEvent)

RabbitMQ exchange kind is fixed at declare time; re-declaring an existing exchange name with a
different kind fails with `PRECONDITION_FAILED`. `QueueNameWebhookEvent` is already declared
`fanout`/`durable=true` in production, so an in-place kind change is not possible.

**Still relevant even at reduced scope: other, unrelated consumers of `QueueNameWebhookEvent`
specifically** (verified, monorepo-wide grep, independent of the `QueueNameAgentEvent`/
`QueueNameTalkEvent` exclusion in §7):
- `bin-agent-manager/cmd/agent-manager/main.go:159` subscribes to `QueueNameWebhookEvent`
  (agent status update logic, unrelated to websocket delivery)
- `bin-timeline-manager/pkg/subscribehandler/main.go:54` subscribes to `QueueNameWebhookEvent`
  (platform-wide audit-log ingestion, along with the other two exchanges which are unaffected
  by this design per §7)

Both use empty-routing-key fanout bindings (`QueueBind(name, "", exchange, ...)`,
`queue.go:158-160`) and, per an earlier verification pass, neither reads or branches on the AMQP
routing key anywhere in their code (`grep RoutingKey` in both services: zero hits) -- a
`#`-wildcard binding on the new topic exchange preserves identical delivery semantics with no
internal logic changes required for either service.

Proposed path (skeleton, not a finalized runbook):

1. Declare a new topic exchange under a new name (e.g. `QueueNameWebhookEvent` + `.topic`
   suffix, or a new `models/outline` constant) alongside the existing fanout exchange. **Must
   declare `durable=true`**, matching today's fanout exchange
   (`bin-common-handler/pkg/rabbitmqhandler/topic.go:7`) -- a non-durable new exchange risks
   silent message loss on a broker restart during the transition window (Open Question 7).
2. `bin-webhook-manager` dual-publishes to both old (fanout, unscoped) and new (topic, scoped)
   exchanges during a transition window. **Bind/unbind ordering during this window needs an
   explicit guarantee** to avoid double-delivery to any consumer bound to both simultaneously --
   not resolved in this doc, flagged as Open Question 3.
3. `bin-api-manager` pods migrate their per-pod queue bindings from the old exchange to the new
   one (scoped, per §9's dynamic bind/unbind). `bin-agent-manager` and `bin-timeline-manager`
   migrate their existing `QueueNameWebhookEvent` bindings to the new topic exchange with a `#`
   wildcard (bind-call-target-only change, per the finding above).
4. Once `bin-api-manager`, `bin-agent-manager`, and `bin-timeline-manager` are all confirmed on
   the new exchange, remove the dual-publish and decommission the old fanout exchange.

## 9. Dynamic bind/unbind at the api-manager side (unchanged from revision 1)

Unlike today (bind once at pod boot, for the pod's whole lifetime), a topic-exchange scoped
binding must track which scopes (`customer_id`/`agent_id` values) currently have at least one
live local websocket subscriber on that pod, and bind/unbind the per-pod queue accordingly:

- On a client's first `subscribe` message for a new scope on that pod: bind
  `customer_id.<uuid>.#` (or `agent_id.<uuid>.#`) to the pod's queue.
- On the last client unsubscribing from / disconnecting from a scope on that pod: unbind.
- **Reference counting required**: multiple local connections (or multiple topic subscriptions
  from the same connection) can share a scope; only unbind when the count reaches zero.
- **Race condition to guard against**: a bind-in-flight event ordering where the client's
  subscribe ack is sent before the AMQP bind is confirmed, causing a missed first event. Bind
  must complete (or be confirmed) before acking the client's subscribe message.
- **Abrupt disconnect must also decrement the refcount -- a distinct code path from explicit
  unsubscribe, verified against source.** Reading
  `bin-api-manager/pkg/websockhandler/subscription.go:32-84`, today's cleanup on an abrupt
  websocket close is whole-object teardown, not incremental: the per-connection `zmqSub`
  (holding that connection's `topics` list) is discarded via `defer zmqSub.Terminate()` (line
  70) when `newCtx` is cancelled -- triggered by `receiveTextFromWebsock` erroring (lines
  122-125) or a write failure in the pinger/ZMQ-run goroutines. No explicit `Unsubscribe(topic)`
  call fires on this path today. The refcount decrement must also be driven from
  `subscriptionRun`'s teardown (the `newCtx.Done()` / deferred-cleanup point), which needs new
  state tracking which scopes each connection currently holds. Getting this wrong means every
  abrupt client disconnect leaks a refcount and leaves a stale scope binding on the pod's queue
  permanently, silently reintroducing the exact problem this design exists to fix.
- Implementation surface: `subscriptionHandleMessage` (`subscription.go:159/168`, currently
  calls `zmqSub.Subscribe`/`Unsubscribe`) is the integration point for explicit
  subscribe/unsubscribe; `subscriptionRun`'s teardown (line 70) is the separate integration
  point required for abrupt disconnect. Both need a parallel call into a new per-pod
  AMQP-bind-refcount component.

## 10. Out of scope (explicit)

- Resource-first or hybrid routing key ordering.
- Functional sharding / resource-type-dedicated api-manager pod pools.
- Application-level zero-subscriber short-circuit ("Alternative B" in prior discussion) -- not
  adopted as a substitute; may be worth layering in later as defense in depth.
- Any change to the client-facing websocket subscribe/unsubscribe wire protocol or topic string
  format -- this design only changes the internal AMQP transport layer; the public API contract
  is unchanged.
- **`QueueNameAgentEvent` and `QueueNameTalkEvent` exchanges and their existing publishers/
  consumers** (`bin-agent-manager` publishes to `QueueNameAgentEvent`, `bin-queue-manager`
  consumes it; `bin-talk-manager` publishes to `QueueNameTalkEvent`, `bin-timeline-manager`
  consumes it) -- unaffected by this design, per §7. Only `bin-api-manager`'s
  now-provably-pointless subscription to both is removed.
- Any change to `bin-common-handler`'s `notifyHandler.PublishEvent`/`PublishWebhookEvent` public
  signatures, or to any of the ~116 bare-`PublishEvent` call sites across ~25+ services --
  moot under the reduced scope (§4), since those events never reached a websocket client.

## 11. Open questions (need resolution before/during implementation)

1. Concrete new exchange name/queue-name constant for the new topic exchange, dual-publish
   transition window length, and rollback trigger criteria.
2. Testing strategy for the dynamic bind/unbind reference-counting logic in `bin-api-manager` --
   needs a plan for simulating multiple concurrent connections per pod sharing/unsharing scopes
   without flaking.
3. Bind/unbind ordering guarantee during the dual-publish transition window, to avoid transient
   double-delivery to any consumer bound to both exchanges simultaneously.
4. Chat-participant-fan-out RPC relocation (§6, harder part 2): confirm the "paid once at
   publish time, not once per pod" framing is accurate once implemented (i.e. does NOT
   accidentally get called once per routing key generated, or once per dual-publish target,
   which would multiply it again); design the new `reqHandler` dependency injection into
   `webhookHandler` cleanly (constructor signature, mock updates, wiring in both
   `cmd/webhook-manager` and `cmd/webhook-control`).
5. How does the per-pod bind/unbind refcount interact with abrupt disconnect / connection
   teardown (not just explicit unsubscribe messages)? See §9 -- `subscriptionRun`'s teardown
   path fires on every disconnect, clean or not, but carries no per-topic unsubscribe signal
   today; the refcount component needs its own state to know what to decrement on this path.
6. Verify (via blast-radius grep / test coverage check) that no other code path in
   `bin-api-manager` depends on the `QueueNameAgentEvent`/`QueueNameTalkEvent` subscriptions
   before removing them (§7) -- expected to be a clean removal given the `processEvent` switch
   statement's single-case structure, but should be confirmed, not assumed.
7. New topic exchange durability: confirm `durable=true` is actually set in implementation (easy
   boolean-flag omission), and whether a broker restart mid-dual-publish-window needs an
   explicit recovery/reconciliation step beyond exchange durability alone (binding durability
   vs. exchange survival).
8. Does moving the nested-envelope unmarshal logic (§6, harder part 1: extracting `OwnerID` from
   `commonWebhookData`) into `bin-webhook-manager` duplicate logic that should instead be shared
   (e.g. via `bin-common-handler`) between webhook-manager and whatever remains of
   `bin-api-manager`'s webhook-processing path, to avoid two independent implementations of the
   same envelope-parsing rules drifting apart over time?
