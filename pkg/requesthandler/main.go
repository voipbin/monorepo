package requesthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package requesthandler -destination ./mock_requesthandler_requesthandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
)

// contents type
var (
	ContentTypeText = "text/plain"
	ContentTypeJSON = "application/json"
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
	metricsNamespace = "flow_manager"

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
	resourceCallCall       resource = "call/calls"
	resourceCallConference resource = "call/conferences"

	resourceFlowFlows resource = "flows"

	resourceNumberNumberFlows resource = "number/number_flows"

	resourceTranscribeCallRecordings resource = "transcrbice/call_recordings"
	resourceTranscribeStreamings     resource = "transcrbice/streamings"
)

func init() {
	prometheus.MustRegister(
		promRequestProcessTime,
	)
}

// RequestHandler intreface for ARI request handler
type RequestHandler interface {
	////// call-manager
	// call
	CMCallAddChainedCall(callID uuid.UUID, chainedCallID uuid.UUID) error
	CMCallCreate(userID uint64, flowID uuid.UUID, source, destination address.Address) (*call.Call, error)
	CMCallGet(callID uuid.UUID) (*call.Call, error)
	CMCallHangup(callID uuid.UUID) (*call.Call, error)

	// conference
	CMConferenceCreate(userID uint64, conferenceType conference.Type, name string, detail string, timeout int) (*conference.Conference, error)
	CMConferenceDelete(conferenceID uuid.UUID) error
	CMConferenceGet(conferenceID uuid.UUID) (*conference.Conference, error)

	//// conference-manager
	// conferences
	CFConferenceCreate(userID uint64, conferenceType cfconference.Type, name string, detail string, timeout int) (*cfconference.Conference, error)
	CFConferenceDelete(conferenceID uuid.UUID) error
	CFConferenceGet(conferenceID uuid.UUID) (*cfconference.Conference, error)

	////// flow-manager
	// FlowActionGet(flowID, actionID uuid.UUID) (*action.Action, error)
	FMFlowCreate(userID uint64, name, detail string, actions []action.Action, persist bool) (*flow.Flow, error)

	////// number-manager
	// number_flows
	NMNumberFlowDelete(flowID uuid.UUID) error

	////// transcribe-manager
	// call_recordings
	TMCallRecordingPost(callID uuid.UUID, language, webhookURI, webhookMethod string, timeout, delay int) error
	TMStreamingsPost(callID uuid.UUID, language, webhookURI, webhookMethod string) (*transcribe.Transcribe, error)
}

type requestHandler struct {
	sock rabbitmqhandler.Rabbit

	exchangeDelay string

	queueCall       string
	queueFlow       string
	queueNumber     string
	queueTranscribe string
	queueConference string
}

// NewRequestHandler create RequesterHandler
func NewRequestHandler(
	sock rabbitmqhandler.Rabbit,
	exchangeDelay string,
	queueCall string,
	queueFlow string,
	queueNumber string,
	queueTranscribe string,
	queueConference string,
) RequestHandler {
	h := &requestHandler{
		sock: sock,

		exchangeDelay:   exchangeDelay,
		queueCall:       queueCall,
		queueFlow:       queueFlow,
		queueNumber:     queueNumber,
		queueTranscribe: queueTranscribe,
		queueConference: queueConference,
	}

	return h
}

// sendRequest sends a request to the given destination.
func (r *requestHandler) sendRequest(queue string, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	log.WithFields(log.Fields{
		"queue":     queue,
		"method":    method,
		"uri":       uri,
		"data_type": dataType,
		"delayed":   delayed,
	}).Debugf("Sending a request. queue: %s, method: %s, uri: %s", queue, method, uri)

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
		if err := r.sendDelayedRequest(ctx, r.exchangeDelay, queue, resource, delayed, req); err != nil {
			return nil, fmt.Errorf("could not publish the delayed request. err: %v", err)
		}
		return nil, nil

	default:
		res, err := r.sendDirectRequest(ctx, queue, resource, req)
		if err != nil {
			return nil, fmt.Errorf("could not publish the RPC. err: %v", err)
		}

		log.WithFields(log.Fields{
			"method": method,
			"uri":    uri,
			"res":    res,
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
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestFlow(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout int, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueFlow, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestCall send a request to the call-manager and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestCall(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueCall, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestNumber send a request to the number-manager and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestNumber(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueNumber, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestTranscribe send a request to the transcribe-manager and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestTranscribe(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueTranscribe, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestConference send a request to the conference-manager and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestConference(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data []byte) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueConference, uri, method, resource, timeout, delayed, dataType, data)
}
