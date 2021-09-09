package transcribehandler

import (
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

func (h *transcribeHandler) CallRecording(callID uuid.UUID, language, webhookURI, webhookMethod string) error {

	// get call info
	c, err := h.reqHandler.CMCallGet(callID)
	if err != nil {
		return err
	}

	for _, recordingID := range c.RecordingIDs {

		// do transcribe recording
		tmp, err := h.transcribeRecording(recordingID, language)
		if err != nil {
			logrus.Errorf("Coudl not convert to text. err: %v", err)
			continue
		}

		s := &transcribe.Transcribe{
			ID:            uuid.Must(uuid.NewV4()),
			Type:          transcribe.TypeRecording,
			ReferenceID:   recordingID,
			Language:      language,
			WebhookURI:    webhookURI,
			WebhookMethod: webhookMethod,
			Transcripts:   []transcribe.Transcript{*tmp},
		}

		// send webhook
		go func() {
			if err := h.sendWebhook(transcribeEventTranscript, s); err != nil {
				logrus.Errorf("Could not send the webhook correctly. err: %v", err)
			}
		}()
	}

	return nil
}
