package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-timeline-manager/pkg/dbhandler"
)

const (
	batchSize     = 100
	flushInterval = 1 * time.Second
	eventChBuffer = 1000
)

// subscribeTargets lists all service event exchanges to subscribe to.
var subscribeTargets = []commonoutline.QueueName{
	commonoutline.QueueNameAIEvent,
	commonoutline.QueueNameAgentEvent,
	commonoutline.QueueNameAsteriskEventAll,
	commonoutline.QueueNameBillingEvent,
	commonoutline.QueueNameCallEvent,
	commonoutline.QueueNameCampaignEvent,
	commonoutline.QueueNameConferenceEvent,
	commonoutline.QueueNameContactEvent,
	commonoutline.QueueNameConversationEvent,
	commonoutline.QueueNameCustomerEvent,
	commonoutline.QueueNameEmailEvent,
	commonoutline.QueueNameFlowEvent,
	commonoutline.QueueNameMessageEvent,
	commonoutline.QueueNameNumberEvent,
	commonoutline.QueueNameOutdialEvent,
	commonoutline.QueueNamePipecatEvent,
	commonoutline.QueueNameQueueEvent,
	commonoutline.QueueNameRegistrarEvent,
	commonoutline.QueueNameRouteEvent,
	commonoutline.QueueNameSentinelEvent,
	commonoutline.QueueNameStorageEvent,
	commonoutline.QueueNameTagEvent,
	commonoutline.QueueNameTalkEvent,
	commonoutline.QueueNameTranscribeEvent,
	commonoutline.QueueNameTransferEvent,
	commonoutline.QueueNameTTSEvent,
}

// topicPatterns is the bind set on the global bin-manager.event topic exchange
// (VOIP-1406). timeline-manager is the archive-everything service, so a single
// catch-all "#" binding replaces the per-service fanout subscriptions: it matches
// every current topic publisher (a superset of the old fanout set, accepted by
// design ruling) and automatically includes any future publisher.
var topicPatterns = []string{"#"}

// fanoutUnbindTargets lists the fanout exchanges to unbind from the subscribe
// queue once every topicPatterns bind has succeeded (VOIP-1406). It is
// subscribeTargets minus the asterisk fanout leg, which is permanently retained
// because asterisk-proxy does not publish to the topic exchange.
var fanoutUnbindTargets = func() []commonoutline.QueueName {
	res := make([]commonoutline.QueueName, 0, len(subscribeTargets)-1)
	for _, target := range subscribeTargets {
		if target == commonoutline.QueueNameAsteriskEventAll {
			continue
		}
		res = append(res, target)
	}
	return res
}()

var (
	metricsNamespace = "timeline_manager"

	promSubscribeBatchInsertTime = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "subscribe_batch_insert_time",
			Help:      "Time in milliseconds for a ClickHouse batch insert",
			Buckets:   []float64{50, 100, 500, 1000, 3000},
		},
	)

	promSubscribeBatchSize = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "subscribe_batch_size",
			Help:      "Number of events per ClickHouse batch insert",
			Buckets:   []float64{1, 5, 10, 25, 50, 100},
		},
	)
)

func init() {
	prometheus.MustRegister(promSubscribeBatchInsertTime)
	prometheus.MustRegister(promSubscribeBatchSize)
}

// SubscribeHandler interface
type SubscribeHandler interface {
	// Run starts the subscribe handler. The returned channel is closed when
	// the flush worker finishes draining after ctx is cancelled.
	Run(ctx context.Context) (<-chan struct{}, error)
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler
	dbHandler   dbhandler.DBHandler
	eventCh     chan *sock.Event
}

// NewSubscribeHandler creates a new SubscribeHandler.
func NewSubscribeHandler(
	sockHandler sockhandler.SockHandler,
	dbHandler dbhandler.DBHandler,
) SubscribeHandler {
	return &subscribeHandler{
		sockHandler: sockHandler,
		dbHandler:   dbHandler,
		eventCh:     make(chan *sock.Event, eventChBuffer),
	}
}

