package transferhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package transferhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/transfer-manager.git/models/transfer"
	"gitlab.com/voipbin/bin-manager/transfer-manager.git/pkg/dbhandler"
)

// TransferHandler define
type TransferHandler interface {
	ServiceStart(ctx context.Context, transferType transfer.Type, TransfererCallID uuid.UUID, transfereeAddresses []commonaddress.Address) (*transfer.Transfer, error)

	GetByGroupcallID(ctx context.Context, groupcallID uuid.UUID) (*transfer.Transfer, error)
	GetByTransfererCallID(ctx context.Context, transfererCallID uuid.UUID) (*transfer.Transfer, error)

	TransfereeAnswer(ctx context.Context, tr *transfer.Transfer, groupcall *cmgroupcall.Groupcall) error
	TransfereeHangup(ctx context.Context, tr *transfer.Transfer, gc *cmgroupcall.Groupcall) error

	TransfererHangup(ctx context.Context, tr *transfer.Transfer, transfererCall *cmcall.Call) error
}

type transferHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler
}

// NewTransferHandler return transfer handler interface
func NewTransferHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
) TransferHandler {

	h := &transferHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
		db:            db,
	}

	return h
}
