package flowhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/flowhandler/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/flowhandler/models/flow"
)

// FlowGet returns flow
func (h *flowHandler) FlowGet(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {
	resFlow, err := h.db.FlowGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return resFlow, nil
}

// FlowCreate creates a flow
func (h *flowHandler) FlowCreate(ctx context.Context, flow *flow.Flow, persist bool) (*flow.Flow, error) {

	if flow.ID == uuid.Nil {
		flow.ID = uuid.Must(uuid.NewV4())
	}

	// set action id
	if persist == false && len(flow.Actions) > 0 {
		nextID := action.IDStart
		for _, act := range flow.Actions {
			act.ID = nextID
			nextID = uuid.Must(uuid.NewV4())
		}
		flow.Actions[len(flow.Actions)-1].Next = action.IDFinish
	}

	switch {
	case persist == true:
		if err := h.db.FlowCreate(ctx, flow); err != nil {
			return nil, err
		}

	default:
		if err := h.db.FlowSetToCache(ctx, flow); err != nil {
			return nil, err
		}
	}

	resFlow, err := h.FlowGet(ctx, flow.ID)
	if err != nil {
		return nil, err
	}

	return resFlow, nil
}
