package transcripthandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
)

// dbGet returns transcript
func (h *transcriptHandler) dbGet(ctx context.Context, id uuid.UUID) (*transcript.Transcript, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "dbGet",
		"transcript_id": id,
	})

	res, err := h.db.TranscriptGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get transcript info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// dbDelete deletes the transcript
func (h *transcriptHandler) dbDelete(ctx context.Context, id uuid.UUID) (*transcript.Transcript, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "dbDelete",
		"transcript_id": id,
	})

	if errDelete := h.db.TranscriptDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete the transcript info. err: %v", errDelete)
		return nil, errDelete
	}

	// get deleted item
	res, err := h.db.TranscriptGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted transcript. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, transcript.EventTypeTranscriptCreated, res)

	return res, nil
}
