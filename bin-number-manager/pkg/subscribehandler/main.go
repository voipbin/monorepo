package subscribehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-number-manager/pkg/numberhandler"
)

// list of publishers
const (
	publisherCustomerManager = commonoutline.ServiceNameQueueManager
	publisherFlowManager     = "flow-manager"
)

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	rabbitSock rabbitmqhandler.Rabbit

	subscribeQueue   string
	subscribeTargets []string

	numberHandler numberhandler.NumberHandler
}

var (
	metricsNamespace = "number_manager"

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
	numberHandler numberhandler.NumberHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		rabbitSock:       rabbitSock,
		subscribeQueue:   subscribeQueue,
		subscribeTargets: subscribeTargets,
		numberHandler:    numberHandler,
	}

	return h
}

func (h *subscribeHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})
	log.Info("Creating rabbitmq queue for listen.")

	if err := h.rabbitSock.QueueCreate(h.subscribeQueue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for subscribeHandler. err: %v", err)
	}

	// subscribe each targets
	for _, target := range h.subscribeTargets {

		// bind each targets
		if errBind := h.rabbitSock.QueueBind(h.subscribeQueue, "", target, false, nil); errBind != nil {
			log.Errorf("Could not subscribe the target. target: %s, err: %v", target, errBind)
			return errBind
		}
	}

	// receive subscribe events
	go func() {
		for {
			if errConsume := h.rabbitSock.ConsumeMessage(h.subscribeQueue, "number-manager", false, false, false, 10, h.processEventRun); errConsume != nil {
				log.Errorf("Could not consume the request message correctly. err: %v", errConsume)
			}
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

	//// customer-manager
	// customer
	case m.Publisher == string(commonoutline.ServiceNameCustomerManager) && (m.Type == string(cucustomer.EventTypeCustomerDeleted)):
		err = h.processEventCUCustomerDeleted(ctx, m)

	//// flow-manager
	// flow
	case m.Publisher == publisherFlowManager && (m.Type == string(fmflow.EventTypeFlowDeleted)):
		err = h.processEventFMFlowDeleted(ctx, m)

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
