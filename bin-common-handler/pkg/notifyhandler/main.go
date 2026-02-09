package notifyhandler

//go:generate mockgen -package notifyhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
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

// clickhouse retry interval
const clickhouseRetryInterval = 30 * time.Second

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

	clickhouseAddress string
	chClient          atomic.Value // stores clickhouse.Conn; Load() returns nil until connected
}

// NewNotifyHandler create NotifyHandler
// queueEvent: queue name for notification. the notify handler will publish the event to this queue name.
// publisher: publisher service name. the notify handler will publish the event with this publisher service name.
// clickhouseAddress: clickhouse address for event logging. if empty, clickhouse publishing is disabled.
func NewNotifyHandler(sockHandler sockhandler.SockHandler, reqHandler requesthandler.RequestHandler, queueEvent commonoutline.QueueName, publisher commonoutline.ServiceName, clickhouseAddress string) NotifyHandler {
	h := &notifyHandler{
		sockHandler: sockHandler,
		reqHandler:  reqHandler,

		queueNotify: queueEvent,

		publisher: publisher,

		clickhouseAddress: clickhouseAddress,
	}

	if err := sockHandler.TopicCreate(string(queueEvent)); err != nil {
		logrus.Errorf("Could not declare the event exchange. err: %v", err)
		return nil
	}

	// Start ClickHouse connection loop if address is provided
	if clickhouseAddress != "" {
		go h.clickhouseConnectionLoop()
	}

	namespace := commonoutline.GetMetricNameSpace(publisher)
	initPrometheus(namespace)

	return h
}

// newClickHouseClient creates a new ClickHouse client connection
func newClickHouseClient(address string) (clickhouse.Conn, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{address},
		Auth: clickhouse.Auth{
			Database: "default",
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	// Ping to verify connection
	if err := conn.Ping(context.Background()); err != nil {
		return nil, err
	}

	return conn, nil
}

// clickhouseConnectionLoop maintains the ClickHouse connection with retry logic
func (h *notifyHandler) clickhouseConnectionLoop() {
	log := logrus.WithFields(logrus.Fields{
		"func":    "clickhouseConnectionLoop",
		"address": h.clickhouseAddress,
	})

	for {
		// Skip if already connected
		if h.chClient.Load() != nil {
			time.Sleep(clickhouseRetryInterval)
			continue
		}

		client, err := newClickHouseClient(h.clickhouseAddress)
		if err != nil {
			log.Errorf("Could not connect to ClickHouse, retrying in %v. err: %v",
				clickhouseRetryInterval, err)
			time.Sleep(clickhouseRetryInterval)
			continue
		}

		log.Info("Successfully connected to ClickHouse")
		h.chClient.Store(client)
		time.Sleep(clickhouseRetryInterval)
	}
}
