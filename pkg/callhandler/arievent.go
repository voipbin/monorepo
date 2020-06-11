package callhandler

import (
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/channel"
)

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

// ARIChannelDestroyed handles ChannelDestroyed ARI event
func (h *callHandler) ARIChannelDestroyed(cn *channel.Channel) error {
	contextType := getContextType(cn.Data["CONTEXT"])
	switch contextType {
	case contextTypeConference:
		return h.confHandler.ARIStasisStart(cn)
	case contextTypeCall:
		return h.Hangup(cn)
	default:
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
