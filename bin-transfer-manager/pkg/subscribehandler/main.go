package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transfer-manager/pkg/transferhandler"
)

// list of publishers
const (
	publisherCallManager = "call-manager"
)

// topicPatterns is the ruled bind set on the global topic exchange `bin-manager.event`
// (VOIP-1406, design §5): one pattern per dispatched (publisher, event-type) pair.
// Pinned by the binding golden test.
var topicPatterns = []string{
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCallManager), cmgroupcall.EventTypeGroupcallProgressing),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCallManager), cmgroupcall.EventTypeGroupcallHangup),
	eventtopic.PatternForEventType(string(commonoutline.ServiceNameCallManager), cmcall.EventTypeCallHangup),
}

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	serviceName string

	sockHandler sockhandler.SockHandler

	subscribeQueue string

	transferHandler transferhandler.TransferHandler
}

var (
	metricsNamespace = "conference_manager"

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
	serviceName string,
	sockHandler sockhandler.SockHandler,
	subscribeQueue string,
	transferHandler transferhandler.TransferHandler,
) SubscribeHandler {

	h := &subscribeHandler{
		serviceName:    serviceName,
		sockHandler:    sockHandler,
		subscribeQueue: subscribeQueue,

		transferHandler: transferHandler,
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

	// VOIP-1406: bind the subscribe queue to the global topic exchange `bin-manager.event`
	// with one pattern per dispatched (publisher, event-type) pair. This block MUST run
	// synchronously here, BEFORE the ConsumeMessage goroutine below -- QueueBind and
	// ConsumeMessage's internal basic.consume share the same underlying AMQP channel for a
	// given queue, and racing them makes the broker close the channel with a 503 (production
	// incident 2026-07-14, VOIP-1258).
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
		if errConsume := h.sockHandler.ConsumeMessage(context.Background(), h.subscribeQueue, h.serviceName, false, false, false, 10, h.processEventRun); errConsume != nil {
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

	var err error
	start := time.Now()
	ctx := context.Background()
	switch {

	//// call-manager
	// groupcall
	case m.Publisher == publisherCallManager && m.Type == string(cmgroupcall.EventTypeGroupcallProgressing):
		err = h.processEventCMGroupcallProgressing(ctx, m)

	case m.Publisher == publisherCallManager && m.Type == string(cmgroupcall.EventTypeGroupcallHangup):
		err = h.processEventCMGroupcallHangup(ctx, m)

	// call
	case m.Publisher == publisherCallManager && m.Type == string(cmcall.EventTypeCallHangup):
		err = h.processEventCMCallHangup(ctx, m)

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		// no event handler found
		return
	}
	elapsed := time.Since(start)
	promEventProcessTime.WithLabelValues(m.Publisher, string(m.Type)).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not process the event correctly. publisher: %s, type: %s, err: %v", m.Publisher, m.Type, err)
	}
}
