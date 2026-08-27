package notifyhandler

//go:generate mockgen -package notifyhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"sync"

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

// global topic exchange (VOIP-1404)
const (
	// topicExchangeKind is the AMQP exchange kind of the global event exchange.
	topicExchangeKind = "topic"

	// result label values of promTopicPublishTotal
	topicPublishResultOK    = "ok"
	topicPublishResultError = "error"
)

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

	// global topic exchange metrics (VOIP-1404). Deliberately separate from the counters above:
	// the topic publish path must never touch promNotifyTotal/promNotifyProcessTime, otherwise
	// the dual publish would double-count the existing fanout metrics.
	promTopicPublishTotal     *prometheus.CounterVec
	promTopicPlaceholderTotal *prometheus.CounterVec

	// initPrometheusMu/initPrometheusDone guard against duplicate MustRegister panics when
	// initPrometheus is called more than once for the same namespace -- e.g. VOIP-1258's
	// second NotifyHandler instance (NewNotifyHandlerForExistingExchange, bound to the new
	// topic exchange) is constructed for the SAME publisher/namespace as the existing
	// fanout-bound NewNotifyHandler call in the same process (webhook-manager, webhook-control).
	// Without this guard, the second initPrometheus call would panic with "duplicate metrics
	// collector registration attempted" on every startup.
	initPrometheusMu   sync.Mutex
	initPrometheusDone = map[string]bool{}
)

func initPrometheus(namespace string) {
	initPrometheusMu.Lock()
	defer initPrometheusMu.Unlock()

	if initPrometheusDone[namespace] {
		return
	}
	initPrometheusDone[namespace] = true

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

	// VOIP-1404: registered unconditionally, regardless of the WithGlobalTopicPublish option.
	// Registering them here -- inside the existing initPrometheusDone guard -- avoids both a
	// permanent non-registration and a duplicate-registration panic, and guarantees the counters
	// are non-nil before any publish can use them.
	promTopicPublishTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "topic_publish_total",
			Help:      "Total number of published events to the global topic exchange.",
		},
		[]string{"type", "result"},
	)

	promTopicPlaceholderTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "topic_placeholder_total",
			Help:      "Total number of published events to the global topic exchange with a placeholder subscription id.",
		},
		[]string{"type"},
	)

	prometheus.MustRegister(
		promNotifyProcessTime,
		promNotifyTotal,
		promTopicPublishTotal,
		promTopicPlaceholderTotal,
	)
}

// NotifyHandler intreface
type NotifyHandler interface {
	PublishEvent(ctx context.Context, eventType string, data interface{})
	PublishEventRaw(ctx context.Context, eventType string, dataType string, data []byte)
	PublishEventWithRoutingKey(ctx context.Context, eventType string, routingKey string, data interface{})

	PublishWebhook(ctx context.Context, customerID uuid.UUID, eventType string, data WebhookMessage)
	PublishWebhookEvent(ctx context.Context, customerID uuid.UUID, eventType string, data WebhookMessage)
}

type notifyHandler struct {
	sockHandler sockhandler.SockHandler
	reqHandler  requesthandler.RequestHandler

	queueNotify commonoutline.QueueName

	publisher commonoutline.ServiceName

	// topicEnabled reports whether this handler also publishes every event to the global topic
	// exchange (VOIP-1404). Opt-in per handler instance via WithGlobalTopicPublish.
	topicEnabled bool

	// topicDisabled reports that the global topic exchange declare failed at construction time,
	// so the topic publish is suppressed for the lifetime of this handler. Per-instance state,
	// never a package global: a process may construct several handlers (webhook-manager does) and
	// one instance's degradation must not silence the others. Set in the constructor only, which
	// keeps it race-free against PublishWebhookEvent's goroutine fan-out.
	topicDisabled bool
}

// Option customizes a NotifyHandler at construction time. Both constructors accept a variadic
// list of them, which keeps every existing call site source-compatible (VOIP-1404 design §5.1).
type Option func(*notifyHandler)

// WithGlobalTopicPublish enables the dual publish: on top of the existing fanout publish, every
// event is also published to the global topic exchange `bin-manager.event` with a
// `<publisher>.<resource>.<subscription-id>.<action>` routing key (VOIP-1404).
//
// Default is off everywhere. The fanout publish stays the system of record while dual publish
// lasts: a topic publish failure never propagates to the caller, and a fanout failure skips the
// topic publish entirely.
func WithGlobalTopicPublish() Option {
	return func(h *notifyHandler) {
		h.topicEnabled = true
	}
}

