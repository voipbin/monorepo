package campaignhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package campaignhandler -destination ./mock_campaignhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/campaigncallhandler"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/outplanhandler"
)

// campaignHandler defines
type campaignHandler struct {
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

	UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*campaign.Campaign, error)
	UpdateResourceInfo(ctx context.Context, id, outplanID, outdialID, queueID uuid.UUID) (*campaign.Campaign, error)
	UpdateNextCampaignID(ctx context.Context, id, nextCampaignID uuid.UUID) (*campaign.Campaign, error)
	// UpdateStatus(ctx context.Context, id uuid.UUID, status campaign.Status) (*campaign.Campaign, error)
	UpdateServiceLevel(ctx context.Context, id uuid.UUID, serviceLevel int) (*campaign.Campaign, error)
	UpdateActions(ctx context.Context, id uuid.UUID, actions []fmaction.Action) (*campaign.Campaign, error)

	UpdateStatusRun(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error)
	UpdateStatusStopping(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error)

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
		db:                  db,
		reqHandler:          reqHandler,
		notifyHandler:       notifyHandler,
		campaigncallHandler: campaigncallHandler,
		outplanHandler:      outplanHandler,
	}

	return h
}
