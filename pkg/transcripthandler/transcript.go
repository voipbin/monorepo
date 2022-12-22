package transcripthandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
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
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "Create",
			"customer_id":   customerID,
			"transcribe_id": transcribeID,
		},
	)

	id := h.utilHandler.CreateUUID()
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
func (h *transcriptHandler) Gets(ctx context.Context, transcribeID uuid.UUID) ([]*transcript.Transcript, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "Gets",
			"transcribe_id": transcribeID,
		},
	)

	res, err := h.db.TranscriptGetsByTranscribeID(ctx, transcribeID)
	if err != nil {
		log.Errorf("Could not get transcripts. err: %v", err)
		return nil, err
	}

	return res, nil
}
