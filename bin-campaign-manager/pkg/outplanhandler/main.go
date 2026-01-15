package outplanhandler

//go:generate mockgen -package outplanhandler -destination ./mock_outplanhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-campaign-manager/models/outplan"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
)

// outplanHandler defines
type outplanHandler struct {
	util          utilhandler.UtilHandler
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
}

// OutplanHandler interface
type OutplanHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
		source *commonaddress.Address,
		dialTimeout int,
		tryInterval int,
		maxTryCount0 int,
		maxTryCount1 int,
		maxTryCount2 int,
		maxTryCount3 int,
		maxTryCount4 int,
	) (*outplan.Outplan, error)
	Delete(ctx context.Context, id uuid.UUID) (*outplan.Outplan, error)
	Get(ctx context.Context, id uuid.UUID) (*outplan.Outplan, error)
	Gets(ctx context.Context, token string, limit uint64, filters map[outplan.Field]any) ([]*outplan.Outplan, error)
	GetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*outplan.Outplan, error)
	UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*outplan.Outplan, error)
	UpdateDialInfo(ctx context.Context, id uuid.UUID, source *commonaddress.Address, dialTimeout, tryInterval, maxTryCount0, maxTryCount1, maxTryCount2, maxTryCount3, maxTryCount4 int) (*outplan.Outplan, error)
}

// NewOutplanHandler return OutplanHandler
func NewOutplanHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
) OutplanHandler {
	h := &outplanHandler{
		util:          utilhandler.NewUtilHandler(),
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
	}

	return h
}
