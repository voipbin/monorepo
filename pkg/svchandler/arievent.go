package svchandler

import (
	"context"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

// ARIChannelEnteredBridge is called when the channel handler received ChannelEnteredBridge.
func (h *svcHandler) ARIChannelEnteredBridge(cn *channel.Channel) error {
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
func (h *svcHandler) ARIChannelLeftBridge(cn *channel.Channel) error {
	ctx := context.Background()

	// get call
	c, err := h.db.CallGetByChannelIDAndAsteriskID(ctx, cn.ID, cn.AsteriskID)
	if err != nil {
		return err
	}

	if err := h.confHandler.Leaved(c.ID, c.ConfID); err != nil {
		return err
	}

	return nil
}
