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

	stackMap := h.stackmapHandler.Create(f.Actions)

	// create activeflow
	tmp := &activeflow.Activeflow{
		Identity: commonidentity.Identity{
			ID:         activeflowID,
			CustomerID: f.CustomerID,
		},

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
		log.Debugf("The activeflow is not ended. Stopping the activeflow. activeflow_id: %s, status: %s", a.Identity.ID, a.Status)
		tmp, err := h.Stop(ctx, id)
		if err != nil {
			return nil, errors.Wrapf(err, "could not stop the activeflow. activeflow_id: %s", id)
		}
		log.Debugf("Stopped activeflow. activeflow_id: %s", tmp.Identity.ID)
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
