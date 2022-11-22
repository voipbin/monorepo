package activeflowhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/stack"
)

// Create creates a new activeflow
func (h *activeflowHandler) Create(ctx context.Context, activeflowID uuid.UUID, referenceType activeflow.ReferenceType, referenceID, flowID uuid.UUID) (*activeflow.Activeflow, error) {
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
		activeflowID = uuid.Must(uuid.NewV4())
		log.Infof("The id is not valid. Created a new id. id: %s", activeflowID)
		log = log.WithField("id", activeflowID)
	}

	// create activeflow
	// curTime := h.util.GetCurTime()
	tmpAF := &activeflow.Activeflow{
		ID: activeflowID,

		CustomerID: f.CustomerID,
		FlowID:     flowID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		StackMap: map[uuid.UUID]*stack.Stack{
			stack.IDMain: {
				ID:             stack.IDMain,
				Actions:        f.Actions,
				ReturnStackID:  stack.IDEmpty,
				ReturnActionID: action.IDEmpty,
			},
		},

		CurrentStackID: stack.IDMain,
		CurrentAction: action.Action{
			ID: action.IDStart,
		},

		ForwardStackID:  stack.IDEmpty,
		ForwardActionID: action.IDEmpty,

		ExecuteCount:    0,
		ExecutedActions: []action.Action{},

		// TMCreate: curTime,
		// TMUpdate: curTime,
		// TMDelete: dbhandler.DefaultTimeStamp,
	}
	if err := h.db.ActiveflowCreate(ctx, tmpAF); err != nil {
		log.Errorf("Could not create the active flow. err: %v", err)
		return nil, err
	}

	// create a new v
	v, err := h.variableHandler.Create(ctx, activeflowID, map[string]string{})
	if err != nil {
		log.Errorf("Could not create variable. err: %v", err)
		return nil, err
	}
	log.WithField("variable", v).Debugf("Created a new variable. variable_id: %s", v.ID)

	// get created active flow
	af, err := h.db.ActiveflowGet(ctx, activeflowID)
	if err != nil {
		log.Errorf("Could not get created active flow. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, af.CustomerID, activeflow.EventTypeActiveflowCreated, af)

	return af, nil
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
	af, err := h.db.ActiveflowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get active flow. err: %v", err)
		return err
	}
	log.WithField("active_flow", af).Debug("Found active flow.")

	// get target action
	targetStackID, targetAction, err := h.stackHandler.GetAction(ctx, af.StackMap, af.CurrentStackID, actionID, false)
	if err != nil {
		log.Errorf("Could not find forward action in the stacks.")
		return fmt.Errorf("forward action not found")
	}

	// update active flow
	af.ForwardStackID = targetStackID
	af.ForwardActionID = targetAction.ID
	if err := h.db.ActiveflowUpdate(ctx, af); err != nil {
		log.Errorf("Could not update the active flow.")
		return err
	}

	// send action next
	if forwardNow {
		switch af.ReferenceType {
		case activeflow.ReferenceTypeCall:
			if err := h.reqHandler.CallV1CallActionNext(ctx, af.ReferenceID, true); err != nil {
				log.Errorf("Could not send action next request. err: %v", err)
				return err
			}
		default:
			log.Errorf("Unsupported reference type for forward now. reference_type: %s", af.ReferenceType)
		}
	}

	return nil
}

// updateCurrentAction updates the current action in active-flow.
// returns updated active flow
func (h *activeflowHandler) updateCurrentAction(ctx context.Context, id uuid.UUID, stackID uuid.UUID, act *action.Action) (*activeflow.Activeflow, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "updateCurrentAction",
			"id":        id,
			"stack_id":  stackID,
			"action_id": act,
		},
	)

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

	if err := h.db.ActiveflowUpdate(ctx, af); err != nil {
		log.Errorf("Could not update the active-flow's current action. err: %v", err)
		return nil, err
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

// Delete deletes activeflow
func (h *activeflowHandler) Delete(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Delete",
		"activeflow_id": id,
	})

	if errDelete := h.db.ActiveflowDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete activeflow. err: %v", errDelete)
		return nil, errDelete
	}

	// get deleted activeflow
	af, err := h.db.ActiveflowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get activeflow. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, af.CustomerID, activeflow.EventTypeActiveflowDeleted, af)

	return af, nil
}

// Get returns activeflow
func (h *activeflowHandler) Get(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {
	log := logrus.WithField("func", "Get")
	resFlow, err := h.db.ActiveflowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get activeflow. err: %v", err)
		return nil, err
	}

	return resFlow, nil
}

// FlowGets returns list of activeflows
func (h *activeflowHandler) GetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*activeflow.Activeflow, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "GetsByCustomerID",
			"customer_id": customerID,
			"token":       token,
			"limit":       limit,
		})
	log.Debug("Getting activeflows.")

	res, err := h.db.ActiveflowGetsByCustomerID(ctx, customerID, token, limit)
	if err != nil {
		log.Errorf("Could not get flows. err: %v", err)
		return nil, err
	}

	return res, nil
}
