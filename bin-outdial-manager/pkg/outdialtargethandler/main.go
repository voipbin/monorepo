package outdialtargethandler

//go:generate mockgen -package outdialtargethandler -destination ./mock_outdialtargethandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-outdial-manager/models/outdialtarget"
	"monorepo/bin-outdial-manager/pkg/dbhandler"
)

// outdialTargetHandler defines
type outdialTargetHandler struct {
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
}

// OutdialTargetHandler interface
type OutdialTargetHandler interface {
	Create(
		ctx context.Context,
		outdialID uuid.UUID,
		name string,
		detail string,
		data string,
		destination0 *commonaddress.Address,
		destination1 *commonaddress.Address,
		destination2 *commonaddress.Address,
		destination3 *commonaddress.Address,
		destination4 *commonaddress.Address,
	) (*outdialtarget.OutdialTarget, error)
	Delete(ctx context.Context, id uuid.UUID) (*outdialtarget.OutdialTarget, error)

	Get(ctx context.Context, id uuid.UUID) (*outdialtarget.OutdialTarget, error)
	GetsByOutdialID(ctx context.Context, outdialID uuid.UUID, token string, limit uint64) ([]*outdialtarget.OutdialTarget, error)
	GetAvailable(
		ctx context.Context,
		outdialID uuid.UUID,
		tryCount0 int,
		tryCount1 int,
		tryCount2 int,
		tryCount3 int,
		tryCount4 int,
		limit uint64,
	) ([]*outdialtarget.OutdialTarget, error)

	UpdateStatus(ctx context.Context, id uuid.UUID, status outdialtarget.Status) (*outdialtarget.OutdialTarget, error)
	UpdateProgressing(ctx context.Context, id uuid.UUID, destinationIndex int) (*outdialtarget.OutdialTarget, error)
}

// NewOutdialTargetHandler return OutdialTargetHandler
func NewOutdialTargetHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
) OutdialTargetHandler {
	h := &outdialTargetHandler{
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
	}

	return h
}
