package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

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
	sockHandler sockhandler.SockHandler
	reqHandler  requesthandler.RequestHandler

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
	sockHandler sockhandler.SockHandler,
	reqHandler requesthandler.RequestHandler,
	subscribeQueueName string,
	subscribeTargets []string,
	zmqpubHandler zmqpubhandler.ZMQPubHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		sockHandler: sockHandler,
		reqHandler:  reqHandler,

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
	if err := h.sockHandler.QueueCreate(h.subscribeQueueNamePod, "volatile"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// subscribe each targets
	for _, target := range h.subscribeTargets {
		if errSubscribe := h.sockHandler.QueueSubscribe(h.subscribeQueueNamePod, target); errSubscribe != nil {
			log.Errorf("Could not subscribe the target. target: %s, err: %v", target, errSubscribe)
			return errSubscribe
		}
	}

	// baseline "#" wildcard binding to the new topic exchange (VOIP-1258 §7 round-2 finding):
	// a topic-kind exchange's empty-key bind (what QueueSubscribe used for the old fanout
	// exchange) only matches messages published with an empty routing key, and every
	// VOIP-1258 publish path uses non-empty scope-first keys, so this pod would receive ZERO
	// events without this explicit bind.
	//
	// CRITICAL: this MUST run synchronously here, BEFORE ConsumeMessage is started below (not
	// after Run() returns, as it originally lived in cmd/api-manager/main.go). QueueBind and
	// ConsumeMessage's internal channel.Consume() share the SAME underlying AMQP channel
	// object (rabbitmqhandler's queue.channel) for a given queue name. AMQP does not allow two
	// synchronous RPCs to race on one channel -- if ConsumeMessage's basic.consume is already
	// in flight on another goroutine when QueueBind fires, the broker closes the channel with
	// "unexpected command received" (503), and ConsumeMessage fails to ever start consuming on
	// this pod. This exact race was reproduced in production in bin-agent-manager (VOIP-1258 PR
	// #1101 round-2 post-deploy verification, 2026-07-14) via the identical after-Run() call
	// site pattern that this file also originally used -- fixed here proactively for the same
	// reason before it recurs on this service too.
	if err := h.sockHandler.QueueBind(h.subscribeQueueNamePod, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil); err != nil {
		log.Errorf("Could not bind to the topic exchange. err: %v", err)
	}

	// receive subscribe events
	go func() {
		if errConsume := h.sockHandler.ConsumeMessage(context.Background(), h.subscribeQueueNamePod, string(commonoutline.ServiceNameAPIManager), false, false, false, 10, h.processEventRun); errConsume != nil {
			logrus.Errorf("Could not consume the request message correctly. err: %v", errConsume)
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
	ctx := context.Background()

	switch {

	//// webhook-manager
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
