package listenhandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-pipecat-manager/pkg/dbhandler"
	"monorepo/bin-pipecat-manager/pkg/pipecatcallhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// pagination parameters
const (
	PageSize  = "page_size"
	PageToken = "page_token"
)

const (
	constConsumerName = "pipecat-manager"
)

// ListenHandler interface
type ListenHandler interface {
	Run(queue, queueVolatile, exchangeDelay string) error
}

type listenHandler struct {
	sockHandler sockhandler.SockHandler

	utilHandler        utilhandler.UtilHandler
	pipecatcallHandler pipecatcallhandler.PipecatcallHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// v1
	// pipecatcalls
	regV1Pipecatcalls       = regexp.MustCompile(`/v1/pipecatcalls$`)
	regV1PipecatcallsID     = regexp.MustCompile("/v1/pipecatcalls/" + regUUID + "$")
	regV1PipecatcallsIDStop = regexp.MustCompile("/v1/pipecatcalls/" + regUUID + "/stop$")

	// messages
	regV1Messages = regexp.MustCompile(`/v1/messages$`)

	// ping
	regV1Ping = regexp.MustCompile(`/v1/ping$`)
)

var (
	metricsNamespace = "pipecat_manager"

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
func simpleResponse(code int) *sock.Response {
	return &sock.Response{
		StatusCode: code,
	}
}

// errorResponse maps a business-handler error to the appropriate sock.Response.
// Resolution order: typed *cerrors.VoipbinError → ToResponse; legacy
// dbhandler.ErrNotFound → 404; else → 500.
func errorResponse(err error) *sock.Response {
	if err == nil {
		logrus.WithField("func", "errorResponse").Warn("errorResponse called with nil error — likely a caller bug; returning 500")
		return simpleResponse(http.StatusInternalServerError)
	}

	var ve *cerrors.VoipbinError
	if stderrors.As(err, &ve) {
		resp, e := cerrors.ToResponse(ve)
		if e == nil {
			return resp
		}
		logrus.WithField("func", "errorResponse").Errorf("cerrors.ToResponse failed for typed VoipbinError: %v", e)
		return simpleResponse(http.StatusInternalServerError)
	}

	if stderrors.Is(err, dbhandler.ErrNotFound) {
		return simpleResponse(http.StatusNotFound)
	}

	return simpleResponse(http.StatusInternalServerError)
}

// NewListenHandler return ListenHandler interface
func NewListenHandler(
	sockHandler sockhandler.SockHandler,
	pipecatcallHandler pipecatcallhandler.PipecatcallHandler,
) ListenHandler {
	h := &listenHandler{
		sockHandler: sockHandler,

		utilHandler:        utilhandler.NewUtilHandler(),
		pipecatcallHandler: pipecatcallHandler,
	}

	return h
}

// runListenQueue listens the queue
func (h *listenHandler) runListenQueue(queue string) error {
	logrus.WithFields(logrus.Fields{
		"queue": queue,
	}).Info("Creating rabbitmq queue for listen.")

	// declare the queue
	if err := h.sockHandler.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// receive requests
	go func() {
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, constConsumerName, false, false, false, 10, h.processRequest); errConsume != nil {
			logrus.Errorf("Could not consume the request message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

// runListenQueueVolatile listens volatile queue
func (h *listenHandler) runListenQueueVolatile(queue string) error {
	logrus.WithFields(logrus.Fields{
		"queue": queue,
	}).Info("Creating rabbitmq queue for listen.")

	// declare the queue
	if err := h.sockHandler.QueueCreate(queue, "volatile"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// receive requests
	go func() {
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, constConsumerName, false, false, false, 10, h.processRequest); errConsume != nil {
			logrus.Errorf("Could not consume the request message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

// Run
func (h *listenHandler) Run(queue, queueVolatile, exchangeDelay string) error {
	log := logrus.WithFields(logrus.Fields{
		"queue":          queue,
		"queue volatile": queueVolatile,
	})
	log.Info("Creating rabbitmq queue for listen.")

	// start queue listen
	if err := h.runListenQueue(queue); err != nil {
		log.Errorf("Could not listen the queue. err: %v", err)
		return err
	}

	// start volatile queue listen
	if err := h.runListenQueueVolatile(queueVolatile); err != nil {
		log.Errorf("Could not listen the volatile queue. err: %v", err)
		return err
	}

	return nil
}

// processRequest handles all of requests of the listen queue.
func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processRequest",
		"request": m,
	})
	ctx := context.Background()

	var requestType string
	var err error
	var response *sock.Response

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	////////////////////
	// pipecatcalls
	////////////////////
	// POST /pipecatcalls
	case regV1Pipecatcalls.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1PipecatcallsPost(ctx, m)
		requestType = "/v1/pipecatcalls"

	// GET /pipecatcalls/<pipecatcall-id>
	case regV1PipecatcallsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1PipecatcallsIDGet(ctx, m)
		requestType = "/v1/pipecatcalls/<pipecatcall-id>"

	// POST /pipecatcalls/<pipecatcall-id>/stop
	case regV1PipecatcallsIDStop.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1PipecatcallsIDStopPost(ctx, m)
		requestType = "/v1/pipecatcalls/<pipecatcall-id>/stop"

	////////////////////
	// messages
	////////////////////
	// POST /messages
	case regV1Messages.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1MessagesPost(ctx, m)
		requestType = "/v1/messages"

	////////////////////
	// ping
	////////////////////
	// GET /v1/ping
	case regV1Ping.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1PingGet(ctx, m)
		requestType = "/v1/ping"

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		log.WithFields(logrus.Fields{
			"uri":    m.URI,
			"method": m.Method,
		}).Errorf("Could not find corresponded message handler. data: %s", m.Data)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}
	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	// default error handler — typed errors and ErrNotFound flow through
	// errorResponse; other errors keep legacy 400.
	if err != nil {
		log.WithFields(logrus.Fields{
			"uri":    m.URI,
			"method": m.Method,
			"error":  err,
		}).Errorf("Could not process the request correctly. data: %s", m.Data)
		var ve *cerrors.VoipbinError
		switch {
		case stderrors.As(err, &ve):
			response = errorResponse(err)
		case stderrors.Is(err, dbhandler.ErrNotFound):
			response = errorResponse(err)
		default:
			response = simpleResponse(400)
		}
		err = nil
	}

	return response, err
}
