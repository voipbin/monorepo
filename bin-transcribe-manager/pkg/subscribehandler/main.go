package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/common"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	cucustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-transcribe-manager/pkg/transcribehandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// topicPatterns is the ruled bind set on the global topic exchange `bin-manager.event`
// (VOIP-1406, design §5): one pattern per dispatched (publisher, event-type) pair.
// Pinned by the binding golden test.
var topicPatterns = []string{
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCallManager), cmcall.EventTypeCallHangup),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCallManager), cmconfbridge.EventTypeConfbridgeTerminated),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCustomerManager), cucustomer.EventTypeCustomerDeleted),
}

// SubscribeHandler intreface for subscribed event listen handler
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler

	subscribeQueue commonoutline.QueueName

	transcribeHandler transcribehandler.TranscribeHandler
}

var (
	metricsNamespace = commonoutline.GetMetricNameSpace(commonoutline.ServiceNameTranscribeManager) //

	promSubscribeProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "subscribe_event_process_time",
			Help:      "Process time of subscribed events",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"publisher", "type"},
	)

	promARIEventTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "ari_event_listen_total",
			Help:      "Total number of received ARI event types.",
		},
		[]string{"type", "asterisk_id"},
	)

	promARIProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "ari_event_listen_process_time",
			Help:      "Process time of received ARI events",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"asterisk_id", "type"},
	)
)

func init() {
	prometheus.MustRegister(
		promSubscribeProcessTime,
		promARIEventTotal,
		promARIProcessTime,
	)
}

// NewSubscribeHandler create EventHandler
func NewSubscribeHandler(
	sock sockhandler.SockHandler,
	subscribeQueue commonoutline.QueueName,
	transcribeHandler transcribehandler.TranscribeHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		sockHandler:       sock,
		subscribeQueue:    subscribeQueue,
		transcribeHandler: transcribeHandler,
	}

	return h
}

// Run starts to receive subscribed event and process it.
func (h *subscribeHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})
	log.Infof("Creating rabbitmq queue for subscribed event receiving.")

	// declare the queue for subscribe
	if err := h.sockHandler.QueueCreate(string(h.subscribeQueue), "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for subscribeHandler. err: %v", err)
	}

	// VOIP-1406: bind the subscribe queue to the global topic exchange `bin-manager.event`
	// with one pattern per dispatched (publisher, event-type) pair. This block MUST run
	// synchronously here, BEFORE the ConsumeMessage goroutine below -- QueueBind and
	// ConsumeMessage's internal basic.consume share the same underlying AMQP channel for a
	// given queue, and racing them makes the broker close the channel with a 503 (production
	// incident 2026-07-14, VOIP-1258).
	if err := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic"); err != nil {
		return fmt.Errorf("could not declare the global topic exchange. err: %v", err)
	}

	for _, pattern := range topicPatterns {
		if err := h.sockHandler.QueueBind(string(h.subscribeQueue), pattern, string(commonoutline.QueueNameEvent), false, nil); err != nil {
			return fmt.Errorf("could not bind the topic pattern. pattern: %s, err: %v", pattern, err)
		}
	}

	// receive subscribe events
	go func() {
		if errConsume := h.sockHandler.ConsumeMessage(context.Background(), string(h.subscribeQueue), string(common.Servicename), false, false, false, 10, h.processEventRun); errConsume != nil {
			logrus.Errorf("Could not consume the subscribed evnet message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

// processEventRun runs the event process handler.
func (h *subscribeHandler) processEventRun(m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventRun",
		"event": m,
	})

	if errProcess := h.processEvent(m); errProcess != nil {
		log.Errorf("Could not consume the ARI event message correctly. err: %v", errProcess)
	}

	return nil
}

// processEvent processes received ARI event
func (h *subscribeHandler) processEvent(m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEvent",
		"event": m,
	})
	ctx := context.Background()

	var err error
	start := time.Now()

	switch {
	//// call-manager
	// call
	case m.Publisher == string(commonoutline.ServiceNameCallManager) && m.Type == cmcall.EventTypeCallHangup:
		err = h.processEventCMCallHangup(ctx, m)

	// confbridge
	case m.Publisher == string(commonoutline.ServiceNameCallManager) && m.Type == cmconfbridge.EventTypeConfbridgeTerminated:
		err = h.processEventCMConfbridgeTerminated(ctx, m)

	//// customer-manager
	// customer
	case m.Publisher == string(commonoutline.ServiceNameCustomerManager) && m.Type == cucustomer.EventTypeCustomerDeleted:
		err = h.processEventCUCustomerDeleted(ctx, m)

	default:
		// ignore the event.
		return nil
	}

	elapsed := time.Since(start)
	promSubscribeProcessTime.WithLabelValues(m.Publisher, m.Type).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not handle the subscribed event correctly. err: %v", err)
		return fmt.Errorf("could not process the ari event correctly. err: %v", err)
	}

	return nil
}
