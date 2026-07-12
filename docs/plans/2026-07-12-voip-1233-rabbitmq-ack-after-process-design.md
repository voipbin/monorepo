# VOIP-1233 Design — Ack-after-process for rabbitmqhandler event consumption

Status: IMPLEMENTED, PR #1094 (github.com/voipbin/monorepo) merged into main. PR review converged (3 rounds, rounds 1 CHANGES_REQUESTED then fixed, rounds 2-3 APPROVED).
Ticket: https://voipbin.atlassian.net/browse/VOIP-1233
Branch: `VOIP-1233-rabbitmq-ack-after-process` (worktree, merged)
Prior artifact: Phase 0.8 analysis (converged, 2 consecutive APPROVED) — see JIRA comment history and `~/agent-hermes/notes/2026-07-11-voip-1233-analysis.md`.
Design review history: round 1 = CHANGES_REQUESTED (1 BLOCKER: Prometheus label-cardinality panic risk; 1 MAJOR: missing DeliveryMode=Persistent transient-loss risk; 2 MINOR), all fixed. Round 2 = APPROVED (3 MINOR notes incorporated). Round 3 (final gate) = APPROVED (4 further MINOR implementation-phase notes incorporated, no BLOCKER/MAJOR). All MINOR notes across rounds are implementation-phase guidance, not open design questions.

## 0. IMPORTANT LIMITATION DISCOVERED DURING PR REVIEW (round 3, 2026-07-12) — READ BEFORE ASSUMING THIS TICKET "FIXED" EVENT LOSS

**This ticket's library fix alone does NOT achieve the stated goal (§1 below) for any of the 20 event-subscribe services in production today.** The retry mechanism this design implements is entirely gated on the `ConsumeMessage` callback returning a genuine `error`. An independent code audit of all 20 services' `pkg/subscribehandler/main.go` (`processEventRun`, the function passed as that callback) found that EVERY one of them, without exception, structurally prevents the real failure from ever reaching the library:

