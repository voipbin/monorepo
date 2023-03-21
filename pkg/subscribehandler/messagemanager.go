package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	mmmessage "gitlab.com/voipbin/bin-manager/message-manager.git/models/message"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
)

// processEventMessageMessageCreated handles the message-manager's message_created event.
func (h *subscribeHandler) processEventMessageMessageCreated(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventMessageMessageCreated",
		"event": m,
	})

	e := &mmmessage.Message{}
	if err := json.Unmarshal([]byte(m.Data), &e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.conversationHandler.Event(ctx, conversation.ReferenceTypeMessage, m.Data); errEvent != nil {
		log.Errorf("Could not handle the event correctly. err: %v", errEvent)
		return errEvent
	}

	return nil
}
