package flowhandler

//go:generate mockgen -destination ./mock_flowhandler_flowhandler.go -package flowhandler gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler FlowHandler

import (
	"context"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/flow"
)

type flowHandler struct {
	db dbhandler.DBHandler
}

// FlowHandler interface
type FlowHandler interface {
	FlowGet(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
	FlowCreate(ctx context.Context, flow *flow.Flow, persist bool) (*flow.Flow, error)

	ActionGet(ctx context.Context, flowID uuid.UUID, actionID uuid.UUID) (*action.Action, error)
}

// NewFlowHandler return FlowHandler
func NewFlowHandler(db dbhandler.DBHandler) FlowHandler {
	h := &flowHandler{
		db: db,
	}

	return h
}
