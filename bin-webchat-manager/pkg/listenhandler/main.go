package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

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

	"monorepo/bin-webchat-manager/pkg/dbhandler"
	"monorepo/bin-webchat-manager/pkg/messagehandler"
	"monorepo/bin-webchat-manager/pkg/sessionhandler"
	"monorepo/bin-webchat-manager/pkg/widgethandler"
)

// pagination parameters
const (
	PageSize  = "page_size"
	PageToken = "page_token"
)

// ListenHandler interface
type ListenHandler interface {
	Run(queue, exchangeDelay string) error
}

type listenHandler struct {
	utilHanlder utilhandler.UtilHandler
	sockHandler sockhandler.SockHandler

	widgetHandler  widgethandler.WidgetHandler
	sessionHandler sessionhandler.SessionHandler
	messageHandler messagehandler.MessageHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}" //nolint:deadcode,unused,varcheck // this is ok
	regAny  = "(.*)"                                                        //nolint:deadcode,unused,varcheck // this is ok

	// v1
	// widgets
	regV1WidgetsGet                  = regexp.MustCompile(`/v1/widgets\?` + regAny + "$")
	regV1Widgets                     = regexp.MustCompile("/v1/widgets$")
	reqV1WidgetsID                   = regexp.MustCompile("/v1/widgets/" + regUUID + "$")
	reqV1WidgetsIDDirectHashRegenerate = regexp.MustCompile("/v1/widgets/" + regUUID + "/direct-hash-regenerate$")

	// sessions
	regV1SessionsGet   = regexp.MustCompile(`/v1/sessions\?` + regAny + "$")
	regV1Sessions      = regexp.MustCompile("/v1/sessions$")
	reqV1SessionsID    = regexp.MustCompile("/v1/sessions/" + regUUID + "$")
	reqV1SessionsIDEnd = regexp.MustCompile("/v1/sessions/" + regUUID + "/end$")

	// messages
	regV1MessagesGet = regexp.MustCompile(`/v1/messages\?` + regAny + "$")
	regV1Messages    = regexp.MustCompile("/v1/messages$")
	reqV1MessagesID  = regexp.MustCompile("/v1/messages/" + regUUID + "$")
)

var (
	metricsNamespace = "webchat_manager"

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

// errorResponse maps a business-handler error to the appropriate sock.Response.
// Resolution order: typed *cerrors.VoipbinError → ToResponse; legacy
// dbhandler.ErrNotFound → 404; else → 500.
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

// NewListenHandler return ListenHandler interface
func NewListenHandler(
	sockHandler sockhandler.SockHandler,
	widgetHandler widgethandler.WidgetHandler,
	sessionHandler sessionhandler.SessionHandler,
	messageHandler messagehandler.MessageHandler,
) ListenHandler {
	h := &listenHandler{
		utilHanlder: utilhandler.NewUtilHandler(),
		sockHandler: sockHandler,

		widgetHandler:  widgetHandler,
		sessionHandler: sessionHandler,
		messageHandler: messageHandler,
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
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, "webchat-manager", false, false, false, 10, h.processRequest); errConsume != nil {
			logrus.Errorf("Could not consume the request message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

// processRequest handles received request
func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
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

	// ////////////
	// // widgets
	// ////////////
	// GET /widgets
	case regV1WidgetsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1WidgetsGet(ctx, m)
		requestType = "/v1/widgets"

	// POST /widgets
	case regV1Widgets.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1WidgetsPost(ctx, m)
		requestType = "/v1/widgets"

	// /widgets/<widget-id>/
	// GET /widgets/<widget-id>
	case reqV1WidgetsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1WidgetsIDGet(ctx, m)
		requestType = "/v1/widgets/<widget-id>/"

	// DELETE /widgets/<widget-id>
	case reqV1WidgetsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1WidgetsIDDelete(ctx, m)
		requestType = "/v1/widgets/<widget-id>/"

	// PUT /widgets/<widget-id>
	case reqV1WidgetsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1WidgetsIDPut(ctx, m)
		requestType = "/v1/widgets/<widget-id>/"

	// POST /widgets/<widget-id>/direct-hash-regenerate
	case reqV1WidgetsIDDirectHashRegenerate.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1WidgetsIDDirectHashRegeneratePost(ctx, m)
		requestType = "/v1/widgets/<widget-id>/direct-hash-regenerate"

	/////////////
	// sessions
	/////////////

	// GET /sessions
	case regV1SessionsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1SessionsGet(ctx, m)
		requestType = "/v1/sessions"

	// POST /sessions
	case regV1Sessions.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1SessionsPost(ctx, m)
		requestType = "/v1/sessions"

	// GET /sessions/<session-id>
	case reqV1SessionsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1SessionsIDGet(ctx, m)
		requestType = "/v1/sessions/<session-id>/"

	// DELETE /sessions/<session-id>
	case reqV1SessionsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1SessionsIDDelete(ctx, m)
		requestType = "/v1/sessions/<session-id>/"

	// POST /sessions/<session-id>/end
	case reqV1SessionsIDEnd.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1SessionsIDEndPost(ctx, m)
		requestType = "/v1/sessions/<session-id>/end"

	/////////////
	// messages
	/////////////

	// GET /messages
	case regV1MessagesGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1MessagesGet(ctx, m)
		requestType = "/v1/messages"

	// POST /messages
	case regV1Messages.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1MessagesPost(ctx, m)
		requestType = "/v1/messages"

	// GET /messages/<message-id>
	case reqV1MessagesID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1MessagesIDGet(ctx, m)
		requestType = "/v1/messages/<message-id>/"

	// DELETE /messages/<message-id>
	case reqV1MessagesID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1MessagesIDDelete(ctx, m)
		requestType = "/v1/messages/<message-id>/"

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		log.Errorf("Could not find corresponded message handler. method: %s, uri: %s", m.Method, m.URI)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}
	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	// default error handler — typed errors and ErrNotFound flow through
	// errorResponse; other errors keep legacy 400.
	if err != nil {
		log.Errorf("Could not handle the message correctly. method: %s, uri: %s", m.Method, m.URI)
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

	log.WithField("response", response).Debugf("Sending response. method: %s, uri: %s", m.Method, m.URI)

	return response, err
}
