package activeflowhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package activeflowhandler -destination ./mock_activeflowhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/actionhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

const (
	maxActiveFlowExecuteCount = 100
)

// activeflowHandler defines
type activeflowHandler struct {
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	actionHandler actionhandler.ActionHandler
}

// ActiveflowHandler defines
type ActiveflowHandler interface {
	Create(ctx context.Context, referenceType activeflow.ReferenceType, referenceID, flowID uuid.UUID) (*activeflow.Activeflow, error)
	Delete(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error)

	GetNextAction(ctx context.Context, callID uuid.UUID, caID uuid.UUID) (*action.Action, error)
	SetForwardActionID(ctx context.Context, callID uuid.UUID, actionID uuid.UUID, forwardNow bool) error

	Execute(ctx context.Context, id uuid.UUID) error
}

// NewActiveflowHandler returns new ActiveflowHandler
func NewActiveflowHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	actionHandler actionhandler.ActionHandler,
) ActiveflowHandler {

	return &activeflowHandler{
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
		actionHandler: actionHandler,
	}
}
