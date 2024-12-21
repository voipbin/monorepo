package subscribehandler

//go:generate mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	"monorepo/bin-call-manager/models/common"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/pkg/arieventhandler"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
)

// SubscribeHandler intreface for subscribed event listen handler
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	sockHandler sockhandler.SockHandler

	subscribeQueue   commonoutline.QueueName
	subscribeTargets []string

	ariEventHandler arieventhandler.ARIEventHandler

	callHandler       callhandler.CallHandler
	groupcallHandler  groupcallhandler.GroupcallHandler
	confbridgeHandler confbridgehandler.ConfbridgeHandler
}

var (
	metricsNamespace = commonoutline.GetMetricNameSpace(common.Servicename)

	promSubscribeProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "subscribe_event_process_time",
			Help:      "Process time of subscribed events",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"publisher", "type"},
	)

	promARIEventTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "ari_event_listen_total",
			Help:      "Total number of received ARI event types.",
		},
		[]string{"type", "asterisk_id"},
	)

	promARIProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "ari_event_listen_process_time",
			Help:      "Process time of received ARI events",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"asterisk_id", "type"},
	)
)

func init() {
	prometheus.MustRegister(
		promSubscribeProcessTime,
		promARIEventTotal,
		promARIProcessTime,
	)
}

// NewSubscribeHandler create EventHandler
func NewSubscribeHandler(
	sock sockhandler.SockHandler,
	subscribeQueue commonoutline.QueueName,
	subscribeTargets []string,
	ariEventHandler arieventhandler.ARIEventHandler,
	callHandler callhandler.CallHandler,
	groupcallHandler groupcallhandler.GroupcallHandler,
	confbridgeHandler confbridgehandler.ConfbridgeHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		sockHandler:       sock,
		subscribeQueue:    subscribeQueue,
		subscribeTargets:  subscribeTargets,
		ariEventHandler:   ariEventHandler,
		callHandler:       callHandler,
		groupcallHandler:  groupcallHandler,
		confbridgeHandler: confbridgeHandler,
	}

	return h
}

// Run starts to receive ARI event and process it.
func (h *subscribeHandler) Run() error {
	// create queue for ari event receive
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})
	log.Infof("Creating rabbitmq queue for ARI event receiving.")

	// declare the queue for subscribe
	if err := h.sockHandler.QueueCreate(string(h.subscribeQueue), "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for subscribeHandler. err: %v", err)
	}

	// subscribe each targets
	for _, target := range h.subscribeTargets {
		if errSubscribe := h.sockHandler.QueueSubscribe(string(h.subscribeQueue), target); errSubscribe != nil {
			log.Errorf("Could not subscribe the target. target: %s, err: %v", target, errSubscribe)
			return errSubscribe
		}
	}

	// receive subscribe events
	go func() {
		for {
			if errConsume := h.sockHandler.ConsumeMessage(string(h.subscribeQueue), string(common.Servicename), false, false, false, 20, h.processEventRun); errConsume != nil {
				logrus.Errorf("Could not consume the subscribed evnet message correctly. err: %v", errConsume)
			}
		}
	}()

	return nil
}

// processEventRun runs the event process handler.
func (h *subscribeHandler) processEventRun(m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventRun",
		"event": m,
	})

	if errProcess := h.processEvent(m); errProcess != nil {
		log.Errorf("Could not consume the ARI event message correctly. err: %v", errProcess)
	}

	return nil
}

// processEvent processes received ARI event
func (h *subscribeHandler) processEvent(m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEvent",
		"event": m,
	})
	ctx := context.Background()

	var err error
	start := time.Now()

	switch {
	// asterisk-proxy
	case m.Publisher == string(commonoutline.ServiceNameAsteriskProxy):
		err = h.processEventAsteriskProxy(ctx, m)

	// customer-manager
	case m.Publisher == string(commonoutline.ServiceNameCustomerManager) && m.Type == cucustomer.EventTypeCustomerDeleted:
		err = h.processEventCUCustomerDeleted(ctx, m)

	// flow-manager
	case m.Publisher == string(commonoutline.ServiceNameFlowManager) && m.Type == fmactiveflow.EventTypeActiveflowUpdated:
		err = h.processEventFMActiveflowUpdated(ctx, m)

	default:
		// ignore the event.
		return nil
	}

	elapsed := time.Since(start)
	promSubscribeProcessTime.WithLabelValues(m.Publisher, m.Type).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not handle the subscribed event correctly. err: %v", err)
		return fmt.Errorf("could not process the ari event correctly. err: %v", err)
	}

	return nil
}
