package flowhandler

import (
	"context"
	"encoding/json"

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
	userID uint64,
	flowType flow.Type,
	name string,
	detail string,
	persist bool,
	webhookURI string,
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
		ID:     id,
		UserID: userID,
		Type:   flowType,

		Name:   name,
		Detail: detail,

		Persist:    persist,
		WebhookURI: webhookURI,

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

// FlowGetsByUserID returns list of flows
func (h *flowHandler) FlowGetsByUserID(ctx context.Context, userID uint64, token string, limit uint64) ([]*flow.Flow, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "FlowGetsByUserID",
			"user_id": userID,
			"token":   token,
			"limit":   limit,
		})
	log.Debug("Getting flows.")

	flows, err := h.db.FlowGetsByUserID(ctx, userID, token, limit)
	if err != nil {
		log.Errorf("Could not get flows. err: %v", err)
		return nil, err
	}

	return flows, nil
}

// FlowGetsByUserIDAndType returns list of flows
func (h *flowHandler) FlowGetsByUserIDAndType(ctx context.Context, userID uint64, flowType flow.Type, token string, limit uint64) ([]*flow.Flow, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "FlowGetsByUserIDAndType",
			"user_id": userID,
			"token":   token,
			"limit":   limit,
		})
	log.Debug("Getting flows.")

	flows, err := h.db.FlowGetsByUserIDAndType(ctx, userID, flowType, token, limit)
	if err != nil {
		log.Errorf("Could not get flows. err: %v", err)
		return nil, err
	}

	return flows, nil
}

// FlowUpdate updates the flow info and return the updated flow
func (h *flowHandler) FlowUpdate(ctx context.Context, f *flow.Flow) (*flow.Flow, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "FlowUpdate",
			"flow_id": f.ID,
		})
	log.WithField("flow", f).Debug("Update flow request.")

	// generates the actions
	actions, err := h.generateFlowActions(ctx, f.Actions)
	if err != nil {
		log.Errorf("Could not generate the flow actions. err: %v", err)
		return nil, err
	}
	f.Actions = actions
	log.WithField("flow", f).Debug("Updating the flow.")

	if err := h.db.FlowUpdate(ctx, f); err != nil {
		log.Errorf("Could not update the flow info. err: %v", err)
		return nil, err
	}

	res, err := h.db.FlowGet(ctx, f.ID)
	if err != nil {
		log.Errorf("Could not get updated flow. err: %v", err)
		return nil, err
	}

	return res, nil
}

// FlowDelete delets the given flow
// And it also removes the related flow_id from the number-manager
func (h *flowHandler) FlowDelete(ctx context.Context, id uuid.UUID) error {
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
		return err
	}

	// send related flow-id clean up request to the number-manager
	if err := h.reqHandler.NMV1NumberFlowDelete(ctx, id); err != nil {
		log.Errorf("Could not clean up the flow_id from the number-manager. err: %v", err)
		// we don't return the err here, because the flow has been removed already.
		// and the numbers which have the removed flow-id are OK too, because
		// when the call-manager request the flow, that request will be failed too.
	}

	return nil
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
