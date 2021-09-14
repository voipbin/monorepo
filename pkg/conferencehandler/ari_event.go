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
	if cn.Type != channel.TypeConf {
		return nil
	}

	return h.leaved(cn, br)
}
