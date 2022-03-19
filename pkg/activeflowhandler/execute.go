package activeflowhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// Execute executes the actions.
// This starts the active-flow.
func (h *activeflowHandler) Execute(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "ActiveFlowNextActionGet",
		"id":   id,
	})

	// get next action from the active
	nextAction, err := h.getNextAction(ctx, id, action.IDStart)
	if err != nil {
		log.Errorf("Could not get next action. err: %v", err)
		return err
	}
	log.WithField("action", nextAction).Debug("Found next action.")

	// execute the active action
	_, err = h.executeAction(ctx, id, nextAction)
	if err != nil {
		log.Errorf("Could not execute the active action. err: %v", err)
		return err
	}

	return nil
}
