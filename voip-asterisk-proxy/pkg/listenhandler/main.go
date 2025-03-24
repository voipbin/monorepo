package listenhandler

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/voip-asterisk-proxy/pkg/servicehandler"

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

	serviceHandler servicehandler.ServiceHandler
}

var (
	regAMI = regexp.MustCompile(`^/ami/`)
	regARI = regexp.MustCompile(`^/ari/`)

	// proxy
	regProxyRecordingFileMove = regexp.MustCompile(`^/proxy/recording_file_move$`)
)

// simpleResponse returns simple rabbitmq response
func simpleResponse(code int) *sock.Response {
	return &sock.Response{
		StatusCode: code,
	}
}

// NewListenHandler returns ListenHandler interface object
func NewListenHandler(
	sockHandler sockhandler.SockHandler,

	rabbitQueueListenRequestPermanent string,
	rabbitQueueListenRequestVolatile string,

	ariAddr string,
	ariAccount string,
	amiSock *amigo.Amigo,

	serviceHandler servicehandler.ServiceHandler,
) ListenHandler {

	handler := &listenHandler{
		sockHandler:                       sockHandler,
		rabbitQueueListenRequestPermanent: rabbitQueueListenRequestPermanent,
		rabbitQueueListenRequestVolatile:  rabbitQueueListenRequestVolatile,

		ariAddr:    ariAddr,
		ariAccount: ariAccount,

		amiSock: amiSock,

		serviceHandler: serviceHandler,
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
	log := logrus.WithField("func", "listenRun")

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
		log.Debugf("Declaring permenant request queue. queue: %s", queue)

		if err := h.sockHandler.QueueCreate(queue, "volatile"); err != nil {
			return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
		}

		// append it to liesten queue
		listenQueues = append(listenQueues, queue)
	}

	// listening the queue
	for _, listenQueue := range listenQueues {
		log.Infof("Running the request listener. queue: %s", listenQueue)
		go func(queue string) {
			if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, "asterisk-proxy", false, false, false, 10, h.processRequest); errConsume != nil {
				log.Errorf("Could not handle the request message correctly. err: %v", errConsume)
			}
		}(listenQueue)
	}

	return nil
}

func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processRequest",
		"request": m,
	})

	var response *sock.Response
	var err error
	switch {
	case regARI.MatchString(m.URI):
		return h.listenHandlerARI(m)

	case regAMI.MatchString(m.URI):
		return h.listenHandlerAMI(m)

	// POST /proxy/recording_file_move
	case regProxyRecordingFileMove.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processProxyRecordingFileMovePost(context.Background(), m)

	default:
		log.Errorf("Could not find corresponded message handler. method: %s, uri: %s", m.Method, m.URI)
		response = simpleResponse(404)
		err = nil
	}

	if err != nil {
		log.Errorf("Could not handle the request message correctly. method: %s, uri: %s, err: %v", m.Method, m.URI, err)
		response = simpleResponse(400)
		err = nil
	}

	return response, err
}
