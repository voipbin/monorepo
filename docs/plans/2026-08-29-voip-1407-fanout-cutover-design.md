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

## 0. Scope exclusion (post-issue-analysis, by explicit user direction)

`voip-asterisk-proxy` is EXCLUDED from this ticket entirely. Its fanout leg to
`asterisk.all.event` "operates differently" and stays untouched, exactly as
`asterisk.all.event` itself has been treated as permanent since VOIP-1404. This resolves
what issue analysis `tasks/todo.md` §6 lists as **resolved item 3** ("§4 item 1
mechanism (RESOLVED rev.11, was deferred item 1)" -- `tasks/todo.md:673`; it was deferred
item 1 in an earlier revision and was moved to the resolved list once the user's
direction settled it) and materially simplifies §1 below.

## 1. Goal and scope

Two independent code changes, both required, neither depending on the other's code
being live first (though both must be fully deployed, per-service, before §5's
exchange-deletion runbook runs):

- **(a) Publish side**: 55 real dual-publish call sites (27 daemons + their `-control`
  CLIs, enumerated in issue analysis §3) stop fanout-publishing; topic becomes their
  sole delivery path. `voip-asterisk-proxy` (excluded, above) and 3 confirmed-dead
  wiring sites (`transfer-manager`/`transfer-control`/`tts-control`) are handled per §4.
  `talk-control`'s one-line broken-exchange-name fix (§2.3.1, new -- R2 finding,
  CRITICAL) is also handled per §4.
- **(b) Consumer side**: 20 VOIP-1406 services delete the fanout `QueueSubscribe` loop,
  `subscribeTargets`, and `fanoutUnbindTargets`/`QueueUnbind` step from
  `pkg/subscribehandler`; call-manager and timeline-manager additionally delete the
  sentinel defensive `TopicCreate` declare. The `asterisk.all.event` `QueueSubscribe` in
  those same two services is explicitly NOT touched (permanent, feeds from the excluded
  `voip-asterisk-proxy`).

Out of scope: `voip-asterisk-proxy` and `asterisk.all.event` entirely (this revision);
VOIP-1258's `NewNotifyHandlerForExistingExchange` runtime path (confirmed orthogonal,
issue analysis §3.C -- its only production caller is `PublishEventWithRoutingKey`, which
never enters the code this design touches). **talk-control's broken wiring is NOT out of
scope** (corrected -- R2 finding, CRITICAL; an earlier revision of this design called it
out of scope, which was true for issue analysis but stopped being true the moment §2.3
introduced `logrus.Fatalf`, see §2.3.1): its empty exchange name is fixed by this PR as
a narrow, directly-motivated companion change, not left as an independent follow-up.

## 2. Publish side: `bin-common-handler/pkg/notifyhandler`

### 2.1 Mechanism (issue analysis §6 resolved item 3, formerly deferred item 1 -- see
§0 -- resolved pre-design by user direction, restated here as the binding design
decision)

`WithGlobalTopicPublish()` is NOT deleted and NOT inverted. It keeps its exact current
shape as an `Option` that sets `h.topicEnabled = true`. Only its **meaning** changes:

| | Today | After this design |
|---|---|---|
| `topicEnabled=true` (55 real call sites) | fanout AND topic (dual) | topic ONLY (fanout removed) |
| `topicEnabled=false` (default; `voip-asterisk-proxy`'s sole non-control caller, plus the 3 dead-wiring sites and talk-control) | fanout ONLY | **unchanged** -- fanout ONLY |

This is the entire mechanism. No opt-out flag, no second constructor, no signature
change to `NewNotifyHandler`. `voip-asterisk-proxy/cmd/asterisk-proxy/main.go:107`'s
call site needs zero edits and is not touched by this PR.

**Three in-code comments now describe the OLD meaning and must be rewritten as part of
this change (R1 finding 5, plus one more found in R2 -- finding 3), since they are the
primary documentation a future reader of this package encounters:**
- `WithGlobalTopicPublish()`'s doc comment (`main.go:175-186`) currently reads "on top
  of the existing fanout publish, every event is **also** published..." and "the
  fanout publish stays the system of record while dual publish lasts." Rewrite to state
  the new meaning: enabling this option makes the instance topic-ONLY (no fanout
  publish, no fanout exchange declared); the default (option omitted) is unchanged
  fanout-only.
- `main.go:70-76`'s comment (on `initPrometheus`'s guard, VOIP-1258 context) claims
  "webhook-manager, webhook-control" make "a fanout-bound `NewNotifyHandler` call in the
  same process" alongside their topic-only instance. Verified false (issue analysis §3,
  R2 finding 7): neither constructs anything but
  `NewNotifyHandlerForExistingExchange`. Correct the comment while touching this file.
