package listenhandler

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

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
func simpleResponse(code int) *sock.Response {
	return &sock.Response{
		StatusCode: code,
	}
}

type listenHandler struct {
	rabbitSock rabbitmqhandler.Rabbit

	utilHandler         utilhandler.UtilHandler
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

		utilHandler:         utilhandler.NewUtilHandler(),
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

	if err := h.rabbitSock.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// receive requests
	go func() {
		for {
			// consume the request
			err := h.rabbitSock.ConsumeRPC(queue, constCosumerName, false, false, false, 10, h.processRequest)
			if err != nil {
				logrus.Errorf("Could not consume the request message correctly. err: %v", err)
			}
		}
	}()

	return nil
}

// processRequest handles all of requests of the listen queue.
func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {

	var requestType string
	var err error
	var response *sock.Response

	ctx := context.Background()
	logrus.WithFields(
		logrus.Fields{
			"uri":       m.URI,
			"method":    m.Method,
			"data_type": m.DataType,
			"data":      m.Data,
		}).Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	////////////////////
	// accounts
	////////////////////
	// GET /accounts
	case regV1AccountsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1AccountsGet(ctx, m)
		requestType = "/v1/accounts"

	// POST /accounts
	case regV1Accounts.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1AccountsPost(ctx, m)
		requestType = "/v1/accounts"

	// GET /accounts/<account-id>
	case regV1AccountsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1AccountsIDGet(ctx, m)
		requestType = "/v1/accounts/<account-id>"

	// PUT /accounts/<account-id>
	case regV1AccountsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1AccountsIDPut(ctx, m)
		requestType = "/v1/accounts/<account-id>"

	// DELETE /accounts/<account-id>
	case regV1AccountsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1AccountsIDDelete(ctx, m)
		requestType = "/v1/accounts/<account-id>"

	////////////////////
	// conversations
	////////////////////
	// GET /conversations
	case regV1ConversationsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ConversationsGet(ctx, m)
		requestType = "/v1/conversations"

	// GET /conversations/<conversation-id>
	case regV1ConversationsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ConversationsIDGet(ctx, m)
		requestType = "/v1/conversations/<conversation-id>"

	// PUT /conversations/<conversation-id>
	case regV1ConversationsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1ConversationsIDPut(ctx, m)
		requestType = "/v1/conversations/<conversation-id>"

	// GET /conversations/<conversation-id>/messages
	case regV1ConversationsIDMessagesGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ConversationsIDMessagesGet(ctx, m)
		requestType = "/v1/conversations/<conversation-id>/messages"

	// POST /conversations/<conversation-id>/messages
	case regV1ConversationsIDMessages.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConversationsIDMessagesPost(ctx, m)
		requestType = "/v1/conversations/<conversation-id>/messages"

	////////////////////
	// hooks
	////////////////////
	// POST /hooks
	case regV1Hooks.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1HooksPost(ctx, m)
		requestType = "/v1/hooks"

	////////////////////
	// messages
	////////////////////
	// GET /messages
	case regV1MessagesGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
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
	).Debugf("Sending response. method: %s, uri: %s", m.Method, m.URI)

	return response, err
}
