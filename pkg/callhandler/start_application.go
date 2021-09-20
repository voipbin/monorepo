package callhandler

import (
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// list of service dialplan context
const (
	serviceContextAMD = "svc-amd"
)

// startHandlerContextApplication handles contextApplication context type of StasisStart event.
func (h *callHandler) applicationHandleAMD(cn *channel.Channel, data map[string]interface{}) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "applicationHandleAMD",
		"channel_id": cn.ID,
		"call_id":    data["call_id"],
	})
	log.Debug("Executing the applciationHandleAMD.")

	h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeApplication))

	// put the cahnnel to the amd
	if errContinue := h.reqHandler.AstChannelContinue(cn.AsteriskID, cn.ID, serviceContextAMD, "", 0, ""); errContinue != nil {
		log.Errorf("Could not continue the channel. err: %v", errContinue)
	}

	return nil
}
