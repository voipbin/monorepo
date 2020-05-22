package conferencehandler

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

func (h *conferenceHandler) ARIStasisStart(cn *channel.Channel) error {

	mapType := map[interface{}]func(*channel.Channel) error{
		contextConferenceEcho: h.ariStasisStartContextEcho,
	}

	handler := mapType[cn.Data["CONTEXT"]]
	if handler == nil {
		log.WithFields(
			log.Fields{
				"channel":     cn.ID,
				"asterisk_id": cn.AsteriskID,
				"data":        cn.Data,
			}).Errorf("Could not find correct event handler.")
	}

	return handler(cn)
}

func (h *conferenceHandler) ariStasisStartContextEcho(cn *channel.Channel) error {
	if cn.Data["BRIDGE_ID"] == nil {
		return nil
	}

	// add the created snoop channel into the bridge
	if err := h.reqHandler.AstBridgeAddChannel(cn.AsteriskID, cn.Data["BRIDGE_ID"].(string), cn.ID, "", false, false); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		return fmt.Errorf("could not add the snoop channel into the bridge. id: %s, bridge: %s", cn.ID, cn.Data["BRIDGE_ID"].(string))
	}

	return nil
}
