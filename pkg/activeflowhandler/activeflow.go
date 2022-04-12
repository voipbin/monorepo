package activeflowhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

// Create creates a active flow
func (h *activeflowHandler) Create(ctx context.Context, referenceType activeflow.ReferenceType, referenceID, flowID uuid.UUID) (*activeflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Create",
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

	// create activeflow
	id := uuid.Must(uuid.NewV4())
	curTime := dbhandler.GetCurTime()
	tmpAF := &activeflow.Activeflow{
		ID: id,

		CustomerID: f.CustomerID,
		FlowID:     flowID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		CurrentAction: action.Action{
			ID: action.IDStart,
		},
		ExecuteCount:    0,
		ForwardActionID: action.IDEmpty,

		Actions:         f.Actions,
		ExecutedActions: []action.Action{},

		TMCreate: curTime,
		TMUpdate: curTime,
		TMDelete: dbhandler.DefaultTimeStamp,
	}
	if err := h.db.ActiveflowCreate(ctx, tmpAF); err != nil {
		log.Errorf("Could not create the active flow. err: %v", err)
		return nil, err
	}

	// get created active flow
	af, err := h.db.ActiveflowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created active flow. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, af.CustomerID, activeflow.EventTypeActiveFlowCreated, af)

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

	// check the action ID exists in the actions.
	found := false
	for _, a := range af.Actions {
		if a.ID == actionID {
			found = true
			break
		}
	}
	if !found {
		log.Errorf("Could not find foward action id in the actions.")
		return fmt.Errorf("foward action id not found")
	}

	// update active flow
	af.ForwardActionID = actionID
	if err := h.db.ActiveflowUpdate(ctx, af); err != nil {
		log.Errorf("Could not update the active flow.")
		return err
	}

	// send action next
	if forwardNow {
		switch af.ReferenceType {
		case activeflow.ReferenceTypeCall:
			if err := h.reqHandler.CMV1CallActionNext(ctx, af.ReferenceID, true); err != nil {
				log.Errorf("Could not send action next request. err: %v", err)
				return err
			}
		default:
			log.Errorf("Unsupported reference type for forward now. reference_type: %s", af.ReferenceType)
		}
	}

	return nil
}

// GetNextAction returns next action from the active-flow
// It sets next action to current action.
func (h *activeflowHandler) GetNextAction(ctx context.Context, id uuid.UUID, caID uuid.UUID) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "GetNextAction",
		"id":                id,
		"current_action_id": caID,
	})

	// get next action from the active
	nextAction, err := h.getNextAction(ctx, id, caID)
	if err != nil {
		log.Errorf("Could not get next action. Deleting activeflow. err: %v", err)
		_, _ = h.Delete(ctx, id)
		return nil, err
	}
	log.WithField("action", nextAction).Debug("Found next action.")

	// execute the active action
	res, err := h.executeAction(ctx, id, nextAction)
	if err != nil {
		log.Errorf("Could not execute the active action. Deleting activeflow. err: %v", err)
		_, _ = h.Delete(ctx, id)
		return nil, err
	}

	return res, nil
}

// updateCurrentAction updates the current action in active-flow.
// returns updated active flow
func (h *activeflowHandler) updateCurrentAction(ctx context.Context, id uuid.UUID, act *action.Action) (*activeflow.Activeflow, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "activeFlowUpdateCurrentAction",
			"id":        id,
			"action_id": act,
		},
	)

	// get af
	af, err := h.db.ActiveflowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get active-flow. err: %v", err)
		return nil, err
	}

	// update active flow
	af.ExecutedActions = append(af.ExecutedActions, af.CurrentAction)
	af.CurrentAction = *act
	af.ForwardActionID = action.IDEmpty
	af.TMUpdate = dbhandler.GetCurTime()
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
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, activeflow.EventTypeActiveFlowUpdated, res)

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
	h.notifyHandler.PublishWebhookEvent(ctx, af.CustomerID, activeflow.EventTypeActiveFlowDeleted, af)

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
