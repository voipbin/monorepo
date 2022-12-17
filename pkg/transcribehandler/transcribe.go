package transcribehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/common"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
)

// Get returns transcribe
func (h *transcribeHandler) Get(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Get",
		},
	)

	res, err := h.db.TranscribeGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Create creates a new transcribe
func (h *transcribeHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	referenceID uuid.UUID,
	transType transcribe.Type,
	language string,
	direction common.Direction,
	transcripts []transcript.Transcript,
) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Create",
		},
	)

	tmpTranscripts := transcripts
	if tmpTranscripts == nil {
		tmpTranscripts = []transcript.Transcript{}
	}

	tr := &transcribe.Transcribe{
		ID:          uuid.Must(uuid.NewV4()),
		CustomerID:  customerID,
		Type:        transType,
		ReferenceID: referenceID,

		HostID:    h.hostID,
		Language:  language,
		Direction: direction,

		Transcripts: tmpTranscripts,
	}

	if err := h.db.TranscribeCreate(ctx, tr); err != nil {
		log.Errorf("Could not create transcribe. err: %v", err)
		return nil, err
	}
	log.WithField("transcribe", tr).Debugf("Created a new transcribe. transcribe_id: %s, reference_id: %s", tr.ID, tr.ReferenceID)

	res, err := h.db.TranscribeGet(ctx, tr.ID)
	if err != nil {
		log.Errorf("Could not get created transcribe. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, transcribe.EventTypeTranscribeCreated, res)

	return res, nil
}

// TranscribeGet returns transcribe
func (h *transcribeHandler) Delete(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Delete",
		},
	)

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
