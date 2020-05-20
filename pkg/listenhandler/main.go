package listenhandler

import (
	"fmt"
	"regexp"
	"time"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

// ListenHandler interface
type ListenHandler interface {
	Run(queue, exchangeDelay string) error
}

type listenHandler struct {
	rabbitSock rabbitmq.Rabbit
	db         dbhandler.DBHandler

	reqHandler  requesthandler.RequestHandler
	callHandler callhandler.CallHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// v1
	// asterisks
	regV1AsterisksIDChannelsIDHealth = regexp.MustCompile("/v1/asterisks/(.*)/channels/(.*)/health-check")

	// calls
	regV1CallsID              = regexp.MustCompile("/v1/calls/" + regUUID)
	regV1CallsIDHealth        = regexp.MustCompile("/v1/calls/" + regUUID + "/health-check")
	regV1CallsIDActionTimeout = regexp.MustCompile("/v1/calls/" + regUUID + "/action-timeout")
)

// simpleResponse returns simple rabbitmq response
func simpleResponse(code int) *rabbitmq.Response {
	return &rabbitmq.Response{
		StatusCode: code,
	}
}

// NewListenHandler return ListenHandler interface
func NewListenHandler(rabbitSock rabbitmq.Rabbit, db dbhandler.DBHandler, reqHandler requesthandler.RequestHandler, callHandler callhandler.CallHandler) ListenHandler {
	h := &listenHandler{
		rabbitSock:  rabbitSock,
		db:          db,
		reqHandler:  reqHandler,
		callHandler: callHandler,
	}

	return h
}

func (h *listenHandler) Run(queue, exchangeDelay string) error {
	logrus.WithFields(logrus.Fields{
		"queue": queue,
	}).Info("Creating rabbitmq queue for listen.")

	// declare the queue
	if err := h.rabbitSock.QueueDeclare(queue, true, false, false, false); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// create a exchange for delayed message
	if err := h.rabbitSock.ExchangeDeclareForDelay(exchangeDelay, true, false, false, false); err != nil {
		return fmt.Errorf("Could not declare the exchange for dealyed message. err: %v", err)
	}

	// bind a queue with delayed exchange
	if err := h.rabbitSock.QueueBind(queue, queue, exchangeDelay, false, nil); err != nil {
		return fmt.Errorf("Could not bind the queue and exchange. err: %v", err)
	}

	// receive ARI event
	go func() {
		for {
			err := h.rabbitSock.ConsumeRPC(queue, "call-manager", h.processRequest)
			if err != nil {
				logrus.Errorf("Could not consume the ARI message correctly. Will try again after 1 second. err: %v", err)
				time.Sleep(time.Second * 1)
			}
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *rabbitmq.Request) (*rabbitmq.Response, error) {

	switch {
	// v1

	// asterisks
	case regV1AsterisksIDChannelsIDHealth.MatchString(m.URI) == true && m.Method == rabbitmq.RequestMethodPost:
		return h.processV1AsterisksIDChannelsIDHealthPost(m)

	// calls
	case regV1CallsID.MatchString(m.URI) == true && m.Method == rabbitmq.RequestMethodGet:
		return h.processV1CallsIDGet(m)

	case regV1CallsIDHealth.MatchString(m.URI) == true && m.Method == rabbitmq.RequestMethodPost:
		return h.processV1CallsIDHealthPost(m)

	case regV1CallsIDActionTimeout.MatchString(m.URI) == true && m.Method == rabbitmq.RequestMethodPost:
		return h.processV1CallsIDActionTimeoutPost(m)
	}

	logrus.WithFields(
		logrus.Fields{
			"uri":    m.URI,
			"method": m.Method,
		}).Errorf("Could not find corresponded message handler. data: %s", m.Data)
	return simpleResponse(404), nil
}
