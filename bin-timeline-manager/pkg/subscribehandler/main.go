package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-timeline-manager/pkg/dbhandler"
)

// subscribeTargets lists all service event exchanges to subscribe to.
var subscribeTargets = []commonoutline.QueueName{
	commonoutline.QueueNameAIEvent,
	commonoutline.QueueNameAgentEvent,
	commonoutline.QueueNameAsteriskEventAll,
	commonoutline.QueueNameBillingEvent,
	commonoutline.QueueNameCallEvent,
	commonoutline.QueueNameCampaignEvent,
	commonoutline.QueueNameConferenceEvent,
	commonoutline.QueueNameContactEvent,
	commonoutline.QueueNameConversationEvent,
	commonoutline.QueueNameCustomerEvent,
	commonoutline.QueueNameEmailEvent,
	commonoutline.QueueNameFlowEvent,
	commonoutline.QueueNameMessageEvent,
	commonoutline.QueueNameNumberEvent,
	commonoutline.QueueNameOutdialEvent,
	commonoutline.QueueNamePipecatEvent,
	commonoutline.QueueNameQueueEvent,
	commonoutline.QueueNameRegistrarEvent,
	commonoutline.QueueNameRouteEvent,
	commonoutline.QueueNameSentinelEvent,
	commonoutline.QueueNameStorageEvent,
	commonoutline.QueueNameTagEvent,
	commonoutline.QueueNameTalkEvent,
	commonoutline.QueueNameTimelineEvent,
	commonoutline.QueueNameTranscribeEvent,
	commonoutline.QueueNameTransferEvent,
	commonoutline.QueueNameTTSEvent,
	commonoutline.QueueNameWebhookEvent,
}

var (
	metricsNamespace = "timeline_manager"

	promSubscribeEventProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "subscribe_event_process_time",
			Help:      "Process time of received subscribe event for ClickHouse insert",
			Buckets:   []float64{50, 100, 500, 1000, 3000},
		},
		[]string{"publisher", "type"},
	)
)

func init() {
	prometheus.MustRegister(promSubscribeEventProcessTime)
}

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler
	dbHandler   dbhandler.DBHandler
}

// NewSubscribeHandler creates a new SubscribeHandler.
func NewSubscribeHandler(
	sockHandler sockhandler.SockHandler,
	dbHandler dbhandler.DBHandler,
) SubscribeHandler {
	return &subscribeHandler{
		sockHandler: sockHandler,
		dbHandler:   dbHandler,
	}
}

// Run creates the subscribe queue, binds to all event exchanges, and starts consuming.
func (h *subscribeHandler) Run() error {
	log := logrus.WithField("func", "Run")
	log.Info("Creating rabbitmq queue for event subscription.")

	subscribeQueue := string(commonoutline.QueueNameTimelineSubscribe)

	// Create durable queue
	if err := h.sockHandler.QueueCreate(subscribeQueue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for subscribeHandler. err: %v", err)
	}

	// Subscribe to all service event exchanges
	for _, target := range subscribeTargets {
		if errSubscribe := h.sockHandler.QueueSubscribe(subscribeQueue, string(target)); errSubscribe != nil {
			log.Errorf("Could not subscribe to target. target: %s, err: %v", target, errSubscribe)
			return errSubscribe
		}
		log.Debugf("Subscribed to event exchange. target: %s", target)
	}

	// Start consuming events
	go func() {
		if errConsume := h.sockHandler.ConsumeMessage(context.Background(), subscribeQueue, "timeline-manager", false, false, false, 10, h.processEventRun); errConsume != nil {
			log.Errorf("Could not consume subscribe events. err: %v", errConsume)
		}
	}()

	log.Infof("Subscribe handler started. subscribed to %d event exchanges.", len(subscribeTargets))
	return nil
}

// processEventRun dispatches event processing in a goroutine.
func (h *subscribeHandler) processEventRun(m *sock.Event) error {
	go h.processEvent(m)
	return nil
}

// processEvent inserts the received event into ClickHouse.
func (h *subscribeHandler) processEvent(m *sock.Event) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "processEvent",
		"publisher":  m.Publisher,
		"event_type": m.Type,
	})

	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.dbHandler.EventInsert(ctx, time.Now(), m.Type, m.Publisher, m.DataType, string(m.Data)); err != nil {
		log.Errorf("Could not insert event into ClickHouse. err: %v", err)
		return
	}

	elapsed := time.Since(start)
	promSubscribeEventProcessTime.WithLabelValues(m.Publisher, m.Type).Observe(float64(elapsed.Milliseconds()))
}
