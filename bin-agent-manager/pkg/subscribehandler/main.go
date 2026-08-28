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
	eventtopic.PatternAction(string(commonoutline.ServiceNameCallManager), "groupcall", "created"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameCallManager), "groupcall", "progressing"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameCustomerManager), "customer", "deleted"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameCustomerManager), "customer", "created"),
}

// fanoutUnbindTargets lists the per-service fanout event exchanges the subscribe
// queue unbinds from once every topicPatterns bind succeeded (VOIP-1406). It is
// the service's subscribeTargets set; the fanout QueueSubscribe calls themselves
// stay in Run() as the rollback surface until VOIP-1407.
var fanoutUnbindTargets = []string{
	string(commonoutline.QueueNameCallEvent),
	string(commonoutline.QueueNameCustomerEvent),
}

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler

	subscribeQueue   string
	subscribeTargets []string

	agentHandler agenthandler.AgentHandler
}

// ensure metricshandler init() registers all metrics
var _ = metricshandler.ReceivedSubscribeEventProcessTime

// NewSubscribeHandler return SubscribeHandler interface
func NewSubscribeHandler(
	sockHandler sockhandler.SockHandler,
	subscribeQueue string,
	subscribeTargets []string,
	agentHandler agenthandler.AgentHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		sockHandler:      sockHandler,
		subscribeQueue:   subscribeQueue,
		subscribeTargets: subscribeTargets,
		agentHandler:     agentHandler,
	}

	return h
}

func (h *subscribeHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "run",
		"subscribe_targets": h.subscribeTargets,
	})
	log.Info("Creating rabbitmq queue for subscribing.")

	if err := h.sockHandler.QueueCreate(h.subscribeQueue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// subscribe each targets
	for _, target := range h.subscribeTargets {
		if errSubscribe := h.sockHandler.QueueSubscribe(h.subscribeQueue, target); errSubscribe != nil {
			log.Errorf("Could not subscribe the target. target: %s, err: %v", target, errSubscribe)
			return errSubscribe
		}
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

	// VOIP-1406: migrate the subscribe queue from the per-service fanout event
	// exchanges to pattern bindings on the global topic exchange bin-manager.event.
	// Same ordering constraint as the VOIP-1258 block above: this MUST run
	// synchronously here, BEFORE ConsumeMessage is started below, because the
	// bind/unbind RPCs share the queue's AMQP channel with basic.consume.
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
