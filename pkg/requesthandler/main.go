package requesthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package requesthandler -destination ./mock_requesthandler_requesthandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	smbucketrecording "gitlab.com/voipbin/bin-manager/storage-manager.git/models/bucketrecording"
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
	metricsNamespace = "transcribe_manager"

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

	resourceStorageRecording resource = "storage/recordings"

	resourceWebhookWebhooks resource = "webhook/webhooks"
)

func init() {
	prometheus.MustRegister(
		promRequestProcessTime,
	)
}

// RequestHandler intreface for ARI request handler
type RequestHandler interface {
	CMCallExternalMedia(
		callID uuid.UUID,
		externalHost string,
		encapsulation string,
		transport string,
		connectionType string,
		format string,
		direction string,
		data string,
	) (addrIP string, addrPort int, errRet error)
	CMCallGet(callID uuid.UUID) (*cmcall.Call, error)
	CMRecordingGet(id uuid.UUID) (*cmrecording.Recording, error)

	SMRecordingGet(id uuid.UUID) (*smbucketrecording.BucketRecording, error)

	WMWebhookPost(reqMethod, reqURI, reqDataType string, reqData []byte) error
}

type requestHandler struct {
	sock rabbitmqhandler.Rabbit

	exchangeDelay string

	queueCall    string
	queueStorage string
	queueWebhook string
}

// NewRequestHandler create RequesterHandler
func NewRequestHandler(
	sock rabbitmqhandler.Rabbit,
	exchangeDelay string,
	queueCall, queueStorage, queueWebhook string) RequestHandler {
	h := &requestHandler{
		sock: sock,

		exchangeDelay: exchangeDelay,

		queueCall:    queueCall,
		queueStorage: queueStorage,
		queueWebhook: queueWebhook,
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

// sendRequestStorage send a request to the storage-manager and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestStorage(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueStorage, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestWebhook send a request to the webhook-manager and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestWebhook(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueWebhook, uri, method, resource, timeout, delayed, dataType, data)
}

// sendRequestCall send a request to the Asterisk-proxy and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestCall(uri string, method rabbitmqhandler.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*rabbitmqhandler.Response, error) {

	return r.sendRequest(r.queueCall, uri, method, resource, timeout, delayed, dataType, data)
}
