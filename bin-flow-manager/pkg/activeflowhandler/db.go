package activeflowhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/stack"
	"monorepo/bin-flow-manager/pkg/dbhandler"
)

// Create creates a new activeflow
func (h *activeflowHandler) Create(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	referenceType activeflow.ReferenceType,
	referenceID uuid.UUID,
	referenceActiveflowID uuid.UUID,
	flowID uuid.UUID,
) (*activeflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Create",
		"id":             id,
		"customer_id":    customerID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"flow_id":        flowID,
	})

	// check id is valid
	if id == uuid.Nil {
		id = h.utilHandler.UUIDCreate()
		log.Infof("The id is not given. Created a new id. id: %s", id)
		log = log.WithField("id", id)
	}

	actions, err := h.actionGetsFromFlow(ctx, flowID, customerID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get actions from the flow. flow_id: %s", flowID)
	}
	stackMap := h.stackmapHandler.Create(actions)

	// create activeflow
	tmp := &activeflow.Activeflow{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		Status: activeflow.StatusRunning,
		FlowID: flowID,

		ReferenceType:         referenceType,
		ReferenceID:           referenceID,
		ReferenceActiveflowID: referenceActiveflowID,

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
		return nil, errors.Wrapf(err, "could not create the active flow. activeflow_id: %s", id)
	}

	// get created activeflow
	res, err := h.db.ActiveflowGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get created active flow. activeflow_id: %s", id)
	}

	// variable create
	v, err := h.variableCreate(ctx, res)
	if err != nil {
		// we could not set the variable. but write the log only.
		log.Errorf("Could not set the variable. err: %v", err)
	}
	log.WithField("variable", v).Debugf("Created a new variable. variable_id: %s", v.ID)

	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, activeflow.EventTypeActiveflowCreated, res)

	return res, nil
}

func (h *activeflowHandler) update(ctx context.Context, af *activeflow.Activeflow) error {
	if errUpdate := h.db.ActiveflowUpdate(ctx, af); errUpdate != nil {
		return errors.Wrapf(errUpdate, "could not update the active flow. activeflow_id: %s", af.ID)
	}
	return nil
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
	targetStackID, targetAction, err := h.stackmapHandler.GetAction(af.StackMap, af.CurrentStackID, actionID, false)
	if err != nil {
		log.Errorf("Could not find forward action in the stacks. err: %v", err)
		return fmt.Errorf("forward action not found")
	}

	// update active flow
	af.ForwardStackID = targetStackID
	af.ForwardActionID = targetAction.ID
	log.Debugf("Updating activeflow's foward action. forward_stack_id: %s, forward_action_id: %s", targetStackID, targetAction.ID)
	if errUpdate := h.update(ctx, af); errUpdate != nil {
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

	// get af
	af, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get activeflow. activeflow_id: %s", id)
	}

	// update active flow
	af.ExecutedActions = append(af.ExecutedActions, af.CurrentAction)
	af.CurrentStackID = stackID
	af.CurrentAction = *act
	af.ForwardStackID = stack.IDEmpty
	af.ForwardActionID = action.IDEmpty
	af.ExecuteCount++

	if errUpdate := h.update(ctx, af); errUpdate != nil {
		return nil, errors.Wrapf(errUpdate, "could not update the active flow. activeflow_id: %s", id)
	}

	// get updated activeflow
	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated activeflow. activeflow_id: %s", id)
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, activeflow.EventTypeActiveflowUpdated, res)

	return res, err
}

