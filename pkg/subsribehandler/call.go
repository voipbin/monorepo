package subscribehandler

import (
	"encoding/json"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/webhookhandler"
)

// processEventCMCallCommon handles the call-manager's call related event
func (h *subscribeHandler) processEventCMCallCommon(m *rabbitmqhandler.Event) error {

	log := logrus.WithFields(
		logrus.Fields{
			"event": m,
		},
	)
	log.Debugf("Sending an event.")

	var evt call.Call
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		// same call-id is already exsit
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}
	log.Debugf("Marshaled message. %v", evt)

	type msg struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	tmpMsg := msg{
		Type: "call_create",
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

	log.WithFields(
		logrus.Fields{
			"response": resp,
		},
	).Debugf("Sent the webhook event. status: %d", resp.StatusCode)

	return nil
}
