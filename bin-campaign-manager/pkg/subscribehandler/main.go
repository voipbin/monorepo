package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_subscribehandler_subscribehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

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

// topicPatterns is this service's bind set on the global topic exchange
// `bin-manager.event` (VOIP-1406): one pattern per dispatch pair handled in
// processEvent. Pinned byte-for-byte by binding_golden_test.go.
var topicPatterns = []string{
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCallManager), cmcall.EventTypeCallHangup),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameFlowManager), fmactiveflow.EventTypeActiveflowDeleted),
}

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler

	subscribeQueue string

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
	sockHandler sockhandler.SockHandler,
	subscribeQueue string,
	campaignHandler campaignhandler.CampaignHandler,
	campaigncallHandler campaigncallhandler.CampaigncallHandler,
	outplanHandler outplanhandler.OutplanHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		sockHandler: sockHandler,

		subscribeQueue: subscribeQueue,

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
	if err := h.sockHandler.QueueCreate(h.subscribeQueue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for subscribeHandler. err: %v", err)
	}

	// VOIP-1407: topicPatterns/QueueBind is the sole intake mechanism -- the fanout
	// QueueSubscribe loop and the fanout-unbind step have been removed, so there is no
	// fallback left to degrade to. Any failure here is fatal.
	if err := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic"); err != nil {
		return fmt.Errorf("could not declare the global topic exchange. err: %v", err)
	}

	for _, pattern := range topicPatterns {
		if err := h.sockHandler.QueueBind(h.subscribeQueue, pattern, string(commonoutline.QueueNameEvent), false, nil); err != nil {
			return fmt.Errorf("could not bind the topic pattern. pattern: %s, err: %v", pattern, err)
		}
	}

	// receive subscribe events
	go func() {
		if errConsume := h.sockHandler.ConsumeMessage(context.Background(), h.subscribeQueue, string(commonoutline.ServiceNameCampaignManager), false, false, false, 10, h.processEventRun); errConsume != nil {
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
