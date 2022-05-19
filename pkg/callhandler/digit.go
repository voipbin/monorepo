package callhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// digitsReceived handles DTMF Recevied event
func (h *callHandler) digitsReceived(cn *channel.Channel, digit string, duration int) error {
	ctx := context.Background()

	log := logrus.WithFields(
		logrus.Fields{
			"channel": cn,
			"func":    "digitsReceived",
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

	if c.Action.Type != fmaction.TypeDigitsReceive {
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

	var option fmaction.OptionDigitsReceive
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
	if !strings.Contains(option.Key, digit) && len(dtmfs) < option.Length {
		// the dtmf receive is not finish yet
		log.Debug("The dtmf receive does not finish yet. Wating next dtmf.")
		return nil
	}
	log.Infof("Finished dtmf receiving. call: %s, dtmfs: %s", c.ID, dtmfs)

	// send next action request
	if errNext := h.reqHandler.CMV1CallActionNext(ctx, c.ID, false); errNext != nil {
		log.Errorf("Could not get next action. err: %v", errNext)
		_ = h.reqHandler.AstChannelHangup(ctx, c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing, 0)
	}

	return nil
}

// DigitsGet returns received dtmfs
func (h *callHandler) DigitsGet(ctx context.Context, id uuid.UUID) (string, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "DigitsGet",
			"call_id": id,
		},
	)

	res, err := h.db.CallDTMFGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get DTMF. err: %v", err)
		return "", nil
	}

	return res, nil
}

// DigitsSet sets the dtmfs
func (h *callHandler) DigitsSet(ctx context.Context, id uuid.UUID, digits string) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "DigitsSet",
			"call_id": id,
		},
	)

	if err := h.db.CallDTMFSet(ctx, id, digits); err != nil {
		log.Errorf("Could not set DTMF. err: %v", err)
		return err
	}

	return nil
}
