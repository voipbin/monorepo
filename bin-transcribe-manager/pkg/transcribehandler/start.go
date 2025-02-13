package transcribehandler

import (
	"context"
	"fmt"

	cmcall "monorepo/bin-call-manager/models/call"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
)

// Start starts a transcribe
func (h *transcribeHandler) Start(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType transcribe.ReferenceType,
	referenceID uuid.UUID,
	language string,
	direction transcribe.Direction,
) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Start",
		"customer_id":    customerID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"language":       language,
		"direction":      direction,
	})

	// check the reference is valid
	if valid := h.isValidReference(ctx, referenceType, referenceID); !valid {
		log.Errorf("The given reference info is not valid for transcribe.")
		return nil, fmt.Errorf("the given reference info is not valid for transcribe")
	}

	// parse the BCP47
	lang := getBCP47LanguageCode(language)
	log.Debugf("Parsed BCP47 language code. lang: %s", lang)

	var res *transcribe.Transcribe
	var err error
	switch referenceType {
	case transcribe.ReferenceTypeRecording:
		res, err = h.startRecording(ctx, customerID, referenceID, lang)
		if err != nil {
			log.Errorf("Could not transcribe the recording reference type. err: %v", err)
			return nil, err
		}

	case transcribe.ReferenceTypeCall, transcribe.ReferenceTypeConfbridge:
		res, err = h.startLive(ctx, customerID, referenceType, referenceID, lang, direction)
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
			log.Errorf("Could not get reference info. type: %s, err: %v", referenceType, err)
			return false
		}
		if tmp.Status != cmcall.StatusProgressing {
			return false
		}

	case transcribe.ReferenceTypeConfbridge:
		tmp, err := h.reqHandler.CallV1ConfbridgeGet(ctx, referenceID)
		if err != nil {
			log.Errorf("Could not get reference info. type: %s, err: %v", referenceType, err)
			return false
		}
		if tmp.TMDelete < dbhandler.DefaultTimeStamp {
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

// startLive starts the streaming transcribe
func (h *transcribeHandler) startLive(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType transcribe.ReferenceType,
	referenceID uuid.UUID,
	language string,
	direction transcribe.Direction,
) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "startLive",
		"customer_id":    customerID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"language":       language,
		"direction":      direction,
	})

	// create transcribe id
	id := h.utilHandler.UUIDCreate()
	log = log.WithField("transcribe_id", id)

	directions := []transcript.Direction{transcript.Direction(direction)}
	if direction == transcribe.DirectionBoth {
		directions = []transcript.Direction{transcript.DirectionIn, transcript.DirectionOut}
	}

	// start streamings
	streamingIDs := []uuid.UUID{}
	for _, dr := range directions {

		// start the streaming transcribe
		st, err := h.streamingHandler.Start(ctx, customerID, id, referenceType, referenceID, language, dr)
		if err != nil {
			log.Errorf("Could not start the streaming stt. direction: %s, err: %v", dr, err)
			return nil, err
		}
		log.WithField("streaming", st).Debugf("Streaming started. streaming_id: %s", st.ID)

		streamingIDs = append(streamingIDs, st.ID)
	}

	// create transcribing
	res, err := h.Create(ctx, id, customerID, referenceType, referenceID, language, direction, streamingIDs)
	if err != nil {
		log.Errorf("Could not create the transcribe. err: %v", err)
		return nil, err
	}
	log.WithField("transcribe", res).Debugf("Created transcribe. transcribe_id: %s", res.ID)

	return res, nil
}
