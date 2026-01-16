package transcripthandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
)

// Create creates a new transcribe
func (h *transcriptHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	transcribeID uuid.UUID,
	direction transcript.Direction,
	message string,
	tmTranscript string,
) (*transcript.Transcript, error) {

	id := h.utilHandler.UUIDCreate()
	tr := &transcript.Transcript{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		TranscribeID: transcribeID,

		Direction: direction,
		Message:   message,

		TMTranscript: tmTranscript,
	}

	if errCreate := h.db.TranscriptCreate(ctx, tr); errCreate != nil {
		return nil, errors.Wrapf(errCreate, "could not create the transcript.")
	}

	res, err := h.db.TranscriptGet(ctx, tr.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get a created transcript.")
	}

	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, transcript.EventTypeTranscriptCreated, res)
	return res, nil
}

// List returns list of transcripts.
func (h *transcriptHandler) List(ctx context.Context, size uint64, token string, filters map[transcript.Field]any) ([]*transcript.Transcript, error) {
	res, err := h.db.TranscriptList(ctx, size, token, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get transcripts. filters: %v", filters)
	}

	return res, nil
}

// Delete deletes the transcript
func (h *transcriptHandler) Delete(ctx context.Context, id uuid.UUID) (*transcript.Transcript, error) {
	// get transcript
	tr, err := h.dbGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get transcript info. transcript_id: %s", id)
	}

	if tr.TMDelete != dbhandler.DefaultTimeStamp {
		// already deleted
		return tr, nil
	}

	res, err := h.dbDelete(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not delete the transcript. transcript_id: %s", id)
	}

	return res, nil
}
