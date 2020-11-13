package listenhandler

import (
	"fmt"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// ListenHandler interface
type ListenHandler interface {
	Run(queue, exchangeDelay string) error
}

type listenHandler struct {
	rabbitSock rabbitmqhandler.Rabbit
	db         dbhandler.DBHandler
	cache      cachehandler.CacheHandler

	reqHandler        requesthandler.RequestHandler
	callHandler       callhandler.CallHandler
	conferenceHandler conferencehandler.ConferenceHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// v1
	// asterisks
	regV1AsterisksIDChannelsIDHealth = regexp.MustCompile("/v1/asterisks/(.*)/channels/(.*)/health-check")

	// calls
	regV1Calls                 = regexp.MustCompile("/v1/calls")
	regV1CallsID               = regexp.MustCompile("/v1/calls/" + regUUID)
	regV1CallsIDHealth         = regexp.MustCompile("/v1/calls/" + regUUID + "/health-check")
	regV1CallsIDActionNext     = regexp.MustCompile("/v1/calls/" + regUUID + "/action-next")
	regV1CallsIDActionTimeout  = regexp.MustCompile("/v1/calls/" + regUUID + "/action-timeout")
	regV1CallsIDChainedCallIDs = regexp.MustCompile("/v1/calls/" + regUUID + "/chained-call-ids")

	// conferences
	regV1ConferencesIDCallsID = regexp.MustCompile("/v1/conferences/" + regUUID + "/calls/" + regUUID)
	regV1ConferencesID        = regexp.MustCompile("/v1/conferences/" + regUUID)
	regV1Conferences          = regexp.MustCompile("/v1/conferences")
)

var (
	metricsNamespace = "call_manager"

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
	callHandler callhandler.CallHandler,
	conferenceHandler conferencehandler.ConferenceHandler,
) ListenHandler {
	h := &listenHandler{
		rabbitSock:        rabbitSock,
		db:                db,
		cache:             cache,
		reqHandler:        reqHandler,
		callHandler:       callHandler,
		conferenceHandler: conferenceHandler,
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

	// create a exchange for delayed message
	if err := h.rabbitSock.ExchangeDeclareForDelay(exchangeDelay, true, false, false, false); err != nil {
		return fmt.Errorf("Could not declare the exchange for dealyed message. err: %v", err)
	}

	// bind a queue with delayed exchange
	if err := h.rabbitSock.QueueBind(queue, queue, exchangeDelay, false, nil); err != nil {
		return fmt.Errorf("Could not bind the queue and exchange. err: %v", err)
	}

	// receive ARI event
	go func() {
		for {
			err := h.rabbitSock.ConsumeRPC(queue, "call-manager", h.processRequest)
			if err != nil {
				logrus.Errorf("Could not consume the ARI message correctly. Will try again after 1 second. err: %v", err)
				time.Sleep(time.Second * 1)
			}
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	var requestType string
	var err error
	var response *rabbitmqhandler.Response

	logrus.WithFields(
		logrus.Fields{
			"uri":       m.URI,
			"method":    m.Method,
			"data_type": m.DataType,
			"data":      m.Data,
		}).Debug("Received request.")

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	////////////
	// asterisks
	////////////
	// POST /asterisks/<asterisk-id>channels/<channel-id>/health-check
	case regV1AsterisksIDChannelsIDHealth.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1AsterisksIDChannelsIDHealthPost(m)
		requestType = "/v1/asterisks/channels/health-check"

	////////
	// calls
	////////
	// POST /calls/<id>/health-check
	case regV1CallsIDHealth.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1CallsIDHealthPost(m)
		requestType = "/v1/calls/health-check"

	// POST /calls/<id>/action-next
	case regV1CallsIDActionNext.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1CallsIDActionNextPost(m)
		requestType = "/v1/calls/action-next"

	// POST /calls/<id>/action-timeout
	case regV1CallsIDActionTimeout.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1CallsIDActionTimeoutPost(m)
		requestType = "/v1/calls/action-timeout"

	// POST /calls/<id>/chained-call-ids
	case regV1CallsIDChainedCallIDs.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1CallsIDChainedCallIDsPost(m)
		requestType = "/v1/calls/chained-call-ids"

	// DELETE /calls/<id>/chained-call-ids
	case regV1CallsIDChainedCallIDs.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1CallsIDChainedCallIDsDelete(m)
		requestType = "/v1/calls/chained-call-ids"

	// GET /calls/<id>
	case regV1CallsID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1CallsIDGet(m)
		requestType = "/v1/calls"

	// POST /calls/<id>
	case regV1CallsID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1CallsIDPost(m)
		requestType = "/v1/calls"

	// DELETE /calls/<id>
	case regV1CallsID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1CallsIDDelete(m)
		requestType = "/v1/calls"

	// POST /calls
	case regV1Calls.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1CallsPost(m)
		requestType = "/v1/calls"

	//////////////
	// conferences
	//////////////
	// DELETE /conferences/<conference-id>/calls/<call-id>
	case regV1ConferencesIDCallsID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1ConferencesIDCallsIDDelete(m)
		requestType = "/v1/conferences/calls"

	// DELETE /conferences/<conference-id>
	case regV1ConferencesID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1ConferencesIDDelete(m)
		requestType = "/v1/conferences"

	// GET /conferences/<conference-id>
	case regV1ConferencesID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1ConferencesIDGet(m)
		requestType = "/v1/conferences"

	// POST /conferences
	case regV1Conferences.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1ConferencesPost(m)
		requestType = "/v1/conferences"

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

	return response, err
}
