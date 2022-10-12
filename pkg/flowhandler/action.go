package flowhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

// ActionGet returns corresponded action.
func (h *flowHandler) ActionGet(ctx context.Context, flowID uuid.UUID, actionID uuid.UUID) (*action.Action, error) {
	flow, err := h.Get(ctx, flowID)
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
