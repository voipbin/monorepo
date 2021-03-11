package flowhandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/requesthandler/models/cmcall"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/requesthandler/models/cmconference"
)

// FlowCreate creates a flow
func (h *flowHandler) ActiveFlowCreate(ctx context.Context, callID, flowID uuid.UUID) (*activeflow.ActiveFlow, error) {

	// get flow
	f, err := h.db.FlowGet(ctx, flowID)
	if err != nil {
		logrus.Errorf("Could not get the flow. err: %v", err)
		return nil, err
	}

	// create activeflow
	curTime := getCurTime()
	tmpAF := &activeflow.ActiveFlow{
		CallID:     callID,
		FlowID:     flowID,
		UserID:     f.UserID,
		WebhookURI: f.WebhookURI,

		CurrentAction: action.Action{
			ID: action.IDStart,
		},

		Actions: f.Actions,

		TMCreate: curTime,
		TMUpdate: curTime,
	}
	if err := h.db.ActiveFlowCreate(ctx, tmpAF); err != nil {
		return nil, err
	}

	// get created active flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		return nil, err
	}

	return af, nil
}

// ActiveFlowNextActionGet returns next action from the active-flow
// It sets next action to current action.
func (h *flowHandler) ActiveFlowNextActionGet(ctx context.Context, callID uuid.UUID, caID uuid.UUID) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"call":              callID,
		"current_action_id": caID,
	})

	// get next action from the active
	nextAction, err := h.activeFlowGetNextAction(ctx, callID, caID)
	if err != nil {
		log.Errorf("Could not get next action. err: %v", err)
		return nil, err
	}

	switch nextAction.Type {
	case action.TypePatch:
		// handle the patch
		// add the patched actions to the active-flow
		if err := h.activeFlowHandleActionPatch(ctx, callID, nextAction); err != nil {
			log.Errorf("Could not handle the patch action correctly. err: %v", err)
			return nil, err
		}

		// do activeflow next action get again.
		return h.ActiveFlowNextActionGet(ctx, callID, nextAction.ID)

	case action.TypeConnect:
		if err := h.activeFlowHandleActionConnect(ctx, callID, nextAction); err != nil {
			log.Errorf("Could not handle the connect action correctly. err: %v", err)
			return nil, err
		}

		// do activeflow next action get again.
		return h.ActiveFlowNextActionGet(ctx, callID, nextAction.ID)
	}

	return nextAction, nil
}

// activeFlowUpdateCurrentAction updates the current action in active-flow.
func (h *flowHandler) activeFlowUpdateCurrentAction(ctx context.Context, callID uuid.UUID, action *action.Action) error {
	log := logrus.WithFields(
		logrus.Fields{
			"call_id": callID,
			"action":  action,
		},
	)

	// get af
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get active-flow. err: %v", err)
		return err
	}

	// set current Action
	af.CurrentAction = *action
	af.TMUpdate = getCurTime()

	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active-flow's current action. err: %v", err)
		return err
	}

	return nil
}

// activeFlowGetNextAction returns next action from the active-flow
// It sets next action to current action.
func (h *flowHandler) activeFlowGetNextAction(ctx context.Context, callID uuid.UUID, caID uuid.UUID) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"call":              callID,
		"current_action_id": caID,
	})
	log.Debug("Getting next action.")

	// get active-flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get active-flow. err: %v", err)
		return nil, err
	}

	// check the empty actions and action id is start id or not.
	switch {
	case len(af.Actions) == 0:
		resAction := *(h.CreateActionHangup())

		// update current action in active-flow
		if err := h.activeFlowUpdateCurrentAction(ctx, callID, &resAction); err != nil {
			log.Errorf("Could not update the current action. err: %v", err)
			return nil, fmt.Errorf("could not update the current action. err: %v", err)
		}

		return &resAction, nil

	case af.CurrentAction.ID == action.IDStart:
		resAction := af.Actions[0]

		// update current action in active-flow
		if err := h.activeFlowUpdateCurrentAction(ctx, callID, &resAction); err != nil {
			log.Errorf("Could not update the current action. err: %v", err)
			return nil, fmt.Errorf("could not update the current action. err: %v", err)
		}
		return &resAction, nil
	}
	log = log.WithFields(logrus.Fields{
		"active_flow_current_action_id": af.CurrentAction.ID,
	})

	// compare current action.
	// if the current action does not match with the active-flow's current action,
	// discard it here
	if af.CurrentAction.ID != caID {
		log.Error("The current action does not match.")
		return nil, fmt.Errorf("current action does not match")
	}

	// get current action's index
	idx := 0
	found := false
	for _, act := range af.Actions {
		if act.ID == caID {
			found = true
			break
		}
		idx++
	}

	// check maximum action execution count
	if idx > 100 {
		log.Errorf("Exceed maximum action execution count. idx: %d", idx)
		return nil, fmt.Errorf("exceed maximum action execution count")
	}

	// get nextAction
	var nextAction action.Action
	if found == false || idx >= (len(af.Actions)-1) {
		// check if the no more actions left, return finishID here.
		log.Infof("No more action left. found: %v, idx: %v", found, idx)

		// create finish hangup
		nextAction = *(h.CreateActionHangup())
	} else {
		nextAction = af.Actions[idx+1]
	}

	// update current action in active-flow
	if err := h.activeFlowUpdateCurrentAction(ctx, callID, &nextAction); err != nil {
		log.Errorf("Could not update the current action. err: %v", err)
		return nil, fmt.Errorf("could not update the current action. err: %v", err)
	}

	return &nextAction, nil
}

