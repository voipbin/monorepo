package listenhandler

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-outdial-manager/pkg/outdialhandler"
	"monorepo/bin-outdial-manager/pkg/outdialtargethandler"
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

	outdialHandler       outdialhandler.OutdialHandler
	outdialTargetHandler outdialtargethandler.OutdialTargetHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// outdials
	regV1Outdials             = regexp.MustCompile("/v1/outdials$")
	regV1OutdialsGet          = regexp.MustCompile(`/v1/outdials\?`)
	regV1OutdialsID           = regexp.MustCompile("/v1/outdials/" + regUUID + "$")
	regV1OutdialsIDAvailable  = regexp.MustCompile("/v1/outdials/" + regUUID + `/available\?`)
	regV1OutdialsIDTargets    = regexp.MustCompile("/v1/outdials/" + regUUID + "/targets$")
	regV1OutdialsIDTargetsGet = regexp.MustCompile("/v1/outdials/" + regUUID + `/targets\?`)
	regV1OutdialsIDCampaignID = regexp.MustCompile("/v1/outdials/" + regUUID + "/campaign_id$")
	regV1OutdialsIDData       = regexp.MustCompile("/v1/outdials/" + regUUID + "/data$")

	// outdialtargets
	regV1OutdialtargetsID            = regexp.MustCompile("/v1/outdialtargets/" + regUUID + "$")
	regV1OutdialtargetsIDProgressing = regexp.MustCompile("/v1/outdialtargets/" + regUUID + "/progressing$")
	regV1OutdialtargetsIDStatus      = regexp.MustCompile("/v1/outdialtargets/" + regUUID + "/status$")
)

var (
	metricsNamespace = "outdial_manager"

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

// NewListenHandler return ListenHandler interface
func NewListenHandler(
	sockHandler sockhandler.SockHandler,
	outdialHandler outdialhandler.OutdialHandler,
	outdialTargetHandler outdialtargethandler.OutdialTargetHandler,
) ListenHandler {
	h := &listenHandler{
		sockHandler: sockHandler,

		outdialHandler:       outdialHandler,
		outdialTargetHandler: outdialTargetHandler,
	}

	return h
}

func (h *listenHandler) Run(queue, exchangeDelay string) error {
	logrus.WithFields(logrus.Fields{
		"queue": queue,
	}).Info("Creating rabbitmq queue for listen.")

	if err := h.sockHandler.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// process the received request
	go func() {
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, "outdial-manager", false, false, false, 10, h.processRequest); errConsume != nil {
			logrus.Errorf("Could not consume the request message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	var requestType string
	var err error
	var response *sock.Response

	ctx := context.Background()

	logrus.WithFields(
		logrus.Fields{
			"uri":       m.URI,
			"method":    m.Method,
			"data_type": m.DataType,
			"data":      m.Data,
		}).Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)

	start := time.Now()
	switch {

	// v1
	// outdials
	case regV1Outdials.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/outdials"
		response, err = h.v1OutdialsPost(ctx, m)

	case regV1OutdialsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/outdials"
		response, err = h.v1OutdialsGet(ctx, m)

	// outdials/<outdial-id>
	case regV1OutdialsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/outdials/<outdial-id>"
		response, err = h.v1OutdialsIDGet(ctx, m)

	case regV1OutdialsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/outdials/<outdial-id>"
		response, err = h.v1OutdialsIDPut(ctx, m)

	case regV1OutdialsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/outdials/<outdial-id>"
		response, err = h.v1OutdialsIDDelete(ctx, m)

	// outdials/<outdial-id>/available
	case regV1OutdialsIDAvailable.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/outdials/<outdial-id>/available"
		response, err = h.v1OutdialsIDAvailableGet(ctx, m)

	// outdials/<outdial-id>/targets
	case regV1OutdialsIDTargets.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/outdials/<outdial-id>/targets"
		response, err = h.v1OutdialsIDTargetsPost(ctx, m)

	case regV1OutdialsIDTargetsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/outdials/<outdial-id>/targets"
		response, err = h.v1OutdialsIDTargetsGet(ctx, m)

	case regV1OutdialsIDCampaignID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/outdials/<outdial-id>/campaign_id"
		response, err = h.v1OutdialsIDCampaignIDPut(ctx, m)

	case regV1OutdialsIDData.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/outdials/<outdial-id>/data"
		response, err = h.v1OutdialsIDDataPut(ctx, m)

	// outdialtargets
	// /v1/outdialtargets/<outdialtarget-id>
	case regV1OutdialtargetsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/outdialtargets/<outdialtarget-id>"
		response, err = h.v1OutdialtargetsIDGet(ctx, m)

	case regV1OutdialtargetsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/outdialtargets/<outdialtarget-id>"
		response, err = h.v1OutdialtargetsIDDelete(ctx, m)

	// /v1/outdialtargets/<outdialtarget-id>/progressing
	case regV1OutdialtargetsIDProgressing.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/outdialtargets/<outdialtarget-id>/progressing"
		response, err = h.v1OutdialtargetsIDProgressingPost(ctx, m)

	// /v1/outdialtargets/<outdialtarget-id>/status
	case regV1OutdialtargetsIDStatus.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/outdialtargets/<outdialtarget-id>/status"
		response, err = h.v1OutdialtargetsIDStatusPut(ctx, m)

	default:
		logrus.WithFields(
			logrus.Fields{
				"uri":    m.URI,
				"method": m.Method,
			}).Errorf("Could not find corresponded message handler. data: %s", m.Data)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}
	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	// default error handler
	if err != nil {
		logrus.WithFields(
			logrus.Fields{
				"uri":    m.URI,
				"method": m.Method,
				"error":  err,
			}).Errorf("Could not process the request correctly. data: %s", m.Data)
		response = simpleResponse(400)
		err = nil
	}

	logrus.WithFields(
		logrus.Fields{
			"response": response,
			"err":      err,
		}).Debugf("Sending response. method: %s, uri: %s", m.Method, m.URI)

	return response, err
}
