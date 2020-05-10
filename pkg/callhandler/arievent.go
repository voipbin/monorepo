package callhandler

import (
	"context"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

// ARIChannelEnteredBridge is called when the channel handler received ChannelEnteredBridge.
func (h *callHandler) ARIChannelEnteredBridge(cn *channel.Channel) error {
	ctx := context.Background()

	// get call
	c, err := h.db.CallGetByChannelIDAndAsteriskID(ctx, cn.ID, cn.AsteriskID)
	if err != nil {
		return err
	}

	if err := h.confHandler.Joined(c.ID, c.ConfID); err != nil {
		return err
	}

	return nil
}

// ARIChannelLeftBridge is called when the channel handler received ChannelLeftBridge.
func (h *callHandler) ARIChannelLeftBridge(cn *channel.Channel) error {
	ctx := context.Background()

	// get call
	c, err := h.db.CallGetByChannelIDAndAsteriskID(ctx, cn.ID, cn.AsteriskID)
	if err != nil {
		return err
	}

	if err := h.confHandler.Leaved(c.ID, c.ConfID); err != nil {
		return err
	}

	// do next action

	return nil
}

// ARIStasisStart is called when the channel handler received StasisStart.
func (h *callHandler) ARIStasisStart(cn *channel.Channel) error {
	contextType := getContextType(cn.Data["CONTEXT"])
	switch contextType {
	case contextTypeConference:
		return h.confHandler.ARIStasisStart(cn)
	default:
		return h.Start(cn)
	}
}
