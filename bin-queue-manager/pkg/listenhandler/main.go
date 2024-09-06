package listenhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-queue-manager/pkg/queuecallhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
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
	utilHanlder utilhandler.UtilHandler
	rabbitSock  rabbitmqhandler.Rabbit

	queueHandler     queuehandler.QueueHandler
	queuecallHandler queuecallhandler.QueuecallHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}" //nolint:deadcode,unused,varcheck // this is ok
	regAny  = "(.*)"                                                         //nolint:deadcode,unused,varcheck // this is ok

	// v1
	// queues
	regV1Queues                = regexp.MustCompile("/v1/queues$")
	regV1QueuesGet             = regexp.MustCompile(`/v1/queues\?` + regAny + "$")
	reqV1QueuesID              = regexp.MustCompile("/v1/queues/" + regUUID + "$")
	reqV1QueuesIDTagIDs        = regexp.MustCompile("/v1/queues/" + regUUID + "/tag_ids$")
	reqV1QueuesIDRoutingMethod = regexp.MustCompile("/v1/queues/" + regUUID + "/routing_method$")
	reqV1QueuesIDWaitActions   = regexp.MustCompile("/v1/queues/" + regUUID + "/wait_actions$")
	reqV1QueuesIDAgentsGet     = regexp.MustCompile("/v1/queues/" + regUUID + `/agents\?`)
	reqV1QueuesIDExecute       = regexp.MustCompile("/v1/queues/" + regUUID + "/execute$")
	reqV1QueuesIDExecuteRun    = regexp.MustCompile("/v1/queues/" + regUUID + "/execute_run$")

	// queuecalls
	regV1QueuecallsGet               = regexp.MustCompile(`/v1/queuecalls\?` + regAny + "$")
	regV1QueuecallsID                = regexp.MustCompile("/v1/queuecalls/" + regUUID + "$")
	regV1QueuecallsIDTimeoutWait     = regexp.MustCompile("/v1/queuecalls/" + regUUID + "/timeout_wait$")
	regV1QueuecallsIDTimeoutService  = regexp.MustCompile("/v1/queuecalls/" + regUUID + "/timeout_service$")
	regV1QueuecallsIDExecute         = regexp.MustCompile("/v1/queuecalls/" + regUUID + "/execute$")
	regV1QueuecallsIDHealthCheck     = regexp.MustCompile("/v1/queuecalls/" + regUUID + "/health-check$")
	regV1QueuecallsIDStatusWaiting   = regexp.MustCompile("/v1/queuecalls/" + regUUID + "/status_waiting$")
	regV1QueuecallsIDKick            = regexp.MustCompile("/v1/queuecalls/" + regUUID + "/kick$")
	regV1QueuecallsReferenceIDID     = regexp.MustCompile("/v1/queuecalls/reference_id/" + regUUID + "$")
	regV1QueuecallsReferenceIDIDKick = regexp.MustCompile("/v1/queuecalls/reference_id/" + regUUID + "/kick$")

	// services
	regV1ServicesTypeQueuecall = regexp.MustCompile("/v1/services/type/queuecall$")
)

var (
	metricsNamespace = "queue_manager"

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
	queueHandler queuehandler.QueueHandler,
	queuecallHandler queuecallhandler.QueuecallHandler,
) ListenHandler {
	h := &listenHandler{
		utilHanlder: utilhandler.NewUtilHandler(),
		rabbitSock:  rabbitSock,

		queueHandler:     queueHandler,
		queuecallHandler: queuecallHandler,
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
			err := h.rabbitSock.ConsumeRPCOpt(queue, "queue-manager", false, false, false, 10, h.processRequest)
			if err != nil {
				logrus.Errorf("Could not consume the request message correctly. err: %v", err)
			}
		}
	}()

	return nil
}