- 16 services (agent, ai, api, campaign, conference, contact, conversation, direct, flow, number, queue, registrar, storage, tag, transfer, webhook-manager): fire-and-forget — `go h.processEvent(m); return nil`. The callback returns success immediately without waiting for or checking the actual outcome.
- 3 services (billing, call, transcribe-manager): synchronous call, but errors are only logged; the callback still always returns `nil` to the library (billing's own `failedEventHandler.Save` failure is the only error that can propagate — the ORIGINAL processing error never does).
- 1 service (timeline-manager): channel-based batching, callback always returns `nil`.

This was not a new bug introduced by this ticket — it is the SAME pre-existing pattern that motivated VOIP-1233 in the first place (see `bin-contact-manager/pkg/subscribehandler/main.go:168-186`'s pre-existing VOIP-1232-era comment explicitly naming VOIP-1233 as the fix that would give these failures "an actual retry/redelivery path" — that comment's expectation is not yet met because the consumer side wasn't touched). This ticket closed the library half of the gap; **VOIP-1251 tracks the consumer-side half** (synchronizing each service's callback to propagate real failures). Until VOIP-1251 lands, `rabbitmqhandler_event_retried_total`/`_dropped_total` (§4.4) will read near-zero in production regardless of actual handler failure rates — that is expected, not a sign the mechanism is broken.

**Practical implication for anyone reading this document to reason about production event-loss risk today: assume the pre-VOIP-1233 behavior (silent single-shot loss on handler failure/crash) still applies to all 20 services until VOIP-1251 (or per-service equivalent work) ships.** The mechanism below is real and correct at the library level, but inert until wired up by a consumer that actually reports failure.

## 1. Goal

Stop `bin-common-handler/pkg/rabbitmqhandler`'s event-subscribe path from silently
losing events on handler failure or pod crash, without introducing a new failure mode
(unbounded requeue loops, prefetch collapse on reconnect) and without requiring a
platform-wide idempotency audit as a blocking precondition.

## 2. Non-goals (explicit scope boundaries)

- **RPC path (`consumeRPCWorker`) is NOT touched by this design.** It keeps ack-before-process. Rationale (from the analysis): RPC failures are already observable to the caller (500 response when `ReplyTo != ""`, or a timeout), so the urgency is lower, and changing it doubles the review/verification surface for one PR. Tracked as a candidate fast-follow, not blocked by this PR.
- **No platform-wide idempotency audit of the 20 event-subscribe services' handlers as a precondition.** See §5 for why this design keeps the redelivery blast radius bounded enough that a full audit is not a blocking dependency (a follow-up audit ticket is still recommended, not gating).
- **No Dead Letter Exchange (DLX) infrastructure in this PR.** Considered and deferred — see §4.3 and §7.
- **No quorum-queue migration.** Classic queues stay; `x-delivery-limit` (quorum-only) is not adopted. See §4.3 for the alternative actually chosen.

## 3. Current behavior (baseline, for reviewers unfamiliar with the analysis)

`consumeMessageWorker` (`bin-common-handler/pkg/rabbitmqhandler/consume.go:73-89`):

```go
func (r *rabbit) consumeMessageWorker(messages <-chan amqp.Delivery, messageConsume sock.CbMsgConsume) {
	for message := range messages {
		if err := message.Ack(false); err != nil {
			log.Errorf("Error acknowledging message: %v", err)
		}
		if err := r.executeConsumeMessage(message, messageConsume); err != nil {
			log.Errorf("Error while processing message: %v", err)
		}
	}
}
```

Ack fires before `executeConsumeMessage` runs. Handler failure and crash-mid-processing
both result in silent, permanent loss (message already removed from the broker).
`QueueQoS(name, 1, 0)` (`queue.go:83`, prefetch=1) is set at queue-creation time but is
made ineffective by ack-before (unacked count returns to 0 instantly, no backpressure).

## 4. Proposed change

**Design decision log (pchero, 2026-07-12):**
- Retry count: **fixed 3 retries** (not configurable per queue for this iteration; see §4.2 for the exact semantics — 3 retries after the first failure = 4 total delivery attempts).
- Metric label: **queue name only** (no separate `service` label) — confirmed, matches the original default proposal.

### 4.1 Ack timing — move to after processing, event path only

```go
func (r *rabbit) consumeMessageWorker(messages <-chan amqp.Delivery, queueName string, messageConsume sock.CbMsgConsume) {
	for message := range messages {
		err := r.executeConsumeMessage(message, messageConsume)
		r.ackOrRetry(message, queueName, err)
	}
}
```

`queueName` is threaded in from `startConsumers` (`reg.queueName`, already known there) —
needed by `ackOrRetry` to republish a retry onto the CORRECT queue (see §4.2; using
`message.RoutingKey` instead would be wrong for topic-routed events where the publish
routing key differs from the queue's own name). **Design-review round-1 finding (MINOR,
clarified here): this signature change means `startConsumers`'s goroutine-spawn line
(§4.5) also changes** — `go r.consumeMessageWorker(messages, reg.cbMessage)` becomes
`go r.consumeMessageWorker(messages, reg.queueName, reg.cbMessage)`. The `// ...
unchanged` comment in §4.5's snippet refers only to the rest of `startConsumers`'s body
below the `Consume` call, not to this spawn line — called out explicitly here so the
implementation phase doesn't read "unchanged" as "no changes needed to make this
compile."

### 4.2 Retry mechanism — header-based counter via the existing delay-exchange, NOT the boolean Redelivered flag

**Revision note:** the original round-0 draft proposed a single retry using the amqp
`Redelivered` boolean flag (`Nack(false, true)` once, then drop). That mechanism can only
ever express "has this been redelivered or not" — it cannot count to 3. Since pchero
decided on a fixed 3 retries, the mechanism is redesigned around an explicit retry-count
header, reusing infrastructure that already exists in this library: **every queue created
via `queueCreateNormal`/`queueCreateVolatile` is already bound to the `QueueNameDelay`
exchange with the queue's own name as the binding key** (`queueConfig`, queue.go:81-98 →
`QueueBind(name, name, string(commonoutline.QueueNameDelay), false, nil)`, queue.go:93).
This is the exact mechanism `EventPublishWithDelay`/`RequestPublishWithDelay`
(publish.go:130-154) already use to schedule delayed messages via the
`x-delayed-message` exchange type (exchange.go:5-6, 43-48, requires the
`rabbitmq-delayed-message-exchange` broker plugin — already a hard runtime dependency of
this codebase, not a new one). Reusing it for bounded retry means **no new queue
topology, no DLX, no quorum queue** — consistent with the topology-avoidance reasoning
in §4.3.

