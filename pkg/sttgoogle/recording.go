package sttgoogle

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
)

// Recording transcribe the recoring
func (h *streamingHandler) Recording(ctx context.Context, recordingID uuid.UUID, language string) (*transcript.Transcript, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "Recording",
			"recording_id": recordingID,
			"language":     language,
		},
	)

	// send a request to storage-manager to get a media link
	recording, err := h.reqHandler.StorageV1RecordingGet(ctx, recordingID)
	if err != nil {
		log.Errorf("Could not get recording info. err: %v", err)
		return nil, err
	}

	// transcribe
	res, err := h.transcribeFromBucket(ctx, recording.BucketURI, language)
	if err != nil {
		log.Errorf("Could not transcribe the recording. err: %v", err)
		return nil, err
	}

	return res, nil
}
