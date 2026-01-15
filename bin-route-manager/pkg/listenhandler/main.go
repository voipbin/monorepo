package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-route-manager/pkg/providerhandler"
	"monorepo/bin-route-manager/pkg/routehandler"
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
	sockHandler sockhandler.SockHandler

	providerHandler providerhandler.ProviderHandler
	routeHandler    routehandler.RouteHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// providers
	regV1Providers    = regexp.MustCompile("/v1/providers$")
	regV1ProvidersGet = regexp.MustCompile(`/v1/providers\?`)
	regV1ProvidersID  = regexp.MustCompile("/v1/providers/" + regUUID + "$")

	// routes
	regV1Routes    = regexp.MustCompile("/v1/routes$")
	regV1RoutesGet = regexp.MustCompile(`/v1/routes(\?.*)?$`)
	regV1RoutesID  = regexp.MustCompile("/v1/routes/" + regUUID + "$")

	// dialroutes
	regV1DialroutesGet = regexp.MustCompile(`/v1/dialroutes(\?.*)?$`)
)

var (
	metricsNamespace = "route_manager"

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

// NewListenHandler return ListenHandler interface
func NewListenHandler(
	sockHandler sockhandler.SockHandler,

	providerHandler providerhandler.ProviderHandler,
	routeHandler routehandler.RouteHandler,
) ListenHandler {
	h := &listenHandler{
		sockHandler: sockHandler,

		providerHandler: providerHandler,
		routeHandler:    routeHandler,
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

	// process the received request
	go func() {
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, "route-manager", false, false, false, 10, h.processRequest); errConsume != nil {
			logrus.Errorf("Could not consume the request message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

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

	// v1
	// routes
	case regV1RoutesGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/routes"
		response, err = h.v1RoutesGet(ctx, m)

	case regV1Routes.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/routes"
		response, err = h.v1RoutesPost(ctx, m)

	// routes/<route-id>
	case regV1RoutesID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/routes/<route-id>"
		response, err = h.v1RoutesIDGet(ctx, m)

	case regV1RoutesID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/routes/<route-id>"
		response, err = h.v1RoutesIDPut(ctx, m)

	case regV1RoutesID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/routes/<route-id>"
		response, err = h.v1RoutesIDDelete(ctx, m)

	// providers
	case regV1ProvidersGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/providers"
		response, err = h.v1ProvidersGet(ctx, m)

	case regV1Providers.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/providers"
		response, err = h.v1ProvidersPost(ctx, m)

	// providers/<provider-id>
	case regV1ProvidersID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/providers/<provider-id>"
		response, err = h.v1ProvidersIDGet(ctx, m)

	case regV1ProvidersID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/providers/<provider-id>"
		response, err = h.v1ProvidersIDPut(ctx, m)

	case regV1ProvidersID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/providers/<provider-id>"
		response, err = h.v1ProvidersIDDelete(ctx, m)

	// dialroute
	case regV1DialroutesGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/dialroutes"
		response, err = h.v1DialroutesGet(ctx, m)

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
			"err":      err,
		}).Debugf("Sending response. method: %s, uri: %s", m.Method, m.URI)

	return response, err
}
