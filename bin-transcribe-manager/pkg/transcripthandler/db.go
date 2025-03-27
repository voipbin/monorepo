package transcripthandler

import (
	"context"

	"monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// dbGet returns transcript
func (h *transcriptHandler) dbGet(ctx context.Context, id uuid.UUID) (*transcript.Transcript, error) {
	res, err := h.db.TranscriptGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get transcript info.")
	}

	return res, nil
}

// dbDelete deletes the transcript
func (h *transcriptHandler) dbDelete(ctx context.Context, id uuid.UUID) (*transcript.Transcript, error) {
	if errDelete := h.db.TranscriptDelete(ctx, id); errDelete != nil {
		return nil, errors.Wrapf(errDelete, "could not delete the transcript info.")
	}

	// get deleted item
	res, err := h.db.TranscriptGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get deleted transcript.")
	}
	h.notifyHandler.PublishEvent(ctx, transcript.EventTypeTranscriptCreated, res)

	return res, nil
}