// Run creates the subscribe queue, binds to all event exchanges, and starts consuming.
// The provided ctx controls the lifetime of the flush worker — when cancelled, the
// worker performs a final flush of any buffered events before returning.
// The returned channel is closed when the flush worker has finished draining.
func (h *subscribeHandler) Run(ctx context.Context) (<-chan struct{}, error) {
	log := logrus.WithField("func", "Run")
	log.Info("Creating rabbitmq queue for event subscription.")

	subscribeQueue := string(commonoutline.QueueNameTimelineSubscribe)

	// Create durable queue
	if err := h.sockHandler.QueueCreate(subscribeQueue, "normal"); err != nil {
		return nil, fmt.Errorf("could not declare the queue for subscribeHandler. err: %v", err)
	}

	// Subscribe to all service event exchanges
	for _, target := range subscribeTargets {
		// The sentinel-manager service is Kubernetes-only (it needs the Kubernetes API) and is
		// not deployed in non-Kubernetes deployments, so its event exchange may never have been
		// declared by its owner. Binding to a missing exchange makes QueueSubscribe fail with an
		// AMQP 404, which closes the shared channel and takes this service down at boot. Declare
		// it ourselves with the same fanout/durable parameters sentinel-manager's own
		// notifyhandler uses, which makes the declare an idempotent no-op when it is deployed.
		if target == commonoutline.QueueNameSentinelEvent {
			if errTopic := h.sockHandler.TopicCreate(string(target)); errTopic != nil {
				log.Errorf("Could not create the topic for the target. target: %s, err: %v", target, errTopic)
				return nil, errTopic
			}
		}

		if errSubscribe := h.sockHandler.QueueSubscribe(subscribeQueue, string(target)); errSubscribe != nil {
			log.Errorf("Could not subscribe to target. target: %s, err: %v", target, errSubscribe)
			return nil, errSubscribe
		}
		log.Debugf("Subscribed to event exchange. target: %s", target)
	}

	// Cut over from the old fanout QueueNameWebhookEvent exchange to the new
	// QueueNameWebhookEventTopic topic exchange with a "#" wildcard binding.
	// Bind new first, then unbind old, to avoid an event-loss window where the
	// queue is briefly bound to neither exchange. This queue is durable and shared
	// (not per-pod), so the old binding persists across deploys unless explicitly
	// removed (VOIP-1258 Task 3.5).
	if errBind := h.sockHandler.QueueBind(subscribeQueue, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil); errBind != nil {
		log.Errorf("Could not bind to the topic exchange. err: %v", errBind)
		// do NOT proceed to unbind the old exchange if this bind failed -- stay on the
		// old exchange rather than risk ending up bound to neither.
	} else if errUnbind := h.sockHandler.QueueUnbind(subscribeQueue, "", string(commonoutline.QueueNameWebhookEvent), nil); errUnbind != nil {
		log.Errorf("CRITICAL: Could not unbind from the old fanout exchange after binding to the new topic exchange. queue: %s is now bound to BOTH exchanges (double-processing resumes). Manual intervention required. err: %v", subscribeQueue, errUnbind)
	}

	// Migrate the subscribe queue from the per-service fanout exchanges to the global
	// bin-manager.event topic exchange (VOIP-1406). Bind new before unbinding old so
	// there is no window where the queue is bound to neither. The declare is idempotent
	// (kind/durable are fixed in the shared handler) and makes start order irrelevant.
	// The asterisk fanout subscription is NOT part of this migration and stays bound.
	if errDeclare := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic"); errDeclare != nil {
		log.Errorf("Could not declare the global topic exchange. Staying fully on the fanout subscriptions. exchange: %s, err: %v", commonoutline.QueueNameEvent, errDeclare)
	} else {
		// Bind ALL patterns -- all-or-nothing. On any bind failure, roll back the
		// partial topic binds (best-effort) and unbind NO fanout exchange; the
		// service keeps running fully on fanout.
		bound := []string{}
		ok := true
		for _, pattern := range topicPatterns {
			if errBind := h.sockHandler.QueueBind(subscribeQueue, pattern, string(commonoutline.QueueNameEvent), false, nil); errBind != nil {
				log.Errorf("Could not bind the pattern to the global topic exchange. pattern: %s, err: %v", pattern, errBind)
				ok = false
				break
			}
			bound = append(bound, pattern)
		}

		if !ok {
			// Best-effort rollback of the partial topic binds, then stay fully on
			// fanout. An unbind failure here leaves partial double delivery.
			for _, pattern := range bound {
				if errUnbind := h.sockHandler.QueueUnbind(subscribeQueue, pattern, string(commonoutline.QueueNameEvent), nil); errUnbind != nil {
					log.Errorf("CRITICAL: partial topic bind could not be rolled back. queue: %s keeps a stray topic binding (partial double delivery). Manual intervention required. pattern: %s, err: %v", subscribeQueue, pattern, errUnbind)
				}
			}
		} else {
			// Unbind ALL old fanout exchanges -- only after every pattern bound.
			// Unbind failure: CRITICAL log, not fatal (double delivery beats loss).
			for _, target := range fanoutUnbindTargets {
				if errUnbind := h.sockHandler.QueueUnbind(subscribeQueue, "", string(target), nil); errUnbind != nil {
					log.Errorf("CRITICAL: still bound to BOTH exchanges (double delivery). Manual intervention required. queue: %s, target: %s, err: %v", subscribeQueue, target, errUnbind)
				}
			}
		}
	}

	// Start the batch flush worker; doneCh is closed when the worker exits.
	doneCh := make(chan struct{})
	go func() {
		h.flushWorker(ctx)
		close(doneCh)
	}()

	// Start consuming events
	go func() {
		if errConsume := h.sockHandler.ConsumeMessage(ctx, subscribeQueue, "timeline-manager", false, false, false, 10, h.processEventRun); errConsume != nil {
			log.Errorf("Could not consume subscribe events. err: %v", errConsume)
		}
	}()

	log.Infof("Subscribe handler started. subscribed to %d event exchanges.", len(subscribeTargets))
	return doneCh, nil
}

