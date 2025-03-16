package actionhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-flow-manager/models/action"
)

// GenerateFlowActions generates actions for flow.
func (h *actionHandler) GenerateFlowActions(ctx context.Context, actions []action.Action) ([]action.Action, error) {
	log := logrus.WithField("func", "GenerateFlowActions")

	res := []action.Action{}
	// validate actions
	if err := h.ValidateActions(actions); err != nil {
		log.Errorf("Could not pass the action validation. err: %v", err)
		return nil, err
	}

	// set action id
	for _, a := range actions {
		tmpAction := a
		if tmpAction.ID == uuid.Nil {
			tmpAction.ID = h.utilHandler.UUIDCreate()
		}
		res = append(res, tmpAction)
	}

	return res, nil
}
