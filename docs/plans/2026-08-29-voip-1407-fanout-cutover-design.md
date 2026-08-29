# VOIP-1407: Cutover -- remove fanout dual publish and per-service event exchanges

- Date: 2026-08-29
- Ticket: VOIP-1407 (Follow-up C of VOIP-1404; blocks nothing, blocked by VOIP-1404/
  1405/1406, all merged and deployed)
- Status: Draft (design review in progress)
- Depends on: `tasks/todo.md`'s issue analysis (rev.11, approved R10/R11 consecutive
  Approve after an 11-round loop; the empirical foundation -- a two-arm broker
  experiment proving `QueueUnbind` against a missing exchange is safe while
  `QueueBind`/`QueueSubscribe` against one is not -- is authoritative and not
  re-litigated here)
- Scope decision (post-issue-analysis, by explicit user direction): `voip-asterisk-proxy`
  is EXCLUDED from this ticket entirely. Its fanout leg to `asterisk.all.event` "operates
  differently" and stays untouched, exactly as `asterisk.all.event` itself has been
  treated as permanent since VOIP-1404. This resolves what the issue analysis called
  §6 deferred item 1 and materially simplifies §1 below.

## 1. Goal and scope

Two independent code changes, both required, neither depending on the other's code
being live first (though both must be fully deployed, per-service, before §5's
exchange-deletion runbook runs):

- **(a) Publish side**: 55 real dual-publish call sites (27 daemons + their `-control`
  CLIs, enumerated in issue analysis §3) stop fanout-publishing; topic becomes their
  sole delivery path. `voip-asterisk-proxy` (excluded, above) and 3 confirmed-dead
  wiring sites (`transfer-manager`/`transfer-control`/`tts-control`) are handled per §4.
- **(b) Consumer side**: 20 VOIP-1406 services delete the fanout `QueueSubscribe` loop,
  `subscribeTargets`, and `fanoutUnbindTargets`/`QueueUnbind` step from
  `pkg/subscribehandler`; call-manager and timeline-manager additionally delete the
  sentinel defensive `TopicCreate` declare. The `asterisk.all.event` `QueueSubscribe` in
  those same two services is explicitly NOT touched (permanent, feeds from the excluded
  `voip-asterisk-proxy`).

Out of scope: `voip-asterisk-proxy` and `asterisk.all.event` entirely (this revision);
VOIP-1258's `NewNotifyHandlerForExistingExchange` runtime path (confirmed orthogonal,
issue analysis §3.C -- its only production caller is `PublishEventWithRoutingKey`, which
never enters the code this design touches); talk-control's pre-existing broken wiring
(independent follow-up, not created or worsened by this ticket).

## 2. Publish side: `bin-common-handler/pkg/notifyhandler`

### 2.1 Mechanism (issue analysis §6 deferred item 1 -- resolved pre-design by user
direction, restated here as the binding design decision)

`WithGlobalTopicPublish()` is NOT deleted and NOT inverted. It keeps its exact current
shape as an `Option` that sets `h.topicEnabled = true`. Only its **meaning** changes:

