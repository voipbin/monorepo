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

// Joined handles joined call
func (h *confbridgeHandler) Joined(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "Joined",
			"channel_id": cn.ID,
			"bridge_id":  br.ID,
		},
	)

	confbridgeID := uuid.FromStringOrNil(cn.StasisData["confbridge_id"])
	callID := uuid.FromStringOrNil(cn.StasisData["call_id"])
	log = log.WithFields(logrus.Fields{
		"conbridge_id": confbridgeID,
		"call_id":      callID,
	})
	log.Debug("Joined channel to the confbridge.")

	cb, err := h.AddChannelCallID(ctx, confbridgeID, cn.ID, callID)
	if err != nil {
		log.Errorf("Could not add the channel/call info to the confbridge. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseUnallocated)
		return errors.Wrap(err, "could not add the confbridge's channel/call info")
	}

	// set confbridge id to the call
	if err := h.db.CallSetConfbridgeID(ctx, callID, confbridgeID); err != nil {
		log.Errorf("Could not set the conference id for a call. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNormalClearing)
		return err
	}

	// Publish the event
	evt := &confbridge.EventConfbridgeJoined{
		Confbridge:   *cb,
		JoinedCallID: callID,
	}
	h.notifyHandler.PublishEvent(ctx, confbridge.EventTypeConfbridgeJoined, evt)

	// get updated call info and notify
	c, err := h.db.CallGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get updated call info. But we are keep moving. err: %v", err)
	}
	h.notifyHandler.PublishWebhookEvent(ctx, c.CustomerID, call.EventTypeCallUpdated, c)

	return nil
}
