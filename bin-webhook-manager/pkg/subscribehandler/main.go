package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_subscribehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cscustomer "monorepo/bin-customer-manager/models/customer"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-webhook-manager/pkg/accounthandler"
	"monorepo/bin-webhook-manager/pkg/cachehandler"
)

// list of publishers
const (
	publisherCustomerManager = "customer-manager"
	publisherFlowManager     = "flow-manager"
)

// topicPatterns is the set of binding patterns this service binds on the global
// bin-manager.event topic exchange (VOIP-1406): exactly one PatternAction per event pair the
// dispatch switch in processEvent handles, built with the same normalization the publish side
// uses to build routing keys. The binding golden test (binding_golden_test.go) pins this list
// byte-for-byte against the design's normative bind set.
var topicPatterns = []string{
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCustomerManager), cscustomer.EventTypeCustomerCreated),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCustomerManager), cscustomer.EventTypeCustomerUpdated),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameFlowManager), fmactiveflow.EventTypeActiveflowCreated),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameFlowManager), fmactiveflow.EventTypeActiveflowUpdated),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameFlowManager), fmactiveflow.EventTypeActiveflowDeleted),
}

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler

	subscribeQueue string

	accountHandler accounthandler.AccountHandler
	cacheHandler   cachehandler.CacheHandler
}

var (
	metricsNamespace = "webhook_manager"

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
	accountHandler accounthandler.AccountHandler,
	cacheHandler cachehandler.CacheHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		sockHandler: sockHandler,

		subscribeQueue: subscribeQueue,

		accountHandler: accountHandler,
		cacheHandler:   cacheHandler,
	}

	return h
}

func (h *subscribeHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})
	log.Info("Creating rabbitmq queue for listen.")

	// declare the queue for subscribe
	if err := h.sockHandler.QueueCreate(h.subscribeQueue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for subscribeHandler. err: %v", err)
	}

	// VOIP-1407: topicPatterns/QueueBind is the sole intake mechanism -- the fanout
	// QueueSubscribe loop and the fanout-unbind step have been removed, so there is no
	// fallback left to degrade to. Any failure here is fatal. This block MUST run
	// synchronously here, BEFORE the ConsumeMessage goroutine below: QueueBind and
	// ConsumeMessage's internal basic.consume share the same underlying AMQP channel for a
	// given queue, and racing them closes the channel with an "unexpected command received"
	// 503 (reproduced in production, VOIP-1258, 2026-07-14).
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
		if errConsume := h.sockHandler.ConsumeMessage(context.Background(), h.subscribeQueue, "webhook-manager", false, false, false, 10, h.processEventRun); errConsume != nil {
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

	log := logrus.WithFields(
		logrus.Fields{
			"message": m,
		},
	)
	log.Debugf("Received subscribed event. publisher: %s, type: %s", m.Publisher, m.Type)

	var err error
	start := time.Now()
	switch {

	// customer-manager
	case m.Publisher == publisherCustomerManager && (m.Type == string(cscustomer.EventTypeCustomerCreated)):
		err = h.processEventCSCustomerCreatedUpdated(m)

	case m.Publisher == publisherCustomerManager && (m.Type == string(cscustomer.EventTypeCustomerUpdated)):
		err = h.processEventCSCustomerCreatedUpdated(m)

	// flow-manager
	case m.Publisher == publisherFlowManager && (m.Type == fmactiveflow.EventTypeActiveflowCreated || m.Type == fmactiveflow.EventTypeActiveflowUpdated):
		err = h.processEventFMActiveflowCreatedUpdated(m)

	case m.Publisher == publisherFlowManager && (m.Type == fmactiveflow.EventTypeActiveflowDeleted):
		err = h.processEventFMActiveflowDeleted(m)

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		// ignore the event
		return
	}
	elapsed := time.Since(start)
	promEventProcessTime.WithLabelValues(m.Publisher, string(m.Type)).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not process the event correctly. publisher: %s, type: %s, err: %v", m.Publisher, m.Type, err)
	}
}