| | Today | After this design |
|---|---|---|
| `topicEnabled=true` (55 real call sites) | fanout AND topic (dual) | topic ONLY (fanout removed) |
| `topicEnabled=false` (default; `voip-asterisk-proxy`'s sole non-control caller, plus the 3 dead-wiring sites and talk-control) | fanout ONLY | **unchanged** -- fanout ONLY |

This is the entire mechanism. No opt-out flag, no second constructor, no signature
change to `NewNotifyHandler`. `voip-asterisk-proxy/cmd/asterisk-proxy/main.go:107`'s
call site needs zero edits and is not touched by this PR.

### 2.2 `publishEvent()` control flow change

Current (`bin-common-handler/pkg/notifyhandler/publish.go`):

```go
func (h *notifyHandler) publishEvent(eventType string, dataType string, data json.RawMessage, timeout int, delay int, subscriptionID string) error {
	evt := &sock.Event{...}
	...
	switch {
	case delay > 0:
		return h.publishDelayedEvent(ctx, delay, evt)
	default:
		if err := h.publishDirectEvent(ctx, evt); err != nil {
			return fmt.Errorf(...)
		}
	}
	promNotifyTotal.WithLabelValues(evt.Type).Inc()
	h.publishTopicEvent(evt, subscriptionID) // gated internally on h.topicEnabled
	return nil
}
```

New:

```go
func (h *notifyHandler) publishEvent(eventType string, dataType string, data json.RawMessage, timeout int, delay int, subscriptionID string) error {
	evt := &sock.Event{...}
	...
	switch {
	case delay > 0:
		return h.publishDelayedEvent(ctx, delay, evt)
	case h.topicEnabled:
		// Topic-only path (VOIP-1407): the 55 real publishers. No fanout publish.
		if err := h.publishTopicEventOrErr(ctx, evt, subscriptionID); err != nil {
			return fmt.Errorf("could not publish the event to the global topic exchange. err: %v", err)
		}
	default:
		// Fanout-only path, UNCHANGED: voip-asterisk-proxy and the 3 dead-wiring sites.
		if err := h.publishDirectEvent(ctx, evt); err != nil {
			return fmt.Errorf("could not publish the event. err: %v", err)
		}
	}
	promNotifyTotal.WithLabelValues(evt.Type).Inc()
	return nil
}
```

`publishTopicEventOrErr` is `publishTopicEvent` (§2.4 below) refactored to return an
`error` instead of only logging, since it is now the primary path, not a best-effort
secondary one. `publishDirectEvent`/`publishDirectEventWithKey` (the VOIP-1258 explicit-
routing-key path, used only by webhook-manager) are otherwise unchanged.

**Delayed-publish branch**: left exactly as-is (issue analysis confirmed it is dead code
-- zero production callers pass `delay>0` -- so its behavior under `topicEnabled=true`
is moot; not touched in this PR, tracked as optional cleanup, §7 item 6).

### 2.3 `NewNotifyHandler` construction-time change

Current: unconditionally declares the per-service fanout exchange
(`sockHandler.TopicCreate(queueEvent)`); only additionally declares the global topic
exchange when `topicEnabled`.

New: declare the fanout exchange ONLY when `!topicEnabled` (preserves today's exact
behavior for `voip-asterisk-proxy` and the 3 dead-wiring sites); declare the global
topic exchange ONLY when `topicEnabled` (unchanged from today).

```go
func NewNotifyHandler(sockHandler sockhandler.SockHandler, reqHandler requesthandler.RequestHandler, queueEvent commonoutline.QueueName, publisher commonoutline.ServiceName, opts ...Option) NotifyHandler {
	h := &notifyHandler{sockHandler: sockHandler, reqHandler: reqHandler, queueNotify: queueEvent, publisher: publisher}
	for _, opt := range opts {
		opt(h)
	}

	if !h.topicEnabled {
		// Fanout-only path, unchanged: voip-asterisk-proxy and 3 confirmed-dead sites.
		if err := sockHandler.TopicCreate(string(queueEvent)); err != nil {
			logrus.Errorf("Could not declare the event exchange. err: %v", err)
			return nil
		}
	}

	namespace := commonoutline.GetMetricNameSpace(publisher)
	initPrometheus(namespace)

	if h.topicEnabled {
		h.initGlobalTopicExchange() // unchanged function; see §2.4 for its failure-semantics change
	}

	return h
}
```

`queueEvent`/`h.queueNotify` field: **kept, unchanged signature** (issue analysis §6
deferred item 2 -- resolved). Rationale: `h.queueNotify` is still read by
`publishDirectEvent`/`publishDirectEventWithKey`/`publishDelayedEvent`, all of which
remain live code paths (fanout-only instances still use it; webhook-manager's explicit-
routing-key path still uses it; the dead delayed-publish path is out of scope, §2.2).
Dropping the parameter would be a breaking signature change across all 62 call sites for
zero behavioral gain -- the field is not vestigial, it is actively read by surviving
code. (The 5 VESTIGIAL `QueueName*Event` outline CONSTANTS this frees up on the 55
topic-only call sites -- since they no longer need to name a real per-service fanout
exchange, though they still pass one positionally -- are a separate, much narrower
question, deferred to §7 item 7; the constructor parameter itself stays.)

### 2.4 Failure semantics (issue analysis §6 deferred item 1 formerly, renumbered item 1
in the resolved list -- this is the design's item, distinct from §2.1's resolved
mechanism item)

Today, `initGlobalTopicExchange()`'s declare failure sets `h.topicDisabled = true` and
degrades silently (topic publish becomes a metered no-op; fanout keeps the event
flowing). Once fanout is removed for `topicEnabled=true` instances, there is no fallback
left to degrade to.

