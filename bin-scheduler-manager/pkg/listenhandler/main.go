package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-scheduler-manager/pkg/backuphandler"
	"monorepo/bin-scheduler-manager/pkg/dbhandler"
	"monorepo/bin-scheduler-manager/pkg/dispatchhandler"
	"monorepo/bin-scheduler-manager/pkg/schedulehandler"
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
	sockHandler sockhandler.SockHandler

	scheduleHandler schedulehandler.ScheduleHandler
	dispatchHandler dispatchhandler.DispatchHandler
	backupHandler   backuphandler.BackupHandler

	// executionRetentionDays is the audit-row retention applied by the
	// /v1/executions/prune endpoint (design §6, config §11).
	executionRetentionDays int
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// v1
	// schedules
	regV1Schedules          = regexp.MustCompile("/v1/schedules$")
	regV1SchedulesGet       = regexp.MustCompile(`/v1/schedules\?(.*)$`)
	regV1SchedulesID        = regexp.MustCompile("/v1/schedules/" + regUUID + "$")
	regV1SchedulesIDExecute = regexp.MustCompile("/v1/schedules/" + regUUID + "/execute$")

	// executions
	regV1ExecutionsGet   = regexp.MustCompile(`/v1/executions\?(.*)$`)
	regV1ExecutionsPrune = regexp.MustCompile("/v1/executions/prune$")

	// backups
	regV1Backups = regexp.MustCompile("/v1/backups$")
)

var (
	metricsNamespace = "scheduler_manager"

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
// Resolution order: typed *cerrors.VoipbinError → ToResponse (409 for
// FailedPrecondition/AlreadyExists, 404 for NotFound, ...); legacy
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
	scheduleHandler schedulehandler.ScheduleHandler,
	dispatchHandler dispatchhandler.DispatchHandler,
	backupHandler backuphandler.BackupHandler,
	executionRetentionDays int,
) ListenHandler {
	h := &listenHandler{
		sockHandler: sockHandler,

		scheduleHandler: scheduleHandler,
		dispatchHandler: dispatchHandler,
		backupHandler:   backupHandler,

		executionRetentionDays: executionRetentionDays,
	}

	return h
}

func (h *listenHandler) Run(queue, exchangeDelay string) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})
	log.WithFields(logrus.Fields{
		"queue":          queue,
		"exchange_delay": exchangeDelay,
	}).Info("Creating rabbitmq queue for listen.")

	if err := h.sockHandler.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// receive requests
	go func() {
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, string(commonoutline.ServiceNameSchedulerManager), false, false, false, 10, h.processRequest); errConsume != nil {
			log.Errorf("Could not consume the request message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	var requestType string
	var err error
	var response *sock.Response

	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"request": m,
		})
	log.Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	////////////
	// schedules
	////////////
	// GET /schedules
	case regV1SchedulesGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1SchedulesGet(ctx, m)
		requestType = "/v1/schedules"

	// POST /schedules
	case regV1Schedules.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1SchedulesPost(ctx, m)
		requestType = "/v1/schedules"

	// POST /schedules/<schedule-id>/execute
	case regV1SchedulesIDExecute.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1SchedulesIDExecutePost(ctx, m)
		requestType = "/v1/schedules/<schedule-id>/execute"

	// GET /schedules/<schedule-id>
	case regV1SchedulesID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1SchedulesIDGet(ctx, m)
		requestType = "/v1/schedules/<schedule-id>"

	// PUT /schedules/<schedule-id>
	case regV1SchedulesID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1SchedulesIDPut(ctx, m)
		requestType = "/v1/schedules/<schedule-id>"

	// DELETE /schedules/<schedule-id>
	case regV1SchedulesID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1SchedulesIDDelete(ctx, m)
		requestType = "/v1/schedules/<schedule-id>"

	////////////
	// executions
	////////////
	// POST /executions/prune
	case regV1ExecutionsPrune.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ExecutionsPrunePost(ctx, m)
		requestType = "/v1/executions/prune"

	// GET /executions
	case regV1ExecutionsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ExecutionsGet(ctx, m)
		requestType = "/v1/executions"

	////////////
	// backups
	////////////
	// POST /backups
	case regV1Backups.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1BackupsPost(ctx, m)
		requestType = "/v1/backups"

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

	// default error handler — typed errors and ErrNotFound flow through
	// errorResponse; other errors keep legacy 400.
	if err != nil {
		log.Errorf("Could not handle the message correctly. method: %s, uri: %s, err: %v", m.Method, m.URI, err)
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
	} else {
		log.WithFields(
			logrus.Fields{
				"response": response,
			},
		).Debugf("Sending response. method: %s, uri: %s", m.Method, m.URI)
	}

	return response, err
}