// activeFlowHandleActionPatch handles action patch with active flow.
// it downloads the actions from the given action(patch) and append it to the active flow.
func (h *flowHandler) activeFlowHandleActionPatch(ctx context.Context, callID uuid.UUID, act *action.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call":   callID,
		"action": act.ID,
	})

	// patch the actions from the remote
	patchedActions, err := h.actionPatchGet(act, callID)
	if err != nil {
		log.Errorf("Could not patch the actions from the remote. err: %v", err)
		return err
	}

	// generate action id
	for _, act := range patchedActions {
		act.ID = uuid.Must(uuid.NewV4())
	}

	// get active flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get active flow. err: %v", err)
		return err
	}

	// append the patched actions to the active flow
	if err := appendActionsAfterID(af, act.ID, patchedActions); err != nil {
		log.Errorf("Could not append new action. err: %v", err)
		return fmt.Errorf("could not append new action. err: %v", err)
	}
	af.TMUpdate = getCurTime()

	// set active flow
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

// activeFlowHandleActionConnect handles action connect with active flow.
func (h *flowHandler) activeFlowHandleActionConnect(ctx context.Context, callID uuid.UUID, act *action.Action) error {
	log := logrus.WithFields(logrus.Fields{
		"call":   callID,
		"action": act.ID,
	})

	// get active-flow
	af, err := h.db.ActiveFlowGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get active-flow. err: %v", err)
		return fmt.Errorf("could not get active-flow. err: %v", err)
	}

	// create conference room for connect
	cf, err := h.reqHandler.CMConferenceCreate(af.UserID, cmconference.TypeConnect, "", "", 86400)
	if err != nil {
		log.Errorf("Could not create conference for connect. err: %v", err)
		return fmt.Errorf("could not create conference for connect. err: %v", err)
	}
	log = log.WithFields(logrus.Fields{
		"conference": cf.ID,
	})
	log.Debug("Created conference for connect.")

	// create a temp flow connect conference join
	optJoin := action.OptionConferenceJoin{
		ConferenceID: cf.ID.String(),
	}
	optString, err := json.Marshal(optJoin)
	if err != nil {
		log.Errorf("Could not marshal the conference join option. err: %v", err)
		return fmt.Errorf("could not marshal the conference join option. err: %v", err)
	}

	tmpCF := &flow.Flow{
		UserID: cf.UserID,
		Actions: []action.Action{
			{
				Type:   action.TypeConferenceJoin,
				Option: optString,
			},
		},
	}

	var optConnect action.OptionConnect
	if err := json.Unmarshal(act.Option, &optConnect); err != nil {
		log.Errorf("Could not unmarshal the connect option. err: %v", err)
		return fmt.Errorf("could not unmarshal the connect option. err: %v", err)
	}

	// create a flow
	connectCF, err := h.FlowCreate(ctx, tmpCF, false)
	if err != nil {
		log.Errorf("Could not create a temporary flow for connect. err: %v", err)
		return fmt.Errorf("could not create a call flow. err: %v", err)
	}

	// create a call for each destination
	successCount := 0
	for _, dest := range optConnect.Destinations {
		source := cmcall.Address{
			Type:   cmcall.AddressType(optConnect.Source.Type),
			Target: optConnect.Source.Target,
			Name:   optConnect.Source.Name,
		}

		destination := cmcall.Address{
			Type:   cmcall.AddressType(dest.Type),
			Target: dest.Target,
			Name:   dest.Name,
		}

		// create a call
		resCall, err := h.reqHandler.CMCallCreate(connectCF.UserID, connectCF.ID, source, destination)
		if err != nil {
			log.Errorf("Could not create a outgoing call for connect. err: %v", err)
			continue
		}

		// add the chained call id if the unchained option is false
		if optConnect.Unchained == false {
			if err := h.reqHandler.CMCallAddChainedCall(callID, resCall.ID); err != nil {
				log.Warnf("Could not add the chained call id. Hangup the call. chained_call_id: %s", resCall.ID)
				h.reqHandler.CMCallHangup(resCall.ID)
				continue
			}
		}

		log.Debugf("Created outgoing call for connect. call: %s", resCall.ID)
		successCount++
	}

	if successCount == 0 {
		log.Errorf("Could not create any successful outgoingcall.")
		return fmt.Errorf("could not create any successful outgoing call")
	}

	// put original call into the created conference
	resAction := action.Action{
		ID:     uuid.Must(uuid.NewV4()),
		Type:   action.TypeConferenceJoin,
		Option: optString,
	}

	// add the created action next to the given action id.
	if err := appendActionsAfterID(af, act.ID, []action.Action{resAction}); err != nil {
		log.Errorf("Could not append new action. err: %v", err)
		return fmt.Errorf("could not append new action. err: %v", err)
	}
	af.TMUpdate = getCurTime()

	// update active flow
	if err := h.db.ActiveFlowSet(ctx, af); err != nil {
		log.Errorf("Could not update the active flow after appended the patched actions. err: %v", err)
		return err
	}

	return nil
}

func appendActionsAfterID(af *activeflow.ActiveFlow, id uuid.UUID, act []action.Action) error {

	var res []action.Action

	// get idx
	idx := -1
	for i, act := range af.Actions {
		if act.ID == id {
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("could not find action index")
	}

	// append
	res = append(res, af.Actions[:idx+1]...)
	res = append(res, act...)
	res = append(res, af.Actions[idx+1:]...)

	af.Actions = res

	return nil
}
