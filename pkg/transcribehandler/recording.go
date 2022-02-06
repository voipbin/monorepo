package transcribehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/common"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
)

// Recording transcribe the recoring and send the webhook
func (h *transcribeHandler) Recording(ctx context.Context, customerID uuid.UUID, recordingID uuid.UUID, language string) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "Recording",
			"recording_id": recordingID,
		},
	)

	lang := getBCP47LanguageCode(language)
	log.Debugf("Parsed BCP47 language code. lang: %s", lang)

	// transcribe the recording
	tmp, err := h.sttGoogle.Recording(ctx, recordingID, lang)
	if err != nil {
		log.Errorf("Coudl not transcribe the recording. err: %v", err)
		return nil, err
	}
	transcripts := []transcript.Transcript{*tmp}

	// create the transcribe
	res, err := h.Create(ctx, customerID, recordingID, transcribe.TypeRecording, lang, common.DirectionBoth, transcripts)
	if err != nil {
		log.Errorf("Could not create the transcribe. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, transcribe.EventTypeTranscribeCreated, res)

	return res, nil
}
