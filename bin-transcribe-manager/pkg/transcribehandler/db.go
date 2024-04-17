package transcribehandler

import (
	"context"

	"monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// dbDelete deletes the transcribe
func (h *transcribeHandler) dbDelete(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "dbDelete",
		"transcribe_id": id,
	})

	if errDelete := h.db.TranscribeDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete the transcribe info. err: %v", errDelete)
		return nil, errDelete
	}

	// get deleted item
	res, err := h.db.TranscribeGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted transcribe. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, transcribe.EventTypeTranscribeDeleted, res)

	return res, nil
}