```go
const (
	headerRetryCount = "x-retry-count"
	maxEventRetries  = 3
)

// retryBackoff returns the delay (ms) before the Nth retry (N = 1..maxEventRetries).
var retryBackoff = []int{5000, 30000, 120000} // 5s, 30s, 120s

func (r *rabbit) ackOrRetry(message amqp.Delivery, queueName string, processErr error) {
	log := logrus.WithFields(logrus.Fields{"func": "ackOrRetry", "queue": queueName})

	if processErr == nil {
		if err := message.Ack(false); err != nil {
			log.Errorf("Error acknowledging message: %v", err)
		}
		return
	}

	retryCount := getRetryCount(message.Headers) // 0 if header absent/malformed

	if retryCount >= maxEventRetries {
		log.Errorf("Message processing failed after %d retries, dropping. err: %v", maxEventRetries, processErr)
		promEventDropped.WithLabelValues(queueName).Inc()
		if err := message.Ack(false); err != nil {
			log.Errorf("Error acknowledging (dropping) exhausted message: %v", err)
		}
		return
	}

	// Publish a delayed copy BEFORE acking the original: if the republish fails
	// (e.g. broker hiccup), we fall back to an immediate Nack(requeue=true) on the
	// original rather than lose it. Worst case is an immediate untimed retry
	// instead of a backed-off one -- never silent loss.
	headers := cloneHeaders(message.Headers)
	headers[headerRetryCount] = retryCount + 1
	delayMs := retryBackoff[retryCount]

	log.Warnf("Message processing failed, scheduling retry %d/%d in %dms. err: %v", retryCount+1, maxEventRetries, delayMs, processErr)
	promEventRetried.WithLabelValues(queueName, strconv.Itoa(retryCount+1)).Inc()

	headers["x-delay"] = delayMs
	if errPub := r.publishExchange(string(commonoutline.QueueNameDelay), queueName, message.Body, headers, amqp.Persistent); errPub != nil {
		// NOTE: publishExchange (publish.go:14) is currently a 4-parameter
		// function (exchange, key, message, headers). Passing amqp.Persistent
		// here as shown requires adding a deliveryMode parameter -- this is a
		// REQUIRED signature change for the implementation phase, not code that
		// compiles as-is today. See the BLOCKER/MAJOR fix note directly below
		// for the full rationale (round-1/round-2 design review).
		log.Errorf("Could not schedule delayed retry, falling back to immediate requeue. err: %v", errPub)
		if err := message.Nack(false, true); err != nil {
			log.Errorf("Error nacking (fallback requeue) message: %v", err)
		}
		return
	}

	if err := message.Ack(false); err != nil {
		// Original could not be acked after a successful republish -- broker may
		// redeliver it too, resulting in a duplicate retry copy in flight. Accepted:
		// duplicate processing, never loss (same principle as §5).
		log.Errorf("Error acknowledging original after scheduling retry (possible duplicate): %v", err)
	}
}

// getRetryCount reads x-retry-count from delivery headers, defaulting to 0 for a
// missing or malformed header (treats it as a first-time failure).
func getRetryCount(headers amqp.Table) int {
	v, ok := headers[headerRetryCount]
	if !ok {
		return 0
	}
	n, ok := v.(int32) // amqp091-go decodes small ints as int32
	if !ok {
		return 0
	}
	return int(n)
}

func cloneHeaders(h amqp.Table) amqp.Table {
	out := make(amqp.Table, len(h)+1)
	for k, v := range h {
		out[k] = v
	}
	return out
}
```

`executeConsumeMessage` itself is unchanged (still just unmarshal + invoke callback);
only the caller's ack/retry decision moves, and the retry path routes through the
existing delay-exchange publish path instead of a bare `Nack`.

**Total attempts**: original delivery + up to 3 retries = **4 attempts maximum**, with
backoff 5s → 30s → 120s between them, before the message is permanently dropped
(Acked without further action; no DLX to inspect it afterward — see §4.3/§7 for the
production-signal-driven fast-follow).

