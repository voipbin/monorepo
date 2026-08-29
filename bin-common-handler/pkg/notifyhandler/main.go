package notifyhandler

//go:generate mockgen -package notifyhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

// WebhookMessage defines
type WebhookMessage interface {
	CreateWebhookEvent() ([]byte, error)
}

// WebhookEventMessage is the payload contract of PublishWebhookEvent (VOIP-1419): the value is
// both a webhook message (CreateWebhookEvent) and an event with an explicit subscription address
// (EventSubscriptionID), because PublishWebhookEvent forwards it into PublishEvent. PublishWebhook
// alone still accepts a plain WebhookMessage -- the webhook RPC path never touches the topic
// exchange, so conversion-only DTOs stay out of the address contract.
type WebhookEventMessage interface {
	WebhookMessage
	eventtopic.SubscriptionIdentifier
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

	// global topic exchange metrics (VOIP-1404). VOIP-1407: promNotifyTotal/promNotifyProcessTime
	// are now shared across both the fanout-only and topic-only publish paths -- there is only one
	// active publish path per instance (topicEnabled selects which), so no double-counting is
	// possible by construction.
	promTopicPublishTotal     *prometheus.CounterVec
	promTopicPlaceholderTotal *prometheus.CounterVec

	// initPrometheusMu/initPrometheusDone guard against duplicate MustRegister panics when
	// initPrometheus is called more than once for the same namespace -- e.g. VOIP-1258's
	// second NotifyHandler instance (NewNotifyHandlerForExistingExchange, bound to the new
	// topic exchange) is constructed for the SAME publisher/namespace as another NotifyHandler
	// instance in the same process. Without this guard, the second initPrometheus call would
	// panic with "duplicate metrics collector registration attempted" on every startup.
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
	PublishEvent(ctx context.Context, eventType string, data eventtopic.SubscriptionIdentifier)
	PublishEventRaw(ctx context.Context, eventType string, dataType string, data []byte)
	PublishEventWithRoutingKey(ctx context.Context, eventType string, routingKey string, data interface{})

	PublishWebhook(ctx context.Context, customerID uuid.UUID, eventType string, data WebhookMessage)
	PublishWebhookEvent(ctx context.Context, customerID uuid.UUID, eventType string, data WebhookEventMessage)
}

type notifyHandler struct {
	sockHandler sockhandler.SockHandler
	reqHandler  requesthandler.RequestHandler

	queueNotify commonoutline.QueueName

	publisher commonoutline.ServiceName

	// topicEnabled reports whether this handler publishes to the global topic exchange (VOIP-1404).
	// VOIP-1407: when true, the topic exchange is the SOLE publish target (fanout removed).
	// Opt-in per handler instance via WithGlobalTopicPublish.
	topicEnabled bool
}

// Option customizes a NotifyHandler at construction time. Both constructors accept a variadic
// list of them, which keeps every existing call site source-compatible (VOIP-1404 design §5.1).
type Option func(*notifyHandler)

// WithGlobalTopicPublish makes this instance publish topic-ONLY (VOIP-1407): every event is
// published exclusively to the global topic exchange `bin-manager.event` with a
// `<publisher>.<resource>.<subscription-id>.<action>` routing key (VOIP-1404). No per-service
// fanout exchange is declared or published to when this option is enabled.
//
// Default (option omitted) is unchanged fanout-only behavior.
func WithGlobalTopicPublish() Option {
	return func(h *notifyHandler) {
		h.topicEnabled = true
	}
}

// NewNotifyHandler create NotifyHandler
// queueEvent: queue name for notification. the notify handler will publish the event to this queue name.
//
//	VOIP-1407: queueEvent is ignored on the WithGlobalTopicPublish() path -- no per-service fanout
//	exchange is declared or published to there.
//
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

	if !h.topicEnabled {
		// Fanout-only path, unchanged: voip-asterisk-proxy and confirmed-dead sites.
		if err := sockHandler.TopicCreate(string(queueEvent)); err != nil {
			logrus.Fatalf("Could not declare the event exchange. err: %v", err)
		}
	}

	namespace := commonoutline.GetMetricNameSpace(publisher)
	initPrometheus(namespace)

	// initGlobalTopicExchange is UNCHANGED here: it keeps its existing internal
	// `if !h.topicEnabled { return }` guard, so this call is a correct no-op for !topicEnabled
	// instances -- no caller-side gating change, no divergent behavior at
	// NewNotifyHandlerForExistingExchange's call site.
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
// NOTE (VOIP-1407): WithGlobalTopicPublish is valid here in principle, but enabling it would be
// actively harmful post-VOIP-1407: topicEnabled now means "topic-ONLY, fanout removed" (not
// "topic added on top of fanout" as when this note was written). This instance's queueNotify
// already targets a topic-kind exchange (QueueNameWebhookEventTopic) and its production traffic
// already goes exclusively through PublishEventWithRoutingKey. Enabling the option would run this
// instance through publishTopicEventOrErr's bin-manager.event path IN ADDITION, with no fanout
// fallback to degrade to if anything about that additional path fails. Nothing enables it today
// (verified); keep it that way.
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
// VOIP-1407: a declare failure here is now FATAL for topicEnabled=true instances -- there is no
// fanout fallback left to degrade to. logrus.Fatalf halts the process directly: no caller of
// NewNotifyHandler/NewNotifyHandlerForExistingExchange has ever checked the return value for nil.
//
// MUST run after initPrometheus so the counters used by the topic-publish path are non-nil.
func (h *notifyHandler) initGlobalTopicExchange() {
	if !h.topicEnabled {
		return
	}

	if err := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), topicExchangeKind); err != nil {
		logrus.Fatalf("Could not declare the global topic exchange. err: %v", err)
	}
}