- **`main.go:64-66`'s comment (R2 finding 3)** on the `promTopicPublishTotal`/
  `promTopicPlaceholderTotal` var block: "the topic publish path must **never** touch
  `promNotifyTotal`/`promNotifyProcessTime`, otherwise the dual publish would double-
  count the existing fanout metrics." §2.2 makes the topic-only branch increment
  `promNotifyTotal` (it is no longer a *dual* publish, so there is nothing left to
  double-count), and §2.5 makes `publishTopicEventOrErr` observe
  `promNotifyProcessTime`. Both directly contradict this comment once fanout is gone;
  rewrite it to state the new invariant (these counters/histogram are shared across
  both paths now that there is only one active path per instance, so no double-counting
  is possible by construction). The near-identical comment on old `publishTopicEvent`
  itself (`publish.go:186-189`, "reusing them would pollute the existing fanout
  metrics") describes the same now-obsolete constraint and is replaced by §2.5's
  rewritten function, which should carry an equivalent updated doc comment explaining
  why observing `promNotifyProcessTime` is now correct rather than a suppressed
  earlier reason not to.

### 2.2 `publishEvent()` control flow change

Current (`bin-common-handler/pkg/notifyhandler/publish.go`):

```go
func (h *notifyHandler) publishEvent(eventType string, dataType string, data json.RawMessage, timeout int, delay int, subscriptionID string) error {
	evt := &sock.Event{...}
	...
	switch {
	case delay > 0:
		if err := h.publishDelayedEvent(ctx, delay, evt); err != nil {
			return fmt.Errorf("could not publish the delayed event. err: %v", err)
		}
		return nil
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
		if err := h.publishDelayedEvent(ctx, delay, evt); err != nil {
			return fmt.Errorf("could not publish the delayed event. err: %v", err)
		}
		return nil
	case h.topicEnabled:
		// Topic-only path (VOIP-1407): the 55 real publishers. No fanout publish.
		// publishTopicEventOrErr's own error already carries full context (including
		// routing_key, §2.5) -- returned directly, not re-wrapped, to avoid a doubled
		// "could not publish the event to the global topic exchange" prefix (R2 finding
		// 11).
		if err := h.publishTopicEventOrErr(ctx, evt, subscriptionID); err != nil {
			return err
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

`publishTopicEventOrErr` is `publishTopicEvent` (§2.5 below) refactored to return an
`error` instead of only logging, since it is now the primary path, not a best-effort
secondary one. `publishDirectEvent`/`publishDirectEventWithKey` (the VOIP-1258 explicit-
routing-key path, used only by webhook-manager) are otherwise unchanged.

**Delayed-publish branch**: left exactly as-is (issue analysis confirmed it is dead code
-- zero production callers pass `delay>0` -- so its behavior under `topicEnabled=true`
is moot; not touched in this PR, tracked as optional cleanup, §7 item 3).

### 2.3 `NewNotifyHandler` construction-time change

Current: unconditionally declares the per-service fanout exchange
(`sockHandler.TopicCreate(queueEvent)`); on failure, `return`s `nil` and logs. Only
additionally declares the global topic exchange when `topicEnabled`, via
`initGlobalTopicExchange()` -- a SHARED private method also called, unconditionally,
from the SEPARATE `NewNotifyHandlerForExistingExchange` constructor (`main.go:250`);
`initGlobalTopicExchange()` self-guards internally (`if !h.topicEnabled { return }`,
`main.go:269-271`), so today it is a safe no-op for webhook-manager's `topicEnabled=false`
`NewNotifyHandlerForExistingExchange` instance.

**Corrected finding (R1 finding 1 -- CRITICAL): "return nil and let the caller's
`== nil` check fail startup" is NOT what happens today, for either failure path.**
Repo-wide sweep of all 62 `NewNotifyHandler(...)` and `NewNotifyHandlerForExistingExchange(...)`
call sites: **zero** check the return value for `nil`. Every one assigns straight into
`notifyHandler := notifyhandler.NewNotifyHandler(...)` and passes it directly into a
downstream constructor. The REAL current behavior of a fanout-declare failure is: the
process boots normally, and the first call to any `NotifyHandler` method (frequently
inside `PublishWebhookEvent`'s fire-and-forget goroutine) panics on a nil-interface
method call -- at an arbitrary later time, often under load, not at boot; for a
`-control` binary that never happens to publish, the failure is invisible forever. This
is a **pre-existing latent bug**, not deliberate fail-fast behavior, and this design
must not build on top of a false premise about it.

**Decision: fix both failure paths (fanout-declare in `NewNotifyHandler`, topic-declare
in `initGlobalTopicExchange`) to call `logrus.Fatalf` directly, rather than returning
`nil`.** This is the minimal-footprint option that genuinely halts startup without a
62-call-site signature change to `(NotifyHandler, error)` (rejected: that ripples into
every one of the 62 call sites' error-handling for zero behavior change beyond what
`Fatalf` already achieves). `logrus.Fatalf`/`log.Fatalf` at process-startup failure is
an established pattern already used elsewhere in this monorepo's `cmd/` entrypoints
(e.g. `bin-agent-manager/cmd/agent-manager/main.go`, `bin-registrar-manager/cmd/
registrar-manager/main.go`); using it directly inside the shared constructor is a
narrow, deliberate exception to "handlers return errors, callers log/fatal them,"
justified because no caller of this specific constructor has ever handled its failure
and 62 call sites is too large a blast radius to retrofit for this ticket.

Fixing the FANOUT-declare path's identical latent bug is in scope, disclosed here
explicitly (not silent scope creep): it is the same function, the exact same class of
bug (an unchecked `nil` return), discovered as a direct consequence of writing this
design, and leaving it unfixed while calling the topic-declare path "matching
precedent" would be citing a bug as the standard to match.

```go
func NewNotifyHandler(sockHandler sockhandler.SockHandler, reqHandler requesthandler.RequestHandler, queueEvent commonoutline.QueueName, publisher commonoutline.ServiceName, opts ...Option) NotifyHandler {
	h := &notifyHandler{sockHandler: sockHandler, reqHandler: reqHandler, queueNotify: queueEvent, publisher: publisher}
	for _, opt := range opts {
		opt(h)
	}

	if !h.topicEnabled {
		// Fanout-only path, unchanged: voip-asterisk-proxy and 3 confirmed-dead sites.
		if err := sockHandler.TopicCreate(string(queueEvent)); err != nil {
			logrus.Fatalf("Could not declare the event exchange. err: %v", err)
		}
	}

	namespace := commonoutline.GetMetricNameSpace(publisher)
	initPrometheus(namespace)

	// initGlobalTopicExchange is UNCHANGED here: it keeps its existing internal
	// `if !h.topicEnabled { return }` guard (§2.4), so this call is a correct no-op for
	// !topicEnabled instances -- no caller-side gating change, no divergent behavior at
	// NewNotifyHandlerForExistingExchange's :250 call site.
	h.initGlobalTopicExchange()

	return h
}
```

`queueEvent`/`h.queueNotify` field: **kept, unchanged signature** (issue analysis §6
deferred item 2 -- resolved). Rationale: `h.queueNotify` is still read by
`publishDirectEvent`/`publishDirectEventWithKey`/`publishDelayedEvent`, all of which
remain live code paths (fanout-only instances still use it; webhook-manager's explicit-
routing-key path still uses it; the dead delayed-publish path is out of scope, §2.2).
Dropping the parameter would require a second constructor for the 55 topic-only call
sites (Go has no optional positional parameters), retaining the original for the 7
fanout-only/existing-exchange sites anyway -- for zero behavioral gain over keeping one
constructor. **On the 55 topic-only call sites, the value passed for `queueEvent` is
stored but never read by any code path they can reach** (R1 finding 7) -- add a doc
comment on `NewNotifyHandler` making this explicit: "`queueEvent` is ignored on the
`WithGlobalTopicPublish()` path -- no per-service fanout exchange is declared or
published to there." (The 5 VESTIGIAL `QueueName*Event` outline CONSTANTS this exposes
on those 55 call sites -- since they name a per-service fanout exchange this ticket
deletes from the broker (§5), though they are still passed positionally -- are a
separate, much narrower question, deferred to §7 item 2; the constructor parameter
itself stays regardless of that follow-up.)

### 2.3.1 Companion fix: `talk-control`'s empty exchange name (R2 finding -- CRITICAL,
newly discovered as a direct consequence of §2.3's `logrus.Fatalf` decision)

`bin-talk-manager/cmd/talk-control/main.go:57`:

```go
notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, nil, "", serviceName)
```

`queueEvent` is the empty string, and no `WithGlobalTopicPublish()` option is passed, so
this instance is `topicEnabled=false` and enters exactly the branch §2.3 changed:
`sockHandler.TopicCreate(string(queueEvent))` calls
`ExchangeDeclare("", "fanout", true, false, false, false, nil)`
(`bin-common-handler/pkg/rabbitmqhandler/topic.go:5-12`) -- declaring the AMQP *default*
exchange (empty name), which every broker rejects
(`ACCESS_REFUSED -- operation not permitted on the default exchange`). This declare
already fails on every `talk-control` invocation today.

**Today's actual behavior (not "boots fine," corrected)**: `talk-control`'s
`notifyHandler` is genuinely live code -- unlike the 3 confirmed-dead sites in §4,
`pkg/{chathandler,messagehandler,reactionhandler,participanthandler}` all call
`h.notifyHandler.PublishWebhookEvent(...)` on their create/update/delete paths (e.g.
`chathandler/chat.go:165,263,292`). Per §2.3's own corrected finding (the nil return is
never checked), `talk-control` boots successfully today (`chat list`, `message list`,
`participant list` never reach the notify call and work fine), and only the
write-path subcommands that DO reach it (`reaction add`, per `bin-talk-manager/
CLAUDE.md`) panic on a nil-interface method call -- but only when actually invoked.

**Why §2.3 makes this materially worse, not merely "already broken"**: once the same
declare failure calls `logrus.Fatalf` instead of logging and returning `nil`, the
process exits during `initHandlers()`, before any subcommand runs. Every `talk-control`
invocation breaks, including the three read-only subcommands that work perfectly today.
This is a strictly larger blast radius than the pre-existing bug, introduced by this
ticket's own change -- not something a "not created or worsened by this ticket" framing
can honestly claim (§1, corrected).

**Decision: fix the call site in this same PR**, following the exact precedent §2.3
already set for the fanout-declare latent bug ("the same function, the exact same class
of bug... discovered as a direct consequence of writing this design"). This is a
one-line change with an existing, correct value to use --
`bin-talk-manager/cmd/talk-manager/main.go:82-87` (the daemon) already constructs
correctly with `commonoutline.QueueNameTalkEvent` (which has been successfully declared
in production continuously since before VOIP-1404, per §2.4's rationale for why FATAL is
safe):

```go
notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, nil, commonoutline.QueueNameTalkEvent, serviceName)
```

(requires adding the `monorepo/bin-common-handler/models/outline` import, aliased
`commonoutline` to match this file's sibling services' convention -- not currently
imported in this file). `WithGlobalTopicPublish()` is deliberately NOT added here: doing
so would move `talk-control` from `topicEnabled=false` to `=true`, a functional scope
expansion (this CLI would start publishing to `bin-manager.event`) this ticket does not
need and issue analysis never characterized as one of the 55 real dual-publish sites --
the minimal fix is exactly the exchange-name correction, nothing else.

**Explicitly NOT fixed by this companion change (separate, unrelated, genuinely out of
scope)**: the `nil` `reqHandler` argument on the same line, and `serviceName`'s type
(untyped string constant `"talk-manager"` rather than
`commonoutline.ServiceNameTalkManager`) -- neither interacts with the declare-failure/
`Fatalf` mechanism this ticket touches, and fixing them is not required to prevent the
boot-failure regression this section exists to close.

### 2.4 `initGlobalTopicExchange` failure semantics (issue analysis §6 deferred item 1)

Today: `initGlobalTopicExchange()` self-guards on `!h.topicEnabled` (unchanged, §2.3),
and on a `TopicCreateWithKind` failure sets `h.topicDisabled = true` and degrades
silently (topic publish becomes a metered no-op; fanout keeps the event flowing). Once
fanout is removed for `topicEnabled=true` instances, there is no fallback left to
degrade to.

**Decision: FATAL, via the same `logrus.Fatalf` mechanism as §2.3's fanout-declare
fix** (not a `bool`-returning refactor with caller-side gating -- that would require
touching the caller-side guard, which §2.3 established must NOT change to avoid
affecting `NewNotifyHandlerForExistingExchange`'s :250 call site):

```go
// initGlobalTopicExchange declares the global topic exchange `bin-manager.event` when the
// WithGlobalTopicPublish option is enabled (VOIP-1404 design §3/§5.2).
//
// VOIP-1407: a declare failure here is now FATAL for topicEnabled=true instances (55 real
// call sites) -- there is no fanout fallback left to degrade to. logrus.Fatalf halts the
// process directly (see §2.3 for why: no caller of NewNotifyHandler/
// NewNotifyHandlerForExistingExchange has ever checked the return value for nil).
func (h *notifyHandler) initGlobalTopicExchange() {
	if !h.topicEnabled {
		return
	}

	if err := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), topicExchangeKind); err != nil {
		logrus.Fatalf("Could not declare the global topic exchange. err: %v", err)
	}
}
```

`topicDisabled` field, `promTopicPublishTotal{result="error"}`'s "suppressed publish"
counting behavior tied to it, and the doc comments describing "degrade, don't abort" are
all removed (they described a silent-degrade branch that no longer exists). This is a
deliberate behavior change from VOIP-1404/1405/1406's dual-publish era, consistent with
this ticket's whole purpose: dual-publish existed BECAUSE degrade-not-abort was safe
while fanout was the backup; once fanout is gone, a topic-exchange declare failure is
exactly as serious as a fanout-declare failure, and now fails exactly the same way
(§2.3) instead of differently.

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
		return fmt.Errorf("could not publish the event to the global topic exchange. routing_key: %s, err: %v", key, err)
	}
	promTopicPublishTotal.WithLabelValues(evt.Type, topicPublishResultOK).Inc()
	return nil
}
```

