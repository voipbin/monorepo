package outdialhandler

//go:generate mockgen -package outdialhandler -destination ./mock_outdialhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-outdial-manager/models/outdial"
	"monorepo/bin-outdial-manager/pkg/dbhandler"
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
	Gets(ctx context.Context, token string, limit uint64, filters map[outdial.Field]any) ([]*outdial.Outdial, error)
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
