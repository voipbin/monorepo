package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"
	cmrecording "monorepo/bin-call-manager/models/recording"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cscustomer "monorepo/bin-customer-manager/models/customer"
	ememail "monorepo/bin-email-manager/models/email"
	mmmessage "monorepo/bin-message-manager/models/message"

	nmnumber "monorepo/bin-number-manager/models/number"
	tmspeaking "monorepo/bin-tts-manager/models/speaking"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/billinghandler"
	"monorepo/bin-billing-manager/pkg/failedeventhandler"
)

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

// topicPatterns is this service's bind set on the global topic exchange
// `bin-manager.event` (VOIP-1406): exactly one PatternAction per (publisher, event
// type) pair handled in processEvent below. The exact pattern strings are pinned by
// binding_golden_test.go.
var topicPatterns = []string{
	eventtopic.PatternAction(string(commonoutline.ServiceNameCallManager), "call", "progressing"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameCallManager), "call", "hangup"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameCallManager), "recording", "started"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameCallManager), "recording", "finished"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameMessageManager), "message", "created"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameEmailManager), "email", "created"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameCustomerManager), "customer", "deleted"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameCustomerManager), "customer", "created"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameCustomerManager), "customer", "frozen"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameCustomerManager), "customer", "recovered"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameNumberManager), "number", "created"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameNumberManager), "number", "renewed"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameTTSManager), "speaking", "started"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameTTSManager), "speaking", "stopped"),
}

