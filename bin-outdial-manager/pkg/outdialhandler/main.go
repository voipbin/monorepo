package outdialhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package outdialhandler -destination ./mock_outdialhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdial"
	"gitlab.com/voipbin/bin-manager/outdial-manager.git/pkg/dbhandler"
)

// outdialHandler defines
type outdialHandler struct {
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
}

// OutdialHandler interface
type OutdialHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		campaignID uuid.UUID,
		name string,
		detail string,
		data string,
	) (*outdial.Outdial, error)
	Delete(ctx context.Context, id uuid.UUID) (*outdial.Outdial, error)
	Get(ctx context.Context, id uuid.UUID) (*outdial.Outdial, error)
	GetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*outdial.Outdial, error)
	UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*outdial.Outdial, error)
	UpdateCampaignID(ctx context.Context, id, campaignID uuid.UUID) (*outdial.Outdial, error)
	UpdateData(ctx context.Context, id uuid.UUID, data string) (*outdial.Outdial, error)
}

// NewOutdialHandler return OutdialHandler
func NewOutdialHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
) OutdialHandler {
	h := &outdialHandler{
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
	}

	return h
}
