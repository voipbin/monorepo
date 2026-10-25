package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-number-manager/models/provider"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/numberhandler"
	"monorepo/bin-number-manager/pkg/providerhandler"
)

// pagination parameters
const (
	PageSize  = "page_size"
	PageToken = "page_token"
)

const (
	constCosumerName = "number-manager"
)

// ListenHandler interface
type ListenHandler interface {
	Run(queue, exchangeDelay string) error
}

type listenHandler struct {
	utilHandler utilhandler.UtilHandler
	sockHandler sockhandler.SockHandler

	numberHandler   numberhandler.NumberHandler
	providerHandler providerhandler.ProviderHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// v1

	// available numbers
	regV1AvailableNumbers = regexp.MustCompile("/v1/available_numbers")

	// number providers (admin). Anchored on both ends, so they cannot overlap each other
	regV1NumberProviders         = regexp.MustCompile(`^/v1/number_providers$`)
	regV1NumberProvidersID       = regexp.MustCompile(`^/v1/number_providers/([a-z0-9_]{1,32})$`)
	regV1NumberProvidersIDSecret = regexp.MustCompile(`^/v1/number_providers/([a-z0-9_]{1,32})/secret$`)

	// numbers
	regV1NumbersGet                    = regexp.MustCompile(`/v1/numbers\?`)
	regV1Numbers                       = regexp.MustCompile(`/v1/numbers$`)
	regV1NumbersID                     = regexp.MustCompile("/v1/numbers/" + regUUID + "$")
	regV1NumbersIDFlowIDs              = regexp.MustCompile("/v1/numbers/" + regUUID + "/flow_ids$")
	regV1NumbersIDMetadata             = regexp.MustCompile("/v1/numbers/" + regUUID + "/metadata$")
	regV1NumbersRenew                  = regexp.MustCompile(`/v1/numbers/renew$`)
	regV1NumbersCountVirtualByCustomer = regexp.MustCompile(`/v1/numbers/count_virtual_by_customer$`)
)

var (
	metricsNamespace = "number_manager"

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

	if stderrors.Is(err, provider.ErrInvalidArgument) {
		return simpleResponse(http.StatusBadRequest)
	}

	if stderrors.Is(err, provider.ErrSecretKeyNotConfigured) {
		return simpleResponse(http.StatusServiceUnavailable)
	}

	return simpleResponse(http.StatusInternalServerError)
}

// redactedData is the body that replaces the body of a request that may hold a secret.
var redactedData = json.RawMessage(`"[REDACTED]"`)

// logSafeRequest returns the request to be logged. The body of the number provider requests
// (the secret input in particular) is replaced, so that no log line can hold it. It decides by
// the URI alone, independent of the routing, so an unrouted or mistyped secret request is
// redacted too.
func logSafeRequest(m *sock.Request) *sock.Request {
	uri := strings.ToLower(m.URI)
	if !strings.HasPrefix(uri, "/v1/number_providers") && !strings.HasSuffix(uri, "/secret") {
		return m
	}

	res := *m
	res.Data = redactedData
	return &res
}

// NewListenHandler return ListenHandler interface
func NewListenHandler(sockHandler sockhandler.SockHandler, numberHandler numberhandler.NumberHandler, providerHandler providerhandler.ProviderHandler) ListenHandler {
	h := &listenHandler{
		utilHandler:     utilhandler.NewUtilHandler(),
		sockHandler:     sockHandler,
		numberHandler:   numberHandler,
		providerHandler: providerHandler,
	}

	return h
}

