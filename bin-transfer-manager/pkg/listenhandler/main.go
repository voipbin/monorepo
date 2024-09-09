package listenhandler

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transfer-manager/pkg/transferhandler"
)

// pagination parameters
const (
	PageSize  = "page_size"
	PageToken = "page_token"
)

const (
	constCosumerName = "transfer-manager"
)

// ListenHandler interface
type ListenHandler interface {
	Run() error
}

type listenHandler struct {
	rabbitSock    rabbitmqhandler.Rabbit
	queueListen   string
	exchangeDelay string

	transferHandler transferhandler.TransferHandler
}

var (
	// v1
	// transfers
	regV1Transfers = regexp.MustCompile("/v1/transfers$")
)

var (
	metricsNamespace = "transfer_manager"

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
	queueListen string,
	exchangeDelay string,
	transferHandler transferhandler.TransferHandler,
) ListenHandler {
	h := &listenHandler{
		rabbitSock:    rabbitSock,
		queueListen:   queueListen,
		exchangeDelay: exchangeDelay,

		transferHandler: transferHandler,
	}

	return h
}

// func (h *listenHandler) Run(queue, exchangeDelay string) error {
func (h *listenHandler) Run() error {
	logrus.WithFields(logrus.Fields{
		"func": "Run",
	}).Info("Creating rabbitmq queue for listen.")

	if err := h.rabbitSock.QueueCreate(h.queueListen, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// receive requests
	go func() {
		for {
			// consume the request
			err := h.rabbitSock.ConsumeRPC(h.queueListen, constCosumerName, false, false, false, 10, h.processRequest)
			if err != nil {
				logrus.Errorf("Could not consume the request message correctly. err: %v", err)
			}
		}
	}()

	return nil
}

// processRequest handles all of requests of the listen queue.
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

	log.Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	//////////////////
	// transfers
	////////////////////
	// POST /transfers
	case regV1Transfers.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1TransfersPost(ctx, m)
		requestType = "/v1/transfers"

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

	log.WithField("response", response).Debugf("Sending response. method: %s, uri: %s", m.Method, m.URI)

	return response, err
}
