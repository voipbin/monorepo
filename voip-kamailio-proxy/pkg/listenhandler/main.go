package listenhandler

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/voip-kamailio-proxy/pkg/siphandler"

	"github.com/sirupsen/logrus"
)

// ListenHandler interface type
type ListenHandler interface {
	Run() error
}

type listenHandler struct {
	sockHandler       sockhandler.SockHandler
	rabbitQueueListen string
	sipTimeout        time.Duration
	sipChecker        siphandler.SIPChecker
}

var (
	regV1ProvidersHealth = regexp.MustCompile(`^/v1/providers/health$`)
)

// simpleResponse returns a simple RabbitMQ response with the given status code.
func simpleResponse(code int) *sock.Response {
	return &sock.Response{
		StatusCode: code,
	}
}

// NewListenHandler returns a ListenHandler.
func NewListenHandler(
	sockHandler sockhandler.SockHandler,
	rabbitQueueListen string,
	sipTimeout time.Duration,
	sipChecker siphandler.SIPChecker,
) ListenHandler {
	return &listenHandler{
		sockHandler:       sockHandler,
		rabbitQueueListen: rabbitQueueListen,
		sipTimeout:        sipTimeout,
		sipChecker:        sipChecker,
	}
}

// Run starts the listen handler in a background goroutine with automatic restart on failure.
func (h *listenHandler) Run() error {
	log := logrus.WithField("func", "Run")

	go func() {
		for {
			if err := h.listenRun(); err != nil {
				log.Errorf("Listen handler exited with error, restarting in 5s: %v", err)
			} else {
				log.Warn("Listen handler exited cleanly, restarting in 1s.")
			}
			time.Sleep(5 * time.Second)
		}
	}()

	return nil
}

// listenRun creates the queue and starts consuming RPC messages.
func (h *listenHandler) listenRun() error {
	log := logrus.WithField("func", "listenRun")

	if err := h.sockHandler.QueueCreate(h.rabbitQueueListen, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	log.Infof("Running the request listener. queue: %s", h.rabbitQueueListen)
	if err := h.sockHandler.ConsumeRPC(
		context.Background(),
		h.rabbitQueueListen,
		"kamailio-proxy",
		false, false, false,
		10,
		h.processRequest,
	); err != nil {
		return fmt.Errorf("could not consume RPC: %v", err)
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
	// POST /v1/providers/health
	case regV1ProvidersHealth.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ProvidersHealthPost(context.Background(), m)

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
