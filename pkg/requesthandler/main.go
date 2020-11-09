package requesthandler

//go:generate mockgen -destination ./mock_requesthandler_requesthandler.go -package requesthandler -source ./main.go RequestHandler

import (
	"context"
	"fmt"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/action"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/activeflow"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/channel"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// contents type
var (
	ContentTypeText = "text/plain"
	ContentTypeJSON = "application/json"
)

// group asterisk id
var (
	AsteriskIDCall       = "call"       // asterisk-call
	AsteriskIDConference = "conference" // asterisk-conference
)

const requestTimeoutDefault int = 3 // default request timeout

// delay units
const (
	DelayNow    int = 0
	DelaySecond int = 1000
	DelayMinute int = DelaySecond * 60
	DelayHour   int = DelayMinute * 60
)

// default stasis application name.
// normally, we don't need to use this, because proxy will set this automatically.
// but, some of Asterisk ARI required application name. this is for that.
const defaultAstStasisApp = "voipbin"

var (
	metricsNamespace = "call_manager"

	promRequestProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "request_process_time",
			Help:      "Process time of send/receiv requests",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"target", "resource", "method"},
	)
)

type resource string

const (
	resourceAstBridges              resource = "ast/bridges"
	resourceAstBridgesAddChannel    resource = "ast/bridges/addchannel"
	resourceAstBridgesRemoveChannel resource = "ast/bridges/removechannel"

	resourceAstAMI resource = "ast/ami"

	resourceAstChannels         resource = "ast/channels"
	resourceAstChannelsAnswer   resource = "ast/channels/answer"
	resourceAstChannelsContinue resource = "ast/channels/continue"
	resourceAstChannelsDial     resource = "ast/channels/dial"
	resourceAstChannelsHangup   resource = "ast/channels/hangup"
	resourceAstChannelsPlay     resource = "ast/channels/play"
	resourceAstChannelsSnoop    resource = "ast/channels/snoop"
	resourceAstChannelsVar      resource = "ast/channels/var"

	resourceCallCalls              resource = "call/calls"
	resourceCallCallsActionNext    resource = "call/calls/action-next"
	resourceCallCallsActionTimeout resource = "call/calls/action-timeout"
	resourceCallCallsHealth        resource = "call/calls/health"
	resourceCallChannelsHealth     resource = "call/channels/health"

	resourceFlowsActions resource = "flows/actions"

	resourceTTSSpeeches resource = "tts/speeches"
)

func init() {
	prometheus.MustRegister(
		promRequestProcessTime,
	)
}

// RequestHandler intreface for ARI request handler
type RequestHandler interface {

	// asterisk AMI
	AstAMIRedirect(asteriskID, channelID, context, exten, priority string) error

	// asterisk bridges
	AstBridgeAddChannel(asteriskID, bridgeID, channelID, role string, absorbDTMF, mute bool) error
	AstBridgeCreate(asteriskID, bridgeID, bridgeName string, bridgeType []bridge.Type) error
	AstBridgeDelete(asteriskID, bridgeID string) error
	AstBridgeGet(asteriskID, bridgeID string) (*bridge.Bridge, error)
	AstBridgeRemoveChannel(asteriskID, bridgeID, channelID string) error

	// asterisk channels
	AstChannelAnswer(asteriskID, channelID string) error
	AstChannelContinue(asteriskID, channelID, context, ext string, pri int, label string) error
	AstChannelCreate(asteriskID, channelID, appArgs, endpoint, otherChannelID, originator, formats string, variables map[string]string) error
	AstChannelCreateSnoop(asteriskID, channelID, snoopID, appArgs string, spy, whisper channel.SnoopDirection) error
	AstChannelDial(asteriskID, channelID, caller string, timeout int) error
	AstChannelDTMF(asteriskID, channelID string, digit string, duration, before, between, after int) error
	AstChannelGet(asteriskID, channelID string) (*channel.Channel, error)
	AstChannelHangup(asteriskID, channelID string, code ari.ChannelCause) error
	AstChannelPlay(asteriskID string, channelID string, actionID uuid.UUID, medias []string, lang string) error
	AstChannelVariableSet(asteriskID, channelID, variable, value string) error

	// call
	CallCallHealth(id uuid.UUID, delay, retryCount int) error
	CallCallActionNext(id uuid.UUID) error
	CallCallActionTimeout(id uuid.UUID, delay int, a *action.Action) error
	CallChannelHealth(asteriskID, channelID string, delay, retryCount, retryCountMax int) error

	// conference
	CallConferenceTerminate(conferenceID uuid.UUID, reason string, delay int) error

	// flow actions
	FlowActionGet(flowID, actionID uuid.UUID) (*action.Action, error)
	FlowActvieFlowPost(callID, flowID uuid.UUID) (*activeflow.ActiveFlow, error)
	FlowActvieFlowNextGet(callID, actionID uuid.UUID) (*action.Action, error)

	// tts speeches
	TTSSpeechesPOST(text, gender, language string) (string, error)
}

type requestHandler struct {
	sock rabbitmqhandler.Rabbit

	exchangeDelay string

	queueCall string
	queueFlow string
	queueTTS  string
}