**Diagnostic-log fidelity (R1 finding 9)**: the current `publishTopicEvent`
(`publish.go:217`) logs `routing_key` alongside the error -- the single most
diagnostic field on this path, since it is the whole point of the topic migration.
The refactor above preserves it by folding it into the returned error string (rather
than a separate `log.Errorf` call, since this function's caller in §2.2 now returns the
error onward instead of swallowing it) so the routing key survives all the way to
`publishEvent`'s caller instead of being dropped at the `promTopicPublishTotal`
increment point.

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
topic-kind exchange (`QueueNameWebhookEventTopic`), enabling the option would ALSO
run this instance through `publishTopicEventOrErr`'s `bin-manager.event`-targeting
path (via `initGlobalTopicExchange`'s existing self-guard, `main.go:270-272`, which
fires -- returns early -- only while `topicEnabled=false`; enabling the option is
precisely what makes the guard STOP firing and lets the declare through, §2.3, R2
finding 12 correcting an earlier backwards phrasing here), in addition to its actual
traffic via `PublishEventWithRoutingKey` --
**corrected premise (R1 finding 3)**: `initGlobalTopicExchange()`'s guard means there is
NO redundant `bin-manager.event` declare inside `NewNotifyHandlerForExistingExchange`
TODAY (`topicEnabled=false` there, so the call at `main.go:250` is currently a no-op);
the hazard is not a divergent EXISTING declare, it is that enabling the option would
newly CREATE one where none exists now, immediately upstream of the same handler's
`publishEvent()` starting to run through the (now fatal-on-failure, §2.4) topic path in
addition to its `PublishEventWithRoutingKey` traffic. **Decision: strengthen the
comment; no code-level guard is added** (a runtime assertion would need to fire in
`NewNotifyHandlerForExistingExchange` itself, which is more invasive than this ticket's
stated scope -- issue analysis confirmed ZERO current call sites pass the option to
this constructor, so there is nothing to migrate, only a comment to update):

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