// processRequest handles received request
func (h *listenHandler) processRequest(m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "processRequest",
			"request": m,
		})

	var requestType string
	var err error
	var response *rabbitmqhandler.Response

	ctx := context.Background()

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	// ////////////
	// // queues
	// ////////////
	// /queues
	// GET /queues
	case regV1QueuesGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1QueuesGet(ctx, m)
		requestType = "/v1/queues"

	// POST /queues
	case regV1Queues.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1QueuesPost(ctx, m)
		requestType = "/v1/queues"

	// /queues/<queue-id>/
	// GET /queues/<queue-id>
	case reqV1QueuesID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1QueuesIDGet(ctx, m)
		requestType = "/v1/queues/<queue-id>/"

	// DELETE /queues/<queue-id>
	case reqV1QueuesID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1QueuesIDDelete(ctx, m)
		requestType = "/v1/queues/<queue-id>/"

	// PUT /queues/<queue-id>
	case reqV1QueuesID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1QueuesIDPut(ctx, m)
		requestType = "/v1/queues/<queue-id>/"

	// PUT /queues/<queue-id>/tag_ids
	case reqV1QueuesIDTagIDs.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1QueuesIDTagIDsPut(ctx, m)
		requestType = "/v1/queues/<queue-id>/tag_ids"

	// PUT /queues/<queue-id>/routing_method
	case reqV1QueuesIDRoutingMethod.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1QueuesIDRoutingMethodPut(ctx, m)
		requestType = "/v1/queues/<queue-id>/routing_method"

	// PUT /queues/<queue-id>/wait_actions
	case reqV1QueuesIDWaitActions.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1QueuesIDWaitActionsPut(ctx, m)
		requestType = "/v1/queues/<queue-id>/wait_actions"

	// GET /queues/<queue-id>/agents
	case reqV1QueuesIDAgentsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1QueuesIDAgentsGet(ctx, m)
		requestType = "/v1/queues/<queue-id>/agents"

	// PUT /queues/<queue-id>/execute
	case reqV1QueuesIDExecute.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1QueuesIDExecutePut(ctx, m)
		requestType = "/v1/queues/<queue-id>/execute"

	// POST /queues/<queue-id>/execute_run
	case reqV1QueuesIDExecuteRun.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1QueuesIDExecuteRunPost(ctx, m)
		requestType = "/v1/queues/<queue-id>/execute_run"

	/////////////
	// queuecalls
	/////////////

	// GET /queuecalls
	case regV1QueuecallsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1QueuecallsGet(ctx, m)
		requestType = "/v1/queuecalls"

	// GET /queuecalls/<queuecall-id>
	case regV1QueuecallsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1QueuecallsIDGet(ctx, m)
		requestType = "/v1/queuecalls"

	// DELETE /queuecalls/<queuecall-id>
	case regV1QueuecallsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1QueuecallsIDDelete(ctx, m)
		requestType = "/v1/queuecalls"

	// POST /queuecalls/<queuecall-id>/timeout_wait
	case regV1QueuecallsIDTimeoutWait.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1QueuecallsIDTimeoutWaitPost(ctx, m)
		requestType = "/v1/queuecalls/<queuecall-id>/timeout_wait"

	// POST /queuecalls/<queuecall-id>/timeout_service
	case regV1QueuecallsIDTimeoutService.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1QueuecallsIDTimeoutServicePost(ctx, m)
		requestType = "/v1/queuecalls/<queuecall-id>/timeout_service"

	// POST /queuecalls/<queuecall-id>/execute
	case regV1QueuecallsIDExecute.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1QueuecallsIDExecutePost(ctx, m)
		requestType = "/v1/queuecalls/<queuecall-id>/execute"

	// POST /queuecalls/<queuecall-id>/health-check
	case regV1QueuecallsIDHealthCheck.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1QueuecallsIDHealthCheckPost(ctx, m)
		requestType = "/v1/queuecalls/<queuecall-id>/health-check"

	// POST /queuecalls/<queuecall-id>/status_waiting
	case regV1QueuecallsIDStatusWaiting.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1QueuecallsIDStatusWaitingPost(ctx, m)
		requestType = "/v1/queuecalls/<queuecall-id>/status_waiting"

	// POST /queuecalls/<queuecall-id>/kick
	case regV1QueuecallsIDKick.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1QueuecallsIDKickPost(ctx, m)
		requestType = "/v1/queuecalls/<queuecall-id>/kick"

	// GET /queuecalls/reference_id/<reference-id>
	case regV1QueuecallsReferenceIDID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1QueuecallsReferenceIDIDGet(ctx, m)
		requestType = "/v1/queuecalls/reference_id/<reference-id>"

	// POST /queuecalls/reference_id/<reference-id>/kick
	case regV1QueuecallsReferenceIDIDKick.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1QueuecallsReferenceIDIDKickPost(ctx, m)
		requestType = "/v1/queuecalls/reference_id/<reference-id>/kick"

	/////////////////
	// services
	////////////////
	// POST /services/type/queuecall
	case regV1ServicesTypeQueuecall.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ServicesTypeQueuecallPost(ctx, m)
		requestType = "/v1/services/type/queuecall"

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		log.Errorf("Could not find corresponded message handler. method: %s, uri: %s", m.Method, m.URI)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}
	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not handle the message correctly. method: %s, uri: %s", m.Method, m.URI)
		response = simpleResponse(400)
		err = nil
	}

	log.WithField("response", response).Debugf("Sending response. method: %s, uri: %s", m.Method, m.URI)

	return response, err
}
