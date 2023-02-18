package confbridgehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
)

// Leaved handles event the channel has left from the bridge
// when the channel has left from the conference bridge, this func will be fired.
func (h *confbridgeHandler) Leaved(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "Leaved",
			"channel_id": cn.ID,
			"bridge_id":  br.ID,
		},
	)

	confbridgeID := uuid.FromStringOrNil(cn.StasisData["confbridge_id"])
	callID := uuid.FromStringOrNil(cn.StasisData["call_id"])
	log = log.WithFields(logrus.Fields{
		"call_id":       callID,
		"confbridge_id": confbridgeID,
	})
	log.Debug("Leaved channel from the confbridge.")

	cb, err := h.RemoveChannelCallID(ctx, confbridgeID, cn.ID)
	if err != nil {
		log.Errorf("Could not remove the channel info from the confbridge. err: %v", err)
		return errors.Wrap(err, "could not remove the channel info from the confbridge")
	}

	// todo: here we are updating the call's confbridge id.
	// but we should not do this. instead of doing this, we need to send the request to the callhandler
	// and let them handle the update work
	// set nil conference id to the call
	// note: here we are setting the conference's id to the call.
	// we don't set the confbridge id to the call.
	if err := h.db.CallSetConfbridgeID(ctx, callID, uuid.Nil); err != nil {
		log.Errorf("Could not set the conference id for a call. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNormalClearing)
		return err
	}

	// check the confbridge type
	if cb.Type == confbridge.TypeConnect && len(cb.ChannelCallIDs) == 1 {
		// kick the other channel
		for _, joinedCallID := range cb.ChannelCallIDs {
			go func(kickID uuid.UUID) {
				log.Debugf("Kicking out the call from the confbridge. call_id: %s", kickID)
				if errKick := h.reqHandler.CallV1ConfbridgeCallKick(ctx, cb.ID, kickID); errKick != nil {
					log.Errorf("Could not kick the call from the confbridge. err: %v", errKick)
				}
			}(joinedCallID)
		}
	}

	// Publish the event
	evt := &confbridge.EventConfbridgeLeaved{
		Confbridge:   *cb,
		LeavedCallID: callID,
	}
	h.notifyHandler.PublishEvent(ctx, confbridge.EventTypeConfbridgeLeaved, evt)

	// get updated c info and notify
	c, err := h.db.CallGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get updated call info. But we are keep moving. err: %v", err)
	}
	h.notifyHandler.PublishWebhookEvent(ctx, c.CustomerID, call.EventTypeCallUpdated, c)

	return nil
}
