package transferhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package transferhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-transfer-manager/models/transfer"
	"monorepo/bin-transfer-manager/pkg/dbhandler"
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
