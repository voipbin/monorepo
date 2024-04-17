package transcribehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
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
