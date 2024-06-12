package subscribehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	amresource "monorepo/bin-agent-manager/models/resource"
	wmwebhook "monorepo/bin-webhook-manager/models/webhook"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/pkg/zmqpubhandler"
)

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	rabbitSock rabbitmqhandler.Rabbit

	subscribeQueueNamePod string // subscribe queue name for this pod
	subscribeTargets      []string

	zmqpubHandler zmqpubhandler.ZMQPubHandler
}

var (
	metricsNamespace = "api_manager"

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
	subscribeQueueName string,
	subscribeTargets []string,
	zmqpubHandler zmqpubhandler.ZMQPubHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		rabbitSock: rabbitSock,

		subscribeQueueNamePod: subscribeQueueName,
		subscribeTargets:      subscribeTargets,

		zmqpubHandler: zmqpubHandler,
	}

	return h
}

func (h *subscribeHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})
	log.Info("Creating rabbitmq queue for listen.")

	// declare the queue for subscribe(pod)
	log.Debugf("Declaring the queue for subscribe(pod). queue_name: %s", h.subscribeQueueNamePod)
	if err := h.rabbitSock.QueueDeclare(h.subscribeQueueNamePod, false, true, false, false); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// subscribe each targets
	for _, target := range h.subscribeTargets {

		// bind each targets
		if err := h.rabbitSock.QueueBind(h.subscribeQueueNamePod, "", target, false, nil); err != nil {
			logrus.Errorf("Could not subscribe the target. target: %s, err: %v", target, err)
			return err
		}
	}

	// receive subscribe events
	go func() {
		for {
			err := h.rabbitSock.ConsumeMessageOpt(h.subscribeQueueNamePod, string(commonoutline.ServiceNameAPIManager), false, false, false, 10, h.processEventRun)
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
			"message": m,
		},
	)
	log.Debugf("Received subscribed event. publisher: %s, type: %s", m.Publisher, m.Type)

	var err error
	start := time.Now()
	ctx := context.Background()

	switch {

	//// agent-managet
	case m.Publisher == string(commonoutline.ServiceNameAgentManager) &&
		(m.Type == string(amresource.EventTypeResourceCreated) || m.Type == string(amresource.EventTypeResourceUpdated) || m.Type == string(amresource.EventTypeResourceDeleted)):
		err = h.processEventAgentManagerResourcePublished(m)

	//// webhook-managet
	case m.Publisher == string(commonoutline.ServiceNameWebhookManager) && (m.Type == string(wmwebhook.EventTypeWebhookPublished)):
		err = h.processEventWebhookManagerWebhookPublished(ctx, m)

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
