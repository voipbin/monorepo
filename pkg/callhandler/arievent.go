package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

// ARIChannelEnteredBridge is called when the channel handler received ChannelEnteredBridge.
func (h *callHandler) ARIChannelEnteredBridge(cn *channel.Channel, bridge *bridge.Bridge) error {
	ctx := context.Background()

	// get call
	c, err := h.db.CallGetByChannelIDAndAsteriskID(ctx, cn.ID, cn.AsteriskID)
	if err != nil {
		return err
	}

	if err := h.db.CallSetConferenceID(ctx, c.ID, bridge.ConferenceID); err != nil {
		return err
	}

	if err := h.confHandler.Joined(c.ID, c.ConfID); err != nil {
		return err
	}

	return nil
}

// ARIChannelLeftBridge is called when the channel handler received ChannelLeftBridge.
func (h *callHandler) ARIChannelLeftBridge(cn *channel.Channel, bridge *bridge.Bridge) error {
	ctx := context.Background()

	// get call
	c, err := h.db.CallGetByChannelIDAndAsteriskID(ctx, cn.ID, cn.AsteriskID)
	if err != nil {
		return err
	}

	// set empty conference id
	if err := h.db.CallSetConferenceID(ctx, c.ID, uuid.Nil); err != nil {
		return err
	}

	// notice to the conference
	if err := h.confHandler.Leaved(c.ID, c.ConfID); err != nil {
		return err
	}

	// do next action
	h.ActionNext(c)

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

func (h *callHandler) ARIChannelDestroyed(cn *channel.Channel) error {
	return h.Hangup(cn)
}
