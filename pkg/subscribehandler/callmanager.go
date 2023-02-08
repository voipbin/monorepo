package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processEventCMConfbridgeJoined handles the call-manager's call related event
func (h *subscribeHandler) processEventCMConfbridgeJoined(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventCMConfbridgeJoined",
			"event": m,
		},
	)
	log.Debugf("Received call event. event: %s", m.Type)

	evt := cmconfbridge.EventConfbridgeJoined{}
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}
	log.WithFields(
		logrus.Fields{
			"event": evt,
		},
	).Debugf("Detail event. event: %s", m.Type)

	// get chatbotcall
	cc, err := h.chatbotcallHandler.GetByReferenceID(ctx, evt.JoinedCallID)
	if err != nil {
		// chatbotcall not found. Not a chatbotcall.
		return nil
	}

	tmp, err := h.chatbotcallHandler.ProcessStart(ctx, cc)
	if err != nil {
		log.Errorf("Could not start the chatbotcall. chatbotcall_id: %s", err)
		return err
	}
	log.WithField("chatbotcall", tmp).Debugf("Start chatbotcall. chatbotcall_id: %s", tmp.ID)

	return nil
}

// processEventCMConfbridgeLeaved handles the call-manager's call related event
func (h *subscribeHandler) processEventCMConfbridgeLeaved(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventCMConfbridgeLeaved",
			"event": m,
		},
	)
	log.Debugf("Received call event. event: %s", m.Type)

	evt := cmconfbridge.EventConfbridgeLeaved{}
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}
	log.WithFields(
		logrus.Fields{
			"event": evt,
		},
	).Debugf("Detail event. event: %s", m.Type)

	// get chatbotcall
	cc, err := h.chatbotcallHandler.GetByReferenceID(ctx, evt.LeavedCallID)
	if err != nil {
		// chatbotcall not found.
		return nil
	}

	tmp, err := h.chatbotcallHandler.ProcessEnd(ctx, cc)
	if err != nil {
		log.Errorf("Could not terminated the chatbotcall call. err: %v", err)
		return err
	}
	log.WithField("chatbotcall", tmp).Debugf("Termianted chatbotcall. chatbotcall_id: %s", tmp.ID)

	return nil
}
