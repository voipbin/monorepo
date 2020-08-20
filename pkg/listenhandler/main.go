package listenhandler

import (
	"fmt"
	"strings"
	"time"

	"github.com/ivahaev/amigo"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/voip/asterisk-proxy/pkg/rabbitmq"
)

// ListenHandler interface type
type ListenHandler interface {
	Run() error
}

type listenHandler struct {
	// rabbitmq settings
	rabbitSock               rabbitmq.Rabbit
	rabbitQueueListenRequest string

	// ari settings
	ariAddr    string
	ariAccount string

	// ami settings
	amiSock *amigo.Amigo
}

// NewListenHandler returns ListenHandler interface object
func NewListenHandler(
	rabbitSock rabbitmq.Rabbit, rabbitQueueListenRequest string,
	ariAddr, ariAccount string,
	amiSock *amigo.Amigo,
) ListenHandler {

	handler := &listenHandler{
		rabbitSock:               rabbitSock,
		rabbitQueueListenRequest: rabbitQueueListenRequest,

		ariAddr:    ariAddr,
		ariAccount: ariAccount,

		amiSock: amiSock,
	}

	return handler
}

// Run runs the listen handler.
func (h *listenHandler) Run() error {

	go h.listenRun()

	return nil
}

func (h *listenHandler) listenRun() error {

	queues := strings.Split(h.rabbitQueueListenRequest, ",")
	for _, queue := range queues {
		logrus.Debugf("Declaring request queue. queue: %s, sock: %v", queue, h.rabbitSock)
		if err := h.rabbitSock.QueueDeclare(queue, false, false, false, false); err != nil {
			logrus.WithFields(
				logrus.Fields{
					"queue": queue,
				}).Errorf("Could not declare the queue. err: %v", err)
			return err
		}

		go func(sock rabbitmq.Rabbit, name string) {
			for {
				sock.ConsumeRPC(name, "", h.listenHandler)
				time.Sleep(time.Second * 1)
			}
		}(h.rabbitSock, queue)
	}

	return nil
}

func (h *listenHandler) listenHandler(request *rabbitmq.Request) (*rabbitmq.Response, error) {

	switch request.URI[0:4] {
	case "/ari":
		return h.listenHandlerARI(request)

	case "/ami":
		return h.listenHandlerAMI(request)

	default:
		logrus.Errorf("Could not find correct listen handler. request: %v", request)
		return nil, fmt.Errorf("no handler found")
	}
}
