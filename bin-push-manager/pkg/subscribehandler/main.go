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

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-push-manager/pkg/devicehandler"
)

// list of publishers
const (
	publisherAgentManager    = string(commonoutline.ServiceNameAgentManager)
	publisherCustomerManager = string(commonoutline.ServiceNameCustomerManager)
)

// topicPatterns is the bind set on the global topic exchange `bin-manager.event`: one pattern per
// dispatched (publisher, event-type) pair. Pinned by the binding golden test.
var topicPatterns = []string{
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameAgentManager), amagent.EventTypeAgentDeleted),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCustomerManager), cmcustomer.EventTypeCustomerDeleted),
}

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler

	subscribeQueue string

	deviceHandler devicehandler.DeviceHandler
}

var (
	metricsNamespace = "push_manager"

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
	deviceHandler devicehandler.DeviceHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		sockHandler:    sockHandler,
		subscribeQueue: subscribeQueue,
		deviceHandler:  deviceHandler,
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
		return fmt.Errorf("could not declare the queue for subscribeHandler. err: %w", err)
	}

	// bind the subscribe queue to the global topic exchange `bin-manager.event` with one pattern per
	// dispatched (publisher, event-type) pair. This block MUST run synchronously here, BEFORE the
	// ConsumeMessage goroutine below: QueueBind and ConsumeMessage's internal basic.consume share the same
	// underlying AMQP channel for a given queue, and racing them makes the broker close the channel.
	if err := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic"); err != nil {
		return fmt.Errorf("could not declare the global topic exchange. err: %w", err)
	}

	for _, pattern := range topicPatterns {
		if err := h.sockHandler.QueueBind(h.subscribeQueue, pattern, string(commonoutline.QueueNameEvent), false, nil); err != nil {
			return fmt.Errorf("could not bind the topic pattern. pattern: %s, err: %w", pattern, err)
		}
	}

	// receive subscribe events
	go func() {
		if errConsume := h.sockHandler.ConsumeMessage(context.Background(), h.subscribeQueue, string(commonoutline.ServiceNamePushManager), false, false, false, 10, h.processEventRun); errConsume != nil {
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
		"func":      "processEvent",
		"publisher": m.Publisher,
		"type":      m.Type,
	})
	ctx := context.Background()

	var err error
	start := time.Now()
	switch {

	// agent-manager
	case m.Publisher == publisherAgentManager && m.Type == amagent.EventTypeAgentDeleted:
		err = h.processEventAMAgentDeleted(ctx, m)

	// customer-manager
	case m.Publisher == publisherCustomerManager && m.Type == cmcustomer.EventTypeCustomerDeleted:
		err = h.processEventCMCustomerDeleted(ctx, m)

	default:
		// ignore the event.
		return
	}
	elapsed := time.Since(start)
	promEventProcessTime.WithLabelValues(m.Publisher, m.Type).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not process the event correctly. err: %v", err)
	}
}
