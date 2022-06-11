package outdialtargethandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package outdialtargethandler -destination ./mock_outdialtargethandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"
	"gitlab.com/voipbin/bin-manager/outdial-manager.git/pkg/dbhandler"
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
