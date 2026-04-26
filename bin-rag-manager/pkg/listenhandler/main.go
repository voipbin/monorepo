package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-rag-manager/pkg/dbhandler"
	"monorepo/bin-rag-manager/pkg/raghandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	PageSize  = "page_size"
	PageToken = "page_token"
)

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// rag routes
	regV1Rags          = regexp.MustCompile(`^/v1/rags(\?.*)?$`)
	regV1RagsID        = regexp.MustCompile(`^/v1/rags/` + regUUID + `(\?.*)?$`)
	regV1RagsIDSources   = regexp.MustCompile(`^/v1/rags/` + regUUID + `/sources(\?.*)?$`)
	regV1RagsIDSourcesID = regexp.MustCompile(`^/v1/rags/` + regUUID + `/sources/` + regUUID + `(\?.*)?$`)

	// query route
	regV1Query = regexp.MustCompile(`^/v1/query$`)
)

// ListenHandler interface for rag-manager listen handler
type ListenHandler interface {
	Run(queue, exchangeDelay string) error
}

type listenHandler struct {
	sockHandler sockhandler.SockHandler
	ragHandler  raghandler.RagHandler
}

var promReceivedRequestProcessTime = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "bin_rag_manager",
		Name:      "receive_request_process_time",
		Help:      "Process time of received requests",
		Buckets:   []float64{50, 100, 500, 1000, 3000, 5000, 10000},
	},
	[]string{"type", "method"},
)

func init() {
	prometheus.MustRegister(promReceivedRequestProcessTime)
}

// NewListenHandler creates a new ListenHandler
func NewListenHandler(sockHandler sockhandler.SockHandler, ragHandler raghandler.RagHandler) ListenHandler {
	return &listenHandler{
		sockHandler: sockHandler,
		ragHandler:  ragHandler,
	}
}

// Run starts listening for RPC requests
func (h *listenHandler) Run(queue, exchangeDelay string) error {
	log := logrus.WithField("func", "Run")

	if err := h.sockHandler.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not create queue: %w", err)
	}

	go func() {
		if err := h.sockHandler.ConsumeRPC(context.Background(), queue, "rag-manager", false, false, false, 10, h.processRequest); err != nil {
			log.Errorf("Could not consume RPC. err: %v", err)
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "processRequest",
		"uri":    m.URI,
		"method": m.Method,
	})
	log.Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)

	ctx := context.Background()
	start := time.Now()
	var requestType string
	var response *sock.Response
	var err error

	switch {
	// rag routes — sources routes MUST come before regV1RagsID (more specific route first)
	case regV1RagsIDSourcesID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1RagsIDSourcesIDDelete(ctx, m)
		requestType = "/v1/rags/<rag-id>/sources/<source-id>"

	case regV1RagsIDSources.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1RagsIDSourcesPost(ctx, m)
		requestType = "/v1/rags/<rag-id>/sources"

	// rag routes — ID routes before collection routes
	case regV1RagsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1RagsIDGet(ctx, m)
		requestType = "/v1/rags/<rag-id>"

	case regV1RagsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1RagsIDDelete(ctx, m)
		requestType = "/v1/rags/<rag-id>"

	case regV1RagsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1RagsIDPut(ctx, m)
		requestType = "/v1/rags/<rag-id>"

	case regV1Rags.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1RagsPost(ctx, m)
		requestType = "/v1/rags"

	case regV1Rags.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1RagsGet(ctx, m)
		requestType = "/v1/rags"

	// query route
	case regV1Query.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1QueryPost(ctx, m)
		requestType = "/v1/query"

	default:
		log.Errorf("Could not find the handler. method: %s, uri: %s", m.Method, m.URI)
		response = simpleResponse(404)
		requestType = "notfound"
	}

	// default error handler — typed errors and ErrNotFound flow through
	// errorResponse; other errors keep legacy 500.
	if err != nil {
		log.Errorf("Could not process request. err: %v", err)
		var ve *cerrors.VoipbinError
		switch {
		case stderrors.As(err, &ve):
			response = errorResponse(err)
		case stderrors.Is(err, dbhandler.ErrNotFound):
			response = errorResponse(err)
		default:
			response = simpleResponse(500)
		}
	}

	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	return response, nil
}

func simpleResponse(statusCode int) *sock.Response {
	return &sock.Response{
		StatusCode: statusCode,
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

func jsonResponse(statusCode int, data any) *sock.Response {
	m, err := json.Marshal(data)
	if err != nil {
		return simpleResponse(500)
	}
	return &sock.Response{
		StatusCode: statusCode,
		DataType:   "application/json",
		Data:       m,
	}
}
