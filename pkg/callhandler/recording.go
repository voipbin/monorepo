package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
)

// RecordingGets returns list of recordings
func (h *callHandler) RecordingGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*recording.Recording, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "RecordingGets",
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
