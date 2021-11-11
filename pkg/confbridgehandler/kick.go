package confbridgehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
)

// Kick kicks out the call from the conference
func (h *confbridgeHandler) Kick(ctx context.Context, id, callID uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
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

	// hangup the join call info
	for joinedChannelID, joinedCallID := range cb.ChannelCallIDs {

		if joinedCallID == callID {

			// get channel info
			ch, err := h.db.ChannelGet(ctx, joinedChannelID)
			if err != nil {
				log.Errorf("Could not get joined channel info. err: %v", err)
				return err
			}

			// hang up the call
			if errHangup := h.reqHandler.AstChannelHangup(ctx, ch.AsteriskID, ch.ID, ari.ChannelCauseNormalClearing); errHangup != nil {
				log.Errorf("Could not hang up the joined call. err: %v", errHangup)
				return errHangup
			}

			log.Debug("Hangup the joined call.")
			return nil
		}
	}

	log.Error("Could not find joined call.")
	return fmt.Errorf("joined call not found")
}
