package flowhandler

//go:generate mockgen -destination ./mock_flowhandler_flowhandler.go -package flowhandler gitlab.com/voipbin/bin-manager/flow-manager/pkg/flowhandler FlowHandler

import (
	"context"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/flowhandler/models/flow"
)

type flowHandler struct {
	db dbhandler.DBHandler
}

// FlowHandler interface
type FlowHandler interface {
	FlowGet(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
	FlowCreate(ctx context.Context, flow *flow.Flow) (*flow.Flow, error)

	ActionGet(ctx context.Context, flowID uuid.UUID, actionID uuid.UUID) (*flow.Action, error)
}

// NewFlowHandler return FlowHandler
func NewFlowHandler(db dbhandler.DBHandler) FlowHandler {
	h := &flowHandler{
		db: db,
	}

	return h
}

// FlowGet returns flow
func (h *flowHandler) FlowGet(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {
	resFlow, err := h.db.FlowGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return resFlow, nil
}

// FlowCreate creates a flow
func (h *flowHandler) FlowCreate(ctx context.Context, flow *flow.Flow) (*flow.Flow, error) {
	flow.ID = uuid.Must(uuid.NewV4())

	if err := h.db.FlowCreate(ctx, flow); err != nil {
		return nil, err
	}

	resFlow, err := h.FlowGet(ctx, flow.ID)
	if err != nil {
		return nil, err
	}

	return resFlow, nil
}

// FlowGet returns flow
func (h *flowHandler) ActionGet(ctx context.Context, flowID uuid.UUID, actionID uuid.UUID) (*flow.Action, error) {
	flow, err := h.FlowGet(ctx, flowID)
	if err != nil {
		return nil, err
	}

	for _, action := range flow.Actions {
		if action.ID == actionID {
			return &action, nil
		}
	}

	return nil, dbhandler.ErrNotFound
}
