package confbridgehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
)

// Kick kicks out the call from the conference
func (h *confbridgeHandler) Kick(ctx context.Context, id uuid.UUID, callID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"confbridge_id": id.String(),
		"call_id":       callID.String(),
	})
	log.Debugf("Kicking out the call from the confbridge.")

	// get confbridge info
	cb, err := h.db.ConfbridgeGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get confbridge info. err: %v", err)
		return err
	}
	log.WithField("confbridge", cb).Debug("Found confbridge info.")

	// hangup the join call info
	for joinedChannelID, joinedCallID := range cb.ChannelCallIDs {

		if joinedCallID == callID {

			// hangup the channels
			tmp, err := h.channelHandler.HangingUp(ctx, joinedChannelID, ari.ChannelCauseNormalClearing)
			if err != nil {
				log.Errorf("Could not hangup the channel correctly. err: %v", err)
				return err
			}
			log.WithField("channel", tmp).Debugf("Kicked the joined channel from the bridge. call_id: %s, channel_id: %s", joinedCallID, joinedChannelID)

			log.Debug("Hangup the joined call.")
			return nil
		}
	}

	log.Error("Could not find joined call.")
	return fmt.Errorf("joined call not found")
}
