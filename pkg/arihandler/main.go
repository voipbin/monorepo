package arihandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package arihandler -destination ./mock_arihandler_arihandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/confbridgehandler"
	db "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
)

// EventHandler intreface for ARI request handler
type EventHandler interface {
	Run(queue, receiver string) error
}

type eventHandler struct {
	db         db.DBHandler
	cache      cachehandler.CacheHandler
	rabbitSock rabbitmqhandler.Rabbit

	reqHandler        requesthandler.RequestHandler
	notifyHandler     notifyhandler.NotifyHandler
	callHandler       callhandler.CallHandler
	confbridgeHandler confbridgehandler.ConfbridgeHandler
}

// List of default values
const (
	defaultTimeStamp = "9999-01-01 00:00:00.000000" // default timestamp
)

var (
	metricsNamespace = "call_manager"

	promARIEventTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "ari_event_total",
			Help:      "Total number of received ARI event types.",
		},
		[]string{"type", "asterisk_id"},
	)

	promARIProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "ari_event_process_time",
			Help:      "Process time of received ARI events",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"asterisk_id", "type"},
	)

	promChannelTransportAndDirection = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "channel_transport_direction_total",
			Help:      "Total number of channel's transport and direction.",
		},
		[]string{"transport", "direction"},
	)
)

func init() {
	prometheus.MustRegister(
		promARIEventTotal,
		promARIProcessTime,
		promChannelTransportAndDirection,
	)

}

// NewEventHandler create EventHandler
func NewEventHandler(
	sock rabbitmqhandler.Rabbit,
	db db.DBHandler,
	cache cachehandler.CacheHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	callHandler callhandler.CallHandler,
	confbridgeHandler confbridgehandler.ConfbridgeHandler,
) EventHandler {
	h := &eventHandler{
		rabbitSock:        sock,
		db:                db,
		cache:             cache,
		reqHandler:        reqHandler,
		notifyHandler:     notifyHandler,
		callHandler:       callHandler,
		confbridgeHandler: confbridgeHandler,
	}

	return h
}

// Run starts to receive ARI event and process it.
func (h *eventHandler) Run(queue, receiver string) error {
	// create queue for ari event receive
	log := logrus.WithFields(logrus.Fields{
		"queue": queue,
	})

	log.Infof("Creating rabbitmq queue for ARI event receiving.")

	err := h.rabbitSock.QueueDeclare(queue, true, false, false, false)
	if err != nil {
		return err
	}

	// Set QoS
	if err := h.rabbitSock.QueueQoS(queue, 1, 0); err != nil {
		log.Errorf("Could not set the queue's qos. err: %v", err)
		return err
	}

	// receive ARI event
	go func() {
		for {
			if err := h.rabbitSock.ConsumeMessageOpt(queue, receiver, false, false, false, h.processEventRun); err != nil {
				log.Errorf("Could not consume the message. err: %v", err)
			}
		}
	}()
	return nil
}

func (h *eventHandler) processEventRun(m *rabbitmqhandler.Event) error {
	go func() {
		if err := h.processEvent(m); err != nil {
			logrus.Errorf("Could not consume the ARI event message correctly. err: %v", err)
		}
	}()

	return nil
}

// processEvent processes received ARI event
func (h *eventHandler) processEvent(m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(
		logrus.Fields{
			"event": m,
		},
	)

	if m.Type != "ari_event" {
		return fmt.Errorf("wrong event type received. type: %s", m.Type)
	}

	// parse
	event, evt, err := ari.Parse([]byte(m.Data))
	if err != nil {
		return fmt.Errorf("could not parse the message. err: %v", err)

	}

	log = log.WithFields(
		logrus.Fields{
			"asterisk": event.AsteriskID,
			"type":     event.Type,
		},
	)

	log.Debugf("Received ARI event. type: %s", event.Type)
	promARIEventTotal.WithLabelValues(string(event.Type), event.AsteriskID).Inc()

	// processMap maps ARIEvent name and event handler.
	var processMap = map[ari.EventType]func(context.Context, interface{}) error{
		ari.EventTypeBridgeCreated:        h.eventHandlerBridgeCreated,
		ari.EventTypeBridgeDestroyed:      h.eventHandlerBridgeDestroyed,
		ari.EventTypeChannelCreated:       h.eventHandlerChannelCreated,
		ari.EventTypeChannelDestroyed:     h.eventHandlerChannelDestroyed,
		ari.EventTypeChannelDtmfReceived:  h.eventHandlerChannelDtmfReceived,
		ari.EventTypeChannelEnteredBridge: h.eventHandlerChannelEnteredBridge,
		ari.EventTypeChannelLeftBridge:    h.eventHandlerChannelLeftBridge,
		ari.EventTypeChannelStateChange:   h.eventHandlerChannelStateChange,
		ari.EventTypeChannelVarset:        h.eventHandlerChannelVarset,
		ari.EventTypeContactStatusChange:  h.eventHandlerContactStatusChange,
		ari.EventTypePlaybackStarted:      h.eventHandlerPlaybackStarted,
		ari.EventTypePlaybackFinished:     h.eventHandlerPlaybackFinished,
		ari.EventTypeRecordingFinished:    h.eventHandlerRecordingFinished,
		ari.EventTypeRecordingStarted:     h.eventHandlerRecordingStarted,
		ari.EventTypeStasisEnd:            h.eventHandlerStasisEnd,
		ari.EventTypeStasisStart:          h.eventHandlerStasisStart,
	}

	handler := processMap[event.Type]
	if handler == nil {
		// no handler
		return nil
	}

	start := time.Now()

	ctx := context.Background()
	err = handler(ctx, evt)
	elapsed := time.Since(start)

	promARIProcessTime.WithLabelValues(event.AsteriskID, string(event.Type)).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		return fmt.Errorf("could not process the ari event correctly. err: %v", err)
	}

	return nil
}

// contextType
type contextType string

// List of contextType types.
const (
	contextTypeConference contextType = "conf"
	contextTypeCall       contextType = "call"
)

const defaultExistTimeout = time.Second * 3

// getContextType returns CONTEXT's type
func getContextType(message interface{}) contextType {
	if message == nil {
		return contextTypeCall
	}

	tmp := strings.Split(message.(string), "-")[0]
	switch tmp {
	case string(contextTypeConference):
		return contextTypeConference
	default:
		return contextTypeCall
	}
}
