package callhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

func (h *callHandler) DTMFReceived(cn *channel.Channel, digit string, duration int) error {
	ctx := context.Background()

	log := logrus.WithFields(
		logrus.Fields{
			"channel": cn,
		},
	)

	c, err := h.db.CallGetByChannelID(ctx, cn.ID)
	if err != nil {
		log.Infof("The channel is not call type.")
		return nil
	}
	log = log.WithFields(
		logrus.Fields{
			"call": c,
		},
	)

	if c.Action.Type != action.TypeDTMFReceive {
		// overwrite dtmf receive cache.
		log.Debugf("The current action is not dtmf receive.")

		// we are setting the dtmf here even it is not dtmf receive action.
		// this is needed, because if the user press the dtmf in the prior of dtmf receive(i.e play action),
		// the user exepects pressed dtmf could be collected in the dtmf received action in next.
		h.db.CallDTMFSet(ctx, c.ID, digit)
		return nil
	}

	var option action.OptionDTMFReceive
	if err := json.Unmarshal(c.Action.Option, &option); err != nil {
		log.Errorf("could not parse the option. err: %v", err)
		return fmt.Errorf("could not parse option. action: %v, err: %v", c.Action, err)
	}

	// get dtmfs
	// we don't check the error, because if it's the first time to setting the dtmf
	// it will return the not found error. and we can't distinguish that it's the db's error
	// or first time setting.
	dtmfs, _ := h.db.CallDTMFGet(ctx, c.ID)
	dtmfs = fmt.Sprintf("%s%s", dtmfs, digit)

	// update the dtmfs
	h.db.CallDTMFSet(ctx, c.ID, dtmfs)

	// check finish condition
	if strings.Contains(option.FinishOnKey, digit) == false && len(dtmfs) < option.MaxNumKey {
		// the dtmf receive is not finish yet
		logrus.Debugf("The dtmf receive does not finish yet. Wating next dtmf. call: %v", c.ID)
		return nil
	}

	logrus.Infof("Finished dtmf receiving. call: %s, dtmfs: %s", c.ID, dtmfs)

	// send next action request
	h.reqHandler.CallCallActionNext(c.ID)

	return nil
}
