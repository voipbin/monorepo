package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcasenote "monorepo/bin-contact-manager/models/casenote"
	cmkase "monorepo/bin-contact-manager/models/kase"
	cvmessage "monorepo/bin-conversation-manager/models/message"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	tkmessage "monorepo/bin-talk-manager/models/message"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-push-manager/pkg/devicehandler"
	"monorepo/bin-push-manager/pkg/eventhandler"
	"monorepo/bin-push-manager/pkg/settingshandler"
)

// list of publishers
const (
	publisherAgentManager        = string(commonoutline.ServiceNameAgentManager)
	publisherCustomerManager     = string(commonoutline.ServiceNameCustomerManager)
	publisherTalkManager         = string(commonoutline.ServiceNameTalkManager)
	publisherConversationManager = string(commonoutline.ServiceNameConversationManager)
	publisherContactManager      = string(commonoutline.ServiceNameContactManager)
)

// maxConcurrentEvents is the number of events that are processed at the same time. When all slots are taken the
// consumer callback waits, so the broker does not hand over more events (together with the prefetch of the
// consumer): the events wait in the queue and not in unbounded goroutines. It is a first value without a
// measurement behind it.
const maxConcurrentEvents = 32

// topicPatterns is the bind set on the global topic exchange `bin-manager.event`: one pattern per
// dispatched (publisher, event-type) pair. Pinned by the binding golden test.
var topicPatterns = []string{
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameAgentManager), amagent.EventTypeAgentDeleted),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCustomerManager), cmcustomer.EventTypeCustomerDeleted),
	eventtopic.PatternForEventType(publisherTalkManager, tkmessage.EventTypeMessageCreated),
	eventtopic.PatternForEventType(publisherConversationManager, cvmessage.EventTypeMessageCreated),
	eventtopic.PatternForEventType(publisherContactManager, cmcasenote.EventTypeCaseNoteCreated),
	eventtopic.PatternForEventType(publisherContactManager, cmkase.EventTypeCaseAssigned),
}

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler

	subscribeQueue string

	deviceHandler   devicehandler.DeviceHandler
	settingsHandler settingshandler.SettingsHandler
	eventHandler    eventhandler.EventHandler

	// sem holds the slots of the events that are being processed.
	sem chan struct{}
}

var (
	metricsNamespace = "push_manager"

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
	deviceHandler devicehandler.DeviceHandler,
	settingsHandler settingshandler.SettingsHandler,
	eventHandler eventhandler.EventHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		sockHandler:     sockHandler,
		subscribeQueue:  subscribeQueue,
		deviceHandler:   deviceHandler,
		settingsHandler: settingsHandler,
		eventHandler:    eventHandler,
		sem:             make(chan struct{}, maxConcurrentEvents),
	}

	return h
}

func (h *subscribeHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})
	log.Info("Creating rabbitmq queue for listen.")

	// declare the queue for subscribe
	if err := h.sockHandler.QueueCreate(h.subscribeQueue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for subscribeHandler. err: %w", err)
	}

	// bind the subscribe queue to the global topic exchange `bin-manager.event` with one pattern per
	// dispatched (publisher, event-type) pair. This block MUST run synchronously here, BEFORE the
	// ConsumeMessage goroutine below: QueueBind and ConsumeMessage's internal basic.consume share the same
	// underlying AMQP channel for a given queue, and racing them makes the broker close the channel.
	if err := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic"); err != nil {
		return fmt.Errorf("could not declare the global topic exchange. err: %w", err)
	}

	for _, pattern := range topicPatterns {
		if err := h.sockHandler.QueueBind(h.subscribeQueue, pattern, string(commonoutline.QueueNameEvent), false, nil); err != nil {
			return fmt.Errorf("could not bind the topic pattern. pattern: %s, err: %w", pattern, err)
		}
	}

	// receive subscribe events
	go func() {
		if errConsume := h.sockHandler.ConsumeMessage(context.Background(), h.subscribeQueue, string(commonoutline.ServiceNamePushManager), false, false, false, 10, h.processEventRun); errConsume != nil {
			log.Errorf("Could not consume the request message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

// processEventRun runs the processEvent in a goroutine once a slot of the pool is free. It waits for the slot,
// which holds back the consumer, and always returns nil: an error would make the broker deliver the event again,
// and a notification that was sent would be sent twice. The slot is given back when the event is over, also
// after a panic, which is logged.
func (h *subscribeHandler) processEventRun(m *sock.Event) error {
	h.sem <- struct{}{}

	go func() {
		defer func() { <-h.sem }()
		defer func() {
			if r := recover(); r != nil {
				logrus.WithFields(logrus.Fields{
					"func":      "processEventRun",
					"publisher": m.Publisher,
					"type":      m.Type,
				}).Errorf("The event handler panicked. recovered: %v", fmt.Sprint(r))
			}
		}()

		h.processEvent(m)
	}()

	return nil
}

// processEvent processes the event message
func (h *subscribeHandler) processEvent(m *sock.Event) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "processEvent",
		"publisher": m.Publisher,
		"type":      m.Type,
	})
	ctx := context.Background()

	var err error
	start := time.Now()
	switch {

	// agent-manager
	case m.Publisher == publisherAgentManager && m.Type == amagent.EventTypeAgentDeleted:
		err = h.processEventAMAgentDeleted(ctx, m)

	// customer-manager
	case m.Publisher == publisherCustomerManager && m.Type == cmcustomer.EventTypeCustomerDeleted:
		err = h.processEventCMCustomerDeleted(ctx, m)

	// talk-manager
	case m.Publisher == publisherTalkManager && m.Type == tkmessage.EventTypeMessageCreated:
		err = h.processEventTKMessageCreated(ctx, m)

	// conversation-manager
	case m.Publisher == publisherConversationManager && m.Type == cvmessage.EventTypeMessageCreated:
		err = h.processEventCVMessageCreated(ctx, m)

	// contact-manager
	case m.Publisher == publisherContactManager && m.Type == cmcasenote.EventTypeCaseNoteCreated:
		err = h.processEventCMCaseNoteCreated(ctx, m)

	case m.Publisher == publisherContactManager && m.Type == cmkase.EventTypeCaseAssigned:
		err = h.processEventCMCaseAssigned(ctx, m)

	default:
		// ignore the event.
		return
	}
	elapsed := time.Since(start)
	promEventProcessTime.WithLabelValues(m.Publisher, m.Type).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not process the event correctly. err: %v", err)
	}
}
