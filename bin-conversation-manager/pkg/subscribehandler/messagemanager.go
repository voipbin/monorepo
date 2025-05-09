package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	mmmessage "monorepo/bin-message-manager/models/message"

	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/conversation"
)

// processEventMessageMessageCreated handles the message-manager's message_created event.
func (h *subscribeHandler) processEventMessageMessageCreated(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventMessageMessageCreated",
		"event": m,
	})

	e := &mmmessage.Message{}
	if err := json.Unmarshal([]byte(m.Data), &e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.conversationHandler.Event(ctx, conversation.TypeMessage, m.Data); errEvent != nil {
		log.Errorf("Could not handle the event correctly. err: %v", errEvent)
		return errEvent
	}

	return nil
}
