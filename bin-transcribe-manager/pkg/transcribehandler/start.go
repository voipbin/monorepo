package transcribehandler

import (
	"context"
	"fmt"

	cmcall "monorepo/bin-call-manager/models/call"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
)

// Start starts a transcribe
func (h *transcribeHandler) Start(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	onEndFlowID uuid.UUID,
	referenceType transcribe.ReferenceType,
	referenceID uuid.UUID,
	language string,
	direction transcribe.Direction,
	provider transcribe.Provider,
) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Start",
		"customer_id":    customerID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"language":       language,
		"direction":      direction,
		"provider":       provider,
	})

	// Normalize the direction. This is the authoritative guard: it protects all
	// callers (flow-manager, transcribe-control, and the api-manager REST path),
	// so an empty or invalid direction (e.g. a typo) is coerced to DirectionBoth
	// instead of flowing into the streaming/snoop layer and failing at runtime.
	// DirectionBoth is the safe catch-all default and matches the long-standing
	// behavior where an omitted direction captured both legs, so this is
	// backward compatible. An empty direction is the "use default" signal and is
	// not warned about; only a non-empty unknown value is logged as invalid.
	// Note: matching is case-sensitive (no trim/case-fold). The direction enum is
	// exposed only in lowercase (both/in/out), so a mismatched case is treated as
	// an invalid value rather than silently accepted.
	if normalized := direction.Normalize(); normalized != direction {
		if direction != "" {
			log.Warnf("Invalid direction. Falling back to both. direction: %s", direction)
		}
		direction = normalized
	}

	// check the reference is valid
	if valid := h.isValidReference(ctx, referenceType, referenceID); !valid {
		return nil, fmt.Errorf("the given reference info is not valid for transcribe. reference_type: %s, reference_id: %s", referenceType, referenceID)
	}

	// parse the BCP47
	lang := getBCP47LanguageCode(language)
	log.Debugf("Parsed BCP47 language code. lang: %s", lang)

	var res *transcribe.Transcribe
	var err error
	switch referenceType {
	case transcribe.ReferenceTypeRecording:
		res, err = h.startRecording(ctx, customerID, activeflowID, onEndFlowID, referenceID, lang, provider)
		if err != nil {
			return nil, errors.Wrapf(err, "could not start the recording transcribe. reference_id: %s", referenceID)
		}

	case transcribe.ReferenceTypeCall, transcribe.ReferenceTypeConfbridge:
		res, err = h.startLive(ctx, customerID, activeflowID, onEndFlowID, referenceType, referenceID, lang, direction, provider)
		if err != nil {
			return nil, errors.Wrapf(err, "could not start the live transcribe. reference_id: %s", referenceID)
		}

	default:
		return nil, errors.Wrapf(err, "unsupported reference type. reference_type: %s", referenceType)
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

		if tmp.Status != cmcall.StatusDialing && tmp.Status != cmcall.StatusRinging && tmp.Status != cmcall.StatusProgressing {
			log.Errorf("Call is not in a valid state for transcribe. status: %s", tmp.Status)
			return false
		}

		if tmp.TMDelete != nil {
			log.Errorf("Call is not valid for transcribe. tm_delete: %v", tmp.TMDelete)
			return false
		}

	case transcribe.ReferenceTypeConfbridge:
		tmp, err := h.reqHandler.CallV1ConfbridgeGet(ctx, referenceID)
		if err != nil {
			log.Errorf("Could not get reference info. type: %s, err: %v", referenceType, err)
			return false
		}
		if tmp.TMDelete != nil {
			log.Errorf("Confbridge is not valid for transcribe. tm_delete: %v", tmp.TMDelete)
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
	activeflowID uuid.UUID,
	onEndFlowID uuid.UUID,
	referenceType transcribe.ReferenceType,
	referenceID uuid.UUID,
	language string,
	direction transcribe.Direction,
	provider transcribe.Provider,
) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "startLive",
		"customer_id":    customerID,
		"activeflow_id":  activeflowID,
		"on_end_flow_id": onEndFlowID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"language":       language,
		"direction":      direction,
		"provider":       provider,
	})

	// create transcribe id
	id := h.utilHandler.UUIDCreate()
	log = log.WithField("transcribe_id", id)
	log.Debugf("Starting live transcribe. transcribe_id: %s", id)

	// Reject a duplicate live session: if a progressing transcribe already
	// exists for the same customer, reference and language, starting another
	// one would silently create parallel streaming sessions (double-click,
	// network retry, or two agents pressing Start simultaneously). The
	// recording path has its own dedup (startRecording returns the existing
	// transcribe); for live we conflict instead of returning the existing one
	// because the caller may have requested a different direction/provider,
	// and the existing session keeps streaming either way.
	//
	// Scoped per customer_id AND language on purpose:
	// - customer_id: bin-ai-manager starts its own summary transcribes on the
	//   SAME call/confbridge under IDAIManager ownership (see
	//   bin-ai-manager/pkg/summaryhandler/start.go) — those must keep
	//   coexisting with the customer's own live transcribe, and a customer
	//   must never be blocked by a hidden system-owned session. The
	//   double-start scenarios this guard exists for are all same-customer.
	// - language: multi-language sessions on one call remain possible.
	//
	// Note this is a read-then-create check, not a DB unique constraint, so a
	// true concurrent race can still slip through; it closes the practical
	// double-start window. Also, a transcribe stuck in progressing (e.g. an
	// orphaned session after a pod restart whose hangup event was lost) keeps
	// returning 409 for its reference until it is stopped or cleaned up —
	// acceptable for call-scoped lifetimes, but worth knowing when debugging.
	dupFilters := map[transcribe.Field]any{
		transcribe.FieldCustomerID:  customerID,
		transcribe.FieldReferenceID: referenceID,
		transcribe.FieldLanguage:    language,
		transcribe.FieldStatus:      transcribe.StatusProgressing,
		transcribe.FieldDeleted:     false,
	}
	existings, err := h.List(ctx, 1, "", dupFilters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not check for an existing live transcribe. reference_id: %s", referenceID)
	}
	if len(existings) > 0 {
		log.Infof("Found an existing progressing transcribe. Rejecting duplicate start. existing_transcribe_id: %s", existings[0].ID)
		return nil, cerrors.AlreadyExists(
			commonoutline.ServiceNameTranscribeManager,
			"TRANSCRIBE_ALREADY_PROGRESSING",
			"A live transcribe is already progressing for this reference and language. Stop the existing transcribe first, or start with a different language.",
		)
	}

	directions := []transcript.Direction{transcript.Direction(direction)}
	if direction == transcribe.DirectionBoth {
		directions = []transcript.Direction{transcript.DirectionIn, transcript.DirectionOut}
	}

	// start streamings
	streamingIDs := []uuid.UUID{}
	for _, dr := range directions {

		// start the streaming transcribe
		st, err := h.streamingHandler.Start(ctx, customerID, id, referenceType, referenceID, language, dr, provider)
		if err != nil {
			log.Errorf("Could not start the streaming stt. direction: %s, err: %v", dr, err)
			return nil, err
		}
		log.WithField("streaming", st).Debugf("Streaming started. streaming_id: %s", st.ID)

		streamingIDs = append(streamingIDs, st.ID)
	}

	// create transcribing
	res, err := h.Create(ctx, id, customerID, activeflowID, onEndFlowID, referenceType, referenceID, language, direction, provider, streamingIDs)
	if err != nil {
		log.Errorf("Could not create the transcribe. err: %v", err)
		return nil, err
	}
	log.WithField("transcribe", res).Debugf("Created transcribe. transcribe_id: %s", res.ID)

	return res, nil
}
