# VOIP-1258: bin-api-manager WebSocket Event Subscription — Broker-Level Scoped Routing

## 1. Origin and decision already made

Tracking ticket: VOIP-1258. Discovered and verified (2 independent adversarial review rounds,
both APPROVED with zero changes requested) in a CPO/CEO discussion on 2026-07-14: per-pod
processing cost for events published to `QueueNameWebhookEvent` / `QueueNameAgentEvent` /
`QueueNameTalkEvent` is paid unconditionally for every published event, multiplied by
api-manager pod count, with zero dependency on connected websocket client count. Chat-type
events additionally pay a live `TalkV1ParticipantList` RPC call per event, also unconditional.

**pchero has already decided the direction**: go broker-level (RabbitMQ `fanout` → `topic`
exchange), with **scope-first** routing key ordering (`customer_id.<uuid>.#` /
`agent_id.<uuid>.#`), explicitly rejecting resource-first ordering and deferring functional
sharding (resource-type-dedicated pod pools) as out of scope. This design doc executes that
decision — it does not revisit the broker-vs-app-level or scope-first-vs-resource-first choice.

## 2. Current implementation (verified from source)

```
webhook/agent/talk-manager
  notifyHandler.PublishEvent(ctx, eventType, data)
    -> publishEvent() -> sockHandler.EventPublish(exchange=queueName, key="", evt)
        -> rabbitmqhandler.EventPublish(exchange, key, evt)
            -> publishExchange(exchange, key, message, ...) [amqp channel.PublishWithContext]

Exchange declared once at NewNotifyHandler() time via sockHandler.TopicCreate(name)
    -> ExchangeDeclare(name, "fanout", ...)   [bin-common-handler/pkg/rabbitmqhandler/topic.go:5-9]

bin-api-manager, at pod boot (cmd/api-manager/main.go:142 runSubscribe, unconditional):
    per-pod queue (QueueNameAPISubscribe-<uuid>) bound via QueueSubscribe -> QueueBind(name, "", exchange, ...)
    [bin-common-handler/pkg/rabbitmqhandler/queue.go:158-160]
    -- empty routing key, irrelevant for fanout, binding lives for the pod's whole lifetime

subscribehandler.processEventRun -> processEvent -> (webhook_published) ->
  processEventWebhookManagerWebhookPublished (bin-api-manager/pkg/subscribehandler/webhookmanager.go:48)
    -- 3x json.Unmarshal, createTopics() [may call TalkV1ParticipantList RPC for chat types],
       zmqpubHandler.Publish(topic, data) per generated topic string -- all unconditional

Local per-connection ZMQ SUB filtering (bin-api-manager/pkg/zmqsubhandler) is the ONLY
connection-count-dependent step, and it sits after all the above.
```

Existing client-facing topic string format (already public API contract, documented in
`bin-api-manager/docsdev/source/websocket_overview.rst` etc.):
```
<scope>:<scope_id>:<resource>:<resource_id>
e.g. customer_id:abc123:call:xyz789
     agent_id:agent123:queue:*
```
This colon-delimited string is currently used ONLY for the final local ZMQ SUB/PUB filter and
the client wire protocol (subscribe/unsubscribe websocket messages) — it is NOT currently an
AMQP routing key (routing key is always `""` today, since fanout ignores it).

`createTopics()` (`bin-api-manager/pkg/subscribehandler/webhookmanager.go:107-175`) is where this
topic-string generation currently lives, on the **consumer** (api-manager) side, after the event
has already been fully received and unmarshalled.

## 3. Design goal

Move scope-based filtering from "after full local processing, at the final ZMQ hop" to "at the
RabbitMQ broker, before the event ever reaches a pod without a matching subscriber." Concretely:
a pod with zero clients subscribed to a given `customer_id`/`agent_id` scope should never receive,
consume, unmarshal, or run `createTopics()`/RPC for that scope's events at all.

## 4. Proposed routing key format (scope-first, confirmed direction)

```
<scope>.<scope_id>.#
e.g. customer_id.a1b2c3d4-....call.xyz789   (as the PUBLISHED routing key)
     customer_id.a1b2c3d4-....#             (as a pod's BOUND pattern, wildcard tail)
     agent_id.98765432-....#
```

