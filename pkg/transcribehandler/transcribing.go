package transcribehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

// TranscribingStart start a live transcribe
func (h *transcribeHandler) TranscribingStart(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType transcribe.ReferenceType,
	referenceID uuid.UUID,
	language string,
	direction transcribe.Direction,
) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":           "TranscribingStart",
			"reference_type": referenceType,
			"reference_id":   referenceID,
		},
	)

	// check the reference is valid
	if valid := h.isValidReference(ctx, referenceType, referenceID); valid != true {
		log.Errorf("The given reference info is not valid for transcribe.")
		return nil, fmt.Errorf("the given reference info is not valid for transcribe")
	}

	// parse the BCP47
	lang := getBCP47LanguageCode(language)
	log.Debugf("Parsed BCP47 language code. lang: %s", lang)

	var res *transcribe.Transcribe = nil
	var err error
	switch referenceType {
	case transcribe.ReferenceTypeRecording:
		res, err = h.Recording(ctx, customerID, referenceID, lang)
		if err != nil {
			log.Errorf("Could not transcribe the recording reference type. err: %v", err)
			return nil, err
		}

	case transcribe.ReferenceTypeCall:
		res, err = h.streamingTranscribeStart(ctx, customerID, referenceType, referenceID, lang, direction)
		if err != nil {
			log.Errorf("Could not transcribe the call reference type. err: %v", err)
			return nil, err
		}

	default:
		log.Errorf("Unsupported reference type. reference_type: %s", referenceType)
		return nil, fmt.Errorf("unsupported reference type. reference_type: %s", referenceType)
	}

	return res, nil
}

// TranscribingStop stops the progressing transcribe process.
func (h *transcribeHandler) TranscribingStop(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "TranscribingStop",
			"transcribe_id": id,
		},
	)

	// get transcribe and evaluate
	tmp, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get transcribe. err: %v", err)
		return nil, err
	}

	if tmp.Status != transcribe.StatusProgressing {
		log.Errorf("Invalid status. old_status: %s, new_status: %s", tmp.Status, transcribe.StatusDone)
		return nil, fmt.Errorf("invalid status")
	}

	switch tmp.ReferenceType {
	case transcribe.ReferenceTypeCall:
		return h.streamingTranscribeStop(ctx, tmp.ID)

	case transcribe.ReferenceTypeConference:
		log.Errorf("Not implemented reference type. reference_type: %s", tmp.ReferenceType)
		return nil, fmt.Errorf("Unimplemented reference type")

	default:
		log.Errorf("Invalid reference type. reference_type: %s", tmp.ReferenceType)
		return nil, fmt.Errorf("invalid reference type")
	}
}

// isValidReference returns false if the given reference is not valid for transcribe.
func (h *transcribeHandler) isValidReference(ctx context.Context, referenceType transcribe.ReferenceType, referenceID uuid.UUID) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":           "isValidReference",
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})

	// check the reference is valid
	switch referenceType {
	case transcribe.ReferenceTypeCall:
		tmp, err := h.reqHandler.CallV1CallGet(ctx, referenceID)
		if err != nil {
			log.Errorf("Could not get reference info. type: %s", referenceType)
			return false
		}
		if tmp.Status != cmcall.StatusProgressing {
			return false
		}

	case transcribe.ReferenceTypeRecording:
		_, err := h.reqHandler.CallV1RecordingGet(ctx, referenceID)
		if err != nil {
			log.Errorf("Could not get reference info. type: %s", referenceType)
			return false
		}

	default:
		log.Errorf("Unsupported reference type. reference_type: %s", referenceType)
		return false
	}

	return true
}
