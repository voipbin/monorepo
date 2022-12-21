package transcribehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/common"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/streaming"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

// StreamingTranscribeStart starts the streaming transcribe
func (h *transcribeHandler) StreamingTranscribeStart(ctx context.Context, customerID uuid.UUID, referenceID uuid.UUID, transType transcribe.Type, language string) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "StreamingTranscribeStart",
			"reference_id": referenceID,
			"type":         transType,
		},
	)
	lang := getBCP47LanguageCode(language)
	log.Debugf("Parsed BCP47 language code. lang: %s", lang)

	tr, err := h.Create(ctx, customerID, referenceID, transType, lang, common.DirectionBoth, nil)
	if err != nil {
		log.Errorf("Could not create transcribe. err: %v", err)
		return nil, err
	}
	log.WithField("transcribe", tr).Debugf("Created transcribe. transcribe_id: %s", tr.ID)

	h.notifyHandler.PublishWebhookEvent(ctx, tr.CustomerID, transcribe.EventTypeTranscribeStarted, tr)

	// create streamings
	streamings := []*streaming.Streaming{}
	for _, dir := range []common.Direction{common.DirectionIn, common.DirectionOut} {

		// currently, we hanve only google's stt
		st, err := h.sttGoogle.Start(ctx, tr, dir)
		if err != nil {
			log.Errorf("Could not start the streaming stt. direction: %s, err: %v", dir, err)
			return nil, err
		}
		streamings = append(streamings, st)
	}
	h.addTranscribeStreamings(tr.ID, streamings)

	return tr, nil
}

// StreamingTranscribeStop stops streaming transcribe.
func (h *transcribeHandler) StreamingTranscribeStop(ctx context.Context, transcribeID uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "StreamingTranscribeStop",
			"transcribe_id": transcribeID,
		},
	)

	// get streamings
	streamings := h.getTranscribeStreamings(transcribeID)

	// stop and delete the streamings
	for _, st := range streamings {
		if errStop := h.sttGoogle.Stop(ctx, st); errStop != nil {
			log.Errorf("Could not stop the streaming. err: %v", errStop)
		}
	}
	h.deleteTranscribeStreamings(transcribeID)

	res, err := h.db.TranscribeGet(ctx, transcribeID)
	if err != nil {
		log.Errorf("Could not get deleted transcribe. err: %v", err)
		return err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, transcribe.EventTypeTranscribeStopped, res)

	return nil
}
