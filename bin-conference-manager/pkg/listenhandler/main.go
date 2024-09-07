package listenhandler

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

	"monorepo/bin-conference-manager/pkg/conferencecallhandler"
	"monorepo/bin-conference-manager/pkg/conferencehandler"
)

// pagination parameters
const (
	PageSize  = "page_size"
	PageToken = "page_token"
)

const (
	constCosumerName = "conference-manager"
)

// ListenHandler interface
type ListenHandler interface {
	Run() error
}

type listenHandler struct {
	utilHandler   utilhandler.UtilHandler
	rabbitSock    rabbitmqhandler.Rabbit
	queueListen   string
	exchangeDelay string

	conferenceHandler     conferencehandler.ConferenceHandler
	conferencecallHandler conferencecallhandler.ConferencecallHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// v1
	// conferences
	regV1Conferences                  = regexp.MustCompile("/v1/conferences$")
	regV1ConferencesGet               = regexp.MustCompile(`/v1/conferences\?`)
	regV1ConferencesID                = regexp.MustCompile("/v1/conferences/" + regUUID + "$")
	regV1ConferencesIDRecordingID     = regexp.MustCompile("/v1/conferences/" + regUUID + "/recording_id$")
	regV1ConferencesIDRecordingStart  = regexp.MustCompile("/v1/conferences/" + regUUID + "/recording_start$")
	regV1ConferencesIDRecordingStop   = regexp.MustCompile("/v1/conferences/" + regUUID + "/recording_stop$")
	regV1ConferencesIDStop            = regexp.MustCompile("/v1/conferences/" + regUUID + "/stop$")
	regV1ConferencesIDTranscribeStart = regexp.MustCompile("/v1/conferences/" + regUUID + "/transcribe_start$")
	regV1ConferencesIDTranscribeStop  = regexp.MustCompile("/v1/conferences/" + regUUID + "/transcribe_stop$")

	// conferencecalls
	regV1ConferencecallsGet           = regexp.MustCompile(`/v1/conferencecalls\?`)
	regV1ConferencecallsID            = regexp.MustCompile("/v1/conferencecalls/" + regUUID + "$")
	regV1ConferencecallsIDHealthCheck = regexp.MustCompile("/v1/conferencecalls/" + regUUID + "/health-check$")

	// services
	regV1ServicesTypeConferencecall = regexp.MustCompile("/v1/services/type/conferencecall$")
)

var (
	metricsNamespace = "conference_manager"

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
	conferenceHandler conferencehandler.ConferenceHandler,
	conferencecallHandler conferencecallhandler.ConferencecallHandler,
) ListenHandler {
	h := &listenHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		rabbitSock:    rabbitSock,
		queueListen:   queueListen,
		exchangeDelay: exchangeDelay,

		conferenceHandler:     conferenceHandler,
		conferencecallHandler: conferencecallHandler,
	}

	return h
}

// func (h *listenHandler) Run(queue, exchangeDelay string) error {
func (h *listenHandler) Run() error {
	logrus.WithFields(logrus.Fields{
		"func": "Run",
	}).Info("Creating rabbitmq queue for listen.")

	// declare the queue
	if err := h.rabbitSock.QueueDeclare(h.queueListen, true, false, false, false); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// Set QoS
	if err := h.rabbitSock.QueueQoS(h.queueListen, 1, 0); err != nil {
		logrus.Errorf("Could not set the queue's qos. err: %v", err)
		return err
	}

	// create a exchange for delayed message
	if err := h.rabbitSock.ExchangeDeclareForDelay(h.exchangeDelay, true, false, false, false); err != nil {
		return fmt.Errorf("could not declare the exchange for dealyed message. err: %v", err)
	}

	// bind a queue with delayed exchange
	if err := h.rabbitSock.QueueBind(h.queueListen, h.queueListen, h.exchangeDelay, false, nil); err != nil {
		return fmt.Errorf("could not bind the queue and exchange. err: %v", err)
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
	// conferences
	////////////////////

	// POST /conferences
	case regV1Conferences.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConferencesPost(ctx, m)
		requestType = "/v1/conferences"

	// GET /conferences
	case regV1ConferencesGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ConferencesGet(ctx, m)
		requestType = "/v1/conferences"

	// GET /conferences/<conference-id>
	case regV1ConferencesID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ConferencesIDGet(ctx, m)
		requestType = "/v1/conferences/<conference-id>"

	// PUT /conferences/<conference-id>
	case regV1ConferencesID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1ConferencesIDPut(ctx, m)
		requestType = "/v1/conferences/<conference-id>"

	// DELETE /conferences/<conference-id>
	case regV1ConferencesID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1ConferencesIDDelete(ctx, m)
		requestType = "/v1/conferences/<conference-id>"

	// PUT /conferences/<conference-id>/recording_id
	case regV1ConferencesIDRecordingID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1ConferencesIDRecordingIDPut(ctx, m)
		requestType = "/v1/conferences/<conference-id>/recording_id"

	// POST /conferences/<conference-id>/recording_start
	case regV1ConferencesIDRecordingStart.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConferencesIDRecordingStartPost(ctx, m)
		requestType = "/v1/conferences/<conference-id>/recording_start"

	// POST /conferences/<conference-id>/recording_stop
	case regV1ConferencesIDRecordingStop.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConferencesIDRecordingStopPost(ctx, m)
		requestType = "/v1/conferences/<conference-id>/recording_stop"

	// POST /conferences/<conference-id>/stop
	case regV1ConferencesIDStop.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConferencesIDStopPost(ctx, m)
		requestType = "/v1/conferences/<conference-id>/stop"

	// POST /conferences/<conference-id>/transcribe_start
	case regV1ConferencesIDTranscribeStart.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConferencesIDTranscribeStartPost(ctx, m)
		requestType = "/v1/conferences/<conference-id>/transcribe_start"

	// POST /conferences/<conference-id>/transcribe_stop
	case regV1ConferencesIDTranscribeStop.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConferencesIDTranscribeStopPost(ctx, m)
		requestType = "/v1/conferences/<conference-id>/transcirbe_stop"

	//////////////////
	// conferencecalls
	////////////////////

	// GET /conferencecalls
	case regV1ConferencecallsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ConferencecallsGet(ctx, m)
		requestType = "/v1/conferencecalls"

	// GET /conferencecalls/<conferencecall-id>
	case regV1ConferencecallsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ConferencecallsIDGet(ctx, m)
		requestType = "/v1/conferencescalls/<conferencecall-id>"

	// DELETE /conferencecalls/<conferencecall-id>
	case regV1ConferencecallsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1ConferencecallsIDDelete(ctx, m)
		requestType = "/v1/conferencescalls/<conferencecall-id>"

	// POST /conferencecalls/<conferencecall-id>/health-check
	case regV1ConferencecallsIDHealthCheck.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConferencecallsIDHealthCheckPost(ctx, m)
		requestType = "/v1/conferencescalls/<conferencecall-id>/health-check"

	/////////////////
	// services
	////////////////
	// POST /services/type/conferencecall
	case regV1ServicesTypeConferencecall.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ServicesTypeConferencecallPost(ctx, m)
		requestType = "/v1/services/type/conferencecall"

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
