package confbridgehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

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

	// update the call's confbridge info
	go func() {
		_, err := h.reqHandler.CallV1CallUpdateConfbridgeID(ctx, callID, uuid.Nil)
		if err != nil {
			log.Errorf("Could not update the call's confbridge info. err: %v", err)
		}
	}()

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

	return nil
}