**Design-review round-1 finding (MAJOR, addressed here): the 4-attempt bound can be
bypassed if `message.Ack()` itself fails on the SUCCESS path.** If `executeConsumeMessage`
succeeds but the subsequent `message.Ack(false)` call errors (e.g. channel already
closed — the same pre-existing edge case flagged in the current code's
`consume.go:81-83` comment), the message stays unacked at the broker and gets
auto-redelivered on channel/connection close, arriving with `Redelivered=true` but
**without any `x-retry-count` header change**, because it never went through the
explicit failure branch that sets that header. If this redelivered copy is then
processed and fails, `ackOrRetry` reads `getRetryCount` off the ORIGINAL headers (still
0, since the header was never touched), and schedules a fresh 3-retry sequence — meaning
a single Ack failure on the happy path can silently reset the retry budget, allowing more
than 4 total attempts in the worst case (bounded by how many times Ack itself keeps
failing, which is an operational/connection-health question, not a design one). This
mirrors the exact same class of issue already accepted for the current code (duplicate
processing via Ack failure is pre-existing, §5), but the NEW risk introduced here is that
it also resets the retry-count HEADER rather than just causing one duplicate delivery.
**Mitigation adopted**: none required beyond documenting it — since Ack failure implies
the channel is already in a degraded/closing state, the broker's own redelivery already
dominates the retry-count mechanism at that point (the connection is about to be torn
down and `reconsumerAll` will re-register consumers from scratch), and bounding this
further (e.g. persisting retry-count externally) is judged not worth the complexity for
a channel-failure edge case that is already logged and already accepted as an idempotency
risk under §5's "processed more than once" umbrella. If production data (§4.4) later
shows this path firing at meaningful volume, revisit with an external/durable counter.

