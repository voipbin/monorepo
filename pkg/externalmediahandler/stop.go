package externalmediahandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
)

// Stop stops the external media processing
func (h *externalMediaHandler) Stop(ctx context.Context, externalMediaID uuid.UUID) (*externalmedia.ExternalMedia, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "Stop",
		"external_media_id": externalMediaID,
	})
	log.Debug("Stopping the external media.")

	// get external media
	res, err := h.db.ExternalMediaGet(ctx, externalMediaID)
	if err != nil || res == nil {
		log.Debug("No external media exist. Nothing to do.")
		return nil, fmt.Errorf("could not find external media")
	}

	// hangup the external media channel
	if errHangup := h.reqHandler.AstChannelHangup(ctx, res.AsteriskID, res.ChannelID, ari.ChannelCauseNormalClearing, 0); errHangup != nil {
		log.Errorf("Could not hangup the external media channel. err: %v", errHangup)
		return nil, fmt.Errorf("could not hangup the external media channel")
	}

	// delete external media info
	if errExtDelete := h.db.ExternalMediaDelete(ctx, externalMediaID); errExtDelete != nil {
		log.Errorf("Could not delete external media info. err: %v", errExtDelete)
		return nil, fmt.Errorf("could not delete external media")
	}

	return res, nil
}
