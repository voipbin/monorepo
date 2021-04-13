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

	if err := h.reqHandler.WMWebhookPost(s.WebhookMethod, s.WebhookURI, requesthandler.ContentTypeJSON, d); err != nil {
		logrus.Errorf("Could not send the webhoook. err: %v", err)
		return err
	}

	return nil
}
