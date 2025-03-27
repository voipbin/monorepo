package transcribehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/models/transcribe"
)

// startRecording transcribe the recoring
// returns created transcribe
func (h *transcribeHandler) startRecording(ctx context.Context, customerID uuid.UUID, activeflowID uuid.UUID, onEndFlowID uuid.UUID, recordingID uuid.UUID, language string) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "startRecording",
		"recording_id": recordingID,
	})

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
		activeflowID,
		onEndFlowID,
		transcribe.ReferenceTypeRecording,
		recordingID,
		language,
		transcribe.DirectionBoth,
		[]uuid.UUID{},
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the transcribe.")
	}
	log.WithField("transcribe", tr).Debugf("Created transcribe. transcribe_id: %s", tr.ID)

	// transcribe the recording
	trsc, err := h.transcriptHandler.Recording(ctx, customerID, tr.ID, recordingID, language)
	if err != nil {
		return nil, errors.Wrapf(err, "could not transcribe the recording.")
	}
	log.WithField("transcript", trsc).Debugf("Transcripted the recording. transcribe_id: %s, transcript_id: %s", tr.ID, trsc.ID)

	// transcribe done
	res, err := h.UpdateStatus(ctx, tr.ID, transcribe.StatusDone)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update the status.")
	}

	return res, nil
}
