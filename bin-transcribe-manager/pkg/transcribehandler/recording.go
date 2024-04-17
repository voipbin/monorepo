package transcribehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

// startRecording transcribe the recoring
// returns created transcribe
func (h *transcribeHandler) startRecording(ctx context.Context, customerID uuid.UUID, recordingID uuid.UUID, language string) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "startRecording",
			"recording_id": recordingID,
		},
	)

	// check if the given recording's transcribe is already exist
	tmp, err := h.GetByReferenceIDAndLanguage(ctx, recordingID, language)
	if err == nil {
		// we have a transcribe already
		log.Debugf("Found existing transcribe. transcribe_id: %s", tmp.ID)
		return tmp, nil
	}

	// create transcribing
	id := h.utilHandler.UUIDCreate()
	tr, err := h.Create(
		ctx,
		id,
		customerID,
		transcribe.ReferenceTypeRecording,
		recordingID,
		language,
		transcribe.DirectionBoth,
		[]uuid.UUID{},
	)
	if err != nil {
		log.Errorf("Could not create the transcribe. err: %v", err)
		return nil, err
	}
	log.WithField("transcribe", tr).Debugf("Created transcribe. transcribe_id: %s", tr.ID)

	// transcribe the recording
	trsc, err := h.transcriptHandler.Recording(ctx, customerID, tr.ID, recordingID, language)
	if err != nil {
		log.Errorf("Coudl not transcribe the recording. err: %v", err)
		return nil, err
	}
	log.WithField("transcript", trsc).Debugf("Transcripted the recording. transcribe_id: %s, transcript_id: %s", tr.ID, trsc.ID)

	// transcribe done
	res, err := h.UpdateStatus(ctx, tr.ID, transcribe.StatusDone)
	if err != nil {
		log.Errorf("Could not update the status. err: %v", err)
		return nil, err
	}

	return res, nil
}
