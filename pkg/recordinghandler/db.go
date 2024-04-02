package recordinghandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
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
) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Start",
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})

	switch referenceType {
	case recording.ReferenceTypeCall:
		return h.recordingReferenceTypeCall(ctx, referenceID, format, endOfSilence, endOfKey, duration)

	case recording.ReferenceTypeConfbridge:
		return h.recordingReferenceTypeConfbridge(ctx, referenceID, format, endOfSilence, endOfKey, duration)

	default:
		log.Errorf("Unimplemented reference type. reference_type: %s, reference_id: %s", referenceType, referenceID)
		return nil, fmt.Errorf("unsupported reference type")
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

// Gets returns list of recordings of the given filters
func (h *recordingHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Gets",
		"filters": filters,
	})

	res, err := h.db.RecordingGets(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get reocordings. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns the recording info
func (h *recordingHandler) Get(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Get",
		"recording_id": id,
	})

	res, err := h.db.RecordingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get recording info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetByRecordingName returns the recording info of the given recording name
func (h *recordingHandler) GetByRecordingName(ctx context.Context, recordingName string) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "GetByRecordingName",
		"recording_name": recordingName,
	})

	res, err := h.db.RecordingGetByRecordingName(ctx, recordingName)
	if err != nil {
		log.Errorf("Could not get recording info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Stop stops the recording
func (h *recordingHandler) Stop(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Stop",
		"recording_id": id,
	})

	r, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get recording info. err: %v", err)
		return nil, err
	}

	switch r.ReferenceType {
	case recording.ReferenceTypeCall:
		err = h.stopReferenceTypeCall(ctx, r)

	case recording.ReferenceTypeConfbridge:
		err = h.stopReferenceTypeConference(ctx, r)

	default:
		log.Errorf("Unsupported reference type. reference_type: %s", r.ReferenceType)
		return nil, fmt.Errorf("unsupported reference type")
	}
	if err != nil {
		log.Errorf("Could not stop the recording. reference_type: %s, reference_id: %s", r.ReferenceType, r.ReferenceID)
		return nil, err
	}

	if errStatus := h.db.RecordingSetStatus(ctx, r.ID, recording.StatusStopping); errStatus != nil {
		log.Errorf("Could not update the status. err: %v", errStatus)
		return nil, errStatus
	}

	res, err := h.db.RecordingGet(ctx, r.ID)
	if err != nil {
		log.Errorf("Could not get updated recording info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// stopReferenceTypeCall stops the reference type call recording.
func (h *recordingHandler) stopReferenceTypeCall(ctx context.Context, r *recording.Recording) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "stopReferenceTypeCall",
		"recording_id": r.ID,
	})

	for _, channelID := range r.ChannelIDs {
		// hangup the channel
		log.WithField("channel_id", channelID).Debugf("Hanging up the recording channel. channel_id: %s", channelID)
		if errHangup := h.reqHandler.AstChannelHangup(ctx, r.AsteriskID, channelID, ari.ChannelCauseNormalClearing, 0); errHangup != nil {
			log.Errorf("Could not hangup the recording channel. err: %v", errHangup)
		}
	}

	return nil
}

// stopReferenceTypeConference stops the reference type conference recording.
func (h *recordingHandler) stopReferenceTypeConference(ctx context.Context, r *recording.Recording) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "stopReferenceTypeConference",
		"recording_id": r.ID,
	})

	filename := fmt.Sprintf("%s_in", r.RecordingName)
	log.Debugf("Stopping conference recording. recording_name: %s", filename)
	if errStop := h.reqHandler.AstRecordingStop(ctx, r.AsteriskID, filename); errStop != nil {
		log.Errorf("Could not stop the recording. err: %v", errStop)
		return errStop
	}

	return nil
}

// Stopped handels stopped recording
func (h *recordingHandler) Stopped(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Stopped",
		"recording_id": id,
	})

	// update recording status
	if errStatus := h.db.RecordingSetStatus(ctx, id, recording.StatusEnded); errStatus != nil {
		log.Errorf("Could not update recording status. err: %v", errStatus)
		return nil, errStatus
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get recording info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, recording.EventTypeRecordingFinished, res)

	return res, nil
}

// Delete deletes recording
func (h *recordingHandler) Delete(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Delete",
		"recording_id": id,
	})

	if errDelete := h.db.RecordingDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not get reocording. err: %v", errDelete)
		return nil, errDelete
	}

	go func() {
		// send request to delete recording files
		log.Debugf("Deleting recording files. recording_id: %s", id)
		if errDelete := h.reqHandler.StorageV1RecordingDelete(ctx, id); errDelete != nil {
			log.Errorf("Could not delete the recording files. err: %v", errDelete)
			return
		}
	}()

	res, err := h.db.RecordingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted recording. err: %v", err)
		return nil, err
	}

	return res, nil
}
