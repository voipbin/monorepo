package conferencehandler

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

func (h *conferenceHandler) ARIStasisStart(cn *channel.Channel, data map[string]interface{}) error {

	mapType := map[interface{}]func(*channel.Channel, map[string]interface{}) error{
		contextConferenceJoin:     h.ariStasisStartContextJoin,
		contextConferenceIncoming: h.ariStasisStartContextIncoming,
	}

	handler := mapType[data["context"].(string)]
	if handler == nil {
		logrus.WithFields(
			logrus.Fields{
				"channel":     cn.ID,
				"asterisk_id": cn.AsteriskID,
				"data":        cn.Data,
			}).Errorf("Could not find correct event handler.")
	}

	return handler(cn, data)
}

// ariStasisStartContextJoin handles the call which has CONTEXT=conf-join in the StasisStart argument.
func (h *conferenceHandler) ariStasisStartContextJoin(cn *channel.Channel, data map[string]interface{}) error {

	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeJoin)); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		return fmt.Errorf("could not set channel var. id: %s, asterisk: %s, bridge: %s, err: %v", cn.ID, cn.AsteriskID, cn.DestinationNumber, err)
	}

	if err := h.reqHandler.AstBridgeAddChannel(cn.AsteriskID, data["bridge_id"].(string), cn.ID, "", false, false); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		return fmt.Errorf("could not put the channel to the bridge. id: %s, asterisk: %s, bridge: %s, err: %v", cn.ID, cn.AsteriskID, cn.DestinationNumber, err)
	}

	if err := h.reqHandler.AstChannelDial(cn.AsteriskID, cn.ID, "", defaultDialTimeout); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		return fmt.Errorf("could not dial the channel. id: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	return nil
}

// ariStasisStartContextIncoming handles the call which has CONTEXT=conf-in in the StasisStart argument.
func (h *conferenceHandler) ariStasisStartContextIncoming(cn *channel.Channel, data map[string]interface{}) error {

	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeConf)); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		return fmt.Errorf("could not set channel var. id: %s, asterisk: %s, bridge: %s, err: %v", cn.ID, cn.AsteriskID, cn.DestinationNumber, err)
	}

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

	// if the call type has joined to the bridge,
	// we have to call the joined handler for add the call info to the conference.
	if cn.Type == channel.TypeCall {
		return h.joined(cn, bridge)
	}

	// do nothing for other types
	return nil
}