// processEventRun pushes the event into the buffered channel for batch processing.
//
// Instrumentation note: this is called concurrently by the consumer's 10
// worker goroutines (Run's ConsumeMessage call). len(h.eventCh) is a safe
// snapshot read and the prometheus operations are atomic, so no extra
// locking is needed. The drop branch deliberately still returns nil - a
// dropped event is acked, not retried (changing that to backpressure via
// an error return is a tracked follow-up, and this counter is the data
// that would justify it).
func (h *subscribeHandler) processEventRun(m *sock.Event) error {
	// Observed BEFORE the enqueue attempt: the occupancy this event faced.
	promSubscribeEventChannelUsage.Observe(float64(len(h.eventCh)) / float64(cap(h.eventCh)))

	select {
	case h.eventCh <- m:
	default:
		promSubscribeEventDropped.Inc()
		logrus.WithField("event", m).Warn("Event channel full, dropping event.")
	}

	promSubscribeEventChannelUsageRatio.Set(float64(len(h.eventCh)) / float64(cap(h.eventCh)))
	return nil
}

// flushWorker drains the event channel and batch-inserts into ClickHouse.
// It flushes when the buffer reaches batchSize or flushInterval elapses.
// When ctx is cancelled, it performs a final flush of any remaining buffered
// and queued events before returning.
func (h *subscribeHandler) flushWorker(ctx context.Context) {
	log := logrus.WithField("func", "flushWorker")
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	buf := make([]eventEntry, 0, batchSize)

	for {
		select {
		case <-ctx.Done():
			// Drain any remaining events from the channel into the buffer
			for {
				select {
				case m := <-h.eventCh:
					buf = append(buf, eventEntry{event: m, receivedAt: time.Now()})
				default:
					if len(buf) > 0 {
						h.flushBatch(buf)
						log.Infof("Final flush completed. count: %d", len(buf))
					}
					log.Info("Flush worker stopped.")
					return
				}
			}

		case m := <-h.eventCh:
			buf = append(buf, eventEntry{event: m, receivedAt: time.Now()})
			// Keep the usage gauge decaying as the channel drains - without
			// this it would freeze at its last enqueue-time value once
			// ingest goes idle. Gauge.Set is an atomic store; negligible
			// even at full drain rate.
			promSubscribeEventChannelUsageRatio.Set(float64(len(h.eventCh)) / float64(cap(h.eventCh)))
			if len(buf) >= batchSize {
				h.flushBatch(buf)
				buf = buf[:0]
				ticker.Reset(flushInterval)
			}

		case <-ticker.C:
			if len(buf) > 0 {
				h.flushBatch(buf)
				buf = buf[:0]
			}
		}
	}
}

// eventEntry pairs an event with its receive timestamp for metrics.
type eventEntry struct {
	event      *sock.Event
	receivedAt time.Time
}

// flushBatch inserts all buffered events into ClickHouse in a single batch.
func (h *subscribeHandler) flushBatch(entries []eventEntry) {
	log := logrus.WithField("func", "flushBatch")

	rows := make([]dbhandler.EventRow, len(entries))
	for i, e := range entries {
		rows[i] = dbhandler.EventRow{
			Timestamp: e.receivedAt,
			EventType: e.event.Type,
			Publisher: e.event.Publisher,
			DataType:  e.event.DataType,
			Data:      string(e.event.Data),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	if err := h.dbHandler.EventBatchInsert(ctx, rows); err != nil {
		log.Errorf("Could not batch insert events into ClickHouse. count: %d, err: %v", len(rows), err)
		return
	}
	elapsed := time.Since(start)

	promSubscribeBatchSize.Observe(float64(len(rows)))
	promSubscribeBatchInsertTime.Observe(float64(elapsed.Milliseconds()))

	log.Debugf("Batch flushed %d events to ClickHouse in %v.", len(rows), elapsed)

	// Additionally project call-manager / conversation-manager events into
	// peer_events (address-searchable peer/local log, additive to the events
	// table above; see design doc §5). Failure here does NOT affect the
	// primary events insert already committed above — peer_events is a
	// secondary projection, not the audit-log source of truth. Uses its own
	// fresh timeout budget (not the events-insert ctx above) so a slow
	// primary insert cannot starve this independent secondary write of its
	// full 10s allowance (Round 1 PR review finding).
	peerRows := buildPeerEventRows(entries)
	if len(peerRows) > 0 {
		peerCtx, peerCancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := h.dbHandler.PeerEventBatchInsert(peerCtx, peerRows); err != nil {
			log.Errorf("Could not batch insert peer events into ClickHouse. count: %d, err: %v", len(peerRows), err)
		}
		peerCancel()
	}
}
