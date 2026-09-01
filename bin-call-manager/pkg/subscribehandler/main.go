package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	"monorepo/bin-call-manager/models/common"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	smcontainer "monorepo/bin-sentinel-manager/models/container"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/pkg/arieventhandler"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
)

// SubscribeHandler intreface for subscribed event listen handler
type SubscribeHandler interface {
	Run() error
}

// topicPatterns is this service's bind set on the global topic exchange
// `bin-manager.event` (VOIP-1406): one pattern per dispatch pair handled in
// processEvent. The asterisk-proxy leg is deliberately NOT here --
// asterisk-proxy does not publish to the topic exchange, so its
// `asterisk.all.event` fanout subscription is permanently retained.
// Pinned by binding_golden_test.go.
var topicPatterns = []string{
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCustomerManager), cucustomer.EventTypeCustomerDeleted),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCustomerManager), cucustomer.EventTypeCustomerFrozen),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameFlowManager), fmactiveflow.EventTypeActiveflowUpdated),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameSentinelManager), smcontainer.EventTypeContainerDied),
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler

	subscribeQueue commonoutline.QueueName

	ariEventHandler arieventhandler.ARIEventHandler

	callHandler       callhandler.CallHandler
	groupcallHandler  groupcallhandler.GroupcallHandler
	confbridgeHandler confbridgehandler.ConfbridgeHandler
}

var (
	metricsNamespace = commonoutline.GetMetricNameSpace(common.Servicename)

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
	ariEventHandler arieventhandler.ARIEventHandler,
	callHandler callhandler.CallHandler,
	groupcallHandler groupcallhandler.GroupcallHandler,
	confbridgeHandler confbridgehandler.ConfbridgeHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		sockHandler:       sock,
		subscribeQueue:    subscribeQueue,
		ariEventHandler:   ariEventHandler,
		callHandler:       callHandler,
		groupcallHandler:  groupcallHandler,
		confbridgeHandler: confbridgeHandler,
	}

	return h
}

// Run starts to receive ARI event and process it.
func (h *subscribeHandler) Run() error {
	// create queue for ari event receive
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})
	log.Infof("Creating rabbitmq queue for ARI event receiving.")

	// declare the queue for subscribe
	if err := h.sockHandler.QueueCreate(string(h.subscribeQueue), "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for subscribeHandler. err: %v", err)
	}

	// The asterisk fanout leg is permanently retained (VOIP-1407 design §3.2):
	// asterisk-proxy does not publish to the global topic exchange, so this is the one
	// fanout QueueSubscribe that survives the fanout-cutover, subscribed as a standalone
	// statement rather than through the (now-deleted) generic fanout target loop.
	if errSubscribe := h.sockHandler.QueueSubscribe(string(h.subscribeQueue), string(commonoutline.QueueNameAsteriskEventAll)); errSubscribe != nil {
		return fmt.Errorf("could not subscribe to the asterisk fanout exchange. err: %v", errSubscribe)
	}

	// Bind this service's inter-service event intake to the global topic exchange
	// `bin-manager.event` (VOIP-1406). topicPatterns is now the sole intake mechanism
	// for inter-service events (fanout publishing was removed in VOIP-1407) -- a declare
	// or bind failure here fails Run() immediately; there is no fanout fallback left to
	// degrade to.
	//
	// CRITICAL: this MUST run synchronously here, BEFORE the ConsumeMessage goroutine
	// below. QueueBind and ConsumeMessage's basic.consume share the SAME underlying AMQP
	// channel; racing them makes the broker close the channel with a 503 (reproduced in
	// production, VOIP-1258, 2026-07-14).
	if errDeclare := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic"); errDeclare != nil {
		return fmt.Errorf("could not declare the global topic exchange. err: %v", errDeclare)
	}

	for _, pattern := range topicPatterns {
		if errBind := h.sockHandler.QueueBind(string(h.subscribeQueue), pattern, string(commonoutline.QueueNameEvent), false, nil); errBind != nil {
			return fmt.Errorf("could not bind the topic pattern. pattern: %s, err: %v", pattern, errBind)
		}
	}

	// receive subscribe events
	go func() {
		if errConsume := h.sockHandler.ConsumeMessage(context.Background(), string(h.subscribeQueue), string(common.Servicename), false, false, false, 20, h.processEventRun); errConsume != nil {
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
	// asterisk-proxy
	case m.Publisher == string(commonoutline.ServiceNameAsteriskProxy):
		err = h.processEventAsteriskProxy(ctx, m)

	// customer-manager
	case m.Publisher == string(commonoutline.ServiceNameCustomerManager) && m.Type == cucustomer.EventTypeCustomerDeleted:
		err = h.processEventCUCustomerDeleted(ctx, m)

	case m.Publisher == string(commonoutline.ServiceNameCustomerManager) && m.Type == cucustomer.EventTypeCustomerFrozen:
		err = h.processEventCUCustomerFrozen(ctx, m)

	// flow-manager
	case m.Publisher == string(commonoutline.ServiceNameFlowManager) && m.Type == fmactiveflow.EventTypeActiveflowUpdated:
		err = h.processEventFMActiveflowUpdated(ctx, m)

	// sentinel-manager
	case m.Publisher == string(commonoutline.ServiceNameSentinelManager) && m.Type == smcontainer.EventTypeContainerDied:
		err = h.processEventSMContainerDied(ctx, m)

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
