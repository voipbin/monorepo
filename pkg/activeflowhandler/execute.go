package activeflowhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// Execute executes the actions.
// This starts the active-flow.
func (h *activeflowHandler) Execute(ctx context.Context, activeflowID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Execute",
		"activeflow_id": activeflowID,
	})

	// get next action from the active
	nextStackID, nextAction, err := h.getNextAction(ctx, activeflowID, action.IDStart)
	if err != nil {
		log.Errorf("Could not get next action. err: %v", err)
		return err
	}
	log.WithField("action", nextAction).Debug("Found next action.")

	// execute the active action
	_, err = h.executeAction(ctx, activeflowID, nextStackID, nextAction)
	if err != nil {
		log.Errorf("Could not execute the active action. err: %v", err)
		return err
	}

	return nil
}

// ExecuteNextAction gets the next action from the activeflow and execute.
func (h *activeflowHandler) ExecuteNextAction(ctx context.Context, activeflowID uuid.UUID, caID uuid.UUID) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "ExecuteNextAction",
		"activeflow_id":     activeflowID,
		"current_action_id": caID,
	})

	// get next action from the active
	nextStackID, nextAction, err := h.getNextAction(ctx, activeflowID, caID)
	if err != nil {
		log.Errorf("Could not get next action. Deleting activeflow. err: %v", err)
		_, _ = h.Delete(ctx, activeflowID)
		return nil, err
	}
	log.WithField("action", nextAction).Debug("Found next action.")

	// execute the active action
	res, err := h.executeAction(ctx, activeflowID, nextStackID, nextAction)
	if err != nil {
		log.Errorf("Could not execute the active action. Deleting activeflow. err: %v", err)
		_, _ = h.Delete(ctx, activeflowID)
		return nil, err
	}

	return res, nil
}

// executeAction execute the active action.
// some of active-actions are flow-manager need to run.
func (h *activeflowHandler) executeAction(ctx context.Context, activeflowID uuid.UUID, stackID uuid.UUID, act *action.Action) (*action.Action, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "executeAction",
			"activeflow_id": activeflowID,
			"stack_id":      stackID,
			"action_id":     act.ID,
		},
	)
	log.Debugf("Executing the action. action_id: %s, action_type: %s", act.ID, act.Type)

	// update current action in active-flow
	af, err := h.updateCurrentAction(ctx, activeflowID, stackID, act)
	if err != nil {
		log.Errorf("Could not update the current action. err: %v", err)
		return nil, fmt.Errorf("could not update the current action. err: %v", err)
	}

	switch act.Type {
	case action.TypeAgentCall:
		if errHandle := h.actionHandleAgentCall(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the agent_call action correctly. err: %v", err)
			return nil, err
		}
		return h.ExecuteNextAction(ctx, activeflowID, act.ID)

	case action.TypeBranch:
		if errHandle := h.actionHandleBranch(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the branch action correctly. err: %v", err)
			return nil, err
		}
		return h.ExecuteNextAction(ctx, activeflowID, act.ID)

	case action.TypeCall:
		if errHandle := h.actionHandleCall(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the call action correctly. err: %v", err)
			return nil, err
		}
		return h.ExecuteNextAction(ctx, activeflowID, act.ID)

	case action.TypeConditionCallDigits:
		if errHandle := h.actionHandleConditionCallDigits(ctx, af); errHandle != nil {
			return nil, err
		}
		return h.ExecuteNextAction(ctx, activeflowID, act.ID)

	case action.TypeConditionCallStatus:
		if errHandle := h.actionHandleConditionCallStatus(ctx, af); errHandle != nil {
			return nil, err
		}
		return h.ExecuteNextAction(ctx, activeflowID, act.ID)

	case action.TypeConferenceJoin:
		if errHandle := h.actionHandleConferenceJoin(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the conference_join action correctly. err: %v", err)
			return nil, err
		}
		return h.ExecuteNextAction(ctx, activeflowID, act.ID)

	case action.TypeConnect:
		if errHandle := h.actionHandleConnect(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the connect action correctly. err: %v", err)
			return nil, err
		}
		return h.ExecuteNextAction(ctx, activeflowID, act.ID)

	case action.TypeGoto:
		if errHandle := h.actionHandleGoto(ctx, af, act); errHandle != nil {
			log.Errorf("Could not handle the goto action correctly. err: %v", err)
			return nil, err
		}
		return h.ExecuteNextAction(ctx, activeflowID, act.ID)

	case action.TypeMessageSend:
		if errHandler := h.actionHandleMessageSend(ctx, af); errHandler != nil {
			log.Errorf("Could not handle the message_send action correctly. err: %v", errHandler)
			return nil, err
		}
		return h.ExecuteNextAction(ctx, activeflowID, act.ID)

	case action.TypeFetch:
		if errHandle := h.actionHandleFetch(ctx, af, act); errHandle != nil {
			log.Errorf("Could not handle the patch action correctly. err: %v", err)
			return nil, err
		}
		return h.ExecuteNextAction(ctx, activeflowID, act.ID)

	case action.TypeFetchFlow:
		if errHandle := h.actionHandleFetchFlow(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the patch_flow action correctly. err: %v", err)
			return nil, err
		}
		return h.ExecuteNextAction(ctx, activeflowID, act.ID)

	case action.TypeQueueJoin:
		if errHandle := h.actionHandleQueueJoin(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the queue_join action correctly. err: %v", err)
			return nil, err
		}
		return h.ExecuteNextAction(ctx, activeflowID, act.ID)

	case action.TypeTranscribeRecording:
		if err := h.actionHandleTranscribeRecording(ctx, af, act); err != nil {
			log.Errorf("Could not handle the recording_to_text action correctly. err: %v", err)
			// we can move on to the next action even it's failed
		}
		return h.ExecuteNextAction(ctx, activeflowID, act.ID)

	case action.TypeTranscribeStart:
		if err := h.actionHandleTranscribeStart(ctx, af, act); err != nil {
			log.Errorf("Could not start the transcribe. err: %v", err)
			// we can move on to the next action even it's failed
		}
		return h.ExecuteNextAction(ctx, activeflowID, act.ID)

	case action.TypeVariableSet:
		if err := h.actionHandleVariableSet(ctx, af); err != nil {
			log.Errorf("Could not handle the variable_set. err: %v", err)
		}
		return h.ExecuteNextAction(ctx, activeflowID, act.ID)
	}

	return act, nil
}
