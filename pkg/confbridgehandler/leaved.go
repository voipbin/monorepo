package confbridgehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler/models/event"
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
	confbridgeID := uuid.FromStringOrNil(cn.StasisData["confbridge_id"])
	callID := uuid.FromStringOrNil(cn.StasisData["call_id"])

	// remove the channel/call info from the confbridge
	if errCallChannelID := h.db.ConfbridgeRemoveChannelCallID(ctx, confbridgeID, cn.ID); errCallChannelID != nil {
		return fmt.Errorf("Could not remove the channel from the confbridge's channel/call info")
	}

	// set nil conference id to the call
	// note: here we are setting the conference's id to the call.
	// we don't set the confbridge id to the call.
	if err := h.db.CallSetConfbridgeID(ctx, callID, uuid.Nil); err != nil {
		log.Errorf("Could not set the conference id for a call. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return err
	}

	// Publish the event
	evt := &event.ConfbridgeJoinedLeaved{
		ID:     confbridgeID,
		CallID: callID,
	}
	h.notifyHandler.PublishEvent(ctx, notifyhandler.EventTypeConfbridgeLeaved, evt)

	// get updated call info and notify
	call, err := h.db.CallGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get updated call info. But we are keep moving. err: %v", err)
	}
	h.notifyHandler.PublishWebhookEvent(ctx, notifyhandler.EventTypeCallUpdated, call.WebhookURI, call)

	return nil
}
