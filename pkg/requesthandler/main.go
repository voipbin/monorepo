package requesthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package requesthandler -destination ./mock_requesthandler_requesthandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requesthandler/models/telnyx"
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

// telnyx
const (
	TelnyxToken string = "KEY017B6ED1E90D8FC5DB6ED95F1ACFE4F5_WzTaTxsXJCdwOviG4t1xMM"
)

// twilio
const (
	TwilioSID   string = "AC3300cb9426b78c9ce48db86a755166f0"
	TwilioToken string = "58c603e14220f52553be7769b209f423"
)

// default stasis application name.
// normally, we don't need to use this, because proxy will set this automatically.
// but, some of Asterisk ARI required application name. this is for that.
const defaultAstStasisApp = "voipbin"

var (
	metricsNamespace = "number_manager"

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
	resourceAstChannelsRecord   resource = "ast/channels/record"
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

	// telnyx
	TelnyxAvailableNumberGets(countryCode, locality, administrativeArea string, limit uint) ([]*telnyx.AvailableNumber, error)

	TelnyxNumberOrdersPost(numbers []string) (*telnyx.OrderNumber, error)

	TelnyxPhoneNumbersGet(size uint, tag, number string) ([]*telnyx.PhoneNumber, error)
	TelnyxPhoneNumbersIDGet(id string) (*telnyx.PhoneNumber, error)
	TelnyxPhoneNumbersIDDelete(id string) (*telnyx.PhoneNumber, error)
	TelnyxPhoneNumbersIDUpdateConnectionID(id string, connectionID string) (*telnyx.PhoneNumber, error)
}

type requestHandler struct {
	sock rabbitmqhandler.Rabbit

	exchangeDelay string

	queueCall      string
	queueFlow      string
	queueTTS       string
	queueRegistrar string
}

// NewRequestHandler create RequesterHandler
func NewRequestHandler(sock rabbitmqhandler.Rabbit, exchangeDelay string) RequestHandler {
	h := &requestHandler{
		sock: sock,

		exchangeDelay: exchangeDelay,
	}

	return h
}

func uriUnescape(u string) string {
	res, err := url.QueryUnescape(u)
	if err != nil {
		return "could not unescape the url"
	}

	return res
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
