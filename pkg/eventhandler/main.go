package eventhandler

//go:generate mockgen -destination ./mock_eventhandler_eventhandler.go -package eventhandler -source ./main.go EventHandler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler"
	db "gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/ari"
	rabbitmq "gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

// ARIEvent is the structure for ARI event parse.
type ARIEvent struct {
	eventType   string
	application string
	asteriskID  string
	timestamp   time.Time
	event       interface{}
}

// EventHandler intreface for ARI request handler
type EventHandler interface {
	Run(queue, receiver string) error
}

type eventHandler struct {
	db         db.DBHandler
	cache      cachehandler.CacheHandler
	rabbitSock rabbitmq.Rabbit

	reqHandler  requesthandler.RequestHandler
	callHandler callhandler.CallHandler
	confHandler conferencehandler.ConferenceHandler
}

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
func NewEventHandler(sock rabbitmq.Rabbit, db db.DBHandler, cache cachehandler.CacheHandler, reqHandler requesthandler.RequestHandler, callHandler callhandler.CallHandler) EventHandler {
	handler := &eventHandler{
		rabbitSock: sock,
		db:         db,
		cache:      cache,
	}

	handler.reqHandler = reqHandler
	handler.callHandler = callHandler
	handler.confHandler = conferencehandler.NewConferHandler(reqHandler, db, cache)

	return handler
}

// Run starts to receive ARI event and process it.
func (h *eventHandler) Run(queue, receiver string) error {
	// create queue for ari event receive
	log.WithFields(log.Fields{
		"queue": queue,
	}).Infof("Creating rabbitmq queue for ARI event receiving.")

	err := h.rabbitSock.QueueDeclare(queue, true, false, false, false)
	if err != nil {
		return err
	}

	// receive ARI event
	go func() {
		for {
			err := h.rabbitSock.ConsumeMessage(queue, receiver, h.processEvent)
			if err != nil {
				log.Errorf("Could not consume the ARI message correctly. Will try again after 1 second. err: %v", err)
				time.Sleep(time.Second * 1)
			}
		}
	}()
	return nil
}

// processEvent processes received ARI event
func (h *eventHandler) processEvent(m *rabbitmq.Event) error {
	if m.Type != "ari_event" {
		return fmt.Errorf("Wrong event type recevied. type: %s", m.Type)
	}

	// parse
	event, evt, err := ari.Parse([]byte(m.Data))
	if err != nil {
		return err
	}

	log := log.WithFields(
		log.Fields{
			"asterisk": event.AsteriskID,
			"type":     event.Type,
		})

	log.WithFields(
		logrus.Fields{
			"event": m,
		}).Debug("Received ARI event.")
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
		ari.EventTypePlaybackFinished:     h.eventHandlerPlaybackFinished,
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
		log.WithFields(
			logrus.Fields{
				"event": m,
			}).Errorf("Could not process the ari event correctly. err: %v", err)
		return err
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
