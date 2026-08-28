package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_subscribehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"strings"
	"time"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cscustomer "monorepo/bin-customer-manager/models/customer"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-webhook-manager/pkg/accounthandler"
	"monorepo/bin-webhook-manager/pkg/cachehandler"
)

// list of publishers
const (
	publisherCustomerManager = "customer-manager"
	publisherFlowManager     = "flow-manager"
)

// topicPatterns is the set of binding patterns this service binds on the global
// bin-manager.event topic exchange (VOIP-1406): exactly one PatternAction per event pair the
// dispatch switch in processEvent handles, built with the same normalization the publish side
// uses to build routing keys. The binding golden test (binding_golden_test.go) pins this list
// byte-for-byte against the design's normative bind set.
var topicPatterns = []string{
	eventtopic.PatternAction(string(commonoutline.ServiceNameCustomerManager), "customer", "created"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameCustomerManager), "customer", "updated"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameFlowManager), "activeflow", "created"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameFlowManager), "activeflow", "updated"),
	eventtopic.PatternAction(string(commonoutline.ServiceNameFlowManager), "activeflow", "deleted"),
}

// fanoutUnbindTargets is the set of legacy per-service fanout event exchanges this queue
// unbinds after every topic pattern is bound (VOIP-1406). This subscribehandler receives its
// subscribe targets as a comma-joined string split inside Run(); this package-level list is
// the equivalent derivation of that split result -- cmd wires exactly these two exchanges --
// with nothing excluded (webhook-manager has no retained asterisk leg). The fanout
// QueueSubscribe calls themselves stay as the rollback surface until VOIP-1407.
var fanoutUnbindTargets = []string{
	string(commonoutline.QueueNameCustomerEvent),
	string(commonoutline.QueueNameFlowEvent),
}

// SubscribeHandler interface
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler

	subscribeQueue    string
	subscribesTargets string

	accountHandler accounthandler.AccountHandler
	cacheHandler   cachehandler.CacheHandler
}

var (
	metricsNamespace = "webhook_manager"

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
	subscribeTargets string,
	accountHandler accounthandler.AccountHandler,
	cacheHandler cachehandler.CacheHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		sockHandler: sockHandler,

		subscribeQueue:    subscribeQueue,
		subscribesTargets: subscribeTargets,

		accountHandler: accountHandler,
		cacheHandler:   cacheHandler,
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

	// subscribe each targets
	targets := strings.Split(h.subscribesTargets, ",")
	for _, target := range targets {
		if errSubscribe := h.sockHandler.QueueSubscribe(h.subscribeQueue, target); errSubscribe != nil {
			log.Errorf("Could not subscribe the target. target: %s, err: %v", target, errSubscribe)
			return errSubscribe
		}
	}

	// Migrate this queue's event intake from the per-service fanout exchanges to the global
	// bin-manager.event topic exchange (VOIP-1406). Bind the new topic patterns FIRST, then
	// unbind the old fanout exchanges, so there is no window where the queue is bound to
	// neither. This MUST run synchronously here, BEFORE the ConsumeMessage goroutine below:
	// QueueBind/QueueUnbind and ConsumeMessage's internal basic.consume share the same
	// underlying AMQP channel for a given queue, and racing them closes the channel with an
	// "unexpected command received" 503 (reproduced in production, VOIP-1258, 2026-07-14).
	if errDeclare := h.sockHandler.TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic"); errDeclare != nil {
		// stay fully on the fanout exchanges; skip binding and unbinding entirely.
		log.Errorf("Could not declare the global topic exchange. Staying fully on the fanout exchanges. exchange: %s, err: %v", string(commonoutline.QueueNameEvent), errDeclare)
	} else {
		// bind ALL patterns -- all-or-nothing.
		bound := []string{}
		ok := true
		for _, pattern := range topicPatterns {
			if errBind := h.sockHandler.QueueBind(h.subscribeQueue, pattern, string(commonoutline.QueueNameEvent), false, nil); errBind != nil {
				log.Errorf("Could not bind the topic pattern. Staying fully on the fanout exchanges. pattern: %s, err: %v", pattern, errBind)
				ok = false
				break
			}
			bound = append(bound, pattern)
		}

		if !ok {
			// best-effort rollback of the partial topic binds, then stay fully on fanout.
			for _, pattern := range bound {
				if errUnbind := h.sockHandler.QueueUnbind(h.subscribeQueue, pattern, string(commonoutline.QueueNameEvent), nil); errUnbind != nil {
					log.Errorf("CRITICAL: partial topic bind could not be rolled back. The queue keeps a stray topic binding (partial double delivery). Manual intervention required. queue: %s, pattern: %s, err: %v", h.subscribeQueue, pattern, errUnbind)
				}
			}
		} else {
			// unbind ALL old fanout exchanges -- only after every pattern bound.
			// unbind failure: CRITICAL log, not fatal (double delivery beats loss).
			for _, target := range fanoutUnbindTargets {
				if errUnbind := h.sockHandler.QueueUnbind(h.subscribeQueue, "", target, nil); errUnbind != nil {
					log.Errorf("CRITICAL: could not unbind the old fanout exchange after binding the topic patterns. queue: %s is still bound to BOTH exchanges (double delivery). Manual intervention required. exchange: %s, err: %v", h.subscribeQueue, target, errUnbind)
				}
			}
		}
	}

	// receive subscribe events
	go func() {
		if errConsume := h.sockHandler.ConsumeMessage(context.Background(), h.subscribeQueue, "webhook-manager", false, false, false, 10, h.processEventRun); errConsume != nil {
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

	log := logrus.WithFields(
		logrus.Fields{
			"message": m,
		},
	)
	log.Debugf("Received subscribed event. publisher: %s, type: %s", m.Publisher, m.Type)

	var err error
	start := time.Now()
	switch {

	// customer-manager
	case m.Publisher == publisherCustomerManager && (m.Type == string(cscustomer.EventTypeCustomerCreated)):
		err = h.processEventCSCustomerCreatedUpdated(m)

	case m.Publisher == publisherCustomerManager && (m.Type == string(cscustomer.EventTypeCustomerUpdated)):
		err = h.processEventCSCustomerCreatedUpdated(m)

	// flow-manager
	case m.Publisher == publisherFlowManager && (m.Type == fmactiveflow.EventTypeActiveflowCreated || m.Type == fmactiveflow.EventTypeActiveflowUpdated):
		err = h.processEventFMActiveflowCreatedUpdated(m)

	case m.Publisher == publisherFlowManager && (m.Type == fmactiveflow.EventTypeActiveflowDeleted):
		err = h.processEventFMActiveflowDeleted(m)

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		// ignore the event
		return
	}
	elapsed := time.Since(start)
	promEventProcessTime.WithLabelValues(m.Publisher, string(m.Type)).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not process the event correctly. publisher: %s, type: %s, err: %v", m.Publisher, m.Type, err)
	}
}
