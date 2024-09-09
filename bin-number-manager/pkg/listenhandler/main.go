package listenhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

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

	"monorepo/bin-number-manager/pkg/numberhandler"
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
	rabbitSock  rabbitmqhandler.Rabbit

	numberHandler numberhandler.NumberHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// v1

	// available numbers
	regV1AvailableNumbers = regexp.MustCompile("/v1/available_numbers")

	// numbers
	regV1NumbersGet       = regexp.MustCompile(`/v1/numbers\?`)
	regV1Numbers          = regexp.MustCompile(`/v1/numbers$`)
	regV1NumbersID        = regexp.MustCompile("/v1/numbers/" + regUUID + "$")
	regV1NumbersIDFlowIDs = regexp.MustCompile("/v1/numbers/" + regUUID + "/flow_ids$")
	regV1NumbersRenew     = regexp.MustCompile(`/v1/numbers/renew$`)
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

// NewListenHandler return ListenHandler interface
func NewListenHandler(rabbitSock rabbitmqhandler.Rabbit, numberHandler numberhandler.NumberHandler) ListenHandler {
	h := &listenHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		rabbitSock:    rabbitSock,
		numberHandler: numberHandler,
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

	log := logrus.WithFields(logrus.Fields{
		"func":      "processRequest",
		"uri":       m.URI,
		"method":    m.Method,
		"data_type": m.DataType,
		"data":      m.Data,
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
	// numbers
	////////////////////

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

	// PUT /numbers/<id>/flow_id
	case regV1NumbersIDFlowIDs.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1NumbersIDFlowIDsPut(ctx, m)
		requestType = "/v1/numbers/<number-id>/flow_id"

	// POST /numbers/renew
	case regV1NumbersRenew.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1NumbersRenewPost(ctx, m)
		requestType = "/v1/numbers/renew"

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
		}).Errorf("Could not find corresponded message handler. data: %s", m.Data)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}
	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	// default error handler
	if err != nil {
		log.WithFields(
			logrus.Fields{
				"uri":    m.URI,
				"method": m.Method,
				"error":  err,
			}).Errorf("Could not process the request correctly. data: %s", m.Data)
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
