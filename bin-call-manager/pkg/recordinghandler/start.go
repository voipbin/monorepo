package recordinghandler

import (
	"context"
	"fmt"
	"monorepo/bin-call-manager/models/recording"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// Start start the recording of the given reference info
// duration: milliseconds
func (h *recordingHandler) Start(
	ctx context.Context,
	referenceType recording.ReferenceType,
	referenceID uuid.UUID,
	format recording.Format,
	endOfSilence int,
	endOfKey string,
	duration int,
	onEndFlowID uuid.UUID,
) (*recording.Recording, error) {

	switch referenceType {
	case recording.ReferenceTypeCall:
		return h.recordingReferenceTypeCall(ctx, referenceID, format, endOfSilence, endOfKey, duration, onEndFlowID)

	case recording.ReferenceTypeConfbridge:
		return h.recordingReferenceTypeConfbridge(ctx, referenceID, format, endOfSilence, endOfKey, duration, onEndFlowID)

	default:
		return nil, fmt.Errorf("unsupported reference type. reference_type: %s, reference_id: %s", referenceType, referenceID)
	}
}

// Started updates recording's status to the recording and notify the event
func (h *recordingHandler) Started(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Started",
		"recording_id": id,
	})
	if errStatus := h.db.RecordingSetStatus(ctx, id, recording.StatusRecording); errStatus != nil {
		log.Errorf("Could not update the recording status. err: %v", errStatus)
		return nil, errStatus
	}

	res, err := h.db.RecordingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the updated recording info. err: %v", err)
		return nil, err
	}

	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, recording.EventTypeRecordingStarted, res)
	return res, nil
}
