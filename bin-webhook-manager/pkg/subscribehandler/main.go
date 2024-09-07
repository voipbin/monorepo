package subscribehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package subscribehandler -destination ./mock_subscribehandler.go -source main.go -build_flags=-mod=mod

import (
	"fmt"
	"strings"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-webhook-manager/pkg/accounthandler"
)

// list of publishers
const (
	publisherCustomerManager = "customer-manager"
)

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	rabbitSock rabbitmqhandler.Rabbit

	subscribeQueue    string
	subscribesTargets string

	accountHandler accounthandler.AccountHandler
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
	rabbitSock rabbitmqhandler.Rabbit,
	subscribeQueue string,
	subscribeTargets string,
	accountHandler accounthandler.AccountHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		rabbitSock: rabbitSock,

		subscribeQueue:    subscribeQueue,
		subscribesTargets: subscribeTargets,

		accountHandler: accountHandler,
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
	targets := strings.Split(h.subscribesTargets, ",")
	for _, target := range targets {

		// bind each targets
		if err := h.rabbitSock.QueueBind(h.subscribeQueue, "", target, false, nil); err != nil {
			logrus.Errorf("Could not subscribe the target. target: %s, err: %v", target, err)
			return err
		}
	}

	// receive subscribe events
	go func() {
		for {
			err := h.rabbitSock.ConsumeMessage(h.subscribeQueue, "webhook-manager", false, false, false, 10, h.processEventRun)
			if err != nil {
				logrus.Errorf("Could not consume the request message correctly. err: %v", err)
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
