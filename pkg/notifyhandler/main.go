package notifyhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package notifyhandler -destination ./mock_notifyhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"
)

// WebhookMessage defines
type WebhookMessage interface {
	CreateWebhookEvent() ([]byte, error)
}

// EventType string
type EventType string

// list of event types
const (
	// call
	EventTypeCallCreated  EventType = "call_created"
	EventTypeCallUpdated  EventType = "call_updated"
	EventTypeCallRinging  EventType = "call_ringing"
	EventTypeCallAnswered EventType = "call_answered"
	EventTypeCallHungup   EventType = "call_hungup"

	// confbridge
	EventTypeConfbridgeCreated EventType = "confbridge_created"
	EventTypeConfbridgeDeleted EventType = "confbridge_deleted"
	EventTypeConfbridgeJoined  EventType = "confbridge_joined"
	EventTypeConfbridgeLeaved  EventType = "confbridge_leaved"

	// conference
	EventTypeConferenceCreated EventType = "conference_created"
	EventTypeConferenceDeleted EventType = "conference_deleted"
	EventTypeConferenceJoined  EventType = "conference_joined"
	EventTypeConferenceLeaved  EventType = "conference_leaved"

	// recording
	EventTypeRecordingStarted  EventType = "recording_started"
	EventTypeRecordingFinished EventType = "recording_finished"
)

// Data types
var (
	dataTypeJSON = "application/json"
)

const requestTimeoutDefault int = 3 // default request timeout

// delay units
const (
	DelayNow    int = 0
	DelaySecond int = 1000
	DelayMinute int = DelaySecond * 60
	DelayHour   int = DelayMinute * 60
)

// list of prometheus metrics
var (
	promNotifyProcessTime *prometheus.HistogramVec
	promNotifyTotal       *prometheus.CounterVec
)

func initPrometheus(namespace string) {

	promNotifyProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "notify_process_time",
			Help:      "Process time of send notification",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"type"},
	)

	promNotifyTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "notify_total",
			Help:      "Total number of sent notification.",
		},
		[]string{"type"},
	)

	prometheus.MustRegister(
		promNotifyProcessTime,
		promNotifyTotal,
	)
}

// NotifyHandler intreface
type NotifyHandler interface {
	PublishEvent(ctx context.Context, t string, c interface{})
	PublishWebhook(ctx context.Context, t string, webhookURI string, c WebhookMessage)
	PublishWebhookEvent(ctx context.Context, eventType string, webhookURI string, message WebhookMessage)
}

type notifyHandler struct {
	sock       rabbitmqhandler.Rabbit
	reqHandler requesthandler.RequestHandler

	exchangeDelay  string
	exchangeNotify string

	publisher string
}

// NewNotifyHandler create NotifyHandler
func NewNotifyHandler(sock rabbitmqhandler.Rabbit, reqHandler requesthandler.RequestHandler, exchangeDelay, exchangeEvent, publisher string) NotifyHandler {
	h := &notifyHandler{
		sock:       sock,
		reqHandler: reqHandler,

		exchangeDelay:  exchangeDelay,
		exchangeNotify: exchangeEvent,

		publisher: publisher,
	}

	if err := sock.ExchangeDeclare(exchangeEvent, "fanout", true, false, false, false, nil); err != nil {
		logrus.Errorf("Could not declare the event exchange. err: %v", err)
		return nil
	}

	namespace := strings.ReplaceAll(publisher, "-", "_")
	initPrometheus(namespace)

	return h
}
