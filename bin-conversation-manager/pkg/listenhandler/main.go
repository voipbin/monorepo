package listenhandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
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
	regV1ConversationsGet                      = regexp.MustCompile(`/v1/conversations\?`)
	regV1Conversations                         = regexp.MustCompile(`/v1/conversations$`)
	regV1ConversationsSelfAndPeer              = regexp.MustCompile(`/v1/conversations/self_and_peer$`)
	regV1ConversationsGetOrCreateBySelfAndPeer = regexp.MustCompile(`/v1/conversations/get_or_create_by_self_and_peer$`)
	regV1ConversationsCreateAndExecuteFlow     = regexp.MustCompile(`/v1/conversations/create_and_execute_flow$`)
	regV1ConversationsID                       = regexp.MustCompile("/v1/conversations/" + regUUID + "$")
	regV1ConversationsIDMetadata               = regexp.MustCompile("/v1/conversations/" + regUUID + "/metadata$")

	// hooks
	regV1Hooks = regexp.MustCompile(`/v1/hooks$`)

	// messages
	regV1MessagesGet    = regexp.MustCompile(`/v1/messages\?`)
	regV1Messages       = regexp.MustCompile(`/v1/messages$`)
	regV1MessagesCreate = regexp.MustCompile(`/v1/messages/create$`)
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
// errorResponse maps a business-handler error to the appropriate sock.Response.
// Resolution order:
//  1. Typed *cerrors.VoipbinError → encoded via cerrors.ToResponse so the
//     api-manager edge recovers domain/reason/message via errors.As over RPC.
//  2. Legacy dbhandler.ErrNotFound (wrapped via pkg/errors) → simpleResponse(404).
//  3. Anything else → simpleResponse(500).
func errorResponse(err error) *sock.Response {
	if err == nil {
		logrus.WithField("func", "errorResponse").Warn("errorResponse called with nil error — likely a caller bug; returning 500")
		return simpleResponse(http.StatusInternalServerError)
	}

	var ve *cerrors.VoipbinError
	if stderrors.As(err, &ve) {
		resp, e := cerrors.ToResponse(ve)
		if e == nil {
			return resp
		}
		logrus.WithField("func", "errorResponse").Errorf("cerrors.ToResponse failed for typed VoipbinError: %v", e)
		return simpleResponse(http.StatusInternalServerError)
	}

	if stderrors.Is(err, dbhandler.ErrNotFound) {
		return simpleResponse(http.StatusNotFound)
	}

	return simpleResponse(http.StatusInternalServerError)
}

func simpleResponse(code int) *sock.Response {
	return &sock.Response{
		StatusCode: code,
	}
}

type listenHandler struct {
	sockHandler sockhandler.SockHandler

	utilHandler         utilhandler.UtilHandler
	accountHandler      accounthandler.AccountHandler
	conversationHandler conversationhandler.ConversationHandler
	messageHandler      messagehandler.MessageHandler
}

// NewListenHandler return ListenHandler interface
func NewListenHandler(
	sockHandler sockhandler.SockHandler,
	accountHandler accounthandler.AccountHandler,
	conversationHandler conversationhandler.ConversationHandler,
	messageHandler messagehandler.MessageHandler,
) ListenHandler {
	h := &listenHandler{
		sockHandler: sockHandler,

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

	if err := h.sockHandler.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// receive requests
	go func() {
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, constCosumerName, false, false, false, 10, h.processRequest); errConsume != nil {
			logrus.Errorf("Could not consume the request message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

// processRequest handles all of requests of the listen queue.
func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processRequest",
		"request": m,
	})

	var requestType string
	var err error
	var response *sock.Response

	ctx := context.Background()

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

	// POST /conversations
	case regV1Conversations.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConversationsPost(ctx, m)
		requestType = "/v1/conversations"

	// GET /conversations/self_and_peer
	case regV1ConversationsSelfAndPeer.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ConversationsSelfAndPeerGet(ctx, m)
		requestType = "/v1/conversations/self_and_peer"

	// POST /conversations/get_or_create_by_self_and_peer
	case regV1ConversationsGetOrCreateBySelfAndPeer.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConversationsGetOrCreateBySelfAndPeerPost(ctx, m)
		requestType = "/v1/conversations/get_or_create_by_self_and_peer"

	// POST /conversations/create_and_execute_flow
	case regV1ConversationsCreateAndExecuteFlow.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConversationsCreateAndExecuteFlowPost(ctx, m)
		requestType = "/v1/conversations/create_and_execute_flow"

	// PUT /conversations/<conversation-id>/metadata (must be checked before
	// the generic regV1ConversationsID PUT route below, since it's a more
	// specific match on the same URI prefix)
	case regV1ConversationsIDMetadata.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1ConversationsIDMetadataPut(ctx, m)
		requestType = "/v1/conversations/<conversation-id>/metadata"

	// GET /conversations/<conversation-id>
	case regV1ConversationsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ConversationsIDGet(ctx, m)
		requestType = "/v1/conversations/<conversation-id>"

	// PUT /conversations/<conversation-id>
	case regV1ConversationsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1ConversationsIDPut(ctx, m)
		requestType = "/v1/conversations/<conversation-id>"

	////////////////////
	// hooks
	////////////////////
	// GET /hooks (Meta hub challenge)
	case regV1Hooks.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1HooksGet(ctx, m)
		requestType = "/v1/hooks-get"

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

	// POST /messages
	case regV1Messages.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1MessagesPost(ctx, m)
		requestType = "/messages"

	// POST /messages/create
	case regV1MessagesCreate.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1MessagesCreatePost(ctx, m)
		requestType = "/messages/create"

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		log.WithFields(
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

	// default error handler — typed *VoipbinError and dbhandler.ErrNotFound
	// flow through errorResponse so the api-manager edge sees the right
	// envelope or 404. Other errors keep the legacy 400.
	if err != nil {
		log.WithFields(
			logrus.Fields{
				"uri":    m.URI,
				"method": m.Method,
				"error":  err,
			}).Errorf("Could not process the request correctly. data: %s", m.Data)
		var ve *cerrors.VoipbinError
		switch {
		case stderrors.As(err, &ve):
			response = errorResponse(err)
		case stderrors.Is(err, dbhandler.ErrNotFound):
			response = errorResponse(err)
		default:
			response = simpleResponse(400)
		}
		err = nil
	}

	return response, err
}
