package notifyhandler

//go:generate mockgen -package notifyhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

// WebhookMessage defines
type WebhookMessage interface {
	CreateWebhookEvent() ([]byte, error)
}

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
	PublishEvent(ctx context.Context, eventType string, data interface{})
	PublishEventRaw(ctx context.Context, eventType string, dataType string, data []byte)

	PublishWebhook(ctx context.Context, customerID uuid.UUID, eventType string, data WebhookMessage)
	PublishWebhookEvent(ctx context.Context, customerID uuid.UUID, eventType string, data WebhookMessage)
}

type notifyHandler struct {
	sockHandler sockhandler.SockHandler
	reqHandler  requesthandler.RequestHandler

	queueNotify commonoutline.QueueName

	publisher commonoutline.ServiceName
}

// NewNotifyHandler create NotifyHandler
// queueEvent: queue name for notification. the notify handler will publish the event to this queue name.
// publisher: publisher service name. the notify handler will publish the event with this publisher service name.
func NewNotifyHandler(sockHandler sockhandler.SockHandler, reqHandler requesthandler.RequestHandler, queueEvent commonoutline.QueueName, publisher commonoutline.ServiceName) NotifyHandler {
	h := &notifyHandler{
		sockHandler: sockHandler,
		reqHandler:  reqHandler,

		queueNotify: queueEvent,

		publisher: publisher,
	}

	if err := sockHandler.TopicCreate(string(queueEvent)); err != nil {
		logrus.Errorf("Could not declare the event exchange. err: %v", err)
		return nil
	}

	namespace := commonoutline.GetMetricNameSpace(publisher)
	initPrometheus(namespace)

	return h
}
