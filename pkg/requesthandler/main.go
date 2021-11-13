package requesthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package requesthandler -destination ./mock_requesthandler_requesthandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	nmavailablenumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"
	nmnumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	rmdomain "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	rmextension "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
	smbucketrecording "gitlab.com/voipbin/bin-manager/storage-manager.git/models/bucketrecording"
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
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
	metricsNamespace = "api_manager"

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

// list of resources
const (
	resourceCallCall       resource = "call/calls"
	resourceCallRecordings resource = "call/recordings"

	resourceConferenceConference resource = "conference/conferences"

	resourceFlowFlows resource = "flow/flows"

	resourceNumberAvailableNumbers resource = "number/available_numbers"
	resourceNumberNumbers          resource = "number/numbers"

	resourceRegistrarDomains    resource = "registrar/domains"
	resourceRegistrarExtensions resource = "registrar/extensions"

	resourceStorageRecording resource = "storage/recordings"
)

func init() {
	prometheus.MustRegister(
		promRequestProcessTime,
	)
}

// RequestHandler intreface for ARI request handler
type RequestHandler interface {

	// call: call
	CMCallCreate(userID uint64, flowID uuid.UUID, source, destination cmaddress.Address) (*cmcall.Call, error)
	CMCallDelete(callID uuid.UUID) error
	CMCallGet(callID uuid.UUID) (*cmcall.Call, error)
	CMCallGets(userID uint64, pageToken string, pageSize uint64) ([]cmcall.Call, error)

	// call: recordings
	CMRecordingGet(id uuid.UUID) (*cmrecording.Recording, error)
	CMRecordingGets(userID uint64, size uint64, token string) ([]cmrecording.Recording, error)

	// conference: conferences
	CFConferenceCreate(userID uint64, conferenceType cfconference.Type, name, detail, webhookURI string, preActions, postActions []fmaction.Action) (*cfconference.Conference, error)
	CFConferenceDelete(conferenceID uuid.UUID) error
	CFConferenceGet(conferenceID uuid.UUID) (*cfconference.Conference, error)
	CFConferenceGets(userID uint64, pageToken string, pageSize uint64, conferenceType string) ([]cfconference.Conference, error)
	CFConferenceUpdate(id uuid.UUID, name string, detail string, timeout int, webhookURI string, preActions, postActions []fmaction.Action) (*cfconference.Conference, error)
	CFConferenceKick(conferenceID, callID uuid.UUID) error

	// flow: flow
	FMFlowCreate(f *fmflow.Flow) (*fmflow.Flow, error)
	FMFlowDelete(flowID uuid.UUID) error
	FMFlowGet(flowID uuid.UUID) (*fmflow.Flow, error)
	FMFlowGets(userID uint64, pageToken string, pageSize uint64) ([]fmflow.Flow, error)
	FMFlowUpdate(f *fmflow.Flow) (*fmflow.Flow, error)

	// number: availalbe_number
	NMAvailableNumbersGet(userID uint64, pageSize uint64, countryCode string) ([]nmavailablenumber.AvailableNumber, error)

	// number: order number
	NMNumberCreate(userID uint64, numb string) (*nmnumber.Number, error)
	NMNumberDelete(id uuid.UUID) (*nmnumber.Number, error)
	NMNumberGet(numberID uuid.UUID) (*nmnumber.Number, error)
	NMNumberGets(userID uint64, pageToken string, pageSize uint64) ([]nmnumber.Number, error)
	NMNumberUpdate(num *nmnumber.Number) (*nmnumber.Number, error)

	// registrar: domain
	RMDomainCreate(userID uint64, domainName, name, detail string) (*rmdomain.Domain, error)
	RMDomainDelete(domainID uuid.UUID) error
	RMDomainGet(domainID uuid.UUID) (*rmdomain.Domain, error)
	RMDomainGets(userID uint64, pageToken string, pageSize uint64) ([]rmdomain.Domain, error)
	RMDomainUpdate(f *rmdomain.Domain) (*rmdomain.Domain, error)

	// registrar: extension
	RMExtensionCreate(e *rmextension.Extension) (*rmextension.Extension, error)
	RMExtensionDelete(extensionID uuid.UUID) error
	RMExtensionGet(extensionID uuid.UUID) (*rmextension.Extension, error)
	RMExtensionGets(domainID uuid.UUID, pageToken string, pageSize uint64) ([]rmextension.Extension, error)
	RMExtensionUpdate(f *rmextension.Extension) (*rmextension.Extension, error)

	// storage: recording
	SMRecordingGet(id uuid.UUID) (*smbucketrecording.BucketRecording, error)

	// transcribe: recording
	TMRecordingPost(id uuid.UUID, language string) (*tmtranscribe.Transcribe, error)
}

type requestHandler struct {
	sock rabbitmqhandler.Rabbit

	exchangeDelay string

	queueRequestCall       string
	queueRequesstFlow      string
	queueRequestStorage    string
	queueRequestRegistrar  string
	queueRequestNumber     string
	queueRequestTranscribe string
	queueRequestConference string
}

// NewRequestHandler create RequesterHandler
func NewRequestHandler(
	sock rabbitmqhandler.Rabbit,
	exchangeDelay string,
	queueCall string,
	queueFlow string,
	queueStorage string,
	queueRegistrar string,
	queueNumber string,
	queueTranscode string,
	queueConference string,
) RequestHandler {
	h := &requestHandler{
		sock: sock,

		exchangeDelay:          exchangeDelay,
		queueRequestCall:       queueCall,
		queueRequesstFlow:      queueFlow,
		queueRequestStorage:    queueStorage,
		queueRequestRegistrar:  queueRegistrar,
		queueRequestNumber:     queueNumber,
		queueRequestTranscribe: queueTranscode,
		queueRequestConference: queueConference,
	}

	return h
}

// sendRequest sends a request to the given destination.
func (r *requestHandler) sendRequest(queue string, uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	log := logrus.WithFields(logrus.Fields{
		"queue":     queue,
		"method":    method,
		"uri":       uri,
		"data_type": dataType,
		"delayed":   delayed,
	})
	log.Debugf("Sending a request. queue: %s, method: %s, uri: %s", queue, method, uri)

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

		log.WithFields(logrus.Fields{
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
func (r *requestHandler) sendRequestFlow(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueRequesstFlow, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestCall send a request to the Asterisk-proxy and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestCall(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueRequestCall, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestStorage send a request to the storage-manager and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestStorage(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueRequestStorage, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestRegistrar send a request to the registrar-manager and return the response
func (r *requestHandler) sendRequestRegistrar(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueRequestRegistrar, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestNumber send a request to the number-manager and return the response
func (r *requestHandler) sendRequestNumber(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueRequestNumber, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestTranscribe send a request to the transcribe-manager and return the response
func (r *requestHandler) sendRequestTranscribe(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueRequestTranscribe, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestConference send a request to the conference-manager and return the response
func (r *requestHandler) sendRequestConference(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueRequestConference, uri, method, resource, timeout, delayed, dataType, data)
}
