package transcripthandler

import (
	"context"
	"fmt"
	"regexp"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/models/transcript"
)

// Recording transcribe the recoring
func (h *transcriptHandler) Recording(ctx context.Context, customerID uuid.UUID, transcribeID uuid.UUID, recordingID uuid.UUID, language string) ([]*transcript.Transcript, error) {
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

		direction := parseDirection(file.Filename)

		bucketPath := fmt.Sprintf("gs://%s/%s", file.BucketName, file.Filepath)
		tmps, err := h.processFromRecording(ctx, bucketPath, language, direction)
		if err != nil {
			return nil, errors.Wrapf(err, "could not transcribe the recording. recording_id: %s", recordingID)
		}
		log.Debugf("Transcripted the recording. transcribe_id: %s, len: %d", transcribeID, len(tmps))

		for _, tmp := range tmps {
			t, err := h.Create(ctx, customerID, transcribeID, direction, tmp.Message, tmp.TMTranscript)
			if err != nil {
				// we could not create transcript here, but we should not return an error
				log.Errorf("Could not create a tracript. message: %s, err: %v", tmp.Message, err)
			}
			log.WithField("transcript", t).Debugf("Created a new transcript. transcript_id: %s, transcribe_id: %s", t.ID, t.TranscribeID)
		}
	}

	return nil, fmt.Errorf("test Error")
}

func parseDirection(filename string) transcript.Direction {
	// Adjust regex to handle any file extension (e.g., .mp3, .ogg, .wav)
	re := regexp.MustCompile(`_(in|out)\.[a-zA-Z0-9]+$`)
	match := re.FindStringSubmatch(filename)

	if len(match) < 2 {
		return transcript.DirectionBoth
	}

	switch match[1] {
	case "in":
		return transcript.DirectionIn
	case "out":
		return transcript.DirectionOut
	default:
		return transcript.DirectionBoth
	}
}
