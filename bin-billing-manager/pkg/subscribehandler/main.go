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
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCallManager), cmcall.EventTypeCallProgressing),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCallManager), cmcall.EventTypeCallHangup),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCallManager), cmrecording.EventTypeRecordingStarted),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCallManager), cmrecording.EventTypeRecordingFinished),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameMessageManager), mmmessage.EventTypeMessageCreated),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameEmailManager), ememail.EventTypeCreated),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCustomerManager), cscustomer.EventTypeCustomerDeleted),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCustomerManager), cscustomer.EventTypeCustomerCreated),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCustomerManager), cscustomer.EventTypeCustomerFrozen),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCustomerManager), cscustomer.EventTypeCustomerRecovered),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameNumberManager), nmnumber.EventTypeNumberCreated),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameNumberManager), nmnumber.EventTypeNumberRenewed),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameTTSManager), tmspeaking.EventTypeSpeakingStarted),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameTTSManager), tmspeaking.EventTypeSpeakingStopped),
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler

	subscribeQueue string

	accountHandler     accounthandler.AccountHandler
	billingHandler     billinghandler.BillingHandler
	failedEventHandler failedeventhandler.FailedEventHandler
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
	accountHandler accounthandler.AccountHandler,
	billingHandler billinghandler.BillingHandler,
	failedEventHandler failedeventhandler.FailedEventHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		sockHandler:    sockHandler,
		subscribeQueue: subscribeQueue,

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
