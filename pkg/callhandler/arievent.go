package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// ARIStasisStart is called when the channel handler received StasisStart.
func (h *callHandler) ARIStasisStart(cn *channel.Channel, data map[string]interface{}) error {
	contextType := getContextType(data["context"])
	switch contextType {
	case contextTypeConference:
		return h.confHandler.ARIStasisStart(cn, data)
	default:
		return h.StartCallHandle(cn, data)
	}
}

// ARIChannelDestroyed handles ChannelDestroyed ARI event
func (h *callHandler) ARIChannelDestroyed(cn *channel.Channel) error {

	switch cn.Type {
	case channel.TypeCall:
		return h.Hangup(cn)

	case channel.TypeConf, channel.TypeJoin:
		// we don't do anything at here.
		// because for the conference context type, ChannelLeftBridge event handle will
		// handle the channel termination.
		return nil

	default:
		logrus.Warnf("Could not find correct event handler. event: %v", cn)
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
