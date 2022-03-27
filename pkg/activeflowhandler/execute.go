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
func (h *activeflowHandler) Execute(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Execute",
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

// executeAction execute the active action.
// some of active-actions are flow-manager need to run.
func (h *activeflowHandler) executeAction(ctx context.Context, id uuid.UUID, act *action.Action) (*action.Action, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "executeAction",
			"id":        id,
			"action_id": act.ID,
		},
	)

	// update current action in active-flow
	af, err := h.updateCurrentAction(ctx, id, act)
	if err != nil {
		log.Errorf("Could not update the current action. err: %v", err)
		return nil, fmt.Errorf("could not update the current action. err: %v", err)
	}

	switch act.Type {
	case action.TypeAgentCall:
		if errHandle := h.actionHandleAgentCall(ctx, id, af); errHandle != nil {
			log.Errorf("Could not handle the agent_call action correctly. err: %v", err)
			return nil, err
		}
		return h.GetNextAction(ctx, id, act.ID)

	case action.TypeBranch:
		if errHandle := h.actionHandleBranch(ctx, id, af); errHandle != nil {
			log.Errorf("Could not handle the branch action correctly. err: %v", err)
			return nil, err
		}
		return h.GetNextAction(ctx, id, act.ID)

	case action.TypeCall:
		if errHandle := h.actionHandleCall(ctx, id, af); errHandle != nil {
			log.Errorf("Could not handle the call action correctly. err: %v", err)
			return nil, err
		}
		return h.GetNextAction(ctx, id, act.ID)

	case action.TypeConditionCallDigits:
		if errHandle := h.actionHandleConditionCallDigits(ctx, id, af); errHandle != nil {
			return nil, err
		}
		return h.GetNextAction(ctx, id, act.ID)

	case action.TypeConditionCallStatus:
		if errHandle := h.actionHandleConditionCallStatus(ctx, id, af); errHandle != nil {
			return nil, err
		}
		return h.GetNextAction(ctx, id, act.ID)

	case action.TypeConferenceJoin:
		if errHandle := h.actionHandleConferenceJoin(ctx, id, af); errHandle != nil {
			log.Errorf("Could not handle the conference_join action correctly. err: %v", err)
			return nil, err
		}
		return h.GetNextAction(ctx, id, act.ID)

	case action.TypeConnect:
		if errHandle := h.actionHandleConnect(ctx, id, af); errHandle != nil {
			log.Errorf("Could not handle the connect action correctly. err: %v", err)
			return nil, err
		}
		return h.GetNextAction(ctx, id, act.ID)

	case action.TypeGoto:
		if errHandle := h.actionHandleGoto(ctx, id, af); errHandle != nil {
			log.Errorf("Could not handle the goto action correctly. err: %v", err)
			return nil, err
		}
		return h.GetNextAction(ctx, id, act.ID)

	case action.TypeMessageSend:
		if errHandler := h.actionHandleMessageSend(ctx, id, af); errHandler != nil {
			log.Errorf("Could not handle the message_send action correctly. err: %v", errHandler)
			return nil, err
		}
		return h.GetNextAction(ctx, id, act.ID)

	case action.TypePatch:
		if errHandle := h.actionHandlePatch(ctx, id, af); errHandle != nil {
			log.Errorf("Could not handle the patch action correctly. err: %v", err)
			return nil, err
		}
		return h.GetNextAction(ctx, id, act.ID)

	case action.TypePatchFlow:
		if errHandle := h.actionHandlePatchFlow(ctx, id, af); errHandle != nil {
			log.Errorf("Could not handle the patch_flow action correctly. err: %v", err)
			return nil, err
		}
		return h.GetNextAction(ctx, id, act.ID)

	case action.TypeQueueJoin:
		if errHandle := h.actionHandleQueueJoin(ctx, id, af); errHandle != nil {
			log.Errorf("Could not handle the queue_join action correctly. err: %v", err)
			return nil, err
		}
		return h.GetNextAction(ctx, id, act.ID)

	case action.TypeTranscribeRecording:
		if err := h.actionHandleTranscribeRecording(ctx, af, id, act); err != nil {
			log.Errorf("Could not handle the recording_to_text action correctly. err: %v", err)
			// we can move on to the next action even it's failed
		}
		return h.GetNextAction(ctx, id, act.ID)

	case action.TypeTranscribeStart:
		if err := h.actionHandleTranscribeStart(ctx, af, id, act); err != nil {
			log.Errorf("Could not start the transcribe. err: %v", err)
			// we can move on to the next action even it's failed
		}
		return h.GetNextAction(ctx, id, act.ID)
	}

	return act, nil
}
