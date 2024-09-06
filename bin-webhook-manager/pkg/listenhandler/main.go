package listenhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package listenhandler -destination ./mock_listenhandler.go -source main.go -build_flags=-mod=mod
import (
	"context"
	"fmt"
	"regexp"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-webhook-manager/pkg/webhookhandler"
)

// pagination parameters
const (
	PageSize  = "page_size"
	PageToken = "page_token"
)

const (
	constCosumerName = "webhook-manager"
)

// ListenHandler interface
type ListenHandler interface {
	Run(queue, exchangeDelay string) error
}

type listenHandler struct {
	rabbitSock rabbitmqhandler.Rabbit

	whHandler webhookhandler.WebhookHandler
}

var (
	// v1
	// webhooks
	regV1Webhooks = regexp.MustCompile("/v1/webhooks")

	// webhook_destinations
	regV1WebhookDestinations = regexp.MustCompile("/v1/webhook_destinations")
)

var (
	metricsNamespace = "webhook_manager"

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
	rabbitSock rabbitmqhandler.Rabbit,
	whHandler webhookhandler.WebhookHandler,
) ListenHandler {
	h := &listenHandler{
		rabbitSock: rabbitSock,
		whHandler:  whHandler,
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
func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {

	var requestType string
	var err error
	var response *sock.Response

	log := logrus.WithFields(
		logrus.Fields{
			"request": m,
		})
	log.Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)

	start := time.Now()
	ctx := context.Background()

	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	////////////////////
	// webhooks
	////////////////////
	// POST /webhooks
	case regV1Webhooks.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1WebhooksPost(ctx, m)
		requestType = "/v1/webhooks"

	////////////////////
	// webhooks_customs
	////////////////////
	// POST /webhook_destinations
	case regV1WebhookDestinations.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1WebhookDestinationsPost(ctx, m)
		requestType = "/v1/webhook_destinations"

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

	// default error handler
	if err != nil {
		log.Errorf("Could not process the request correctly. method: %s, uri: %s, err: %v", m.Method, m.URI, err)
		response = simpleResponse(400)
		err = nil
	}

	log.WithFields(
		logrus.Fields{
			"response": response,
		},
	).Debugf("Sending response. method: %s, uri: %s", m.Method, m.URI)

	return response, err
}
