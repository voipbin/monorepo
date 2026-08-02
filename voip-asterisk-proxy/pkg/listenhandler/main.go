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
	Run() (<-chan error, error)
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

// Run declares all listen queues (permanent + volatile) synchronously and,
// on success, starts one consumer goroutine per queue. It returns an error
// immediately if queue-name validation or queue declaration fails (no
// consumer is started in that case). On success it returns a channel on
// which each consumer goroutine reports its own (non-nil) ConsumeRPC error,
// buffered to the number of listen queues so no goroutine blocks on send.
func (h *listenHandler) Run() (<-chan error, error) {
	log := logrus.WithField("func", "Run")

	permQueues, volQueues, err := h.buildListenQueues()
	if err != nil {
		return nil, fmt.Errorf("could not build the listen queue list. err: %v", err)
	}

	// queue declare for permanent
	for _, queue := range permQueues {
		log.Debugf("Declaring permanent request queue. queue: %s", queue)

		if errCreate := h.sockHandler.QueueCreate(queue, "normal"); errCreate != nil {
			return nil, fmt.Errorf("could not declare the permanent queue for listenHandler. queue: %s, err: %v", queue, errCreate)
		}
	}

	// queue declare for volatile
	for _, queue := range volQueues {
		log.Debugf("Declaring volatile request queue. queue: %s", queue)

		if errCreate := h.sockHandler.QueueCreate(queue, "volatile"); errCreate != nil {
			return nil, fmt.Errorf("could not declare the volatile queue for listenHandler. queue: %s, err: %v", queue, errCreate)
		}
	}

	// all queues are declared successfully. start a consumer per queue.
	listenQueues := append(append([]string{}, permQueues...), volQueues...)
	chErr := make(chan error, len(listenQueues))
	for _, listenQueue := range listenQueues {
		log.Infof("Running the request listener. queue: %s", listenQueue)
		go func(queue string) {
			errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, "asterisk-proxy", false, false, false, 10, h.processRequest)
			if errConsume != nil {
				log.Errorf("Could not handle the request message correctly. err: %v", errConsume)
				chErr <- errConsume
			}
		}(listenQueue)
	}

	return chErr, nil
}

// buildListenQueues splits the permanent and volatile listen queue name
// configuration and validates the combined list: no empty elements, no
// duplicate elements (either within a list or across the two).
func (h *listenHandler) buildListenQueues() (permQueues []string, volQueues []string, err error) {
	permQueues = strings.Split(h.rabbitQueueListenRequestPermanent, ",")
	volQueues = strings.Split(h.rabbitQueueListenRequestVolatile, ",")

	listenQueues := append(append([]string{}, permQueues...), volQueues...)

	seen := make(map[string]bool, len(listenQueues))
	for _, queue := range listenQueues {
		if queue == "" {
			return nil, nil, fmt.Errorf("listen queue name must not be empty")
		}

		if seen[queue] {
			return nil, nil, fmt.Errorf("duplicate listen queue name: %s", queue)
		}
		seen[queue] = true
	}

	return permQueues, volQueues, nil
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
