package subscribehandler

import (
	"context"
	"encoding/json"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/sirupsen/logrus"
)

// processEventCMConfbridgeJoined handles the call-manager's call related event
func (h *subscribeHandler) processEventCMConfbridgeJoined(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMConfbridgeJoined",
		"event": m,
	})

	evt := cmconfbridge.EventConfbridgeJoined{}
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	// get chatbotcall
	cc, err := h.chatbotcallHandler.GetByReferenceID(ctx, evt.JoinedCallID)
	if err != nil {
		// chatbotcall not found. Not a chatbotcall.
		return nil
	}

	_, err = h.chatbotcallHandler.ProcessStart(ctx, cc)
	if err != nil {
		log.Errorf("Could not start the chatbotcall. chatbotcall_id: %s", err)
		return err
	}

	return nil
}

// processEventCMConfbridgeLeaved handles the call-manager's call related event
func (h *subscribeHandler) processEventCMConfbridgeLeaved(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMConfbridgeLeaved",
		"event": m,
	})

	evt := cmconfbridge.EventConfbridgeLeaved{}
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	// get chatbotcall
	cc, err := h.chatbotcallHandler.GetByReferenceID(ctx, evt.LeavedCallID)
	if err != nil {
		// chatbotcall not found.
		return nil
	}

	_, err = h.chatbotcallHandler.ProcessEnd(ctx, cc)
	if err != nil {
		log.Errorf("Could not terminated the chatbotcall call. err: %v", err)
		return err
	}

	return nil
}
