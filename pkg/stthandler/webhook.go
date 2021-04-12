package stthandler

import (
	"encoding/json"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/stt-manager.git/models/stt"
	"gitlab.com/voipbin/bin-manager/stt-manager.git/pkg/requesthandler"
)

// sendWebhook send the stt webhook.
func (h *sttHandler) sendWebhook(s *stt.STT) error {

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
