package subscribehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/arieventhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
)

// list of publishers
const (
	publisherAsteriskProxy = "asterisk-proxy"
	publisherFlowManager   = "flow-manager"
)

// SubscribeHandler intreface for subscribed event listen handler
type SubscribeHandler interface {
	Run() error
}

type subscribeHandler struct {
	serviceName string
	rabbitSock  rabbitmqhandler.Rabbit

	subscribeQueue    string
	subscribesTargets string

	ariEventHandler arieventhandler.ARIEventHandler
	callHandler     callhandler.CallHandler
}

var (
	metricsNamespace = "call_manager"

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
	serviceName string,
	sock rabbitmqhandler.Rabbit,
	subscribeQueue string,
	subscribeTargets string,
	ariEventHandler arieventhandler.ARIEventHandler,
	callHandler callhandler.CallHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		serviceName:       serviceName,
		rabbitSock:        sock,
		subscribeQueue:    subscribeQueue,
		subscribesTargets: subscribeTargets,
		ariEventHandler:   ariEventHandler,
		callHandler:       callHandler,
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
	if err := h.rabbitSock.QueueDeclare(h.subscribeQueue, true, true, false, false); err != nil {
		log.Errorf("Could not declare the queue for subscribe. err: %v", err)
		return errors.Wrap(err, "could not declare the queue for listenHandler.")
	}

	// subscribe each targets
	targets := strings.Split(h.subscribesTargets, ",")
	for _, target := range targets {

		// bind each targets
		if err := h.rabbitSock.QueueBind(h.subscribeQueue, "", target, false, nil); err != nil {
			logrus.Errorf("Could not subscribe the target. target: %s, err: %v", target, err)
			return err
		}
	}

	// receive subscribe events
	go func() {
		for {
			if errConsume := h.rabbitSock.ConsumeMessageOpt(h.subscribeQueue, "call-manager", false, false, false, 10, h.processEventRun); errConsume != nil {
				logrus.Errorf("Could not consume the subscribed evnet message correctly. err: %v", errConsume)
			}
		}
	}()

	return nil
}

// processEventRun runs the event process handler.
func (h *subscribeHandler) processEventRun(m *rabbitmqhandler.Event) error {
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
func (h *subscribeHandler) processEvent(m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEvent",
		"event": m,
	})

	ctx := context.Background()

	var err error
	start := time.Now()

	switch {
	// asterisk-proxy
	case m.Publisher == publisherAsteriskProxy:
		err = h.processEventAsteriskProxy(ctx, m)

	case m.Publisher == publisherFlowManager && m.Type == fmactiveflow.EventTypeActiveflowDeleted:
		err = h.processEventActiveflowDeleted(ctx, m)

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
