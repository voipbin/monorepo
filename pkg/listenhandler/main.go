package listenhandler

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/transcribehandler"
)

// pagination parameters
const (
	PageSize  = "page_size"
	PageToken = "page_token"
)

const (
	constCosumerName = "transcribe-manager"
)

// ListenHandler interface
type ListenHandler interface {
	Run(queue, queueVolatile, exchangeDelay string) error
}

type listenHandler struct {
	hostID     uuid.UUID
	rabbitSock rabbitmqhandler.Rabbit

	reqHandler        requesthandler.RequestHandler
	transcribeHandler transcribehandler.TranscribeHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// v1

	// // call-recordings
	// regV1CallRecordings = regexp.MustCompile("/v1/call_recordings")

	// recordings
	regV1Recordings = regexp.MustCompile("/v1/recordings")

	// streamings
	regV1Streamings   = regexp.MustCompile("/v1/streamings$")
	regV1StreamingsID = regexp.MustCompile("/v1/streamings/" + regUUID + "$")

	// transcribes
	regV1TranscribesGet = regexp.MustCompile(`/v1/transcribes\?`)
	regV1TranscribesID  = regexp.MustCompile("/v1/transcribes/" + regUUID + "$")
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
	hostID uuid.UUID,
	rabbitSock rabbitmqhandler.Rabbit,
	reqHandler requesthandler.RequestHandler,
	transcribeHandler transcribehandler.TranscribeHandler,
) ListenHandler {
	h := &listenHandler{
		hostID:            hostID,
		rabbitSock:        rabbitSock,
		reqHandler:        reqHandler,
		transcribeHandler: transcribeHandler,
	}

	return h
}

// runListenQueue listens the queue
func (h *listenHandler) runListenQueue(queue string) error {
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

// runListenQueueVolatile listens volatile queue
func (h *listenHandler) runListenQueueVolatile(queue string) error {
	logrus.WithFields(logrus.Fields{
		"queue": queue,
	}).Info("Creating rabbitmq queue for listen.")

	// declare the queue
	if err := h.rabbitSock.QueueDeclare(queue, false, true, false, false); err != nil {
		return fmt.Errorf("could not declare the queue volatile. err: %v", err)
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

// runDeclareDelayQueue declares delay queue
func (h *listenHandler) runDeclareDelayQueue(queue, exchangeDelay string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "runDeclareDelayQueue",
		"queue": queue,
	})

	// create a exchange for delayed message
	if err := h.rabbitSock.ExchangeDeclareForDelay(queue, true, false, false, false); err != nil {
		log.Errorf("Could not declare the exchange for dealyed message. err: %v", err)
		return err
	}

	// bind a queue with delayed exchange
	if err := h.rabbitSock.QueueBind(queue, queue, exchangeDelay, false, nil); err != nil {
		log.Errorf("Could not bind the queue and exchange. err: %v", err)
		return err
	}

	return nil
}

// Run
func (h *listenHandler) Run(queue, queueVolatile, exchangeDelay string) error {
	log := logrus.WithFields(logrus.Fields{
		"queue":          queue,
		"queue volatile": queueVolatile,
	})
	log.Info("Creating rabbitmq queue for listen.")

	// start queue listen
	if err := h.runListenQueue(queue); err != nil {
		log.Errorf("Could not listen the queue. err: %v", err)
		return err
	}

	// start volatile queue listen
	if err := h.runListenQueueVolatile(queueVolatile); err != nil {
		log.Errorf("Could not listen the volatile queue. err: %v", err)
		return err
	}

	// delcare the delay queue
	if err := h.runDeclareDelayQueue(queue, exchangeDelay); err != nil {
		log.Errorf("Could not declare the delay queue. err: %v", err)
		return err
	}

	return nil
}

// processRequest handles all of requests of the listen queue.
func (h *listenHandler) processRequest(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	var requestType string
	var err error
	var response *rabbitmqhandler.Response

	ctx := context.Background()

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

	////////////////////
	// recordings
	////////////////////
	// POST /recordings
	// case regV1Recordings.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
	// 	response, err = h.processV1RecordingsPost(m)
	// 	requestType = "/v1/recordings"

	// ////////////////////
	// // call-recordings
	// ////////////////////
	// // POST /call-recordings
	// case regV1CallRecordings.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
	// 	response, err = h.processV1CallRecordingsPost(m)
	// 	requestType = "/v1/call_recordings"

	////////////////////
	// streamings
	////////////////////
	// // POST /streamings
	// case regV1Streamings.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
	// 	response, err = h.processV1StreamingsPost(m)
	// 	requestType = "/v1/streamings"

	// // DELETE /streamings/<id>
	// case regV1StreamingsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
	// 	response, err = h.processV1StreamingsIDDelete(m)
	// 	requestType = "/v1/streamings"

	////////////////////
	// transcribes
	////////////////////
	// GET /transcribes
	case regV1TranscribesGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1TranscribesGet(ctx, m)
		requestType = "/v1/transcribes"

	// GET /transcribes/<id>
	case regV1TranscribesID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1TranscribesIDGet(ctx, m)
		requestType = "/v1/transcribes/<transcribe-id>"

	// DELETE /transcribes/<id>
	case regV1TranscribesID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1TranscribesIDDelete(ctx, m)
		requestType = "/v1/transcribes/<transcribe-id>"

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
	}

	logrus.WithFields(
		logrus.Fields{
			"response": response,
		},
	).Debugf("Sending response. method: %s, uri: %s", m.Method, uri)

	return response, err
}