// fanoutUnbindTargets lists the per-service fanout event exchanges this queue
// unbinds after ALL topicPatterns are bound (VOIP-1406). It equals the cmd wiring's
// subscribeTargets (billing has no asterisk leg). The fanout QueueSubscribe calls in
// Run() stay in place until VOIP-1407 as the rollback/degrade surface.
var fanoutUnbindTargets = []string{
	string(commonoutline.QueueNameCallEvent),
	string(commonoutline.QueueNameMessageEvent),
	string(commonoutline.QueueNameEmailEvent),
	string(commonoutline.QueueNameCustomerEvent),
	string(commonoutline.QueueNameNumberEvent),
	string(commonoutline.QueueNameTTSEvent),
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler

	subscribeQueue   string
	subscribeTargets []string

	accountHandler      accounthandler.AccountHandler
	billingHandler      billinghandler.BillingHandler
	failedEventHandler  failedeventhandler.FailedEventHandler
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
	sockHandler sockhandler.SockHandler,
	subscribeQueue string,
	subscribeTargets []string,
	accountHandler accounthandler.AccountHandler,
	billingHandler billinghandler.BillingHandler,
	failedEventHandler failedeventhandler.FailedEventHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		sockHandler:      sockHandler,
		subscribeQueue:   subscribeQueue,
		subscribeTargets: subscribeTargets,

		accountHandler:     accountHandler,
		billingHandler:     billingHandler,
		failedEventHandler: failedEventHandler,
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

	// Migrate this queue onto the global topic exchange `bin-manager.event`
	// (VOIP-1406): declare the exchange (idempotent, both-sides-declare), bind every
	// dispatch pattern, and only after ALL patterns are bound unbind the old
	// per-service fanout exchanges. Bind-new-before-unbind-old: there is never a
	// window bound to neither. Any bind failure rolls back the partial topic binds
	// (best-effort) and leaves the service fully on fanout. The fanout
	// QueueSubscribe calls above stay until VOIP-1407 as the rollback surface.
	//
	// CRITICAL: this MUST run synchronously here, BEFORE the ConsumeMessage
	// goroutine below. QueueBind/QueueUnbind and ConsumeMessage's internal
	// channel.Consume() share the same underlying AMQP channel for this queue;
	// racing them makes the broker close the channel with a 503 "unexpected command
	// received" (VOIP-1258 PR #1101 post-deploy incident, 2026-07-14).
	if errDeclare := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic"); errDeclare != nil {
		log.Errorf("Could not declare the global topic exchange. Staying fully on fanout. exchange: %s, err: %v", string(commonoutline.QueueNameEvent), errDeclare)
	} else {
		bound := []string{}
		ok := true
		for _, pattern := range topicPatterns {
			if errBind := h.sockHandler.QueueBind(h.subscribeQueue, pattern, string(commonoutline.QueueNameEvent), false, nil); errBind != nil {
				log.Errorf("Could not bind the topic pattern. Staying fully on fanout. pattern: %s, err: %v", pattern, errBind)
				ok = false
				break
			}
			bound = append(bound, pattern)
		}

		if !ok {
			// best-effort rollback of the partial topic binds, then stay fully on
			// fanout. NO fanout exchange is unbound on this path.
			for _, pattern := range bound {
				if errUnbind := h.sockHandler.QueueUnbind(h.subscribeQueue, pattern, string(commonoutline.QueueNameEvent), nil); errUnbind != nil {
					log.Errorf("CRITICAL: partial topic bind could not be rolled back. The queue keeps a stale topic binding (partial double delivery). Manual intervention required. pattern: %s, err: %v", pattern, errUnbind)
				}
			}
		} else {
			// all patterns bound -- unbind every old fanout exchange. An unbind
			// failure is CRITICAL but not fatal: double delivery beats loss.
			for _, target := range fanoutUnbindTargets {
				if errUnbind := h.sockHandler.QueueUnbind(h.subscribeQueue, "", target, nil); errUnbind != nil {
					log.Errorf("CRITICAL: still bound to BOTH exchanges (double delivery). Manual intervention required. exchange: %s, err: %v", target, errUnbind)
				}
			}
		}
	}

	// receive subscribe events
	go func() {
		if errConsume := h.sockHandler.ConsumeMessage(context.Background(), h.subscribeQueue, string(commonoutline.ServiceNameBillingManager), false, false, false, 10, h.processEventRun); errConsume != nil {
			log.Errorf("Could not consume the request message correctly. err: %v", errConsume)
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
		log.Errorf("Could not consume the subscribed event message correctly. Persisting for retry. err: %v", errProcess)
		if errSave := h.failedEventHandler.Save(context.Background(), m, errProcess); errSave != nil {
			log.Errorf("CRITICAL: Could not save failed event. Data loss possible. Returning error to NACK message. err: %v", errSave)
			return errSave
		}
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

	//// email-manager
	// email
	case m.Publisher == string(commonoutline.ServiceNameEmailManager) && m.Type == ememail.EventTypeCreated:
		err = h.processEventEMEmailCreated(ctx, m)

	//// customer-manager
	// customer
	case m.Publisher == string(commonoutline.ServiceNameCustomerManager) && m.Type == cscustomer.EventTypeCustomerDeleted:
		err = h.processEventCMCustomerDeleted(ctx, m)

	case m.Publisher == string(commonoutline.ServiceNameCustomerManager) && m.Type == cscustomer.EventTypeCustomerCreated:
		err = h.processEventCMCustomerCreated(ctx, m)

	case m.Publisher == string(commonoutline.ServiceNameCustomerManager) && m.Type == cscustomer.EventTypeCustomerFrozen:
		err = h.processEventCUCustomerFrozen(ctx, m)

	case m.Publisher == string(commonoutline.ServiceNameCustomerManager) && m.Type == cscustomer.EventTypeCustomerRecovered:
		err = h.processEventCUCustomerRecovered(ctx, m)

	//// number-manager
	// number
	case m.Publisher == string(commonoutline.ServiceNameNumberManager) && m.Type == nmnumber.EventTypeNumberCreated:
		err = h.processEventNMNumberCreated(ctx, m)

	case m.Publisher == string(commonoutline.ServiceNameNumberManager) && m.Type == nmnumber.EventTypeNumberRenewed:
		err = h.processEventNMNumberRenewed(ctx, m)

	//// tts-manager
	// speaking
	case m.Publisher == string(commonoutline.ServiceNameTTSManager) && m.Type == tmspeaking.EventTypeSpeakingStarted:
		err = h.processEventTTSSpeakingStarted(ctx, m)

	case m.Publisher == string(commonoutline.ServiceNameTTSManager) && m.Type == tmspeaking.EventTypeSpeakingStopped:
		err = h.processEventTTSSpeakingStopped(ctx, m)

	//// call-manager
	// recording
	case m.Publisher == string(commonoutline.ServiceNameCallManager) && m.Type == cmrecording.EventTypeRecordingStarted:
		err = h.processEventCMRecordingStarted(ctx, m)

	case m.Publisher == string(commonoutline.ServiceNameCallManager) && m.Type == cmrecording.EventTypeRecordingFinished:
		err = h.processEventCMRecordingFinished(ctx, m)

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
		return err
	}

	return nil
}

// GetEventProcessor returns the processEvent function as a callback for the failed event handler retry loop.
func GetEventProcessor(sh SubscribeHandler) failedeventhandler.EventProcessor {
	h := sh.(*subscribeHandler)
	return h.processEvent
}

// SetFailedEventHandler sets the failed event handler on the subscribe handler.
// This is used to break the circular dependency during initialization.
func SetFailedEventHandler(sh SubscribeHandler, feh failedeventhandler.FailedEventHandler) {
	h := sh.(*subscribeHandler)
	h.failedEventHandler = feh
}
