package requesthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package requesthandler -destination ./mock_requesthandler_requesthandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
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

var (
	metricsNamespace = "conference_manager"

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
	resourceCFConferences resource = "cf/conferences"

	resourceCMConfbridges resource = "cm/confbridges"

	resourceWMWebhooks resource = "wm/webhooks"

	resourceFlowsActions resource = "flows/actions"
	resourceFMFlows      resource = "fm/flows"
)

func init() {
	prometheus.MustRegister(
		promRequestProcessTime,
	)
}

// RequestHandler intreface for ARI request handler
type RequestHandler interface {

	// conference manager conferences
	CFConferencesIDDelete(conferenceID uuid.UUID, delay int) error

	// cm confbridges
	CMConfbridgesPost(conferenceID uuid.UUID) (*confbridge.Confbridge, error)
	CMConfbridgesIDDelete(conferenceID uuid.UUID) error
	CMConfbridgesIDCallsIDDelete(conferenceID uuid.UUID, callID uuid.UUID) error
	CMConfbridgesIDCallsIDPost(conferenceID uuid.UUID, callID uuid.UUID) error

	// fm flows
	FMFlowCreate(f *fmflow.Flow) (*fmflow.Flow, error)
	FMFlowDelete(flowID uuid.UUID) error
	FMFlowGet(flowID uuid.UUID) (*fmflow.Flow, error)
	FMFlowGets(userID uint64, pageToken string, pageSize uint64) ([]fmflow.Flow, error)
	FMFlowUpdate(f *fmflow.Flow) (*fmflow.Flow, error)

	// fm actions
	FlowActionGet(flowID, actionID uuid.UUID) (*action.Action, error)
	FlowActvieFlowPost(callID, flowID uuid.UUID) (*activeflow.ActiveFlow, error)
	FlowActvieFlowNextGet(callID, actionID uuid.UUID) (*action.Action, error)

	// wm webhooks
	WMWebhookPOST(webhookMethod, webhookURI, dataType, messageType string, messageData []byte) error
}

type requestHandler struct {
	sock rabbitmqhandler.Rabbit

	exchangeDelay string

	queueConference string
	queueCall       string
	queueFlow       string
	queueWebhook    string
}

// NewRequestHandler create RequesterHandler
func NewRequestHandler(sock rabbitmqhandler.Rabbit, exchangeDelay, queueConference, queueCall, queueFlow, queueWebhook string) RequestHandler {
	h := &requestHandler{
		sock: sock,

		exchangeDelay: exchangeDelay,

		queueConference: queueConference,
		queueCall:       queueCall,
		queueFlow:       queueFlow,
		queueWebhook:    queueWebhook,
	}

	return h
}

//nolint:deadcode,unused // this is ok
func uriUnescape(u string) string {
	res, err := url.QueryUnescape(u)
	if err != nil {
		return "could not unescape the url"
	}

	return res
}

// sendRequest sends a request to the given destination.
func (r *requestHandler) sendRequest(queue string, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	// creat a request message
	req := &rabbitmqhandler.Request{
		URI:      uri,
		Method:   method,
		DataType: dataType,
		Data:     data,
	}

	log := logrus.WithFields(logrus.Fields{
		"queue":   queue,
		"delayed": delayed,
		"request": req,
	})
	log.Debugf("Sending a request. queue: %s, method: %s, uri: %s", queue, method, uri)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))
	defer cancel()

	switch {
	case delayed > 0:
		// send scheduled message.
		// we don't expect the response message here.
		if err := r.sendDelayedRequest(ctx, r.exchangeDelay, queue, resource, delayed, req); err != nil {
			return nil, fmt.Errorf("could not publish the delayed request. err: %v", err)
		}
		return nil, nil

	default:
		res, err := r.sendDirectRequest(ctx, queue, resource, req)
		if err != nil {
			return nil, fmt.Errorf("could not publish the RPC. err: %v", err)
		}

		log.WithFields(logrus.Fields{
			"response": res,
		}).Debugf("Received result. queue: %s, method: %s, uri: %s, status_code: %d", queue, method, uri, res.StatusCode)
		return res, nil
	}
}

// sendDirectRequest sends the request to the target without delay
func (r *requestHandler) sendDirectRequest(ctx context.Context, target string, resource resource, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	start := time.Now()
	res, err := r.sock.PublishRPC(ctx, target, req)
	elapsed := time.Since(start)
	promRequestProcessTime.WithLabelValues(target, string(resource), string(req.Method)).Observe(float64(elapsed.Milliseconds()))

	return res, err
}

// sendDelayedRequest sends the delayed request to the target
// delay unit is millisecond.
func (r *requestHandler) sendDelayedRequest(ctx context.Context, target string, queue string, resource resource, delay int, req *rabbitmqhandler.Request) error {

	start := time.Now()
	err := r.sock.PublishExchangeDelayedRequest(r.exchangeDelay, queue, req, delay)
	elapsed := time.Since(start)
	promRequestProcessTime.WithLabelValues(target, string(resource), string(req.Method)).Observe(float64(elapsed.Milliseconds()))

	return err
}

// sendRequestFlow send a request to the flow-manager and return the response
func (r *requestHandler) sendRequestFlow(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueFlow, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestWM send a request to the webhook-manager and return the response
func (r *requestHandler) sendRequestWM(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueWebhook, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestCall send a request to the call-manager and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestCall(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueCall, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestConference send a request to the conference-manager and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestConference(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueConference, uri, method, resource, timeout, delayed, dataType, data)
}
