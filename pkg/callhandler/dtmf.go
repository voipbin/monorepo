package callhandler

import (
	"context"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/channel"
)

func (h *callHandler) DTMFReceived(cn *channel.Channel, digit string, duration int) error {
	ctx := context.Background()

	c, err := h.db.CallGetByChannelID(ctx, cn.ID)
	if err != nil {
		return err
	}

	// check call's current action
	// currently, the only echo type supported.
	if c.Action.Type != action.TypeEcho {
		// nothing todo now
		return nil
	}

	// send the echo dtmf
	if err := h.reqHandler.AstChannelDTMF(c.AsteriskID, c.ChannelID, digit, duration, 0, 0, 0); err != nil {
		return err
	}

	return nil
}
