package activeflowhandler

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
)

// Execute executes the actions.
// This starts the activeflow.
func (h *activeflowHandler) Execute(ctx context.Context, activeflowID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Execute",
		"activeflow_id": activeflowID,
	})

	// execute the next action
	_, err := h.ExecuteNextAction(ctx, activeflowID, action.IDStart)
	if err != nil {
		return errors.Wrapf(err, "could not execute the next action. activeflow_id: %s", activeflowID)
	}
	log.Debugf("Completed the activeflow execution. activeflow_id: %s", activeflowID)

	return nil
}

// ExecuteContinue continues the actions.
// This unblocks the blocked activeflow and triggers the flow execution.
func (h *activeflowHandler) ExecuteContinue(ctx context.Context, activeflowID uuid.UUID, caID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "ExecuteContinue",
		"activeflow_id":     activeflowID,
		"current_action_id": caID,
	})

	// get activeflow info
	af, err := h.Get(ctx, activeflowID)
	if err != nil {
		return errors.Wrapf(err, "could not get activeflow info. activeflow_id: %s", activeflowID)
	}

	// todo: Maybe, we need to add the status for pause or similar for the activeflow
	// and check here.
	// but for now, we just check the current action type.
	if af.CurrentAction.Type != action.TypeBlock {
		log.Infof("The activeflow is not blocked. current_action_type: %s", af.CurrentAction.Type)
		return fmt.Errorf("the activeflow is not blocked. activeflow_id: %s", activeflowID)
	}

	// execute the next action with continue action id
	tmp, err := h.ExecuteNextAction(ctx, activeflowID, caID)
	if err != nil {
		return errors.Wrapf(err, "could not execute the next action. activeflow_id: %s", activeflowID)
	}
	log.WithField("action", tmp).Debugf("Completed the activeflow execution. activeflow_id: %s", activeflowID)

	return nil
}

// ExecuteNextAction gets the next action from the activeflow and execute.
func (h *activeflowHandler) ExecuteNextAction(ctx context.Context, activeflowID uuid.UUID, caID uuid.UUID) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "ExecuteNextAction",
		"activeflow_id":     activeflowID,
		"current_action_id": caID,
	})
	log.Debugf("Getting next action. activeflow_id: %s", activeflowID)

	for range maxNextActionLoopCount {
		// get next action from the activeflow
		af, err := h.updateNextAction(ctx, activeflowID, caID)
		if err != nil {
			log.Errorf("Could not get next action. Stopping activeflow. err: %v", err)
			h.stopWithoutReturn(ctx, activeflowID)
			return nil, errors.Wrapf(err, "could not get next action. activeflow_id: %s", activeflowID)
		}
		log.WithField("next_action", af.CurrentAction).Debugf("Found next action. action_type: %s", af.CurrentAction.Type)
		caID = af.CurrentAction.ID

		if af.CurrentAction.ID == action.IDFinish {
			log.Debugf("Next action is finish. Stop the flow execution. activeflow_id: %s", activeflowID)
			h.stopWithoutReturn(ctx, activeflowID)
			return &action.ActionFinish, nil
		}

		// execute the current action
		res, err := h.executeAction(ctx, af)
		if err != nil {
			log.Errorf("Could not execute the active action. Deleting activeflow. err: %v", err)
			h.stopWithoutReturn(ctx, activeflowID)
			return nil, errors.Wrapf(err, "could not execute the active action. activeflow_id: %s", activeflowID)
		}

		if res == &action.ActionNext {
			log.Debugf("The current action is next. Continue the flow execution. activeflow_id: %s", activeflowID)
			continue
		}

		log.WithField("action", res).Debugf("Found next action. action_type: %s", res.Type)
		return res, nil
	}

	// if we reach here, it means we have looped too many times without finding a valid action
	log.Errorf("Reached maximum loop iterations without finding a valid action. Stopping activeflow.")
	h.stopWithoutReturn(ctx, activeflowID)
	return nil, fmt.Errorf("reached maximum loop iterations without finding a valid action")
}

