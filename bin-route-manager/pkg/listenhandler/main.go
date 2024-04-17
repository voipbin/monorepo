package listenhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

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
	rabbitSock rabbitmqhandler.Rabbit

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
	regV1RoutesGet = regexp.MustCompile(`/v1/routes\?`)
	regV1RoutesID  = regexp.MustCompile("/v1/routes/" + regUUID + "$")

	// dialroutes
	regV1DialroutesGet = regexp.MustCompile(`/v1/dialroutes\?`)
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
func simpleResponse(code int) *rabbitmqhandler.Response {
	return &rabbitmqhandler.Response{
		StatusCode: code,
	}
}

// NewListenHandler return ListenHandler interface
func NewListenHandler(
	rabbitSock rabbitmqhandler.Rabbit,

	providerHandler providerhandler.ProviderHandler,
	routeHandler routehandler.RouteHandler,
) ListenHandler {
	h := &listenHandler{
		rabbitSock: rabbitSock,

		providerHandler: providerHandler,
		routeHandler:    routeHandler,
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
		logrus.Errorf("Could not set the queue's qos. err: %v", err)
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

	// process the received request
	go func() {
		for {
			err := h.rabbitSock.ConsumeRPCOpt(queue, "route-manager", false, false, false, 10, h.processRequest)
			if err != nil {
				logrus.Errorf("Could not consume the request message correctly. err: %v", err)
			}
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	var requestType string
	var err error
	var response *rabbitmqhandler.Response

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
	case regV1RoutesGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		requestType = "/routes"
		response, err = h.v1RoutesGet(ctx, m)

	case regV1Routes.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		requestType = "/routes"
		response, err = h.v1RoutesPost(ctx, m)

	// routes/<route-id>
	case regV1RoutesID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		requestType = "/routes/<route-id>"
		response, err = h.v1RoutesIDGet(ctx, m)

	case regV1RoutesID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		requestType = "/routes/<route-id>"
		response, err = h.v1RoutesIDPut(ctx, m)

	case regV1RoutesID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		requestType = "/routes/<route-id>"
		response, err = h.v1RoutesIDDelete(ctx, m)

	// providers
	case regV1ProvidersGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		requestType = "/providers"
		response, err = h.v1ProvidersGet(ctx, m)

	case regV1Providers.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		requestType = "/providers"
		response, err = h.v1ProvidersPost(ctx, m)

	// providers/<provider-id>
	case regV1ProvidersID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		requestType = "/providers/<provider-id>"
		response, err = h.v1ProvidersIDGet(ctx, m)

	case regV1ProvidersID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		requestType = "/providers/<provider-id>"
		response, err = h.v1ProvidersIDPut(ctx, m)

	case regV1ProvidersID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		requestType = "/providers/<provider-id>"
		response, err = h.v1ProvidersIDDelete(ctx, m)

	// dialroute
	case regV1DialroutesGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
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
