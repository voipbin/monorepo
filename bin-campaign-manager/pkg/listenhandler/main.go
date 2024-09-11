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

	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	"monorepo/bin-campaign-manager/pkg/campaignhandler"
	"monorepo/bin-campaign-manager/pkg/outplanhandler"
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

	campaignHandler     campaignhandler.CampaignHandler
	campaigncallHandler campaigncallhandler.CampaigncallHandler
	outplanHandler      outplanhandler.OutplanHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// campaigns
	regV1Campaigns                 = regexp.MustCompile("/v1/campaigns$")
	regV1CampaignsGet              = regexp.MustCompile(`/v1/campaigns\?`)
	regV1CampaignsID               = regexp.MustCompile("/v1/campaigns/" + regUUID + "$")
	regV1CampaignsIDExecute        = regexp.MustCompile("/v1/campaigns/" + regUUID + "/execute$")
	regV1CampaignsIDStatus         = regexp.MustCompile("/v1/campaigns/" + regUUID + "/status$")
	regV1CampaignsIDServiceLevel   = regexp.MustCompile("/v1/campaigns/" + regUUID + "/service_level$")
	regV1CampaignsIDActions        = regexp.MustCompile("/v1/campaigns/" + regUUID + "/actions$")
	regV1CampaignsIDResourceInfo   = regexp.MustCompile("/v1/campaigns/" + regUUID + "/resource_info$")
	regV1CampaignsIDNextCampaignID = regexp.MustCompile("/v1/campaigns/" + regUUID + "/next_campaign_id$")

	// campaigncalls
	regV1CampaigncallsGet = regexp.MustCompile(`/v1/campaigncalls\?`)
	regV1CampaigncallsID  = regexp.MustCompile("/v1/campaigncalls/" + regUUID + "$")

	// outplans
	regV1Outplans        = regexp.MustCompile("/v1/outplans$")
	regV1OutplansGet     = regexp.MustCompile(`/v1/outplans\?`)
	regV1OutplansID      = regexp.MustCompile("/v1/outplans/" + regUUID + "$")
	regV1OutplansIDDials = regexp.MustCompile("/v1/outplans/" + regUUID + "/dials$")
)

var (
	metricsNamespace = "campaign_manager"

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

	outplanHandler outplanhandler.OutplanHandler,
	campaignHandler campaignhandler.CampaignHandler,
	campaigncallHandler campaigncallhandler.CampaigncallHandler,
) ListenHandler {
	h := &listenHandler{
		sockHandler: sockHandler,

		campaignHandler:     campaignHandler,
		campaigncallHandler: campaigncallHandler,
		outplanHandler:      outplanHandler,
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
		for {
			err := h.sockHandler.ConsumeRPC(queue, "campaign-manager", false, false, false, 10, h.processRequest)
			if err != nil {
				logrus.Errorf("Could not consume the request message correctly. err: %v", err)
			}
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
	// campaigns
	// /v1/campaigns
	case regV1Campaigns.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/v1/campaigns"
		response, err = h.v1CampaignsPost(ctx, m)

	case regV1CampaignsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/v1/campaigns"
		response, err = h.v1CampaignsGet(ctx, m)

	// /v1/campaigns/<campaign-id>
	case regV1CampaignsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/v1/campaigns/<campaign-id>"
		response, err = h.v1CampaignsIDGet(ctx, m)

	case regV1CampaignsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/v1/campaigns/<campaign-id>"
		response, err = h.v1CampaignsIDDelete(ctx, m)

	case regV1CampaignsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/v1/campaigns/<campaign-id>"
		response, err = h.v1CampaignsIDPut(ctx, m)

	// /v1/campaigns/<campaign-id>/execute
	case regV1CampaignsIDExecute.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/v1/campaigns/<campaign-id>/execute"
		response, err = h.v1CampaignsIDExecutePost(ctx, m)

	// /v1/campaigns/<campaign-id>/status
	case regV1CampaignsIDStatus.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/v1/campaigns/<campaign-id>/status"
		response, err = h.v1CampaignsIDStatusPut(ctx, m)

	// /v1/campaigns/<campaign-id>/service_level
	case regV1CampaignsIDServiceLevel.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/v1/campaigns/<campaign-id>/service_level"
		response, err = h.v1CampaignsIDServiceLevelPut(ctx, m)

	// /v1/campaigns/<campaign-id>/actions
	case regV1CampaignsIDActions.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/v1/campaigns/<campaign-id>/actions"
		response, err = h.v1CampaignsIDActionsPut(ctx, m)

	// /v1/campaigns/<campaign-id>/resource_info
	case regV1CampaignsIDResourceInfo.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/v1/campaigns/<campaign-id>/resource_info"
		response, err = h.v1CampaignsIDResourceInfoPut(ctx, m)

	// /v1/campaigns/<campaign-id>/next_campaign_id
	case regV1CampaignsIDNextCampaignID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/v1/campaigns/<campaign-id>/next_campaign_id"
		response, err = h.v1CampaignsIDNextCampaignIDPut(ctx, m)

	// campaigncalls
	// /v1/campaigncalls
	case regV1CampaigncallsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/v1/campaigncalls"
		response, err = h.v1CampaigncallsGet(ctx, m)

	// /v1/campaigncalls/<campaigncall_id>
	case regV1CampaigncallsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/v1/campaigncalls/<campaigncall-id>"
		response, err = h.v1CampaigncallsIDGet(ctx, m)

	// /v1/campaigncalls/<campaigncall_id>
	case regV1CampaigncallsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/v1/campaigncalls/<campaigncall-id>"
		response, err = h.v1CampaigncallsIDDelete(ctx, m)

	// outplans
	// /v1/outplans
	case regV1Outplans.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/v1/outplans"
		response, err = h.v1OutplansPost(ctx, m)

	case regV1OutplansGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/v1/outplans"
		response, err = h.v1OutplansGet(ctx, m)

	// /v1/outplans/<outplan-id>
	case regV1OutplansID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/v1/outplans/<outplan-id>"
		response, err = h.v1OutplansIDGet(ctx, m)

	case regV1OutplansID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/v1/outplans/<outplan-id>"
		response, err = h.v1OutplansIDDelete(ctx, m)

	case regV1OutplansID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/v1/outplans/<outplan-id>"
		response, err = h.v1OutplansIDPut(ctx, m)

	// /v1/outrplans/<outplan-id>/dials
	case regV1OutplansIDDials.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/v1/outdials/<outplan-id>/dials"
		response, err = h.v1OutplansIDDialsPut(ctx, m)

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
