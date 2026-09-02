package campaignhandler

//go:generate mockgen -package campaignhandler -destination ./mock_campaignhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
	"monorepo/bin-campaign-manager/pkg/outplanhandler"
)

var (
	metricsNamespace = "campaign_manager"

	promCampaignCreateTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "campaign_create_total",
			Help:      "Total number of campaigns created.",
		},
	)

	promCampaignStatusRunTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "campaign_status_run_total",
			Help:      "Total number of campaigns set to run status.",
		},
	)

	promCampaignStatusStopTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "campaign_status_stop_total",
			Help:      "Total number of campaigns stopped.",
		},
	)

	promCampaignExecuteTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "campaign_execute_total",
			Help:      "Total number of campaign execution loops.",
		},
	)

	// promCampaignFlowDeleteFailedTotal counts failures to delete a campaign's
	// backing flow during campaign delete. The delete is best-effort (the
	// campaign itself is already deleted by the time this runs) so a failure
	// here does not fail the campaign delete request; this counter is the only
	// durable, always-on signal that a flow was left orphaned, since the flow
	// count is capped per customer (see bin-flow-manager's maxFlowCount).
	// reason="not_found" is a benign, idempotent re-delete of an already-gone
	// flow; reason="error" is a real failure and a leak candidate.
	promCampaignFlowDeleteFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "campaign_flow_delete_failed_total",
			Help:      "Total number of failures to delete a campaign's backing flow, by reason.",
		},
		[]string{"reason"},
	)
)

func init() {
	prometheus.MustRegister(
		promCampaignCreateTotal,
		promCampaignStatusRunTotal,
		promCampaignStatusStopTotal,
		promCampaignExecuteTotal,
		promCampaignFlowDeleteFailedTotal,
	)

	// pre-initialize both label series so they read 0 from process start,
	// rather than "No data" until the first failure. Without this, a
	// dashboard panel on this metric can't distinguish "healthy, zero
	// failures" from "metric never registered / service not being scraped."
	promCampaignFlowDeleteFailedTotal.WithLabelValues("not_found")
	promCampaignFlowDeleteFailedTotal.WithLabelValues("error")
}

// campaignHandler defines
type campaignHandler struct {
	util          utilhandler.UtilHandler
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	campaigncallHandler campaigncallhandler.CampaigncallHandler
	outplanHandler      outplanhandler.OutplanHandler
}

// CampaignHandler interface
type CampaignHandler interface {
	Create(
		ctx context.Context,
		id uuid.UUID,
		customerID uuid.UUID,
		campaignType campaign.Type,
		name string,
		detail string,
		actions []fmaction.Action,
		serviceLevel int,
		endHandle campaign.EndHandle,
		outplanID uuid.UUID,
		outdialID uuid.UUID,
		queueID uuid.UUID,
		nextCampaignID uuid.UUID,
	) (*campaign.Campaign, error)
	Delete(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error)
	Get(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error)
	List(ctx context.Context, token string, limit uint64, filters map[campaign.Field]any) ([]*campaign.Campaign, error)
	ListByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*campaign.Campaign, error)

	UpdateBasicInfo(ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
		campaignType campaign.Type,
		serviceLevel int,
		endHandle campaign.EndHandle,
	) (*campaign.Campaign, error)
	UpdateResourceInfo(ctx context.Context, id, outplanID, outdialID, queueID, nextCampaignID uuid.UUID) (*campaign.Campaign, error)
	UpdateNextCampaignID(ctx context.Context, id, nextCampaignID uuid.UUID) (*campaign.Campaign, error)
	UpdateServiceLevel(ctx context.Context, id uuid.UUID, serviceLevel int) (*campaign.Campaign, error)
	UpdateActions(ctx context.Context, id uuid.UUID, actions []fmaction.Action) (*campaign.Campaign, error)

	UpdateStatus(ctx context.Context, id uuid.UUID, status campaign.Status) (*campaign.Campaign, error)

	Execute(ctx context.Context, id uuid.UUID)

	EventHandleActiveflowDeleted(ctx context.Context, campaignID uuid.UUID) error
	EventHandleReferenceCallHungup(ctx context.Context, campaignID uuid.UUID) error
}

// NewCampaignHandler return CampaignHandler
func NewCampaignHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	campaigncallHandler campaigncallhandler.CampaigncallHandler,
	outplanHandler outplanhandler.OutplanHandler,
) CampaignHandler {
	h := &campaignHandler{
		util:                utilhandler.NewUtilHandler(),
		db:                  db,
		reqHandler:          reqHandler,
		notifyHandler:       notifyHandler,
		campaigncallHandler: campaigncallHandler,
		outplanHandler:      outplanHandler,
	}

	return h
}
