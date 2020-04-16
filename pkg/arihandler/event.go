package arihandler

//go:generate mockgen -destination ./mock_arihandler_eventhandler.go -package arihandler gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler EventHandler

import (
	"time"

	log "github.com/sirupsen/logrus"

	ari "gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	rabbitmq "gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
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
	SetSock(sock rabbitmq.Rabbit)

	eventHandlerStasisStart(evt interface{}) error
	handleStasisStartIncoming(m *ari.StasisStart) error
}

type eventHandler struct {
	rabbitSock rabbitmq.Rabbit

	reqHandler RequestHandler
}

// NewEventHandler create EventHandler
func NewEventHandler() EventHandler {
	evtHandler := &eventHandler{}
	evtHandler.reqHandler = NewRequestHandler()

	return evtHandler
}

// SetSock sets amqp sock
func (h *eventHandler) SetSock(sock rabbitmq.Rabbit) {
	h.rabbitSock = sock

	h.reqHandler.SetSock(sock)
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
	log.Debugf("Event recevied. event: %s", m)

	event, evt, err := ari.Parse(m)
	if err != nil {
		return err
	}
	promARIEventTotal.WithLabelValues(event.Type, event.AsteriskID).Inc()

	// processMap maps ARIEvent name and event handler.
	var processMap = map[string]func(interface{}) error{
		"StasisStart": h.eventHandlerStasisStart,
	}

	handler := processMap[event.Type]
	if handler == nil {
		// no handler
		return nil
	}

	return handler(evt)
}

// eventHandlerStasisStart handles StasisStart ARI event
func (h *eventHandler) eventHandlerStasisStart(evt interface{}) error {
	e := evt.(*ari.StasisStart)

	if e.Args["CONTEXT"] == "in-voipbin" {
		return h.handleStasisStartIncoming(e)
	}

	return nil
}

func (h *eventHandler) handleStasisStartIncoming(m *ari.StasisStart) error {
	log.Debugf("handleStasisStartIncoming started.")
	if m.Args["DOMAIN"] == "echo.voipbin.net" {

		// answer
		if err := h.reqHandler.ChannelAnswer(m.AsteriskID, m.Channel.ID); err != nil {
			return err
		}

		// set timeout for 180 sec
		if err := h.reqHandler.ChannelVariableSet(m.AsteriskID, m.Channel.ID, "TIMEOUT(absolute)", "180"); err != nil {
			return err
		}

		// send it to svc-echo
		log.WithFields(
			log.Fields{
				"asterisk_id": m.AsteriskID,
				"channel":     m.Channel.ID,
			}).Debugf("Sending echo service")
		err := h.reqHandler.ChannelContinue(m.AsteriskID, m.Channel.ID, "svc-echo", m.Channel.Dialplan.Exten, 1, "")
		if err != nil {
			return err
		}
	}

	return nil
}
