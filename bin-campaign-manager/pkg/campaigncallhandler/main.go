package campaigncallhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package campaigncallhandler -destination ./mock_campaigncallhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/dbhandler"
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
