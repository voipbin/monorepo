package listenhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package listenhandler -destination ./mock_listenhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/confbridgehandler"
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
	rabbitSock        rabbitmqhandler.Rabbit
	callHandler       callhandler.CallHandler
	confbridgeHandler confbridgehandler.ConfbridgeHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"
	regAny  = "(.*)"

	// v1
	// asterisks
	regV1AsterisksIDChannelsIDHealth = regexp.MustCompile("/v1/asterisks/" + regAny + "/channels/" + regAny + "/health-check")

	// calls
	regV1Calls                    = regexp.MustCompile("/v1/calls$")
	regV1CallsGet                 = regexp.MustCompile(`/v1/calls\?`)
	regV1CallsID                  = regexp.MustCompile("/v1/calls/" + regUUID + "$")
	regV1CallsIDHealth            = regexp.MustCompile("/v1/calls/" + regUUID + "/health-check$")
	regV1CallsIDActionNext        = regexp.MustCompile("/v1/calls/" + regUUID + "/action-next$")
	regV1CallsIDActionTimeout     = regexp.MustCompile("/v1/calls/" + regUUID + "/action-timeout$")
	regV1CallsIDChainedCallIDs    = regexp.MustCompile("/v1/calls/" + regUUID + "/chained-call-ids$")
	regV1CallsIDChainedCallIDsIDs = regexp.MustCompile("/v1/calls/" + regUUID + "/chained-call-ids/" + regUUID + "$")
	regV1CallsIDExternalMedia     = regexp.MustCompile("/v1/calls/" + regUUID + "/external-media$")

	// confbridges
	regV1Confbridges          = regexp.MustCompile("/v1/confbridges$")
	regV1ConfbridgesID        = regexp.MustCompile("/v1/confbridges/" + regUUID + "$")
	regV1ConfbridgesIDCallsID = regexp.MustCompile("/v1/confbridges/" + regUUID + "/calls/" + regUUID + "$")

	// recordings
	regV1RecordingsID = regexp.MustCompile("/v1/recordings/" + regAny)
	regV1Recordings   = regexp.MustCompile("/v1/recordings")
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
	callHandler callhandler.CallHandler,
	confbridgeHandler confbridgehandler.ConfbridgeHandler,
) ListenHandler {
	h := &listenHandler{
		rabbitSock:        rabbitSock,
		callHandler:       callHandler,
		confbridgeHandler: confbridgeHandler,
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
			err := h.rabbitSock.ConsumeRPCOpt(queue, "call-manager", false, false, false, h.processRequest)
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

	uri, err := url.QueryUnescape(m.URI)
	if err != nil {
		uri = "could not unescape uri"
	}

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

	////////////
	// asterisks
	////////////
	// POST /asterisks/<asterisk-id>channels/<channel-id>/health-check
	case regV1AsterisksIDChannelsIDHealth.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1AsterisksIDChannelsIDHealthPost(ctx, m)
		requestType = "/v1/asterisks/channels/health-check"

	////////
	// calls
	////////
	// POST /calls/<id>/health-check
	case regV1CallsIDHealth.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1CallsIDHealthPost(ctx, m)
		requestType = "/v1/calls/health-check"

	// POST /calls/<id>/action-next
	case regV1CallsIDActionNext.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1CallsIDActionNextPost(ctx, m)
		requestType = "/v1/calls/action-next"

	// POST /calls/<id>/action-timeout
	case regV1CallsIDActionTimeout.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1CallsIDActionTimeoutPost(ctx, m)
		requestType = "/v1/calls/action-timeout"

	// POST /calls/<id>/chained-call-ids
	case regV1CallsIDChainedCallIDs.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1CallsIDChainedCallIDsPost(ctx, m)
		requestType = "/v1/calls/chained-call-ids"

	// DELETE /calls/<id>/chained-call-ids/<chaied_call_id>
	case regV1CallsIDChainedCallIDsIDs.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1CallsIDChainedCallIDsDelete(ctx, m)
		requestType = "/v1/calls/chained-call-ids"

	// POST /calls/<id>/external-media
	case regV1CallsIDExternalMedia.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1CallsIDExternalMediaPost(ctx, m)
		requestType = "/v1/calls/external-media"

	// GET /calls/<id>
	case regV1CallsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1CallsIDGet(ctx, m)
		requestType = "/v1/calls"

	// POST /calls/<id>
	case regV1CallsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1CallsIDPost(ctx, m)
		requestType = "/v1/calls"

	// DELETE /calls/<id>
	case regV1CallsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1CallsIDDelete(ctx, m)
		requestType = "/v1/calls"

	// GET /calls
	case regV1CallsGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1CallsGet(ctx, m)
		requestType = "/v1/calls"

	// POST /calls
	case regV1Calls.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1CallsPost(ctx, m)
		requestType = "/v1/calls"

	//////////////
	// confbridges
	//////////////

	// DELETE /confbridges/<confbridge-id>/calls/<call-id>
	case regV1ConfbridgesIDCallsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1ConfbridgesIDCallsIDDelete(ctx, m)
		requestType = "/v1/confbridges"

	// POST /confbridges/<confbridge-id>/calls/<call-id>
	case regV1ConfbridgesIDCallsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1ConfbridgesIDCallsIDPost(ctx, m)
		requestType = "/v1/confbridges"

	// GET /confbridges/<confbridge-id>
	case regV1ConfbridgesID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1ConfbridgesIDGet(ctx, m)
		requestType = "/v1/confbridges"

	// DELETE /confbridges/<confbridge-id>
	case regV1ConfbridgesID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1ConfbridgesIDDelete(ctx, m)
		requestType = "/v1/confbridges"

	// POST /confbridges
	case regV1Confbridges.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1ConfbridgesPost(ctx, m)
		requestType = "/v1/confbridges"

	//////////////
	// recordings
	//////////////
	// GET /recordings/<recording-id>
	case regV1RecordingsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1RecordingsIDGet(ctx, m)
		requestType = "/v1/recordings"

	// GET /recordings
	case regV1Recordings.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1RecordingsGet(ctx, m)
		requestType = "/v1/recordings"

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

	if err != nil {
		log.Errorf("Could not find corresponded message handler. method: %s, uri: %s", m.Method, uri)
		response = simpleResponse(400)
		err = nil
	} else {
		log.WithFields(
			logrus.Fields{
				"response": response,
			},
		).Debugf("Sending response. method: %s, uri: %s", m.Method, uri)
	}

	return response, err
}
