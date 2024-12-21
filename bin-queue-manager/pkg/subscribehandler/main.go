package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-queue-manager/pkg/queuecallhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
)

// list of publishers
const (
	publisherCallManager = string(commonoutline.ServiceNameCallManager)
)

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler

	subscribeQueue    string
	subscribesTargets []string

	queueHandler     queuehandler.QueueHandler
	queuecallHandler queuecallhandler.QueuecallHandler
}

var (
	metricsNamespace = "queue_manager"

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
	queueHandler queuehandler.QueueHandler,
	queuecallHandler queuecallhandler.QueuecallHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		sockHandler: sockHandler,

		subscribeQueue:    subscribeQueue,
		subscribesTargets: subscribeTargets,

		queueHandler:     queueHandler,
		queuecallHandler: queuecallHandler,
	}

	return h
}

func (h *subscribeHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})
	log.Info("Creating rabbitmq queue for listen.")

	if err := h.sockHandler.QueueCreate(h.subscribeQueue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// subscribe each targets
	for _, target := range h.subscribesTargets {
		if errSubscribe := h.sockHandler.QueueSubscribe(h.subscribeQueue, target); errSubscribe != nil {
			log.Errorf("Could not subscribe the target. target: %s, err: %v", target, errSubscribe)
			return errSubscribe
		}
	}

	// receive subscribe events
	go func() {
		if errConsume := h.sockHandler.ConsumeMessage(context.Background(), h.subscribeQueue, "queue-manager", false, false, false, 10, h.processEventRun); errConsume != nil {
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
			"func":  "processEvent",
			"event": m,
		},
	)

	var err error
	start := time.Now()
	ctx := context.Background()

	switch {

	//// call-manager
	// call
	case m.Publisher == string(commonoutline.ServiceNameCallManager) && (m.Type == string(cmcall.EventTypeCallHangup)):
		err = h.processEventCMCallHangup(ctx, m)

	// confbridge
	case m.Publisher == string(commonoutline.ServiceNameCallManager) && (m.Type == string(cmconfbridge.EventTypeConfbridgeJoined)):
		err = h.processEventCMConfbridgeJoined(ctx, m)

	case m.Publisher == string(commonoutline.ServiceNameCallManager) && (m.Type == string(cmconfbridge.EventTypeConfbridgeLeaved)):
		err = h.processEventCMConfbridgeLeaved(ctx, m)

	//// customer-manager
	// customer
	case m.Publisher == string(commonoutline.ServiceNameCustomerManager) && (m.Type == string(cucustomer.EventTypeCustomerDeleted)):
		err = h.processEventCUCustomerDeleted(ctx, m)

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
