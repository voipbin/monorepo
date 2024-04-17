package subscribehandler

import (
	"context"
	"encoding/json"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/sirupsen/logrus"
)

// processEventCMCallHangup handles the call-manager's call_hangup event.
func (h *subscribeHandler) processEventCMCallHangup(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMCallHangup",
		"event": m,
	})

	c := &cmcall.Call{}
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}
	log.Debugf("Executing the event handler.")

	if errEvent := h.transcribeHandler.EventCMCallHangup(ctx, c); errEvent != nil {
		log.Errorf("Could not handle the event correctly. the transcribe handler returned an error. err: %v", errEvent)
	}

	return nil
}

// processEventCMConfbridgeTerminated handles the call-manager's confbridge_terminated event.
func (h *subscribeHandler) processEventCMConfbridgeTerminated(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMConfbridgeTerminated",
		"event": m,
	})

	c := &cmconfbridge.Confbridge{}
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}
	log.Debugf("Executing the event handler.")

	if errEvent := h.transcribeHandler.EventCMConfbridgeTerminated(ctx, c); errEvent != nil {
		log.Errorf("Could not handle the event correctly. the transcribe handler returned an error. err: %v", errEvent)
	}

	return nil
}