func (h *listenHandler) Run(queue, exchangeDelay string) error {
	logrus.WithFields(logrus.Fields{
		"queue": queue,
	}).Info("Creating rabbitmq queue for listen.")

	if err := h.sockHandler.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %w", err)
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

	var requestType string
	var err error
	var response *sock.Response

	ctx := context.Background()

	// every log line uses logM, never m: the body of a secret request must not be logged
	logM := logSafeRequest(m)

	log := logrus.WithFields(logrus.Fields{
		"func":      "processRequest",
		"uri":       logM.URI,
		"method":    logM.Method,
		"data_type": logM.DataType,
		"data":      logM.Data,
	})
	log.Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	////////////////////
	// available_numbers
	////////////////////
	// GET /available_numbers
	case regV1AvailableNumbers.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1AvailableNumbersGet(ctx, m)
		requestType = "/v1/available_numbers"

	////////////////////
	// number_providers
	////////////////////
	// GET /number_providers
	case regV1NumberProviders.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1NumberProvidersGet(ctx, logM)
		requestType = "/v1/number_providers"

	// GET /number_providers/<name>
	case regV1NumberProvidersID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1NumberProvidersIDGet(ctx, logM, regV1NumberProvidersID.FindStringSubmatch(m.URI))
		requestType = "/v1/number_providers/<name>"

	// PUT /number_providers/<name>
	case regV1NumberProvidersID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1NumberProvidersIDPut(ctx, m, logM, regV1NumberProvidersID.FindStringSubmatch(m.URI))
		requestType = "/v1/number_providers/<name>"

	// PUT /number_providers/<name>/secret
	case regV1NumberProvidersIDSecret.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1NumberProvidersIDSecretPut(ctx, m, logM, regV1NumberProvidersIDSecret.FindStringSubmatch(m.URI))
		requestType = "/v1/number_providers/<name>/secret"

	// DELETE /number_providers/<name>/secret
	case regV1NumberProvidersIDSecret.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1NumberProvidersIDSecretDelete(ctx, m, logM, regV1NumberProvidersIDSecret.FindStringSubmatch(m.URI))
		requestType = "/v1/number_providers/<name>/secret"

	////////////////////
	// numbers
	////////////////////

	// PUT /numbers/<id>/flow_id
	case regV1NumbersIDFlowIDs.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1NumbersIDFlowIDsPut(ctx, m)
		requestType = "/v1/numbers/<number-id>/flow_id"

	// PUT /numbers/<id>/metadata
	case regV1NumbersIDMetadata.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1NumbersIDMetadataPut(ctx, m)
		requestType = "/v1/numbers/<number-id>/metadata"

	// DELETE /numbers/<number-id>
	case regV1NumbersID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1NumbersIDDelete(ctx, m)
		requestType = "/v1/numbers/<number-id>"

	// GET /numbers/<number-id>
	case regV1NumbersID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1NumbersIDGet(ctx, m)
		requestType = "/v1/numbers/<number-id>"

	// PUT /numbers/<number-id>
	case regV1NumbersID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1NumbersIDPut(ctx, m)
		requestType = "/v1/numbers/<number-id>"

	// POST /numbers/renew
	case regV1NumbersRenew.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1NumbersRenewPost(ctx, m)
		requestType = "/v1/numbers/renew"

	// GET /numbers/count_virtual_by_customer
	case regV1NumbersCountVirtualByCustomer.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1NumbersCountVirtualByCustomerGet(ctx, m)
		requestType = "/v1/numbers/count_virtual_by_customer"

	// POST /numbers
	case regV1Numbers.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1NumbersPost(ctx, m)
		requestType = "/v1/numbers"

	// GET /numbers
	case regV1NumbersGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1NumbersGet(ctx, m)
		requestType = "/v1/numbers"

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		log.WithFields(logrus.Fields{
			"uri":    m.URI,
			"method": m.Method,
		}).Errorf("Could not find corresponded message handler. data: %s", logM.Data)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}
	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	// default error handler — typed errors and ErrNotFound flow through
	// errorResponse; other errors keep legacy 400 (mostly unmarshal failures).
	if err != nil {
		log.WithFields(
			logrus.Fields{
				"uri":    m.URI,
				"method": m.Method,
				"error":  err,
			}).Errorf("Could not process the request correctly. data: %s", logM.Data)
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

	log.WithFields(
		logrus.Fields{
			"response": response,
		},
	).Debugf("Sending response. method: %s, uri: %s", m.Method, m.URI)

	return response, err
}
