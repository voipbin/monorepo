package listenhandler

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
)

// pagination parameters
const (
	PageSize  = "page_size"
	PageToken = "page_token"
)

const (
	constCosumerName = "conversation-manager"
)

// ListenHandler interface
type ListenHandler interface {
	Run(queue, exchangeDelay string) error
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// v1

	// accounts
	regV1AccountsGet = regexp.MustCompile(`/v1/accounts\?`)
	regV1Accounts    = regexp.MustCompile("/v1/accounts$")
	regV1AccountsID  = regexp.MustCompile("/v1/accounts/" + regUUID + "$")

	// conversations
	regV1ConversationsGet           = regexp.MustCompile(`/v1/conversations\?`)
	regV1ConversationsID            = regexp.MustCompile("/v1/conversations/" + regUUID + "$")
	regV1ConversationsIDMessagesGet = regexp.MustCompile("/v1/conversations/" + regUUID + `/messages\?`)
	regV1ConversationsIDMessages    = regexp.MustCompile("/v1/conversations/" + regUUID + "/messages$")

	// hooks
	regV1Hooks = regexp.MustCompile(`/v1/hooks$`)

	// messages
	regV1MessagesGet = regexp.MustCompile(`/v1/messages\?`)
)

var (
	metricsNamespace = "message_manager"

	promReceivedRequestProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "receive_request_process_time",
			Help:      "Process time of received request",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"type", "method"},
	)
)

func init() {
	prometheus.MustRegister(
		promReceivedRequestProcessTime,
	)
}

// simpleResponse returns simple rabbitmq response
func simpleResponse(code int) *rabbitmqhandler.Response {
	return &rabbitmqhandler.Response{
		StatusCode: code,
	}
}

type listenHandler struct {
	rabbitSock rabbitmqhandler.Rabbit

	accountHandler      accounthandler.AccountHandler
	conversationHandler conversationhandler.ConversationHandler
	messageHandler      messagehandler.MessageHandler
}

// NewListenHandler return ListenHandler interface
func NewListenHandler(
	rabbitSock rabbitmqhandler.Rabbit,
	accountHandler accounthandler.AccountHandler,
	conversationHandler conversationhandler.ConversationHandler,
	messageHandler messagehandler.MessageHandler,
) ListenHandler {
	h := &listenHandler{
		rabbitSock: rabbitSock,

		accountHandler:      accountHandler,
		conversationHandler: conversationHandler,
		messageHandler:      messageHandler,
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

	// Set QoS
	if err := h.rabbitSock.QueueQoS(queue, 1, 0); err != nil {
		logrus.Errorf("could not set the queue's qos. err: %v", err)
		return err
	}

	// create a exchange for delayed message
	if err := h.rabbitSock.ExchangeDeclareForDelay(exchangeDelay, true, false, false, false); err != nil {
		return fmt.Errorf("could not declare the exchange for dealyed message. err: %v", err)
	}

	// bind a queue with delayed exchange
	if err := h.rabbitSock.QueueBind(queue, queue, exchangeDelay, false, nil); err != nil {
		return fmt.Errorf("could not bind the queue and exchange. err: %v", err)
	}

	// receive requests
	go func() {
		for {
			// consume the request
			err := h.rabbitSock.ConsumeRPCOpt(queue, constCosumerName, false, false, false, 10, h.processRequest)
			if err != nil {
				logrus.Errorf("Could not consume the request message correctly. err: %v", err)
			}
		}
	}()

	return nil
}

// processRequest handles all of requests of the listen queue.
func (h *listenHandler) processRequest(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	var requestType string
	var err error
	var response *rabbitmqhandler.Response

	ctx := context.Background()

	uri, err := url.QueryUnescape(m.URI)
	if err != nil {
		uri = "could not unescape uri"
	}
	m.URI = uri

	logrus.WithFields(
		logrus.Fields{
			"uri":       m.URI,
			"method":    m.Method,
			"data_type": m.DataType,
			"data":      m.Data,
		}).Debugf("Received request. method: %s, uri: %s", m.Method, uri)

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	////////////////////
	// accounts
	////////////////////
	// GET /accounts
	case regV1AccountsGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1AccountsGet(ctx, m)
		requestType = "/v1/accounts"

	// POST /accounts
	case regV1Accounts.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1AccountsPost(ctx, m)
		requestType = "/v1/accounts"

	// GET /accounts/<account-id>
	case regV1AccountsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1AccountsIDGet(ctx, m)
		requestType = "/v1/accounts/<account-id>"

	// PUT /accounts/<account-id>
	case regV1AccountsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1AccountsIDPut(ctx, m)
		requestType = "/v1/accounts/<account-id>"

	// DELETE /accounts/<account-id>
	case regV1AccountsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1AccountsIDDelete(ctx, m)
		requestType = "/v1/accounts/<account-id>"

	////////////////////
	// conversations
	////////////////////
	// GET /conversations
	case regV1ConversationsGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1ConversationsGet(ctx, m)
		requestType = "/v1/conversations"

	// GET /conversations/<conversation-id>
	case regV1ConversationsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1ConversationsIDGet(ctx, m)
		requestType = "/v1/conversations/<conversation-id>"

	// PUT /conversations/<conversation-id>
	case regV1ConversationsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1ConversationsIDPut(ctx, m)
		requestType = "/v1/conversations/<conversation-id>"

	// GET /conversations/<conversation-id>/messages
	case regV1ConversationsIDMessagesGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1ConversationsIDMessagesGet(ctx, m)
		requestType = "/v1/conversations/<conversation-id>/messages"

	// POST /conversations/<conversation-id>/messages
	case regV1ConversationsIDMessages.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1ConversationsIDMessagesPost(ctx, m)
		requestType = "/v1/conversations/<conversation-id>/messages"

	////////////////////
	// hooks
	////////////////////
	// POST /hooks
	case regV1Hooks.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1HooksPost(ctx, m)
		requestType = "/v1/hooks"

	////////////////////
	// messages
	////////////////////
	// GET /messages
	case regV1MessagesGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1MessagesGet(ctx, m)
		requestType = "/messages"

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		logrus.WithFields(
			logrus.Fields{
				"uri":    m.URI,
				"method": m.Method,
			}).Errorf("Could not find corresponded message handler. data: %s", m.Data)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}
	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	// default error handler
	if err != nil {
		logrus.WithFields(
			logrus.Fields{
				"uri":    m.URI,
				"method": m.Method,
				"error":  err,
			}).Errorf("Could not process the request correctly. data: %s", m.Data)
		response = simpleResponse(400)
		err = nil
	}

	logrus.WithFields(
		logrus.Fields{
			"response": response,
		},
	).Debugf("Sending response. method: %s, uri: %s", m.Method, uri)

	return response, err
}
