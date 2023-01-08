package callhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// RecordingCreate creates a new recording
func (h *callHandler) RecordingCreate(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	referenceType recording.ReferenceType,
	referenceID uuid.UUID,
	format string,
	recordingName string,
	filenames []string,
	asteriskID string,
	channelIDs []string,
) (*recording.Recording, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "RecordingCreate",
			"customer_id":  customerID,
			"recording_id": id,
		},
	)

	tmp := &recording.Recording{
		ID:         id,
		CustomerID: customerID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		Status:        recording.StatusInitiating,
		Format:        format,
		RecordingName: recordingName,
		Filenames:     filenames,

		AsteriskID: asteriskID,
		ChannelIDs: channelIDs,

		TMStart: dbhandler.DefaultTimeStamp,
		TMEnd:   dbhandler.DefaultTimeStamp,
	}

	if err := h.db.RecordingCreate(ctx, tmp); err != nil {
		log.Errorf("Could not create the record. err: %v", err)
		return nil, fmt.Errorf("could not create the record. err: %v", err)
	}

	res, err := h.db.RecordingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created reocordings. err: %v", err)
		return nil, err
	}

	return res, nil
}

// RecordingGets returns list of recordings
func (h *callHandler) RecordingGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*recording.Recording, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "RecordingGets",
			"customer_id": customerID,
		},
	)

	res, err := h.db.RecordingGets(ctx, customerID, size, token)
	if err != nil {
		log.Errorf("Could not get reocordings. err: %v", err)
		return nil, err
	}

	return res, nil
}

// RecordingGet returns recording
func (h *callHandler) RecordingGet(ctx context.Context, recordingID uuid.UUID) (*recording.Recording, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "RecordingGet",
			"recording_id": recordingID,
		},
	)

	res, err := h.db.RecordingGet(ctx, recordingID)
	if err != nil {
		log.Errorf("Could not get reocording. err: %v", err)
		return nil, err
	}

	return res, nil
}

// RecordingDelete deletes recording
func (h *callHandler) RecordingDelete(ctx context.Context, recordingID uuid.UUID) (*recording.Recording, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "RecordingDelete",
			"recording_id": recordingID,
		},
	)

	if errDelete := h.db.RecordingDelete(ctx, recordingID); errDelete != nil {
		log.Errorf("Could not get reocording. err: %v", errDelete)
		return nil, errDelete
	}

	go func() {
		// send request to delete recording files
		log.Debugf("Deleting recording files. recording_id: %s", recordingID)
		if errDelete := h.reqHandler.StorageV1RecordingDelete(ctx, recordingID); errDelete != nil {
			log.Errorf("Could not delete the recording files. err: %v", errDelete)
			return
		}
	}()

	res, err := h.db.RecordingGet(ctx, recordingID)
	if err != nil {
		log.Errorf("Could not get deleted recording. err: %v", err)
		return nil, err
	}

	return res, nil
}
