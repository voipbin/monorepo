package requesthandler

//go:generate mockgen -destination ./mock_requesthandler_requesthandler.go -package requesthandler -source main.go RequestHandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/rabbitmq"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/rabbitmq/models"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/cmcall"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/cmconference"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/fmflow"
	// rabbitmq "gitlab.com/voipbin/bin-manager/api-manager.git/pkg/rabbitmq/models"
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

// default stasis application name.
// normally, we don't need to use this, because proxy will set this automatically.
// but, some of Asterisk ARI required application name. this is for that.
const defaultAstStasisApp = "voipbin"

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

const (
	resourceCallCall       resource = "call/calls"
	resourceCallConference resource = "call/conferences"

	resourceFlowActions resource = "flows/actions"
	resourceFlowFlows   resource = "flows"
)

func init() {
	prometheus.MustRegister(
		promRequestProcessTime,
	)
}

// RequestHandler intreface for ARI request handler
type RequestHandler interface {
	// call
	CMCallCreate(userID uint64, flowID uuid.UUID, source, destination cmcall.Address) (*cmcall.Call, error)
	CMCallGet(callID uuid.UUID) (*cmcall.Call, error)
	// CallCallHealth(id uuid.UUID, delay, retryCount int) error
	// CallCallActionNext(id uuid.UUID) error
	// CallCallActionTimeout(id uuid.UUID, delay int, a *action.Action) error
	// CallChannelHealth(asteriskID, channelID string, delay, retryCount, retryCountMax int) error

	// conference
	CMConferenceCreate(userID uint64, conferenceType cmconference.Type, name string, detail string) (*cmconference.Conference, error)
	CMConferenceDelete(conferenceID uuid.UUID) error
	CMConferenceGet(conferenceID uuid.UUID) (*cmconference.Conference, error)

	// flow actions
	// FlowActionGet(flowID, actionID uuid.UUID) (*action.Action, error)
	FMFlowCreate(userID uint64, id uuid.UUID, name, detail string, actions []action.Action, persist bool) (*fmflow.Flow, error)
}

type requestHandler struct {
	sock rabbitmq.Rabbit

	exchangeDelay string

	queueCall string
	queueFlow string
}

// Inject injects requesthandler to gin context
func Inject(sock rabbitmq.Rabbit, exchangeDelay, queueCall, queueFlow string) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestHandler := NewRequestHandler(sock, exchangeDelay, queueCall, queueFlow)

		c.Set("requestHandler", requestHandler)
		c.Next()
	}
}

// NewRequestHandler create RequesterHandler
func NewRequestHandler(sock rabbitmq.Rabbit, exchangeDelay, queueCall, queueFlow string) RequestHandler {
	h := &requestHandler{
		sock: sock,

		exchangeDelay: exchangeDelay,
		queueCall:     queueCall,
		queueFlow:     queueFlow,
	}

	return h
}

// sendRequestFlow send a request to the flow-manager and return the response
func (r *requestHandler) sendRequestFlow(uri string, method models.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*models.Response, error) {
	log.WithFields(log.Fields{
		"method":    method,
		"uri":       uri,
		"data_type": dataType,
		"delayed":   delayed,
	}).Debugf("Sending request to flow-manager. data: %s", data)

	// creat a request message
	req := &models.Request{
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
		if err := r.sendDelayedRequest(ctx, r.exchangeDelay, r.queueFlow, resource, delayed, req); err != nil {
			return nil, fmt.Errorf("could not publish the delayed request. err: %v", err)
		}
		return nil, nil

	default:
		res, err := r.sendRequest(ctx, r.queueFlow, resource, req)
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

// sendRequestCall send a request to the Asterisk-proxy and return the response
// timeout second
// delayed millisecond
func (r *requestHandler) sendRequestCall(uri string, method models.RequestMethod, resource resource, timeout, delayed int, dataType string, data json.RawMessage) (*models.Response, error) {
	log.WithFields(log.Fields{
		"method":    method,
		"uri":       uri,
		"data_type": dataType,
		"delayed":   delayed,
	}).Debugf("Sending request to call-manager. data: %s", data)

	// creat a request message
	req := &models.Request{
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
		if err := r.sendDelayedRequest(ctx, r.exchangeDelay, r.queueCall, resource, delayed, req); err != nil {
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
func (r *requestHandler) sendRequest(ctx context.Context, target string, resource resource, req *models.Request) (*models.Response, error) {

	start := time.Now()
	res, err := r.sock.PublishRPC(ctx, target, req)
	elapsed := time.Since(start)
	promRequestProcessTime.WithLabelValues(target, string(resource), string(req.Method)).Observe(float64(elapsed.Milliseconds()))

	return res, err
}

// sendDelayedRequest sends the delayed request to the target
// delay unit is millisecond.
func (r *requestHandler) sendDelayedRequest(ctx context.Context, target string, queue string, resource resource, delay int, req *models.Request) error {

	start := time.Now()
	err := r.sock.PublishExchangeDelayedRequest(r.exchangeDelay, queue, req, delay)
	elapsed := time.Since(start)
	promRequestProcessTime.WithLabelValues(target, string(resource), string(req.Method)).Observe(float64(elapsed.Milliseconds()))

	return err
}
