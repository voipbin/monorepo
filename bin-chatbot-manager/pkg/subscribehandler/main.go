package subscribehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package subscribehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	commonoutline "gitlab.com/voipbin/bin-manager/common-handler.git/models/outline"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	tmtranscript "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"

	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/pkg/chatbotcallhandler"
)

// list of publishers
const (
	publisherCallManager       = string(commonoutline.ServiceNameCallManager)
	publisherTranscribeManager = string(commonoutline.ServiceNameTranscribeManager)
)

// SubscribeHandler intreface for subscribed event listen handler
type SubscribeHandler interface {
	Run() error
}

// subscribeHandler define
type subscribeHandler struct {
	serviceName string
	rabbitSock  rabbitmqhandler.Rabbit

	subscribeQueue   string
	subscribeTargets []string

	chatbotcallHandler chatbotcallhandler.ChatbotcallHandler
}

var (
	metricsNamespace = "chatbot_manager"

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
)

func init() {
	prometheus.MustRegister(
		promSubscribeProcessTime,
	)
}

// NewSubscribeHandler create EventHandler
func NewSubscribeHandler(
	serviceName string,
	sock rabbitmqhandler.Rabbit,
	subscribeQueue string,
	subscribeTargets []string,
	chatbotcallHandler chatbotcallhandler.ChatbotcallHandler,
) SubscribeHandler {
	h := &subscribeHandler{
		serviceName:        serviceName,
		rabbitSock:         sock,
		subscribeQueue:     subscribeQueue,
		subscribeTargets:   subscribeTargets,
		chatbotcallHandler: chatbotcallHandler,
	}

	return h
}

// Run starts to receive subscribed event and process it.
func (h *subscribeHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})
	log.Infof("Creating rabbitmq queue for subscribed event receiving.")

	// declare the queue for subscribe
	if err := h.rabbitSock.QueueDeclare(h.subscribeQueue, true, true, false, false); err != nil {
		log.Errorf("Could not declare the queue for subscribe. err: %v", err)
		return errors.Wrap(err, "could not declare the queue for listenHandler.")
	}

	// subscribe each targets
	for _, target := range h.subscribeTargets {

		// bind each targets
		if err := h.rabbitSock.QueueBind(h.subscribeQueue, "", target, false, nil); err != nil {
			log.Errorf("Could not subscribe the target. target: %s, err: %v", target, err)
			return err
		}
	}

	// receive subscribe events
	go func() {
		for {
			if errConsume := h.rabbitSock.ConsumeMessageOpt(h.subscribeQueue, string(commonoutline.ServiceNameChatbotManager), false, false, false, 10, h.processEventRun); errConsume != nil {
				log.Errorf("Could not consume the subscribed evnet message correctly. err: %v", errConsume)
			}
		}
	}()

	return nil
}

// processEventRun runs the event process handler.
func (h *subscribeHandler) processEventRun(m *rabbitmqhandler.Event) error {
	go h.processEvent(m)

	return nil
}

// processEvent processes received event
func (h *subscribeHandler) processEvent(m *rabbitmqhandler.Event) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEvent",
		"event": m,
	})

	ctx := context.Background()

	var err error
	start := time.Now()

	switch {

	// call-manager
	case m.Publisher == publisherCallManager && m.Type == string(cmconfbridge.EventTypeConfbridgeJoined):
		err = h.processEventCMConfbridgeJoined(ctx, m)

	case m.Publisher == publisherCallManager && m.Type == string(cmconfbridge.EventTypeConfbridgeLeaved):
		err = h.processEventCMConfbridgeLeaved(ctx, m)

	// transcribe-manager
	case m.Publisher == publisherTranscribeManager && m.Type == string(tmtranscript.EventTypeTranscriptCreated):
		err = h.processEventTMTranscriptCreated(ctx, m)

	default:
		// ignore the event.
		return
	}

	elapsed := time.Since(start)
	promSubscribeProcessTime.WithLabelValues(m.Publisher, m.Type).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not process the event correctly. publisher: %s, type: %s, err: %v", m.Publisher, m.Type, err)
	}
}
