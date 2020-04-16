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
	handleStasisStartInVoipbin(m *ari.StasisStart) error
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
	context := e.Args["CONTEXT"]

	switch context {
	case "in-voipbin":
		return h.handleStasisStartInVoipbin(e)
	}

	return nil
}

// handleStasisStartInVoipbin handles in-voipbin type of stasisstart event
func (h *eventHandler) handleStasisStartInVoipbin(m *ari.StasisStart) error {

	log.WithFields(log.Fields{
		"context":    m.Args["CONTEXT"],
		"doamin":     m.Args["DOMAIN"],
		"channel_id": m.Channel.ID,
	}).Debugf("Executing in-voip context handler.")

	domain := m.Args["DOMAIN"]
	switch domain {
	case "echo.voipbin.net":
		return handleServiceEcho(h.reqHandler, m)
	}

	return nil
}

// handleServiceEcho handles echo service.
func handleServiceEcho(h RequestHandler, m *ari.StasisStart) error {
	// answer
	if err := h.ChannelAnswer(m.AsteriskID, m.Channel.ID); err != nil {
		return err
	}

	// set timeout for 180 sec
	if err := h.ChannelVariableSet(m.AsteriskID, m.Channel.ID, "TIMEOUT(absolute)", "180"); err != nil {
		return err
	}

	// continue to svc-echo
	if err := h.ChannelContinue(m.AsteriskID, m.Channel.ID, "svc-echo", m.Channel.Dialplan.Exten, 1, ""); err != nil {
		return err
	}

	return nil
}
