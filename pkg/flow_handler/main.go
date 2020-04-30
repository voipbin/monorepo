package flowhandler

//go:generate mockgen -destination ./mock_flowhandler_flowhandler.go -package flowhandler gitlab.com/voipbin/bin-manager/flow-manager/pkg/flow_handler FlowHandler

import (
	"context"

	"github.com/gofrs/uuid"
	dbhandler "gitlab.com/voipbin/bin-manager/flow-manager/pkg/db_handler"
	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/flow"
)

type flowHandler struct {
	db dbhandler.DBHandler
}

// FlowHandler interface
type FlowHandler interface {
	FlowGet(ctx context.Context, id, revision uuid.UUID) (*flow.Flow, error)
	FlowCreate(ctx context.Context, flow *flow.Flow) (*flow.Flow, error)

	ActionGet(ctx context.Context, flowID uuid.UUID, revision, actionID uuid.UUID) (*flow.Action, error)
}

// NewFlowHandler return FlowHandler
func NewFlowHandler(db dbhandler.DBHandler) FlowHandler {
	h := &flowHandler{
		db: db,
	}

	return h
}

// FlowGet returns flow
func (h *flowHandler) FlowGet(ctx context.Context, id, revision uuid.UUID) (*flow.Flow, error) {
	resFlow, err := h.db.FlowGet(ctx, id, revision)
	if err != nil {
		return nil, err
	}

	return resFlow, nil
}

// FlowCreate creates a flow
func (h *flowHandler) FlowCreate(ctx context.Context, flow *flow.Flow) (*flow.Flow, error) {
	flow.ID = uuid.Must(uuid.NewV4())
	flow.Revision = uuid.Nil

	if err := h.db.FlowCreate(ctx, flow); err != nil {
		return nil, err
	}

	resFlow, err := h.FlowGet(ctx, flow.ID, flow.Revision)
	if err != nil {
		return nil, err
	}

	return resFlow, nil
}

// FlowGet returns flow
func (h *flowHandler) ActionGet(ctx context.Context, flowID uuid.UUID, revision, actionID uuid.UUID) (*flow.Action, error) {
	flow, err := h.FlowGet(ctx, flowID, revision)
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
