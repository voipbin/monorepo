package servicehandler

import (
	"context"
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// ServiceAgentConversationGet gets the conversation of the given id.
// It returns conversation if it succeed.
func (h *serviceHandler) ServiceAgentConversationGet(ctx context.Context, a *amagent.Agent, conversationID uuid.UUID) (*cvconversation.WebhookMessage, error) {
	// get
	tmp, err := h.conversationGet(ctx, a, conversationID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get conversation.")
	}

	if tmp.OwnerID != a.ID {
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentConversationGets sends a request to conversation-manager
// to getting the list of conversation.
// it returns list of chatroom messages if it succeed.
func (h *serviceHandler) ServiceAgentConversationGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cvconversation.WebhookMessage, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// filters
	filters := map[string]string{
		"owner_id": a.ID.String(),
		"deleted":  "false", // we don't need deleted items
	}

	tmps, err := h.conversationGets(ctx, a, size, token, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get the conversations.")
	}

	// create result
	res := []*cvconversation.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}
