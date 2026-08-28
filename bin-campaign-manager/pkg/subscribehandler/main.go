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
	eventtopic.PatternAction(string(commonoutline.ServiceNameCallManager), "call", "hangup"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameFlowManager), "activeflow", "deleted"),
}

// fanoutUnbindTargets lists the per-service fanout event exchanges the subscribe
// queue unbinds from once every topicPatterns bind succeeded (VOIP-1406). It is
// the service's subscribeTargets set; the fanout QueueSubscribe calls themselves
// stay in Run() as the rollback surface until VOIP-1407.
var fanoutUnbindTargets = []string{
	string(commonoutline.QueueNameCallEvent),
	string(commonoutline.QueueNameFlowEvent),
}

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler

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
	sockHandler sockhandler.SockHandler,
	subscribeQueue string,
	subscribeTargets []string,
	campaignHandler campaignhandler.CampaignHandler,
	campaigncallHandler campaigncallhandler.CampaigncallHandler,
	outplanHandler outplanhandler.OutplanHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		sockHandler: sockHandler,

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

	// VOIP-1406: migrate the subscribe queue from the per-service fanout event
	// exchanges to pattern bindings on the global topic exchange bin-manager.event.
	// This MUST run synchronously here, BEFORE ConsumeMessage is started below,
	// because the bind/unbind RPCs share the queue's AMQP channel with
	// basic.consume (the VOIP-1258 2026-07-14 503 channel race).
	// Bind new before unbinding old (no window bound to neither), all-or-nothing:
	// any pattern bind failure rolls back the partial topic binds (best-effort)
	// and unbinds NO fanout exchange, leaving the service fully on fanout. The
	// fanout QueueSubscribe calls above stay as the rollback surface until
	// VOIP-1407.
	if errDeclare := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic"); errDeclare != nil {
		log.Errorf("Could not declare the global topic exchange. Staying fully on the fanout subscriptions. exchange: %s, err: %v", string(commonoutline.QueueNameEvent), errDeclare)
	} else {
		bound := []string{}
		ok := true
		for _, pattern := range topicPatterns {
			if errBind := h.sockHandler.QueueBind(h.subscribeQueue, pattern, string(commonoutline.QueueNameEvent), false, nil); errBind != nil {
				log.Errorf("Could not bind the topic pattern. pattern: %s, err: %v", pattern, errBind)
				ok = false
				break
			}
			bound = append(bound, pattern)
		}

		if !ok {
			// best-effort rollback of the partial topic binds, then stay fully on
			// fanout. An unbind failure here leaves partial double delivery.
			for _, pattern := range bound {
				if errUnbind := h.sockHandler.QueueUnbind(h.subscribeQueue, pattern, string(commonoutline.QueueNameEvent), nil); errUnbind != nil {
					log.Errorf("CRITICAL: partial topic bind could not be rolled back. queue: %s, pattern: %s, err: %v", h.subscribeQueue, pattern, errUnbind)
				}
			}
		} else {
			// unbind the old fanout exchanges only after EVERY pattern bound.
			// Unbind failure: CRITICAL log, not fatal (double delivery beats loss).
			for _, target := range fanoutUnbindTargets {
				if errUnbind := h.sockHandler.QueueUnbind(h.subscribeQueue, "", target, nil); errUnbind != nil {
					log.Errorf("CRITICAL: could not unbind the old fanout exchange after the topic binds. queue: %s is now bound to BOTH exchanges (double delivery). Manual intervention required. exchange: %s, err: %v", h.subscribeQueue, target, errUnbind)
				}
			}
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
