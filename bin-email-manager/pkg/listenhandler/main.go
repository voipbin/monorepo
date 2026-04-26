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
	"monorepo/bin-email-manager/pkg/dbhandler"
	"monorepo/bin-email-manager/pkg/emailhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
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
	utilHandler utilhandler.UtilHandler
	sockHandler sockhandler.SockHandler

	emailHandler emailhandler.EmailHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// emails
	regV1EmailsGet = regexp.MustCompile(`/v1/emails\?`)
	regV1Emails    = regexp.MustCompile("/v1/emails$")
	regV1EmailsID  = regexp.MustCompile("/v1/emails/" + regUUID + "$")

	// hooks
	regV1Hooks = regexp.MustCompile(`/v1/hooks$`)
)

var (
	metricsNamespace = "flow_manager"

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
	emailHandler emailhandler.EmailHandler,
) ListenHandler {
	h := &listenHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		sockHandler: sockHandler,

		emailHandler: emailHandler,
	}

	return h
}

func (h *listenHandler) Run(queue, exchangeDelay string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Run",
		"queue":          queue,
		"exchange_delay": exchangeDelay,
	})
	log.Info("Creating rabbitmq queue for listen.")

	if err := h.sockHandler.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// process the received request
	go func() {
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, "flow-manager", false, false, false, 10, h.processRequest); errConsume != nil {
			log.Errorf("Could not consume the request message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processRequest",
		"request": m,
	})

	ctx := context.Background()

	var requestType string
	var err error
	var response *sock.Response

	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))
	}()

	switch {

	// v1
	// emails
	case regV1EmailsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/emails"
		response, err = h.v1EmailsGet(ctx, m)

	case regV1Emails.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/emails"
		response, err = h.v1EmailsPost(ctx, m)

	// emails/<email-id>
	case regV1EmailsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/emails/<email-id>"
		response, err = h.v1EmailsIDGet(ctx, m)

	case regV1EmailsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/emails/<email-id>"
		response, err = h.v1EmailsIDDelete(ctx, m)

	// POST /hooks
	case regV1Hooks.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1HooksPost(ctx, m)
		requestType = "/v1/hooks"

	default:
		log.Errorf("Could not find corresponded request handler. data: %s", m.Data)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}

	// default error handler — typed errors and ErrNotFound flow through
	// errorResponse; other errors keep legacy 400.
	if err != nil {
		log.Errorf("Could not process the request correctly. err: %v", err)
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
