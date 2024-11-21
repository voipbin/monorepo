package campaigncallhandler

//go:generate mockgen -package campaigncallhandler -destination ./mock_campaigncallhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	cmcall "monorepo/bin-call-manager/models/call"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
)

// campaigncallHandler defines
type campaigncallHandler struct {
	util          utilhandler.UtilHandler
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
		flowID uuid.UUID,

		referenceType campaigncall.ReferenceType,
		referenceID uuid.UUID,
		source *commonaddress.Address,
		destination *commonaddress.Address,
		destinationIndex int,
		tryCount int,
	) (*campaigncall.Campaigncall, error)
	Delete(ctx context.Context, id uuid.UUID) (*campaigncall.Campaigncall, error)
	Get(ctx context.Context, id uuid.UUID) (*campaigncall.Campaigncall, error)
	GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*campaigncall.Campaigncall, error)
	GetByActiveflowID(ctx context.Context, activeflowID uuid.UUID) (*campaigncall.Campaigncall, error)
	GetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*campaigncall.Campaigncall, error)
	GetsByCampaignID(ctx context.Context, campaignID uuid.UUID, token string, limit uint64) ([]*campaigncall.Campaigncall, error)
	GetsByCampaignIDAndStatus(ctx context.Context, campaignID uuid.UUID, status campaigncall.Status, token string, limit uint64) ([]*campaigncall.Campaigncall, error)
	GetsOngoingByCampaignID(ctx context.Context, campaignID uuid.UUID, token string, limit uint64) ([]*campaigncall.Campaigncall, error)

	// status
	Done(ctx context.Context, id uuid.UUID, result campaigncall.Result) (*campaigncall.Campaigncall, error)
	Progressing(ctx context.Context, id uuid.UUID) (*campaigncall.Campaigncall, error)

	// eventhandle
	EventHandleReferenceCallHungup(ctx context.Context, c *cmcall.Call, cc *campaigncall.Campaigncall) (*campaigncall.Campaigncall, error)
	EventHandleActiveflowDeleted(ctx context.Context, cc *campaigncall.Campaigncall) (*campaigncall.Campaigncall, error)
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
