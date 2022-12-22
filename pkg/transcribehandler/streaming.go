package transcribehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/streaming"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
)

// streamingTranscribeStart starts the streaming transcribe
func (h *transcribeHandler) streamingTranscribeStart(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType transcribe.ReferenceType,
	referenceID uuid.UUID,
	language string,
	direction transcribe.Direction,
) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":           "streamingTranscribeStart",
			"reference_type": referenceType,
			"reference_id":   referenceID,
			"language":       language,
			"direction":      direction,
		},
	)

	// create transcribing
	res, err := h.Create(
		ctx,
		customerID,
		referenceType,
		referenceID,
		language,
		direction,
	)
	if err != nil {
		log.Errorf("Could not create the transcribe. err: %v", err)
		return nil, err
	}
	log.WithField("transcribe", res).Debugf("Created transcribe. transcribe_id: %s", res.ID)

	// start transcript streaming
	streamings := []*streaming.Streaming{}

	directions := []transcript.Direction{transcript.Direction(direction)}
	if direction == transcribe.DirectionBoth {
		directions = []transcript.Direction{transcript.DirectionIn, transcript.DirectionOut}
	}

	for _, dir := range directions {

		// currently, we hanve only google's stt
		st, err := h.transcriptHandler.Start(ctx, res, dir)
		if err != nil {
			log.Errorf("Could not start the streaming stt. direction: %s, err: %v", dir, err)
			return nil, err
		}
		streamings = append(streamings, st)
	}
	h.addTranscribeStreamings(res.ID, streamings)

	return res, nil
}

// streamingTranscribeStop stops streaming transcribe.
func (h *transcribeHandler) streamingTranscribeStop(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "streamingTranscribeStop",
			"transcribe_id": id,
		},
	)

	// get streamings
	streamings := h.getTranscribeStreamings(id)

	// stop and delete the streamings
	for _, st := range streamings {
		if errStop := h.transcriptHandler.Stop(ctx, st); errStop != nil {
			log.Errorf("Could not stop the streaming. err: %v", errStop)
		}
	}
	h.deleteTranscribeStreamings(id)

	res, err := h.UpdateStatus(ctx, id, transcribe.StatusDone)
	if err != nil {
		log.Errorf("Could not update the status. err: %v", err)
		return nil, err
	}
	log.WithField("transcribe", res).Debugf("Updated transcribe status done. transcribe_id: %s", id)

	return res, nil
}
