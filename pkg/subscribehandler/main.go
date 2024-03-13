package subscribehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	commonoutline "gitlab.com/voipbin/bin-manager/common-handler.git/models/outline"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	cmcustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/agenthandler"
)

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	rabbitSock rabbitmqhandler.Rabbit

	subscribeQueue    string
	subscribesTargets []string

	agentHandler agenthandler.AgentHandler
}

var (
	metricsNamespace = commonoutline.GetMetricNameSpace(commonoutline.ServiceNameAgentManager)

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
	agentHandler agenthandler.AgentHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		rabbitSock:        rabbitSock,
		subscribeQueue:    subscribeQueue,
		subscribesTargets: subscribeTargets,
		agentHandler:      agentHandler,
	}

	return h
}

func (h *subscribeHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})
	log.Info("Creating rabbitmq queue for listen.")

	// declare the queue for subscribe
	if err := h.rabbitSock.QueueDeclare(h.subscribeQueue, true, true, false, false); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// subscribe each targets
	for _, target := range h.subscribesTargets {

		// bind each targets
		if errBind := h.rabbitSock.QueueBind(h.subscribeQueue, "", target, false, nil); errBind != nil {
			log.Errorf("Could not subscribe the target. target: %s, err: %v", target, errBind)
			return errBind
		}
	}

	// receive subscribe events
	go func() {
		for {
			if errConsume := h.rabbitSock.ConsumeMessageOpt(h.subscribeQueue, string(commonoutline.ServiceNameAgentManager), false, false, false, 10, h.processEventRun); errConsume != nil {
				log.Errorf("Could not consume the request message correctly. err: %v", errConsume)
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
	log := logrus.WithFields(logrus.Fields{
		"func":    "processEvent",
		"message": m,
	})
	ctx := context.Background()

	var err error
	start := time.Now()
	switch {

	//// call-manager
	// groupcall
	case m.Publisher == string(commonoutline.ServiceNameCallManager) && (m.Type == string(cmgroupcall.EventTypeGroupcallCreated)):
		err = h.processEventCMGroupcallCreated(ctx, m)

	case m.Publisher == string(commonoutline.ServiceNameCallManager) && (m.Type == string(cmgroupcall.EventTypeGroupcallProgressing)):
		err = h.processEventCMGroupcallProgressing(ctx, m)

	//// customer-manager
	// customer
	case m.Publisher == string(commonoutline.ServiceNameCustomerManager) && (m.Type == string(cmcustomer.EventTypeCustomerDeleted)):
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
