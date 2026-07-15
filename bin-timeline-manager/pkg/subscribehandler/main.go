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
func (h *subscribeHandler) processEventRun(m *sock.Event) error {
	select {
	case h.eventCh <- m:
	default:
		logrus.Warn("Event channel full, dropping event.")
	}
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
}
