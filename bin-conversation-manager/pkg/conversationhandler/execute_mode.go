package conversationhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/message"
)

// getExecuteMode reads the conversation's Owner snapshot and returns the dispatch mode.
// See docs/plans/2026-04-30-assignable-conversation-design.md §3.1: callers MUST NOT re-fetch
// the Conversation in the dispatch path; the snapshot already loaded by the inbound handler is authoritative.
func (h *conversationHandler) getExecuteMode(cv *conversation.Conversation) ExecuteMode {
	if cv.OwnerType == commonidentity.OwnerTypeAgent && cv.OwnerID != uuid.Nil {
		return ExecuteModeAgent
	}
	return ExecuteModeFlow
}

// runExecuteModeAgent handles inbound messages on conversations owned by an agent.
// The agent UI learns of new messages via the existing `message_created` event filtered on cv.OwnerID.
// No new event is published; no flow is triggered. Logging only.
func (h *conversationHandler) runExecuteModeAgent(ctx context.Context, cv *conversation.Conversation, m *message.Message) error {
	log := logrus.WithFields(logrus.Fields{
		"func":            "runExecuteModeAgent",
		"conversation_id": cv.ID,
		"message_id":      m.ID,
		"owner_id":        cv.OwnerID,
	})
	log.Infof("Conversation owned by agent. Skipping flow trigger.")
	return nil
}
