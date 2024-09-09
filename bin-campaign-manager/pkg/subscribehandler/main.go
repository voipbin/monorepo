package subscribehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package subscribehandler -destination ./mock_subscribehandler_subscribehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	"monorepo/bin-campaign-manager/pkg/campaignhandler"
	"monorepo/bin-campaign-manager/pkg/outplanhandler"
)

// list of publishers
const (
	publisherCallManager = "call-manager"
	publisherFlowManager = "flow-manager"
)

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	rabbitSock rabbitmqhandler.Rabbit

	subscribeQueue   string
	subscribeTargets []string

	campaignHandler     campaignhandler.CampaignHandler
	campaigncallHandler campaigncallhandler.CampaigncallHandler
	outplanHandler      outplanhandler.OutplanHandler
}

var (
	metricsNamespace = "campaign_manager"

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
	campaignHandler campaignhandler.CampaignHandler,
	campaigncallHandler campaigncallhandler.CampaigncallHandler,
	outplanHandler outplanhandler.OutplanHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		rabbitSock: rabbitSock,

		subscribeQueue:   subscribeQueue,
		subscribeTargets: subscribeTargets,

		campaignHandler:     campaignHandler,
		campaigncallHandler: campaigncallHandler,
		outplanHandler:      outplanHandler,
	}

	return h
}

func (h *subscribeHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})
	log.Info("Creating rabbitmq queue for listen.")

	// declare the queue for subscribe
	if err := h.rabbitSock.QueueCreate(h.subscribeQueue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for subscribeHandler. err: %v", err)
	}

	// subscribe each targets
	for _, target := range h.subscribeTargets {
		if errSubscribe := h.rabbitSock.QueueSubscribe(h.subscribeQueue, target); errSubscribe != nil {
			log.Errorf("Could not subscribe the target. target: %s, err: %v", target, errSubscribe)
			return errSubscribe
		}
	}

	// receive subscribe events
	go func() {
		for {
			err := h.rabbitSock.ConsumeMessage(h.subscribeQueue, string(commonoutline.ServiceNameQueueManager), false, false, false, 10, h.processEventRun)
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

	ctx := context.Background()

	var err error
	start := time.Now()
	switch {

	//// call-manager
	// call
	case m.Publisher == publisherCallManager && (m.Type == string(cmcall.EventTypeCallHangup)):
		err = h.processEventCMCallHungup(ctx, m)

	//// flow-manager
	// activeflow
	case m.Publisher == publisherFlowManager && (m.Type == string(fmactiveflow.EventTypeActiveflowDeleted)):
		err = h.processEventFMActiveflowDeleted(ctx, m)

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
