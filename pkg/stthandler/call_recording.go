package stthandler

import (
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/stt-manager.git/models/stt"
)

func (h *sttHandler) CallRecording(callID uuid.UUID, language, webhookURI, webhookMethod string) error {

	// get call info
	c, err := h.reqHandler.CMCallGet(callID)
	if err != nil {
		return err
	}

	for _, recordingID := range c.RecordingIDs {

		// do stt recording
		tmp, err := h.transcribeRecording(recordingID, language)
		if err != nil {
			logrus.Errorf("Coudl not convert to text. err: %v", err)
			continue
		}

		s := &stt.STT{
			ID:            uuid.Must(uuid.NewV4()),
			Type:          stt.TypeRecording,
			ReferenceID:   recordingID,
			Language:      language,
			WebhookURI:    webhookURI,
			WebhookMethod: webhookMethod,
			Transcript:    tmp,
		}

		// send webhook
		go func() {
			if err := h.sendWebhook(s); err != nil {
				logrus.Errorf("Could not send the webhook correctly. err: %v", err)
			}
		}()
	}

	return nil
}
