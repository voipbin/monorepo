package listenhandler

import (
	"fmt"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/stt-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/stt-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/stt-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/stt-manager.git/pkg/stthandler"
)

// pagination parameters
const (
	PageSize  = "page_size"
	PageToken = "page_token"
)

const (
	constCosumerName = "stt-manager"
)

// ListenHandler interface
type ListenHandler interface {
	Run(queue, exchangeDelay string) error
}

type listenHandler struct {
	rabbitSock rabbitmqhandler.Rabbit
	db         dbhandler.DBHandler
	cache      cachehandler.CacheHandler

	reqHandler requesthandler.RequestHandler
	sttHandler stthandler.STTHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"
	regAny  = "(.*)"

	// v1

	// // available numbers
	// regV1AvailableNumbers = regexp.MustCompile("/v1/available_numbers")

	// // numbers
	// regV1Numbers       = regexp.MustCompile("/v1/numbers")
	// regV1NumbersID     = regexp.MustCompile("/v1/numbers/" + regUUID)
	// regV1NumbersNumber = regexp.MustCompile("/v1/numbers/+" + regAny)

	// // numberflows
	// regV1NumberFlowsID = regexp.MustCompile("/v1/number_flows/" + regUUID)
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
func simpleResponse(code int) *rabbitmqhandler.Response {
	return &rabbitmqhandler.Response{
		StatusCode: code,
	}
}

// NewListenHandler return ListenHandler interface
func NewListenHandler(
	rabbitSock rabbitmqhandler.Rabbit,
	db dbhandler.DBHandler,
	cache cachehandler.CacheHandler,
	reqHandler requesthandler.RequestHandler,
	sttHandler stthandler.STTHandler,
) ListenHandler {
	h := &listenHandler{
		rabbitSock: rabbitSock,
		db:         db,
		cache:      cache,
		reqHandler: reqHandler,
		sttHandler: sttHandler,
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
		return fmt.Errorf("Could not declare the exchange for dealyed message. err: %v", err)
	}

	// bind a queue with delayed exchange
	if err := h.rabbitSock.QueueBind(queue, queue, exchangeDelay, false, nil); err != nil {
		return fmt.Errorf("Could not bind the queue and exchange. err: %v", err)
	}

	// receive requests
	go func() {
		for {
			// consume the request
			err := h.rabbitSock.ConsumeRPCOpt(queue, constCosumerName, false, false, false, h.processRequest)
			if err != nil {
				logrus.Errorf("Could not consume the request message correctly. err: %v", err)
			}
		}
	}()

	return nil
}

// processRequest handles all of requests of the listen queue.
func (h *listenHandler) processRequest(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	var requestType string
	var err error
	var response *rabbitmqhandler.Response

	uri, err := url.QueryUnescape(m.URI)
	if err != nil {
		uri = "could not unescape uri"
	}
	m.URI = uri

	logrus.WithFields(
		logrus.Fields{
			"uri":       m.URI,
			"method":    m.Method,
			"data_type": m.DataType,
			"data":      m.Data,
		}).Debugf("Received request. method: %s, uri: %s", m.Method, uri)

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	// ////////////////////
	// // available_numbers
	// ////////////////////
	// // GET /available_numbers
	// case regV1AvailableNumbers.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodGet:
	// 	response, err = h.processV1AvailableNumbersGet(m)
	// 	requestType = "/v1/available_numbers"

	// ////////////////////
	// // numbers
	// ////////////////////

	// // DELETE /numbers/<id>
	// case regV1NumbersID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodDelete:
	// 	response, err = h.processV1NumbersIDDelete(m)
	// 	requestType = "/v1/numbers"

	// // GET /numbers/<id>
	// case regV1NumbersID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodGet:
	// 	response, err = h.processV1NumbersIDGet(m)
	// 	requestType = "/v1/numbers"

	// // PUT /numbers/<id>
	// case regV1NumbersNumber.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodPut:
	// 	response, err = h.processV1NumbersIDPut(m)
	// 	requestType = "/v1/numbers"

	// // GET /numbers/<number>
	// case regV1NumbersNumber.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodGet:
	// 	response, err = h.processV1NumbersNumberGet(m)
	// 	requestType = "/v1/numbers"

	// // POST /numbers
	// case regV1Numbers.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodPost:
	// 	response, err = h.processV1NumbersPost(m)
	// 	requestType = "/v1/numbers"

	// // GET /numbers
	// case regV1Numbers.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodGet:
	// 	response, err = h.processV1NumbersGet(m)
	// 	requestType = "/v1/numbers"

	// ////////////////////
	// // number_flows
	// ////////////////////

	// // DELETE /number_flows/<flow_id>
	// case regV1NumberFlowsID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodDelete:
	// 	response, err = h.processV1NumberFlowsDelete(m)
	// 	requestType = "/v1/numbers_flows"

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
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
		requestType = "notfound"
	}

	logrus.WithFields(
		logrus.Fields{
			"response": response,
		},
	).Debugf("Sending response. method: %s, uri: %s", m.Method, uri)

	return response, err
}
