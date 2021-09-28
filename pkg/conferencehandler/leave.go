package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// Leave outs the call from the conference
func (h *conferenceHandler) Leave(id, callID uuid.UUID) error {
	ctx := context.Background()

	log := log.WithFields(
		log.Fields{
			"conference": id.String(),
			"call":       callID.String(),
		})
	log.Debugf("Leaving the call from the conference.")

	// get call info
	c, err := h.db.CallGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get call. err: %v", err)
		return err
	}

	// get bridge info
	br, err := h.db.BridgeGet(ctx, c.BridgeID)
	if err != nil {
		log.Errorf("Could not get bridge info. err: %v", err)
		return err
	}

	// get join channel
	var joinChannel *channel.Channel = nil
	for _, tmpID := range br.ChannelIDs {
		ch, err := h.db.ChannelGet(ctx, tmpID)
		if err != nil {
			log.Errorf("Could not get channel info. err: %v", err)
			return err
		}

		if ch.Type == channel.TypeJoin {
			joinChannel = ch
		}
	}

	if joinChannel == nil {
		log.Errorf("Could not find join channel.")
		return nil
	}

	// hangup the join channel
	if err := h.reqHandler.AstChannelHangup(joinChannel.AsteriskID, joinChannel.ID, ari.ChannelCauseNormalClearing); err != nil {
		log.WithFields(
			logrus.Fields{
				"bridge":  joinChannel.BridgeID,
				"channel": joinChannel.ID,
			}).Errorf("Could not kick out the call from the conference. err: %v", err)
		return err
	}

	return nil
}
