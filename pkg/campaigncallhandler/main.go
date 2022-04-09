package campaigncallhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package campaigncallhandler -destination ./mock_campaigncallhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/dbhandler"
)

// campaigncallHandler defines
type campaigncallHandler struct {
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
}

// CampaigncallHandler interface
type CampaigncallHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		campaignID uuid.UUID,
		outplanID uuid.UUID,
		outdialID uuid.UUID,
		outdialTargetID uuid.UUID,
		queueID uuid.UUID,
		activeflowID uuid.UUID,
		referenceType campaigncall.ReferenceType,
		referenceID uuid.UUID,
		source *cmaddress.Address,
		destination *cmaddress.Address,
		destinationIndex int,
		tryCount int,
	) (*campaigncall.Campaigncall, error)
	Get(ctx context.Context, id uuid.UUID) (*campaigncall.Campaigncall, error)
	GetsByCampaignID(ctx context.Context, campaignID uuid.UUID, token string, limit uint64) ([]*campaigncall.Campaigncall, error)
	GetsByCampaignIDAndStatus(ctx context.Context, campaignID uuid.UUID, status campaigncall.Status, token string, limit uint64) ([]*campaigncall.Campaigncall, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status campaigncall.Status) (*campaigncall.Campaigncall, error)
}

// NewCampaigncallHandler returns CampaignCallHandler
func NewCampaigncallHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
) CampaigncallHandler {
	h := &campaigncallHandler{
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
	}

	return h
}
