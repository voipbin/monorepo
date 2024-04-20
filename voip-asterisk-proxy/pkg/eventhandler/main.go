package eventhandler

import (
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gorilla/websocket"
	"github.com/ivahaev/amigo"
	"github.com/sirupsen/logrus"
)

// EventHandler interface type
type EventHandler interface {
	Run() error
}

type eventHandler struct {
	notifyhandler notifyhandler.NotifyHandler

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
	notifyHandler notifyhandler.NotifyHandler,
	rabbitSock rabbitmqhandler.Rabbit,
	rabbitQueuePublishEvents string,
	ariAddr string,
	ariAccount string,
	ariSubscribeAll string,
	ariApplication string,
	amiSock *amigo.Amigo,
	amiEventFilter []string,
) EventHandler {
	handler := &eventHandler{
		notifyhandler: notifyHandler,

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
	log := logrus.New().WithField("func", "Run")
	go func() {
		if err := h.eventARIRun(); err != nil {
			log.Errorf("Could not run eventARIRun correctly. err: %v", err)
		}
	}()

	return nil
}
