package flowhandler

//go:generate mockgen -package flowhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/activeflowhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
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

	CountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error)
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		flowType flow.Type,
		name string,
		detail string,
		persist bool,
		actions []action.Action,
		onCompleteFlowID uuid.UUID,
	) (*flow.Flow, error)
	Delete(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
	Get(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
	List(ctx context.Context, token string, size uint64, filters map[flow.Field]any) ([]*flow.Flow, error)
	Update(
		ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
		actions []action.Action,
		onCompleteFlowID uuid.UUID,
	) (*flow.Flow, error)
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
