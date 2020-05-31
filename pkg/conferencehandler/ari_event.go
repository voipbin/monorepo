package conferencehandler

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

func (h *conferenceHandler) ARIStasisStart(cn *channel.Channel) error {

	mapType := map[interface{}]func(*channel.Channel) error{
		contextConferenceEcho:     h.ariStasisStartContextEcho,
		contextConferenceJoin:     h.ariStasisStartcontextJoin,
		contextConferenceIncoming: h.ariStasisStartcontextIn,
	}

	handler := mapType[cn.Data["CONTEXT"]]
	if handler == nil {
		logrus.WithFields(
			logrus.Fields{
				"channel":     cn.ID,
				"asterisk_id": cn.AsteriskID,
				"data":        cn.Data,
			}).Errorf("Could not find correct event handler.")
	}

	return handler(cn)
}

// ariStasisStartContextEcho handles the call which has CONTEXT=conf-echo in the StasisStart argument.
func (h *conferenceHandler) ariStasisStartContextEcho(cn *channel.Channel) error {
	if cn.Data["BRIDGE_ID"] == nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		return nil
	}

	// add the created snoop channel into the bridge
	if err := h.reqHandler.AstBridgeAddChannel(cn.AsteriskID, cn.Data["BRIDGE_ID"].(string), cn.ID, "", false, false); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		return fmt.Errorf("could not add the snoop channel into the bridge. id: %s, bridge: %s", cn.ID, cn.Data["BRIDGE_ID"].(string))
	}

	return nil
}

// ariStasisStartcontextJoin handles the call which has CONTEXT=conf-join in the StasisStart argument.
func (h *conferenceHandler) ariStasisStartcontextJoin(cn *channel.Channel) error {

	if err := h.reqHandler.AstBridgeAddChannel(cn.AsteriskID, cn.Data["BRIDGE_ID"].(string), cn.ID, "", false, false); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		return fmt.Errorf("could not put the channel to the bridge. id: %s, asterisk: %s, bridge: %s, err: %v", cn.ID, cn.AsteriskID, cn.DestinationNumber, err)
	}

	if err := h.reqHandler.AstChannelDial(cn.AsteriskID, cn.ID, "", 30); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		return fmt.Errorf("could not dial the channel. id: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	return nil
}

// ariStasisStartcontextIn handles the call which has CONTEXT=conf-in in the StasisStart argument.
func (h *conferenceHandler) ariStasisStartcontextIn(cn *channel.Channel) error {
	// answer the call. it is safe to call this for answered call.
	if err := h.reqHandler.AstChannelAnswer(cn.AsteriskID, cn.ID); err != nil {
		logrus.Errorf("Could not answer the call. err: %v", err)
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		return err
	}

	if err := h.reqHandler.AstBridgeAddChannel(cn.AsteriskID, cn.DestinationNumber, cn.ID, "", false, false); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		return fmt.Errorf("could not put the channel to the bridge. id: %s, asterisk: %s, bridge: %s, err: %v", cn.ID, cn.AsteriskID, cn.DestinationNumber, err)
	}

	return nil
}

func (h *conferenceHandler) ARIChannelLeftBridge(cn *channel.Channel, br *bridge.Bridge) error {
	return h.leaved(cn, br)
}

// ARIChannelEnteredBridge is called when the channel handler received ChannelEnteredBridge.
func (h *conferenceHandler) ARIChannelEnteredBridge(cn *channel.Channel, bridge *bridge.Bridge) error {
	if cn.GetContextType() == channel.ContextTypeConference {
		// nothing to do here
		return nil
	}

	return h.joined(cn, bridge)
}