**Design-review round-1 finding (BLOCKER, fixed above): the code snippet's Prometheus
call did not match its own metric definition.** §4.4 defines `promEventRetried` with TWO
labels (`queue`, `retry_count`), but the original snippet in this section called
`promEventRetried.WithLabelValues(queueName)` with only ONE argument. `CounterVec` panics
on a label-count mismatch ("inconsistent label cardinality") — this would have crashed
the process on the very first retry, i.e. exactly the first time this fix's own retry
path is exercised. Fixed: call now passes `strconv.Itoa(retryCount+1)` as the second
argument (the 1/2/3 retry attempt number, matching §4.4's stated label). This needs a
`strconv` import in the real implementation.

**Design-review round-1 finding (MAJOR, fixed above): `publishExchange`/`amqp.Publishing`
did not set `DeliveryMode`, so retry copies were transient (in-memory only) despite the
original message's durable delivery.** `amqp.Publishing.DeliveryMode` defaults to 0
(transient) unless explicitly set to `amqp.Persistent` (2). A message sitting in the
`x-delayed-message` exchange's internal wait state for up to 120s (§4.2's longest
backoff) with `DeliveryMode` unset would be silently lost if the broker restarts during
that window — undermining the "never silent loss" framing this whole design is built on.
Fixed: the retry republish now explicitly passes `amqp.Persistent`. This requires
`publishExchange` (`publish.go:14-43`) to accept an optional `deliveryMode` parameter (or
a new small wrapper) — a signature change flagged for the implementation phase, not
committed to a specific shape here. Note this durability caveat is NOT new to this
design: `EventPublish`/`EventPublishWithDelay` elsewhere in this codebase have the exact
same gap today (also call `publishExchange` without setting `DeliveryMode`) — this design
only fixes it for the NEW retry-republish path introduced here; the pre-existing gap in
the other call sites is out of scope for this ticket but worth flagging as a related
follow-up (the analysis's "observe, then build more" principle applies equally here).

**Design-review round-1 finding (MINOR): `publishExchange` failure fallback has no
backoff, risking a busy-loop under sustained broker/channel trouble.** If
`publishExchange` keeps failing (e.g. a channel that's already unhealthy) while the local
handler callback itself keeps succeeding-then-failing-to-schedule, the
`Nack(false, true)`-then-immediate-redelivery fallback path has no rate limit, which
could spin CPU/log volume under a pathological failure mode. Accepted for this
iteration (the failure mode requires BOTH a working consume channel AND a broken publish
channel simultaneously — an unusual combination) but flagged for the implementation phase
to consider a minimal backoff (e.g. a short `time.Sleep` before the fallback `Nack`) if
observed in practice.

**Ordering guarantee (none, by design)**: because retries are republished as brand-new
deliveries on the (possibly re-delayed) queue, a retried message can be interleaved with
newer messages on the same queue — no FIFO guarantee across a retry. This matches the
existing behavior of `EventPublishWithDelay` elsewhere in the codebase and is judged
acceptable for event processing (handlers are expected to be order-tolerant per the
idempotency principle in §5).


### 4.3 Alternative considered and rejected: dedicated DLX/dead-letter queue

A separate dead-letter exchange/queue (distinct from the retry mechanism in §4.2, which
reuses the existing delay-exchange) would let a permanently-dropped message be inspected
manually after all 3 retries are exhausted, instead of just being Acked away with a log
line. Rejected for THIS PR because:
- It requires NEW queue topology (`x-dead-letter-exchange` arg, a new exchange, and either
  20 new dead-letter queues or one shared one) on top of the delay-exchange reuse in
  §4.2 — §4.2 already avoids new topology for the retry path itself; adding a DLX only
  for the terminal-drop case would reintroduce exactly the topology cost this design
  otherwise avoids, for a smaller payoff (post-mortem inspection, not correctness).
- The 3-retry-with-backoff policy (§4.2) already converts the two silent-loss scenarios
  (handler error, crash) into visible, retried, metered ones (`promEventRetried`/
  `promEventDropped`, §4.4) — a message reaching `promEventDropped` has already had 4
  chances across up to ~155s of backoff, which is a materially different risk profile
  from today's zero-chance silent loss.
- Recommended as a fast-follow ticket once `promEventDropped` production data shows which
  queues actually need post-drop inspection/replay (same "observe before building more"
  principle already applied to VOIP-1232's metrics).

Quorum queue migration was also compared and rejected for the same reason (topology
change, broker resource increase) plus it is a bigger unit of work than this PR's scope.

**Design-review round-2 findings (all MINOR, implementation-phase notes):**
1. `queueConfig`'s original `QueueQoS(name, 1, 0)` call (`queue.go:83`, runs once at queue creation) becomes dead/superseded once `startConsumers` applies `Qos(reg.numWorkers, ...)` on every registration (§4.5) — the final state is still correct (the later call wins), but the stale `1, 0` call should be removed or clearly commented as intentionally-overridden during implementation, to avoid a future reader thinking prefetch is still 1.
2. `pkg/rabbitmqhandler` currently registers NO Prometheus metrics (only `pkg/requesthandler` does, per `bin-common-handler/CLAUDE.md`'s "do not register duplicate metric names" rule). §4.4 introduces two new `CounterVec`s here for the first time in this package. **Design-review round-3 finding (MINOR, clarified here): "following requesthandler's existing pattern" is imprecise about WHICH pattern.** `requesthandler`'s `initPrometheus` (`requesthandler/main.go:160-193`) is a plain function, NOT a Go `func init()` — it is called explicitly from `NewRequestHandler()` (and from `TestMain` in tests), so its "exactly once" safety depends on `NewRequestHandler()` itself being called exactly once per process, not on Go runtime guarantees. The implementation must pick ONE of two options explicitly rather than conflating them: (a) a real `func init()` in the rabbitmqhandler package — safe regardless of how many times `NewRabbit()` is called, since Go guarantees package-level `init()` runs exactly once; or (b) `requesthandler`'s pattern — an explicit registration call from inside `NewRabbit()`, which then requires `NewRabbit()` itself to be called at most once per process (spot-checked as true in practice — e.g. `bin-call-manager/cmd/call-manager/main.go:138` calls `NewSockHandler`→`NewRabbit()` exactly once — but this is a usage convention, not an enforced invariant). Recommendation: prefer (a), a real `func init()`, since it removes the dependency on call-count discipline entirely and is strictly safer for a shared library used by 37 services.
3. **Design-review round-3 finding (MINOR): the §4.4 metric snippet omits a `Namespace` field, unlike every existing metric in `requesthandler/main.go:168-187`** (which all set `Namespace: namespace` via `commonoutline.GetMetricNameSpace(publisher)`, producing a per-service-prefixed name). Without it, `rabbitmqhandler_event_retried_total`/`_dropped_total` would be identical metric names across all 20 event-subscribe services (Prometheus `job`/`instance` labels still distinguish scrape targets, so this is not a naming collision, just an inconsistency with the codebase's established per-service-namespace convention). Implementation should either add the same `Namespace` pattern for consistency, or explicitly note why this metric is deliberately un-namespaced (e.g. it's a library-level signal meant to be aggregated across services, not per-service).
4. **Design-review round-3 finding (MINOR): the one-queue-one-registration guard's atomicity is not specified.** §4.5's guard ("reject a second registration attempt for a `queueName` already present in `r.consumers`") must perform the existence-check and the append under the SAME lock acquisition (`r.mu.Lock()`) to avoid a TOCTOU race between two concurrent `ConsumeMessage`/`ConsumeRPC` calls for the same queue name — a naive "check under RLock, then separately Lock to append" implementation would still race. Given the Phase 0.8 analysis found no actual same-queue double-registration in the codebase today, this is a defensive guard against a case that isn't currently exercised, but the implementation should still get the locking right the first time.
5. (Documentation-only tidiness) The "publishExchange needs a signature change" callout ended up several paragraphs after the code snippet that uses the not-yet-existing parameter; an inline `NOTE:` comment was added directly in the §4.2 snippet (see the retry-publish block above) so a reader skimming only the code block doesn't mistake it for already-compiling code.

### 4.4 Observability

New Prometheus counters in `rabbitmqhandler`, following the existing pattern in
`pkg/requesthandler/main.go` (`promEventCount` etc.):

```go
promEventRetried = prometheus.NewCounterVec(prometheus.CounterOpts{
    Name: "rabbitmqhandler_event_retried_total",
    Help: "Count of event messages that failed processing and were scheduled for a delayed retry.",
}, []string{"queue", "retry_count"})

promEventDropped = prometheus.NewCounterVec(prometheus.CounterOpts{
    Name: "rabbitmqhandler_event_dropped_total",
    Help: "Count of event messages dropped after exhausting all retries.",
}, []string{"queue"})
```

`queueName` (threaded from `startConsumers`/`reg.queueName`, §4.1) is used as the `queue`
label — confirmed as the sole label dimension (pchero decision, no separate `service`
label; queue names already encode the service, e.g.
`bin-manager.billing-manager.subscribe`). `promEventRetried` additionally carries
`retry_count` (the 1/2/3 value being scheduled) so a dashboard can show the retry-depth
distribution, not just a single aggregate count — useful for judging whether 3 is the
right ceiling once production data exists. These two counters are the primary signal for
prioritizing the idempotency audit (§5) and the DLX fast-follow (§4.3) — the
services/queues with the highest `promEventDropped` rate are audited/upgraded first.

### 4.5 QoS fix — mandatory companion change (round-1 analysis-review finding)

Raise prefetch from 1 to `numWorkers`, and move the `Qos` call so it is re-applied on
every consumer (re-)registration, not only at queue creation.

Current: `queueConfig` (`queue.go:81-98`) calls `QueueQoS(name, 1, 0)`, but `queueConfig`
only runs from `queueCreateNormal`/`queueCreateVolatile` (`queue.go:57,74`), i.e. once at
`QueueCreate` time. `redeclareAll` (`main.go:261`) calls `QueueDeclare` directly on
reconnection — a fresh channel, Qos never reapplied — then `reconsumerAll` (`main.go:311`)
re-registers consumers on it via `startConsumers`. Today this is harmless (ack-before
makes prefetch moot); after this PR it would NOT be harmless (a reconnect would silently
uncap prefetch, defeating the very backpressure this fix depends on).

Fix: move the `Qos` call into `startConsumers` (`consume.go:15-44`), which already has
`queueGet(reg.queueName)` (channel access) and `reg.numWorkers`, and which is the single
call site used by both the initial `ConsumeMessage`/`ConsumeRPC` registration path and
the `reconsumerAll` reconnection path:

```go
func (r *rabbit) startConsumers(reg *consumerRegistration) error {
	queue := r.queueGet(reg.queueName)
	if queue == nil {
		return fmt.Errorf("queue '%s' not found", reg.queueName)
	}

	// Prefetch must cover the full worker pool, or ack-after-process would
	// serialize workers behind a single in-flight message (see design §4.5).
	// Re-applied on every call so it survives reconnection via reconsumerAll.
	if err := queue.channel.Qos(reg.numWorkers, 0, false); err != nil {
		return fmt.Errorf("could not set qos for queue '%s': %v", reg.queueName, err)
	}

	messages, err := queue.channel.Consume(...)
	// ... unchanged
}
```

This applies to BOTH `ConsumeMessage` and `ConsumeRPC` registrations (both go through
`startConsumers`). The RPC path keeping ack-before (§2) is unaffected in correctness by a
higher prefetch — it only means more RPC deliveries can be in flight per channel
simultaneously, which is strictly compatible with today's behavior (ack still fires
immediately).

`reg.numWorkers` is caller-supplied per queue (varies by service); using it directly
(rather than a fixed constant) keeps prefetch matched to each queue's actual worker pool
size with no new configuration surface.

**Design-review round-1 finding (MAJOR, addressed here): same-queue multiple
registrations would silently overwrite each other's Qos value.** `startConsumers` is
called per `consumerRegistration` (`main.go:315-317` iterates `r.consumers`, a slice with
no uniqueness constraint on `queueName`). If two registrations ever target the same
queue with different `numWorkers` (nothing in `queueGet`/`r.consumers`'s structure
prevents this today, even though it is not the intended usage pattern — each service
normally calls `ConsumeMessage`/`ConsumeRPC` exactly once per queue it owns), the LAST
`startConsumers` call's `Qos(reg.numWorkers, ...)` silently wins, and an earlier
registration's workers now run under a prefetch smaller than their own worker count,
reintroducing the serialization problem this fix exists to solve. Fix: `Qos` is a
per-CHANNEL setting (not per-consumer), so the design adds an explicit invariant instead
of a max()-tracking mechanism (simpler, and matches actual usage): **one queue = one
`consumerRegistration` = one channel-owning `startConsumers` call, enforced by a guard in
`ConsumeMessage`/`ConsumeRPC`** — reject (return an error) a second registration attempt
for a `queueName` already present in `r.consumers`. This is a one-line addition to the
existing registration path and matches how the codebase is actually used (grep of all
service `main.go`s during the Phase 0.8 analysis found no case of a queue registered
twice); it converts a latent footgun into a fail-fast error instead of leaving it as an
unstated assumption.

**Design-review round-1 finding (MINOR, clarified here): RPC-path prefetch increase has
no compatibility risk beyond buffer sizing.** `consumeRPCWorker` (`consume.go:131-148`)
keeps ack-before-process (§2) unconditionally — raising `Qos` on an RPC queue only grows
how many unacked deliveries the broker can have in flight on that channel at once; actual
concurrency is still bounded by `reg.numWorkers` goroutines (`startConsumers`,
`consume.go:34-41`), and delivery ordering/semantics for RPC requests are unaffected
since ack timing there is unchanged. No RPC-path behavior change beyond a larger
broker-side unacked buffer.

## 5. Idempotency — bounded risk, not a blocking precondition

At-least-once delivery is not new: the analysis established that duplicate processing
is ALREADY possible today via the ack-failure path (`message.Ack` erroring while
processing continues, `consume.go:81-83` in the current code). This PR makes duplicate
delivery a designed, bounded behavior (at most 4 total attempts, §4.2) instead of an
undocumented edge case.

Given the bound is 4 attempts (not unbounded retry), the blast radius of not having
completed a full 20-service idempotency audit before merging is judged acceptable:
- Worst case per message: processed up to 4 times (1 original + 3 retries), spread over
  up to ~155s of backoff. Not processed unbounded times, and each retry attempt is
  independently observable via the `retry_count`-labeled metric.
- `promEventDropped`/`promEventRetried` (§4.4) give production signal to prioritize which
  services need idempotency hardening first, rather than gating this fix on a full audit
  that would delay closing the silent-loss defect.
- **Recommendation, not a blocker**: file a follow-up ticket to audit the 20 event-subscribe
  services for non-idempotent append operations (billing record creation — flagged in the
  analysis as high-risk since `failedeventhandler` has no dedup; webhook delivery —
  duplicate send is customer-visible), prioritized by `promEventDropped`/`promEventRetried`
  volume once this PR is in production for a observation window.

## 6. Rollout

Single change in `bin-common-handler/pkg/rabbitmqhandler`. Per the package's admission
rules (bin-common-handler CLAUDE.md), any public-API-adjacent change here requires
building every consumer. In this case the public API (`ConsumeMessage`, `Rabbit`
interface) is unchanged — only internal behavior of `consumeMessageWorker` and
`startConsumers` changes — so no consumer code changes are required. Verification burden
is still the full monorepo verification workflow (`go mod tidy && go mod vendor && go
generate ./... && go test ./... && golangci-lint run` in `bin-common-handler`, plus a
build check across consumers) per CLAUDE.md's "When changing a public API" section,
applied conservatively even though the exported signatures don't change.

Deployment: standard rebuild/redeploy of all 20 event-subscribe services picks up the
new library behavior; no config or DB migration involved.

## 7. Testing plan (for the implementation phase, not this design doc)

- Unit tests for `ackOrRetry`: success→Ack; failure with retryCount=0/1/2→`publishExchange` called on `QueueNameDelay` with `x-retry-count` incremented and correct `x-delay` from `retryBackoff`, then original Acked; failure with retryCount==3 (exhausted)→dropped via Ack + `promEventDropped` incremented, no republish; `publishExchange` failure during retry scheduling→falls back to `Nack(false, true)`; Ack-error/Nack-error on any path is logged, not fatal (doesn't panic or block the worker loop).
- Unit tests for `getRetryCount`: missing header→0; malformed/wrong-type header→0 (fail open toward "treat as first attempt", not toward silently exceeding the retry cap); present int32 header→correct count.
- Unit test for `startConsumers`: Qos is called with `(reg.numWorkers, 0, false)` before `Consume`; Qos error surfaces before Consume is attempted; a second registration attempt for an already-registered `queueName` is rejected (§4.5 invariant).
- Existing `mock_rabbitmqhandler.go`/`main_test.go` mock channel patterns (`mockChannel`, `mockChannelWithConsumeCounter`) already have `Qos` mocked (per Phase 0.8 analysis grep) — reusable as a starting point. **Design-review round-1 finding (MINOR, noted for implementation): the existing mocks return a fixed nil error and do not capture call arguments** (no `qosCallCount`/`qosArgs` fields on `mockChannel`), so the "Qos called with (reg.numWorkers, 0, false)" assertion needs a small addition to the mock (an argument-capturing field or a `gomock.Call` `.Do()`/`.Times()` expectation using the generated `MockamqpChannel` in `mock_rabbitmqhandler.go`, which already supports argument matching via gomock) — not a design concern, but flagged so the implementation phase doesn't assume zero mock changes are needed.
- Integration-level note (not required to merge, nice-to-have): a real RabbitMQ (with the `rabbitmq-delayed-message-exchange` plugin, already required by this codebase) test verifying 3 consecutive failures actually result in the message landing nowhere (queue and delay-exchange both empty) after the ~155s total backoff — codifies the "bounded to 4 attempts" guarantee end-to-end.

## 8. Open questions for design review

1. ~~Is a single retry the right default, or should retry count be configurable per queue?~~ **Resolved (pchero, 2026-07-12): fixed 3 retries (4 total attempts) for all queues, not configurable per queue in this iteration.** Revisit only if production data (§4.4 counters) shows a strong need for per-queue tuning.
2. ~~Should the metrics carry a separate `service` label?~~ **Resolved (pchero, 2026-07-12): queue name only, no separate service label.**
3. Confirm no existing test/mock relies on `consumeMessageWorker`'s current ack-before ordering as an assertion (would need updating, not a design concern but a review one) — **in progress, being checked by the design review loop (round 1).**
4. **New question from the retry-mechanism revision**: is the fixed backoff schedule (5s/30s/120s) reasonable, or should it be exponential-from-a-base-config-value instead of a hardcoded 3-element slice? Current proposal: hardcoded slice for simplicity/readability, since the retry count itself (3) is already fixed per decision #1 above — revisit together if #1 is ever revisited.
5. **New question**: `publishExchange` (used for the retry republish, §4.2) opens and closes a brand-new AMQP channel per call (`publish.go:14-22`). Under sustained high retry volume, is per-retry channel churn an operational concern (channel open/close is not free)? Flagged for design review to assess; not blocking since retry volume is expected to be low relative to total event volume (retries only happen on failure, not on the happy path).

