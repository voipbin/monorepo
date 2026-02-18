package externalmediahandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/externalmedia"
)

// Stop stops the external media processing
func (h *externalMediaHandler) Stop(ctx context.Context, externalMediaID uuid.UUID) (*externalmedia.ExternalMedia, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "Stop",
		"external_media_id": externalMediaID,
	})
	log.Debug("Stopping the external media.")

	res, err := h.UpdateStatus(ctx, externalMediaID, externalmedia.StatusTerminating)
	if err != nil {
		return nil, fmt.Errorf("could not update external media status: %w", err)
	}
	promExternalMediaStopTotal.WithLabelValues(string(res.ReferenceType)).Inc()

	// hangup the external media channel
	if errHangup := h.channelHandler.HangingUpWithAsteriskID(ctx, res.AsteriskID, res.ChannelID, ari.ChannelCauseNormalClearing); errHangup != nil {
		return nil, errors.Wrapf(errHangup, "could not hangup the external media channel")
	}

	// delete external media info
	if errExtDelete := h.db.ExternalMediaDelete(ctx, externalMediaID); errExtDelete != nil {
		return nil, errors.Wrapf(errExtDelete, "could not delete external media info from db")
	}

	// remove the external media ID from the parent's ExternalMediaIDs array
	switch res.ReferenceType {
	case externalmedia.ReferenceTypeCall:
		if errRemove := h.db.CallRemoveExternalMediaID(ctx, res.ReferenceID, externalMediaID); errRemove != nil {
			log.Errorf("Could not remove external media id from call. call_id: %s, err: %v", res.ReferenceID, errRemove)
		}
	case externalmedia.ReferenceTypeConfbridge:
		if errRemove := h.db.ConfbridgeRemoveExternalMediaID(ctx, res.ReferenceID, externalMediaID); errRemove != nil {
			log.Errorf("Could not remove external media id from confbridge. confbridge_id: %s, err: %v", res.ReferenceID, errRemove)
		}
	}

	return res, nil
}
