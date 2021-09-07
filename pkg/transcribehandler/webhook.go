package transcribehandler

import (
	"encoding/json"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/requesthandler"
)

// sendWebhook send the transcript result via webhook.
func (h *transcribeHandler) sendWebhook(s *transcribe.Transcribe) error {

	d, err := json.Marshal(s)
	if err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return err
	}

	go func() {
		if err := h.reqHandler.WMWebhookPost(s.WebhookMethod, s.WebhookURI, requesthandler.ContentTypeJSON, d); err != nil {
			logrus.Errorf("Could not send the webhoook. err: %v", err)
			return
		}
	}()

	return nil
}

// sendWebhook send the transcript result via webhook.
func (h *transcribeHandler) sendWebhookTranscript(s *transcribe.Transcribe, t *transcribe.Transcript) error {

	log := logrus.WithFields(logrus.Fields{
		"func":          "sendWebhookTranscript",
		"transcribe_id": s.ID,
	})

	tmp := &transcribe.TranscriptWebhook{
		ID:          s.ID,
		Type:        s.Type,
		ReferenceID: s.ReferenceID,
		Transcript:  *t,
	}

	d, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the data. transcript: %v, err: %v", t, err)
		return err
	}

	go func() {
		if err := h.reqHandler.WMWebhookPost(s.WebhookMethod, s.WebhookURI, requesthandler.ContentTypeJSON, d); err != nil {
			log.Errorf("Could not send the webhoook. transcript: %v, err: %v", t, err)
			return
		}
	}()

	return nil
}
