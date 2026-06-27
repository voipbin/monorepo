package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	convmsg "monorepo/bin-conversation-manager/models/message"
)

// processEventConversationManagerMessageCreated handles the conversation-manager's
// conversation_message_created event and projects it into the CRM interaction timeline.
func (h *subscribeHandler) processEventConversationManagerMessageCreated(ctx context.Context, m *sock.Event) error {
	var payload convmsg.WebhookMessage
	if err := json.Unmarshal(m.Data, &payload); err != nil {
		return err
	}

	return h.contactHandler.EventConversationMessageCreated(ctx, &payload)
}
