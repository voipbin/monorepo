package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"errors"
	"fmt"
	"time"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-contact-manager/pkg/casehandler"
	"monorepo/bin-contact-manager/pkg/contacthandler"
)

// list of publishers
const (
	publisherCustomerManager = string(commonoutline.ServiceNameCustomerManager)
)

// topicPatterns is the ruled bind set on the global topic exchange `bin-manager.event`
// (VOIP-1406, design §5): one pattern per dispatched (publisher, event-type) pair.
// contact-manager dispatches a single pair -- customer deletion cascade. Pinned by the
// binding golden test.
var topicPatterns = []string{
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCustomerManager), cmcustomer.EventTypeCustomerDeleted),
}

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler

	subscribeQueue string

	contactHandler contacthandler.ContactHandler
}

var (
	metricsNamespace = "contact_manager"

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
	contactHandler contacthandler.ContactHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		sockHandler:    sockHandler,
		subscribeQueue: subscribeQueue,
		contactHandler: contactHandler,
	}

	return h
}

func (h *subscribeHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})
	log.Info("Creating rabbitmq queue for subscribe.")

	// declare the queue for subscribe
	if err := h.sockHandler.QueueCreate(h.subscribeQueue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for subscribeHandler. err: %v", err)
	}

	// VOIP-1407: topicPatterns/QueueBind is the sole intake mechanism -- the fanout
	// QueueSubscribe loop and the fanout-unbind step have been removed, so there is no
	// fallback left to degrade to. Any failure here is fatal.
	if err := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic"); err != nil {
		return fmt.Errorf("could not declare the global topic exchange. err: %v", err)
	}

	for _, pattern := range topicPatterns {
		if err := h.sockHandler.QueueBind(h.subscribeQueue, pattern, string(commonoutline.QueueNameEvent), false, nil); err != nil {
			return fmt.Errorf("could not bind the topic pattern. pattern: %s, err: %v", pattern, err)
		}
	}

	// receive subscribe events
	go func() {
		if errConsume := h.sockHandler.ConsumeMessage(context.Background(), h.subscribeQueue, string(commonoutline.ServiceNameContactManager), false, false, false, 10, h.processEventRun); errConsume != nil {
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

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// customer-manager events
	/////////////////////////////////////////////////////////////////////////////////////////////////

	// customer deleted - cleanup all contacts for this customer
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
		// VOIP-1232: distinctly tag the two new GetOrCreate failure modes
		// (deadlock-retry exhaustion, peer-lock acquisition timeout) from
		// generic GetOrCreate errors, so operators can triage which
		// mechanism is firing. NOTE: none of these three outcomes have a
		// recovery path yet -- the library (rabbitmqhandler) now acks based
		// on the callback's return value (VOIP-1233 ack-after-process), but
		// processEventRun spawns this processing in a fire-and-forget
		// goroutine and returns nil immediately, so the message was already
		// Ack'd before this branch runs and it can only log. VOIP-1233
		// tracks the remaining follow-up: propagate the processing result
		// back to the consumer callback so these failures get an actual
		// retry/redelivery path.
		switch {
		case errors.Is(err, casehandler.ErrDeadlockExhausted):
			log.Errorf("GetOrCreate exhausted all deadlock retries; event dropped (ack-before-process, no DLQ -- see VOIP-1233). publisher: %s, type: %s, err: %v", m.Publisher, m.Type, err)
		case errors.Is(err, casehandler.ErrPeerLockTimeout):
			log.Errorf("GetOrCreate could not acquire peer serialization lock within timeout; event dropped (ack-before-process, no DLQ -- see VOIP-1233). publisher: %s, type: %s, err: %v", m.Publisher, m.Type, err)
		default:
			log.Errorf("Could not process the event correctly. publisher: %s, type: %s, err: %v", m.Publisher, m.Type, err)
		}
	}
}
