package transcripthandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/models/transcript"
)

// Recording transcribe the recoring
func (h *transcriptHandler) Recording(ctx context.Context, customerID uuid.UUID, transcribeID uuid.UUID, recordingID uuid.UUID, language string) (*transcript.Transcript, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "Recording",
			"transcribe_id": transcribeID,
			"recording_id":  recordingID,
			"language":      language,
		},
	)

	// send a request to storage-manager to get a media link
	recording, err := h.reqHandler.StorageV1RecordingGet(ctx, recordingID, defaultBucketTimeout)
	if err != nil {
		log.Errorf("Could not get recording info. err: %v", err)
		return nil, err
	}

	// transcribe
	tmp, err := h.processFromBucket(ctx, recording.BucketURI, language)
	if err != nil {
		log.Errorf("Could not transcribe the recording. err: %v", err)
		return nil, err
	}

	// create
	ts := "0000-00-00 00:00:00.00000"
	res, err := h.Create(ctx, customerID, transcribeID, transcript.DirectionBoth, tmp.Message, ts)
	if err != nil {
		log.Errorf("Could not create the transcript. err: %v", err)
		return nil, err
	}

	return res, nil
}
