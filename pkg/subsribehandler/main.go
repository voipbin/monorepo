package subscribehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package subscribehandler -destination ./mock_subscribehandler_subscribehandler.go -source main.go -build_flags=-mod=mod

import (
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/webhookhandler"
)

// list of publishers
const (
	publisherAPIManager       = "api-manager"
	publisherCallManager      = "call-manager"
	publisherFlowManager      = "flow-manager"
	publisherNumberManager    = "number-manager"
	publisherRegistrarManager = "registrar-manager"
	publisherStorageManager   = "storage-manager"
	publisherTTSManager       = "tts-manager"
)

// list of call-manager event type
const (
	cmCallCreate = "call_created"
	cmCallUpdate = "call_updated"
	cmCallHangup = "call_hungup"

	cmRecordingStarted  = "recording_started"
	cmRecordingFinished = "recording_finished"
)

// SubscribeHandler interface
type SubscribeHandler interface {
	Run(queue, exchangeDelay string) error
}

type subscribeHandler struct {
	rabbitSock rabbitmqhandler.Rabbit
	db         dbhandler.DBHandler
	cache      cachehandler.CacheHandler

	subscribeQueue    string
	subscribesTargets string

	webhookHandler webhookhandler.WebhookHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"
	regAny  = "(.*)"

	// v1
)

var (
	metricsNamespace = "webhook_manager"

	promEventProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "receive_request_process_time",
			Help:      "Process time of received request",
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

// simpleResponse returns simple rabbitmq response
func simpleResponse(code int) *rabbitmqhandler.Response {
	return &rabbitmqhandler.Response{
		StatusCode: code,
	}
}

// NewSubscribeHandler return SubscribeHandler interface
func NewSubscribeHandler(
	rabbitSock rabbitmqhandler.Rabbit,
	db dbhandler.DBHandler,
	cache cachehandler.CacheHandler,
	subscribeQueue string,
	subscribeTargets string,
) SubscribeHandler {
	webhookHandler := webhookhandler.NewWebhookHandler(db, cache)

	h := &subscribeHandler{
		rabbitSock: rabbitSock,
		db:         db,
		cache:      cache,

		subscribeQueue:    subscribeQueue,
		subscribesTargets: subscribeTargets,
		webhookHandler:    webhookHandler,
	}

	return h
}

func (h *subscribeHandler) Run(subscribeQueue, subscribeTargets string) error {
	logrus.WithFields(logrus.Fields{
		"subscribe_queue":   subscribeQueue,
		"subscribe_targets": subscribeTargets,
	}).Info("Creating rabbitmq queue for listen.")

	// declare the queue for subscribe
	if err := h.rabbitSock.QueueDeclare(subscribeQueue, true, true, false, false); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// subscribe each targets
	targets := strings.Split(subscribeTargets, ",")
	for _, target := range targets {

		// bind each targets
		if err := h.rabbitSock.QueueBind(subscribeQueue, "", target, false, nil); err != nil {
			logrus.Errorf("Could not subscribe the target. target: %s, err: %v", target, err)
			return err
		}
	}

	// receive subscribe events
	go func() {
		for {
			err := h.rabbitSock.ConsumeMessageOpt(subscribeQueue, "webhook-manager", false, false, false, h.processEventRun)
			if err != nil {
				logrus.Errorf("Could not consume the request message correctly. err: %v", err)
			}
		}
	}()

	return nil
}

// processEventRun runs the processEvent
func (h *subscribeHandler) processEventRun(m *rabbitmqhandler.Event) error {
	go h.processEvent(m)

	return nil
}

// processEvent processes the event message
func (h *subscribeHandler) processEvent(m *rabbitmqhandler.Event) {

	log := logrus.WithFields(
		logrus.Fields{
			"message": m,
		},
	)
	log.Debugf("Received subscribed event. publisher: %s, type: %s", m.Publisher, m.Type)

	var err error
	start := time.Now()
	switch {

	//// call-manager
	// call
	case m.Publisher == publisherCallManager && (m.Type == cmCallCreate || m.Type == cmCallUpdate || m.Type == cmCallHangup):
		err = h.processEventCMCallCommon(m)

	// recording
	case m.Publisher == publisherCallManager && (m.Type == cmRecordingStarted || m.Type == cmRecordingFinished):
		err = h.processEventCMRecordingCommon(m)

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		err = fmt.Errorf("could not find an event handler")
	}
	elapsed := time.Since(start)
	promEventProcessTime.WithLabelValues(m.Publisher, string(m.Type)).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not process the event correctly. publisher: %s, type: %s, err: %v", m.Publisher, m.Type, err)
		return
	}

	return
}