// updateNextAction updates the next action to the current action.
// It sets next action to current action.
func (h *activeflowHandler) updateNextAction(ctx context.Context, activeflowID uuid.UUID, caID uuid.UUID) (*activeflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                      "updateNextAction",
		"activeflow_id":             activeflowID,
		"request_current_action_id": caID,
	})
	log.Debug("Getting next action.")

	// get activeflow with lock
	af, err := h.GetWithLock(ctx, activeflowID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get activeflow. activeflow_id: %s", activeflowID)
	}
	defer func() {
		_ = h.ReleaseLock(ctx, activeflowID)
	}()

	// check execute count.
	if af.ExecuteCount > maxActiveFlowExecuteCount {
		return nil, fmt.Errorf("exceed maximum action execution count. execute_count: %d", af.ExecuteCount)
	}

	if af.Status == activeflow.StatusEnded {
		return nil, fmt.Errorf("the activeflow ended. status: %s", af.Status)
	}

	if af.CurrentAction.ID != action.IDEmpty && af.CurrentAction.ID != caID {
		return nil, fmt.Errorf("current action does not match. current_action_id: %s, target_current_action_id: %s", af.CurrentAction.ID, caID)
	}

	// get next action
	var resStackID uuid.UUID
	var resAct *action.Action
	if af.ForwardStackID != stack.IDEmpty && af.ForwardActionID != action.IDEmpty {
		log.Debugf("The forward action ID exist. forward_stack_id: %s, forward_action_id: %s", af.ForwardStackID, af.ForwardActionID)
		resStackID, resAct, err = h.stackmapHandler.GetAction(af.StackMap, af.ForwardStackID, af.ForwardActionID, true)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get action. activeflow_id: %s, forward_stack_id: %s, forward_action_id: %s", activeflowID, af.ForwardStackID, af.ForwardActionID)
		}
	} else {
		log.Debugf("The forward action ID does not exist. current_stack_id: %s, current_action_id: %s", af.CurrentStackID, &af.CurrentAction.ID)
		resStackID, resAct = h.stackmapHandler.GetNextAction(af.StackMap, af.CurrentStackID, &af.CurrentAction, true)
	}
	log.Debugf("Found next action. stack_id: %s, action_id: %s, action_type: %s", resStackID, resAct.ID, resAct.Type)

	// substitute the option variables.
	v, err := h.variableHandler.Get(ctx, activeflowID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get variables. activeflow_id: %s", activeflowID)
	}
	h.variableHandler.SubstituteOption(ctx, resAct.Option, v)

	// update current action in activeflow
	res, err := h.updateCurrentAction(ctx, activeflowID, resStackID, resAct)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update the current action. activeflow_id: %s", activeflowID)
	}
	log.WithField("action", res.CurrentAction).Debugf("Updated current action. action_type: %s", res.CurrentAction.Type)

	return res, nil
}

// Delete deletes activeflow
func (h *activeflowHandler) Delete(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Delete",
		"activeflow_id": id,
	})

	// get activeflow
	a, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get activeflow. activeflow_id: %s", id)
	}

	// check the activeflow has been
	if a.TMDelete != dbhandler.DefaultTimeStamp {
		// already deleted
		return a, nil
	}

	if a.Status != activeflow.StatusEnded {
		log.Debugf("The activeflow is not ended. Stopping the activeflow. activeflow_id: %s, status: %s", a.ID, a.Status)
		tmp, err := h.Stop(ctx, id)
		if err != nil {
			return nil, errors.Wrapf(err, "could not stop the activeflow. activeflow_id: %s", id)
		}
		log.Debugf("Stopped activeflow. activeflow_id: %s", tmp.ID)
	}

	res, err := h.delete(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not delete the activeflow. activeflow_id: %s", id)
	}

	return res, nil
}

// delete deletes activeflow
func (h *activeflowHandler) delete(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {

	if errDelete := h.db.ActiveflowDelete(ctx, id); errDelete != nil {
		return nil, errors.Wrapf(errDelete, "could not delete activeflow. activeflow_id: %s", id)
	}

	// get deleted activeflow
	res, err := h.db.ActiveflowGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get deleted activeflow. activeflow_id: %s", id)
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, activeflow.EventTypeActiveflowDeleted, res)

	return res, nil
}

// Get returns activeflow
func (h *activeflowHandler) Get(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {

	res, err := h.db.ActiveflowGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get activeflow. activeflow_id: %s", id)
	}

	return res, nil
}

// GetWithLock returns activeflow
func (h *activeflowHandler) GetWithLock(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {

	res, err := h.db.ActiveflowGetWithLock(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get activeflow. activeflow_id: %s", id)
	}

	return res, nil
}

// ReleaseLock releases activeflow lock
func (h *activeflowHandler) ReleaseLock(ctx context.Context, id uuid.UUID) error {
	return h.db.ActiveflowReleaseLock(ctx, id)
}

// Gets returns list of activeflows
func (h *activeflowHandler) Gets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*activeflow.Activeflow, error) {

	res, err := h.db.ActiveflowGets(ctx, token, size, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get activeflows. token: %s, size: %d, filters: %v", token, size, filters)
	}

	return res, nil
}
