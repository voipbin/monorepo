package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-tag-manager/pkg/taghandler"
)

// list of publishers
const (
	publisherCallManager     = string(commonoutline.ServiceNameCallManager)
	publisherCustomerManager = string(commonoutline.ServiceNameCustomerManager)
)

// topicPatterns is the ruled bind set on the global topic exchange `bin-manager.event`
// (VOIP-1406, design §5): one pattern per dispatched (publisher, event-type) pair.
// Pinned by the binding golden test.
var topicPatterns = []string{
	eventtopic.PatternAction(string(commonoutline.ServiceNameCustomerManager), "customer", "deleted"),
}

// fanoutUnbindTargets lists the old per-service fanout event exchanges to unbind once
// every topicPatterns bind has succeeded (VOIP-1406). It equals the full subscribeTargets
// set wired in cmd/tag-manager. The fanout QueueSubscribe calls in Run() stay until
// VOIP-1407 as the rollback/degrade surface.
var fanoutUnbindTargets = []string{
	string(commonoutline.QueueNameCustomerEvent),
}

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler

	subscribeQueue   string
	subscribeTargets []string

	tagHandler taghandler.TagHandler
}

var (
	metricsNamespace = "tag_manager"

	promEventProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "receive_subscribe_event_process_time",
			Help:      "Process time of received subscribe event",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"publisher", "type"},
	)
)

func init() {
	prometheus.MustRegister(
		promEventProcessTime,
	)
}

// NewSubscribeHandler return SubscribeHandler interface
func NewSubscribeHandler(
	sockHandler sockhandler.SockHandler,
	subscribeQueue string,
	subscribeTargets []string,
	tagHandler taghandler.TagHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		sockHandler:      sockHandler,
		subscribeQueue:   subscribeQueue,
		subscribeTargets: subscribeTargets,
		tagHandler:       tagHandler,
	}

	return h
}

func (h *subscribeHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})
	log.Info("Creating rabbitmq queue for listen.")

	// declare the queue for subscribe
	if err := h.sockHandler.QueueCreate(h.subscribeQueue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for subscribeHandler. err: %v", err)
	}

	// subscribe each targets
	for _, target := range h.subscribeTargets {
		if errSubscribe := h.sockHandler.QueueSubscribe(h.subscribeQueue, target); errSubscribe != nil {
			log.Errorf("Could not subscribe the target. target: %s, err: %v", target, errSubscribe)
			return errSubscribe
		}
	}

	// VOIP-1406: bind the subscribe queue to the global topic exchange `bin-manager.event`
	// with one pattern per dispatched (publisher, event-type) pair, then unbind the old
	// per-service fanout exchanges. Bind-new-BEFORE-unbind-old: the queue is never bound to
	// neither exchange. This block MUST run synchronously here, BEFORE the ConsumeMessage
	// goroutine below -- QueueBind/QueueUnbind and ConsumeMessage's internal basic.consume
	// share the same underlying AMQP channel for a given queue, and racing them makes the
	// broker close the channel with a 503 (production incident 2026-07-14, VOIP-1258).
	if errDeclare := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic"); errDeclare != nil {
		// stay fully on the fanout subscriptions; skip binds and unbinds.
		log.Errorf("Could not declare the global topic exchange. Staying on fanout subscriptions. exchange: %s, err: %v", string(commonoutline.QueueNameEvent), errDeclare)
	} else {
		bound := []string{}
		ok := true
		for _, pattern := range topicPatterns {
			if errBind := h.sockHandler.QueueBind(h.subscribeQueue, pattern, string(commonoutline.QueueNameEvent), false, nil); errBind != nil {
				log.Errorf("Could not bind the topic pattern. Staying on fanout subscriptions. pattern: %s, err: %v", pattern, errBind)
				ok = false
				break
			}
			bound = append(bound, pattern)
		}

		if !ok {
			// all-or-nothing: best-effort rollback of the partial topic binds; unbind NO
			// fanout exchange -- the service keeps running fully on fanout.
			for _, pattern := range bound {
				if errUnbind := h.sockHandler.QueueUnbind(h.subscribeQueue, pattern, string(commonoutline.QueueNameEvent), nil); errUnbind != nil {
					log.Errorf("CRITICAL: partial topic bind could not be rolled back. queue: %s keeps a stray topic binding (partial double delivery). Manual intervention required. pattern: %s, err: %v", h.subscribeQueue, pattern, errUnbind)
				}
			}
		} else {
			// every pattern bound: unbind the old fanout exchanges. Unbind failure is
			// CRITICAL but not fatal -- double delivery beats event loss.
			for _, target := range fanoutUnbindTargets {
				if errUnbind := h.sockHandler.QueueUnbind(h.subscribeQueue, "", target, nil); errUnbind != nil {
					log.Errorf("CRITICAL: could not unbind the fanout exchange after the topic binds succeeded. queue: %s is now bound to BOTH exchanges (double delivery). Manual intervention required. target: %s, err: %v", h.subscribeQueue, target, errUnbind)
				}
			}
		}
	}

	// receive subscribe events
	go func() {
		if errConsume := h.sockHandler.ConsumeMessage(context.Background(), h.subscribeQueue, string(commonoutline.ServiceNameTagManager), false, false, false, 10, h.processEventRun); errConsume != nil {
			log.Errorf("Could not consume the request message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

// processEventRun runs the processEvent
func (h *subscribeHandler) processEventRun(m *sock.Event) error {
	go h.processEvent(m)

	return nil
}

// processEvent processes the event message
func (h *subscribeHandler) processEvent(m *sock.Event) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processEvent",
		"message": m,
	})
	ctx := context.Background()

	var err error
	start := time.Now()
	switch {

	//// call-manager

	// customer
	case m.Publisher == publisherCustomerManager && (m.Type == string(cmcustomer.EventTypeCustomerDeleted)):
		err = h.processEventCMCustomerDeleted(ctx, m)

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		// ignore the event.
		return
	}
	elapsed := time.Since(start)
	promEventProcessTime.WithLabelValues(m.Publisher, string(m.Type)).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not process the event correctly. publisher: %s, type: %s, err: %v", m.Publisher, m.Type, err)
	}
}
