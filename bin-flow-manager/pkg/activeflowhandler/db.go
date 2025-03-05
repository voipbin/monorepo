package activeflowhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/stack"
)

// Create creates a new activeflow
func (h *activeflowHandler) Create(ctx context.Context, activeflowID uuid.UUID, referenceType activeflow.ReferenceType, referenceID uuid.UUID, flowID uuid.UUID) (*activeflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Create",
		"id":             activeflowID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"flow_id":        flowID,
	})

	// get flow
	f, err := h.db.FlowGet(ctx, flowID)
	if err != nil {
		log.Errorf("Could not get the flow. err: %v", err)
		return nil, err
	}

	// check id is valid
	if activeflowID == uuid.Nil {
		activeflowID = h.utilHandler.UUIDCreate()
		log.Infof("The id is not given. Created a new id. id: %s", activeflowID)
		log = log.WithField("id", activeflowID)
	}

	stackMap := h.stackHandler.InitStackMap(ctx, f.Actions)

	// create activeflow
	tmp := &activeflow.Activeflow{
		ID:         activeflowID,
		CustomerID: f.CustomerID,

		Status: activeflow.StatusRunning,
		FlowID: flowID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		StackMap: stackMap,

		CurrentStackID: stack.IDMain,
		CurrentAction: action.Action{
			ID: action.IDStart,
		},

		ForwardStackID:  stack.IDEmpty,
		ForwardActionID: action.IDEmpty,

		ExecuteCount:    0,
		ExecutedActions: []action.Action{},
	}
	if err := h.db.ActiveflowCreate(ctx, tmp); err != nil {
		log.Errorf("Could not create the active flow. err: %v", err)
		return nil, err
	}

	// create a new activeflow
	v, err := h.variableHandler.Create(ctx, activeflowID, map[string]string{})
	if err != nil {
		log.Errorf("Could not create variable. err: %v", err)
		return nil, err
	}
	log.WithField("variable", v).Debugf("Created a new variable. variable_id: %s", v.ID)

	// get created activeflow
	res, err := h.db.ActiveflowGet(ctx, activeflowID)
	if err != nil {
		log.Errorf("Could not get created active flow. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, activeflow.EventTypeActiveflowCreated, res)

	return res, nil
}

// SetForwardActionID sets the forward action id of the call.
func (h *activeflowHandler) SetForwardActionID(ctx context.Context, id uuid.UUID, actionID uuid.UUID, forwardNow bool) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "SetForwardActionID",
		"activeflow_id":     id,
		"forward_action_id": actionID,
		"forward_now":       forwardNow,
	})

	// get active flow
	af, err := h.GetWithLock(ctx, id)
	if err != nil {
		log.Errorf("Could not get active flow. err: %v", err)
		return err
	}
	defer func() {
		_ = h.ReleaseLock(ctx, id)
	}()

	// get target action
	targetStackID, targetAction, err := h.stackHandler.GetAction(ctx, af.StackMap, af.CurrentStackID, actionID, false)
	if err != nil {
		log.Errorf("Could not find forward action in the stacks. err: %v", err)
		return fmt.Errorf("forward action not found")
	}

	// update active flow
	af.ForwardStackID = targetStackID
	af.ForwardActionID = targetAction.ID
	log.Debugf("Updating activeflow's foward action. forward_stack_id: %s, forward_action_id: %s", targetStackID, targetAction.ID)
	if errUpdate := h.db.ActiveflowUpdate(ctx, af); errUpdate != nil {
		log.Errorf("Could not update the active flow. err :%v", errUpdate)
		return errUpdate
	}

	go func() {
		// send action next
		if forwardNow {
			switch af.ReferenceType {
			case activeflow.ReferenceTypeCall:
				if err := h.reqHandler.CallV1CallActionNext(ctx, af.ReferenceID, true); err != nil {
					log.Errorf("Could not send action next request. err: %v", err)
					return
				}
			default:
				log.Errorf("Unsupported reference type for forward now. reference_type: %s", af.ReferenceType)
			}
		}
	}()

	return nil
}

