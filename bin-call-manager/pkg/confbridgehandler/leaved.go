package confbridgehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
)

// Leaved handles event the channel has left from the bridge
// when the channel has left from the conference bridge, this func will be fired.
func (h *confbridgeHandler) Leaved(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Leaved",
		"channel": cn,
		"bridge":  br,
	})

	confbridgeID := uuid.FromStringOrNil(cn.StasisData[channel.StasisDataTypeConfbridgeID])
	callID := uuid.FromStringOrNil(cn.StasisData[channel.StasisDataTypeCallID])
	log = log.WithFields(logrus.Fields{
		"call_id":       callID,
		"confbridge_id": confbridgeID,
	})
	log.Debug("Leaved channel from the confbridge.")

	// hang up the channel
	_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNormalClearing)

	cb, err := h.RemoveChannelCallID(ctx, confbridgeID, cn.ID, callID)
	if err != nil {
		log.Errorf("Could not remove the channel info from the confbridge. err: %v", err)
		return errors.Wrap(err, "could not remove the channel info from the confbridge")
	}

	// update the call's confbridge info
	go func() {
		_, err := h.reqHandler.CallV1CallUpdateConfbridgeID(ctx, callID, uuid.Nil)
		if err != nil {
			log.Errorf("Could not update the call's confbridge info. err: %v", err)
		}
	}()

	if cb.Type == confbridge.TypeConference {
		// nothing to do
		return nil
	}

	if len(cb.ChannelCallIDs) == 1 && !h.flagExist(ctx, cb.Flags, confbridge.FlagNoAutoLeave) {
		_, err := h.Terminating(ctx, cb.ID)
		if err != nil {
			log.Errorf("Could not terminating the confbridge. err: %v", err)
			return errors.Wrap(err, "could not terminating the confbridge")
		}
	}

	return nil
}