**Decision: make it FATAL, matching `NewNotifyHandler`'s own existing fanout-declare
failure precedent** (which already `return`s `nil` and lets the caller's `NewNotifyHandler
== nil` check fail startup). Concretely:

```go
func (h *notifyHandler) initGlobalTopicExchange() bool {
	if err := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), topicExchangeKind); err != nil {
		logrus.Errorf("Could not declare the global topic exchange. err: %v", err)
		return false // caller (NewNotifyHandler) returns nil on false
	}
	return true
}
```

`NewNotifyHandler` becomes:

```go
	if h.topicEnabled {
		if ok := h.initGlobalTopicExchange(); !ok {
			return nil
		}
	}
```

`topicDisabled` field, `promTopicPublishTotal{result="error"}`'s "suppressed publish"
counting behavior tied to it, and the doc comments describing "degrade, don't abort" are
all removed for the `topicEnabled=true` path (they still describe nothing, since there
is no more silent-degrade branch to have them for). This is a deliberate behavior
change from VOIP-1404/1405/1406's dual-publish era, consistent with this ticket's whole
purpose: dual-publish existed BECAUSE degrade-not-abort was safe while fanout was the
backup; once fanout is gone, a topic-exchange declare failure is exactly as serious as
today's fanout-declare failure already is, and should fail exactly the same way.

**Rationale for FATAL over an alternative (e.g. retry-with-backoff, or keep degrading to
a black hole)**: `TopicCreateWithKind` is declared via the shared `sockhandler` helper
with hardcoded `durable=true`/`autoDelete=false`; a mismatch-driven 406 is the only
realistic failure mode in production (the exchange has existed and been declared
successfully by every VOIP-1404/1405/1406 service continuously since VOIP-1404 shipped),
and matches this ticket's design principle throughout (§3.3): once a fallback is
removed, its former failure path becomes a hard failure, not a silent one -- exactly the
reasoning issue analysis §1's rollback table already applies to the consumer side.

### 2.5 `notify_process_time` observability (issue analysis §6 deferred item 3 --
resolved)

**Decision: fold it into `publishTopicEventOrErr`, keeping the existing metric name.**
Today `promNotifyProcessTime` is observed in `publishDirectEvent`,
`publishDirectEventWithKey`, and `publishDelayedEvent`. Add the same
`prometheus.NewTimer`-style observation to `publishTopicEventOrErr`, so the histogram
keeps recording publish latency for the 55 real publishers under their exact existing
metric name and labels (`type`) -- no new metric, no renamed metric, no dashboard
changes needed (issue analysis §2 already confirmed zero external dashboard/alert
consumers reference this name, so there is no compatibility constraint either way; this
choice is purely "preserve the existing signal" over "let it silently go dark").

```go
func (h *notifyHandler) publishTopicEventOrErr(ctx context.Context, evt *sock.Event, subscriptionID string) error {
	start := time.Now()
	defer func() {
		promNotifyProcessTime.WithLabelValues(evt.Type).Observe(float64(time.Since(start).Milliseconds()))
	}()

	if eventtopic.IsPlaceholderSubscriptionID(subscriptionID) {
		promTopicPlaceholderTotal.WithLabelValues(evt.Type).Inc()
	}

	key := eventtopic.RoutingKey(string(h.publisher), evt.Type, subscriptionID)
	if err := h.sockHandler.EventPublish(string(commonoutline.QueueNameEvent), key, evt); err != nil {
		promTopicPublishTotal.WithLabelValues(evt.Type, topicPublishResultError).Inc()
		return err
	}
	promTopicPublishTotal.WithLabelValues(evt.Type, topicPublishResultOK).Inc()
	return nil
}
```

`promTopicPublishTotal`/`promTopicPlaceholderTotal` are unchanged (they already exist
and already correctly attribute topic-path outcomes independent of this metric).

### 2.6 webhook-manager option-surface safeguard (issue analysis §6 deferred item 4 --
resolved)

`NewNotifyHandlerForExistingExchange`'s existing warning comment ("must NOT be enabled
for webhook-manager's scope-first instance -- that would triple-publish webhook events")
is now materially MORE important, not less: with `topicEnabled=true` now meaning
"fanout removed, topic only" instead of "topic added on top of fanout," a future
maintainer adding `WithGlobalTopicPublish()` to this instance would not just double-
publish (the old risk) -- since this instance's `h.queueNotify` already targets a
topic-kind exchange (`QueueNameWebhookEventTopic`), enabling the option would attempt to
ALSO run this instance through `publishTopicEventOrErr`'s `bin-manager.event`-targeting
path, in addition to its actual traffic via `PublishEventWithRoutingKey`, on top of a
`topicEnabled=true` handler that no longer even has a fanout declare to fall back on if
the (redundant) `bin-manager.event` declare inside `NewNotifyHandlerForExistingExchange`
somehow diverged. **Decision: strengthen the comment; no code-level guard is added**
(a runtime assertion would need to fire in `NewNotifyHandlerForExistingExchange` itself,
which is more invasive than this ticket's stated scope -- issue analysis confirmed ZERO
current call sites pass the option to this constructor, so there is nothing to migrate,
only a comment to update):

```go
// NOTE (VOIP-1407): WithGlobalTopicPublish is valid here in principle, but enabling it
// would be actively harmful post-VOIP-1407: topicEnabled now means "topic-ONLY, fanout
// removed" (not "topic added on top of fanout" as when this note was written). This
// instance's queueNotify already targets a topic-kind exchange
// (QueueNameWebhookEventTopic) and its production traffic already goes exclusively
// through PublishEventWithRoutingKey. Enabling the option would run this instance
// through publishTopicEventOrErr's bin-manager.event path IN ADDITION, with no fanout
// fallback to degrade to if anything about that additional path fails. Nothing enables
// it today (verified); keep it that way.
```

## 3. Consumer side: 20 VOIP-1406 services

### 3.1 The change, per service

In `pkg/subscribehandler/main.go`, delete:
1. The `subscribeTargets` package-level var (or, for webhook-manager, its comma-joined-
   string wiring at `main.go:97,105`; for timeline-manager, its package-level
   `[]string` including `QueueNameAsteriskEventAll` -- see §3.2 for the one entry that
   MUST survive this deletion).
2. The fanout `QueueSubscribe` loop in `Run()`.
3. `fanoutUnbindTargets` and the `QueueUnbind` step that consumed it.
4. In `cmd/*-manager/main.go`: the wiring that fed `subscribeTargets` into the
   `subscribeHandler` constructor (dead parameter once #1 is gone).

`topicPatterns`/`QueueBind` (the VOIP-1406 topic-binding block) becomes the sole intake
mechanism -- unconditional, not gated behind "if the fanout declare failed, stay on
fanout" (that branch no longer exists, matching §3.3's failure-semantics decision).

### 3.2 call-manager and timeline-manager: two exceptions, not one

- **Sentinel defensive `TopicCreate` declare**: DELETED (issue analysis §2b, design doc
  citation VOIP-1406 `:124-130`: "The declare is deleted in VOIP-1407 together with the
  fanout QueueSubscribe lines").
- **`asterisk.all.event` `QueueSubscribe`**: **NOT DELETED.** This is the one fanout
  `QueueSubscribe` line, across all 20 services, that survives this ticket -- it feeds
  from `voip-asterisk-proxy`, which is excluded from this ticket's scope entirely (§0).
  Concretely: the loop deletion in §3.1 item 2 must NOT remove the specific iteration
  that subscribes to `QueueNameAsteriskEventAll`; if the loop is deleted wholesale, this
  one `QueueSubscribe(subscribeQueue, string(commonoutline.QueueNameAsteriskEventAll))`
  call is re-added as a standalone statement immediately before `go ConsumeMessage`, in
  the same position the loop used to occupy.

### 3.3 Failure semantics (issue analysis §6 deferred item 5, formerly item 6 -- resolved)

**Decision: FATAL, consistent with §2.4's publish-side ruling.** Today's `Run()`
degrades ("stay fully on fanout subscriptions") if the topic-exchange declare or any
`QueueBind` fails. Once the fanout `QueueSubscribe` loop is deleted, there is nothing
left to degrade TO -- the service would silently intake zero events forever, which is
strictly worse than a loud boot failure an operator/orchestrator will notice and
restart-loop on. Concretely, in `Run()`:

```go
	if err := h.sockHandler.QueueCreate(h.subscribeQueue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for subscribeHandler. err: %v", err)
	}

	// call-manager/timeline-manager only: the one retained fanout leg.
	if err := h.sockHandler.QueueSubscribe(h.subscribeQueue, string(commonoutline.QueueNameAsteriskEventAll)); err != nil {
		return fmt.Errorf("could not subscribe to the asterisk fanout exchange. err: %v", err)
	}

	if err := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic"); err != nil {
		return fmt.Errorf("could not declare the global topic exchange. err: %v", err)
	}

	for _, pattern := range topicPatterns {
		if err := h.sockHandler.QueueBind(h.subscribeQueue, pattern, string(commonoutline.QueueNameEvent), false, nil); err != nil {
			return fmt.Errorf("could not bind the topic pattern. pattern: %s, err: %v", pattern, err)
		}
	}

	go func() { ... ConsumeMessage ... }()
```

No more "bound so far, roll back on partial failure" logic either -- that machinery
existed specifically to protect the fanout fallback during the all-or-nothing bind
attempt (VOIP-1406 §2's D-RULE); with no fallback to protect, a bind failure simply
fails `Run()` outright, and the orchestrator's normal pod-restart-with-backoff handles
recovery, matching every other startup-declare failure in this codebase (e.g. `QueueCreate`
above, unchanged).

### 3.4 Stale-binding precondition, pre-rollout AND post-rollout (issue analysis §2b,
carried forward verbatim as this design's rollout gate)

Before this consumer-side change deploys to a given service: confirm ZERO stray fanout
bindings for that service (broker-binding inspection, same technique as issue analysis
§1). After it deploys: re-confirm, since a rolling-deploy window (an old-image pod's
`Run()` re-subscribing to fanout mid-rollout) or an image rollback could reintroduce a
stray binding that -- once this service's fanout-unbind machinery is gone -- has no
automatic remediation path left (issue analysis §2b's finding: VOIP-1406's
`QueueUnbind` loop was the only automatic self-heal for this; the manual `rabbitmqadmin`
remediation documented in VOIP-1406 §7 still works, it just needs a human to run it).

## 4. Per-call-site disposition (issue analysis §3's 62 sites, resolved individually)

| Sites | Disposition |
|---|---|
| 55 dual-publish (27 daemons + `-control` CLIs) | Topic-only per §2. |
| 1 (`voip-asterisk-proxy/cmd/asterisk-proxy/main.go:107`) | **Excluded, zero changes** (§0/§2.1). |
| 2 (`bin-transfer-manager/cmd/{transfer-manager,transfer-control}/main.go`) | **Deleted** (§7 item 4 -- confirmed dead: zero calls anywhere in `pkg/transferhandler`). Delete the `notifyHandler` field, its constructor parameter, and both `NewNotifyHandler` construction sites. |
| 1 (`bin-tts-manager/cmd/tts-control/main.go:38`) | **Deleted**, same reasoning (`pkg/ttshandler` never calls it). |
| 1 (`bin-talk-manager/cmd/talk-control/main.go:57`) | **Untouched** -- pre-existing, independent bug (empty exchange name), out of scope for this ticket, tracked separately. |
| 2 (`bin-webhook-manager/cmd/{webhook-manager,webhook-control}/main.go`, `NewNotifyHandlerForExistingExchange`) | **Comment-only change** (§2.6). |

**Decision on the 3 dead-wiring deletions (issue analysis §6 deferred item 6,
formerly item 7 -- resolved)**: delete them in this same PR. They are trivial (remove an
unused field, an unused constructor parameter, and the `NewNotifyHandler` call plus its
now-unused imports), the fact of their deadness is already established and re-verified
across 5 issue-analysis revisions, and leaving them would mean carrying 3 more
`WithGlobalTopicPublish()`-less call sites through this refactor's mechanical sweep for
no reason -- lower total review surface to delete them now than to re-justify leaving
them.

## 5. Per-service fanout exchange deletion (issue analysis §4 item 4, operational
runbook -- not application code)

Exactly the 28 exchanges issue analysis §1 enumerated (`bin-manager.agent-manager.event`
through `bin-manager.webchat-manager.event`, alphabetically -- full list in issue
analysis §1, not repeated here since it is normative there). Explicitly NOT included:
`asterisk.all.event` (permanent), `bin-manager.event` (the topic exchange itself),
`bin-manager.webhook-manager.event.topic` (VOIP-1258, different exchange),
`bin-manager.delay` (unrelated retry exchange), and the 5 `QueueName*Event`-family
exchanges that are already absent from the broker (`api-manager`, `rag-manager`,
`timeline-manager`, `user-manager` -- never existed; `webhook-manager` -- already gone
since VOIP-1296, cause unestablished but irrelevant since there is nothing to delete).

**Precondition (issue analysis §4 item 4, widened per that document's R2 finding)**:
every publisher daemon+control binary (§2/§4 above) AND every one of the 20 consumer
services (§3) that declares, binds, or subscribes to one of these 28 exchanges must be
rebuilt/redeployed with that code removed, with a CONFIRMED restart-survival check per
service (§3.4), before this runbook runs.

**Mechanism**: a documented one-time `rabbitmqadmin`/management-API cleanup, matching
VOIP-1406's stale-binding-runbook precedent (`docs/reference/rabbitmq-queues-
reference.md`'s existing runbook section). Exact commands (per exchange:
`rabbitmqadmin delete exchange name=bin-manager.<x>-manager.event`) to be added to that
doc as part of this PR's documentation update (§6).

**Post-deletion re-check**: re-list exchanges/bindings after a defined soak window (same
technique as issue analysis §1's live queries) to catch resurrection from an
un-redeployed `-control` CLI invocation run between deploy and deletion.

## 6. Docs

`docs/reference/rabbitmq-queues-reference.md`: rewrite the dual-publish framing
throughout (routing-key section, publish-path section, declaration-invariant section,
"Consumer state (VOIP-1406)" note) for a topic-only world for the 55+20 services this
ticket touches, while explicitly documenting `asterisk.all.event`/`voip-asterisk-proxy`
as a permanently-retained, deliberately-excluded exception (not migration debris);
add the exchange-deletion runbook commands (§5). Every touched service's
`docs/architecture.md`/`docs/dependencies.md`: publish-side prose for the ~27 publisher
services, events-section prose for the 20 consumer services (union, not sum -- most
services are both). `bin-common-handler/docs/architecture.md`'s `notifyhandler` section:
update to describe the topic-only default and the `WithGlobalTopicPublish()` meaning
change.

## 7. Remaining open items (all non-blocking, tracked, none gate this PR)

1. `queueEvent`/`h.queueNotify` constructor field: kept (§2.3) -- not actually open,
   listed here for completeness of what issue analysis §6 deferred and this design
   resolved.
2. The 5 vestigial `QueueName*Event` outline constants (4 fully dead + 1 legacy,
   issue analysis §4 item 1/§1): NOT deleted in this PR. Deleting them would touch the
   8 gomock expectations and 3 doc mentions issue analysis catalogued, for a cosmetic
   win unrelated to this ticket's actual goal; tracked as an optional follow-up.
3. Delayed-publish dead code (`publishDelayedEvent`, `DelaySecond`/`DelayMinute`/
   `DelayHour` in `notifyhandler`): not touched (§2.2) -- optional follow-up.
4. transfer-manager/transfer-control/tts-control dead-wiring deletion: **decided IN
   scope, §4** -- not actually open, listed here for completeness.
5. talk-control's broken wiring: independent follow-up, explicitly out of scope.
6. Whether to also delete the now-unused `topicDisabled` field's remaining references
   for the `topicEnabled=false` path (it is only ever set on the `topicEnabled=true`
   path in current code, so this is likely a pure removal with no live-path impact --
   confirm during implementation, not a design-level decision).

## 8. Testing

- `bin-common-handler/pkg/notifyhandler`: table-driven tests for `publishEvent()`'s new
  branch split (topicEnabled true/false × delay zero/nonzero), `NewNotifyHandler`'s new
  conditional-declare logic (both branches), `initGlobalTopicExchange`'s fatal-on-
  failure behavior (constructor returns nil), and `publishTopicEventOrErr`'s metric
  observations (reuse the existing `promNotifyProcessTime`/`promTopicPublishTotal`
  assertions, extended to the topic-only path).
- Each of the 20 consumer services: update `binding_golden_test.go` (drop
  fanout-target pins) and the `Run()` sequencing test (gomock `InOrder`: `QueueCreate`
  -> [asterisk `QueueSubscribe`, call-manager/timeline-manager only] -> `TopicCreateWithKind`
  -> `QueueBind`×N -> `ConsumeMessage`; failure-path cases updated to assert `Run()`
  returns the error immediately, matching §3.3, replacing the old roll-back-and-degrade
  assertions).
- Live verification (post-merge, pre-deletion): the restart-survival + stale-binding
  sweep from §3.4, run per service as each is deployed.
