package listenhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package listenhandler -destination ./mock_listenhandler_listenhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuecallhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuehandler"
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
	rabbitSock rabbitmqhandler.Rabbit
	db         dbhandler.DBHandler
	reqHandler requesthandler.RequestHandler

	queueHandler     queuehandler.QueueHandler
	queuecallHandler queuecallhandler.QueuecallHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}" //nolint:deadcode,unused,varcheck // this is ok
	regAny  = "(.*)"                                                         //nolint:deadcode,unused,varcheck // this is ok

	// v1
	// queues
	regV1Queues                = regexp.MustCompile("/v1/queues$")
	regV1QueuesGet             = regexp.MustCompile(`/v1/queues\?(.*)$`)
	reqV1QueuesID              = regexp.MustCompile("/v1/queues/" + regUUID + "$")
	reqV1QueuesIDTagIDs        = regexp.MustCompile("/v1/queues/" + regUUID + "/tag_ids$")
	reqV1QueuesIDRoutingMethod = regexp.MustCompile("/v1/queues/" + regUUID + "/routing_method$")
	reqV1QueuesIDQueuecalls    = regexp.MustCompile("/v1/queues/" + regUUID + "/queuecalls$")
	reqV1QueuesIDWaitActions   = regexp.MustCompile("/v1/queues/" + regUUID + "/wait_actions$")

	// queuecalls
	regV1QueuecallsID               = regexp.MustCompile("/v1/queuecalls/" + regUUID + "$")
	regV1QueuecallsIDExecute        = regexp.MustCompile("/v1/queuecalls/" + regUUID + "/execute$")
	regV1QueuecallsIDTimeoutWait    = regexp.MustCompile("/v1/queuecalls/" + regUUID + "/timeout_wait$")
	regV1QueuecallsIDTimeoutService = regexp.MustCompile("/v1/queuecalls/" + regUUID + "/timeout_service$")

	// queuecallreferences
	regV1QueuecallreferencesID = regexp.MustCompile("/v1/queuecallreferences/" + regUUID + "$")
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
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	queueHandler queuehandler.QueueHandler,
	queuecallHandler queuecallhandler.QueuecallHandler,
) ListenHandler {
	h := &listenHandler{
		rabbitSock: rabbitSock,
		db:         db,
		reqHandler: reqHandler,

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
			err := h.rabbitSock.ConsumeRPCOpt(queue, "call-manager", false, false, false, h.processRequest)
			if err != nil {
				logrus.Errorf("Could not consume the request message correctly. err: %v", err)
			}
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	var requestType string
	var err error
	var response *rabbitmqhandler.Response

	ctx := context.Background()

	uri, err := url.QueryUnescape(m.URI)
	if err != nil {
		uri = "could not unescape uri"
	}

	log := logrus.WithFields(
		logrus.Fields{
			"request": m,
		})
	log.Debugf("Received request. method: %s, uri: %s", m.Method, uri)

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	// ////////////
	// // queues
	// ////////////
	// GET /queues
	case regV1QueuesGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1QueuesGet(ctx, m)
		requestType = "/v1/queues"

	// POST /queues
	case regV1Queues.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1QueuesPost(ctx, m)
		requestType = "/v1/queues"

	// PUT /queues/<queue-id>
	case reqV1QueuesID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1QueuesIDPut(ctx, m)
		requestType = "/v1/queues"

	// PUT /queues/<queue-id>/tag_ids
	case reqV1QueuesIDTagIDs.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1QueuesIDTagIDsPut(ctx, m)
		requestType = "/v1/queues"

	// PUT /queues/<queue-id>/routing_method
	case reqV1QueuesIDRoutingMethod.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1QueuesIDRoutingMethodPut(ctx, m)
		requestType = "/v1/queues"

	// POST /queues/<queue-id>/queuecalls
	case reqV1QueuesIDQueuecalls.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1QueuesIDQueuecallsPost(ctx, m)
		requestType = "/v1/queues"

	// PUT /queues/<queue-id>/wait_actions
	case reqV1QueuesIDWaitActions.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1QueuesIDWaitActionsPut(ctx, m)
		requestType = "/v1/queues"

	/////////////
	// queuecalls
	/////////////

	// DELETE /queuecalls/<queuecall-id>
	case regV1QueuecallsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1QueuecallsIDDelete(ctx, m)
		requestType = "/v1/queuecalls"

	// PUT /queuecalls/<queuecall-id>/execute
	case regV1QueuecallsIDExecute.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1QueuecallsIDExecutePost(ctx, m)
		requestType = "/v1/queuecalls"

	// POST /queuecalls/<queuecall-id>/timeout_wait
	case regV1QueuecallsIDTimeoutWait.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1QueuecallsIDTimeoutWaitPost(ctx, m)
		requestType = "/v1/queuecalls"

	// POST /queuecalls/<queuecall-id>/timeout_service
	case regV1QueuecallsIDTimeoutService.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1QueuecallsIDTimeoutServicePost(ctx, m)
		requestType = "/v1/queuecalls"

	//////////////////////
	// queuecallreferences
	//////////////////////
	// DELETE /queuecallreferences/<queuecallreference-id>
	case regV1QueuecallreferencesID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1QueuecallreferencesIDDelete(ctx, m)
		requestType = "/v1/queuecalls"

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		log.Errorf("Could not find corresponded message handler. method: %s, uri: %s", m.Method, uri)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}
	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not find corresponded message handler. method: %s, uri: %s", m.Method, uri)
		response = simpleResponse(400)
		err = nil
	} else {
		log.WithFields(
			logrus.Fields{
				"response": response,
			},
		).Debugf("Sending response. method: %s, uri: %s", m.Method, uri)
	}

	return response, err
}
