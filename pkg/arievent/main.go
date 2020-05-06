package arievent

//go:generate mockgen -destination ./mock_arievent_eventhandler.go -package arievent gitlab.com/voipbin/bin-manager/call-manager/pkg/arievent EventHandler

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	ari "gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	db "gitlab.com/voipbin/bin-manager/call-manager/pkg/db_handler"
	rabbitmq "gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/svchandler"
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
	HandleARIEvent(queue, receiver string) error

	processEvent(m *rabbitmq.Event) error

	eventHandlerStasisStart(ctx context.Context, evt interface{}) error
	eventHandlerChannelCreated(ctx context.Context, evt interface{}) error
}

type eventHandler struct {
	db         db.DBHandler
	rabbitSock rabbitmq.Rabbit

	reqHandler requesthandler.RequestHandler
	svcHandler svchandler.SVCHandler
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
)

func init() {
	prometheus.MustRegister(
		promARIEventTotal,
		promARIProcessTime,
	)

}

// NewEventHandler create EventHandler
func NewEventHandler(sock rabbitmq.Rabbit, db db.DBHandler, reqHandler requesthandler.RequestHandler, svcHandler svchandler.SVCHandler) EventHandler {
	evtHandler := &eventHandler{
		rabbitSock: sock,
		db:         db,
	}

	evtHandler.reqHandler = reqHandler
	evtHandler.svcHandler = svcHandler

	return evtHandler
}

// HandleARIEvent recevies ARI event and process it.
func (h *eventHandler) HandleARIEvent(queue, receiver string) error {
	// create queue for ari event receive
	log.WithFields(log.Fields{
		"queue": queue,
	}).Infof("Creating rabbitmq queue for ARI event receiving.")

	err := h.rabbitSock.DeclareQueue(queue, true, false, false, false)
	if err != nil {
		return err
	}

	// receive ARI event
	if err := h.rabbitSock.ConsumeMessage(queue, receiver, h.processEvent); err != nil {
		return err
	}
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

	log.WithFields(log.Fields{
		"asterisk_id": event.AsteriskID,
		"type":        event.Type,
	}).Debugf("Received ARI event. message: %s", m)
	promARIEventTotal.WithLabelValues(string(event.Type), event.AsteriskID).Inc()

	// processMap maps ARIEvent name and event handler.
	var processMap = map[ari.EventType]func(context.Context, interface{}) error{
		ari.EventTypeChannelCreated:     h.eventHandlerChannelCreated,
		ari.EventTypeChannelDestroyed:   h.eventHandlerChannelDestroyed,
		ari.EventTypeChannelStateChange: h.eventHandlerChannelStateChange,
		ari.EventTypeStasisEnd:          h.eventHandlerStasisEnd,
		ari.EventTypeStasisStart:        h.eventHandlerStasisStart,
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
		log.Errorf("Could not process the ari event correctly. err: %v", err)
		return err
	}

	return nil
}
