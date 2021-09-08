package subscribehandler

import (
	"encoding/json"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processEventCMCallCommon handles the call-manager's call related event
func (h *subscribeHandler) processEventCMCallCommon(m *rabbitmqhandler.Event) error {

	log := logrus.WithFields(
		logrus.Fields{
			"event": m,
		},
	)
	log.Debugf("Received call event. event: %s", m.Type)

	var evt call.Call
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		// same call-id is already exsit
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}
	log.WithFields(
		logrus.Fields{
			"event": evt,
		},
	).Debugf("Detail event. event: %s", m.Type)

	return nil
}
