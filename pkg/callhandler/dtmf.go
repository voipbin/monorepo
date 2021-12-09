package callhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// DTMFReceived handles DTMF Recevied event
func (h *callHandler) DTMFReceived(cn *channel.Channel, digit string, duration int) error {
	ctx := context.Background()

	log := logrus.WithFields(
		logrus.Fields{
			"channel": cn,
			"func":    "DTMFReceived",
		},
	)

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

	if c.Action.Type != action.TypeDTMFReceive {
		// overwrite dtmf receive cache.
		log.WithField("action_type", c.Action.Type).Debug("The current action is not dtmf receive.")

		// check the condition

		// we are setting the dtmf here even it is not dtmf receive action.
		// this is needed, because if the user press the dtmf in the prior of dtmf receive(i.e play action),
		// the user exepects pressed dtmf could be collected in the dtmf received action in next.
		if err := h.db.CallDTMFSet(ctx, c.ID, digit); err != nil {
			log.Errorf("Could not set DTMF. err: %v", err)
		}
		return nil
	}

	var option action.OptionDTMFReceive
	if err := json.Unmarshal(c.Action.Option, &option); err != nil {
		log.WithField("action", c.Action).Errorf("could not parse the option. err: %v", err)
		return fmt.Errorf("could not parse option. action: %v, err: %v", c.Action, err)
	}

	// get dtmfs
	// we don't check the error, because if it's the first time to setting the dtmf
	// it will return the not found error. and we can't distinguish that it's the db's error
	// or first time setting.
	dtmfs, _ := h.db.CallDTMFGet(ctx, c.ID)
	dtmfs = fmt.Sprintf("%s%s", dtmfs, digit)

	// update the dtmfs
	if errDTMFSet := h.db.CallDTMFSet(ctx, c.ID, dtmfs); errDTMFSet != nil {
		log.Errorf("Could not set dtmf. err: %v", errDTMFSet)
		return nil
	}

	// check finish condition
	if !strings.Contains(option.FinishOnKey, digit) && len(dtmfs) < option.MaxNumKey {
		// the dtmf receive is not finish yet
		log.Debug("The dtmf receive does not finish yet. Wating next dtmf.")
		return nil
	}
	log.Infof("Finished dtmf receiving. call: %s, dtmfs: %s", c.ID, dtmfs)

	// send next action request
	if errNext := h.reqHandler.CMV1CallActionNext(ctx, c.ID, false); errNext != nil {
		log.Errorf("Could not get next action. err: %v", errNext)
		_ = h.reqHandler.AstChannelHangup(ctx, c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
	}

	return nil
}
