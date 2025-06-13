package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"monorepo/bin-call-manager/models/common"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
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
	utilHandler          utilhandler.UtilHandler
	sockHandler          sockhandler.SockHandler
	callHandler          callhandler.CallHandler
	confbridgeHandler    confbridgehandler.ConfbridgeHandler
	channelHandler       channelhandler.ChannelHandler
	recordingHandler     recordinghandler.RecordingHandler
	externalMediaHandler externalmediahandler.ExternalMediaHandler
	groupcallHandler     groupcallhandler.GroupcallHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"
	regAny  = ".*"

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
	regV1CallsIDHold              = regexp.MustCompile("/v1/calls/" + regUUID + "/hold$")
	regV1CallsIDMute              = regexp.MustCompile("/v1/calls/" + regUUID + "/mute$")
	regV1CallsIDMOH               = regexp.MustCompile("/v1/calls/" + regUUID + "/moh$")
	regV1CallsIDSilence           = regexp.MustCompile("/v1/calls/" + regUUID + "/silence$")
	regV1CallsIDConfbridgeID      = regexp.MustCompile("/v1/calls/" + regUUID + "/confbridge_id$")
	regV1CallsIDRecordingID       = regexp.MustCompile("/v1/calls/" + regUUID + "/recording_id$")
	regV1CallsIDRecordingStart    = regexp.MustCompile("/v1/calls/" + regUUID + "/recording_start$")
	regV1CallsIDRecordingStop     = regexp.MustCompile("/v1/calls/" + regUUID + "/recording_stop$")
	regV1CallsIDTalk              = regexp.MustCompile("/v1/calls/" + regUUID + "/talk$")
	regV1CallsIDPlay              = regexp.MustCompile("/v1/calls/" + regUUID + "/play$")
	regV1CallsIDMediaStop         = regexp.MustCompile("/v1/calls/" + regUUID + "/media_stop$")

	// channels
	regV1ChannelsIDHealth = regexp.MustCompile("/v1/channels/" + regAny + "/health-check$")

	// confbridges
	regV1Confbridges                 = regexp.MustCompile("/v1/confbridges$")
	regV1ConfbridgesID               = regexp.MustCompile("/v1/confbridges/" + regUUID + "$")
	regV1ConfbridgesIDAnswer         = regexp.MustCompile("/v1/confbridges/" + regUUID + "/answer$")
	regV1ConfbridgesIDExternalMedia  = regexp.MustCompile("/v1/confbridges/" + regUUID + "/external-media$")
	regV1ConfbridgesIDCallsID        = regexp.MustCompile("/v1/confbridges/" + regUUID + "/calls/" + regUUID + "$")
	regV1ConfbridgesIDRecordingStart = regexp.MustCompile("/v1/confbridges/" + regUUID + "/recording_start$")
	regV1ConfbridgesIDRecordingStop  = regexp.MustCompile("/v1/confbridges/" + regUUID + "/recording_stop$")
	regV1ConfbridgesIDRing           = regexp.MustCompile("/v1/confbridges/" + regUUID + "/ring$")
	regV1ConfbridgesIDFlags          = regexp.MustCompile("/v1/confbridges/" + regUUID + "/flags$")
	regV1ConfbridgesIDTerminate      = regexp.MustCompile("/v1/confbridges/" + regUUID + "/terminate$")

	// external-medias
	regV1ExternalMedias    = regexp.MustCompile("/v1/external-medias$")
	regV1ExternalMediasGet = regexp.MustCompile(`/v1/external-medias\?`)
	regV1ExternalMediasID  = regexp.MustCompile("/v1/external-medias/" + regUUID + "$")

	// groupcalls
	regV1Groupcalls                    = regexp.MustCompile("/v1/groupcalls$")
	regV1GroupcallsGet                 = regexp.MustCompile(`/v1/groupcalls\?`)
	regV1GroupcallsID                  = regexp.MustCompile("/v1/groupcalls/" + regUUID + "$")
	regV1GroupcallsIDHangup            = regexp.MustCompile("/v1/groupcalls/" + regUUID + "/hangup$")
	regV1GroupcallsIDHangupGroupcall   = regexp.MustCompile("/v1/groupcalls/" + regUUID + "/hangup_groupcall$")
	regV1GroupcallsIDHangupCall        = regexp.MustCompile("/v1/groupcalls/" + regUUID + "/hangup_call$")
	regV1GroupcallsIDAnswerGroupcallID = regexp.MustCompile("/v1/groupcalls/" + regUUID + "/answer_groupcall_id$")

	// recovery
	regV1Recovery = regexp.MustCompile("/v1/recovery$")

	// recordings
	regV1RecordingsGet    = regexp.MustCompile(`/v1/recordings\?`)
	regV1Recordings       = regexp.MustCompile(`/v1/recordings$`)
	regV1RecordingsID     = regexp.MustCompile("/v1/recordings/" + regUUID + "$")
	regV1RecordingsIDStop = regexp.MustCompile("/v1/recordings/" + regUUID + "/stop$")
)

