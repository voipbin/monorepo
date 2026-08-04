package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	wmwebhook "monorepo/bin-webhook-manager/models/webhook"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/pkg/pubsubhandler"
)

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler
	reqHandler  requesthandler.RequestHandler

	subscribeQueueNamePod string // subscribe queue name for this pod
	subscribeTargets      []string

	pubHandler pubsubhandler.PubHandler
}

var (
	metricsNamespace = "api_manager"

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
	reqHandler requesthandler.RequestHandler,
	subscribeQueueName string,
	subscribeTargets []string,
	pubHandler pubsubhandler.PubHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		sockHandler: sockHandler,
		reqHandler:  reqHandler,

		subscribeQueueNamePod: subscribeQueueName,
		subscribeTargets:      subscribeTargets,

		pubHandler: pubHandler,
	}

	return h
}

func (h *subscribeHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})
	log.Info("Creating rabbitmq queue for listen.")

	// declare the queue for subscribe(pod)
	log.Debugf("Declaring the queue for subscribe(pod). queue_name: %s", h.subscribeQueueNamePod)
	if err := h.sockHandler.QueueCreate(h.subscribeQueueNamePod, "volatile"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// subscribe each targets
	for _, target := range h.subscribeTargets {
		if errSubscribe := h.sockHandler.QueueSubscribe(h.subscribeQueueNamePod, target); errSubscribe != nil {
			log.Errorf("Could not subscribe the target. target: %s, err: %v", target, errSubscribe)
			return errSubscribe
		}
	}

	// NOTE (VOIP-1296 final cutover): the unconditional "#" wildcard baseline bind to the topic
	// exchange that used to live here has been removed. It was a rollout safety net kept
	// alongside pkg/websockhandler's scopeRefCount mechanism (VOIP-1258 §9), which already
	// binds/unbinds this pod's queue per active client subscription scope. Keeping both meant
	// every event was still delivered to every pod regardless of connected client count --
	// exactly the problem VOIP-1258 set out to fix. scopeRefCount is the sole binding mechanism
	// for this pod's queue going forward.

	// receive subscribe events
	go func() {
		if errConsume := h.sockHandler.ConsumeMessage(context.Background(), h.subscribeQueueNamePod, string(commonoutline.ServiceNameAPIManager), false, false, false, 10, h.processEventRun); errConsume != nil {
			logrus.Errorf("Could not consume the request message correctly. err: %v", errConsume)
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

	log := logrus.WithFields(
		logrus.Fields{
			"message": m,
		},
	)
	log.Debugf("Received subscribed event. publisher: %s, type: %s", m.Publisher, m.Type)

	var err error
	start := time.Now()
	ctx := context.Background()

	switch {

	//// webhook-manager: OLD fanout path -- the wrapped {"type":"webhook_published","data":
	//// {"type":<resource event type>,"data":{...}}} envelope. VOIP-1296 (Task 4.6's cutover)
	//// removed the fanout publish in bin-webhook-manager, so this case is now unreachable in
	//// practice (this pod's queue was already only ever bound to the topic exchange, never the
	//// fanout one). Left in place defensively rather than deleted, to avoid scope creep in the
	//// cutover PR; a follow-up can remove this case and processEventWebhookManagerWebhookPublished
	//// once confirmed dead in production.
	case m.Publisher == string(commonoutline.ServiceNameWebhookManager) && (m.Type == string(wmwebhook.EventTypeWebhookPublished)):
		err = h.processEventWebhookManagerWebhookPublished(ctx, m)

	//// webhook-manager: NEW topic-exchange routing-keyed path (VOIP-1258 §6/§8). Published via
	//// PublishEventWithRoutingKey with the REAL resource event type as m.Type (e.g.
	//// "call_created") and the UNWRAPPED resource object as m.Data (bin-webhook-manager's
	//// publishRoutingKeyedEvent already did the envelope unwrap at publish time -- see that
	//// function's doc comment). This is NOT the same shape as the fanout path above (which is
	//// still the doubly-wrapped envelope), so it needs its own handler, not reuse of
	//// processEventWebhookManagerWebhookPublished (which expects the wrapped shape and would
	//// fail to unmarshal this one correctly).
	////
	//// CRITICAL (production bug found 2026-07-15, post-envelope-fix verification): before this
	//// case existed, every event arriving via the new topic exchange had m.Type set to the real
	//// resource event type (never "webhook_published"), so it always fell through to `default:
	//// return` below and was silently discarded -- the AMQP message reached this pod's queue
	//// correctly (confirmed via RabbitMQ queue/binding inspection) but was never handed to
	//// pubHandler.Publish, so it never reached the browser's websocket. This is why the
	//// AMQP-level fix (envelope unwrapping in bin-webhook-manager) alone was insufficient --
	//// the consumer side needed a matching case for the new event shape.
	case m.Publisher == string(commonoutline.ServiceNameWebhookManager) && (m.Type != string(wmwebhook.EventTypeWebhookPublished)):
		err = h.processEventWebhookManagerRoutingKeyedEvent(ctx, m)

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		// ignore the event.
		return
	}
	elapsed := time.Since(start)
	promEventProcessTime.WithLabelValues(m.Publisher, string(m.Type)).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not process the event correctly. publisher: %s, type: %s, err: %v", m.Publisher, m.Type, err)
	}
}