In `pkg/subscribehandler/main.go` (plus `cmd/*-manager/main.go` where item 1 below is
constructed outside the package -- see the corrected variance table, R2 finding 2),
delete:
1. Whatever constructs the fanout target list feeding `Run()`'s subscribe loop --
   concretely a `cmd/*-manager/main.go`-local slice literal passed in as a constructor
   parameter for 18 of 20 services, but a genuine `pkg/subscribehandler`-package-level
   `subscribeTargets` var (with no `cmd/` wiring at all) for timeline-manager, and a
   comma-joined-string variant of the cmd-local pattern for webhook-manager -- see §3.2
   for the one entry (`QueueNameAsteriskEventAll`) that MUST survive this deletion.
2. The fanout `QueueSubscribe` loop in `Run()`.
3. `fanoutUnbindTargets` and the `QueueUnbind` step that consumed it.
4. For the 18 cmd-local services: the `cmd/*-manager/main.go` slice literal and the
   constructor-parameter wiring that passed it in (dead once #1/#2 are gone). Does NOT
   apply to timeline-manager (no `cmd/` wiring exists to delete -- issue analysis §2b)
   or webhook-manager (its `cmd/` deletion is the `strings.Join` construction, per the
   table below, not a plain slice literal).

`topicPatterns`/`QueueBind` (the VOIP-1406 topic-binding block) becomes the sole
consumer-side intake mechanism for 17 of 20 services -- unconditional, not gated behind
"if the fanout declare failed, stay on fanout" (that branch no longer exists, matching
§3.3's failure-semantics decision). The other 3 (call-manager, agent-manager,
timeline-manager) additionally retain one or both of §3.2's exceptions.

**Per-service variance in how item 1/4 above are expressed (R1 finding 4; corrected
per R2 finding 2 -- the original table had the typical/atypical cases backwards and
misidentified call-manager's citation)**:

| Service | `subscribeTargets` shape | Notes |
|---|---|---|
| 18 of 20 (typical -- e.g. call-manager, agent-manager, tag-manager, billing-manager) | `cmd/*-manager/main.go`-local slice literal (e.g. `subscribeTargets := []string{...}`, `bin-call-manager/cmd/call-manager/main.go:180-185`), passed as a constructor parameter into a `pkg/subscribehandler` struct field (e.g. `bin-call-manager/pkg/subscribehandler/main.go:63,121,130`) | Delete the `cmd/`-local slice literal and the constructor-parameter wiring, per items 1/4. `string(commonoutline.QueueName*Event)` conversions some services use when building the slice (e.g. call-manager) disappear along with it -- nothing extra to do. |
| timeline-manager | Genuine package-level `var subscribeTargets = []commonoutline.QueueName{...}` (`bin-timeline-manager/pkg/subscribehandler/main.go:27`, typed, not `[]string`) -- **no `cmd/` wiring exists** (`NewSubscribeHandler(sockHandler, dbHandler)` takes no targets parameter, `main.go:119-128`) | Delete the package-level var only; there is no item-4 counterpart for this service. Also has a two-value `Run(ctx) (<-chan struct{}, error)` signature (`main.go:109,134`, not the one-value `Run(ctx) error` the rest of §3.1/§3.3 assume) -- every `return err` in §3.3's rewritten `Run()` becomes `return nil, err` for this service specifically. |
| webhook-manager | Comma-joined `string` constructor parameter (`main.go:97`), `strings.Split(h.subscribesTargets, ",")` at the top of `Run()` (`main.go:126`); built via `strings.Join([]string{...}, ",")` in `cmd/webhook-manager/main.go:193` | Delete the `strings.Join` construction in `cmd/`, the constructor parameter, and the `strings.Split` call in `Run()` -- same net effect as item 1/4, different concrete lines from the typical case. |

### 3.2 Per-service exceptions to the generic change in §3.1

Two independent mechanisms sit outside the generic subscribeTargets/topicPatterns
machinery §3.1 describes, in two different (overlapping) service pairs. Neither is
touched by this ticket. **(CRITICAL, R1 finding 2)**: an earlier draft of this design
omitted the second one entirely; following §3.1 as originally written would have
deleted a live, in-production binding and caused webhook-event intake loss for both
services it applies to. Both are called out explicitly below so the implementation
sweep does not regress them.

**call-manager and timeline-manager -- the asterisk fanout leg:**

- **Sentinel defensive `TopicCreate` declare**: DELETED (issue analysis §2b, design doc
  citation VOIP-1406 `:124-130`: "The declare is deleted in VOIP-1407 together with the
  fanout QueueSubscribe lines").
- **`asterisk.all.event` `QueueSubscribe`**: **NOT DELETED.** This is the one fanout
  `QueueSubscribe` line, across all 20 services, that survives this ticket -- it feeds
  from `voip-asterisk-proxy`, which is excluded from this ticket's scope entirely (§0).
  Concretely: the loop deletion in §3.1 item 2 must NOT remove the specific iteration
  that subscribes to `QueueNameAsteriskEventAll`; if the loop is deleted wholesale, this
  one `QueueSubscribe(subscribeQueue, string(commonoutline.QueueNameAsteriskEventAll))`
  call is re-added as a standalone statement **in the exact position the loop used to
  occupy** -- immediately after `QueueCreate`, before the VOIP-1258 block and the
  topic-declare line, matching §3.3's snippet exactly (R2 finding 6 corrected an earlier
  version of this bullet that also said "immediately before `go ConsumeMessage`", a
  different and wrong position for call-manager/timeline-manager's actual source
  layout).

**agent-manager and timeline-manager -- the VOIP-1258 webhook-topic-bind block:**

Confirmed via source (`bin-agent-manager/pkg/subscribehandler/main.go:97-120`,
`bin-timeline-manager/pkg/subscribehandler/main.go` -- same block) and
`docs/reference/rabbitmq-queues-reference.md:299`: these two services (only -- not
call-manager, not any other of the 20; `bin-api-manager` also binds
`QueueNameWebhookEventTopic` but through an unrelated mechanism, `pkg/websockhandler`'s
per-pod scoped-routing queue, not `pkg/subscribehandler`, so it is structurally outside
this ticket regardless) carry a self-contained block, positioned in `Run()` BETWEEN the
fanout `QueueSubscribe` loop (§3.1 item 2) and the VOIP-1406 topic-pattern bind block
(§3.1's `topicPatterns`/`QueueBind`):

```go
if errBind := h.sockHandler.QueueBind(h.subscribeQueue, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil); errBind != nil {
    log.Errorf("Could not bind to the topic exchange. err: %v", errBind)
    // do NOT proceed to unbind the old exchange if this bind failed -- stay on the
    // old exchange rather than risk ending up bound to neither.
} else if errUnbind := h.sockHandler.QueueUnbind(h.subscribeQueue, "", string(commonoutline.QueueNameWebhookEvent), nil); errUnbind != nil {
    log.Errorf("CRITICAL: Could not unbind from the old fanout exchange after binding to the new topic exchange. ... err: %v", errUnbind)
}
```

**Ruling: RETAINED VERBATIM AND IN POSITION.** This block is not part of §3.1's generic
deletion target -- it does not read `subscribeTargets`/`fanoutUnbindTargets`/
`topicPatterns` and is unrelated to the fanout-vs-topic cutover this ticket performs. Two
sub-rulings, both scope-narrowing (neither is touched by this PR):

1. **Failure semantics stay log-only, NOT promoted to fatal.** §3.3's fatal-on-failure
   ruling is scoped to the VOIP-1406 topic-pattern block specifically, because that
   block's failure mode changes meaning once its sibling fanout loop is deleted (nothing
   left to degrade to). This VOIP-1258 block's degrade target -- staying bound to
   `QueueNameWebhookEvent` on a failed bind -- is a different exchange, untouched by
   anything in §2 or §3.1, and remains a functioning fallback regardless of this ticket.
   Making it fatal would be an unrelated behavior change outside this ticket's stated
   scope (§0).
2. **The legacy `QueueUnbind(..., QueueNameWebhookEvent, ...)` call is left untouched.**
   Issue analysis established `QueueNameWebhookEvent`'s exchange is redundant/safe to
   remove at the broker level (§1), but removing this call site -- VOIP-1258 cleanup, a
   different ticket's concern -- is explicitly OUT of scope here; touching it adds risk
   for zero benefit to this ticket's goal (§1's goal is the fanout-vs-topic *publish*
   cutover, not VOIP-1258's webhook-topic migration, which already shipped).

### 3.3 Failure semantics (issue analysis §6 deferred item 5, formerly item 6 -- resolved)

**Decision: FATAL, consistent with §2.4's publish-side ruling.** Today's `Run()`
degrades ("stay fully on fanout subscriptions") if the topic-exchange declare or any
`QueueBind` fails. Once the fanout `QueueSubscribe` loop is deleted, there is nothing
left to degrade TO -- the service would silently intake zero events forever, which is
strictly worse than a loud boot failure an operator/orchestrator will notice and
restart-loop on. Concretely, in `Run()` (the snippet below shows the union of both §3.2
exceptions for illustration; only timeline-manager carries both -- call-manager has the
asterisk line but not the webhook block, agent-manager has the webhook block but not the
asterisk line, the other 17 services have neither and skip straight from `QueueCreate` to
the topic-declare line). **timeline-manager only (§3.1's variance table)**: every
`return fmt.Errorf(...)` below becomes `return nil, fmt.Errorf(...)`, matching its
two-value `Run(ctx) (<-chan struct{}, error)` signature:

```go
	if err := h.sockHandler.QueueCreate(h.subscribeQueue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for subscribeHandler. err: %v", err)
	}

	// call-manager/timeline-manager only: the one retained fanout leg.
	if err := h.sockHandler.QueueSubscribe(h.subscribeQueue, string(commonoutline.QueueNameAsteriskEventAll)); err != nil {
		return fmt.Errorf("could not subscribe to the asterisk fanout exchange. err: %v", err)
	}

	// agent-manager/timeline-manager only: the VOIP-1258 webhook-topic-bind block
	// (§3.2). RETAINED VERBATIM, unchanged failure semantics (log-only, non-fatal --
	// §3.2 ruling 1). Not part of this ticket's fatal-on-failure change below.
	if errBind := h.sockHandler.QueueBind(h.subscribeQueue, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil); errBind != nil {
		log.Errorf("Could not bind to the topic exchange. err: %v", errBind)
	} else if errUnbind := h.sockHandler.QueueUnbind(h.subscribeQueue, "", string(commonoutline.QueueNameWebhookEvent), nil); errUnbind != nil {
		log.Errorf("CRITICAL: Could not unbind from the old fanout exchange after binding to the new topic exchange. ... err: %v", errUnbind)
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
`QueueUnbind` loop was the only automatic self-heal for this; the manual `curl`-based
management-API remediation documented in `docs/reference/rabbitmq-queues-
reference.md:301-312` still works, it just needs a human to run it -- **corrected per R2
finding 5**: an earlier version of this bullet said `rabbitmqadmin`, which is not the
tool that doc or §5's runbook actually uses on bm-nyc-01).

## 4. Per-call-site disposition (issue analysis §3's 62 sites, resolved individually)

| Sites | Disposition |
|---|---|
| 55 dual-publish (27 daemons + `-control` CLIs) | Topic-only per §2. |
| 1 (`voip-asterisk-proxy/cmd/asterisk-proxy/main.go:107`) | **Excluded, zero changes** (§0/§2.1). |
| 2 (`bin-transfer-manager/cmd/{transfer-manager,transfer-control}/main.go`) | **Deleted** (§7 item 4 -- confirmed dead: zero calls anywhere in `pkg/transferhandler`). Delete the `notifyHandler` field, its constructor parameter, and both `NewNotifyHandler` construction sites. |
| 1 (`bin-tts-manager/cmd/tts-control/main.go:38`) | **Deleted**, same reasoning (`pkg/ttshandler` never calls it). |
| 1 (`bin-talk-manager/cmd/talk-control/main.go:57`) | **Fixed, one line** (§2.3.1 -- corrected from an earlier "untouched, out of scope" ruling: this ticket's own `logrus.Fatalf` change would otherwise turn a narrow pre-existing bug into a guaranteed boot crash for the whole CLI). `queueEvent` changed from `""` to `commonoutline.QueueNameTalkEvent`; `WithGlobalTopicPublish()` deliberately NOT added. |
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

Exactly the 28 exchanges issue analysis §1 enumerated -- full list in issue analysis §1,
not repeated here since it is normative there; this design's §6 doc update must enumerate
all 28 names verbatim in the runbook rather than describe them by range. Explicitly NOT
included: `asterisk.all.event` (permanent), `bin-manager.event` (the topic exchange
itself), `bin-manager.webhook-manager.event.topic` (VOIP-1258, different exchange),
`bin-manager.delay` (unrelated retry exchange), and the 5 `QueueName*Event`-family
exchanges that are already absent from the broker (`api-manager`, `rag-manager`,
`timeline-manager`, `user-manager` -- never existed; `webhook-manager` -- absent from the
broker, cause unestablished (issue analysis §1), irrelevant here since there is nothing
to delete either way).

**Precondition (issue analysis §4 item 4, widened per that document's R2 finding)**:
every publisher daemon+control binary (§2/§4 above) AND every one of the 20 consumer
services (§3) that declares, binds, or subscribes to one of these 28 exchanges must be
rebuilt/redeployed with that code removed, with a CONFIRMED restart-survival check per
service (§3.4), before this runbook runs.

**Mechanism**: a documented one-time management-API cleanup via `curl` against the
broker's HTTP API (port 80, not 15672 on bm-nyc-01), matching the existing runbook
pattern already in `docs/reference/rabbitmq-queues-reference.md:301-312` (NOT
`rabbitmqadmin`, which that doc does not use). Exact commands (per exchange:
`curl -u "$RABBITMQ_USER:$RABBITMQ_PASS" -X DELETE
"http://<host>/api/exchanges/%2f/bin-manager.<x>-manager.event"`) to be added to that
doc, enumerating all 28 names verbatim, as part of this PR's documentation update (§6).

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
5. talk-control's broken wiring: **decided IN scope, §2.3.1/§4** -- not actually open,
   listed here for completeness (an earlier revision called this out of scope; R2's
   review found that stance untenable once §2.3 introduced `logrus.Fatalf`).
6. `topicDisabled` field: not actually open, listed here for completeness -- §2.4
   already decided this (deleted entirely, along with its `promTopicPublishTotal{
   result="error"}` "suppressed publish" counting and the "degrade, don't abort" doc
   comments), since the silent-degrade branch it existed for no longer exists once
   `initGlobalTopicExchange`'s failure path becomes `logrus.Fatalf`. Implementation
   should grep for any remaining reference to it after the §2.4 change and confirm
   zero survive.

## 8. Testing

- `bin-common-handler/pkg/notifyhandler`: table-driven tests for `publishEvent()`'s new
  branch split (topicEnabled true/false × delay zero/nonzero), and
  `publishTopicEventOrErr`'s metric observations (reuse the existing
  `promNotifyProcessTime`/`promTopicPublishTotal` assertions, extended to the
  topic-only path). `NewNotifyHandler`'s and `initGlobalTopicExchange`'s fatal-on-
  failure paths call `logrus.Fatalf`, not a nil return -- test via
  `logrus.StandardLogger().ExitFunc` temporarily overridden to a non-exiting stub plus a
  hook/formatter capturing the fatal-level entry, asserting the entry was logged at
  `logrus.FatalLevel` with the expected message, rather than asserting a return value.
- **Existing tests this design invalidates, named explicitly (R2 finding 4 -- §8
  previously described this work only in the abstract)**:
  - `Test_NewNotifyHandler_globalTopicDeclareFailure` (`main_test.go:258-294`) pins
    the EXACT degrade-not-abort contract §2.4 removes: non-nil handler on declare
    failure, `h.topicDisabled == true`, fanout `EventPublish` still firing, and
    `topicPublishResultError` incrementing. Under `logrus.Fatalf` this test would not
    fail -- it would terminate the whole `notifyhandler` test binary if left as-is. It
    must be rewritten (per this bullet's `ExitFunc`-override strategy) to assert the
    fatal-log behavior instead, not merely updated in place.
  - `Test_WithGlobalTopicPublish_declaresGlobalExchange`'s `h.topicDisabled` assertions
    (`main_test.go:251-253`) must be removed along with the field itself (§2.4/§7 item
    6).
  - `Test_NewNotifyHandler_withoutOption` (`main_test.go:296-318`) already asserts
    `TopicCreate` is called and a non-nil handler returned for the `topicEnabled=false`
    path -- this is the existing, already-passing form of this section's second
    regression pin below (the `voip-asterisk-proxy` bit-identical-behavior contract);
    cite it directly rather than writing a new test from scratch.
- **Regression pins (R1 finding 13) -- lock in behavior this design deliberately does
  NOT change, so a future edit to the shared `initGlobalTopicExchange` doesn't silently
  break either exception**:
  - `NewNotifyHandlerForExistingExchange` with `topicEnabled=false` (webhook-manager's
    actual construction today) still does NOT declare `bin-manager.event` and still
    returns a non-nil handler -- confirms §2.6's "no redundant declare exists today"
    correction stays true after this PR.
  - A `topicEnabled=false` `NewNotifyHandler` construction (`voip-asterisk-proxy`'s
    only call site, excluded per §0/§2.1) still calls `TopicCreate(queueEvent)` and
    routes every publish through `publishDirectEvent`/`publishDirectEventWithKey`,
    bit-identical to pre-PR behavior -- confirms this ticket's publish-side change is
    inert for the one excluded caller.
- **`bin-talk-manager/cmd/talk-control`** (§2.3.1): a build/smoke check that
  `initHandlers()` no longer reaches `logrus.Fatalf` -- i.e. that `TopicCreate(string(
  commonoutline.QueueNameTalkEvent))` succeeds against a real broker post-fix, where it
  previously failed against `""`. Not a unit test target (no existing test harness
  around `cmd/talk-control/main.go`'s `initHandlers()`); verify via the manual
  live-broker smoke check pattern issue analysis used elsewhere (declare the exchange
  once via the daemon, which already does so continuously in production, then run
  `talk-control chat list` against a real environment and confirm it exits 0 instead of
  calling `os.Exit(1)` during startup).
- Each of the 20 consumer services: update `binding_golden_test.go` (drop
  fanout-target pins) and the `Run()` sequencing test (gomock `InOrder`: `QueueCreate`
  -> [asterisk `QueueSubscribe`, call-manager/timeline-manager only] -> [VOIP-1258
  webhook-topic `QueueBind`+`QueueUnbind`, agent-manager/timeline-manager only,
  asserting §3.2's unchanged log-only failure behavior is preserved, not just its
  happy path] -> `TopicCreateWithKind` -> `QueueBind`×N -> `ConsumeMessage`;
  failure-path cases updated to assert `Run()` returns the error immediately (`return
  nil, err` for timeline-manager per §3.1's variance table, `return err` elsewhere),
  matching §3.3, replacing the old roll-back-and-degrade assertions).
- Live verification (post-merge, pre-deletion): the restart-survival + stale-binding
  sweep from §3.4, run per service as each is deployed.
