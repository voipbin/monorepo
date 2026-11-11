package messagehandler

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-talk-manager/models/message"
)

// MentionList returns the messages that mention the agent, newest first.
func (h *messageHandler) MentionList(ctx context.Context, customerID, agentID uuid.UUID, token string, size uint64) ([]*message.Message, error) {
	return h.dbHandler.MentionList(ctx, customerID, agentID, token, size)
}
