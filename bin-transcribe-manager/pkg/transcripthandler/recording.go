package transcripthandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/models/transcript"
)

// Recording transcribe the recoring
func (h *transcriptHandler) Recording(ctx context.Context, customerID uuid.UUID, transcribeID uuid.UUID, recordingID uuid.UUID, language string) (*transcript.Transcript, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Recording",
		"transcribe_id": transcribeID,
		"recording_id":  recordingID,
		"language":      language,
	})

	filters := map[string]string{
		"deleted":      "false",
		"reference_id": recordingID.String(),
	}
	files, err := h.reqHandler.StorageV1FileGets(ctx, "", 100, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the files. recording_id: %s", recordingID)
	}
	log.WithField("files", files).Debugf("Got the files. recording_id: %s", recordingID)

	for _, file := range files {
		tmp, err := h.processFromBucket(ctx, file.URIBucket, language)
		if err != nil {
			return nil, errors.Wrapf(err, "could not transcribe the recording. recording_id: %s", recordingID)
		}

		log.WithField("transcript", tmp).Debugf("Transcripted the recording. transcribe_id: %s, transcript_id: %s", transcribeID, tmp.ID)
	}

	return nil, fmt.Errorf("test Error")

	// // send a request to storage-manager to get a media link
	// recording, err := h.reqHandler.StorageV1RecordingGet(ctx, recordingID, defaultBucketTimeout)
	// if err != nil {
	// 	return nil, errors.Wrapf(err, "could not get recording info. recording_id: %s", recordingID)
	// }

	// // transcribe
	// tmp, err := h.processFromBucket(ctx, recording.BucketURI, language)
	// if err != nil {
	// 	return nil, errors.Wrapf(err, "could not transcribe the recording. recording_id: %s", recordingID)
	// }

	// // create
	// ts := "0000-00-00 00:00:00.00000"
	// res, err := h.Create(ctx, customerID, transcribeID, transcript.DirectionBoth, tmp.Message, ts)
	// if err != nil {
	// 	return nil, errors.Wrapf(err, "could not create the transcript.")
	// }

	// return res, nil
}
