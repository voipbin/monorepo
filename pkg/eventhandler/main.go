package eventhandler

import (
	"github.com/gorilla/websocket"
	"github.com/ivahaev/amigo"

	"gitlab.com/voipbin/voip/asterisk-proxy/pkg/rabbitmq"
)

// EventHandler interface type
type EventHandler interface {
	Run() error
}

type eventHandler struct {
	// rabbitmq settings
	rabbitSock               rabbitmq.Rabbit
	rabbitQueueListenRequest string
	rabbitQueuePublishEvent  string

	// ari settings
	ariAddr         string // ari target address
	ariAccount      string // ari account
	ariSubscribeAll string // ari subsrcibe all option
	ariApplication  string // ari application
	ariSock         *websocket.Conn

	// ami settings
	amiSock        *amigo.Amigo // ami sock
	amiEventFilter []string
}

func init() {
	// do nothing
}

// NewEventHandler returns eventhandler
func NewEventHandler(
	rabbitSock rabbitmq.Rabbit, rabbitQueueListenRequest string, rabbitQueuePublishEvents string,
	ariAddr string, ariAccount string, ariSubscribeAll string, ariApplication string,
	amiSock *amigo.Amigo, amiEventFilter []string,
) EventHandler {
	handler := &eventHandler{
		rabbitSock:               rabbitSock,
		rabbitQueueListenRequest: rabbitQueueListenRequest,
		rabbitQueuePublishEvent:  rabbitQueuePublishEvents,

		ariAddr:         ariAddr,
		ariAccount:      ariAccount,
		ariSubscribeAll: ariSubscribeAll,
		ariApplication:  ariApplication,

		amiSock:        amiSock,
		amiEventFilter: amiEventFilter,
	}

	return handler
}

func (h *eventHandler) Run() error {
	go h.eventARIRun()

	return nil
}
