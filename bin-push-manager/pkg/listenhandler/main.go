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
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-push-manager/pkg/dbhandler"
	"monorepo/bin-push-manager/pkg/devicehandler"
	"monorepo/bin-push-manager/pkg/notificationhandler"
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

	deviceHandler       devicehandler.DeviceHandler
	notificationHandler notificationhandler.NotificationHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// v1
	// devices
	regV1Devices    = regexp.MustCompile("/v1/devices$")
	regV1DevicesGet = regexp.MustCompile(`/v1/devices\?(.*)$`)
	regV1DevicesID  = regexp.MustCompile("/v1/devices/" + regUUID + "$")

	// notifications
	regV1Notifications = regexp.MustCompile("/v1/notifications$")
)

var (
	metricsNamespace = "push_manager"

	promReceivedRequestProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "receive_request_process_time",
			Help:      "Process time of received request",
			Buckets: []float64{
				50, 100, 500, 1000, 3000, 8000,
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
// Resolution order: typed *cerrors.VoipbinError, then dbhandler.ErrNotFound as 404, else 500.
func errorResponse(err error) *sock.Response {
	if err == nil {
		logrus.WithField("func", "errorResponse").Warn("errorResponse called with nil error. returning 500")
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
func NewListenHandler(sockHandler sockhandler.SockHandler, deviceHandler devicehandler.DeviceHandler, notificationHandler notificationhandler.NotificationHandler) ListenHandler {
	return &listenHandler{
		sockHandler:         sockHandler,
		deviceHandler:       deviceHandler,
		notificationHandler: notificationHandler,
	}
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
		return fmt.Errorf("could not declare the queue for listenHandler. err: %w", err)
	}

	// receive requests
	go func() {
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, "push-manager", false, false, false, 10, h.processRequest); errConsume != nil {
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
			"method": m.Method,
			"uri":    m.URI,
		})
	// the request body can hold a push token, so it is never logged
	log.Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	////////////
	// devices
	////////////
	// GET /devices
	case regV1DevicesGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1DevicesGet(ctx, m)
		requestType = "/v1/devices"

	// POST /devices
	case regV1Devices.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1DevicesPost(ctx, m)
		requestType = "/v1/devices"

	// GET /devices/<device-id>
	case regV1DevicesID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1DevicesIDGet(ctx, m)
		requestType = "/v1/devices/<device-id>"

	// DELETE /devices/<device-id>
	case regV1DevicesID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1DevicesIDDelete(ctx, m)
		requestType = "/v1/devices/<device-id>"

	////////////
	// notifications
	////////////
	// POST /notifications
	case regV1Notifications.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1NotificationsPost(ctx, m)
		requestType = "/v1/notifications"

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

	// default error handler: typed errors and ErrNotFound map to their status, any other error is a 500
	if err != nil {
		log.Errorf("Could not handle the message correctly. method: %s, uri: %s, err: %v", m.Method, m.URI, err)
		response = errorResponse(err)
		err = nil
	} else {
		log.Debugf("Sending response. method: %s, uri: %s, status: %d", m.Method, m.URI, response.StatusCode)
	}

	return response, err
}
