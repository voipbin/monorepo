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

### 4.1 Ack timing — move to after processing, event path only

```go
func (r *rabbit) consumeMessageWorker(messages <-chan amqp.Delivery, messageConsume sock.CbMsgConsume) {
	for message := range messages {
		err := r.executeConsumeMessage(message, messageConsume)
		r.ackOrRetry(message, err)
	}
}

// ackOrRetry acknowledges a successfully processed message, or applies a
// bounded-retry policy for a failed one. See §4.2 for the retry policy.
func (r *rabbit) ackOrRetry(message amqp.Delivery, processErr error) {
	log := logrus.WithFields(logrus.Fields{"func": "ackOrRetry", "queue": message.RoutingKey})

	if processErr == nil {
		if err := message.Ack(false); err != nil {
			log.Errorf("Error acknowledging message: %v", err)
		}
		return
	}

	if !message.Redelivered {
		// First failure for this delivery: give it exactly one retry.
		log.Warnf("Message processing failed, requeueing for one retry. err: %v", processErr)
		promEventRetried.WithLabelValues(message.RoutingKey).Inc()
		if err := message.Nack(false, true); err != nil {
			log.Errorf("Error nacking (requeue) message: %v", err)
		}
		return
	}

	// Already redelivered once (or arrived redelivered due to a crash/reconnect)
	// and failed again: drop it. See §4.2/§7 for the known limitation that this
	// flag cannot distinguish "failed twice" from "crashed once, then failed once".
	log.Errorf("Message processing failed after retry, dropping. err: %v", processErr)
	promEventDropped.WithLabelValues(message.RoutingKey).Inc()
	if err := message.Nack(false, false); err != nil {
		log.Errorf("Error nacking (drop) message: %v", err)
	}
}
```

`executeConsumeMessage` itself is unchanged (still just unmarshal + invoke callback);
only the caller's ack/nack decision moves.

### 4.2 Retry policy — bounded via `Redelivered`, not unbounded requeue

- Success → `Ack(false)`.
- First failure (`message.Redelivered == false`) → `Nack(false, requeue=true)`. The broker redelivers; the next delivery carries `Redelivered == true`.
- Second failure of the same delivery, OR any failure of a delivery that was already redelivered for another reason (reconnect, crash) → `Nack(false, requeue=false)`. The broker permanently discards it (no DLX configured — see §4.3).
- **This guarantees at most 2 delivery attempts per message**, never an infinite requeue loop (the historical "poison message" risk explicitly flagged in the analysis is closed).
- **Known accepted limitation (carried over from the analysis, §7 of the analysis doc):** the `Redelivered` flag cannot distinguish "this is a genuine second failure" from "this message was redelivered because of a crash/reconnect and is now failing for the first time genuinely." Worst case: a crash-redelivered message gets only 1 attempt instead of 2 before being dropped. This is a strictly better outcome than today (0 attempts, always silently dropped on any failure or crash) and is judged acceptable without building delivery-count tracking (`x-death` headers) for this iteration.

### 4.3 Alternative considered and rejected: DLX now

A dead-letter exchange per event queue would let a dropped message be inspected/replayed
manually instead of permanently discarded. Rejected for THIS PR because:
- It requires a queue topology change (`x-dead-letter-exchange` arg on `QueueDeclare`, a new DLX exchange, and either 20 new dead-letter queues or one shared one) across every event-subscribe queue — significantly larger blast radius for a single PR (queue re-declaration is not always in-place; some brokers require queue deletion+recreation to add DLX args to an existing durable queue).
- The `Nack(requeue=true)`-once policy already converts the two silent-loss scenarios (handler error, crash) into visible ones (log line + Prometheus counter, §4.4) without needing operators to consume a dead-letter queue.
- Recommended as a fast-follow ticket once `promEventDropped` data from production shows which queues actually need replay capability (same "observe before building more" principle already applied to VOIP-1232's metrics).

Quorum queue migration was also compared and rejected for the same reason (topology
change, broker resource increase) plus it is a bigger unit of work than this PR's scope.

### 4.4 Observability

New Prometheus counters in `rabbitmqhandler`, following the existing pattern in
`pkg/requesthandler/main.go` (`promEventCount` etc.):

```go
promEventRetried = prometheus.NewCounterVec(prometheus.CounterOpts{
    Name: "rabbitmqhandler_event_retried_total",
    Help: "Count of event messages that failed processing once and were requeued for a single retry.",
}, []string{"queue"})

promEventDropped = prometheus.NewCounterVec(prometheus.CounterOpts{
    Name: "rabbitmqhandler_event_dropped_total",
    Help: "Count of event messages dropped after failing processing on a redelivered attempt.",
}, []string{"queue"})
```

`message.RoutingKey` is used as the `queue` label (already available on `amqp.Delivery`,
no plumbing change needed). These two counters are the primary signal for prioritizing
the idempotency audit (§5) and the DLX fast-follow (§4.3) — the services/queues with the
highest `promEventDropped` rate are audited/upgraded first.

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
delivery a designed, bounded behavior (at most 2 attempts, §4.2) instead of an
undocumented edge case.

Given the bound is 2 attempts (not unbounded retry), the blast radius of not having
completed a full 20-service idempotency audit before merging is judged acceptable:
- Worst case per message: processed twice. Not processed N times.
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

- Unit tests for `ackOrRetry`: success→Ack, first-failure-not-redelivered→Nack(requeue=true), failure-and-redelivered→Nack(requeue=false), Ack-error-logged-not-fatal, Nack-error-logged-not-fatal.
- Unit test for `startConsumers`: Qos is called with `(reg.numWorkers, 0, false)` before `Consume`; Qos error surfaces before Consume is attempted.
- Existing `mock_rabbitmqhandler.go`/`main_test.go` mock channel patterns (`mockChannel`, `mockChannelWithConsumeCounter`) already have `Qos` mocked (per Phase 0.8 analysis grep) — reusable for the new assertions.
- Integration-level note (not required to merge, nice-to-have): a real RabbitMQ test verifying two consecutive failures actually results in the message landing nowhere (queue empty) — codifies the "bounded to 2 attempts" guarantee end-to-end.

## 8. Open questions for design review

1. Is `Nack(false, true)`-once-then-drop the right default, or should retry count be configurable per queue (e.g. billing wants 2 retries, a low-value queue wants 0)? Current proposal: single fixed policy for simplicity; revisit if production data (§4.4 counters) shows a need for per-queue tuning.
2. Should `promEventDropped`/`promEventRetried` also carry a `service` label (derived from queue name parsing) for easier dashboarding, or is `queue` alone sufficient? Current proposal: `queue` alone (it already encodes the service in its name, e.g. `bin-manager.billing-manager.subscribe`).
3. Confirm no existing test/mock relies on `consumeMessageWorker`'s current ack-before ordering as an assertion (would need updating, not a design concern but a review one).