Rationale already established in discussion (recapped for the record): resource_id is unknown
at bind time (client hasn't received the event yet), so any pattern putting it before the scope
segment forces the scope filter to collapse into an unfiltered wildcard, defeating the purpose.
Scope-first keeps the wildcard tail (`#`) confined to the low-value, high-cardinality segments
(resource type + resource id), while the scope segment — known at websocket-connect/subscribe
time, low cardinality per pod — does the actual filtering work at the trie's first level.

Two scope namespaces (`customer_id`, `agent_id`) must coexist. Proposed: publish BOTH routing
keys per event when both a customer and an owner/agent scope apply (mirrors the existing
dual-topic generation already done in `createTopics()` today, which already emits both
`customer_id:...` and `agent_id:...` topic strings for the same event where applicable — this is
not new complexity, just moving existing dual-key generation earlier in the pipeline).

## 5. Who computes the routing key: publisher-side move

**Decision point carried into this doc, not yet resolved — see Open Questions.** Today
`createTopics()` runs on the consumer (api-manager) after receiving the full event. For topic-
exchange filtering to work, the AMQP routing key must exist at `channel.PublishWithContext` time,
i.e. on the publisher (webhook/agent/talk-manager) side, before the message ever reaches
RabbitMQ.

Verified asymmetry that affects this move:
- `notifyHandler.PublishWebhookEvent(ctx, customerID uuid.UUID, eventType string, data WebhookMessage)`
  (`bin-common-handler/pkg/notifyhandler/publish.go:22`) already receives `customerID` as an
  explicit parameter (used today for the webhook HTTP delivery path) — computing
  `customer_id.<customerID>.#`-shaped keys here is straightforward.
- `notifyHandler.PublishEvent(ctx, eventType string, data interface{})`
  (`bin-common-handler/pkg/notifyhandler/main.go:74`) does NOT receive customerID as a parameter
  — it is embedded inside `data`. `PublishWebhookEvent` itself calls the bare `PublishEvent`
  internally (`publish.go:23`), so this signature sits underneath both public entry points.

**Blast radius, verified by full-monorepo grep (not a handful of call sites):** bare
`notifyHandler.PublishEvent(` (excluding `PublishWebhookEvent`) has **~116 call sites across
~25+ services** (customer, tag, tts, billing, sentinel, call, pipecat, agent, webhook,
registrar, direct, conference, flow, queue, route, contact, transcribe, number, storage, ai,
campaign, conversation, outdial, email, talk-manager, and more). Because `PublishEvent`'s
signature lives in `bin-common-handler` (subject to the "3+ services" admission rule), any
signature change or added interface requirement touches close to the whole monorepo's
verification matrix — this is a large-blast-radius change, not an isolated one.

**The two extraction options in the prior draft are NOT symmetric — option (a) below is
disqualified for a real, non-trivial subset of call sites, confirmed by reading actual payloads:**
- (a) ~~A shared `Owner`/`Identity` interface with a `GetCustomerID()` method~~ — does not work
  universally. Several bare `PublishEvent` calls pass `map[string]uuid.UUID` or
  `map[string]interface{}` as `data`, not a domain struct embedding
  `commonidentity.Identity`/`Owner`:
  - `bin-contact-manager/pkg/casehandler/casenote.go:53-57` —
    `map[string]uuid.UUID{"id":..., "case_id":..., "customer_id": customerID}`
  - `bin-contact-manager/pkg/casehandler/case_tag.go:92-95,130-133` — same map pattern; note
    this map carries only `case_id`/`tag_id` keys and has **no `customer_id` key at all**, so
    these two call sites cannot even reach a `customer_id`-scoped routing key without an
    additional lookup or a broader signature change beyond the map-to-struct migration itself —
    a strictly harder case than the other two examples, not just "also a map."
  - `bin-call-manager/pkg/callhandler/outgoing_call.go:213-217` —
    `map[string]interface{}{"customer_id":..., "call_id":..., ...}`

  These have no embedded Go type to satisfy a `GetCustomerID()` interface; reflection would have
  to special-case map string keys anyway, which is exactly the "fragile type-switch per caller"
  approach this design wants to avoid. Interface extraction is not a clean universal solution.
- (b) Migrate every bare `PublishEvent` call site to an explicit-customerID signature (mirroring
  `PublishWebhookEvent`) is the only fully general option, at the ~116-site blast radius above.
  This is the option this design should plan around, not (a).
- `agent_id` scope: is there an equivalent "owner/agent" identity readily available at publish
  time for all three services, or does this only cleanly apply to `bin-agent-manager`'s own
  events? Needs verification per-service before finalizing (unchanged from prior draft, still
  open — see Open Questions).

## 6. Exchange migration strategy

RabbitMQ exchange kind is fixed at declare time; re-declaring an existing exchange name with a
different kind fails with `PRECONDITION_FAILED`. Since `QueueNameWebhookEvent` /
`QueueNameAgentEvent` / `QueueNameTalkEvent` are already declared `fanout` in production, an
in-place kind change is not possible.

**CRITICAL, verified: these three exchanges are NOT exclusively consumed by bin-api-manager.**
A monorepo-wide grep confirms other, unrelated consumers subscribe to the same exchanges for
their own purposes, all via `QueueSubscribe(queue, exchange)` -> `QueueBind(name, "", exchange,
...)` (`bin-common-handler/pkg/rabbitmqhandler/queue.go:158-160`) — an **empty routing key**,
which is irrelevant under `fanout` but under `topic` only matches messages published with an
exactly-empty routing key:
- `bin-agent-manager/cmd/agent-manager/main.go:159` subscribes to `QueueNameWebhookEvent`
  (agent status update logic, unrelated to websocket delivery)
- `bin-queue-manager/cmd/queue-manager/main.go:149` subscribes to `QueueNameAgentEvent`
  (queue routing / agent-tag matching)
- `bin-timeline-manager/pkg/subscribehandler/main.go:29,50,54` subscribes to **all three**
  (`QueueNameAgentEvent`, `QueueNameTalkEvent`, `QueueNameWebhookEvent`) as part of its
  platform-wide audit-log ingestion

Because exchange kind is global per exchange name (not scoped per-consumer), converting these
exchanges to `topic` with dot-formatted routing keys will **silently stop delivering events to
bin-agent-manager, bin-queue-manager, and bin-timeline-manager** the moment the old fanout
exchange is decommissioned, unless these three services' bindings are also migrated (e.g. to a
wildcard `#` binding, since they need every event regardless of scope, not scoped filtering).
This would break agent status updates, queue routing, and the platform audit log — a real,
previously undisclosed regression risk, not covered by "api-manager pods migrate their per-pod
queue bindings" alone. **The migration plan below is revised to explicitly include these three
consumers**, and this is escalated to Open Question 7.

