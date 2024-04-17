package flowhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package flowhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	cmcustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/actionhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/activeflowhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

type flowHandler struct {
	util          utilhandler.UtilHandler
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	actionHandler     actionhandler.ActionHandler
	activeflowHandler activeflowhandler.ActiveflowHandler
}

// FlowHandler interface
type FlowHandler interface {
	ActionGet(ctx context.Context, flowID uuid.UUID, actionID uuid.UUID) (*action.Action, error)

	Create(
		ctx context.Context,
		customerID uuid.UUID,
		flowType flow.Type,
		name string,
		detail string,
		persist bool,
		actions []action.Action,
	) (*flow.Flow, error)
	Delete(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
	Get(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
	Gets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*flow.Flow, error)
	Update(ctx context.Context, id uuid.UUID, name, detail string, actions []action.Action) (*flow.Flow, error)
	UpdateActions(ctx context.Context, id uuid.UUID, actions []action.Action) (*flow.Flow, error)

	EventCustomerDeleted(ctx context.Context, cu *cmcustomer.Customer) error
}

// NewFlowHandler return FlowHandler
func NewFlowHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	actionHandler actionhandler.ActionHandler,
	activeflowHandler activeflowhandler.ActiveflowHandler,
) FlowHandler {
	h := &flowHandler{
		util:          utilhandler.NewUtilHandler(),
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,

		actionHandler:     actionHandler,
		activeflowHandler: activeflowHandler,
	}

	return h
}