// executeAction execute the active action.
// some of active-actions are flow-manager need to run.
func (h *activeflowHandler) executeAction(ctx context.Context, af *activeflow.Activeflow) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "executeAction",
		"activeflow_id": af.ID,
		"stack_id":      af.CurrentStackID,
		"action_id":     af.CurrentAction.ID,
		"action_type":   af.CurrentAction.Type,
	})
	log.Debugf("Executing the action. action_id: %s, action_type: %s", af.CurrentAction.ID, af.CurrentAction.Type)

	// verify the reference type and action type
	if !h.verifyActionType(af) {
		log.Infof("The action type and reference type are not valid. Move to the next action. action_type: %s, reference_type: %s", af.CurrentAction.Type, af.ReferenceType)
		return &action.ActionNext, nil
	}

	actionType := af.CurrentAction.Type
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		promActionExecuteDuration.WithLabelValues(string(actionType)).Observe(float64(elapsed.Milliseconds()))
	}()

	switch actionType {
	case action.TypeAISummary:
		if errHandle := h.actionHandleAISummary(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the ai summary action correctly. err: %v", errHandle)
			return nil, errHandle
		}
		return &action.ActionNext, nil

	case action.TypeAITalk:
		if errHandle := h.actionHandleAITalk(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the ai talk action correctly. err: %v", errHandle)
			return nil, errHandle
		}
		return &action.ActionNext, nil

	case action.TypeAITask:
		if errHandle := h.actionHandleAITask(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the ai task action correctly. err: %v", errHandle)
			return nil, errHandle
		}
		return &action.ActionNext, nil

	case action.TypeBlock:
		if errHandle := h.actionHandleBlock(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the block action correctly. err: %v", errHandle)
			return nil, errHandle
		}

		// note: after block action, no further action is executed until unblocked.
		return &action.ActionEmpty, nil

	case action.TypeBranch:
		if errHandle := h.actionHandleBranch(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the branch action correctly. err: %v", errHandle)
			return nil, errHandle
		}
		return &action.ActionNext, nil

	case action.TypeCall:
		if errHandle := h.actionHandleCall(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the call action correctly. err: %v", errHandle)
			return nil, errHandle
		}
		return &action.ActionNext, nil

	case action.TypeConditionCallDigits:
		if errHandle := h.actionHandleConditionCallDigits(ctx, af); errHandle != nil {
			return nil, errHandle
		}
		return &action.ActionNext, nil

	case action.TypeConditionCallStatus:
		if errHandle := h.actionHandleConditionCallStatus(ctx, af); errHandle != nil {
			return nil, errHandle
		}
		return &action.ActionNext, nil

	case action.TypeConditionDatetime:
		if errHandle := h.actionHandleConditionDatetime(ctx, af); errHandle != nil {
			return nil, errHandle
		}
		return &action.ActionNext, nil

	case action.TypeConditionVariable:
		if errHandle := h.actionHandleConditionVariable(ctx, af); errHandle != nil {
			return nil, errHandle
		}
		return &action.ActionNext, nil

	case action.TypeConferenceJoin:
		if errHandle := h.actionHandleConferenceJoin(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the conference_join action correctly. err: %v", errHandle)
			return nil, errHandle
		}
		return &action.ActionNext, nil

	case action.TypeConnect:
		if errHandle := h.actionHandleConnect(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the connect action correctly. err: %v", errHandle)
			return nil, errHandle
		}
		return &action.ActionNext, nil

	case action.TypeConversationSend:
		if errHandle := h.actionHandleConversationSend(ctx, af); errHandle != nil {
			log.Errorf("Could not send the conversation message correctly. err: %v", errHandle)
		}
		return &action.ActionNext, nil

	case action.TypeEmailSend:
		if errHandle := h.actionHandleEmailSend(ctx, af); errHandle != nil {
			log.Errorf("Could not send the email correctly. err: %v", errHandle)
		}
		return &action.ActionNext, nil

	case action.TypeEmpty:
		log.Debugf("The action is empty. Move to the next action. action_id: %s", af.CurrentAction.ID)
		return &action.ActionNext, nil

	case action.TypeGoto:
		if errHandle := h.actionHandleGoto(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the goto action correctly. err: %v", errHandle)
			return nil, errHandle
		}
		return &action.ActionNext, nil

	case action.TypeMessageSend:
		if errHandle := h.actionHandleMessageSend(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the message_send action correctly. err: %v", errHandle)
			return nil, errHandle
		}
		return &action.ActionNext, nil

	case action.TypeFetch:
		if errHandle := h.actionHandleFetch(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the patch action correctly. err: %v", errHandle)
			return nil, errHandle
		}
		return &action.ActionNext, nil

	case action.TypeFetchFlow:
		if errHandle := h.actionHandleFetchFlow(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the patch_flow action correctly. err: %v", errHandle)
			return nil, errHandle
		}
		return &action.ActionNext, nil

	case action.TypeQueueJoin:
		if errHandle := h.actionHandleQueueJoin(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the queue_join action correctly. err: %v", errHandle)
			return nil, errHandle
		}
		return &action.ActionNext, nil

	case action.TypeStop:
		if errHandle := h.actionHandleStop(ctx, af); errHandle != nil {
			log.Errorf("Could not handle the stop action correctly. err: %v", errHandle)
			return nil, errHandle
		}
		return &action.ActionNext, nil

	case action.TypeTranscribeRecording:
		if err := h.actionHandleTranscribeRecording(ctx, af); err != nil {
			log.Errorf("Could not handle the recording_to_text action correctly. err: %v", err)
			// we can move on to the next action even it's failed
		}
		return &action.ActionNext, nil

	case action.TypeTranscribeStart:
		if err := h.actionHandleTranscribeStart(ctx, af); err != nil {
			log.Errorf("Could not start the transcribe. err: %v", err)
			// we can move on to the next action even it's failed
		}
		return &action.ActionNext, nil

	case action.TypeVariableSet:
		if err := h.actionHandleVariableSet(ctx, af); err != nil {
			log.Errorf("Could not handle the variable_set. err: %v", err)
		}
		return &action.ActionNext, nil

	case action.TypeWebhookSend:
		if err := h.actionHandleWebhookSend(ctx, af); err != nil {
			log.Errorf("Could not handle the webhook_send. err: %v", err)
		}
		return &action.ActionNext, nil
	}

	return &af.CurrentAction, nil
}

// verifyActionType verifies the given activeflow's action is valid for the reference type.
// return true if the reference type and action type are valid
func (h *activeflowHandler) verifyActionType(af *activeflow.Activeflow) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":          "verifyActionType",
		"activeflow_id": af.ID,
	})

	// get action media type from the reference type
	mediaType, ok := activeflow.MapActionMediaTypeByReferenceType[af.ReferenceType]
	if !ok {
		log.Errorf("The reference type is not found in the action media map. reference_type: %s", af.ReferenceType)
		return false
	}

	// get required media types for the action type
	requiredMediaTypes, ok := action.MapRequiredMediasByType[af.CurrentAction.Type]
	if !ok {
		log.Errorf("The action type is not found in the required media map. action_type: %s", af.CurrentAction.Type)
		return false
	}

	for _, requiredMediaType := range requiredMediaTypes {
		if requiredMediaType == action.MediaTypeNone {
			// no specific media type is required
			return true
		}

		if requiredMediaType == mediaType {
			// the media type is matched
			return true
		}
	}

	return false
}
