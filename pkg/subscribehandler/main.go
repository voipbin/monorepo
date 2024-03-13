package subscribehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	commonoutline "gitlab.com/voipbin/bin-manager/common-handler.git/models/outline"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuecallhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuehandler"
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
	rabbitSock rabbitmqhandler.Rabbit

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
	rabbitSock rabbitmqhandler.Rabbit,
	subscribeQueue string,
	subscribeTargets []string,
	queueHandler queuehandler.QueueHandler,
	queuecallHandler queuecallhandler.QueuecallHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		rabbitSock: rabbitSock,

		subscribeQueue:    subscribeQueue,
		subscribesTargets: subscribeTargets,

		queueHandler:     queueHandler,
		queuecallHandler: queuecallHandler,
	}

	return h
}

func (h *subscribeHandler) Run() error {
	logrus.WithFields(logrus.Fields{
		"func": "Run",
	}).Info("Creating rabbitmq queue for listen.")

	// declare the queue for subscribe
	if err := h.rabbitSock.QueueDeclare(h.subscribeQueue, true, true, false, false); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// subscribe each targets
	for _, target := range h.subscribesTargets {

		// bind each targets
		if err := h.rabbitSock.QueueBind(h.subscribeQueue, "", target, false, nil); err != nil {
			logrus.Errorf("Could not subscribe the target. target: %s, err: %v", target, err)
			return err
		}
	}

	// receive subscribe events
	go func() {
		for {
			err := h.rabbitSock.ConsumeMessageOpt(h.subscribeQueue, "queue-manager", false, false, false, 10, h.processEventRun)
			if err != nil {
				logrus.Errorf("Could not consume the request message correctly. err: %v", err)
			}
		}
	}()

	return nil
}

// processEventRun runs the processEvent
func (h *subscribeHandler) processEventRun(m *rabbitmqhandler.Event) error {
	go h.processEvent(m)

	return nil
}

// processEvent processes the event message
func (h *subscribeHandler) processEvent(m *rabbitmqhandler.Event) {
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
	case m.Publisher == publisherCallManager && (m.Type == string(cmcall.EventTypeCallHangup)):
		err = h.processEventCMCallHangup(ctx, m)

	// confbridge
	case m.Publisher == publisherCallManager && (m.Type == string(cmconfbridge.EventTypeConfbridgeJoined)):
		err = h.processEventCMConfbridgeJoined(ctx, m)

	case m.Publisher == publisherCallManager && (m.Type == string(cmconfbridge.EventTypeConfbridgeLeaved)):
		err = h.processEventCMConfbridgeLeaved(ctx, m)

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
