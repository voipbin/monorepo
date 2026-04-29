package conversationhandler

import (
	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-conversation-manager/models/conversation"
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
