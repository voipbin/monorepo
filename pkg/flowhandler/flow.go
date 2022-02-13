package flowhandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

// FlowGet returns flow
func (h *flowHandler) FlowGet(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {
	log := logrus.WithField("func", "FlowGet")
	resFlow, err := h.db.FlowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get flow. err: %v", err)
		return nil, err
	}

	return resFlow, nil
}

// FlowCreate creates a new flow
func (h *flowHandler) FlowCreate(
	ctx context.Context,
	customerID uuid.UUID,
	flowType flow.Type,
	name string,
	detail string,
	persist bool,
	actions []action.Action,
) (*flow.Flow, error) {
	log := logrus.WithField("func", "FlowCreate")

	// generates the actions
	a, err := h.generateFlowActions(ctx, actions)
	if err != nil {
		log.Errorf("Could not generate the flow actions. err: %v", err)
		return nil, err
	}

	id := uuid.Must(uuid.NewV4())
	f := &flow.Flow{
		ID:         id,
		CustomerID: customerID,
		Type:       flowType,

		Name:   name,
		Detail: detail,

		Persist: persist,

		Actions: a,

		TMCreate: dbhandler.GetCurTime(),
		TMUpdate: dbhandler.DefaultTimeStamp,
		TMDelete: dbhandler.DefaultTimeStamp,
	}
	log.WithField("flow", f).Debug("Creating a new flow.")

	switch {
	case f.Persist:
		if err := h.db.FlowCreate(ctx, f); err != nil {
			log.Errorf("Could not create the flow in the database. err: %v", err)
			return nil, err
		}

	default:
		if err := h.db.FlowSetToCache(ctx, f); err != nil {
			log.Errorf("Could not create the flow in the cache. err: %v", err)
			return nil, err
		}
	}

	res, err := h.FlowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created flow. err: %v", err)
		return nil, err
	}

	return res, nil
}

// FlowGets returns list of flows
func (h *flowHandler) FlowGets(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*flow.Flow, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "FlowGets",
			"customer_id": customerID,
			"token":       token,
			"limit":       limit,
		})
	log.Debug("Getting flows.")

	flows, err := h.db.FlowGets(ctx, customerID, token, limit)
	if err != nil {
		log.Errorf("Could not get flows. err: %v", err)
		return nil, err
	}

	return flows, nil
}

// FlowGetsByType returns list of flows
func (h *flowHandler) FlowGetsByType(ctx context.Context, customerID uuid.UUID, flowType flow.Type, token string, limit uint64) ([]*flow.Flow, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "FlowGetsByType",
			"customer_id": customerID,
			"token":       token,
			"limit":       limit,
		})
	log.Debug("Getting flows.")

	flows, err := h.db.FlowGetsByType(ctx, customerID, flowType, token, limit)
	if err != nil {
		log.Errorf("Could not get flows. err: %v", err)
		return nil, err
	}

	return flows, nil
}

// FlowUpdate updates the flow info and return the updated flow
func (h *flowHandler) FlowUpdate(ctx context.Context, id uuid.UUID, name, detail string, actions []action.Action) (*flow.Flow, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "FlowUpdate",
			"flow_id": id,
		})
	log.WithFields(
		logrus.Fields{
			"name":    name,
			"detail":  detail,
			"actions": actions,
		},
	).Debug("Updating the flow.")

	// generates the tmpActions
	tmpActions, err := h.generateFlowActions(ctx, actions)
	if err != nil {
		log.Errorf("Could not generate the flow actions. err: %v", err)
		return nil, err
	}
	log.WithField("new_actions", tmpActions).Debug("Created the new actions tmp.")

	if err := h.db.FlowUpdate(ctx, id, name, detail, tmpActions); err != nil {
		log.Errorf("Could not update the flow info. err: %v", err)
		return nil, err
	}

	res, err := h.db.FlowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated flow. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, flow.EventTypeFlowUpdated, res)

	return res, nil
}

// FlowDelete deletes the flow
// And it also removes the related flow_id from the number-manager
func (h *flowHandler) FlowDelete(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "FlowDelete",
			"flow_id": id,
		},
	)
	log.Debug("Deleting the flow.")

	err := h.db.FlowDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the flow. err: %v", err)
		return nil, err
	}

	// send related flow-id clean up request to the number-manager
	if err := h.reqHandler.NMV1NumberFlowDelete(ctx, id); err != nil {
		log.Errorf("Could not clean up the flow_id from the number-manager. err: %v", err)
		// we don't return the err here, because the flow has been removed already.
		// and the numbers which have the removed flow-id are OK too, because
		// when the call-manager request the flow, that request will be failed too.
	}

	res, err := h.db.FlowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted flow. err: %v", err)
		return nil, fmt.Errorf("could not get deleted flow")
	}
	h.notifyHandler.PublishEvent(ctx, flow.EventTypeFlowDeleted, res)

	return res, nil
}

// generateFlowActions generates actions for flow.
func (h *flowHandler) generateFlowActions(ctx context.Context, actions []action.Action) ([]action.Action, error) {
	log := logrus.WithField("func", "generateFlowActions")

	res := []action.Action{}
	// validate actions
	if err := h.ValidateActions(actions); err != nil {
		log.Errorf("Could not pass the action validation. err: %v", err)
		return nil, err
	}

	// set action id
	for _, a := range actions {
		// a.ID = uuid.Must(uuid.NewV4())
		tmpAction := a
		tmpAction.ID = uuid.Must(uuid.NewV4())
		res = append(res, tmpAction)
	}

	// parse the flow change options
	for i, a := range res {
		// goto type
		if a.Type == action.TypeGoto {
			var option action.OptionGoto
			if err := json.Unmarshal(a.Option, &option); err != nil {
				log.Errorf("Could not unmarshal the option. err: %v", err)
				return nil, err
			}

			option.TargetID = res[option.TargetIndex].ID
			tmp, err := json.Marshal(option)
			if err != nil {
				log.Errorf("Could not marshal the option")
				return nil, err
			}

			a.Option = tmp
			res[i] = a
		}
	}

	return res, nil
}

// generateFlowForAgentCall creates a flow for the agent call action.
func (h *flowHandler) generateFlowForAgentCall(ctx context.Context, customerID, confbridgeID uuid.UUID) (*flow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "generateFlowForAgentCall",
		"confbridge_id": confbridgeID,
	})

	opt, err := json.Marshal(action.OptionConfbridgeJoin{
		ConfbridgeID: confbridgeID,
	})
	if err != nil {
		log.Errorf("Could not marshal the action. err: %v", err)
		return nil, err
	}

	// create actions
	actions := []action.Action{
		{
			Type:   action.TypeConfbridgeJoin,
			Option: opt,
		},
	}

	// create a flow for agent dial.
	res, err := h.FlowCreate(ctx, customerID, flow.TypeFlow, "automatically generated for the agent call", "", false, actions)
	if err != nil {
		log.Errorf("Could not create the flow. err: %v", err)
		return nil, err
	}
	log.WithField("flow", res).Debug("Created a flow.")

	return res, nil
}
