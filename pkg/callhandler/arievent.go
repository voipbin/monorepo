package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// ARIStasisStart is called when the channel handler received StasisStart.
func (h *callHandler) ARIStasisStart(cn *channel.Channel, data map[string]interface{}) error {
	logrus.WithField("func", "ARIStasisStart").Debugf("Execute the stasis start event handler for call.")

	return h.StartCallHandle(cn, data)
}

// ARIChannelDestroyed handles ChannelDestroyed ARI event
func (h *callHandler) ARIChannelDestroyed(cn *channel.Channel) error {
	log := logrus.WithField("func", "ARIChannelDestroyed")

	switch cn.Type {
	case channel.TypeCall:
		return h.Hangup(cn)

	case channel.TypeConf, channel.TypeJoin, channel.TypeExternal, channel.TypeRecording, channel.TypeApplication:
		// we don't do anything at here.
		return nil

	default:
		log.WithField("channel", cn).Errorf("Unsupported channel type. type: %v", cn.Type)
		return nil
	}
}

// ARIChannelDtmfReceived handles ChannelDtmfReceived ARI event
func (h *callHandler) ARIChannelDtmfReceived(cn *channel.Channel, digit string, duration int) error {

	// support pjsip type only for now.
	if cn.Tech != channel.TechPJSIP {
		return nil
	}

	if err := h.DTMFReceived(cn, digit, duration); err != nil {
		return err
	}

	return nil
}

// ARIPlaybackFinished handles PlaybackFinished ARI event
// parsed playbackID to the action id, and execute the next action if its correct.
func (h *callHandler) ARIPlaybackFinished(cn *channel.Channel, playbackID string) error {
	ctx := context.Background()

	actionID := uuid.FromStringOrNil(playbackID)

	c, err := h.db.CallGetByChannelID(ctx, cn.ID)
	if err != nil {
		logrus.Errorf("Could not get call info. channel: %s, err: %v", cn.ID, err)
		return err
	}

	// compare actionID
	if c.Action.ID != actionID {
		logrus.Debugf("The call's action id does not match. call: %s, channel: %s, action: %s", c.ID, cn.ID, actionID)
		return nil
	}

	// go to next action
	return h.ActionNext(c)
}

func (h *callHandler) ARIChannelStateChange(cn *channel.Channel) error {
	ctx := context.Background()

	status := call.GetStatusByChannelState(cn.State)
	if status != call.StatusRinging && status != call.StatusProgressing {
		// the call cares only riniging/progressing at here.
		// other statuses will be handled in the other func.
		return nil
	}

	// get call
	c, err := h.db.CallGetByChannelID(ctx, cn.ID)
	if err == dbhandler.ErrNotFound {
		return nil
	} else if err != nil {
		return err
	}

	// we care only ringing/progress at here.
	if status == call.StatusRinging {
		return h.updateStatusRinging(ctx, cn, c)
	} else if status == call.StatusProgressing {
		return h.updateStatusProgressing(ctx, cn, c)
	}
	return nil
}

func (h *callHandler) ARIChannelLeftBridge(cn *channel.Channel, br *bridge.Bridge) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "ARIChannelLeftBridge",
		"asterisk_id": cn.AsteriskID,
		"channel_id":  cn.ID,
	})

	switch cn.Type {
	case channel.TypeCall:
		return nil
	case channel.TypeJoin:
		return h.bridgeLeftJoin(ctx, cn, br)
	case channel.TypeExternal:
		return h.bridgeLeftExternal(ctx, cn, br)

	default:
		log.WithFields(logrus.Fields{
			"channel": cn,
			"bridge":  br,
		}).Errorf("Could not find correct event handler.")
	}
	return nil
}