// NewNotifyHandler create NotifyHandler
// queueEvent: queue name for notification. the notify handler will publish the event to this queue name.
// publisher: publisher service name. the notify handler will publish the event with this publisher service name.
// opts: optional behavior modifiers. see WithGlobalTopicPublish.
func NewNotifyHandler(sockHandler sockhandler.SockHandler, reqHandler requesthandler.RequestHandler, queueEvent commonoutline.QueueName, publisher commonoutline.ServiceName, opts ...Option) NotifyHandler {
	h := &notifyHandler{
		sockHandler: sockHandler,
		reqHandler:  reqHandler,

		queueNotify: queueEvent,

		publisher: publisher,
	}
	for _, opt := range opts {
		opt(h)
	}

	if err := sockHandler.TopicCreate(string(queueEvent)); err != nil {
		logrus.Errorf("Could not declare the event exchange. err: %v", err)
		return nil
	}

	namespace := commonoutline.GetMetricNameSpace(publisher)
	initPrometheus(namespace)

	h.initGlobalTopicExchange()

	return h
}

// NewNotifyHandlerForExistingExchange creates a NotifyHandler for an exchange that has ALREADY
// been declared by the caller (e.g. via sockHandler.TopicCreateWithKind for a non-fanout kind).
// Unlike NewNotifyHandler, this does NOT call sockHandler.TopicCreate/TopicCreateWithKind
// internally -- it assumes the exchange already exists with the correct kind, and skips the
// redundant (and, for non-fanout kinds, conflicting) declare. Added for VOIP-1258 (see design
// doc §6, implementation plan Task 3.1) to support a second NotifyHandler instance bound to a
// topic-kind exchange, alongside the existing fanout-only NewNotifyHandler used everywhere else.
//
// NOTE (VOIP-1404): WithGlobalTopicPublish is valid here as well, but it must NOT be enabled for
// webhook-manager's scope-first instance -- that would triple-publish webhook events. Nothing
// enables it there today.
func NewNotifyHandlerForExistingExchange(sockHandler sockhandler.SockHandler, reqHandler requesthandler.RequestHandler, queueEvent commonoutline.QueueName, publisher commonoutline.ServiceName, opts ...Option) NotifyHandler {
	h := &notifyHandler{
		sockHandler: sockHandler,
		reqHandler:  reqHandler,

		queueNotify: queueEvent,

		publisher: publisher,
	}
	for _, opt := range opts {
		opt(h)
	}

	// NOTE: deliberately NOT calling sockHandler.TopicCreate/TopicCreateWithKind here -- the
	// caller is responsible for declaring the exchange BEFORE calling this constructor.
	// The global topic exchange below is a different exchange, shared by every publisher, and is
	// declared by this package on both constructors when the option is enabled.

	namespace := commonoutline.GetMetricNameSpace(publisher)
	initPrometheus(namespace)

	h.initGlobalTopicExchange()

	return h
}

// initGlobalTopicExchange declares the global topic exchange `bin-manager.event` when the
// WithGlobalTopicPublish option is enabled (VOIP-1404 design §3/§5.2).
//
// Declaration goes through the shared sockhandler helper only -- durable=true/autoDelete=false
// are hardcoded there -- because a redeclare with mismatched parameters closes the channel with
// 406 PRECONDITION_FAILED.
//
// Unlike NewNotifyHandler's fanout declare, a failure here does NOT nil out the handler: the
// topic exchange is strictly secondary while dual publish lasts, and the fanout path stays fully
// alive. The failure is not swallowed either (VOIP-1258 lesson): it is logged at Error level and
// every subsequently suppressed topic publish increments the error counter, so the degradation is
// visible both in the log and in the metrics.
//
// MUST run after initPrometheus so the counters used by the suppressed-publish path are non-nil.
func (h *notifyHandler) initGlobalTopicExchange() {
	if !h.topicEnabled {
		return
	}

	if err := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), topicExchangeKind); err != nil {
		logrus.Errorf("Could not declare the global topic exchange. the global topic publish has been disabled. err: %v", err)
		h.topicDisabled = true
	}
}

