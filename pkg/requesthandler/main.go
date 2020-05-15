package requesthandler

//go:generate mockgen -destination ./mock_requesthandler_requesthandler.go -package requesthandler gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler RequestHandler

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
	rabbitmq "gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"

	uuid "github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// contents type
var (
	ContentTypeText = "text/plain"
	ContentTypeJSON = "application/json"
)

const requestTimeoutDefault int64 = 3 // default request timeout

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

	resourceAstChannels         resource = "ast/channels"
	resourceAstChannelsAnswer   resource = "ast/channels/answer"
	resourceAstChannelsContinue resource = "ast/channels/continue"
	resourceAstChannelsHangup   resource = "ast/channels/hangup"
	resourceAstChannelsSnoop    resource = "ast/channels/snoop"
	resourceAstChannelsVar      resource = "ast/channels/var"

	resourceFlowsActions resource = "flows/actions"
)

func init() {
	prometheus.MustRegister(
		promRequestProcessTime,
	)
}

// RequestHandler intreface for ARI request handler
type RequestHandler interface {
	// asterisk bridges
	AstBridgeAddChannel(asteriskID, bridgeID, channelID, role string, absorbDTMF, mute bool) error
	AstBridgeCreate(asteriskID, bridgeID, bridgeName string, bridgeType bridge.Type) error
	AstBridgeDelete(asteriskID, bridgeID string) error
	AstBridgeRemoveChannel(asteriskID, bridgeID, channelID string) error

	// asterisk channels
	AstChannelAnswer(asteriskID, channelID string) error
	AstChannelContinue(asteriskID, channelID, context, ext string, pri int, label string) error
	AstChannelCreateSnoop(asteriskID, channelID, snoopID, appArgs string, spy, whisper channel.SnoopDirection) error
	AstChannelHangup(asteriskID, channelID string, code ari.ChannelCause) error
	AstChannelVariableSet(asteriskID, channelID, variable, value string) error

	// flow actions
	FlowActionGet(flowID, actionID uuid.UUID) (*action.Action, error)
}

type requestHandler struct {
	sock rabbitmq.Rabbit
}

// NewRequestHandler create RequesterHandler
func NewRequestHandler(sock rabbitmq.Rabbit) RequestHandler {
	h := &requestHandler{
		sock: sock,
	}

	return h
}

// SendARIRequest send a request to the Asterisk-proxy and return the response
func (r *requestHandler) sendRequestAst(asteriskID, uri string, method rabbitmq.RequestMethod, resource resource, timeout int64, dataType, data string) (*rabbitmq.Response, error) {
	log.WithFields(log.Fields{
		"asterisk_id": asteriskID,
		"method":      method,
		"uri":         uri,
		"data_type":   dataType,
	}).Debugf("Sending ARI request. data: %s", data)

	// creat a request message
	m := &rabbitmq.Request{
		URI:      uri,
		Method:   method,
		DataType: dataType,
		Data:     data,
	}

	// create target
	var requestTargetPrefix = "asterisk_ari_request"
	target := fmt.Sprintf("%s-%s", requestTargetPrefix, asteriskID)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
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
func (r *requestHandler) sendRequestFlow(uri string, method rabbitmq.RequestMethod, resource resource, timeout int64, dataType, data string) (*rabbitmq.Response, error) {
	log.WithFields(log.Fields{
		"uri":       uri,
		"method":    method,
		"data_type": dataType,
		"data":      data,
	}).Debugf("Sending request to Flow. data: %s", data)

	// creat a request message
	m := &rabbitmq.Request{
		URI:      uri,
		Method:   method,
		DataType: dataType,
		Data:     data,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	target := "flow_manager-request"
	res, err := r.sendRequest(ctx, target, resource, m)
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

// sendRequest sends the request to the target
func (r *requestHandler) sendRequest(ctx context.Context, target string, resource resource, req *rabbitmq.Request) (*rabbitmq.Response, error) {

	start := time.Now()
	res, err := r.sock.PublishRPC(ctx, target, req)
	elapsed := time.Since(start)
	promRequestProcessTime.WithLabelValues(target, string(resource), string(req.Method)).Observe(float64(elapsed.Milliseconds()))

	return res, err
}
