package listenhandler

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/conferencecallhandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/dbhandler"
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
	Run(queue, exchangeDelay string) error
}

type listenHandler struct {
	rabbitSock    rabbitmqhandler.Rabbit
	notifyHandler notifyhandler.NotifyHandler

	conferenceHandler     conferencehandler.ConferenceHandler
	conferencecallHandler conferencecallhandler.ConferencecallHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// v1
	// conferences
	regV1Conferences       = regexp.MustCompile("/v1/conferences$")
	regV1ConferencesGet    = regexp.MustCompile(`/v1/conferences\?`)
	regV1ConferencesID     = regexp.MustCompile("/v1/conferences/" + regUUID)
	regV1ConferencesIDJoin = regexp.MustCompile("/v1/conferences/" + regUUID + "/join$")

	// conferencecalls
	regV1Conferencecalls   = regexp.MustCompile("/v1/conferencecalls$")
	regV1ConferencecallsID = regexp.MustCompile("/v1/conferencecalls/" + regUUID + "$")
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
func simpleResponse(code int) *rabbitmqhandler.Response {
	return &rabbitmqhandler.Response{
		StatusCode: code,
	}
}

// NewListenHandler return ListenHandler interface
func NewListenHandler(
	rabbitSock rabbitmqhandler.Rabbit,
	db dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
	conferenceHandler conferencehandler.ConferenceHandler,
	conferencecallHandler conferencecallhandler.ConferencecallHandler,
) ListenHandler {
	h := &listenHandler{
		rabbitSock:            rabbitSock,
		notifyHandler:         notifyHandler,
		conferenceHandler:     conferenceHandler,
		conferencecallHandler: conferencecallHandler,
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
	ctx := context.Background()

	uri, err := url.QueryUnescape(m.URI)
	if err != nil {
		uri = "could not unescape uri"
	}
	m.URI = uri

	log := logrus.WithFields(
		logrus.Fields{
			"request": m,
		})
	log.Debugf("Received request. method: %s, uri: %s", m.Method, uri)

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	//////////////////
	// conferences
	////////////////////

	// POST /conferences/<conference-id>/join
	case regV1ConferencesIDJoin.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1ConferencesIDJoinPost(ctx, m)
		requestType = "/v1/conferences/<conference-id>/join"

	// GET /conferences/<conference-id>
	case regV1ConferencesID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1ConferencesIDGet(ctx, m)
		requestType = "/v1/conferences/<conference-id>"

	// PUT /conferences/<conference-id>
	case regV1ConferencesID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1ConferencesIDPut(ctx, m)
		requestType = "/v1/conferences/<conference-id>"

	// DELETE /conferences/<conference-id>
	case regV1ConferencesID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1ConferencesIDDelete(ctx, m)
		requestType = "/v1/conferences/<conference-id>"

	// POST /conferences
	case regV1Conferences.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1ConferencesPost(ctx, m)
		requestType = "/v1/conferences"

	// GET /conferences
	case regV1ConferencesGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1ConferencesGet(ctx, m)
		requestType = "/v1/conferences"

	//////////////////
	// conferencecalls
	////////////////////

	// POST /conferencecalls
	case regV1Conferencecalls.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1ConferencecallsPost(ctx, m)
		requestType = "/v1/conferencescalls/<conferencecall-id>"

	// GET /conferencecalls/<conferencecall-id>
	case regV1ConferencecallsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1ConferencecallsIDGet(ctx, m)
		requestType = "/v1/conferencescalls/<conferencecall-id>"

	// DELETE /conferencecalls/<conferencecall-id>
	case regV1ConferencecallsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1ConferencecallsIDDelete(ctx, m)
		requestType = "/v1/conferencescalls/<conferencecall-id>"

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		log.Errorf("Could not find corresponded message handler. method: %s, uri: %s", m.Method, uri)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}
	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	// default error handler
	if err != nil {
		log.Errorf("Could not process the request correctly. method: %s, uri: %s, err: %v", m.Method, uri, err)
		response = simpleResponse(400)
		err = nil
	}

	log.WithFields(
		logrus.Fields{
			"response": response,
		},
	).Debugf("Sending response. method: %s, uri: %s", m.Method, uri)

	return response, err
}
