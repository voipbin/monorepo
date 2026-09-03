package transcribehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-transcribe-manager/models/transcribe"
)

// startRecording transcribe the recoring
// returns created transcribe
//
// id is the already-resolved id to create with (either server-generated or
// caller-supplied); callerSpecifiedID indicates which, and is required
// because id alone is always non-nil by the time this is called (Start
// resolves it before dispatch) -- see the design doc's B10 correction.
func (h *transcribeHandler) startRecording(ctx context.Context, id uuid.UUID, callerSpecifiedID bool, customerID uuid.UUID, activeflowID uuid.UUID, onEndFlowID uuid.UUID, recordingID uuid.UUID, language string, provider transcribe.Provider) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "startRecording",
		"recording_id": recordingID,
	})

	// check if the given recording's transcribe is already exist
	tmp, err := h.GetByReferenceIDAndLanguage(ctx, recordingID, language)
	if err == nil {
		if callerSpecifiedID {
			// Start's pre-check guarantees a caller-specified id isn't already
			// in use, so a row found here by reference+language is guaranteed
			// to have a different id -- this is always a conflict, never a
			// match on the id itself.
			return nil, cerrors.FailedPrecondition(
				commonoutline.ServiceNameTranscribeManager,
				"TRANSCRIBE_ALREADY_EXISTS_DIFFERENT_ID",
				"A transcribe for this recording/language already exists with a different id.",
			)
		}

		// we have a transcribe already
		log.Debugf("Found existing transcribe. transcribe_id: %s", tmp.ID)
		return tmp, nil
	}

	// create transcribing
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
		provider,
		[]uuid.UUID{},
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the transcribe.")
	}
	log.WithField("transcribe", tr).Debugf("Created transcribe. transcribe_id: %s", tr.ID)

	// transcribe the recording
	transcripts, err := h.transcriptHandler.Recording(ctx, customerID, tr.ID, recordingID, language)
	if err != nil {
		return nil, errors.Wrapf(err, "could not transcribe the recording.")
	}
	log.Debugf("Transcripted the recording. transcribe_id: %s, len: %d", tr.ID, len(transcripts))

	// transcribe done
	res, err := h.UpdateStatus(ctx, tr.ID, transcribe.StatusDone)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update the status.")
	}

	return res, nil
}
