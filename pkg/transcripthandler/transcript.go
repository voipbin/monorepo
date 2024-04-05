package transcripthandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
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
	log := logrus.WithFields(logrus.Fields{
		"func":          "Create",
		"customer_id":   customerID,
		"transcribe_id": transcribeID,
		"direction":     direction,
		"message":       message,
	})

	id := h.utilHandler.UUIDCreate()
	tr := &transcript.Transcript{
		ID:           id,
		CustomerID:   customerID,
		TranscribeID: transcribeID,

		Direction: direction,
		Message:   message,

		TMTranscript: tmTranscript,
	}

	if errCreate := h.db.TranscriptCreate(ctx, tr); errCreate != nil {
		log.Errorf("Could not create a tracript. err: %v", errCreate)
		return nil, errCreate
	}

	res, err := h.db.TranscriptGet(ctx, tr.ID)
	if err != nil {
		log.Errorf("Could not get a created transcript. err: %v", err)
		return nil, err
	}
	log.WithField("transcript", res).Debugf("Created a new transcript. transcript_id: %s, transcribe_id: %s", res.ID, res.TranscribeID)

	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, transcript.EventTypeTranscriptCreated, res)

	return res, nil
}

// Gets returns list of transcripts.
func (h *transcriptHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*transcript.Transcript, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Gets",
		"filters": filters,
	})

	res, err := h.db.TranscriptGets(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get transcripts. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Delete deletes the transcript
func (h *transcriptHandler) Delete(ctx context.Context, id uuid.UUID) (*transcript.Transcript, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Delete",
		"transcript_id": id,
	})

	// get transcript
	tr, err := h.dbGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get transcript info. err: %v", err)
		return nil, err
	}

	if tr.TMDelete != dbhandler.DefaultTimeStamp {
		// already deleted
		return tr, nil
	}

	res, err := h.dbDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the transcript. err: %v", err)
		return nil, err
	}

	return res, nil
}
