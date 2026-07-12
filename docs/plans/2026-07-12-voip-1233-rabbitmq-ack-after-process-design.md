# VOIP-1233 Design — Ack-after-process for rabbitmqhandler event consumption

Status: DRAFT (design review round 0, pre-review)
Ticket: https://voipbin.atlassian.net/browse/VOIP-1233
Branch: `VOIP-1233-rabbitmq-ack-after-process` (worktree)
Prior artifact: Phase 0.8 analysis (converged, 2 consecutive APPROVED) — see JIRA comment history and `~/agent-hermes/notes/2026-07-11-voip-1233-analysis.md`.

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
routing key differs from the queue's own name).

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
	promEventRetried.WithLabelValues(queueName).Inc()

	headers["x-delay"] = delayMs
	if errPub := r.publishExchange(string(commonoutline.QueueNameDelay), queueName, message.Body, headers); errPub != nil {
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
- Unit test for `startConsumers`: Qos is called with `(reg.numWorkers, 0, false)` before `Consume`; Qos error surfaces before Consume is attempted.
- Existing `mock_rabbitmqhandler.go`/`main_test.go` mock channel patterns (`mockChannel`, `mockChannelWithConsumeCounter`) already have `Qos` mocked (per Phase 0.8 analysis grep) — reusable for the new assertions.
- Integration-level note (not required to merge, nice-to-have): a real RabbitMQ (with the `rabbitmq-delayed-message-exchange` plugin, already required by this codebase) test verifying 3 consecutive failures actually result in the message landing nowhere (queue and delay-exchange both empty) after the ~155s total backoff — codifies the "bounded to 4 attempts" guarantee end-to-end.

## 8. Open questions for design review

1. ~~Is a single retry the right default, or should retry count be configurable per queue?~~ **Resolved (pchero, 2026-07-12): fixed 3 retries (4 total attempts) for all queues, not configurable per queue in this iteration.** Revisit only if production data (§4.4 counters) shows a strong need for per-queue tuning.
2. ~~Should the metrics carry a separate `service` label?~~ **Resolved (pchero, 2026-07-12): queue name only, no separate service label.**
3. Confirm no existing test/mock relies on `consumeMessageWorker`'s current ack-before ordering as an assertion (would need updating, not a design concern but a review one) — **in progress, being checked by the design review loop (round 1).**
4. **New question from the retry-mechanism revision**: is the fixed backoff schedule (5s/30s/120s) reasonable, or should it be exponential-from-a-base-config-value instead of a hardcoded 3-element slice? Current proposal: hardcoded slice for simplicity/readability, since the retry count itself (3) is already fixed per decision #1 above — revisit together if #1 is ever revisited.
5. **New question**: `publishExchange` (used for the retry republish, §4.2) opens and closes a brand-new AMQP channel per call (`publish.go:14-22`). Under sustained high retry volume, is per-retry channel churn an operational concern (channel open/close is not free)? Flagged for design review to assess; not blocking since retry volume is expected to be low relative to total event volume (retries only happen on failure, not on the happy path).

