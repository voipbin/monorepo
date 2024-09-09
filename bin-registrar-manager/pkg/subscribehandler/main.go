package subscribehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"monorepo/bin-registrar-manager/pkg/extensionhandler"
	"monorepo/bin-registrar-manager/pkg/trunkhandler"
)

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	rabbitSock rabbitmqhandler.Rabbit

	subscribeQueue    string
	subscribesTargets []string

	extensionHandler extensionhandler.ExtensionHandler
	trunkHandler     trunkhandler.TrunkHandler
}

var (
	metricsNamespace = commonoutline.GetMetricNameSpace(commonoutline.ServiceNameRegistrarManager)

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
	extensionHandler extensionhandler.ExtensionHandler,
	trunkHandler trunkhandler.TrunkHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		rabbitSock:        rabbitSock,
		subscribeQueue:    subscribeQueue,
		subscribesTargets: subscribeTargets,
		extensionHandler:  extensionHandler,
		trunkHandler:      trunkHandler,
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
	for _, target := range h.subscribesTargets {
		if errSubscribe := h.rabbitSock.QueueSubscribe(h.subscribeQueue, target); errSubscribe != nil {
			log.Errorf("Could not subscribe the target. target: %s, err: %v", target, errSubscribe)
			return errSubscribe
		}
	}

	// receive subscribe events
	go func() {
		for {
			if errConsume := h.rabbitSock.ConsumeMessage(h.subscribeQueue, string(commonoutline.ServiceNameRegistrarManager), false, false, false, 10, h.processEventRun); errConsume != nil {
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
