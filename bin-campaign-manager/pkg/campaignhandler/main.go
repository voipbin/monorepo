package campaignhandler

//go:generate mockgen -package campaignhandler -destination ./mock_campaignhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
	"monorepo/bin-campaign-manager/pkg/outplanhandler"
)

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
	GetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*campaign.Campaign, error)

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
