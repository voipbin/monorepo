package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/sirupsen/logrus"

	"monorepo/bin-agent-manager/pkg/agenthandler"
	"monorepo/bin-agent-manager/pkg/metricshandler"
)

// topicPatterns is this service's bind set on the global topic exchange
// `bin-manager.event` (VOIP-1406): one pattern per dispatch pair handled in
// processEvent. Pinned byte-for-byte by binding_golden_test.go.
var topicPatterns = []string{
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCallManager), cmgroupcall.EventTypeGroupcallCreated),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCallManager), cmgroupcall.EventTypeGroupcallProgressing),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCustomerManager), cmcustomer.EventTypeCustomerDeleted),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCustomerManager), cmcustomer.EventTypeCustomerCreated),
}

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler

	subscribeQueue string

	agentHandler agenthandler.AgentHandler
}

// ensure metricshandler init() registers all metrics
var _ = metricshandler.ReceivedSubscribeEventProcessTime

// NewSubscribeHandler return SubscribeHandler interface
func NewSubscribeHandler(
	sockHandler sockhandler.SockHandler,
	subscribeQueue string,
	agentHandler agenthandler.AgentHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		sockHandler:    sockHandler,
		subscribeQueue: subscribeQueue,
		agentHandler:   agentHandler,
	}

	return h
}

func (h *subscribeHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})
	log.Info("Creating rabbitmq queue for subscribing.")

	if err := h.sockHandler.QueueCreate(h.subscribeQueue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// Cut over from the old fanout QueueNameWebhookEvent exchange to the new
	// QueueNameWebhookEventTopic topic exchange with a "#" wildcard binding.
	// Bind new first, then unbind old, to avoid an event-loss window where the
	// queue is briefly bound to neither exchange (VOIP-1258 Task 3.5).
	//
	// CRITICAL: this MUST run synchronously here, BEFORE ConsumeMessage is started below (not
	// after Run() returns, as it originally lived in cmd/agent-manager/main.go). QueueBind/
	// QueueUnbind and ConsumeMessage's internal channel.Consume() share the SAME underlying
	// AMQP channel object (rabbitmqhandler's queue.channel) for a given queue name. AMQP does
	// not allow two synchronous RPCs to race on one channel -- if ConsumeMessage's
	// basic.consume is already in flight on another goroutine when QueueBind/QueueUnbind fires,
	// the broker closes the channel with "unexpected command received" (503), and
	// ConsumeMessage fails to ever start consuming on this pod. This exact race was reproduced
	// in production (VOIP-1258 PR #1101 round-2 post-deploy verification, 2026-07-14): one of
	// two agent-manager pods hit this 503, silently never registered as a consumer, and a
	// message ended up stuck unacknowledged on the queue for the pod that DID consume it.
	// Sequencing this before the async ConsumeMessage goroutine below eliminates the race.
	if errBind := h.sockHandler.QueueBind(h.subscribeQueue, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil); errBind != nil {
		log.Errorf("Could not bind to the topic exchange. err: %v", errBind)
		// do NOT proceed to unbind the old exchange if this bind failed -- stay on the
		// old exchange rather than risk ending up bound to neither.
	} else if errUnbind := h.sockHandler.QueueUnbind(h.subscribeQueue, "", string(commonoutline.QueueNameWebhookEvent), nil); errUnbind != nil {
		log.Errorf("CRITICAL: Could not unbind from the old fanout exchange after binding to the new topic exchange. queue: %s is now bound to BOTH exchanges (double-processing resumes). Manual intervention required. err: %v", h.subscribeQueue, errUnbind)
	}

	// VOIP-1406: the subscribe queue's sole intake mechanism is pattern bindings on
	// the global topic exchange bin-manager.event -- unconditional, not gated behind
	// "if the fanout declare failed, stay on fanout" (VOIP-1407 removed the fanout
	// fallback entirely). Same ordering constraint as the VOIP-1258 block above: this
	// MUST run synchronously here, BEFORE ConsumeMessage is started below, because the
	// bind RPCs share the queue's AMQP channel with basic.consume. A declare or bind
	// failure here is fatal: with no fanout fallback left to degrade to, the service
	// would otherwise silently intake zero events forever, which is strictly worse
	// than a loud boot failure the orchestrator will restart-loop on.
	if errDeclare := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic"); errDeclare != nil {
		return fmt.Errorf("could not declare the global topic exchange. err: %v", errDeclare)
	}

	for _, pattern := range topicPatterns {
		if errBind := h.sockHandler.QueueBind(h.subscribeQueue, pattern, string(commonoutline.QueueNameEvent), false, nil); errBind != nil {
			return fmt.Errorf("could not bind the topic pattern. pattern: %s, err: %v", pattern, errBind)
		}
	}

	// receive subscribe events
	go func() {
		if errConsume := h.sockHandler.ConsumeMessage(context.Background(), h.subscribeQueue, string(commonoutline.ServiceNameAgentManager), false, false, false, 10, h.processEventRun); errConsume != nil {
			log.Errorf("Could not consume the request message correctly. err: %v", errConsume)
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
	log.WithField("event", m).Debugf("Received subscribed event. publisher: %s, type: %s", m.Publisher, m.Type)

	var err error
	start := time.Now()
	switch {

	//// call-manager
	case m.Publisher == string(commonoutline.ServiceNameCallManager):
		switch m.Type {

		// groupcall
		case string(cmgroupcall.EventTypeGroupcallCreated):
			err = h.processEventCMGroupcallCreated(ctx, m)

		case string(cmgroupcall.EventTypeGroupcallProgressing):
			err = h.processEventCMGroupcallProgressing(ctx, m)
		}

	//// customer-manager
	// customer
	case m.Publisher == string(commonoutline.ServiceNameCustomerManager) && (m.Type == string(cmcustomer.EventTypeCustomerDeleted)):
		err = h.processEventCMCustomerDeleted(ctx, m)

	case m.Publisher == string(commonoutline.ServiceNameCustomerManager) && (m.Type == string(cmcustomer.EventTypeCustomerCreated)):
		err = h.processEventCMCustomerCreated(ctx, m)

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		// ignore the event.
		return
	}
	elapsed := time.Since(start)
	metricshandler.ReceivedSubscribeEventProcessTime.WithLabelValues(m.Publisher, string(m.Type)).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not process the event correctly. publisher: %s, type: %s, err: %v", m.Publisher, m.Type, err)
	}
}