// updateCurrentAction updates the current action in active-flow.
// returns updated active flow
func (h *activeflowHandler) updateCurrentAction(ctx context.Context, id uuid.UUID, stackID uuid.UUID, act *action.Action) (*activeflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "updateCurrentAction",
		"id":        id,
		"stack_id":  stackID,
		"action_id": act,
	})

	// get af
	af, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get active-flow. err: %v", err)
		return nil, err
	}

	// update active flow
	af.ExecutedActions = append(af.ExecutedActions, af.CurrentAction)
	af.CurrentStackID = stackID
	af.CurrentAction = *act
	af.ForwardStackID = stack.IDEmpty
	af.ForwardActionID = action.IDEmpty
	af.ExecuteCount++

	if errUpdate := h.db.ActiveflowUpdate(ctx, af); errUpdate != nil {
		log.Errorf("Could not update the active-flow's current action. err: %v", errUpdate)
		return nil, errUpdate
	}

	// get updated activeflow
	res, err := h.db.ActiveflowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated active flow. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, activeflow.EventTypeActiveflowUpdated, res)

	return res, err
}

// dbDelete deletes activeflow
func (h *activeflowHandler) dbDelete(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "dbDelete",
		"activeflow_id": id,
	})

	if errDelete := h.db.ActiveflowDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete activeflow. err: %v", errDelete)
		return nil, errDelete
	}

	// get deleted activeflow
	res, err := h.db.ActiveflowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get activeflow. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, activeflow.EventTypeActiveflowDeleted, res)

	return res, nil
}

// Get returns activeflow
func (h *activeflowHandler) Get(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Get",
		"activeflow_id": id,
	})

	res, err := h.db.ActiveflowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get activeflow. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetWithLock returns activeflow
func (h *activeflowHandler) GetWithLock(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "GetWithLock",
		"activeflow_id": id,
	})

	res, err := h.db.ActiveflowGetWithLock(ctx, id)
	if err != nil {
		log.Errorf("Could not get activeflow. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ReleaseLock releases activeflow lock
func (h *activeflowHandler) ReleaseLock(ctx context.Context, id uuid.UUID) error {
	return h.db.ActiveflowReleaseLock(ctx, id)
}

// Gets returns list of activeflows
func (h *activeflowHandler) Gets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*activeflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Gets",
		"token":   token,
		"limit":   size,
		"filters": filters,
	})
	log.Debug("Getting activeflows.")

	res, err := h.db.ActiveflowGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get activeflows. err: %v", err)
		return nil, err
	}

	return res, nil
}

// PushStack pushes the given action to the stack with a new stack
func (h *activeflowHandler) PushStack(ctx context.Context, af *activeflow.Activeflow, stackID uuid.UUID, actions []action.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "PushStack",
		"activeflow_id": af.ID,
		"stack_id":      stackID,
		"actions":       actions,
	})

	resStackID, resAction, err := h.stackHandler.Push(ctx, af.StackMap, stackID, actions, af.CurrentStackID, af.CurrentAction.ID)
	if err != nil {
		log.Errorf("Could not push the actions. err: %s", err)
		return err
	}

	// update forward actions
	af.ForwardStackID = resStackID
	af.ForwardActionID = resAction.ID

	// update activeflow
	if err := h.db.ActiveflowUpdate(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// PushActions pushes the given actions in a new stack.
// pushed new stack will be executed in a next action request.
func (h *activeflowHandler) PushActions(ctx context.Context, id uuid.UUID, actions []action.Action) (*activeflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "PushActions",
		"activeflow_id": id,
		"actions":       actions,
	})

	af, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get activeflow info. err: %v", err)
		return nil, err
	}

	flowActions, err := h.actionHandler.GenerateFlowActions(ctx, actions)
	if err != nil {
		log.Errorf("Could not generate the flow actions. err: %v", err)
		return nil, errors.Wrap(err, "could not generate the flow actions")
	}

	if errPush := h.PushStack(ctx, af, uuid.Nil, flowActions); errPush != nil {
		log.Errorf("Could not push the new stack for flow actions. err: %v", errPush)
		return nil, errors.Wrap(err, "could not push the new stack for flow actions")
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated activeflow info. err: %v", err)
		return nil, errors.Wrap(err, "could not get updated activeflow info")
	}

	return res, nil
}
