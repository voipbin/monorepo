package subscribehandler

import (
	"encoding/json"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/event"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/webhookhandler"
)

// processEventCMCallCommon handles the call-manager's call related event
func (h *subscribeHandler) processEventCMRecordingCommon(m *rabbitmqhandler.Event) error {

	log := logrus.WithFields(
		logrus.Fields{
			"event": m,
		},
	)
	log.Debugf("processEventCMRecordingCommon. Sending an event.")

	var evt recording.Recording
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		// same call-id is already exsit
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}
	log.WithFields(
		logrus.Fields{
			"event": evt,
		},
	).Debugf("Marshaled message.")

	if evt.WebhookURI == "" {
		// no webhook uri
		return nil
	}

	tmpMsg := event.Event{
		Type: m.Type,
		Data: m.Data,
	}
	tmp, err := json.Marshal(tmpMsg)
	if err != nil {
		return err
	}

	// send the webhook event
	resp, err := h.webhookHandler.SendEvent(evt.WebhookURI, webhookhandler.MethodTypePOST, webhookhandler.DataTypeJSON, tmp)
	if err != nil {
		return err
	}
	log.Debugf("Sent the webhook event. status: %d", resp.StatusCode)

	return nil
}
