package callhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// digitsReceived handles DTMF Recevied event
func (h *callHandler) digitsReceived(ctx context.Context, cn *channel.Channel, digit string, duration int) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "digitsReceived",
		"channel":  cn,
		"digits":   digit,
		"duration": duration,
	})

	c, err := h.db.CallGetByChannelID(ctx, cn.ID)
	if err != nil {
		log.Infof("The channel is not call type.")
		return nil
	}
	log.WithField("call", c).Debug("Found call info.")

	log = log.WithFields(
		logrus.Fields{
			"call_id": c.ID,
		},
	)

	switch c.Action.Type {
	case fmaction.TypeDigitsReceive:

		digits := fmt.Sprintf("${%s}%s", variableCallDigits, digit)
		variables := map[string]string{
			variableCallDigits: digits,
		}
		if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, c.ActiveFlowID, variables); errSet != nil {
			log.Errorf("Could not set DTMF. err: %v", err)
			return nil
		}

		var option fmaction.OptionDigitsReceive
		if err := json.Unmarshal(c.Action.Option, &option); err != nil {
			log.WithField("action", c.Action).Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse option. action: %v, err: %v", c.Action, err)
		}

		condition, err := h.checkDigitsCondition(ctx, c.ActiveFlowID, &option)
		if err != nil {
			log.Errorf("Could not validate the digits. err: %v", err)
			return nil
		}

		if !condition {
			log.Debug("The digit recieved not finished yet. Waiting next digit.")
			return nil
		}

		if errNext := h.reqHandler.CallV1CallActionNext(ctx, c.ID, false); errNext != nil {
			log.Errorf("Could not get next action. err: %v", errNext)
			_, _ = h.HangingUp(ctx, c.ID, call.HangupReasonNormal)
		}

		return nil

	case fmaction.TypeTalk:
		// overwrite dtmf receive cache.
		// we are setting the dtmf here even it is not dtmf receive action.
		// this is needed, because if the user press the dtmf in the prior of dtmf receive(i.e play action),
		// the user exepects pressed dtmf could be collected in the dtmf received action in next.
		variables := map[string]string{
			variableCallDigits: digit,
		}
		if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, c.ActiveFlowID, variables); errSet != nil {
			log.Errorf("Could not set DTMF. err: %v", err)
		}

		var option fmaction.OptionTalk
		if err := json.Unmarshal(c.Action.Option, &option); err != nil {
			log.WithField("action", c.Action).Errorf("could not parse the option. err: %v", err)
			return fmt.Errorf("could not parse option. action: %v, err: %v", c.Action, err)
		}

		// send next action request
		if option.DigitsHandle == fmaction.OptionTalkDigitsHandleNext {
			log.Debugf("Digits handle is next. Moving to the next. digits_handle: %s", option.DigitsHandle)
			if errNext := h.reqHandler.CallV1CallActionNext(ctx, c.ID, true); errNext != nil {
				log.Errorf("Could not get next action. err: %v", errNext)
				_, _ = h.HangingUp(ctx, c.ID, call.HangupReasonNormal)
			}
		}

		return nil

	default:
		log.WithField("action_type", c.Action.Type).Debug("The current action is not dtmf receive.")

		// overwrite dtmf receive cache.
		// we are setting the dtmf here even it is not dtmf receive action.
		// this is needed, because if the user press the dtmf in the prior of dtmf receive(i.e play action),
		// the user exepects pressed dtmf could be collected in the dtmf received action in next.
		variables := map[string]string{
			variableCallDigits: digit,
		}
		if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, c.ActiveFlowID, variables); errSet != nil {
			log.Errorf("Could not set DTMF. err: %v", err)
		}

		return nil
	}
}

// DigitsGet returns received dtmfs
func (h *callHandler) DigitsGet(ctx context.Context, id uuid.UUID) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "DigitsGet",
		"call_id": id,
	})

	// get call
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return "", err
	}

	vars, err := h.reqHandler.FlowV1VariableGet(ctx, c.ActiveFlowID)
	if err != nil {
		log.Errorf("Could not get variables. err: %v", err)
		return "", err
	}

	// check finish condition
	res := vars.Variables[variableCallDigits]

	return res, nil
}

// DigitsSet sets the dtmfs
func (h *callHandler) DigitsSet(ctx context.Context, id uuid.UUID, digits string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "DigitsSet",
		"call_id": id,
	})

	// get call
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return err
	}

	variables := map[string]string{
		variableCallDigits: digits,
	}
	if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, c.ActiveFlowID, variables); errSet != nil {
		log.Errorf("Could not set DTMF. err: %v", err)
		return errSet
	}

	return nil
}

// checkDigitsCondition checks the received digits with option
func (h *callHandler) checkDigitsCondition(ctx context.Context, variableID uuid.UUID, option *fmaction.OptionDigitsReceive) (bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "checkDigitsCondition",
		"variable_id": variableID,
		"option":      option,
	})

	vars, err := h.reqHandler.FlowV1VariableGet(ctx, variableID)
	if err != nil {
		log.Errorf("Could not get activeflow variable. err: %v", err)
		return false, err
	}

	digits := vars.Variables[variableCallDigits]
	if option.Length > 0 && len(digits) >= option.Length {
		// matched length
		log.Debugf("Length matched. digits: %s, len: %d", digits, option.Length)
		return true, nil
	}

	l := len(digits)
	for i := 0; i < l; i++ {
		if strings.Contains(option.Key, string(digits[i])) {
			log.Debugf("Key matched. digits: %s, key: %s", digits, option.Key)
			return true, nil
		}
	}

	return false, nil
}