// NewRequestHandler create RequesterHandler
func NewRequestHandler(sock rabbitmqhandler.Rabbit, exchangeDelay, queueCall, queueFlow, queueTTS string) RequestHandler {
	h := &requestHandler{
		sock: sock,

		exchangeDelay: exchangeDelay,
		queueCall:     queueCall,
		queueFlow:     queueFlow,
		queueTTS:      queueTTS,
	}

	return h
}

// SendARIRequest send a request to the Asterisk-proxy and return the response
func (r *requestHandler) sendRequestAst(asteriskID, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {
	log.WithFields(log.Fields{
		"asterisk_id": asteriskID,
		"method":      method,
		"uri":         uri,
		"data_type":   dataType,
	}).Debugf("Sending ARI request. data: %s", data)

	// creat a request message
	m := &rabbitmqhandler.Request{
		URI:      uri,
		Method:   method,
		DataType: dataType,
		Data:     data,
	}

	// create target
	target := fmt.Sprintf("asterisk.%s.request", asteriskID)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))
	defer cancel()

	res, err := r.sendRequest(ctx, target, resource, m)
	if err != nil {
		return nil, fmt.Errorf("could not publish the RPC. err: %v", err)
	}

	log.WithFields(log.Fields{
		"asterisk_id": asteriskID,
		"method":      method,
		"uri":         uri,
		"status_code": res.StatusCode,
	}).Debugf("Received result. data: %s", res.Data)

	return res, nil
}

// sendRequestFlow send a request to the flow-manager and return the response
func (r *requestHandler) sendRequestFlow(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {
	log.WithFields(log.Fields{
		"uri":       uri,
		"method":    method,
		"data_type": dataType,
		"data":      data,
	}).Debugf("Sending request to Flow. data: %s", data)

	// creat a request message
	m := &rabbitmqhandler.Request{
		URI:      uri,
		Method:   method,
		DataType: dataType,
		Data:     data,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))
	defer cancel()

	res, err := r.sendRequest(ctx, r.queueFlow, resource, m)
	if err != nil {
		return nil, fmt.Errorf("could not publish the RPC. err: %v", err)
	}

	log.WithFields(log.Fields{
		"uri":         uri,
		"method":      method,
		"status_code": res.StatusCode,
	}).Debugf("Received result. data: %s", res.Data)

	return res, nil
}

// sendRequestTTS send a request to the tts-manager and return the response
func (r *requestHandler) sendRequestTTS(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {
	log.WithFields(log.Fields{
		"uri":       uri,
		"method":    method,
		"data_type": dataType,
		"data":      data,
	}).Debugf("Sending request to TTS. data: %s", data)

	// creat a request message
	m := &rabbitmqhandler.Request{
		URI:      uri,
		Method:   method,
		DataType: dataType,
		Data:     data,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))
	defer cancel()

	res, err := r.sendRequest(ctx, r.queueTTS, resource, m)
	if err != nil {
		return nil, fmt.Errorf("could not publish the RPC. err: %v", err)
	}

	log.WithFields(log.Fields{
		"uri":         uri,
		"method":      method,
		"status_code": res.StatusCode,
	}).Debugf("Received result. data: %s", res.Data)

	return res, nil
}

// sendRequestCall send a request to the Asterisk-proxy and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestCall(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {
	log.WithFields(log.Fields{
		"method":    method,
		"uri":       uri,
		"data_type": dataType,
		"delayed":   delayed,
	}).Debugf("Sending request to call-manager. data: %s", data)

	// creat a request message
	req := &rabbitmqhandler.Request{
		URI:      uri,
		Method:   method,
		DataType: dataType,
		Data:     data,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))
	defer cancel()

	switch {
	case delayed > 0:
		// send scheduled message.
		// we don't expect the response message here.
		if err := r.sendDelayedRequest(ctx, r.exchangeDelay, resource, delayed, req); err != nil {
			return nil, fmt.Errorf("could not publish the delayed request. err: %v", err)
		}
		return nil, nil

	default:
		res, err := r.sendRequest(ctx, r.queueCall, resource, req)
		if err != nil {
			return nil, fmt.Errorf("could not publish the RPC. err: %v", err)
		}

		log.WithFields(log.Fields{
			"method":      method,
			"uri":         uri,
			"status_code": res.StatusCode,
		}).Debugf("Received result. data: %s", res.Data)
		return res, nil
	}
}

// sendRequest sends the request to the target
func (r *requestHandler) sendRequest(ctx context.Context, target string, resource resource, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	start := time.Now()
	res, err := r.sock.PublishRPC(ctx, target, req)
	elapsed := time.Since(start)
	promRequestProcessTime.WithLabelValues(target, string(resource), string(req.Method)).Observe(float64(elapsed.Milliseconds()))

	return res, err
}

// sendDelayedRequest sends the delayed request to the target
// delay unit is millisecond.
func (r *requestHandler) sendDelayedRequest(ctx context.Context, target string, resource resource, delay int, req *rabbitmqhandler.Request) error {

	start := time.Now()
	err := r.sock.PublishExchangeDelayedRequest(r.exchangeDelay, r.queueCall, req, delay)
	elapsed := time.Since(start)
	promRequestProcessTime.WithLabelValues(target, string(resource), string(req.Method)).Observe(float64(elapsed.Milliseconds()))

	return err
}