var (
	metricsNamespace = commonoutline.GetMetricNameSpace(common.Servicename)

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
	sockHandler sockhandler.SockHandler,
	callHandler callhandler.CallHandler,
	confbridgeHandler confbridgehandler.ConfbridgeHandler,
	channelHandler channelhandler.ChannelHandler,
	recordingHandler recordinghandler.RecordingHandler,
	externalMediaHandler externalmediahandler.ExternalMediaHandler,
	groupcallHandler groupcallhandler.GroupcallHandler,
) ListenHandler {
	h := &listenHandler{
		utilHandler:          utilhandler.NewUtilHandler(),
		sockHandler:          sockHandler,
		callHandler:          callHandler,
		confbridgeHandler:    confbridgeHandler,
		channelHandler:       channelHandler,
		recordingHandler:     recordingHandler,
		externalMediaHandler: externalMediaHandler,
		groupcallHandler:     groupcallHandler,
	}

	return h
}

func (h *listenHandler) Run(queue, exchangeDelay string) error {
	logrus.WithFields(logrus.Fields{
		"queue": queue,
	}).Info("Creating rabbitmq queue for listen.")

	// declare the queue
	if err := h.sockHandler.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// process requests
	go func() {
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, string(common.Servicename), false, false, false, 10, h.processRequest); errConsume != nil {
			logrus.Errorf("Could not consume the request message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

// processRequest processes the request.
func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processRequest",
		"request": m,
	})

	var requestType string
	var err error
	var response *sock.Response

	ctx := context.Background()

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	////////
	// calls
	////////
	// POST /calls/<call-id>/health-check
	case regV1CallsIDHealth.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CallsIDHealthPost(ctx, m)
		requestType = "/v1/calls/health-check"

	// Get /calls/<call-id>/digits
	case regV1CallsIDDigits.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1CallsIDDigitsGet(ctx, m)
		requestType = "/v1/calls/digits"

	// Get /calls/<call-id>/digits
	case regV1CallsIDDigits.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CallsIDDigitsSet(ctx, m)
		requestType = "/v1/calls/digits"

	// POST /calls/<call-id>/action-next
	case regV1CallsIDActionNext.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CallsIDActionNextPost(ctx, m)
		requestType = "/v1/calls/action-next"

	// POST /calls/<call-id>/action-timeout
	case regV1CallsIDActionTimeout.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CallsIDActionTimeoutPost(ctx, m)
		requestType = "/v1/calls/action-timeout"

	// POST /calls/<call-id>/chained-call-ids
	case regV1CallsIDChainedCallIDs.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CallsIDChainedCallIDsPost(ctx, m)
		requestType = "/v1/calls/chained-call-ids"

	// DELETE /calls/<call-id>/chained-call-ids/<chaied_call_id>
	case regV1CallsIDChainedCallIDsIDs.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1CallsIDChainedCallIDsDelete(ctx, m)
		requestType = "/v1/calls/chained-call-ids"

	// POST /calls/<call-id>/external-media
	case regV1CallsIDExternalMedia.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CallsIDExternalMediaPost(ctx, m)
		requestType = "/v1/calls/external-media"

	// DELETE /calls/<call-id>/external-media
	case regV1CallsIDExternalMedia.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1CallsIDExternalMediaDelete(ctx, m)
		requestType = "/v1/calls/external-media"

	// GET /calls/<call-id>
	case regV1CallsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1CallsIDGet(ctx, m)
		requestType = "/v1/calls/<call-id>"

	// POST /calls/<call-id>
	case regV1CallsID.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CallsIDPost(ctx, m)
		requestType = "/v1/calls/<call-id>"

	// DELETE /calls/<call-id>
	case regV1CallsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1CallsIDDelete(ctx, m)
		requestType = "/v1/calls/<call-id>"

	// POST /calls/<call-id>/hangup
	case regV1CallsIDHangup.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CallsIDHangupPost(ctx, m)
		requestType = "/v1/calls/<call-id>/hangup"

	// PUT /calls/<call-id>/recording_id
	case regV1CallsIDConfbridgeID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1CallsIDConfbridgeIDPut(ctx, m)
		requestType = "/v1/calls/<call-id>/recording_id"

	// PUT /calls/<call-id>/recording_id
	case regV1CallsIDRecordingID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1CallsIDRecordingIDPut(ctx, m)
		requestType = "/v1/calls/<call-id>/recording_id"

	// POST /calls/<call-id>/recording_start
	case regV1CallsIDRecordingStart.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CallsIDRecordingStartPost(ctx, m)
		requestType = "/v1/calls/<call-id>/recording_start"

	// POST /calls/<call-id>/recording_stop
	case regV1CallsIDRecordingStop.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CallsIDRecordingStopPost(ctx, m)
		requestType = "/v1/calls/<call-id>/recording_stop"

	// POST /calls/<call-id>/talk
	case regV1CallsIDTalk.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CallsIDTalkPost(ctx, m)
		requestType = "/v1/calls/<call-id>/talk"

	// POST /calls/<call-id>/play
	case regV1CallsIDPlay.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CallsIDPlayPost(ctx, m)
		requestType = "/v1/calls/<call-id>/play"

	// POST /calls/<call-id>/media_stop
	case regV1CallsIDMediaStop.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CallsIDMediaStopPost(ctx, m)
		requestType = "/v1/calls/<call-id>/media_stop"

	// POST /calls/<call-id>/hold
	case regV1CallsIDHold.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CallsIDHoldPost(ctx, m)
		requestType = "/v1/calls/<call-id>/hold"

	// DELETE /calls/<call-id>/hold
	case regV1CallsIDHold.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1CallsIDHoldDelete(ctx, m)
		requestType = "/v1/calls/<call-id>/hold"

	// POST /calls/<call-id>/mute
	case regV1CallsIDMute.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CallsIDMutePost(ctx, m)
		requestType = "/v1/calls/<call-id>/mute"

	// DELETE /calls/<call-id>/mute
	case regV1CallsIDMute.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1CallsIDMuteDelete(ctx, m)
		requestType = "/v1/calls/<call-id>/mute"

	// POST /calls/<call-id>/moh
	case regV1CallsIDMOH.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CallsIDMOHPost(ctx, m)
		requestType = "/v1/calls/<call-id>/moh"

	// DELETE /calls/<call-id>/moh
	case regV1CallsIDMOH.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1CallsIDMOHDelete(ctx, m)
		requestType = "/v1/calls/<call-id>/moh"

	// POST /calls/<call-id>/silence
	case regV1CallsIDSilence.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CallsIDSilencePost(ctx, m)
		requestType = "/v1/calls/<call-id>/silence"

	// DELETE /calls/<call-id>/silence
	case regV1CallsIDSilence.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1CallsIDSilenceDelete(ctx, m)
		requestType = "/v1/calls/<call-id>/silence"

	// GET /calls
	case regV1CallsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1CallsGet(ctx, m)
		requestType = "/v1/calls"

	// POST /calls
	case regV1Calls.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CallsPost(ctx, m)
		requestType = "/v1/calls"

	////////////
	// channels
	////////////
	// POST /channels/<channel-id>/health-check
	case regV1ChannelsIDHealth.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ChannelsIDHealthPost(ctx, m)
		requestType = "/v1/channels/<channel-id>/health-check"

	//////////////
	// confbridges
	//////////////

	// DELETE /confbridges/<confbridge-id>/calls/<call-id>
	case regV1ConfbridgesIDCallsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1ConfbridgesIDCallsIDDelete(ctx, m)
		requestType = "/v1/confbridges/<confbridge-id>/calls/<call-id>"

	// POST /confbridges/<confbridge-id>/calls/<call-id>
	case regV1ConfbridgesIDCallsID.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConfbridgesIDCallsIDPost(ctx, m)
		requestType = "/v1/confbridges/<confbridge-id>/calls/<call-id>"

	// GET /confbridges/<confbridge-id>
	case regV1ConfbridgesID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ConfbridgesIDGet(ctx, m)
		requestType = "/v1/confbridges/<confbridge-id>"

	// DELETE /confbridges/<confbridge-id>
	case regV1ConfbridgesID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1ConfbridgesIDDelete(ctx, m)
		requestType = "/v1/confbridges/<confbridge-id>"

	// POST /confbridges
	case regV1Confbridges.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConfbridgesPost(ctx, m)
		requestType = "/v1/confbridges"

	// POST /confbridges/<confbridge-id>/external-media
	case regV1ConfbridgesIDExternalMedia.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConfbridgesIDExternalMediaPost(ctx, m)
		requestType = "/v1/confbridges/<confbridge-id>/external-media"

	// DELETE /confbridges/<confbridge-id>/external-media
	case regV1ConfbridgesIDExternalMedia.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1ConfbridgesIDExternalMediaDelete(ctx, m)
		requestType = "/v1/confbridges/<confbridge-id>/external-media"

	// POST /confbridges/<confbridge-id>/recording_start
	case regV1ConfbridgesIDRecordingStart.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConfbridgesIDRecordingStartPost(ctx, m)
		requestType = "/v1/confbridges/<confbridge-id>/recording_start"

	// POST /confbridges/<confbridge-id>/recording_stop
	case regV1ConfbridgesIDRecordingStop.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConfbridgesIDRecordingStopPost(ctx, m)
		requestType = "/v1/confbridges/<confbridge-id>/recording_stop"

	// POST /confbridges/<confbridge-id>/flags
	case regV1ConfbridgesIDFlags.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConfbridgesIDFlagsPost(ctx, m)
		requestType = "/v1/confbridges/<confbridge-id>/flags"

	// DELETE /confbridges/<confbridge-id>/flags
	case regV1ConfbridgesIDFlags.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1ConfbridgesIDFlagsDelete(ctx, m)
		requestType = "/v1/confbridges/<confbridge-id>/flags"

	// POST /confbridges/<confbridge-id>/terminate
	case regV1ConfbridgesIDTerminate.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConfbridgesIDTerminatePost(ctx, m)
		requestType = "/v1/confbridges/<confbridge-id>/terminate"

	// POST /confbridges/<confbridge-id>/ring
	case regV1ConfbridgesIDRing.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConfbridgesIDRingPost(ctx, m)
		requestType = "/v1/confbridges/<confbridge-id>/ring"

	// POST /confbridges/<confbridge-id>/answer
	case regV1ConfbridgesIDAnswer.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ConfbridgesIDAnswerPost(ctx, m)
		requestType = "/v1/confbridges/<confbridge-id>/answer"

	////////////////////
	// external-medias
	////////////////////
	// POST /external-medias
	case regV1ExternalMedias.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ExternalMediasPost(ctx, m)
		requestType = "/v1/external-medias"

	// GET /external-medias
	case regV1ExternalMediasGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ExternalMediasGet(ctx, m)
		requestType = "/v1/external-medias"

	// GET /external-medias/<external-media-id>
	case regV1ExternalMediasID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ExternalMediasIDGet(ctx, m)
		requestType = "/v1/external-medias/<external-media-id>"

	// DELETE /external-medias/<external-media-id>
	case regV1ExternalMediasID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1ExternalMediasIDDelete(ctx, m)
		requestType = "/v1/external-medias/<external-media-id>"

	//////////////
	// groupcalls
	//////////////
	// POST /groupcalls
	case regV1Groupcalls.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1GroupcallsPost(ctx, m)
		requestType = "/v1/groupcalls"

	// GET /groupcalls
	case regV1GroupcallsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1GroupcallsGet(ctx, m)
		requestType = "/v1/groupcalls"

	// GET /groupcalls/<groupcall-id>
	case regV1GroupcallsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1GroupcallsIDGet(ctx, m)
		requestType = "/v1/groupcalls/<groupcall-id>"

	// DELETE /groupcalls/<groupcall-id>
	case regV1GroupcallsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1GroupcallsIDDelete(ctx, m)
		requestType = "/v1/groupcalls/<groupcall-id>"

	// POST /groupcalls/<groupcall-id>/hangup
	case regV1GroupcallsIDHangup.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1GroupcallsIDHangupPost(ctx, m)
		requestType = "/v1/groupcalls/<groupcall-id>/hangup"

	// POST /groupcalls/<groupcall-id>/hangup_groupcall
	case regV1GroupcallsIDHangupGroupcall.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1GroupcallsIDHangupGroupcallPost(ctx, m)
		requestType = "/v1/groupcalls/<groupcall-id>/hangup_groupcall"

	// POST /groupcalls/<groupcall-id>/hangup_call
	case regV1GroupcallsIDHangupCall.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1GroupcallsIDHangupCallPost(ctx, m)
		requestType = "/v1/groupcalls/<groupcall-id>/hangup_call"

	// POST /groupcalls/<groupcall-id>/answer_groupcall_id
	case regV1GroupcallsIDAnswerGroupcallID.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1GroupcallsIDAnswerGroupcallIDPost(ctx, m)
		requestType = "/v1/groupcalls/<groupcall-id>/answer_groupcall_id"

	//////////////
	// recovery
	//////////////
	// POST /recovery
	case regV1Recovery.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1RecoveryPost(ctx, m)
		requestType = "/v1/recovery"

	//////////////
	// recordings
	//////////////
	// GET /recordings
	case regV1RecordingsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1RecordingsGet(ctx, m)
		requestType = "/v1/recordings"

	// POST /recordings
	case regV1Recordings.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1RecordingsPost(ctx, m)
		requestType = "/v1/recordings"

	// GET /recordings/<recording-id>
	case regV1RecordingsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1RecordingsIDGet(ctx, m)
		requestType = "/v1/recordings/<recording-id>"

	// DELETE /recordings/<recording-id>
	case regV1RecordingsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1RecordingsIDDelete(ctx, m)
		requestType = "/v1/recordings/<recording-id>"

	// POST /recordings/<recording-id>/stop
	case regV1RecordingsIDStop.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1RecordingsIDStopPost(ctx, m)
		requestType = "/v1/recordings/<recording-id>/stop"

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		log.Errorf("Could not find corresponded message handler. method: %s, uri: %s", m.Method, m.URI)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}

	if err != nil {
		log.Errorf("Could not handle the request message correctly. method: %s, uri: %s, err: %v", m.Method, m.URI, err)
		response = simpleResponse(400)
		err = nil
	}

	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	return response, err
}
