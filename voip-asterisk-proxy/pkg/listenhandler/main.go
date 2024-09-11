package listenhandler

import (
	"fmt"
	"strings"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/ivahaev/amigo"
	"github.com/sirupsen/logrus"
)

// ListenHandler interface type
type ListenHandler interface {
	Run() error
}

type listenHandler struct {
	// rabbitmq settings
	sockHandler                       sockhandler.SockHandler // rabbitmq socket
	rabbitQueueListenRequestPermanent string                  // permanent listen queue name for request(asterisk.<asterisk type>.request)
	rabbitQueueListenRequestVolatile  string                  // volatile listen queue name for request(asterisk.<asterisk id>.request)

	// ari settings
	ariAddr    string
	ariAccount string

	// ami settings
	amiSock *amigo.Amigo
}

// NewListenHandler returns ListenHandler interface object
func NewListenHandler(
	sockHandler sockhandler.SockHandler,

	rabbitQueueListenRequestPermanent string,
	rabbitQueueListenRequestVolatile string,

	ariAddr string,
	ariAccount string,
	amiSock *amigo.Amigo,
) ListenHandler {

	handler := &listenHandler{
		sockHandler:                       sockHandler,
		rabbitQueueListenRequestPermanent: rabbitQueueListenRequestPermanent,
		rabbitQueueListenRequestVolatile:  rabbitQueueListenRequestVolatile,

		ariAddr:    ariAddr,
		ariAccount: ariAccount,

		amiSock: amiSock,
	}

	return handler
}

// Run runs the listen handler.
func (h *listenHandler) Run() error {
	log := logrus.New().WithField("func", "Run")

	go func() {
		if err := h.listenRun(); err != nil {
			log.Errorf("Could not exeucte the listen handler: %v", err)
		}
	}()

	return nil
}

// listenRun initiate and start to listening the request from the rabbitmq queue.
func (h *listenHandler) listenRun() error {

	listenQueues := []string{}

	// queue declare for permanent
	permQueues := strings.Split(h.rabbitQueueListenRequestPermanent, ",")
	for _, queue := range permQueues {

		if err := h.sockHandler.QueueCreate(queue, "normal"); err != nil {
			return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
		}

		// append it to listen queue
		listenQueues = append(listenQueues, queue)
	}

	// queue declare for volatile
	volQueues := strings.Split(h.rabbitQueueListenRequestVolatile, ",")
	for _, queue := range volQueues {

		// declare queue
		logrus.Debugf("Declaring permenant request queue. queue: %s", queue)

		if err := h.sockHandler.QueueCreate(queue, "volatile"); err != nil {
			return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
		}

		// append it to liesten queue
		listenQueues = append(listenQueues, queue)
	}

	// listening the queue
	for _, listenQueue := range listenQueues {
		logrus.Infof("Running the request listener. queue: %s", listenQueue)
		go func(queue string) {
			for {
				if err := h.sockHandler.ConsumeRPC(queue, "", false, false, false, 10, h.listenHandler); err != nil {
					logrus.Errorf("Could not handle the request message correctly. err: %v", err)
				}
			}
		}(listenQueue)
	}

	return nil
}

func (h *listenHandler) listenHandler(request *sock.Request) (*sock.Response, error) {

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
