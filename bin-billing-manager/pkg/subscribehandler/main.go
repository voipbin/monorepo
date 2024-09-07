package subscribehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"
	mmmessage "monorepo/bin-message-manager/models/message"

	nmnumber "monorepo/bin-number-manager/models/number"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/billinghandler"
)

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	rabbitSock rabbitmqhandler.Rabbit

	subscribeQueue   string
	subscribeTargets []string

	accountHandler accounthandler.AccountHandler
	billingHandler billinghandler.BillingHandler
}

var (
	metricsNamespace = "billing_manager"

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
	accountHandler accounthandler.AccountHandler,
	billingHandler billinghandler.BillingHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		rabbitSock:       rabbitSock,
		subscribeQueue:   subscribeQueue,
		subscribeTargets: subscribeTargets,

		accountHandler: accountHandler,
		billingHandler: billingHandler,
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
	for _, target := range h.subscribeTargets {

		// bind each targets
		if err := h.rabbitSock.QueueBind(h.subscribeQueue, "", target, false, nil); err != nil {
			logrus.Errorf("Could not subscribe the target. target: %s, err: %v", target, err)
			return err
		}
	}

	// receive subscribe events
	go func() {
		for {
			err := h.rabbitSock.ConsumeMessage(h.subscribeQueue, string(commonoutline.ServiceNameBillingManager), false, false, false, 10, h.processEventRun)
			if err != nil {
				logrus.Errorf("Could not consume the request message correctly. err: %v", err)
			}
		}
	}()

	return nil
}

// processEventRun runs the processEvent
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

// processEvent processes the event message
func (h *subscribeHandler) processEvent(m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processEvent",
		"message": m,
	})
	log.Debugf("Received subscribed event. publisher: %s, type: %s", m.Publisher, m.Type)
	ctx := context.Background()

	var err error

	start := time.Now()
	switch {

	//// call-manager
	// call
	case m.Publisher == string(commonoutline.ServiceNameCallManager) && m.Type == cmcall.EventTypeCallProgressing:
		err = h.processEventCMCallProgressing(ctx, m)

	case m.Publisher == string(commonoutline.ServiceNameCallManager) && m.Type == cmcall.EventTypeCallHangup:
		err = h.processEventCMCallHangup(ctx, m)

	//// message-manager
	// message
	case m.Publisher == string(commonoutline.ServiceNameMessageManager) && m.Type == mmmessage.EventTypeMessageCreated:
		err = h.processEventMMMessageCreated(ctx, m)

	//// customer-manager
	// customer
	case m.Publisher == string(commonoutline.ServiceNameCustomerManager) && m.Type == cscustomer.EventTypeCustomerDeleted:
		err = h.processEventCMCustomerDeleted(ctx, m)

	//// number-manager
	// number
	case m.Publisher == string(commonoutline.ServiceNameNumberManager) && m.Type == nmnumber.EventTypeNumberCreated:
		err = h.processEventNMNumberCreated(ctx, m)

	case m.Publisher == string(commonoutline.ServiceNameNumberManager) && m.Type == nmnumber.EventTypeNumberRenewed:
		err = h.processEventNMNumberRenewed(ctx, m)

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		// nothing to do
		return nil
	}
	elapsed := time.Since(start)
	promEventProcessTime.WithLabelValues(m.Publisher, string(m.Type)).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not process the event correctly. publisher: %s, type: %s, err: %v", m.Publisher, m.Type, err)
		return errors.Wrap(err, "could not process the event correctly.")
	}

	return nil
}
