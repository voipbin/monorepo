package actionhandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// GenerateFlowActions generates actions for flow.
func (h *actionHandler) GenerateFlowActions(ctx context.Context, actions []action.Action) ([]action.Action, error) {
	log := logrus.WithField("func", "GenerateFlowActions")

	res := []action.Action{}
	// validate actions
	if err := h.ValidateActions(actions); err != nil {
		log.Errorf("Could not pass the action validation. err: %v", err)
		return nil, err
	}

	// set action id
	for _, a := range actions {
		tmpAction := a
		tmpAction.ID = uuid.Must(uuid.NewV4())
		res = append(res, tmpAction)
	}

	// parse the flow change options
	for i, a := range res {
		switch a.Type {
		case action.TypeConditionCallDigits:
			tmp, err := h.generateFlowActionsConditionDigits(ctx, res, &a)
			if err != nil {
				log.Errorf("Could not parse the branch action. err: %v", err)
				return nil, err
			}
			res[i] = *tmp

		case action.TypeGoto:
			tmp, err := h.generateFlowActionsGoto(ctx, res, &a)
			if err != nil {
				log.Errorf("Could not parse the goto action. err: %v", err)
				return nil, err
			}
			res[i] = *tmp

		case action.TypeBranch:
			tmp, err := h.generateFlowActionsBranch(ctx, res, &a)
			if err != nil {
				log.Errorf("Could not parse the branch action err: %v", err)
				return nil, err
			}
			res[i] = *tmp
		}
	}

	return res, nil
}

// getIndexActionID returns action id of given action index.
func (h *actionHandler) getIndexActionID(actions []action.Action, index int) (uuid.UUID, error) {

	if index > len(actions) {
		return uuid.Nil, fmt.Errorf("out of index")
	}

	a := actions[index]
	return a.ID, nil
}

// generateFlowActionsBranch parse the branch action for generate flow actions
func (h *actionHandler) generateFlowActionsBranch(ctx context.Context, actions []action.Action, act *action.Action) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "generateFlowActionsParseBranch",
	})

	var opt action.OptionBranch
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the option. err: %v", err)
		return nil, err
	}
	if opt.TargetIDs == nil {
		opt.TargetIDs = map[string]uuid.UUID{}
	}

	defaultID, err := h.getIndexActionID(actions, opt.DefaultIndex)
	if err != nil {
		log.Errorf("Could not get action id of the index. err: %v", err)
		return nil, err
	}
	opt.DefaultID = defaultID
	log.Debugf("Parsed default id. default_id: %s", defaultID)

	for dtmf, idx := range opt.TargetIndexes {
		targetID, err := h.getIndexActionID(actions, idx)
		if err != nil {
			log.Errorf("Could not get index target id. err: %v", err)
			return nil, err
		}
		opt.TargetIDs[dtmf] = targetID
	}

	tmp, err := json.Marshal(opt)
	if err != nil {
		log.Errorf("Could not marshal the option")
		return nil, err
	}

	res := *act
	res.Option = tmp
	return &res, nil
}

// generateFlowActionsGoto parse the goto action for generate flow actions
func (h *actionHandler) generateFlowActionsGoto(ctx context.Context, actions []action.Action, act *action.Action) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "generateFlowActionsParseGoto",
	})

	var opt action.OptionGoto
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the option. err: %v", err)
		return nil, err
	}

	targetID, err := h.getIndexActionID(actions, opt.TargetIndex)
	if err != nil {
		log.Errorf("Could not get action id of the index. err: %v", err)
		return nil, err
	}
	opt.TargetID = targetID
	log.Debugf("Parsed target id. target_id: %s", targetID)

	tmp, err := json.Marshal(opt)
	if err != nil {
		log.Errorf("Could not marshal the option")
		return nil, err
	}

	res := *act
	res.Option = tmp
	return &res, nil
}

// generateFlowActionsConditionDigits parse the condition_digits action for generate flow actions
func (h *actionHandler) generateFlowActionsConditionDigits(ctx context.Context, actions []action.Action, act *action.Action) (*action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "generateFlowActionsParseConditionDigits",
	})

	var opt action.OptionConditionCallDigits
	if err := json.Unmarshal(act.Option, &opt); err != nil {
		log.Errorf("Could not unmarshal the option. err: %v", err)
		return nil, err
	}

	falseTargetID, err := h.getIndexActionID(actions, opt.FalseTargetIndex)
	if err != nil {
		log.Errorf("Could not get action id of the index. err: %v", err)
		return nil, err
	}
	opt.FalseTargetID = falseTargetID
	log.Debugf("Parsed false target id. false_target_id: %s", falseTargetID)

	tmp, err := json.Marshal(opt)
	if err != nil {
		log.Errorf("Could not marshal the option")
		return nil, err
	}

	res := *act
	res.Option = tmp
	return &res, nil
}
