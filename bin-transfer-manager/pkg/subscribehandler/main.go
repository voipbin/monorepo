package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transfer-manager/pkg/transferhandler"
)

// list of publishers
const (
	publisherCallManager = "call-manager"
)

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	serviceName string

	sockHandler sockhandler.SockHandler

	subscribeQueue   string
	subscribeTargets []string

	transferHandler transferhandler.TransferHandler
}

var (
	metricsNamespace = "conference_manager"

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
	serviceName string,
	sockHandler sockhandler.SockHandler,
	subscribeQueue string,
	subscribeTargets []string,
	transferHandler transferhandler.TransferHandler,
) SubscribeHandler {

	h := &subscribeHandler{
		serviceName:      serviceName,
		sockHandler:      sockHandler,
		subscribeQueue:   subscribeQueue,
		subscribeTargets: subscribeTargets,

		transferHandler: transferHandler,
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

	// subscribe each targets
	for _, target := range h.subscribeTargets {
		if errSubscribe := h.sockHandler.QueueSubscribe(h.subscribeQueue, target); errSubscribe != nil {
			log.Errorf("Could not subscribe the target. target: %s, err: %v", target, errSubscribe)
			return errSubscribe
		}
	}

	// receive subscribe events
	go func() {
		if errConsume := h.sockHandler.ConsumeMessage(context.Background(), h.subscribeQueue, h.serviceName, false, false, false, 10, h.processEventRun); errConsume != nil {
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

	var err error
	start := time.Now()
	ctx := context.Background()
	switch {

	//// call-manager
	// groupcall
	case m.Publisher == publisherCallManager && m.Type == string(cmgroupcall.EventTypeGroupcallProgressing):
		err = h.processEventCMGroupcallProgressing(ctx, m)

	case m.Publisher == publisherCallManager && m.Type == string(cmgroupcall.EventTypeGroupcallHangup):
		err = h.processEventCMGroupcallHangup(ctx, m)

	// call
	case m.Publisher == publisherCallManager && m.Type == string(cmcall.EventTypeCallHangup):
		err = h.processEventCMCallHangup(ctx, m)

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		// no event handler found
		return
	}
	elapsed := time.Since(start)
	promEventProcessTime.WithLabelValues(m.Publisher, string(m.Type)).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not process the event correctly. publisher: %s, type: %s, err: %v", m.Publisher, m.Type, err)
	}
}
