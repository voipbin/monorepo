package arievent

//go:generate mockgen -destination ./mock_arievent_eventhandler.go -package arievent gitlab.com/voipbin/bin-manager/call-manager/pkg/arievent EventHandler

import (
	"context"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	ari "gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arirequest"
	db "gitlab.com/voipbin/bin-manager/call-manager/pkg/db_handler"
	rabbitmq "gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
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

	processEvent(m []byte) error

	eventHandlerStasisStart(ctx context.Context, evt interface{}) error
	eventHandlerChannelCreated(ctx context.Context, evt interface{}) error
}

type eventHandler struct {
	db         db.DBHandler
	rabbitSock rabbitmq.Rabbit

	reqHandler arirequest.RequestHandler
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
func NewEventHandler(sock rabbitmq.Rabbit, db db.DBHandler) EventHandler {
	evtHandler := &eventHandler{
		rabbitSock: sock,
		db:         db,
	}
	evtHandler.reqHandler = arirequest.NewRequestHandler(sock)
	evtHandler.svcHandler = svchandler.NewServiceHandler(evtHandler.reqHandler, db)
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
	h.rabbitSock.ConsumeMessage(queue, receiver, h.processEvent)
	return nil
}

// processEvent processes received ARI event
func (h *eventHandler) processEvent(m []byte) error {
	// parse
	event, evt, err := ari.Parse(m)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"asterisk_id": event.AsteriskID,
		"type":        event.Type,
	}).Debugf("Received ARI event. message: %s", m)
	promARIEventTotal.WithLabelValues(event.Type, event.AsteriskID).Inc()

	// processMap maps ARIEvent name and event handler.
	var processMap = map[string]func(context.Context, interface{}) error{
		"ChannelCreated":   h.eventHandlerChannelCreated,
		"ChannelDestroyed": h.eventHandlerChannelDestroyed,
		"StasisStart":      h.eventHandlerStasisStart,
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

	promARIProcessTime.WithLabelValues(event.AsteriskID, event.Type).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		return err
	}

	return nil
}

// getTech returns tech from channel name
func getTech(name string) string {
	res := strings.Split(name, "/")
	if len(res) < 1 {
		return ""
	}

	return strings.ToLower(res[0])
}

// getTS convert ARI's timestamp to datetime(6) acceptable timestamp
func getTS(timestamp string) string {
	return strings.TrimSuffix(timestamp, "+0000")
}
