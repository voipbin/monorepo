package confbridgehandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
)

// Leaved handles event the channel has left from the bridge
// when the channel has left from the conference bridge, this func will be fired.
func (h *confbridgeHandler) Leaved(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "leaved",
			"confbridge_id": br.ReferenceID,
			"channel_id":    cn.ID,
		},
	)
	confbridgeID := br.ReferenceID

	// remove the channel/call info from the confbridge
	if errCallChannelID := h.db.ConfbridgeRemoveChannelCallID(ctx, confbridgeID, cn.ID); errCallChannelID != nil {
		return fmt.Errorf("Could not remove the channel from the confbridge's channel/call info")
	}

	// get confbridge info and notify
	cb, err := h.db.ConfbridgeGet(ctx, confbridgeID)
	if err != nil {
		log.Errorf("Could not get confbridge info. err: %v", err)
	}
	h.notifyHandler.NotifyEvent(notifyhandler.EventTypeConfbridgeLeaved, "", cb)

	return nil
}
