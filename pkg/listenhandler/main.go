package listenhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

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
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/confbridgehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/recordinghandler"
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
	channelHandler    channelhandler.ChannelHandler
	recordingHandler  recordinghandler.RecordingHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	//// v1

	// calls
	regV1Calls                    = regexp.MustCompile("/v1/calls$")
	regV1CallsGet                 = regexp.MustCompile(`/v1/calls\?`)
	regV1CallsID                  = regexp.MustCompile("/v1/calls/" + regUUID + "$")
	regV1CallsIDHealth            = regexp.MustCompile("/v1/calls/" + regUUID + "/health-check$")
	regV1CallsIDDigits            = regexp.MustCompile("/v1/calls/" + regUUID + "/digits$")
	regV1CallsIDActionNext        = regexp.MustCompile("/v1/calls/" + regUUID + "/action-next$")
	regV1CallsIDActionTimeout     = regexp.MustCompile("/v1/calls/" + regUUID + "/action-timeout$")
	regV1CallsIDChainedCallIDs    = regexp.MustCompile("/v1/calls/" + regUUID + "/chained-call-ids$")
	regV1CallsIDChainedCallIDsIDs = regexp.MustCompile("/v1/calls/" + regUUID + "/chained-call-ids/" + regUUID + "$")
	regV1CallsIDExternalMedia     = regexp.MustCompile("/v1/calls/" + regUUID + "/external-media$")
	regV1CallsIDHangup            = regexp.MustCompile("/v1/calls/" + regUUID + "/hangup$")
	regV1CallsIDRecordingID       = regexp.MustCompile("/v1/calls/" + regUUID + "/recording_id$")

	// channels
	regV1ChannelsIDHealth = regexp.MustCompile("/v1/channels/" + regUUID + "/health-check$")

	// confbridges
	regV1Confbridges          = regexp.MustCompile("/v1/confbridges$")
	regV1ConfbridgesID        = regexp.MustCompile("/v1/confbridges/" + regUUID + "$")
	regV1ConfbridgesIDCallsID = regexp.MustCompile("/v1/confbridges/" + regUUID + "/calls/" + regUUID + "$")

	// recordings
	regV1RecordingsGet = regexp.MustCompile(`/v1/recordings\?`)
	regV1Recordings    = regexp.MustCompile(`/v1/recordings$`)
	regV1RecordingsID  = regexp.MustCompile("/v1/recordings/" + regUUID + "$")
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
	channelHandler channelhandler.ChannelHandler,
	recordingHandler recordinghandler.RecordingHandler,
) ListenHandler {
	h := &listenHandler{
		rabbitSock:        rabbitSock,
		callHandler:       callHandler,
		confbridgeHandler: confbridgeHandler,
		channelHandler:    channelHandler,
		recordingHandler:  recordingHandler,
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

	////////
	// calls
	////////
	// POST /calls/<id>/health-check
	case regV1CallsIDHealth.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1CallsIDHealthPost(ctx, m)
		requestType = "/v1/calls/health-check"

	// Get /calls/<id>/digits
	case regV1CallsIDDigits.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1CallsIDDigitsGet(ctx, m)
		requestType = "/v1/calls/digits"

	// Get /calls/<id>/digits
	case regV1CallsIDDigits.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1CallsIDDigitsSet(ctx, m)
		requestType = "/v1/calls/digits"

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
		requestType = "/v1/calls/<call-id>"

	// POST /calls/<id>
	case regV1CallsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1CallsIDPost(ctx, m)
		requestType = "/v1/calls/<call-id>"

	// DELETE /calls/<id>
	case regV1CallsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1CallsIDDelete(ctx, m)
		requestType = "/v1/calls/<call-id>"

	// POST /calls/<id>/hangup
	case regV1CallsIDHangup.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1CallsIDHangupPost(ctx, m)
		requestType = "/v1/calls/<call-id>/hangup"

	// PUT /calls/<id>/recording_id
	case regV1CallsIDRecordingID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1CallsIDRecordingIDPut(ctx, m)
		requestType = "/v1/calls/<call-id>/recording_id"

	// GET /calls
	case regV1CallsGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1CallsGet(ctx, m)
		requestType = "/v1/calls"

	// POST /calls
	case regV1Calls.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1CallsPost(ctx, m)
		requestType = "/v1/calls"

	////////////
	// channels
	////////////
	// POST /channels/<channel-id>/health-check
	case regV1ChannelsIDHealth.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1ChannelsIDHealthPost(ctx, m)
		requestType = "/v1/channels/<channel-id>/health-check"

	//////////////
	// confbridges
	//////////////

	// DELETE /confbridges/<confbridge-id>/calls/<call-id>
	case regV1ConfbridgesIDCallsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1ConfbridgesIDCallsIDDelete(ctx, m)
		requestType = "/v1/confbridges/<confbridge-id>/calls/<call-id>"

	// POST /confbridges/<confbridge-id>/calls/<call-id>
	case regV1ConfbridgesIDCallsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1ConfbridgesIDCallsIDPost(ctx, m)
		requestType = "/v1/confbridges/<confbridge-id>/calls/<call-id>"

	// GET /confbridges/<confbridge-id>
	case regV1ConfbridgesID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1ConfbridgesIDGet(ctx, m)
		requestType = "/v1/confbridges/<confbridge-id>"

	// DELETE /confbridges/<confbridge-id>
	case regV1ConfbridgesID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1ConfbridgesIDDelete(ctx, m)
		requestType = "/v1/confbridges/<confbridge-id>"

	// POST /confbridges
	case regV1Confbridges.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1ConfbridgesPost(ctx, m)
		requestType = "/v1/confbridges"

	//////////////
	// recordings
	//////////////
	// GET /recordings
	case regV1RecordingsGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1RecordingsGet(ctx, m)
		requestType = "/v1/recordings"

	// POST /recordings
	case regV1Recordings.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1RecordingsPost(ctx, m)
		requestType = "/v1/recordings"

	// GET /recordings/<recording-id>
	case regV1RecordingsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1RecordingsIDGet(ctx, m)
		requestType = "/v1/recordings/<recording-id>"

	// DELETE /recordings/<recording-id>
	case regV1RecordingsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1RecordingsIDDelete(ctx, m)
		requestType = "/v1/recordings/<recording-id>"

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
