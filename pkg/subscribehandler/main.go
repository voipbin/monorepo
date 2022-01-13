package subscribehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package subscribehandler -destination ./mock_subscribehandler_subscribehandler.go -source main.go -build_flags=-mod=mod

import (
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/dbhandler"
)

// list of publishers
const (
	publisherCallManager = "call-manager"
)

// SubscribeHandler interface
type SubscribeHandler interface {
	Run(queue, exchangeDelay string) error
}

type subscribeHandler struct {
	rabbitSock rabbitmqhandler.Rabbit
	db         dbhandler.DBHandler

	subscribeQueue    string
	subscribesTargets string

	conferenceHandler conferencehandler.ConferenceHandler
}

var (
	metricsNamespace = "conference_manager"

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
	rabbitSock rabbitmqhandler.Rabbit,
	db dbhandler.DBHandler,
	subscribeQueue string,
	subscribeTargets string,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	conferenceHandler conferencehandler.ConferenceHandler,
) SubscribeHandler {

	h := &subscribeHandler{
		rabbitSock: rabbitSock,
		db:         db,

		subscribeQueue:    subscribeQueue,
		subscribesTargets: subscribeTargets,
		conferenceHandler: conferenceHandler,
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
	case m.Publisher == publisherCallManager &&
		(m.Type == string(cmconfbridge.EventTypeConfbridgeJoined) || m.Type == string(cmconfbridge.EventTypeConfbridgeLeaved)):
		err = h.processEventCMConfbridgeJoinedLeaved(m)

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		log.Debugf("Could not find correct event handler.")
	}
	elapsed := time.Since(start)
	promEventProcessTime.WithLabelValues(m.Publisher, string(m.Type)).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not process the event correctly. publisher: %s, type: %s, err: %v", m.Publisher, m.Type, err)
	}
}