Proposed path (to be detailed further in implementation planning; the steps below are a skeleton,
not a finalized runbook):

1. Declare new topic exchanges under new names (e.g. suffix `.topic` or a v2 queue-name constant)
   alongside the existing fanout exchanges.
2. Publishers dual-publish to both old (fanout, unscoped) and new (topic, scoped) exchanges during
   a transition window. **Bind/unbind ordering during this window needs an explicit guarantee**:
   if any consumer (api-manager pod, or the three non-websocket consumers above) ever binds to
   the new exchange before fully cutting over from the old one, it will double-receive events
   during the overlap. The doc does not yet specify a concrete ordering protocol for this —
   flagged as part of Open Question 7, not resolved here.
3. api-manager pods migrate their per-pod queue bindings from the old exchange to the new one
   (scoped, per §7's dynamic bind/unbind). **bin-agent-manager, bin-queue-manager, and
   bin-timeline-manager migrate their existing queues to bind the new topic exchange with a `#`
   wildcard** (they need the full, unscoped event stream for their own purposes — this is a
   like-for-like fanout-equivalent binding, not scoped filtering, and requires no changes to
   their internal event-handling logic, only the bind call's target exchange name and key).
4. Once all pods and all four consumer services are confirmed on the new exchange, remove the
   dual-publish and decommission the old fanout exchange.

This is standard blue/green exchange migration; needs concrete queue-name constants and a
rollback plan before implementation, not resolved in this doc.

## 7. Dynamic bind/unbind at the api-manager side

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
- **Abrupt disconnect must also decrement the refcount — this is a distinct code path from
  explicit unsubscribe, verified against source.** Reading
  `bin-api-manager/pkg/websockhandler/subscription.go:32-84`, today's cleanup on an abrupt
  websocket close (network drop, tab close, mobile backgrounding) is whole-object teardown, not
  incremental: the per-connection `zmqSub` (holding that connection's `topics` list) is discarded
  via `defer zmqSub.Terminate()` (line 70) when `newCtx` is cancelled — triggered by
  `receiveTextFromWebsock` erroring (lines 122-125) or a write failure in the pinger/ZMQ-run
  goroutines. **No explicit `Unsubscribe(topic)` call fires on this path today.** The new
  pod-level AMQP-bind-refcount component (shared across connections on a pod, per the paragraph
  above) has nothing hooking it to decrement that connection's contribution to each scope's
  refcount on this teardown path — `subscriptionHandleMessage`'s explicit-unsubscribe branch is
  not sufficient by itself. The refcount decrement must also be driven from `subscriptionRun`'s
  teardown (the `newCtx.Done()` / deferred-cleanup point), which needs new state tracking which
  scopes each connection currently holds (this state doesn't fully exist yet in a pod-level-
  reachable form — `zmqSub.topics` is per-connection only). Getting this wrong means every
  abrupt client disconnect leaks a refcount and leaves a stale scope binding on the pod's queue
  permanently, silently reintroducing the exact problem (unconditional processing for pods with
  zero real subscribers) this design exists to fix.
- Implementation surface: `bin-api-manager/pkg/websockhandler/subscription.go`
  `subscriptionHandleMessage` (currently calls `zmqSub.Subscribe(topic)`/`Unsubscribe(topic)`,
  lines 159/168) is the integration point for explicit subscribe/unsubscribe; `subscriptionRun`'s
  teardown (line 70 `defer zmqSub.Terminate()`, triggered via `newCtx` cancellation) is the
  separate integration point required for abrupt disconnect. Both need a parallel call into a new
  per-pod AMQP-bind-refcount component.

## 8. Out of scope (explicit, per pchero's decision)

- Resource-first or hybrid (`resource_name.customer_id.<uuid>.#`) routing key ordering.
- Functional sharding / resource-type-dedicated api-manager pod pools.
- Application-level zero-subscriber short-circuit (discussed as "Alternative B" in prior
  conversation) — not adopted as a substitute; may still be worth layering in later as defense
  in depth, but is not part of this design.
- Any change to the client-facing websocket subscribe/unsubscribe wire protocol or topic string
  format (`customer_id:<uuid>:<resource>:<resource_id>`) — this design only changes the internal
  AMQP transport layer between publishers and api-manager pods; the public API contract is
  unchanged.

## 9. Open questions (need resolution before/during implementation)

1. Bare `PublishEvent` (no explicit customerID) call sites: adopt an `Owner`/`Identity` interface
   extraction, or migrate all call sites to explicit-customerID signatures?
2. Does `agent_id` scope routing apply uniformly across webhook/agent/talk-manager publishers, or
   only to a subset? Needs per-service verification of what identity data is available at
   publish time.
3. Exchange migration: concrete new queue-name constants, dual-publish transition window length,
   and rollback trigger criteria.
4. Testing strategy for the dynamic bind/unbind reference-counting logic — needs a plan for
   simulating multiple concurrent connections per pod sharing/unsharing scopes without flaking.
5. Should `PublishEvent`/`PublishWebhookEvent`'s public interface signature change (e.g. add an
   explicit routing-key-relevant scope parameter), given `bin-common-handler`'s "3+ services"
   admission rule and the fact this interface is shared library code consumed by all 37 services?
   Given the confirmed ~116-call-site blast radius (§5) and that the map-payload call sites
   disqualify a non-invasive interface-extraction alternative, this is effectively asking
   "are we prepared to touch ~25+ services' publish call sites," not a low-cost signature tweak.
6. How does the per-pod bind/unbind refcount interact with abrupt disconnect / connection
   teardown (not just explicit unsubscribe messages)? See §7 — `subscriptionRun`'s teardown path
   (`newCtx` cancellation, `defer zmqSub.Terminate()`) fires on every disconnect, clean or not,
   but carries no per-topic unsubscribe signal today; the refcount component needs its own state
   to know what to decrement on this path, and that state doesn't exist in the codebase yet.
7. **(New, blocking) Non-websocket consumers of the same exchanges.** `bin-agent-manager`,
   `bin-queue-manager`, and `bin-timeline-manager` all subscribe to one or more of
   `QueueNameWebhookEvent`/`QueueNameAgentEvent`/`QueueNameTalkEvent` today via empty-routing-key
   fanout bindings, for purposes unrelated to websocket delivery (agent status updates, queue
   routing, audit-log ingestion — see §6). Their migration to `#`-wildcard bindings on the new
   topic exchange is sketched in §6 step 3 but not fully planned: does each of these three
   services need code changes beyond the bind call (e.g. do they rely on routing-key value
   anywhere, even though they ignore it today), and what is the verification/rollback plan
   specific to each, given they are unrelated teams'/features' code paths from the websocket
   subscription feature this design is otherwise scoped to?
